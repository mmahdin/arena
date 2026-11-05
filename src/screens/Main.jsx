import React, { useEffect, useLayoutEffect, useRef, useState } from 'react';
import {
  ArrowDown,
  BarChart3,
  Bookmark,
  Check,
  ChevronDown,
  ChevronLeft,
  Copy,
  ExternalLink,
  Flower2,
  History as HistoryIcon,
  Info,
  LogOut,
  Mail,
  Menu as MenuIcon,
  MessageSquare,
  MoreHorizontal,
  Pencil,
  Play,
  Plus,
  Send,
  Settings2,
  Share2,
  SlidersHorizontal,
} from 'lucide-react';
import {
  BackButton,
  BottomSheet,
  EnvelopeArt,
  MiniEnvelope,
  StatusBar,
} from '../components/UI';
import { LETTER_KIND, SAMPLE_LETTER } from '../features/letters/model';

const formatDate = (value) => {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? 'تاریخ نامشخص' : new Intl.DateTimeFormat('fa-IR', { month: 'short', day: 'numeric' }).format(date);
};
const formatTime = (value) => {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '' : new Intl.DateTimeFormat('fa-IR', { hour: '2-digit', minute: '2-digit' }).format(date);
};

function Home({ navigate, name, underlay = false, activeConversation, letterPhase, letters = [] }) {
  const hasPersonalLetter = letterPhase === 'ready' && letters.length > 0;
  return (
    <section className={`app-screen theme-dark home-screen ${underlay ? 'is-underlay' : ''}`}>
      <StatusBar dark />
      <button type="button" className="home-menu-button" onClick={() => navigate('menu')} aria-label="بازکردن منو">
        <MenuIcon size={25} strokeWidth={2} />
      </button>
      <header className="home-greeting">
        <h1>سلام،<br />{name || 'دوست من'}</h1>
      </header>
      <div className="home-letter">
        <p className="home-letter-eyebrow"><Mail size={15} aria-hidden="true" /> نامه‌ای از همراه</p>
        <EnvelopeArt
          onClick={() => navigate('letter')}
          label={hasPersonalLetter ? 'نامهٔ تازه' : 'نامهٔ همراه'}
          ariaLabel={hasPersonalLetter ? 'باز کردن نامهٔ همراه برای تو' : 'باز کردن نمونهٔ نامهٔ همراه'}
        />
        <div className="home-letter-hint">
          <strong>{hasPersonalLetter ? 'یک نامه برای تو' : 'یک نمونه برای آشنایی'}</strong>
          <span>{hasPersonalLetter ? 'بر پایهٔ گفت‌وگوهای تو' : 'هنوز نامهٔ شخصی ساخته نشده'}</span>
        </div>
      </div>
      <div className="home-chat-entry">
        <p className="home-chat-note">
          {activeConversation ? `ادامهٔ «${activeConversation.title || 'گفت‌وگوی قبلی'}»` : 'هر وقت آماده بودی، حرف بزنیم.'}
        </p>
        <button type="button" className="chat-fab" onClick={() => navigate('chat')}>
          <MessageSquare size={20} aria-hidden="true" />
          <span>گفت‌وگو با همراه</span>
          <ChevronLeft size={19} aria-hidden="true" />
        </button>
      </div>
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

function LetterPaper({ letter }) {
  const sample = letter.kind === LETTER_KIND.SAMPLE;
  return (
    <article className={`letter-paper ${sample ? 'letter-paper--sample' : ''}`}>
      <div className="letter-paper-meta">
        <span className="letter-badge">{sample ? 'نمونهٔ نمایشی' : 'نامهٔ شخصی'}</span>
        <time dateTime={letter.createdAt}>{new Intl.DateTimeFormat('fa-IR', { dateStyle: 'long' }).format(new Date(letter.createdAt))}</time>
      </div>
      <h2>{letter.title}</h2>
      <p className="letter-salutation">{letter.salutation || 'دوست من،'}</p>
      {letter.paragraphs.map((paragraph, index) => <p key={index}>{paragraph}</p>)}
      {letter.question && <p className="letter-question">{letter.question}</p>}
      <p className="letter-signature">همراه</p>
    </article>
  );
}

function Letters({ back, navigate, letterPhase = 'unavailable', letters = [], letterError,
  onReloadLetters, onOpenConversation, letterGenerating = false, letterGenerationError,
  letterDeletingId, letterDeletionError, onCreateLetter, onDeleteLetter }) {
  const personalLetters = letterPhase === 'ready' ? letters : [];
  const [view, setView] = useState(personalLetters.length ? 'personal' : 'sample');
  const [selectedId, setSelectedId] = useState(null);
  useEffect(() => {
    if (letterPhase === 'ready' && letters.length > 0) setView('personal');
  }, [letterPhase, letters.length]);
  const personalLetter = personalLetters.find(item => item.id === selectedId) || personalLetters[0] || null;
  const letter = view === 'sample' ? SAMPLE_LETTER : personalLetter;
  const sample = view === 'sample';

  const personalState = () => {
    if (letterPhase === 'loading') {
      return <div className="letter-state" role="status"><span className="letter-spinner" aria-hidden="true" /><p>در حال بارگذاری نامه‌ها…</p></div>;
    }
    if (letterPhase === 'error') {
      return <div className="letter-state connected-notice" role="alert">
        <p>نامه‌ها بارگذاری نشدن. دوباره تلاش کن.</p>
        <button type="button" onClick={onReloadLetters}>تلاش دوباره</button>
      </div>;
    }
    if (letterPhase === 'unavailable' || letterPhase === 'idle') {
      return <div className="letter-state">
        <Mail size={30} aria-hidden="true" />
        <h2>نامه‌های شخصی هنوز فعال نشده‌اند.</h2>
        <p>این بخش آمادهٔ دریافت نامه‌های واقعی است. فعلاً می‌تونی نمونه رو ببینی یا گفت‌وگو رو ادامه بدی.</p>
      </div>;
    }
    if (!personalLetter) {
      return <div className="letter-state">
        <Mail size={30} aria-hidden="true" />
        <h2>هنوز نامه‌ای برای تو نداریم.</h2>
        <p>می‌تونی بعد از یک گفت‌وگوی معنادار، خودت درخواست یک نامه بدی.</p>
      </div>;
    }
    return <>
      {personalLetters.length > 1 && <div className="letter-archive" aria-label="نامه‌های قبلی">
        {personalLetters.map(item => <button type="button" key={item.id}
          aria-pressed={personalLetter.id === item.id}
          className={personalLetter.id === item.id ? 'is-active' : ''}
          onClick={() => setSelectedId(item.id)}>{item.title}</button>)}
      </div>}
      <LetterPaper letter={personalLetter} />
      {onDeleteLetter && <div className="letter-manage">
        <button type="button" disabled={Boolean(letterDeletingId)} onClick={() => {
          if (window.confirm('این نامه حذف شود؟ این کار گفت‌وگوی اصلی را حذف نمی‌کند.')) void onDeleteLetter(personalLetter.id);
        }}>{letterDeletingId === personalLetter.id ? 'در حال حذف…' : 'حذف این نامه'}</button>
      </div>}
    </>;
  };

  return (
    <section className="app-screen theme-dark letter-screen scrolling-screen">
      <div className="letter-art-panel">
        <StatusBar />
        <BackButton onClick={back} />
        <div className="letter-hero-art" aria-hidden="true"><MiniEnvelope /></div>
      </div>
      <div className="letter-content">
        <header className="letter-heading">
          <p className="letter-eyebrow">نامه‌ای از همراه</p>
          <h1>{sample ? 'نمونهٔ نامه' : 'نامه‌ای برای تو'}</h1>
        </header>
        <div className="letter-tabs" role="tablist" aria-label="بخش نامه‌ها">
          <button type="button" role="tab" id="letter-tab-personal" aria-selected={view === 'personal'} aria-controls="letter-panel" className={view === 'personal' ? 'is-active' : ''} onClick={() => setView('personal')}>نامه‌های من</button>
          <button type="button" role="tab" id="letter-tab-sample" aria-selected={view === 'sample'} aria-controls="letter-panel" className={view === 'sample' ? 'is-active' : ''} onClick={() => setView('sample')}>نمونه</button>
        </div>
        <div id="letter-panel" role="tabpanel" aria-labelledby={view === 'personal' ? 'letter-tab-personal' : 'letter-tab-sample'}>
          {sample ? <LetterPaper letter={SAMPLE_LETTER} /> : personalState()}
        </div>
        {!sample && letterPhase === 'ready' && <div className="letter-create">
          <p>اگر بخوای، همراه از آخرین گفت‌وگوی واجد شرایطت برای نوشتن یک نامه استفاده می‌کنه. متن نامه ذخیره می‌شه و هر زمان خواستی می‌تونی حذفش کنی.</p>
          <button type="button" onClick={onCreateLetter} disabled={!onCreateLetter || letterGenerating || Boolean(letterDeletingId)}>
            {letterGenerating ? 'در حال نوشتن نامه…' : 'با رضایت من، نامه‌ای بنویس'}
          </button>
          {letterGenerationError && <p role="alert" className="letter-action-error">{
            letterGenerationError === 'INSUFFICIENT_CONVERSATION_CONTEXT'
              ? 'برای نوشتن نامه، حداقل سه پیام از تو در یک گفت‌وگو لازم است.'
              : letterGenerationError === 'LETTER_GENERATION_UNAVAILABLE'
                ? 'نوشتن نامه فعلاً در دسترس نیست. بعداً دوباره امتحان کن.'
                : 'نامه ساخته نشد. دوباره تلاش کن.'
          }</p>}
          {letterDeletionError && <p role="alert" className="letter-action-error">حذف نامه انجام نشد. دوباره تلاش کن.</p>}
        </div>}
        {sample && <p className="letter-demo-note">این متن فقط برای نمایش شکل نامه‌هاست و از گفت‌وگوهای تو ساخته نشده.</p>}
        {!sample && personalLetter && personalLetter.sourceConversationIds.length > 0 && (
          <div className="letter-provenance">
            <span>بر پایهٔ گفت‌وگوهای تو نوشته شده.</span>
            {onOpenConversation && <button type="button" onClick={() => onOpenConversation(personalLetter.sourceConversationIds[0])}>دیدن گفت‌وگوی مرتبط</button>}
          </div>
        )}
        <div className="letter-footer">
          <p>نامه‌ها برای خواندن و درنگ‌اند؛ برای حرف زدن، گفت‌وگو همیشه در دسترسه.</p>
          <button type="button" className="letter-chat-button" onClick={() => navigate('chat')}>
            <MessageSquare size={19} aria-hidden="true" /> گفت‌وگو با همراه
          </button>
        </div>
      </div>
    </section>
  );
}

function History({ back, navigate, conversations = [], conversationId, historyPhase, historyError, onReloadHistory, onOpenConversation, onDeleteConversation, onRenameConversation, onNewConversation, deletingId, renamingId, bookmarks = [], bookmarkPhase, bookmarkError, onReloadBookmarks }) {
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
                  <span className="history-actions">
                    {item.id === conversationId ? <Check size={20} aria-hidden="true" /> : <MessageSquare size={22} aria-hidden="true" />}
                    <ChevronLeft size={21} aria-hidden="true" />
                  </span>
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
        <div className="saved-history-section" aria-live="polite">
          <h2>پیام‌های ذخیره‌شده</h2>
          {bookmarkPhase === 'loading' && <p className="connected-status">در حال بارگذاری پیام‌ها…</p>}
          {bookmarkPhase === 'error' && <div className="connected-notice" role="alert"><p>پیام‌های ذخیره‌شده بارگذاری نشدن.</p><button type="button" onClick={onReloadBookmarks}>تلاش دوباره</button></div>}
          {bookmarkPhase === 'ready' && bookmarks.length === 0 && <p className="connected-status">هنوز پیامی ذخیره نکردی.</p>}
          {bookmarks.map(item => <button type="button" key={item.id} className="saved-history-entry" onClick={() => onOpenConversation(item.conversationId)}>
            <Bookmark size={19} fill="currentColor" aria-hidden="true" />
            <span>{item.content.length > 125 ? `${item.content.slice(0, 125)}…` : item.content}</span>
            <ChevronLeft size={19} aria-hidden="true" />
          </button>)}
        </div>
      </div>
    </section>
  );
}

const MENU_ITEMS = [
  { label: 'گفت‌وگوهای قبلی', icon: HistoryIcon, route: 'history' },
  { label: 'نامه‌های همراه', icon: Mail, route: 'letter' },
  { label: 'سبک گفت‌وگو', icon: SlidersHorizontal, route: 'approach' },
  { label: 'اسم من', icon: Info, route: 'name' },
  { label: 'حریم خصوصی', icon: Settings2, route: 'privacy' },
  { label: 'درباره', icon: Info, route: 'research' },
  { label: 'پشتیبانی و نظرها', icon: MessageSquare, action: 'support' },
  { label: 'پرسش‌نامهٔ حال روحی', icon: BarChart3, route: 'survey' },
];

function Drawer({ navigate, name, notify, reset, onLogout, onSwitchTester }) {
  const action = async (item) => {
    if (item.route) {
      navigate(item.route);
      return;
    }
    if (item.action === 'support') notify('نظرت رو توی گفت‌وگو بگو.');
  };
  const share = async () => {
    const data = { title: 'همراه', text: 'همراه کمک می‌کنه خودت رو بهتر بشناسی.' };
    try {
      if (navigator.share) await navigator.share(data);
      else {
        await navigator.clipboard.writeText(data.text);
        notify('متن کپی شد.');
      }
    } catch (error) {
      if (error?.name !== 'AbortError') notify('اشتراک‌گذاری انجام نشد.');
    }
  };
  return (
    <div className="drawer-screen">
      <Home navigate={() => {}} name={name} underlay />
      <button type="button" className="drawer-backdrop" onClick={() => navigate('home')} aria-label="بستن منو" />
      <aside className="drawer-panel" aria-label="منوی اصلی">
        <div className="drawer-brand">همراه</div>
        <nav className="drawer-nav">
          {MENU_ITEMS.map((item) => {
            const Icon = item.icon;
            return (
              <button type="button" key={item.label} onClick={() => action(item)}>
                <Icon size={24} strokeWidth={1.65} />
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>
        <div className="drawer-footer">
          <button type="button" onClick={() => notify('اطلاعات خطوط کمک هنوز آماده نیست.')}><ExternalLink size={23} /><span>شماره‌های کمک</span></button>
          <button type="button" onClick={share}><Share2 size={23} /><span>معرفی همراه</span></button>
          <button type="button" aria-label="تعویض کاربر آزمایشی" onClick={() => onSwitchTester?.()}><Settings2 size={23} /><span>عوض کردن حساب</span></button>
          <button type="button" onClick={() => onLogout?.()}><LogOut size={23} /><span>خروج</span></button>
        </div>
      </aside>
    </div>
  );
}

function MessageActions({ message, close, notify, saved, saving, bookmarkReady, onToggleBookmark }) {
  const text = message.content;
  const copy = async () => {
    try { await navigator.clipboard.writeText(text); notify('پیام کپی شد.'); }
    catch { notify('کپی پیام ممکن نیست.'); }
    close();
  };
  const readAloud = () => {
    if ('speechSynthesis' in window && 'SpeechSynthesisUtterance' in window) {
      window.speechSynthesis.cancel();
      const utterance = new SpeechSynthesisUtterance(text);
      utterance.lang = 'fa-IR';
      window.speechSynthesis.speak(utterance);
      notify('خواندن با صدای دستگاه شروع شد.');
    } else notify('پخش صوتی روی این دستگاه ممکن نیست.');
    close();
  };
  return (
    <BottomSheet open onClose={close} title="گزینه‌های پیام" className="message-actions-sheet">
      <div className="message-actions">
        <button type="button" onClick={copy}><Copy size={21} /><span>کپی پیام</span></button>
        <button type="button" onClick={readAloud}><Play size={21} /><span>خواندن با صدای دستگاه</span></button>
        <button type="button" disabled={!bookmarkReady || saving} onClick={async () => {
          const ok = await onToggleBookmark(message);
          notify(ok ? (saved ? 'از ذخیره‌شده‌ها حذف شد.' : 'پیام ذخیره شد.') : 'پیام ذخیره نشد. دوباره تلاش کن.');
          close();
        }}><Bookmark size={21} fill={saved ? 'currentColor' : 'none'} /><span>{saving ? 'در حال ذخیره…' : saved ? 'حذف از ذخیره‌شده‌ها' : 'ذخیره پیام'}</span></button>
      </div>
    </BottomSheet>
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
        <span className="conversation-new-icon"><Plus size={20} aria-hidden="true" /></span>
        <span><strong>گفت‌وگوی جدید</strong><small>یک فضای جدا برای حرف‌های تازه</small></span>
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
            <span className="conversation-item-icon">{item.id === conversationId ? <Check size={18} aria-hidden="true" /> : <MessageSquare size={18} aria-hidden="true" />}</span>
            <span className="conversation-item-copy">
              <strong>{item.title || 'گفت‌وگوی بدون عنوان'}</strong>
              <small>{formatDate(item.updatedAt)}</small>
            </span>
            <ChevronLeft size={18} aria-hidden="true" />
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
      <div className="chat-skeleton chat-skeleton--assistant" aria-hidden="true"><i /><i /><i /></div>
      <div className="chat-skeleton chat-skeleton--user" aria-hidden="true"><i /><i /></div>
      <div className="chat-skeleton chat-skeleton--assistant" aria-hidden="true"><i /><i /><i /><i /></div>
    </div>
  );
}

function Chat({
  navigate, onboardingGreeting = '', messages = [], pendingTurn, sending, chatLoading, chatError,
  onSendMessage, onRetryMessage, onRetryChatLoad, onNewConversation, notify,
  bookmarks = [], bookmarkPhase, bookmarkError, bookmarkBusyId, onToggleBookmark, onReloadBookmarks,
  conversations = [], conversationId, activeConversation, historyPhase, historyError, onReloadHistory,
  onOpenConversation, chatDraft = '', onDraftChange,
}) {
  const [menuFor, setMenuFor] = useState(null);
  const [sheetOpen, setSheetOpen] = useState(false);
  const [showJump, setShowJump] = useState(false);
  const scrollRef = useRef(null);
  const inputRef = useRef(null);
  const composerRef = useRef(null);
  const pressTimer = useRef(null);
  const stickToBottom = useRef(true);
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
    stickToBottom.current = true;
    setShowJump(false);
  }, [conversationId]);
  useEffect(() => {
    if (!chatLoading && stickToBottom.current) scrollToBottom(messages.length ? 'smooth' : 'auto');
  }, [messages, pendingTurn, sending, chatLoading]);
  useEffect(() => () => window.clearTimeout(pressTimer.current), []);

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
    setDraft('');
    void onSendMessage(clean);
  };
  const pressStart = (id) => {
    window.clearTimeout(pressTimer.current);
    pressTimer.current = window.setTimeout(() => setMenuFor(id), 520);
  };
  const pressEnd = () => window.clearTimeout(pressTimer.current);
  const menuMessage = messages.find((message) => message.id === menuFor);
  const title = activeConversation?.title || (conversationId ? 'گفت‌وگو' : 'گفت‌وگوی تازه');
  const sendDisabled = !draft.trim() || sending || Boolean(pendingTurn) || chatLoading;

  return (
    <section className="app-screen theme-dark chat-screen">
      <StatusBar dark />
      <header className="chat-header">
        <BackButton dark onClick={() => navigate('home')} />
        <button type="button" className="chat-title-button" onClick={() => setSheetOpen(true)} aria-haspopup="dialog" aria-expanded={sheetOpen}>
          <span className="chat-title">{title}</span>
          <small className="chat-subtitle">{conversationId ? 'برای ادامهٔ همین گفت‌وگو' : 'یک فضای تازه برای حرف‌ها'}</small>
          <ChevronDown size={16} aria-hidden="true" />
        </button>
        <button type="button" className="chat-history-button" onClick={() => setSheetOpen(true)} aria-label="گفت‌وگوها و گفت‌وگوی جدید">
          <HistoryIcon size={21} aria-hidden="true" />
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
              <span className="message-author"><Flower2 size={14} aria-hidden="true" /> همراه</span>
              <div className="message-text">
                {onboardingGreeting.split(/\n\s*\n/).map((paragraph, index) => <p key={index}>{paragraph}</p>)}
              </div>
            </div>
          ) : (
            <div className="chat-empty">
              <Flower2 size={34} strokeWidth={1.4} aria-hidden="true" />
              <h2>اینجا می‌تونی راحت حرف بزنی.</h2>
              <p>هر وقت آماده بودی، از هر چیزی که ذهنت رو مشغول کرده شروع کن.</p>
            </div>
          ))}
          {messages.map((message, index) => {
            const previous = messages[index - 1];
            const next = messages[index + 1];
            const showAuthor = message.role === 'assistant' && (!previous || previous.role !== message.role);
            const lastInGroup = !next || next.role !== message.role;
            return (
              <div key={message.id} className={`message-block message-${message.role}`}>
                {showAuthor && <span className="message-author"><Flower2 size={14} aria-hidden="true" /> همراه</span>}
                <div className={`message-text ${menuFor === message.id ? 'is-highlighted' : ''}`}
                  onContextMenu={(event) => { event.preventDefault(); if (message.role === 'assistant') setMenuFor(message.id); }}
                  onPointerDown={() => message.role === 'assistant' && pressStart(message.id)}
                  onPointerUp={pressEnd} onPointerCancel={pressEnd} onPointerMove={pressEnd}>
                  {message.content.split(/\n\s*\n/).map((paragraph, pIndex) => <p key={pIndex}>{paragraph}</p>)}
                </div>
                <div className="message-meta">
                  {lastInGroup && message.createdAt && <time dateTime={message.createdAt}>{formatTime(message.createdAt)}</time>}
                  {message.role === 'assistant' && (
                    <button type="button" className="message-more" onClick={() => setMenuFor(message.id)} aria-label="گزینه‌های پیام">
                      <MoreHorizontal size={18} aria-hidden="true" />
                    </button>
                  )}
                </div>
              </div>
            );
          })}
          {pendingTurn && <div className="message-block message-user connected-pending">
            <div className="message-text"><p>{pendingTurn.text}</p></div>
            <small>{sending ? 'در حال ارسال…' : 'پیامت ارسال نشده.'}</small>
          </div>}
          {sending && <div className="message-block message-assistant">
            <span className="message-author"><Flower2 size={14} aria-hidden="true" /> همراه</span>
            <div className="typing-indicator" aria-label="دارم جواب می‌دم…"><i /><i /><i /></div>
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
      {showJump && <button type="button" className="jump-to-latest" onClick={() => { stickToBottom.current = true; setShowJump(false); scrollToBottom(); }}>
        <ArrowDown size={16} aria-hidden="true" /> آخرین پیام
      </button>}
      <form className="composer-wrap" ref={composerRef} onSubmit={(event) => { event.preventDefault(); send(draft); }}>
        <div className="composer">
          <textarea ref={inputRef} rows={1} wrap="soft" value={draft} onChange={(event) => setDraft(event.target.value)} maxLength={4000}
            disabled={Boolean(pendingTurn) || chatLoading}
            placeholder={sending ? 'همراه داره جواب می‌ده…' : 'پیامت رو بنویس…'} aria-label="متن پیام"
            onKeyDown={(event) => {
              if (event.key === 'Enter' && (event.ctrlKey || event.metaKey) && !event.nativeEvent.isComposing) {
                event.preventDefault();
                send(draft);
              }
            }} />
          <button type="submit" className="send-button" disabled={sendDisabled} aria-label="فرستادن پیام">
            <Send size={20} aria-hidden="true" />
          </button>
        </div>
        <p className="composer-hint">Enter برای خط جدید · Ctrl+Enter برای ارسال</p>
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
      {menuMessage && <MessageActions message={menuMessage} close={() => setMenuFor(null)} notify={notify}
        saved={bookmarks.some(item => item.id === menuMessage.id)}
        saving={bookmarkBusyId === menuMessage.id} bookmarkReady={bookmarkPhase === 'ready'} onToggleBookmark={onToggleBookmark} />}
    </section>
  );
}

export default function MainScreen(props) {
  const { route } = props;
  if (route === 'home') return <Home {...props} />;
  if (route === 'survey') return <Survey {...props} />;
  if (['letter', 'day', 'insight'].includes(route)) return <Letters {...props} />;
  if (route === 'history') return <History {...props} />;
  if (route === 'menu') return <Drawer {...props} />;
  if (route === 'chat') return <Chat {...props} />;
  return null;
}
