import React, { createContext, useCallback, useContext, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Check, ChevronLeft, ChevronRight } from 'lucide-react';
import { APPROACHES, APPROACH_EXAMPLE_USER, FLOW, QUESTIONS, VOICES } from '../data';
import { resistVoiceDrag, voiceSwipeTarget } from '../features/onboarding/voiceCarousel';
import {
  BackButton,
  ChoiceButton,
  PageHeader,
  PagerDots,
  PrimaryButton,
  StatusBar,
  VoiceWave,
} from '../components/UI';
import { LightStudy, MoodLight, TheiaMark, TheiaWordmark } from '../components/Brand';

const nextFrom = (route, navigate) => {
  const nextIndex = FLOW.indexOf(route) + 1;
  if (nextIndex > 0 && nextIndex < FLOW.length) navigate(FLOW[nextIndex]);
};

const ONBOARDING_LENGTH = FLOW.filter((item) => item !== 'home').length;
const ProgressContext = createContext(0);

function Shell({ theme = 'light', back, children, className = '' }) {
  const progress = useContext(ProgressContext);
  return (
    <section className={`app-screen theme-${theme} ${className}`}>
      <span className="screen-progress" aria-hidden="true"><i style={{ '--progress': progress }} /></span>
      <StatusBar />
      {back && <BackButton onClick={back} dark={theme === 'dark'} />}
      {children}
    </section>
  );
}

/* ---------- Welcome: a light in a dark room ---------- */

function Welcome({ navigate }) {
  return (
    <Shell theme="dark" className="welcome-screen">
      <div className="welcome-brand">
        <span className="welcome-brand__mark">
          <TheiaMark size={30} color="var(--text-paper)" />
          <TheiaWordmark size={27} />
        </span>
        <span className="welcome-brand__tag">همراهی برای دیدنِ روشن‌تر</span>
      </div>
      <div className="welcome-copy">
        <h1>یه جای آرام برای حرف زدن.</h1>
        <p>تیا کنارته تا احساساتت رو ببینی و خودت رو بهتر بشناسی.</p>
      </div>
      <div className="screen-art screen-art--wide">
        <LightStudy scene="welcome" />
      </div>
      <PrimaryButton onClick={() => nextFrom('welcome', navigate)}>شروع کنیم</PrimaryButton>
      <p className="welcome-footnote">اول چند سؤال کوتاه ازت می‌پرسم.</p>
    </Shell>
  );
}

/* ---------- Privacy ---------- */

function Privacy({ navigate, back, privacySharing = true, privacyLoading = false, privacySaving = false,
  privacyLoaded = false, privacyError = false, onTogglePrivacy, onRetryPrivacy }) {
  const sharing = privacySharing;
  return (
    <Shell back={back} className="privacy-screen">
      <PageHeader
        title="حرف‌های تو، پیش خودت می‌مونه."
        description="گفت‌وگوهات ذخیره می‌شن تا هر وقت برگشتی ادامه بدی. همیشه می‌تونی این اجازه رو عوض کنی."
      />
      <div className="screen-art">
        <LightStudy scene="privacy" />
      </div>
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
        <span>استفاده از حرف‌هام برای پاسخ‌های بهتر</span>
      </button>
      {privacyError && <div className="connected-notice" role="alert">
        <p>تنظیمات حریم خصوصی در دسترس نیست. دوباره تلاش کن.</p>
        <button type="button" onClick={onRetryPrivacy}>تلاش دوباره</button>
      </div>}
      <PrimaryButton onClick={() => nextFrom('privacy', navigate)}>ادامه</PrimaryButton>
    </Shell>
  );
}

/* ---------- The story between questions ---------- */

const INTERSTITIALS = {
  'simple-intro': {
    scene: 'simple',
    title: 'اول چند سؤال ساده.',
    description: 'جواب‌هات کمک می‌کنن بهتر بشناسمت.',
  },
  'deeper-intro': {
    scene: 'deeper',
    title: 'حالا یه‌کم عمیق‌تر.',
    description: 'چند سؤال کوتاه، تا بفهمم این روزها چی برات مهم‌تره.',
  },
  'know-intro': {
    scene: 'know',
    title: 'خودت رو بهتر ببین.',
    description: 'هر جواب، یه تکه از تصویری که از خودت داری.',
  },
  'almost-intro': {
    scene: 'almost',
    title: 'نزدیک آخر راهیم.',
    description: 'حالا صدای تیا و شکل حرف زدنش رو انتخاب کن.',
  },
};

