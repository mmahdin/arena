import React, { useCallback, useContext, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Check, ChevronLeft, ChevronRight, Flame, Landmark } from 'lucide-react';
import { AGE_RANGES, APPROACHES, APPROACH_EXAMPLE_USER, FLOW, QUESTIONS, REVIEWS, VOICES } from '../data';
import { resistVoiceDrag, voiceSwipeTarget } from '../features/onboarding/voiceCarousel';
import { TheiaMark } from '../components/TheiaBrand';
import { TheiaInstrument, TheiaSigil } from '../components/TheiaSigil';
import {
  BackButton,
  ChoiceButton,
  PageHeader,
  PagerDots,
  PrimaryButton,
  RatingStars,
  StatusBar,
  VoiceWave,
} from '../components/UI';

const nextFrom = (route, navigate) => {
  const nextIndex = FLOW.indexOf(route) + 1;
  if (nextIndex > 0 && nextIndex < FLOW.length) navigate(FLOW[nextIndex]);
};

const fa = (value) => Number(value).toLocaleString('fa-IR');

/**
 * The onboarding tells one quiet story: Theia, the faceless Greek Titaness of
 * sight and far-shining light, never appears as a body — only as an open ring,
 * an aperture of light and the horizon it measures. Six chapters carry the
 * metaphor from night to a completed circle; each chapter names the emotional
 * movement ("not knowing is where seeing begins") instead of decorating it.
 */
const CHAPTERS = [
  { name: 'آغاز', routes: ['welcome', 'privacy'] },
  { name: 'شب', routes: ['simple-intro', 'name', 'age', 'identity', 'therapist'] },
  { name: 'دیدن', routes: ['deeper-intro', 'mood', 'concerns', 'focus', 'feelings'] },
  { name: 'سپیده', routes: ['know-intro', 'faith', 'neurodivergent', 'relationships', 'interests', 'strengths', 'barriers', 'recharge'] },
  { name: 'سه نور', routes: ['almost-intro', 'approach', 'voice', 'checkins', 'referral'] },
  { name: 'حلقه', routes: ['ready'] },
];

function chapterIndexFor(route) {
  const index = CHAPTERS.findIndex((chapter) => chapter.routes.includes(route));
  return index < 0 ? 0 : index;
}

const OnboardingRouteContext = React.createContext('welcome');

function ChapterSeam({ index }) {
  const chapter = CHAPTERS[index];
  return (
    <div className="ob-seam" role="group" aria-label={`فصل ${chapter.name}`}>
      {CHAPTERS.map((item, position) => (
        <i
          key={item.name}
          aria-hidden="true"
          className={position < index ? 'is-past' : position === index ? 'is-now' : ''}
        />
      ))}
      <span className="sr-only" aria-live="polite">
        {`فصل ${fa(index + 1)} از ${fa(CHAPTERS.length)}`}
      </span>
    </div>
  );
}

function Shell({ theme = 'paper', back, children, className = '', footer }) {
  const route = useContext(OnboardingRouteContext);
  const dark = theme === 'ink' || theme === 'dark' || theme === 'black';
  const surface = dark ? 'ink' : 'paper';
  return (
    <section className={`app-screen onboarding-screen ob-shell ob-${surface} ${className}`.trim()} data-ob-theme={surface}>
      <StatusBar />
      <div className="ob-chrome">
        {back
          ? <BackButton onClick={back} dark={dark} />
          : <span className="ob-chrome__sigil" aria-hidden="true"><TheiaSigil size={19} /></span>}
        <ChapterSeam index={chapterIndexFor(route)} />
        <span className="ob-chrome__tail" aria-hidden="true" />
      </div>
      <div className="ob-flow">{children}</div>
      {footer ? <div className="ob-footer">{footer}</div> : null}
    </section>
  );
}

/** One heading, one quiet paragraph. The chapter word is carried by the seam. */
function ObHeader({ title, lede, epigraph, align = 'start', className = '' }) {
  return (
    <header className={`ob-header ob-header--${align} ${className}`.trim()}>
      {epigraph && <p className="ob-epigraph">{epigraph}</p>}
      <h1>{title}</h1>
      {lede && <p className="ob-lede">{lede}</p>}
    </header>
  );
}

function Instrument({ state, tone = 'ink', className = '', label }) {
  const chapter = ({ welcome: 0, veil: 0, night: 1, sight: 2, dawn: 3, orbits: 4, pulse: 5, seal: 6 })[state] ?? 0;
  const shutters = state === 'veil';
  const lantern = state === 'pulse' || chapter > 5;
  const dawn = chapter >= 3;
  const visibility = [0.12, 0.2, 0.5, 0.7, 1, 1, 1][chapter];
  const moonY = 107 - Math.min(chapter, 5) * 5;
  const moonR = chapter === 2 ? 26 : 22;
  return (
    <div className={`ob-art rooftop-art ${className}`.trim()}>
      <svg viewBox="0 0 390 330" role="img" aria-label={label || 'بام آرامی رو به سپیده؛ ماه هلالی با روشناییِ زمین'}>
        <defs>
          <linearGradient id="roof-sky" x2="0" y2="1"><stop stopColor={dawn ? '#202b40' : '#090f1c'}/><stop offset=".72" stopColor={dawn ? '#475064' : '#111b2c'}/><stop offset="1" stopColor={dawn ? '#bd9b91' : '#172438'}/></linearGradient>
          <radialGradient id="moon-ashen"><stop stopColor="#899bae" stopOpacity={visibility}/><stop offset="1" stopColor="#536478" stopOpacity={visibility*.8}/></radialGradient>
          <clipPath id="moon-clip"><circle cx="279" cy={moonY} r={moonR}/></clipPath>
        </defs>
        <rect width="390" height="330" fill="url(#roof-sky)"/>
        {Array.from({length: 25}, (_,i)=><circle key={i} cx={(i*83+23)%390} cy={(i*47+19)%175} r={i%5===0?1:0.55} fill="#d8d8d0" opacity={dawn?.22:.48}/>) }
        {dawn && <path d="M0 205 Q190 174 390 201V245H0Z" fill="#c8a6a0" opacity=".14"/>}
        <circle cx="279" cy={moonY} r={moonR} fill="url(#moon-ashen)" opacity={visibility}/>
        <g clipPath="url(#moon-clip)" opacity={visibility*.3} fill="#37495c"><ellipse cx="270" cy={moonY-4} rx="7" ry="4"/><ellipse cx="287" cy={moonY+8} rx="8" ry="5"/><ellipse cx="273" cy={moonY+12} rx="4" ry="3"/></g>
        <path d={`M279 ${moonY-moonR} A${moonR} ${moonR} 0 1 0 279 ${moonY+moonR} A${moonR*.77} ${moonR*.77} 0 1 1 279 ${moonY-moonR}`} fill="#e6e7e3"/>
        {shutters && <g fill="#322b2b"><path d="M0 0h115v330H0z"/><path d="M275 0h115v330H275z"/></g>}
        {shutters && <g stroke="#765b48" strokeWidth="2" opacity=".65"><path d="M15 20h85M15 80h85M15 140h85M15 200h85M15 260h85M290 20h85M290 80h85M290 140h85M290 200h85M290 260h85"/></g>}
        {!shutters && <g><path d="M0 248 Q180 226 390 247V330H0Z" fill="#111722"/><path d="M0 247 Q180 226 390 247" stroke="#59606b" strokeWidth="3"/><path d="M35 282l130-10 27 50H15z" fill="#4b4650" opacity=".8"/><path d="M35 282l130-10M15 322l177 0" stroke="#867b72" strokeWidth="1" opacity=".45"/>
          {chapter>=3 && <path d="M0 229 Q190 213 390 231" stroke="#c9a7a0" strokeWidth="2" opacity=".35"/>}
          {chapter>=4 && <><path d="M0 236 Q190 217 390 238" stroke="#e7bb91" strokeWidth="3" opacity=".68"/><circle cx="35" cy="225" r="2" fill="#e8c9a0"/></>}
          {lantern && <g><path d="M301 215h14l-2 20h-10z" fill="#8b6847"/><ellipse cx="308" cy="225" rx="4" ry="7" fill="#e2ae68"/><path d="M301 218h14" stroke="#bd9c71"/></g>}
          {chapter===6 && <path d="M8 228L96 248" stroke="#e7c69c" strokeWidth="8" opacity=".13"/>}
        </g>}
      </svg>
    </div>
  );
}

