import React, { useEffect, useLayoutEffect, useRef, useState } from 'react';
import {
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
  X,
} from 'lucide-react';
import { BackButton, EnvelopeArt, StatusBar } from '../components/UI';
import { formatLetterDate } from '../features/letter/model';

const conversationDate = value => {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '' : new Intl.DateTimeFormat('fa-IR', { month: 'short', day: 'numeric' }).format(date);
};

function Home({ navigate, name, letter = null, underlay = false }) {
  return (
    <section className={`app-screen theme-dark home-screen ${underlay ? 'is-underlay' : ''}`}>
      <StatusBar dark />
      <button type="button" className="home-menu-button" onClick={() => navigate('menu')} aria-label="بازکردن منو">
        <MenuIcon size={25} strokeWidth={2} />
      </button>
      <header className="home-greeting">
        <span className="home-eyebrow">فضای آرام تو</span>
        <h1>سلام،<br />{name || 'دوست من'}</h1>
      </header>
      <div className={`home-letter ${letter ? 'has-letter' : ''}`}>
        <span className="home-letter-label"><Mail size={16} /> نامه‌ای برای تو</span>
        <EnvelopeArt onClick={() => navigate('letter')} dateLabel={letter ? formatLetterDate(letter.createdAt) : ''} />
        <div className="home-letter-copy">
          <strong>{letter?.title || 'جایی برای نامه‌های شخصی‌ات'}</strong>
          <p>{letter ? 'بزن تا نامه رو بخونی.' : 'وقتی نامه‌ای آماده باشه، اینجا منتظرت می‌مونه.'}</p>
        </div>
      </div>
      <button type="button" className="chat-fab" onClick={() => navigate('chat')}>
        <MessageSquare size={20} fill="currentColor" />
        <span>گفت‌وگو</span>
      </button>
    </section>
  );
}

function Survey({ back, answers, setAnswer, navigate }) {
  const options = ['اصلاً', 'چند روز', 'بیشتر از نصف روزها', 'تقریباً هر روز'];
  const timer = useRef(null);
  useEffect(() => () => window.clearTimeout(timer.current), []);
  const choose = (option) => {
    setAnswer('healthSurvey', option);
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => navigate('home'), 320);
  };
  return (
    <section className="app-screen theme-dark survey-screen">
      <StatusBar dark />
      <BackButton dark onClick={back} />
      <div className="survey-copy">
        <span className="survey-kicker">پرسش‌نامهٔ حال روحی</span>
        <h1>توی دو هفتهٔ گذشته، چقدر این موضوع اذیتت کرده؟</h1>
        <p>کم شدن علاقه یا لذت از کارهایی که قبلاً دوست داشتی</p>
      </div>
      <div className="survey-options" role="radiogroup" aria-label="میزان تکرار این حس">
        {options.map((option, index) => {
          const selected = answers.healthSurvey === option;
          return (
            <button type="button" role="radio" aria-checked={selected} key={option}
              className={selected ? 'is-selected' : ''} onClick={() => choose(option)}>
              <span className="survey-radio" aria-hidden="true">{selected && <Check size={15} />}</span>
              <span>{option}</span>
              <small>{['اصلاً', 'گاهی', 'اغلب', 'بیشتر روزها'][index]}</small>
            </button>
          );
        })}
      </div>
      <p className="survey-footnote">نزدیک‌ترین جواب به تجربه‌ات رو انتخاب کن.</p>
    </section>
  );
}

