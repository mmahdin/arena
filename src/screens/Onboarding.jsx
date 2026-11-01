import React, { useState } from 'react';
import { ONBOARD_STEPS } from '../data.js';
import { Pill, TopBar, faNum } from '../components/UI.jsx';
import { Mark } from '../components/Icons.jsx';

/* one continuous winding ribbon, 4 windows of 844 units */
const RIBBON =
  'M 300 -60 C 250 140, 120 260, 108 430 C 100 560, 130 660, 170 780 ' +
  'C 210 900, 250 1000, 262 1180 C 272 1330, 250 1480, 200 1620 ' +
  'C 170 1710, 190 1800, 220 1900 C 260 2020, 305 2060, 306 2150 ' +
  'C 307 2280, 220 2400, 150 2520 C 120 2580, 140 2660, 168 2760 ' +
  'C 190 2850, 105 2900, 100 2960 C 96 3050, 200 3180, 260 3300 C 290 3360, 300 3420, 300 3460';

export function OnboardingPath({ onNext }) {
  const [step, setStep] = useState(0);
  const s = ONBOARD_STEPS[step];
  const last = step === ONBOARD_STEPS.length - 1;

  const go = (n) => {
    if (last) { onNext(); return; }
    setStep(Math.min(n, ONBOARD_STEPS.length - 1));
  };

  return (
    <div className="screen ob">
      <div className="ribbon-wrap">
        <svg
          viewBox="0 0 390 3460"
          preserveAspectRatio="xMidYMin slice"
          style={{ width: '100%', height: 'auto', transform: `translateY(${-step * (844 / 3460) * 100}%)` }}
          aria-hidden
        >
          <defs>
            <linearGradient id="rib" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stopColor="#b3bd6d" />
              <stop offset="0.28" stopColor="#8ba055" />
              <stop offset="0.55" stopColor="#66803c" />
              <stop offset="0.8" stopColor="#48622c" />
              <stop offset="1" stopColor="#3a5226" />
            </linearGradient>
          </defs>
          <path d={RIBBON} stroke="url(#rib)" strokeWidth="86" fill="none" strokeLinecap="round" />
        </svg>
      </div>

      <TopBar dark />

      {/* marker dot on the path */}
      <div className="dot" style={{ insetInlineStart: `${s.dot.x}%`, top: `${s.dot.y}%` }}>
        <i />
      </div>

      {/* card */}
      <div
        className="card"
        key={s.id}
        style={
          s.side === 'start'
            ? { insetInlineStart: '6%', top: '45.5%' }
            : { insetInlineEnd: '6%', top: '45.5%' }
        }
      >
        <p className="eyebrow">{s.eyebrow}</p>
        <h2 className="h-serif">{s.title}</h2>
        <p>{s.body}</p>
      </div>

      <div className="next">
        <Pill variant="light" onClick={() => go(step + 1)}>بعدی</Pill>
      </div>
    </div>
  );
}

export function Splash({ onNext }) {
  return (
    <div className="screen grad bg-splash splash">
      <div className="mark">
        <Mark size={132} color="#f6f7f8" sw={7} />
      </div>
      <h1 className="word">رهراه</h1>
      <div className="content">
        <div className="wcard center">
          <h2 className="h-serif">مسیر تو از اینجا شروع می‌شود.</h2>
          <p>تنها در پنج دقیقه، اولین گام روشن به سوی آینده‌ای جذاب‌تر را خواهی داشت.</p>
        </div>
        <div style={{ height: 18 }} />
        <Pill variant="light" onClick={onNext}>شروع</Pill>
      </div>
    </div>
  );
}
