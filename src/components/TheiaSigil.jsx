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
/* The instrument: one drawing, changing light across the story arc.  */
/* ------------------------------------------------------------------ */
/*
 * Every onboarding illustration is the same engraved plate: a graduated limb,
 * a slowly turning rete of star pointers, a tympan disc and the horizon it
 * measures. Only the light changes — night, veil, sight, dawn, three lights,
 * pulse, seal — so the journey reads as one object observed over time instead
 * of a set of unrelated pictures. Everything is deterministic vector geometry,
 * so it stays crisp, re-tones itself for ink and paper, and animates without a
 * single raster asset.
 */

const CX = 180;
const CY = 174;
const HORIZON = 254;
const DISC_R = 104;
const LIMB_R = 157;

const round = (value) => Number(value.toFixed(2));

/** Polar coordinates; 0° is north, angles increase clockwise (matching SVG). */
function polar(radius, angle, cx = CX, cy = CY) {
  const theta = ((angle - 90) * Math.PI) / 180;
  return [round(cx + radius * Math.cos(theta)), round(cy + radius * Math.sin(theta))];
}

function arcPath(radius, from, to, cx = CX, cy = CY) {
  const [x1, y1] = polar(radius, from, cx, cy);
  const [x2, y2] = polar(radius, to, cx, cy);
  const sweep = (to - from + 360) % 360;
  const large = sweep > 180 ? 1 : 0;
  return `M${x1} ${y1}A${radius} ${radius} 0 ${large} 1 ${x2} ${y2}`;
}

function ellipsePoint(cx, cy, rx, ry, rotation, angle) {
  const t = (angle * Math.PI) / 180;
  const r = (rotation * Math.PI) / 180;
  const x = rx * Math.cos(t);
  const y = ry * Math.sin(t);
  return [round(cx + x * Math.cos(r) - y * Math.sin(r)), round(cy + x * Math.sin(r) + y * Math.cos(r))];
}

/** A four-point star with a hand-tuned waist, the product's single spark. */
function starPath(cx = CX, cy = CY, radius = 14, waist = 0.085) {
  const k = radius * waist;
  const r = radius;
  const x = round(cx);
  const y = round(cy);
  return [
    `M${x} ${round(y - r)}`,
    `Q${round(x + k)} ${round(y - k)} ${round(x + r)} ${y}`,
    `Q${round(x + k)} ${round(y + k)} ${x} ${round(y + r)}`,
    `Q${round(x - k)} ${round(y + k)} ${round(x - r)} ${y}`,
    `Q${round(x - k)} ${round(y - k)} ${x} ${round(y - r)}`,
    'Z',
  ].join(' ');
}

/* Star dust is scattered on a golden-angle spiral so it reads as sky rather
   than decoration: deterministic, identical on every render, never random. */
const DUST = Array.from({ length: 46 }, (_, index) => {
  const angle = (index * 137.508 + 19) % 360;
  const radius = 110 + ((index * 37) % 44);
  const [x, y] = polar(radius, angle);
  return {
    x,
    y,
    radius: round(0.42 + ((index * 13) % 6) * 0.15),
    opacity: round(0.14 + ((index * 17) % 8) * 0.05),
    glint: index % 13 === 0,
    delay: round((index % 8) * 1.4),
  };
}).filter((star) => star.y < HORIZON - 12);

const ORBIT = {
  helios: { cx: 180, cy: 176, rx: 94, ry: 37, rot: -22, at: 12 },
  selene: { cx: 180, cy: 176, rx: 71, ry: 28, rot: 34, at: 193 },
  eos: { cx: 180, cy: 176, rx: 47, ry: 19, rot: -6, at: 116 },
};

/**
 * Two tonal readings of the same plate: light on night ink, and a warm etching
 * on ivory paper. The geometry never changes — only the ink does.
 */
const PALETTES = {
  ink: {
    mode: 'ink',
    frame: '#EDE1C9',
    frameSoft: '#C3B190',
    rim: '#E8D8BA',
    goldLow: '#F9EDD2',
    goldMid: '#E3C48C',
    goldDeep: '#AF814A',
    bright: '#FDF3DC',
    rose: '#EDBCA4',
    glow: '#F3D8A3',
    star: '#F7EBD3',
    weight: 1,
  },
  paper: {
    mode: 'paper',
    frame: '#332D24',
    frameSoft: '#6E6350',
    rim: '#4B4234',
    goldLow: '#C69256',
    goldMid: '#A4713A',
    goldDeep: '#7B5024',
    bright: '#855726',
    rose: '#A96F58',
    glow: '#D6A468',
    star: '#7C6C52',
    weight: 1.1,
  },
};

