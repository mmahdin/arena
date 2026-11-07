import React, { useId } from 'react';

/**
 * THEIA — brand primitives.
 *
 * One idea holds the identity together: light arriving through an opening.
 * The mark is an incomplete, brush-drawn ring with a single spark resting in
 * its gap. The illustrations are quiet "light studies" — horizons, veils, still
 * water and a small faceless figure walking toward that opening. Nothing is a
 * sun, an eye, a flower or a temple; every shape is drawn for Theia alone.
 */

export const THEIA_COLORS = Object.freeze({
  ink: '#100F0C',
  inkSoft: '#1A1713',
  paper: '#F2EDE3',
  sand: '#E6DAC6',
  ember: '#E09A5F',
  emberDeep: '#A9532B',
  verdigris: '#7E9A8C',
});

/* ---------- Palettes (art is independent of UI surface tokens) ---------- */

const NIGHT = {
  bg1: '#0B0A08', bg2: '#16130E',
  line: 'rgba(238,229,212,.22)', lineSoft: 'rgba(238,229,212,.10)',
  cool: 'rgba(150,178,164,.20)',
  core: '#FFE9CC', glow: '#DE9C62',
  dust: 'rgba(248,231,206,.6)', figure: '#141210', rim: 'rgba(232,186,134,.65)',
  ground: '#090908', shadow: 'rgba(0,0,0,.55)',
};

const PAPER = {
  bg1: 'rgba(0,0,0,0)', bg2: 'rgba(0,0,0,0)',
  line: 'rgba(36,30,24,.24)', lineSoft: 'rgba(36,30,24,.10)',
  cool: 'rgba(74,104,90,.26)',
  core: '#FCE3C2', glow: '#D68F55',
  dust: 'rgba(96,79,60,.4)', figure: '#2B251E', rim: 'rgba(190,140,92,.8)',
  ground: 'rgba(36,30,24,.05)', shadow: 'rgba(70,52,34,.18)',
};

const SAND = {
  ...PAPER,
  line: 'rgba(52,40,26,.24)', lineSoft: 'rgba(52,40,26,.10)',
  ground: 'rgba(70,52,30,.06)', shadow: 'rgba(80,58,34,.2)',
};

const DAWN = {
  bg1: '#FBF2E4', bg2: '#EBD6B8',
  line: 'rgba(118,80,44,.26)', lineSoft: 'rgba(118,80,44,.13)',
  cool: 'rgba(120,132,110,.2)',
  core: '#FFF8EC', glow: '#EFA96C',
  dust: 'rgba(160,110,62,.45)', figure: '#3A2A1B', rim: 'rgba(255,236,206,.95)',
  ground: 'rgba(140,98,52,.10)', shadow: 'rgba(120,84,44,.22)',
};

/* ---------- The mark: a brush ring, open at one point ---------- */

function ringPath({ cx = 24, cy = 24, radius = 15.6, gapFrom = -64, gapTo = -6, maxWeight = 3.1 }) {
  // Draw the long way round: the gap is what is left open.
  // A filled, tapered arc — a drawn stroke rather than a geometric circle.
  const steps = 64;
  const outer = [];
  const inner = [];
  for (let i = 0; i <= steps; i += 1) {
    const t = i / steps;
    const angle = ((gapTo + (gapFrom + 360 - gapTo) * t) * Math.PI) / 180;
    const weight = 0.34 + 0.66 * Math.sin(Math.PI * t) ** 0.6;
    const half = (maxWeight * weight) / 2;
    // A drawn circle breathes: its radius is never perfectly constant.
    const wobble = 1 + 0.013 * Math.sin(angle * 2.1 + 0.6) + 0.006 * Math.sin(angle * 3.7);
    const x = Math.cos(angle);
    const y = Math.sin(angle);
    outer.push([cx + (radius * wobble + half) * x, cy + (radius * wobble + half) * y]);
    inner.push([cx + (radius * wobble - half) * x, cy + (radius * wobble - half) * y]);
  }
  const toPath = (points) => points.map(([x, y], i) => `${i ? 'L' : 'M'}${x.toFixed(2)} ${y.toFixed(2)}`).join(' ');
  return `${toPath(outer)} ${toPath(inner.reverse()).replace('M', 'L')} Z`;
}

export function TheiaMark({ size = 26, className = '', glow = false, color = 'currentColor', spark = THEIA_COLORS.ember }) {
  const id = useId().replace(/[^a-zA-Z0-9_-]/g, '');
  return (
    <svg
      className={`theia-mark ${className}`}
      width={size}
      height={size}
      viewBox="0 0 48 48"
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <radialGradient id={`mark-${id}`}>
          <stop offset="0%" stopColor={spark} stopOpacity=".38" />
          <stop offset="55%" stopColor={spark} stopOpacity=".1" />
          <stop offset="100%" stopColor={spark} stopOpacity="0" />
        </radialGradient>
      </defs>
      {glow && <circle cx="24" cy="24" r="21" fill={`url(#mark-${id})`} />}
      <path d={ringPath({})} fill={color} />
      <circle cx="38.4" cy="12.6" r="2.35" fill={spark} />
    </svg>
  );
}

/** Brand wordmark. Naskh gives the Persian name a literary, unhurried voice. */
export function TheiaWordmark({ size = 22, className = '', spark = true, tone }) {
  return (
    <span className={`theia-wordmark ${tone ? `theia-wordmark--${tone}` : ''} ${className}`} style={{ fontSize: `${size}px` }}>
      <span className="theia-wordmark__name">تیا</span>
      {spark && <i className="theia-wordmark__spark" aria-hidden="true" />}
    </span>
  );
}

