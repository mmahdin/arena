/**
 * سپیدهٔ درون — "The Inner Dawn".
 *
 * One continuous illustrated scene carries the whole onboarding story: the sky
 * over a Persian garden still-water (حوض), from the darkest night to the risen
 * sun. Each onboarding chapter advances a single `progress` value (0 = full
 * night, 1 = sunrise), so moving through the flow literally raises the light.
 *
 * The symbolism binds Theia's myth to the product's purpose:
 *   - Theia, Titaness of sight, mother of the Sun (Helios), Moon (Selene) and
 *     Dawn (Eos) — here her daughters rise in order: the crescent moon lingers
 *     while the sun (as shamseh, the Persian sunburst medallion) climbs a
 *     two-peak Alborz-like ridge and finally stands over the howz.
 *   - Still water mirrors the light only when the surface is calm — attention,
 *     "looking inward", is the product. The pool fills with reflection as the
 *     user moves deeper into the onboarding.
 *   - The sparse stars thin as dawn approaches; one faint turquoise star
 *     (the stone she taught to shine, firouzeh) never fully goes out.
 *   - Two cypress silhouettes (سرو) frame the scene the way they frame a
 *     Persian garden — dignity and stillness.
 *
 * Everything is deterministic (seeded stars) and generated as a plain tree so
 * the React layer and the QA rasterizer share one source of truth.
 */

const TAU = Math.PI * 2;
const round = (n) => Number(n.toFixed(2));

