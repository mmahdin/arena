import React from 'react';

/**
 * Kenar (کنار) visual identity primitives.
 *
 * The whole identity is drawn with one rule: every contour is an inked line that
 * wanders by a millimetre. Nothing is a perfect circle, because nothing human is.
 * The character («هم‌نشین») is a small sand-coloured presence that sits on that
 * line — always beside the words, never in front of them. The line under it is
 * the conversation; the short second line beneath is the reader's own place.
 */

const SAND = 'var(--kenar-sand, #e9d0ae)';
const CLAY = 'var(--kenar-clay, #b9744f)';

const round = (value) => Math.round(value * 10) / 10;
const cache = new Map();
const memo = (key, compute) => {
  if (!cache.has(key)) cache.set(key, compute());
  return cache.get(key);
};

function catmullRom(points) {
  let d = `M${round(points[0][0])} ${round(points[0][1])}`;
  for (let i = 0; i < points.length - 1; i += 1) {
    const p0 = points[i - 1] || points[i];
    const p1 = points[i];
    const p2 = points[i + 1];
    const p3 = points[i + 2] || p2;
    d += ` C${round(p1[0] + (p2[0] - p0[0]) / 6)} ${round(p1[1] + (p2[1] - p0[1]) / 6)}`
      + ` ${round(p2[0] - (p3[0] - p1[0]) / 6)} ${round(p2[1] - (p3[1] - p1[1]) / 6)}`
      + ` ${round(p2[0])} ${round(p2[1])}`;
  }
  return d;
}

function resample(points, step) {
  const out = [points[0]];
  for (let i = 1; i < points.length; i += 1) {
    const [x0, y0] = points[i - 1];
    const [x1, y1] = points[i];
    const count = Math.max(1, Math.round(Math.hypot(x1 - x0, y1 - y0) / step));
    for (let k = 1; k <= count; k += 1) {
      out.push([x0 + ((x1 - x0) * k) / count, y0 + ((y1 - y0) * k) / count]);
    }
  }
  return out;
}

/** A deterministic, low-frequency sine "hand" that displaces a path sideways. */
function wobble(points, { amp = 1, seed = 1, freq = 0.05 } = {}) {
  const sampled = resample(points, 4);
  const phases = [(seed * 1.7) % 6.283, (seed * 3.1) % 6.283, (seed * 5.3) % 6.283];
  return sampled.map(([x, y], i) => {
    const before = sampled[Math.max(0, i - 1)];
    const after = sampled[Math.min(sampled.length - 1, i + 1)];
    const tx = after[0] - before[0];
    const ty = after[1] - before[1];
    const length = Math.hypot(tx, ty) || 1;
    const nx = -ty / length;
    const ny = tx / length;
    const t = i * freq;
    const noise = Math.sin(t * 6.283 + phases[0]) * 0.62
      + Math.sin(t * 13.7 + phases[1]) * 0.28
      + Math.sin(t * 27.1 + phases[2]) * 0.1;
    return [x + nx * noise * amp, y + ny * noise * amp];
  });
}

const hand = (points, options) => catmullRom(wobble(points, options));

const straight = (x0, y0, x1, y1, segments = 5) => Array.from(
  { length: segments + 1 },
  (_, i) => [x0 + ((x1 - x0) * i) / segments, y0 + ((y1 - y0) * i) / segments],
);

function dome(cx, baseY, halfWidth, height, lean = 0) {
  const points = [];
  const steps = 28;
  for (let i = 0; i <= steps; i += 1) {
    const a = Math.PI - (Math.PI * i) / steps;
    points.push([
      cx + Math.cos(a) * halfWidth + lean * Math.sin(a),
      baseY - Math.sin(a) * height + Math.sin(a * 2) * height * 0.018,
    ]);
  }
  points[0] = [cx - halfWidth, baseY];
  points[points.length - 1] = [cx + halfWidth, baseY];
  return points;
}

