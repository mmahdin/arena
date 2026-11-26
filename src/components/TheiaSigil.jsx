import React from 'react';

/**
 * Theia's 2026 identity is intentionally faceless.
 *
 * In the myth Theia (Θεία, "the far-shining") is the Titaness of sight and of
 * the light of the sky. She was never given a cult image, a temple or a statue:
 * what survives of her is light itself. Her children are Helios, Selene and Eos
 * — sun, moon and dawn. The brand therefore shows light, never a body.
 *
 * The onboarding scenes are one place seen at different hours: a wide sky, a
 * low ridge, still water and a cypress headland. Nothing moves except the
 * light and the water. Each chapter changes only the hour, so the story reads
 * night → first light → sight → dawn → three lights → a closed dawn. The small
 * mark keeps the open ring (the product promise: the app does not close around
 * you); the last screen is the only place it closes.
 *
 * Geometry is hand-authored SVG so the mark stays sharp at 11px and can carry
 * subtle motion (honouring prefers-reduced-motion) where a raster cannot.
 */

const SIGIL_STAR = (cx, cy, radius) => {
  const k = radius * 0.085;
  const r = radius;
  return [
    `M${cx} ${cy - r}`,
    `Q${cx + k} ${cy - k} ${cx + r} ${cy}`,
    `Q${cx + k} ${cy + k} ${cx} ${cy + r}`,
    `Q${cx - k} ${cy + k} ${cx - r} ${cy}`,
    `Q${cx - k} ${cy - k} ${cx} ${cy - r}`,
    'Z',
  ].join(' ');
};

export function TheiaSigil({ size = 32, luminous = false, complete = false, className = '', title }) {
  const id = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const gold = `sigil-gold-${id}`;
  const glow = `sigil-glow-${id}`;
  // The ring stops short of closing at the bottom. On the final screen the seam
  // meets (complete) — the only place in the product where the circle closes.
  const openRing = 'M19.4 54.7A26 26 0 1 1 44.6 54.7';
  return (
    <svg
      className={`theia-sigil ${luminous ? 'theia-sigil--luminous' : ''} ${className}`.trim()}
      width={size}
      height={size}
      viewBox="0 0 64 64"
      fill="none"
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : 'true'}
      aria-label={title}
      focusable="false"
    >
      <defs>
        <linearGradient id={gold} x1="8" y1="8" x2="56" y2="58" gradientUnits="userSpaceOnUse">
          <stop stopColor="#F7E7C6" />
          <stop offset=".46" stopColor="#E3C48C" />
          <stop offset="1" stopColor="#B98A4E" />
        </linearGradient>
        <filter id={glow} x="-40%" y="-40%" width="180%" height="180%">
          <feGaussianBlur stdDeviation="1.5" result="b" />
          <feMerge>
            <feMergeNode in="b" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>
      <g filter={luminous ? `url(#${glow})` : undefined}>
        <path
          d={complete ? 'M32 6A26 26 0 1 1 31.99 6' : openRing}
          stroke={`url(#${gold})`}
          strokeWidth={complete ? 2.6 : 2.2}
          strokeLinecap="round"
        />
        <g transform="translate(32 31) scale(0.74 1) translate(-32 -31)">
          <path d={SIGIL_STAR(32, 31, 11)} fill={`url(#${gold})`} opacity="0.97" />
        </g>
      </g>
    </svg>
  );
}

export function TheiaMark({ size = 24, luminous = false, complete = false, className = '' }) {
  return (
    <span
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      style={{ width: size, height: size }}
      aria-hidden="true"
    >
      <TheiaSigil size="100%" luminous={luminous} complete={complete} />
    </span>
  );
}

/* ------------------------------------------------------------------ */
/* The shore: one place, changing light across the story.              */
/* ------------------------------------------------------------------ */

const W = 360;
const HORIZON = 228;

/** Four-point spark with concave shoulders — drawn, not a glyph. */
const STAR = (cx, cy, r) => {
  const k = r * 0.11;
  return `M${cx} ${cy - r}Q${cx + k} ${cy - k} ${cx + r} ${cy}Q${cx + k} ${cy + k} ${cx} ${cy + r}Q${cx - k} ${cy + k} ${cx - r} ${cy}Q${cx - k} ${cy - k} ${cx} ${cy - r}Z`;
};

/** Fixed star field, hand-placed so the same constellations return each hour. */
const STARS = [
  [38, 58, 1.5], [86, 30, 1.1], [130, 70, 1.3], [204, 38, 1.2],
  [252, 58, 1.4], [300, 28, 1.0], [322, 92, 1.1], [56, 108, 1.0],
  [160, 22, 0.9], [230, 110, 0.9], [102, 128, 0.8], [288, 132, 0.8],
  [22, 148, 0.8], [340, 160, 0.8], [146, 94, 0.7], [268, 150, 0.7],
];
const SPARKS = [[86, 30, 5.2], [252, 58, 6.0], [300, 28, 4.0]];

