// Geometry and composition for Theia's visual identity.
//
// Theia is never drawn as a person. In Greek myth she is the Titaness of divine
// light and of *sight* — thea, "a seeing" — and the mother of Helios (sun),
// Selene (moon) and Eos (dawn). The identity is therefore built from the two
// things her myth is actually made of: a threshold that is looked through, and
// the light that appears in it, chapter by chapter.
//
// The threshold is a two-centred Persian arch (the tāq of a mihrab or an
// iwan), drawn as a hairline rather than as ornament. That choice is
// deliberate for an Iranian audience: it is a shape everyone has stood in
// front of, it carries "entering / being received" without any religious
// claim, and it lets the product borrow dignity from architecture instead of
// borrowing a woman's body to stand for a psyche. Every chapter keeps the same
// arch and changes only the light inside it.
//
// Scenes are plain data. The React renderer, the design-review renderer and the
// generated app icons all consume the same numbers, so what ships is exactly
// what was reviewed, and the geometry can be unit-tested without a browser.

export const TAU = Math.PI * 2;

export const PALETTE = {
  ink: '#0b0a09',
  night: '#0e1019',
  lapis: '#1b2438',
  lapisLight: '#2b3a58',
  cream: '#f2ece1',
  gold: '#c9a86a',
  goldLight: '#e6cd9a',
  moon: '#dcdfef',
  turquoise: '#8fbcb4',
};

export const polar = (cx, cy, radius, angle) => ({
  x: cx + radius * Math.cos(angle),
  y: cy + radius * Math.sin(angle),
});

export const round = (value) => Math.round(value * 100) / 100;

/**
 * Two-centred pointed arch. `half` is the half-span, `d` the offset of each
 * arc's centre from the axis (a larger offset gives a taller, sharper apex),
 * `spring` the height at which the arcs begin and `base` where the jambs end.
 */
export function archGeometry({ cx = 120, half = 56, spring = 216, d = 32, base = 268 } = {}) {
  const radius = half + d;
  const apexY = spring - Math.sqrt(radius * radius - d * d);
  const left = cx - half;
  const right = cx + half;
  return {
    cx,
    left,
    right,
    spring,
    base,
    apexY: round(apexY),
    radius: round(radius),
    path: `M${round(left)} ${round(base)}L${round(left)} ${round(spring)}` +
      `A${round(radius)} ${round(radius)} 0 0 1 ${round(cx)} ${round(apexY)}` +
      `A${round(radius)} ${round(radius)} 0 0 1 ${round(right)} ${round(spring)}` +
      `L${round(right)} ${round(base)}`,
  };
}

/** A four-pointed dawn star: the last star to leave the sky at sunrise. */
export function dawnStarPath(cx, cy, outer, inner = outer * 0.3, turn = -Math.PI / 2) {
  const points = [];
  for (let i = 0; i < 8; i += 1) {
    const radius = i % 2 === 0 ? outer : inner;
    const { x, y } = polar(cx, cy, radius, turn + (i * TAU) / 8);
    points.push(`${round(x)} ${round(y)}`);
  }
  return `M${points.join('L')}Z`;
}

/**
 * Crescent from the intersection of two circles: the outer rim of radius
 * `radius`, and a "night" circle of `inner` radius pushed `offset` to the right.
 * The crescent opens toward +x, so a call site can mirror or rotate it.
 */
export function crescentPath({ cx = 120, cy = 190, radius = 30, inner = 27, offset = 10 } = {}) {
  const a = (offset * offset + radius * radius - inner * inner) / (2 * offset);
  const h = Math.sqrt(Math.max(0.0001, radius * radius - a * a));
  const top = { x: cx + a, y: cy - h };
  const bottom = { x: cx + a, y: cy + h };
  return `M${round(top.x)} ${round(top.y)}` +
    `A${round(radius)} ${round(radius)} 0 1 0 ${round(bottom.x)} ${round(bottom.y)}` +
    `A${round(inner)} ${round(inner)} 0 0 1 ${round(top.x)} ${round(top.y)}Z`;
}