function Continue({ children, ...rest }) {
  return <PrimaryButton {...rest}>{children}</PrimaryButton>;
}

function Welcome({ navigate }) {
  return (
    <Shell theme="ink" className="welcome-screen" footer={(
      <Continue onClick={() => nextFrom('welcome', navigate)}>شروع</Continue>
    )}>
      <Instrument state="welcome" className="ob-art--hero" label="حلقهٔ باز تیا، با ستاره‌ای از سپیده در مرکز" />
      <ObHeader
        epigraph="روشناییِ دوربرد"
        title="تیا را نمی‌بینی؛ با تیا می‌بینی."
        lede="در اسطوره‌ها، تیا الههٔ نور و دیدن بود. این‌جا چهره‌ای ندارد؛ همراهی است که کمک می‌کند آن‌چه از پیش در توست، روشن‌تر دیده شود. تیا همراهی هوش‌مصنوعی است، نه درمانگر."
      />
    </Shell>
  );
}

function Testimonials({ navigate, back }) {
  return (
    <Shell theme="beige" back={back} className="testimonials-screen scrolling-screen">
      <PageHeader
        title="بقیه دربارهٔ تیا چی می‌گن؟"
        description="نظر کاربران تیا"
      />
      <div className="reviews-list">
        {REVIEWS.map((review) => (
          <article className="review-card" key={review.name}>
            <div className="review-meta">
              <span className="review-avatar">{review.initial}</span>
              <strong>{review.name}</strong>
              <RatingStars />
            </div>
            <p>{review.text}</p>
          </article>
        ))}
      </div>
      <PrimaryButton onClick={() => nextFrom('testimonials', navigate)}>ادامه</PrimaryButton>
    </Shell>
  );
}

function Research({ navigate, back }) {
  return (
    <Shell theme="dark" back={back} className="research-screen">
      <PageHeader
        title="تیا با کمک پژوهش‌های علمی طراحی شده."
        description="پژوهش‌های مرتبط با تیا"
      />
      <div className="research-logos" aria-label="مراکز پژوهشی">
        <span className="university-wordmark">نورث‌وسترن</span>
        <span className="university-lockup"><Flame size={34} /><b>دانشگاه نیویورک</b></span>
        <span className="university-wordmark stanford">استنفورد</span>
        <span className="university-lockup virginia"><Landmark size={34} /><b>دانشگاه ویرجینیا</b></span>
      </div>
      <div className="press-strip" aria-label="رسانه‌ها">
        <span>تایمز</span><span>فوربس</span><span>وال‌استریت ژورنال</span>
      </div>
      <PrimaryButton light onClick={() => nextFrom('research', navigate)}>ادامه</PrimaryButton>
    </Shell>
  );
}

function Experts({ navigate, back }) {
  return (
    <Shell theme="dark" back={back} className="experts-screen">
      <PageHeader title="با کمک متخصصان" />
      <div className="experts-gallery">
        <article className="expert-card">
          <img src="/assets/expert-man.jpg" alt="یکی از متخصصان تیم" />
          <div><strong>دکتر آرش نادری</strong><span>متخصص سلامت روان</span></div>
        </article>
        <article className="expert-card">
          <img src="/assets/expert-woman.jpg" alt="مدیر پژوهش بالینی" />
          <div><strong>دکتر سارا احمدی</strong><span>مدیر پژوهش بالینی</span></div>
        </article>
      </div>
      <PrimaryButton light onClick={() => nextFrom('experts', navigate)}>ادامه</PrimaryButton>
    </Shell>
  );
}

function NameScreen({ navigate, back, name, setName, profileEdit = false, savingProfile = false, onSaveName, onSaveOnboardingName }) {
  const submit = (event) => {
    event.preventDefault();
    if (name.trim()) {
      if (profileEdit) void onSaveName();
      else void onSaveOnboardingName();
    }
  };
  return (
    <Shell
      back={back}
      className="name-screen"
      footer={<Continue type="submit" form="onboarding-name" disabled={!name.trim() || savingProfile}>{savingProfile ? 'در حال ذخیره…' : profileEdit ? 'ذخیرهٔ نام' : 'ادامه'}</Continue>}
    >
      <ObHeader
        title={profileEdit ? 'اسمت را عوض کن.' : 'تو را چه صدا کنم؟'}
        lede={profileEdit ? 'هر وقت خواستی می‌توانی اسمت را تغییر بدهی.' : 'اسم اولین نوری است که به این حلقه می‌تابد.'}
      />
      <form id="onboarding-name" className="ob-name-form" onSubmit={submit}>
        <label className="sr-only" htmlFor="name">اسمت</label>
        <input
          id="name"
          className="name-input"
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="اسمت"
          autoComplete="given-name"
          maxLength={60}
        />
      </form>
    </Shell>
  );
}