/**
 * Two tonal readings of the same shore: night (on ink surfaces) and an etching
 * of the same view on ivory paper. One place, two lights.
 */
const PALETTES = {
  ink: {
    ridgeFar: '#1E2736',
    ridgeNear: '#141B26',
    rim: '#E3C48C',
    headland: '#080B11',
    tree: '#04060A',
    star: '#EDE0C8',
    gold: '#E3C48C',
    goldBright: '#FBEED4',
    rose: '#E7B6A0',
    moon: '#EAE4D4',
    bird: '#0A0D13',
    curtain: '#0A0D13',
    curtainLit: '#1B2231',
    sea: ['#2A3448', '#0A0D13'],
    seaGlow: '#E8B87A',
  },
  paper: {
    ridgeFar: '#D9CBAF',
    ridgeNear: '#B3A385',
    rim: '#A9783F',
    headland: '#332E25',
    tree: '#221E17',
    star: '#6E6555',
    gold: '#A9783F',
    goldBright: '#C79A5C',
    rose: '#BE8A68',
    moon: '#AD8549',
    bird: '#332E25',
    curtain: '#24211A',
    curtainLit: '#4A4030',
    sea: ['#EEDFC0', '#C9B692'],
    seaGlow: '#D9B679',
  },
};

const SKY_STOPS = {
  welcome: [[0, '#04060C'], [0.26, '#101828'], [0.5, '#33283A'], [0.7, '#7A4A3A'], [0.86, '#C8834E'], [1, '#F2B87E']],
  night: [[0, '#03040A'], [0.4, '#070B15'], [0.72, '#101827'], [0.92, '#1A2436'], [1, '#28374E']],
  sight: [[0, '#05070C'], [0.32, '#111927'], [0.6, '#3E2F3C'], [0.82, '#8A5A42'], [1, '#D99C68']],
  dawn: [[0, '#0B0E1A'], [0.28, '#2C2340'], [0.52, '#6E4250'], [0.74, '#B96E44'], [0.9, '#E4A06E'], [1, '#F8C08A']],
  orbits: [[0, '#05060C'], [0.34, '#101828'], [0.64, '#2E2740'], [0.86, '#57404C'], [1, '#8A5A52']],
  seal: [[0, '#0A0F1C'], [0.26, '#242A44'], [0.52, '#5E4C4E'], [0.76, '#B9855E'], [0.92, '#E8B383'], [1, '#F9D0A0']],
  veil: [[0, '#FDF8EF'], [0.5, '#F8EDDA'], [1, '#ECD6B0']],
  pulse: [[0, '#FDF8EF'], [0.5, '#F8EDDA'], [1, '#ECD6B0']],
};

/** Sun x position per state, so reflections and bloom stay registered. */
const SUN_X = { welcome: 210, dawn: 188, seal: 180, veil: 180 };

function StarField({ palette, opacity = 1, sparkOpacity = 1 }) {
  return (
    <g opacity={opacity}>
      {STARS.map(([x, y, r], i) => (
        <circle key={i} cx={x} cy={y} r={r} fill={palette.star} opacity={0.5 + (i % 3) * 0.17} />
      ))}
      <g opacity={sparkOpacity}>
        {SPARKS.map(([x, y, r], i) => (
          <path
            key={i}
            d={STAR(x, y, r)}
            fill={palette.star}
            className="ti-twinkle"
            style={{ animationDelay: `${i * 1.5}s` }}
          />
        ))}
      </g>
    </g>
  );
}

/** Horizon bloom: warm air gathered over the water where the light returns. */
function HorizonGlow({ uid, x = 180, y = HORIZON - 6, opacity = 0.8, rx = 170, ry = 52 }) {
  return <ellipse cx={x} cy={y} rx={rx} ry={ry} fill={`url(#${uid}-bloom)`} opacity={opacity} />;
}

function Sun({ x, y, r, uid, palette, halo = 0.6, haloScale = 2.4 }) {
  return (
    <g>
      <circle cx={x} cy={y} r={r * haloScale} fill={`url(#${uid}-halo)`} opacity={halo} className="ti-sun-glow" />
      <circle cx={x} cy={y} r={r} fill={`url(#${uid}-sun)`} />
      <circle cx={x} cy={y} r={r} fill="none" stroke={palette.goldBright} strokeWidth="0.6" opacity="0.4" />
    </g>
  );
}

