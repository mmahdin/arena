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
        <h1>عصر بخیر،<br />{name || 'دوست من'}</h1>
      </header>
      <div className="home-letter">
        <EnvelopeArt onClick={() => navigate('day')} />
        <span>برای بازکردن، ضربه بزن</span>
      </div>
      <button type="button" className="reflection-card" onClick={() => navigate('day')}>
        <span>تأمل</span>
        <strong>امروز چه کاری برای خودت انجام دادی؟</strong>
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
        <p>در دو هفته‌ی گذشته، چقدر این مسئله برایت آزاردهنده بوده است؟</p>
        <h1>کمبود علاقه یا لذت در انجام کارها</h1>
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
    if (dirty && !window.confirm('تغییرات این روز هنوز ذخیره نشده‌اند. روز دیگری باز شود؟')) return false;
    setSelectedDate(date);
    setDraft('');
    setDirty(false);
    return true;
  };
  const save = async (event) => {
    event.preventDefault();
    const ok = await onSaveReflection(selectedDate, draft);
    if (ok) setDirty(false);
    notify(ok ? 'تأمل روزانه ذخیره شد.' : 'ذخیره انجام نشد. دوباره تلاش کن.');
  };
  const remove = async () => {
    if (!reflectionEntry || !window.confirm('تأمل این روز حذف شود؟ این کار قابل بازگشت نیست.')) return;
    const ok = await onDeleteReflection(selectedDate);
    if (ok) { setDraft(''); setDirty(false); notify('تأمل حذف شد.'); }
    else notify('حذف انجام نشد. دوباره تلاش کن.');
  };
  const safeNavigate = (target) => {
    if (dirty && !window.confirm('تغییرات این روز هنوز ذخیره نشده‌اند. از صفحه خارج شوی؟')) return;
    navigate(target);
  };
  const safeBack = () => {
    if (dirty && !window.confirm('تغییرات این روز هنوز ذخیره نشده‌اند. از صفحه خارج شوی؟')) return;
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
          <span>رفتن به متن گفت‌وگو</span>
        </button>
        <h1>{dayLabel}</h1>
        <form className="daily-reflection-form" onSubmit={save}>
          <label htmlFor="daily-reflection-text">امروز چه کاری برای خودت انجام دادی؟</label>
          <JalaliDateField id="daily-reflection-date" label="تاریخ تأمل" value={dateInput} required
            disabled={reflectionSaving} maxDate={gregorianDateOnlyToJalali(localReflectionDate())}
            onChange={value => {
              setDateInput(value);
              const date = jalaliDateToGregorianDateOnly(value);
              if (date && date <= localReflectionDate() && !changeDay(date)) setDateInput(gregorianDateOnlyToJalali(selectedDate));
            }} />
          {loading && <p className="connected-status" role="status">در حال دریافت تأمل…</p>}
          {reflectionDayError && <div className="connected-notice" role="alert">
            <p>دریافت یا ذخیرهٔ تأمل انجام نشد. دوباره تلاش کن.</p>
            {reflectionDayPhase === 'error' && <button type="button" onClick={() => onOpenReflection(selectedDate)}>تلاش دوباره</button>}
          </div>}
          <textarea id="daily-reflection-text" rows={5} maxLength={4000} value={draft}
            disabled={loading || reflectionSaving || reflectionDayPhase === 'error'}
            onChange={event => { setDraft(event.target.value); setDirty(true); }}
            placeholder="هرقدر دوست داری دربارهٔ امروزت بنویس…" />
          <div className="daily-reflection-actions">
            <button type="submit" disabled={loading || reflectionSaving || !draft.trim() || reflectionDayPhase === 'error' || !dirty}>
              {reflectionSaving ? 'در حال ذخیره…' : 'ذخیرهٔ تأمل'}
            </button>
            {reflectionEntry && <button type="button" className="daily-reflection-delete" onClick={remove} disabled={reflectionSaving || loading}>حذف تأمل</button>}
          </div>
        </form>
        <div className="daily-reflection-archive" aria-live="polite">
          <h2>تأمل‌های گذشته</h2>
          {reflectionListPhase === 'loading' && <p className="connected-status">در حال دریافت تأمل‌ها…</p>}
          {reflectionListError && <div className="connected-notice" role="alert"><p>تأمل‌های گذشته دریافت نشدند.</p>
            <button type="button" onClick={onReloadReflections}>تلاش دوباره</button></div>}
          {reflectionListPhase === 'ready' && reflections.length === 0 && <p className="connected-status">هنوز تأملی ثبت نکرده‌ای.</p>}
          {reflections.map(entry => <button type="button" key={entry.date} className="daily-reflection-history-row"
            onClick={() => changeDay(entry.date)} aria-current={entry.date === selectedDate ? 'date' : undefined}>
            <span>{new Intl.DateTimeFormat('fa-IR', {month:'short', day:'numeric'}).format(new Date(`${entry.date}T12:00:00`))}</span>
            <span>{entry.text.length > 55 ? `${entry.text.slice(0,55)}…` : entry.text}</span>
            <ChevronLeft size={17} />
          </button>)}
        </div>
        <button type="button" className="letter-card" onClick={() => safeNavigate('insight')}>
          <span>نامه</span>
          <strong>فهمیدن و فکرکردنِ بیش‌ازحد، دوقلوها هستند</strong>
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
      <h1 className="history-title">تاریخچه</h1>
      <div className="history-content" aria-live="polite">
        <h2>گفتگوهای من</h2>
        {historyPhase === 'loading' && <p className="connected-status">در حال دریافت گفتگوها…</p>}
        {historyError && <div className="connected-notice" role="alert">
          <p>دریافت یا حذف گفتگو انجام نشد. اتصال را بررسی کن و دوباره تلاش کن.</p>
          <button type="button" onClick={onReloadHistory}>تلاش دوباره</button>
        </div>}
        {historyPhase === 'ready' && conversations.length === 0 && <p className="connected-status">هنوز گفتگویی ثبت نکرده‌ای. از صفحهٔ خانه گفتگوی جدیدی شروع کن.</p>}
        {conversations.map((item) => (
          <div className="connected-history-entry" key={item.id}>
            {editingId === item.id ? (
              <form className="conversation-title-editor" onSubmit={(event) => finishRename(event, item.id)}>
                <label htmlFor={`rename-${item.id}`}>تغییر نام گفتگو</label>
                <input id={`rename-${item.id}`} autoFocus value={draftTitle} maxLength={60} onChange={event => { setDraftTitle(event.target.value); setRenameError(false); }} disabled={renamingId === item.id} />
                {renameError && <small role="alert">تغییر نام انجام نشد. عنوان را بررسی کن و دوباره تلاش کن.</small>}
                <div className="conversation-title-editor-actions">
                  <button type="submit" disabled={renamingId === item.id}>{renamingId === item.id ? 'در حال ذخیره…' : 'ذخیره نام'}</button>
                  <button type="button" disabled={renamingId === item.id} onClick={() => { setEditingId(null); setRenameError(false); }}>انصراف</button>
                </div>
              </form>
            ) : (
              <>
                <button type="button" className="history-row" onClick={() => onOpenConversation(item.id)}>
                  <span className="history-date"><strong>{item.title || 'گفتگوی بدون عنوان'}</strong><small>{formatDate(item.updatedAt)}</small></span>
                  <span className="history-actions"><MessageSquare size={22} /><ChevronLeft size={21} /></span>
                </button>
                <button type="button" className="connected-rename" disabled={Boolean(renamingId) || deletingId === item.id} onClick={() => beginRename(item)} aria-label={`تغییر نام ${item.title || 'گفتگو'}`}>
                  <Pencil size={17} aria-hidden="true" />
                </button>
                <button type="button" className="connected-delete" disabled={deletingId === item.id || Boolean(renamingId)} onClick={() => onDeleteConversation(item.id)} aria-label={`حذف ${item.title || 'گفتگو'}`}>
                  {deletingId === item.id ? 'در حال حذف…' : 'حذف'}
                </button>
              </>
            )}
          </div>
        ))}
        <div className="saved-history-section" aria-live="polite">
          <h2>پیام‌های ذخیره‌شده</h2>
          {bookmarkPhase === 'loading' && <p className="connected-status">در حال دریافت پیام‌های ذخیره‌شده…</p>}
          {bookmarkPhase === 'error' && <div className="connected-notice" role="alert"><p>دریافت پیام‌های ذخیره‌شده انجام نشد.</p><button type="button" onClick={onReloadBookmarks}>تلاش دوباره</button></div>}
          {bookmarkPhase === 'ready' && bookmarks.length === 0 && <p className="connected-status">هنوز پیامی ذخیره نکرده‌ای.</p>}
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
  { label: 'تاریخچه', icon: HistoryIcon, route: 'history' },
  { label: 'ترجیحات', icon: SlidersHorizontal, route: 'approach' },
  { label: 'نام نمایشی', icon: Info, route: 'name' },
  { label: 'حساب و داده‌ها', icon: Settings2, route: 'privacy' },
  { label: 'درباره', icon: Info, route: 'research' },
  { label: 'پشتیبانی و بازخورد', icon: MessageSquare, action: 'support' },
  { label: 'تمرین‌ها', icon: Flower2, route: 'home' },
  { label: 'پرسش‌نامه‌ی سلامت', icon: BarChart3, route: 'survey', dot: true },
];

