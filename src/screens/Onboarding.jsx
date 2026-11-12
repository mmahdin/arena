import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Check, ChevronLeft, ChevronRight, Flame, Landmark } from 'lucide-react';
import { AGE_RANGES, APPROACHES, APPROACH_EXAMPLE_USER, FLOW, QUESTIONS, REVIEWS, VOICES } from '../data';
import { resistVoiceDrag, voiceSwipeTarget } from '../features/onboarding/voiceCarousel';
import {
  TheiaMark,
  TheiaWelcomeArt,
  TheiaPrivacyArt,
  TheiaPulseArt,
  TheiaReadyArt,
  TheiaPointArt,
  TheiaOpeningArt,
  TheiaOrbitArt,
  TheiaSkyArt,
} from '../components/TheiaBrand';
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

function Shell({ theme = 'light', back, children, className = '' }) {
  const dark = theme === 'dark';
  return (
    <section className={`app-screen theme-${theme} ${className}`}>
      <StatusBar dark={dark} />
      {back && <BackButton onClick={back} dark={dark} />}
      {children}
    </section>
  );
}

function Welcome({ navigate }) {
  return (
    <Shell className="welcome-screen">
      <PageHeader
        title="با تیا آشنا شو"
        description="تیا کمک می‌کنه دربارهٔ احساساتت حرف بزنی و خودت رو بهتر بشناسی."
      />
      <div className="welcome-art artwork-wrap">
        <TheiaWelcomeArt />
      </div>
      <PrimaryButton onClick={() => nextFrom('welcome', navigate)}>شروع</PrimaryButton>
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
    <Shell back={back} className="name-screen">
      <PageHeader title="دوست داری چی صدات کنم؟" description="با همین اسم صدات می‌کنم." />
      <form onSubmit={submit}>
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
        <PrimaryButton type="submit" disabled={!name.trim() || savingProfile}>{savingProfile ? 'در حال ذخیره…' : profileEdit ? 'ذخیره نام' : 'ادامه'}</PrimaryButton>
      </form>
    </Shell>
  );
}

const INTERSTITIALS = {
  'simple-intro': {
    title: 'اول چند سؤال ساده بپرسم.',
    description: 'جواب‌هات کمک می‌کنن بهتر بشناسمت.',
    art: 'small',
    Art: TheiaPointArt,
  },
  'deeper-intro': {
    title: 'حالا بیشتر دربارهٔ خودت بگو.',
    description: 'با چند سؤال کوتاه کمکم کن بفهمم چرا اینجایی و چه چیزهایی برات مهم‌ترن.',
    art: 'medium',
    Art: TheiaOpeningArt,
  },
  'know-intro': {
    title: 'بذار بیشتر بشناسمت.',
    description: '',
    art: 'large',
    Art: TheiaOrbitArt,
  },
  'almost-intro': {
    title: 'دیگه چیزی نمونده!',
    description: 'حالا صدا و سبک حرف زدنم رو انتخاب کن.',
    art: 'tall',
    Art: TheiaSkyArt,
  },
};

function Interstitial({ route, navigate, back }) {
  const item = INTERSTITIALS[route];
  return (
    <Shell theme="dark" back={back} className={`interstitial-screen art-${item.art}`}>
      <PageHeader title={item.title} description={item.description} />
      <div className="interstitial-art artwork-wrap">
        <item.Art />
      </div>
      <PrimaryButton light onClick={() => nextFrom(route, navigate)}>ادامه</PrimaryButton>
    </Shell>
  );
}

function Privacy({ navigate, back, privacySharing = true, privacyLoading = false, privacySaving = false,
  privacyLoaded = false, privacyError = false, onTogglePrivacy, onRetryPrivacy }) {
  const sharing = privacySharing;
  return (
    <Shell back={back} className="privacy-screen">
      <PageHeader
        title="حریم خصوصی تو مهمه."
        description="گفت‌وگوها و پاسخ‌های آشنایی اولیه روی سرور ذخیره می‌شن. برای پاسخ متناسب با تو، اطلاعات مرتبط به سرویس هوش مصنوعی فرستاده می‌شن."
      />
      <TheiaPrivacyArt className="privacy-art" />
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
      <PrimaryButton onClick={() => nextFrom('privacy', navigate)}>ادامه</PrimaryButton>
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
    <Shell back={back} className="age-screen">
      <PageHeader title="چند سالته؟" description="بکش یا اسکرول کن تا بازهٔ سنیت رو انتخاب کنی." />
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
      <PrimaryButton onClick={() => { setAnswer('age', AGE_RANGES[activeRef.current]); nextFrom('age', navigate); }}>ادامه</PrimaryButton>
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
  const theme = config.theme || 'light';

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
    <Shell theme={theme} back={back} className={`question-screen question-${route} ${config.skip ? 'question-screen--skip' : ''}`}>
      <PageHeader title={config.title} description={config.description} />
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
      {config.type === 'multi' && (
        <PrimaryButton
          className={config.skip ? 'with-skip' : ''}
          disabled={!hasSelection && !config.allowEmpty}
          onClick={() => nextFrom(route, navigate)}
        >{config.confirmLabel || 'ادامه'}</PrimaryButton>
      )}
      {config.skip && (
        <button type="button" className="skip-button" onClick={() => nextFrom(route, navigate)}>فعلاً رد کن</button>
      )}
    </Shell>
  );
}

