import React, { useCallback, useEffect, useRef, useState } from 'react';
import { CheckCircle2 } from 'lucide-react';
import { FLOW, LEGACY_INFO_ROUTES } from './data';
import { createOnboardingGreeting } from './features/profile/onboardingGreeting';
import { VALID_ROUTES, ONBOARDING_ROUTES, canOpenRoute, canVisitOnboarding, onboardingMove, visibleOnboardingRoute, previousOnboardingRoute } from './features/profile/navigation';
import { approachForStyle, styleForApproach, validateDisplayName } from './features/profile/preferences';
import OnboardingScreen from './screens/Onboarding';
import MainScreen from './screens/Main';
import DevLogin from './features/devsession/DevLogin';
import { devSessionApi } from './features/devsession/api';
import { createSessionStore } from './features/devsession/sessionStore';
import { onboardingProgressApi } from './features/profile/onboardingProgress';
import { privacyPreferencesApi, createPrivacyPreferencesStore } from './features/profile/privacyPreferences';
import { screenSurfaceForRoute } from './features/appearance/screenSurface';
import { conversationApi } from './features/conversation/api';
import { createConversationStore } from './features/conversation/store';
import { normalizeLetter } from './features/letter/model';

const DEFAULT_BACK = { home: 'welcome', survey: 'menu', history: 'home', menu: 'home', chat: 'home', letter: 'home' };
function routeFromHash() {
  const route = window.location.hash.replace(/^#\/?/, '').split('/')[0];
  return VALID_ROUTES.has(route) ? route : 'welcome';
}
function replaceRoute(route) {
  window.history.replaceState(null, '', `#/${route}`);
}

export default function App() {
  const [route, setRoute] = useState(routeFromHash);
  const [name, setName] = useState('');
  const [answers, setAnswers] = useState({});
  const [onboardingGreeting, setOnboardingGreeting] = useState('');
  const enteringChatAfterOnboarding = useRef(false);
  const [toast, setToast] = useState('');
  const backOverrides = useRef({});
  const identityVersion = useRef(0);
  // Server owns the monotonic route cursor. Sensitive answers stay in React memory only.
  const onboardingProgress = useRef({ userId: null, furthest: 0 });
  const progressRequest = useRef(null);
  const progressWritePending = useRef(false);
  const [progressPhase, setProgressPhase] = useState('loading');
  const [progressRetry, setProgressRetry] = useState(0);
  const [progressSaving, setProgressSaving] = useState(false);
  const sessionStoreRef = useRef(null);
  const conversationStoreRef = useRef(null);
  const privacyStoreRef = useRef(null);
  // Reserved for validated backend-authored letter data; never seeded with demo insights.
  const [letter] = useState(null);
  if (privacyStoreRef.current === null) {
    privacyStoreRef.current = createPrivacyPreferencesStore(privacyPreferencesApi, {
      onUnauthorized: () => { void sessionStoreRef.current?.bootstrap(); },
    });
  }
  const privacyStore = privacyStoreRef.current;
  if (conversationStoreRef.current === null) {
    conversationStoreRef.current = createConversationStore(conversationApi, { onUnauthorized: () => { void sessionStoreRef.current?.bootstrap(); } });
  }
  const chatStore = conversationStoreRef.current;
  if (sessionStoreRef.current === null) {
    sessionStoreRef.current = createSessionStore(devSessionApi, {
      onInvalidate: () => {
        identityVersion.current += 1;
        setName('');
        setAnswers({});
        setOnboardingGreeting('');
        enteringChatAfterOnboarding.current = false;
        progressRequest.current?.abort();
        onboardingProgress.current = { userId: null, furthest: 0 };
        progressWritePending.current = false;
        // Invalidating the session fences any pending name save.
        nameWritePending.current = false;
        setProgressSaving(false);
        setProgressPhase('loading');
        chatStore.reset();
        privacyStore.reset();
        setToast('');
        setProfileSaving(false);
        backOverrides.current = {};
      },
    });
  }
  const store = sessionStoreRef.current;
  const [session, setSession] = useState(store.getState);
  const [chat, setChat] = useState(chatStore.getState);
  const [privacy, setPrivacy] = useState(privacyStore.getState);
  const [profileSaving, setProfileSaving] = useState(false);
  const active = session.phase === 'authenticated' || session.phase === 'saving-profile';
  const complete = Boolean(session.profile?.onboardingCompletedAt);

  useEffect(() => {
    const unsubscribe = store.subscribe(setSession);
    store.bootstrap();
    return () => { unsubscribe(); store.cancel(); };
  }, [store]);

  useEffect(() => {
    const unsubscribe = chatStore.subscribe(setChat);
    return () => { unsubscribe(); chatStore.cancel(); };
  }, [chatStore]);

  useEffect(() => {
    chatStore.activate(complete ? session.profile?.userId : null);
  }, [chatStore, complete, session.profile?.userId]);

  useEffect(() => {
    return privacyStore.subscribe(setPrivacy);
  }, [privacyStore]);

  useEffect(() => {
    privacyStore.activate(active ? session.profile?.userId : null);
  }, [privacyStore, active, session.profile?.userId]);

  useEffect(() => {
    if (active && route === 'privacy') void privacyStore.load();
  }, [active, route, session.profile?.userId, privacyStore]);

  useEffect(() => {
    if (active && complete && ['chat', 'history'].includes(route)) void chatStore.list();
    if (active && complete && ['chat', 'history'].includes(route)) void chatStore.listBookmarks();
  }, [active, complete, route, chatStore]);

  // Resolve the cursor before showing incomplete onboarding. Never trust a hash URL as progress.
  useEffect(() => {
    if (!active || !session.profile) return undefined;
    setName(session.profile.displayName);
    if (complete) {
      progressRequest.current?.abort();
      setProgressPhase('ready');
      const afterFinish = enteringChatAfterOnboarding.current;
      enteringChatAfterOnboarding.current = false;
      if (afterFinish) chatStore.startNew();
      replaceRoute(afterFinish ? 'chat' : 'home');
      setRoute(afterFinish ? 'chat' : 'home');
      return undefined;
    }
    const controller = new AbortController();
    progressRequest.current = controller;
    const version = identityVersion.current;
    setProgressPhase('loading');
    void onboardingProgressApi.read(controller.signal).then(async (furthest) => {
      if (controller.signal.aborted || identityVersion.current !== version) return;
      const destination = visibleOnboardingRoute(ONBOARDING_ROUTES[furthest]);
      const targetStep = ONBOARDING_ROUTES.indexOf(destination);
      const saved = await onboardingProgressApi.advanceTo(targetStep, furthest, controller.signal);
      if (controller.signal.aborted || identityVersion.current !== version) return;
      onboardingProgress.current = { userId: session.profile.userId, furthest: saved };
      replaceRoute(destination);
      setRoute(destination);
      setProgressPhase('ready');
    }).catch((error) => {
      if (controller.signal.aborted || identityVersion.current !== version) return;
      if (error?.status === 401) { void store.bootstrap(); return; }
      setProgressPhase('error');
    });
    return () => controller.abort();
  }, [active, complete, session.profile?.userId, progressRetry]);

  useEffect(() => {
    const onHashChange = () => setRoute(routeFromHash());
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  useEffect(() => {
    if (!active || (!complete && progressPhase !== 'ready')) return;
    if (!canOpenRoute(route, complete) || (!complete && !canVisitOnboarding(route, onboardingProgress.current.furthest))) {
      const destination = complete ? 'home' : visibleOnboardingRoute(ONBOARDING_ROUTES[onboardingProgress.current.furthest]);
      replaceRoute(destination);
      setRoute(destination);
    }
  }, [active, complete, route, progressPhase]);

  useEffect(() => {
    if (!active) return;
    window.scrollTo(0, 0);
    document.documentElement.style.colorScheme = ['home', 'survey', 'history', 'menu', 'chat', 'letter', 'research', 'experts', 'simple-intro', 'deeper-intro', 'know-intro', 'almost-intro', 'voice'].includes(route) ? 'dark' : 'light';
  }, [active, route]);

  useEffect(() => {
    if (!toast) return undefined;
    const timer = window.setTimeout(() => setToast(''), 3500);
    return () => window.clearTimeout(timer);
  }, [toast]);

  const finishProfile = useCallback(async () => {
    if (profileSaving) return;
    // Name is saved at step four; use the persisted value when resuming an older
    // in-progress session. For users affected by the earlier bug who already
    // passed that step with no saved name, finish without an extra name prompt.
    const displayName = validateDisplayName(name)
      || validateDisplayName(session.profile?.displayName)
      || 'دوست من';
    const version = identityVersion.current;
    // The session store publishes the completed profile *before* its promise
    // resolves. Set this intent before saving, so the profile-gate effect can
    // reliably open Chat even if React commits the subscription update first.
    enteringChatAfterOnboarding.current = true;
    setOnboardingGreeting(createOnboardingGreeting(displayName, answers));
    setProfileSaving(true);
    try {
      const success = await store.saveProfile(displayName, styleForApproach(answers.approach));
      if (version === identityVersion.current) {
        if (success) {
          setAnswers({}); // The entire intake stays ephemeral, never sent to the API.
        } else {
          enteringChatAfterOnboarding.current = false;
          setOnboardingGreeting('');
          setToast('ذخیره نشد. دوباره تلاش کن.');
        }
      }
      // Successful save updates /me and the profile gate opens a new chat.
    } finally { if (version === identityVersion.current) setProfileSaving(false); }
  }, [name, answers, profileSaving, store, session.profile?.displayName]);

  // Reuse the original Name and Approach screens as profile editors from the menu.
  // No additional stored fields or backend endpoints are introduced.
  const saveExistingProfile = useCallback(async (displayName, conversationStyle) => {
    const clean = validateDisplayName(displayName);
    if (!clean) { setToast('اسمت باید بین ۱ تا ۶۰ کاراکتر باشه.'); return false; }
    if (profileSaving) return false;
    const version = identityVersion.current;
    setProfileSaving(true);
    try {
      const success = await store.saveProfile(clean, conversationStyle);
      if (version !== identityVersion.current) return false;
      if (!success) { setToast('ذخیره نشد. دوباره تلاش کن.'); return false; }
      setName(clean);
      backOverrides.current = {};
      replaceRoute('menu');
      setRoute('menu');
      setToast('تغییراتت ذخیره شد.');
      return true;
    } finally {
      if (version === identityVersion.current) setProfileSaving(false);
    }
  }, [profileSaving, store]);
  const saveName = useCallback(() => saveExistingProfile(name, session.profile?.conversationStyle || 'balanced'),
    [saveExistingProfile, name, session.profile?.conversationStyle]);
  const saveApproach = useCallback((approach) => saveExistingProfile(session.profile?.displayName || name, styleForApproach(approach)),
    [saveExistingProfile, name, session.profile?.displayName]);

  const navigate = useCallback((requestedTarget) => {
    if (!VALID_ROUTES.has(requestedTarget) || !active || (!complete && progressPhase !== 'ready')) return;
    const target = complete && route === 'menu'
      ? requestedTarget : visibleOnboardingRoute(requestedTarget) || requestedTarget;
    if (!complete) {
      if (route === 'ready' && target === 'home') { void finishProfile(); return; }
      const nextFurthest = onboardingMove(route, target, onboardingProgress.current.furthest);
      if (nextFurthest === null || progressWritePending.current) return;
      if (nextFurthest > onboardingProgress.current.furthest) {
        const version = identityVersion.current;
        const fromUser = session.profile?.userId;
        const controller = new AbortController();
        progressRequest.current = controller;
        progressWritePending.current = true;
        setProgressSaving(true);
        void onboardingProgressApi.advanceTo(nextFurthest, onboardingProgress.current.furthest, controller.signal, (saved) => {
          if (!controller.signal.aborted && identityVersion.current === version) onboardingProgress.current.furthest = saved;
        }).then((saved) => {
          if (controller.signal.aborted || identityVersion.current !== version || session.profile?.userId !== fromUser) return;
          onboardingProgress.current.furthest = saved;
          window.location.hash = `/${target}`;
        }).catch((error) => {
          if (controller.signal.aborted || identityVersion.current !== version) return;
          if (error?.status === 401) { void store.bootstrap(); return; }
          setToast('پیشرفت آنبوردینگ در سرور ذخیره نشد. اگر مشکل ادامه داشت، به پشتیبانی اطلاع بده.');
        }).finally(() => {
          if (identityVersion.current === version) { progressWritePending.current = false; setProgressSaving(false); }
        });
        return;
      }
    }
    // The original screens remain navigable, whether or not their eventual APIs exist.
    // Only unfinished onboarding is guarded from server-backed chat/history.
    // Entering Chat resumes the latest thread or shows a local empty composer.
    // A server conversation is only created by the first submitted message.
    if (target === 'chat' && route !== 'history' && route !== 'chat') void chatStore.resume();
    if (route === 'menu' && ['name', 'approach', 'privacy', 'research'].includes(target)) {
      backOverrides.current[target] = 'menu';
      if (target === 'name') setName(session.profile?.displayName || '');
      if (target === 'approach') setAnswers((previous) => ({ ...previous, approach: approachForStyle(session.profile?.conversationStyle) }));
    }
    if (route === 'chat' && target !== 'chat') setOnboardingGreeting('');
    if (target === route) return;
    window.location.hash = `/${target}`;
  }, [route, active, complete, progressPhase, finishProfile, chatStore, store, session.profile?.userId, session.profile?.displayName, session.profile?.conversationStyle]);

  const nameWritePending = useRef(false);
  const saveOnboardingName = useCallback(async () => {
    if (nameWritePending.current || progressWritePending.current || complete || route !== 'name') return;
    const displayName = validateDisplayName(name);
    if (!displayName) { setToast('اسمت باید بین ۱ تا ۶۰ کاراکتر باشه.'); return; }
    nameWritePending.current = true;
    setProfileSaving(true);
    const version = identityVersion.current;
    try {
      const updated = await devSessionApi.saveOnboardingName(displayName);
      if (identityVersion.current !== version || updated.userId !== session.profile?.userId) return;
      setName(updated.displayName);
      navigate('age');
    } catch (error) {
      if (identityVersion.current !== version) return;
      if (error?.status === 401) { void store.bootstrap(); return; }
      setToast('اسمت ذخیره نشد. اتصال رو بررسی کن و دوباره امتحان کن.');
    } finally {
      if (identityVersion.current === version) {
        nameWritePending.current = false;
        setProfileSaving(false);
      }
    }
  }, [complete, route, name, navigate, session.profile?.userId, store]);

  const back = useCallback(() => {
    const override = backOverrides.current[route];
    if (override) {
      delete backOverrides.current[route];
      if (route === 'name') setName(session.profile?.displayName || '');
      if (route === 'approach') setAnswers((previous) => ({ ...previous, approach: approachForStyle(session.profile?.conversationStyle) }));
      navigate(override);
      return;
    }
    const previous = previousOnboardingRoute(route);
    if (previous) { navigate(previous); return; }
    const flowIndex = FLOW.indexOf(route);
    if (flowIndex > 0) { navigate(FLOW[flowIndex - 1]); return; }
    navigate(DEFAULT_BACK[route] || 'home');
  }, [complete, navigate, route, store, session.profile?.displayName, session.profile?.conversationStyle]);

  const setAnswer = useCallback((key, value) => setAnswers((current) => ({ ...current, [key]: value })), []);
  const notify = useCallback((message) => setToast(message), []);
  const reset = useCallback(() => {
    setAnswers({}); setOnboardingGreeting(''); chatStore.reset(); setName(''); backOverrides.current = {};
    progressRequest.current?.abort();
    onboardingProgress.current = { userId: null, furthest: 0 };
    progressWritePending.current = false;
    nameWritePending.current = false;
  }, [chatStore]);

  const openConversation = useCallback((id) => {
    void chatStore.open(id);
    window.location.hash = '/chat';
  }, [chatStore]);
  const deleteConversation = useCallback(async (id) => {
    if (!window.confirm('گفت‌وگو حذف بشه؟ دیگه برنمی‌گرده.')) return;
    const ok = await chatStore.remove(id);
    if (!ok) setToast('گفت‌وگو حذف نشد. دوباره تلاش کن.');
  }, [chatStore]);

  const shared = {
    route, navigate, back, name, setName, answers, setAnswer, onboardingGreeting,
    letter: normalizeLetter(letter),
    messages: chat.messages, conversations: chat.conversations, conversationId: chat.conversationId,
    pendingTurn: chat.pending, sending: chat.sending, chatLoading: chat.messagesPhase === 'loading',
    chatError: chat.messageError, historyPhase: chat.historyPhase, historyError: chat.historyError,
    deletingId: chat.deletingId, renamingId: chat.renamingId,
    bookmarks: chat.bookmarks, bookmarkPhase: chat.bookmarkPhase, bookmarkError: chat.bookmarkError,
    bookmarkBusyId: chat.bookmarkBusyId,
    onToggleBookmark: (message) => chatStore.toggleBookmark(message),
    onReloadBookmarks: () => chatStore.listBookmarks(),
    onSendMessage: (text) => { setOnboardingGreeting(''); return chatStore.send(text); },
    onRetryMessage: () => chatStore.retry(), onNewConversation: () => {
      if (chat.pending && !window.confirm('پاسخ قبلیت هنوز آماده نشده. گفت‌وگوی جدید بسازی؟')) return false;
      setOnboardingGreeting('');
      return chatStore.startNew();
    },
    onOpenConversation: openConversation, onDeleteConversation: deleteConversation,
    onRenameConversation: (id, title) => chatStore.rename(id, title),
    onReloadHistory: () => chatStore.list(),
    notify, reset, savingProfile: profileSaving,
    privacySharing: privacy.enabled, privacyLoading: privacy.loading, privacySaving: privacy.saving,
    privacyLoaded: privacy.loaded, privacyError: privacy.error,
    onTogglePrivacy: () => privacyStore.toggle(), onRetryPrivacy: () => privacyStore.load(),
    profileEdit: complete && ['name', 'approach'].includes(route) && backOverrides.current[route] === 'menu',
    onSaveName: saveName, onSaveOnboardingName: saveOnboardingName, onSaveApproach: saveApproach,
    onLogout: () => store.leave(),
    onSwitchTester: () => store.leave(),
  };

  return (
    <div className="app-shell" data-screen-surface={active ? screenSurfaceForRoute(route) : 'light'} dir="rtl">
      {active && !complete && progressPhase !== 'ready' ? (
        <section className="connected-status" role="status">
          {progressPhase === 'error' ? <><p>مرحلهٔ قبلیت پیدا نشد.</p><button type="button" onClick={() => setProgressRetry(n => n + 1)}>تلاش دوباره</button></> : <p>در حال آماده‌سازی…</p>}
        </section>
      ) : active ? (
        <main className="screen-transition" key={`${session.profile?.userId}:${route}`} aria-busy={progressSaving && !complete}>
          {(FLOW.includes(route) && route !== 'home') || LEGACY_INFO_ROUTES.includes(route) ? <OnboardingScreen {...shared} /> : <MainScreen {...shared} />}
        </main>
      ) : (
        <DevLogin session={session} login={(phone) => store.login(phone)} retry={() => session.error?.action === 'logout' ? store.leave() : store.bootstrap()} />
      )}
      {toast && <div className="toast" role="status"><CheckCircle2 size={19} /><span>{toast}</span></div>}
    </div>
  );
}
