/**
 * Hamrah identity scenes — «همسنگ» (ham-sang), the companion stone.
 *
 * The whole visual language is three things:
 *   1. pebbles  — ink stones cut from one parametric quarry
 *   2. the ember — a single warm dot: attention, feeling, the thing worth noticing
 *   3. a ground line — a hand-pressed clay stroke that everything rests on
 *
 * A companion is one upright pebble with an ember at its heart and two small
 * feet. It leans, it strides, it sits; it never has a face. Feeling is carried
 * by posture and by one warm point of light — the same restraint the brand mark
 * asks for.
 *
 * Scenes are returned as plain layer data so the React renderer, the brand
 * documentation and the print assets always draw identical geometry.
 */
import { pebblePath, groundLine } from './geometry.js';

export const PALETTES = {
  light: { stone: '#23201b', ember: '#c1664c', ground: '#c9c1b4', marble: '#f3f2ee' },
  dark: { stone: '#f0ede6', ember: '#c9705a', ground: '#3d3934', marble: '#12110d' },
  beige: { stone: '#23201b', ember: '#a8503c', ground: '#a2937f', marble: '#cec4b8' },
};

const r1 = (n) => Math.round(n * 10) / 10;
const circle = (cx, cy, r, fill) => ({ t: 'circle', cx: r1(cx), cy: r1(cy), r: r1(r), fill });
const path = (d, fill, extra = {}) => ({ t: 'path', d, fill, ...extra });
const stroke = (d, color, width, extra = {}) => path(d, 'none', { stroke: color, width: r1(width), cap: 'round', ...extra });

/* ------------------------------------------------------------- primitives */

/** Two tapered legs with small flat feet. `stride` shifts the feet apart. */
function legs(x, y, spread, length, width, color, stride = 0) {
  const out = [];
  for (const side of [-1, 1]) {
    const sx = x + side * spread;
    const dx = side * (1.6 + stride * 6);
    out.push(stroke(
      `M${r1(sx)},${r1(y - 3)}C${r1(sx + dx * 0.2)},${r1(y + length * 0.42)} ${r1(sx + dx * 0.72)},${r1(y + length * 0.78)} ${r1(sx + dx)},${r1(y + length)}`,
      color, width,
    ));
    out.push(stroke(
      `M${r1(sx + dx)},${r1(y + length)}q${r1(side * 5.8)},${r1(2.8)} ${r1(side * 9)},${r1(0.9)}`,
      color, width,
    ));
  }
  return out;
}

/**
 * A Hamrah companion. Origin is the point between its feet.
 * pose: 'stand' | 'walk' | 'sit' | 'wave' | 'hold' | 'reach'
 */