const INTERSTITIALS = {
  'simple-intro': {
    state: 'night',
    chapter: 'فصل دو · شب',
    title: 'از تاریکی شروع می‌کنیم.',
    lede: 'نه چون چیزی خراب است؛ چون هر دیدن از ندیدن آغاز می‌شود. این‌جا لازم نیست از قبل بدانی چه چیزی در تو می‌گذرد.',
  },
  'deeper-intro': {
    state: 'sight',
    chapter: 'فصل سه · دیدن',
    title: 'آن‌چه تاریک به نظر می‌رسد، خالی نیست.',
    lede: 'بخش تاریک ماه هم کمی روشن است: نوری که از زمین به آن می‌رسد، به چشم تو برمی‌گردد. تیا چیزی به تو اضافه نمی‌کند و چیزی از تو نمی‌گیرد؛ فقط کمک می‌کند آن‌چه هست واضح‌تر دیده شود، حتی وقتی دیدنش آسان نیست.',
  },
  'know-intro': {
    state: 'dawn',
    chapter: 'فصل چهار · سپیده',
    title: 'سپیده یک‌باره نمی‌آید.',
    lede: 'کم‌کم روشن می‌شود، آن‌قدر آرام که اول متوجهش نمی‌شوی. گاهی نوری می‌آید و دوباره پس می‌رود؛ آن هم بخشی از راه است. حال آدم هم اغلب همین‌طور عوض می‌شود.',
  },
  'almost-intro': {
    state: 'orbits',
    chapter: 'فصل پنج · سه نور',
    title: 'سه نور؛ و هیچ‌کدام همیشگی نیست.',
    lede: 'خورشید، ماه و سپیده، هر سه فرزندانِ تیا بودند: سه حالتِ یک نور. حال‌های تو هم همین‌طورند: می‌آیند و می‌روند، و هیچ‌کدام تمامِ تو نیست.',
  },
};

function Interstitial({ route, navigate, back }) {
  const item = INTERSTITIALS[route];
  return (
    <Shell
      theme="ink"
      back={back}
      className={`interstitial-screen art-${item.state}`}
      footer={<Continue onClick={() => nextFrom(route, navigate)}>ادامه</Continue>}
    >
      <p className="ob-chapter">{item.chapter}</p>
      <Instrument state={item.state} className="ob-art--hero" />
      <ObHeader title={item.title} lede={item.lede} />
    </Shell>
  );
}

function Privacy({ navigate, back, privacySharing = true, privacyLoading = false, privacySaving = false,
  privacyLoaded = false, privacyError = false, onTogglePrivacy, onRetryPrivacy }) {
  const sharing = privacySharing;
  return (
    <Shell
      back={back}
      className="privacy-screen"
      footer={<Continue onClick={() => nextFrom('privacy', navigate)}>ادامه</Continue>}
    >
      <ObHeader
        title="چه‌قدر بگویی، انتخاب توست."
        lede="فقط تو تعیین می‌کنی چه‌قدر از خودت بگویی. برای اینکه پاسخ‌ها به تو نزدیک‌تر باشند، گفت‌وگوها و پاسخ‌های آشنایی اولیه روی سرور ذخیره می‌شوند و اطلاعات مرتبط به سرویس هوش مصنوعی فرستاده می‌شود."
      />
      <Instrument state="veil" tone="paper" className="ob-art--small" label="پرده‌ای که در میانه باز شده و شکافی از نور را نشان می‌دهد" />
      <button
        type="button"
        className="privacy-card"
        role="switch"
        aria-checked={sharing}
        aria-disabled={!privacyLoaded || privacyLoading || privacySaving}
        disabled={!privacyLoaded || privacyLoading || privacySaving}
        onClick={onTogglePrivacy}
      >
        <span className={`toggle ${sharing ? 'is-on' : ''}`} aria-hidden="true"><span /></span>
        <span>اجازهٔ استفاده از گفت‌وگوها</span>
      </button>
      {privacyError && <div className="connected-notice" role="alert">
        <p>تنظیمات حریم خصوصی در دسترس نیست. دوباره تلاش کن.</p>
        <button type="button" onClick={onRetryPrivacy}>تلاش دوباره</button>
      </div>}
    </Shell>
  );
}

// The age step is a physical wheel, not a list that re-stacks itself. One native
// scroll container holds all ranges with snap points, so touch momentum,
// trackpad and wheel input behave exactly like the platform. Every range is
// scaled, tilted and faded from its live distance to the centre instead of
// being placed with precomputed pixel offsets, so the selection glides.
const AGE_FALLBACK = '۲۵–۲۹';

const reducedMotion = () => (typeof window === 'undefined'
  ? false
  : Boolean(window.matchMedia?.('(prefers-reduced-motion: reduce)').matches));