const horizonPath = (x0, x1, y, amp, seed) => memo(`h${x0}|${x1}|${y}|${amp}|${seed}`, () => hand(
  [[x0, y], [(x0 + x1) * 0.3, y - amp * 0.7], [(x0 + x1) * 0.55, y + amp * 0.6],
    [(x0 + x1) * 0.8, y - amp * 0.4], [x1, y + amp * 0.3]],
  { amp: amp * 0.9, seed, freq: 0.035 },
));

const echoPath = (x0, x1, y, amp, seed) => memo(`e${x0}|${x1}|${y}|${amp}|${seed}`, () => hand(
  [[x0, y], [(x0 + x1) * 0.35, y - amp], [(x0 + x1) * 0.7, y + amp * 0.8], [x1, y]],
  { amp, seed, freq: 0.05 },
));

function Horizon({ x0, x1, y, width = 2.6, opacity = 1, seed = 5, draw = false }) {
  return (
    <path
      className={draw ? 'kenar-line kenar-line--draw' : 'kenar-line'}
      d={horizonPath(x0, x1, y, 1.4, seed)}
      fill="none"
      stroke="currentColor"
      strokeWidth={width}
      strokeLinecap="round"
      opacity={opacity}
      pathLength={draw ? 1 : undefined}
    />
  );
}

function Echo({ x0, x1, y, opacity = 0.34, seed = 23, width = 1.7 }) {
  return (
    <path
      className="kenar-echo"
      d={echoPath(x0, x1, y, 1.1, seed)}
      fill="none"
      stroke="currentColor"
      strokeWidth={width}
      strokeLinecap="round"
      opacity={opacity}
    />
  );
}

function Arc({ cx, cy, rx, ry, from, to, opacity = 0.3, width = 1.5, seed = 31 }) {
  const path = memo(`a${cx}|${cy}|${rx}|${ry}|${from}|${to}|${seed}`, () => {
    const points = [];
    const steps = 22;
    for (let i = 0; i <= steps; i += 1) {
      const a = from + ((to - from) * i) / steps;
      points.push([cx + Math.cos(a) * rx, cy + Math.sin(a) * ry]);
    }
    return hand(points, { amp: 1.2, seed, freq: 0.05 });
  });
  return (
    <path d={path} fill="none" stroke="currentColor" strokeWidth={width}
      strokeLinecap="round" opacity={opacity} />
  );
}


/** A distant, half-shaped shore line; it gives the page depth without scenery. */
function Far({ x0, x1, y, opacity = 0.2, width = 1.2, seed = 91 }) {
  return (
    <path
      d={memo(`f${x0}|${x1}|${y}|${seed}`, () => hand(
        [[x0, y], [(x0 + x1) * 0.28, y - 2.4], [(x0 + x1) * 0.52, y + 1.6],
          [(x0 + x1) * 0.78, y - 1.4], [x1, y + 0.6]],
        { amp: 1.7, seed, freq: 0.028 },
      ))}
      fill="none" stroke="currentColor" strokeWidth={width} strokeLinecap="round" opacity={opacity}
    />
  );
}

/**
 * The companion. `pose` changes only its attitude — the silhouette, the ink and
 * the line underneath never change, which is what makes it recognisable.
 */