function ConversationSheet({ open, close, conversations, historyPhase, historyError, onReloadHistory,
  onOpenConversation, onNewConversation, navigate, conversationId }) {
  const closeRef = useRef(null);
  useEffect(() => {
    if (!open) return undefined;
    const previous = document.activeElement;
    closeRef.current?.focus();
    const keydown = event => { if (event.key === 'Escape') close(); };
    document.addEventListener('keydown', keydown);
    return () => { document.removeEventListener('keydown', keydown); previous?.focus?.(); };
  }, [open, close]);
  if (!open) return null;
  return (
    <div className="conversation-sheet-layer">
      <button type="button" className="conversation-sheet-backdrop" onClick={close} aria-label="بستن فهرست گفت‌وگوها" />
      <section className="conversation-sheet" role="dialog" aria-modal="true" aria-labelledby="conversation-sheet-title">
        <div className="sheet-handle" aria-hidden="true" />
        <header>
          <div><span>فضای گفت‌وگو</span><h2 id="conversation-sheet-title">گفت‌وگوهای تو</h2></div>
          <button ref={closeRef} type="button" className="sheet-close" onClick={close} aria-label="بستن"><X size={21} /></button>
        </header>
        <button type="button" className="sheet-new-conversation" onClick={() => { if (onNewConversation()) close(); }}>
          <Plus size={20} /><span>شروع گفت‌وگوی جدید</span>
        </button>
        <div className="sheet-conversations" aria-live="polite">
          {historyPhase === 'loading' && <p className="connected-status">در حال بارگذاری گفت‌وگوها…</p>}
          {historyError && <div className="connected-notice" role="alert"><p>فهرست گفت‌وگوها دریافت نشد.</p><button type="button" onClick={onReloadHistory}>تلاش دوباره</button></div>}
          {historyPhase === 'ready' && conversations.length === 0 && <p className="sheet-empty">اولین گفت‌وگوت رو شروع کن.</p>}
          {conversations.map(item => (
            <button type="button" key={item.id} className={`sheet-conversation ${item.id === conversationId ? 'is-current' : ''}`}
              onClick={() => { close(); onOpenConversation(item.id); }}>
              <span className="sheet-conversation-icon"><MessageSquare size={19} /></span>
              <span><strong>{item.title || 'گفت‌وگوی بدون عنوان'}</strong><small>{conversationDate(item.updatedAt)}</small></span>
              {item.id === conversationId ? <Check size={18} /> : <ChevronLeft size={18} />}
            </button>
          ))}
        </div>
        <button type="button" className="sheet-manage" onClick={() => { close(); navigate('history'); }}>مدیریت و پیام‌های ذخیره‌شده<ChevronLeft size={17} /></button>
      </section>
    </div>
  );
}

