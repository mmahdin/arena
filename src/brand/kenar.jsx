import React from 'react';

/**
 * Kenar (کنار) visual identity primitives.
 *
 * The whole identity comes from one sentence: «کنار یعنی کسی همین نزدیکی.»
 * Kenar is not a mascot looking at the camera; it is a *presence beside you*.
 * So the system has exactly one subject — two presences seated side by side on
 * a single, unbroken shore line — and one colour idea: warm clay on pale paper
 * (day) or lamplight on deep night (dark screens).
 *
 * Every illustration is built from the primitives below, which is what makes
 * the product feel designed as Kenar from the first screen to the last.
 * Colour is inherited from CSS (`--k-fg`, `--k-accent`, `--k-soft`) so the same
 * drawing works on the light onboarding, the dark home, and the beige screens.
 */

/** A seated presence seen from behind: head + shoulders, grounded on the shore. */
export function SittingFigure({ x = 0, y = 0, scale = 1, lean = 0, shawl = false, tone = 'fg' }) {
  return (
    <g transform={`translate(${x} ${y}) scale(${scale}) rotate(${lean})`} className={`k-figure k-${tone}`}>
      <path d="M -27 0 C -29 -19 -25 -33 -13 -40 C -6 -44 6 -44 13 -40 C 24 -35 28 -21 27 0 Z" />
      <circle cx="0" cy="-54" r="13" />
      {shawl && <path className="k-shawl" d="M -23 -11 C -10 -5 10 -7 24 -15" />}
    </g>
  );
}

/** The low, unbroken line every scene rests on: the shore, the "beside" line. */
export function ShoreLine({ d = 'M 4 252 C 96 250 208 254 296 251', className = 'k-line' }) {
  return <path d={d} className={className} />;
}

/** The soft dusk disc: the light that stays on the horizon while you talk. */
export function DuskDisc({ cx = 198, cy = 148, r = 22, opacity = 0.18 }) {
  return (
    <g className="k-dusk" style={{ opacity }}>
      <circle cx={cx} cy={cy} r={r} className="k-accent-fill" />
      <circle cx={cx} cy={cy} r={Math.round(r * 1.7)} className="k-accent-stroke" opacity=".55" />
    </g>
  );
}

/** Quiet listening rings, drawn as ripples around the pair, never as a frame. */
export function ListeningRings({ cx = 96, cy = 150, radii = [34, 52, 70], className = 'k-ripple' }) {
  return (
    <g className={className}>
      {radii.map((r, index) => (
        <circle key={r} cx={cx} cy={cy} r={r} style={{ opacity: 0.55 - index * 0.15 }} />
      ))}
    </g>
  );
}

/** The logomark: two presences on one shore. Monochrome variant for small marks. */
export function KenarMark({ className = '', mono = false, title }) {
  return (
    <svg
      className={`kenar-mark ${mono ? 'kenar-mark--mono' : ''} ${className}`}
      viewBox="0 0 48 48"
      role={title ? 'img' : 'presentation'}
      aria-label={title || undefined}
      aria-hidden={title ? undefined : 'true'}
      focusable="false"
    >
      <path className="k-line" d="M 5 38.4 C 15 37.3 33 38.7 43 37.6" />
      <SittingFigure x={17.5} y={38.4} scale={0.34} lean={-2} tone={mono ? 'mono' : 'fg'} />
      <SittingFigure x={31} y={38.4} scale={0.27} lean={3} tone={mono ? 'mono' : 'accent'} />
    </svg>
  );
}

/** The wordmark: Naskh lettering with the shore line drawn under it. */
export function KenarWordmark({ className = '', as: Tag = 'span', showPebble = true }) {
  return (
    <Tag className={`kenar-wordmark ${className}`} aria-label="کنار">
      <span className="kenar-wordmark-text" aria-hidden="true">کنار</span>
      <svg className="kenar-wordmark-shore" viewBox="0 0 120 12" aria-hidden="true" focusable="false">
        <path className="k-line" d="M 4 8 C 34 5.6 82 9.4 116 6.6" />
      </svg>
      {showPebble && <span className="kenar-wordmark-pebble" aria-hidden="true" />}
    </Tag>
  );
}

/** A pebble: the smallest possible trace of the presence (badges, unread dots). */
export function Pebble({ className = '', tone = 'accent' }) {
  return (
    <svg className={`kenar-pebble ${className}`} viewBox="0 0 24 18" aria-hidden="true" focusable="false">
      <path className={`k-${tone}-fill`} d="M 3 15 C 1.4 8.6 5 2.6 11.6 2.2 C 18 1.8 22.2 6.6 21 15 Z" />
    </svg>
  );
}

/* ------------------------------------------------------------------ scenes */

function Scene({ viewBox, className = '', label, children }) {
  return (
    <svg
      className={`kenar-scene ${className}`}
      viewBox={viewBox}
      role={label ? 'img' : 'presentation'}
      aria-label={label || undefined}
      aria-hidden={label ? undefined : 'true'}
      preserveAspectRatio="xMidYMax meet"
      focusable="false"
    >
      {children}
    </svg>
  );
}

/**
 * The onboarding story in six frames. Nothing is explained in words here:
 * Kenar is already at the shore, it sets a pebble down to keep your place,
 * you arrive and sit on that spot, and the two of you end up shoulder to
 * shoulder. The copy on each screen carries the phrasing.
 */