export function companion({
  x = 0,
  base = 0,
  scale = 1,
  w = 33,
  h = 52,
  lean = 8,
  seed = 1.3,
  pose = 'stand',
  ember: showEmber = true,
  emberY = 0.42,
  emberScale = 1,
  ground = null,
  palette = PALETTES.light,
  flip = false,
}) {
  const p = palette;
  const s = scale;
  const dir = flip ? -1 : 1;
  const layers = [];
  if (ground) layers.push(stroke(groundLine(ground), p.ground, 3.4));

  const stride = pose === 'walk' || pose === 'hold' ? 0.62 : 0;
  const legLength = pose === 'sit' ? 0 : 23;
  const legW = 5.6 * s;
  const bodyBase = base - legLength * s;
  const bodyH = h * s;
  const bodyW = w * s;

  if (pose === 'sit') {
    // At rest: a low heap with a smaller companion stone leaning against it.
    const seated = pebblePath({
      x, base: base - 2 * s, w: bodyW * 1.28, h: bodyH * 0.54, lean: lean * 0.4,
      seed, n: 1.86, m: 2.92, taper: 0.03, wobble: 1.5,
    });
    layers.push(path(seated, p.stone));
    const small = pebblePath({
      x: x + dir * bodyW * 1.02, base: base - 2 * s,
      w: bodyW * 0.46, h: bodyH * 0.4, lean: -dir * 16, seed: seed + 5.4, n: 1.9, m: 2.8, wobble: 1.4,
    });
    layers.push(path(small, p.stone));
    layers.push(circle(x + dir * bodyW * 0.86, base - bodyH * 0.5, Math.max(4, bodyW * 0.16), p.ember));
    return layers;
  }
  {
    const torso = pebblePath({ x, base: bodyBase, w: bodyW, h: bodyH, lean: lean * dir, seed, n: 1.92, m: 2.62 });
    layers.push(path(torso, p.stone));
    layers.push(...legs(x, bodyBase, bodyW * 0.3, legLength * s, legW, p.stone, stride));
  }

  if (pose === 'wave') {
    layers.push(stroke(
      `M${r1(x + dir * bodyW * 0.92)},${r1(bodyBase - bodyH * 0.58)}q${r1(dir * 17 * s)},${r1(-4 * s)} ${r1(dir * 23 * s)},${r1(-21 * s)}`,
      p.stone, 5.2 * s,
    ));
  }
  if (pose === 'reach') {
    // One arm reaches out and slightly up, toward whoever is next.
    layers.push(stroke(
      `M${r1(x + dir * bodyW * 0.86)},${r1(bodyBase - bodyH * 0.6)}`
      + `C${r1(x + dir * bodyW * 1.3)},${r1(bodyBase - bodyH * 0.66)} ${r1(x + dir * bodyW * 1.6)},${r1(bodyBase - bodyH * 0.82)} ${r1(x + dir * bodyW * 1.86)},${r1(bodyBase - bodyH * 0.98)}`,
      p.stone, 5.2 * s,
    ));
  }
  if (pose === 'hold') {
    // One arm stays behind, one reaches level with the neighbour's shoulders.
    layers.push(stroke(
      `M${r1(x + dir * bodyW * 0.9)},${r1(bodyBase - bodyH * 0.52)}`
      + `C${r1(x + dir * bodyW * 1.34)},${r1(bodyBase - bodyH * 0.54)} ${r1(x + dir * bodyW * 1.7)},${r1(bodyBase - bodyH * 0.6)} ${r1(x + dir * bodyW * 2.0)},${r1(bodyBase - bodyH * 0.66)}`,
      p.stone, 5.2 * s,
    ));
  }

  if (pose === 'offer') {
    // Both arms rise in front of the body to cup the ember at chest height.
    const cy = bodyBase - bodyH * 0.46;
    for (const side of [-1, 1]) {
      layers.push(stroke(
        `M${r1(x + side * bodyW * 0.9)},${r1(bodyBase - bodyH * 0.5)}`
        + `C${r1(x + side * bodyW * 1.22)},${r1(bodyBase - bodyH * 0.62)} ${r1(x + side * bodyW * 0.5)},${r1(cy + 12 * s)} ${r1(x + side * bodyW * 0.3)},${r1(cy + 4 * s)}`,
        p.stone, 5.2 * s,
      ));
    }
    layers.push(circle(x, cy, 7.4 * s, p.ember));
  }

  if (showEmber) {
    if (pose === 'offer') return layers;
    const minR = 4.2;
    const radius = Math.max(minR, Math.min(bodyW * 0.19, 8.4 * s)) * emberScale;
    layers.push(circle(x + dir * bodyW * 0.04, bodyBase - bodyH * emberY, radius, p.ember));
  }
  return layers;
}

/** A stone that simply sits: flat base, quiet asymmetry. */
export function stone({ x = 0, base = 0, w = 40, h = 26, lean = 6, seed = 5, palette = PALETTES.light, scale = 1 }) {
  return path(
    pebblePath({ x, base, w: w * scale, h: h * scale, lean, seed, n: 1.88, m: 2.95, taper: 0.04, wobble: 1.4 }),
    palette.stone,
  );
}
export const stones = (options) => [stone(options)];

/** A cairn — the path-marker of stone upon stone, crowned by the ember. */
export function cairn({ x = 0, base = 0, scale = 1, count = 3, palette = PALETTES.light, spark = true, sparkR = 6 }) {
  const layers = [];
  const specs = [
    { w: 46, h: 24, dx: 0, seed: 8.1, lean: 6 },
    { w: 33, h: 18, dx: 2, seed: 2.6, lean: -5 },
    { w: 23, h: 13, dx: -1, seed: 6.4, lean: 5 },
    { w: 15, h: 10, dx: 1, seed: 9.7, lean: -4 },
    { w: 9, h: 7, dx: 0, seed: 3.9, lean: 3 },
  ];
  let top = base;
  for (let i = 0; i < Math.min(count, specs.length); i += 1) {
    const sp = specs[i];
    layers.push(stone({ x: x + sp.dx * scale, base: top, w: sp.w, h: sp.h, lean: sp.lean, seed: sp.seed, palette, scale }));
    top -= sp.h * 2 * scale * 0.92;
  }
  if (spark) layers.push(circle(x + 2 * scale, top + 1 * scale, sparkR * scale, palette.ember));
  return layers;
}

/**
 * A vessel stone: a wide pebble with a carved hollow that holds the ember.
 * The hollow is drawn as a second filled circle in the background colour so it
 * reads identically in every renderer (no fill-rule dependency).
 */
