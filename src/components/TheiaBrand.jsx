import React, { useId } from 'react';

/**
 * Theia (تیا) — visual identity: «دریچهٔ نور» / The Aperture of First Light.
 *
 * No figure is depicted. The brand is built from one quiet idea:
 * an aperture opening onto an inner sky. A fine ring (the aperture), a horizon,
 * and a single point of light recur at every scale — in the mark, the onboarding
 * plates, the letter seal and the home emblem. Across onboarding the aperture
 * opens a little more on each plate; by the final screen it has become dawn.
 */

const DAY = {
  ink: '#23201b',
  paper: '#f3f2ee',
  brass: '#c99a5f',
  brassLight: '#e9c08a',
  lavender: '#a9afd8',
};
const NIGHT = {
  ink: '#f3efe6',
  field: '#000000',
  brass: '#f6cf98',
  brassLight: '#fbe9cb',
  lavender: '#d5d9f7',
};

const rad = (deg) => (deg * Math.PI) / 180;
const point = (cx, cy, r, deg) => [cx + r * Math.cos(rad(deg)), cy + r * Math.sin(rad(deg))];
const fmt = (n) => Math.round(n * 100) / 100;

/** Clockwise arc, 0° = right, 90° = down. */
function arc(cx, cy, r, startDeg, endDeg) {
  const [x1, y1] = point(cx, cy, r, startDeg);
  const [x2, y2] = point(cx, cy, r, endDeg);
  const sweep = ((endDeg - startDeg) % 360 + 360) % 360;
  return `M${fmt(x1)} ${fmt(y1)}A${r} ${r} 0 ${sweep > 180 ? 1 : 0} 1 ${fmt(x2)} ${fmt(y2)}`;
}

/** An aperture ring with a soft gap centred at the top. gap is in degrees. */
function ringPath(cx, cy, r, gap = 34) {
  return arc(cx, cy, r, -90 + gap / 2, 270 - gap / 2);
}

function useUid(prefix) {
  return `${prefix}-${useId().replace(/[^a-zA-Z0-9_-]/g, '')}`;
}

/**
 * The mark: an aperture with a horizon and a single point of first light.
 * `opening` widens the ring gap (degrees); `luminous` adds the warm light.
 */
export function TheiaMark({ size = 24, luminous = false, mono = false, opening = 58, className = '' }) {
  const uid = useUid('mark');
  const ring = luminous ? NIGHT.ink : 'currentColor';
  const light = mono ? 'currentColor' : NIGHT.brass;
  const detailed = size >= 22;

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 40 40"
      fill="none"
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <radialGradient id={`${uid}-sun`} cx="50%" cy="36%" r="62%">
          <stop offset="0%" stopColor="#fdeccf" />
          <stop offset="66%" stopColor="#f6cf98" />
          <stop offset="100%" stopColor="#e2ab6b" />
        </radialGradient>
      </defs>
      <path
        d={ringPath(20, 20, 15.6, opening)}
        stroke={ring}
        strokeWidth="1.45"
        strokeLinecap="round"
        opacity={luminous ? '.92' : '.9'}
      />
      {detailed && (
        <path d="M6 25.2H31.2" stroke={ring} strokeWidth=".9" opacity=".42" strokeLinecap="round" />
      )}
      <circle
        cx="17.4"
        cy="22.6"
        r={detailed ? 3.9 : 4.4}
        fill={mono ? 'currentColor' : `url(#${uid}-sun)`}
      />
      {detailed && (
        <path
          d={arc(20, 20, 15.6, 160, 266)}
          stroke={light}
          strokeWidth="1.75"
          strokeLinecap="round"
          opacity=".95"
        />
      )}
    </svg>
  );
}

/** Tiny wax-seal glyph (10–14px): light held inside a quiet ring. */
export function TheiaSealIcon({ size = 11, className = '' }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <circle cx="8" cy="8" r="6.1" stroke="currentColor" strokeWidth="1" opacity=".8" />
      <path d="M3.4 9.8H12.6" stroke="currentColor" strokeWidth=".8" opacity=".5" />
      <circle cx="8" cy="9.2" r="2" fill="currentColor" />
    </svg>
  );
}

/* ---------------------------------------------------------------------------
 * Onboarding plates — one story, five movements, told with the same grammar.
 * ------------------------------------------------------------------------- */

function Plate({ viewBox, label, className = '', children }) {
  return (
    <svg
      viewBox={viewBox}
      className={`theia-plate ${className}`.trim()}
      role={label ? 'img' : undefined}
      aria-label={label || undefined}
      aria-hidden={label ? undefined : 'true'}
      focusable="false"
      preserveAspectRatio="xMidYMid meet"
    >
      {children}
    </svg>
  );
}