/** Deterministic star field — no Math.random, so renders never drift. */
export function starField({ count = 26, width = 240, height = 300, seed = 11, minY = 0, maxY = height }) {
  const stars = [];
  let state = seed;
  const next = () => {
    state = (state * 1103515245 + 12345) % 2147483648;
    return state / 2147483648;
  };
  for (let i = 0; i < count; i += 1) {
    stars.push({
      x: round(6 + next() * (width - 12)),
      y: round(minY + next() * (maxY - minY)),
      r: round(0.45 + next() * 1.05),
      o: round(0.14 + next() * 0.46),
      delay: round(next() * 8),
    });
  }
  return stars;
}

/** Radial ticks of a light source, alternating long and short like real rays. */
export function starRays({ cx, cy, from, to, count, offset = 0, long = 1.16 }) {
  return Array.from({ length: count }, (_, i) => {
    const angle = ((i * 360) / count + offset) * (Math.PI / 180);
    const a = polar(cx, cy, from, angle);
    const b = polar(cx, cy, i % 2 ? to : to * long, angle);
    return {
      x1: round(a.x), y1: round(a.y), x2: round(b.x), y2: round(b.y),
      strong: i % 2 === 0,
    };
  });
}

const C = PALETTE;

const star = (x, y, size, opacity = 0.85, className = 'theia-star', delay = 0) => ({
  t: 'path', d: dawnStarPath(x, y, size, size * 0.3), fill: 'cream', opacity, className, delay,
});

const dot = (s) => ({
  t: 'circle', cx: s.x, cy: s.y, r: s.r, fill: 'cream', opacity: s.o, className: 'theia-star', delay: s.delay,
});

const horizonLines = (y, widths, color = 'cream', step = 13) => widths.map((half, i) => ({
  t: 'line', x1: 120 - half, y1: y + i * step, x2: 120 + half, y2: y + i * step,
  stroke: color, strokeWidth: 0.55, opacity: round(0.24 - i * 0.05),
}));

const OUTER = archGeometry();
const INNER = archGeometry({ half: 45, spring: 222, d: 26 });

/** Prologue: a threshold, a field of stars, and one star already inside. */
export function welcomeScene() {
  return {
    viewBox: '0 0 240 300',
    defs: [
      { id: 'welcome-glow', type: 'radial', stops: [[0, 'gold', 0.22], [0.58, 'lapis', 0.1], [1, 'lapis', 0]] },
    ],
    elements: [
      { t: 'circle', cx: 120, cy: 170, r: 118, fill: 'welcome-glow' },
      ...starField({ count: 26, maxY: 246, seed: 5 }).map(dot),
      { t: 'path', d: archGeometry({ half: 63, spring: 212, d: 38 }).path, stroke: 'gold', strokeWidth: 0.55, opacity: 0.2, dash: '1 8', className: 'theia-drift' },
      { t: 'path', d: OUTER.path, stroke: 'cream', strokeWidth: 1, opacity: 0.32 },
      { t: 'path', d: INNER.path, stroke: 'cream', strokeWidth: 0.6, opacity: 0.13 },
      { t: 'path', d: 'M64 268H176', stroke: 'cream', strokeWidth: 0.6, opacity: 0.14 },
      star(120, 178, 9.5, 0.9, 'theia-star theia-star--focus'),
    ],
  };
}

/** I — Night. Selene: the moon only lends edges to things. */
export function nightScene() {
  return {
    viewBox: '0 0 240 300',
    defs: [
      { id: 'night-halo', type: 'radial', stops: [[0, 'moon', 0.26], [0.42, 'moon', 0.07], [1, 'moon', 0]] },
    ],
    elements: [
      ...starField({ count: 30, maxY: 240, seed: 17 }).map(dot),
      { t: 'path', d: OUTER.path, stroke: 'cream', strokeWidth: 0.9, opacity: 0.26 },
      { t: 'circle', cx: 128, cy: 176, r: 58, fill: 'night-halo' },
      { t: 'path', d: crescentPath({ cx: 120, cy: 176, radius: 30, inner: 27.5, offset: 11 }), fill: 'moon', opacity: 0.86, className: 'theia-drift' },
      star(170, 154, 3.6, 0.5),
      star(76, 138, 2.4, 0.3),
      ...horizonLines(236, [32, 24, 16, 9], 'moon', 9),
    ],
  };
}

