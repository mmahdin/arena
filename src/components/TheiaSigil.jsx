import React from 'react';

/**
 * Theia's 2026 identity is intentionally faceless.
 *
 * In the myth Theia (Θεία, "the far-shining") is the Titaness of sight and of
 * the light of the sky. She was never given a cult image, a temple or a statue:
 * what survives of her is light itself. Her children are Helios, Selene and Eos
 * — sun, moon and dawn. The brand therefore shows an instrument of light, never
 * a body: an astrolabe-like ring left deliberately open, one spark of dawn
 * inside it, and the quiet horizon it measures. The open ring is the product
 * promise: the app does not close around you or complete you; it lets light in
 * and leaves the circle open for the person using it.
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
/* The instrument: one drawing, changing state across the story arc.  */
/*                                                                    */
/* 2026-11-26 — second pass on the artwork. The first pass drew every */
/* idea as a full plate: each state carried the same dark disc, the   */
/* same heavy halo and the same ruled horizon. Rendered at the ~300px */
/* the screens actually use, the states lost their differences and    */
/* the palette flattened into one mood.                               */
/*                                                                    */
/* This pass keeps the identity — the open ring, the dawn star, the   */
/* measured horizon — and rebuilds the drawing as a sky plate:        */
/*   · a graduated astrolabe rim drawn in hairlines, no filled disc   */
/*   · a curved horizon (a planet limb) with two flat water-lines     */
/*   · slow star rings that keep the plate alive without noise        */
/*   · one dawn star, the same glyph in every state, lit differently  */
/*                                                                    */
/* Each state is composed around negative space rather than a painted */
/* circle, so the chapters read as one object seen at different hours */
/* instead of eight unrelated illustrations.                          */
/* ------------------------------------------------------------------ */

const CX = 180;
const CY = 180;

/** Polar coordinates; 0° is north, angles increase clockwise (matching SVG). */
function polar(radius, angle, cx = CX, cy = CY) {
  const theta = ((angle - 90) * Math.PI) / 180;
  return [cx + radius * Math.cos(theta), cy + radius * Math.sin(theta)];
}
const n2 = (value) => Number(value.toFixed(2));
const pt = (radius, angle, cx = CX, cy = CY) => {
  const [x, y] = polar(radius, angle, cx, cy);
  return [n2(x), n2(y)];
};
function arcPath(radius, from, to, cx = CX, cy = CY) {
  const [x1, y1] = polar(radius, from, cx, cy);
  const [x2, y2] = polar(radius, to, cx, cy);
  const sweep = (to - from + 360) % 360;
  const large = sweep > 180 ? 1 : 0;
  return `M${n2(x1)} ${n2(y1)}A${radius} ${radius} 0 ${large} 1 ${n2(x2)} ${n2(y2)}`;
}

/** The four-point dawn star; one glyph at every scale. */
function starPath(cx, cy, r, k = Math.max(0.7, r * 0.12)) {
  return [
    `M${n2(cx)} ${n2(cy - r)}`,
    `Q${n2(cx + k)} ${n2(cy - k)} ${n2(cx + r)} ${n2(cy)}`,
    `Q${n2(cx + k)} ${n2(cy + k)} ${n2(cx)} ${n2(cy + r)}`,
    `Q${n2(cx - k)} ${n2(cy + k)} ${n2(cx - r)} ${n2(cy)}`,
    `Q${n2(cx - k)} ${n2(cy - k)} ${n2(cx)} ${n2(cy - r)}`,
    'Z',
  ].join('');
}

/**
 * A voice is drawn as a light-line, not a waveform: the same thin gold stroke as
 * the instrument's horizons and seams, ending in the dawn star. Each voice gets
 * a different *shape* of breath — still, measured, warm, lifted, low — so the
 * cards can be told apart at a glance without reading them.
 */
