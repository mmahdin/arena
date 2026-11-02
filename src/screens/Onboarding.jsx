import React, { useRef, useState } from 'react';
import { Flame, Landmark } from 'lucide-react';
import { APPROACHES, QUESTIONS, REVIEWS, VOICES } from '../data';
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

const nextFrom = (route, navigate) => navigate(NEXT[route] || 'home');
const NEXT = {
  welcome: 'testimonials',
  testimonials: 'research',
  research: 'experts',
  experts: 'name',
  name: 'simple-intro',
  'simple-intro': 'privacy',
  privacy: 'deeper-intro',
  'deeper-intro': 'therapist',
  therapist: 'age',
  age: 'concerns',
  concerns: 'focus',
  focus: 'feelings',
  feelings: 'neurodivergent',
  neurodivergent: 'faith',
  faith: 'know-intro',
  'know-intro': 'interests',
  interests: 'relationships',
  relationships: 'strengths',
  strengths: 'barriers',
  barriers: 'almost-intro',
  'almost-intro': 'voice',
  voice: 'approach',
  approach: 'ready',
  ready: 'home',
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
        title="با همراه سلام کن."
        description="همراه، هوشی مصنوعی برای سلامت روان و حمایت عاطفی است."
      />
      <div className="welcome-art artwork-wrap">
        <img src="/assets/welcome-figures.png" alt="دو شخصیت گل‌مانند دست‌در دست" />
      </div>
      <PrimaryButton onClick={() => nextFrom('welcome', navigate)}>شروع کنیم</PrimaryButton>
    </Shell>
  );
}