function AgeScreen({ navigate, back, answers, setAnswer }) {
  const startValue = AGE_RANGES.includes(answers.age) ? answers.age : AGE_FALLBACK;
  const startIndex = Math.max(0, AGE_RANGES.indexOf(startValue));
  const [active, setActive] = useState(startIndex);
  const [tick, setTick] = useState(0);
  const wheelRef = useRef(null);
  const optionRefs = useRef([]);
  const startIndexRef = useRef(startIndex);
  const activeRef = useRef(startIndex);
  // -1 means "nothing chosen yet": the visual default stays a preview and no
  // answer is written until the user actually touches the wheel.
  const chosenRef = useRef(AGE_RANGES.includes(answers.age) ? startIndex : -1);
  const frameRef = useRef(0);
  const settleRef = useRef(0);
  const interactedRef = useRef(false);
  const dragRef = useRef(null);
  const draggedRef = useRef(false);

  // A short tick is the physical feedback of passing a detent: the marker
  // pulse and, where the platform allows it, a brief haptic tap.
  const tickFeedback = useCallback(() => {
    setTick((value) => value + 1);
    if (typeof navigator !== 'undefined' && typeof navigator.vibrate === 'function') {
      try { navigator.vibrate(8); } catch { /* haptics are best effort */ }
    }
  }, []);

  const centeredOffset = useCallback((node) => {
    const wheel = wheelRef.current;
    if (!wheel || !node) return 0;
    return node.offsetTop + node.offsetHeight / 2 - wheel.clientHeight / 2;
  }, []);

  const scrollToIndex = useCallback((index) => {
    const wheel = wheelRef.current;
    const node = optionRefs.current[index];
    if (!wheel || !node) return;
    wheel.scrollTo({
      top: centeredOffset(node),
      behavior: reducedMotion() ? 'auto' : 'smooth',
    });
  }, [centeredOffset]);

  // Paint each range from its scroll distance. Only custom properties change,
  // so the browser never re-lays-out the list mid-gesture.
  const paint = useCallback(() => {
    frameRef.current = 0;
    const wheel = wheelRef.current;
    if (!wheel) return;
    const radius = Math.max(1, wheel.clientHeight / 2);
    const center = wheel.scrollTop + radius;
    let nearest = activeRef.current;
    let nearestDistance = Infinity;
    optionRefs.current.forEach((node, index) => {
      if (!node) return;
      const distance = node.offsetTop + node.offsetHeight / 2 - center;
      const progress = Math.min(1, Math.abs(distance) / radius);
      node.style.setProperty('--age-scale', (1 - 0.42 * progress ** 1.12).toFixed(3));
      node.style.setProperty('--age-fade', (1 - 0.5 * progress).toFixed(3));
      const tilt = Math.sign(distance) * progress * 12;
      node.style.setProperty('--age-tilt', `${(-tilt).toFixed(2)}deg`);
      if (Math.abs(distance) < nearestDistance) {
        nearestDistance = Math.abs(distance);
        nearest = index;
      }
    });
    if (nearest !== activeRef.current) {
      activeRef.current = nearest;
      setActive(nearest);
      if (interactedRef.current) tickFeedback();
    }
  }, [tickFeedback]);

  // Answer is committed once the wheel settles, not on every pixel of movement.
  const commit = useCallback(() => {
    const index = activeRef.current;
    if (!interactedRef.current || chosenRef.current === index) return;
    chosenRef.current = index;
    setAnswer('age', AGE_RANGES[index]);
  }, [setAnswer]);

  const handleScroll = () => {
    if (!frameRef.current) frameRef.current = window.requestAnimationFrame(paint);
    window.clearTimeout(settleRef.current);
    settleRef.current = window.setTimeout(commit, 130);
  };

  const choose = (index) => {
    const next = Math.max(0, Math.min(AGE_RANGES.length - 1, index));
    interactedRef.current = true;
    activeRef.current = next;
    chosenRef.current = next;
    setActive(next);
    setAnswer('age', AGE_RANGES[next]);
    tickFeedback();
    scrollToIndex(next);
    optionRefs.current[next]?.focus({ preventScroll: true });
  };

  const handleKeyDown = (event) => {
    const last = AGE_RANGES.length - 1;
    const step = { ArrowDown: 1, ArrowUp: -1, PageDown: 2, PageUp: -2 }[event.key];
    const target = step === undefined ? { Home: 0, End: last }[event.key] : activeRef.current + step;
    if (target === undefined) return;
    event.preventDefault();
    choose(target);
  };

  // A scroll container does not drag with a mouse, so the same gesture is added
  // for pointers. Touch keeps the platform's own momentum scrolling.
  const handlePointerDown = (event) => {
    interactedRef.current = true;
    // A press always clears the "just dragged" guard, so the tap that follows
    // is a real selection again on every input type.
    draggedRef.current = false;
    const wheel = wheelRef.current;
    if (!wheel || dragRef.current || event.pointerType !== 'mouse' || event.button !== 0) return;
    const startTop = wheel.scrollTop;
    const startY = event.clientY;
    const snap = wheel.style.scrollSnapType;
    const behavior = wheel.style.scrollBehavior;
    // Snap and smooth scrolling are paused mid-drag so the wheel tracks the
    // pointer one-to-one, then everything settles on release.
    wheel.style.scrollSnapType = 'none';
    wheel.style.scrollBehavior = 'auto';
    const move = (moveEvent) => {
      const delta = moveEvent.clientY - startY;
      if (Math.abs(delta) > 4) draggedRef.current = true;
      wheel.scrollTo({ top: startTop - delta });
    };
    const finish = () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', finish);
      window.removeEventListener('pointercancel', finish);
      dragRef.current = null;
      wheel.style.scrollSnapType = snap;
      wheel.style.scrollBehavior = behavior;
      if (!draggedRef.current) return;
      paint();
      scrollToIndex(activeRef.current);
    };
    dragRef.current = { finish };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', finish);
    window.addEventListener('pointercancel', finish);
  };

  useLayoutEffect(() => {
    const wheel = wheelRef.current;
    if (!wheel) return undefined;
    // Half a viewport above the first range and below the last one, so both
    // ends can reach the marker. Measured instead of a percentage height, which
    // scroll containers do not resolve consistently.
    const sizeAndCenter = (index) => {
      const node = optionRefs.current[index];
      if (!node) return;
      wheel.style.setProperty('--age-pad', `${Math.max(0, (wheel.clientHeight - node.offsetHeight) / 2)}px`);
      wheel.scrollTop = centeredOffset(node);
    };
    sizeAndCenter(startIndexRef.current);
    paint();
    // Keep the chosen range under the marker when the viewport, keyboard or
    // orientation changes the wheel height.
    const observer = typeof ResizeObserver === 'function'
      ? new ResizeObserver(() => {
        sizeAndCenter(activeRef.current);
        paint();
      })
      : null;
    observer?.observe(wheel);
    return () => {
      observer?.disconnect();
      if (frameRef.current) window.cancelAnimationFrame(frameRef.current);
      window.clearTimeout(settleRef.current);
      dragRef.current?.finish();
    };
  }, [centeredOffset, paint]);

  // Answers can change from outside this screen (back/forward through the flow).
  useEffect(() => {
    const index = AGE_RANGES.indexOf(answers.age);
    if (index < 0 || index === activeRef.current) return;
    activeRef.current = index;
    chosenRef.current = index;
    setActive(index);
    scrollToIndex(index);
  }, [answers.age, scrollToIndex]);

  return (
    <Shell
      back={back}
      className="age-screen"
      footer={<Continue onClick={() => { setAnswer('age', AGE_RANGES[activeRef.current]); nextFrom('age', navigate); }}>ادامه</Continue>}
    >
      <ObHeader title="چند سالت است؟" lede="بکش یا اسکرول کن تا بازهٔ سنی‌ات را انتخاب کنی." />
      <div className="age-stage">
        <span key={`age-marker-${tick}`} className="age-marker" aria-hidden="true" />
        <div
          className="age-wheel"
          ref={wheelRef}
          role="radiogroup"
          aria-label="بازهٔ سنی"
          aria-orientation="vertical"
          onScroll={handleScroll}
          onPointerDown={handlePointerDown}
          onWheel={() => { interactedRef.current = true; }}
          onKeyDown={handleKeyDown}
        >
          <span className="age-wheel__pad" aria-hidden="true" />
          {AGE_RANGES.map((range, index) => (
            <button
              type="button"
              key={range}
              ref={(node) => { optionRefs.current[index] = node; }}
              role="radio"
              aria-checked={index === active}
              tabIndex={index === active ? 0 : -1}
              className={`age-option${index === active ? ' is-active' : ''}`}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => { if (!draggedRef.current) choose(index); }}
            >{range}</button>
          ))}
          <span className="age-wheel__pad" aria-hidden="true" />
        </div>
      </div>
      <p className="sr-only" aria-live="polite">{`بازهٔ انتخاب‌شده: ${AGE_RANGES[active]}`}</p>
    </Shell>
  );
}