const ToneContext = React.createContext(null);
const useArt = () => React.useContext(ToneContext);

function Frame({ detail = 1 }) {
  const { p } = useArt();
  const weight = p.weight * detail;
  return (
    <g className="ti-frame">
      <circle cx={CX} cy={CY} r={LIMB_R} stroke={p.frame} strokeWidth="0.5" opacity={0.1 * weight} />
      <circle cx={CX} cy={CY} r="148" stroke={p.frame} strokeWidth="0.6" opacity={0.22 * weight} />
      <g className="ti-graduation">
        {Array.from({ length: 72 }, (_, index) => {
          const angle = index * 5;
          const major = angle % 30 === 0;
          const mid = angle % 15 === 0;
          const [x1, y1] = polar(major ? 142.5 : mid ? 146 : 148.5, angle);
          const [x2, y2] = polar(LIMB_R, angle);
          return (
            <line
              key={angle}
              x1={x1} y1={y1} x2={x2} y2={y2}
              stroke={p.frame}
              strokeWidth={major ? 0.8 : 0.55}
              opacity={round((major ? 0.4 : mid ? 0.24 : 0.13) * weight)}
            />
          );
        })}
      </g>
      <circle cx={CX} cy={CY} r="139" stroke={p.frame} strokeWidth="0.5" opacity={0.14 * weight} />
      <circle
        cx={CX} cy={CY} r="135" stroke={p.frameSoft} strokeWidth="0.7"
        strokeDasharray="1 6" opacity={0.34 * weight} className="ti-drift"
      />
      <circle cx={CX} cy={CY} r="123" stroke={p.frameSoft} strokeWidth="0.4" opacity={0.09 * weight} />

      {/* horizon plate: the line every light in the story is measured against */}
      <path d={`M26 ${HORIZON}H334`} stroke={p.horizon} strokeWidth="0.9" opacity={0.85 * detail} />
      <path d={`M58 ${HORIZON + 6}H302`} stroke={p.frame} strokeWidth="0.5" opacity={0.15 * weight} />
      <path d={`M96 ${HORIZON + 12}H264`} stroke={p.frame} strokeWidth="0.45" opacity={0.09 * weight} />
      <path d={`M132 ${HORIZON + 19}H228`} stroke={p.frame} strokeWidth="0.4" opacity={0.06 * weight} />
      <g opacity={round(0.32 * weight)}>
        {Array.from({ length: 11 }, (_, index) => {
          const x = 130 + index * 10;
          const tall = index % 5 === 0;
          return (
            <line key={x} x1={x} y1={HORIZON + 1.5} x2={x} y2={HORIZON + (tall ? 7.5 : 4.5)}
              stroke={p.frame} strokeWidth="0.55" strokeLinecap="round" />
          );
        })}
      </g>

      <g className="ti-dust">
        {DUST.map((star, index) => (star.glint ? (
          <path
            key={index}
            d={starPath(star.x, star.y, round(star.radius * 3.2), 0.13)}
            fill={p.star}
            opacity={round(Math.min(0.65, star.opacity * 1.5))}
            className="ti-twinkle"
            style={{ animationDelay: `${star.delay}s` }}
          />
        ) : (
          <circle key={index} cx={star.x} cy={star.y} r={star.radius} fill={p.star} opacity={star.opacity} />
        )))}
      </g>
    </g>
  );
}

function Disc({ detail = 1, shade = 0 }) {
  const { p, ids } = useArt();
  return (
    <g>
      <circle cx={CX} cy={CY} r={DISC_R} fill={p.disc} />
      <circle cx={CX} cy={CY} r={DISC_R} stroke={p.rim} strokeWidth="0.6" opacity={0.2 * detail} />
      {/* the engraved tympan: a few meridians, never a grid */}
      <g fill="none">
        {[38, 68, 90].map((radius) => (
          <circle key={radius} cx={CX} cy={CY} r={radius} stroke={p.rim}
            strokeWidth="0.45" opacity={radius > 80 ? 0.07 * detail : 0.13 * detail} />
        ))}
        {[18, 72, 126, 198, 252, 306].map((angle) => {
          const [x1, y1] = polar(26, angle);
          const [x2, y2] = polar(DISC_R - 10, angle);
          return <line key={angle} x1={x1} y1={y1} x2={x2} y2={y2}
            stroke={p.rim} strokeWidth="0.4" opacity={0.09 * detail} />;
        })}
      </g>
      {shade > 0 && (
        <circle cx={CX} cy={CY} r={DISC_R} fill={p.shade} opacity={shade * (p.mode === 'paper' ? 0.4 : 1)} />
      )}
    </g>
  );
}

