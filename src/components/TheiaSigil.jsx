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
/* ------------------------------------------------------------------ */

const CX = 180;
const CY = 180;

/** Polar coordinates; 0° is north, angles increase clockwise (matching SVG). */
function polar(radius, angle, cx = CX, cy = CY) {
  const theta = ((angle - 90) * Math.PI) / 180;
  return [cx + radius * Math.cos(theta), cy + radius * Math.sin(theta)];
}

function arcPath(radius, from, to) {
  const [x1, y1] = polar(radius, from);
  const [x2, y2] = polar(radius, to);
  const sweep = (to - from + 360) % 360;
  const large = sweep > 180 ? 1 : 0;
  return `M${x1.toFixed(2)} ${y1.toFixed(2)}A${radius} ${radius} 0 ${large} 1 ${x2.toFixed(2)} ${y2.toFixed(2)}`;
}

const STAR = (cx, cy, r) => {
  const k = r * 0.09;
  return `M${cx} ${cy - r}Q${cx + k} ${cy - k} ${cx + r} ${cy}Q${cx + k} ${cy + k} ${cx} ${cy + r}Q${cx - k} ${cy + k} ${cx - r} ${cy}Q${cx - k} ${cy - k} ${cx} ${cy - r}Z`;
};

/**
 * Two tonal readings of the same instrument: night (on ink surfaces) and an
 * ink etching (on ivory paper). Both are the same geometry, so the story keeps
 * one object while the light changes around it.
 */
const TONES = {
  ink: {
    frame: '#EDE0C8',
    frameSoft: '#E4D3B4',
    disc: 'url(#ti-disc)',
    rim: '#EBDCC0',
    gold: 'url(#ti-gold)',
    goldFlat: '#EBD8B2',
    bright: '#FBEED4',
    horizon: 'url(#ti-horizon)',
    halo: 'url(#ti-halo)',
    slit: 'url(#ti-slit-soft)',
    glowFilter: 'url(#ti-soft-sm)',
    boost: 1,
  },
  paper: {
    frame: '#26231C',
    frameSoft: '#332F27',
    disc: 'url(#ti-disc-paper)',
    rim: '#8D8474',
    gold: 'url(#ti-gold-paper)',
    goldFlat: '#A9783F',
    bright: '#8E5F2C',
    horizon: 'url(#ti-horizon-paper)',
    halo: 'url(#ti-halo-paper)',
    slit: 'url(#ti-slit-paper)',
    glowFilter: 'url(#ti-soft-sm)',
    boost: 1.9,
  },
};

const ToneContext = React.createContext(TONES.ink);
const useTone = () => React.useContext(ToneContext);

function Frame({ detail = 1 }) {
  const tone = useTone();
  const boost = tone.boost || 1;
  return (
    <g className="ti-frame">
      <circle cx={CX} cy={CY} r="132" stroke={tone.frame} strokeWidth="0.6" opacity={0.16 * detail * boost} />
      <circle cx={CX} cy={CY} r="126" stroke={tone.frame} strokeWidth="0.6" opacity={0.07 * detail * boost} />
      <g className="ti-ticks" opacity={0.32 * detail * boost}>
        {Array.from({ length: 60 }, (_, i) => {
          const angle = i * 6;
          const long = angle % 30 === 0;
          const [x1, y1] = polar(long ? 126 : 129, angle);
          const [x2, y2] = polar(132, angle);
          return (
            <line
              key={angle}
              x1={x1.toFixed(2)} y1={y1.toFixed(2)} x2={x2.toFixed(2)} y2={y2.toFixed(2)}
              stroke={tone.frame} strokeWidth="0.6" opacity={long ? 0.55 : 0.28}
            />
          );
        })}
      </g>
      <circle cx={CX} cy={CY} r="104" stroke={tone.frameSoft} strokeWidth="0.7" strokeDasharray="1 7" opacity={0.5 * detail * boost} className="ti-drift" />
      <circle cx={CX} cy={CY} r="96" stroke={tone.frameSoft} strokeWidth="0.5" opacity={0.16 * detail * boost} />
      <path d="M40 252H320" stroke={tone.horizon} strokeWidth="0.8" opacity={0.55 * detail} />
      <path d="M40 258H320" stroke={tone.frameSoft} strokeWidth="0.4" opacity={0.12 * detail} />
    </g>
  );
}

function Star({ radius = 15, opacity = 1, glow = true }) {
  const tone = useTone();
  return (
    <g opacity={opacity}>
      {glow && <circle cx={CX} cy={CY} r={radius * 3.4} fill={tone.halo} opacity="0.5" />}
      <path d={STAR(CX, CY, radius)} fill={tone.gold} />
    </g>
  );
}

