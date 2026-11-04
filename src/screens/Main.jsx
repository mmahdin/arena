import React, { useEffect, useLayoutEffect, useRef, useState } from 'react';
import {
  AudioLines,
  BarChart3,
  Bookmark,
  ChevronLeft,
  Copy,
  ExternalLink,
  FileText,
  Flag,
  Flower2,
  Grid2X2Plus,
  History as HistoryIcon,
  Info,
  LogOut,
  Mail,
  Menu as MenuIcon,
  MessageSquare,
  Pencil,
  Play,
  RotateCcw,
  Send,
  Settings2,
  Share2,
  SlidersHorizontal,
  Sparkles,
  ThumbsDown,
  ThumbsUp,
} from 'lucide-react';
import { localReflectionDate, validReflectionDate } from '../features/reflection/api';
import { JalaliDateField } from '../components/date/JalaliDateField';
import { gregorianDateOnlyToJalali, jalaliDateToGregorianDateOnly } from '../components/date/jalaliDate';
import {
  BackButton,
  ConstellationArt,
  EnvelopeArt,
  InsightSpiral,
  MiniEnvelope,
  ScribbleArt,
  StatusBar,
} from '../components/UI';

function Home({ navigate, name, underlay = false }) {
  return (
    <section className={`app-screen theme-dark home-screen ${underlay ? 'is-underlay' : ''}`}>
      <StatusBar dark />
      <button type="button" className="home-menu-button" onClick={() => navigate('menu')} aria-label="بازکردن منو">
        <MenuIcon size={25} strokeWidth={2} />
        <span className="notification-dot" />
      </button>
      <header className="home-greeting">
        <h1>سلام،<br />{name || 'دوست من'}</h1>
      </header>
      <div className="home-letter">
        <EnvelopeArt onClick={() => navigate('day')} />
        <span>بزن تا ببینی</span>
      </div>
      <button type="button" className="reflection-card" onClick={() => navigate('day')}>
        <span>یادداشت روزانه</span>
        <strong>امروز برای حال خوبت چی کار کردی؟</strong>
      </button>
      <button type="button" className="chat-fab" onClick={() => navigate('chat')}>
        <MessageSquare size={20} fill="currentColor" />
        <span>گفت‌وگو</span>
      </button>
    </section>
  );
}

function Survey({ back, answers, setAnswer, navigate, notify }) {
  const options = ['اصلاً', 'چند روز', 'بیشتر از نصف روزها', 'تقریباً هر روز'];
  const choose = (option) => {
    setAnswer('healthSurvey', option);
    notify('انتخاب شد.');
    window.setTimeout(() => navigate('home'), 350);
  };
  return (
    <section className="app-screen theme-dark survey-screen">
      <StatusBar dark />
      <BackButton dark onClick={back} />
      <div className="survey-copy">
        <p>توی دو هفتهٔ گذشته، چند وقت یه بار این حس رو داشتی؟</p>
        <h1>کم شدن علاقه یا لذتت از کارهایی که قبلاً دوست داشتی</h1>
      </div>
      <div className="survey-options">
        {options.map((option) => (
          <button
            type="button"
            key={option}
            className={answers.healthSurvey === option ? 'is-selected' : ''}
            onClick={() => choose(option)}
          >{option}</button>
        ))}
      </div>
    </section>
  );
}

