import React, { useEffect, useRef, useState } from 'react';
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
      <button type="button" className="reflection-card" onClick={() => navigate('chat')}>
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
    notify('پاسخت ثبت شد.');
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

function DayDetail({ back, navigate }) {
  return (
    <section className="app-screen theme-dark day-screen scrolling-screen">
      <div className="day-art-panel">
        <StatusBar />
        <BackButton onClick={back} />
        <ScribbleArt />
      </div>
      <div className="day-content">
        <button type="button" className="transcript-button" onClick={() => navigate('chat')}>
          <FileText size={23} />
          <span>رفتن به متن گفت‌وگو</span>
        </button>
        <h1>شنبه، ۳۱ اکتبر</h1>
        <p>امروز درباره‌ی رابطه‌ای حرف زدی که ذهنت را درگیر کرده بود؛ درباره‌ی روزمرگی و چیزهایی که سنگین می‌شدند. اولین قدم را برداشتی و نامِ آنچه در ذهنت است را گفتی. در ادامه هم، با سرعت خودت پیش می‌رویم.</p>
        <button type="button" className="letter-card" onClick={() => navigate('insight')}>
          <span>نامه</span>
          <strong>فهمیدن و فکرکردنِ بیش‌ازحد، دوقلوها هستند</strong>
          <MiniEnvelope />
        </button>
      </div>
    </section>
  );
}

function History({ back, navigate }) {
  return (
    <section className="app-screen theme-dark history-screen">
      <StatusBar dark />
      <BackButton dark onClick={back} />
      <h1 className="history-title">تاریخچه</h1>
      <div className="history-content">
        <h2>این هفته</h2>
        <div className="weekly-progress">
          <strong>۱/۲</strong>
          <p>روز گفت‌وگو تا بازشدن خلاصه‌ی هفتگی با بینش‌ها و الگوها</p>
        </div>
        <button type="button" className="history-row" onClick={() => navigate('day')}>
          <span className="history-date"><strong>شنبه</strong><small>۳۱ اکتبر</small></span>
          <span className="history-actions">
            <span className="history-chat-count" onClick={(event) => { event.stopPropagation(); navigate('chat'); }}><MessageSquare size={23} /><b>۱</b></span>
            <span className="history-mail"><Mail size={25} /></span>
            <ChevronLeft size={21} />
          </span>
        </button>
      </div>
    </section>
  );
}

const MENU_ITEMS = [
  { label: 'تاریخچه', icon: HistoryIcon, route: 'history' },
  { label: 'ترجیحات', icon: SlidersHorizontal, route: 'approach' },
  { label: 'حساب و داده‌ها', icon: Settings2, route: 'privacy' },
  { label: 'درباره', icon: Info, route: 'research' },
  { label: 'پشتیبانی و بازخورد', icon: MessageSquare, action: 'support' },
  { label: 'تمرین‌ها', icon: Flower2, route: 'home' },
  { label: 'پرسش‌نامه‌ی سلامت', icon: BarChart3, route: 'survey', dot: true },
];

function Drawer({ navigate, name, notify, reset }) {
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
          <button type="button" onClick={() => { reset(); navigate('welcome'); }}><LogOut size={23} /><span>خروج</span></button>
        </div>
      </aside>
    </div>
  );
}

function MessageActions({ message, close, notify, onRegenerate }) {
  const text = message.paragraphs.join('\n\n');
  const copy = async () => {
    try { await navigator.clipboard.writeText(text); } catch { /* clipboard can be unavailable in preview */ }
    notify('پیام کپی شد.');
    close();
  };
  const read = () => {
    if ('speechSynthesis' in window) {
      window.speechSynthesis.cancel();
      const utterance = new SpeechSynthesisUtterance(text);
      utterance.lang = 'fa-IR';
      utterance.rate = 0.9;
      window.speechSynthesis.speak(utterance);
    }
    notify('خواندن پیام شروع شد.');
    close();
  };
  return (
    <>
      <button type="button" className="message-menu-backdrop" onClick={close} aria-label="بستن گزینه‌ها" />
      <div className="message-actions" role="menu">
        <button type="button" role="menuitem" onClick={copy}><Copy size={21} /><span>کپی پیام</span></button>
        <button type="button" role="menuitem" onClick={() => { notify('پیام ذخیره شد.'); close(); }}><Bookmark size={21} /><span>ذخیره پیام</span></button>
        <button type="button" role="menuitem" onClick={read}><Play size={21} /><span>خواندن با صدا</span></button>
        <button type="button" role="menuitem" onClick={() => { onRegenerate(message.id); close(); }}><RotateCcw size={21} /><span>ساختن دوباره‌ی پاسخ</span></button>
        <button type="button" role="menuitem" onClick={() => { notify('گزارش پیام ثبت شد.'); close(); }}><Flag size={21} /><span>گزارش پیام</span></button>
      </div>
    </>
  );
}