function Character({
  cx, baseY, halfWidth: hw, height: h, lean = 0, eyes = 'open',
  voice = false, curl = true, seed = 11, animated = true, className = '',
}) {
  const stroke = Math.max(2, Math.min(3.2, h * 0.032));
  const body = memo(`b${cx}|${baseY}|${hw}|${h}|${lean}|${seed}`, () => hand(
    dome(cx, baseY, hw, h, lean), { amp: Math.max(0.6, h * 0.01), seed, freq: 0.05 },
  ));
  const curlPath = memo(`c${cx}|${baseY}|${h}|${lean}`, () => {
    const s = h / 100;
    const x = cx + lean;
    const top = baseY - h;
    const p = (dx, dy) => `${round(x + dx * s)} ${round(top + dy * s)}`;
    return `M${p(6, 7)} C${p(4, -4)} ${p(12, -17)} ${p(23, -16)}`
      + ` C${p(33, -15)} ${p(35, -3)} ${p(26, -2)}`
      + ` C${p(19, -1.5)} ${p(18, -10)} ${p(23, -12)}`;
  });

  const eyeX = cx + hw * 0.3;
  const eyeY = baseY - h * 0.46;
  const eyeR = Math.max(1.8, h * 0.047);
  const gap = h * 0.24;
  const mouth = `M${round(eyeX + 2)} ${round(eyeY + h * 0.13)} C ${round(eyeX + gap * 0.32)} ${round(eyeY + h * 0.2)}, ${round(eyeX + gap * 0.78)} ${round(eyeY + h * 0.19)}, ${round(eyeX + gap * 1.04)} ${round(eyeY + h * 0.11)}`;

  return (
    <g className={`kenar-character ${animated ? 'is-animated' : ''} ${className}`.trim()}>
      <path d={body} fill={SAND} stroke="currentColor" strokeWidth={stroke} strokeLinejoin="round" />
      {curl && (
        <path className="kenar-curl" d={curlPath} fill="none" stroke="currentColor"
          strokeWidth={stroke * 0.95} strokeLinecap="round" />
      )}
      <g className={`kenar-eyes ${eyes === 'open' && animated ? 'is-blinking' : ''}`.trim()}
        fill="currentColor" stroke="currentColor" strokeWidth={stroke * 0.9} strokeLinecap="round">
        {eyes === 'closed' ? (
          <>
            <path d={`M${round(eyeX - eyeR * 2.2)} ${round(eyeY)} C ${round(eyeX - eyeR * 1.3)} ${round(eyeY + eyeR)}, ${round(eyeX - eyeR * 0.3)} ${round(eyeY + eyeR)}, ${round(eyeX + eyeR * 0.5)} ${round(eyeY)}`} fill="none" />
            <path d={`M${round(eyeX + gap - eyeR * 0.6)} ${round(eyeY)} C ${round(eyeX + gap + eyeR * 0.3)} ${round(eyeY + eyeR)}, ${round(eyeX + gap + eyeR * 1.3)} ${round(eyeY + eyeR)}, ${round(eyeX + gap + eyeR * 2.1)} ${round(eyeY)}`} fill="none" />
          </>
        ) : (
          <>
            <circle cx={round(eyeX)} cy={round(eyeY)} r={round(eyeR)} stroke="none" />
            <circle cx={round(eyeX + gap)} cy={round(eyeY)} r={round(eyeR * 0.94)} stroke="none" />
          </>
        )}
      </g>
      <path className="kenar-smile" d={mouth} fill="none" stroke="currentColor"
        strokeWidth={stroke * 0.9} strokeLinecap="round" />
      <circle className="kenar-blush" cx={round(eyeX + h * 0.35)} cy={round(eyeY + h * 0.12)}
        r={round(Math.max(2, h * 0.042))} fill={CLAY} opacity="0.42" />
      {voice && (
        <g className="kenar-voice" fill="none" stroke="currentColor" strokeWidth={stroke * 0.7}
          strokeLinecap="round" opacity="0.75">
          <path d={`M${round(eyeX + gap * 1.9)} ${round(eyeY - h * 0.04)} C ${round(eyeX + gap * 2.2)} ${round(eyeY + h * 0.05)}, ${round(eyeX + gap * 2.2)} ${round(eyeY + h * 0.14)}, ${round(eyeX + gap * 1.9)} ${round(eyeY + h * 0.22)}`} />
          <path d={`M${round(eyeX + gap * 2.4)} ${round(eyeY - h * 0.12)} C ${round(eyeX + gap * 2.9)} ${round(eyeY + h * 0.02)}, ${round(eyeX + gap * 2.9)} ${round(eyeY + h * 0.17)}, ${round(eyeX + gap * 2.4)} ${round(eyeY + h * 0.31)}`} />
          <path d={`M${round(eyeX + gap * 2.9)} ${round(eyeY - h * 0.2)} C ${round(eyeX + gap * 3.5)} ${round(eyeY + 0)}, ${round(eyeX + gap * 3.5)} ${round(eyeY + h * 0.19)}, ${round(eyeX + gap * 2.9)} ${round(eyeY + h * 0.39)}`} />
        </g>
      )}
    </g>
  );
}

