import React, { useEffect, useLayoutEffect, useRef, useState } from 'react';
import {
  ArrowDown,
  ArrowUp,
  Bookmark,
  Check,
  ChevronDown,
  ChevronLeft,
  Copy,
  History as HistoryIcon,
  Info,
  LogOut,
  Mail,
  Menu as MenuIcon,
  Pencil,
  Plus,
  Settings2,
  Volume2,
  X,
} from 'lucide-react';
import { KenarMark, KenarSeal } from '../components/UI';
import {
  BackButton,
  BottomSheet,
  MiniEnvelope,
  StatusBar,
} from '../components/UI';
import Letters from './Letters';
import { parseLetterReplyMessage } from '../features/letters/discuss';
import { validateDisplayName } from '../features/profile/preferences';

const formatDate = (value) => {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? 'تاریخ نامشخص' : new Intl.DateTimeFormat('fa-IR', { month: 'short', day: 'numeric' }).format(date);
};

function Home({ navigate, name, underlay = false, onNewConversation, letterPhase = 'idle', letters = [], readLetterIds = [] }) {
  const hasUnreadLetter = letterPhase === 'ready' && letters.some(letter => !readLetterIds.includes(letter.id));
  const startConversation = () => {
    // Do not leave an in-flight turn behind if the user declines the store's guard.
    if (onNewConversation?.()) navigate('chat');
  };

  return (
    <section className={`app-screen theme-dark home-screen ${underlay ? 'is-underlay' : ''}`} dir="rtl">
      <StatusBar dark />
      <button type="button" className="home-menu-button" onClick={() => navigate('menu')} aria-label="بازکردن منو">
        <MenuIcon size={23} strokeWidth={1.8} aria-hidden="true" />
      </button>

      <div className="home-invitation">
        <button type="button" className="home-start-conversation" onClick={startConversation} aria-label="شروع گفت‌وگوی تازه">
          <span className="home-conversation-art" aria-hidden="true">
            <svg className="home-conversation-orbit" viewBox="0 0 240 240" fill="none" focusable="false">
              <path className="home-orbit-trace home-orbit-trace--first" d="M187 48C157 19 113 21 81 38C39 61 21 105 35 148C45 182 73 205 107 211" />
              <path className="home-orbit-trace home-orbit-trace--second" d="M58 177C83 205 127 219 163 202C200 185 222 151 211 109C205 88 195 75 184 65" />
              <path className="home-orbit-accent" d="M60 76C76 58 97 52 112 51" />
              <circle className="home-orbit-point" cx="111" cy="211" r="2.6" />
            </svg>
            <KenarMark className="home-conversation-mark" size={52} strokeWidth={1.05} aria-hidden="true" />
          </span>
          <span className="home-start-label">بیا حرف بزنیم</span>
        </button>
      </div>

      <button
        type="button"
        className={`home-letter-link ${hasUnreadLetter ? 'is-unread' : ''}`}
        onClick={() => navigate('letter')}
        aria-label={hasUnreadLetter ? 'خواندن نامهٔ تازهٔ کنار' : 'بازکردن نامه‌های کنار'}
        title={hasUnreadLetter ? 'نامهٔ تازه' : 'نامه‌های کنار'}
      >
        <span className="home-letter-symbol" aria-hidden="true">
          <MiniEnvelope />
          {hasUnreadLetter && <span className="home-letter-seal"><KenarSeal size={10} strokeWidth={1.4} /></span>}
        </span>
      </button>

      <nav className="home-navigation" aria-label="گفت‌وگوها">
        <button type="button" className="home-history-link" onClick={() => navigate('history')} aria-label="نمایش گفت‌وگوهای قبلی">
          <span className="home-history-stack" aria-hidden="true">
            <span className="home-history-sheet home-history-sheet--back" />
            <span className="home-history-sheet home-history-sheet--front">
              <span className="home-history-stroke" />
              <span className="home-history-stroke home-history-stroke--short" />
            </span>
          </span>
          <span className="home-history-caption">
            <span>گفت‌وگوهای قبلی</span>
            <ChevronLeft size={15} strokeWidth={1.6} aria-hidden="true" />
          </span>
        </button>
      </nav>
    </section>
  );
}