function Drawer({ navigate, name, notify, reset, onLogout, onSwitchTester }) {
  const action = async (item) => {
    if (item.route) {
      navigate(item.route);
      return;
    }
    if (item.action === 'support') notify('می‌توانی بازخوردت را همین‌جا در گفت‌وگو بنویسی.');
  };
  const share = async () => {
    const data = { title: 'همراه', text: 'همراه؛ گفت‌وگویی آرام برای شناخت بهتر خودت.' };
    try {
      if (navigator.share) await navigator.share(data);
      else {
        await navigator.clipboard.writeText(data.text);
        notify('معرفی همراه کپی شد.');
      }
    } catch (error) {
      if (error?.name !== 'AbortError') notify('اشتراک‌گذاری در دسترس نیست.');
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
          <button type="button" onClick={() => notify('راهنمای خطوط یاری در نسخه‌ی کامل نمایش داده می‌شود.')}><ExternalLink size={23} /><span>پیداکردن خط یاری</span></button>
          <button type="button" onClick={share}><Share2 size={23} /><span>معرفی همراه</span></button>
          <button type="button" aria-label="تعویض کاربر آزمایشی" onClick={() => onSwitchTester?.()}><Settings2 size={23} /><span>تغییر حساب</span></button>
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
    catch { notify('کپی پیام در این مرورگر در دسترس نیست.'); }
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
            notify('خواندن پیام با امکانات دستگاه شروع شد.');
          } else notify('خواندن پیام روی این دستگاه پشتیبانی نمی‌شود.');
          close();
        }}><Play size={21} /><span>خواندن با صدای دستگاه</span></button>
        <button type="button" role="menuitem" disabled={!bookmarkReady || saving} onClick={async () => {
          const ok = await onToggleBookmark(message);
          notify(ok ? (saved ? 'ذخیرهٔ پیام برداشته شد.' : 'پیام ذخیره شد.') : 'ذخیرهٔ پیام انجام نشد. دوباره تلاش کن.');
          close();
        }}><Bookmark size={21} fill={saved ? 'currentColor' : 'none'} /><span>{saving ? 'در حال ذخیره…' : saved ? 'برداشتن از ذخیره‌شده‌ها' : 'ذخیره پیام'}</span></button>
        <button type="button" role="menuitem" onClick={() => { notify('ساختن دوبارهٔ پاسخ به‌زودی فعال می‌شود.'); close(); }}><RotateCcw size={21} /><span>ساختن دوباره</span></button>
        <button type="button" role="menuitem" onClick={() => { notify('گزارش پیام به‌زودی فعال می‌شود.'); close(); }}><Flag size={21} /><span>گزارش پیام</span></button>
      </div>
    </>
  );
}