/* ---------- Shared art scaffolding ---------- */

function Palette({ id, p }) {
  return (
    <defs>
      <linearGradient id={`${id}-sky`} x1="0" y1="0" x2=".25" y2="1">
        <stop offset="0%" stopColor={p.bg1} />
        <stop offset="100%" stopColor={p.bg2} />
      </linearGradient>
      <radialGradient id={`${id}-haze`}>
        <stop offset="0%" stopColor={p.glow} stopOpacity=".34" />
        <stop offset="45%" stopColor={p.glow} stopOpacity=".12" />
        <stop offset="100%" stopColor={p.glow} stopOpacity="0" />
      </radialGradient>
      <radialGradient id={`${id}-bloom`}>
        <stop offset="0%" stopColor={p.core} stopOpacity=".9" />
        <stop offset="34%" stopColor={p.glow} stopOpacity=".45" />
        <stop offset="70%" stopColor={p.glow} stopOpacity=".12" />
        <stop offset="100%" stopColor={p.glow} stopOpacity="0" />
      </radialGradient>
      <radialGradient id={`${id}-disc`}>
        <stop offset="0%" stopColor="#FFFDF7" stopOpacity="1" />
        <stop offset="34%" stopColor={p.core} stopOpacity="1" />
        <stop offset="68%" stopColor={p.glow} stopOpacity=".62" />
        <stop offset="100%" stopColor={p.glow} stopOpacity="0" />
      </radialGradient>
      <radialGradient id={`${id}-shadow`}>
        <stop offset="0%" stopColor={p.shadow} stopOpacity=".95" />
        <stop offset="60%" stopColor={p.shadow} stopOpacity=".35" />
        <stop offset="100%" stopColor={p.shadow} stopOpacity="0" />
      </radialGradient>
      <linearGradient id={`${id}-veilWarm`} x1="0" y1="0" x2="1" y2=".15">
        <stop offset="0%" stopColor={p.glow} stopOpacity="0" />
        <stop offset="48%" stopColor={p.glow} stopOpacity=".38" />
        <stop offset="100%" stopColor={p.glow} stopOpacity="0" />
      </linearGradient>
      <linearGradient id={`${id}-veilCool`} x1="0" y1="0" x2="1" y2=".15">
        <stop offset="0%" stopColor={p.cool} stopOpacity="0" />
        <stop offset="52%" stopColor={p.cool} stopOpacity=".5" />
        <stop offset="100%" stopColor={p.cool} stopOpacity="0" />
      </linearGradient>
      <radialGradient id={`${id}-vignette`} cx="50%" cy="42%" r="74%">
        <stop offset="52%" stopColor={p.shadow} stopOpacity="0" />
        <stop offset="100%" stopColor={p.shadow} stopOpacity=".75" />
      </radialGradient>
      <filter id={`${id}-soft`} x="-50%" y="-50%" width="200%" height="200%">
        <feGaussianBlur stdDeviation="7" />
      </filter>
      <filter id={`${id}-grain`}>
        <feTurbulence type="fractalNoise" baseFrequency=".86" numOctaves="3" stitchTiles="stitch" />
        <feColorMatrix type="saturate" values="0" />
      </filter>
    </defs>
  );
}

function Grain({ id, opacity = 0.055 }) {
  return <rect width="360" height="400" filter={`url(#${id}-grain)`} opacity={opacity} style={{ mixBlendMode: 'overlay' }} aria-hidden="true" />;
}

/** A light source with an edge: haze, bloom, disc, rim. Never a formless lens flare. */
function Light({ id, cx, cy, r = 14, cool = false }) {
  const core = cool ? '#EAF2EC' : '#FFF8EC';
  return (
    <g className="study__light">
      <circle cx={cx} cy={cy} r={r * 6.4} fill={`url(#${id}-haze)`} />
      <circle cx={cx} cy={cy} r={r * 2.5} fill={`url(#${id}-bloom)`} />
      <circle cx={cx} cy={cy} r={r * 1.45} fill={`url(#${id}-disc)`} />
      <circle cx={cx} cy={cy} r={r * 0.5} fill="#FFFDF7" opacity=".9" />
      <circle cx={cx} cy={cy} r={r * 0.22} fill="#FFFFFF" opacity=".95" />
    </g>
  );
}

function Dust({ points, color }) {
  return (
    <g className="study__dust" aria-hidden="true">
      {points.map(([cx, cy, r, o], index) => (
        <circle key={index} cx={cx} cy={cy} r={r} fill={color} opacity={o} />
      ))}
    </g>
  );
}