export function TheiaVoiceline({ path, active = false, className = '' }) {
  const id = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const gradient = `voiceline-${id}`;
  return (
    <svg
      className={`theia-voiceline ${active ? 'is-active' : ''} ${className}`.trim()}
      viewBox="0 0 280 140"
      preserveAspectRatio="none"
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <linearGradient id={gradient} x1="0" y1="0" x2="280" y2="0" gradientUnits="userSpaceOnUse">
          <stop stopColor="currentColor" stopOpacity="0.05" />
          <stop offset="0.18" stopColor="currentColor" stopOpacity="0.45" />
          <stop offset="0.82" stopColor="currentColor" stopOpacity="0.45" />
          <stop offset="1" stopColor="currentColor" stopOpacity="0.05" />
        </linearGradient>
      </defs>
      <path d="M6 70H256" stroke="currentColor" strokeWidth="0.5" strokeOpacity="0.16" strokeDasharray="1 7" />
      <path d={path} fill="none" stroke="currentColor" strokeWidth="6" strokeOpacity={active ? 0.12 : 0.06} strokeLinecap="round" strokeLinejoin="round" />
      <path d={path} fill="none" stroke={`url(#${gradient})`} strokeWidth="1.35" strokeLinecap="round" strokeLinejoin="round" />
      {/* the line ends where the voice begins: at the dawn star */}
      <circle cx="266" cy="70" r="16" fill="currentColor" fillOpacity={active ? 0.1 : 0.05} />
      <path d={starPath(266, 70, 5.4)} fill="currentColor" fillOpacity={active ? 0.92 : 0.32} />
    </svg>
  );
}

/**
 * The horizon is a shallow circular limb, not a ruled line: the plate looks
 * down on a planet, and light is measured where a sky meets a surface.
 */
const HORIZON_Y = 250;
const HORIZON_D = `M52 ${HORIZON_Y}Q180 ${HORIZON_Y + 26} 308 ${HORIZON_Y}`;
const horizonY = (t) => HORIZON_Y + 4 * 6.5 * t * (1 - t);
const horizonAt = (t) => [n2(52 + 256 * t), n2(horizonY(t))];

/** A point on a rotated ellipse — the three orbits of Helios, Selene and Eos. */
function orbitPoint(cx, cy, rx, ry, degrees, rotation) {
  const t = (degrees * Math.PI) / 180;
  const x = rx * Math.cos(t);
  const y = ry * Math.sin(t);
  const r = (rotation * Math.PI) / 180;
  return [n2(cx + x * Math.cos(r) - y * Math.sin(r)), n2(cy + x * Math.sin(r) + y * Math.cos(r))];
}

/**
 * Two tonal readings of the same instrument: night (on ink surfaces) and an
 * etched ink (on ivory paper). Both are the same geometry, so the story keeps
 * one object while the light around it changes.
 */
const PALETTES = {
  ink: {
    boost: 1,
    glow: 1,
    frame: '#E6DAC1',
    frameSoft: '#AFA189',
    gold: ['#FCEFD6', '#E7C990', '#B3834A'],
    goldFlat: '#EFDAB2',
    bright: '#FDF3DF',
    ember: '#F0C992',
    rose: '#E8AE8C',
    haze: '#F3D8A3',
    fade: '#08090B',
  },
  paper: {
    // hairlines that read as light on night ink read as almost nothing on ivory,
    // so the etching is drawn slightly heavier and denser instead of darker
    boost: 2.1,
    glow: 1.3,
    frame: '#181511',
    frameSoft: '#6F6658',
    gold: ['#D8AC6D', '#B9854A', '#895B2B'],
    goldFlat: '#A9783F',
    bright: '#7A4F24',
    ember: '#B07A42',
    rose: '#AE775B',
    haze: '#E7C089',
    fade: '#F2EDE4',
  },
};

/**
 * The plate. Everything every chapter shares: the graduated rim, the star
 * rings, the meridian, the planet limb. `detail` fades the instrument down on
 * the night screen, where the point is how little can be seen.
 */