function QuestionScreen({ route, navigate, back, answers, setAnswer, notify }) {
  const advanceTimer = useRef(null);
  useEffect(() => () => window.clearTimeout(advanceTimer.current), []);
  const config = QUESTIONS[route];
  const value = answers[route];
  const selected = config.type === 'multi' ? (value || []) : value;
  const hasSelection = config.type === 'multi' ? selected.length > 0 : Boolean(selected);

  const choose = (option) => {
    if (config.type === 'single') {
      setAnswer(route, option);
      window.clearTimeout(advanceTimer.current);
      advanceTimer.current = window.setTimeout(() => nextFrom(route, navigate), 160);
      return;
    }
    const exists = selected.includes(option);
    if (!exists && config.limit && selected.length >= config.limit) {
      notify(`می‌تونی حداکثر ${Number(config.limit).toLocaleString('fa-IR')} مورد را انتخاب کنی.`);
      return;
    }
    setAnswer(route, exists ? selected.filter((item) => item !== option) : [...selected, option]);
  };

  return (
    <Shell
      back={back}
      className={`question-screen question-${route} ${config.skip ? 'question-screen--skip' : ''}`}
      footer={(
        <>
          {config.type === 'multi' && (
            <Continue
              disabled={!hasSelection && !config.allowEmpty}
              onClick={() => nextFrom(route, navigate)}
            >{config.confirmLabel || 'ادامه'}</Continue>
          )}
          {config.skip && (
            <button type="button" className="skip-button" onClick={() => nextFrom(route, navigate)}>فعلاً رد کن</button>
          )}
        </>
      )}
    >
      <ObHeader title={config.title} lede={config.description} />
      <div className="choice-scroll">
        <div className={`choice-list ${config.type === 'single' ? 'choice-list--single' : ''}`}>
          {config.options.map((option) => (
            <ChoiceButton
              key={option}
              multi={config.type === 'multi'}
              selected={config.type === 'multi' ? selected.includes(option) : selected === option}
              onClick={() => choose(option)}
            >{option}</ChoiceButton>
          ))}
        </div>
      </div>
    </Shell>
  );
}

// Deliberately drawn as native SVG, not screenshots, so it stays sharp and
// changes smoothly at all five mood positions. Direction follows the reference:
// left is struggling and right is generally good, regardless of page RTL.
const MOODS = [
  { label: 'این روزها سخت می‌گذرد', note: 'لازم نیست توضیح بدهی. از همین‌جا شروع می‌کنیم.' },
  { label: 'انرژی زیادی ندارم', note: 'با چیزهای کوچک شروع می‌کنیم؛ لازم نیست بزرگ باشد.' },
  { label: 'یک روز خوب، یک روز سخت', note: 'روزها یکسان نیستند. کنار هم پیش می‌رویم.' },
  { label: 'دارم می‌گذرونم', note: 'داری راه می‌افتی. همین کافی است.' },
  { label: 'در کل حالم خوب است', note: 'خوب است. بگذار بیشتر بشناسیمش.' },
];

function MoodScribble({ level }) {
  // A continuous hand-drawn contour evolves from angular and tangled to round.
  const path = React.useMemo(() => {
    const points = [];
    const turns = [9.5, 7.4, 5.0, 3.0, 2.2][level];
    const count = 430;
    for (let i = 0; i <= count; i++) {
      const progress = i / count;
      const theta = progress * turns * Math.PI * 2;
      const rough = level <= 1 ? (1 + 0.24 * Math.sin(theta * 2.7) + 0.12 * Math.cos(theta * 4.3)) : 1;
      const radius = (0.45 + progress * 0.50) * rough;
      const tilt = level === 4 ? 0.15 : 0.38;
      const x = 170 + 113 * radius * Math.cos(theta + tilt) + (level === 2 ? 13 * Math.sin(theta * 0.4) : 0);
      const y = 150 + 105 * radius * Math.sin(theta) + (level === 0 ? 12 * Math.sin(theta * 1.5) : 0);
      points.push(`${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`);
    }
    return points.join(' ');
  }, [level]);
  // The line itself is the illustration: a tangled thread at the hard end of
  // the scale, a nearly closed ring at the easy end, and a dawn-coloured light
  // that grows as the contour untangles.
  const tone = ['#8C94B8', '#B0A0A0', '#D2AE86', '#E3C48C', '#F2C98D'][level];
  return (
    <svg className="mood-scribble" viewBox="0 0 340 300" aria-hidden="true" focusable="false">
      <defs>
        <linearGradient id="mood-line" x1="40" y1="270" x2="300" y2="40" gradientUnits="userSpaceOnUse">
          <stop stopColor={tone} stopOpacity="0.55" />
          <stop offset="0.55" stopColor={tone} />
          <stop offset="1" stopColor="#F7E7C6" />
        </linearGradient>
        <radialGradient id="mood-halo" cx="50%" cy="50%" r="50%">
          <stop stopColor="#F3D8A3" stopOpacity={0.1 + level * 0.11} />
          <stop offset="1" stopColor="#F3D8A3" stopOpacity="0" />
        </radialGradient>
      </defs>
      <circle cx="170" cy="150" r="128" fill="url(#mood-halo)" />
      <path d={path} fill="none" stroke="url(#mood-line)" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" />
      {level >= 3 && (
        <path
          d="M170 137L172.6 144.8L180.4 147.4L172.6 150L170 157.8L167.4 150L159.6 147.4L167.4 144.8L170 137Z"
          fill={tone}
          opacity={level === 4 ? '0.95' : '0.5'}
        />
      )}
    </svg>
  );
}

