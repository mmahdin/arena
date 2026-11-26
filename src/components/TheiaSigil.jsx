import React from 'react';

/**
 * Theia's identity is intentionally faceless.
 *
 * In the myth Theia (Θεία, "the far-shining") is the Titaness of sight and of
 * the light of the sky. She was never given a cult image, a temple or a statue:
 * what survives of her is light itself. Her children are Helios, Selene and Eos
 * — sun, moon and dawn. The brand therefore shows an instrument of light, never
 * a body: an astrolabe left deliberately open, one spark of dawn inside it, and
 * the quiet horizon it measures. The open ring is the product promise: the app
 * does not close around you or complete you; it lets light in and leaves the
 * circle open for the person using it.
 *
 * The artwork is drawn as a layered instrument rather than an icon: a plate with
 * an engraved degree ring, a rete lattice, a horizon chord, dust in the night air
 * and light that behaves like real dawn light. Every state is hand-authored SVG
 * so it stays sharp, re-tones itself for ink or paper, and moves slowly enough to
 * feel alive without asking for attention.
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
          <stop stopColor="#F9EDD3" />
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

const C = 200; // centre of the plate
const PLATE = 158;
const DISC = 104;
const HORIZON = 252;

/** Polar coordinates; 0° is north, angles increase clockwise (matching SVG). */
function polar(radius, angle, cx = C, cy = C) {
  const theta = ((angle - 90) * Math.PI) / 180;
  return [cx + radius * Math.cos(theta), cy + radius * Math.sin(theta)];
}

const px = (n) => Number(n.toFixed(2));

function arcPath(radius, from, to) {
  const [x1, y1] = polar(radius, from);
  const [x2, y2] = polar(radius, to);
  const sweep = (to - from + 360) % 360;
  const large = sweep > 180 ? 1 : 0;
  return `M${px(x1)} ${px(y1)}A${radius} ${radius} 0 ${large} 1 ${px(x2)} ${px(y2)}`;
}

const STAR = (cx, cy, r) => {
  const k = r * 0.09;
  return `M${cx} ${cy - r}Q${cx + k} ${cy - k} ${cx + r} ${cy}Q${cx + k} ${cy + k} ${cx} ${cy + r}Q${cx - k} ${cy + k} ${cx - r} ${cy}Q${cx - k} ${cy - k} ${cx} ${cy - r}Z`;
};

/**
 * Deterministic dust: the same night sky every render, so nothing flickers
 * between screens and snapshots stay stable.
 */
const DUST = (() => {
  let seed = 20261126;
  const random = () => {
    seed = (seed * 1103515245 + 12345) % 2147483648;
    return seed / 2147483648;
  };
  return Array.from({ length: 54 }, () => {
    const angle = random() * Math.PI * 2;
    // Biased to the upper half of the sky, thinning out near the horizon.
    const radius = 30 + Math.pow(random(), 0.62) * 190;
    const x = C + Math.cos(angle) * radius;
    const y = C + Math.sin(angle) * radius * 0.92;
    return {
      x: px(x),
      y: px(y),
      r: Number((0.35 + random() * 0.85).toFixed(2)),
      o: Number((0.06 + random() * 0.4).toFixed(2)),
      warm: random() > 0.72,
      drift: Number((26 + random() * 40).toFixed(1)),
      delay: Number((random() * 18).toFixed(1)),
    };
  });
})();

const TONES = {
  ink: {
    atmosphere: 'url(#ti-sky-ink)',
    vignette: 'url(#ti-vignette-ink)',
    frame: '#EBDCC0',
    frameSoft: '#D9C7A6',
    plate: '#0C0E12',
    disc: 'url(#ti-disc-ink)',
    rim: '#F2E3C6',
    gold: 'url(#ti-gold)',
    goldFlat: '#E7CD9C',
    bright: '#FFF6E3',
    horizon: 'url(#ti-horizon-ink)',
    halo: 'url(#ti-halo)',
    slit: 'url(#ti-slit-soft)',
    dawnBand: 'url(#ti-dawn-ink)',
    ray: '#F6E3BC',
    sunBody: '#FFF4DA',
    sunRim: '#E9C68F',
    dust: '#F4E4C4',
    boost: 1,
    grain: 0.055,
    discShadow: 0,
  },
  paper: {
    atmosphere: 'url(#ti-sky-paper)',
    vignette: 'url(#ti-vignette-paper)',
    frame: '#2A261D',
    frameSoft: '#4A4234',
    plate: '#F7F1E5',
    disc: 'url(#ti-disc-paper)',
    rim: '#8A7D66',
    gold: 'url(#ti-gold-paper)',
    goldFlat: '#A9783F',
    bright: '#8E5F2C',
    horizon: 'url(#ti-horizon-paper)',
    halo: 'url(#ti-halo-paper)',
    slit: 'url(#ti-slit-paper)',
    dawnBand: 'url(#ti-dawn-paper)',
    ray: '#B4884C',
    sunBody: '#FFF7E6',
    sunRim: '#C08C4E',
    dust: '#3B3428',
    boost: 1.7,
    grain: 0.05,
    discShadow: 0.35,
  },
};

