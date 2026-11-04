import React, { useEffect, useRef, useState } from 'react';
import { Flame, Flower2, Landmark, Sparkles } from 'lucide-react';
import { APPROACHES, APPROACH_USER_MESSAGE, FLOW, QUESTIONS, REVIEWS, VOICES } from '../data';
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
        title="با همراه آشنا شو"
        description="همراه کمک می‌کنه دربارهٔ احساساتت حرف بزنی و خودت رو بهتر بشناسی."
      />
      <div className="welcome-art artwork-wrap">
        <img src="/assets/welcome-figures.png" alt="دو شخصیت گل‌مانند دست‌در دست" />
      </div>
      <PrimaryButton onClick={() => nextFrom('welcome', navigate)}>شروع</PrimaryButton>
    </Shell>
  );
}

function Testimonials({ navigate, back }) {
  return (
    <Shell theme="beige" back={back} className="testimonials-screen scrolling-screen">
      <PageHeader
        title="بقیه دربارهٔ همراه چی می‌گن؟"
        description="نظر کاربران همراه"
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
        title="همراه با کمک پژوهش‌های علمی طراحی شده."
        description="پژوهش‌های مرتبط با همراه"
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
  },
  'deeper-intro': {
    title: 'حالا بیشتر دربارهٔ خودت بگو.',
    description: 'با چند سؤال کوتاه کمکم کن بفهمم چرا اینجایی و چه چیزهایی برات مهم‌ترن.',
    art: 'medium',
  },
  'know-intro': {
    title: 'بذار بیشتر بشناسمت.',
    description: '',
    art: 'large',
  },
  'almost-intro': {
    title: 'دیگه چیزی نمونده!',
    description: 'حالا صدا و سبک حرف زدنم رو انتخاب کن.',
    art: 'tall',
  },
};