export function vessel({
  x = 0, base = 0, scale = 1, w = 58, h = 34, lean = -4, seed = 7.7,
  palette = PALETTES.light, sparkR = 8, bg = null,
}) {
  const stoneW = w * scale;
  const stoneH = h * scale;
  const hollowR = stoneW * 0.30;
  const hollowY = base - stoneH * 1.16;
  return {
    layers: [
      path(pebblePath({ x, base, w: stoneW, h: stoneH, lean, seed, n: 1.88, m: 2.95, taper: 0.04 }), palette.stone),
      circle(x - stoneW * 0.06, hollowY, hollowR, bg || palette.marble),
      circle(x - stoneW * 0.14, hollowY + hollowR * 0.1, sparkR * scale, palette.ember),
    ],
  };
}

/* ----------------------------------------------------------------- scenes */

const W = 360;
const H = 330;

export const SCENES = {
  /** Welcome — two companions and a thread of attention between them. */
  welcome: (palette = PALETTES.light) => ({
    viewBox: `0 0 ${W} ${H}`,
    layers: [
      stroke(groundLine({ x1: 52, x2: 308, y: 292, sag: 3.6 }), palette.ground, 3.4),
      ...companion({ x: 132, base: 292, scale: 1.16, lean: 10, ember: false, palette }),
      ...companion({ x: 232, base: 292, scale: 1.0, lean: -11, ember: false, flip: true, palette }),
      stroke('M152,198C170,210 196,210 212,198', palette.stone, 4.4),
      circle(182, 152, 9, palette.ember),
    ],
  }),

  /** Name — one companion looks up at the ember it is about to be given. */
  name: (palette = PALETTES.light) => ({
    viewBox: `0 0 ${W} ${H}`,
    layers: [
      stroke(groundLine({ x1: 116, x2: 246, y: 286, sag: 3.2 }), palette.ground, 3.2),
      ...companion({ x: 178, base: 286, scale: 1.2, lean: 12, ember: false, palette }),
      circle(206, 116, 8, palette.ember),
    ],
  }),

  /** Simple intro — the first stone is laid; a companion sits with it. */
  'simple-intro': (palette = PALETTES.dark) => ({
    viewBox: `0 0 ${W} ${H}`,
    layers: [
      stroke(groundLine({ x1: 64, x2: 296, y: 282, sag: 3.4 }), palette.ground, 3.2),
      ...companion({ x: 126, base: 282, scale: 1.1, lean: 9, ember: false, palette }),
      ...stones({ x: 240, base: 282, w: 46, h: 30, lean: -7, seed: 4.2, palette }),
      circle(232, 218, 7.4, palette.ember),
    ],
  }),

  /** Deeper intro — stones grow into a stair, and the ember rests on the tallest. */
  'deeper-intro': (palette = PALETTES.dark) => ({
    viewBox: `0 0 ${W} ${H}`,
    layers: [
      stroke(groundLine({ x1: 52, x2: 308, y: 288, sag: 3.8 }), palette.ground, 3.4),
      ...stones({ x: 96, base: 288, w: 27, h: 18, lean: 5, seed: 1.7, palette }),
      ...stones({ x: 174, base: 288, w: 34, h: 27, lean: -6, seed: 7.3, palette }),
      ...stones({ x: 254, base: 288, w: 42, h: 40, lean: 4, seed: 3.1, palette }),
      // the ember nests in a shallow carved dip — never balanced on top
      path('M236,208a22,22 0 1,0 44,0a22,22 0 1,0 -44,0Z', palette.marble),
      circle(258, 210, 7.6, palette.ember),
    ],
  }),

  /** Know intro — a path of stones leads to a companion at rest. */
  'know-intro': (palette = PALETTES.dark) => ({
    viewBox: `0 0 ${W} ${H}`,
    layers: [
      stroke(groundLine({ x1: 44, x2: 316, y: 278, sag: 3.8 }), palette.ground, 3.4),
      ...stones({ x: 76, base: 278, w: 23, h: 15, lean: 9, seed: 2.2, palette }),
      ...stones({ x: 134, base: 278, w: 28, h: 19, lean: -6, seed: 5.8, palette }),
      ...stones({ x: 194, base: 278, w: 24, h: 16, lean: 5, seed: 8.4, palette }),
      ...companion({ x: 268, base: 278, scale: 1.06, pose: 'stand', lean: -6, ember: false, flip: true, palette }),
      // the one warm thing: the companion's own ember
      circle(266, 196, 8, palette.ember),
    ],
  }),

  /** Almost intro — a tall cairn, and the companion who built it. */
  'almost-intro': (palette = PALETTES.dark) => ({
    viewBox: `0 0 ${W} ${H}`,
    layers: [
      stroke(groundLine({ x1: 54, x2: 306, y: 292, sag: 3.6 }), palette.ground, 3.4),
      ...cairn({ x: 158, base: 292, scale: 1.12, count: 4, palette, sparkR: 7 }),
      ...companion({ x: 268, base: 292, scale: 0.98, lean: -9, ember: false, flip: true, palette }),
    ],
  }),

  /** Ready — the two walk on together. */
  ready: (palette = PALETTES.light) => ({
    viewBox: `0 0 ${W} ${H}`,
    layers: [
      stroke(groundLine({ x1: 44, x2: 316, y: 292, sag: 3.8 }), palette.ground, 3.4),
      ...companion({ x: 128, base: 292, scale: 1.2, pose: 'walk', lean: 14, ember: false, palette }),
      ...companion({ x: 238, base: 292, scale: 1.04, pose: 'walk', lean: -13, ember: false, flip: true, palette }),
      stroke('M152,214C168,222 194,222 210,214', palette.stone, 4.2),
      circle(182, 146, 9.4, palette.ember),
    ],
  }),

  /** Privacy — the companion keeps watch over a vessel stone that holds the ember. */
  privacy: (palette = PALETTES.light) => {
    // A wide, low stone with a carved hollow: the ember is kept, not displayed.
    const v = vessel({ x: 236, base: 268, scale: 1.05, palette, sparkR: 10 });
    return {
      viewBox: `0 0 ${W} ${H}`,
      layers: [
        stroke(groundLine({ x1: 50, x2: 318, y: 268, sag: 3.2 }), palette.ground, 3.2),
        ...companion({ x: 106, base: 268, scale: 1.14, lean: 7, ember: false, palette }),
        ...v.layers,
      ],
    };
  },

  /** A resting stone carrying the day's ember. */
  day: (palette = PALETTES.beige) => {
    const v = vessel({ x: 196, base: 178, scale: 0.9, palette, sparkR: 7.4, bg: palette.marble });
    return {
      viewBox: '0 0 360 220',
      layers: [
        stroke(groundLine({ x1: 88, x2: 278, y: 178, sag: 3 }), palette.ground, 3),
        ...stones({ x: 96, base: 178, w: 22, h: 13, lean: 7, seed: 2.9, palette }),
        ...v.layers,
      ],
    };
  },

  /** Insight — stones joined by a clay thread: a pattern seen from above. */
  insight: (palette = PALETTES.dark) => ({
    viewBox: '0 0 390 215',
    layers: [
      path('M88,76C152,52 228,92 288,130', 'none', { stroke: palette.stone, width: 3, cap: 'round', dash: '1 10' }),
      path('M98,150C160,168 234,152 288,114', 'none', { stroke: palette.stone, width: 3, cap: 'round', dash: '1 10' }),
      ...stones({ x: 88, base: 76, w: 20, h: 13, lean: 8, seed: 3.3, palette }),
      ...stones({ x: 178, base: 90, w: 25, h: 16, lean: -6, seed: 7.1, palette }),
      ...stones({ x: 288, base: 130, w: 21, h: 13, lean: 4, seed: 1.9, palette }),
      ...stones({ x: 98, base: 150, w: 18, h: 11, lean: -8, seed: 9.2, palette }),
      ...stones({ x: 234, base: 158, w: 22, h: 14, lean: 6, seed: 4.8, palette }),
      circle(196, 120, 7.6, palette.ember),
    ],
  }),

  /**
   * The mark itself — «همسنگ».
   * A small stone leans into a taller one; the ember sits in the notch where
   * they meet, so the gap is never empty. Optically centred in a 240 square.
   */
  mark: (palette = PALETTES.light) => ({
    viewBox: '0 0 240 240',
    layers: [
      path(pebblePath({ x: 84, base: 234, w: 46, h: 58, lean: 22, seed: 3.2, n: 1.95, m: 2.7 }), palette.stone),
      path(pebblePath({ x: 172, base: 234, w: 50, h: 112, lean: -6, seed: 6.4, n: 1.95, m: 2.7 }), palette.stone),
      circle(126, 128, 8.5, palette.ember),
    ],
  }),
};

/** A compact scene for small spots: the companion and its ember. */
export function companionScene(palette = PALETTES.light, options = {}) {
  return {
    viewBox: '0 0 160 160',
    layers: [
      stroke(groundLine({ x1: 38, x2: 122, y: 132, sag: 2.4 }), palette.ground, 2.6),
      ...companion({ x: 80, base: 132, scale: 0.82, ember: false, ...options, palette }),
      circle(96, 54, 6.4, palette.ember),
    ],
  };
}