/** The small faceless presence that walks through the whole onboarding story. */
function Figure({ x, y, s = 1, p, walking = false, lit = 0, reflection = false, shadow = true }) {
  const body = (
    <>
      {lit > 0 && (
        <g fill={p.glow} opacity={Math.min(1, lit)}>
          <ellipse cx="0" cy="-2" rx={40 * lit} ry={46 * lit} opacity=".07" />
          <ellipse cx="0" cy="-2" rx={26 * lit} ry={32 * lit} opacity=".11" />
          <ellipse cx="0" cy="-2" rx={15 * lit} ry={19 * lit} opacity=".18" />
        </g>
      )}
      <circle cx="1.2" cy="-31.4" r="5.4" fill={p.figure} />
      <path
        d="M0.8 -26.4 C 5.6 -26.2 8.8 -23 10.4 -17.8 C 12.4 -11 13.4 -4 13.8 3.4 L 15 32.4 C 15.2 34.9 13.6 36.2 11.2 36.2 L -11.4 36.2 C -13.8 36.2 -15.4 34.9 -15.2 32.4 L -14.2 3.4 C -13.8 -4 -12.8 -11 -10.8 -17.8 C -9.2 -23 -5.6 -26.4 -0.4 -26.4 Z"
        fill={p.figure}
      />
      <path
        d="M-3.6 -35 C -6.8 -32 -7.8 -28 -7 -24 C -10 -17.4 -12 -9.6 -12.4 -1.6 L -13.2 22"
        fill="none"
        stroke={p.rim}
        strokeWidth="1.45"
        strokeLinecap="round"
        opacity=".5"
      />
      <path d="M-1.6 -36.3 A5.4 5.4 0 0 1 5 -33" fill="none" stroke={p.rim} strokeWidth="1.15" strokeLinecap="round" opacity=".65" />
    </>
  );
  return (
    <g transform={`translate(${x} ${y}) scale(${s})`} aria-hidden="true">
      {shadow && <ellipse cx="2" cy="38" rx="30" ry="7" fill={p.shadow} opacity=".3" />}
      {reflection && (
        <g transform="translate(0 2) scale(1 -0.66) translate(0 -38)" opacity=".18">
          {body}
        </g>
      )}
      <g className={walking ? 'study__figure study__figure--walking' : 'study__figure'} transform={walking ? 'rotate(-2.8)' : undefined}>
        {body}
      </g>
    </g>
  );
}

/* ---------- Scenes ---------- */

function WelcomeScene({ id, p }) {
  return (
    <>
      <rect width="360" height="400" fill={`url(#${id}-sky)`} />
      <Light id={id} cx={246} cy={112} r={11} />
      <g fill="none" strokeLinecap="round">
        <path d="M-20 214 C 54 188 108 204 156 182 C 210 158 268 168 380 132" stroke={p.lineSoft} strokeWidth=".9" />
        <path d="M-20 252 C 62 226 120 244 174 220 C 234 194 292 204 380 172" stroke={p.lineSoft} strokeWidth="1" />
        <path d="M-20 298 C 70 272 130 290 190 266 C 250 242 306 254 380 222" stroke={p.line} strokeWidth="1.1" />
        <path d="M-20 338 C 84 314 146 332 208 308 C 268 284 318 298 380 268" stroke={p.line} strokeWidth="1.25" />
      </g>
      <path d="M0 326 C 74 308 134 326 196 306 C 254 288 314 304 360 286 L360 400 L0 400 Z" fill={p.ground} opacity=".95" />
      <path d="M0 330 C 74 312 134 330 196 310 C 254 292 314 308 360 290" fill="none" stroke={p.lineSoft} strokeWidth="1.1" />
      <Figure x={96} y={322} s={0.62} p={p} />
      <Dust color={p.dust} points={[
        [72, 190, 1.1, .45], [126, 158, .9, .32], [168, 224, 1.3, .26], [206, 138, .8, .4],
        [288, 182, 1, .28], [318, 122, 1.1, .36], [56, 258, .8, .26], [232, 244, .9, .24],
        [148, 104, .8, .2], [334, 226, .8, .22],
      ]} />
      <rect width="360" height="400" fill={`url(#${id}-vignette)`} />
      <Grain id={id} />
    </>
  );
}

function SimpleScene({ id, p }) {
  // The first steps: a path already drawn, a small light waiting at its end.
  return (
    <>
      <rect width="360" height="400" fill={`url(#${id}-sky)`} />
      <Light id={id} cx={112} cy={244} r={10} />
      <g fill="none" strokeLinecap="round">
        <path d="M-20 150 C 56 132 116 152 182 136 C 246 120 306 136 380 116" stroke={p.lineSoft} strokeWidth=".9" />
        <path d="M-20 194 C 58 176 120 196 188 178 C 250 162 308 178 380 156" stroke={p.lineSoft} strokeWidth=".9" opacity=".8" />
      </g>
      {/* one continuous path, from where the walker stands toward the light */}
      <path
        d="M352 402 C 320 356 330 330 286 312 C 246 296 232 286 196 272 C 164 260 138 254 92 250"
        fill="none"
        stroke={p.line}
        strokeWidth="1.35"
        strokeLinecap="round"
      />
      <path
        d="M352 402 C 320 356 330 330 286 312 C 246 296 232 286 196 272 C 164 260 138 254 92 250"
        fill="none"
        stroke={p.lineSoft}
        strokeWidth="6"
        strokeLinecap="round"
        opacity=".18"
      />
      <g fill={p.dust}>
        <circle cx="318" cy="372" r="1.6" opacity=".42" />
        <circle cx="300" cy="334" r="1.3" opacity=".36" />
        <circle cx="264" cy="306" r="1.5" opacity=".3" />
        <circle cx="216" cy="280" r="1.2" opacity=".26" />
        <circle cx="170" cy="264" r="1.1" opacity=".22" />
      </g>
      <Figure x={300} y={366} s={0.66} p={p} walking />
      <Dust color={p.dust} points={[
        [78, 196, 1, .3], [152, 168, .8, .24], [206, 212, 1.1, .24], [268, 178, .9, .28],
        [326, 210, .8, .2], [54, 288, .9, .22], [240, 128, .8, .18],
      ]} />
      <rect width="360" height="400" fill={`url(#${id}-vignette)`} />
      <Grain id={id} />
    </>
  );
}

