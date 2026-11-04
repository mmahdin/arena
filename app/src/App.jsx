import React, { useMemo, useState } from 'react';
import {
  ArrowBack, Close, Check, Mic, Info, Search, Wave, Clock, Calendar, Dots,
  Eye, Camera, Video, Sprout, Compass, History, Gear, Google, Apple, Logo, Hourglass,
} from './icons.jsx';
import * as D from './data.js';
import { TopBar, OptionList, Foot, TextareaField, InfoCard, DarkCardScreen, StatScreen, GradCardScreen, ReflectionScreen, SearchBox } from './components.jsx';

/* ================= Onboarding flow definition (photo_61 → photo_23) ================= */
const STEPS = [
  { t: 'dark', card: D.DARK_CARDS[0] },
  { t: 'dark', card: D.DARK_CARDS[1] },
  { t: 'dark', card: D.DARK_CARDS[2] },
  { t: 'dark', card: D.DARK_CARDS[3] },
  { t: 'name', prog: 1 },                    // 57
  { t: 'splash' },                           // 56
  { t: 'language', prog: 1 },                // 55
  { t: 'roadmap', prog: 1 },                 // 54
  { t: 'welcome', prog: 1 },                 // 53
  { t: 'gender', prog: 1, q: 'جنسیتت را انتخاب کن', options: D.GENDERS },   // 52
  { t: 'age', prog: 1 },                     // 51
  { t: 'signup', prog: 1 },                  // 50
  { t: 'stat', variant: 'orange', num: '۲۲٫۱٪', text: 'تقريباً یک نفر از هر چهار نفر، جلسه‌ای را', hl: 'تغییر‌دهنده زندگی', tail: 'نامیده است.', quote: 'بالاخره اپی که احساس کردم واقعاً گوش می‌دهد.', source: 'بابی جی. · گوگل Play' }, // 49
  { t: 'country', prog: 2 },                 // 48
  { t: 'textarea', prog: 2, q: 'درباره خودت به ما بگو.', sub: 'هر چه به ذهنت می‌رسد بنویس؛ بعداً با هم مرورش می‌کنیم.', ph: 'شروع کن به نوشتن…', suggestion: 'چه کسی کنار توست — و چه کسی نیست؟', skip: true }, // 47
  { t: 'reflection', prog: 2, eyebrow: 'این چیزی است که می‌شنویم', text: 'عشق جایی است که توجه‌ات اول می‌رود — و گاهی همان‌جا می‌ماند.' }, // 46
  { t: 'options', prog: 2, q: 'آیا قبلاً روان‌درمانی یا مربی‌گری داشته‌ای؟', options: D.THERAPY_OPTIONS, def: 'خیر' }, // 45
  { t: 'textarea', prog: 2, q: 'چه چیزی امروز تو را به این‌جا آورده؟', sub: 'هر چه دوست داری بنویس؛ بعداً با هم مرورش می‌کنیم.', ph: 'بنویس…', suggestion: 'چه چیزی الان بیشتر از همه روی ذهنت سنگینی می‌کند؟' }, // 44
  { t: 'reflection', prog: 2, eyebrow: 'تو در دستان خوبی هستی', text: 'این فقط شروع است — و ما در تمام مسیر کنارت هستیم.' }, // 43
  { t: 'textarea', prog: 2, q: 'نگرانی‌ای درباره انجام این کار برای اولین بار داری؟', ph: 'بنویس…', skip: true }, // 42
  { t: 'options', prog: 2, q: 'کدام‌یک بهتر توصیف می‌کند چیزی را که با خود حمل می‌کنی؟', options: D.CARRYING_OPTIONS }, // 41
  { t: 'slider', prog: 2, q: 'چقدر می‌خواهی چیزها تغییر کنند یا متفاوت باشند؟' }, // 40
  { t: 'reflection', prog: 2, eyebrow: 'و ما آن را می‌شنویم', text: 'منتظر پیامی که هرگز نمی‌آید…' }, // 39
  { t: 'stat', variant: 'blue', num: '۶۸٪', text: 'از کاربران گفتند جلسات برایشان', hl: 'ژرف‌بینانه', tail: 'بوده است.', quote: 'هر جلسه باعث می‌شد چیز تازه‌ای درباره خودم ببینم.', source: 'کارا دی.' }, // 38
  { t: 'options', prog: 2, q: 'اگر اینیک هم سبک‌تر می‌شد، چه تغییری در روزت ایجاد می‌کرد؟', options: D.EASED_OPTIONS, skip: true }, // 37
  { t: 'gradcard', variant: 'orange-card', title: 'جلسات می‌توانند بین ۵ تا ۳۰ دقیقه باشند.', sub: 'کوتاه و متمرکز، یا طولانی و عمیق — هر بار خودت انتخاب می‌کنی.' }, // 36
  { t: 'options', prog: 2, q: 'آیا اخیراً روزی بوده که کمتر احساس سنگینی کنی؟', options: D.LESS_HEAVY_OPTIONS }, // 35
  { t: 'options', prog: 2, q: 'اگر امروز فقط یک چیز یک پله بهتر شود، چه می‌خواهی باشد؟', options: D.MOVE_UP_OPTIONS }, // 34
  { t: 'stat', variant: 'green', num: '۶۲٫۴٪', text: 'از جلسات خارج از ساعت کاری ۹ تا ۵ انجام می‌شود.', quote: 'جایش را در شکاف‌های روزم پیدا می‌کند — درست همان‌جا که بیشتر بهش نیاز دارم.', source: 'سارا اس.' }, // 33
  { t: 'options', prog: 2, q: 'به زمانی فکر کن که بیشتر از همیشه احساس می‌کردی خودت بودی…', options: D.LIKE_YOURSELF_OPTIONS }, // 32
  { t: 'options', prog: 2, q: 'حتی اگر اوضاع سخت بماند… چه نوع آدمی می‌خواهی باشی در میانه آن؟', options: D.PERSON_OPTIONS }, // 31
  { t: 'gradcard', variant: 'blue-card', title: 'این یک جای خصوصی و بدون عجله است.', sub: 'آنچه این‌جا می‌گویی، از این فضا خارج نمی‌شود.' }, // 30
  { t: 'stat', variant: 'green', num: '۵۷٫۱٪', text: 'گفتند در جلسه احساس', hl: 'دیده شدن', tail: 'کرده‌اند.', quote: 'برای اولین بار احساس کردم واقعاً دیده می‌شوم.', source: 'اپ استور' }, // 29
  { t: 'phq', prog: 3, start: 0 },           // 28
  { t: 'phq', prog: 3, start: 0, answered: [1] }, // 27
  { t: 'options', prog: 3, q: 'چطور مسیر را پیدا کردی؟', options: D.FOUND_OPTIONS, skip: true }, // 26
  { t: 'phq', prog: 3, start: 2, toast: true },  // 25
  { t: 'phq2', prog: 3 },                    // 24
  { t: 'options', prog: 3, q: 'کدام‌یک؟', options: D.AI_OPTIONS, skip: true }, // 23
];