function MoodScreen({ navigate, back, answers, setAnswer }) {
  const mood = Number.isInteger(answers.mood) && answers.mood >= 0 && answers.mood <= 4 ? answers.mood : 2;
  return (
    <Shell
      back={back}
      className="mood-screen"
      footer={<Continue onClick={() => { setAnswer('mood', mood); nextFrom('mood', navigate); }}>ادامه</Continue>}
    >
      <ObHeader title="این روزها چه حالی داری؟" lede="نوار را بکش تا نزدیک‌ترین حالت را پیدا کنی؛ لازم نیست دقیق باشد." />
      <div className="mood-stage">
        <div className="mood-glow" data-mood-level={mood} aria-hidden="true" />
        <MoodScribble level={mood} />
        <h2 aria-live="polite">{MOODS[mood].label}</h2>
        <div className="mood-slider-wrap" dir="ltr">
          <div className="mood-ticks" aria-hidden="true">
            {MOODS.map((item, index) => <i key={index} style={{ '--tick-index': index }} className={index === mood ? 'is-active' : ''} />)}
          </div>
          <input className="mood-slider" type="range" min="0" max="4" step="1"
            value={mood} onChange={event => setAnswer('mood', Number(event.target.value))}
            aria-label="حس این روزهای من" aria-valuetext={MOODS[mood].label} />
        </div>
      </div>
      <p className="mood-footnote" aria-live="polite">{MOODS[mood].note}</p>
    </Shell>
  );
}

function CheckinsScreen({ navigate, back }) {
  return (
    <Shell
      back={back}
      className="checkins-screen"
      footer={<Continue onClick={() => nextFrom('checkins', navigate)}>ادامه</Continue>}
    >
      <ObHeader
        title="اگر بخواهی، چراغ روشن می‌ماند."
        lede="تیا می‌تواند گاهی سراغ حالت را بگیرد، بی‌اصرار. هر وقت بخواهی می‌توانی یادآوری‌ها را خاموش کنی."
      />
      <Instrument state="pulse" tone="paper" className="ob-art--small" label="حلقه‌ای که نورش بازمی‌گردد" />
    </Shell>
  );
}