/** The compact seal: the companion's silhouette, closed into a logo. */
export function KenarMark({ size = 24, className = '', label, withCurl = false }) {
  const body = memo('mark-body', () => hand(dome(24, 34.6, 15.4, 20.4, 1.4), { amp: 0.4, seed: 9, freq: 0.1 }));
  const line = memo('mark-line', () => horizonPath(3.4, 44.6, 35.3, 0.9, 5));
  const eyeY = 34.6 - 20.4 * 0.46;
  return (
    <svg viewBox="0 0 48 48" width={size} height={size} className={`kenar-mark ${className}`.trim()}
      role={label ? 'img' : undefined} aria-label={label} aria-hidden={label ? undefined : 'true'} focusable="false">
      <path d={line} fill="none" stroke="currentColor" strokeWidth="2.3" strokeLinecap="round" />
      <path d={body} fill={SAND} stroke="currentColor" strokeWidth="2.5" strokeLinejoin="round" />
      {withCurl && (
        <path d="M25.4 15.4 C24 9.4 29.4 5.8 33.6 7.6 C37 9 36.4 13.2 33.2 12.9"
          fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      )}
      <g fill="currentColor">
        <circle cx="21.8" cy={round(eyeY)} r="2.4" />
        <circle cx="28.4" cy={round(eyeY)} r="2.2" />
      </g>
      <path d={`M23.6 ${round(eyeY + 5.4)} C 25.6 ${round(eyeY + 8)}, 28 ${round(eyeY + 7.7)}, 29.4 ${round(eyeY + 5.2)}`}
        fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
      <circle cx="32.4" cy={round(eyeY + 3.8)} r="1.9" fill={CLAY} opacity="0.5" />
    </svg>
  );
}

/** Tiny silhouette for 10–16px seals where a face would turn into noise. */
export function KenarGlyph({ size = 14, className = '' }) {
  const body = memo('glyph-body', () => hand(dome(12, 16, 7.6, 10, 0.8), { amp: 0.3, seed: 9, freq: 0.12 }));
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} className={`kenar-glyph ${className}`.trim()}
      aria-hidden="true" focusable="false">
      <path d={body} fill="currentColor" stroke="currentColor" strokeWidth="1.4" strokeLinejoin="round" />
      <path d="M2 17.4 C 7 16.6, 13 18, 22 17.2" fill="none" stroke="currentColor"
        strokeWidth="1.5" strokeLinecap="round" opacity="0.55" />
    </svg>
  );
}

/** Wordmark: the name, and the two-line "beside" mark that belongs to it. */
export function KenarWordmark({ className = '' }) {
  return (
    <span className={`kenar-wordmark ${className}`.trim()}>
      <span className="kenar-wordmark-lockup">
        <KenarMark size={30} className="kenar-wordmark-mark" />
        <span className="kenar-wordmark-text">کنار</span>
      </span>
      <svg className="kenar-wordmark-rule" viewBox="0 0 96 12" preserveAspectRatio="none" aria-hidden="true" focusable="false">
        <path d="M93 4 C 74 3.2, 52 4.6, 31 3.9 C 22 3.6, 12 4.2, 4 4" fill="none"
          stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" opacity="0.55" />
        <path d="M86 9 C 66 8.4, 42 9.6, 22 9" fill="none" stroke="currentColor"
          strokeWidth="1.1" strokeLinecap="round" opacity="0.28" />
        <circle cx="4.4" cy="4" r="1.9" fill="var(--kenar-clay, #b9744f)" />
      </svg>
    </span>
  );
}

/* ---------- Scenes: one character, one line, four ways of being beside ---------- */