const persianNum = (n) => n.toLocaleString('fa-IR');

/* ================= App ================= */
export default function App() {
  const [phase, setPhase] = useState('onb');       // 'onb' | 'app'
  const [step, setStep] = useState(0);
  const [screen, setScreen] = useState('home');    // home|guides|detail|expressions|focus|duration|intro|chat
  const [guideId, setGuideId] = useState('marcus');
  const [drawer, setDrawer] = useState(false);
  const [about, setAbout] = useState(false);
  const [hasFocus, setHasFocus] = useState(false);
  const [name, setName] = useState('مهدی');
  const [ans, setAns] = useState({});               // onboarding answers
  const [toast, setToast] = useState(false);

  const guide = D.GUIDES.find((g) => g.id === guideId) || D.GUIDES[10];
  const next = () => {
    if (phase === 'onb') {
      if (step + 1 >= STEPS.length) { setPhase('app'); setScreen('home'); }
      else setStep(step + 1);
    }
  };
  const back = () => {
    if (phase === 'onb') { if (step > 0) setStep(step - 1); }
    else setScreen('home');
  };
  const set = (k, v) => setAns((a) => ({ ...a, [k]: v }));

  const onbScreen = useMemo(() => {
    if (phase !== 'onb') return null;
    const s = STEPS[step];
    const key = `s${step}`;
    const val = ans[key];
    const setVal = (v) => set(key, v);
    const words = typeof val === 'string' ? val.trim().split(/\s+/).filter(Boolean).length : 0;

    switch (s.t) {
      case 'dark':
        return <DarkCardScreen card={s.card} onNext={next} />;

      case 'name':
        return (
          <div className="screen light">
            <TopBar onBack={back} prog={s.prog} />
            <h1 className="display q-title">چه نامی صدات کنیم؟</h1>
            <div className="field-card" style={{ marginTop: 14 }}>
              <input className="text-input" value={name} onChange={(e) => setName(e.target.value)} placeholder="نام تو" />
            </div>
            <InfoCard>نامت فقط روی این دستگاه ذخیره می‌شود و با هیچ‌کس به اشتراک گذاشته نمی‌شود.</InfoCard>
            <Foot><button className="btn btn-black" disabled={!name.trim()} onClick={next}>ادامه</button></Foot>
          </div>
        );

      case 'splash':
        return (
          <div className="screen splash">
            <div className="splash-inner">
              <div className="brand-big">
                <Hourglass size={44} style={{ color: '#2b2b2d' }} />
                <span className="word">مسیر</span>
              </div>
              <div className="splash-card">
                <h2 className="display">مسیر تو از این‌جا شروع می‌شود.</h2>
                <p>ظرف فقط پنج دقیقه، با چند سؤال ساده، راهنمای هوش مصنوعیِ مناسبِ تو را پیدا می‌کنیم.</p>
                <button className="btn btn-black" onClick={next}>شروع</button>
              </div>
            </div>
          </div>
        );

      case 'language':
        return (
          <div className="screen light">
            <TopBar onBack={back} prog={s.prog} />
            <h1 className="display q-title">چه زبانی را ترجیح می‌دهی؟</h1>
            <OptionList options={D.LANGUAGES} value={val ?? 'فارسی'} onChange={setVal} />
            <Foot><button className="btn btn-black" onClick={next}>ادامه</button></Foot>
          </div>
        );

      case 'roadmap':
        return (
          <div className="screen light">
            <TopBar onBack={back} prog={s.prog} />
            <h1 className="display q-title">ببین چند دقیقه آینده چطور می‌گذرد</h1>
            <div className="steps">
              {['به چند سؤال درباره خودت جواب بده', 'یک راهنمای هوش مصنوعی انتخاب کن', 'اولین حوزه تمرکزت را بساز', 'اولین جلسه‌ات را شروع کن', 'پیشرفت‌ات را دنبال کن'].map((t, i) => (
                <div className="step-row" key={t}><span className="step-num">{persianNum(i + 1)}</span><span>{t}</span></div>
              ))}
            </div>
            <Foot>
              <button className="btn btn-black" onClick={next}>شروع کنیم</button>
              <button className="btn btn-outline">ورود</button>
            </Foot>
          </div>
        );

      case 'welcome':
        return (
          <div className="screen pastel">
            <div style={{ height: 54 }} />
            <h1 className="display q-title">خوش آمدی به مسیر، {name}.</h1>
            <div className="white-card">
              <p>همه‌چیز آماده است. با چند سؤال ساده، راهنمای درست‌ات را پیدا می‌کنیم و اولین حوزه تمرکزت را می‌سازیم.</p>
            </div>
            <Foot><button className="btn btn-black" onClick={next}>ادامه</button></Foot>
          </div>
        );

      case 'gender':
      case 'options':
        return (
          <div className="screen light">
            <TopBar onBack={back} prog={s.prog} />
            <h1 className="display q-title">{s.q}</h1>
            <OptionList options={s.options} value={val ?? s.def} onChange={setVal} />
            <Foot>
              <button className="btn btn-black" disabled={val === undefined && !s.def && !s.skip} onClick={next}>ادامه</button>
              {s.skip && <button className="btn btn-outline" onClick={next}>رد کردن</button>}
            </Foot>
          </div>
        );

      case 'age':
        return (
          <div className="screen light">
            <TopBar onBack={back} prog={s.prog} />
            <h1 className="display q-title">چند سال داری؟</h1>
            <p className="q-sub">به ما کمک می‌کند پیشنهادهایمان را متناسب با تو کنیم.</p>
            <div className="field-card">
              <input className="text-input" type="number" inputMode="numeric" value={val ?? ''} onChange={(e) => setVal(e.target.value)} placeholder="سن تو" />
            </div>
            <InfoCard>سن تو فقط برای شخصی‌سازی استفاده می‌شود.</InfoCard>
            <Foot><button className="btn btn-black" disabled={!val} onClick={next}>ادامه</button></Foot>
          </div>
        );

      case 'signup':
        return (
          <div className="screen light">
            <TopBar onBack={back} prog={s.prog} />
            <h1 className="display q-title">پیشرفت‌ات را ذخیره کن</h1>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10, margin: '14px 20px 0' }}>
              <button className="auth-btn"><Google /> ادامه با گوگل</button>
              <button className="auth-btn"><Apple /> ادامه با اپل</button>
            </div>
            <div className="divider">یا با ایمیل‌ات ادامه بده</div>
            <div className="field-card">
              <input className="text-input" type="email" value={val ?? ''} onChange={(e) => setVal(e.target.value)} placeholder="ایمیل تو" />
            </div>
            <Foot>
              <button className="btn btn-black" disabled={!val} onClick={next}>ادامه</button>
              <p className="legal">با ادامه، با <a>شرایط سرویس</a> و <a>حریم خصوصی</a> ما موافقی.</p>
            </Foot>
          </div>
        );

      case 'country': {
        const q = (val?.q || '').trim();
        const list = D.COUNTRIES.filter((c) => c.includes(q) || q === '');
        return (
          <div className="screen light">
            <TopBar onBack={back} prog={s.prog} />
            <h1 className="display q-title">کشور محل اقامتت را انتخاب کن</h1>
            <SearchBox value={val?.q ?? ''} onChange={(v) => set(key, { ...(val || {}), q: v })} placeholder="جستجوی کشور" />
            <OptionList options={list} value={val?.sel} onChange={(v) => set(key, { ...(val || {}), sel: v })} />
            <Foot><button className="btn btn-black" disabled={!val?.sel} onClick={next}>ادامه</button></Foot>
          </div>
        );
      }

      case 'textarea':
        return (
          <div className="screen light">
            <TopBar onBack={back} prog={s.prog} />
            <h1 className="display q-title">{s.q}</h1>
            {s.sub && <p className="q-sub">{s.sub}</p>}
            <TextareaField value={val || ''} onChange={setVal} placeholder={s.ph} words={words} suggestion={s.suggestion} />
            <Foot>
              <button className="btn btn-black" disabled={words === 0 && !s.skip} onClick={next}>ادامه</button>
              {s.skip && <button className="btn btn-outline" onClick={next}>رد کردن</button>}
            </Foot>
          </div>
        );

      case 'reflection':
        return <ReflectionScreen eyebrow={s.eyebrow} text={s.text} onNext={next} onBack={back} />;

      case 'slider': {
        const v = val ?? 1;
        return (
          <div className="screen light">
            <TopBar onBack={back} prog={s.prog} />
            <h1 className="display q-title">{s.q}</h1>
            <div className="slider-wrap">
              <div className="slider-labels"><span>اصلاً</span><span>کمی</span><span>خیلی</span></div>
              <div className="slider-track">
                <div className="slider-fill" style={{ insetInlineEnd: `${(2 - v) * 50}%` }} />
                <div className="slider-handle" style={{ insetInlineStart: `${v * 50}%` }} />
                <input className="slider-input" type="range" min="0" max="2" step="1" value={v}
                  onChange={(e) => setVal(Number(e.target.value))} />
              </div>
            </div>
            <Foot><button className="btn btn-black" onClick={next}>ادامه</button></Foot>
          </div>
        );
      }

      case 'stat':
        return (
          <StatScreen variant={s.variant} num={s.num}
            text={s.text} hl={s.hl} tail={s.tail} quote={s.quote} source={s.source} onNext={next} />
        );

      case 'gradcard':
        return <GradCardScreen variant={s.variant} title={s.title} sub={s.sub} onNext={next} />;

      case 'phq':
      case 'phq2': {
        const items = s.t === 'phq2' ? [2, 3] : [s.start];
        const answered = s.answered || [];
        const cur = items[0];
        const done = ans[key] || {};
        return (
          <div className="screen light">
            {s.toast && toast && <div className="toast">مسیر این اسکرین‌شات را تشخیص داد</div>}
            <TopBar onBack={back} prog={s.prog} />
            <h1 className="display q-title" style={{ fontSize: 24 }}>در دو هفته گذشته، چند وقت تحت تأثیر این مشکلات بودی؟</h1>
            {answered.map((ai) => (
              <div className="phq-collapsed" key={ai}>
                <span>{D.PHQ_ITEMS[ai]}</span><span className="ans">— {done[ai] || 'اصلاً'}</span>
              </div>
            ))}
            {items.map((it, i) => (
              <div className="phq-item" key={it}>
                <div className="phq-q">{D.PHQ_ITEMS[it]}</div>
                <OptionList options={D.PHQ_OPTIONS} value={done[it]} onChange={(v) => { set(key, { ...done, [it]: v }); if (s.toast) setToast(true); }} />
              </div>
            ))}
            <Foot><button className="btn btn-black" disabled={!done[cur]} onClick={next}>ادامه</button></Foot>
          </div>
        );
      }

      default:
        return null;
    }
  }, [phase, step, ans, name, toast]);

  /* ---------- main app screens ---------- */
  const home = (
    <div className="screen light">
      <div className={`home-head ${hasFocus ? 'head-forest' : 'head-sunset'}`}>
        <button className="hamb" onClick={() => setDrawer(true)} aria-label="منو"><i /><i /><i /></button>
        <h1 className="display greet">عصر بخیر، محمدمهدی</h1>
      </div>
      <div className="home-body">
        {hasFocus ? (
          <button className="home-card focus-card" style={{ textAlign: 'right', width: '100%' }} onClick={() => setScreen('focus')}>
            <div className="fc-top">
              <h3 className="display fc-title">اضطراب در درگیری رابطه</h3>
              <img className="avatar" src={guide.img} alt={guide.name} />
            </div>
            <p className="fc-desc">وقتی با شریک عاطفی‌ام جر‌وبحث می‌شود، قلبم تند می‌زند و نمی‌توانم درست فکر کنم.</p>
            <span className="chip-pink">عشق</span>
            <div className="fc-time"><Clock /> ۱۷ دقیقه پیش</div>
          </button>
        ) : (
          <div className="home-card">
            <div className="sprout"><Sprout /></div>
            <h3 className="display">اولین حوزه تمرکزت را شروع کن</h3>
            <p className="sub">با چند سؤال کوتاه، راهنمای درست‌ات را پیدا می‌کنیم و شروع می‌کنیم.</p>
          </div>
        )}
        <div className="spacer" />
        <button className="btn btn-black" onClick={() => setScreen('guides')}>شروع حوزه تمرکز جدید</button>
      </div>
      {drawer && (
        <>
          <div className="drawer-overlay" onClick={() => setDrawer(false)} />
          <div className="drawer">
            <Logo />
            <button className="drawer-item active"><Compass /> حوزه‌های تمرکز</button>
            <button className="drawer-item"><History /> جلسات گذشته</button>
            <div className="spacer" />
            <button className="drawer-item"><Gear /> تنظیمات</button>
          </div>
        </>
      )}
    </div>
  );

  const guides = (
    <div className="screen dark">
      <div className="guides-head">
        <button className="icon-btn" onClick={() => setScreen('home')}><ArrowBack size={22} /></button>
        <h1 className="display">راهنمای هوش مصنوعی‌ات را انتخاب کن</h1>
        <span className="icon-btn" style={{ pointerEvents: 'none' }}><Wave size={20} /></span>
      </div>
      <div className="guide-list">
        {D.GUIDES.map((g, i) => (
          <div className="guide-card" key={g.id}>
            <img src={g.img} alt={g.name} />
            <div className="shade" />
            {g.locked && <span className="lock-chip"><LockIco /></span>}
            <button className="select-pill" onClick={() => { setGuideId(g.id); setScreen('detail'); }}>انتخاب {g.name}</button>
            <div className="meta">
              <div className="display name">{g.name}</div>
              <p className="desc">{g.desc}</p>
              <div className="tags">{g.tags.map((t) => <span className="tag" key={t}>{t}</span>)}</div>
            </div>
            <span className="wave-chip"><Wave size={18} /></span>
          </div>
        ))}
      </div>
    </div>
  );

  const detail = (
    <div className="screen dark">
      <div className="detail-photo">
        <img src={guide.img} alt={guide.name} />
        <button className="icon-btn x" style={{ background: 'rgba(0,0,0,.4)' }} onClick={() => setScreen('guides')}><Close size={20} /></button>
      </div>
      <div className="detail-body">
        <div className="eyebrow">راهنمای تو</div>
        <div className="display name">{guide.name}</div>
        <div className="tags">{guide.tags.map((t) => <span className="tag" key={t}>{t}</span>)}</div>
        <div className="detail-listen"><Wave size={18} /> گوش کن</div>
        {guide.quote && <p className="display detail-quote">«{guide.quote}»</p>}
        <button className="about-block" style={{ textAlign: 'right' }} onClick={() => setAbout(true)}>
          <h3 className="display">درباره من</h3>
          <p>{guide.aboutText}</p>
        </button>
        <button className="btn btn-white" onClick={() => setScreen('expressions')}>انتخاب {guide.name}</button>
      </div>
      {about && (
        <div className="screen modal-screen" style={{ position: 'absolute' }}>
          <div className="modal-card">
            <h3 className="display">درباره من</h3>
            <p>{guide.aboutText}</p>
            <h3 className="display">چطور با هم کار می‌کنیم</h3>
            <p>{guide.howText}</p>
            <button className="btn btn-white" onClick={() => { setAbout(false); setScreen('expressions'); }}>انتخاب {guide.name}</button>
          </div>
        </div>
      )}
    </div>
  );

  const expressions = (
    <div className="screen expr-screen light">
      <div style={{ display: 'grid', gridTemplateColumns: '40px 1fr 40px', padding: '14px 16px 0', alignItems: 'center' }}>
        <span />
        <span />
        <button className="icon-btn" onClick={() => setScreen('detail')} style={{ justifySelf: 'end' }}><Close size={22} /></button>
      </div>
      <div className="expr-hero">
        <div className="rings">
          <span className="ring" style={{ width: 250, height: 250 }} />
          <span className="ring" style={{ width: 190, height: 190, background: '#cfe6ec' }} />
          <span className="ring" style={{ width: 135, height: 135, background: '#bfe0e8' }} />
        </div>
        <span className="side-chip" style={{ right: '16%', top: '30%' }}><Camera /></span>
        <span className="side-chip" style={{ left: '16%', top: '62%' }}><Video /></span>
        <span className="eye-core"><Eye size={34} /></span>
      </div>
      <div className="expr-body">
        <div className="eyebrow">بیان چهره</div>
        <h2 className="display">نوعی تازه از هماهنگی</h2>
        <p>در طول جلسه، حالت چهره‌ات به راهنمایت کمک می‌کند تا با تو هماهنگ‌تر شود.</p>
        <div className="expr-data-card">
          <h4>با داده‌های تو چه می‌کنیم</h4>
          <div className="ok"><Check size={15} /><span>هرگز داده‌ات را نمی‌فروشیم.</span></div>
        </div>
        <p className="expr-note">می‌توانی هر زمان بیان چهره را در تنظیمات خاموش کنی.</p>
        <button className="btn btn-black" onClick={() => { setHasFocus(true); setScreen('home'); }}>فعال کردن بیان چهره</button>
        <div style={{ height: 10 }} />
        <button className="btn btn-outline" onClick={() => { setHasFocus(true); setScreen('home'); }}>الان نه</button>
      </div>
    </div>
  );

  const focusDetail = (
    <div className="screen light" style={{ position: 'relative' }}>
      <div className="fd-head">
        <button className="icon-btn" onClick={() => setScreen('home')}><ArrowBack size={22} /></button>
        <h1 className="display fd-title">اضطراب در درگیری رابطه</h1>
        <p className="fd-desc">وقتی با شریک عاطفی‌ام جر‌وبحث می‌شود، قلبم تند می‌زند و نمی‌توانم نفس عمیق بکشم.</p>
      </div>
      <div className="fd-body">
        <div className="review-card">
          <div className="rc-title">در حال بررسی جلسه‌ات</div>
          <div className="rc-sub"><span className="dots3"><i /><i /><i /></span> چند لحظه دیگر دوباره سر بزن</div>
        </div>
        <div className="eyebrow" style={{ padding: '6px 6px 0' }}>گزارش‌های جلسه</div>
        <div className="report-card">
          <div className="rc-icons">
            <span><Calendar /> ۱۰ آبان</span>
            <span><Clock /> ۱۶ دقیقه</span>
          </div>
          <div>
            <h4>احساس بی‌اهمیتی و فراموش شدن</h4>
            <p>در این جلسه روی واکنش تو هنگام جر‌وبحث کار کردیم و یک تمرین تنفسی ساده ساختیم.</p>
          </div>
        </div>
        <button className="btn btn-outline archive-btn">بایگانی کردن حوزه تمرکز</button>
      </div>
      <div className="fd-bottom">
        <img src={guide.img} alt={guide.name} />
        <span className="ph">چه فکری در سر داری؟</span>
        <button className="btn btn-white" onClick={() => setScreen('duration')}>شروع جلسه</button>
      </div>
    </div>
  );

  const duration = (
    <div className="screen dark dur-screen">
      <div style={{ height: 20 }} />
      <button className="icon-btn" style={{ alignSelf: 'flex-start', marginInlineStart: 8 }} onClick={() => setScreen('focus')}><ArrowBack size={22} /></button>
      <h1 className="display dur-title">مدت جلسه را انتخاب کن</h1>
      <div className="wheel">
        {[10, 15, 20, 25, 30].map((m) => (
          <div key={m} className={`wheel-item ${m === 20 ? 'sel' : ''}`}>{persianNum(m)} دقیقه</div>
        ))}
      </div>
      <p className="dur-note">می‌توانی هر زمان خواستی جلسه را زودتر تمام کنی.</p>
      <div style={{ width: '100%', padding: '0 20px 28px' }}>
        <button className="btn btn-white" onClick={() => setScreen('intro')}>شروع جلسه</button>
      </div>
    </div>
  );

  const intro = (
    <div className="screen dark">
      <div className="intro-photo">
        <img src={guide.img} alt={guide.name} />
        <button className="icon-btn x" style={{ background: 'rgba(0,0,0,.4)' }} onClick={() => setScreen('focus')}><Close size={20} /></button>
      </div>
      <div className="intro-body">
        <div className="eyebrow">راهنمای تو</div>
        <div className="display name">{guide.name}</div>
        <div className="tags">{guide.tags.map((t) => <span className="tag" key={t}>{t}</span>)}</div>
        <h3 className="display">به اولین جلسه‌ات خوش آمدی.</h3>
        <p>اینجا جای توست؛ هر چه در ذهنت هست بنویس. {guide.name} با دقت می‌خواند و جواب می‌دهد.</p>
        <span className="voice-tip">برای امتحان حالت صوت</span>
        <button className="chat-input" style={{ width: '100%', textAlign: 'right' }} onClick={() => setScreen('chat')}>
          <span className="ph">اینجا بنویس…</span>
          <span className="mic"><Mic size={17} /></span>
        </button>
      </div>
    </div>
  );

  const chat = (
    <div className="screen dark">
      <div className="chat-head">
        <img src={guide.img} alt={guide.name} />
        <div className="who">
          <div className="n">{guide.name}</div>
          <div className="t"><Clock /> ۸ دقیقه مانده</div>
        </div>
        <button className="icon-btn" onClick={() => setScreen('intro')}><Dots /></button>
      </div>
      <div className="chat-msgs">
        <div className="bubble user">
          سلام مارکوس، دیشب باز هم قبل از خواب اون بازی رو شروع کردم و تا ۳ صبح... 
          <div className="bubble-meta">دیروز، ۲۳:۴۲</div>
        </div>
        <div className="paste-toast">مسیر از کلیپ‌بورد تو کپی کرد</div>
        <div className="typing">
          <img src={guide.img} alt="" />
          <span className="tdots"><i /><i /><i /></span>
        </div>
      </div>
      <div style={{ padding: '0 14px 18px' }}>
        <div className="chat-input">
          <span className="ph">راهنمای تو در حال فکر کردن است…</span>
          <span className="mic"><Mic size={17} /></span>
        </div>
      </div>
    </div>
  );

  let body;
  if (phase === 'onb') body = onbScreen;
  else if (screen === 'home') body = home;
  else if (screen === 'guides') body = guides;
  else if (screen === 'detail') body = detail;
  else if (screen === 'expressions') body = expressions;
  else if (screen === 'focus') body = focusDetail;
  else if (screen === 'duration') body = duration;
  else if (screen === 'intro') body = intro;
  else body = chat;

  return <div className="phone">{body}</div>;
}

const LockIco = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
    <rect x="5" y="11" width="14" height="9" rx="2.5" /><path d="M8 11V8a4 4 0 0 1 8 0v3" />
  </svg>
);