function Interstitial({ route, navigate, back }) {
  const item = INTERSTITIALS[route];
  return (
    <Shell theme="dark" back={back} className="interstitial-screen">
      <PageHeader title={item.title} description={item.description} />
      <div className="screen-art screen-art--tall">
        <LightStudy scene={item.scene} />
      </div>
      <PrimaryButton onClick={() => nextFrom(route, navigate)}>ادامه</PrimaryButton>
    </Shell>
  );
}

/* ---------- Name ---------- */

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
      <PageHeader title="چی صدات کنم؟" description="با همین اسم صدات می‌کنم." />
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

/* ---------- Age: a physical wheel ---------- */

const AGE_RANGES = ['۵۵–۶۴', '۴۵–۵۴', '۳۵–۴۴', '۳۰–۳۴', '۲۵–۲۹', '۱۸–۲۴', 'کمتر از ۱۸'];
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
  // answer is written until the person actually touches the wheel.
  const chosenRef = useRef(AGE_RANGES.includes(answers.age) ? startIndex : -1);
  const frameRef = useRef(0);
  const settleRef = useRef(0);
  const interactedRef = useRef(false);
  const dragRef = useRef(null);
  const draggedRef = useRef(false);

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
      node.style.setProperty('--age-scale', (1 - 0.34 * progress ** 1.14).toFixed(3));
      node.style.setProperty('--age-fade', (1 - 0.55 * progress).toFixed(3));
      const tilt = Math.sign(distance) * progress * 9;
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
    draggedRef.current = false;
    const wheel = wheelRef.current;
    if (!wheel || dragRef.current || event.pointerType !== 'mouse' || event.button !== 0) return;
    const startTop = wheel.scrollTop;
    const startY = event.clientY;
    const snap = wheel.style.scrollSnapType;
    const behavior = wheel.style.scrollBehavior;
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
    const sizeAndCenter = (index) => {
      const node = optionRefs.current[index];
      if (!node) return;
      wheel.style.setProperty('--age-pad', `${Math.max(0, (wheel.clientHeight - node.offsetHeight) / 2)}px`);
      wheel.scrollTop = centeredOffset(node);
    };
    sizeAndCenter(startIndexRef.current);
    paint();
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
        <span key={`age-marker-${tick}`} className="age-marker" aria-hidden="true" />
      </div>
      <p className="sr-only" aria-live="polite">{`بازهٔ انتخاب‌شده: ${AGE_RANGES[active]}`}</p>
      <PrimaryButton onClick={() => nextFrom('age', navigate)}>ادامه</PrimaryButton>
    </Shell>
  );
}

/* ---------- Questions ---------- */

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

/* ---------- Mood: one line that loosens ---------- */

const MOODS = [
  { label: 'این روزها سخت می‌گذره', note: 'آروم و قدم‌به‌قدم پیش می‌ریم.' },
  { label: 'انرژی زیادی ندارم', note: 'فعلاً از چیزهای ساده شروع می‌کنیم.' },
  { label: 'یه روز خوب، یه روز سخت', note: 'روزبه‌روز جلو می‌ریم.' },
  { label: 'دارم می‌گذرونم', note: 'با هم بهترش می‌کنیم.' },
  { label: 'در کل حالم خوبه', note: 'چه خوب! بیا بیشتر ازش بگیم.' },
];

function MoodScreen({ navigate, back, answers, setAnswer }) {
  const mood = Number.isInteger(answers.mood) && answers.mood >= 0 && answers.mood <= 4 ? answers.mood : 2;
  return (
    <Shell theme="dark" back={back} className="mood-screen">
      <PageHeader title="این روزها چه حسی داری؟" />
      <div className="mood-stage">
        <MoodLight level={mood} />
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
      <PrimaryButton onClick={() => nextFrom('mood', navigate)}>ادامه</PrimaryButton>
    </Shell>
  );
}

/* ---------- Check-ins ---------- */

function CheckinsScreen({ navigate, back }) {
  return (
    <Shell back={back} className="checkins-screen">
      <PageHeader title="یه یادآوری، درست وقتی لازمه." description="تیا می‌تونه گاهی حالت رو بپرسه. هر وقت بخوای می‌تونی یادآوری‌ها رو خاموش کنی." />
      <div className="screen-art">
        <LightStudy scene="checkins" />
      </div>
      <PrimaryButton onClick={() => nextFrom('checkins', navigate)}>ادامه</PrimaryButton>
    </Shell>
  );
}