function Moon({ x, y, r, uid, palette, opacity = 1 }) {
  return (
    <g opacity={opacity}>
      <circle cx={x} cy={y} r={r * 2.6} fill={`url(#${uid}-halo)`} opacity="0.26" />
      <mask id={`${uid}-moon`}>
        <rect x="0" y="0" width={W} height={W} fill="#000" />
        <circle cx={x} cy={y} r={r} fill="#fff" />
        <circle cx={x + r * 0.5} cy={y - r * 0.28} r={r * 0.94} fill="#000" />
      </mask>
      <g mask={`url(#${uid}-moon)`}>
        <circle cx={x} cy={y} r={r} fill={palette.moon} />
      </g>
    </g>
  );
}

/** Eos, the dawn star: a four-point spark with one thin horizontal ray. */
function DawnSpark({ x, y, r, uid, palette, opacity = 1 }) {
  return (
    <g opacity={opacity}>
      <circle cx={x} cy={y} r={r * 2.8} fill={`url(#${uid}-halo)`} opacity="0.45" />
      <path d={`M${x - r * 2.4} ${y}H${x + r * 2.4}`} stroke={palette.rose} strokeWidth="1" strokeLinecap="round" opacity="0.55" />
      <path d={STAR(x, y, r)} fill={palette.goldBright} />
    </g>
  );
}

/** Two low ridge lines, drawn once and reused at every hour. */
function Ridges({ palette, far = 0.9, near = 1, rim = 0.3 }) {
  return (
    <g>
      <path
        d={`M0 ${HORIZON} L0 190 C26 184 50 178 80 182 C110 186 130 196 156 198 C184 200 206 186 236 182 C266 178 294 188 324 192 C342 194 352 196 360 194 L360 ${HORIZON} Z`}
        fill={palette.ridgeFar}
        opacity={far}
      />
      <path
        d={`M0 ${HORIZON} L0 206 C30 198 54 196 86 202 C118 208 146 214 178 214 C212 214 238 204 268 202 C296 200 330 206 360 210 L360 ${HORIZON} Z`}
        fill={palette.ridgeNear}
        opacity={near}
      />
      <path
        d={`M0 190 C26 184 50 178 80 182 C110 186 130 196 156 198 C184 200 206 186 236 182 C266 178 294 188 324 192 C342 194 352 196 360 194`}
        fill="none"
        stroke={palette.rim}
        strokeWidth="0.8"
        opacity={rim}
      />
      <path
        d={`M0 206 C30 198 54 196 86 202 C118 208 146 214 178 214 C212 214 238 204 268 202 C296 200 330 206 360 210`}
        fill="none"
        stroke={palette.rim}
        strokeWidth="0.6"
        opacity={rim * 0.6}
      />
    </g>
  );
}

/**
 * A light path on still water: one soft bloom against the horizon and a few
 * broken, thinning strokes, so it reads as reflected light rather than a ladder.
 */
function WaterLight({ x, uid, palette, strength = 0.8, spread = 30, strokes = 4 }) {
  return (
    <g opacity={strength} className="ti-shimmer">
      <ellipse cx={x} cy={HORIZON + 16} rx={spread * 1.35} ry="26" fill={`url(#${uid}-reflect)`} />
      <ellipse cx={x} cy={HORIZON + 54} rx={spread * 0.85} ry="26" fill={`url(#${uid}-reflect)`} opacity="0.55" />
      {Array.from({ length: strokes }, (_, i) => {
        const t = i / strokes;
        const y = HORIZON + 5 + t * 88;
        const half = spread * (0.9 - t * 0.62);
        const drift = (i % 2 === 0 ? 1 : -1) * spread * (0.05 + t * 0.07);
        const bow = 2 + t * 2.4;
        return (
          <path
            key={i}
            d={`M${(x - half + drift).toFixed(1)} ${y.toFixed(1)} Q${x.toFixed(1)} ${(y + bow).toFixed(1)} ${(x + half + drift).toFixed(1)} ${y.toFixed(1)}`}
            stroke={palette.gold}
            strokeWidth={(1.3 - t * 0.7).toFixed(2)}
            strokeLinecap="round"
            opacity={(0.5 - t * 0.38).toFixed(2)}
          />
        );
      })}
    </g>
  );
}

/** The water body: still, dark, with the sky's warmth caught along its edge. */
function Sea({ uid, tone }) {
  const isPaper = tone === 'paper';
  return (
    <g>
      <rect x="0" y={HORIZON} width={W} height={W - HORIZON} fill={`url(#${uid}-sea)`} />
      <rect x="0" y={HORIZON} width={W} height="54" fill={`url(#${uid}-seahl)`} opacity={isPaper ? 0.7 : 0.5} />
    </g>
  );
}