/** II — Dawn. Eos: the first line of light returns the shape of things. */
export function dawnScene() {
  const horizon = 224;
  return {
    viewBox: '0 0 240 300',
    defs: [
      { id: 'dawn-sky', type: 'linear', from: [0, 96], to: [0, 224], stops: [[0, 'night', 1], [0.46, 'lapis', 1], [0.82, 'lapisLight', 1], [1, 'gold', 0.9]] },
      { id: 'dawn-water', type: 'linear', from: [0, 224], to: [0, 272], stops: [[0, 'gold', 0.22], [0.4, 'lapis', 0.5], [1, 'ink', 0.9]] },
      { id: 'dawn-glow', type: 'radial', stops: [[0, 'goldLight', 0.72], [0.5, 'gold', 0.26], [1, 'gold', 0]] },
    ],
    elements: [
      { t: 'clip', id: 'dawn-clip', children: [{ t: 'path', d: OUTER.path }] },
      {
        t: 'group', clip: 'dawn-clip', children: [
          { t: 'rect', x: 56, y: 96, width: 128, height: 130, fill: 'dawn-sky' },
          { t: 'rect', x: 56, y: horizon, width: 128, height: 48, fill: 'dawn-water' },
          { t: 'circle', cx: 120, cy: horizon + 6, r: 86, fill: 'dawn-glow', className: 'theia-breathe' },
          ...starField({ count: 12, minY: 130, maxY: 196, seed: 31 }).map(dot),
          ...starRays({ cx: 120, cy: horizon, from: 60, to: 76, count: 6, offset: -90 - 50 }).map((r) => ({
            t: 'line', ...r, stroke: 'goldLight', strokeWidth: 0.7, opacity: r.strong ? 0.5 : 0.24,
          })),
          { t: 'path', d: `M62 ${horizon}A58 58 0 0 1 178 ${horizon}`, stroke: 'goldLight', strokeWidth: 1.5, fill: 'none', opacity: 0.95, className: 'theia-rise' },
          { t: 'path', d: `M82 ${horizon}A38 38 0 0 1 158 ${horizon}`, stroke: 'gold', strokeWidth: 0.7, fill: 'none', opacity: 0.4 },
          { t: 'line', x1: 56, y1: horizon, x2: 184, y2: horizon, stroke: 'cream', strokeWidth: 0.9, opacity: 0.75, className: 'theia-horizon' },
          ...horizonLines(horizon + 14, [54, 44, 34, 22], 'goldLight', 10),
        ],
      },
      { t: 'path', d: OUTER.path, stroke: 'cream', strokeWidth: 1.1, opacity: 0.46 },
      star(158, 138, 2.6, 0.32),
    ],
  };
}

/** III — Sun. Helios: light that hides nothing, and judges nothing. */
export function sunScene() {
  const cx = 120;
  const cy = 186;
  return {
    viewBox: '0 0 240 300',
    defs: [
      { id: 'sun-glow', type: 'radial', stops: [[0, 'goldLight', 0.5], [0.42, 'gold', 0.2], [1, 'gold', 0]] },
      { id: 'sun-core', type: 'radial', stops: [[0, 'goldLight', 0.62], [0.6, 'goldLight', 0.16], [1, 'goldLight', 0]] },
    ],
    elements: [
      { t: 'path', d: OUTER.path, stroke: 'cream', strokeWidth: 1, opacity: 0.34 },
      { t: 'circle', cx, cy, r: 96, fill: 'sun-glow', className: 'theia-breathe' },
      ...starRays({ cx, cy, from: 26, to: 46, count: 20 }).map((r) => ({
        t: 'line', ...r, stroke: 'goldLight', strokeWidth: 0.55, opacity: r.strong ? 0.4 : 0.15,
      })),
      { t: 'circle', cx, cy, r: 24, fill: 'sun-core' },
      { t: 'circle', cx, cy, r: 24, stroke: 'goldLight', strokeWidth: 0.9, opacity: 0.55 },
      { t: 'circle', cx, cy, r: 18, stroke: 'goldLight', strokeWidth: 0.5, opacity: 0.24, dash: '1.5 5.5' },
      star(cx, cy, 8, 1, 'theia-star theia-star--focus'),
      { t: 'path', d: crescentPath({ cx: 120, cy: 262, radius: 6, inner: 5.5, offset: 2.1 }), fill: 'moon', opacity: 0.2 },
    ],
  };
}

