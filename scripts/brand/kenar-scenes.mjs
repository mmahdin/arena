import { C, pebble, face, f } from './kenar-geometry.mjs';

const P = C;
const NIGHT = { stone: '#39463d', stone2: '#2e3a32', edge: '#7f9486', ink: '#d9d2c1' };
const PAPER = { stone: C.sand, stone2: C.sandDeep, edge: C.edge, ink: C.ink };

/* pebble() returns a path roughly 1.9× wide and 1.96× tall per unit, and is not
   centred on (cx,cy) — so everything here is specified in *bounding-box* px and
   placed by measured edges. */
const KW = 1.902, KH = 1.961;

function bboxOf(d) {
  const n = d.match(/-?\d+(?:\.\d+)?/g).map(Number);
  let x1 = Infinity, x2 = -Infinity, y1 = Infinity, y2 = -Infinity;
  for (let i = 0; i < n.length; i += 2) {
    x1 = Math.min(x1, n[i]); x2 = Math.max(x2, n[i]);
    y1 = Math.min(y1, n[i + 1]); y2 = Math.max(y2, n[i + 1]);
  }
  return { x1, x2, y1, y2, w: x2 - x1, h: y2 - y1 };
}

/* A shore you can place things on: quadratic curve, sample for point + tangent. */
function shorePath({ x1, y1, cx, cy, x2, y2 }) {
  const at = (t) => {
    const m = 1 - t;
    return [m * m * x1 + 2 * m * t * cx + t * t * x2, m * m * y1 + 2 * m * t * cy + t * t * y2];
  };
  const tForX = (x) => {
    let best = { d: Infinity, t: .5 };
    for (let i = 0; i <= 400; i++) {
      const t = i / 400;
      const d = Math.abs(at(t)[0] - x);
      if (d < best.d) best = { d, t };
    }
    return best.t;
  };
  return { d: `M${f(x1)} ${f(y1)} Q${f(cx)} ${f(cy)} ${f(x2)} ${f(y2)}`, at, tForX };
}

/* One stone, sitting on the shore at parameter t. `wide` is the bbox width. */
function stoneOn(s, t, o) {
  const [sx, sy] = s.at(t);
  const seed = o.seed ?? 21;
  const wobble = o.wobble ?? .06;
  const sq = o.squash ?? .84;               // flat-bottomed river stone
  const raw = pebble({ w: o.wide / KW, h: (o.tall ?? o.wide * .8) / KH, seed, wobble, flat: .8, tilt: o.tilt ?? -.02, cx: 0, cy: 0 });
  const b = bboxOf(raw);
  const dx = sx - (b.x1 + b.x2) / 2;
  const dy = sy - b.y2 * sq - (o.lift || 0);
  const cyFace = dy + (b.y1 + b.h * (o.faceDrop ?? .46)) * sq;
  const e = o.wide * (o.eye ?? .165);
  return `<g transform="translate(${f(dx)},${f(dy)}) scale(1,${sq})"><path d="${raw}" fill="${o.fill}" stroke="${o.edge}" stroke-width="${f(o.sw / sq)}" stroke-linejoin="round"/></g>` +
    face({ x: sx, y: cyFace, e, color: o.ink, mouth: o.mouth !== false, sw: Math.max(3.2, e * .26) });
}

/* ------------------------------------------------------------------ */
/* Every scene authored at the real slot aspect ratio, doubled.        */
/* ------------------------------------------------------------------ */
const S = {
  /* `x` is the stone's centre as a fraction of the frame width, so nothing can
     ever run off an edge no matter how the copy above it reflows. */
  // .welcome-art → 346×491 — the player's stone has just arrived
  welcome: {
    W: 692, H: 982,
    shore: { x1: -40, y1: 900, cx: 320, cy: 660, x2: 740, y2: 300 },
    big: { x: .60, wide: 300, tall: 248, fs: 4.0 },
    small: { x: .21, wide: 152, tall: 126, fs: 2.05, mouth: false, faceDrop: .40 },
    dots: 6, trail: true,
  },
  // .art-small → 306×288 — the first hello; Kenar alone, at rest
  small: {
    W: 612, H: 576, night: true,
    shore: { x1: -40, y1: 420, cx: 300, cy: 340, x2: 660, y2: 250 },
    big: { x: .50, wide: 296, tall: 245, fs: 4.0 },
    ripples: 3,
  },
  // .art-medium → 306×314 — a second stone appears far up the shore
  medium: {
    W: 612, H: 628, night: true,
    shore: { x1: -40, y1: 470, cx: 300, cy: 380, x2: 660, y2: 300 },
    big: { x: .64, wide: 268, tall: 222, fs: 3.8 },
    small: { x: .17, wide: 120, tall: 99, fs: 1.72, mouth: false, faceDrop: .40 },
  },
  // .art-large → 306×373 — closer
  large: {
    W: 612, H: 746, night: true,
    shore: { x1: -40, y1: 540, cx: 300, cy: 440, x2: 660, y2: 370 },
    big: { x: .62, wide: 272, tall: 226, fs: 3.85 },
    small: { x: .22, wide: 132, tall: 109, fs: 1.85, mouth: false, faceDrop: .40 },
  },
  // .art-tall → 314×491 — side by side, nearly touching
  tall: {
    W: 628, H: 982, night: true,
    shore: { x1: -40, y1: 700, cx: 310, cy: 560, x2: 678, y2: 470 },
    big: { x: .60, wide: 288, tall: 239, fs: 3.95 },
    small: { x: .26, wide: 142, tall: 118, fs: 1.95, mouth: false, faceDrop: .42 },
  },
  // .ready-art → 334×470 — walking the shore together
  ready: {
    W: 668, H: 940,
    shore: { x1: -40, y1: 690, cx: 300, cy: 550, x2: 718, y2: 450 },
    big: { x: .60, wide: 284, tall: 236, fs: 3.9 },
    small: { x: .25, wide: 144, tall: 120, fs: 2.0, faceDrop: .42 },
    dots: 5, ripples: 2, trail: true,
  },
  // .checkins-art → 342×295
  checkins: {
    W: 684, H: 590,
    shore: { x1: -40, y1: 420, cx: 320, cy: 350, x2: 734, y2: 300 },
    big: { x: .50, wide: 300, tall: 249, fs: 4.0 },
    ripples: 3, dots: 4,
  },
  // .privacy-art → 102×116 — a stone kept inside an unbroken ring
  privacy: {
    W: 306, H: 348,
    shore: { x1: -30, y1: 258, cx: 150, cy: 248, x2: 336, y2: 238 },
    big: { x: .50, wide: 154, tall: 128, fs: 2.2 },
    ring: true,
  },
};