function Interstitial({ route, navigate, back }) {
  const item = INTERSTITIALS[route];
  return (
    <Shell theme="dark" back={back} className={`interstitial-screen art-${item.art}`}>
      <PageHeader title={item.title} description={item.description} />
      <div className="interstitial-art artwork-wrap">
        <img src={`/assets/flower-person-${item.art}.png`} alt="شخصیت گل‌مانند همراه" />
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
        description="گفت‌وگوهات ذخیره می‌شن تا بعداً ادامه‌شون بدی."
      />
      <img className="privacy-art" src="/assets/privacy-flower.png" alt="گل با نشان قفل" />
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

function AgeScreen({ navigate, back, answers, setAnswer }) {
  const values = ['۵۵–۶۴', '۴۵–۵۴', '۳۵–۴۴', '۳۰–۳۴', '۲۵–۲۹', '۱۸–۲۴', 'کمتر از ۱۸'];
  const selected = answers.age || '۲۵–۲۹';
  const index = values.indexOf(selected);
  const touchStart = useRef(0);
  const move = (direction) => {
    const next = Math.max(0, Math.min(values.length - 1, index + direction));
    setAnswer('age', values[next]);
  };
  return (
    <Shell back={back} className="age-screen">
      <PageHeader title="چند سالته؟" description="بازهٔ سنیت رو انتخاب کن." />
      <div
        className="age-picker"
        onWheel={(event) => move(event.deltaY > 0 ? 1 : -1)}
        onTouchStart={(event) => { touchStart.current = event.touches[0].clientY; }}
        onTouchEnd={(event) => {
          const distance = touchStart.current - event.changedTouches[0].clientY;
          if (Math.abs(distance) > 25) move(distance > 0 ? 1 : -1);
        }}
      >
        {values.map((value, itemIndex) => (
          <button
            type="button"
            key={value}
            className={`${value === selected ? 'is-selected' : ''} age-distance-${Math.min(3, Math.abs(itemIndex - index))}`}
            style={{
              '--age-shift': `${itemIndex === index ? 0 : itemIndex < index
                ? -[0, 58, 96, 126, 146, 166, 186][index - itemIndex]
                : [0, 42, 86, 118, 148, 178, 208][itemIndex - index]}px`,
            }}
            onClick={() => setAnswer('age', value)}
          >{value}</button>
        ))}
      </div>
      <PrimaryButton onClick={() => nextFrom('age', navigate)}>ادامه</PrimaryButton>
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
    <Shell theme={theme} back={back} className={`question-screen question-${route} ${config.skip ? 'question-screen--skip' : ''} ${config.type === 'single' ? 'question-screen--single' : 'question-screen--multi'}`}>
      <PageHeader title={config.title} description={config.description} />
      {config.type === 'multi' && config.limit && (
        <div className="question-selection-meta" aria-live="polite">
          <span>{selected.length.toLocaleString('fa-IR')} از {Number(config.limit).toLocaleString('fa-IR')} انتخاب شده</span>
        </div>
      )}
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
    </svg>
  );
}

function MoodScreen({ navigate, back, answers, setAnswer }) {
  const mood = Number.isInteger(answers.mood) && answers.mood >= 0 && answers.mood <= 4 ? answers.mood : 2;
  const maxIndex = MOODS.length - 1;
  return (
    <Shell back={back} className="mood-screen">
      <PageHeader title="این روزها چه حسی داشتی؟" description="نوار رو بکش تا به حالت نزدیک‌تر بشه." />
      <div className="mood-stage">
        <div className="mood-glow" aria-hidden="true" />
        <MoodScribble level={mood} />
        <h2 aria-live="polite">{MOODS[mood].label}</h2>
        <div className="mood-slider-wrap" dir="ltr" style={{ '--mood-index': mood }}>
          <div className="mood-track" aria-hidden="true">
            <div className="mood-track-line" />
            <div className="mood-ticks">
              {MOODS.map((_, index) => (
                <i
                  key={index}
                  className={index === mood ? 'is-active' : ''}
                  style={{ left: `${(index / maxIndex) * 100}%` }}
                />
              ))}
            </div>
            <span
              className="mood-indicator"
              style={{ left: `${(mood / maxIndex) * 100}%` }}
            >
              <span className="mood-indicator-core" />
            </span>
          </div>
          <input
            className="mood-slider"
            type="range"
            min="0"
            max="4"
            step="1"
            value={mood}
            onChange={(event) => setAnswer('mood', Number(event.target.value))}
            aria-label="حس این روزهای من"
            aria-valuetext={MOODS[mood].label}
          />
        </div>
      </div>
      <p className="mood-footnote" aria-live="polite">{MOODS[mood].note}</p>
      <PrimaryButton onClick={() => nextFrom('mood', navigate)}>ادامه</PrimaryButton>
    </Shell>
  );
}

function CheckinsScreen({ navigate, back }) {
  return (
    <Shell back={back} className="checkins-screen">
      <PageHeader title="یه یادآوری، درست وقتی لازمه." description="همراه می‌تونه گاهی حالت رو بپرسه. هر وقت بخوای می‌تونی یادآوری‌ها رو خاموش کنی." />
      <div className="checkins-art" aria-hidden="true">
        <svg viewBox="0 0 300 265" focusable="false">
          <rect width="300" height="265" rx="18" fill="var(--beige)" />
          <path d="M119 189 L106 110 Q107 59 151 55 Q194 49 207 108 L213 189Z" fill="#fff" />
          <g fill="none" stroke="#211f1c" strokeWidth="4.5" strokeLinecap="round" strokeLinejoin="round">
            <path d="M126 57 Q122 35 139 34 Q153 35 151 55" />
            <path d="M102 192 Q152 177 238 160" />
            <path d="M151 184 Q156 211 174 207 Q184 204 186 183" />
            <path d="M212 68 Q220 53 234 59 Q250 55 258 70 Q274 80 260 91 Q267 105 251 108 Q241 123 225 111 Q207 118 202 103 Q187 92 201 82 Q198 72 212 68Z" />
            <path d="M230 80 Q241 71 245 83 Q251 95 237 96 Q225 94 230 80Z" />
          </g>
        </svg>
      </div>
      <PrimaryButton onClick={() => nextFrom('checkins', navigate)}>ادامه</PrimaryButton>
    </Shell>
  );
}

function VoiceScreen({ navigate, back, answers, setAnswer }) {
  const initial = Math.max(0, VOICES.findIndex((voice) => voice.name === answers.voice));
  const [active, setActive] = useState(initial);
  const [direction, setDirection] = useState(1);
  const [drag, setDrag] = useState(0);
  const [resonating, setResonating] = useState(false);
  const gesture = useRef(null);
  const pulseTimer = useRef(null);

  useEffect(() => () => window.clearTimeout(pulseTimer.current), []);

  const prevIndex = (active - 1 + VOICES.length) % VOICES.length;
  const nextIndex = (active + 1) % VOICES.length;
  const voice = VOICES[active];
  const prevVoice = VOICES[prevIndex];
  const nextVoice = VOICES[nextIndex];

  const select = (index, preferredDir) => {
    const next = ((index % VOICES.length) + VOICES.length) % VOICES.length;
    if (next === active) {
      setResonating(true);
      window.clearTimeout(pulseTimer.current);
      pulseTimer.current = window.setTimeout(() => setResonating(false), 620);
      return;
    }
    setDirection(preferredDir ?? (next > active ? 1 : -1));
    setResonating(false);
    setActive(next);
    setAnswer('voice', VOICES[next].name);
  };

  const finishGesture = (event, cancelled = false) => {
    const start = gesture.current;
    if (!start || start.id !== event.pointerId) return;
    gesture.current = null;
    setDrag(0);
    if (event.currentTarget.hasPointerCapture?.(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    const dx = event.clientX - start.x;
    const dy = event.clientY - start.y;
    if (!cancelled && Math.abs(dx) > 32 && Math.abs(dx) > Math.abs(dy) * 1.15) {
      // In RTL, swiping right (dx > 0) reveals the next card on the left; swiping left (dx < 0) reveals previous,
      // matching natural horizontal strip dragging where cards follow the finger.
      const step = dx < 0 ? 1 : -1;
      select(active + step, step);
    }
  };

  return (
    <Shell theme="dark" back={back} className="voice-screen">
      <PageHeader title="دوست داری چه صدایی داشته باشم؟" description="ورق بزن یا یکی از صداها رو امتحان کن. بعداً هم قابل تغییره." />
      <div
        className="voice-stage"
        role="region"
        aria-label="انتخاب صدای همراه"
        tabIndex={0}
        onKeyDown={(event) => {
          if (event.key === 'ArrowLeft') { event.preventDefault(); select(active + 1, 1); }
          else if (event.key === 'ArrowRight') { event.preventDefault(); select(active - 1, -1); }
          else if (event.key === 'Home') { event.preventDefault(); select(0, -1); }
          else if (event.key === 'End') { event.preventDefault(); select(VOICES.length - 1, 1); }
        }}
        onPointerDown={(event) => {
          if (!event.isPrimary || event.button !== 0 || event.target.closest('.voice-pill')) return;
          gesture.current = { id: event.pointerId, x: event.clientX, y: event.clientY };
          event.currentTarget.setPointerCapture?.(event.pointerId);
        }}
        onPointerMove={(event) => {
          const start = gesture.current;
          if (!start || start.id !== event.pointerId) return;
          const dx = event.clientX - start.x;
          if (Math.abs(dx) > Math.abs(event.clientY - start.y)) {
            setDrag(Math.max(-52, Math.min(52, dx * 0.38)));
          }
        }}
        onPointerUp={finishGesture}
        onPointerCancel={(event) => finishGesture(event, true)}
        onLostPointerCapture={(event) => finishGesture(event, true)}
      >
        <div className="voice-deck">
          <button
            type="button"
            className="voice-peek-card voice-peek-card--prev"
            dir="rtl"
            onClick={() => select(prevIndex, -1)}
            aria-label={`صدای ${prevVoice.name}، ${prevVoice.description}`}
          >
            <div className="voice-peek-wave">
              <VoiceWave path={prevVoice.path} />
            </div>
            <span className="voice-peek-name">{prevVoice.name}</span>
          </button>

          <article
            className={`voice-active-card ${drag ? 'is-dragging' : ''}`}
            dir="rtl"
            style={{ '--voice-drag': `${drag}px`, '--voice-enter': `${direction * 18}px` }}
            aria-live="polite"
          >
            <button
              type="button"
              className="voice-preview"
              onClick={() => select(active)}
              aria-label={`موج صدای ${voice.name} — ${voice.description}`}
            >
              <span className="voice-index-badge">
                {(active + 1).toLocaleString('fa-IR')} از {VOICES.length.toLocaleString('fa-IR')}
              </span>
              <VoiceWave path={voice.path} resonating={resonating} />
            </button>
            <div className="voice-copy" key={voice.name}>
              <h2>{voice.name}</h2>
              <p>{voice.description}</p>
              {voice.character && <small>{voice.character}</small>}
            </div>
          </article>

          <button
            type="button"
            className="voice-peek-card voice-peek-card--next"
            dir="rtl"
            onClick={() => select(nextIndex, 1)}
            aria-label={`صدای ${nextVoice.name}، ${nextVoice.description}`}
          >
            <div className="voice-peek-wave">
              <VoiceWave path={nextVoice.path} />
            </div>
            <span className="voice-peek-name">{nextVoice.name}</span>
          </button>
        </div>

        <div className="voice-pills" role="tablist" aria-label="فهرست صداها">
          {VOICES.map((item, index) => (
            <button
              key={item.name}
              type="button"
              role="tab"
              aria-selected={index === active}
              className={`voice-pill ${index === active ? 'is-active' : ''}`}
              onClick={() => select(index, index > active ? 1 : -1)}
            >
              {item.name}
            </button>
          ))}
        </div>
      </div>
      <PagerDots count={VOICES.length} active={active} onChange={(index) => select(index, index > active ? 1 : -1)} dark />
      <PrimaryButton
        light
        onClick={() => {
          setAnswer('voice', voice.name);
          nextFrom('voice', navigate);
        }}
      >
        انتخاب صدای «{voice.name}»
      </PrimaryButton>
    </Shell>
  );
}

function ApproachScreen({ navigate, back, answers, setAnswer, profileEdit = false, savingProfile = false, onSaveApproach }) {
  const initial = answers.approach ? Math.max(0, APPROACHES.findIndex((item) => item.name === answers.approach)) : 2;
  const [active, setActive] = useState(initial);
  const [direction, setDirection] = useState(1);
  const [drag, setDrag] = useState(0);
  const gesture = useRef(null);
  const selectedIndex = useRef(initial);
  const pillsRef = useRef(null);
  const approach = APPROACHES[active];

  useEffect(() => {
    const activeChip = pillsRef.current?.querySelector('.is-active');
    activeChip?.scrollIntoView?.({ block: 'nearest', inline: 'center', behavior: 'smooth' });
  }, [active]);

  const select = (index) => {
    const next = Math.max(0, Math.min(APPROACHES.length - 1, index));
    if (next === selectedIndex.current) return;
    setDirection(next > selectedIndex.current ? 1 : -1);
    selectedIndex.current = next;
    setActive(next);
    setAnswer('approach', APPROACHES[next].name);
  };

  const finishGesture = (event, cancelled = false) => {
    const start = gesture.current;
    if (!start || start.id !== event.pointerId) return;
    gesture.current = null;
    setDrag(0);
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
    const dx = event.clientX - start.x;
    const dy = event.clientY - start.y;
    if (!cancelled && Math.abs(dx) > 35 && Math.abs(dx) > Math.abs(dy) * 1.2) select(active + (dx < 0 ? 1 : -1));
  };

  return (
    <Shell theme="beige" back={back} className="approach-screen">
      <PageHeader title="دوست داری چطوری باهات حرف بزنم؟" description="یک پیام ثابت و پنج شیوهٔ همراهی — سبک‌ها رو جابه‌جا کن تا تفاوتشون رو ببینی." />
      <div className="approach-pills" ref={pillsRef} role="tablist" aria-label="سبک‌های گفت‌وگو">
        {APPROACHES.map((item, index) => (
          <button
            key={item.name}
            type="button"
            role="tab"
            aria-selected={index === active}
            className={`approach-pill ${index === active ? 'is-active' : ''}`}
            onClick={() => select(index)}
          >
            {item.name}
          </button>
        ))}
      </div>
      <div
        className="approach-stage"
        role="region"
        aria-label="نمونهٔ سبک گفت‌وگو"
        tabIndex={0}
        onKeyDown={(event) => {
          const target = { ArrowLeft: active + 1, ArrowRight: active - 1, Home: 0, End: APPROACHES.length - 1 }[event.key];
          if (target === undefined) return;
          event.preventDefault();
          select(target);
        }}
        onPointerDown={(event) => {
          if (!event.isPrimary || event.button !== 0 || event.target.closest('button')) return;
          gesture.current = { id: event.pointerId, x: event.clientX, y: event.clientY };
          event.currentTarget.setPointerCapture(event.pointerId);
        }}
        onPointerMove={(event) => {
          const start = gesture.current;
          if (!start || start.id !== event.pointerId) return;
          const dx = event.clientX - start.x;
          if (Math.abs(dx) > Math.abs(event.clientY - start.y)) setDrag(Math.max(-48, Math.min(48, dx * 0.35)));
        }}
        onPointerUp={finishGesture}
        onPointerCancel={(event) => finishGesture(event, true)}
        onLostPointerCapture={(event) => finishGesture(event, true)}
      >
        <button type="button" className="approach-peek approach-peek--before" aria-label="سبک قبلی" disabled={active === 0} onClick={() => select(active - 1)} />
        <article className={`approach-card${drag ? ' is-dragging' : ''}`} style={{ '--approach-drag': `${drag}px` }} aria-live="polite" aria-atomic="true">
          <div className="approach-chat-topbar">
            <span className="approach-chat-brand">
              <Flower2 size={15} strokeWidth={1.8} aria-hidden="true" />
              <strong>همراه</strong>
            </span>
            <span className="approach-style-badge" key={`badge-${approach.name}`}>
              <Sparkles size={12} aria-hidden="true" />
              <span>{approach.name}</span>
            </span>
          </div>
          <div className="approach-chat-body">
            <div className="approach-user-wrap">
              <span className="approach-sender-label">پیام تو</span>
              <p className="approach-user">{APPROACH_USER_MESSAGE}</p>
            </div>
            <div className="approach-reply-wrap" key={approach.name} style={{ '--approach-enter': `${direction * 16}px` }}>
              <div className="approach-reply-meta">
                <span className="approach-reply-dot" aria-hidden="true" />
                <span>پاسخ همراه · {approach.subtitle}</span>
              </div>
              <p className="approach-response">{approach.response}</p>
            </div>
          </div>
        </article>
        <button type="button" className="approach-peek approach-peek--after" aria-label="سبک بعدی" disabled={active === APPROACHES.length - 1} onClick={() => select(active + 1)} />
      </div>
      <PagerDots count={APPROACHES.length} active={active} onChange={select} />
      <PrimaryButton className="with-skip" disabled={savingProfile} onClick={() => {
        if (profileEdit) onSaveApproach(approach.name);
        else { setAnswer('approach', approach.name); nextFrom('approach', navigate); }
      }}>{profileEdit ? (savingProfile ? 'در حال ذخیره…' : 'ذخیرهٔ سبک گفت‌وگو') : `انتخاب «${approach.label || approach.name}»`}</PrimaryButton>
      <button type="button" className="skip-button" onClick={() => profileEdit ? back() : nextFrom('approach', navigate)}>{profileEdit ? 'انصراف' : 'فعلاً رد کن'}</button>
    </Shell>
  );
}

function Ready({ navigate, savingProfile = false }) {
  return (
    <Shell className="ready-screen">
      <PageHeader
        title="من آماده‌ام!"
        description=""
      />
      <div className="ready-art artwork-wrap"><img src="/assets/ready-daisy.png" alt="گل همراه در حال قدم‌زدن" /></div>
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