function Testimonials({ navigate, back }) {
  return (
    <Shell theme="beige" back={back} className="testimonials-screen scrolling-screen">
      <PageHeader
        title="فقط حرف ما را باور نکن."
        description="تجربه‌ی دیگران؛ بیش از ۶٬۰۰۰ امتیاز و همچنان بیشتر."
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
        title="همراه بر پایه‌ی پژوهش‌های معتبر ساخته شده است."
        description="بیش از ۱۴ مقاله‌ی پژوهشی، و ادامه‌ی مسیر."
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
      <PageHeader title="با همراهی تیمی از متخصصان." />
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

function NameScreen({ navigate, back, name, setName }) {
  const submit = (event) => {
    event.preventDefault();
    if (name.trim()) nextFrom('name', navigate);
  };
  return (
    <Shell back={back} className="name-screen">
      <PageHeader title="اسمت چیست؟" description="همراه با همین اسم صدایت می‌کند." />
      <form onSubmit={submit}>
        <label className="sr-only" htmlFor="name">نام تو</label>
        <input
          id="name"
          className="name-input"
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="نام تو"
          autoComplete="given-name"
          maxLength={30}
        />
        <PrimaryButton type="submit" disabled={!name.trim()}>ادامه</PrimaryButton>
      </form>
    </Shell>
  );
}

const INTERSTITIALS = {
  'simple-intro': {
    title: 'از ساده شروع کنیم.',
    description: 'همراه با دانستن زمینه بهتر کار می‌کند. هرچه بیشتر بداند، گفت‌وگوی اول بهتری خواهید داشت.',
    art: 'small',
  },
  'deeper-intro': {
    title: 'حالا کمی عمیق‌تر برویم.',
    description: 'به چند پرسش کوتاه جواب بده تا همراه بفهمد چرا اینجایی.',
    art: 'medium',
  },
  'know-intro': {
    title: 'حالا کمی بهتر بشناسیمت.',
    description: 'به همراه کمک کن تصویر بزرگ‌تری ببیند. هرچه بیشتر بگویی، بینش بهتری می‌گیری.',
    art: 'large',
  },
  'almost-intro': {
    title: 'تقریباً رسیدی.',
    description: 'ممنون که این‌قدر باز بودی. حالا همراه را مطابق ترجیحاتت تنظیم کنیم.',
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
      <PrimaryButton light onClick={() => nextFrom(route, navigate)}>آماده‌ام</PrimaryButton>
    </Shell>
  );
}

function Privacy({ navigate, back, answers, setAnswer }) {
  const sharing = answers.shareAnonymized ?? true;
  return (
    <Shell back={back} className="privacy-screen">
      <PageHeader
        title="تعهد ما به تو."
        description="سلامت روان، مسئله‌ای شخصی و خصوصی است. گفت‌وگوها امن روی گوشی تو ذخیره می‌شوند و هر وقت بخواهی می‌توانی حسابت را حذف کنی."
      />
      <img className="privacy-art" src="/assets/privacy-flower.png" alt="گل با نشان قفل" />
      <button
        type="button"
        className="privacy-card"
        role="switch"
        aria-checked={sharing}
        onClick={() => setAnswer('shareAnonymized', !sharing)}
      >
        <span className={`toggle ${sharing ? 'is-on' : ''}`} aria-hidden="true"><span /></span>
        <span>با اشتراک‌گذاری گفت‌وگوهای ناشناسم به بهترشدن همراه برای همه کمک می‌کنم.</span>
      </button>
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
      <PageHeader title="چند سالته؟" description="برای انتخاب، بالا و پایین برو." />
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
  const config = QUESTIONS[route];
  const value = answers[route];
  const selected = config.type === 'multi' ? (value || []) : value;
  const hasSelection = config.type === 'multi' ? selected.length > 0 : Boolean(selected);
  const theme = config.theme || 'light';

  const choose = (option) => {
    if (config.type === 'single') {
      setAnswer(route, option);
      window.setTimeout(() => nextFrom(route, navigate), 160);
      return;
    }
    const exists = selected.includes(option);
    if (!exists && config.limit && selected.length >= config.limit) {
      notify(`می‌توانی حداکثر ${Number(config.limit).toLocaleString('fa-IR')} مورد را انتخاب کنی.`);
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
          disabled={!hasSelection}
          onClick={() => nextFrom(route, navigate)}
        >{config.confirmLabel || 'ادامه'}</PrimaryButton>
      )}
      {config.skip && (
        <button type="button" className="skip-button" onClick={() => nextFrom(route, navigate)}>فعلاً رد کن</button>
      )}
    </Shell>
  );
}

function VoiceScreen({ navigate, back, answers, setAnswer }) {
  const initial = Math.max(0, VOICES.findIndex((voice) => voice.name === answers.voice));
  const [active, setActive] = useState(initial);
  const touchStart = useRef(0);
  const voice = VOICES[active];
  const select = (index) => {
    const next = (index + VOICES.length) % VOICES.length;
    setActive(next);
    setAnswer('voice', VOICES[next].name);
  };
  return (
    <Shell theme="dark" back={back} className="voice-screen">
      <PageHeader title="کدام صدا را بیشتر می‌پسندی؟" description="هر وقت بخواهی می‌توانی صدای همراه را عوض کنی." />
      <div
        className="voice-carousel"
        onTouchStart={(event) => { touchStart.current = event.touches[0].clientX; }}
        onTouchEnd={(event) => {
          const distance = touchStart.current - event.changedTouches[0].clientX;
          if (Math.abs(distance) > 35) select(active + (distance > 0 ? 1 : -1));
        }}
      >
        <button type="button" className="voice-preview" onClick={() => select(active + 1)} aria-label="شنیدن نمونه و رفتن به صدای بعد">
          <VoiceWave path={voice.path} />
        </button>
        <div className="voice-copy" key={voice.name}>
          <h2>{voice.name}</h2>
          <p>{voice.description}</p>
        </div>
      </div>
      <PagerDots count={VOICES.length} active={active} onChange={select} dark />
      <p className="voice-note">اگر گوشی در حالت بی‌صداست، نمونه‌ی صداها را نخواهی شنید.</p>
      <PrimaryButton light onClick={() => nextFrom('voice', navigate)}>همین صدا</PrimaryButton>
    </Shell>
  );
}

function ApproachScreen({ navigate, back, answers, setAnswer }) {
  const initial = answers.approach ? Math.max(0, APPROACHES.findIndex((item) => item.name === answers.approach)) : 2;
  const [active, setActive] = useState(initial);
  const touchStart = useRef(0);
  const approach = APPROACHES[active];
  const select = (index) => {
    const next = Math.max(0, Math.min(APPROACHES.length - 1, index));
    setActive(next);
    setAnswer('approach', APPROACHES[next].name);
  };
  return (
    <Shell theme="beige" back={back} className="approach-screen">
      <PageHeader title="کدام سبک گفت‌وگو را ترجیح می‌دهی؟" />
      <div
        className="approach-stage"
        onTouchStart={(event) => { touchStart.current = event.touches[0].clientX; }}
        onTouchEnd={(event) => {
          const distance = touchStart.current - event.changedTouches[0].clientX;
          if (Math.abs(distance) > 35) select(active + (distance > 0 ? 1 : -1));
        }}
      >
        <span className="approach-peek approach-peek--before" />
        <article className="approach-card" key={approach.name}>
          <p className="approach-user">سلام همراه، کمی گیر کرده‌ام.</p>
          <p className="approach-response">{approach.response}</p>
        </article>
        <span className="approach-peek approach-peek--after" />
      </div>
      <PagerDots count={APPROACHES.length} active={active} onChange={select} />
      <PrimaryButton className="with-skip" onClick={() => nextFrom('approach', navigate)}>امتحان «{approach.name}»</PrimaryButton>
      <button type="button" className="skip-button" onClick={() => nextFrom('approach', navigate)}>فعلاً رد کن</button>
    </Shell>
  );
}

function Ready({ navigate }) {
  return (
    <Shell className="ready-screen">
      <PageHeader
        title="همراهِ تو بالاخره آماده است."
        description="ممنون که این‌قدر از خودت گفتی! اگر آماده‌ای، همراه از اینجا به بعد با هدف‌ها و نیازهایت هماهنگ می‌شود."
      />
      <div className="ready-art artwork-wrap"><img src="/assets/ready-daisy.png" alt="گل همراه در حال قدم‌زدن" /></div>
      <PrimaryButton onClick={() => nextFrom('ready', navigate)}>آماده‌ام</PrimaryButton>
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
  if (QUESTIONS[route]) return <QuestionScreen {...props} />;
  if (route === 'voice') return <VoiceScreen {...props} />;
  if (route === 'approach') return <ApproachScreen {...props} />;
  if (route === 'ready') return <Ready {...props} />;
  return null;
}