/** IV — Sight. Theia: the light and the seeing of it are one event. */
export function sightScene() {
  const cx = 120;
  const cy = 190;
  return {
    viewBox: '0 0 240 300',
    defs: [
      { id: 'sight-sky', type: 'linear', stops: [[0, 'night', 1], [0.5, 'lapis', 1], [1, 'gold', 0.5]] },
      { id: 'sight-bloom', type: 'radial', stops: [[0, 'goldLight', 0.34], [0.58, 'gold', 0.1], [1, 'gold', 0]] },
    ],
    elements: [
      { t: 'circle', cx, cy: cy - 6, r: 112, fill: 'sight-bloom', className: 'theia-breathe' },
      { t: 'clip', id: 'sight-clip', children: [{ t: 'path', d: OUTER.path }] },
      {
        t: 'group', clip: 'sight-clip', children: [
          { t: 'rect', x: 56, y: 100, width: 128, height: 172, fill: 'sight-sky' },
          { t: 'circle', cx, cy: 250, r: 120, fill: 'sight-bloom' },
          { t: 'line', x1: 56, y1: 250, x2: 184, y2: 250, stroke: 'goldLight', strokeWidth: 0.6, opacity: 0.6 },
          ...horizonLines(258, [44, 33, 22], 'goldLight', 11),
        ],
      },
      { t: 'path', d: OUTER.path, stroke: 'goldLight', strokeWidth: 1.1, opacity: 0.6 },
      { t: 'path', d: INNER.path, stroke: 'cream', strokeWidth: 0.6, opacity: 0.18 },
      star(cx, 176, 10, 0.95, 'theia-star theia-star--focus'),
      star(78, 152, 2.4, 0.26),
      star(168, 140, 2, 0.2),
    ],
  };
}

/** Privacy: the threshold is closed, so the light stays inside its boundary. */
export function sealScene() {
  return {
    viewBox: '0 0 240 300',
    defs: [
      { id: 'seal-glow', type: 'radial', stops: [[0, 'gold', 0.16], [0.55, 'lapis', 0.1], [1, 'lapis', 0]] },
    ],
    elements: [
      { t: 'circle', cx: 120, cy: 182, r: 100, fill: 'seal-glow' },
      ...starField({ count: 16, maxY: 150, seed: 71 }).map(dot),
      { t: 'path', d: OUTER.path, stroke: 'cream', strokeWidth: 1, opacity: 0.42 },
      { t: 'path', d: INNER.path, stroke: 'cream', strokeWidth: 0.55, opacity: 0.2 },
      { t: 'circle', cx: 120, cy: 186, r: 30, stroke: 'gold', strokeWidth: 0.6, opacity: 0.32, dash: '1.5 6', className: 'theia-drift' },
      { t: 'path', d: 'M120 156V216', stroke: 'gold', strokeWidth: 0.5, opacity: 0.16, dash: '2 7' },
      star(120, 186, 6, 0.78, 'theia-star theia-star--focus'),
      { t: 'line', x1: 84, y1: 244, x2: 156, y2: 244, stroke: 'cream', strokeWidth: 0.55, opacity: 0.2 },
    ],
  };
}