function DeeperScene({ id, p }) {
  return (
    <>
      <rect width="360" height="400" fill={`url(#${id}-sky)`} />
      <Light id={id} cx={158} cy={182} r={19} />
      {/* veils pass in front of the light: the world is only partly revealed */}
      <g fill="none" strokeLinecap="round">
        <path d="M-20 92 C 54 74 112 96 176 82 C 240 68 302 82 380 60" stroke={p.lineSoft} strokeWidth=".9" />
        <path d="M-20 134 C 48 118 106 138 176 120 C 244 102 306 118 380 96" stroke={p.line} strokeWidth="1" />
        <path d="M-20 186 C 50 168 108 190 180 170 C 250 150 308 168 380 142" stroke={`url(#${id}-veilWarm)`} strokeWidth="7" opacity=".55" filter={`url(#${id}-soft)`} />
        <path d="M-20 186 C 50 168 108 190 180 170 C 250 150 308 168 380 142" stroke={p.line} strokeWidth="1.05" />
        <path d="M-20 232 C 46 214 102 236 176 214 C 248 192 308 210 380 182" stroke={p.line} strokeWidth="1.1" />
        <path d="M-20 268 C 48 250 104 272 176 250 C 248 228 306 246 380 218" stroke={`url(#${id}-veilCool)`} strokeWidth="10" opacity=".5" filter={`url(#${id}-soft)`} />
        <path d="M-20 286 C 52 268 112 290 188 266 C 258 244 312 264 380 236" stroke={p.lineSoft} strokeWidth="1" />
        <path d="M-20 330 C 60 312 122 334 196 310 C 266 288 318 308 380 280" stroke={p.lineSoft} strokeWidth="1.1" />
      </g>
      <path d="M0 340 C 72 322 132 342 200 320 C 264 300 320 318 360 302 L360 400 L0 400 Z" fill={p.ground} opacity=".96" />
      <Figure x={244} y={344} s={1.16} p={p} lit={.5} />
      <Dust color={p.dust} points={[
        [72, 158, 1.1, .3], [116, 108, .9, .24], [212, 120, .8, .26], [286, 168, 1, .22],
        [44, 226, .9, .22], [326, 234, .9, .2], [256, 92, .8, .18],
      ]} />
      <rect width="360" height="400" fill={`url(#${id}-vignette)`} />
      <Grain id={id} />
    </>
  );
}

