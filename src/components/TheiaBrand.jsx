import React from 'react';

/**
 * THEIA — «دایرهٔ باز» / The Open Circle
 *
 * The identity is drawn from a single idea: light does not arrive all at once.
 * It arrives as an opening. The mark is a circle left unfinished — a brushline
 * that lifts before it closes, holding one quiet point of light.
 *
 * The onboarding artworks are chapters of the same drawing. As the user is
 * seen and understood, the opening narrows (72° → 20°) until the circle closes
 * on the Ready screen. Home opens it again: the conversation continues.
 */

const TAU = Math.PI * 2;

/**
 * Generates a filled, pressure-modulated ring — a brush stroke, not a stroke.
 * Width builds from a blunt entry, breathes through the body and lifts to a
 * fine point at the tail. This is what keeps the symbol from reading as a
 * generic progress ring.
 */
const ringCache = new Map();

export function cachedBrushRing(options) {
  const key = JSON.stringify(options);
  if (!ringCache.has(key)) ringCache.set(key, brushRingPath(options));
  return ringCache.get(key);
}

export function brushRingPath({
  cx = 32,
  cy = 32,
  r = 20,
  gap = 66,
  gapCenter = 124,
  width = 4.4,
  entry = 1.5,
  exit = 0.4,
  steps = 96,
  build = 0.18,
  tail = 0.22,
} = {}) {
  const start = gapCenter + gap / 2;
  const sweep = 360 - gap;
  const outer = [];
  const inner = [];
  for (let index = 0; index <= steps; index += 1) {
    const t = index / steps;
    const angle = ((start + t * sweep) * Math.PI) / 180;
    let pressure;
    if (t < build) {
      pressure = entry + (width - entry) * Math.sin((t / build) * (Math.PI / 2));
    } else if (t > 1 - tail) {
      pressure = width + (exit - width) * Math.pow((t - (1 - tail)) / tail, 1.3);
    } else {
      pressure = width * (0.9 + 0.1 * Math.sin(((t - build) / (1 - build - tail)) * Math.PI));
    }
    const half = Math.max(0.05, pressure) / 2;
    outer.push([cx + (r + half) * Math.cos(angle), cy + (r + half) * Math.sin(angle)]);
    inner.push([cx + (r - half) * Math.cos(angle), cy + (r - half) * Math.sin(angle)]);
  }
  const points = [...outer, ...inner.reverse()];
  return `${points.map(([x, y], index) => `${index ? 'L' : 'M'}${x.toFixed(2)} ${y.toFixed(2)}`).join('')}Z`;
}

/** The five chapters of the onboarding narrative, with the opening angle of
 * each circle. The values are the story, not decoration: the circle closes as
 * trust is built and re-opens when the conversation begins. */
export const THEIA_CHAPTERS = Object.freeze({
  welcome: { chapter: 'شب', gap: 72, gapCenter: 130 },
  privacy: { chapter: 'آستانه', gap: 0, gapCenter: 130 },
  simple: { chapter: 'نخستین نور', gap: 54, gapCenter: 128 },
  deeper: { chapter: 'بازتاب', gap: 44, gapCenter: 124 },
  know: { chapter: 'سپیده', gap: 32, gapCenter: 120 },
  almost: { chapter: 'نامه', gap: 26, gapCenter: 116 },
  checkins: { chapter: 'بازگشت', gap: 40, gapCenter: 110 },
  ready: { chapter: 'روشنایی', gap: 0, gapCenter: 116 },
  home: { chapter: 'گفت‌وگو', gap: 66, gapCenter: 124 },
});

export const THEIA_COLORS = Object.freeze({
  night: '#12110d',
  ink: '#23201b',
  paper: '#f3f2ee',
  linen: '#e9e3d8',
  ivory: '#f3efe6',
  dawn: '#f6cf98',
  ember: '#e3ae72',
  gold: '#c9975e',
  dusk: '#d5d9f7',
  haze: '#b9bedc',
});

const uidFor = (raw) => String(raw).replace(/[^a-zA-Z0-9_-]/g, '');

function LightDot({ cx, cy, r = 3, color = THEIA_COLORS.gold, glow = true, glowId = '', className = '' }) {
  return (
    <g className={className}>
      {glow && glowId && <circle cx={cx} cy={cy} r={r * 4.6} fill={`url(#${glowId})`} opacity="0.85" />}
      <circle cx={cx} cy={cy} r={r} fill={color} />
    </g>
  );
}