// Deliberately drawn as native SVG, not screenshots, so it stays sharp and
// changes smoothly at all five mood positions. Direction follows the reference:
// left is struggling and right is generally good, regardless of page RTL.
const MOODS = [
  { label: 'این روزها سخت می‌گذره', note: 'آروم و قدم‌به‌قدم پیش می‌ریم.' },
  { label: 'انرژی زیادی ندارم', note: 'فعلاً از چیزهای ساده شروع می‌کنیم.' },
  { label: 'یه روز خوب، یه روز سخت', note: 'روزبه‌روز جلو می‌ریم.' },
  { label: 'دارم می‌گذرونم', note: 'با هم بهترش می‌کنیم.' },
  { label: 'در کل حالم خوبه', note: 'چه خوب! بیا بیشتر ازش بگیم.' },
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
  return (
    <svg className="mood-scribble" viewBox="0 0 340 300" aria-hidden="true" focusable="false">
      <path d={path} fill="none" stroke="currentColor" strokeWidth="2.7" strokeLinecap="round" strokeLinejoin="round" />
      {level >= 3 && (
        <path
          d="M170 137L172.6 144.8L180.4 147.4L172.6 150L170 157.8L167.4 150L159.6 147.4L167.4 144.8L170 137Z"
          fill="currentColor"
          opacity={level === 4 ? '0.88' : '0.45'}
        />
      )}
    </svg>
  );
}

function MoodScreen({ navigate, back, answers, setAnswer }) {
  const mood = Number.isInteger(answers.mood) && answers.mood >= 0 && answers.mood <= 4 ? answers.mood : 2;
  return (
    <Shell back={back} className="mood-screen">
      <PageHeader title="این روزها چه حسی داشتی؟" description="نوار رو بکش تا به حالت نزدیک‌تر بشه." />
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
      <PrimaryButton onClick={() => { setAnswer('mood', mood); nextFrom('mood', navigate); }}>ادامه</PrimaryButton>
    </Shell>
  );
}

function CheckinsScreen({ navigate, back }) {
  return (
    <Shell back={back} className="checkins-screen">
      <PageHeader title="یه یادآوری، درست وقتی لازمه." description="تیا می‌تونه گاهی حالت رو بپرسه. هر وقت بخوای می‌تونی یادآوری‌ها رو خاموش کنی." />
      <div className="checkins-art">
        <TheiaPulseArt />
      </div>
      <PrimaryButton onClick={() => nextFrom('checkins', navigate)}>ادامه</PrimaryButton>
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
    <Shell theme="dark" back={back} className="voice-screen">
      <PageHeader title="دوست داری چه صدایی داشته باشم؟" />
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
      <PagerDots count={VOICES.length} active={active} onChange={select} dark label="انتخاب صدا" labels={VOICES.map((item) => item.name)} />
      <PrimaryButton light onClick={() => nextFrom('voice', navigate)}>ادامه با «{voice.name}»</PrimaryButton>
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
    <Shell theme="beige" back={back} className="approach-screen">
      <PageHeader title="چطور باهات حرف بزنم؟" />
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
    <Shell className="ready-screen">
      <PageHeader
        title="من آماده‌ام!"
        description="با شروع گفت‌وگو، پاسخ‌های آشنایی اولیه‌ات ذخیره می‌شن و برای شخصی‌سازی چت در اختیار سرویس هوش مصنوعی قرار می‌گیرن."
      />
      <div className="ready-art artwork-wrap"><TheiaReadyArt /></div>
      <PrimaryButton disabled={savingProfile} onClick={() => nextFrom('ready', navigate)}>{savingProfile ? 'در حال ذخیره…' : 'شروع گفت‌وگو'}</PrimaryButton>
    </Shell>
  );
}

export default function OnboardingScreen(props) {
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