function Horizon({ y, w = 340, color, opacity = 0.28, ticks = [] }) {
  return (
    <g>
      <path d={`M0 ${y}H${w}`} stroke={color} strokeWidth=".8" opacity={opacity} />
      {ticks.map((x) => (
        <path key={x} d={`M${x} ${y}v${x % 80 === 0 ? 5 : 3}`} stroke={color} strokeWidth=".8" opacity={opacity * 0.7} />
      ))}
    </g>
  );
}

/**
 * 1. Welcome — «پنجره» The Window. A night sky held inside an aperture.
 */
export function TheiaWelcomeArt({ className = '', label = 'دریچه‌ای گِرد که آسمانِ شب و سپیده‌دم را در خود نگه داشته است' }) {
  const uid = useUid('welcome');
  const c = { x: 160, y: 222, r: 138 };
  return (
    <Plate viewBox="0 0 320 470" label={label} className={`theia-welcome ${className}`.trim()}>
      <defs>
        <linearGradient id={`${uid}-night`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#1c1a15" />
          <stop offset="58%" stopColor="#12110d" />
          <stop offset="100%" stopColor="#0a0907" />
        </linearGradient>
        <radialGradient id={`${uid}-glow`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#f6cf98" stopOpacity=".5" />
          <stop offset="42%" stopColor="#f0bf82" stopOpacity=".15" />
          <stop offset="100%" stopColor="#f0bf82" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`${uid}-sun`} cx="50%" cy="38%" r="62%">
          <stop offset="0%" stopColor="#fdf0d8" />
          <stop offset="66%" stopColor="#f6cf98" />
          <stop offset="100%" stopColor="#e2ab6b" />
        </radialGradient>
        <clipPath id={`${uid}-clip`}>
          <circle cx={c.x} cy={c.y} r={c.r} />
        </clipPath>
        <mask id={`${uid}-half`}>
          <rect x="0" y="0" width="320" height="318" fill="#fff" />
        </mask>
      </defs>

      <ellipse cx="160" cy="232" rx="196" ry="236" fill="none" stroke={DAY.ink} strokeWidth=".8" opacity=".1" />
      <ellipse
        cx="166" cy="238" rx="150" ry="190" fill="none" stroke={DAY.ink}
        strokeWidth=".7" opacity=".07" transform="rotate(-12 166 238)"
      />

      <circle cx={c.x} cy={c.y} r={c.r} fill={`url(#${uid}-night)`} />
      <g clipPath={`url(#${uid}-clip)`}>
        <circle cx="170" cy="318" r="118" fill={`url(#${uid}-glow)`} />
        <path d="M16 318H304" stroke="#f3efe6" strokeWidth=".8" opacity=".38" />
        <g mask={`url(#${uid}-half)`}>
          <circle cx="176" cy="330" r="31" fill={`url(#${uid}-sun)`} className="theia-glow--soft" />
        </g>
        <circle cx="176" cy="318" r="45" fill="none" stroke="#f6cf98" strokeWidth=".55" opacity=".16" />
        <path d="M140 348h62" stroke="#f6cf98" strokeWidth=".7" opacity=".14" />
        <g className="theia-stars">
          <circle cx="196" cy="252" r="1.6" fill="#d5d9f7" opacity=".95" />
          <circle cx="110" cy="164" r="1.3" fill="#f3efe6" opacity=".6" />
          <circle cx="212" cy="128" r="1" fill="#f3efe6" opacity=".45" />
          <circle cx="140" cy="222" r=".9" fill="#f3efe6" opacity=".28" />
          <circle cx="230" cy="232" r=".9" fill="#d5d9f7" opacity=".4" />
        </g>
      </g>

      <path d={ringPath(c.x, c.y, c.r, 26)} fill="none" stroke="#f3efe6" strokeWidth=".8" opacity=".16" />
      <path d={arc(c.x, c.y, c.r, 176, 262)} fill="none" stroke={DAY.brass} strokeWidth="1.1" opacity=".65" strokeLinecap="round" />

      <g stroke={DAY.ink} strokeWidth=".8" opacity=".22">
        <path d={`M${c.x} ${c.y - c.r - 9}v-7`} />
        <path d={`M${c.x} ${c.y + c.r + 9}v7`} />
        <path d={`M${c.x - c.r - 9} ${c.y}h-7`} />
        <path d={`M${c.x + c.r + 9} ${c.y}h7`} />
      </g>
      <circle cx="92" cy="392" r="1.2" fill={DAY.ink} opacity=".3" />
      <circle cx="252" cy="404" r="1" fill={DAY.lavender} opacity=".8" />
      <g stroke={DAY.ink} strokeWidth=".8" opacity=".2">
        <path d="M18 430v5" />
      </g>
    </Plate>
  );
}

/** 2. «نقطه» The Point — a single line finds the first light. */
export function TheiaPointArt({ className = '', label = 'خطی باریک در تاریکی به نخستین نقطهٔ نور می‌رسد' }) {
  const uid = useUid('point');
  return (
    <Plate viewBox="0 0 340 310" label={label} className={`theia-point ${className}`.trim()}>
      <defs>
        <radialGradient id={`${uid}-glow`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#f6cf98" stopOpacity=".5" />
          <stop offset="40%" stopColor="#f0bf82" stopOpacity=".16" />
          <stop offset="100%" stopColor="#f0bf82" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`${uid}-sun`} cx="50%" cy="38%" r="62%">
          <stop offset="0%" stopColor="#fdf0d8" />
          <stop offset="66%" stopColor="#f6cf98" />
          <stop offset="100%" stopColor="#e2ab6b" />
        </radialGradient>
      </defs>
      <circle cx="170" cy="430" r="250" fill="none" stroke={NIGHT.ink} strokeWidth=".8" opacity=".08" />
      <ellipse cx="206" cy="176" rx="168" ry="106" fill="none" stroke={NIGHT.ink} strokeWidth=".7" opacity=".07" transform="rotate(-18 206 176)" />
      <path
        d="M-16 282C58 266 132 208 216 178"
        fill="none" stroke={NIGHT.ink} strokeWidth="1" opacity=".4" strokeLinecap="round"
      />
      <path d={arc(216, 178, 44, 100, 262)} fill="none" stroke={NIGHT.ink} strokeWidth=".8" opacity=".2" strokeLinecap="round" />
      <Horizon y={252} color={NIGHT.ink} opacity={0.3} ticks={[64, 170, 276]} />
      <circle cx="216" cy="178" r="68" fill={`url(#${uid}-glow)`} />
      <circle cx="216" cy="178" r="4.4" fill={`url(#${uid}-sun)`} />
      <g className="theia-stars">
        <circle cx="82" cy="84" r="1.3" fill={NIGHT.lavender} opacity=".85" />
        <circle cx="276" cy="118" r="1" fill={NIGHT.ink} opacity=".42" />
        <circle cx="130" cy="132" r=".9" fill={NIGHT.ink} opacity=".26" />
        <circle cx="292" cy="222" r=".8" fill={NIGHT.ink} opacity=".3" />
      </g>
      <g stroke={NIGHT.ink} strokeWidth=".8" opacity=".18">
        <path d="M18 288v-5" />
        <path d="M24 288v-5" />
      </g>
    </Plate>
  );
}

/** 3. «گشودن» The Opening — the ring widens; light gathers inside. */
export function TheiaOpeningArt({ className = '', label = 'دریچه‌ای که آرام گشوده شده و نور در آن گرد آمده است' }) {
  const uid = useUid('opening');
  const c = { x: 170, y: 190, r: 116 };
  return (
    <Plate viewBox="0 0 340 350" label={label} className={`theia-opening ${className}`.trim()}>
      <defs>
        <radialGradient id={`${uid}-glass`} cx="50%" cy="62%" r="62%">
          <stop offset="0%" stopColor="#191712" />
          <stop offset="100%" stopColor="#0a0907" />
        </radialGradient>
        <radialGradient id={`${uid}-glow`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#f6cf98" stopOpacity=".42" />
          <stop offset="44%" stopColor="#f0bf82" stopOpacity=".13" />
          <stop offset="100%" stopColor="#f0bf82" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`${uid}-sun`} cx="50%" cy="38%" r="62%">
          <stop offset="0%" stopColor="#fdf0d8" />
          <stop offset="66%" stopColor="#f6cf98" />
          <stop offset="100%" stopColor="#e2ab6b" />
        </radialGradient>
        <clipPath id={`${uid}-clip`}><circle cx={c.x} cy={c.y} r={c.r} /></clipPath>
      </defs>
      <ellipse cx="170" cy="196" rx="158" ry="158" fill="none" stroke={NIGHT.ink} strokeWidth=".7" opacity=".06" transform="rotate(-14 170 196)" />
      <circle cx={c.x} cy={c.y} r={c.r} fill={`url(#${uid}-glass)`} />
      <g clipPath={`url(#${uid}-clip)`}>
        <circle cx="184" cy="232" r="88" fill={`url(#${uid}-glow)`} />
        <path d="M40 238H306" stroke="#f3efe6" strokeWidth=".8" opacity=".36" />
        <circle cx="192" cy="238" r="12" fill={`url(#${uid}-sun)`} />
        <circle cx="192" cy="238" r="22" fill="none" stroke="#f6cf98" strokeWidth=".55" opacity=".2" />
        <path d="M164 268h54" stroke="#f6cf98" strokeWidth=".7" opacity=".16" />
        <g className="theia-stars">
          <circle cx="122" cy="144" r="1.3" fill={NIGHT.lavender} opacity=".85" />
          <circle cx="220" cy="118" r="1" fill={NIGHT.ink} opacity=".4" />
          <circle cx="150" cy="196" r=".8" fill={NIGHT.ink} opacity=".26" />
        </g>
      </g>
      <path d={ringPath(c.x, c.y, c.r, 70)} fill="none" stroke={NIGHT.ink} strokeWidth="1" opacity=".3" strokeLinecap="round" />
      <path d={arc(c.x, c.y, c.r, 154, 268)} fill="none" stroke={NIGHT.brass} strokeWidth="1.3" opacity=".55" strokeLinecap="round" />
      <g stroke={NIGHT.ink} strokeWidth=".8" opacity=".2">
        <path d={`M${c.x} ${c.y + c.r + 9}v8`} />
        <path d={`M${c.x - c.r - 9} ${c.y}h-8`} />
        <path d={`M${c.x + c.r + 9} ${c.y}h8`} />
      </g>

    </Plate>
  );
}

/** 4. «مدار» The Orbit — light learns a path; the aperture widens again. */
export function TheiaOrbitArt({ className = '', label = 'مدارهایی نازک گرد دریچه‌ای می‌چرخند و نوری بر افق می‌درخشد' }) {
  const uid = useUid('orbit');
  const c = { x: 152, y: 226, r: 102 };
  return (
    <Plate viewBox="0 0 340 415" label={label} className={`theia-orbit ${className}`.trim()}>
      <defs>
        <radialGradient id={`${uid}-glass`} cx="50%" cy="62%" r="62%">
          <stop offset="0%" stopColor="#191712" />
          <stop offset="100%" stopColor="#0a0907" />
        </radialGradient>
        <radialGradient id={`${uid}-glow`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#f6cf98" stopOpacity=".4" />
          <stop offset="44%" stopColor="#f0bf82" stopOpacity=".12" />
          <stop offset="100%" stopColor="#f0bf82" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`${uid}-sun`} cx="50%" cy="38%" r="62%">
          <stop offset="0%" stopColor="#fdf0d8" />
          <stop offset="66%" stopColor="#f6cf98" />
          <stop offset="100%" stopColor="#e2ab6b" />
        </radialGradient>
        <clipPath id={`${uid}-clip`}><circle cx={c.x} cy={c.y} r={c.r} /></clipPath>
      </defs>
      <circle cx="170" cy="200" r="166" fill="none" stroke={NIGHT.ink} strokeWidth=".7" opacity=".06" />
      <g className="theia-orbit-lines">
        <ellipse cx="190" cy="200" rx="166" ry="94" fill="none" stroke={NIGHT.ink} strokeWidth=".7" opacity=".11" transform="rotate(-28 190 200)" />
        <ellipse cx="170" cy="212" rx="148" ry="126" fill="none" stroke={NIGHT.lavender} strokeWidth=".6" opacity=".11" transform="rotate(38 170 212)" />
        <circle cx="322" cy="198" r="2" fill={NIGHT.brass} opacity=".55" />
        <circle cx="44" cy="302" r="1.4" fill={NIGHT.lavender} opacity=".6" />
      </g>
      <circle cx={c.x} cy={c.y} r={c.r} fill={`url(#${uid}-glass)`} />
      <g clipPath={`url(#${uid}-clip)`}>
        <circle cx="162" cy="256" r="76" fill={`url(#${uid}-glow)`} />
        <path d="M40 262H264" stroke="#f3efe6" strokeWidth=".8" opacity=".34" />
        <circle cx="170" cy="262" r="11" fill={`url(#${uid}-sun)`} />
        <circle cx="170" cy="262" r="20" fill="none" stroke="#f6cf98" strokeWidth=".55" opacity=".18" />
        <path d="M144 288h48" stroke="#f6cf98" strokeWidth=".7" opacity=".15" />
        <g className="theia-stars">
          <circle cx="110" cy="190" r="1.2" fill={NIGHT.lavender} opacity=".8" />
          <circle cx="198" cy="172" r=".9" fill={NIGHT.ink} opacity=".42" />
          <circle cx="140" cy="222" r=".8" fill={NIGHT.ink} opacity=".24" />
        </g>
      </g>
      <path d={ringPath(c.x, c.y, c.r, 90)} fill="none" stroke={NIGHT.ink} strokeWidth=".9" opacity=".28" strokeLinecap="round" />
      <path d={arc(c.x, c.y, c.r, 152, 268)} fill="none" stroke={NIGHT.brass} strokeWidth="1.2" opacity=".52" strokeLinecap="round" />
      <g stroke={NIGHT.ink} strokeWidth=".8" opacity=".2">
        <path d={`M${c.x} ${c.y - c.r - 9}v-8`} />
        <path d={`M${c.x} ${c.y + c.r + 9}v8`} />
      </g>
      <g className="theia-stars">
        <circle cx="196" cy="96" r="1.6" fill={NIGHT.lavender} />
        <circle cx="66" cy="118" r="1" fill={NIGHT.ink} opacity=".4" />
        <circle cx="288" cy="320" r="1.1" fill={NIGHT.ink} opacity=".5" />
        <circle cx="90" cy="366" r=".9" fill={NIGHT.lavender} opacity=".5" />
      </g>

    </Plate>
  );
}

/** 5. «آسمان» The Sky — the aperture is a horizon now; the night is wide. */
export function TheiaSkyArt({ className = '', label = 'آسمانی پرستاره با مداری بزرگ و نوری که نزدیک افق می‌درخشد' }) {
  const uid = useUid('sky');
  return (
    <Plate viewBox="0 0 320 520" label={label} className={`theia-sky ${className}`.trim()}>
      <defs>
        <radialGradient id={`${uid}-glow`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#f6cf98" stopOpacity=".5" />
          <stop offset="44%" stopColor="#f0bf82" stopOpacity=".13" />
          <stop offset="100%" stopColor="#f0bf82" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`${uid}-sun`} cx="50%" cy="38%" r="62%">
          <stop offset="0%" stopColor="#fdf0d8" />
          <stop offset="66%" stopColor="#f6cf98" />
          <stop offset="100%" stopColor="#e2ab6b" />
        </radialGradient>
      </defs>
      <g className="theia-orbit-lines">
        <ellipse cx="160" cy="246" rx="158" ry="250" fill="none" stroke={NIGHT.ink} strokeWidth=".7" opacity=".11" />
        <ellipse cx="160" cy="246" rx="112" ry="226" fill="none" stroke={NIGHT.ink} strokeWidth=".6" opacity=".06" transform="rotate(6 160 246)" />
      </g>
      <path d="M160 4V516" stroke={NIGHT.ink} strokeWidth=".5" opacity=".07" strokeDasharray="1 7" />
      <path d={arc(160, 470, 164, 186, 354)} fill="none" stroke={NIGHT.ink} strokeWidth=".8" opacity=".13" />
      <Horizon y={404} color={NIGHT.ink} opacity={0.22} ticks={[42, 122, 198, 278]} />
      <circle cx="196" cy="392" r="86" fill={`url(#${uid}-glow)`} />
      <circle cx="196" cy="392" r="10.5" fill={`url(#${uid}-sun)`} />
      <circle cx="196" cy="392" r="20" fill="none" stroke="#f6cf98" strokeWidth=".55" opacity=".18" />
      <g className="theia-stars">
        <circle cx="78" cy="86" r="1.5" fill={NIGHT.lavender} opacity=".9" />
        <circle cx="240" cy="60" r="1.1" fill={NIGHT.lavender} opacity=".7" />
        <circle cx="126" cy="146" r="1" fill={NIGHT.ink} opacity=".45" />
        <circle cx="286" cy="150" r="1.4" fill={NIGHT.lavender} opacity=".8" />
        <circle cx="44" cy="200" r=".9" fill={NIGHT.ink} opacity=".3" />
        <circle cx="206" cy="180" r=".8" fill={NIGHT.ink} opacity=".25" />
        <circle cx="76" cy="300" r="1" fill={NIGHT.ink} opacity=".38" />
        <circle cx="266" cy="240" r=".9" fill={NIGHT.ink} opacity=".3" />
        <circle cx="148" cy="228" r="1.2" fill={NIGHT.ink} opacity=".45" />
        <circle cx="58" cy="440" r="1" fill={NIGHT.ink} opacity=".22" />
        <circle cx="270" cy="452" r=".9" fill={NIGHT.ink} opacity=".2" />
        <circle cx="300" cy="72" r="2" fill={NIGHT.brass} opacity=".55" />
      </g>
      <circle cx="152" cy="54" r="2.4" fill="none" stroke={NIGHT.lavender} strokeWidth=".9" opacity=".7" />

    </Plate>
  );
}

/** Privacy — «مُهر» The Seal. The aperture is closed; the light is kept whole. */
export function TheiaPrivacyArt({ className = '', label = 'مهری گرد و آرام که نوری کوچک را در خود نگاه داشته است' }) {
  const uid = useUid('privacy');
  return (
    <Plate viewBox="0 0 120 136" label={label} className={`theia-privacy ${className}`.trim()}>
      <defs>
        <radialGradient id={`${uid}-sun`} cx="50%" cy="38%" r="62%">
          <stop offset="0%" stopColor="#fdf0d8" />
          <stop offset="66%" stopColor="#f6cf98" />
          <stop offset="100%" stopColor="#e2ab6b" />
        </radialGradient>
      </defs>
      <circle cx="60" cy="66" r="55" fill="none" stroke={DAY.ink} strokeWidth=".7" opacity=".28" />
      <circle cx="60" cy="66" r="46" fill="none" stroke={DAY.ink} strokeWidth=".6" opacity=".13" />
      <circle cx="60" cy="66" r="33" fill={DAY.ink} />
      <path d="M34 74H86" stroke="#f3efe6" strokeWidth=".7" opacity=".45" />
      <circle cx="60" cy="71" r="4" fill={`url(#${uid}-sun)`} />
      <path d={arc(60, 66, 46, 186, 268)} fill="none" stroke={DAY.brass} strokeWidth="1" opacity=".7" strokeLinecap="round" />
      <g stroke={DAY.ink} strokeWidth=".7" opacity=".22">
        <path d="M60 6v-3" />
        <path d="M60 126v3" />
        <path d="M4 66h-3" />
        <path d="M116 66h3" />
      </g>
    </Plate>
  );
}

/** Check-ins — «نبض» The Pulse. A gentle reminder, like light moving outward. */
export function TheiaPulseArt({ className = '', label = 'نوری که چون نبضی آرام به بیرون می‌تابد' }) {
  const uid = useUid('pulse');
  const c = { x: 146, y: 128 };
  return (
    <Plate viewBox="0 0 340 250" label={label} className={`theia-pulse ${className}`.trim()}>
      <defs>
        <radialGradient id={`${uid}-glow`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#c99a5f" stopOpacity=".3" />
          <stop offset="45%" stopColor="#c99a5f" stopOpacity=".08" />
          <stop offset="100%" stopColor="#c99a5f" stopOpacity="0" />
        </radialGradient>
      </defs>
      <circle cx={c.x} cy={c.y} r="112" fill="none" stroke={DAY.ink} strokeWidth=".8" opacity=".2" />
      <circle cx={c.x} cy={c.y} r="88" fill="none" stroke={DAY.ink} strokeWidth=".7" opacity=".13" />
      <path d={arc(c.x, c.y, 112, 182, 268)} fill="none" stroke={DAY.brass} strokeWidth="1" opacity=".5" strokeLinecap="round" />
      <circle cx={c.x} cy={c.y} r="72" fill={`url(#${uid}-glow)`} />
      <g className="theia-pulse-waves" fill="none">
        <circle cx={c.x} cy={c.y} r="34" stroke={DAY.ink} strokeWidth="1.15" opacity=".3" />
        <circle cx={c.x} cy={c.y} r="64" stroke={DAY.ink} strokeWidth="1" opacity=".18" />
        <circle cx={c.x} cy={c.y} r="94" stroke={DAY.ink} strokeWidth=".9" opacity=".1" />
      </g>
      <circle cx={c.x} cy={c.y} r="9.5" fill={DAY.brass} opacity=".16" />
      <circle cx={c.x} cy={c.y} r="4.8" fill={DAY.brass} />
      <Horizon y={208} color={DAY.ink} opacity={0.3} ticks={[56, 146, 246]} />
      <circle cx="262" cy="82" r="1.2" fill={DAY.ink} opacity=".3" />
      <circle cx="294" cy="148" r="1" fill={DAY.ink} opacity=".22" />
      <g stroke={DAY.ink} strokeWidth=".8" opacity=".2">
        <path d="M18 228v-5" />
        <path d="M24 228v-5" />
        <path d="M30 228v-5" />
        <path d="M36 228v-5" />
      </g>
    </Plate>
  );
}

/** Ready — «سپیده» First Light. The aperture is open; the sun has risen. */
export function TheiaReadyArt({ className = '', label = 'دریچه‌ای گشوده و خورشیدی که از افق بالا آمده است' }) {
  const uid = useUid('ready');
  const c = { x: 160, y: 196, r: 142 };
  return (
    <Plate viewBox="0 0 320 430" label={label} className={`theia-ready ${className}`.trim()}>
      <defs>
        <radialGradient id={`${uid}-glow`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#f6cf98" stopOpacity=".52" />
          <stop offset="42%" stopColor="#f0bf82" stopOpacity=".18" />
          <stop offset="100%" stopColor="#f0bf82" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`${uid}-sun`} cx="50%" cy="34%" r="64%">
          <stop offset="0%" stopColor="#fdf1da" />
          <stop offset="58%" stopColor="#f6cf98" />
          <stop offset="100%" stopColor="#dfa868" />
        </radialGradient>
        <mask id={`${uid}-half`}><rect x="0" y="0" width="320" height="301" fill="#fff" /></mask>
      </defs>
      <ellipse cx="160" cy="198" rx="176" ry="132" fill="none" stroke={DAY.ink} strokeWidth=".7" opacity=".09" transform="rotate(-18 160 198)" />
      <path d={ringPath(c.x, c.y, c.r, 106)} fill="none" stroke={DAY.ink} strokeWidth=".8" opacity=".2" strokeLinecap="round" />
      <path d={arc(c.x, c.y, c.r, 150, 262)} fill="none" stroke={DAY.brass} strokeWidth="1.15" opacity=".62" strokeLinecap="round" />
      <circle cx="160" cy="300" r="124" fill={`url(#${uid}-glow)`} className="theia-glow--soft" />
      <g mask={`url(#${uid}-half)`}>
        <circle cx="160" cy="300" r="30" fill={`url(#${uid}-sun)`} />
      </g>
      <Horizon y={300} color={DAY.ink} opacity={0.38} ticks={[40, 104, 216, 280]} />
      <g stroke={DAY.brass} strokeWidth=".8" opacity=".42" strokeLinecap="round">
        <path d="M160 258v-10" />
        <path d="M138 262l-5-8" />
        <path d="M182 262l5-8" />
        <path d="M122 275l-8-4" />
        <path d="M198 275l8-4" />
      </g>
      <g stroke={DAY.brass} strokeWidth=".8" opacity=".26" strokeLinecap="round">
        <path d="M140 314h40" />
        <path d="M148 326h24" />
        <path d="M154 338h12" />
      </g>
      <g className="theia-stars">
        <circle cx="86" cy="112" r="1.3" fill={DAY.lavender} />
        <circle cx="250" cy="84" r="1.1" fill={DAY.lavender} opacity=".8" />
        <circle cx="212" cy="146" r=".9" fill={DAY.ink} opacity=".26" />
      </g>
      <g stroke={DAY.ink} strokeWidth=".8" opacity=".18">
        <path d="M160 44v-7" />
        <path d="M4 198h8" />
        <path d="M316 198h8" />
      </g>
      <g stroke={DAY.ink} strokeWidth=".8" opacity=".18">
        <path d="M18 406v-5" />
        <path d="M24 406v-5" />
        <path d="M30 406v-5" />
        <path d="M36 406v-5" />
      </g>
    </Plate>
  );
}

/** Sign-in door art: the same window, held small and quiet. */
export function TheiaLoginArt({ className = '' }) {
  const uid = useUid('login');
  const c = { x: 170, y: 100, r: 78 };
  return (
    <svg
      viewBox="0 0 340 200"
      className={`theia-plate theia-login ${className}`.trim()}
      aria-hidden="true"
      focusable="false"
      preserveAspectRatio="xMidYMid meet"
    >
      <defs>
        <linearGradient id={`${uid}-night`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#1c1a15" />
          <stop offset="100%" stopColor="#0b0a08" />
        </linearGradient>
        <radialGradient id={`${uid}-glow`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#f6cf98" stopOpacity=".48" />
          <stop offset="42%" stopColor="#f0bf82" stopOpacity=".15" />
          <stop offset="100%" stopColor="#f0bf82" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`${uid}-sun`} cx="50%" cy="38%" r="62%">
          <stop offset="0%" stopColor="#fdf0d8" />
          <stop offset="66%" stopColor="#f6cf98" />
          <stop offset="100%" stopColor="#e2ab6b" />
        </radialGradient>
        <clipPath id={`${uid}-clip`}><circle cx={c.x} cy={c.y} r={c.r} /></clipPath>
      </defs>
      <ellipse cx="170" cy="104" rx="120" ry="86" fill="none" stroke={DAY.ink} strokeWidth=".7" opacity=".1" />
      <circle cx={c.x} cy={c.y} r={c.r} fill={`url(#${uid}-night)`} />
      <g clipPath={`url(#${uid}-clip)`}>
        <circle cx="176" cy="142" r="64" fill={`url(#${uid}-glow)`} />
        <path d="M91 152H249" stroke="#f3efe6" strokeWidth=".7" opacity=".38" />
        <circle cx="180" cy="152" r="13" fill={`url(#${uid}-sun)`} />
        <circle cx="180" cy="152" r="21" fill="none" stroke="#f6cf98" strokeWidth=".55" opacity=".2" />
        <circle cx="140" cy="66" r="1.1" fill="#d5d9f7" opacity=".8" />
        <circle cx="206" cy="52" r=".9" fill="#f3efe6" opacity=".4" />
      </g>
      <path d={ringPath(c.x, c.y, c.r, 34)} fill="none" stroke="#f3efe6" strokeWidth=".8" opacity=".14" />
      <path d={arc(c.x, c.y, c.r, 176, 262)} fill="none" stroke={DAY.brass} strokeWidth="1" opacity=".6" strokeLinecap="round" />
      <g stroke={DAY.ink} strokeWidth=".8" opacity=".2">
        <path d={`M${c.x} ${c.y - c.r - 7}v-6`} />
        <path d={`M${c.x - c.r - 8} ${c.y}h-6`} />
        <path d={`M${c.x + c.r + 8} ${c.y}h6`} />
      </g>
      <circle cx="66" cy="150" r="1.2" fill={DAY.ink} opacity=".28" />
      <circle cx="282" cy="132" r="1" fill={DAY.lavender} opacity=".8" />
      <g stroke={DAY.ink} strokeWidth=".8" opacity=".16">
        <path d="M18 180v-5" />
      </g>
    </svg>
  );
}

/**
 * Home emblem — «دریچهٔ زنده» The Living Aperture.
 * Preserves the Home orbit structure and breathing motion, without a figure.
 */
export function TheiaHomeArt() {
  const uid = useUid('home');
  return (
    <span className="home-conversation-art" aria-hidden="true">
      <svg className="home-conversation-orbit" viewBox="0 0 240 240" fill="none" focusable="false">
        <defs>
          <radialGradient id={`${uid}-glow`} cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#f6cf98" stopOpacity=".32" />
            <stop offset="44%" stopColor="#f0bf82" stopOpacity=".08" />
            <stop offset="100%" stopColor="#f0bf82" stopOpacity="0" />
          </radialGradient>
          <radialGradient id={`${uid}-sun`} cx="50%" cy="36%" r="62%">
            <stop offset="0%" stopColor="#fdf0d8" />
            <stop offset="66%" stopColor="#f6cf98" />
            <stop offset="100%" stopColor="#e2ab6b" />
          </radialGradient>
          <clipPath id={`${uid}-clip`}><circle cx="120" cy="122" r="64" /></clipPath>
        </defs>
        <g className="home-orbit-lines">
          <path className="home-orbit-trace home-orbit-trace--first" d="M197 54C168 24 121 22 88 43C47 69 31 116 48 158C61 190 88 208 118 213" />
          <path className="home-orbit-trace home-orbit-trace--second" d="M52 178C76 207 120 219 156 202C195 184 215 146 204 104C198 81 188 68 177 58" />
          <path className="home-orbit-accent" d="M64 92C79 65 104 51 131 49" />
          <circle className="home-orbit-point home-orbit-point--sun" cx="118" cy="213" r="2.6" />
          <circle className="home-orbit-point home-orbit-point--moon" cx="204" cy="104" r="2" />
        </g>
        <circle cx="120" cy="122" r="86" fill={`url(#${uid}-glow)`} />
        <circle cx="120" cy="122" r="80" fill="none" stroke="#f3efe6" strokeWidth=".7" opacity=".08" />
        <circle cx="120" cy="122" r="64" fill="#0b0a08" />
        <g clipPath={`url(#${uid}-clip)`}>
          <circle cx="130" cy="152" r="46" fill={`url(#${uid}-glow)`} />
          <path d="M56 152H184" stroke="#f3efe6" strokeWidth=".8" opacity=".34" />
          <circle cx="138" cy="152" r="11.5" fill={`url(#${uid}-sun)`} className="home-sun" />
          <circle cx="138" cy="152" r="20" fill="none" stroke="#f6cf98" strokeWidth=".55" opacity=".2" />
          <path d="M120 176h34" stroke="#f6cf98" strokeWidth=".6" opacity=".14" />
          <circle cx="96" cy="100" r="1.1" fill="#d5d9f7" opacity=".7" />
          <circle cx="146" cy="82" r=".9" fill="#f3efe6" opacity=".35" />
        </g>
        <path d={ringPath(120, 122, 64, 48)} fill="none" stroke="#f3efe6" strokeWidth="1.15" opacity=".62" strokeLinecap="round" />
        <path d={arc(120, 122, 64, 158, 266)} fill="none" stroke="#f6cf98" strokeWidth="1.5" opacity=".72" strokeLinecap="round" />
        <g stroke="#f3efe6" strokeWidth=".8" opacity=".22">
          <path d="M120 46v-6" />
          <path d="M42 122h-6" />
          <path d="M198 122h6" />
        </g>
      </svg>
    </span>
  );
}

/** Quiet empty-inbox illustration for Letters. */
export function TheiaLetterEmpty({ size = 44 }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 48 48"
      fill="none"
      className="theia-letter-empty"
      aria-hidden="true"
      focusable="false"
    >
      <path
        d="M9 18.5C9 16.57 10.57 15 12.5 15H35.5C37.43 15 39 16.57 39 18.5V33.5C39 35.43 37.43 37 35.5 37H12.5C10.57 37 9 35.43 9 33.5V18.5Z"
        stroke="currentColor"
        strokeWidth="1.3"
      />
      <path d="M10.5 17L24 26.6L37.5 17" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M33.5 4.5A5 5 0 0 1 38.5 9.5" stroke="#f6cf98" strokeWidth="1.2" strokeLinecap="round" opacity=".8" />
      <path d="M17.5 12.6H21" stroke="currentColor" strokeWidth=".8" opacity=".3" />
      <circle cx="19.2" cy="15.6" r="1" fill="#f6cf98" opacity=".85" />
    </svg>
  );
}