function Survey({ back, answers, setAnswer, navigate, notify }) {
  const options = ['اصلاً', 'چند روز', 'بیشتر از نصف روزها', 'تقریباً هر روز'];
  const advanceTimer = useRef(null);
  useEffect(() => () => window.clearTimeout(advanceTimer.current), []);
  const choose = (option) => {
    setAnswer('healthSurvey', option);
    notify('انتخاب شد.');
    window.clearTimeout(advanceTimer.current);
    advanceTimer.current = window.setTimeout(() => navigate('home'), 350);
  };
  return (
    <section className="app-screen theme-dark survey-screen">
      <StatusBar dark />
      <BackButton dark onClick={back} />
      <div className="survey-copy">
        <p className="survey-context" id="survey-context">توی دو هفتهٔ گذشته، چند وقت یه بار این حس رو داشتی؟</p>
        <h1 id="survey-question">کم شدن علاقه یا لذتت از کارهایی که قبلاً دوست داشتی</h1>
        <p className="survey-support">پاسخ درست یا غلط وجود نداره؛ نزدیک‌ترین گزینه رو انتخاب کن.</p>
      </div>
      <fieldset className="survey-options" aria-labelledby="survey-question" aria-describedby="survey-context">
        <legend className="sr-only">گزینه‌های پاسخ</legend>
        {options.map((option) => {
          const selected = answers.healthSurvey === option;
          return (
            <label key={option} className={`survey-option ${selected ? 'is-selected' : ''}`}>
              <input
                type="radio"
                name="healthSurvey"
                value={option}
                checked={selected}
                onChange={() => choose(option)}
              />
              <span>{option}</span>
              <Check size={18} className="survey-check" aria-hidden="true" />
            </label>
          );
        })}
      </fieldset>
    </section>
  );
}

function History({ back, navigate, conversations = [], conversationId, historyPhase, historyError, onReloadHistory, onOpenConversation, onDeleteConversation, onRenameConversation, onNewConversation, deletingId, renamingId }) {
  const [editingId, setEditingId] = useState(null);
  const [draftTitle, setDraftTitle] = useState('');
  const [renameError, setRenameError] = useState(false);
  const beginRename = (item) => { setEditingId(item.id); setDraftTitle(item.title); setRenameError(false); };
  const finishRename = async (event, id) => {
    event.preventDefault();
    if (!draftTitle.trim() || [...draftTitle.trim()].length > 60) { setRenameError(true); return; }
    const success = await onRenameConversation(id, draftTitle);
    if (success) { setEditingId(null); setRenameError(false); }
    else setRenameError(true);
  };
  const startNew = () => {
    if (onNewConversation?.()) navigate('chat');
  };
  return (
    <section className="app-screen theme-dark history-screen">
      <StatusBar dark />
      <BackButton dark onClick={back} />
      <h1 className="history-title">گفت‌وگوهای قبلی</h1>
      <button type="button" className="history-new" onClick={startNew} aria-label="گفت‌وگوی جدید">
        <Plus size={20} aria-hidden="true" />
      </button>
      <div className="history-content" aria-live="polite">
        {historyPhase === 'loading' && <p className="connected-status">در حال بارگذاری گفت‌وگوها…</p>}
        {historyError && <div className="connected-notice" role="alert">
          <p>گفت‌وگو بارگذاری یا حذف نشد. دوباره تلاش کن.</p>
          <button type="button" onClick={onReloadHistory}>تلاش دوباره</button>
        </div>}
        {historyPhase === 'ready' && conversations.length === 0 && <p className="connected-status">هنوز گفت‌وگویی نداری.</p>}
        {conversations.map((item) => (
          <div className={`connected-history-entry ${item.id === conversationId ? 'is-active' : ''}`} key={item.id}>
            {editingId === item.id ? (
              <form className="conversation-title-editor" onSubmit={(event) => finishRename(event, item.id)}>
                <label htmlFor={`rename-${item.id}`}>تغییر نام گفت‌وگو</label>
                <input id={`rename-${item.id}`} autoFocus value={draftTitle} maxLength={60} onChange={event => { setDraftTitle(event.target.value); setRenameError(false); }} disabled={renamingId === item.id} />
                {renameError && <small role="alert">نام تغییر نکرد. دوباره امتحان کن.</small>}
                <div className="conversation-title-editor-actions">
                  <button type="submit" disabled={renamingId === item.id}>{renamingId === item.id ? 'در حال ذخیره…' : 'ذخیره نام'}</button>
                  <button type="button" disabled={renamingId === item.id} onClick={() => { setEditingId(null); setRenameError(false); }}>انصراف</button>
                </div>
              </form>
            ) : (
              <>
                <button type="button" className="history-row" onClick={() => onOpenConversation(item.id)} aria-current={item.id === conversationId ? 'true' : undefined}>
                  <span className="history-date"><strong>{item.title || 'گفت‌وگوی بدون عنوان'}</strong><small>{formatDate(item.updatedAt)}</small></span>
                </button>
                <button type="button" className="connected-rename" disabled={Boolean(renamingId) || deletingId === item.id} onClick={() => beginRename(item)} aria-label={`تغییر نام ${item.title || 'گفت‌وگو'}`}>
                  <Pencil size={17} aria-hidden="true" />
                </button>
                <button type="button" className="connected-delete" disabled={deletingId === item.id || Boolean(renamingId)} onClick={() => onDeleteConversation(item.id)} aria-label={`حذف ${item.title || 'گفت‌وگو'}`}>
                  {deletingId === item.id ? 'در حال حذف…' : 'حذف'}
                </button>
              </>
            )}
          </div>
        ))}
      </div>
    </section>
  );
}