function VoiceScreen({ navigate, back, answers, setAnswer }) {
  const initial = Math.max(0, VOICES.findIndex((item) => item.name === answers.voice));
  const [active, setActive] = useState(initial);
  const [drag, setDrag] = useState(0);
  const [dragging, setDragging] = useState(false);
  const [cardMetrics, setCardMetrics] = useState({ width: 210, step: 224 });
  const trackRef = useRef(null);
  const cardRefs = useRef([]);
  const activeRef = useRef(initial);
  const gestureRef = useRef(null);
  const suppressClickRef = useRef(false);
  const lastWheelRef = useRef(-Infinity);
  const voice = VOICES[active];

  // Offset measurements use layout sizes, not transformed bounding boxes.
  // Recalculate at responsive breakpoints so the centered card never drifts.
  useLayoutEffect(() => {
    const measure = () => {
      const first = cardRefs.current[0];
      const second = cardRefs.current[1];
      if (!first) return;
      const width = first.offsetWidth;
      const step = second ? second.offsetLeft - first.offsetLeft : width + 14;
      setCardMetrics((previous) => previous.width === width && previous.step === step
        ? previous : { width, step });
    };
    measure();
    const observer = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(measure);
    if (observer && trackRef.current) observer.observe(trackRef.current);
    window.addEventListener('resize', measure);
    return () => {
      observer?.disconnect();
      window.removeEventListener('resize', measure);
    };
  }, []);

  const select = (index) => {
    const next = Math.max(0, Math.min(VOICES.length - 1, index));
    if (next === activeRef.current) {
      // Tapping the already-centered default voice must still save that choice.
      if (answers.voice !== VOICES[next].name) setAnswer('voice', VOICES[next].name);
      return;
    }
    activeRef.current = next;
    setActive(next);
    setAnswer('voice', VOICES[next].name);
  };

  const onPointerDown = (event) => {
    if (!event.isPrimary || (event.pointerType === 'mouse' && event.button !== 0)) return;
    suppressClickRef.current = false;
    // If a second gesture interrupts an in-flight snap, begin from the
    // visible position rather than jumping to the pending destination.
    const transform = trackRef.current ? window.getComputedStyle(trackRef.current).transform : 'none';
    const visualX = transform === 'none' || typeof DOMMatrixReadOnly === 'undefined'
      ? null : new DOMMatrixReadOnly(transform).m41;
    const targetX = -activeRef.current * cardMetrics.step - cardMetrics.width / 2;
    gestureRef.current = {
      id: event.pointerId, x: event.clientX, y: event.clientY,
      residual: visualX === null ? 0 : visualX - targetX,
      startTime: event.timeStamp, axis: null,
      samples: [{ x: event.clientX, time: event.timeStamp }],
    };
  };

  const onPointerMove = (event) => {
    const gesture = gestureRef.current;
    if (!gesture || gesture.id !== event.pointerId) return;
    const dx = event.clientX - gesture.x;
    const dy = event.clientY - gesture.y;
    if (!gesture.axis && Math.hypot(dx, dy) > 7) {
      gesture.axis = Math.abs(dx) > Math.abs(dy) ? 'x' : 'y';
      if (gesture.axis === 'x') {
        event.currentTarget.setPointerCapture(event.pointerId);
        setDragging(true);
      }
    }
    if (gesture.axis !== 'x') return;
    // The browser handles vertical page gestures; horizontal movement is ours.
    // At either end, add gentle resistance instead of revealing empty space.
    setDrag(gesture.residual + resistVoiceDrag(dx, activeRef.current, VOICES.length));
    gesture.samples.push({ x: event.clientX, time: event.timeStamp });
    gesture.samples = gesture.samples.filter((sample) => event.timeStamp - sample.time <= 100);
  };

  const finishGesture = (event, cancelled = false) => {
    const gesture = gestureRef.current;
    if (!gesture || gesture.id !== event.pointerId) return;
    gestureRef.current = null;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    setDragging(false);
    setDrag(0);
    if (gesture.axis !== 'x') return;
    // Captured pointer events must not trigger a stray card click on release.
    suppressClickRef.current = true;
    if (cancelled) return;

    const dx = event.clientX - gesture.x;
    const samples = gesture.samples;
    const oldest = samples[0];
    const velocity = oldest && event.timeStamp > oldest.time
      ? (event.clientX - oldest.x) / (event.timeStamp - oldest.time)
      : dx / Math.max(1, event.timeStamp - gesture.startTime);
    select(voiceSwipeTarget({
      index: activeRef.current, count: VOICES.length, dx, velocity, step: cardMetrics.step,
    }));
  };

  const onKeyDown = (event) => {
    const target = { ArrowRight: activeRef.current + 1, ArrowLeft: activeRef.current - 1, Home: 0, End: VOICES.length - 1 }[event.key];
    if (target === undefined) return;
    event.preventDefault();
    select(target);
  };

  const onWheel = (event) => {
    if (Math.abs(event.deltaX) < 8 || Math.abs(event.deltaX) <= Math.abs(event.deltaY)) return;
    event.preventDefault();
    if (event.timeStamp - lastWheelRef.current < 340) return;
    lastWheelRef.current = event.timeStamp;
    select(activeRef.current + (event.deltaX > 0 ? 1 : -1));
  };

  const focusIndex = active - drag / Math.max(1, cardMetrics.step);
  const offset = drag - active * cardMetrics.step - cardMetrics.width / 2;

  return (
    <Shell
      theme="ink"
      back={back}
      className="voice-screen"
      footer={(
        <>
          <PagerDots count={VOICES.length} active={active} onChange={select} dark label="انتخاب صدا" labels={VOICES.map((item) => item.name)} />
          <Continue onClick={() => nextFrom('voice', navigate)}>ادامه با «{voice.name}»</Continue>
        </>
      )}
    >
      <ObHeader
        title="با چه صدایی حرف بزنم؟"
        lede="صدا فقط انتخاب لحن است؛ هنوز پخش صوتی نداریم و هر وقت خواستی می‌توانی عوضش کنی."
      />
      <div
        className={`voice-carousel ${dragging ? 'is-dragging' : ''}`}
        dir="ltr"
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={finishGesture}
        onPointerCancel={(event) => finishGesture(event, true)}
        onLostPointerCapture={(event) => finishGesture(event, true)}
        onClickCapture={(event) => {
          if (!suppressClickRef.current) return;
          event.preventDefault();
          event.stopPropagation();
          suppressClickRef.current = false;
        }}
        onWheel={onWheel}
        onKeyDown={onKeyDown}
        tabIndex={0}
        role="group"
        aria-label="گزینه‌های صدا"
      >
        <div className="voice-track" ref={trackRef} style={{ transform: `translate3d(${offset}px, 0, 0)` }}>
          {VOICES.map((item, index) => {
            const proximity = Math.max(0, 1 - Math.abs(index - focusIndex));
            return (
              <button
                type="button"
                key={item.name}
                ref={(node) => { cardRefs.current[index] = node; }}
                className={`voice-card ${index === active ? 'is-active' : ''}`}
                tabIndex={index === active ? 0 : -1}
                style={dragging ? {
                  opacity: 0.55 + 0.45 * proximity,
                  transform: `scale(${0.92 + 0.08 * proximity})`,
                } : undefined}
                onClick={() => select(index)}
                aria-pressed={index === active}
                aria-label={`صدای ${item.name}، ${item.description}`}
              >
                <VoiceWave path={item.path} />
                <span className="voice-card-copy" dir="rtl">
                  <strong>{item.name}</strong>
                  <small>{item.description}</small>
                </span>
                {index === active && <span className="voice-card-check" aria-hidden="true"><Check size={16} /></span>}
              </button>
            );
          })}
        </div>
      </div>
    </Shell>
  );
}