const ToneContext = React.createContext(TONES.ink);
const useTone = () => React.useContext(ToneContext);

/* ------------------------------------------------------------------ */
/* Layers                                                             */
/* ------------------------------------------------------------------ */

/** Night air, horizon warmth, dust and a soft vignette behind the plate. */
function Atmosphere({ calm = 1 }) {
  const tone = useTone();
  const weight = tone.goldFlat === '#A9783F' ? 0.42 : 1;
  return (
    <g className="ti-atmosphere" aria-hidden="true">
      <circle cx={C} cy={C} r="200" fill={tone.atmosphere} />
      <rect x="0" y="0" width="400" height="400" fill={tone.vignette} />
      <g className="ti-dust" fill={tone.dust}>
        {DUST.map((star, index) => (
          <circle
            key={index}
            cx={star.x}
            cy={star.y}
            r={star.r}
            opacity={star.o * calm * weight * (star.warm ? 1.15 : 0.9)}
            className="ti-dust__grain"
            style={{ '--drift': `${star.drift}s`, '--delay': `-${star.delay}s` }}
          />
        ))}
      </g>
    </g>
  );
}

/** The astrolabe: degree ring, rete lattice, horizon chord, index marks. */
function Plate({ detail = 1, showRete = true }) {
  const tone = useTone();
  const boost = tone.boost;
  const opacity = (value) => Math.min(1, value * detail * boost);

  return (
    <g className="ti-plate">
      {/* Rim and its shadow, as if the plate sits on the page. */}
      <circle cx={C} cy={C} r={PLATE - 0.5} fill={tone.plate} opacity={0.55 * detail} />
      <circle cx={C} cy={C + 2} r={PLATE + 1.5} fill={tone.frame} opacity={0.035 * detail} />
      <circle cx={C} cy={C} r={PLATE} stroke={tone.frame} strokeWidth="0.9" opacity={opacity(0.2)} />
      <circle cx={C} cy={C} r={PLATE - 1.5} stroke={tone.frame} strokeWidth="0.5" opacity={opacity(0.09)} />

      {/* Engraved degree scale. */}
      <g className="ti-scale" opacity={opacity(0.3)}>
        {Array.from({ length: 72 }, (_, i) => {
          const angle = i * 5;
          const major = angle % 30 === 0;
          const medium = angle % 10 === 0;
          const [x1, y1] = polar(PLATE - 1, angle);
          const [x2, y2] = polar(PLATE - (major ? 11 : medium ? 7 : 4.5), angle);
          return (
            <line
              key={angle}
              x1={px(x1)} y1={px(y1)} x2={px(x2)} y2={px(y2)}
              stroke={tone.frame}
              strokeWidth={major ? 0.85 : 0.5}
              opacity={major ? 0.75 : medium ? 0.5 : 0.3}
            />
          );
        })}
      </g>

      {/* Index marks at the four cardinal points. */}
      <g opacity={opacity(0.32)}>
        {[0, 90, 180, 270].map((angle) => {
          const [x1, y1] = polar(PLATE + 2, angle);
          const [x2, y2] = polar(PLATE + 10, angle);
          return <line key={angle} x1={px(x1)} y1={px(y1)} x2={px(x2)} y2={px(y2)} stroke={tone.frame} strokeWidth="0.9" strokeLinecap="round" />;
        })}
      </g>

      {showRete && (
        <g className="ti-rete" opacity={opacity(0.42)}>
          {/* The rete: two long arcs and the ecliptic, with star pointers. */}
          <g clipPath="url(#ti-plate-clip)">
            <circle cx={C - 66} cy={C - 26} r="196" stroke={tone.frameSoft} strokeWidth="0.7" opacity="0.42" />
            <circle cx={C + 66} cy={C + 26} r="196" stroke={tone.frameSoft} strokeWidth="0.7" opacity="0.42" />
            <circle cx={C + 18} cy={C + 92} r="212" stroke={tone.frameSoft} strokeWidth="0.55" opacity="0.3" />
            <circle cx={C + 18} cy={C + 92} r="188" stroke={tone.frameSoft} strokeWidth="0.55" opacity="0.22" />
          </g>
          {[
            [286, 96], [114, 96], [286, 304], [114, 304],
          ].map(([x, y], index) => (
            <g key={index} opacity="0.75">
              <line x1={x} y1={y} x2={x} y2={y + 9} stroke={tone.frameSoft} strokeWidth="0.5" opacity="0.6" />
              <circle cx={x} cy={y + 12} r="1.5" fill={tone.goldFlat} opacity="0.9" />
            </g>
          ))}
        </g>
      )}

      <g className="ti-drift-slow">
        <circle cx={C} cy={C} r={DISC + 24} stroke={tone.frameSoft} strokeWidth="0.6" strokeDasharray="0.5 6" opacity={opacity(0.32)} />
      </g>

      {/* Horizon chord and its faint echo. */}
      <path d={`M33 ${HORIZON}H367`} stroke={tone.horizon} strokeWidth="0.9" opacity={opacity(0.5)} />
      <path d={`M${C - 118} ${HORIZON + 7}H${C + 118}`} stroke={tone.frameSoft} strokeWidth="0.45" opacity={opacity(0.16)} />
    </g>
  );
}