function Scene({ viewBox, className, children, caption }) {
  return (
    // A span, not a div: the scene is also mounted inside buttons (Home's orbit) and
    // must stay phrasing content, exactly like the inline artwork it replaces.
    <span className={`kenar-scene ${className}`.trim()}>
      <svg viewBox={viewBox} role="presentation" aria-hidden="true" focusable="false" preserveAspectRatio="xMidYMid meet">
        {children}
      </svg>
      {caption && <span className="kenar-caption">{caption}</span>}
    </span>
  );
}

export function KenarScene({ variant = 'welcome', className = '' }) {
  if (variant === 'welcome') {
    return (
      <Scene viewBox="0 0 360 480" className={`kenar-scene--welcome ${className}`.trim()}
        caption="سلام. من کنارم.">
        <Far x0={34} x1={326} y={132} opacity={0.2} width={1.3} seed={3} />
        <Far x0={14} x1={346} y={236} opacity={0.13} width={1.1} seed={8} />
        <circle cx="302" cy="434" r="3.2" fill={CLAY} opacity="0.45" />
        <circle cx="280" cy="434" r="2.1" fill={CLAY} opacity="0.28" />
        <Echo x0={14} x1={346} y={448} opacity={0.3} />
        <Horizon x0={4} x1={356} y={434} seed={5} draw />
        <Character cx={176} baseY={434} halfWidth={72} height={104} lean={6} seed={11} />
      </Scene>
    );
  }

  if (variant === 'small') {
    return (
      <Scene viewBox="0 0 360 340" className={`kenar-scene--small ${className}`.trim()}>
        <circle cx="98" cy="280" r="2.8" fill={CLAY} opacity="0.38" />
        <circle cx="266" cy="280" r="2.1" fill={CLAY} opacity="0.26" />
        <Echo x0={22} x1={338} y={292} opacity={0.28} />
        <Horizon x0={8} x1={352} y={280} seed={6} draw />
        <Character cx={180} baseY={280} halfWidth={52} height={70} lean={4} seed={13} />
      </Scene>
    );
  }

  if (variant === 'medium') {
    return (
      <Scene viewBox="0 0 360 360" className={`kenar-scene--medium ${className}`.trim()}>
        <Echo x0={16} x1={344} y={322} opacity={0.28} />
        <Horizon x0={4} x1={356} y={310} seed={9} draw />
        <Character cx={176} baseY={310} halfWidth={62} height={86} lean={11} seed={15} />
      </Scene>
    );
  }

  if (variant === 'large') {
    return (
      <Scene viewBox="0 0 360 420" className={`kenar-scene--large ${className}`.trim()}>
        <Far x0={44} x1={316} y={128} opacity={0.18} width={1.3} seed={12} />
        <Echo x0={20} x1={340} y={384} opacity={0.28} />
        <Horizon x0={4} x1={356} y={372} seed={14} draw />
        <Character cx={180} baseY={372} halfWidth={72} height={94} lean={4} eyes="closed" seed={17} />
      </Scene>
    );
  }

  if (variant === 'tall') {
    return (
      <Scene viewBox="0 0 360 560" className={`kenar-scene--tall ${className}`.trim()}>
        <circle cx="276" cy="488" r="3.4" fill={CLAY} opacity="0.42" />
        <circle cx="252" cy="488" r="2.2" fill={CLAY} opacity="0.26" />
        <Echo x0={18} x1={342} y={500} opacity={0.26} />
        <Horizon x0={4} x1={356} y={488} seed={19} draw />
        <Character cx={152} baseY={488} halfWidth={66} height={96} lean={6} voice seed={21} />
      </Scene>
    );
  }

  if (variant === 'privacy') {
    return (
      <Scene viewBox="0 0 140 140" className={`kenar-scene--privacy ${className}`.trim()}>
        <path className="kenar-loop"
          d={memo('privacy-loop', () => hand(Array.from({ length: 29 }, (_, i) => {
            const a = (i / 28) * Math.PI * 2;
            return [70 + Math.cos(a) * 50, 70 + Math.sin(a) * 46];
          }), { amp: 1.2, seed: 27, freq: 0.06 }))}
          fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" opacity="0.42" />
        <Echo x0={16} x1={124} y={116} opacity={0.32} seed={29} />
        <Horizon x0={10} x1={130} y={108} width={2.2} seed={33} draw />
        <Character cx={70} baseY={108} halfWidth={34} height={48} lean={3} seed={35} animated={false} />
      </Scene>
    );
  }

  if (variant === 'checkins') {
    return (
      <Scene viewBox="0 0 360 320" className={`kenar-scene--checkins ${className}`.trim()}>
        <Arc cx={180} cy={232} rx={70} ry={44} from={Math.PI * 1.2} to={Math.PI * 1.8} opacity={0.4} width={1.8} seed={41} />
        <Arc cx={180} cy={232} rx={104} ry={66} from={Math.PI * 1.22} to={Math.PI * 1.78} opacity={0.26} width={1.5} seed={43} />
        <Arc cx={180} cy={232} rx={138} ry={88} from={Math.PI * 1.24} to={Math.PI * 1.76} opacity={0.14} width={1.3} seed={45} />
        <circle cx="258" cy="276" r="3.6" fill={CLAY} opacity="0.46" />
        <circle cx="284" cy="276" r="2.3" fill={CLAY} opacity="0.28" />
        <Echo x0={18} x1={342} y={288} opacity={0.28} />
        <Horizon x0={4} x1={356} y={276} seed={47} draw />
        <Character cx={176} baseY={276} halfWidth={58} height={80} lean={5} seed={49} />
      </Scene>
    );
  }

  if (variant === 'ready') {
    return (
      <Scene viewBox="0 0 360 460" className={`kenar-scene--ready ${className}`.trim()}>
        <Far x0={30} x1={330} y={122} opacity={0.18} width={1.3} seed={51} />
        <circle cx="272" cy="400" r="4" fill={CLAY} opacity="0.5" />
        <circle cx="304" cy="400" r="2.8" fill={CLAY} opacity="0.32" />
        <circle cx="328" cy="400" r="2" fill={CLAY} opacity="0.18" />
        <Echo x0={10} x1={350} y={412} opacity={0.3} seed={53} />
        <Horizon x0={4} x1={356} y={400} seed={55} draw />
        <Character cx={150} baseY={400} halfWidth={64} height={94} lean={6} seed={57} />
      </Scene>
    );
  }

  if (variant === 'login') {
    return (
      <Scene viewBox="0 0 360 240" className={`kenar-scene--login ${className}`.trim()}>
        <Echo x0={24} x1={336} y={196} opacity={0.26} />
        <Horizon x0={8} x1={352} y={184} seed={61} draw />
        <Character cx={180} baseY={184} halfWidth={48} height={64} lean={4} seed={63} />
      </Scene>
    );
  }

  return (
    <Scene viewBox="0 0 240 240" className={`kenar-scene--home ${className}`.trim()}>
      <Far x0={64} x1={176} y={84} opacity={0.18} width={1.1} seed={73} />
      <Echo x0={92} x1={148} y={204} opacity={0.28} seed={69} />
      <Horizon x0={72} x1={168} y={192} width={2.2} seed={67} draw />
      <Character cx={120} baseY={192} halfWidth={46} height={68} lean={4} seed={71} />
    </Scene>
  );
}

/** A single companion, without a scene, for avatars and inline moments. */
export function KenarCompanion({ pose = 'sit', className = '', animated = true }) {
  const eyes = pose === 'rest' ? 'closed' : 'open';
  const lean = pose === 'lean' ? 12 : pose === 'sit' ? 4 : 2;
  return (
    <svg viewBox="0 0 240 200" className={`kenar-companion ${className}`.trim()}
      role="presentation" aria-hidden="true" focusable="false">
      <Horizon x0={6} x1={234} y={178} width={2.2} seed={77} draw />
      <Echo x0={30} x1={210} y={190} opacity={0.28} seed={79} />
      <Character cx={120} baseY={178} halfWidth={pose === 'peek' ? 40 : 54} height={pose === 'peek' ? 52 : 74}
        lean={lean} eyes={eyes} voice={pose === 'voice'} seed={81} animated={animated} />
    </svg>
  );
}