function Chat({ navigate, messages, setMessages, notify }) {
  const [draft, setDraft] = useState('');
  const [typing, setTyping] = useState(false);
  const [listening, setListening] = useState(false);
  const [menuFor, setMenuFor] = useState(null);
  const scrollRef = useRef(null);
  const sentRef = useRef(false);
  const pressTimer = useRef(null);

  useEffect(() => {
    if (sentRef.current && scrollRef.current) {
      scrollRef.current.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' });
    }
  }, [messages, typing]);

  useEffect(() => () => window.clearTimeout(pressTimer.current), []);

  const send = (text) => {
    const clean = text.trim();
    if (!clean || typing) return;
    sentRef.current = true;
    setDraft('');
    setMessages((current) => [...current, { id: Date.now(), role: 'user', paragraphs: [clean] }]);
    setTyping(true);
    window.setTimeout(() => {
      setMessages((current) => [...current, {
        id: Date.now() + 1,
        role: 'assistant',
        paragraphs: ['شنیدمت. همین که این حس را به زبان می‌آوری قدم مهمی است.', 'اگر بخواهی، می‌توانیم آرام ببینیم سخت‌ترین بخش این موقعیت برای تو کدام است.'],
      }]);
      setTyping(false);
    }, 850);
  };

  const regenerate = (id) => {
    setMessages((current) => current.map((message) => message.id === id ? {
      ...message,
      paragraphs: ['بیایید این را از زاویه‌ای دیگر ببینیم. احساست ارزش شنیده‌شدن دارد و لازم نیست برایش قضاوت شوی.', ...message.paragraphs.slice(1)],
    } : message));
    notify('پاسخ تازه‌ای ساخته شد.');
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
        <button type="button" className="sparkle-button" onClick={() => navigate('insight')} aria-label="مشاهده‌ی بینش"><Sparkles size={27} fill="currentColor" /></button>
      </header>
      <div className="messages-scroll" ref={scrollRef}>
        <div className="messages-list">
          {messages.map((message) => (
            <div key={message.id} className={`message-block message-${message.role}`}>
              <div
                className={`message-text ${menuFor === message.id ? 'is-highlighted' : ''}`}
                onContextMenu={(event) => { event.preventDefault(); if (message.role === 'assistant') setMenuFor(message.id); }}
                onPointerDown={() => message.role === 'assistant' && pressStart(message.id)}
                onPointerUp={pressEnd}
                onPointerCancel={pressEnd}
                onPointerMove={pressEnd}
              >
                {message.paragraphs.map((paragraph, index) => <p key={index}>{paragraph}</p>)}
              </div>
              {message.suggestions && (
                <div className="suggestions-row">
                  {message.suggestions.map((suggestion) => <button type="button" key={suggestion} onClick={() => send(suggestion)}>{suggestion}</button>)}
                </div>
              )}
              {message.insight && (
                <div className="insight-wrap">
                  <span className="insight-created"><Sparkles size={16} />همراه یک بینش تازه برایت ساخت</span>
                  <button type="button" className="insight-card" onClick={() => navigate('insight')}>
                    <strong>احساسات او را بیشتر از احساسات خودت محافظت می‌کنی</strong>
                    <InsightSpiral />
                  </button>
                </div>
              )}
            </div>
          ))}
          {typing && <div className="typing-indicator" aria-label="همراه در حال نوشتن است"><i /><i /><i /></div>}
        </div>
      </div>
      <form className="composer-wrap" onSubmit={(event) => { event.preventDefault(); send(draft); }}>
        {listening && <span className="listening-label">در حال شنیدن…</span>}
        <div className="composer">
          <button type="button" className="composer-plus" onClick={() => notify('افزودن پیوست در نسخه‌ی نمایشی غیرفعال است.')} aria-label="افزودن پیوست"><Grid2X2Plus size={23} /></button>
          <input value={draft} onChange={(event) => setDraft(event.target.value)} placeholder={listening ? 'در حال شنیدن…' : 'پیامت را بنویس'} aria-label="متن پیام" />
          {draft.trim() ? (
            <button type="submit" className="voice-button send-button" aria-label="فرستادن پیام"><Send size={21} /></button>
          ) : (
            <button type="button" className={`voice-button ${listening ? 'is-listening' : ''}`} onClick={() => setListening((value) => !value)} aria-label="ورودی صوتی"><AudioLines size={25} /></button>
          )}
        </div>
      </form>
      {menuMessage && <MessageActions message={menuMessage} close={() => setMenuFor(null)} notify={notify} onRegenerate={regenerate} />}
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