/** The inner plate: brushed night metal, engraved rings, a soft rim light. */
function Disc({ opacity = 1, lit = 0, light = 1, open = 0 }) {
  const tone = useTone();
  return (
    <g className="ti-disc" opacity={opacity}>
      {tone.discShadow > 0 && (
        <ellipse cx={C} cy={C + 6} rx={DISC + 2} ry={DISC + 2} fill="#6B5B41" opacity={tone.discShadow * 0.16} filter="url(#ti-soft)" />
      )}
      <circle cx={C} cy={C} r={DISC} fill={tone.disc} opacity="0.98" />
      {open > 0
        ? <path d={arcPath(DISC, 138 + open, 138 - open + 360)} stroke={tone.rim} strokeWidth="0.7" opacity="0.98" />
        : <circle cx={C} cy={C} r={DISC} stroke={tone.rim} strokeWidth="0.7" opacity="0.98" />}
      <circle cx={C} cy={C} r={DISC} fill="url(#ti-sheen)" opacity="0.38" />
      <g clipPath="url(#ti-disc-clip)">
        <circle cx={C} cy={C} r={DISC - 8} stroke={tone.rim} strokeWidth="0.4" opacity="0.1" />
        <circle cx={C} cy={C} r={DISC - 20} stroke={tone.rim} strokeWidth="0.4" opacity="0.07" />
        <rect x={C - DISC} y={C - DISC} width={DISC * 2} height={DISC * 2} filter="url(#ti-grain)" opacity={tone.grain} />
        {lit > 0 && <circle cx={C} cy={C + DISC * 0.55} r={DISC} fill={tone.dawnBand} opacity={lit} />}
      </g>
      {/* Rim light: dawn catches the plate on one side only. */}
      <path d={arcPath(DISC - 0.6, open > 0 ? 138 + open + 5 : 96, 246)} stroke={tone.rim} strokeWidth="1.1" strokeLinecap="round" opacity={0.2 * light} />
      <path d={arcPath(DISC - 0.6, 250, 322)} stroke={tone.goldFlat} strokeWidth="0.9" strokeLinecap="round" opacity={0.16 * light} />
    </g>
  );
}

function Star({ radius = 15, opacity = 1, glow = true, bloom = 1 }) {
  const tone = useTone();
  return (
    <g opacity={opacity}>
      {glow && <circle cx={C} cy={C} r={radius * 5.4} fill={tone.halo} opacity={0.34 * bloom} />}
      {glow && <circle cx={C} cy={C} r={radius * 2.5} fill={tone.halo} opacity={0.5 * bloom} />}
      <path d={STAR(C, C, radius + 1.6)} fill={tone.halo} opacity={0.55} />
      <path d={STAR(C, C, radius)} fill={tone.gold} />
      <path d={STAR(C, C, radius * 0.38)} fill={tone.bright} opacity="0.85" />
    </g>
  );
}

function Seam({ from, to, radius = DISC, width = 1.8, opacity = 1, spark = false }) {
  const tone = useTone();
  const [ex, ey] = polar(radius, to);
  // Light blooms on night, but on paper the same seam would smudge: keep it a
  // drawn hairline with only a breath of warmth around it.
  const k = tone.boost > 1 ? 0.6 : 1;
  return (
    <g opacity={opacity}>
      <path d={arcPath(radius, from, to)} stroke={tone.gold} strokeWidth={width * 5} strokeLinecap="round" opacity={0.18 * k} filter="url(#ti-soft)" />
      <path d={arcPath(radius, from, to)} stroke={tone.gold} strokeWidth={width * 2} strokeLinecap="round" opacity={0.35 * k} />
      <path d={arcPath(radius, from, to)} stroke={tone.gold} strokeWidth={width * k} strokeLinecap="round" />
      {spark && (
        <>
          <circle cx={px(ex)} cy={px(ey)} r="12" fill={tone.halo} opacity="0.55" />
          <circle cx={px(ex)} cy={px(ey)} r="2.2" fill={tone.bright} />
        </>
      )}
    </g>
  );
}

function Rays({ count = 14, from = DISC + 12, to = DISC + 40, opacity = 0.35, rotation = 0 }) {
  const tone = useTone();
  return (
    <g opacity={opacity}>
      {Array.from({ length: count }, (_, i) => {
        const angle = rotation + (360 / count) * i;
        const [x1, y1] = polar(from, angle);
        const [x2, y2] = polar(to, angle);
        return <line key={i} x1={px(x1)} y1={px(y1)} x2={px(x2)} y2={px(y2)} stroke={tone.ray} strokeWidth="0.7" strokeLinecap="round" />;
      })}
    </g>
  );
}