/**
 * The core mark. `luminous` is for ink-dark surfaces (Home, drawer, letters);
 * the default ink-and-gold version is for paper surfaces.
 */
export function TheiaMark({ size = 24, luminous = false, className = '', seal = false }) {
  const uid = uidFor(React.useId());
  const ring = luminous ? THEIA_COLORS.ivory : 'currentColor';
  const dot = luminous ? THEIA_COLORS.dawn : THEIA_COLORS.gold;
  const path = cachedBrushRing({ r: seal ? 19.5 : 20, gap: seal ? 74 : 66, width: seal ? 5.6 : 4.4, entry: seal ? 1.8 : 1.5 });
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      fill="none"
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      aria-hidden="true"
      focusable="false"
    >
      {luminous && (
        <defs>
          <radialGradient id={`${uid}-halo`} cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor={THEIA_COLORS.dawn} stopOpacity=".34" />
            <stop offset="60%" stopColor={THEIA_COLORS.dawn} stopOpacity=".06" />
            <stop offset="100%" stopColor={THEIA_COLORS.dawn} stopOpacity="0" />
          </radialGradient>
        </defs>
      )}
      {luminous && <circle cx="35" cy="30" r="17" fill={`url(#${uid}-halo)`} />}
      <path d={path} fill={ring} />
      <circle cx="34.6" cy="30.2" r={seal ? 2.5 : 3} fill={dot} />
    </svg>
  );
}

/** Tiny wax-seal glint: the pearl with its ring, legible at 10–14px. */
export function TheiaSealIcon({ size = 11, className = '' }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} aria-hidden="true" focusable="false">
      <circle cx="8" cy="8" r="6" stroke="currentColor" strokeWidth="1.15" opacity=".75" />
      <circle cx="8" cy="8" r="2.2" fill="currentColor" />
    </svg>
  );
}

/* ------------------------------------------------------------------ *
 * Onboarding chapters
 * Each scene is authored, not imported: one continuous drawing of light.
 * ------------------------------------------------------------------ */

function SceneDefs({ uid, mode }) {
  const dark = mode === 'dark';
  return (
    <defs>
      <radialGradient id={`${uid}-light`} cx="50%" cy="50%" r="50%">
        <stop offset="0%" stopColor={THEIA_COLORS.dawn} stopOpacity={dark ? '.9' : '.55'} />
        <stop offset="46%" stopColor={THEIA_COLORS.ember} stopOpacity={dark ? '.28' : '.16'} />
        <stop offset="100%" stopColor={THEIA_COLORS.gold} stopOpacity="0" />
      </radialGradient>
      <radialGradient id={`${uid}-disc`} cx="50%" cy="46%" r="54%">
        <stop offset="0%" stopColor="#fbe4bd" stopOpacity=".95" />
        <stop offset="58%" stopColor={THEIA_COLORS.dawn} stopOpacity=".55" />
        <stop offset="100%" stopColor={THEIA_COLORS.ember} stopOpacity=".05" />
      </radialGradient>
      <linearGradient id={`${uid}-sea`} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stopColor={THEIA_COLORS.dusk} stopOpacity={dark ? '.16' : '.14'} />
        <stop offset="55%" stopColor={THEIA_COLORS.dusk} stopOpacity=".04" />
        <stop offset="100%" stopColor={THEIA_COLORS.night} stopOpacity="0" />
      </linearGradient>
      <linearGradient id={`${uid}-dawn`} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stopColor={THEIA_COLORS.dawn} stopOpacity=".34" />
        <stop offset="60%" stopColor={THEIA_COLORS.ember} stopOpacity=".12" />
        <stop offset="100%" stopColor={THEIA_COLORS.ember} stopOpacity="0" />
      </linearGradient>
      <linearGradient id={`${uid}-page`} x1="0" y1="0" x2="1" y2="1">
        <stop offset="0%" stopColor="#f7ead2" stopOpacity=".98" />
        <stop offset="100%" stopColor="#d9bc93" stopOpacity=".9" />
      </linearGradient>
      <radialGradient id={`${uid}-paperwash`} cx="50%" cy="50%" r="50%">
        <stop offset="0%" stopColor={THEIA_COLORS.ember} stopOpacity=".22" />
        <stop offset="55%" stopColor={THEIA_COLORS.dawn} stopOpacity=".1" />
        <stop offset="100%" stopColor={THEIA_COLORS.paper} stopOpacity="0" />
      </radialGradient>
      <linearGradient id={`${uid}-water`} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stopColor={THEIA_COLORS.night} stopOpacity=".22" />
        <stop offset="34%" stopColor={THEIA_COLORS.night} stopOpacity=".82" />
        <stop offset="100%" stopColor={THEIA_COLORS.night} stopOpacity=".98" />
      </linearGradient>
      <linearGradient id={`${uid}-shimmer`} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stopColor={THEIA_COLORS.dawn} stopOpacity=".3" />
        <stop offset="55%" stopColor={THEIA_COLORS.ember} stopOpacity=".08" />
        <stop offset="100%" stopColor={THEIA_COLORS.ember} stopOpacity="0" />
      </linearGradient>
      <linearGradient id={`${uid}-cool`} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stopColor={THEIA_COLORS.dusk} stopOpacity={dark ? '.1' : '.07'} />
        <stop offset="62%" stopColor={THEIA_COLORS.dusk} stopOpacity="0" />
      </linearGradient>
      <radialGradient id={`${uid}-horizonglow`} cx="50%" cy="50%" r="50%">
        <stop offset="0%" stopColor={THEIA_COLORS.dawn} stopOpacity=".5" />
        <stop offset="52%" stopColor={THEIA_COLORS.ember} stopOpacity=".13" />
        <stop offset="100%" stopColor={THEIA_COLORS.ember} stopOpacity="0" />
      </radialGradient>
      <linearGradient id={`${uid}-hairline`} x1="0" y1="0" x2="1" y2="0">
        <stop offset="0%" stopColor={dark ? THEIA_COLORS.ivory : THEIA_COLORS.ink} stopOpacity="0" />
        <stop offset="22%" stopColor={dark ? THEIA_COLORS.ivory : THEIA_COLORS.ink} stopOpacity={dark ? '.34' : '.24'} />
        <stop offset="50%" stopColor={dark ? THEIA_COLORS.dawn : THEIA_COLORS.gold} stopOpacity=".46" />
        <stop offset="78%" stopColor={dark ? THEIA_COLORS.ivory : THEIA_COLORS.ink} stopOpacity={dark ? '.34' : '.24'} />
        <stop offset="100%" stopColor={dark ? THEIA_COLORS.ivory : THEIA_COLORS.ink} stopOpacity="0" />
      </linearGradient>
    </defs>
  );
}