function DayDetail({ back, navigate, notify, reflections = [], reflectionListPhase, reflectionListError, reflectionDayPhase,
  reflectionDayError, reflectionDate, reflectionEntry, reflectionSaving, onOpenReflection,
  onReloadReflections, onSaveReflection, onDeleteReflection }) {
  const [selectedDate, setSelectedDate] = useState(localReflectionDate);
  const [dateInput, setDateInput] = useState(() => gregorianDateOnlyToJalali(localReflectionDate()));
  useEffect(() => { setDateInput(gregorianDateOnlyToJalali(selectedDate)); }, [selectedDate]);
  const [draft, setDraft] = useState('');
  const [dirty, setDirty] = useState(false);
  useEffect(() => { void onOpenReflection(selectedDate); }, [selectedDate, onOpenReflection]);
  useEffect(() => {
    if (reflectionDayPhase === 'ready' && reflectionDate === selectedDate) {
      setDraft(reflectionEntry?.text ?? '');
      setDirty(false);
    }
  }, [reflectionDayPhase, reflectionDate, reflectionEntry?.updatedAt, selectedDate]);
  const changeDay = (date) => {
    if (!validReflectionDate(date) || date > localReflectionDate()) return false;
    if (date === selectedDate) return true;
    if (dirty && !window.confirm('یادداشتت ذخیره نشده. روز رو عوض می‌کنی؟')) return false;
    setSelectedDate(date);
    setDraft('');
    setDirty(false);
    return true;
  };
  const save = async (event) => {
    event.preventDefault();
    const ok = await onSaveReflection(selectedDate, draft);
    if (ok) setDirty(false);
    notify(ok ? 'یادداشتت ذخیره شد.' : 'یادداشت ذخیره نشد. دوباره تلاش کن.');
  };
  const remove = async () => {
    if (!reflectionEntry || !window.confirm('این یادداشت حذف بشه؟ دیگه برنمی‌گرده.')) return;
    const ok = await onDeleteReflection(selectedDate);
    if (ok) { setDraft(''); setDirty(false); notify('یادداشت حذف شد.'); }
    else notify('حذف نشد. دوباره تلاش کن.');
  };
  const safeNavigate = (target) => {
    if (dirty && !window.confirm('یادداشتت ذخیره نشده. خارج می‌شی؟')) return;
    navigate(target);
  };
  const safeBack = () => {
    if (dirty && !window.confirm('یادداشتت ذخیره نشده. خارج می‌شی؟')) return;
    back();
  };
  const loading = reflectionDayPhase === 'loading';
  const dayLabel = new Intl.DateTimeFormat('fa-IR', { dateStyle: 'long' }).format(new Date(`${selectedDate}T12:00:00`));
  return (
    <section className="app-screen theme-dark day-screen scrolling-screen">
      <div className="day-art-panel">
        <StatusBar />
        <BackButton onClick={safeBack} />
        <ScribbleArt />
      </div>
      <div className="day-content">
        <button type="button" className="transcript-button" onClick={() => safeNavigate('chat')}>
          <FileText size={23} />
          <span>رفتن به گفت‌وگو</span>
        </button>
        <h1>{dayLabel}</h1>
        <form className="daily-reflection-form" onSubmit={save}>
          <label htmlFor="daily-reflection-text">امروز برای حال خوبت چی کار کردی؟</label>
          <JalaliDateField id="daily-reflection-date" label="تاریخ یادداشت" value={dateInput} required
            disabled={reflectionSaving} maxDate={gregorianDateOnlyToJalali(localReflectionDate())}
            onChange={value => {
              setDateInput(value);
              const date = jalaliDateToGregorianDateOnly(value);
              if (date && date <= localReflectionDate() && !changeDay(date)) setDateInput(gregorianDateOnlyToJalali(selectedDate));
            }} />
          {loading && <p className="connected-status" role="status">در حال بارگذاری یادداشت…</p>}
          {reflectionDayError && <div className="connected-notice" role="alert">
            <p>یادداشت بارگذاری یا ذخیره نشد. دوباره تلاش کن.</p>
            {reflectionDayPhase === 'error' && <button type="button" onClick={() => onOpenReflection(selectedDate)}>تلاش دوباره</button>}
          </div>}
          <textarea id="daily-reflection-text" rows={5} maxLength={4000} value={draft}
            disabled={loading || reflectionSaving || reflectionDayPhase === 'error'}
            onChange={event => { setDraft(event.target.value); setDirty(true); }}
            placeholder="از امروزت بنویس…" />
          <div className="daily-reflection-actions">
            <button type="submit" disabled={loading || reflectionSaving || !draft.trim() || reflectionDayPhase === 'error' || !dirty}>
              {reflectionSaving ? 'در حال ذخیره…' : 'ذخیره'}
            </button>
            {reflectionEntry && <button type="button" className="daily-reflection-delete" onClick={remove} disabled={reflectionSaving || loading}>حذف</button>}
          </div>
        </form>
        <div className="daily-reflection-archive" aria-live="polite">
          <h2>یادداشت‌های قبلی</h2>
          {reflectionListPhase === 'loading' && <p className="connected-status">در حال بارگذاری یادداشت‌ها…</p>}
          {reflectionListError && <div className="connected-notice" role="alert"><p>یادداشت‌ها بارگذاری نشدن.</p>
            <button type="button" onClick={onReloadReflections}>تلاش دوباره</button></div>}
          {reflectionListPhase === 'ready' && reflections.length === 0 && <p className="connected-status">هنوز یادداشتی ننوشتی.</p>}
          {reflections.map(entry => <button type="button" key={entry.date} className="daily-reflection-history-row"
            onClick={() => changeDay(entry.date)} aria-current={entry.date === selectedDate ? 'date' : undefined}>
            <span>{new Intl.DateTimeFormat('fa-IR', {month:'short', day:'numeric'}).format(new Date(`${entry.date}T12:00:00`))}</span>
            <span>{entry.text.length > 55 ? `${entry.text.slice(0,55)}…` : entry.text}</span>
            <ChevronLeft size={17} />
          </button>)}
        </div>
        <button type="button" className="letter-card" onClick={() => safeNavigate('insight')}>
          <span>نامه‌ای برای تو</span>
          <strong>فکر زیاد همیشه به معنی فهمیدن نیست.</strong>
          <MiniEnvelope />
        </button>
      </div>
    </section>
  );
}