const FRAMES = [
  { kenarX: 128, kenarScale: 1.36, youX: null, youScale: 0, pebble: false, dusk: { cx: 212, cy: 128, r: 24 } },
  { kenarX: 130, kenarScale: 1.48, youX: null, youScale: 0, pebble: true, dusk: { cx: 208, cy: 124, r: 26 } },
  { kenarX: 124, kenarScale: 1.56, youX: 218, youScale: 1.16, pebble: false, dusk: { cx: 204, cy: 120, r: 27 } },
  { kenarX: 122, kenarScale: 1.7, youX: 212, youScale: 1.52, pebble: false, dusk: { cx: 198, cy: 116, r: 28 } },
  { kenarX: 120, kenarScale: 1.86, youX: 202, youScale: 1.7, pebble: false, dusk: { cx: 192, cy: 112, r: 29 } },
  { kenarX: 118, kenarScale: 2.04, youX: 194, youScale: 1.9, pebble: false, dusk: { cx: 186, cy: 108, r: 30 } },
];

export function PresenceScene({ frame = 0, className = '', label, viewBox = '0 0 300 360' }) {
  const config = FRAMES[Math.max(0, Math.min(FRAMES.length - 1, frame))];
  const base = 300;
  return (
    <Scene viewBox={viewBox} className={className} label={label}>
      <DuskDisc {...config.dusk} opacity={0.16} />
      <path className="k-far" d={`M -10 ${base - 62} C 80 ${base - 68} 220 ${base - 68} 310 ${base - 62}`} />
      <ShoreLine d={`M 4 ${base} C 96 ${base - 3} 208 ${base + 3} 296 ${base - 2}`} />
      {config.pebble && (
        <g className="k-place">
          <ellipse className="k-accent-fill" cx="212" cy={base - 12} rx="18" ry="11" />
          <path className="k-ripple-line" d={`M 184 ${base - 3} C 198 ${base - 16} 228 ${base - 16} 240 ${base - 3}`} />
        </g>
      )}
      {config.youX !== null && (
        <SittingFigure x={config.youX} y={base} scale={config.youScale} lean={-3} />
      )}
      <SittingFigure x={config.kenarX} y={base} scale={config.kenarScale} lean={4} shawl tone="accent" />
    </Scene>
  );
}

export function WelcomeScene({ className = '' }) {
  return <PresenceScene frame={0} className={className} label="کنار، تنها، کنارِ کرانه منتظر توست" />;
}

export function ReadyScene({ className = '' }) {
  // The final frame of the story is the closest one: a tighter crop, so the two
  // presences fill the screen the way the relationship fills the product.
  return <PresenceScene frame={5} className={className} viewBox="16 40 268 276" label="تو و کنار، کنار هم روی کرانه نشسته‌اید" />;
}

/** Privacy: your words stay inside the stone. */
export function PrivacyScene({ className = '' }) {
  return (
    <Scene viewBox="0 0 120 132" className={className} label="سنگی پهن با سوراخ کلید؛ گفت‌وگوها همین‌جا می‌مانند">
      <ShoreLine d="M 8 112 C 38 110 84 113 112 111" />
      <SittingFigure x={24} y={112} scale={0.36} lean={-4} />
      <path className="k-accent-fill" d="M 36 110 C 30 96 42 82 65 81 C 89 80 102 94 96 110 Z" />
      <path className="k-stone-shade" d="M 48 104 C 44 94 51 87 61 85" />
      <circle className="k-paper-fill" cx="68" cy="91" r="6.4" />
      <path className="k-paper-fill" d="M 62.8 108 L 73.2 108 L 71 95 L 65 95 Z" />
    </Scene>
  );
}

/** Check-ins: one small bell, and someone sitting with you while it rings. */
export function CheckinsScene({ className = '' }) {
  return (
    <Scene viewBox="0 0 418 372" className={className} label="زنگی کوچک زیر یک طاق، و کنار که کنارت نشسته است">
      <path className="k-arch" d="M 62 306 C 62 196 356 196 356 306" />
      <path className="k-line" d="M 24 308 C 150 305 280 310 396 307" />
      <g className="k-bell">
        <path className="k-line" d="M 209 214 L 209 240" />
        <path className="k-accent-fill" d="M 193 268 C 193 246 225 246 225 268 C 229 276 233 282 236 288 L 182 288 C 185 282 189 276 193 268 Z" />
        <circle className="k-fill" cx="209" cy="294" r="4" />
      </g>
      <g className="k-ripple">
        <path d="M 168 252 C 178 240 240 240 250 252" style={{ opacity: 0.32 }} />
        <path d="M 152 240 C 168 222 250 222 266 240" style={{ opacity: 0.18 }} />
      </g>
      <SittingFigure x={166} y={308} scale={0.7} lean={-3} />
      <SittingFigure x={252} y={308} scale={0.6} lean={4} shawl tone="accent" />
    </Scene>
  );
}

/** Home: two presences inside the listening rings, waiting for you to speak. */
export function HomePresence({ className = '' }) {
  return (
    <Scene viewBox="0 0 240 240" className={className} label="دو همنشین، کنار هم">
      <ListeningRings cx={116} cy={158} radii={[44, 66, 88]} />
      <circle className="k-accent-fill k-dusk-dot" cx="188" cy="66" r="6" />
      <ShoreLine d="M 14 176 C 76 174 168 178 228 175" />
      <SittingFigure x={94} y={176} scale={0.84} lean={-3} />
      <SittingFigure x={138} y={176} scale={0.73} lean={4} shawl tone="accent" />
    </Scene>
  );
}

/** Chat empty state: a single presence, seated, and nothing asked of you yet. */
export function ChatPresence({ className = '' }) {
  return (
    <Scene viewBox="0 0 120 96" className={className} label="کنار، نشسته و منتظر">
      <ListeningRings cx={60} cy={74} radii={[27, 44]} />
      <ShoreLine d="M 16 76 C 44 74 82 77 106 75" />
      <SittingFigure x={60} y={76} scale={0.62} lean={3} shawl tone="accent" />
    </Scene>
  );
}