function Seam({ from, to, radius = 96, width = 1.8, opacity = 1 }) {
  const tone = useTone();
  return (
    <g opacity={opacity}>
      <path d={arcPath(radius, from, to)} stroke={tone.gold} strokeWidth={width * 3.4} strokeLinecap="round" opacity="0.22" filter="url(#ti-soft)" />
      <path d={arcPath(radius, from, to)} stroke={tone.gold} strokeWidth={width} strokeLinecap="round" />
    </g>
  );
}

function Disc({ opacity = 1 }) {
  const tone = useTone();
  return (
    <>
      <circle cx={CX} cy={CY} r="92" fill={tone.disc} opacity={opacity} />
      <circle cx={CX} cy={CY} r="92" stroke={tone.rim} strokeWidth="0.55" opacity="0.18" />
    </>
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
  const palette = TONES[tone] || TONES.ink;
  return (
    <ToneContext.Provider value={palette}>
      <svg
        className={`theia-instrument theia-instrument--${state} theia-instrument--tone-${tone} ${className}`.trim()}
        viewBox="0 0 360 360"
        fill="none"
        role="img"
        aria-label={label || STATE_LABEL[state] || STATE_LABEL.welcome}
        focusable="false"
      >
        <defs>
          <radialGradient id="ti-disc" cx="50%" cy="42%" r="62%">
            <stop stopColor="#181C23" />
            <stop offset="0.62" stopColor="#0D0F13" />
            <stop offset="1" stopColor="#090A0C" />
          </radialGradient>
          <radialGradient id="ti-disc-paper" cx="50%" cy="38%" r="64%">
            <stop stopColor="#FFFCF4" />
            <stop offset="0.68" stopColor="#F7F0E3" />
            <stop offset="1" stopColor="#EFE6D5" />
          </radialGradient>
          <linearGradient id="ti-gold" x1="70" y1="60" x2="300" y2="312" gradientUnits="userSpaceOnUse">
            <stop stopColor="#F8EBCE" />
            <stop offset="0.42" stopColor="#E3C48C" />
            <stop offset="1" stopColor="#B0824A" />
          </linearGradient>
          <linearGradient id="ti-gold-paper" x1="70" y1="60" x2="300" y2="312" gradientUnits="userSpaceOnUse">
            <stop stopColor="#C79A5C" />
            <stop offset="0.46" stopColor="#AC7B40" />
            <stop offset="1" stopColor="#8A5C2C" />
          </linearGradient>
          <linearGradient id="ti-horizon" x1="40" y1="0" x2="320" y2="0" gradientUnits="userSpaceOnUse">
            <stop stopColor="#E4D3B4" stopOpacity="0" />
            <stop offset="0.5" stopColor="#E4D3B4" stopOpacity="0.9" />
            <stop offset="1" stopColor="#E4D3B4" stopOpacity="0" />
          </linearGradient>
          <linearGradient id="ti-horizon-paper" x1="40" y1="0" x2="320" y2="0" gradientUnits="userSpaceOnUse">
            <stop stopColor="#3A352C" stopOpacity="0" />
            <stop offset="0.5" stopColor="#3A352C" stopOpacity="0.5" />
            <stop offset="1" stopColor="#3A352C" stopOpacity="0" />
          </linearGradient>
          <radialGradient id="ti-halo" cx="50%" cy="50%" r="50%">
            <stop stopColor="#F3D8A3" stopOpacity="0.55" />
            <stop offset="1" stopColor="#F3D8A3" stopOpacity="0" />
          </radialGradient>
          <radialGradient id="ti-halo-paper" cx="50%" cy="50%" r="50%">
            <stop stopColor="#E7C089" stopOpacity="0.45" />
            <stop offset="1" stopColor="#E7C089" stopOpacity="0" />
          </radialGradient>
          <radialGradient id="ti-dawn" cx="50%" cy="100%" r="78%">
            <stop stopColor="#F2C98D" stopOpacity="0.55" />
            <stop offset="0.55" stopColor="#C98D5A" stopOpacity="0.14" />
            <stop offset="1" stopColor="#0A0B0D" stopOpacity="0" />
          </radialGradient>
          <linearGradient id="ti-slit-soft" x1="0" y1="86" x2="0" y2="274" gradientUnits="userSpaceOnUse">
            <stop stopColor="#F8EBCE" stopOpacity="0" />
            <stop offset="0.32" stopColor="#F6E2BA" stopOpacity="0.85" />
            <stop offset="0.62" stopColor="#F3D8A3" stopOpacity="0.6" />
            <stop offset="1" stopColor="#F3D8A3" stopOpacity="0" />
          </linearGradient>
          <linearGradient id="ti-slit-paper" x1="0" y1="86" x2="0" y2="274" gradientUnits="userSpaceOnUse">
            <stop stopColor="#D9B782" stopOpacity="0" />
            <stop offset="0.34" stopColor="#D9B782" stopOpacity="0.55" />
            <stop offset="0.64" stopColor="#C79A5C" stopOpacity="0.3" />
            <stop offset="1" stopColor="#C79A5C" stopOpacity="0" />
          </linearGradient>
          <clipPath id="ti-disc-clip">
            <circle cx={CX} cy={CY} r="92" />
          </clipPath>
          <linearGradient id="ti-fold-fade" x1="0" y1="88" x2="0" y2="272" gradientUnits="userSpaceOnUse">
            <stop stopColor="#fff" stopOpacity="0" />
            <stop offset="0.28" stopColor="#fff" stopOpacity="1" />
            <stop offset="0.72" stopColor="#fff" stopOpacity="1" />
            <stop offset="1" stopColor="#fff" stopOpacity="0" />
          </linearGradient>
          <mask id="ti-fold-fade">
            <rect x="80" y="80" width="200" height="200" fill="url(#ti-fold-fade)" />
          </mask>
          <mask id="ti-moon-mask">
            <rect x="0" y="0" width="360" height="360" fill="#000" />
            <circle cx={polar(74, 296)[0]} cy={polar(74, 296)[1]} r="8.4" fill="#fff" />
            <circle cx={polar(74, 296)[0] + 4.2} cy={polar(74, 296)[1] - 2} r="7.6" fill="#000" />
          </mask>
          <filter id="ti-soft" x="-30%" y="-30%" width="160%" height="160%">
            <feGaussianBlur stdDeviation="7" />
          </filter>
          <filter id="ti-soft-sm" x="-30%" y="-30%" width="160%" height="160%">
            <feGaussianBlur stdDeviation="2.2" />
          </filter>
        </defs>

        <Frame detail={state === 'night' ? 0.5 : 1} />

        {['welcome', 'night', 'veil', 'sight', 'dawn', 'pulse', 'seal'].includes(state) && (
          <Disc opacity={state === 'night' ? 0.94 : 1} />
        )}

        {state === 'welcome' && (
          <g>
            <circle cx={CX} cy={CY} r="140" fill={palette.halo} opacity="0.16" />
            <Seam from={142} to={212} radius={96} width={2} />
            <circle cx={CX} cy={CY} r="58" fill={palette.halo} opacity="0.35" />
            <Star radius={19} />
          </g>
        )}

        {state === 'night' && (
          <g>
            <Seam from={200} to={238} radius={96} width={1.1} opacity={0.35} />
            <circle cx={CX} cy={CY} r="2.4" fill={palette.goldFlat} opacity="0.5" filter={palette.glowFilter} />
            <Star radius={5.4} opacity={0.33} glow={false} />
          </g>
        )}

        {state === 'veil' && (
          <g clipPath="url(#ti-disc-clip)">
            {/* A curtain parted at the middle: the opening is chosen, not taken. */}
            <g mask="url(#ti-fold-fade)">
              {Array.from({ length: 7 }, (_, i) => {
                const offset = 16 + i * 17;
                return [-1, 1].map((side) => (
                  <path
                    key={`${side}-${i}`}
                    d={`M${CX + side * offset} 70 Q${CX + side * (offset + 10)} 180 ${CX + side * offset} 290`}
                    stroke={palette.rim}
                    strokeWidth={i % 2 ? 0.7 : 1}
                    opacity={0.2 - i * 0.014}
                  />
                ));
              })}
            </g>
            <path d="M180 86 Q199 180 180 274 Q161 180 180 86Z" fill={palette.slit} filter="url(#ti-soft-sm)" />
            <path d="M180 116 Q188 180 180 244 Q172 180 180 116Z" fill={palette.goldFlat} opacity="0.3" />
            <Star radius={13} opacity={0.9} glow={false} />
          </g>
        )}

        {state === 'sight' && (
          <g>
            <path d="M112 180Q180 142 248 180Q180 218 112 180Z" stroke={palette.rim} strokeWidth="0.9" opacity="0.45" fill="none" />
            <path d="M126 180Q180 154 234 180" stroke={palette.rim} strokeWidth="0.6" opacity="0.2" fill="none" />
            <circle cx={CX} cy={CY} r="26" fill={palette.halo} opacity="0.55" />
            <circle cx={CX} cy={CY} r="6.4" fill={palette.gold} />
            <g className="ti-sight-ring">
              <circle cx={CX} cy={CY} r="17" stroke={palette.rim} strokeWidth="0.6" opacity="0.3" fill="none" />
            </g>
            <Star radius={4} opacity={0.9} glow={false} />
          </g>
        )}

        {state === 'dawn' && (
          <g clipPath="url(#ti-disc-clip)">
            <ellipse cx={CX} cy="250" rx="126" ry="62" fill={palette.halo} opacity="0.75" />
            <ellipse cx={CX} cy="246" rx="86" ry="34" fill="url(#ti-dawn)" opacity="0.65" />
            <Seam from={132} to={318} radius={92} width={1.5} opacity={0.9} />
            <circle cx={CX} cy={236} r="30" fill={palette.halo} opacity="0.9" />
            <path d={`M${CX - 30} 238H${CX + 30}`} stroke={palette.bright} strokeWidth="1.4" strokeLinecap="round" opacity="0.8" />
            <circle cx={CX} cy={238} r="2.6" fill={palette.bright} filter={palette.glowFilter} />
            <Star radius={11} opacity={0.9} />
          </g>
        )}

        {state === 'orbits' && (
          <g>
            <circle cx={CX} cy={CY} r="50" stroke={palette.rim} strokeWidth="0.5" opacity="0.2" />
            <circle cx={CX} cy={CY} r="74" stroke={palette.rim} strokeWidth="0.5" opacity="0.16" />
            <circle cx={CX} cy={CY} r="104" stroke={palette.rim} strokeWidth="0.5" opacity="0.12" />
            {/* Helios — the sun: a small disc with a fine corona. */}
            <g>
              <circle cx={polar(104, 34)[0]} cy={polar(104, 34)[1]} r="19" fill={palette.halo} />
              <circle cx={polar(104, 34)[0]} cy={polar(104, 34)[1]} r="6.6" fill={palette.gold} />
            </g>
            {/* Selene — the moon: a crescent, the light it borrows. */}
            <g mask="url(#ti-moon-mask)">
              <circle cx={polar(74, 296)[0]} cy={polar(74, 296)[1]} r="8.4" fill={palette.goldFlat} opacity="0.9" />
            </g>
            {/* Eos — the dawn: a low pale seam where the sun comes back. */}
            <g>
              <circle cx={polar(60, 152)[0]} cy={polar(60, 152)[1]} r="16" fill={palette.halo} opacity="0.6" />
              <path d={`M${polar(60, 152)[0] - 13} ${polar(60, 152)[1]}H${polar(60, 152)[0] + 13}`} stroke="#E7B6A0" strokeWidth="1.5" strokeLinecap="round" opacity="0.9" />
            </g>
            <Star radius={8} opacity={0.75} glow={false} />
          </g>
        )}

        {state === 'pulse' && (
          <g>
            <Seam from={150} to={250} radius={96} width={2} />
            <circle cx={CX} cy={CY} r="96" stroke={palette.goldFlat} strokeWidth="1" fill="none" opacity="0.55" className="ti-pulse" />
            <circle cx={CX} cy={CY} r="96" stroke={palette.goldFlat} strokeWidth="1" fill="none" opacity="0.3" className="ti-pulse ti-pulse--late" />
            <Star radius={14} />
          </g>
        )}

        {state === 'seal' && (
          <g>
            <circle cx={CX} cy={CY} r="132" stroke={palette.gold} strokeWidth="1.4" opacity="0.7" />
            <g className="ti-rays">
              {Array.from({ length: 16 }, (_, i) => {
                const [x1, y1] = polar(136, i * 22.5);
                const [x2, y2] = polar(146, i * 22.5);
                return <line key={i} x1={x1.toFixed(2)} y1={y1.toFixed(2)} x2={x2.toFixed(2)} y2={y2.toFixed(2)} stroke={palette.rim} strokeWidth="0.8" opacity="0.4" />;
              })}
            </g>
            <circle cx={CX} cy={CY} r="96" stroke={palette.gold} strokeWidth="2" fill="none" opacity="0.9" />
            <circle cx={CX} cy={CY} r="64" fill={palette.halo} opacity="0.5" />
            <Star radius={23} />
          </g>
        )}
      </svg>
    </ToneContext.Provider>
  );
}