/** Slender cypress — the one living mark on the headland. Same trees, every hour. */
function Cypress({ x, base, height = 58, width = 12, fill, accent }) {
  const tip = base - height;
  const mid = base - height * 0.52;
  return (
    <g>
      <path
        d={`M${x} ${tip}
            C${x + width * 0.36} ${tip + height * 0.2} ${x + width * 0.54} ${mid} ${x + width * 0.3} ${base - height * 0.1}
            C${x + width * 0.18} ${base - height * 0.03} ${x - width * 0.18} ${base - height * 0.03} ${x - width * 0.3} ${base - height * 0.1}
            C${x - width * 0.54} ${mid} ${x - width * 0.36} ${tip + height * 0.2} ${x} ${tip} Z`}
        fill={fill}
      />
      <path
        d={`M${x - width * 0.14} ${tip + height * 0.16} C${x - width * 0.24} ${mid} ${x - width * 0.18} ${base - height * 0.28} ${x - width * 0.08} ${base - height * 0.12}`}
        stroke={accent}
        strokeWidth="0.9"
        strokeLinecap="round"
        opacity="0.42"
        fill="none"
      />
    </g>
  );
}

/**
 * The headland: one promontory entering from the lower left and settling into
 * the bottom edge, leaving open water where the light can fall.
 */
function Headland({ palette, tone }) {
  const crest = `M0 244 C26 236 54 234 82 240 C116 248 142 266 178 282 C224 302 296 316 360 320`;
  return (
    <g>
      <path d={`${crest} L360 360 L0 360 Z`} fill={palette.headland} />
      <path d={crest} fill="none" stroke={palette.rim} strokeWidth="0.9" opacity={tone === 'paper' ? 0.45 : 0.3} />
      <Cypress x={46} base={243} height={72} width={13} fill={palette.tree} accent={palette.gold} />
      <Cypress x={70} base={246} height={50} width={10} fill={palette.tree} accent={palette.gold} />
      <Cypress x={88} base={250} height={32} width={7} fill={palette.tree} accent={palette.gold} />
    </g>
  );
}

/** Three small birds far out over the water — distance, scale, morning. */
function Birds({ palette, opacity = 0.75 }) {
  return (
    <g fill="none" stroke={palette.bird} strokeWidth="1.25" strokeLinecap="round" opacity={opacity}>
      <path d="M126 104 Q131 99 136 104 Q141 99 146 104" />
      <path d="M250 84 Q254 80 258 84 Q262 80 266 84" />
      <path d="M214 120 Q217 116 220 120 Q223 116 226 120" opacity="0.65" />
    </g>
  );
}

const STATE_LABEL = {
  welcome: 'ساحلی در نخستین نور: خورشید بر افق، سه سرو بر دماغه و بازتاب نور بر آب',
  night: 'شب بر همان ساحل: هلال ماه، ستاره‌ها و تک‌اخگری بر افق',
  veil: 'پرده: دو پرده که در میانه کنار رفته‌اند و شکافی از نور را باز کرده‌اند',
  sight: 'دیدن: چشمی از خطوط نازک بر آسمان، با خورشیدی در مردمک',
  dawn: 'سپیده: خورشید بر فراز تپه‌ها و پرتوهای کوتاه آن بر آب',
  orbits: 'سه نور: خورشید، ماه و ستارهٔ سپیده در یک آسمان',
  pulse: 'فانوس دریایی: نوری که هر شب بازمی‌گردد',
  seal: 'سپیدهٔ کامل: خورشید در حلقهٔ بسته و بازتاب آرام آن',
};