function SavedMessages({ back, bookmarks = [], bookmarkPhase, bookmarkError, onReloadBookmarks, onOpenSavedMessage }) {
  return (
    <section className="app-screen theme-dark history-screen saved-messages-screen" dir="rtl">
      <StatusBar dark />
      <BackButton dark onClick={back} />
      <h1 className="history-title">پیام‌های ذخیره‌شده</h1>
      <div className="history-content saved-messages-list" aria-live="polite">
        {bookmarkPhase === 'loading' && <p className="connected-status">در حال بارگذاری پیام‌ها…</p>}
        {bookmarkPhase === 'error' && <div className="connected-notice" role="alert">
          <p>پیام‌های ذخیره‌شده بارگذاری نشدن.</p>
          <button type="button" onClick={onReloadBookmarks}>تلاش دوباره</button>
        </div>}
        {bookmarkPhase === 'ready' && bookmarks.length === 0 && <p className="connected-status">هنوز پیامی ذخیره نکردی.</p>}
        {bookmarks.map(item => (
          <button type="button" key={item.id} className="saved-history-entry" onClick={() => onOpenSavedMessage(item)}>
            <Bookmark size={18} fill="currentColor" aria-hidden="true" />
            <span>{item.content.length > 125 ? `${item.content.slice(0, 125)}…` : item.content}</span>
          </button>
        ))}
      </div>
    </section>
  );
}

const MENU_ITEMS = [
  { label: 'گفت‌وگوهای قبلی', icon: HistoryIcon, route: 'history' },
  { label: 'پیام‌های ذخیره‌شده', icon: Bookmark, route: 'saved' },
  { label: 'نامه‌های کنار', icon: Mail, route: 'letter' },
  { label: 'تنظیمات', icon: Settings2, route: 'settings' },
  { label: 'درباره', icon: Info, route: 'research' },
];

function Drawer({ navigate, name, onLogout, onSwitchTester }) {
  return (
    <div className="drawer-screen">
      <Home navigate={() => {}} name={name} underlay />
      <button type="button" className="drawer-backdrop" onClick={() => navigate('home')} aria-label="بستن منو" />
      <aside className="drawer-panel" aria-label="منوی اصلی">
        <div className="drawer-brand">کنار</div>
        <nav className="drawer-nav">
          {MENU_ITEMS.map(({ label, icon: Icon, route }) => (
            <button type="button" key={route} onClick={() => navigate(route)}>
              <Icon size={24} strokeWidth={1.65} aria-hidden="true" />
              <span>{label}</span>
            </button>
          ))}
        </nav>
        <div className="drawer-footer">
          <button type="button" aria-label="تعویض کاربر آزمایشی" onClick={() => onSwitchTester?.()}><Settings2 size={23} /><span>عوض کردن حساب</span></button>
          <button type="button" onClick={() => onLogout?.()}><LogOut size={23} /><span>خروج</span></button>
        </div>
      </aside>
    </div>
  );
}

