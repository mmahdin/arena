/* ============================================================
   KENAR — visual identity kit (generator)
   Concept: "کنار" = beside / the shoreline.
   Two river stones resting side by side on a shore line.
   ============================================================ */
export const C = {
  paper: '#f7f3e9',      // the page: warm, dusty paper
  sand:  '#d8c7a6',      // stone body
  sandDeep: '#bfa87e',   // the smaller stone
  edge:  '#8a6849',      // stone outline: riverbank ochre
  ink:   '#1e2a23',      // deep pine ink
  clay:  '#c37c52',      // warmth accent (ember / sunset on water)
  sage:  '#a6c1ab',      // plant
  water: '#b8cbc3',      // water
  night: '#101510',
  nightLine: '#d9d2c1',
  nightStone: '#39463d',
  nightStone2: '#2e3a32',
  nightEdge: '#7f9486',
};

/* ---------------- primitives ---------------- */
export const f = (n) => Number(n.toFixed(2));

export function rng(seed) {
  let s = (seed >>> 0) || 1;
  return () => { s ^= s << 13; s >>>= 0; s ^= s >>> 17; s ^= s << 5; s >>>= 0; return s / 4294967296; };
}

export function smoothClosed(points, tension = 0.92) {
  const n = points.length;
  let d = `M${f(points[0][0])} ${f(points[0][1])}`;
  for (let i = 0; i < n; i++) {
    const p0 = points[(i - 1 + n) % n], p1 = points[i], p2 = points[(i + 1) % n], p3 = points[(i + 2) % n];
    const c1 = [p1[0] + ((p2[0] - p0[0]) / 6) * tension, p1[1] + ((p2[1] - p0[1]) / 6) * tension];
    const c2 = [p2[0] - ((p3[0] - p1[0]) / 6) * tension, p2[1] - ((p3[1] - p1[1]) / 6) * tension];
    d += ` C${f(c1[0])} ${f(c1[1])} ${f(c2[0])} ${f(c2[1])} ${f(p2[0])} ${f(p2[1])}`;
  }
  return `${d} Z`;
}

function relax(pts, rounds = 4, k = 0.5) {
  let cur = pts;
  for (let r = 0; r < rounds; r++) {
    const n = cur.length;
    cur = cur.map((p, i) => {
      const a = cur[(i - 1 + n) % n], b = cur[(i + 1) % n];
      return [p[0] + ((a[0] + b[0]) / 2 - p[0]) * k, p[1] + ((a[1] + b[1]) / 2 - p[1]) * k];
    });
  }
  return cur;
}

/** A river pebble: superellipse, gentle harmonics, flat base so it rests. */
export function pebble({ w = 60, h = 46, seed = 7, wobble = 0.07, flat = 0.42, tilt = 0, cx = 0, cy = 0, n = 2.05, round = 5 } = {}) {
  const random = rng(seed);
  const h1 = (random() - 0.5) * wobble * 2, h2 = (random() - 0.5) * wobble, p1 = random() * 6.28, p2 = random() * 6.28;
  const count = 72, raw = [];
  for (let i = 0; i < count; i++) {
    const a = (i / count) * Math.PI * 2, ca = Math.cos(a), sa = Math.sin(a);
    const wob = 1 + h1 * Math.cos(2 * a + p1) + h2 * Math.cos(3 * a + p2);
    const r = 1 / Math.pow(Math.pow(Math.abs(ca), n) + Math.pow(Math.abs(sa), n), 1 / n);
    raw.push([ca * r * w * wob, sa * r * h * wob]);
  }
  const baseY = h * flat;
  const pts = relax(raw.map(([x, y]) => [x, Math.min(y, baseY)]), round, 0.5);
  const cos = Math.cos(tilt), sin = Math.sin(tilt);
  return smoothClosed(pts.map(([x, y]) => [cx + x * cos - y * sin, cy + x * sin + y * cos]));
}

/** The companion's face: closed, unhurried eyes and a small mouth. */
/* Closed, resting eyes — an arc with the ends dropped, like a sleeping curve.
   Deliberately not a `^` caret: the face must read calm, never happy-clown. */
/* Closed, resting eyes. `e` is the half-width in px, so the face scales with
   the stone instead of with the scene. */
export function face({ x = 0, y = 0, e = 16, color = C.ink, sw = 4.2, mouth = true } = {}) {
  const md = e * 0.60;
  const mw = e * 0.46;
  const gap = e * 0.26;
  return `<g transform="translate(${x},${y})" stroke="${color}" stroke-width="${f(sw)}" stroke-linecap="round" fill="none">
    <path d="M${f(-e)} -2.4 C${f(-e * .74)} 5.4 ${f(-e * .46)} 5.4 ${f(-gap)} -1.6"/>
    <path d="M${f(gap)} -1.6 C${f(e * .46)} 5.4 ${f(e * .74)} 5.4 ${f(e)} -2.4"/>
    ${mouth ? `<path d="M${f(-mw)} ${f(md)} C${f(-mw * .45)} ${f(md + e * .26)} ${f(mw * .45)} ${f(md + e * .26)} ${f(mw)} ${f(md)}"/>` : ''}
  </g>`;
}

/** A shore line — the recurring horizon of the identity. */
export function shore(x1, x2, y, lift = 9) {
  const mid = (x1 + x2) / 2;
  return `M${f(x1)} ${f(y + 2)} C${f(x1 + (mid - x1) * 0.66)} ${f(y + lift)} ${f(mid + (x2 - mid) * 0.3)} ${f(y + lift)} ${f(x2)} ${f(y - lift * 0.4)}`;
}

/** Water ripple under a stone, as a quiet echo. */
export function ripple(cx, cy, rx, ry, color, { sw = 2.2, op = 0.55 } = {}) {
  return `<ellipse cx="${f(cx)}" cy="${f(cy)}" rx="${f(rx)}" ry="${f(ry)}" fill="none" stroke="${color}" stroke-width="${sw}" opacity="${op}"/>`;
}

/** A companion: stone + optional companion stone beside it. Positions are local. */
export function companion({
  big, small, palette, face: faceOpts = null, sw = 5.4, swSmall = 5,
} = {}) {
  const p = palette;
  let out = '';
  if (small) {
    out += `<path d="${pebble(small)}" fill="${p.stone2}" stroke="${p.edge}" stroke-width="${swSmall}" stroke-linejoin="round"/>`;
  }
  out += `<path d="${pebble(big)}" fill="${p.stone}" stroke="${p.edge}" stroke-width="${sw}" stroke-linejoin="round"/>`;
  if (faceOpts) out += face({ color: p.ink, ...faceOpts });
  return out;
}

export const paletteLight = { stone: C.sand, stone2: C.sandDeep, edge: C.edge, ink: C.ink };
export const paletteNight = { stone: C.nightStone, stone2: C.nightStone2, edge: C.nightEdge, ink: C.nightLine };