/* ---------- Voice ---------- */

function VoiceScreen({ navigate, back, answers, setAnswer }) {
  const initial = Math.max(0, VOICES.findIndex((item) => item.name === answers.voice));
  const [active, setActive] = useState(initial);
  const [drag, setDrag] = useState(0);
  const [dragging, setDragging] = useState(false);
  const [base, setBase] = useState(0);
  const [cardMetrics, setCardMetrics] = useState({ width: 188, step: 202 });
  const carouselRef = useRef(null);
  const trackRef = useRef(null);
  const cardRefs = useRef([]);
  const activeRef = useRef(initial);
  const gestureRef = useRef(null);
  const suppressClickRef = useRef(false);
  const lastWheelRef = useRef(-Infinity);
  const voice = VOICES[active];

  // Offsets are measured from layout, never from transforms, so interrupting a
  // snap or resizing never shifts the centred card.
  useLayoutEffect(() => {
    const measure = () => {
      const first = cardRefs.current[0];
      const second = cardRefs.current[1];
      if (!first) return;
      const width = first.offsetWidth;
      const step = second ? Math.abs(second.offsetLeft - first.offsetLeft) : width + 14;
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

  useLayoutEffect(() => {
    const card = cardRefs.current[active];
    const carousel = carouselRef.current;
    if (!card || !carousel) return;
    setBase(carousel.clientWidth / 2 - (card.offsetLeft + card.offsetWidth / 2));
  }, [active, cardMetrics]);

  const select = (index) => {
    const next = Math.max(0, Math.min(VOICES.length - 1, index));
    if (next === activeRef.current) {
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
    gestureRef.current = {
      id: event.pointerId, x: event.clientX, y: event.clientY,
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
    setDrag(resistVoiceDrag(dx, activeRef.current, VOICES.length));
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
  const offset = base + drag;

  return (
    <Shell theme="dark" back={back} className="voice-screen">
      <PageHeader title="تیا با چه صدایی حرف بزنه؟" description="این صداها فعلاً فقط برای حس‌وحال انتخابن، پخش صوتی نداریم." />
      <div
        className={`voice-carousel ${dragging ? 'is-dragging' : ''}`}
        ref={carouselRef}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={finishGesture}
        onPointerCancel={(event) => finishGesture(event, true)}
        onLostPointerCapture={(event) => finishGesture(event, true)}
        onClickCapture={(event) => {
          if (suppressClickRef.current) { event.preventDefault(); event.stopPropagation(); suppressClickRef.current = false; }
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
                  opacity: 0.5 + 0.5 * proximity,
                  transform: `scale(${0.93 + 0.07 * proximity})`,
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
                {index === active && <span className="voice-card-check" aria-hidden="true"><Check size={13} strokeWidth={2.4} /></span>}
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

/* ---------- Approach: one shared prompt, five ways of listening ---------- */

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
          const offset = atEdge ? Math.max(-42, Math.min(42, dx * 0.16)) : Math.max(-distance, Math.min(distance, dx));
          setDrag(offset);
        }}
        onPointerUp={finishGesture}
        onPointerCancel={(event) => finishGesture(event, true)}
        onLostPointerCapture={(event) => finishGesture(event, true)}
      >
        <div className="approach-picker">
          <button type="button" className="approach-direction" aria-label="سبک قبلی" disabled={active === 0 || savingProfile} onClick={() => select(active - 1)}>
            <ChevronRight size={17} strokeWidth={1.5} aria-hidden="true" />
          </button>
          <h2 id="approach-style">{approach.label || approach.name}</h2>
          <button type="button" className="approach-direction" aria-label="سبک بعدی" disabled={active === APPROACHES.length - 1 || savingProfile} onClick={() => select(active + 1)}>
            <ChevronLeft size={17} strokeWidth={1.5} aria-hidden="true" />
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
              <span className="approach-mark" aria-hidden="true"><TheiaMark size={19} color="var(--ember-deep)" /></span>
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

/* ---------- Ready ---------- */

function Ready({ navigate, savingProfile = false }) {
  return (
    <Shell className="ready-screen">
      <PageHeader title="از اینجا شروع می‌شه." description="من همین‌جام؛ هر وقت خواستی حرف بزنیم." />
      <div className="screen-art screen-art--wide">
        <LightStudy scene="ready" />
      </div>
      <PrimaryButton disabled={savingProfile} onClick={() => nextFrom('ready', navigate)}>{savingProfile ? 'در حال ذخیره…' : 'شروع گفت‌وگو'}</PrimaryButton>
    </Shell>
  );
}

/* ---------- Legacy info routes, rewritten honestly for the new brand ---------- */

function About({ navigate, back }) {
  return (
    <Shell back={back} className="about-screen">
      <PageHeader title="دربارهٔ تیا" description="یک همراه آرام برای حرف زدن؛ بدون قضاوت و بدون قولِ معجزه." />
      <div className="screen-art">
        <LightStudy scene="horizon" />
      </div>
      <div className="about-body">
        <p>تیا درمانگر نیست و تشخیص نمی‌دهد. اگر حالت خوب نیست، حرف زدن با یک متخصص یا یک آدم قابل‌اعتماد همیشه گزینهٔ درستی است.</p>
        <ul className="about-list">
          <li>تو را به خوشحالی مجبور نمی‌کند؛ فقط کمک می‌کند ببینی.</li>
          <li>در موقعیت بحرانی: اورژانس اجتماعی ۱۲۳ یا یک آدم قابل‌اعتماد.</li>
        </ul>
      </div>
      <PrimaryButton onClick={() => navigate('home')}>برگشت</PrimaryButton>
    </Shell>
  );
}

function Voices({ back, title, description, paragraphs, action, onAction }) {
  return (
    <Shell theme="dark" back={back} className="about-screen">
      <PageHeader title={title} description={description} />
      <div className="screen-art">
        <LightStudy scene="almost" />
      </div>
      <div className="about-body">
        {paragraphs.map((paragraph) => <p key={paragraph}>{paragraph}</p>)}
      </div>
      {action && <PrimaryButton light onClick={onAction}>{action}</PrimaryButton>}
    </Shell>
  );
}

function Testimonials({ navigate, back }) {
  return (
    <Voices
      back={back}
      title="حرفِ آدم‌ها"
      description="اینجا هنوز حرفی از کسی نیست."
      paragraphs={[
        'تیا تازه شروع کرده و ترجیح می‌دهد به‌جای ساختنِ تعریف‌های تازه، اول کارش را درست انجام بدهد.',
        'اگر روزی کسی بخواهد تجربه‌اش را به اشتراک بگذارد، فقط با اجازهٔ خودش اینجا می‌آید.',
      ]}
      action="ادامه"
      onAction={() => navigate('home')}
    />
  );
}

function Research({ navigate, back }) {
  return <About navigate={navigate} back={back} />;
}

function Experts({ navigate, back }) {
  return (
    <Voices
      back={back}
      title="پشتِ تیا"
      description="یک تیم کوچک، بی‌عجله."
      paragraphs={[
        'تیا با یک تیم کوچک طراحی و ساخته می‌شود. اسم‌ها و همکاری‌ها وقتی نهایی شوند، همین‌جا با جزئیات می‌آیند.',
        'تا آن روز، هیچ مدرک، تأییدیه یا همکاری‌ای را ادعا نمی‌کنیم.',
      ]}
      action="ادامه"
      onAction={() => navigate('home')}
    />
  );
}

export default function OnboardingScreen(props) {
  const { route } = props;
  const index = FLOW.filter((item) => item !== 'home').indexOf(route);
  const progress = index >= 0 ? Math.min(0.99, (index + 1) / ONBOARDING_LENGTH) : 0;
  let screen = null;
  if (route === 'welcome') screen = <Welcome {...props} />;
  else if (route === 'testimonials') screen = <Testimonials {...props} />;
  else if (route === 'research') screen = <Research {...props} />;
  else if (route === 'experts') screen = <Experts {...props} />;
  else if (route === 'name') screen = <NameScreen {...props} />;
  else if (INTERSTITIALS[route]) screen = <Interstitial {...props} />;
  else if (route === 'privacy') screen = <Privacy {...props} />;
  else if (route === 'age') screen = <AgeScreen {...props} />;
  else if (route === 'mood') screen = <MoodScreen {...props} />;
  else if (route === 'checkins') screen = <CheckinsScreen {...props} />;
  else if (QUESTIONS[route]) screen = <QuestionScreen {...props} />;
  else if (route === 'voice') screen = <VoiceScreen {...props} />;
  else if (route === 'approach') screen = <ApproachScreen {...props} />;
  else if (route === 'ready') screen = <Ready {...props} />;
  return <ProgressContext.Provider value={progress}>{screen}</ProgressContext.Provider>;
}
