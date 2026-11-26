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
 * sight and far-shining light, never appears as a body — only as one celestial
 * shore (sky, horizon, sea) whose light changes from night to dawn. Six
 * chapters carry the metaphor from night to a completed dawn; each chapter
 * names the emotional movement instead of decorating it.
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
  return (
    <div className={`ob-art ${className}`.trim()}>
      <TheiaInstrument state={state} tone={tone} label={label} />
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
      <Instrument state="welcome" className="ob-art--hero" label="ساحلی در نخستین نور: خورشید بر افق، سروها بر دماغه و بازتاب نور بر آب" />
      <ObHeader
        epigraph="روشناییِ دوربرد"
        title="تیا را نمی‌بینی؛ با تیا می‌بینی."
        lede="تیا، الههٔ یونانیِ نور و دیدن، هیچ تصویری از خودش ندارد؛ تنها چیزی که از او مانده روشنایی است. این‌جا هم همین‌طور است: نه یک چهره، بلکه نوری که کمک می‌کند خودت را واضح‌تر ببینی."
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
    title: 'تیا نور را نمی‌سازد؛ آن را می‌بیند.',
    lede: 'پس چیزی به تو اضافه نمی‌کند و چیزی هم از تو نمی‌گیرد. فقط کمک می‌کند آن‌چه هست را واضح‌تر ببینی — حتی وقتی واضح دیدن راحت نیست.',
  },
  'know-intro': {
    state: 'dawn',
    chapter: 'فصل چهار · سپیده',
    title: 'سپیده یک‌باره نمی‌آید.',
    lede: 'کم‌کم روشن می‌شود، آن‌قدر آرام که اول متوجهش نمی‌شوی. حال آدم هم همین‌طور عوض می‌شود.',
  },
  'almost-intro': {
    state: 'orbits',
    chapter: 'فصل پنج · سه نور',
    title: 'خورشید، ماه، سپیده.',
    lede: 'هر سه فرزندان تیا بودند؛ سه حالتِ نور، و هیچ‌کدام همیشگی نبود. حال تو هم همین‌طور است: می‌آید و می‌رود و هیچ‌کدام تمامِ تو نیست.',
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
        title="حریم تو، مثل این حلقه است."
        lede="فقط تو تعیین می‌کنی چقدر باز شود. برای اینکه پاسخ‌ها متناسب با تو باشند، گفت‌وگوها و پاسخ‌های آشنایی اولیه روی سرور ذخیره می‌شوند و اطلاعات مرتبط به سرویس هوش مصنوعی فرستاده می‌شود."
      />
      <Instrument state="veil" tone="paper" className="ob-art--small" label="پرده‌ای که در میانه کنار رفته و شکافی از نور را بر ساحل باز کرده است" />
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

// Deliberately drawn as native SVG, not screenshots, so it stays sharp at all
// five mood positions. Direction follows the reference: left is struggling and
// right is generally good, regardless of page RTL.
const MOODS = [
  { label: 'این روزها سخت می‌گذرد', note: 'لازم نیست توضیح بدهی. از همین‌جا شروع می‌کنیم.' },
  { label: 'انرژی زیادی ندارم', note: 'با چیزهای کوچک شروع می‌کنیم؛ لازم نیست بزرگ باشد.' },
  { label: 'یک روز خوب، یک روز سخت', note: 'روزها یکسان نیستند. کنار هم پیش می‌رویم.' },
  { label: 'دارم می‌گذرونم', note: 'داری راه می‌افتی. همین کافی است.' },
  { label: 'در کل حالم خوب است', note: 'خوب است. بگذار بیشتر بشناسیمش.' },
];

/**
 * The mood is drawn as the same celestial shore as the onboarding scenes:
 * night with a fading moon at the hard end, the sun rising at the horizon as
 * the slider moves right. The place never changes — only the light does.
 */
// Five hours of the same shore, mirrored from the onboarding scenes: the mood
// slider moves the light, not the place. Left is night, right is full morning.
const MOOD_SKIES = [
  {
    sky: ['#04060C', '#0A1120', '#16203A', '#26344E'],
    sunY: 214, sunR: 0, moon: 0.9, moonY: 62, stars: 1, rim: 0.12,
    water: ['#1E2A3E', '#070A11'], glow: 0.1, path: 0.12,
  },
  {
    sky: ['#05070F', '#101828', '#2A2A3E', '#5A4446'],
    sunY: 200, sunR: 17, moon: 0.45, moonY: 56, stars: 0.65, rim: 0.24,
    water: ['#24304A', '#080B12'], glow: 0.35, path: 0.32,
  },
  {
    sky: ['#081020', '#1E2438', '#4A3442', '#B0714A'],
    sunY: 184, sunR: 26, moon: 0.1, moonY: 50, stars: 0.28, rim: 0.4,
    water: ['#2A3450', '#0A0D14'], glow: 0.6, path: 0.55,
  },
  {
    sky: ['#0D1626', '#2A2C40', '#6E4450', '#D99364'],
    sunY: 170, sunR: 29, moon: 0, moonY: 46, stars: 0.12, rim: 0.52,
    water: ['#2E3852', '#0B0E15'], glow: 0.78, path: 0.72,
  },
  {
    sky: ['#16203A', '#3A3448', '#8A5650', '#F2B47E'],
    sunY: 156, sunR: 33, moon: 0, moonY: 44, stars: 0, rim: 0.64,
    water: ['#333D56', '#0C1017'], glow: 0.95, path: 0.9,
  },
];
const MOOD_STARS = [
  [42, 52, 1.3], [92, 30, 1.0], [138, 66, 1.1], [206, 40, 1.1],
  [254, 72, 1.2], [302, 42, 0.9], [62, 104, 0.9], [292, 104, 0.8],
  [176, 24, 0.8], [238, 96, 0.7],
];
const HORIZON_Y = 196;


/**
 * The mood is drawn as the same celestial shore as the onboarding scenes: at
 * the hard end a fading moon over dark water, at the easy end the sun risen
 * over the ridge with its light on the water. The place never changes.
 */
function MoodScribble({ level }) {
  const scene = MOOD_SKIES[level] || MOOD_SKIES[2];
  const sunX = 172;
  const ridgeFar = 'M0 178 C22 172 44 166 70 170 C96 174 116 184 140 186 C166 188 188 176 214 172 C240 168 264 176 288 180 C306 183 320 184 340 182';
  const ridgeNear = 'M0 196 C26 188 48 186 76 192 C106 198 130 202 158 202 C188 202 212 194 238 192 C264 190 296 196 340 200';
  return (
    <svg className="mood-scribble" viewBox="0 0 340 300" aria-hidden="true" focusable="false">
      <defs>
        <linearGradient id="mood-sky" x1="0" y1="0" x2="0" y2={HORIZON_Y} gradientUnits="userSpaceOnUse">
          {scene.sky.map((c, i) => (
            <stop key={i} offset={[0, 0.34, 0.66, 0.9][i]} stopColor={c} />
          ))}
        </linearGradient>
        <radialGradient id="mood-sun" cx="50%" cy="38%" r="62%">
          <stop stopColor="#FFF3DA" />
          <stop offset="0.45" stopColor="#F2C588" />
          <stop offset="1" stopColor="#C98650" />
        </radialGradient>
        <radialGradient id="mood-halo" cx="50%" cy="50%" r="50%">
          <stop stopColor="#F8DFAE" stopOpacity="0.55" />
          <stop offset="0.55" stopColor="#F2C58A" stopOpacity="0.18" />
          <stop offset="1" stopColor="#F2C58A" stopOpacity="0" />
        </radialGradient>
        <radialGradient id="mood-bloom" cx="50%" cy="50%" r="50%">
          <stop stopColor="#FFC98C" stopOpacity="0.62" />
          <stop offset="0.6" stopColor="#F0A868" stopOpacity="0.18" />
          <stop offset="1" stopColor="#F0A868" stopOpacity="0" />
        </radialGradient>
        <radialGradient id="mood-reflect" cx="50%" cy="50%" r="50%">
          <stop stopColor="#E3C48C" stopOpacity="0.3" />
          <stop offset="1" stopColor="#E3C48C" stopOpacity="0" />
        </radialGradient>
        <linearGradient id="mood-sea" x1="0" y1={HORIZON_Y} x2="0" y2="300" gradientUnits="userSpaceOnUse">
          <stop stopColor={scene.water[0]} />
          <stop offset="1" stopColor={scene.water[1]} />
        </linearGradient>
        <linearGradient id="mood-sea-glow" x1="0" y1={HORIZON_Y} x2="0" y2={HORIZON_Y + 44} gradientUnits="userSpaceOnUse">
          <stop stopColor="#E8B87A" stopOpacity="0.5" />
          <stop offset="1" stopColor="#E8B87A" stopOpacity="0" />
        </linearGradient>
        <mask id="mood-moon">
          <rect x="0" y="0" width="340" height="300" fill="#000" />
          <circle cx="256" cy={scene.moonY} r="15" fill="#fff" />
          <circle cx="264" cy={scene.moonY - 4} r="14" fill="#000" />
        </mask>
        <radialGradient id="mood-vignette" cx="50%" cy="46%" r="72%">
          <stop stopColor="#000" stopOpacity="0" />
          <stop offset="0.72" stopColor="#000" stopOpacity="0" />
          <stop offset="1" stopColor="#000" stopOpacity="0.34" />
        </radialGradient>
      </defs>

      <rect x="0" y="0" width="340" height="300" rx="18" fill="url(#mood-sky)" />

      <g className="mood-fade" style={{ opacity: scene.stars }}>
        {MOOD_STARS.map(([x, y, r], i) => (
          <circle key={i} cx={x} cy={y} r={r} fill="#EDE0C8" opacity={0.5 + (i % 3) * 0.17} />
        ))}
      </g>
      <g className="mood-fade" style={{ opacity: scene.moon }}>
        <circle cx="256" cy={scene.moonY} r="34" fill="url(#mood-halo)" opacity="0.26" />
        <g mask="url(#mood-moon)">
          <circle cx="256" cy={scene.moonY} r="15" fill="#EAE4D4" />
        </g>
      </g>
      <g className="mood-fade" style={{ opacity: scene.sunR ? 1 : 0 }}>
        <ellipse cx={sunX} cy={HORIZON_Y - 8} rx="150" ry="46" fill="url(#mood-bloom)" opacity={scene.glow} />
        <circle cx={sunX} cy={scene.sunY} r={scene.sunR * 2.6} fill="url(#mood-halo)" opacity="0.7" />
        <circle cx={sunX} cy={scene.sunY} r={scene.sunR} fill="url(#mood-sun)" />
      </g>

      <path d={`${ridgeFar} L340 ${HORIZON_Y} L0 ${HORIZON_Y} Z`} fill="#1E2736" />
      <path d={`${ridgeNear} L340 ${HORIZON_Y} L0 ${HORIZON_Y} Z`} fill="#141B26" />
      <path d={ridgeFar} fill="none" stroke="#E3C48C" strokeWidth="0.8" opacity={scene.rim} />
      <path d={ridgeNear} fill="none" stroke="#E3C48C" strokeWidth="0.6" opacity={scene.rim * 0.6} />

      <rect x="0" y={HORIZON_Y} width="340" height={300 - HORIZON_Y} fill="url(#mood-sea)" />
      <rect x="0" y={HORIZON_Y} width="340" height="44" fill="url(#mood-sea-glow)" opacity="0.45" />

      <g className="mood-fade" style={{ opacity: scene.path }}>
        <ellipse cx={sunX} cy={HORIZON_Y + 14} rx="42" ry="24" fill="url(#mood-reflect)" />
        <path d={`M${sunX - 26} ${HORIZON_Y + 6} Q${sunX} ${HORIZON_Y + 9} ${sunX + 26} ${HORIZON_Y + 6}`} stroke="#E3C48C" strokeWidth="1.3" strokeLinecap="round" opacity="0.5" />
        <path d={`M${sunX - 19} ${HORIZON_Y + 22} Q${sunX} ${HORIZON_Y + 26} ${sunX + 19} ${HORIZON_Y + 22}`} stroke="#E3C48C" strokeWidth="1" strokeLinecap="round" opacity="0.3" />
        <path d={`M${sunX - 12} ${HORIZON_Y + 38} Q${sunX} ${HORIZON_Y + 42} ${sunX + 12} ${HORIZON_Y + 38}`} stroke="#E3C48C" strokeWidth="0.9" strokeLinecap="round" opacity="0.18" />
      </g>

      {/* the same cypress headland entering from the lower left */}
      <path d="M0 262 C26 254 54 252 82 258 C116 266 142 284 178 300 L0 300 Z" fill="#080B11" />
      <path d="M0 262 C26 254 54 252 82 258 C116 266 142 284 178 300" fill="none" stroke="#E3C48C" strokeWidth="0.9" opacity="0.3" />
      <g fill="#04060A">
        <path d="M43 186 C47.5 204 49 232 45 256 C44 262 40 262 39 256 C35 232 38 204 43 186 Z" />
        <path d="M67 200 C70.5 216 71.5 238 68 256 C67 261 64 261 63 256 C60 238 63 216 67 200 Z" />
        <path d="M84 224 C86 234 86.5 246 85 256 C84.5 259 82.5 259 82 256 C80.5 246 82 234 84 224 Z" />
      </g>

      {level >= 3 && (
        <g fill="none" stroke="#0A0D13" strokeWidth="1.2" strokeLinecap="round" opacity="0.7">
          <path d="M118 92 Q123 87 128 92 Q133 87 138 92" />
          <path d="M232 68 Q236 64 240 68 Q244 64 248 68" />
        </g>
      )}

      <rect x="0" y="0" width="340" height="300" rx="18" fill="url(#mood-vignette)" />
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
        title="یادآوری، درست وقتی لازم است."
        lede="تیا می‌تواند گاهی حالت را بپرسد. هر وقت بخواهی می‌توانی یادآوری‌ها را خاموش کنی."
      />
      <Instrument state="pulse" tone="paper" className="ob-art--small" label="فانوس دریایی بر دماغه؛ نوری که هر شب بازمی‌گردد" />
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
      <Instrument state="seal" className="ob-art--hero" label="سپیدهٔ کامل: خورشید در حلقهٔ بسته و بازتاب آرام آن بر آب" />
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