export function TheiaInstrument({ state = 'welcome', tone = 'ink', className = '', label }) {
  const uid = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const palette = PALETTES[tone] || PALETTES.ink;
  const isPaper = tone === 'paper';
  const skyStops = SKY_STOPS[state] || SKY_STOPS.welcome;

  return (
    <svg
      className={`theia-instrument theia-instrument--${state} theia-instrument--tone-${tone} ${className}`.trim()}
      viewBox={`0 0 ${W} ${W}`}
      fill="none"
      role="img"
      aria-label={label || STATE_LABEL[state] || STATE_LABEL.welcome}
      focusable="false"
    >
      <defs>
        <linearGradient id={`${uid}-sky`} x1="0" y1="0" x2="0" y2={HORIZON} gradientUnits="userSpaceOnUse">
          {skyStops.map(([offset, color]) => (
            <stop key={offset} offset={offset} stopColor={color} />
          ))}
        </linearGradient>
        <linearGradient id={`${uid}-sun`} x1="0.2" y1="0.15" x2="0.8" y2="0.95">
          {isPaper ? (
            <>
              <stop stopColor="#F4D9AC" />
              <stop offset="0.55" stopColor="#CD9E60" />
              <stop offset="1" stopColor="#A9783F" />
            </>
          ) : (
            <>
              <stop stopColor="#FFF3DA" />
              <stop offset="0.45" stopColor="#F2C588" />
              <stop offset="1" stopColor="#C98650" />
            </>
          )}
        </linearGradient>
        <radialGradient id={`${uid}-halo`} cx="50%" cy="50%" r="50%">
          <stop stopColor={isPaper ? '#D9B679' : '#F8DFAE'} stopOpacity="0.55" />
          <stop offset="0.55" stopColor={isPaper ? '#D9B679' : '#F2C58A'} stopOpacity="0.18" />
          <stop offset="1" stopColor="#F2C58A" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`${uid}-bloom`} cx="50%" cy="50%" r="50%">
          <stop stopColor={isPaper ? '#E4B478' : '#FFC98C'} stopOpacity="0.6" />
          <stop offset="0.6" stopColor={isPaper ? '#E4B478' : '#F0A868'} stopOpacity="0.18" />
          <stop offset="1" stopColor="#F0A868" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`${uid}-reflect`} cx="50%" cy="50%" r="50%">
          <stop stopColor={palette.gold} stopOpacity={isPaper ? 0.32 : 0.28} />
          <stop offset="1" stopColor={palette.gold} stopOpacity="0" />
        </radialGradient>
        <linearGradient id={`${uid}-sea`} x1="0" y1={HORIZON} x2="0" y2={W} gradientUnits="userSpaceOnUse">
          <stop stopColor={palette.sea[0]} />
          <stop offset="1" stopColor={palette.sea[1]} />
        </linearGradient>
        <linearGradient id={`${uid}-seahl`} x1="0" y1={HORIZON} x2="0" y2={HORIZON + 54} gradientUnits="userSpaceOnUse">
          <stop stopColor={palette.seaGlow} stopOpacity="0.5" />
          <stop offset="1" stopColor={palette.seaGlow} stopOpacity="0" />
        </linearGradient>
        <linearGradient id={`${uid}-waterline`} x1="0" y1="0" x2="1" y2="0">
          <stop stopColor={palette.gold} stopOpacity="0" />
          <stop offset="0.22" stopColor={palette.gold} stopOpacity="0.5" />
          <stop offset="0.78" stopColor={palette.gold} stopOpacity="0.5" />
          <stop offset="1" stopColor={palette.gold} stopOpacity="0" />
        </linearGradient>
        <linearGradient id={`${uid}-beam`} x1="0" y1="0" x2="1" y2="0">
          <stop stopColor={palette.gold} stopOpacity="0.45" />
          <stop offset="0.6" stopColor={palette.gold} stopOpacity="0.14" />
          <stop offset="1" stopColor={palette.gold} stopOpacity="0" />
        </linearGradient>
        <linearGradient id={`${uid}-beamL`} x1="1" y1="0" x2="0" y2="0">
          <stop stopColor={palette.gold} stopOpacity="0.45" />
          <stop offset="0.6" stopColor={palette.gold} stopOpacity="0.14" />
          <stop offset="1" stopColor={palette.gold} stopOpacity="0" />
        </linearGradient>
        <linearGradient id={`${uid}-curtainL`} x1="0" y1="0" x2="1" y2="0">
          <stop stopColor={palette.curtain} />
          <stop offset="0.55" stopColor={palette.curtain} />
          <stop offset="1" stopColor={palette.curtainLit} />
        </linearGradient>
        <linearGradient id={`${uid}-curtainR`} x1="1" y1="0" x2="0" y2="0">
          <stop stopColor={palette.curtain} />
          <stop offset="0.55" stopColor={palette.curtain} />
          <stop offset="1" stopColor={palette.curtainLit} />
        </linearGradient>
        <radialGradient id={`${uid}-vignette`} cx="50%" cy="46%" r="72%">
          <stop stopColor="#000000" stopOpacity="0" />
          <stop offset="0.72" stopColor="#000000" stopOpacity="0" />
          <stop offset="1" stopColor="#000000" stopOpacity={isPaper ? 0.16 : 0.42} />
        </radialGradient>
      </defs>

      {/* Sky — always the same sky; only its hour changes. */}
      <rect x="0" y="0" width={W} height={W} fill={`url(#${uid}-sky)`} />

      {state === 'welcome' && (
        <g>
          <StarField palette={palette} opacity={0.55} sparkOpacity={0.7} />
          <HorizonGlow uid={uid} x={SUN_X.welcome} opacity={1} rx={180} ry={56} />
          <Sun x={SUN_X.welcome} y={HORIZON - 44} r={40} uid={uid} palette={palette} halo={0.8} haloScale={2.9} />
          <Birds palette={palette} opacity={0.6} />
          <path d={STAR(150, 92, 5.4)} fill={palette.goldBright} opacity="0.85" className="ti-twinkle" />
          <Ridges palette={palette} rim={0.4} />
          <Sea uid={uid} tone={tone} />
          <WaterLight x={SUN_X.welcome} uid={uid} palette={palette} strength={1} spread={34} strokes={5} />
          <Headland palette={palette} tone={tone} />
          <path d={`M14 ${HORIZON}H346`} stroke={`url(#${uid}-waterline)`} strokeWidth="1" />
          {/* The brand ring as the sun's corona: it arcs over the light and
              stops short of closing. The only closed ring is the last screen. */}
          <path
            d="M158 218 A62 62 0 1 1 262 218"
            fill="none"
            stroke={palette.goldBright}
            strokeWidth="5"
            strokeLinecap="round"
            opacity="0.1"
          />
          <path
            d="M158 218 A62 62 0 1 1 262 218"
            fill="none"
            stroke={palette.gold}
            strokeWidth="1"
            strokeLinecap="round"
            opacity="0.5"
          />
        </g>
      )}

      {state === 'night' && (
        <g>
          <StarField palette={palette} opacity={1} sparkOpacity={1} />
          <Moon x={254} y={84} r={23} uid={uid} palette={palette} />
          <Ridges palette={palette} far={0.7} rim={0.14} />
          <Sea uid={uid} tone={tone} />
          <WaterLight x={254} uid={uid} palette={palette} strength={0.32} spread={16} strokes={3} />
          <Headland palette={palette} tone={tone} />
          {/* The first ember of the story, still far off on the water. */}
          <circle cx={124} cy={HORIZON + 1} r={9} fill={`url(#${uid}-halo)`} opacity="0.45" />
          <circle cx={124} cy={HORIZON + 1} r={1.9} fill={palette.goldBright} opacity="0.9" />
          <path d={`M14 ${HORIZON}H346`} stroke={`url(#${uid}-waterline)`} strokeWidth="0.7" opacity="0.28" />
        </g>
      )}

      {state === 'veil' && (
        <g>
          <StarField palette={palette} opacity={0.26} sparkOpacity={0.3} />
          <HorizonGlow uid={uid} opacity={0.85} rx={150} ry={58} />
          <Sun x={180} y={HORIZON - 62} r={25} uid={uid} palette={palette} halo={0.7} haloScale={3.2} />
          <Ridges palette={palette} far={0.8} near={0.85} rim={0.35} />
          <Sea uid={uid} tone={tone} />
          <WaterLight x={180} uid={uid} palette={palette} strength={0.85} spread={24} strokes={4} />
          <Headland palette={palette} tone={tone} />
          {/* Two draped curtains drawn apart: privacy is a doorway the person
              opens, never a wall built around them. */}
          <g>
            <path
              d="M0 0 L112 0 C118 74 124 148 130 200 C134 258 136 314 142 360 L0 360 Z"
              fill={`url(#${uid}-curtainL)`}
            />
            <path
              d="M360 0 L248 0 C242 74 236 148 230 200 C226 258 224 314 218 360 L360 360 Z"
              fill={`url(#${uid}-curtainR)`}
            />
            {/* a few quiet folds, never bands */}
            <g fill="none" stroke={palette.gold} strokeLinecap="round">
              <path d="M34 0 C40 92 44 250 50 360" strokeWidth="0.7" opacity="0.1" />
              <path d="M72 0 C78 86 84 262 90 360" strokeWidth="0.6" opacity="0.07" />
              <path d="M326 0 C320 92 316 250 310 360" strokeWidth="0.7" opacity="0.1" />
              <path d="M288 0 C282 86 276 262 270 360" strokeWidth="0.6" opacity="0.07" />
            </g>
            {/* the lit inner edge of each curtain — light touching the opening */}
            <path d="M112 0 C118 74 124 148 130 200 C134 258 136 314 142 360" fill="none" stroke={palette.goldBright} strokeWidth="2.6" opacity="0.1" />
            <path d="M112 0 C118 74 124 148 130 200 C134 258 136 314 142 360" fill="none" stroke={palette.gold} strokeWidth="0.9" opacity="0.55" />
            <path d="M248 0 C242 74 236 148 230 200 C226 258 224 314 218 360" fill="none" stroke={palette.goldBright} strokeWidth="2.6" opacity="0.1" />
            <path d="M248 0 C242 74 236 148 230 200 C226 258 224 314 218 360" fill="none" stroke={palette.gold} strokeWidth="0.9" opacity="0.55" />
            {/* the ring from the mark, held open: the gap at its base is the
                part of yourself you choose to keep or to share */}
            <g>
              <path
                d="M118 240 A88 88 0 1 1 242 240"
                fill="none"
                stroke={palette.goldBright}
                strokeWidth="6"
                strokeLinecap="round"
                opacity="0.14"
              />
              <path
                d="M118 240 A88 88 0 1 1 242 240"
                fill="none"
                stroke={palette.gold}
                strokeWidth="1.5"
                strokeLinecap="round"
                opacity="0.72"
              />
            </g>
            {/* curtain rail */}
            <path d="M0 4 H360" stroke={palette.gold} strokeWidth="0.9" opacity="0.22" />
          </g>
        </g>
      )}

      {state === 'sight' && (
        <g>
          <StarField palette={palette} opacity={0.5} sparkOpacity={0.55} />
          <HorizonGlow uid={uid} opacity={0.55} rx={150} ry={44} />
          <Ridges palette={palette} far={0.55} near={0.65} rim={0.18} />
          <Sea uid={uid} tone={tone} />
          <WaterLight x={180} uid={uid} palette={palette} strength={0.4} spread={18} strokes={3} />
          {/* The eye of Theia: lids made from the same air as the horizon. */}
          <g>
            <path d="M56 140 Q180 72 304 140 Q180 208 56 140 Z" fill={palette.ridgeNear} opacity="0.5" />
            <circle cx="180" cy="140" r="74" fill={`url(#${uid}-halo)`} opacity="0.5" />
            <path d="M56 140 Q180 72 304 140 Q180 208 56 140 Z" fill="none" stroke={palette.gold} strokeWidth="1.4" />
            <path d="M78 140 Q180 94 282 140 Q180 186 78 140 Z" fill="none" stroke={palette.gold} strokeWidth="0.5" opacity="0.4" />
            <path d="M104 120 Q180 88 256 120" fill="none" stroke={palette.gold} strokeWidth="0.6" opacity="0.28" />
            <path d="M104 160 Q180 192 256 160" fill="none" stroke={palette.gold} strokeWidth="0.6" opacity="0.16" />
            <Sun x={180} y={140} r={37} uid={uid} palette={palette} halo={0.6} haloScale={1.9} />
            <circle cx="180" cy="140" r="9.5" fill="#08090D" />
            <path d={STAR(180, 140, 4.4)} fill={palette.goldBright} />
          </g>
          <Headland palette={palette} tone={tone} />
        </g>
      )}

      {state === 'dawn' && (
        <g>
          <StarField palette={palette} opacity={0.12} sparkOpacity={0.14} />
          <HorizonGlow uid={uid} x={SUN_X.dawn} opacity={1} rx={190} ry={58} y={HORIZON - 34} />
          <g stroke={palette.gold} strokeWidth="0.9" strokeLinecap="round" opacity="0.28">
            <path d={`M${SUN_X.dawn} 136 L${SUN_X.dawn} 116`} />
            <path d="M152 144 L138 128" />
            <path d="M224 144 L238 128" />
            <path d="M124 166 L106 158" />
            <path d="M252 166 L270 158" />
            <path d="M112 192 L94 190" />
            <path d="M264 192 L282 190" />
          </g>
          <Sun x={SUN_X.dawn} y={HORIZON - 34} r={42} uid={uid} palette={palette} halo={0.85} haloScale={2.9} />
          <Birds palette={palette} opacity={0.85} />
          <Ridges palette={palette} rim={0.55} />
          <Sea uid={uid} tone={tone} />
          <WaterLight x={SUN_X.dawn} uid={uid} palette={palette} strength={1} spread={36} strokes={5} />
          <Headland palette={palette} tone={tone} />
          <path d={`M14 ${HORIZON}H346`} stroke={`url(#${uid}-waterline)`} strokeWidth="1.1" />
        </g>
      )}

      {state === 'orbits' && (
        <g>
          <StarField palette={palette} opacity={0.7} sparkOpacity={0.75} />
          {/* One faint course threads the three lights: Helios, Selene, Eos. */}
          <path
            d="M46 214 C86 186 140 168 196 160 C244 152 288 134 318 104"
            fill="none"
            stroke={palette.gold}
            strokeWidth="0.7"
            strokeDasharray="1 7"
            strokeLinecap="round"
            opacity="0.3"
          />
          <Sun x={78} y={158} r={28} uid={uid} palette={palette} halo={0.55} haloScale={2.4} />
          <Moon x={276} y={86} r={17} uid={uid} palette={palette} />
          <DawnSpark x={192} y={166} r={10} uid={uid} palette={palette} />
          <Ridges palette={palette} far={0.85} rim={0.3} />
          <Sea uid={uid} tone={tone} />
          <WaterLight x={78} uid={uid} palette={palette} strength={0.55} spread={22} strokes={3} />
          <WaterLight x={272} uid={uid} palette={palette} strength={0.26} spread={13} strokes={2} />
          <Headland palette={palette} tone={tone} />
          <path d={`M14 ${HORIZON}H346`} stroke={`url(#${uid}-waterline)`} strokeWidth="0.8" opacity="0.5" />
        </g>
      )}

      {state === 'pulse' && (
        <g>
          <StarField palette={palette} opacity={0.26} sparkOpacity={0.28} />
          <Moon x={180} y={68} r={15} uid={uid} palette={palette} opacity={0.85} />
          <Ridges palette={palette} far={0.85} rim={0.3} />
          {/* A lighthouse on the far point: light that keeps returning. */}
          <g>
            <polygon points="262,206 60,176 60,192 262,216" fill={`url(#${uid}-beamL)`} className="ti-beam" />
            <polygon points="272,206 336,190 336,204 272,216" fill={`url(#${uid}-beam)`} className="ti-beam" style={{ animationDelay: '0.7s' }} />
            <circle cx="267" cy="202" r="44" fill={`url(#${uid}-halo)`} opacity="0.6" />
            {/* the rock the tower stands on */}
            <path d="M234 300 C242 290 254 285 267 285 C280 285 292 291 298 300 Z" fill={palette.headland} />
            {/* tower: tapered, quietly banded */}
            <path d="M258 288 L261.5 214 L272.5 214 L276 288 Z" fill={palette.headland} />
            <path d="M262 268 L274 268" stroke={palette.gold} strokeWidth="0.7" opacity="0.32" />
            <path d="M261 246 L275 246" stroke={palette.gold} strokeWidth="0.7" opacity="0.26" />
            <path d="M260.5 228 L275.5 228" stroke={palette.gold} strokeWidth="0.7" opacity="0.2" />
            {/* gallery, lamp room, dome */}
            <rect x="257" y="209" width="20" height="5" rx="1" fill={palette.headland} />
            <rect x="261.5" y="197" width="11" height="12" rx="1" fill={palette.goldBright} opacity="0.95" className="ti-beam" />
            <circle cx="267" cy="203" r="8" fill={palette.goldBright} opacity="0.35" />
            <path d="M259.5 197 Q267 186 274.5 197 Z" fill={palette.headland} />
            <path d="M267 186 L267 182" stroke={palette.headland} strokeWidth="1.2" strokeLinecap="round" />
          </g>
          <Headland palette={palette} tone={tone} />
        </g>
      )}

      {state === 'seal' && (
        <g>
          <StarField palette={palette} opacity={0.18} sparkOpacity={0.2} />
          <HorizonGlow uid={uid} opacity={0.9} y={HORIZON - 44} rx={170} ry={54} />
          {/* Rays sit outside the seal ring, so the closed circle stays clean. */}
          <g stroke={palette.gold} strokeWidth="0.9" strokeLinecap="round" opacity="0.42">
            {[-150, -125, -100, -75, -50, -25].map(deg => {
              const angle = (deg * Math.PI) / 180;
              const x1 = 180 + Math.cos(angle) * 68;
              const y1 = 142 + Math.sin(angle) * 68;
              const x2 = 180 + Math.cos(angle) * 78;
              const y2 = 142 + Math.sin(angle) * 78;
              return <line key={deg} x1={x1.toFixed(1)} y1={y1.toFixed(1)} x2={x2.toFixed(1)} y2={y2.toFixed(1)} />;
            })}
          </g>
          <Sun x={180} y={142} r={45} uid={uid} palette={palette} halo={0.85} haloScale={2.6} />
          <circle cx="180" cy="142" r={59} fill="none" stroke={palette.goldBright} strokeWidth="5" opacity="0.12" />
          <circle cx="180" cy="142" r={59} fill="none" stroke={palette.gold} strokeWidth="1.3" opacity="0.9" />
          <circle cx="180" cy="142" r={59.9} fill="none" stroke={palette.gold} strokeWidth="3.4" opacity="0.18" />
          <g opacity="0.55">
            <path d="M62 108 Q67 103 72 108 Q77 103 82 108" fill="none" stroke={palette.bird} strokeWidth="1.2" strokeLinecap="round" />
            <path d="M292 116 Q296 112 300 116 Q304 112 308 116" fill="none" stroke={palette.bird} strokeWidth="1.2" strokeLinecap="round" />
          </g>
          <Ridges palette={palette} rim={0.55} />
          <Sea uid={uid} tone={tone} />
          <WaterLight x={180} uid={uid} palette={palette} strength={1} spread={38} strokes={5} />
          <Headland palette={palette} tone={tone} />
          <path d={`M14 ${HORIZON}H346`} stroke={`url(#${uid}-waterline)`} strokeWidth="1.1" />
        </g>
      )}

      {/* A soft edge shade keeps the light centred and the frame quiet. */}
      <rect x="0" y="0" width={W} height={W} fill={`url(#${uid}-vignette)`} pointerEvents="none" />
    </svg>
  );
}