function Chat({ navigate, messages = [], pendingTurn, sending, chatLoading, chatError, onSendMessage, onRetryMessage, onNewConversation, notify, bookmarks = [], bookmarkPhase, bookmarkError, bookmarkBusyId, onToggleBookmark, onReloadBookmarks }) {
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
    if ([...clean].length > 4000) { notify('پیام باید کمتر از ۴۰۰۰ نویسه باشد.'); return; }
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
        <button type="button" className="connected-new-chat" disabled={sending} onClick={() => { if (onNewConversation()) setDraft(''); }} aria-label="گفتگوی جدید">گفتگوی جدید</button>
      </header>
      <div className="messages-scroll" ref={scrollRef} aria-live="polite">
        <div className="messages-list">
      {bookmarkPhase === 'error' && <div className="connected-notice" role="alert">
        <p>وضعیت پیام‌های ذخیره‌شده دریافت نشد.</p>
        <button type="button" onClick={onReloadBookmarks}>تلاش دوباره</button>
      </div>}

          {chatLoading && <p className="connected-status">در حال دریافت پیام‌ها…</p>}
          {!chatLoading && messages.length === 0 && !pendingTurn && !chatError && <p className="connected-status">برای شروع گفتگو، اولین پیامت را بنویس.</p>}
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
            <small>{sending ? 'در حال ارسال…' : 'پیام هنوز تأیید نشده است'}</small>
          </div>}
          {sending && <div className="typing-indicator" aria-label="همراه در حال پاسخ‌دادن است"><i /><i /><i /></div>}
          {chatError && <div className="connected-notice" role="alert">
            <p>{chatError === 'TURN_ALREADY_PROCESSING' ? 'درخواست قبلی هنوز در حال پردازش است. با همان پیام دوباره تلاش کن.' : chatError === 'RESOURCE_NOT_FOUND' ? 'این گفتگو پیدا نشد یا دسترسی به آن ممکن نیست.' : 'دریافت پاسخ انجام نشد؛ ممکن است مشکل شبکه یا سرویس باشد.'}</p>
            {chatError === 'RESOURCE_NOT_FOUND' ? <button type="button" onClick={() => { if (onNewConversation()) setDraft(''); }}>گفتگوی جدید</button> : pendingTurn ? <button type="button" disabled={sending} onClick={onRetryMessage}>تلاش دوباره با همان پیام</button> : <button type="button" onClick={() => navigate('history')}>رفتن به تاریخچه</button>}
          </div>}
        </div>
      </div>
      <form className="composer-wrap" ref={composerRef} onSubmit={(event) => { event.preventDefault(); send(draft); }}>
        <div className="composer">
          <button type="button" className="composer-plus" onClick={() => notify('پیوست فعلاً پشتیبانی نمی‌شود.')} aria-label="افزودن پیوست"><Grid2X2Plus size={23} /></button>
          <textarea ref={inputRef} rows={1} wrap="soft" value={draft} onChange={(event) => setDraft(event.target.value)} maxLength={4000} disabled={Boolean(pendingTurn) || sending || chatLoading} placeholder="پیامت را بنویس" aria-label="متن پیام"
            onKeyDown={(event) => {
              if (event.key === 'Enter' && (event.ctrlKey || event.metaKey) && !event.nativeEvent.isComposing) {
                event.preventDefault();
                send(draft);
              }
            }} />
          <button type={draft.trim() ? 'submit' : 'button'} className="voice-button send-button" disabled={Boolean(pendingTurn) || sending || chatLoading}
            onClick={!draft.trim() ? () => notify('ورودی صوتی هنوز پشتیبانی نمی‌شود.') : undefined} aria-label={draft.trim() ? 'فرستادن پیام' : 'ورودی صوتی'}>
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
        <h1>احساسات او را بیشتر از احساسات خودت محافظت می‌کنی</h1>
        <blockquote>معمولاً برای اینکه بین ما همه‌چیز خوب شود، آخرش من عذرخواهی می‌کنم.</blockquote>
        <p>اینجا آمدی در حالی که آسیب دیده بودی؛ می‌خواستی نزد کسی که دوستش داری احساس اهمیت کنی. اما همین‌جور که حرف زدیم، چیزی تغییر کرد.</p>
        <p>شروع کردی به زیرسؤال‌بردنِ معتربودنِ احساساتت. تو هستی که منتظر می‌ماند، چک می‌کند و درد دارد؛ اما وقتی می‌خواهی حرفش را بزنی، آخرش تو او را دلداری می‌دهی.</p>
        <p>خودت گفتی: «همیشه آخرش حس می‌کنم مقصر منم، حتی وقتی اول من آسیب دیده‌ام.» این فکرکردنِ بیش‌ازحد نیست؛ الگویی است که ارزش توجه‌دارد.</p>
      </article>
      <div className="feedback-panel">
        <span>بازخورد</span>
        <div>
          <button type="button" aria-label="کپی" onClick={() => navigator.clipboard?.writeText('احساسات او را بیشتر از احساسات خودت محافظت می‌کنی')}><Copy size={22} /></button>
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
