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
import { createLetterStore } from './features/letters/store';
import { letterApi } from './features/letters/api';
import { loadReadLetterIds, markReadLetter } from './features/letters/readingState';
import { buildLetterReplyMessage, isLetterReference } from './features/letters/discuss';

const DEFAULT_BACK = { home: 'welcome', settings: 'menu', survey: 'menu', day: 'home', letter: 'home', history: 'home', saved: 'menu', menu: 'home', chat: 'home', insight: 'home' };
function routeFromHash() {
  const route = window.location.hash.replace(/^#\/?/, '').split('/')[0];
  return VALID_ROUTES.has(route) ? route : 'welcome';
}
function replaceRoute(route) {
  window.history.replaceState(null, '', `#/${route}`);
}

export default function App() {
  const [route, setRoute] = useState(routeFromHash);
  const [savedMessageTarget, setSavedMessageTarget] = useState(null);
  const [letterReply, setLetterReply] = useState(null);
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
  const letterStoreRef = useRef(null);
  if (letterStoreRef.current === null) {
    letterStoreRef.current = createLetterStore(letterApi, {
      onUnauthorized: () => { void sessionStoreRef.current?.bootstrap(); },
    });
  }
  const letterStore = letterStoreRef.current;
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
        setLetterReply(null);
        enteringChatAfterOnboarding.current = false;
        progressRequest.current?.abort();
        onboardingProgress.current = { userId: null, furthest: 0 };
        progressWritePending.current = false;
        nameWritePending.current = false;
        setProgressSaving(false);
        setProgressPhase('loading');
        chatStore.reset();
        privacyStore.reset();
        letterStore.reset();
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
  const [letters, setLetters] = useState(letterStore.getState);
  const [letterReads, setLetterReads] = useState({ userId: null, ids: [] });
  const [profileSaving, setProfileSaving] = useState(false);
  const active = session.phase === 'authenticated' || session.phase === 'saving-profile';
  const complete = Boolean(session.profile?.onboardingCompletedAt);
  const letterUserId = complete ? session.profile?.userId : null;
  useEffect(() => {
    setLetterReads({ userId: letterUserId, ids: loadReadLetterIds(letterUserId) });
  }, [letterUserId]);
  // Also clear the pending letter reference on browser Back / direct hash navigation.
  useEffect(() => { if (route !== 'chat') setLetterReply(null); }, [route]);
  const onReadLetter = useCallback((letterId) => {
    if (!letterUserId || !letterId) return;
    const ids = markReadLetter(letterUserId, letterId);
    setLetterReads(prev => prev.userId === letterUserId && prev.ids.includes(letterId)
      ? prev : { userId: letterUserId, ids });
  }, [letterUserId]);

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

  useEffect(() => letterStore.subscribe(setLetters), [letterStore]);

  useEffect(() => {
    letterStore.activate(complete ? session.profile?.userId : null);
  }, [letterStore, complete, session.profile?.userId]);

  useEffect(() => {
    if (active && complete && ['letter', 'day', 'insight'].includes(route)) void letterStore.load();
  }, [letterStore, active, complete, route]);

  useEffect(() => {
    return privacyStore.subscribe(setPrivacy);
  }, [privacyStore]);

  useEffect(() => {
    privacyStore.activate(active ? session.profile?.userId : null);
  }, [privacyStore, active, session.profile?.userId]);

  useEffect(() => {
    if (active && ['privacy', 'settings'].includes(route)) void privacyStore.load();
  }, [active, route, session.profile?.userId, privacyStore]);

  useEffect(() => {
    if (!active || !complete) return;
    if (route === 'history') void chatStore.list();
    if (route === 'saved') void chatStore.listBookmarks();
    if (route === 'chat') {
      // Opening Chat resumes the current in-memory conversation, or restores the
      // latest server-owned conversation after reload. It never auto-creates one.
      void chatStore.resumeLatest();
      void chatStore.listBookmarks();
    }
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

  // A saved-message jump is ephemeral navigation state, not a URL or persisted record.
  useEffect(() => {
    if (route !== 'chat') setSavedMessageTarget(null);
  }, [route]);
  useEffect(() => { setSavedMessageTarget(null); }, [session.profile?.userId]);

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
    document.documentElement.style.colorScheme = ['home', 'survey', 'day', 'letter', 'history', 'saved', 'settings', 'menu', 'chat', 'insight', 'research', 'experts', 'simple-intro', 'deeper-intro', 'know-intro', 'almost-intro', 'voice'].includes(route) ? 'dark' : 'light';
  }, [active, route]);

  // Keep the mobile composer above the visual viewport when the keyboard opens.
  useEffect(() => {
    const root = document.documentElement;
    const update = () => {
      const viewport = window.visualViewport;
      const height = viewport?.height ?? window.innerHeight;
      root.style.setProperty('--app-height', `${Math.round(height)}px`);
    };
    update();
    window.addEventListener('resize', update);
    window.visualViewport?.addEventListener('resize', update);
    window.visualViewport?.addEventListener('scroll', update);
    return () => {
      window.removeEventListener('resize', update);
      window.visualViewport?.removeEventListener('resize', update);
      window.visualViewport?.removeEventListener('scroll', update);
      root.style.removeProperty('--app-height');
    };
  }, []);

  useEffect(() => {
    if (!toast) return undefined;
    const timer = window.setTimeout(() => setToast(''), 3500);
    return () => window.clearTimeout(timer);
  }, [toast]);

  const finishProfile = useCallback(async () => {
    if (profileSaving) return;
    const displayName = validateDisplayName(name)
      || validateDisplayName(session.profile?.displayName)
      || 'دوست من';
    const version = identityVersion.current;
    enteringChatAfterOnboarding.current = true;
    setOnboardingGreeting(createOnboardingGreeting(displayName, answers));
    setProfileSaving(true);
    try {
      const success = await store.saveProfile(displayName, styleForApproach(answers.approach));
      if (version === identityVersion.current) {
        if (success) {
          setAnswers({});
        } else {
          enteringChatAfterOnboarding.current = false;
          setOnboardingGreeting('');
          setToast('ذخیره نشد. دوباره تلاش کن.');
        }
      }
    } finally { if (version === identityVersion.current) setProfileSaving(false); }
  }, [name, answers, profileSaving, store, session.profile?.displayName]);

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
  const saveSettings = useCallback(async (displayName, conversationStyle) => {
    const clean = validateDisplayName(displayName);
    if (!clean || profileSaving) return false;
    const version = identityVersion.current;
    setProfileSaving(true);
    try {
      const success = await store.saveProfile(clean, conversationStyle);
      if (identityVersion.current !== version || !success) return false;
      setName(clean);
      // Stay in Settings; changes are persisted in the existing profile API.
      return true;
    } finally {
      if (identityVersion.current === version) setProfileSaving(false);
    }
  }, [profileSaving, store]);

  // Reuse the original example-rich onboarding screen when changing the style
  // from Settings. Save the selected style, then return to Settings directly.
  const saveApproach = useCallback(async (approach) => {
    if (backOverrides.current.approach !== 'settings') {
      return saveExistingProfile(session.profile?.displayName || name, styleForApproach(approach));
    }
    const saved = await saveSettings(session.profile?.displayName || name, styleForApproach(approach));
    if (!saved) { setToast('سبک گفت‌وگو ذخیره نشد. دوباره تلاش کن.'); return false; }
    delete backOverrides.current.approach;
    window.location.hash = '/settings';
    return true;
  }, [saveExistingProfile, saveSettings, session.profile?.displayName, name]);

  const navigate = useCallback((requestedTarget) => {
    if (!VALID_ROUTES.has(requestedTarget) || !active || (!complete && progressPhase !== 'ready')) return;
    // Legacy diary/insight hashes resolve to the letter experience, never diary UI.
    const requested = ['day', 'insight'].includes(requestedTarget) ? 'letter' : requestedTarget;
    const target = complete && route === 'menu'
      ? requested : visibleOnboardingRoute(requested) || requested;
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
    if (route === 'menu' && ['name', 'approach', 'privacy', 'research', 'letter', 'settings'].includes(target)) {
      backOverrides.current[target] = 'menu';
      if (target === 'name') setName(session.profile?.displayName || '');
      if (target === 'approach') setAnswers((previous) => ({ ...previous, approach: approachForStyle(session.profile?.conversationStyle) }));
    }
    if (route === 'settings' && target === 'approach') {
      backOverrides.current.approach = 'settings';
      setAnswers((previous) => ({ ...previous, approach: approachForStyle(session.profile?.conversationStyle) }));
    }
    if (route === 'chat' && target !== 'chat') { setOnboardingGreeting(''); setLetterReply(null); }
    if (target === route) return;
    if (target !== 'chat') setSavedMessageTarget(null);
    window.location.hash = `/${target}`;
  }, [route, active, complete, progressPhase, finishProfile, store, session.profile?.userId, session.profile?.displayName, session.profile?.conversationStyle]);

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
    setAnswers({}); setOnboardingGreeting(''); setSavedMessageTarget(null); setLetterReply(null); chatStore.reset(); letterStore.reset(); setName(''); backOverrides.current = {};
    progressRequest.current?.abort();
    onboardingProgress.current = { userId: null, furthest: 0 };
    progressWritePending.current = false;
    nameWritePending.current = false;
  }, [chatStore, letterStore]);

  const confirmLeavePendingTurn = useCallback(() => {
    if (chat.sending) {
      setToast('صبر کن پاسخ قبلی آماده بشه.');
      return false;
    }
    if (chat.pending && !window.confirm('ارسال پیام قبلی کامل نشده. اگر گفت‌وگو رو عوض کنی، فعلاً نمی‌تونی همون پیام رو دوباره تلاش کنی. ادامه می‌دی؟')) return false;
    return true;
  }, [chat.sending, chat.pending]);

  const openConversation = useCallback((id) => {
    if (!confirmLeavePendingTurn()) return false;
    setLetterReply(null);
    setSavedMessageTarget(null);
    void chatStore.open(id);
    window.location.hash = '/chat';
    return true;
  }, [chatStore, confirmLeavePendingTurn]);
  const openSavedMessage = useCallback((item) => {
    if (!item?.conversationId || !item?.id || !confirmLeavePendingTurn()) return false;
    setLetterReply(null);
    setSavedMessageTarget({ conversationId: item.conversationId, messageId: item.id });
    void chatStore.open(item.conversationId);
    window.location.hash = '/chat';
    return true;
  }, [chatStore, confirmLeavePendingTurn]);
  // Start a new, empty chat with a letter reference in the composer. No text is
  // sent until the person writes and sends their own first message.
  const discussLetter = useCallback((letter) => {
    if (!isLetterReference(letter) || !confirmLeavePendingTurn()) return false;
    if (!chatStore.startNew()) return false;
    chatStore.setDraft('');
    setLetterReply(letter);
    setSavedMessageTarget(null);
    setOnboardingGreeting('');
    window.location.hash = '/chat';
    return true;
  }, [chatStore, confirmLeavePendingTurn]);
  const clearSavedMessageTarget = useCallback((handled) => {
    setSavedMessageTarget((current) => current?.conversationId === handled.conversationId &&
      current?.messageId === handled.messageId ? null : current);
  }, []);

  const deleteConversation = useCallback(async (id) => {
    if (!window.confirm('گفت‌وگو حذف بشه؟ دیگه برنمی‌گرده.')) return;
    const ok = await chatStore.remove(id);
    if (!ok) setToast('گفت‌وگو حذف نشد. دوباره تلاش کن.');
  }, [chatStore]);
  const retryChatLoad = useCallback(() => {
    if (chat.conversationId) return chatStore.open(chat.conversationId);
    return chatStore.resumeLatest(true);
  }, [chatStore, chat.conversationId]);

  const activeConversation = chat.conversations.find((item) => item.id === chat.conversationId) || null;
  const shared = {
    route, navigate, back, name, setName, answers, setAnswer, onboardingGreeting,
    messages: chat.messages, conversations: chat.conversations,
    conversationId: chat.conversationId, activeConversation,
    pendingTurn: chat.pending, sending: chat.sending, chatLoading: chat.messagesPhase === 'loading',
    chatError: chat.messageError, historyPhase: chat.historyPhase, historyError: chat.historyError,
    deletingId: chat.deletingId, renamingId: chat.renamingId,
    chatDraft: chat.draft, onDraftChange: chatStore.setDraft,
    letterReply, onDismissLetterReply: () => setLetterReply(null),
    bookmarks: chat.bookmarks, bookmarkPhase: chat.bookmarkPhase, bookmarkError: chat.bookmarkError,
    bookmarkBusyId: chat.bookmarkBusyId,
    onToggleBookmark: (message) => chatStore.toggleBookmark(message),
    onReloadBookmarks: () => chatStore.listBookmarks(),
    onSendMessage: (text) => {
      const message = letterReply ? buildLetterReplyMessage(text, letterReply) : text;
        if (!message) { setToast('پیامت رو کمی کوتاه‌تر بنویس تا متن نامه هم کنارش فرستاده بشه.'); return false; }
      setOnboardingGreeting('');
      const queued = chatStore.send(message);
      // The store synchronously creates a pending turn when it accepts a send.
      // Retries use that exact context-bearing turn; don't attach it twice.
      if (letterReply && chatStore.getState().pending) setLetterReply(null);
      return queued;
    },
    onRetryMessage: () => chatStore.retry(), onRetryChatLoad: retryChatLoad,
    onNewConversation: () => {
      if (!confirmLeavePendingTurn()) return false;
      setSavedMessageTarget(null);
      setLetterReply(null);
      setOnboardingGreeting('');
      return chatStore.startNew();
    },
    onOpenConversation: openConversation, onOpenSavedMessage: openSavedMessage,
    savedMessageTarget, onSavedMessageTargetDone: clearSavedMessageTarget, onDeleteConversation: deleteConversation,
    onRenameConversation: (id, title) => chatStore.rename(id, title),
    onReloadHistory: () => chatStore.list(),
    letterPhase: letters.phase, letters: letters.letters, letterError: letters.error,
    readLetterIds: letterReads.userId === letterUserId ? letterReads.ids : loadReadLetterIds(letterUserId),
    onReadLetter,
    onReloadLetters: () => letterStore.load(),
    onDiscussLetter: discussLetter,
    notify, reset, savingProfile: profileSaving,
    profileName: session.profile?.displayName || '', conversationStyle: session.profile?.conversationStyle || 'balanced',
    onSaveSettings: saveSettings, onOpenStyle: () => navigate('approach'),
    privacySharing: privacy.enabled, privacyLoading: privacy.loading, privacySaving: privacy.saving,
    privacyLoaded: privacy.loaded, privacyError: privacy.error,
    onTogglePrivacy: () => privacyStore.toggle(), onRetryPrivacy: () => privacyStore.load(),
    profileEdit: complete && ['name', 'approach'].includes(route) && ['menu', 'settings'].includes(backOverrides.current[route]),
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