/** Edit profile and privacy here; the example-rich style selector is reused standalone. */
function Settings({ back, profileName = '', conversationStyle = 'balanced', savingProfile = false,
  onSaveSettings, onOpenStyle, privacySharing, privacyLoading, privacySaving, privacyLoaded, privacyError,
  onTogglePrivacy, onRetryPrivacy }) {
  const [draftName, setDraftName] = useState(profileName);
  const [feedback, setFeedback] = useState('');
  const nameValid = Boolean(validateDisplayName(draftName));
  const changed = draftName.trim() !== profileName;
  const pending = savingProfile || privacySaving;

  const submit = async (event) => {
    event.preventDefault();
    setFeedback('');
    if (!nameValid || !changed || pending) return;
    const ok = await onSaveSettings?.(draftName.trim(), conversationStyle);
    setFeedback(ok ? 'تغییرات ذخیره شد.' : 'ذخیره نشد. دوباره تلاش کن.');
  };
  return (
    <section className="app-screen theme-dark settings-screen" dir="rtl">
      <StatusBar dark />
      <BackButton dark onClick={back} />
      <h1 className="settings-page-title">تنظیمات</h1>
      <div className="settings-scroll">
        <form id="settings-profile-form" onSubmit={submit}>
          <div className="settings-group">
            <label className="settings-label" htmlFor="settings-name">اسم من</label>
            <div className="settings-name-row">
              <input
                id="settings-name" className="settings-name-input" type="text" dir="auto"
                autoComplete="nickname" maxLength={60} value={draftName}
                disabled={savingProfile}
                onChange={(event) => { setDraftName(event.target.value); setFeedback(''); }}
                aria-invalid={!nameValid}
                aria-describedby={!nameValid ? 'settings-name-error' : undefined}
              />
              <button type="submit" className="settings-save" disabled={!changed || !nameValid || pending}
                aria-label={savingProfile ? 'در حال ذخیره نام' : 'ذخیره نام'}>
                {savingProfile ? '…' : 'ذخیره'}
              </button>
            </div>
            {!nameValid && <p className="settings-error" id="settings-name-error">اسمت باید بین ۱ تا ۶۰ کاراکتر باشه.</p>}
            {feedback && <p className="settings-feedback" role="status" aria-live="polite">{feedback}</p>}
          </div>
        </form>
        <div className="settings-group settings-style-group">
          <button type="button" className="settings-style-link" onClick={onOpenStyle} disabled={savingProfile}
            aria-label="تغییر سبک گفت‌وگو و دیدن نمونه‌های هر سبک">
            <span>تغییر سبک گفت‌وگو</span>
            <ChevronLeft size={19} strokeWidth={1.7} aria-hidden="true" />
          </button>
        </div>
        <section className="settings-group settings-privacy" aria-labelledby="settings-privacy-heading">
          <h2 className="settings-label" id="settings-privacy-heading">حریم خصوصی</h2>
          <button type="button" className="settings-privacy-toggle" role="switch" aria-checked={Boolean(privacySharing)}
            aria-disabled={!privacyLoaded || privacyLoading || privacySaving}
            disabled={!privacyLoaded || privacyLoading || privacySaving} onClick={onTogglePrivacy}>
            <span>اجازهٔ استفاده از گفت‌وگوها</span>
            <span className={`settings-switch ${privacySharing ? 'is-on' : ''}`} aria-hidden="true"><span /></span>
          </button>
          {privacyLoading && <p className="settings-muted" role="status">در حال بارگذاری…</p>}
          {privacyError && <div className="settings-privacy-error" role="alert">
            <span>تنظیمات حریم خصوصی در دسترس نیست.</span>
            <button type="button" onClick={onRetryPrivacy}>تلاش دوباره</button>
          </div>}
        </section>
      </div>
    </section>
  );
}

