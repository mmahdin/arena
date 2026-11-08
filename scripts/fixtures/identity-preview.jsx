import React, { useCallback, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import '@fontsource-variable/estedad';
import '@fontsource-variable/vazirmatn';
import '@fontsource-variable/noto-naskh-arabic';
import OnboardingScreen from '../../src/screens/Onboarding';
import MainScreen from '../../src/screens/Main';
import DevLogin from '../../src/features/devsession/DevLogin';
import { FLOW, LEGACY_INFO_ROUTES } from '../../src/data';
import { screenSurfaceForRoute } from '../../src/features/appearance/screenSurface';
import '../../src/styles.css';

/**
 * Isolated, synthetic identity preview.
 *
 * It renders the real screens with fixture data so a reviewer can walk the whole
 * Kenar visual identity — onboarding story, home, chat, letters, history, menu —
 * without a backend, a tester session, or any API call. It is design tooling,
 * never part of the shipped product.
 */

const PREVIEW_ROUTES = [
  ['welcome', 'خوش‌آمد'],
  ['privacy', 'حریم خصوصی'],
  ['simple-intro', 'میان‌پرده ۱۰'],
  ['name', 'نام'],
  ['age', 'سن'],
  ['mood', 'حال'],
  ['checkins', 'یادآوری'],
  ['deeper-intro', 'میان‌پرده ۲۲'],
  ['know-intro', 'میان‌پرده ۳۳'],
  ['almost-intro', 'میان‌پرده ۴۴'],
  ['approach', 'سبک گفت‌وگو'],
  ['voice', 'صدا'],
  ['ready', 'آماده'],
  ['home', 'خانه'],
  ['chat', 'گفت‌وگو'],
  ['letter', 'نامه‌ها'],
  ['history', 'تاریخچه'],
  ['saved', 'ذخیره‌شده'],
  ['menu', 'منو'],
  ['settings', 'تنظیمات'],
  ['survey', 'پرسش‌نامه'],
  ['devlogin', 'ورود'],
];

// Preview-only page chrome. It is injected after styles.css so it can lay the
// real mobile shell on a review stage without changing any product CSS.
const PREVIEW_CSS = `
  html, body, #root { height: auto; min-height: 100%; }
  body { overflow: auto !important; background: #1f1d19; place-items: initial; }
  .preview-bar {
    position: sticky; top: 0; z-index: 60;
    display: flex; flex-wrap: wrap; align-items: center; gap: 10px;
    padding: 10px 16px; background: #1a1815; color: #efe9df;
    border-bottom: 1px solid #33302a;
    font-family: 'Estedad Variable', Tahoma, sans-serif; font-size: 13px;
  }
  .preview-bar-title { font-weight: 600; letter-spacing: -.2px; color: #e3b79d; }
  .preview-bar select {
    flex: 1 1 190px; min-width: 150px; padding: 7px 10px; border-radius: 9px;
    border: 1px solid #43403a; background: #26231f; color: #f2ece2; font: inherit;
  }
  .preview-bar-note { color: #8f8a80; font-size: 11px; }
  .preview-stage {
    min-height: calc(100dvh - 52px); padding: 16px 12px 26px;
    display: flex; align-items: flex-start; justify-content: center;
  }
  .preview-stage .app-shell {
    height: min(880px, calc(100dvh - 84px));
    box-shadow: 0 18px 70px rgb(0 0 0 / 50%);
  }
  @media (max-width: 520px) {
    .preview-stage { padding: 0; min-height: 0; }
    .preview-stage .app-shell { height: calc(100dvh - 46px); box-shadow: none; }
  }
`;

const LETTERS = [
  {
    id: 'a1b2c3d4-0000-4000-8000-000000000001',
    title: 'دربارهٔ آن عصرِ طولانی',
    salutation: 'سلام دوست من،',
    paragraphs: [
      'امروز گفتی که حس می‌کنی بین کار و استراحت گیر کرده‌ای. چیزی که برایم روشن بود این بود که خستگی‌ات فقط از کارِ زیاد نیست؛ از این است که هنوز به خودت اجازه ندادی بنشینی.',
      'خواستم یادت بیاورم که نشستن، تنبلی نیست. همان‌قدر که کار کردن مهم است، همان‌جا ماندن و نفسی تازه کردن هم بخشی از ادامه دادن است.',
    ],
    question: 'این هفته، کدام نیم‌ساعت را می‌توانی فقط برای خودت نگه داری؟',
    createdAt: new Date(Date.now() - 86400000 * 2).toISOString(),
    kind: 'sample',
  },
  {
    id: 'a1b2c3d4-0000-4000-8000-000000000002',
    title: 'یک چیز کوچک که گفتی',
    paragraphs: ['گفتی دوست داری کمتر نگران قضاوت بقیه باشی. این را نوشتم تا فراموش نشود.'],
    createdAt: new Date(Date.now() - 86400000 * 6).toISOString(),
    kind: 'sample',
  },
];

const MESSAGES = [
  { id: 'm1', role: 'user', content: 'این روزها حس می‌کنم هیچ‌کس واقعاً حالم رو نمی‌فهمه.' },
  {
    id: 'm2',
    role: 'assistant',
    content: 'ممنون که گفتی.\n\nاینکه احساس کنی دیده نمی‌شی، خسته‌کننده‌ست — مخصوصاً وقتی خودت برای بقیه وقت می‌گذاری.\n\nدوست داری از کجای این حس بگی؟',
  },
  { id: 'm3', role: 'user', content: 'همیشه منم که پیام می‌دم. یک بار صبر کردم، هیچ‌کس نپرسید.' },
  {
    id: 'm4',
    role: 'assistant',
    content: 'پس یک بار هم سکوت کردی و همان سکوت جواب داد که چقدر تنهایی.\n\nاین را که گفتی مهم است. الان بیشتر دلت می‌خواهد کسی سراغت را بگیرد، یا بفهمی چطور می‌شود این را بی‌آنکه دعوا شود گفت؟',
  },
];

const CONVERSATIONS = [
  { id: 'c1', title: 'دربارهٔ دلتنگی', updatedAt: new Date(Date.now() - 3600000).toISOString() },
  { id: 'c2', title: 'کار و استراحت', updatedAt: new Date(Date.now() - 86400000).toISOString() },
  { id: 'c3', title: 'گفت‌وگوی بدون عنوان', updatedAt: new Date(Date.now() - 86400000 * 4).toISOString() },
];

function Preview() {
  const params = new URLSearchParams(window.location.search);
  const initial = params.get('route');
  const [route, setRoute] = useState(PREVIEW_ROUTES.some(([key]) => key === initial) ? initial : 'welcome');
  const [answers, setAnswers] = useState({ mood: 3, approach: 'همدلی و شنیدن' });
  const [name, setName] = useState('نگار');
  const [letterReply, setLetterReply] = useState(null);
  const [toast, setToast] = useState('');
  const [pending, setPending] = useState(false);

  const navigate = useCallback((next) => {
    setRoute(next);
    window.history.replaceState(null, '', `?route=${next}`);
  }, []);

  const notify = useCallback((message) => {
    setToast(message);
    window.setTimeout(() => setToast(''), 2400);
  }, []);
  const setAnswer = useCallback((key, value) => setAnswers((current) => ({ ...current, [key]: value })), []);

  // The preview never sends a turn; it shows the composer's real pending state.
  const sendMessage = () => {
    setPending(true);
    notify('پیش‌نمایش: پیامی به سرور فرستاده نشد.');
    window.setTimeout(() => setPending(false), 1400);
    return true;
  };

  const shared = useMemo(() => ({
    route, navigate, back: () => navigate(FLOW[Math.max(0, FLOW.indexOf(route) - 1)] || 'welcome'),
    name, setName, answers, setAnswer, notify,
    onboardingGreeting: '',
    messages: MESSAGES, conversations: CONVERSATIONS, conversationId: 'c1', activeConversation: CONVERSATIONS[0],
    pendingTurn: pending ? { text: 'یک پیام آزمایشی' } : null, sending: pending,
    chatLoading: false, chatError: '', historyPhase: 'ready', historyError: '', deletingId: null, renamingId: null,
    chatDraft: '', onDraftChange: () => {}, letterReply, onDismissLetterReply: () => setLetterReply(null),
    bookmarks: [{ id: 'm2', conversationId: 'c1', content: 'اینکه احساس کنی دیده نمی‌شی، خسته‌کننده‌ست.' }],
    bookmarkPhase: 'ready', bookmarkError: '', bookmarkBusyId: null,
    onToggleBookmark: () => true, onReloadBookmarks: () => {},
    onSendMessage: sendMessage, onRetryMessage: () => {}, onRetryChatLoad: () => {},
    onNewConversation: () => { navigate('chat'); return true; },
    onOpenConversation: (id) => { notify(`پیش‌نمایش: گفت‌وگوی ${id}`); return true; },
    onOpenSavedMessage: () => true, savedMessageTarget: null, onSavedMessageTargetDone: () => {},
    onDeleteConversation: () => {}, onRenameConversation: () => true, onReloadHistory: () => {},
    letterPhase: 'ready', letters: LETTERS, letterError: null, readLetterIds: [LETTERS[1].id],
    onReadLetter: () => {}, onReloadLetters: () => {},
    onDiscussLetter: (letter) => { setLetterReply(letter); navigate('chat'); return true; },
    reset: () => {},
    savingProfile: false, profileName: name, conversationStyle: 'balanced',
    onSaveSettings: () => true, onOpenStyle: () => navigate('approach'),
    privacySharing: false, privacyLoading: false, privacySaving: false, privacyLoaded: true, privacyError: false,
    onTogglePrivacy: () => {}, onRetryPrivacy: () => {},
    profileEdit: false, onSaveName: () => true, onSaveOnboardingName: () => { navigate('age'); },
    onSaveApproach: () => {}, onLogout: () => notify('پیش‌نمایش: خروج'), onSwitchTester: () => notify('پیش‌نمایش: تعویض کاربر'),
  }), [route, name, answers, letterReply, pending, navigate, notify, setAnswer]);

  const isOnboarding = (FLOW.includes(route) && route !== 'home') || LEGACY_INFO_ROUTES.includes(route);
  const surface = screenSurfaceForRoute(route);

  return (
    <>
      <style>{PREVIEW_CSS}</style>
      <div className="preview-bar" role="region" aria-label="پیش‌نمایش طراحی">
        <span className="preview-bar-title">پیش‌نمایش هویت کنار</span>
        <select value={route} onChange={(event) => navigate(event.target.value)} aria-label="انتخاب صفحه">
          {PREVIEW_ROUTES.map(([key, title]) => <option key={key} value={key}>{title} — {key}</option>)}
        </select>
        <span className="preview-bar-note">دادهٔ ساختگی · بدون سرور</span>
      </div>
      <div className="preview-stage">
        <div className="app-shell" data-screen-surface={surface} dir="rtl">
          <main className="screen-transition" key={route}>
            {route === 'devlogin'
              ? <DevLogin session={{ phase: 'idle' }} login={() => {}} retry={() => {}} />
              : isOnboarding ? <OnboardingScreen {...shared} /> : <MainScreen {...shared} />}
          </main>
          {toast && <div className="toast" role="status">{toast}</div>}
        </div>
      </div>
    </>
  );
}

createRoot(document.getElementById('root')).render(<React.StrictMode><Preview /></React.StrictMode>);