export function scene(key) {
  const s = S[key];
  const { W, H } = s;
  const night = !!s.night;
  const c = night ? NIGHT : PAPER;
  const k = W / 700;
  const sh = shorePath(s.shore);
  const out = [];

  const posOf = (o) => sh.at(o.x != null ? sh.tForX(W * o.x) : o.t);
  const [bx, by] = posOf(s.big);

  out.push(`<path d="${sh.d}" fill="none" stroke="${c.ink}" stroke-width="${f(6.4 * k)}" stroke-linecap="round" opacity="${night ? .78 : 1}"/>`);

  // Quiet sand grains beneath the stone — the only warm accent in the frame.
  if (s.ripples) {
    for (let i = 0; i < 3; i++) {
      const w = (s.big.wide * .30 + i * 30 * k);
      const ry = by + (18 + i * 13) * k;
      out.push(`<circle cx="${f(bx - w)}" cy="${f(ry)}" r="${f((2.6 - i * .5) * k)}" fill="${P.clay}" opacity="${(0.44 - i * .10).toFixed(2)}"/>`);
      out.push(`<circle cx="${f(bx + w)}" cy="${f(ry + 4 * k)}" r="${f((2.2 - i * .4) * k)}" fill="${P.clay}" opacity="${(0.36 - i * .09).toFixed(2)}"/>`);
    }
  }

  if (s.ring) {
    const [rx0, ry0] = posOf(s.big);
    const cyc = ry0 - s.big.tall * .52;
    out.push(`<ellipse cx="${f(rx0)}" cy="${f(cyc)}" rx="${f(s.big.wide * .76)}" ry="${f(s.big.tall * .78)}" fill="none" stroke="${P.clay}" stroke-width="${f(3.2 * k)}" stroke-linecap="round" stroke-dasharray="2 11" opacity=".55"/>`);
  }

  if (s.small) out.push(stoneOn(sh, sh.tForX(W * (s.small.x ?? 0.4)), { ...s.small, fill: c.stone2, edge: c.edge, ink: c.ink, sw: 5 * k }));
  out.push(stoneOn(sh, sh.tForX(W * (s.big.x ?? 0.5)), { ...s.big, fill: c.stone, edge: c.edge, ink: c.ink, sw: 6.4 * k }));

  // the clay trail: a few grains the arriving stone left on the sand
  if (s.trail) {
    const [tx, ty] = sh.at(.06);
    out.push(`<circle cx="${f(tx)}" cy="${f(ty - 10 * k)}" r="${f(4 * k)}" fill="${P.clay}" opacity=".5"/>`);
    out.push(`<circle cx="${f(tx + 34 * k)}" cy="${f(ty - 26 * k)}" r="${f(3 * k)}" fill="${P.clay}" opacity=".34"/>`);
    out.push(`<path d="M${f(tx + 58 * k)} ${f(ty - 40 * k)} C${f(tx + 88 * k)} ${f(ty - 62 * k)} ${f(tx + 128 * k)} ${f(ty - 70 * k)} ${f(tx + 166 * k)} ${f(ty - 62 * k)}" fill="none" stroke="${P.clay}" stroke-width="${f(2.6 * k)}" stroke-linecap="round" opacity=".34"/>`);
  }

  if (s.dots) {
    for (let i = 0; i < s.dots; i++) {
      const [dx, dy] = sh.at(.05 + i * .05);
      out.push(`<circle cx="${f(dx)}" cy="${f(dy + 11 * k + (i % 2) * 8 * k)}" r="${f((2.8 - i * .16) * k)}" fill="${P.clay}" opacity="${(0.4 - i * .035).toFixed(2)}"/>`);
    }
  }
  return out.join('\n');
}

export const SCENES = S;
export const DIM = Object.fromEntries(Object.entries(S).map(([k, v]) => [k, [v.W, v.H]]));