function History({ back, conversations = [], historyPhase, historyError, onReloadHistory, onOpenConversation,
  onDeleteConversation, onRenameConversation, deletingId, renamingId, bookmarks = [], bookmarkPhase,
  bookmarkError, onReloadBookmarks }) {
  const [editingId, setEditingId] = useState(null);
  const [draftTitle, setDraftTitle] = useState('');
  const [renameError, setRenameError] = useState(false);
  const beginRename = item => { setEditingId(item.id); setDraftTitle(item.title); setRenameError(false); };
  const finishRename = async (event, id) => {
    event.preventDefault();
    if (!draftTitle.trim() || [...draftTitle.trim()].length > 60) { setRenameError(true); return; }
    const success = await onRenameConversation(id, draftTitle);
    if (success) { setEditingId(null); setRenameError(false); } else setRenameError(true);
  };
  return (
    <section className="app-screen theme-dark history-screen">
      <StatusBar dark />
      <BackButton dark onClick={back} />
      <h1 className="history-title">گفت‌وگوها و ذخیره‌شده‌ها</h1>
      <div className="history-content" aria-live="polite">
        <h2>گفت‌وگوهای قبلی</h2>
        {historyPhase === 'loading' && <p className="connected-status">در حال بارگذاری گفت‌وگوها…</p>}
        {historyError && <div className="connected-notice" role="alert"><p>گفت‌وگو بارگذاری یا حذف نشد. دوباره تلاش کن.</p><button type="button" onClick={onReloadHistory}>تلاش دوباره</button></div>}
        {historyPhase === 'ready' && conversations.length === 0 && <p className="connected-status">هنوز گفت‌وگویی نداری.</p>}
        {conversations.map(item => (
          <div className="connected-history-entry" key={item.id}>
            {editingId === item.id ? (
              <form className="conversation-title-editor" onSubmit={event => finishRename(event, item.id)}>
                <label htmlFor={`rename-${item.id}`}>تغییر نام گفت‌وگو</label>
                <input id={`rename-${item.id}`} autoFocus value={draftTitle} maxLength={60} onChange={event => { setDraftTitle(event.target.value); setRenameError(false); }} disabled={renamingId === item.id} />
                {renameError && <small role="alert">نام تغییر نکرد. دوباره امتحان کن.</small>}
                <div className="conversation-title-editor-actions"><button type="submit" disabled={renamingId === item.id}>{renamingId === item.id ? 'در حال ذخیره…' : 'ذخیره نام'}</button><button type="button" disabled={renamingId === item.id} onClick={() => setEditingId(null)}>انصراف</button></div>
              </form>
            ) : <>
              <button type="button" className="history-row" onClick={() => onOpenConversation(item.id)}>
                <span className="history-date"><strong>{item.title || 'گفت‌وگوی بدون عنوان'}</strong><small>{conversationDate(item.updatedAt)}</small></span>
                <span className="history-actions"><ChevronLeft size={21} /></span>
              </button>
              <button type="button" className="connected-rename" disabled={Boolean(renamingId) || deletingId === item.id} onClick={() => beginRename(item)} aria-label={`تغییر نام ${item.title || 'گفت‌وگو'}`}><Pencil size={17} /></button>
              <button type="button" className="connected-delete" disabled={deletingId === item.id || Boolean(renamingId)} onClick={() => onDeleteConversation(item.id)} aria-label={`حذف ${item.title || 'گفت‌وگو'}`}>{deletingId === item.id ? '…' : 'حذف'}</button>
            </>}
          </div>
        ))}
        <div className="saved-history-section" aria-live="polite">
          <h2>پیام‌های ذخیره‌شده</h2>
          {bookmarkPhase === 'loading' && <p className="connected-status">در حال بارگذاری پیام‌ها…</p>}
          {bookmarkPhase === 'error' && <div className="connected-notice" role="alert"><p>پیام‌های ذخیره‌شده بارگذاری نشدن.</p><button type="button" onClick={onReloadBookmarks}>تلاش دوباره</button></div>}
          {bookmarkPhase === 'ready' && bookmarks.length === 0 && <p className="connected-status">هنوز پیامی ذخیره نکردی.</p>}
          {bookmarks.map(item => <button type="button" key={item.id} className="saved-history-entry" onClick={() => onOpenConversation(item.conversationId)}><Bookmark size={19} fill="currentColor" /><span>{item.content}</span><ChevronLeft size={19} /></button>)}
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
  { label: 'پرسش‌نامهٔ حال روحی', icon: Info, route: 'survey', dot: true },
];

function Drawer({ navigate, name, letter, notify, onLogout, onSwitchTester }) {
  const action = item => {
    if (item.route) navigate(item.route);
    else if (item.action === 'support') notify('نظرت رو توی گفت‌وگو بگو.');
  };
  const share = async () => {
    const data = { title: 'همراه', text: 'همراه کمک می‌کنه خودت رو بهتر بشناسی.' };
    try {
      if (navigator.share) await navigator.share(data);
      else { await navigator.clipboard.writeText(data.text); notify('متن کپی شد.'); }
    } catch (error) { if (error?.name !== 'AbortError') notify('اشتراک‌گذاری انجام نشد.'); }
  };
  return (
    <div className="drawer-screen">
      <Home navigate={() => {}} name={name} letter={letter} underlay />
      <button type="button" className="drawer-backdrop" onClick={() => navigate('home')} aria-label="بستن منو" />
      <aside className="drawer-panel" aria-label="منوی اصلی">
        <div className="drawer-brand">همراه</div>
        <nav className="drawer-nav">{MENU_ITEMS.map(item => { const Icon = item.icon; return <button type="button" key={item.label} onClick={() => action(item)}><Icon size={24} strokeWidth={1.65} /><span>{item.label}</span>{item.dot && <i className="notification-dot" />}</button>; })}</nav>
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
  const copy = async () => {
    try { await navigator.clipboard.writeText(message.content); notify('پیام کپی شد.'); }
    catch { notify('کپی پیام ممکن نیست.'); }
    close();
  };
  return <>
    <button type="button" className="message-menu-backdrop" onClick={close} aria-label="بستن گزینه‌ها" />
    <div className="message-actions" role="menu">
      <button type="button" role="menuitem" onClick={copy}><Copy size={21} /><span>کپی پیام</span></button>
      <button type="button" role="menuitem" onClick={() => {
        if ('speechSynthesis' in window && 'SpeechSynthesisUtterance' in window) {
          window.speechSynthesis.cancel();
          const utterance = new SpeechSynthesisUtterance(message.content); utterance.lang = 'fa-IR';
          window.speechSynthesis.speak(utterance); notify('پخش پیام شروع شد.');
        } else notify('پخش صوتی روی این دستگاه ممکن نیست.');
        close();
      }}><Play size={21} /><span>خواندن پیام</span></button>
      <button type="button" role="menuitem" disabled={!bookmarkReady || saving} onClick={async () => {
        const ok = await onToggleBookmark(message);
        notify(ok ? (saved ? 'از ذخیره‌شده‌ها حذف شد.' : 'پیام ذخیره شد.') : 'پیام ذخیره نشد. دوباره تلاش کن.'); close();
      }}><Bookmark size={21} fill={saved ? 'currentColor' : 'none'} /><span>{saving ? 'در حال ذخیره…' : saved ? 'حذف از ذخیره‌شده‌ها' : 'ذخیره پیام'}</span></button>
    </div>
  </>;
}

function Chat({ navigate, conversations = [], conversationId, onboardingGreeting = '', messages = [], pendingTurn,
  sending, chatLoading, chatError, historyPhase, historyError, onReloadHistory, onOpenConversation,
  onSendMessage, onRetryMessage, onNewConversation, notify, bookmarks = [], bookmarkPhase, bookmarkError,
  bookmarkBusyId, onToggleBookmark, onReloadBookmarks }) {
  const [draft, setDraft] = useState('');
  const [menuFor, setMenuFor] = useState(null);
  const [sheetOpen, setSheetOpen] = useState(false);
  const scrollRef = useRef(null);
  const inputRef = useRef(null);
  const composerRef = useRef(null);
  const pressTimer = useRef(null);
  const closeSheet = React.useCallback(() => setSheetOpen(false), []);
  const activeConversation = conversations.find(item => item.id === conversationId);
  const resizeInput = () => {
    const input = inputRef.current; if (!input) return;
    input.style.height = '0px'; input.style.height = `${Math.min(input.scrollHeight, 140)}px`;
    input.style.overflowY = input.scrollHeight > 140 ? 'auto' : 'hidden';
  };
  useLayoutEffect(resizeInput, [draft]);
  useLayoutEffect(() => {
    const composer = composerRef.current;
    const update = () => composer.closest('.chat-screen')?.style.setProperty('--composer-height', `${composer.offsetHeight}px`);
    const observer = new ResizeObserver(update); observer.observe(composer); update();
    window.addEventListener('resize', resizeInput);
    window.visualViewport?.addEventListener('resize', update);
    return () => { observer.disconnect(); window.removeEventListener('resize', resizeInput); window.visualViewport?.removeEventListener('resize', update); };
  }, []);
  useEffect(() => {
    const behavior = window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth';
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior });
  }, [messages, pendingTurn, sending]);
  useEffect(() => () => window.clearTimeout(pressTimer.current), []);
  const send = text => {
    const clean = text.trim(); if (!clean || sending || pendingTurn || chatLoading) return;
    if ([...clean].length > 4000) { notify('پیام می‌تونه تا ۴۰۰۰ کاراکتر باشه.'); return; }
    setDraft(''); void onSendMessage(clean);
  };
  const newConversation = () => {
    const created = onNewConversation(); if (created) { setDraft(''); setMenuFor(null); inputRef.current?.focus(); } return created;
  };
  const pressStart = id => { window.clearTimeout(pressTimer.current); pressTimer.current = window.setTimeout(() => setMenuFor(id), 520); };
  const pressEnd = () => window.clearTimeout(pressTimer.current);
  const menuMessage = messages.find(message => message.id === menuFor);
  return (
    <section className="app-screen theme-dark chat-screen">
      <StatusBar dark />
      <header className="chat-header">
        <BackButton dark onClick={() => navigate('home')} />
        <button type="button" className="chat-title-button" onClick={() => setSheetOpen(true)} aria-label="انتخاب گفت‌وگو">
          <strong>{activeConversation?.title || (conversationId ? 'گفت‌وگو' : 'گفت‌وگوی جدید')}</strong><ChevronDown size={15} />
        </button>
        <div className="chat-header-actions">
          <button type="button" disabled={sending} onClick={newConversation} aria-label="گفت‌وگوی جدید"><Plus size={22} /></button>
          <button type="button" onClick={() => setSheetOpen(true)} aria-label="گفت‌وگوهای قبلی"><HistoryIcon size={20} /></button>
        </div>
      </header>
      <div className="messages-scroll" ref={scrollRef} aria-live="polite">
        <div className="messages-list">
          {bookmarkPhase === 'error' && <div className="connected-notice" role="alert"><p>وضعیت پیام‌های ذخیره‌شده در دسترس نیست.</p><button type="button" onClick={onReloadBookmarks}>تلاش دوباره</button></div>}
          {chatLoading && <div className="chat-loading" role="status"><span /><p>در حال بازکردن گفت‌وگو…</p></div>}
          {!chatLoading && messages.length === 0 && !pendingTurn && !chatError && (onboardingGreeting ? (
            <div className="message-block message-assistant onboarding-welcome"><div className="message-text">{onboardingGreeting.split(/\n\s*\n/).map((paragraph, index) => <p key={index}>{paragraph}</p>)}</div></div>
          ) : <div className="chat-empty"><span><Flower2 size={31} strokeWidth={1.3} /></span><h1>از هر جا دوست داری شروع کن.</h1><p>این گفت‌وگو فضای مستقل خودش رو داره.<br />هر وقت آماده بودی، بنویس.</p></div>)}
          {messages.map(message => (
            <article key={message.id} className={`message-block message-${message.role}`} aria-label={message.role === 'user' ? 'پیام تو' : 'پاسخ همراه'}>
              {message.role === 'assistant' && <div className="message-author"><Flower2 size={15} /><span>همراه</span></div>}
              <div className={`message-text ${menuFor === message.id ? 'is-highlighted' : ''}`}
                onContextMenu={event => { event.preventDefault(); if (message.role === 'assistant') setMenuFor(message.id); }}
                onPointerDown={() => message.role === 'assistant' && pressStart(message.id)} onPointerUp={pressEnd} onPointerCancel={pressEnd} onPointerMove={pressEnd}>
                {message.content.split(/\n\s*\n/).map((paragraph, index) => <p key={index}>{paragraph}</p>)}
              </div>
              {message.role === 'assistant' && <button type="button" className="message-more" onClick={() => setMenuFor(message.id)} aria-label="گزینه‌های پیام"><MoreHorizontal size={19} /></button>}
            </article>
          ))}
          {pendingTurn && <div className="message-block message-user connected-pending"><div className="message-text"><p>{pendingTurn.text}</p></div><small>{sending ? 'در حال ارسال…' : 'پیامت ارسال نشده.'}</small></div>}
          {sending && <div className="typing-indicator" role="status" aria-label="همراه در حال نوشتن است"><i /><i /><i /></div>}
          {chatError && <div className="connected-notice chat-error" role="alert">
            <p>{chatError === 'TURN_ALREADY_PROCESSING' ? 'پاسخ قبلیت هنوز آماده نشده. دوباره تلاش کن.' : chatError === 'RESOURCE_NOT_FOUND' ? 'این گفت‌وگو در دسترس نیست.' : 'پاسخ نیومد. پیامت حفظ شده؛ دوباره تلاش کن.'}</p>
            {chatError === 'RESOURCE_NOT_FOUND' ? <button type="button" onClick={newConversation}>گفت‌وگوی جدید</button> : pendingTurn ? <button type="button" disabled={sending} onClick={onRetryMessage}>تلاش دوباره</button> : <button type="button" onClick={() => setSheetOpen(true)}>گفت‌وگوهای قبلی</button>}
          </div>}
        </div>
      </div>
      <form className="composer-wrap" ref={composerRef} onSubmit={event => { event.preventDefault(); send(draft); }}>
        <div className="composer">
          <textarea ref={inputRef} rows={1} wrap="soft" value={draft} onChange={event => setDraft(event.target.value)} maxLength={4000}
            disabled={Boolean(pendingTurn) || sending || chatLoading} placeholder="هر چی توی ذهنته بنویس…" aria-label="متن پیام"
            onKeyDown={event => {
              if (event.key === 'Enter' && (event.ctrlKey || event.metaKey) && !event.nativeEvent.isComposing) { event.preventDefault(); send(draft); }
            }} />
          <button type="submit" className="voice-button send-button" disabled={!draft.trim() || Boolean(pendingTurn) || sending || chatLoading} aria-label="فرستادن پیام"><Send size={20} /></button>
        </div>
      </form>
      <ConversationSheet open={sheetOpen} close={closeSheet} conversations={conversations} historyPhase={historyPhase}
        historyError={historyError} onReloadHistory={onReloadHistory} onOpenConversation={onOpenConversation}
        onNewConversation={newConversation} navigate={navigate} conversationId={conversationId} />
      {menuMessage && <MessageActions message={menuMessage} close={() => setMenuFor(null)} notify={notify}
        saved={bookmarks.some(item => item.id === menuMessage.id)} saving={bookmarkBusyId === menuMessage.id}
        bookmarkReady={bookmarkPhase === 'ready'} onToggleBookmark={onToggleBookmark} />}
    </section>
  );
}