function History({ back, conversations = [], historyPhase, historyError, onReloadHistory, onOpenConversation, onDeleteConversation, onRenameConversation, deletingId, renamingId, bookmarks = [], bookmarkPhase, bookmarkError, onReloadBookmarks }) {
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
  const formatDate = (value) => {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? 'تاریخ نامشخص' : new Intl.DateTimeFormat('fa-IR', { month: 'short', day: 'numeric' }).format(date);
  };
  return (
    <section className="app-screen theme-dark history-screen">
      <StatusBar dark />
      <BackButton dark onClick={back} />
      <h1 className="history-title">گفت‌وگوهای قبلی</h1>
      <div className="history-content" aria-live="polite">
        {historyPhase === 'loading' && <p className="connected-status">در حال بارگذاری گفت‌وگوها…</p>}
        {historyError && <div className="connected-notice" role="alert">
          <p>گفت‌وگو بارگذاری یا حذف نشد. دوباره تلاش کن.</p>
          <button type="button" onClick={onReloadHistory}>تلاش دوباره</button>
        </div>}
        {historyPhase === 'ready' && conversations.length === 0 && <p className="connected-status">هنوز گفت‌وگویی نداری.</p>}
        {conversations.map((item) => (
          <div className="connected-history-entry" key={item.id}>
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
                <button type="button" className="history-row" onClick={() => onOpenConversation(item.id)}>
                  <span className="history-date"><strong>{item.title || 'گفت‌وگوی بدون عنوان'}</strong><small>{formatDate(item.updatedAt)}</small></span>
                  <span className="history-actions"><MessageSquare size={22} /><ChevronLeft size={21} /></span>
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
  { label: 'سبک گفت‌وگو', icon: SlidersHorizontal, route: 'approach' },
  { label: 'اسم من', icon: Info, route: 'name' },
  { label: 'حریم خصوصی', icon: Settings2, route: 'privacy' },
  { label: 'درباره', icon: Info, route: 'research' },
  { label: 'پشتیبانی و نظرها', icon: MessageSquare, action: 'support' },
  { label: 'تمرین‌ها', icon: Flower2, route: 'home' },
  { label: 'پرسش‌نامهٔ حال روحی', icon: BarChart3, route: 'survey', dot: true },
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
                {item.dot && <i className="notification-dot" />}
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
  return (
    <>
      <button type="button" className="message-menu-backdrop" onClick={close} aria-label="بستن گزینه‌ها" />
      <div className="message-actions" role="menu">
        <button type="button" role="menuitem" onClick={copy}><Copy size={21} /><span>کپی پیام</span></button>
        <button type="button" role="menuitem" onClick={() => {
          if ('speechSynthesis' in window && 'SpeechSynthesisUtterance' in window) {
            window.speechSynthesis.cancel();
            const utterance = new SpeechSynthesisUtterance(text);
            utterance.lang = 'fa-IR';
            window.speechSynthesis.speak(utterance);
            notify('پخش پیام شروع شد.');
          } else notify('پخش صوتی روی این دستگاه ممکن نیست.');
          close();
        }}><Play size={21} /><span>خواندن پیام</span></button>
        <button type="button" role="menuitem" disabled={!bookmarkReady || saving} onClick={async () => {
          const ok = await onToggleBookmark(message);
          notify(ok ? (saved ? 'از ذخیره‌شده‌ها حذف شد.' : 'پیام ذخیره شد.') : 'پیام ذخیره نشد. دوباره تلاش کن.');
          close();
        }}><Bookmark size={21} fill={saved ? 'currentColor' : 'none'} /><span>{saving ? 'در حال ذخیره…' : saved ? 'حذف از ذخیره‌شده‌ها' : 'ذخیره پیام'}</span></button>
        <button type="button" role="menuitem" onClick={() => { notify('این قابلیت هنوز فعال نیست.'); close(); }}><RotateCcw size={21} /><span>ساخت دوباره</span></button>
        <button type="button" role="menuitem" onClick={() => { notify('این قابلیت هنوز فعال نیست.'); close(); }}><Flag size={21} /><span>گزارش پیام</span></button>
      </div>
    </>
  );
}

function Chat({ navigate, onboardingGreeting = '', messages = [], pendingTurn, sending, chatLoading, chatError, onSendMessage, onRetryMessage, onNewConversation, notify, bookmarks = [], bookmarkPhase, bookmarkError, bookmarkBusyId, onToggleBookmark, onReloadBookmarks }) {
  const [draft, setDraft] = useState('');
  const [menuFor, setMenuFor] = useState(null);
  const scrollRef = useRef(null);
  const inputRef = useRef(null);
  const composerRef = useRef(null);
  const pressTimer = useRef(null);
  const resizeInput = () => {
    const input = inputRef.current;
    if (!input) return;
    input.style.height = '0px';
    input.style.height = `${Math.min(input.scrollHeight, 140)}px`;
    input.style.overflowY = input.scrollHeight > 140 ? 'auto' : 'hidden';
  };
  useLayoutEffect(resizeInput, [draft]);
  useLayoutEffect(() => {
    const composer = composerRef.current;
    const observer = new ResizeObserver(() => {
      composer.closest('.chat-screen').style.setProperty('--composer-height', `${composer.offsetHeight}px`);
    });
    observer.observe(composer);
    window.addEventListener('resize', resizeInput);
    return () => { observer.disconnect(); window.removeEventListener('resize', resizeInput); };
  }, []);
  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' });
  }, [messages, pendingTurn, sending]);
  useEffect(() => () => window.clearTimeout(pressTimer.current), []);
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
  return (
    <section className="app-screen theme-dark chat-screen">
      <StatusBar dark />
      <header className="chat-header">
        <BackButton dark onClick={() => navigate('home')} />
        <button type="button" className="connected-new-chat" disabled={sending} onClick={() => { if (onNewConversation()) setDraft(''); }} aria-label="گفت‌وگوی جدید">گفت‌وگوی جدید</button>
      </header>
      <div className="messages-scroll" ref={scrollRef} aria-live="polite">
        <div className="messages-list">
      {bookmarkPhase === 'error' && <div className="connected-notice" role="alert">
        <p>وضعیت پیام‌های ذخیره‌شده در دسترس نیست.</p>
        <button type="button" onClick={onReloadBookmarks}>تلاش دوباره</button>
      </div>}

          {chatLoading && <p className="connected-status">در حال بارگذاری پیام‌ها…</p>}
          {!chatLoading && messages.length === 0 && !pendingTurn && !chatError && (onboardingGreeting ? (
            <div className="message-block message-assistant onboarding-welcome"><div className="message-text">
              {onboardingGreeting.split(/\n\s*\n/).map((paragraph, index) => <p key={index}>{paragraph}</p>)}
            </div></div>
          ) : <p className="connected-status">هر چی توی ذهنته بهم بگو.</p>)}
          {messages.map((message) => (
            <div key={message.id} className={`message-block message-${message.role}`}>
              <div className={`message-text ${menuFor === message.id ? 'is-highlighted' : ''}`}
                onContextMenu={(event) => { event.preventDefault(); if (message.role === 'assistant') setMenuFor(message.id); }}
                onPointerDown={() => message.role === 'assistant' && pressStart(message.id)}
                onPointerUp={pressEnd} onPointerCancel={pressEnd} onPointerMove={pressEnd}>
                {message.content.split(/\n\s*\n/).map((paragraph, index) => <p key={index}>{paragraph}</p>)}
              </div>
            </div>
          ))}
          {pendingTurn && <div className="message-block message-user connected-pending">
            <div className="message-text"><p>{pendingTurn.text}</p></div>
            <small>{sending ? 'در حال ارسال…' : 'پیامت ارسال نشده.'}</small>
          </div>}
          {sending && <div className="typing-indicator" aria-label="دارم جواب می‌دم…"><i /><i /><i /></div>}
          {chatError && <div className="connected-notice" role="alert">
            <p>{chatError === 'TURN_ALREADY_PROCESSING' ? 'پاسخ قبلیت هنوز آماده نشده. دوباره تلاش کن.' : chatError === 'RESOURCE_NOT_FOUND' ? 'این گفت‌وگو در دسترس نیست.' : 'پاسخ نیومد. دوباره تلاش کن.'}</p>
            {chatError === 'RESOURCE_NOT_FOUND' ? <button type="button" onClick={() => { if (onNewConversation()) setDraft(''); }}>گفت‌وگوی جدید</button> : pendingTurn ? <button type="button" disabled={sending} onClick={onRetryMessage}>تلاش دوباره</button> : <button type="button" onClick={() => navigate('history')}>گفت‌وگوهای قبلی</button>}
          </div>}
        </div>
      </div>
      <form className="composer-wrap" ref={composerRef} onSubmit={(event) => { event.preventDefault(); send(draft); }}>
        <div className="composer">
          <button type="button" className="composer-plus" onClick={() => notify('ارسال فایل هنوز فعال نیست.')} aria-label="افزودن پیوست"><Grid2X2Plus size={23} /></button>
          <textarea ref={inputRef} rows={1} wrap="soft" value={draft} onChange={(event) => setDraft(event.target.value)} maxLength={4000} disabled={Boolean(pendingTurn) || sending || chatLoading} placeholder="پیامت رو بنویس…" aria-label="متن پیام"
            onKeyDown={(event) => {
              if (event.key === 'Enter' && (event.ctrlKey || event.metaKey) && !event.nativeEvent.isComposing) {
                event.preventDefault();
                send(draft);
              }
            }} />
          <button type={draft.trim() ? 'submit' : 'button'} className="voice-button send-button" disabled={Boolean(pendingTurn) || sending || chatLoading}
            onClick={!draft.trim() ? () => notify('ارسال صوتی هنوز فعال نیست.') : undefined} aria-label={draft.trim() ? 'فرستادن پیام' : 'پیام صوتی'}>
            {draft.trim() ? <Send size={21} /> : <AudioLines size={25} />}
          </button>
        </div>
      </form>
      {menuMessage && <MessageActions message={menuMessage} close={() => setMenuFor(null)} notify={notify}
        saved={bookmarks.some(item => item.id === menuMessage.id)}
        saving={bookmarkBusyId === menuMessage.id} bookmarkReady={bookmarkPhase === 'ready'} onToggleBookmark={onToggleBookmark} />}
    </section>
  );
}

function Insight({ back }) {
  const [feedback, setFeedback] = useState(null);
  return (
    <section className="app-screen theme-dark insight-screen scrolling-screen">
      <div className="insight-art-panel">
        <StatusBar />
        <BackButton onClick={back} />
        <ConstellationArt />
      </div>
      <article className="insight-detail">
        <h1>احساسات اون رو جلوتر می‌ذاری؟</h1>
        <blockquote>آخرش من عذرخواهی می‌کنم تا بحث تموم بشه.</blockquote>
        <p>ناراحت بودی که شاید براش مهم نباشی؛ ولی گفت‌وگو به موضوع دیگه‌ای رسید.</p>
        <p>وقتی ناراحتیت رو می‌گی، آخرش خودت باید آرومش کنی.</p>
        <p>گفتی آخرش خودت رو مقصر می‌دونی. می‌خوای بیشتر بررسیش کنیم؟</p>
      </article>
      <div className="feedback-panel">
        <span>بازخورد</span>
        <div>
          <button type="button" aria-label="کپی" onClick={() => navigator.clipboard?.writeText('احساسات اون رو جلوتر می‌ذاری؟')}><Copy size={22} /></button>
          <button type="button" className={feedback === 'up' ? 'is-active' : ''} aria-label="مفید بود" onClick={() => setFeedback('up')}><ThumbsUp size={22} fill={feedback === 'up' ? 'currentColor' : 'none'} /></button>
          <button type="button" className={feedback === 'down' ? 'is-active' : ''} aria-label="مفید نبود" onClick={() => setFeedback('down')}><ThumbsDown size={22} fill={feedback === 'down' ? 'currentColor' : 'none'} /></button>
        </div>
      </div>
    </section>
  );
}

export default function MainScreen(props) {
  const { route } = props;
  if (route === 'home') return <Home {...props} />;
  if (route === 'survey') return <Survey {...props} />;
  if (route === 'day') return <DayDetail {...props} />;
  if (route === 'history') return <History {...props} />;
  if (route === 'menu') return <Drawer {...props} />;
  if (route === 'chat') return <Chat {...props} />;
  if (route === 'insight') return <Insight {...props} />;
  return null;
}