function KnowScene({ id, p }) {
  return (
    <>
      <defs>
        <linearGradient id={`${id}-water`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#1B201C" stopOpacity=".95" />
          <stop offset="100%" stopColor="#0B0C0A" stopOpacity="1" />
        </linearGradient>
      </defs>
      <rect width="360" height="400" fill={`url(#${id}-sky)`} />
      <Light id={id} cx={214} cy={172} r={15} cool />
      <g fill="none" strokeLinecap="round" stroke={p.lineSoft} strokeWidth=".9">
        <path d="M-20 106 C 54 92 108 110 168 94 C 232 78 296 92 380 68" />
        <path d="M-20 140 C 58 126 112 144 174 126 C 238 108 298 124 380 100" />
      </g>
      <rect y="244" width="360" height="156" fill={`url(#${id}-water)`} />
      <path d="M0 244 L360 244" stroke={p.line} strokeWidth="1.1" />
      {/* broken reflection: the light is seen, never mirrored whole */}
      <g fill={p.glow} className="study__reflection">
        {[
          [214, 254, 26, 1.9, .5], [214, 262, 20, 1.6, .4], [214, 273, 27, 1.4, .32],
          [214, 284, 16, 1.2, .26], [214, 297, 22, 1.1, .2], [214, 312, 13, 1, .16],
          [214, 328, 18, .9, .12], [214, 346, 11, .8, .09],
        ].map(([cx, cy, rx, ry, o], index) => (
          <ellipse key={index} cx={cx} cy={cy} rx={rx} ry={ry} opacity={o} />
        ))}
      </g>
      <g fill="none" stroke={p.lineSoft} strokeWidth="1">
        <ellipse cx="214" cy="270" rx="58" ry="7" />
        <ellipse cx="214" cy="292" rx="88" ry="10" opacity=".7" />
        <ellipse cx="214" cy="324" rx="120" ry="13" opacity=".45" />
      </g>
      <Figure x={92} y={248} s={0.8} p={p} reflection />
      <Dust color={p.dust} points={[
        [64, 186, 1, .28], [142, 152, .8, .24], [272, 124, 1.1, .3], [316, 190, .9, .22],
        [148, 216, .9, .2], [286, 214, .8, .18],
      ]} />
      <rect width="360" height="400" fill={`url(#${id}-vignette)`} />
      <Grain id={id} />
    </>
  );
}

function AlmostScene({ id, p }) {
  return (
    <>
      <rect width="360" height="400" fill={`url(#${id}-sky)`} />
      <ellipse cx="196" cy="196" rx="200" ry="180" fill={`url(#${id}-haze)`} />
      {/* the opening, now as large as the world */}
      <g fill="none" strokeLinecap="round">
        <path d="M-24 244 C 56 222 120 244 186 224 C 250 204 314 222 384 196" stroke={p.lineSoft} strokeWidth="1" />
        <path d="M-24 292 C 64 270 130 292 200 270 C 266 250 322 268 384 242" stroke={p.lineSoft} strokeWidth="1" />
      </g>
      <g className="study__light">
        <circle cx="286" cy="118" r="92" fill={`url(#${id}-bloom)`} />
        <circle cx="286" cy="118" r="25" fill={`url(#${id}-disc)`} />
        <circle cx="286" cy="118" r="6" fill="#FFFDF7" opacity=".7" />
      </g>
      <g fill="none">
        <path d={ringPath({ cx: 196, cy: 200, radius: 122, gapFrom: -56, gapTo: -20, maxWeight: 4.6 })} fill={p.lineSoft} />
        <path d="M-20 344 C 62 322 128 344 198 324 C 266 304 322 322 384 298" stroke={p.lineSoft} strokeWidth="1" />
      </g>
      <circle cx="316" cy="76" r="2.4" fill={p.core} className="study__spark" />
      <path d="M0 356 C 78 338 140 358 204 340 C 266 322 322 340 360 326 L360 400 L0 400 Z" fill={p.ground} opacity=".96" />
      <Figure x={196} y={354} s={0.92} p={p} lit={.95} walking />
      <Dust color={p.dust} points={[
        [92, 128, 1.2, .38], [148, 88, .9, .26], [252, 260, 1, .26], [292, 300, .9, .22],
        [52, 216, 1, .24], [330, 232, 1, .22],
      ]} />
      <rect width="360" height="400" fill={`url(#${id}-vignette)`} />
      <Grain id={id} />
    </>
  );
}

function ReadyScene({ id, p }) {
  return (
    <>
      <defs>
        <linearGradient id={`${id}-dawn`} x1=".1" y1="0" x2=".4" y2="1">
          <stop offset="0%" stopColor="#F7EAD6" />
          <stop offset="46%" stopColor="#F3DFC3" />
          <stop offset="100%" stopColor="#E2C49E" />
        </linearGradient>
        <linearGradient id={`${id}-land`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="rgba(186,132,76,.22)" />
          <stop offset="100%" stopColor="rgba(186,132,76,.04)" />
        </linearGradient>
      </defs>
      <rect width="360" height="400" fill={`url(#${id}-dawn)`} />
      <ellipse cx="196" cy="250" rx="210" ry="150" fill={`url(#${id}-haze)`} />
      {/* a low band of light where the world meets the sky */}
      <g className="study__light">
        <circle cx="196" cy="262" r="52" fill={`url(#${id}-disc)`} />
        <ellipse cx="196" cy="262" rx="150" ry="16" fill={`url(#${id}-bloom)`} opacity=".9" />
      </g>
      <g fill="none" strokeLinecap="round">
        <path d="M-10 128 C 62 108 118 132 180 112 C 244 92 302 110 380 82" stroke={p.lineSoft} strokeWidth="1" />
        <path d="M-10 170 C 66 150 120 174 184 154 C 248 134 304 152 380 124" stroke={p.lineSoft} strokeWidth=".9" />
        <path d="M-10 214 C 70 194 126 216 190 196 C 254 176 308 194 380 168" stroke={p.line} strokeWidth="1" />
        <path d="M-10 300 C 74 282 132 302 196 282 C 258 262 314 280 380 254" stroke={p.line} strokeWidth="1.15" />
        <path d="M-10 334 C 86 316 148 336 212 314 C 272 294 320 312 380 286" stroke={p.lineSoft} strokeWidth="1" />
      </g>
      <path d="M0 282 C 80 264 140 286 204 266 C 264 248 318 266 360 250 L360 400 L0 400 Z" fill={`url(#${id}-land)`} />
      <path d="M0 286 C 80 268 140 290 204 270 C 264 252 318 270 360 254" fill="none" stroke={p.line} strokeWidth="1.1" />
      <Figure x={196} y={316} s={1} p={p} walking lit={1.1} shadow={false} />
      <ellipse cx="200" cy="326" rx="34" ry="6" fill="rgba(120,80,40,.12)" />
      <Dust color={p.dust} points={[
        [88, 176, 1.2, .4], [128, 216, .9, .3], [250, 156, 1, .3], [300, 210, .9, .26],
        [58, 252, .8, .24], [322, 268, 1, .2], [212, 130, .8, .26], [142, 96, .8, .2],
      ]} />
      <Grain id={id} opacity={.04} />
    </>
  );
}

function PrivacyScene({ id, p }) {
  // A light held in a cupped contour. Held, not locked.
  return (
    <>
      <ellipse cx="180" cy="208" rx="150" ry="126" fill={`url(#${id}-haze)`} />
      <g fill="none" strokeLinecap="round">
        <path d="M92 168 C 78 224 116 274 178 290 C 240 274 274 226 258 172" stroke={p.line} strokeWidth="1.15" />
        <path d="M107 184 C 96 226 128 262 178 276 C 226 262 256 228 244 186" stroke={p.line} strokeWidth="1" strokeOpacity=".78" />
        <path d="M122 200 C 114 228 140 250 178 260 C 214 250 240 230 232 202" stroke={p.line} strokeWidth=".9" strokeOpacity=".58" />
      </g>
      <g className="study__light">
        <circle cx="176" cy="196" r="48" fill={`url(#${id}-bloom)`} />
        <circle cx="176" cy="192" r="20" fill={`url(#${id}-disc)`} />
        <circle cx="176" cy="192" r="5" fill="#FFFDF7" opacity=".85" />
      </g>
      <g fill="none" stroke={p.lineSoft} strokeWidth="1">
        <path d="M78 300 C 118 288 158 302 196 290" />
        <path d="M120 318 C 152 310 180 320 210 310" opacity=".7" />
      </g>
      <circle cx="250" cy="152" r="2" fill={p.dust} opacity=".5" />
      <circle cx="108" cy="246" r="1.6" fill={p.dust} opacity=".4" />
      <Dust color={p.dust} points={[[96, 142, .9, .26], [268, 214, .8, .22], [60, 200, .8, .18]]} />
      <Grain id={id} opacity={.03} />
    </>
  );
}

function CheckinsScene({ id, p }) {
  // A quiet signal: three arcs leave a small light, each fainter than the last.
  return (
    <>
      <ellipse cx="196" cy="200" rx="146" ry="126" fill={`url(#${id}-haze)`} />
      <g className="study__light">
        <circle cx="196" cy="196" r="46" fill={`url(#${id}-bloom)`} />
        <circle cx="196" cy="196" r="24" fill={`url(#${id}-disc)`} />
        <circle cx="196" cy="196" r="5" fill="#FFFDF7" opacity=".9" />
      </g>
      <g fill="none" strokeLinecap="round">
        <path d="M236 158 A52 52 0 0 1 250 196 A52 52 0 0 1 236 234" stroke={p.line} strokeWidth="1.4" />
        <path d="M258 136 A84 84 0 0 1 276 196 A84 84 0 0 1 258 256" stroke={p.line} strokeWidth="1.15" strokeOpacity=".8" />
        <path d="M282 112 A120 120 0 0 1 302 196 A120 120 0 0 1 282 280" stroke={p.lineSoft} strokeWidth="1.1" />
      </g>
      <g fill="none" stroke={p.lineSoft} strokeWidth="1">
        <path d="M74 168 C 108 158 136 168 158 162" />
        <path d="M60 300 C 104 288 148 300 192 288" />
      </g>
      <g fill={p.dust}>
        <circle cx="196" cy="106" r="2.2" opacity=".5" />
        <circle cx="196" cy="286" r="2" opacity=".42" />
        <circle cx="126" cy="160" r="1.7" opacity=".36" />
        <circle cx="118" cy="234" r="1.5" opacity=".3" />
      </g>
      <g fill="none" stroke={p.lineSoft} strokeWidth="1">
        <path d="M60 320 C 108 306 156 320 204 306 C 240 296 268 304 300 294" />
      </g>
      <Grain id={id} opacity={.03} />
    </>
  );
}

function HorizonScene({ id, p }) {
  return (
    <>
      <ellipse cx="234" cy="214" rx="160" ry="132" fill={`url(#${id}-haze)`} />
      <g className="study__light">
        <circle cx="234" cy="228" r="22" fill={`url(#${id}-disc)`} />
        <ellipse cx="234" cy="228" rx="86" ry="12" fill={`url(#${id}-bloom)`} opacity=".75" />
      </g>
      <path d="M0 238 L360 238" stroke={p.line} strokeWidth="1.1" />
      <g fill="none" strokeLinecap="round">
        <path d="M-10 160 C 60 144 112 164 170 148 C 232 130 290 148 372 120" stroke={p.lineSoft} strokeWidth="1" />
        <path d="M-10 196 C 66 180 118 200 178 184 C 240 166 298 184 372 158" stroke={p.lineSoft} strokeWidth=".9" />
        <path d="M-10 292 C 70 274 128 294 194 274 C 256 256 312 272 372 246" stroke={p.line} strokeWidth="1" />
        <path d="M-10 326 C 82 308 144 328 208 306 C 268 288 318 304 372 280" stroke={p.lineSoft} strokeWidth="1" />
      </g>
      <path d="M0 300 C 80 284 140 304 204 286 C 264 268 318 284 360 268 L360 400 L0 400 Z" fill={p.ground} opacity=".85" />
      <Figure x={104} y={292} s={0.7} p={p} />
      <Dust color={p.dust} points={[[86, 168, .9, .28], [158, 118, .8, .22], [300, 158, 1, .26], [64, 228, .8, .2], [322, 264, .9, .18]]} />
      <Grain id={id} opacity={.03} />
    </>
  );
}

const SCENES = {
  welcome: { palette: NIGHT, draw: WelcomeScene },
  simple: { palette: NIGHT, draw: SimpleScene },
  deeper: { palette: NIGHT, draw: DeeperScene },
  know: { palette: NIGHT, draw: KnowScene },
  almost: { palette: NIGHT, draw: AlmostScene },
  ready: { palette: DAWN, draw: ReadyScene },
  privacy: { palette: PAPER, draw: PrivacyScene },
  checkins: { palette: PAPER, draw: CheckinsScene },
  horizon: { palette: SAND, draw: HorizonScene },
};

/** A single scene from the onboarding story, drawn on a 360 x 400 canvas. */
export function LightStudy({ scene = 'welcome', className = '' }) {
  const id = useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const config = SCENES[scene] || SCENES.welcome;
  const Draw = config.draw;
  return (
    <svg
      className={`light-study light-study--${scene} ${className}`}
      viewBox="0 0 360 400"
      role="img"
      aria-hidden="true"
      focusable="false"
      preserveAspectRatio="xMidYMid meet"
    >
      <Palette id={id} p={config.palette} />
      <Draw id={id} p={config.palette} />
    </svg>
  );
}

/* ---------- Mood: a light that opens ---------- */

const MOOD_SHAPES = [
  { turns: 3.35, start: 8, end: 54, wobble: .13, bloom: .32, warmth: 0, driftX: 0, driftY: 0, squash: .8 },
  { turns: 2.85, start: 9, end: 66, wobble: .1, bloom: .48, warmth: .25, driftX: 5, driftY: -4, squash: .82 },
  { turns: 2.3, start: 10, end: 80, wobble: .07, bloom: .66, warmth: .5, driftX: 9, driftY: -9, squash: .85 },
  { turns: 1.72, start: 12, end: 96, wobble: .045, bloom: .84, warmth: .78, driftX: 14, driftY: -15, squash: .88 },
  { turns: 1.28, start: 14, end: 114, wobble: .028, bloom: 1, warmth: 1, driftX: 18, driftY: -20, squash: .92 },
];

function moodColor(warmth, alpha) {
  const cool = [116, 148, 134];
  const warm = [222, 152, 93];
  const rgb = cool.map((value, index) => Math.round(value + (warm[index] - value) * warmth));
  return `rgba(${rgb[0]},${rgb[1]},${rgb[2]},${alpha})`;
}

/**
 * One line enters the frame, knots around a small light, and leaves.
 * The knot loosens and the light warms as the feeling opens — same gesture,
 * five degrees of clarity.
 */
function moodLine(shape) {
  const cx = 176 + shape.driftX * 0.4;
  const cy = 196 + shape.driftY * 0.4;
  const theta0 = -Math.PI * 0.62;
  const points = [];
  const spiralStart = [cx + shape.start * Math.cos(theta0) * 1.04, cy + shape.start * Math.sin(theta0) * shape.squash];
  const enter = [14, 146 + shape.driftY * 0.25];
  for (let i = 0; i <= 20; i += 1) {
    const t = i / 20;
    const ease = t * t * (3 - 2 * t);
    points.push([
      enter[0] + (spiralStart[0] - enter[0]) * t,
      enter[1] + (spiralStart[1] - enter[1]) * ease + Math.sin(t * Math.PI) * 12,
    ]);
  }
  const steps = 200;
  for (let i = 1; i <= steps; i += 1) {
    const t = i / steps;
    const theta = theta0 + t * shape.turns * Math.PI * 2;
    const radius = shape.start + (shape.end - shape.start) * t ** 1.02;
    const ripple = 1 + shape.wobble * Math.sin(theta * 1.3 + 0.7);
    points.push([
      cx + radius * ripple * Math.cos(theta) * 1.04,
      cy + radius * ripple * Math.sin(theta) * shape.squash,
    ]);
  }
  const last = points[points.length - 1];
  const exit = [346, 238 + shape.driftY * 0.35];
  for (let i = 1; i <= 20; i += 1) {
    const t = i / 20;
    const ease = t * t * (3 - 2 * t);
    points.push([
      last[0] + (exit[0] - last[0]) * t,
      last[1] + (exit[1] - last[1]) * ease - Math.sin(t * Math.PI) * 10,
    ]);
  }
  return `M${points.map(([x, y]) => `${x.toFixed(1)} ${y.toFixed(1)}`).join(' L')}`;
}

/** Mood is shown as a single line loosening around the light. */
export function MoodLight({ level = 2, className = '' }) {
  const id = useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const shape = MOOD_SHAPES[Math.max(0, Math.min(4, level))];
  const path = React.useMemo(() => moodLine(shape), [shape]);
  const lx = 176 + shape.driftX * 0.4;
  const ly = 196 + shape.driftY * 0.4;
  return (
    <svg className={`mood-light ${className}`} viewBox="0 0 360 400" role="img" aria-hidden="true" focusable="false">
      <defs>
        <radialGradient id={`mood-haze-${id}`}>
          <stop offset="0%" stopColor={moodColor(shape.warmth, .38)} stopOpacity="1" />
          <stop offset="55%" stopColor={moodColor(shape.warmth, .11)} stopOpacity="1" />
          <stop offset="100%" stopColor={moodColor(shape.warmth, 0)} stopOpacity="1" />
        </radialGradient>
        <radialGradient id={`mood-core-${id}`}>
          <stop offset="0%" stopColor="#FFFBF2" stopOpacity=".95" />
          <stop offset="38%" stopColor={moodColor(shape.warmth, .7)} stopOpacity=".8" />
          <stop offset="100%" stopColor={moodColor(shape.warmth, 0)} stopOpacity="0" />
        </radialGradient>
        <filter id={`mood-grain-${id}`}>
          <feTurbulence type="fractalNoise" baseFrequency=".86" numOctaves="3" stitchTiles="stitch" />
          <feColorMatrix type="saturate" values="0" />
        </filter>
      </defs>
      <ellipse cx={lx} cy={ly} rx={46 + shape.bloom * 104} ry={42 + shape.bloom * 92} fill={`url(#mood-haze-${id})`} />
      <circle cx={lx} cy={ly} r={9 + shape.bloom * 24} fill={`url(#mood-core-${id})`} />
      <g fill="none" strokeLinecap="round" strokeLinejoin="round">
        <path d={path} stroke={moodColor(shape.warmth, .16)} strokeWidth="5" className="mood-light__halo" />
        <path d={path} stroke={moodColor(shape.warmth, .88)} strokeWidth="1.5" className="mood-light__line" />
      </g>
      <circle cx={lx} cy={ly} r="2.3" fill={moodColor(shape.warmth, .95)} />
      <rect width="360" height="400" filter={`url(#mood-grain-${id})`} opacity=".045" style={{ mixBlendMode: 'overlay' }} />
    </svg>
  );
}

/* ---------- Small brand moments ---------- */

/** A letter: one drawn sheet, a warm seal. Works on ink and on paper. */
export function LetterSymbol({ size = 44, className = '', tone = 'dark' }) {
  return (
    <svg
      className={`letter-symbol letter-symbol--${tone} ${className}`}
      width={size}
      height={size * 0.9}
      viewBox="0 0 64 58"
      aria-hidden="true"
      focusable="false"
    >
      <g transform="rotate(-6 32 29)" fill="none" strokeLinecap="round" strokeLinejoin="round">
        <path
          d="M16 10.6 C 23.4 9.2 40.6 9.2 48 10.6 C 49 22 49 36 48 47.4 C 40.6 48.8 23.4 48.8 16 47.4 C 15 36 15 22 16 10.6 Z"
          stroke="currentColor"
          strokeWidth="1.15"
          opacity=".9"
        />
        <path d="M16.6 18.6 C 24 17.8 40 17.8 47.4 18.6" stroke="currentColor" strokeWidth=".8" opacity=".34" />
        <g stroke="currentColor">
          <path d="M21.4 26 C 28 25.4 36 25.4 42.6 26" strokeWidth=".95" opacity=".42" />
          <path d="M21.4 32 C 27 31.4 33.6 31.4 39.4 32" strokeWidth=".95" opacity=".32" />
          <path d="M21.4 37.6 C 25.6 37 30.6 37 34.4 37.6" strokeWidth=".95" opacity=".24" />
        </g>
      </g>
      <circle cx="46.2" cy="43.2" r="4.2" fill={THEIA_COLORS.ember} />
      <circle cx="46.2" cy="43.2" r="6.6" fill="none" stroke={THEIA_COLORS.ember} strokeWidth=".7" opacity=".18" />
    </svg>
  );
}

export function Aperture({ className = '' }) {
  const id = useId().replace(/[^a-zA-Z0-9_-]/g, '');
  return (
    <svg className={`aperture ${className}`} viewBox="0 0 240 240" aria-hidden="true" focusable="false">
      <defs>
        <radialGradient id={`ap-${id}`}>
          <stop offset="0%" stopColor="#FFE9CC" stopOpacity=".3" />
          <stop offset="46%" stopColor="#DE9C62" stopOpacity=".12" />
          <stop offset="100%" stopColor="#DE9C62" stopOpacity="0" />
        </radialGradient>
      </defs>
      <circle cx="120" cy="120" r="104" fill={`url(#ap-${id})`} className="aperture__bloom" />
      <path
        d={ringPath({ cx: 120, cy: 120, radius: 92, gapFrom: -60, gapTo: -9, maxWeight: 2.3 })}
        fill="currentColor"
        opacity=".82"
        className="aperture__ring"
      />
      <path
        d={ringPath({ cx: 120, cy: 120, radius: 70, gapFrom: 112, gapTo: 148, maxWeight: 1.15 })}
        fill="currentColor"
        opacity=".3"
        className="aperture__ring aperture__ring--inner"
      />
      <circle cx="213" cy="76" r="4.2" fill="#DE9C62" className="aperture__spark" />
      <circle cx="27" cy="164" r="1.6" fill="#FFE9CC" opacity=".5" />
      <circle cx="196" cy="196" r="1.9" fill="#FFE9CC" opacity=".35" />
      <circle cx="57" cy="52" r="1.4" fill="#FFE9CC" opacity=".3" />
    </svg>
  );
}

/** Quiet empty / error vignette: the aperture waiting, not a sad icon. */
export function QuietVignette({ size = 92, tone = 'dark', className = '' }) {
  const id = useId().replace(/[^a-zA-Z0-9_-]/g, '');
  return (
    <svg className={`quiet-vignette quiet-vignette--${tone} ${className}`} width={size} height={size} viewBox="0 0 120 120" aria-hidden="true" focusable="false">
      <defs>
        <radialGradient id={`quiet-${id}`}>
          <stop offset="0%" stopColor={THEIA_COLORS.ember} stopOpacity=".26" />
          <stop offset="100%" stopColor={THEIA_COLORS.ember} stopOpacity="0" />
        </radialGradient>
      </defs>
      <circle cx="60" cy="62" r="52" fill={`url(#quiet-${id})`} />
      <path d={ringPath({ cx: 60, cy: 62, radius: 32, gapFrom: -58, gapTo: -12, maxWeight: 3.4 })} fill="currentColor" opacity=".55" />
      <circle cx="88" cy="41" r="3" fill={THEIA_COLORS.ember} />
      <circle cx="60" cy="62" r="26" fill="none" stroke="currentColor" strokeWidth=".6" opacity=".18" />
    </svg>
  );
}

/** The brand's loading state: one light, breathing. */
export function TheiaLoader({ label = 'در حال آماده‌سازی…', tone = 'dark' }) {
  return (
    <div className={`theia-loader theia-loader--${tone}`} role="status">
      <span className="theia-loader__ring" aria-hidden="true" />
      <span className="theia-loader__spark" aria-hidden="true" />
      <span className="sr-only">{label}</span>
    </div>
  );
}