const STATE_LABEL = {
  welcome: 'حلقهٔ باز تیا: صفحهٔ اسطرلاب با کمانی از نور سپیده و ستاره‌ای در مرکز',
  night: 'شب: صفحهٔ تاریک است و تنها اخگری در مرکزش مانده',
  veil: 'پرده: پرده‌ای که در میانه باز شده و شکافی از نور را نشان می‌دهد',
  sight: 'دیدن: چشمی از خطوط نازک در دل صفحه، با مردمکی از نور',
  dawn: 'سپیده: خورشید نیمه‌برآمده در افق و کمانی که کم‌کم روشن می‌شود',
  orbits: 'سه نور: خورشید، ماه و سپیده در سه مدار جدا',
  pulse: 'یادآوری: حلقه‌ای که نورش بازمی‌گردد',
  seal: 'حلقهٔ کامل: دایره بسته می‌شود و ستاره در مرکز می‌درخشد',
};

export function TheiaInstrument({ state = 'welcome', tone = 'ink', className = '', label }) {
  const palette = TONES[tone] || TONES.ink;
  const ink = tone !== 'paper';
  const cite = Boolean(label);
  return (
    <ToneContext.Provider value={palette}>
      <svg
        className={`theia-instrument theia-instrument--${state} theia-instrument--tone-${tone} ${className}`.trim()}
        viewBox="0 0 400 400"
        fill="none"
        role={cite ? 'img' : undefined}
        aria-hidden={cite ? undefined : 'true'}
        aria-label={cite ? label : undefined}
        focusable="false"
      >
        <defs>
          <radialGradient id="ti-sky-ink" cx="50%" cy="82%" r="78%">
            <stop stopColor="#C08A4C" stopOpacity="0.16" />
            <stop offset="0.42" stopColor="#33405C" stopOpacity="0.13" />
            <stop offset="0.78" stopColor="#0B0D11" stopOpacity="0.05" />
            <stop offset="1" stopColor="#0A0B0D" stopOpacity="0" />
          </radialGradient>
          <radialGradient id="ti-sky-paper" cx="50%" cy="80%" r="76%">
            <stop stopColor="#D7A76A" stopOpacity="0.16" />
            <stop offset="0.5" stopColor="#E9D9BF" stopOpacity="0.22" />
            <stop offset="1" stopColor="#F2EDE4" stopOpacity="0" />
          </radialGradient>
          <radialGradient id="ti-vignette-ink" cx="50%" cy="46%" r="66%">
            <stop stopColor="#000" stopOpacity="0" />
            <stop offset="0.74" stopColor="#000" stopOpacity="0" />
            <stop offset="1" stopColor="#000" stopOpacity="0.32" />
          </radialGradient>
          <radialGradient id="ti-vignette-paper" cx="50%" cy="46%" r="68%">
            <stop stopColor="#6B5940" stopOpacity="0" />
            <stop offset="0.76" stopColor="#6B5940" stopOpacity="0" />
            <stop offset="1" stopColor="#6B5940" stopOpacity="0.1" />
          </radialGradient>
          <radialGradient id="ti-disc-ink" cx="50%" cy="34%" r="70%">
            <stop stopColor="#1B2130" />
            <stop offset="0.55" stopColor="#0E1118" />
            <stop offset="1" stopColor="#080A0D" />
          </radialGradient>
          <radialGradient id="ti-disc-paper" cx="50%" cy="30%" r="72%">
            <stop stopColor="#FFFDF7" />
            <stop offset="0.62" stopColor="#F8F1E3" />
            <stop offset="1" stopColor="#EDE2CE" />
          </radialGradient>
          <linearGradient id="ti-sheen" x1="90" y1="80" x2="320" y2="330" gradientUnits="userSpaceOnUse">
            <stop stopColor="#FFFFFF" stopOpacity="0.1" />
            <stop offset="0.42" stopColor="#FFFFFF" stopOpacity="0.015" />
            <stop offset="1" stopColor="#000000" stopOpacity="0.14" />
          </linearGradient>
          <linearGradient id="ti-gold" x1="60" y1="46" x2="340" y2="350" gradientUnits="userSpaceOnUse">
            <stop stopColor="#FBEFD4" />
            <stop offset="0.38" stopColor="#E9CE9B" />
            <stop offset="0.78" stopColor="#C99B5E" />
            <stop offset="1" stopColor="#A87C45" />
          </linearGradient>
          <linearGradient id="ti-gold-paper" x1="60" y1="46" x2="340" y2="350" gradientUnits="userSpaceOnUse">
            <stop stopColor="#C99C61" />
            <stop offset="0.5" stopColor="#AC7B40" />
            <stop offset="1" stopColor="#855826" />
          </linearGradient>
          <linearGradient id="ti-horizon-ink" x1="33" y1="0" x2="367" y2="0" gradientUnits="userSpaceOnUse">
            <stop stopColor="#E4D3B4" stopOpacity="0" />
            <stop offset="0.32" stopColor="#EFDFC0" stopOpacity="0.55" />
            <stop offset="0.5" stopColor="#FFF3DA" stopOpacity="0.95" />
            <stop offset="0.68" stopColor="#EFDFC0" stopOpacity="0.55" />
            <stop offset="1" stopColor="#E4D3B4" stopOpacity="0" />
          </linearGradient>
          <linearGradient id="ti-horizon-paper" x1="33" y1="0" x2="367" y2="0" gradientUnits="userSpaceOnUse">
            <stop stopColor="#3A352C" stopOpacity="0" />
            <stop offset="0.5" stopColor="#3A352C" stopOpacity="0.45" />
            <stop offset="1" stopColor="#3A352C" stopOpacity="0" />
          </linearGradient>
          <radialGradient id="ti-halo" cx="50%" cy="50%" r="50%">
            <stop stopColor="#FFE7BC" stopOpacity="0.62" />
            <stop offset="0.42" stopColor="#F0CD93" stopOpacity="0.22" />
            <stop offset="1" stopColor="#E7B877" stopOpacity="0" />
          </radialGradient>
          <radialGradient id="ti-halo-paper" cx="50%" cy="50%" r="50%">
            <stop stopColor="#E7C08A" stopOpacity="0.5" />
            <stop offset="0.45" stopColor="#DFB478" stopOpacity="0.18" />
            <stop offset="1" stopColor="#D7A76A" stopOpacity="0" />
          </radialGradient>
          <linearGradient id="ti-dawn-ink" x1="0" y1="152" x2="0" y2="300" gradientUnits="userSpaceOnUse">
            <stop stopColor="#E9A96B" stopOpacity="0" />
            <stop offset="0.55" stopColor="#E7B478" stopOpacity="0.22" />
            <stop offset="0.86" stopColor="#F5D7A4" stopOpacity="0.5" />
            <stop offset="1" stopColor="#FFEFD0" stopOpacity="0.62" />
          </linearGradient>
          <linearGradient id="ti-dawn-paper" x1="0" y1="152" x2="0" y2="300" gradientUnits="userSpaceOnUse">
            <stop stopColor="#D9B782" stopOpacity="0" />
            <stop offset="0.6" stopColor="#D9B782" stopOpacity="0.18" />
            <stop offset="1" stopColor="#C79A5C" stopOpacity="0.34" />
          </linearGradient>
          <linearGradient id="ti-slit-soft" x1="0" y1="96" x2="0" y2="304" gradientUnits="userSpaceOnUse">
            <stop stopColor="#F8EBCE" stopOpacity="0" />
            <stop offset="0.3" stopColor="#F6E2BA" stopOpacity="0.85" />
            <stop offset="0.62" stopColor="#F3D8A3" stopOpacity="0.55" />
            <stop offset="1" stopColor="#F3D8A3" stopOpacity="0" />
          </linearGradient>
          <linearGradient id="ti-slit-paper" x1="0" y1="96" x2="0" y2="304" gradientUnits="userSpaceOnUse">
            <stop stopColor="#D9B782" stopOpacity="0" />
            <stop offset="0.34" stopColor="#D9B782" stopOpacity="0.5" />
            <stop offset="0.64" stopColor="#C79A5C" stopOpacity="0.26" />
            <stop offset="1" stopColor="#C79A5C" stopOpacity="0" />
          </linearGradient>
          <linearGradient id="ti-fold-ramp" x1="0" y1="96" x2="0" y2="304" gradientUnits="userSpaceOnUse">
            <stop stopColor="#fff" stopOpacity="0" />
            <stop offset="0.26" stopColor="#fff" stopOpacity="1" />
            <stop offset="0.74" stopColor="#fff" stopOpacity="1" />
            <stop offset="1" stopColor="#fff" stopOpacity="0" />
          </linearGradient>
          <clipPath id="ti-disc-clip">
            <circle cx={C} cy={C} r={DISC} />
          </clipPath>
          <clipPath id="ti-plate-clip">
            <circle cx={C} cy={C} r={PLATE - 1} />
          </clipPath>
          <linearGradient id="ti-beam-fade" x1="0" y1="0" x2="1" y2="0">
            <stop stopColor="#fff" stopOpacity="0" />
            <stop offset="0.28" stopColor="#fff" stopOpacity="1" />
            <stop offset="0.72" stopColor="#fff" stopOpacity="1" />
            <stop offset="1" stopColor="#fff" stopOpacity="0" />
          </linearGradient>
          <mask id="ti-beam-mask">
            <rect x="160" y="80" width="80" height="240" fill="url(#ti-beam-fade)" />
          </mask>
          <mask id="ti-fold-fade">
            <rect x="70" y="80" width="260" height="240" fill="url(#ti-fold-ramp)" />
          </mask>
          <mask id="ti-moon-mask">
            <rect x="0" y="0" width="400" height="400" fill="#000" />
            <circle cx={polar(96, 296)[0]} cy={polar(96, 296)[1]} r="11.6" fill="#fff" />
            <circle cx={polar(96, 296)[0] - 5.8} cy={polar(96, 296)[1] + 2.6} r="10.7" fill="#000" />
          </mask>
          <filter id="ti-soft" x="-40%" y="-40%" width="180%" height="180%">
            <feGaussianBlur stdDeviation="9" />
          </filter>
          <filter id="ti-soft-sm" x="-40%" y="-40%" width="180%" height="180%">
            <feGaussianBlur stdDeviation="2.4" />
          </filter>
          <filter id="ti-grain" x="0" y="0" width="100%" height="100%">
            <feTurbulence type="fractalNoise" baseFrequency="0.85" numOctaves="3" seed="11" result="noise" />
            <feColorMatrix in="noise" type="saturate" values="0" />
          </filter>
        </defs>

        <Atmosphere calm={state === 'night' ? 0.55 : 1} />
        <Plate detail={state === 'night' ? 0.5 : 1} showRete={state !== 'night'} />

        {/* ---------------- welcome: the open ring, the first spark -------- */}
        {state === 'welcome' && (
          <g>
            <Disc open={15} />
            <g className="ti-bloom">
              <circle cx={C} cy={C} r="150" fill={palette.halo} opacity="0.13" />
            </g>
            <Seam from={126} to={214} radius={DISC + 18} width={2.4} spark />
            <Seam from={300} to={330} radius={DISC + 18} width={1.1} opacity={0.3} />
            <Rays count={10} from={DISC + 26} to={DISC + 52} opacity={0.12} rotation={12} />
            <Star radius={19} bloom={1.1} />
          </g>
        )}

        {/* ---------------- night: almost nothing is visible --------------- */}
        {state === 'night' && (
          <g>
            <Disc opacity={0.9} light={0.3} />
            <g clipPath="url(#ti-disc-clip)">
              <path d="M200 96V304" stroke={palette.rim} strokeWidth="0.4" opacity="0.1" />
              <path d={arcPath(70, 24, 156)} stroke={palette.rim} strokeWidth="0.4" opacity="0.09" />
              <rect x="96" y="96" width="208" height="208" fill={palette.dawnBand} opacity="0.2" />
            </g>
            <circle cx={C} cy={C} r="120" fill={palette.halo} opacity="0.05" />
            <g stroke={palette.rim} strokeWidth="0.35" opacity="0.22">
              <path d="M236 158L262 176L250 206" strokeLinejoin="round" />
            </g>
            <g fill={palette.rim}>
              <circle cx="236" cy="158" r="1.6" opacity="0.5" />
              <circle cx="262" cy="176" r="1.1" opacity="0.34" />
              <circle cx="250" cy="206" r="1.9" opacity="0.42" />
            </g>
            <Seam from={196} to={240} radius={DISC - 6} width={1.1} opacity={0.32} spark />
            <circle cx={C} cy={C} r="9" fill={palette.halo} opacity={ink ? 0.5 : 0.35} />
            <Star radius={ink ? 8 : 7} opacity={ink ? 0.62 : 0.55} glow={false} />
            <circle cx={C} cy={C} r="2.4" fill={palette.bright} opacity="0.8" filter="url(#ti-soft-sm)" />
          </g>
        )}

        {/* ---------------- privacy: a curtain parted at the middle -------- */}
        {state === 'veil' && (
          <g>
            <Disc />
            <g clipPath="url(#ti-disc-clip)">
              {/* The light that stays behind the curtain. */}
              <path d="M198 96Q188 200 180 304L220 304Q212 200 202 96Z" fill={palette.slit} opacity="0.85" mask="url(#ti-beam-mask)" />
              <g mask="url(#ti-fold-fade)">
                {/* Two hems, and the folds that hang from them. */}
                <path d="M197 96Q186 200 179 304" stroke={palette.rim} strokeWidth="1.05" opacity="0.55" />
                <path d="M203 96Q214 200 221 304" stroke={palette.rim} strokeWidth="1.05" opacity="0.55" />
                <path d="M194 96Q183 200 175 304" stroke={palette.rim} strokeWidth="0.5" opacity="0.2" />
                <path d="M206 96Q217 200 225 304" stroke={palette.rim} strokeWidth="0.5" opacity="0.2" />
                {[
                  [14, 0.3, 0.86, 1.04],
                  [31, 0.22, 1.06, 0.9],
                  [52, 0.15, 0.78, 1.16],
                ].map(([k, o, bulge, foot]) => (
                  <g key={k} opacity={o}>
                    <path
                      d={`M${197 - k} 96Q${186 - k * bulge} 200 ${179 - k * foot} 304`}
                      stroke={palette.rim}
                      strokeWidth="0.8"
                      strokeLinecap="round"
                    />
                    <path
                      d={`M${203 + k} 96Q${214 + k * (2 - bulge) * 0.86} 200 ${221 + k * foot} 304`}
                      stroke={palette.rim}
                      strokeWidth="0.8"
                      strokeLinecap="round"
                    />
                  </g>
                ))}
              </g>
              <circle cx={C} cy={C} r="26" fill={palette.halo} opacity="0.3" />
              <path d={STAR(C, C, 7)} fill={palette.bright} opacity="0.92" />
            </g>
          </g>
        )}

        {/* ---------------- sight: the eye that sees light ----------------- */}
        {state === 'sight' && (
          <g>
            <Disc />
            <g className="ti-eye">
              <path d="M118 200Q200 152 282 200Q200 248 118 200Z" fill={palette.halo} opacity="0.16" />
              <path d="M118 200Q200 152 282 200Q200 248 118 200Z" stroke={palette.rim} strokeWidth="0.95" opacity="0.6" />
              <path d="M133 200Q200 164 267 200" stroke={palette.rim} strokeWidth="0.55" opacity="0.22" />
              <path d="M133 200Q200 236 267 200" stroke={palette.rim} strokeWidth="0.55" opacity="0.14" />
              <circle cx={C} cy={C} r="36" stroke={palette.rim} strokeWidth="0.5" opacity="0.28" />
              <circle cx={C} cy={C} r="46" stroke={palette.rim} strokeWidth="0.5" opacity="0.14" />
              <g className="ti-iris">
                <circle cx={C} cy={C} r="30" stroke={palette.goldFlat} strokeWidth="0.6" opacity="0.35" />
                <circle cx={C} cy={C} r="21" stroke={palette.goldFlat} strokeWidth="0.7" opacity="0.5" />
                <g className="ti-iris-rays" opacity="0.5">
                  {Array.from({ length: 16 }, (_, i) => {
                    const [x1, y1] = polar(29, i * 22.5);
                    const [x2, y2] = polar(34.5, i * 22.5);
                    return <line key={i} x1={px(x1)} y1={px(y1)} x2={px(x2)} y2={px(y2)} stroke={palette.goldFlat} strokeWidth="0.6" />;
                  })}
                </g>
              </g>
              <circle cx={C} cy={C} r="26" fill={palette.halo} opacity="0.55" />
              <circle cx={C} cy={C} r="7.6" fill={palette.gold} opacity={ink ? 1 : 0.3} />
              <circle cx={C} cy={C} r="7.6" stroke={palette.rim} strokeWidth="0.9" opacity={ink ? 0.25 : 0.9} />
              <circle cx={C} cy={C} r={ink ? 7.6 : 2.6} fill={palette.bright} opacity={ink ? 0.55 : 0.95} />
              <g className="ti-sight-ring">
                <circle cx={C} cy={C} r="18" stroke={palette.rim} strokeWidth="0.7" opacity="0.4" />
              </g>
              <path d={STAR(C - 26, C - 30, 2.6)} fill={palette.bright} opacity="0.7" />
              <path d={STAR(C + 34, C + 26, 1.9)} fill={palette.bright} opacity="0.5" />
            </g>
          </g>
        )}

        {/* ---------------- dawn: the sun comes back slowly ---------------- */}
        {state === 'dawn' && (
          <g>
            <Disc lit={0.55} />
            <g clipPath="url(#ti-disc-clip)">
              <rect x={C - DISC} y="152" width={DISC * 2} height="152" fill={palette.dawnBand} />
              <g className="ti-sun">
                <circle cx={C} cy={HORIZON - 3} r="46" fill={palette.halo} opacity="0.85" />
                <g opacity="0.35">
                  {[-62, -31, 0, 31, 62].map((angle) => {
                    const [x1, y1] = polar(20, angle);
                    const [x2, y2] = polar(30, angle);
                    return <line key={angle} x1={px(x1)} y1={px(y1 + 0)} x2={px(x2)} y2={px(y2)} stroke={palette.goldFlat} strokeWidth="0.8" strokeLinecap="round" />;
                  })}
                </g>
                <circle cx={C} cy={HORIZON - 1} r="13" fill={palette.sunBody} opacity={ink ? 0.9 : 0.98} />
                <circle cx={C} cy={HORIZON - 1} r="13" stroke={palette.sunRim} strokeWidth="0.9" opacity="0.75" />
                <path d={`M${C - 4} ${HORIZON - 9}A6 6 0 0 1 ${C + 4} ${HORIZON - 9}Z`} fill="#FFFFFF" opacity={ink ? 0.4 : 0.9} />
              </g>
              <path d={`M${C - DISC} ${HORIZON}H${C + DISC}`} stroke={palette.bright} strokeWidth="1.2" opacity="0.55" />
              <g className="ti-dawn-ripple" opacity="0.5">
                {[1, 1.7, 2.4].map((scale, index) => (
                  <path
                    key={index}
                    d={`M${C - 26 * scale} ${HORIZON + 10 + index * 9}H${C + 26 * scale}`}
                    stroke={palette.goldFlat}
                    strokeWidth="0.8"
                    strokeLinecap="round"
                    opacity={0.55 - index * 0.14}
                  />
                ))}
              </g>
            </g>
            <Seam from={142} to={322} radius={DISC + 6} width={1.6} opacity={ink ? 0.85 : 0.45} />
            <Rays count={12} from={DISC + 16} to={DISC + 34} opacity={0.16} rotation={6} />
            <Star radius={11} opacity={0.55} glow={false} />
          </g>
        )}

        {/* ---------------- three lights: sun, moon, dawn ------------------ */}
        {state === 'orbits' && (
          <g>
            <Disc opacity={0.72} />
            <g className="ti-orbits">
              {[64, 96, 128].map((radius, index) => (
                <circle key={radius} cx={C} cy={C} r={radius} stroke={palette.rim} strokeWidth="0.5" opacity={0.22 - index * 0.05} strokeDasharray={index === 1 ? '2 6' : undefined} />
              ))}
              {/* Helios: a sun with a corona and a short ray fan. */}
              <g>
                <circle cx={polar(128, 32)[0]} cy={polar(128, 32)[1]} r="26" fill={palette.halo} opacity={ink ? 0.75 : 0.5} />
                <circle cx={polar(128, 32)[0]} cy={polar(128, 32)[1]} r="9" fill={ink ? palette.gold : palette.sunBody} />
                <circle cx={polar(128, 32)[0]} cy={polar(128, 32)[1]} r="9" stroke={palette.sunRim} strokeWidth="0.9" opacity={ink ? 0.35 : 0.9} />
                <circle cx={polar(128, 32)[0]} cy={polar(128, 32)[1]} r={ink ? 3.4 : 2.4} fill={ink ? palette.bright : palette.goldFlat} opacity="0.9" />
                <g opacity="0.5">
                  {Array.from({ length: 8 }, (_, i) => {
                    const angle = i * 45;
                    const [x1, y1] = polar(15, angle, polar(128, 32)[0], polar(128, 32)[1]);
                    const [x2, y2] = polar(20, angle, polar(128, 32)[0], polar(128, 32)[1]);
                    return <line key={i} x1={px(x1)} y1={px(y1)} x2={px(x2)} y2={px(y2)} stroke={palette.goldFlat} strokeWidth="0.7" strokeLinecap="round" />;
                  })}
                </g>
              </g>
              {/* Selene: the moon, an unlit body and the crescent it borrows. */}
              <circle cx={polar(96, 296)[0]} cy={polar(96, 296)[1]} r="26" fill={palette.halo} opacity="0.34" />
              <circle cx={polar(96, 296)[0]} cy={polar(96, 296)[1]} r="11.6" fill={palette.goldFlat} opacity="0.14" />
              <circle cx={polar(96, 296)[0]} cy={polar(96, 296)[1]} r="11.6" stroke={palette.rim} strokeWidth="0.5" opacity="0.3" />
              <g mask="url(#ti-moon-mask)">
                <circle cx={polar(96, 296)[0]} cy={polar(96, 296)[1]} r="11.6" fill={palette.goldFlat} opacity="0.95" />
              </g>
              {/* Eos: a small sun returning over its own horizon. */}
              <g>
                <circle cx={polar(64, 156)[0]} cy={polar(64, 156)[1] + 2} r="18" fill={palette.halo} opacity="0.55" />
                <path
                  d={`M${px(polar(64, 156)[0] - 5)} ${px(polar(64, 156)[1] + 4)}A5 5 0 0 1 ${px(polar(64, 156)[0] + 5)} ${px(polar(64, 156)[1] + 4)}Z`}
                  fill={palette.ray}
                  opacity="0.95"
                />
                <path d={`M${px(polar(64, 156)[0] - 13)} ${px(polar(64, 156)[1] + 4)}H${px(polar(64, 156)[0] + 13)}`} stroke={palette.ray} strokeWidth="1.1" strokeLinecap="round" opacity="0.8" />
                <path d={`M${px(polar(64, 156)[0] - 8)} ${px(polar(64, 156)[1] + 10)}H${px(polar(64, 156)[0] + 8)}`} stroke={palette.ray} strokeWidth="0.8" strokeLinecap="round" opacity="0.4" />
                <path d={`M${px(polar(64, 156)[0] - 3)} ${px(polar(64, 156)[1] - 5)}H${px(polar(64, 156)[0] + 3)}`} stroke={palette.ray} strokeWidth="0.8" strokeLinecap="round" opacity="0.35" />
              </g>
            </g>
            <Star radius={5.5} opacity={0.5} />
          </g>
        )}

        {/* ---------------- reminder: the light returns -------------------- */}
        {state === 'pulse' && (
          <g>
            <Disc />
            <Seam from={146} to={254} radius={DISC + 2} width={2.4} spark />
            <circle cx={C} cy={C} r={DISC + 18} stroke={palette.goldFlat} strokeWidth="1.1" fill="none" opacity="0.5" className="ti-pulse" />
            <circle cx={C} cy={C} r={DISC + 18} stroke={palette.goldFlat} strokeWidth="1.1" fill="none" opacity="0.3" className="ti-pulse ti-pulse--late" />
            <circle cx={C} cy={C} r={DISC + 18} stroke={palette.goldFlat} strokeWidth="1.1" fill="none" opacity="0.16" className="ti-pulse ti-pulse--later" />
            <circle cx={C} cy={C} r="120" fill={palette.halo} opacity="0.18" />
            <Star radius={14} />
          </g>
        )}

        {/* ---------------- ready: the circle closes ----------------------- */}
        {state === 'seal' && (
          <g>
            <Disc lit={0.28} />
            <circle cx={C} cy={C} r={DISC + 18} stroke={palette.gold} strokeWidth="1.6" opacity="0.9" />
            <circle cx={C} cy={C} r={DISC + 25} stroke={palette.goldFlat} strokeWidth="0.8" opacity="0.4" />
            <Rays count={24} from={DISC + 28} to={DISC + 40} opacity={0.42} />
            <g className="ti-rays-flicker">
              <Rays count={12} from={DISC + 44} to={DISC + 72} opacity={0.2} />
            </g>
            <circle cx={C} cy={C} r="130" fill={palette.halo} opacity="0.2" />
            <Star radius={22} bloom={1.25} />
            <path d={arcPath(PLATE - 12, 0, 360)} stroke={palette.frame} strokeWidth="0.6" opacity="0.12" />
          </g>
        )}

      </svg>
    </ToneContext.Provider>
  );
}