function Motes({ uid, mode, points }) {
  return (
    <g fill={mode === 'dark' ? THEIA_COLORS.ivory : THEIA_COLORS.ink} aria-hidden="true">
      {points.map(([x, y, r, o], index) => (
        <circle key={index} className="theia-mote" style={{ '--i': index }} cx={x} cy={y} r={r} opacity={o} />
      ))}
    </g>
  );
}

function OrbitTrace({ cx, cy, r, from = 0, to = 220, color, opacity = 0.16, width = 1 }) {
  const centerX = Number(cx);
  const centerY = Number(cy);
  const radius = Number(r);
  const start = (from * Math.PI) / 180;
  const end = (to * Math.PI) / 180;
  const large = Math.abs(to - from) > 180 ? 1 : 0;
  const x1 = centerX + radius * Math.cos(start);
  const y1 = centerY + radius * Math.sin(start);
  const x2 = centerX + radius * Math.cos(end);
  const y2 = centerY + radius * Math.sin(end);
  return (
    <path
      className="theia-orbit-trace"
      d={`M${x1.toFixed(2)} ${y1.toFixed(2)}A${radius} ${radius} 0 ${large} 1 ${x2.toFixed(2)} ${y2.toFixed(2)}`}
      stroke={color}
      strokeWidth={width}
      strokeOpacity={opacity}
      strokeLinecap="round"
      fill="none"
    />
  );
}

function Horizon({ uid, y, mode, inset = 24, glow = 0.4 }) {
  return (
    <g>
      <ellipse cx="180" cy={Number(y)} rx="196" ry="34" fill={`url(#${uid}-horizonglow)`} opacity={glow} />
      <path d={`M${inset} ${y}H${360 - inset}`} stroke={`url(#${uid}-hairline)`} strokeWidth="1" fill="none" />
    </g>
  );
}

