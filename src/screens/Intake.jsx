import React, { useMemo, useRef, useState } from 'react';
import {
  LANGUAGES, ROADMAP, GENDERS, COUNTRIES, CARRYING, EASED, LESS_HEAVY,
  MOVE_UP, YOURSELF, PERSON, FOUND, WHICH_AI, SCALE_OPTS, PHQ, GAD,
  REFLECTIONS, STATS,
} from '../data.js';
import { Pill, TopBar, Stepper, Options, Note, faNum } from '../components/UI.jsx';
import { ChatCheck, Mic, Google, Apple, CheckCircle } from '../components/Icons.jsx';

export function IntakeShell({ back, onBack, p, children, footer, top }) {
  return (
    <div className="screen" style={{ background: 'var(--light)' }}>
      <TopBar back={back} onBack={onBack} />
      {p !== undefined && <Stepper p={p} />}
      {top}
      <div className="scroll">{children}</div>
      {footer && <div className="btn-row">{footer}</div>}
    </div>
  );
}

const H = ({ children, sub }) => (
  <div style={{ padding: '18px 24px 0' }}>
    <h1 className="h-serif">{children}</h1>
    {sub && <p className="sub-sans">{sub}</p>}
  </div>
);

/* ---------- name ---------- */
export function NameScreen({ onBack, onNext, state, set }) {
  const ok = state.name.trim().length > 0;
  return (
    <IntakeShell
      back onBack={onBack} p={0.15}
      footer={<Pill disabled={!ok} onClick={onNext}>ادامه</Pill>}
    >
      <H>چه صدایت کنیم؟</H>
      <div className="field-under">
        <input
          value={state.name}
          onChange={(e) => set({ name: e.target.value })}
          placeholder="نام تو"
          autoFocus
        />
        <div className="rule" />
      </div>
      <Note>
        درمانگر یا مربی هوش مصنوعی تو بین جلسه‌ها هم حالت را می‌پرسد و آنچه برایت مهم است به خاطر می‌سپارد. هر وقت بخواهی، هست.
      </Note>
    </IntakeShell>
  );
}

/* ---------- language ---------- */
export function LanguageScreen({ onBack, onNext, state, set }) {
  return (
    <IntakeShell
      back onBack={onBack} p={0.15}
      footer={<Pill disabled={!state.language} onClick={onNext}>ادامه</Pill>}
    >
      <H>به چه زبانی فکر می‌کنی؟</H>
      <Options items={LANGUAGES} value={state.language} onChange={(v) => set({ language: v })} />
      <div style={{ height: 24 }} />
    </IntakeShell>
  );
}

/* ---------- roadmap ---------- */
export function RoadmapScreen({ onBack, onNext, onSignin }) {
  return (
    <IntakeShell
      back onBack={onBack} p={0.12}
      footer={
        <div className="btn-row gap" style={{ padding: 0 }}>
          <Pill onClick={onNext}>بزن بریم</Pill>
          <Pill variant="outline-olive" onClick={onSignin}>
            حساب داری؟ <b>وارد شو</b>
          </Pill>
        </div>
      }
    >
      <H>چند دقیقهٔ پیشِ رو این‌طور می‌گذرد.</H>
      <div className="roadmap">
        <div className="steps">
          {ROADMAP.map((t, i) => (
            <div className="step" key={t}>
              <div className="col">
                <span className="ring"><i /></span>
                {i < ROADMAP.length - 1 && <span className="link" />}
              </div>
              <div className="txt" style={{ paddingBottom: i < ROADMAP.length - 1 ? 26 : 0 }}>{t}</div>
            </div>
          ))}
        </div>
      </div>
      <div style={{ height: 30 }} />
    </IntakeShell>
  );
}

/* ---------- welcome (name echo) ---------- */
export function WelcomeScreen({ onNext, name }) {
  return (
    <div className="screen grad bg-welcome">
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '0 40px' }}>
        <h1 className="h-serif" style={{ color: '#22232a', textAlign: 'center', fontSize: 30, lineHeight: 1.65 }}>
          به رهراه خوش آمدی، {name || 'دوست عزیز'}.
        </h1>
      </div>
      <div className="content">
        <div className="wcard center">
          <h2 className="h-serif">در پنج دقیقه، در مسیر نتیجه‌های ماندگار خواهی بود.</h2>
          <p>چند پرسش کوتاه تا کمکت کنیم زندگی‌ای بسازی که لایقش هستی.</p>
        </div>
        <div style={{ height: 18 }} />
        <Pill variant="dark" onClick={onNext}>ادامه</Pill>
      </div>
    </div>
  );
}

/* ---------- simple radio screens ---------- */
export function RadioScreen({ title, sub, items, value, onChange, onBack, onNext, p, skip }) {
  return (
    <IntakeShell
      back onBack={onBack} p={p}
      footer={
        skip ? <Pill variant="outline" onClick={onNext}>رد شدن</Pill>
          : <Pill disabled={!value} onClick={onNext}>ادامه</Pill>
      }
    >
      <H sub={sub}>{title}</H>
      <Options items={items} value={value} onChange={onChange} />
      <div style={{ height: 24 }} />
    </IntakeShell>
  );
}

