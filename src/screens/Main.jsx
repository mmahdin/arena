import React, { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import {
  ArrowDown,
  AudioLines,
  BarChart3,
  Bookmark,
  Check,
  ChevronDown,
  ChevronLeft,
  Copy,
  ExternalLink,
  Flag,
  Flower2,
  Grid2X2Plus,
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
  RotateCcw,
  Send,
  Settings2,
  Share2,
  SlidersHorizontal,
  Sparkles,
  SquarePen,
  ThumbsDown,
  ThumbsUp,
  Trash2,
  X,
} from 'lucide-react';
import { CHAT_STARTER_PROMPTS } from '../data';
import { LETTER_FORMAT_PREVIEWS } from '../features/letter/model';
import {
  BackButton,
  EnvelopeArt,
  StatusBar,
} from '../components/UI';

function formatPersianDate(value, options = { month: 'short', day: 'numeric' }) {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  try {
    return new Intl.DateTimeFormat('fa-IR-u-ca-persian', options).format(date);
  } catch {
    return new Intl.DateTimeFormat('fa-IR', options).format(date);
  }
}

function formatConversationRelativeDate(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'تاریخ نامشخص';
  const now = new Date();
  const sameDay = date.getFullYear() === now.getFullYear()
    && date.getMonth() === now.getMonth()
    && date.getDate() === now.getDate();
  if (sameDay) {
    return `امروز · ${new Intl.DateTimeFormat('fa-IR', { hour: '2-digit', minute: '2-digit' }).format(date)}`;
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  const isYesterday = date.getFullYear() === yesterday.getFullYear()
    && date.getMonth() === yesterday.getMonth()
    && date.getDate() === yesterday.getDate();
  if (isYesterday) return 'دیروز';
  return formatPersianDate(date, { month: 'short', day: 'numeric' });
}

function Home({ navigate, name, letters = [], underlay = false }) {
  const todayLabel = useMemo(
    () => formatPersianDate(new Date(), { weekday: 'long', day: 'numeric', month: 'long' }),
    [],
  );
  const latestLetter = letters[0] || null;
  const hasUnread = letters.some((item) => !item.readAt);
  const envelopeLabel = latestLetter
    ? formatPersianDate(latestLetter.createdAt, { day: 'numeric', month: 'long' }) || 'نامهٔ تازه'
    : 'نامهٔ همراه';

  return (
    <section className={`app-screen theme-dark home-screen ${underlay ? 'is-underlay' : ''}`}>
      <StatusBar dark />
      <button type="button" className="home-menu-button" onClick={() => navigate('menu')} aria-label="بازکردن منو">
        <MenuIcon size={25} strokeWidth={2} />
        <span className="notification-dot" />
      </button>

      <div className="home-layout">
        <header className="home-greeting">
          {todayLabel && <span className="home-date-whisper">{todayLabel}</span>}
          <h1>سلام،<br />{name || 'دوست من'}</h1>
        </header>

        <div className="home-letter">
          <div className="home-letter-aura" aria-hidden="true" />
          <EnvelopeArt
            onClick={() => navigate('day')}
            dateLabel={envelopeLabel}
            unread={hasUnread}
          />
          <div className="home-letter-caption">
            <strong>{latestLetter ? latestLetter.title : 'نامهٔ شخصی از گفت‌وگوهای تو'}</strong>
            <span>{latestLetter ? 'بزن تا بخونی' : 'بزن تا ببینی چطور شکل می‌گیره'}</span>
          </div>
        </div>

        <div className="home-footer">
          <button type="button" className="chat-fab" onClick={() => navigate('chat')}>
            <MessageSquare size={20} fill="currentColor" />
            <span>گفت‌وگو با همراه</span>
          </button>
        </div>
      </div>
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
    <section className="app-screen theme-dark survey-screen scrolling-screen">
      <StatusBar dark />
      <header className="survey-topbar">
        <BackButton dark onClick={back} />
        <span className="survey-badge">پرسش‌نامهٔ حال روحی</span>
      </header>

      <div className="survey-body">
        <div className="survey-copy">
          <p className="survey-intro">توی دو هفتهٔ گذشته، چند وقت یه بار این حس رو داشتی؟</p>
          <h1>کم شدن علاقه یا لذتت از کارهایی که قبلاً دوست داشتی</h1>
        </div>

        <div className="survey-options" role="radiogroup" aria-label="گزینه‌های پاسخ">
          {options.map((option) => {
            const selected = answers.healthSurvey === option;
            return (
              <button
                type="button"
                role="radio"
                aria-checked={selected}
                key={option}
                className={selected ? 'is-selected' : ''}
                onClick={() => choose(option)}
              >
                <span className="survey-option-label">{option}</span>
                <span className="survey-option-radio" aria-hidden="true">
                  {selected && <Check size={16} strokeWidth={2.5} />}
                </span>
              </button>
            );
          })}
        </div>
      </div>
    </section>
  );
}

function LetterDocument({ letter, recipientName, notify }) {
  const [feedback, setFeedback] = useState(null);
  const salutation = letter.salutation || `برای ${recipientName || 'دوست من'} عزیز،`;
  const createdLabel = letter.createdAt ? formatPersianDate(letter.createdAt, { year: 'numeric', month: 'long', day: 'numeric' }) : '';

  const copyLetter = async () => {
    const content = [letter.title, salutation, ...(letter.paragraphs || []), letter.signoff].filter(Boolean).join('\n\n');
    try {
      await navigator.clipboard.writeText(content);
      notify?.('متن نامه کپی شد.');
    } catch {
      notify?.('کپی نامه ممکن نیست.');
    }
  };

  return (
    <article className={`letter-sheet ${letter.isSamplePreview ? 'letter-sheet--preview' : ''}`}>
      {letter.isSamplePreview && (
        <div className="letter-preview-banner" role="note">
          <Sparkles size={15} aria-hidden="true" />
          <span>نمونهٔ نمایشی قالب نامه — این متن از گفت‌وگوهای شما ساخته نشده است.</span>
        </div>
      )}

      <header className="letter-sheet-header">
        <div className="letter-seal-row">
          <span className="letter-wax-medallion" aria-hidden="true">
            <Flower2 size={22} strokeWidth={1.5} />
          </span>
          {createdLabel && <time className="letter-date">{createdLabel}</time>}
        </div>

        {letter.themes?.length > 0 && (
          <div className="letter-themes" aria-label="محورهای نامه">
            {letter.themes.map((theme) => (
              <span key={theme} className="letter-theme-chip">{theme}</span>
            ))}
          </div>
        )}

        <h1>{letter.title}</h1>
        <p className="letter-salutation">{salutation}</p>
      </header>

      {letter.quote && (
        <blockquote className="letter-quote">
          {letter.quote}
        </blockquote>
      )}

      <div className="letter-body">
        {letter.paragraphs.map((paragraph, index) => (
          <p key={index}>{paragraph}</p>
        ))}
      </div>

      <footer className="letter-signoff">
        <span>{letter.signoff || 'با مهر، همراه'}</span>
      </footer>

      {!letter.isSamplePreview && (
        <div className="letter-feedback">
          <span>این نامه چقدر به شناختت نزدیک بود؟</span>
          <div className="letter-feedback-actions">
            <button type="button" aria-label="کپی نامه" onClick={copyLetter}>
              <Copy size={19} />
            </button>
            <button
              type="button"
              className={feedback === 'up' ? 'is-active' : ''}
              aria-label="مفید بود"
              onClick={() => setFeedback('up')}
            >
              <ThumbsUp size={19} fill={feedback === 'up' ? 'currentColor' : 'none'} />
            </button>
            <button
              type="button"
              className={feedback === 'down' ? 'is-active' : ''}
              aria-label="مفید نبود"
              onClick={() => setFeedback('down')}
            >
              <ThumbsDown size={19} fill={feedback === 'down' ? 'currentColor' : 'none'} />
            </button>
          </div>
        </div>
      )}
    </article>
  );
}

function DayDetail({
  back,
  navigate,
  notify,
  name,
  conversations = [],
  letters = [],
  activeLetter = null,
  letterListPhase = 'ready',
  letterDetailPhase = 'idle',
  letterError = null,
  onOpenLetter,
  onMarkLetterRead,
  onReloadLetters,
}) {
  const [previewKind, setPreviewKind] = useState(null);
  const displayedLetter = previewKind ? LETTER_FORMAT_PREVIEWS[previewKind] : activeLetter;

  useEffect(() => {
    if (!previewKind && letters.length > 0 && !activeLetter && letterDetailPhase === 'idle') {
      void onOpenLetter?.(letters[0].id);
    }
  }, [letters, activeLetter, letterDetailPhase, previewKind, onOpenLetter]);

  useEffect(() => {
    if (activeLetter && !activeLetter.readAt) {
      void onMarkLetterRead?.(activeLetter.id);
    }
  }, [activeLetter, onMarkLetterRead]);

  return (
    <section className="app-screen theme-dark day-screen letter-screen scrolling-screen">
      <StatusBar dark />
      <header className="letter-topbar">
        <BackButton dark onClick={back} />
        <span className="letter-topbar-title">نامهٔ شخصی همراه</span>
      </header>

      <div className="letter-scroll-body">
        {letters.length > 1 && !previewKind && (
          <div className="letter-archive-pills" role="tablist" aria-label="نامه‌های قبلی">
            {letters.map((item) => {
              const selected = activeLetter?.id === item.id;
              return (
                <button
                  type="button"
                  role="tab"
                  key={item.id}
                  aria-selected={selected}
                  className={`letter-archive-pill ${selected ? 'is-active' : ''}`}
                  onClick={() => onOpenLetter?.(item.id)}
                >
                  <span>{formatPersianDate(item.createdAt, { day: 'numeric', month: 'short' }) || item.title}</span>
                  {!item.readAt && <i className="letter-unread-dot" aria-hidden="true" />}
                </button>
              );
            })}
          </div>
        )}

        {(letterListPhase === 'loading' || letterDetailPhase === 'loading') && !displayedLetter && (
          <div className="letter-sheet letter-sheet--loading" role="status">
            <div className="skeleton-line skeleton-line--short" />
            <div className="skeleton-line skeleton-line--title" />
            <div className="skeleton-line" />
            <div className="skeleton-line" />
            <div className="skeleton-line skeleton-line--medium" />
            <p className="sr-only">در حال باز کردن نامه…</p>
          </div>
        )}

        {letterError && !displayedLetter && (
          <div className="connected-notice" role="alert">
            <p>نامه در حال حاضر باز نشد. دوباره تلاش کن.</p>
            <button type="button" onClick={onReloadLetters}>تلاش دوباره</button>
          </div>
        )}

        {displayedLetter ? (
          <>
            <LetterDocument letter={displayedLetter} recipientName={name} notify={notify} />
            {previewKind && (
              <div className="letter-preview-switcher">
                <div className="letter-preview-tabs" role="group" aria-label="اندازهٔ نمونهٔ نامه">
                  <button
                    type="button"
                    className={previewKind === 'short' ? 'is-active' : ''}
                    onClick={() => setPreviewKind('short')}
                  >
                    نمونهٔ کوتاه
                  </button>
                  <button
                    type="button"
                    className={previewKind === 'long' ? 'is-active' : ''}
                    onClick={() => setPreviewKind('long')}
                  >
                    نمونهٔ بلند
                  </button>
                </div>
                <button type="button" className="letter-preview-close" onClick={() => setPreviewKind(null)}>
                  بازگشت به وضعیت واقعی نامه
                </button>
              </div>
            )}
          </>
        ) : (
          letterListPhase !== 'loading' && letterDetailPhase !== 'loading' && !letterError && (
            <article className="letter-sheet letter-sheet--awaiting">
              <div className="letter-seal-row">
                <span className="letter-wax-medallion" aria-hidden="true">
                  <Flower2 size={24} strokeWidth={1.45} />
                </span>
                <span className="letter-status-pill">به‌زودی از گفت‌وگوهای تو</span>
              </div>

              <h1>نامه‌هایی برای شناختِ عمیق‌تر خودت</h1>
              <p className="letter-salutation">{name ? `${name} عزیز،` : 'دوست عزیز،'}</p>

              <div className="letter-body">
                <p>
                  نامه‌های شخصی همراه قراره آینه‌ای آروم و دقیق از گفت‌وگوهای واقعی خودت باشن؛ جایی برای دیدنِ الگوهای تکرارشونده، نقطه‌های قوت، و چیزهایی که شاید وسطِ شلوغی روزمره کمتر بهشون توجه می‌کنی.
                </p>
                <p>
                  تا وقتی که بخشِ ساختِ نامه در سرور فعال بشه، هیچ تحلیل ساختگی یا غیرواقعی اینجا نشون داده نمی‌شه. هر زمان خواستی دربارهٔ چیزی که ذهنت رو مشغول کرده حرف بزنی، گفت‌وگو همیشه بازه.
                </p>
              </div>

              <div className="letter-awaiting-actions">
                <button type="button" className="letter-primary-cta" onClick={() => navigate('chat')}>
                  <MessageSquare size={18} />
                  <span>{conversations.length > 0 ? 'ادامهٔ گفت‌وگو با همراه' : 'شروع اولین گفت‌وگو'}</span>
                </button>
              </div>

              <div className="letter-sample-prompt">
                <span>مشاهدهٔ قالب طراحی نامه (نمونهٔ غیرواقعی):</span>
                <div className="letter-sample-buttons">
                  <button type="button" onClick={() => setPreviewKind('short')}>قالب نامهٔ کوتاه</button>
                  <button type="button" onClick={() => setPreviewKind('long')}>قالب نامهٔ بلند</button>
                </div>
              </div>
            </article>
          )
        )}
      </div>
    </section>
  );
}

function History({
  back,
  conversationId = null,
  conversations = [],
  historyPhase,
  historyError,
  onReloadHistory,
  onOpenConversation,
  onNewConversation,
  onDeleteConversation,
  onRenameConversation,
  deletingId,
  renamingId,
  bookmarks = [],
  bookmarkPhase,
  onReloadBookmarks,
}) {
  const [editingId, setEditingId] = useState(null);
  const [draftTitle, setDraftTitle] = useState('');
  const [renameError, setRenameError] = useState(false);

  const beginRename = (item) => {
    setEditingId(item.id);
    setDraftTitle(item.title);
    setRenameError(false);
  };

  const finishRename = async (event, id) => {
    event.preventDefault();
    if (!draftTitle.trim() || [...draftTitle.trim()].length > 60) {
      setRenameError(true);
      return;
    }
    const success = await onRenameConversation(id, draftTitle);
    if (success) {
      setEditingId(null);
      setRenameError(false);
    } else {
      setRenameError(true);
    }
  };

  return (
    <section className="app-screen theme-dark history-screen">
      <StatusBar dark />
      <header className="history-header">
        <BackButton dark onClick={back} />
        <h1 className="history-title">گفت‌وگوهای قبلی</h1>
        <button
          type="button"
          className="history-new-button"
          onClick={() => onNewConversation?.()}
          aria-label="گفت‌وگوی جدید"
        >
          <Plus size={18} />
          <span>جدید</span>
        </button>
      </header>

      <div className="history-content" aria-live="polite">
        <section className="history-threads-section">
          {historyPhase === 'loading' && <p className="connected-status">در حال بارگذاری گفت‌وگوها…</p>}
          {historyError && (
            <div className="connected-notice" role="alert">
              <p>گفت‌وگو بارگذاری یا حذف نشد. دوباره تلاش کن.</p>
              <button type="button" onClick={onReloadHistory}>تلاش دوباره</button>
            </div>
          )}
          {historyPhase === 'ready' && conversations.length === 0 && (
            <div className="history-empty-card">
              <p className="connected-status">هنوز گفت‌وگویی نداری.</p>
              <button type="button" className="history-empty-cta" onClick={() => onNewConversation?.()}>
                <SquarePen size={17} />
                <span>شروع اولین گفت‌وگو</span>
              </button>
            </div>
          )}

          {conversations.map((item) => {
            const isCurrent = item.id === conversationId;
            return (
              <div className={`connected-history-entry ${isCurrent ? 'is-current-thread' : ''}`} key={item.id}>
                {editingId === item.id ? (
                  <form className="conversation-title-editor" onSubmit={(event) => finishRename(event, item.id)}>
                    <label htmlFor={`rename-${item.id}`}>تغییر نام گفت‌وگو</label>
                    <input
                      id={`rename-${item.id}`}
                      autoFocus
                      value={draftTitle}
                      maxLength={60}
                      onChange={(event) => { setDraftTitle(event.target.value); setRenameError(false); }}
                      disabled={renamingId === item.id}
                    />
                    {renameError && <small role="alert">نام تغییر نکرد. دوباره امتحان کن.</small>}
                    <div className="conversation-title-editor-actions">
                      <button type="submit" disabled={renamingId === item.id}>{renamingId === item.id ? 'در حال ذخیره…' : 'ذخیره نام'}</button>
                      <button type="button" disabled={renamingId === item.id} onClick={() => { setEditingId(null); setRenameError(false); }}>انصراف</button>
                    </div>
                  </form>
                ) : (
                  <>
                    <button type="button" className="history-row" onClick={() => onOpenConversation(item.id)}>
                      <span className="history-date">
                        <span className="history-title-line">
                          <strong>{item.title || 'گفت‌وگوی بدون عنوان'}</strong>
                          {isCurrent && <em className="history-active-pill">فعال</em>}
                        </span>
                        <small>{formatConversationRelativeDate(item.updatedAt)}</small>
                      </span>
                      <span className="history-actions"><ChevronLeft size={19} /></span>
                    </button>
                    <button
                      type="button"
                      className="connected-rename"
                      disabled={Boolean(renamingId) || deletingId === item.id}
                      onClick={() => beginRename(item)}
                      aria-label={`تغییر نام ${item.title || 'گفت‌وگو'}`}
                    >
                      <Pencil size={16} aria-hidden="true" />
                    </button>
                    <button
                      type="button"
                      className="connected-delete"
                      disabled={deletingId === item.id || Boolean(renamingId)}
                      onClick={() => onDeleteConversation(item.id)}
                      aria-label={`حذف ${item.title || 'گفت‌وگو'}`}
                    >
                      {deletingId === item.id ? 'در حال حذف…' : 'حذف'}
                    </button>
                  </>
                )}
              </div>
            );
          })}
        </section>

        <section className="saved-history-section" aria-live="polite">
          <h2>پیام‌های ذخیره‌شده</h2>
          {bookmarkPhase === 'loading' && <p className="connected-status">در حال بارگذاری پیام‌ها…</p>}
          {bookmarkPhase === 'error' && (
            <div className="connected-notice" role="alert">
              <p>پیام‌های ذخیره‌شده بارگذاری نشدن.</p>
              <button type="button" onClick={onReloadBookmarks}>تلاش دوباره</button>
            </div>
          )}
          {bookmarkPhase === 'ready' && bookmarks.length === 0 && <p className="connected-status">هنوز پیامی ذخیره نکردی.</p>}
          {bookmarks.map((item) => (
            <button type="button" key={item.id} className="saved-history-entry" onClick={() => onOpenConversation(item.conversationId)}>
              <Bookmark size={18} fill="currentColor" aria-hidden="true" />
              <span>{item.content.length > 125 ? `${item.content.slice(0, 125)}…` : item.content}</span>
              <ChevronLeft size={18} aria-hidden="true" />
            </button>
          ))}
        </section>
      </div>
    </section>
  );
}

const MENU_ITEMS = [
  { label: 'گفت‌وگوهای قبلی', icon: HistoryIcon, route: 'history' },
  { label: 'نامهٔ همراه', icon: Mail, route: 'day' },
  { label: 'سبک گفت‌وگو', icon: SlidersHorizontal, route: 'approach' },
  { label: 'اسم من', icon: Info, route: 'name' },
  { label: 'حریم خصوصی', icon: Settings2, route: 'privacy' },
  { label: 'درباره', icon: Info, route: 'research' },
  { label: 'پشتیبانی و نظرها', icon: MessageSquare, action: 'support' },
  { label: 'پرسش‌نامهٔ حال روحی', icon: BarChart3, route: 'survey', dot: true },
];

function Drawer({ navigate, name, letters, notify, onLogout, onSwitchTester }) {
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
      <Home navigate={() => {}} name={name} letters={letters} underlay />
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

function ConversationSwitcherSheet({
  close,
  conversations = [],
  conversationId,
  isEmptyDraft,
  historyPhase,
  deletingId,
  renamingId,
  onSelectConversation,
  onStartNew,
  onRenameConversation,
  onDeleteConversation,
  onOpenFullHistory,
}) {
  const [editingId, setEditingId] = useState(null);
  const [draftTitle, setDraftTitle] = useState('');
  const [renameError, setRenameError] = useState(false);

  const submitRename = async (event, id) => {
    event.preventDefault();
    if (!draftTitle.trim() || [...draftTitle.trim()].length > 60) {
      setRenameError(true);
      return;
    }
    const ok = await onRenameConversation?.(id, draftTitle);
    if (ok) {
      setEditingId(null);
      setRenameError(false);
    } else {
      setRenameError(true);
    }
  };

  return (
    <>
      <button type="button" className="thread-sheet-backdrop" onClick={close} aria-label="بستن فهرست گفت‌وگوها" />
      <section className="thread-sheet" role="dialog" aria-modal="true" aria-label="گفت‌وگوهای من">
        <div className="thread-sheet-handle" aria-hidden="true" />
        <header className="thread-sheet-header">
          <div>
            <h2>گفت‌وگوهای من</h2>
            <p>بین موضوع‌های مختلف جابه‌جا شو یا گفت‌وگوی تازه‌ای باز کن.</p>
          </div>
          <button type="button" className="thread-sheet-close" onClick={close} aria-label="بستن">
            <X size={20} />
          </button>
        </header>

        <button
          type="button"
          className={`thread-new-action ${isEmptyDraft ? 'is-current' : ''}`}
          onClick={() => {
            if (isEmptyDraft) {
              close();
              return;
            }
            if (onStartNew()) close();
          }}
        >
          <span className="thread-new-icon" aria-hidden="true"><SquarePen size={18} /></span>
          <span className="thread-new-copy">
            <strong>گفت‌وگوی تازه</strong>
            <small>{isEmptyDraft ? 'الان در همین گفت‌وگوی تازه هستی' : 'برای شروع یک موضوع مستقل'}</small>
          </span>
          {isEmptyDraft && <Check size={18} className="thread-active-check" aria-hidden="true" />}
        </button>

        <div className="thread-sheet-list">
          {historyPhase === 'loading' && conversations.length === 0 && (
            <p className="thread-sheet-empty">در حال بارگذاری گفت‌وگوها…</p>
          )}
          {historyPhase === 'ready' && conversations.length === 0 && (
            <p className="thread-sheet-empty">هنوز گفت‌وگوی ذخیره‌شده‌ای نداری. اولین پیامت یک گفت‌وگوی تازه می‌سازه.</p>
          )}
          {conversations.slice(0, 15).map((item) => {
            const active = item.id === conversationId;
            return (
              <div key={item.id} className={`thread-sheet-row ${active ? 'is-active' : ''}`}>
                {editingId === item.id ? (
                  <form className="conversation-title-editor" onSubmit={(event) => submitRename(event, item.id)}>
                    <label htmlFor={`sheet-rename-${item.id}`}>تغییر نام گفت‌وگو</label>
                    <input
                      id={`sheet-rename-${item.id}`}
                      autoFocus
                      value={draftTitle}
                      maxLength={60}
                      disabled={renamingId === item.id}
                      onChange={(event) => { setDraftTitle(event.target.value); setRenameError(false); }}
                    />
                    {renameError && <small role="alert">نام تغییر نکرد. دوباره امتحان کن.</small>}
                    <div className="conversation-title-editor-actions">
                      <button type="submit" disabled={renamingId === item.id}>{renamingId === item.id ? 'در حال ذخیره…' : 'ذخیره نام'}</button>
                      <button type="button" disabled={renamingId === item.id} onClick={() => { setEditingId(null); setRenameError(false); }}>انصراف</button>
                    </div>
                  </form>
                ) : (
                  <>
                    <button
                      type="button"
                      className="thread-sheet-select"
                      onClick={() => {
                        onSelectConversation(item.id);
                        close();
                      }}
                    >
                      <span className="thread-sheet-meta">
                        <strong>{item.title || 'گفت‌وگوی بدون عنوان'}</strong>
                        <small>{formatConversationRelativeDate(item.updatedAt)}</small>
                      </span>
                      {active ? (
                        <span className="thread-current-badge">فعال</span>
                      ) : (
                        <ChevronLeft size={17} aria-hidden="true" />
                      )}
                    </button>
                    <button
                      type="button"
                      className="thread-sheet-rename"
                      disabled={Boolean(renamingId) || deletingId === item.id}
                      onClick={() => {
                        setEditingId(item.id);
                        setDraftTitle(item.title);
                        setRenameError(false);
                      }}
                      aria-label={`تغییر نام ${item.title || 'گفت‌وگو'}`}
                    >
                      <Pencil size={14} aria-hidden="true" />
                    </button>
                    <button
                      type="button"
                      className="thread-sheet-delete"
                      disabled={deletingId === item.id || Boolean(renamingId)}
                      onClick={() => onDeleteConversation(item.id)}
                      aria-label={`حذف ${item.title || 'گفت‌وگو'}`}
                    >
                      <Trash2 size={15} aria-hidden="true" />
                    </button>
                  </>
                )}
              </div>
            );
          })}
        </div>

        <footer className="thread-sheet-footer">
          <button
            type="button"
            onClick={() => {
              close();
              onOpenFullHistory();
            }}
          >
            <HistoryIcon size={17} aria-hidden="true" />
            <span>همهٔ گفت‌وگوها و پیام‌های ذخیره‌شده</span>
            <ChevronLeft size={17} aria-hidden="true" />
          </button>
        </footer>
      </section>
    </>
  );
}

function Chat({
  navigate,
  onboardingGreeting = '',
  conversationId = null,
  conversations = [],
  messages = [],
  pendingTurn,
  sending,
  chatLoading,
  chatError,
  historyPhase,
  deletingId,
  renamingId,
  onSendMessage,
  onRetryMessage,
  onRestoreChat,
  onNewConversation,
  onOpenConversation,
  onRenameConversation,
  onDeleteConversation,
  onReloadHistory,
  notify,
  bookmarks = [],
  bookmarkPhase,
  bookmarkBusyId,
  onToggleBookmark,
  onReloadBookmarks,
}) {
  const [draft, setDraft] = useState('');
  const [menuFor, setMenuFor] = useState(null);
  const [threadsOpen, setThreadsOpen] = useState(false);
  const [showScrollDown, setShowScrollDown] = useState(false);
  const scrollRef = useRef(null);
  const inputRef = useRef(null);
  const composerRef = useRef(null);
  const pressTimer = useRef(null);
  const lastScrolledThreadRef = useRef(undefined);
  const prevMsgCountRef = useRef(messages.length);

  const activeConversation = useMemo(
    () => conversations.find((item) => item.id === conversationId) || null,
    [conversations, conversationId],
  );
  const isEmptyDraft = !conversationId && messages.length === 0 && !pendingTurn && !chatLoading;
  const threadTitle = activeConversation?.title
    || (conversationId ? 'گفت‌وگوی فعلی' : 'گفت‌وگوی تازه');

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
    if (!composer) return undefined;
    const screen = composer.closest('.chat-screen');
    const syncLayout = () => {
      if (!screen) return;
      screen.style.setProperty('--composer-height', `${composer.offsetHeight}px`);
      if (window.visualViewport) {
        const keyboardInset = Math.max(0, window.innerHeight - window.visualViewport.height - window.visualViewport.offsetTop);
        screen.style.setProperty('--keyboard-offset', `${keyboardInset}px`);
      }
      resizeInput();
    };
    syncLayout();
    const observer = new ResizeObserver(syncLayout);
    observer.observe(composer);
    window.addEventListener('resize', syncLayout);
    window.visualViewport?.addEventListener('resize', syncLayout);
    return () => {
      observer.disconnect();
      window.removeEventListener('resize', syncLayout);
      window.visualViewport?.removeEventListener('resize', syncLayout);
    };
  }, []);

  const scrollToLatest = (behavior = 'smooth') => {
    const node = scrollRef.current;
    if (!node) return;
    node.scrollTo({ top: node.scrollHeight, behavior });
  };

  useEffect(() => {
    if (chatLoading) return;
    const threadChanged = lastScrolledThreadRef.current !== conversationId;
    const messageAdded = messages.length > prevMsgCountRef.current;
    lastScrolledThreadRef.current = conversationId;
    prevMsgCountRef.current = messages.length;

    if (threadChanged) {
      scrollToLatest('auto');
      setShowScrollDown(false);
    } else if (messageAdded || pendingTurn || sending) {
      scrollToLatest('smooth');
    }
  }, [conversationId, chatLoading, messages.length, pendingTurn, sending]);

  useEffect(() => () => window.clearTimeout(pressTimer.current), []);

  const handleScroll = () => {
    const node = scrollRef.current;
    if (!node) return;
    const distanceFromBottom = node.scrollHeight - node.scrollTop - node.clientHeight;
    setShowScrollDown(distanceFromBottom > 150);
  };

  const send = (text) => {
    const clean = text.trim();
    if (!clean || sending || pendingTurn || chatLoading) return;
    if ([...clean].length > 4000) {
      notify('پیام می‌تونه تا ۴۰۰۰ کاراکتر باشه.');
      return;
    }
    setDraft('');
    void onSendMessage(clean);
  };

  const chooseStarter = (promptText) => {
    setDraft(promptText);
    window.requestAnimationFrame(() => {
      inputRef.current?.focus();
    });
  };

  const openThreadsSheet = () => {
    if (historyPhase === 'idle' || historyPhase === 'error') {
      onReloadHistory?.();
    }
    setThreadsOpen(true);
  };

  const copyMessageQuick = async (text) => {
    try {
      await navigator.clipboard.writeText(text);
      notify('پیام کپی شد.');
    } catch {
      notify('کپی پیام ممکن نیست.');
    }
  };

  const pressStart = (id) => {
    window.clearTimeout(pressTimer.current);
    pressTimer.current = window.setTimeout(() => setMenuFor(id), 520);
  };
  const pressEnd = () => window.clearTimeout(pressTimer.current);
  const menuMessage = messages.find((message) => message.id === menuFor);
  const draftLength = [...draft].length;

  return (
    <section className="app-screen theme-dark chat-screen">
      <StatusBar dark />
      <header className="chat-header">
        <BackButton dark onClick={() => navigate('home')} />

        <button
          type="button"
          className={`chat-thread-switcher ${threadsOpen ? 'is-open' : ''}`}
          onClick={openThreadsSheet}
          aria-expanded={threadsOpen}
          aria-haspopup="dialog"
          aria-label={`تغییر گفت‌وگو — فعلی: ${threadTitle}`}
        >
          <span className="chat-thread-title">{threadTitle}</span>
          <ChevronDown size={16} strokeWidth={2.1} aria-hidden="true" />
        </button>

        <button
          type="button"
          className="chat-compose-button"
          disabled={sending || isEmptyDraft}
          onClick={() => { if (onNewConversation()) setDraft(''); }}
          aria-label="گفت‌وگوی جدید"
          title="گفت‌وگوی جدید"
        >
          <SquarePen size={19} strokeWidth={1.9} />
        </button>
      </header>

      <div className="messages-scroll" ref={scrollRef} onScroll={handleScroll} aria-live="polite">
        <div className="messages-list">
          {bookmarkPhase === 'error' && (
            <div className="connected-notice" role="alert">
              <p>وضعیت پیام‌های ذخیره‌شده در دسترس نیست.</p>
              <button type="button" onClick={onReloadBookmarks}>تلاش دوباره</button>
            </div>
          )}

          {chatLoading && (
            <div className="chat-loading-skeleton" role="status">
              <p className="connected-status">در حال بارگذاری پیام‌ها…</p>
              <div className="chat-skeleton-bubble chat-skeleton-bubble--user" aria-hidden="true" />
              <div className="chat-skeleton-bubble chat-skeleton-bubble--assistant" aria-hidden="true">
                <span />
                <span />
                <span />
              </div>
            </div>
          )}

          {!chatLoading && messages.length === 0 && !pendingTurn && !chatError && (
            onboardingGreeting ? (
              <div className="message-block message-assistant onboarding-welcome">
                <div className="assistant-presence" aria-hidden="true">
                  <Flower2 size={15} strokeWidth={1.75} />
                  <span>همراه</span>
                </div>
                <div className="message-text">
                  {onboardingGreeting.split(/\n\s*\n/).map((paragraph, index) => <p key={index}>{paragraph}</p>)}
                </div>
              </div>
            ) : (
              <div className="chat-sanctuary-empty">
                <span className="chat-sanctuary-icon" aria-hidden="true">
                  <Flower2 size={28} strokeWidth={1.4} />
                </span>
                <h2>هر چی توی ذهنته بهم بگو</h2>
                <p>این یک گفت‌وگوی مستقله. هر بار بخوای می‌تونی برگردی سراغش یا موضوع تازه‌ای باز کنی.</p>
                <div className="chat-starter-chips" aria-label="پیشنهاد برای شروع گفت‌وگو">
                  {CHAT_STARTER_PROMPTS.map((starter) => (
                    <button key={starter} type="button" onClick={() => chooseStarter(starter)}>
                      {starter}
                    </button>
                  ))}
                </div>
              </div>
            )
          )}

          {messages.map((message, idx) => {
            const isAssistant = message.role === 'assistant';
            const prevRole = idx > 0 ? messages[idx - 1].role : null;
            const showSpeaker = isAssistant && prevRole !== 'assistant';
            const isSaved = isAssistant && bookmarks.some((item) => item.id === message.id);
            const isSavingBookmark = bookmarkBusyId === message.id;

            return (
              <div
                key={message.id}
                className={`message-block message-${message.role} ${prevRole === message.role ? 'is-grouped' : ''}`}
              >
                {showSpeaker && (
                  <div className="assistant-presence" aria-hidden="true">
                    <Flower2 size={14} strokeWidth={1.8} />
                    <span>همراه</span>
                  </div>
                )}

                <div
                  className={`message-text ${menuFor === message.id ? 'is-highlighted' : ''}`}
                  onContextMenu={(event) => {
                    event.preventDefault();
                    if (isAssistant) setMenuFor(message.id);
                  }}
                  onPointerDown={() => isAssistant && pressStart(message.id)}
                  onPointerUp={pressEnd}
                  onPointerCancel={pressEnd}
                  onPointerMove={pressEnd}
                >
                  {message.content.split(/\n\s*\n/).map((paragraph, index) => (
                    <p key={index}>{paragraph}</p>
                  ))}
                </div>

                {isAssistant && (
                  <div className="assistant-inline-bar">
                    <button
                      type="button"
                      onClick={() => copyMessageQuick(message.content)}
                      aria-label="کپی پاسخ"
                    >
                      <Copy size={14} />
                    </button>
                    <button
                      type="button"
                      className={isSaved ? 'is-saved' : ''}
                      disabled={bookmarkPhase !== 'ready' || isSavingBookmark}
                      onClick={async () => {
                        const ok = await onToggleBookmark(message);
                        notify(ok ? (isSaved ? 'از ذخیره‌شده‌ها حذف شد.' : 'پیام ذخیره شد.') : 'پیام ذخیره نشد. دوباره تلاش کن.');
                      }}
                      aria-label={isSaved ? 'حذف از ذخیره‌شده‌ها' : 'ذخیره این پاسخ'}
                    >
                      <Bookmark size={14} fill={isSaved ? 'currentColor' : 'none'} />
                    </button>
                    <button
                      type="button"
                      onClick={() => setMenuFor(message.id)}
                      aria-label="گزینه‌های بیشتر پیام"
                    >
                      <MoreHorizontal size={15} />
                    </button>
                  </div>
                )}
              </div>
            );
          })}

          {pendingTurn && (
            <div className="message-block message-user connected-pending">
              <div className="message-text"><p>{pendingTurn.text}</p></div>
              <small>{sending ? 'در حال ارسال…' : 'پیامت ارسال نشده.'}</small>
            </div>
          )}

          {sending && (
            <div className="typing-indicator" role="status" aria-label="همراه در حال نوشتن پاسخ…">
              <Flower2 size={15} className="typing-flower" aria-hidden="true" />
              <span className="typing-dots" aria-hidden="true"><i /><i /><i /></span>
              <span className="typing-caption">در حال تامل…</span>
            </div>
          )}

          {chatError && (
            <div className="connected-notice" role="alert">
              <p>
                {chatError === 'TURN_ALREADY_PROCESSING'
                  ? 'پاسخ قبلیت هنوز آماده نشده. دوباره تلاش کن.'
                  : chatError === 'RESOURCE_NOT_FOUND'
                    ? 'این گفت‌وگو در دسترس نیست.'
                    : 'ارتباط برقرار نشد یا پاسخ نیومد. دوباره تلاش کن.'}
              </p>
              <div className="connected-notice-actions">
                {chatError === 'RESOURCE_NOT_FOUND' ? (
                  <button type="button" onClick={() => { if (onNewConversation()) setDraft(''); }}>گفت‌وگوی جدید</button>
                ) : pendingTurn ? (
                  <button type="button" disabled={sending} onClick={onRetryMessage}>تلاش دوباره</button>
                ) : (
                  <>
                    <button type="button" onClick={() => onRestoreChat?.()}>تلاش دوباره</button>
                    <button type="button" className="secondary-notice-btn" onClick={() => { if (onNewConversation()) setDraft(''); }}>گفت‌وگوی تازه</button>
                  </>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {showScrollDown && (
        <button
          type="button"
          className="scroll-bottom-button"
          onClick={() => scrollToLatest('smooth')}
          aria-label="رفتن به آخرین پیام"
        >
          <ArrowDown size={18} />
        </button>
      )}

      <form className="composer-wrap" ref={composerRef} onSubmit={(event) => { event.preventDefault(); send(draft); }}>
        {draftLength > 3400 && (
          <div className={`composer-char-counter ${draftLength > 4000 ? 'is-over' : ''}`} aria-live="polite">
            {draftLength.toLocaleString('fa-IR')} / ۴٬۰۰۰
          </div>
        )}
        <div className="composer">
          <button
            type="button"
            className="composer-plus"
            onClick={() => notify('ارسال فایل هنوز فعال نیست.')}
            aria-label="افزودن پیوست"
          >
            <Grid2X2Plus size={21} />
          </button>

          <textarea
            ref={inputRef}
            rows={1}
            wrap="soft"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            maxLength={4000}
            disabled={Boolean(pendingTurn) || sending || chatLoading}
            placeholder="برای همراه بنویس…"
            aria-label="متن پیام"
            onKeyDown={(event) => {
              if (event.key === 'Enter' && (event.ctrlKey || event.metaKey) && !event.nativeEvent.isComposing) {
                event.preventDefault();
                send(draft);
              }
            }}
          />

          <button
            type={draft.trim() ? 'submit' : 'button'}
            className={`voice-button send-button ${draft.trim() ? 'is-ready' : ''}`}
            disabled={Boolean(pendingTurn) || sending || chatLoading}
            onClick={!draft.trim() ? () => notify('ارسال صوتی هنوز فعال نیست.') : undefined}
            aria-label={draft.trim() ? 'فرستادن پیام' : 'پیام صوتی'}
          >
            {draft.trim() ? <Send size={19} /> : <AudioLines size={22} />}
          </button>
        </div>
      </form>

      {threadsOpen && (
        <ConversationSwitcherSheet
          close={() => setThreadsOpen(false)}
          conversations={conversations}
          conversationId={conversationId}
          isEmptyDraft={isEmptyDraft}
          historyPhase={historyPhase}
          deletingId={deletingId}
          renamingId={renamingId}
          onSelectConversation={onOpenConversation}
          onStartNew={() => {
            const ok = onNewConversation();
            if (ok) setDraft('');
            return ok;
          }}
          onRenameConversation={onRenameConversation}
          onDeleteConversation={onDeleteConversation}
          onOpenFullHistory={() => navigate('history')}
        />
      )}

      {menuMessage && (
        <MessageActions
          message={menuMessage}
          close={() => setMenuFor(null)}
          notify={notify}
          saved={bookmarks.some((item) => item.id === menuMessage.id)}
          saving={bookmarkBusyId === menuMessage.id}
          bookmarkReady={bookmarkPhase === 'ready'}
          onToggleBookmark={onToggleBookmark}
        />
      )}
    </section>
  );
}

function Insight(props) {
  // Route `insight` now shares the dedicated non-fabricated Personalized Letter reading surface.
  return <DayDetail {...props} />;
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
