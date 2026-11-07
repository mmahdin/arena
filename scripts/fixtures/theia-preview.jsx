import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import '@fontsource-variable/estedad';
import '@fontsource-variable/vazirmatn';
import '@fontsource-variable/noto-naskh-arabic';
import '../../src/styles.css';
import OnboardingScreen from '../../src/screens/Onboarding';
import MainScreen from '../../src/screens/Main';
import { FLOW, QUESTIONS } from '../../src/data';
import { screenSurfaceForRoute } from '../../src/features/appearance/screenSurface';
import { SAMPLE_LETTER } from '../../src/features/letters/model';

/** Design-only fixture: renders the production screens with local synthetic state. */
const MAIN = new Set(['home', 'survey', 'day', 'letter', 'history', 'saved', 'settings', 'menu', 'chat', 'insight']);
const routeFromHash = () => {
  const route = window.location.hash.replace(/^#\/?/, '').split('/')[0];
  return FLOW.includes(route) || MAIN.has(route) || ['testimonials', 'research', 'experts'].includes(route) ? route : 'welcome';
};

function Preview() {
  const [route, setRoute] = useState(routeFromHash);
  const [name, setName] = useState('دوستِ تیا');
  const [answers, setAnswers] = useState({});
  const [toast, setToast] = useState('');
  const [reads, setReads] = useState([]);
  useEffect(() => {
    const onHash = () => setRoute(routeFromHash());
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);
  useEffect(() => {
    const root = document.documentElement;
    const update = () => root.style.setProperty('--app-height', `${Math.round(window.visualViewport?.height ?? window.innerHeight)}px`);
    update();
    window.addEventListener('resize', update);
    window.visualViewport?.addEventListener('resize', update);
    return () => { window.removeEventListener('resize', update); window.visualViewport?.removeEventListener('resize', update); };
  }, []);
  useEffect(() => { if (!toast) return undefined; const t = window.setTimeout(() => setToast(''), 2200); return () => window.clearTimeout(t); }, [toast]);
  const navigate = useCallback((next) => { window.location.hash = `/${next}`; setRoute(next); }, []);
  const back = useCallback(() => {
    const index = FLOW.indexOf(route);
    navigate(index > 0 ? FLOW[index - 1] : 'home');
  }, [navigate, route]);
  const setAnswer = useCallback((key, value) => setAnswers(current => ({ ...current, [key]: value })), []);
  const notify = useCallback(setToast, []);
  const letters = useMemo(() => [SAMPLE_LETTER], []);
  const shared = {
    route, navigate, back, name, setName, answers, setAnswer, notify,
    profileName: name, conversationStyle: 'balanced', savingProfile: false,
    privacySharing: true, privacyLoaded: true, privacyLoading: false, privacySaving: false, privacyError: false,
    onTogglePrivacy: () => notify('این فقط پیش‌نمایش طراحی است.'), onRetryPrivacy: () => {},
    onSaveName: async () => true, onSaveOnboardingName: async () => true, onSaveApproach: () => navigate('settings'), onSaveSettings: async () => true,
    onOpenStyle: () => navigate('approach'), profileEdit: false,
    messages: [], conversations: [], conversationId: null, activeConversation: null,
    pendingTurn: null, sending: false, chatLoading: false, chatError: null,
    historyPhase: 'ready', historyError: null, bookmarks: [], bookmarkPhase: 'ready', bookmarkError: null, bookmarkBusyId: null,
    onSendMessage: () => notify('این فقط پیش‌نمایش طراحی است.'), onRetryMessage: () => {}, onRetryChatLoad: () => {},
    onNewConversation: () => true, onOpenConversation: () => navigate('chat'), onDeleteConversation: () => {}, onRenameConversation: async () => true,
    onReloadHistory: () => {}, onToggleBookmark: async () => true, onReloadBookmarks: () => {},
    letterPhase: 'ready', letters, readLetterIds: reads, onReadLetter: id => setReads(current => current.includes(id) ? current : [...current, id]),
    onReloadLetters: () => {}, onDiscussLetter: () => navigate('chat'),
    onLogout: () => navigate('welcome'), onSwitchTester: () => navigate('welcome'),
    onboardingGreeting: '', chatDraft: '', onDraftChange: () => {}, letterReply: null, onDismissLetterReply: () => {},
    savedMessageTarget: null, onSavedMessageTargetDone: () => {},
  };
  const isOnboarding = (FLOW.includes(route) && route !== 'home') || ['testimonials', 'research', 'experts'].includes(route);
  return (
    <div className="app-shell" data-screen-surface={screenSurfaceForRoute(route)} dir="rtl">
      <main className="screen-transition" key={route}>
        {isOnboarding ? <OnboardingScreen {...shared} /> : <MainScreen {...shared} />}
      </main>
      {toast && <div className="toast" role="status"><span>{toast}</span></div>}
    </div>
  );
}

createRoot(document.getElementById('root')).render(<React.StrictMode><Preview /></React.StrictMode>);