function ApproachScreen({ navigate, back, answers, setAnswer, profileEdit = false, savingProfile = false, onSaveApproach }) {
  const initial = answers.approach ? Math.max(0, APPROACHES.findIndex((item) => item.name === answers.approach)) : 2;
  const [active, setActive] = useState(initial);
  const [drag, setDrag] = useState(0);
  const [dragging, setDragging] = useState(false);
  const gesture = useRef(null);
  const selectedIndex = useRef(initial);
  const approach = APPROACHES[active];

  const select = useCallback((index) => {
    const next = Math.max(0, Math.min(APPROACHES.length - 1, index));
    if (next === selectedIndex.current) return;
    selectedIndex.current = next;
    setActive(next);
    setAnswer('approach', APPROACHES[next].name);
  }, [setAnswer]);

  const onKeyDown = (event) => {
    const target = { ArrowLeft: selectedIndex.current + 1, ArrowRight: selectedIndex.current - 1, Home: 0, End: APPROACHES.length - 1 }[event.key];
    if (target === undefined) return;
    event.preventDefault();
    select(target);
  };

  const finishGesture = (event, cancelled = false) => {
    const start = gesture.current;
    if (!start || start.id !== event.pointerId) return;
    gesture.current = null;
    setDragging(false);
    setDrag(0);
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
    if (cancelled || start.axis !== 'x') return;
    const dx = event.clientX - start.x;
    const velocity = Math.abs(dx) / Math.max(1, event.timeStamp - start.time);
    const distance = Math.max(1, event.currentTarget.querySelector('.approach-replies')?.clientWidth || 320);
    // Only commit on release. During the gesture both the active reply and the
    // incoming neighboring reply follow the finger at the same pixel offset.
    if (Math.abs(dx) > distance * 0.24 || (Math.abs(dx) > 24 && velocity > 0.5)) {
      select(selectedIndex.current + (dx < 0 ? 1 : -1));
    }
  };

  return (
    <Shell theme="ink" back={back} className="approach-screen">
      <ObHeader title="چطور با تو حرف بزنم؟" lede="لحن تیا را انتخاب کن؛ این‌ها فقط نمونه‌اند و چیزی فرستاده نمی‌شود." />
      <div
        className="approach-stage"
        role="region"
        aria-label="نمونهٔ گفت‌وگو"
        aria-describedby="approach-style approach-keyboard-hint"
        tabIndex={0}
        onKeyDown={onKeyDown}
        onPointerDown={(event) => {
          if (!event.isPrimary || event.button !== 0 || event.target.closest('button, a, input, textarea, select, label')) return;
          gesture.current = { id: event.pointerId, x: event.clientX, y: event.clientY, time: event.timeStamp, axis: null };
          event.currentTarget.setPointerCapture(event.pointerId);
        }}
        onPointerMove={(event) => {
          const start = gesture.current;
          if (!start || start.id !== event.pointerId) return;
          const dx = event.clientX - start.x;
          const dy = event.clientY - start.y;
          if (!start.axis) {
            if (Math.abs(dx) < 8 && Math.abs(dy) < 8) return;
            start.axis = Math.abs(dx) > Math.abs(dy) * 1.15 ? 'x' : 'y';
          }
          if (start.axis !== 'x') return;
          const atEdge = (dx < 0 && selectedIndex.current === APPROACHES.length - 1) || (dx > 0 && selectedIndex.current === 0);
          const distance = Math.max(1, event.currentTarget.querySelector('.approach-replies')?.clientWidth || 320);
          setDragging(true);
          // Full 1:1 tracking reveals the next reply progressively. Apply
          // resistance only at the first/last style where no neighbor exists.
          const offset = atEdge ? Math.max(-42, Math.min(42, dx * 0.16)) : Math.max(-distance, Math.min(distance, dx));
          setDrag(offset);
        }}
        onPointerUp={finishGesture}
        onPointerCancel={(event) => finishGesture(event, true)}
        onLostPointerCapture={(event) => finishGesture(event, true)}
      >
        <div className="approach-picker">
          <button type="button" className="approach-direction" aria-label="سبک قبلی" disabled={active === 0 || savingProfile} onClick={() => select(active - 1)}>
            <ChevronRight size={18} strokeWidth={1.5} aria-hidden="true" />
          </button>
          <h2 id="approach-style">{approach.label || approach.name}</h2>
          <button type="button" className="approach-direction" aria-label="سبک بعدی" disabled={active === APPROACHES.length - 1 || savingProfile} onClick={() => select(active + 1)}>
            <ChevronLeft size={18} strokeWidth={1.5} aria-hidden="true" />
          </button>
          <span id="approach-keyboard-hint" className="sr-only">با کلیدهای جهت چپ و راست هم می‌تونی سبک رو تغییر بدی.</span>
        </div>
        <div className="approach-message approach-message--user">
          <p>{APPROACH_EXAMPLE_USER}</p>
        </div>
        <div className={`approach-replies${dragging ? ' is-dragging' : ''}`}>
          {APPROACHES.map((item, index) => (
            <article
              key={item.name}
              className={`approach-reply${index === active ? ' is-active' : ''}${Math.abs(index - active) <= 1 ? ' is-near' : ''}`}
              style={{ '--slide-position': `${(index - active) * 100}%`, '--reply-drag': `${drag}px` }}
              aria-hidden={index !== active}
              inert={index !== active}
            >
              <span className="approach-mark" aria-hidden="true"><TheiaMark size={18} /></span>
              <p className="approach-response">{item.exampleResponse || item.response}</p>
            </article>
          ))}
        </div>
        <div className="approach-position" aria-hidden="true">
          {APPROACHES.map((item, index) => <span key={item.name} className={index === active ? 'is-active' : ''} />)}
        </div>
      </div>
      <p className="sr-only" role="status" aria-live="polite" aria-atomic="true">
        {approach.label || approach.name}. گزینهٔ {(active + 1).toLocaleString('fa-IR')} از {APPROACHES.length.toLocaleString('fa-IR')}. {approach.exampleResponse || approach.response}
      </p>
      <div className="approach-actions">
        <PrimaryButton
          ariaLabel={profileEdit ? 'ذخیرهٔ سبک گفت‌وگو' : `ادامه با ${approach.label || approach.name}`}
          disabled={savingProfile}
          onClick={() => {
            if (profileEdit) onSaveApproach(approach.name);
            else { setAnswer('approach', approach.name); nextFrom('approach', navigate); }
          }}
        >{savingProfile ? 'در حال ذخیره…' : profileEdit ? 'ذخیره' : 'ادامه'}</PrimaryButton>
        <button type="button" className="skip-button" onClick={() => profileEdit ? back() : nextFrom('approach', navigate)}>
          {profileEdit ? 'انصراف' : 'رد کردن'}
        </button>
      </div>
    </Shell>
  );
}

function Ready({ navigate, savingProfile = false }) {
  return (
    <Shell
      theme="ink"
      className="ready-screen"
      footer={(
        <Continue disabled={savingProfile} onClick={() => nextFrom('ready', navigate)}>
          {savingProfile ? 'در حال ذخیره…' : 'شروع گفت‌وگو'}
        </Continue>
      )}
    >
      <p className="ob-chapter">فصل شش · حلقه</p>
      <Instrument state="seal" className="ob-art--hero" label="حلقهٔ کامل تیا؛ دایره بسته می‌شود و ستاره در مرکز می‌درخشد" />
      <ObHeader
        title="حلقه کامل می‌شود."
        lede="از این‌جا با هم شروع می‌کنیم. پاسخ‌های آشنایی اولیه‌ات ذخیره می‌شوند و برای شخصی‌سازی گفت‌وگو در اختیار سرویس هوش مصنوعی قرار می‌گیرند."
      />
    </Shell>
  );
}

export default function OnboardingScreen(props) {
  const { route } = props;
  return (
    <OnboardingRouteContext.Provider value={route}>
      <OnboardingRoute {...props} />
    </OnboardingRouteContext.Provider>
  );
}

function OnboardingRoute(props) {
  const { route } = props;
  if (route === 'welcome') return <Welcome {...props} />;
  if (route === 'testimonials') return <Testimonials {...props} />;
  if (route === 'research') return <Research {...props} />;
  if (route === 'experts') return <Experts {...props} />;
  if (route === 'name') return <NameScreen {...props} />;
  if (INTERSTITIALS[route]) return <Interstitial {...props} />;
  if (route === 'privacy') return <Privacy {...props} />;
  if (route === 'age') return <AgeScreen {...props} />;
  if (route === 'mood') return <MoodScreen {...props} />;
  if (route === 'checkins') return <CheckinsScreen {...props} />;
  if (QUESTIONS[route]) return <QuestionScreen {...props} />;
  if (route === 'voice') return <VoiceScreen {...props} />;
  if (route === 'approach') return <ApproachScreen {...props} />;
  if (route === 'ready') return <Ready {...props} />;
  return null;
}
