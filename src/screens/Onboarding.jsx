import React, { useCallback, useContext, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Check, ChevronLeft, ChevronRight, Flame, Landmark } from 'lucide-react';
import { AGE_RANGES, APPROACHES, APPROACH_EXAMPLE_USER, FLOW, QUESTIONS, REVIEWS, VOICES } from '../data';
import { resistVoiceDrag, voiceSwipeTarget } from '../features/onboarding/voiceCarousel';
import { EarthshineScene, HeldMoonFigure, HeldMoonGlyph, MoonProgress } from '../components/EarthshineScene';
import { TheiaSigil } from '../components/TheiaSigil';
import { TheiaMark } from '../components/TheiaBrand';
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
 * Earthshine — The Last Hours of the Night.
 *
 * The whole onboarding is one pre-dawn, seen from one fixed place under one
 * moon: a flat roof facing east, a low parapet, a kilim, a cypress beyond the
 * wall. A thin waning crescent rises only in the last hours before sunrise, so
 * the entire story fits inside a single night's ending.
 *
 * Every scene is the same camera with different light (`EarthshineScene`), and
 * the story's progress bar is the legibility of the ashen disc — the Earth's
 * light coming back off the unlit part of the moon. Theia is never drawn: she is
 * the seeing, the patience of the eye adjusting, the dark that lets faint things
 * show. Her signature is the earthshine itself.
 */
const CHAPTERS = [
  { name: 'آغاز', routes: ['welcome', 'privacy'] },
  { name: 'شب', routes: ['simple-intro', 'name', 'age', 'identity', 'therapist'] },
  { name: 'دیدن', routes: ['deeper-intro', 'mood', 'concerns', 'focus', 'feelings'] },
  { name: 'سپیده', routes: ['know-intro', 'faith', 'neurodivergent', 'relationships', 'interests', 'strengths', 'barriers', 'recharge'] },
  { name: 'سه نور', routes: ['almost-intro', 'approach', 'voice', 'checkins', 'referral'] },
  { name: 'حلقه', routes: ['ready'] },
];

/** Which of the eight stills lights each route. */
const SCENE_FOR_ROUTE = {
  welcome: 'welcome',
  privacy: 'privacy',
  'simple-intro': 'night',
  name: 'night',
  age: 'night',
  identity: 'night',
  therapist: 'night',
  'deeper-intro': 'seeing',
  // the mood screen holds the moon itself at large size, so its backdrop is the
  // roof with the moon taken out of the sky: two moons would fight
  mood: 'mood',
  concerns: 'seeing',
  focus: 'seeing',
  feelings: 'seeing',
  'know-intro': 'dawn',
  faith: 'dawn',
  neurodivergent: 'dawn',
  relationships: 'dawn',
  interests: 'dawn',
  strengths: 'dawn',
  barriers: 'dawn',
  recharge: 'dawn',
  'almost-intro': 'three',
  approach: 'three',
  voice: 'three',
  checkins: 'checkins',
  referral: 'checkins',
  ready: 'final',
};

function chapterIndexFor(route) {
  const index = CHAPTERS.findIndex((chapter) => chapter.routes.includes(route));
  return index < 0 ? 0 : index;
}

function progressFor(route) {
  const index = FLOW.indexOf(route);
  if (index < 0) return 0;
  return Math.max(0.02, Math.min(1, (index + 1) / (FLOW.length - 1)));
}

const OnboardingRouteContext = React.createContext('welcome');

/**
 * The story's progress is the legibility of the ashen disc, not a percentage:
 * a small moon in the chrome whose unlit half brightens from nearly invisible
 * to fully readable, with the travelled arc drawn as a single thread of dawn.
 */
function StoryProgress({ route }) {
  const chapter = chapterIndexFor(route);
  const item = CHAPTERS[chapter];
  return (
    <div className="ob-progress" role="group" aria-label={`فصل ${item.name}`}>
      <MoonProgress route={route} progress={progressFor(route)} />
      <span className="ob-progress__chapter">{item.name}</span>
      <span className="sr-only" aria-live="polite">
        {`فصل ${fa(chapter + 1)} از ${fa(CHAPTERS.length)}`}
      </span>
    </div>
  );
}