/* ---------- age ---------- */
export function AgeScreen({ onBack, onNext, state, set }) {
  const ok = Number(state.age) > 0;
  return (
    <IntakeShell
      back onBack={onBack} p={0.7}
      footer={<Pill disabled={!ok} onClick={onNext}>ادامه</Pill>}
    >
      <H sub="کمک می‌کند پیشنهادها را دقیق‌تر کنیم.">چند سالته؟</H>
      <div className="field-under soft">
        <input
          value={state.age}
          onChange={(e) => set({ age: e.target.value.replace(/[^\d۰-۹]/g, '').slice(0, 3) })}
          placeholder="سن تو"
          inputMode="numeric"
        />
        <div className="rule" />
      </div>
      <Note>
        هر چه با رهراه در میان می‌گذاری خصوصی و بدون قضاوت می‌ماند — فقط برای این استفاده می‌شود که حمایت بیشتر شبیه خودت باشد.
      </Note>
    </IntakeShell>
  );
}

/* ---------- signup ---------- */
export function SignupScreen({ onBack, onNext, state, set }) {
  const ok = /.+@.+\..+/.test(state.email);
  return (
    <div className="screen auth" style={{ background: 'var(--light)' }}>
      <TopBar />
      <Stepper p={0.9} />
      <div style={{ padding: '18px 24px 0' }}>
        <h1 className="h-serif">پیشرفتت را ذخیره کن</h1>
        <p className="sub-sans">برای ادامه، ثبت‌نام کن.</p>
      </div>
      <div className="scroll" />
      <div className="btn-row gap" style={{ paddingTop: 0 }}>
        <button className="social" type="button" onClick={onNext}>
          <Google /> ادامه با Google
        </button>
        <button className="social" type="button" onClick={onNext}>
          <Apple /> ادامه با Apple
        </button>
        <div className="divider">یا با ایمیل</div>
        <input
          className="email"
          value={state.email}
          onChange={(e) => set({ email: e.target.value })}
          placeholder="name@example.com"
          dir="ltr"
          style={{ textAlign: 'end' }}
        />
        <Pill disabled={!ok} onClick={onNext}>ادامه</Pill>
        <p className="legal">
          با ساخت حساب، با <u>شرایط خدمات</u> موافقت می‌کنی و <u>سیاست حریم خصوصی</u> و <u>سیاست حریم خصوصی سلامت</u> را می‌پذیری.
        </p>
      </div>
    </div>
  );
}

/* ---------- stat screens ---------- */
export function StatScreen({ k, onBack, onNext }) {
  const s = STATS[k];
  return (
    <div className={`screen grad ${s.bg}`}>
      <TopBar dark back onBack={onBack} />
      <div className="stat" style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', paddingBottom: 40 }}>
        <div>
          <p className="big">{s.big}</p>
          <p className="cap">{s.cap}</p>
        </div>
      </div>
      <div className="content" style={{ padding: '0 24px calc(18px + env(safe-area-inset-bottom))' }}>
        <div className="quote">
          <p>{s.quote}</p>
          <div className="who">{s.who}</div>
        </div>
        <div style={{ height: 18 }} />
        <Pill variant="light" onClick={onNext}>ادامه</Pill>
      </div>
    </div>
  );
}

/* ---------- country ---------- */
export function CountryScreen({ onBack, onNext, state, set }) {
  const list = useMemo(
    () => COUNTRIES.filter((c) => c.includes(state.countryQ || '')),
    [state.countryQ],
  );
  return (
    <IntakeShell
      back onBack={onBack} p={0.95}
      footer={<Pill disabled={!state.country} onClick={onNext}>ادامه</Pill>}
    >
      <H>کشور محل اقامتت کجاست؟</H>
      <div className="searchbox">
        <input
          value={state.countryQ || ''}
          onChange={(e) => set({ countryQ: e.target.value })}
          placeholder="جست‌وجوی کشور…"
        />
      </div>
      <Options items={list} value={state.country} onChange={(v) => set({ country: v })} />
      <div style={{ height: 24 }} />
    </IntakeShell>
  );
}