function MessageTools({ message, open, saved, saving, bookmarkReady, onToggleBookmark }) {
  const assistant = message.role === 'assistant';
  const [copied, setCopied] = useState(false);
  const [speaking, setSpeaking] = useState(false);
  const [flash, setFlash] = useState('');
  const copyTimer = useRef(0);
  const flashTimer = useRef(0);
  const utterance = useRef(null);

  const say = (text) => {
    window.clearTimeout(flashTimer.current);
    setFlash(text);
    flashTimer.current = window.setTimeout(() => setFlash(''), 2200);
  };
  useEffect(() => () => {
    window.clearTimeout(copyTimer.current);
    window.clearTimeout(flashTimer.current);
    if (utterance.current) window.speechSynthesis?.cancel();
  }, []);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(message.content);
      setCopied(true);
      window.clearTimeout(copyTimer.current);
      copyTimer.current = window.setTimeout(() => setCopied(false), 1600);
    } catch {
      say('کپی نشد');
    }
  };
  const speak = () => {
    const synth = window.speechSynthesis;
    if (!synth || !('SpeechSynthesisUtterance' in window)) { say('پخش صدا ممکن نیست'); return; }
    if (speaking) { utterance.current = null; setSpeaking(false); synth.cancel(); return; }
    synth.cancel();
    const next = new SpeechSynthesisUtterance(message.content);
    next.lang = 'fa-IR';
    const finish = () => { if (utterance.current === next) { utterance.current = null; setSpeaking(false); } };
    next.onend = finish;
    next.onerror = finish;
    utterance.current = next;
    setSpeaking(true);
    synth.speak(next);
  };
  const toggleSave = async () => {
    const wasSaved = saved;
    const ok = await onToggleBookmark(message);
    if (!ok) say(wasSaved ? 'حذف نشد' : 'ذخیره نشد');
  };

  return (
    <div className={`message-tools ${open ? 'is-open' : ''}`}>
      <div className="message-tools-inner" role="group" aria-label="گزینه‌های پیام">
        <button type="button" className={`message-tool ${copied ? 'is-done' : ''}`} onClick={copy} aria-label="کپی پیام" title="کپی پیام">
          {copied ? <Check size={17} aria-hidden="true" /> : <Copy size={17} aria-hidden="true" />}
        </button>
        <button type="button" className={`message-tool message-tool--save ${saved ? 'is-saved' : ''} ${saving ? 'is-busy' : ''}`}
          onClick={toggleSave} disabled={!bookmarkReady || saving}
          aria-label={saved ? 'حذف از ذخیره‌شده‌ها' : 'ذخیره پیام'} title={saved ? 'حذف از ذخیره‌شده‌ها' : 'ذخیره پیام'}>
          <Bookmark size={17} fill={saved ? 'currentColor' : 'none'} aria-hidden="true" />
        </button>
        <button type="button" className={`message-tool ${speaking ? 'is-speaking' : ''}`} onClick={speak}
          aria-label={speaking ? 'توقف خواندن' : 'خواندن با صدای دستگاه'} title={speaking ? 'توقف خواندن' : 'خواندن با صدای دستگاه'}>
          <Volume2 size={17} aria-hidden="true" />
        </button>
        {flash && <span className="message-flash" role="status">{flash}</span>}
      </div>
    </div>
  );
}

function ConversationSheet({ open, onClose, conversations = [], conversationId, historyPhase, historyError, onReloadHistory, onOpenConversation, onNewConversation, navigate, sending }) {
  const startNew = () => {
    if (onNewConversation?.()) {
      onClose();
      navigate('chat');
    }
  };
  const openItem = (id) => {
    if (onOpenConversation(id)) onClose();
  };
  return (
    <BottomSheet open={open} onClose={onClose} title="گفت‌وگوها" className="conversation-sheet">
      <button type="button" className="conversation-new" onClick={startNew} disabled={sending}>
        <span className="conversation-new-icon"><Plus size={18} aria-hidden="true" /></span>
        <strong>گفت‌وگوی جدید</strong>
      </button>
      <div className="conversation-list" aria-live="polite">
        {historyPhase === 'loading' && <p className="connected-status">در حال بارگذاری گفت‌وگوها…</p>}
        {historyError && <div className="connected-notice" role="alert">
          <p>گفت‌وگوها بارگذاری نشدن.</p>
          <button type="button" onClick={onReloadHistory}>تلاش دوباره</button>
        </div>}
        {historyPhase === 'ready' && conversations.length === 0 && <p className="connected-status">هنوز گفت‌وگویی نداری.</p>}
        {conversations.map((item) => (
          <button type="button" key={item.id} className={`conversation-item ${item.id === conversationId ? 'is-active' : ''}`} onClick={() => openItem(item.id)} aria-current={item.id === conversationId ? 'true' : undefined}>
            <span className="conversation-item-copy">
              <strong>{item.title || 'گفت‌وگوی بدون عنوان'}</strong>
              <small>{formatDate(item.updatedAt)}</small>
            </span>
            {item.id === conversationId && <i className="conversation-item-dot" aria-hidden="true" />}
          </button>
        ))}
      </div>
      <button type="button" className="conversation-manage" onClick={() => { onClose(); navigate('history'); }}>
        مدیریت گفت‌وگوها
      </button>
    </BottomSheet>
  );
}

function ChatSkeleton() {
  return (
    <div className="chat-loading" role="status" aria-label="در حال بارگذاری پیام‌ها">
      <span className="sr-only">در حال بارگذاری پیام‌ها…</span>
      <div className="chat-skeleton chat-skeleton--user" aria-hidden="true"><i /><i /></div>
      <div className="chat-skeleton chat-skeleton--assistant" aria-hidden="true"><i /><i /><i /></div>
      <div className="chat-skeleton chat-skeleton--assistant" aria-hidden="true"><i /><i /><i /><i /></div>
    </div>
  );
}