function WelcomeScene({ uid }) {
  const chapter = THEIA_CHAPTERS.welcome;
  const ring = brushRingPath({ cx: 184, cy: 236, r: 118, gap: chapter.gap, gapCenter: chapter.gapCenter, width: 6.4, entry: 1.9, exit: 0.2, steps: 200 });
  return (
    <svg className="theia-scene" viewBox="0 0 360 480" preserveAspectRatio="xMidYMid meet" fill="none" aria-hidden="true" focusable="false">
      <SceneDefs uid={uid} mode="light" />
      <rect width="360" height="480" fill={`url(#${uid}-cool)`} />
      <circle cx="176" cy="258" r="204" fill={`url(#${uid}-paperwash)`} opacity="0.62" />
      <circle cx="146" cy="286" r="58" fill={`url(#${uid}-disc)`} opacity="0.24" />
      <path d={ring} fill={THEIA_COLORS.ink} fillOpacity="0.88" className="theia-stroke" />
      <Horizon uid={uid} y="374" mode="light" glow={0.36} />
      <LightDot cx="146" cy="286" r="4.8" color={THEIA_COLORS.gold} glowId={`${uid}-light`} />

      <Motes uid={uid} mode="light" points={[[94, 150, 1.7, 0.24], [274, 174, 2, 0.18], [256, 312, 1.5, 0.2], [112, 330, 2.2, 0.13]]} />
    </svg>
  );
}

function SimpleScene({ uid }) {
  const chapter = THEIA_CHAPTERS.simple;
  const ring = brushRingPath({ cx: 180, cy: 222, r: 104, gap: chapter.gap, gapCenter: chapter.gapCenter, width: 5.8, entry: 1.5, exit: 0.15, steps: 200 });
  return (
    <svg className="theia-scene" viewBox="0 0 360 480" preserveAspectRatio="xMidYMid meet" fill="none" aria-hidden="true" focusable="false">
      <SceneDefs uid={uid} mode="dark" />
      <rect width="360" height="480" fill={`url(#${uid}-cool)`} />
      <circle cx="180" cy="232" r="196" fill={`url(#${uid}-light)`} opacity="0.4" />
      <path d={ring} fill={THEIA_COLORS.ivory} fillOpacity="0.93" className="theia-stroke" />
      <Horizon uid={uid} y="366" mode="dark" glow={0.4} />
      <LightDot cx="180" cy="222" r="5" color={THEIA_COLORS.dawn} glowId={`${uid}-light`} />
      <Motes uid={uid} mode="dark" points={[[82, 136, 1.6, 0.34], [290, 160, 1.9, 0.28], [304, 284, 1.4, 0.22], [72, 302, 2.1, 0.18], [216, 100, 1.3, 0.26]]} />
    </svg>
  );
}