/* ---------- free text ---------- */
export function FreeTextScreen({ title, sub, placeholder, hint, onBack, onNext, p, skip, value, onChange }) {
  const [rec, setRec] = useState(false);
  const words = value.trim() ? value.trim().split(/\s+/).length : 0;
  return (
    <IntakeShell
      back onBack={onBack} p={p}
      footer={
        skip ? <Pill variant="outline" onClick={onNext}>رد شدن</Pill>
          : <Pill disabled={!words} onClick={onNext}>ادامه</Pill>
      }
    >
      <H sub={sub}>{title}</H>
      <div className="ft">
        <div className="ta">
          <textarea value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} />
          <button className={`mic ${rec ? 'rec' : ''}`} type="button" onClick={() => setRec(!rec)} aria-label="ضبط صدا">
            <Mic />
          </button>
        </div>
        <div className="rule" style={{ margin: '0 24px' }} />
        <div className="meta" style={{ margin: '14px 24px 0' }}>{faNum(words)} واژه</div>
        <div className="wbar" style={{ margin: '12px 24px 0' }}>
          <i style={{ width: `${Math.min(33 + words * 4, 100)}%` }} />
        </div>
        <div className="hint" style={{ margin: '26px 24px 0' }}>{hint}</div>
      </div>
      <div style={{ height: 30 }} />
    </IntakeShell>
  );
}

/* ---------- reflection ---------- */
export function ReflectionScreen({ k, onBack, onNext }) {
  return (
    <IntakeShell
      back onBack={onBack}
      footer={<Pill onClick={onNext}>ادامه</Pill>}
    >
      <div className="refl" style={{ paddingTop: '18vh' }}>
        <span className="ic"><ChatCheck /></span>
        <p className="eyebrow gray">این چیزی است که می‌شنویم</p>
        <p className="body">{REFLECTIONS[k]}</p>
      </div>
    </IntakeShell>
  );
}

/* ---------- slider ---------- */
export function SliderScreen({ onBack, onNext, state, set }) {
  const ref = useRef(null);
  const v = state.change ?? 5;
  const setFrom = (clientX) => {
    const r = ref.current.getBoundingClientRect();
    const t = 1 - (clientX - r.left) / r.width; /* RTL: start = right */
    set({ change: Math.max(0, Math.min(10, Math.round(t * 10))) });
  };
  return (
    <IntakeShell back onBack={onBack} p={1.4} footer={<Pill onClick={onNext}>ادامه</Pill>}>
      <H>چقدر می‌خواهی چیزها عوض شوند یا فرق داشته باشند؟</H>
      <div className="sliderbox">
        <div
          className="slider-track"
          ref={ref}
          onPointerDown={(e) => { e.currentTarget.setPointerCapture(e.pointerId); setFrom(e.clientX); }}
          onPointerMove={(e) => { if (e.buttons) setFrom(e.clientX); }}
        >
          <div className="rail" />
          <div className="fill" style={{ width: `${v * 10}%` }} />
          {Array.from({ length: 11 }).map((_, i) => (
            <span key={i} className={`tick ${i <= v ? 'on' : ''}`} style={{ insetInlineStart: `${i * 10}%` }} />
          ))}
          <button className="thumb" style={{ insetInlineStart: `${v * 10}%`} } type="button" aria-label="میزان">
            <i />
          </button>
        </div>
        <div className="slider-labels">
          <span>نه</span><span>تا حدی</span><span>خیلی</span>
        </div>
      </div>
    </IntakeShell>
  );
}

/* ---------- scale (PHQ / GAD) ---------- */
export function ScaleScreen({ statements, onBack, onNext, p }) {
  const [answers, setAnswers] = useState([]);
  const cur = answers.length;
  const done = cur >= statements.length;
  return (
    <IntakeShell
      back onBack={onBack} p={p}
      footer={<Pill disabled={!done} onClick={onNext}>ادامه</Pill>}
    >
      <H>در دو هفتهٔ گذشته، هر یک از مشکلات زیر چقدر اذیتت کرده؟</H>
      <div className="scale">
        {statements.slice(0, cur).map((s, i) => (
          <div className="answered" key={s}>
            <span>{s}</span>
            <span style={{ flex: 'none' }}>{answers[i]}</span>
          </div>
        ))}
        {!done && (
          <>
            <div className="stmt">{statements[cur]}</div>
            <div className="opts">
              {SCALE_OPTS.map((o) => (
                <button
                  key={o} type="button"
                  className={`opt plain ${answers[cur] === o ? 'selected' : ''}`}
                  onClick={() => setAnswers((a) => [...a.slice(0, cur), o])}
                >
                  <span className="lbl">{o}</span>
                </button>
              ))}
            </div>
          </>
        )}
      </div>
      <div style={{ height: 24 }} />
    </IntakeShell>
  );
}

/* ---------- interstitial ---------- */
export function InterstitialScreen({ bg, title, sub, onBack, onNext }) {
  return (
    <div className={`screen grad ${bg}`}>
      <TopBar dark back onBack={onBack} />
      <div className="inter" style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
        <h1 className="h-serif" style={{ color: '#fff', lineHeight: 1.6, maxWidth: '82%', margin: '0 auto', fontWeight: 600 }}>{title}</h1>
        {sub && <p className="sub">{sub}</p>}
      </div>
      <div className="content">
        <Pill variant="light" onClick={onNext}>ادامه</Pill>
      </div>
    </div>
  );
}

export { CARRYING, EASED, LESS_HEAVY, MOVE_UP, YOURSELF, PERSON, FOUND, WHICH_AI, PHQ, GAD };