function Halo({ x = CX, y = CY, radius = 60, opacity = 0.5 }) {
  const { p } = useArt();
  return <circle cx={x} cy={y} r={radius} fill={p.halo} opacity={opacity} />;
}

function Star({ x = CX, y = CY, radius = 16, opacity = 1, glow = true, rotate = 0, waist }) {
  const { p } = useArt();
  return (
    <g opacity={opacity} transform={`rotate(${rotate} ${x} ${y})`}>
      {glow && <circle cx={x} cy={y} r={round(radius * 3.1)} fill={p.halo} opacity="0.5" />}
      <path d={waist ? starPath(x, y, radius, waist) : starPath(x, y, radius)} fill={p.gold} />
    </g>
  );
}

function Flare({ x = CX, y = CY, length = 64, opacity = 0.26 }) {
  const { p } = useArt();
  return (
    <g opacity={opacity} stroke={p.bright} strokeWidth="0.7" strokeLinecap="round">
      <line x1={x} y1={y - length} x2={x} y2={y + length} />
      <line x1={x - length * 0.52} y1={y} x2={x + length * 0.52} y2={y} opacity="0.7" />
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
};

export function TheiaInstrument({ state = 'welcome', tone = 'ink', className = '', label }) {
  const base = PALETTES[tone] || PALETTES.ink;
  const raw = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const id = (name) => `ti-${raw}-${name}`;
  const ids = React.useMemo(() => ({
    gold: id('gold'),
    halo: id('halo'),
    dawn: id('dawn'),
    slit: id('slit'),
    horizon: id('horizon'),
    disc: id('disc'),
    shade: id('shade'),
    sun: id('sun'),
    soft: id('soft'),
    softSm: id('soft-sm'),
    moon: id('moon'),
    discClip: id('disc-clip'),
    reflect: id('reflect'),
    fold: id('fold'),
    panel: id('panel'),
    shadow: id('shadow'),
  }), [raw]);
  const p = React.useMemo(() => ({
    ...base,
    gold: `url(#${ids.gold})`,
    halo: `url(#${ids.halo})`,
    dawn: `url(#${ids.dawn})`,
    slit: `url(#${ids.slit})`,
    horizon: `url(#${ids.horizon})`,
    disc: `url(#${ids.disc})`,
    shade: `url(#${ids.shade})`,
  }), [base, ids]);

  const helios = ellipsePoint(ORBIT.helios.cx, ORBIT.helios.cy, ORBIT.helios.rx, ORBIT.helios.ry, ORBIT.helios.rot, ORBIT.helios.at);
  const selene = ellipsePoint(ORBIT.selene.cx, ORBIT.selene.cy, ORBIT.selene.rx, ORBIT.selene.ry, ORBIT.selene.rot, ORBIT.selene.at);
  const eos = ellipsePoint(ORBIT.eos.cx, ORBIT.eos.cy, ORBIT.eos.rx, ORBIT.eos.ry, ORBIT.eos.rot, ORBIT.eos.at);

  const softLights = ['welcome', 'night', 'veil', 'sight', 'dawn', 'pulse', 'seal'];

  return (
    <ToneContext.Provider value={{ p, ids }}>
      <svg
        className={`theia-instrument theia-instrument--${state} theia-instrument--tone-${tone} ${className}`.trim()}
        viewBox="0 0 360 360"
        fill="none"
        role="img"
        aria-label={label || STATE_LABEL[state] || STATE_LABEL.welcome}
        focusable="false"
      >
        <defs>
          <linearGradient id={ids.gold} x1="64" y1="44" x2="300" y2="316" gradientUnits="userSpaceOnUse">
            <stop stopColor={p.goldLow} />
            <stop offset="0.44" stopColor={p.goldMid} />
            <stop offset="1" stopColor={p.goldDeep} />
          </linearGradient>
          <radialGradient id={ids.halo} cx="50%" cy="50%" r="50%">
            <stop stopColor={p.glow} stopOpacity={base.mode === 'paper' ? 0.34 : 0.5} />
            <stop offset="0.55" stopColor={p.glow} stopOpacity={base.mode === 'paper' ? 0.12 : 0.16} />
            <stop offset="1" stopColor={p.glow} stopOpacity="0" />
          </radialGradient>
          <radialGradient id={ids.dawn} cx="50%" cy="100%" r="78%">
            <stop stopColor={p.glow} stopOpacity="0.62" />
            <stop offset="0.52" stopColor={p.rose} stopOpacity="0.16" />
            <stop offset="1" stopColor={p.rose} stopOpacity="0" />
          </radialGradient>
          <linearGradient id={ids.slit} x1="0" y1="86" x2="0" y2="274" gradientUnits="userSpaceOnUse">
            <stop stopColor={p.goldLow} stopOpacity="0" />
            <stop offset="0.3" stopColor={p.goldLow} stopOpacity="0.62" />
            <stop offset="0.58" stopColor={p.glow} stopOpacity="0.44" />
            <stop offset="1" stopColor={p.glow} stopOpacity="0" />
          </linearGradient>
          <linearGradient id={ids.horizon} x1="20" y1="0" x2="340" y2="0" gradientUnits="userSpaceOnUse">
            <stop stopColor={p.frame} stopOpacity="0" />
            <stop offset="0.26" stopColor={p.frame} stopOpacity="0.35" />
            <stop offset="0.5" stopColor={p.frame} stopOpacity="0.72" />
            <stop offset="0.74" stopColor={p.frame} stopOpacity="0.35" />
            <stop offset="1" stopColor={p.frame} stopOpacity="0" />
          </linearGradient>
          <radialGradient id={ids.disc} cx="50%" cy="40%" r="62%">
            {base.mode === 'paper' ? (
              <>
                <stop stopColor="#FFFCF4" stopOpacity="0.8" />
                <stop offset="0.66" stopColor="#F8F0DF" stopOpacity="0.34" />
                <stop offset="1" stopColor="#F1E8D6" stopOpacity="0" />
              </>
            ) : (
              <>
                <stop stopColor="#171C24" />
                <stop offset="0.6" stopColor="#0D1015" />
                <stop offset="1" stopColor="#08090B" />
              </>
            )}
          </radialGradient>
          <linearGradient id={ids.fold} x1="0" y1="62" x2="0" y2="288" gradientUnits="userSpaceOnUse">
            <stop stopColor={p.goldMid} stopOpacity="0" />
            <stop offset="0.2" stopColor={p.goldMid} stopOpacity="1" />
            <stop offset="0.5" stopColor={p.goldLow} stopOpacity="1" />
            <stop offset="0.8" stopColor={p.goldMid} stopOpacity="1" />
            <stop offset="1" stopColor={p.goldMid} stopOpacity="0" />
          </linearGradient>
          <linearGradient id={ids.reflect} x1="0" y1={HORIZON} x2="0" y2={HORIZON + 46} gradientUnits="userSpaceOnUse">
            <stop stopColor={p.glow} stopOpacity="0.42" />
            <stop offset="0.55" stopColor={p.glow} stopOpacity="0.12" />
            <stop offset="1" stopColor={p.glow} stopOpacity="0" />
          </linearGradient>
          <radialGradient id={ids.shade} cx="50%" cy="44%" r="62%">
            <stop stopColor={base.mode === 'paper' ? '#E3D8C2' : '#070809'} />
            <stop offset="1" stopColor={base.mode === 'paper' ? '#D8CBB1' : '#040506'} />
          </radialGradient>
          <radialGradient id={ids.sun} cx="50%" cy="50%" r="50%">
            {base.mode === 'paper' ? (
              <>
                <stop stopColor="#FFFCF1" />
                <stop offset="0.32" stopColor="#FBE8C2" />
                <stop offset="0.72" stopColor="#DFAE6C" />
                <stop offset="1" stopColor="#B87F3F" />
              </>
            ) : (
              <>
                <stop stopColor="#FFFDF6" />
                <stop offset="0.42" stopColor="#FBEBCB" />
                <stop offset="1" stopColor="#C99A5C" />
              </>
            )}
          </radialGradient>
          <clipPath id={ids.discClip}>
            <circle cx={CX} cy={CY} r={DISC_R} />
          </clipPath>
          <mask id={ids.moon}>
            <rect x="0" y="0" width="360" height="360" fill="#000" />
            <circle cx={selene[0]} cy={selene[1]} r="8.6" fill="#fff" />
            <circle cx={selene[0] + 4.8} cy={selene[1] - 2.2} r="7.7" fill="#000" />
          </mask>
          <filter id={ids.soft} x="-40%" y="-40%" width="180%" height="180%">
            <feGaussianBlur stdDeviation="9" />
          </filter>
          <filter id={ids.softSm} x="-40%" y="-40%" width="180%" height="180%">
            <feGaussianBlur stdDeviation="2.4" />
          </filter>
        </defs>

        <Frame detail={state === 'night' ? 0.42 : 1} />

        {softLights.includes(state) && !(state === 'night' && base.mode === 'paper') && (
          <Disc shade={state === 'night' ? 0.94 : 0} />
        )}

        {state === 'welcome' && (
          <g>
            <Halo radius={142} opacity={0.38} />
            <Halo radius={80} opacity={0.5} />
            {/* the open ring: light that refuses to close around anyone */}
            <path d={arcPath(112, 200, 160)} stroke={p.gold} strokeWidth="10" strokeLinecap="round"
              opacity="0.16" filter={`url(#${ids.soft})`} />
            <path d={arcPath(112, 200, 160)} stroke={p.gold} strokeWidth="2" strokeLinecap="round" />
            {[200, 160].map((angle) => {
              const [x, y] = polar(112, angle);
              return <circle key={angle} cx={x} cy={y} r="1.6" fill={p.bright} opacity="0.8" />;
            })}
            <ellipse cx={CX} cy={HORIZON + 2} rx="62" ry="5" fill={p.dawn} opacity="0.3" />
            <ellipse cx={CX} cy={HORIZON} rx="26" ry="2" fill={p.bright} opacity="0.16" />
            <Flare length={66} opacity={0.16} />
            <Star radius={20} />
          </g>
        )}

        {state === 'night' && (
          <g>
            {/* the dark is not empty: the ring is still there, unlit */}
            <circle cx={CX} cy={CY} r="112" stroke={p.gold} strokeWidth="0.8" opacity="0.12" fill="none" />
            <path d={arcPath(112, 206, 154)} stroke={p.gold} strokeWidth="1.1" strokeLinecap="round" opacity="0.26" />
            {[[150, 132, 2.4, 0.55], [214, 154, 1.7, 0.32], [192, 112, 1.3, 0.26], [122, 186, 1.4, 0.2]].map(([x, y, r, o], index) => (
              <circle key={index} cx={x} cy={y} r={r} fill={p.star} opacity={o} />
            ))}
            <Halo x={168} y={198} radius={38} opacity={0.24} />
            <Star x={168} y={198} radius={4.4} opacity={0.42} glow={false} />
            <circle cx={168} cy={198} r="1.3" fill={p.bright} opacity="0.6" />
            <path d={`M118 ${HORIZON - 2}H242`} stroke={p.gold} strokeWidth="0.7" opacity="0.16" />
            <path d={`M146 ${HORIZON + 6}H214`} stroke={p.gold} strokeWidth="0.5" opacity="0.08" />
          </g>
        )}

        {state === 'veil' && (
          <g clipPath={`url(#${ids.discClip})`}>
            {/* light waits behind the curtain; the folds pass in front of it */}
            <Halo radius={82} opacity={0.34} />
            {[-1, 1].map((side) => (
              <g key={side} transform={`translate(${CX} 0) scale(${side} 1) translate(${-CX} 0)`}>
                <path
                  d={`M180 64C${180 - 34} 118 ${180 - 30} 232 180 288L64 288L64 64Z`}
                  fill={`url(#${ids.panel})`}
                />
                {[1, 2, 3, 4, 5].map((step) => {
                  const x = round(CX - (11 + step * 12.5));
                  const bow = round(4 + step * 3.2);
                  return (
                    <g key={step}>
                      <path
                        d={`M${x} 64Q${round(x - bow)} 174 ${x} 288`}
                        stroke={`url(#${ids.fold})`}
                        strokeWidth={step === 1 ? 0.9 : 0.55}
                        opacity={round(step === 1 ? 0.45 : 0.3 - step * 0.04)}
                        fill="none"
                      />
                      <path
                        d={`M${round(x - 2.4)} 70Q${round(x - bow - 2.4)} 174 ${round(x - 2.4)} 284`}
                        stroke={`url(#${ids.shadow})`}
                        strokeWidth="1.1"
                        opacity={round(0.16 - step * 0.022)}
                        fill="none"
                      />
                    </g>
                  );
                })}
                <path
                  d={`M180 64C${180 - 34} 118 ${180 - 30} 232 180 288`}
                  stroke={`url(#${ids.fold})`}
                  strokeWidth="1"
                  opacity="0.6"
                  fill="none"
                />
              </g>
            ))}
            {/* the chosen opening: a soft seam of light, not a spotlight */}
            <rect x={CX - 12} y={86} width="24" height="176" fill={p.slit} opacity="0.34" filter={`url(#${ids.soft})`} />
            <path d={`M180 126Q191 174 180 224Q169 174 180 126Z`} fill={p.slit} opacity="0.46" filter={`url(#${ids.softSm})`} />
            <Halo radius={34} opacity={0.42} />
            <Star radius={8.5} opacity={0.92} glow={false} />
          </g>
        )}

        {state === 'sight' && (
          <g>
            <Halo radius={86} opacity={0.3} />
            {/* the eye is only two hairlines, a lit pupil and its iris fibres */}
            <path d="M100 174Q180 112 260 174" stroke={p.rim} strokeWidth="0.95" opacity="0.46" fill="none" />
            <path d="M100 174Q180 216 260 174" stroke={p.rim} strokeWidth="0.65" opacity="0.26" fill="none" />
            <path d="M118 174Q180 126 242 174" stroke={p.rim} strokeWidth="0.4" opacity="0.12" fill="none" />
            <Halo radius={38} opacity={0.55} />
            <circle cx={CX} cy={CY} r="30" fill={p.halo} opacity="0.3" />
            <circle cx={CX} cy={CY} r="30" stroke={p.rim} strokeWidth="0.5" opacity="0.28" fill="none" />
            <circle cx={CX} cy={CY} r="20" stroke={p.rim} strokeWidth="0.4" opacity="0.2" fill="none" />
            <g opacity="0.22">
              {Array.from({ length: 24 }, (_, index) => {
                const angle = index * 15;
                const [x1, y1] = polar(9, angle);
                const [x2, y2] = polar(19, angle);
                return <line key={angle} x1={x1} y1={y1} x2={x2} y2={y2} stroke={p.rim} strokeWidth="0.4" />;
              })}
            </g>
            <circle cx={CX} cy={CY} r="6" fill={p.gold} />
            <circle cx={CX} cy={CY} r="2" fill={p.bright} />
            <g className="ti-sight-ring" style={{ transformOrigin: `${CX}px ${CY}px` }}>
              <circle cx={CX} cy={CY} r="40" stroke={p.rim} strokeWidth="0.5" opacity="0.22" fill="none" />
            </g>
            <Star x={CX - 11} y={CY - 11} radius={2.4} opacity={0.72} glow={false} />
          </g>
        )}

        {state === 'dawn' && (
          <g clipPath={`url(#${ids.discClip})`}>
            <ellipse cx={CX} cy={HORIZON} rx="128" ry="56" fill={p.halo} opacity="0.5" />
            {/* dawn does not arrive at once: three quiet bands, then the disc */}
            {[120, 94, 70].map((radius, index) => (
              <path
                key={radius}
                d={arcPath(radius, 263, 97, CX, HORIZON)}
                stroke={p.gold}
                strokeWidth={1.1 - index * 0.22}
                strokeLinecap="round"
                opacity={0.26 - index * 0.055}
              />
            ))}
            {Array.from({ length: 11 }, (_, index) => {
              const angle = 262 + index * 16;
              const [x1, y1] = polar(50, angle, CX, HORIZON);
              const [x2, y2] = polar(58 + (index % 3) * 9, angle, CX, HORIZON);
              return <line key={index} x1={x1} y1={y1} x2={x2} y2={y2} stroke={p.gold} strokeWidth="0.6" opacity={0.14 + (index % 3) * 0.04} />;
            })}
            <Halo x={CX} y={HORIZON} radius={92} opacity={0.55} />
            <path d={`M${CX - 44} ${HORIZON}A44 44 0 0 1 ${CX + 44} ${HORIZON}Z`} fill={`url(#${ids.sun})`} opacity="0.95" />
            <path d={arcPath(44, 270, 90, CX, HORIZON)} stroke={p.bright} strokeWidth="0.9" opacity="0.7" />
            <circle cx={CX} cy={HORIZON} r="2.4" fill={p.bright} opacity="0.9" />
            <path d={`M${CX - 32} 233H${CX + 26}`} stroke={p.goldDeep} strokeWidth="0.9" opacity="0.3" strokeLinecap="round" />
            <path d={`M${CX + 6} 244H${CX + 52}`} stroke={p.goldDeep} strokeWidth="0.7" opacity="0.22" strokeLinecap="round" />
            {/* what the horizon gives back */}
            <path d={`M${CX - 38} ${HORIZON}H${CX + 38}Q${CX + 18} ${HORIZON + 26} ${CX} ${HORIZON + 36}Q${CX - 18} ${HORIZON + 26} ${CX - 38} ${HORIZON}Z`} fill={`url(#${ids.reflect})`} opacity="0.8" />
            <path d={`M${CX - 26} ${HORIZON + 11}H${CX + 26}`} stroke={p.gold} strokeWidth="0.6" opacity="0.22" />
            <path d={`M${CX - 15} ${HORIZON + 21}H${CX + 15}`} stroke={p.gold} strokeWidth="0.55" opacity="0.14" />
            <Star x={264} y={112} radius={4.2} opacity={0.38} glow={false} />
          </g>
        )}

        {state === 'orbits' && (
          <g>
            {/* three paths, three lights: none of them is permanent */}
            {[ORBIT.helios, ORBIT.selene, ORBIT.eos].map((orbit, index) => (
              <g key={index}>
                <ellipse
                  cx={orbit.cx} cy={orbit.cy} rx={orbit.rx} ry={orbit.ry}
                  transform={`rotate(${orbit.rot} ${orbit.cx} ${orbit.cy})`}
                  stroke={p.rim}
                  strokeWidth="0.5"
                  opacity={0.26 - index * 0.055}
                  fill="none"
                />
                <ellipse
                  cx={orbit.cx} cy={orbit.cy} rx={orbit.rx} ry={orbit.ry}
                  transform={`rotate(${orbit.rot} ${orbit.cx} ${orbit.cy})`}
                  stroke={p.rim}
                  strokeWidth="0.4"
                  opacity={0.1 - index * 0.02}
                  strokeDasharray="1 7"
                  fill="none"
                />
              </g>
            ))}
            {[[ORBIT.helios, helios], [ORBIT.selene, selene], [ORBIT.eos, eos]].map(([orbit, point], index) => (
              <g key={`trail-${index}`}>
                {[9, 18, 27, 36, 45].map((back, step) => {
                  const [x, y] = ellipsePoint(orbit.cx, orbit.cy, orbit.rx, orbit.ry, orbit.rot, orbit.at - back);
                  return (
                    <circle key={back} cx={x} cy={y}
                      r={round(1.7 - step * 0.28)}
                      fill={p.star}
                      opacity={round(0.4 - step * 0.072)} />
                  );
                })}
              </g>
            ))}
            {/* Helios — the sun: a disc with a fine corona */}
            <g>
              <circle cx={helios[0]} cy={helios[1]} r="26" fill={p.halo} opacity="0.72" />
              {Array.from({ length: 16 }, (_, index) => {
                const angle = index * 22.5;
                const [x1, y1] = polar(10.5, angle, helios[0], helios[1]);
                const [x2, y2] = polar(index % 2 ? 13 : 15.5, angle, helios[0], helios[1]);
                return <line key={angle} x1={x1} y1={y1} x2={x2} y2={y2} stroke={p.gold} strokeWidth="0.55" opacity="0.4" />;
              })}
              <circle cx={helios[0]} cy={helios[1]} r="7.6" fill={`url(#${ids.sun})`} />
              <circle cx={helios[0]} cy={helios[1]} r="2.4" fill={base.mode === 'paper' ? '#FFF8E8' : p.bright} opacity="0.9" />
            </g>
            {/* Selene — the moon: only the light it borrows */}
            <g mask={`url(#${ids.moon})`}>
              <circle cx={selene[0]} cy={selene[1]} r="8.4" fill={p.gold} opacity="0.9" />
            </g>
            <circle cx={selene[0] + 2} cy={selene[1]} r="17" fill={p.halo} opacity="0.14" />
            {/* Eos — the dawn: a low pale seam where the sun comes back */}
            <g>
              <circle cx={eos[0]} cy={eos[1]} r="22" fill={p.halo} opacity="0.45" />
              <path d={`M${eos[0] - 16} ${eos[1]}H${eos[0] + 16}`} stroke={p.rose} strokeWidth="1.4" strokeLinecap="round" opacity="0.9" />
              <path d={`M${eos[0] - 7} ${eos[1]}H${eos[0] + 7}`} stroke={p.bright} strokeWidth="0.7" strokeLinecap="round" opacity="0.7" />
              <circle cx={eos[0]} cy={eos[1]} r="1.5" fill={p.bright} />
            </g>
            {/* the still centre that measures all three */}
            <path d={`M${CX} ${CY - 40}V${CY - 26}`} stroke={p.rim} strokeWidth="0.5" opacity="0.22" />
            <path d={`M${CX} ${CY + 26}V${CY + 40}`} stroke={p.rim} strokeWidth="0.5" opacity="0.22" />
            <path d={`M${CX - 40} ${CY}H${CX - 26}`} stroke={p.rim} strokeWidth="0.5" opacity="0.16" />
            <path d={`M${CX + 26} ${CY}H${CX + 40}`} stroke={p.rim} strokeWidth="0.5" opacity="0.16" />
            <circle cx={CX} cy={CY} r="2" stroke={p.rim} strokeWidth="0.5" opacity="0.28" fill="none" />
            <Star radius={6} opacity={0.75} glow={false} />
          </g>
        )}

        {state === 'pulse' && (
          <g>
            <Halo radius={92} opacity={0.42} />
            {/* a light that returns: the ring breathes outward and comes back */}
            {[130, 116].map((radius, index) => (
              <path
                key={radius}
                d={arcPath(radius, 202 + index * 5, 158 - index * 5)}
                stroke={p.gold}
                strokeWidth="0.9"
                strokeLinecap="round"
                opacity={0.16 - index * 0.045}
              />
            ))}
            <path d={arcPath(100, 198, 162)} stroke={p.gold} strokeWidth="10" strokeLinecap="round"
              opacity="0.14" filter={`url(#${ids.soft})`} />
            <path d={arcPath(100, 198, 162)} stroke={p.gold} strokeWidth="1.8" strokeLinecap="round" />
            <circle cx={CX} cy={CY} r="100" stroke={p.gold} strokeWidth="1" opacity="0.42" fill="none"
              className="ti-pulse" style={{ transformOrigin: `${CX}px ${CY}px` }} />
            <circle cx={CX} cy={CY} r="100" stroke={p.gold} strokeWidth="0.9" opacity="0.22" fill="none"
              className="ti-pulse ti-pulse--late" style={{ transformOrigin: `${CX}px ${CY}px` }} />
            <Star radius={13} />
          </g>
        )}

        {state === 'seal' && (
          <g>
            <circle cx={CX} cy={CY} r="132" stroke={p.gold} strokeWidth="1.2" opacity="0.66" />
            <circle cx={CX} cy={CY} r="126" stroke={p.gold} strokeWidth="0.6" opacity="0.32" />
            <g className="ti-rays" style={{ transformBox: 'view-box', transformOrigin: `${CX}px ${CY}px` }}>
              {Array.from({ length: 24 }, (_, index) => {
                const angle = index * 15;
                const long = index % 3 === 0;
                const [x1, y1] = polar(long ? 136 : 139, angle);
                const [x2, y2] = polar(long ? 146 : 143.5, angle);
                return <line key={angle} x1={x1} y1={y1} x2={x2} y2={y2} stroke={p.frame} strokeWidth="0.7" opacity={long ? 0.5 : 0.26} />;
              })}
            </g>
            <Halo radius={104} opacity={0.4} />
            <circle cx={CX} cy={CY} r="104" stroke={p.gold} strokeWidth="1.7" fill="none" opacity="0.9" />
            <circle cx={CX} cy={CY} r="98" stroke={p.rim} strokeWidth="0.45" fill="none" opacity="0.2" />
            <Halo radius={66} opacity={0.55} />
            <Flare length={92} opacity={0.26} />
            <Star radius={23} />
            <Star radius={13} rotate={45} opacity={0.92} glow={false} />
          </g>
        )}
      </svg>
    </ToneContext.Provider>
  );
}