function Shell({ theme = 'night', back, children, className = '', footer, scene, frame = 'band' }) {
  const route = useContext(OnboardingRouteContext);
  const night = theme !== 'paper';
  const still = scene ?? SCENE_FOR_ROUTE[route] ?? 'welcome';
  return (
    <section
      className={`app-screen onboarding-screen ob-shell ob-${night ? 'night' : 'paper'} ob-frame-${frame} ${className}`.trim()}
      data-ob-theme={night ? 'night' : 'paper'}
      data-ob-scene={still}
    >
      <div className="ob-world" aria-hidden="true">
        <EarthshineScene scene={still} />
      </div>
      <div className="ob-wash" aria-hidden="true" />
      <StatusBar />
      <div className="ob-chrome">
        {back
          ? <BackButton onClick={back} dark={night} />
          : <span className="ob-chrome__moon" aria-hidden="true"><HeldMoonGlyph size={18} /></span>}
        <StoryProgress route={route} />
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

function Continue({ children, ...rest }) {
  return <PrimaryButton {...rest}>{children}</PrimaryButton>;
}

/**
 * Scene 1 — Welcome. The deepest dark, the crescent low in the east, its ashen
 * disc almost below the threshold of perception, a kilim that says somebody has
 * been sitting here. The first emotion must be stillness, not promise.
 */
function Welcome({ navigate }) {
  return (
    <Shell theme="night" frame="full" className="welcome-screen" footer={(
      <Continue onClick={() => nextFrom('welcome', navigate)}>شروع</Continue>
    )}>
      <div className="ob-void" aria-hidden="true" />
      <ObHeader
        className="ob-caption ob-caption--welcome"
        epigraph="سپیده‌دم، پیش از آن‌که کسی بیدار شود"
        title="تیا را نمی‌بینی؛ با تیا می‌بینی."
        lede="در اسطوره‌ها، تیا الههٔ نور و دیدن بود. این‌جا چهره‌ای ندارد؛ همراهی است که کمک می‌کند آن‌چه از پیش در توست، روشن‌تر دیده شود."
      />
      <p className="ob-disclosure">تیا یک همراهِ هوش مصنوعی است، نه درمانگر.</p>
    </Shell>
  );
}

function Testimonials({ navigate, back }) {
  return (
    <Shell theme="paper" frame="sheet" scene="dawn" back={back} className="testimonials-screen scrolling-screen">
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
    <Shell theme="night" frame="sheet" scene="three" back={back} className="research-screen">
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
    <Shell theme="night" frame="sheet" scene="three" back={back} className="experts-screen">
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

/**
 * The four chapter openings. Each is a still from the same window at a later
 * hour, and each names one emotional movement rather than describing a scene.
 */
const INTERSTITIALS = {
  'simple-intro': {
    scene: 'night',
    chapter: 'فصل دو · شب',
    title: 'از تاریکی شروع می‌کنیم.',
    lede: 'نه چون چیزی خراب است؛ چون هر دیدنی از ندیدن شروع می‌شود. لازم نیست از پیش بدانی در تو چه می‌گذرد؛ چشم که عادت کند، چیزها کم‌کم پیدا می‌شوند.',
  },
  'deeper-intro': {
    scene: 'seeing',
    chapter: 'فصل سه · دیدن',
    title: 'آن‌چه تاریک به نظر می‌رسد، خالی نیست.',
    lede: 'بخشِ تاریکِ ماه هم کمی روشن است: نوری که از زمین به آن می‌رسد، به چشم تو برمی‌گردد. تیا چیزی به تو اضافه نمی‌کند و چیزی از تو نمی‌گیرد؛ فقط کمک می‌کند آن‌چه هست واضح‌تر دیده شود، حتی وقتی دیدنش آسان نیست.',
  },
  'know-intro': {
    scene: 'dawn',
    chapter: 'فصل چهار · سپیده',
    title: 'سپیده یک‌باره نمی‌آید.',
    lede: 'کم‌کم روشن می‌شود، آن‌قدر آرام که اول متوجهش نمی‌شوی. گاهی نوری می‌آید و دوباره پس می‌رود؛ آن هم بخشی از راه است. حال آدم هم اغلب همین‌طور عوض می‌شود.',
  },
  'almost-intro': {
    scene: 'three',
    chapter: 'فصل پنج · سه نور',
    title: 'سه نور؛ و هیچ‌کدام همیشگی نیست.',
    lede: 'خورشید، ماه و سپیده، هر سه فرزندانِ تیا بودند: سه حالتِ یک نور. حال‌های تو هم همین‌طورند: می‌آیند و می‌روند، و هیچ‌کدام تمامِ تو نیست.',
  },
};

function Interstitial({ route, navigate, back }) {
  const item = INTERSTITIALS[route];
  return (
    <Shell
      theme="night"
      frame="hero"
      back={back}
      scene={item.scene}
      className={`interstitial-screen scene-${item.scene}`}
      footer={<Continue onClick={() => nextFrom(route, navigate)}>ادامه</Continue>}
    >
      <p className="ob-chapter">{item.chapter}</p>
      <div className="ob-void" aria-hidden="true" />
      <ObHeader className="ob-caption" title={item.title} lede={item.lede} />
    </Shell>
  );
}

/**
 * Scene 2 — Privacy. The same sky, now through a window with two shutter leaves
 * and a gap between them that is exactly as wide as the user chooses. A window,
 * not a vault: you choose what passes through, and the screen says plainly where
 * the data actually goes.
 */
function Privacy({ navigate, back, privacySharing = true, privacyLoading = false, privacySaving = false,
  privacyLoaded = false, privacyError = false, onTogglePrivacy, onRetryPrivacy }) {
  const sharing = privacySharing;
  return (
    <Shell
      theme="night"
      frame="sheet"
      back={back}
      className="privacy-screen"
      footer={<Continue onClick={() => nextFrom('privacy', navigate)}>ادامه</Continue>}
    >
      <ObHeader
        className="ob-caption"
        title="چه‌قدر بگویی، انتخاب توست."
        lede="فقط تو تعیین می‌کنی چه‌قدر از خودت بگویی. برای اینکه پاسخ‌ها به تو نزدیک‌تر باشند، گفت‌وگوها و پاسخ‌های آشنایی اولیه روی سرور ذخیره می‌شوند و اطلاعات مرتبط به سرویس هوش مصنوعی فرستاده می‌شود."
      />
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

/**
 * Mood is shown as terrain, not a switch: the Moon's named seas as a small map
 * of the inner weather. The same five positions as before, the same answer, the
 * same slider semantics — only the picture changed from an abstract scribble to
 * the moon the whole story has been about.
 */
/**
 * The five answers, and the sea each one moves to. The order is the order of the
 * slider and of the seas themselves — crises, cold, clouds, tranquility, serenity
 * — so the light always travels in the direction the answer does.
 */
const MOODS = [
  { label: 'این روزها سخت می‌گذرد', note: 'لازم نیست توضیح بدهی. از همین‌جا شروع می‌کنیم.' },
  { label: 'انرژی زیادی ندارم', note: 'با چیزهای کوچک شروع می‌کنیم؛ لازم نیست بزرگ باشد.' },
  { label: 'یک روز خوب، یک روز سخت', note: 'روزها یکسان نیستند. کنار هم پیش می‌رویم.' },
  { label: 'دارم می‌گذرونم', note: 'داری راه می‌افتی. همین کافی است.' },
  { label: 'در کل حالم خوب است', note: 'خوب است. بگذار بیشتر بشناسیمش.' },
];

/**
 * The five seas of the held moon (دریاهای ماه), which the story has already named:
 * Crises, Cold, Clouds, Tranquility and Serenity. The slider moves between them,
 * so the answer is a place on the moon rather than a number — and the ashen disc
 * brightens as the answer does, exactly like the progress bar across the journey.
 * Positions are (x, y) in disc radii; the drawing is the real maria, in the same
 * blue-grey as the still behind it.
 */
const MOOD_SEAS = [
  {
    key: 'crises', name: 'دریای بحران‌ها', x: 0.34, y: -0.42, rx: 0.34, ry: 0.26,
    aria: 'دریای بحران‌ها؛ جای طوفان‌ها و تنگه‌ها',
  },
  {
    key: 'cold', name: 'دریای سرما', x: -0.44, y: -0.34, rx: 0.4, ry: 0.28,
    aria: 'دریای سرما؛ سمت سرد و آرام ماه',
  },
  {
    key: 'clouds', name: 'دریای ابرها', x: 0.16, y: 0.46, rx: 0.36, ry: 0.2,
    aria: 'دریای ابرها؛ جایی که هوا روی هم می‌نشیند',
  },
  {
    key: 'tranquility', name: 'دریای آرامش', x: 0.46, y: 0.04, rx: 0.44, ry: 0.4,
    aria: 'دریای آرامش؛ دشت بزرگ و باز ماه',
  },
  {
    key: 'serenity', name: 'دریای آسودگی', x: -0.16, y: -0.06, rx: 0.5, ry: 0.34,
    aria: 'دریای آسودگی؛ بخش روشن و هموار ماه',
  },
];

/**
 * A sea is a lobed lava plain, not an ellipse. The outline is a closed spline
 * through 14 points whose radii wobble on two frequencies, so the shape has bays
 * and headlands the way the real maria do — and no shape ever repeats.
 */
function seaPath(x, y, rx, ry, seed = 1) {
  const wobble = (k) => 1 + Math.sin(seed * 3.7 + k * 1.9) * 0.13 + Math.sin(seed * 7.1 + k * 4.3) * 0.07;
  const N = 14;
  const pts = Array.from({ length: N }, (_, i) => {
    const a = (i / N) * Math.PI * 2;
    const k = wobble(i);
    return [x + Math.cos(a) * rx * k, y + Math.sin(a) * ry * k];
  });
  // Catmull–Rom through the points, rendered as cubic Béziers
  const at = (i) => pts[(i + N) % N];
  let d = `M${at(0)[0].toFixed(2)} ${at(0)[1].toFixed(2)}`;
  for (let i = 0; i < N; i += 1) {
    const p0 = at(i - 1);
    const p1 = at(i);
    const p2 = at(i + 1);
    const p3 = at(i + 2);
    const c1 = [p1[0] + (p2[0] - p0[0]) / 6, p1[1] + (p2[1] - p0[1]) / 6];
    const c2 = [p2[0] - (p3[0] - p1[0]) / 6, p2[1] - (p3[1] - p1[1]) / 6];
    d += `C${c1[0].toFixed(2)} ${c1[1].toFixed(2)} ${c2[0].toFixed(2)} ${c2[1].toFixed(2)} ${p2[0].toFixed(2)} ${p2[1].toFixed(2)}`;
  }
  return `${d}Z`;
}

function MoodMoon({ level }) {
  // The same moon as the still behind the screen — same geometry, same palette —
  // and only the disc's legibility changes with the answer.
  const ash = 0.24 + level * 0.19;         // 0.24 → 1.0, the shared scale
  const R = 96;
  const active = MOOD_SEAS[level];
  const activeIndex = level;
  const seas = MOOD_SEAS.map((sea, index) => ({
    key: sea.key,
    d: seaPath(sea.x * R, sea.y * R, sea.rx * R, sea.ry * R, index + 3),
  }));
  return (
    <HeldMoonFigure
      className="mood-moon"
      ash={ash}
      seas={seas}
      activeIndex={activeIndex}
      label={`ماهِ در آغوش؛ جای این روزهای تو: ${active.name}. ${active.aria}`}
    />
  );
}

function MoodScreen({ navigate, back, answers, setAnswer }) {
  const mood = Number.isInteger(answers.mood) && answers.mood >= 0 && answers.mood <= 4 ? answers.mood : 2;
  return (
    <Shell
      theme="night"
      frame="sheet"
      scene="mood"
      back={back}
      className="mood-screen"
      footer={<Continue onClick={() => { setAnswer('mood', mood); nextFrom('mood', navigate); }}>ادامه</Continue>}
    >
      <ObHeader
        className="mood-head"
        title="این روزها چه حالی داری؟"
        lede="نوار را بکش تا نزدیک‌ترین حالت را پیدا کنی؛ لازم نیست دقیق باشد."
      />
      <div className="mood-stage">
        <MoodMoon level={mood} />
        <h2 aria-live="polite">{MOODS[mood].label}</h2>
        <p className="mood-sea-name" aria-hidden="true">{MOOD_SEAS[mood].name}</p>
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
      theme="night"
      frame="band"
      scene="checkins"
      back={back}
      className="checkins-screen"
      footer={<Continue onClick={() => nextFrom('checkins', navigate)}>ادامه</Continue>}
    >
      <div className="ob-void" aria-hidden="true" />
      <ObHeader
        className="ob-caption"
        title="اگر بخواهی، چراغ روشن می‌ماند."
        lede="تیا می‌تواند گاهی سراغ حالت را بگیرد، بی‌اصرار. هر وقت بخواهی می‌توانی یادآوری‌ها را خاموش کنی."
      />
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
      theme="night"
      frame="band"
      scene="three"
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
    <Shell theme="night" frame="sheet" scene="three" back={back} className="approach-screen">
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

/**
 * Scene 8 — the ring closes. The welcome frame again, now at first light: the
 * sun's ray reaches the kilim where somebody was sitting, the stair door stands
 * open, and the moon is still there, because the moon does not leave when the
 * day comes.
 */
function Ready({ navigate, savingProfile = false }) {
  return (
    <Shell
      theme="night"
      frame="full"
      scene="final"
      className="ready-screen"
      footer={(
        <Continue disabled={savingProfile} onClick={() => nextFrom('ready', navigate)}>
          {savingProfile ? 'در حال ذخیره…' : 'شروع گفت‌وگو'}
        </Continue>
      )}
    >
      <div className="ob-void" aria-hidden="true" />
      <p className="ob-chapter">فصل شش · آغاز مسیر</p>
      <ObHeader
        className="ob-caption ob-caption--ready"
        title="از این‌جا شروع می‌کنیم."
        lede="آسمان همان است؛ فقط چشمت عادت کرده. پاسخ‌های آشنایی اولیه‌ات ذخیره می‌شوند و برای شخصی‌سازی گفت‌وگو در اختیار سرویس هوش مصنوعی قرار می‌گیرند."
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