/** Check-ins: light that comes back on its own, in a rhythm you set. */
export function returnScene() {
  const cy = 188;
  return {
    viewBox: '0 0 240 300',
    defs: [
      { id: 'return-glow', type: 'radial', stops: [[0, 'gold', 0.22], [0.55, 'lapis', 0.09], [1, 'lapis', 0]] },
    ],
    elements: [
      { t: 'circle', cx: 120, cy, r: 104, fill: 'return-glow' },
      ...starField({ count: 18, maxY: 158, seed: 83 }).map(dot),
      { t: 'path', d: OUTER.path, stroke: 'cream', strokeWidth: 0.8, opacity: 0.22 },
      ...[0, 1, 2].map((i) => ({
        t: 'circle', cx: 120, cy, r: 16 + i * 20, stroke: 'gold', strokeWidth: 0.6,
        opacity: round(0.38 - i * 0.1), className: 'theia-ripple', delay: round(i * 1.6),
      })),
      star(120, cy, 7.5, 0.92, 'theia-star theia-star--focus'),
      { t: 'path', d: crescentPath({ cx: 174, cy: 148, radius: 8, inner: 7.3, offset: 2.8 }), fill: 'moon', opacity: 0.4 },
    ],
  };
}

/** Finale: the threshold is a gate now, and the light is on your side of it. */
export function readyScene() {
  const cy = 190;
  return {
    viewBox: '0 0 240 300',
    defs: [
      { id: 'ready-bloom', type: 'radial', stops: [[0, 'goldLight', 0.5], [0.46, 'gold', 0.18], [1, 'gold', 0]] },
      { id: 'ready-core', type: 'radial', stops: [[0, 'goldLight', 0.7], [0.55, 'goldLight', 0.2], [1, 'goldLight', 0]] },
    ],
    elements: [
      { t: 'circle', cx: 120, cy, r: 132, fill: 'ready-bloom', className: 'theia-breathe' },
      ...starField({ count: 14, maxY: 148, seed: 61 }).map(dot),
      { t: 'path', d: archGeometry({ half: 70, spring: 208, d: 42, base: 274 }).path, stroke: 'gold', strokeWidth: 0.5, opacity: 0.22 },
      { t: 'path', d: OUTER.path, stroke: 'goldLight', strokeWidth: 1.2, opacity: 0.62 },
      { t: 'path', d: archGeometry({ half: 34, spring: 234, d: 20, base: 268 }).path, stroke: 'cream', strokeWidth: 0.7, opacity: 0.3 },
      { t: 'circle', cx: 120, cy: cy + 16, r: 48, fill: 'ready-core', opacity: 0.5, className: 'theia-breathe' },
      star(120, cy, 11, 1, 'theia-star theia-star--focus'),
      { t: 'line', x1: 34, y1: 268, x2: 206, y2: 268, stroke: 'goldLight', strokeWidth: 0.7, opacity: 0.4 },
    ],
  };
}

export const SCENES = {
  welcome: welcomeScene,
  night: nightScene,
  dawn: dawnScene,
  sun: sunScene,
  sight: sightScene,
  seal: sealScene,
  return: returnScene,
  ready: readyScene,
};

/** The narrative spine. Numbers, names and mythic anchors live in one place. */
export const MOVEMENTS = Object.freeze([
  {
    id: 'night',
    numeral: 'I',
    chapter: 'فصل یکم',
    name: 'شب',
    myth: 'سِلنه، ماه',
    scene: 'night',
  },
  {
    id: 'dawn',
    numeral: 'II',
    chapter: 'فصل دوم',
    name: 'سپیده',
    myth: 'ائوس، سپیده‌دم',
    scene: 'dawn',
  },
  {
    id: 'sun',
    numeral: 'III',
    chapter: 'فصل سوم',
    name: 'خورشید',
    myth: 'هلیوس، خورشید',
    scene: 'sun',
  },
  {
    id: 'sight',
    numeral: 'IV',
    chapter: 'فصل چهارم',
    name: 'دیدن',
    myth: 'تیا، نورِ بینا',
    scene: 'sight',
  },
]);

export const movementById = (id) => MOVEMENTS.find((item) => item.id === id) || null;