function mulberry32(seed) {
  return function animate() {
    let t = (seed += 0x6d2b79f5);
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

const hex = (c) => {
  const n = c.replace('#', '');
  return [parseInt(n.slice(0, 2), 16), parseInt(n.slice(2, 4), 16), parseInt(n.slice(4, 6), 16)];
};
const rgbToHex = ([r, g, b]) => `#${[r, g, b].map((v) => Math.round(Math.max(0, Math.min(255, v))).toString(16).padStart(2, '0')).join('')}`;

/** Piecewise-linear colour across keyframes: [[p, '#hex'], ...] */
export function keyLerp(stops, p) {
  const clamped = Math.max(0, Math.min(1, p));
  for (let i = 0; i < stops.length - 1; i += 1) {
    const [p0, c0] = stops[i];
    const [p1, c1] = stops[i + 1];
    if (clamped <= p1 || i === stops.length - 2) {
      const t = p1 === p0 ? 0 : Math.max(0, Math.min(1, (clamped - p0) / (p1 - p0)));
      const a = hex(c0);
      const b = hex(c1);
      return rgbToHex([a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t]);
    }
  }
  return stops[stops.length - 1][1];
}

const scalar = (stops, p) => {
  const clamped = Math.max(0, Math.min(1, p));
  for (let i = 0; i < stops.length - 1; i += 1) {
    const [p0, v0] = stops[i];
    const [p1, v1] = stops[i + 1];
    if (clamped <= p1 || i === stops.length - 2) {
      const t = p1 === p0 ? 0 : Math.max(0, Math.min(1, (clamped - p0) / (p1 - p0)));
      return v0 + (v1 - v0) * t;
    }
  }
  return stops[stops.length - 1][1];
};

/* ---- scene palette keyframes, p = 0 (night) → 1 (sunrise) ---- */
const SKY_TOP = [[0, '#05070d'], [0.34, '#090c1d'], [0.67, '#131231'], [1, '#241d3a']];
const SKY_MID = [[0, '#090d19'], [0.34, '#121026'], [0.67, '#211d3b'], [1, '#3c2a45']];
const SKY_HORIZON = [[0, '#12172a'], [0.34, '#261f31'], [0.67, '#4e322e'], [1, '#8a4e33']];
const FAR_RIDGE = [[0, '#131725'], [0.34, '#171427'], [0.67, '#1f1a2d'], [1, '#2c2130']];
const NEAR_RIDGE = [[0, '#090b13'], [0.34, '#0d0e1a'], [0.67, '#131122'], [1, '#1c1623']];
const WATER_TOP = [[0, '#0c1022'], [0.34, '#1b1626'], [0.67, '#31202c'], [1, '#59322c']];
const WATER_BOTTOM = [[0, '#04060c'], [0.34, '#070610'], [0.67, '#0c0913'], [1, '#130e09']];
const CYPRESS = [[0, '#04060c'], [0.34, '#06070f'], [0.67, '#0a0913'], [1, '#100c11']];

export const HORIZON_Y = 424;
export const SCENE_VIEWBOX = '0 0 340 680';

const sparkle = (cx, cy, r, opacity, fill) => {
  const w = r * 0.30;
  const d = [
    `M ${round(cx)} ${round(cy - r)}`,
    `Q ${round(cx + w)} ${round(cy - w)} ${round(cx + r)} ${round(cy)}`,
    `Q ${round(cx + w)} ${round(cy + w)} ${round(cx)} ${round(cy + r)}`,
    `Q ${round(cx - w)} ${round(cy + w)} ${round(cx - r)} ${round(cy)}`,
    `Q ${round(cx - w)} ${round(cy - w)} ${round(cx)} ${round(cy - r)}`,
    'Z',
  ].join(' ');
  return { tag: 'path', attrs: { d, fill, opacity: round(opacity) }, children: [] };
};

function starField(p) {
  const rand = mulberry32(41);
  const starAlpha = scalar([[0, 1], [0.34, 0.66], [0.67, 0.32], [1, 0.09]], p);
  const stars = [];
  for (let i = 0; i < 46; i += 1) {
    const x = round(10 + rand() * 320);
    const y = round(12 + rand() * 268);
    const r = round(0.45 + rand() * 0.95);
    const o = round(starAlpha * (0.42 + rand() * 0.58));
    if (o < 0.02) continue;
    // One quiet firouzeh (turquoise) star — the wordless gems Theia endowed with light.
    const firouzeh = i === 17;
    stars.push({
      tag: 'circle',
      attrs: { cx: x, cy: y, r, fill: firouzeh ? '#8fd6cb' : '#e8e9f2', opacity: o },
      children: [],
    });
  }
  // Brightest fixed stars: small four-point khatam sparkles.
  const sparkleSeeds = [[48, 46, 2.7], [126, 92, 2.2], [240, 38, 3.0], [298, 138, 2.0], [180, 168, 1.7]];
  for (const [x, y, r] of sparkleSeeds) {
    const o = Math.min(1, starAlpha * 1.15) * 0.92;
    if (o >= 0.02) stars.push(sparkle(x, y, r, o, '#f4f0e6'));
  }
  // The morning star lingers longest, low over the ridge.
  const morning = Math.max(0, Math.min(1, (0.62 - p) / 0.62)) * 0.95;
  if (morning > 0.03) stars.push(sparkle(260, 330, 3.2, morning, '#f6cf98'));
  return stars;
}

const crescent = (cx, cy, r, opacity) => ({
  tag: 'g',
  attrs: { opacity: round(opacity) },
  children: [
    {
      tag: 'mask',
      attrs: { id: 'theia-moon-cut' },
      children: [
        { tag: 'rect', attrs: { x: cx - r - 4, y: cy - r - 4, width: 2 * r + 8, height: 2 * r + 8, fill: '#fff' }, children: [] },
        { tag: 'circle', attrs: { cx: cx + r * 0.52, cy: cy - r * 0.22, r: r * 0.86, fill: '#000' }, children: [] },
      ],
    },
    { tag: 'circle', attrs: { cx, cy, r, fill: '#dfe3ff', mask: 'url(#theia-moon-cut)' }, children: [] },
    { tag: 'circle', attrs: { cx: cx - r * 0.18, cy: cy + r * 0.2, r: r * 0.16, fill: '#c9cdf2', mask: 'url(#theia-moon-cut)', opacity: 0.7 }, children: [] },
  ],
});

const cypress = (cx, baseY, height, width, fill) => {
  const tip = baseY - height;
  const d = [
    `M ${round(cx)} ${round(tip)}`,
    `C ${round(cx + width * 0.30)} ${round(tip + height * 0.22)} ${round(cx + width * 0.42)} ${round(tip + height * 0.52)} ${round(cx + width * 0.30)} ${round(baseY - height * 0.16)}`,
    `C ${round(cx + width * 0.22)} ${round(baseY - height * 0.04)} ${round(cx + width * 0.1)} ${round(baseY)} ${round(cx)} ${round(baseY)}`,
    `C ${round(cx - width * 0.1)} ${round(baseY)} ${round(cx - width * 0.22)} ${round(baseY - height * 0.04)} ${round(cx - width * 0.30)} ${round(baseY - height * 0.16)}`,
    `C ${round(cx - width * 0.42)} ${round(tip + height * 0.52)} ${round(cx - width * 0.30)} ${round(tip + height * 0.22)} ${round(cx)} ${round(tip)}`,
    'Z',
  ].join(' ');
  return { tag: 'path', attrs: { d, fill }, children: [] };
};

export function buildDawnScene(progress = 0) {
  const p = Math.max(0, Math.min(1, progress));
  const horizon = HORIZON_Y;

  const skyTop = keyLerp(SKY_TOP, p);
  const skyMid = keyLerp(SKY_MID, p);
  const skyHorizon = keyLerp(SKY_HORIZON, p);
  const farRidge = keyLerp(FAR_RIDGE, p);
  const nearRidge = keyLerp(NEAR_RIDGE, p);
  const waterTop = keyLerp(WATER_TOP, p);
  const waterBottom = keyLerp(WATER_BOTTOM, p);
  const cypressFill = keyLerp(CYPRESS, p);

  // The sun's climb: hidden below the ridge at p=0, fully risen at p=1.
  const sunY = horizon + 46 - scalar([[0, 0], [0.34, 34], [0.67, 96], [1, 152]], p);
  const sunX = 170;
  const sunR = 26;
  const raysAlpha = scalar([[0, 0], [0.3, 0.28], [0.62, 0.72], [1, 1]], p);
  const glowAlpha = scalar([[0, 0.1], [0.3, 0.4], [0.67, 0.78], [1, 1]], p);
  const hairlineAlpha = scalar([[0, 0.04], [0.3, 0.3], [0.67, 0.62], [1, 0.92]], p);
  const waterGlowAlpha = scalar([[0, 0.05], [0.3, 0.3], [0.67, 0.68], [1, 1]], p);
  const moonAlpha = scalar([[0, 0.92], [0.34, 0.74], [0.67, 0.48], [1, 0.26]], p);
  const reflectionY = horizon + (horizon - sunY) * 0.38 + 8;

  const rays = [];
  for (let i = 0; i < 16; i += 1) {
    const long = i % 2 === 0;
    const angle = -Math.PI / 2 + (TAU * i) / 16 + Math.PI / 16;
    const r1 = sunR + 6;
    const r2 = long ? sunR + 21 : sunR + 13;
    rays.push({
      tag: 'line',
      attrs: {
        x1: round(sunX + r1 * Math.cos(angle)), y1: round(sunY + r1 * Math.sin(angle)),
        x2: round(sunX + r2 * Math.cos(angle)), y2: round(sunY + r2 * Math.sin(angle)),
        stroke: '#ffd58f', 'stroke-width': long ? 1.9 : 1.35, 'stroke-linecap': 'round',
        opacity: round(raysAlpha * (long ? 0.95 : 0.7)),
      },
      children: [],
    });
  }

  const farRidgePath = 'M 0 398 L 33 368 L 62 390 L 94 354 L 124 394 L 157 372 L 197 396 L 229 358 L 267 392 L 300 382 L 340 402 L 340 446 L 0 446 Z';
  const nearRidgePath = 'M 0 422 L 47 400 L 87 420 L 125 392 L 172 422 L 214 402 L 259 423 L 305 404 L 340 423 L 340 448 L 0 448 Z';

  const children = [
    { tag: 'defs', attrs: {}, children: [
      { tag: 'linearGradient', attrs: { id: 'dusk-sky', x1: 0, y1: 0, x2: 0, y2: 1 }, children: [
        { tag: 'stop', attrs: { offset: '0', 'stop-color': skyTop }, children: [] },
        { tag: 'stop', attrs: { offset: '0.62', 'stop-color': skyMid }, children: [] },
        { tag: 'stop', attrs: { offset: '1', 'stop-color': skyHorizon }, children: [] },
      ] },
      { tag: 'linearGradient', attrs: { id: 'dusk-water', x1: 0, y1: 0, x2: 0, y2: 1 }, children: [
        { tag: 'stop', attrs: { offset: '0', 'stop-color': waterTop }, children: [] },
        { tag: 'stop', attrs: { offset: '1', 'stop-color': waterBottom }, children: [] },
      ] },
      { tag: 'radialGradient', attrs: { id: 'theia-sun', cx: 0.5, cy: 0.45, r: 0.62 }, children: [
        { tag: 'stop', attrs: { offset: '0', 'stop-color': '#ffefb8' }, children: [] },
        { tag: 'stop', attrs: { offset: '0.55', 'stop-color': '#f6c168' }, children: [] },
        { tag: 'stop', attrs: { offset: '1', 'stop-color': '#e08a3c' }, children: [] },
      ] },
      { tag: 'radialGradient', attrs: { id: 'theia-glow', cx: 0.5, cy: 0.5, r: 0.5 }, children: [
        { tag: 'stop', attrs: { offset: '0', 'stop-color': '#f8cd8f', 'stop-opacity': 0.85 }, children: [] },
        { tag: 'stop', attrs: { offset: '0.45', 'stop-color': '#f2b26a', 'stop-opacity': 0.28 }, children: [] },
        { tag: 'stop', attrs: { offset: '1', 'stop-color': '#f2b26a', 'stop-opacity': 0 }, children: [] },
      ] },
      { tag: 'radialGradient', attrs: { id: 'theia-reflection', cx: 0.5, cy: 0.5, r: 0.5 }, children: [
        { tag: 'stop', attrs: { offset: '0', 'stop-color': '#ffd98f', 'stop-opacity': 0.9 }, children: [] },
        { tag: 'stop', attrs: { offset: '1', 'stop-color': '#ffd98f', 'stop-opacity': 0 }, children: [] },
      ] },
      { tag: 'linearGradient', attrs: { id: 'theia-streak', x1: 0, y1: 0, x2: 0, y2: 1 }, children: [
        { tag: 'stop', attrs: { offset: '0', 'stop-color': '#ffdd9c', 'stop-opacity': 0.75 }, children: [] },
        { tag: 'stop', attrs: { offset: '1', 'stop-color': '#ffdd9c', 'stop-opacity': 0 }, children: [] },
      ] },
      { tag: 'radialGradient', attrs: { id: 'dusk-vignette', cx: 0.5, cy: 0.42, r: 0.85 }, children: [
        { tag: 'stop', attrs: { offset: '0.55', 'stop-color': '#000', 'stop-opacity': 0 }, children: [] },
        { tag: 'stop', attrs: { offset: '1', 'stop-color': '#000', 'stop-opacity': 0.5 }, children: [] },
      ] },
    ] },

    /* sky */
    { tag: 'rect', attrs: { x: 0, y: 0, width: 340, height: horizon + 2, fill: 'url(#dusk-sky)' }, children: [] },
    ...starField(p),
    crescent(80, 92, 14, moonAlpha),

    /* the shamseh sun and her rays */
    { tag: 'circle', attrs: { cx: sunX, cy: round(sunY), r: 92, fill: 'url(#theia-glow)', opacity: round(glowAlpha) }, children: [] },
    { tag: 'g', attrs: { class: 'dawn-rays' }, children: rays },
    { tag: 'circle', attrs: { cx: sunX, cy: round(sunY), r: sunR, fill: 'url(#theia-sun)', opacity: round(scalar([[0, 0.92], [1, 1]], p)) }, children: [] },

    /* ridges — she rises from behind the mountains of the Alborz */
    { tag: 'path', attrs: { d: farRidgePath, fill: farRidge }, children: [] },
    { tag: 'path', attrs: { d: nearRidgePath, fill: nearRidge }, children: [] },

    /* the first ray along the horizon */
    { tag: 'rect', attrs: { x: 0, y: horizon - 0.75, width: 340, height: 1.5, fill: '#f7cd8d', opacity: round(hairlineAlpha) }, children: [] },

    /* the howz — mind as still water, holding the light */
    { tag: 'rect', attrs: { x: 0, y: horizon, width: 340, height: 680 - horizon, fill: 'url(#dusk-water)' }, children: [] },
    { tag: 'ellipse', attrs: { cx: sunX, cy: round(reflectionY), rx: 46, ry: 9.5, fill: 'url(#theia-reflection)', opacity: round(waterGlowAlpha) }, children: [] },
    { tag: 'path', attrs: {
      d: `M ${sunX - 14} ${horizon + 2} L ${sunX + 14} ${horizon + 2} L ${sunX + 26} 664 L ${sunX - 26} 664 Z`,
      fill: 'url(#theia-streak)', opacity: round(Math.pow(waterGlowAlpha, 1.55) * 0.85),
    }, children: [] },
    { tag: 'ellipse', attrs: { cx: sunX, cy: 452, rx: 58, ry: 2.8, fill: 'none', stroke: '#ffd58f', 'stroke-width': 1.15, opacity: round(Math.pow(waterGlowAlpha, 1.5) * 0.55) }, children: [] },
    { tag: 'ellipse', attrs: { cx: sunX, cy: 486, rx: 96, ry: 3.7, fill: 'none', stroke: '#ffd58f', 'stroke-width': 1.0, opacity: round(Math.pow(waterGlowAlpha, 1.5) * 0.32) }, children: [] },
    { tag: 'ellipse', attrs: { cx: sunX, cy: 524, rx: 138, ry: 4.6, fill: 'none', stroke: '#ffd58f', 'stroke-width': 0.9, opacity: round(Math.pow(waterGlowAlpha, 1.5) * 0.18) }, children: [] },

    /* two khatam glints on the water at full rise */
    ...(p > 0.55 ? [
      sparkle(134, 466, 2.5, (p - 0.55) * 1.6 * 0.8, '#ffe3a1'),
      sparkle(209, 450, 1.9, (p - 0.55) * 1.6 * 0.6, '#8fd6cb'),
    ] : []),

    /* cypresses — the garden's quiet sentinels */
    cypress(22, 676, 148, 36, cypressFill),
    cypress(318, 676, 176, 42, cypressFill),

    /* embracing darkness at the corners */
    { tag: 'rect', attrs: { x: 0, y: 0, width: 340, height: 680, fill: 'url(#dusk-vignette)' }, children: [] },
  ];

  return { tag: 'svg', attrs: { viewBox: SCENE_VIEWBOX, preserveAspectRatio: 'xMidYMid slice' }, children: [{ tag: 'g', attrs: {}, children }] };
}

/** Serialize any element tree (shared with the emblem serializer format). */
export function treeToSVGString(tree, { size } = {}) {
  const esc = (value) => String(value)
    .replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;');
  const render = (node) => {
    const open = (tag, attrs) => {
      const attrText = Object.entries(attrs).map(([k, v]) => `${k}="${esc(v)}"`).join(' ');
      return attrText ? `<${tag} ${attrText}` : `<${tag}`;
    };
    const inner = (node.children || []).map(render).join('');
    if (!inner) return `${open(node.tag, node.attrs)}/>`;
    return `${open(node.tag, node.attrs)}>${inner}</${node.tag}>`;
  };
  const sizeAttrs = size ? { width: Math.round((size * 340) / 680), height: size } : {};
  const merged = { ...tree, attrs: { ...tree.attrs, xmlns: 'http://www.w3.org/2000/svg', ...sizeAttrs } };
  return render(merged);
}