function Plate({ p, detail = 1 }) {
  const alpha = (base) => n2(Math.min(1, base * detail * p.boost));
  const ticks = [];
  for (let i = 0; i < 60; i += 1) {
    const angle = i * 6;
    const long = angle % 30 === 0;
    const inner = long ? 128 : 133;
    const outer = 137.5;
    const [x1, y1] = pt(inner, angle);
    const [x2, y2] = pt(outer, angle);
    ticks.push(
      <line
        key={`tick-${angle}`}
        x1={x1}
        y1={y1}
        x2={x2}
        y2={y2}
        stroke={p.frame}
        strokeWidth={long ? 0.75 : 0.5}
        opacity={long ? alpha(0.32) : alpha(0.14)}
      />,
    );
  }
  const stars = [];
  for (let i = 0; i < 24; i += 1) {
    const [x, y] = polar(115.5, i * 15 + 7.5);
    const bright = i % 6 === 0;
    stars.push(
      <circle
        key={`star-${i}`}
        cx={n2(x)}
        cy={n2(y)}
        r={bright ? 1.25 : 0.8}
        fill={p.frame}
        opacity={alpha(bright ? 0.34 : 0.18)}
      />,
    );
  }
  return (
    <g className="ti-plate">
      {/* plate edge: one hairline, then the graduated scale */}
      <circle cx={CX} cy={CY} r="145" stroke={p.frame} strokeWidth="0.5" opacity={alpha(0.08)} />
      <circle cx={CX} cy={CY} r="139" stroke={p.frame} strokeWidth="0.5" opacity={alpha(0.16)} />
      <g>{ticks}</g>
      {/* the star ring turns once every five minutes: motion sensed, not watched */}
      <g className="ti-ring">{stars}</g>
      <circle cx={CX} cy={CY} r="107" stroke={p.frameSoft} strokeWidth="0.45" strokeDasharray="1 8" opacity={alpha(0.45)} />
      <circle cx={CX} cy={CY} r="88" stroke={p.frameSoft} strokeWidth="0.4" opacity={alpha(0.12)} />
      {/* meridian */}
      <path d={`M${CX} 48V${HORIZON_Y - 8}`} stroke={p.frame} strokeWidth="0.5" opacity={alpha(0.09)} />
      {/* the horizon: a curved limb, then two flat water-lines echoing it */}
      <path d={HORIZON_D} stroke={p.frame} strokeWidth="0.9" opacity={alpha(0.42)} />
      <path d="M74 272H286" stroke={p.frame} strokeWidth="0.5" opacity={alpha(0.16)} />
      <path d="M102 279H258" stroke={p.frame} strokeWidth="0.45" opacity={alpha(0.09)} />
    </g>
  );
}

/** The dawn star with its own light: the only filled shape in the drawing. */
function DawnStar({ uid, p, r = 12, glow = 1, opacity = 1, cx = CX, cy = CY, halo = 5.2 }) {
  return (
    <g opacity={opacity}>
      {glow > 0 && <circle cx={cx} cy={cy} r={n2(r * halo)} fill={`url(#ti-halo-${uid})`} opacity={n2(0.55 * glow)} />}
      <circle cx={cx} cy={cy} r={n2(Math.max(6, r * 1.7))} fill={`url(#ti-core-${uid})`} opacity={n2(0.55 * glow)} />
      <path d={starPath(cx, cy, r)} fill={`url(#ti-gold-${uid})`} />
    </g>
  );
}

const STATE_LABEL = {
  welcome: 'حلقهٔ باز تیا: دایره‌ای از نور که کامل نمی‌شود و ستاره‌ای از سپیده در مرکزش',
  night: 'شب: حلقه تاریک است و تنها اخگری در مرکز آن مانده',
  veil: 'پرده: پرده‌ای که در میانه باز شده و شکافی از نور را نشان می‌دهد',
  sight: 'دیدن: چشمی از خطوط نازک در دل حلقه، با مردمکی از نور',
  dawn: 'سپیده: نوار نور در افق و کمانی که کم‌کم روشن می‌شود',
  orbits: 'سه نور: خورشید، ماه و سپیده در سه مدار جدا',
  pulse: 'یادآوری: حلقه‌ای که نورش بازمی‌گردد',
  seal: 'حلقهٔ کامل: دایره بسته می‌شود و ستاره در مرکز می‌درخشد',
  spark: 'اخگر تیا: ستاره‌ای تنها با کوتاه‌ترین پرتوها',
};