function DeeperScene({ uid }) {
  const chapter = THEIA_CHAPTERS.deeper;
  const ring = brushRingPath({ cx: 180, cy: 206, r: 100, gap: chapter.gap, gapCenter: chapter.gapCenter, width: 5.4, entry: 1.4, exit: 0.2, steps: 200 });
  return (
    <svg className="theia-scene" viewBox="0 0 360 480" preserveAspectRatio="xMidYMid meet" fill="none" aria-hidden="true" focusable="false">
      <SceneDefs uid={uid} mode="dark" />
      <defs>
        <linearGradient id={`${uid}-abovefade`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#fff" />
          <stop offset="58%" stopColor="#fff" />
          <stop offset="65%" stopColor="#000" />
          <stop offset="100%" stopColor="#000" />
        </linearGradient>
        <linearGradient id={`${uid}-belowfade`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#000" />
          <stop offset="60.5%" stopColor="#000" />
          <stop offset="62.5%" stopColor="#fff" stopOpacity=".5" />
          <stop offset="67%" stopColor="#fff" stopOpacity=".2" />
          <stop offset="74%" stopColor="#000" />
          <stop offset="100%" stopColor="#000" />
        </linearGradient>
        <filter id={`${uid}-waterblur`} x="-20%" y="-20%" width="140%" height="140%">
          <feGaussianBlur stdDeviation="0.9" />
        </filter>
        <mask id={`${uid}-above`}><rect width="360" height="480" fill={`url(#${uid}-abovefade)`} /></mask>
        <mask id={`${uid}-below`}><rect width="360" height="480" fill={`url(#${uid}-belowfade)`} /></mask>
      </defs>
      <rect width="360" height="480" fill={`url(#${uid}-cool)`} />
      <circle cx="180" cy="196" r="176" fill={`url(#${uid}-light)`} opacity="0.24" />
      <g mask={`url(#${uid}-above)`}>
        <path d={ring} fill={THEIA_COLORS.ivory} fillOpacity="0.95" className="theia-stroke" />
        <circle cx="180" cy="206" r="4.8" fill={THEIA_COLORS.dawn} />
        <circle cx="180" cy="206" r="22" fill={`url(#${uid}-light)`} opacity="0.8" />
      </g>
      <g mask={`url(#${uid}-below)`} filter={`url(#${uid}-waterblur)`}>
        <g transform="translate(0 499.8) scale(1 -0.7)" opacity="0.2">
          <path d={ring} fill={THEIA_COLORS.ivory} />
          <circle cx="180" cy="206" r="4.8" fill={THEIA_COLORS.dawn} opacity="0.6" />
        </g>
      </g>
      <ellipse cx="180" cy="300" rx="96" ry="20" fill={`url(#${uid}-horizonglow)`} opacity="0.42" />
      <Horizon uid={uid} y="294" mode="dark" glow={0.26} />
      <path d="M48 294H312" stroke={THEIA_COLORS.dusk} strokeOpacity="0.24" strokeWidth="1" />
      <path d="M52 294H308" stroke={THEIA_COLORS.dusk} strokeOpacity="0.16" strokeWidth="1" />
      <g stroke={THEIA_COLORS.dawn} strokeLinecap="round">
        <path d="M76 312H284" strokeWidth="1" strokeOpacity="0.26" />
        <path d="M116 330H244" strokeWidth="0.8" strokeOpacity="0.15" />
        <path d="M150 350H210" strokeWidth="0.7" strokeOpacity="0.08" />
      </g>
      <Motes uid={uid} mode="dark" points={[[98, 142, 1.5, 0.3], [268, 178, 1.8, 0.22], [122, 258, 1.3, 0.24], [246, 262, 1.4, 0.18], [176, 420, 1.2, 0.1]]} />
    </svg>
  );
}

function KnowScene({ uid }) {
  const chapter = THEIA_CHAPTERS.know;
  const ring = brushRingPath({ cx: 180, cy: 218, r: 104, gap: chapter.gap, gapCenter: chapter.gapCenter, width: 5.8, entry: 1.5, exit: 0.15, steps: 200 });
  return (
    <svg className="theia-scene" viewBox="0 0 360 480" preserveAspectRatio="xMidYMid meet" fill="none" aria-hidden="true" focusable="false">
      <SceneDefs uid={uid} mode="dark" />
      <rect width="360" height="480" fill={`url(#${uid}-cool)`} />
      <ellipse cx="176" cy="318" rx="250" ry="164" fill={`url(#${uid}-horizonglow)`} opacity="0.72" />
      <g opacity="0.6">
        <rect x="0" y="250" width="360" height="22" fill={`url(#${uid}-horizonglow)`} opacity="0.25" />
        <rect x="0" y="280" width="360" height="16" fill={`url(#${uid}-horizonglow)`} opacity="0.32" />
        <rect x="0" y="300" width="360" height="10" fill={`url(#${uid}-horizonglow)`} opacity="0.4" />
      </g>
      <path d={ring} fill={THEIA_COLORS.ivory} fillOpacity="0.96" className="theia-stroke" />
      <Horizon uid={uid} y="314" mode="dark" glow={0.5} />
      <path d="M24 314H336" stroke={THEIA_COLORS.dawn} strokeOpacity="0.42" strokeWidth="1.1" />
      <g stroke={THEIA_COLORS.dawn} strokeLinecap="round">
        <path d="M70 336H150" strokeWidth="1" strokeOpacity="0.5" />
        <path d="M96 354H206" strokeWidth="0.9" strokeOpacity="0.36" />
        <path d="M132 372H248" strokeWidth="0.8" strokeOpacity="0.24" />
        <path d="M168 392H286" strokeWidth="0.7" strokeOpacity="0.14" />
      </g>
      <LightDot cx="180" cy="218" r="5.2" color={THEIA_COLORS.dawn} glowId={`${uid}-light`} />
      <Motes uid={uid} mode="dark" points={[[90, 146, 1.9, 0.4], [282, 132, 1.5, 0.3], [298, 244, 1.7, 0.24], [66, 270, 1.4, 0.2], [224, 104, 1.3, 0.3]]} />
    </svg>
  );
}

function AlmostScene({ uid }) {
  const chapter = THEIA_CHAPTERS.almost;
  const ring = brushRingPath({ cx: 180, cy: 236, r: 138, gap: chapter.gap, gapCenter: chapter.gapCenter, width: 6.2, entry: 1.8, exit: 0.3, steps: 220 });
  return (
    <svg className="theia-scene" viewBox="0 0 360 480" preserveAspectRatio="xMidYMid meet" fill="none" aria-hidden="true" focusable="false">
      <SceneDefs uid={uid} mode="dark" />
      <rect width="360" height="480" fill={`url(#${uid}-cool)`} />
      <circle cx="168" cy="248" r="196" fill={`url(#${uid}-light)`} opacity="0.3" />
      <path d={ring} fill={THEIA_COLORS.ivory} fillOpacity="0.96" className="theia-stroke" />
      <LightDot cx="168" cy="248" r="6.2" color={THEIA_COLORS.dawn} glowId={`${uid}-light`} />
      <Horizon uid={uid} y="418" mode="dark" glow={0.26} />
      <Motes uid={uid} mode="dark" points={[[82, 126, 1.6, 0.3], [286, 158, 1.9, 0.26], [66, 300, 1.4, 0.2], [300, 296, 1.5, 0.2], [126, 402, 1.2, 0.14]]} />
    </svg>
  );
}

function waxRim(cx, cy, r, steps = 120) {
  const points = [];
  for (let index = 0; index <= steps; index += 1) {
    const angle = (index / steps) * TAU;
    // Hand-pressed edge: two low, incommensurate wobbles. Never a perfect disc.
    const radius = r + Math.sin(angle * 5 + 0.7) * 0.9 + Math.sin(angle * 9 + 2.1) * 0.45;
    points.push(`${(cx + radius * Math.cos(angle)).toFixed(2)} ${(cy + radius * Math.sin(angle)).toFixed(2)}`);
  }
  return `M${points.join('L')}Z`;
}

function PrivacyScene({ uid }) {
  // Privacy is a seal, not a cage: dark wax pressed once, the circle stamped
  // in light. What is said here stays here.
  const rim = waxRim(80, 80, 47);
  return (
    <svg className="theia-scene" viewBox="0 0 160 185" preserveAspectRatio="xMidYMid meet" fill="none" aria-hidden="true" focusable="false">
      <defs>
        <radialGradient id={`${uid}-wax`} cx="36%" cy="30%" r="86%">
          <stop offset="0%" stopColor="#524434" />
          <stop offset="55%" stopColor="#352b20" />
          <stop offset="100%" stopColor="#221b14" />
        </radialGradient>
        <radialGradient id={`${uid}-waxedge`} cx="50%" cy="46%" r="54%">
          <stop offset="0%" stopColor={THEIA_COLORS.ink} stopOpacity="0" />
          <stop offset="68%" stopColor={THEIA_COLORS.ink} stopOpacity="0" />
          <stop offset="100%" stopColor="#000" stopOpacity=".34" />
        </radialGradient>
        <filter id={`${uid}-soft`} x="-40%" y="-40%" width="180%" height="180%">
          <feGaussianBlur stdDeviation="5" />
        </filter>
      </defs>
      <circle cx="180" cy="70" r="115" fill={`url(#${uid}-paperwash)`} opacity="0.7" />
      <ellipse cx="80" cy="137" rx="40" ry="6" fill={THEIA_COLORS.ink} opacity="0.16" filter={`url(#${uid}-soft)`} />
      <path d={rim} fill={`url(#${uid}-wax)`} />
      <path d={rim} fill={`url(#${uid}-waxedge)`} />
      <path d={rim} stroke="#6f5940" strokeOpacity="0.55" strokeWidth="0.7" />
      <path d="M42 58A44 44 0 0 1 74 34" stroke="#a98d68" strokeOpacity="0.5" strokeWidth="1.1" strokeLinecap="round" />
      <circle cx="80" cy="80" r="31" fill="none" stroke="#000" strokeOpacity="0.42" strokeWidth="4.4" transform="translate(0.7 1.6)" />
      <circle cx="80" cy="80" r="31" fill="none" stroke="#e9c99b" strokeOpacity="0.94" strokeWidth="4.4" />
      <LightDot cx="80" cy="80" r="3" color={THEIA_COLORS.dawn} glowId={`${uid}-light`} />
      <Horizon uid={uid} y="160" mode="light" glow={0.14} />
    </svg>
  );
}

function CheckinsScene({ uid }) {
  // A week is not a progress bar. It is a thread of small lights along a
  // horizon; one of them is lit today. No enclosing cycle, no completion.
  const marks = [
    { x: 46, r: 3, o: 0.4 },
    { x: 92, r: 3.5, o: 0.52 },
    { x: 140, r: 2.8, o: 0.36 },
    { x: 190, r: 5.4, o: 1, lit: true },
    { x: 238, r: 3.2, o: 0.46 },
    { x: 286, r: 3.7, o: 0.55 },
    { x: 318, r: 2.6, o: 0.32 },
  ];
  const arcY = (x) => {
    const t = Math.max(0, Math.min(1, (x - 36) / 288));
    return (1 - t) * (1 - t) * 206 + 2 * (1 - t) * t * 150 + t * t * 200;
  };
  return (
    <svg className="theia-scene" viewBox="0 0 360 260" preserveAspectRatio="xMidYMid meet" fill="none" aria-hidden="true" focusable="false">
      <defs>
        <radialGradient id={`${uid}-cardglow`} cx="52%" cy="46%" r="60%">
          <stop offset="0%" stopColor={THEIA_COLORS.dawn} stopOpacity=".2" />
          <stop offset="46%" stopColor={THEIA_COLORS.ember} stopOpacity=".07" />
          <stop offset="100%" stopColor={THEIA_COLORS.night} stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`${uid}-pearl`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor={THEIA_COLORS.dawn} stopOpacity=".55" />
          <stop offset="55%" stopColor={THEIA_COLORS.dawn} stopOpacity=".14" />
          <stop offset="100%" stopColor={THEIA_COLORS.dawn} stopOpacity="0" />
        </radialGradient>
        <linearGradient id={`${uid}-thread`} x1="0" y1="0" x2="1" y2="0">
          <stop offset="0%" stopColor={THEIA_COLORS.ivory} stopOpacity="0" />
          <stop offset="14%" stopColor={THEIA_COLORS.ivory} stopOpacity=".24" />
          <stop offset="86%" stopColor={THEIA_COLORS.ivory} stopOpacity=".24" />
          <stop offset="100%" stopColor={THEIA_COLORS.ivory} stopOpacity="0" />
        </linearGradient>
      </defs>
      <ellipse cx="180" cy="178" rx="210" ry="132" fill={`url(#${uid}-cardglow)`} />
      <path d="M36 206Q180 150 324 200" stroke={`url(#${uid}-thread)`} strokeWidth="1.2" fill="none" />
      {marks.map((mark) => {
        const y = arcY(mark.x);
        return mark.lit ? (
          <g key={mark.x}>
            <ellipse cx={mark.x} cy={y + 10} rx="44" ry="12" fill={`url(#${uid}-pearl)`} opacity="0.5" />
            <circle cx={mark.x} cy={y} r="34" fill={`url(#${uid}-pearl)`} />
            <circle cx={mark.x} cy={y} r={mark.r} fill={THEIA_COLORS.dawn} />
            <circle cx={mark.x} cy={y} r="11" stroke={THEIA_COLORS.dawn} strokeOpacity=".2" strokeWidth="0.8" />
          </g>
        ) : (
          <circle key={mark.x} cx={mark.x} cy={y} r={mark.r} fill={THEIA_COLORS.ivory} opacity={mark.o} />
        );
      })}
      <Horizon uid={uid} y="232" mode="dark" glow={0.3} />
      <Motes uid={uid} mode="dark" points={[[64, 84, 1.4, 0.22], [268, 70, 1.7, 0.24], [312, 118, 1.2, 0.14]]} />
    </svg>
  );
}

function ReadyScene({ uid }) {
  // Daybreak is quiet: the brushline has resolved into a perfect circle.
  // No rays, no sun disc — light is held, not announced.
  return (
    <svg className="theia-scene" viewBox="0 0 360 480" preserveAspectRatio="xMidYMid meet" fill="none" aria-hidden="true" focusable="false">
      <SceneDefs uid={uid} mode="light" />
      <rect width="360" height="480" fill={`url(#${uid}-cool)`} />
      <circle cx="180" cy="238" r="206" fill={`url(#${uid}-paperwash)`} />
      <circle cx="180" cy="230" r="96" fill={`url(#${uid}-disc)`} opacity="0.4" />
      <circle cx="180" cy="230" r="102" fill="none" stroke={THEIA_COLORS.ink} strokeOpacity="0.78" strokeWidth="1.8" />
      <Horizon uid={uid} y="376" mode="light" glow={0.3} />
      <LightDot cx="180" cy="230" r="4.2" color={THEIA_COLORS.dawn} glowId={`${uid}-light`} />
      <Motes uid={uid} mode="light" points={[[88, 142, 1.6, 0.24], [280, 170, 2, 0.2], [268, 310, 1.4, 0.18], [104, 304, 1.8, 0.16]]} />
    </svg>
  );
}

const SCENES = {
  welcome: WelcomeScene,
  simple: SimpleScene,
  deeper: DeeperScene,
  know: KnowScene,
  almost: AlmostScene,
  privacy: PrivacyScene,
  checkins: CheckinsScene,
  ready: ReadyScene,
};

/**
 * A chapter of the onboarding story. Always decorative: the surrounding
 * component supplies the accessible description in Persian.
 */
export function TheiaScene({ variant = 'welcome', className = '' }) {
  const uid = uidFor(React.useId());
  const Scene = SCENES[variant] || WelcomeScene;
  return <span className={`theia-art ${className}`.trim()}><Scene uid={uid} /></span>;
}

/**
 * Home's living emblem: the open circle, two slow orbits and a single light.
 * Replaces the former figure; keeps the 240px composition and breathing motion.
 */
export function TheiaHomeArt() {
  const uid = uidFor(React.useId());
  const ring = cachedBrushRing({ cx: 60, cy: 60, r: 39, gap: THEIA_CHAPTERS.home.gap, gapCenter: THEIA_CHAPTERS.home.gapCenter, width: 2.7, entry: 0.9, exit: 0.2, steps: 160 });
  return (
    <span className="home-conversation-art" aria-hidden="true">
      <svg className="home-conversation-orbit" viewBox="0 0 240 240" fill="none" focusable="false">
        <defs>
          <radialGradient id={`${uid}-homehalo`} cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor={THEIA_COLORS.dawn} stopOpacity=".26" />
            <stop offset="58%" stopColor={THEIA_COLORS.dusk} stopOpacity=".06" />
            <stop offset="100%" stopColor={THEIA_COLORS.night} stopOpacity="0" />
          </radialGradient>
        </defs>
        <circle cx="120" cy="120" r="96" fill={`url(#${uid}-homehalo)`} />
        <path className="home-orbit-trace home-orbit-trace--first" d="M188 48C158 20 114 22 82 39C40 62 22 106 36 149C46 183 74 206 108 212" stroke="#7a7870" strokeWidth="1.2" strokeLinecap="round" />
        <path className="home-orbit-trace home-orbit-trace--second" d="M57 178C82 206 126 220 162 203C199 186 221 152 210 110C204 89 194 76 183 66" stroke="#b0aaa0" strokeWidth="1.45" strokeLinecap="round" />
        <path className="home-orbit-accent" d="M58 78C76 56 101 46 126 47" stroke={THEIA_COLORS.dawn} strokeWidth="1.6" strokeLinecap="round" opacity="0.55" />
        <circle className="home-orbit-point home-orbit-point--sun" cx="111" cy="211" r="3" fill={THEIA_COLORS.dawn} />
        <circle className="home-orbit-point home-orbit-point--moon" cx="183" cy="66" r="2.1" fill={THEIA_COLORS.dusk} opacity="0.8" />
      </svg>
      <svg className="home-conversation-mark theia-home-emblem" width="146" height="146" viewBox="0 0 120 120" fill="none" focusable="false">
        <circle cx="60" cy="60" r="54" fill={`url(#${uid}-homehalo)`} opacity="0.8" />
        <path d={ring} fill={THEIA_COLORS.ivory} />
        <circle cx="60" cy="60" r="27" fill={`url(#${uid}-homehalo)`} />
        <circle cx="60" cy="60" r="4.2" fill={THEIA_COLORS.dawn} />
      </svg>
    </span>
  );
}

/** Quiet empty-inbox glyph: the open circle becomes a letter seal. */
export function TheiaLetterEmpty({ size = 44 }) {
  const ring = cachedBrushRing({ cx: 24, cy: 24, r: 15, gap: 68, gapCenter: 128, width: 2.2, entry: 0.8, exit: 0.2, steps: 90 });
  return (
    <svg width={size} height={size} viewBox="0 0 48 48" fill="none" className="theia-letter-empty" aria-hidden="true" focusable="false">
      <path d={ring} fill="currentColor" opacity="0.9" />
      <circle cx="26" cy="22" r="2.4" fill={THEIA_COLORS.dawn} />
      <path d="M8 34.5H40" stroke="currentColor" strokeWidth="0.9" strokeOpacity="0.4" strokeLinecap="round" />
    </svg>
  );
}
