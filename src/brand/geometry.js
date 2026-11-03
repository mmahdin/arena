/**
 * Hamrah brand geometry — «همسنگ» (ham-sang): the companion stone.
 *
 * Two stones lean together and leave a warm ember in the crook between them.
 * Every character in the identity is cut from the same quarry: one parametric
 * pebble, described by a soft superellipse, a leaning crown and a flat base.
 *
 * The generator is deterministic (no Math.random), so the same stone is drawn
 * identically on every device, in every render, forever.
 */

const TAU = Math.PI * 2;

/** Deterministic hash → [0, 1). Keeps stone "character" stable across renders. */
function hash(seed) {
  let x = Math.sin(seed * 127.1 + 311.7) * 43758.5453;
  return x - Math.floor(x);
}

/**
 * Sample a pebble outline.
 * x, base: the stone sits on `base` and is centred on `x`.
 * w, h: half-width and height.
 * lean: how far the crown leans (positive leans right).
 * n, m: superellipse exponents — lower n is boxier, higher m is pointier.
 * wobble: amount of hand-cut irregularity.
 */
export function pebbleSamples({
  x = 0,
  base = 0,
  w = 40,
  h = 50,
  lean = 0,
  n = 2.05,
  m = 2.55,
  taper = 0.12,
  wobble = 1,
  seed = 1,
  samples = 84,
} = {}) {
  const pts = [];
  const s1 = hash(seed) * TAU;
  const s2 = hash(seed + 7.3) * TAU;
  for (let i = 0; i < samples; i += 1) {
    const th = (TAU * i) / samples;
    const c = Math.cos(th);
    const s = Math.sin(th);
    let px = Math.sign(c) * Math.abs(c) ** (2 / n) * w;
    const py = Math.sign(s) * Math.abs(s) ** (2 / m) * h;
    const ynorm = py / h; // -1 crown … +1 base
    if (ynorm < 0) {
      const t = -ynorm;
      px *= 1 - taper * t ** 2.2;
      px += lean * t ** 1.55;
    }
    px += wobble * 0.55 * Math.sin(3 * th + s1) * (1 - 0.55 * Math.abs(ynorm));
    px += wobble * 0.32 * Math.sin(5 * th + s2) * (1 - 0.45 * Math.abs(ynorm));
    pts.push([x + px, base - h + py]);
  }
  // A stone has a flat base: nothing sits below the ground line.
  return pts.map(([px, py]) => [px, Math.min(py, base)]);
}

/** Evenly resample a closed polygon so the curve stays calm. */
export function resample(points, count = 96) {
  const n = points.length;
  const lengths = [0];
  for (let i = 0; i < n; i += 1) {
    const [x1, y1] = points[i];
    const [x2, y2] = points[(i + 1) % n];
    lengths.push(lengths[lengths.length - 1] + Math.hypot(x2 - x1, y2 - y1));
  }
  const total = lengths[lengths.length - 1] || 1;
  const out = [];
  for (let k = 0; k < count; k += 1) {
    const target = (total * k) / count;
    let i = 0;
    while (i < n - 1 && lengths[i + 1] < target) i += 1;
    const seg = lengths[i + 1] - lengths[i];
    const t = seg === 0 ? 0 : (target - lengths[i]) / seg;
    const [x1, y1] = points[i];
    const [x2, y2] = points[(i + 1) % n];
    out.push([x1 + (x2 - x1) * t, y1 + (y2 - y1) * t]);
  }
  return out;
}

const r1 = (value) => Math.round(value * 10) / 10;

/** Closed Catmull-Rom spline → compact cubic-bezier path. */
export function closedSpline(points, tension = 1) {
  const n = points.length;
  let d = `M${r1(points[0][0])},${r1(points[0][1])}`;
  for (let i = 0; i < n; i += 1) {
    const p0 = points[(i - 1 + n) % n];
    const p1 = points[i];
    const p2 = points[(i + 1) % n];
    const p3 = points[(i + 2) % n];
    const c1x = p1[0] + ((p2[0] - p0[0]) / 6) * tension;
    const c1y = p1[1] + ((p2[1] - p0[1]) / 6) * tension;
    const c2x = p2[0] - ((p3[0] - p1[0]) / 6) * tension;
    const c2y = p2[1] - ((p3[1] - p1[1]) / 6) * tension;
    d += `C${r1(c1x)},${r1(c1y)} ${r1(c2x)},${r1(c2y)} ${r1(p2[0])},${r1(p2[1])}`;
  }
  return `${d}Z`;
}

/** Open spline for freehand clay lines (ground, thread, motion). */
export function openSpline(points, tension = 1) {
  const n = points.length;
  let d = `M${r1(points[0][0])},${r1(points[0][1])}`;
  for (let i = 0; i < n - 1; i += 1) {
    const p0 = points[Math.max(0, i - 1)];
    const p1 = points[i];
    const p2 = points[i + 1];
    const p3 = points[Math.min(n - 1, i + 2)];
    const c1x = p1[0] + ((p2[0] - p0[0]) / 6) * tension;
    const c1y = p1[1] + ((p2[1] - p0[1]) / 6) * tension;
    const c2x = p2[0] - ((p3[0] - p1[0]) / 6) * tension;
    const c2y = p2[1] - ((p3[1] - p1[1]) / 6) * tension;
    d += `C${r1(c1x)},${r1(c1y)} ${r1(c2x)},${r1(c2y)} ${r1(p2[0])},${r1(p2[1])}`;
  }
  return d;
}

/** A finished pebble outline as path data. */
export function pebblePath(options = {}) {
  const { count = 88, tension = 1, ...rest } = options;
  return closedSpline(resample(pebbleSamples(rest), count), tension);
}

/** A hand-drawn-looking clay line that rests on the ground. */
export function groundLine({ x1, x2, y, sag = 2.4, steps = 9 }) {
  const pts = [];
  for (let i = 0; i <= steps; i += 1) {
    const t = i / steps;
    pts.push([x1 + (x2 - x1) * t, y + Math.sin(t * Math.PI) * sag * 0.5 + Math.sin(t * 9.1) * 0.5]);
  }
  return openSpline(pts, 0.9);
}

/** The ember: a circle, always the only warm thing in the frame. */
export const ember = (cx, cy, r = 6) => ({ cx, cy, r });