function LetterReference({ title, dismiss }) {
  return (
    <div className={`chat-letter-reference ${dismiss ? 'is-composer-reference' : 'is-message-reference'}`} role="note" aria-label={`ارجاع به نامهٔ ${title}`}>
      <Mail size={15} strokeWidth={1.7} aria-hidden="true" />
      <span className="chat-letter-reference-title" dir="auto">{title}</span>
      {dismiss && <button type="button" className="chat-letter-reference-close" onClick={dismiss} aria-label="حذف ارجاع به نامه">
        <X size={15} strokeWidth={1.7} aria-hidden="true" />
      </button>}
    </div>
  );
}

function Chat({
  navigate, onboardingGreeting = '', messages = [], pendingTurn, sending, chatLoading, chatError,
  onSendMessage, onRetryMessage, onRetryChatLoad, onNewConversation, notify,
  bookmarks = [], bookmarkPhase, bookmarkError, bookmarkBusyId, onToggleBookmark, onReloadBookmarks,
  conversations = [], conversationId, activeConversation, historyPhase, historyError, onReloadHistory,
  onOpenConversation, savedMessageTarget, onSavedMessageTargetDone, chatDraft = '', onDraftChange,
  letterReply, onDismissLetterReply,
}) {
  // undefined → tools follow the latest assistant message; null → all closed; id → that message.
  const [toolsFor, setToolsFor] = useState(undefined);
  const [sheetOpen, setSheetOpen] = useState(false);
  const [showJump, setShowJump] = useState(false);
  const scrollRef = useRef(null);
  const inputRef = useRef(null);
  const composerRef = useRef(null);
  const stickToBottom = useRef(true);
  const knownIds = useRef(null);
  const freshIds = useRef(new Set());
  const draft = chatDraft;
  const setDraft = (value) => onDraftChange?.(value);

  const resizeInput = () => {
    const input = inputRef.current;
    if (!input) return;
    input.style.height = '0px';
    input.style.height = `${Math.min(input.scrollHeight, 140)}px`;
    input.style.overflowY = input.scrollHeight > 140 ? 'auto' : 'hidden';
  };
  const scrollToBottom = (behavior = 'smooth') => {
    const node = scrollRef.current;
    if (!node) return;
    const reduce = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
    node.scrollTo({ top: node.scrollHeight, behavior: reduce ? 'auto' : behavior });
  };
  useLayoutEffect(resizeInput, [draft]);
  useLayoutEffect(() => {
    const composer = composerRef.current;
    if (!composer) return undefined;
    const observer = new ResizeObserver(() => {
      composer.closest('.chat-screen')?.style.setProperty('--composer-height', `${composer.offsetHeight}px`);
    });
    observer.observe(composer);
    window.addEventListener('resize', resizeInput);
    return () => { observer.disconnect(); window.removeEventListener('resize', resizeInput); };
  }, []);
  useEffect(() => {
    // A bookmark jump must win over the usual scroll-to-latest behavior.
    stickToBottom.current = savedMessageTarget?.conversationId !== conversationId;
    setShowJump(false);
    setToolsFor(undefined);
    knownIds.current = null;
    freshIds.current = new Set();
  }, [conversationId]);
  useEffect(() => {
    if (savedMessageTarget?.conversationId === conversationId) return;
    if (!chatLoading && stickToBottom.current) scrollToBottom(messages.length ? 'smooth' : 'auto');
  }, [messages, pendingTurn, sending, chatLoading]);
  useLayoutEffect(() => {
    if (!savedMessageTarget || savedMessageTarget.conversationId !== conversationId || chatLoading || chatError) return undefined;
    const scroller = scrollRef.current;
    const message = scroller?.querySelector(`[data-message-id="${savedMessageTarget.messageId}"]`);
    if (!message) {
      onSavedMessageTargetDone?.(savedMessageTarget);
      notify('این پیام در گفت‌وگو پیدا نشد.');
      return undefined;
    }

    // Scroll the chat container itself, never the entire screen; center the
    // message when it fits, and start at its top for very long messages.
    stickToBottom.current = false;
    const containerRect = scroller.getBoundingClientRect();
    const messageRect = message.getBoundingClientRect();
    const offset = Math.max(0, (scroller.clientHeight - messageRect.height) / 2);
    const top = scroller.scrollTop + messageRect.top - containerRect.top - offset;
    const reduceMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
    scroller.scrollTo({ top, behavior: reduceMotion ? 'auto' : 'smooth' });
    message.focus({ preventScroll: true });
    const timer = window.setTimeout(() => onSavedMessageTargetDone?.(savedMessageTarget), 2800);
    return () => window.clearTimeout(timer);
  }, [savedMessageTarget, conversationId, chatLoading, chatError, messages, onSavedMessageTargetDone, notify]);
  useEffect(() => { setToolsFor(undefined); }, [messages.length]);
  // Only messages that arrive after the thread is on screen get the entrance motion.
  useEffect(() => {
    knownIds.current = chatLoading ? null : new Set(messages.map((message) => message.id));
  }, [messages, chatLoading]);

  const onScroll = () => {
    const node = scrollRef.current;
    if (!node) return;
    const distance = node.scrollHeight - node.scrollTop - node.clientHeight;
    stickToBottom.current = distance < 120;
    setShowJump(distance > 220);
  };
  const send = (text) => {
    const clean = text.trim();
    if (!clean || sending || pendingTurn || chatLoading) return;
    if ([...clean].length > 4000) { notify('پیام می‌تونه تا ۴۰۰۰ کاراکتر باشه.'); return; }
    // Chat store clears the draft only after accepting the turn; retain it on validation errors.
    void onSendMessage(clean);
  };
  const onComposerKeyDown = (event) => {
    // Enter sends; Shift+Enter keeps the native new line. Never intercept IME composition.
    if (event.key !== 'Enter' || event.shiftKey || event.altKey || event.nativeEvent.isComposing) return;
    event.preventDefault();
    send(draft);
  };

  const lastAssistantId = [...messages].reverse().find((message) => message.role === 'assistant')?.id ?? null;
  const openToolsId = toolsFor === undefined ? lastAssistantId : toolsFor;
  const toggleTools = (id) => {
    if (window.getSelection?.()?.toString()) return; // selecting text must not toggle anything
    setToolsFor(openToolsId === id ? null : id);
  };
  const title = activeConversation?.title || (conversationId ? 'گفت‌وگو' : 'گفت‌وگوی تازه');
  const sendDisabled = !draft.trim() || sending || Boolean(pendingTurn) || chatLoading;

  return (
    <section className="app-screen theme-dark chat-screen">
      <StatusBar dark />
      <header className="chat-header">
        <BackButton dark onClick={() => navigate('home')} />
        <button type="button" className="chat-title-button" onClick={() => setSheetOpen(true)} aria-haspopup="dialog" aria-expanded={sheetOpen} aria-label={`${title}، گفت‌وگوها`}>
          <span className="chat-title">{title}</span>
          <ChevronDown size={16} aria-hidden="true" />
        </button>
      </header>
      <div className="messages-scroll" ref={scrollRef} onScroll={onScroll} role="log" aria-live="polite" aria-busy={chatLoading}>
        <div className="messages-list">
          {bookmarkPhase === 'error' && <div className="connected-notice" role="alert">
            <p>وضعیت پیام‌های ذخیره‌شده در دسترس نیست.</p>
            <button type="button" onClick={onReloadBookmarks}>تلاش دوباره</button>
          </div>}
          {chatLoading && <ChatSkeleton />}
          {!chatLoading && messages.length === 0 && !pendingTurn && !chatError && (onboardingGreeting ? (
            <div className="message-block message-assistant onboarding-welcome">
              <div className="message-text">
                {onboardingGreeting.split(/\n\s*\n/).map((paragraph, index) => <p key={index}>{paragraph}</p>)}
              </div>
            </div>
          ) : (
            <div className="chat-empty">
              <KenarMark size={30} strokeWidth={1.3} aria-hidden="true" />
              <h2>اینجا می‌تونی راحت حرف بزنی.</h2>
            </div>
          ))}
          {messages.map((message) => {
            const assistant = message.role === 'assistant';
            const shown = assistant ? { text: message.content, reference: null } : parseLetterReplyMessage(message.content);
            if (assistant && knownIds.current && !knownIds.current.has(message.id)) freshIds.current.add(message.id);
            const open = openToolsId === message.id;
            const saved = bookmarks.some((item) => item.id === message.id);
            return (
              <div key={message.id} data-message-id={message.id}
                tabIndex={savedMessageTarget?.conversationId === conversationId && savedMessageTarget?.messageId === message.id ? -1 : undefined}
                className={`message-block message-${message.role} ${assistant && freshIds.current.has(message.id) ? 'is-fresh' : ''} ${open ? 'is-open' : ''} ${savedMessageTarget?.conversationId === conversationId && savedMessageTarget?.messageId === message.id ? 'is-targeted' : ''}`}>
                <div className="message-text"
                  onClick={assistant ? () => toggleTools(message.id) : undefined}>
                  {shown.reference && <LetterReference title={shown.reference.title} />}
                  {shown.text.split(/\n\s*\n/).map((paragraph, pIndex) => <p key={pIndex}>{paragraph}</p>)}
                </div>
                {assistant && <MessageTools message={message} open={open} saved={saved}
                  saving={bookmarkBusyId === message.id} bookmarkReady={bookmarkPhase === 'ready'} onToggleBookmark={onToggleBookmark} />}
              </div>
            );
          })}
          {pendingTurn && <div className={`message-block message-user connected-pending ${sending ? 'is-sending' : ''}`}>
            <div className="message-text">
              {parseLetterReplyMessage(pendingTurn.text).reference && <LetterReference title={parseLetterReplyMessage(pendingTurn.text).reference.title} />}
              <p>{parseLetterReplyMessage(pendingTurn.text).text}</p>
            </div>
            {!sending && <small>پیامت ارسال نشده.</small>}
          </div>}
          {sending && <div className="message-block message-assistant">
            <div className="typing-indicator" role="status" aria-label="دارم جواب می‌دم…"><i /><i /><i /></div>
          </div>}
          {chatError && <div className="connected-notice" role="alert">
            <p>{chatError === 'TURN_ALREADY_PROCESSING' ? 'پاسخ قبلیت هنوز آماده نشده. دوباره تلاش کن.' : chatError === 'RESOURCE_NOT_FOUND' ? 'این گفت‌وگو در دسترس نیست.' : 'پاسخ نیومد. دوباره تلاش کن.'}</p>
            {chatError === 'RESOURCE_NOT_FOUND' ? (
              <button type="button" onClick={() => { if (onNewConversation()) setDraft(''); }}>گفت‌وگوی جدید</button>
            ) : pendingTurn ? (
              <button type="button" disabled={sending} onClick={onRetryMessage}>تلاش دوباره</button>
            ) : (
              <button type="button" onClick={onRetryChatLoad}>تلاش دوباره</button>
            )}
          </div>}
        </div>
      </div>
      <button type="button" className={`jump-to-latest ${showJump ? 'is-visible' : ''}`} tabIndex={showJump ? 0 : -1} aria-hidden={!showJump} aria-label="آخرین پیام"
        onClick={() => { stickToBottom.current = true; setShowJump(false); scrollToBottom(); }}>
        <ArrowDown size={18} aria-hidden="true" />
      </button>
      <form className="composer-wrap" ref={composerRef} onSubmit={(event) => { event.preventDefault(); send(draft); }}>
        {letterReply && <LetterReference title={letterReply.title} dismiss={onDismissLetterReply} />}
        <div className="composer">
          <textarea ref={inputRef} rows={1} wrap="soft" value={draft} onChange={(event) => setDraft(event.target.value)} maxLength={4000}
            disabled={Boolean(pendingTurn) || chatLoading} enterKeyHint="send"
            placeholder="پیامت رو بنویس…" aria-label="متن پیام"
            onKeyDown={onComposerKeyDown} />
          <button type="submit" className={`send-button ${sendDisabled ? '' : 'is-ready'}`} disabled={sendDisabled} aria-label="فرستادن پیام">
            <ArrowUp size={20} strokeWidth={2.2} aria-hidden="true" />
          </button>
        </div>
      </form>
      <ConversationSheet
        open={sheetOpen}
        onClose={() => setSheetOpen(false)}
        conversations={conversations}
        conversationId={conversationId}
        historyPhase={historyPhase}
        historyError={historyError}
        onReloadHistory={onReloadHistory}
        onOpenConversation={onOpenConversation}
        onNewConversation={onNewConversation}
        navigate={navigate}
        sending={sending}
      />
    </section>
  );
}

export default function MainScreen(props) {
  const { route } = props;
  if (route === 'home') return <Home {...props} />;
  if (route === 'survey') return <Survey {...props} />;
  if (['letter', 'day', 'insight'].includes(route)) return <Letters {...props} />;
  if (route === 'history') return <History {...props} />;
  if (route === 'saved') return <SavedMessages {...props} />;
  if (route === 'settings') return <Settings {...props} />;
  if (route === 'menu') return <Drawer {...props} />;
  if (route === 'chat') return <Chat {...props} />;
  return null;
}