export function TheiaInstrument({ state = 'welcome', tone = 'ink', className = '', label }) {
  const uid = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const p = PALETTES[tone] || PALETTES.ink;
  const detail = state === 'night' ? 0.5 : state === 'spark' ? 0.7 : 1;
  const gold = `url(#ti-gold-${uid})`;
  const frame = `url(#ti-frame-${uid})`;

  return (
    <svg
      className={`theia-instrument theia-instrument--${state} theia-instrument--tone-${tone} ${className}`.trim()}
      viewBox="0 0 360 360"
      fill="none"
      role="img"
      aria-label={label || STATE_LABEL[state] || STATE_LABEL.welcome}
      focusable="false"
    >
      <defs>
        <linearGradient id={`ti-gold-${uid}`} x1="62" y1="46" x2="302" y2="306" gradientUnits="userSpaceOnUse">
          <stop stopColor={p.gold[0]} />
          <stop offset="0.46" stopColor={p.gold[1]} />
          <stop offset="1" stopColor={p.gold[2]} />
        </linearGradient>
        <linearGradient id={`ti-frame-${uid}`} x1="180" y1="30" x2="180" y2="330" gradientUnits="userSpaceOnUse">
          <stop stopColor={p.frame} />
          <stop offset="0.68" stopColor={p.frameSoft} />
          <stop offset="1" stopColor={p.frameSoft} stopOpacity="0.4" />
        </linearGradient>
        <linearGradient id={`ti-dawn-${uid}`} x1="180" y1="100" x2="180" y2="260" gradientUnits="userSpaceOnUse">
          <stop stopColor={p.ember} stopOpacity="0.9" />
          <stop offset="0.55" stopColor={p.gold[1]} stopOpacity="0.4" />
          <stop offset="1" stopColor={p.gold[2]} stopOpacity="0.06" />
        </linearGradient>
        <linearGradient id={`ti-seam-${uid}`} x1="0" y1="0" x2="1" y2="0">
          <stop stopColor={p.ember} stopOpacity="0" />
          <stop offset="0.5" stopColor={p.bright} stopOpacity="0.9" />
          <stop offset="1" stopColor={p.ember} stopOpacity="0" />
        </linearGradient>
        <radialGradient id={`ti-halo-${uid}`} cx="50%" cy="50%" r="50%">
          <stop stopColor={p.haze} stopOpacity={n2(0.6 * p.glow)} />
          <stop offset="0.4" stopColor={p.ember} stopOpacity={n2(0.16 * p.glow)} />
          <stop offset="1" stopColor={p.ember} stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`ti-core-${uid}`} cx="50%" cy="50%" r="50%">
          <stop stopColor={p.bright} stopOpacity={n2(0.9 * p.glow)} />
          <stop offset="0.5" stopColor={p.gold[0]} stopOpacity={n2(0.3 * p.glow)} />
          <stop offset="1" stopColor={p.gold[0]} stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`ti-rise-${uid}`} cx="50%" cy="50%" r="50%">
          <stop stopColor={p.haze} stopOpacity="0.5" />
          <stop offset="0.3" stopColor={p.ember} stopOpacity="0.2" />
          <stop offset="0.66" stopColor={p.ember} stopOpacity="0.05" />
          <stop offset="1" stopColor={p.fade} stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`ti-night-field-${uid}`} cx="50%" cy="44%" r="56%">
          <stop stopColor={p.gold[1]} stopOpacity="0.05" />
          <stop offset="1" stopColor={p.fade} stopOpacity="0" />
        </radialGradient>
        <filter id={`ti-soft-${uid}`} x="-45%" y="-45%" width="190%" height="190%">
          <feGaussianBlur stdDeviation="6.5" />
        </filter>
        <filter id={`ti-soft-sm-${uid}`} x="-45%" y="-45%" width="190%" height="190%">
          <feGaussianBlur stdDeviation="2.6" />
        </filter>
      </defs>

      <Plate p={p} detail={detail} />

      {state === 'welcome' && (
        <g>
          <circle cx={CX} cy={CY} r="150" fill={`url(#ti-halo-${uid})`} opacity="0.2" />
          {/* the aperture: one open ring, glowing where light enters */}
          <path d={arcPath(97, 27, 333)} stroke={gold} strokeWidth="7" strokeLinecap="round" opacity="0.1" filter={`url(#ti-soft-${uid})`} />
          <path d={arcPath(97, 27, 333)} stroke={gold} strokeWidth="1.6" strokeLinecap="round" />
          <circle cx={CX} cy={CY} r="58" stroke={p.frame} strokeWidth="0.55" strokeDasharray="1 5" opacity="0.34" />
          <circle cx={CX} cy={CY} r="44" stroke={p.frameSoft} strokeWidth="0.4" opacity="0.14" />
          <DawnStar uid={uid} p={p} r={20} />
        </g>
      )}

      {state === 'night' && (
        <g>
          <circle cx={CX} cy={CY} r="140" fill={`url(#ti-night-field-${uid})`} opacity="0.75" />
          <circle cx={CX} cy={CY} r="97" stroke={p.frame} strokeWidth="0.6" strokeDasharray="1 7" opacity={n2(Math.min(1, 0.26 * p.boost))} />
          {/* not-knowing drawn as almost-no-light: five faint stars and one ember */}
          {[[-62, -38, 0.6], [74, -54, 0.45], [-88, 26, 0.5], [92, 34, 0.55], [-30, -88, 0.4]].map(([dx, dy, o], i) => (
            <circle key={i} cx={CX + dx} cy={CY + dy} r="0.9" fill={p.frame} opacity={o * 0.5} />
          ))}
          <path
            d={`M${horizonAt(0.4)[0]} ${horizonAt(0.4)[1]}Q180 ${n2(horizonY(0.5) + 1)} ${horizonAt(0.6)[0]} ${horizonAt(0.6)[1]}`}
            stroke={p.gold[1]}
            strokeWidth="1"
            strokeLinecap="round"
            opacity="0.26"
          />
          <circle cx={CX} cy={CY} r="34" fill={`url(#ti-halo-${uid})`} opacity="0.16" />
          <DawnStar uid={uid} p={p} r={4.6} glow={0.55} opacity={0.75} halo={7} />
        </g>
      )}

      {state === 'veil' && (
        <g>
          {/* a curtain drawn as fine folds, parted at the middle: the opening is chosen */}
          {[30, 55, 81, 109].map((offset, i) => (
            [-1, 1].map((side) => {
              const x = 180 + side * offset;
              const bend = side * (5 - i);
              return (
                <path
                  key={`fold-${offset}-${side}`}
                  d={`M${x - bend} 98C${x + bend} 150 ${x - bend} 206 ${x + bend * 0.6} 258`}
                  stroke={p.frame}
                  strokeWidth={i === 0 ? 0.85 : 0.6 - i * 0.06}
                  opacity={n2(0.32 - i * 0.07)}
                  strokeLinecap="round"
                />
              );
            })
          ))}
          {/* the gathered edges of the opening, and the slit of light between them */}
          {[-1, 1].map((side) => (
            <path
              key={`edge-${side}`}
              d={`M${180 + side * 20} 106C${180 + side * 26} 150 ${180 + side * 22} 206 ${180 + side * 18} 250`}
              stroke={p.frame}
              strokeWidth="0.7"
              opacity={n2(Math.min(1, 0.26 * p.boost))}
              strokeLinecap="round"
            />
          ))}
          <path d="M180 110V248" stroke={p.frame} strokeWidth="0.5" opacity={n2(Math.min(1, 0.12 * p.boost))} strokeDasharray="1 6" />
          <path d="M180 112Q192 180 180 246Q168 180 180 112Z" fill={`url(#ti-dawn-${uid})`} opacity="0.5" filter={`url(#ti-soft-sm-${uid})`} />
          <path d="M180 126Q186.5 180 180 232Q173.5 180 180 126Z" fill={gold} opacity="0.45" />
          <circle cx={CX} cy={CY} r="20" fill={`url(#ti-core-${uid})`} opacity="0.45" />
          <DawnStar uid={uid} p={p} r={9.5} glow={0.6} />
        </g>
      )}

      {state === 'sight' && (
        <g>
          <circle cx={CX} cy={CY} r="112" fill={`url(#ti-night-field-${uid})`} opacity="0.8" />
          {/* an eye made of hairlines: two lids, an iris, a lit pupil */}
          <path d="M98 180Q180 128 262 180" stroke={frame} strokeWidth="0.85" opacity="0.58" fill="none" />
          <path d="M98 180Q180 232 262 180" stroke={frame} strokeWidth="0.8" opacity="0.32" fill="none" />
          <path d="M112 180Q180 140 248 180" stroke={p.frame} strokeWidth="0.45" opacity="0.14" fill="none" />
          <path d="M122 180Q180 149 238 180" stroke={p.frame} strokeWidth="0.55" opacity="0.2" fill="none" />
          <path d="M122 180Q180 211 238 180" stroke={p.frame} strokeWidth="0.55" opacity="0.12" fill="none" />
          <path d="M151 180A29 29 0 0 1 209 180A29 20 0 0 1 151 180Z" stroke={p.frame} strokeWidth="0.65" opacity="0.28" fill="none" />
          <path d="M160 180A20 20 0 0 1 200 180A20 13 0 0 1 160 180Z" stroke={p.frame} strokeWidth="0.5" opacity="0.16" fill="none" />
          {[-62, -40, 0, 40, 62].map((deg) => {
            const [x1, y1] = polar(29, 90 + deg);
            const [x2, y2] = polar(37, 90 + deg);
            return <line key={deg} x1={n2(x1)} y1={n2(y1)} x2={n2(x2)} y2={n2(y2)} stroke={p.frame} strokeWidth="0.45" opacity="0.16" strokeLinecap="round" />;
          })}
          <g className="ti-sight-rays">
            {Array.from({ length: 16 }, (_, i) => {
              const angle = i * 22.5 + 11.25;
              const [x1, y1] = pt(69, angle);
              const [x2, y2] = pt(77, angle);
              return <line key={i} x1={x1} y1={y1} x2={x2} y2={y2} stroke={p.frame} strokeWidth="0.5" opacity={i % 2 ? 0.16 : 0.28} strokeLinecap="round" />;
            })}
          </g>
          <circle cx={CX} cy={CY} r="30" fill={`url(#ti-halo-${uid})`} opacity="0.75" />
          <DawnStar uid={uid} p={p} r={8} glow={0.85} halo={6} />
          <circle cx={CX} cy={CY} r="48" stroke={p.gold[2]} strokeWidth="0.6" opacity="0.22" className="ti-look" />
        </g>
      )}

      {state === 'dawn' && (
        <g>
          {/* the sun returns through the one place the ring is left open */}
          <circle cx={CX} cy={n2(horizonY(0.5) - 2)} r="132" fill={`url(#ti-rise-${uid})`} opacity="0.95" />
          <path d={arcPath(118, 218, 502)} stroke={gold} strokeWidth="6" strokeLinecap="round" opacity="0.1" filter={`url(#ti-soft-${uid})`} />
          <path d={arcPath(118, 218, 502)} stroke={gold} strokeWidth="1.4" strokeLinecap="round" />
          <path d={arcPath(70, 226, 494)} stroke={frame} strokeWidth="0.6" opacity="0.2" />
          <path d="M132 266H228" stroke={`url(#ti-seam-${uid})`} strokeWidth="1.5" strokeLinecap="round" opacity="0.85" />
          <path d="M156 274H204" stroke={`url(#ti-seam-${uid})`} strokeWidth="0.9" strokeLinecap="round" opacity="0.35" />
          <DawnStar uid={uid} p={p} r={12} glow={1} cy={n2(horizonY(0.5) - 1)} halo={7} />
        </g>
      )}

      {state === 'orbits' && (
        <g>
          {/* three lights, three orbits, none of them permanent */}
          {[[112, 38, 0.2], [85, 28, 0.16], [58, 19, 0.13]].map(([rx, ry, o]) => (
            <ellipse key={rx} cx={CX} cy="174" rx={rx} ry={ry} stroke={p.frame} strokeWidth="0.5" opacity={o} transform={`rotate(-8 ${CX} 174)`} />
          ))}
          {(() => {
            const sun = orbitPoint(CX, 174, 112, 38, -34, -8);
            const moon = orbitPoint(CX, 174, 85, 28, 148, -8);
            const eos = orbitPoint(CX, 174, 58, 19, 64, -8);
            const mote = orbitPoint(CX, 174, 85, 28, -122, -8);
            return (
              <g>
                {/* Helios — a sun with a short corona */}
                <circle cx={sun[0]} cy={sun[1]} r="20" fill={`url(#ti-halo-${uid})`} opacity="0.95" />
                <circle cx={sun[0]} cy={sun[1]} r="9.2" stroke={p.gold[1]} strokeWidth="0.5" opacity="0.35" />
                <circle cx={sun[0]} cy={sun[1]} r="6.2" fill={gold} />
                {Array.from({ length: 16 }, (_, i) => {
                  const t = (i * Math.PI) / 8;
                  const long = i % 2 === 0;
                  return (
                    <line
                      key={i}
                      x1={n2(sun[0] + Math.cos(t) * 9.6)}
                      y1={n2(sun[1] + Math.sin(t) * 9.6)}
                      x2={n2(sun[0] + Math.cos(t) * (long ? 13.4 : 11.4))}
                      y2={n2(sun[1] + Math.sin(t) * (long ? 13.4 : 11.4))}
                      stroke={p.gold[1]}
                      strokeWidth="0.55"
                      opacity={long ? 0.5 : 0.3}
                      strokeLinecap="round"
                    />
                  );
                })}
                {/* Selene — a crescent, the light she borrows */}
                <circle cx={moon[0]} cy={moon[1]} r="14" fill={`url(#ti-halo-${uid})`} opacity="0.34" />
                <path
                  d={`M${moon[0]} ${n2(moon[1] - 8)}A8 8 0 1 0 ${n2(moon[0] + 3.4)} ${n2(moon[1] + 7.2)}A6.6 6.6 0 0 1 ${moon[0]} ${n2(moon[1] - 8)}Z`}
                  fill={p.goldFlat}
                  opacity="0.85"
                />
                {/* Eos — the dawn, a low seam of warm light */}
                <circle cx={eos[0]} cy={eos[1]} r="12" fill={`url(#ti-halo-${uid})`} opacity="0.55" />
                <line x1={n2(eos[0] - 7.5)} y1={eos[1]} x2={n2(eos[0] + 7.5)} y2={eos[1]} stroke={p.rose} strokeWidth="1.4" strokeLinecap="round" opacity="0.9" />
                <line x1={n2(eos[0] - 4)} y1={n2(eos[1] - 3.4)} x2={n2(eos[0] + 4)} y2={n2(eos[1] - 3.4)} stroke={p.rose} strokeWidth="0.8" strokeLinecap="round" opacity="0.45" />
                {/* one small planet, so the sky is not a diagram */}
                <circle cx={mote[0]} cy={mote[1]} r="1.5" fill={p.frame} opacity="0.4" />
              </g>
            );
          })()}
          <circle cx={CX} cy="174" r="30" fill={`url(#ti-halo-${uid})`} opacity="0.22" />
          <DawnStar uid={uid} p={p} r={9} glow={0.55} cy={174} halo={6} />
        </g>
      )}

      {state === 'pulse' && (
        <g>
          {/* the ring keeps its light and gives it back, once, quietly */}
          <circle cx={CX} cy={CY} r="97" stroke={p.frame} strokeWidth="0.5" strokeDasharray="1 7" opacity={n2(Math.min(1, 0.22 * p.boost))} />
          <path d={arcPath(97, 128, 232)} stroke={gold} strokeWidth="4.5" strokeLinecap="round" opacity="0.13" filter={`url(#ti-soft-${uid})`} />
          <path d={arcPath(97, 128, 232)} stroke={gold} strokeWidth="1.7" strokeLinecap="round" />
          <path d={arcPath(97, 158, 202)} stroke={p.bright} strokeWidth="2.6" strokeLinecap="round" opacity="0.55" filter={`url(#ti-soft-sm-${uid})`} />
          {/* one afterglow, breathing outward; static-safe under reduced motion */}
          <circle cx={CX} cy={CY} r="97" stroke={p.gold[1]} strokeWidth="0.9" fill="none" opacity="0.3" className="ti-pulse" />
          <circle cx={CX} cy={CY} r="74" stroke={p.gold[2]} strokeWidth="0.5" fill="none" opacity="0.18" />
          <DawnStar uid={uid} p={p} r={12} halo={5.6} />
        </g>
      )}

      {state === 'seal' && (
        <g>
          <circle cx={CX} cy={CY} r="150" fill={`url(#ti-halo-${uid})`} opacity="0.16" />
          <circle cx={CX} cy={CY} r="137.2" stroke={gold} strokeWidth="1" opacity="0.35" />
          {/* a completed measure: cardinal rays, then the fine scale between them */}
          <g className="ti-rays">
            {Array.from({ length: 60 }, (_, i) => {
              const angle = i * 6;
              const cardinal = angle % 90 === 0;
              const major = angle % 45 === 0;
              if (!cardinal && !major && i % 2) return null;
              const inner = cardinal ? 141 : major ? 143 : 146;
              const [x1, y1] = pt(inner, angle);
              const [x2, y2] = pt(153, angle);
              return (
                <line
                  key={i}
                  x1={x1}
                  y1={y1}
                  x2={x2}
                  y2={y2}
                  stroke={p.gold[1]}
                  strokeWidth={cardinal ? 0.9 : 0.55}
                  opacity={cardinal ? 0.48 : major ? 0.28 : 0.16}
                  strokeLinecap="round"
                />
              );
            })}
          </g>
          <circle cx={CX} cy={CY} r="97" stroke={gold} strokeWidth="1.9" fill="none" opacity="0.95" />
          <circle cx={CX} cy={CY} r="103" stroke={p.gold[1]} strokeWidth="0.5" fill="none" opacity="0.4" />
          <circle cx={CX} cy={CY} r="60" stroke={p.frame} strokeWidth="0.55" strokeDasharray="1 6" opacity="0.24" fill="none" />
          <DawnStar uid={uid} p={p} r={22} halo={4.6} />
        </g>
      )}

      {state === 'spark' && (
        <g>
          <circle cx={CX} cy={CY} r="86" fill={`url(#ti-halo-${uid})`} opacity="0.16" />
          {[0, 90, 180, 270].map((angle) => {
            const [x1, y1] = pt(27, angle);
            const [x2, y2] = pt(41, angle);
            return <line key={angle} x1={x1} y1={y1} x2={x2} y2={y2} stroke={p.gold[1]} strokeWidth="0.7" opacity="0.4" strokeLinecap="round" />;
          })}
          {[45, 135, 225, 315].map((angle) => {
            const [x, y] = pt(53, angle);
            return <circle key={angle} cx={x} cy={y} r="1" fill={p.frame} opacity="0.35" />;
          })}
          <DawnStar uid={uid} p={p} r={13} glow={0.9} halo={6} />
        </g>
      )}
    </svg>
  );
}