function LetterScreen({ back, letter = null }) {
  return (
    <section className="app-screen theme-dark letter-screen scrolling-screen">
      <StatusBar dark />
      <BackButton dark onClick={back} />
      <header className="letter-header"><span><Mail size={17} /> نامه‌ای برای تو</span>{letter && <time dateTime={letter.createdAt}>{formatLetterDate(letter.createdAt)}</time>}</header>
      {letter ? (
        <article className="personal-letter">
          {letter.greeting && <p className="letter-greeting">{letter.greeting}</p>}
          <h1>{letter.title}</h1>
          <div className="letter-body">{letter.blocks.map((block, index) => {
            if (block.type === 'heading') return <h2 key={index}>{block.text}</h2>;
            if (block.type === 'question') return <p className="letter-question" key={index}>{block.text}</p>;
            return <p key={index}>{block.text}</p>;
          })}</div>
          <footer><Flower2 size={23} strokeWidth={1.3} /><span>همراه</span></footer>
        </article>
      ) : (
        <div className="letter-empty"><EnvelopeArt /><h1>اینجا برای نامه‌های واقعی توئه.</h1><p>هنوز نامه‌ای آماده نشده. نامه‌ها فقط از گفت‌وگوهای واقعی تو ساخته می‌شن؛ بدون حدس‌زدن درباره‌ات.</p></div>
      )}
    </section>
  );
}

export default function MainScreen(props) {
  const { route } = props;
  if (route === 'home') return <Home {...props} />;
  if (route === 'survey') return <Survey {...props} />;
  if (route === 'history') return <History {...props} />;
  if (route === 'menu') return <Drawer {...props} />;
  if (route === 'chat') return <Chat {...props} />;
  if (route === 'letter') return <LetterScreen {...props} />;
  return null;
}
