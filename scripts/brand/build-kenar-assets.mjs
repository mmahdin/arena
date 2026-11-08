// Rasterising is optional: the SVG is the source of truth and the committed PNGs
// are only fallbacks. Install @resvg/resvg-js to regenerate them.
let Resvg = null;
try { ({ Resvg } = await import('@resvg/resvg-js')); } catch { /* svg-only run */ }
import { writeFileSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { C, pebble, face, f } from './kenar-geometry.mjs';
import { scene, SCENES } from './kenar-scenes.mjs';

const ROOT = new URL('../../public/assets/', import.meta.url).pathname;
mkdirSync(join(ROOT, 'brand'), { recursive: true });
const P = C;
const night = { stone: '#39463d', stone2: '#2e3a32', edge: '#7f9486', ink: '#d9d2c1' };

const stone = (o, fill, edge, sw = 0) =>
  `<path d="${pebble(o)}" fill="${fill}" stroke="${edge}" stroke-width="${sw}" stroke-linejoin="round"/>`;
const shoreLine = (x1, x2, y, lift, color, sw = 6, op = 1) =>
  `<path d="M${f(x1)} ${f(y)} C${f(x1 + (x2 - x1) * .3)} ${f(y + lift)} ${f(x1 + (x2 - x1) * .72)} ${f(y + lift)} ${f(x2)} ${f(y - lift * .35)}" fill="none" stroke="${color}" stroke-width="${sw}" stroke-linecap="round" opacity="${op}"/>`;
const dot = (x, y, r, color, op) => `<circle cx="${f(x)}" cy="${f(y)}" r="${f(r)}" fill="${color}" opacity="${op}"/>`;
const doc = (w, h, body) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${w} ${h}" width="${w}" height="${h}">${body}</svg>`;

const emit = (rel, w, h, body, { png = w, bg = '#00000000' } = {}) => {
  const svg = doc(w, h, body);
  writeFileSync(join(ROOT, rel), svg);
  if (png && Resvg) writeFileSync(join(ROOT, rel.replace(/\.svg$/, '.png')), new Resvg(svg, { fitTo: { mode: 'width', value: png }, background: bg }).render().asPng());
};

// faced stone helper
const faced = ({ w, h, seed, cx, cy, flat = .9, tilt = 0, fs = 1.7, fill, edge, sw = 6, ink = P.ink, mouth = true }) =>
  `${stone({ w, h, seed, wobble: .045, flat, tilt, cx, cy }, fill, edge, sw)}${face({ x: cx, y: cy + h * .2, s: fs, color: ink, mouth })}`;

/* ===================== BRAND ===================== */
const MARK_BIG = { w: 31, h: 27.5, seed: 21, wobble: .05, flat: .93, tilt: -.04, cx: 52, cy: 63 };
const MARK_SMALL = { w: 15.5, h: 13, seed: 8, wobble: .05, flat: .9, tilt: .05, cx: 106, cy: 79 };
const mark = ({ ink = P.ink, s1 = P.sand, s2 = P.sandDeep, edge = P.edge, sw = 0 } = {}) => `<g>
  ${stone(MARK_SMALL, s2, edge, sw)}
  ${stone(MARK_BIG, s1, edge, sw)}
  <path d="M20 97 C46 105 96 105 132 94" fill="none" stroke="${ink}" stroke-width="5.4" stroke-linecap="round"/>
</g>`;

emit('brand/kenar-mark.svg', 160, 120, mark());
emit('brand/kenar-mark-mono.svg', 160, 120, mark({ s1: P.ink, s2: P.ink, edge: P.ink }));
emit('brand/kenar-mark-light.svg', 160, 120, mark({ ink: night.ink, s1: night.stone, s2: night.stone2, edge: night.edge }));
emit('brand/kenar-mark-paper.svg', 160, 120, `<rect width="160" height="120" fill="${P.paper}"/>${mark()}`, { bg: P.paper });

// icons
const iconInner = `
  ${stone({ ...MARK_SMALL, cx: 104, cy: 92 }, P.sandDeep, P.sandDeep)}
  ${stone({ ...MARK_BIG, cx: 54, cy: 76 }, P.sand, P.sand)}
  <path d="M26 110 C50 117 96 117 128 107" fill="none" stroke="${P.paper}" stroke-width="6.4" stroke-linecap="round"/>`;
emit('brand/kenar-icon.svg', 160, 160, `<rect width="160" height="160" rx="36" fill="${P.ink}"/>${iconInner}`, { png: 512, bg: '#00000000' });
emit('brand/kenar-icon-maskable.svg', 160, 160, `<rect width="160" height="160" fill="${P.ink}"/><g transform="translate(80,80) scale(.7) translate(-80,-80)">${iconInner}</g>`, { png: 512 });

/* ===================== ONBOARDING ART ===================== */
/* Each scene is authored at the true aspect ratio of its CSS slot (ported from
   src/styles.css), so `object-fit: contain` fills the frame instead of leaving a
   wide banner floating in a tall box. Compositor: ./kenar-scenes.mjs */
const sceneFile = {
  welcome: 'kenar/welcome-figures.svg',
  small: 'kenar/companion-small.svg',
  medium: 'kenar/companion-medium.svg',
  large: 'kenar/companion-large.svg',
  tall: 'kenar/companion-tall.svg',
  ready: 'kenar/ready-companions.svg',
  privacy: 'kenar/privacy-ring.svg',
  checkins: 'kenar/checkins-rhythm.svg',
};
for (const [key, rel] of Object.entries(sceneFile)) {
  const { W, H, night } = SCENES[key];
  emit(rel, W, H, scene(key), { png: Math.round(W / 2), bg: night ? '#101510' : P.paper });
}

console.log(`built all assets${Resvg ? '' : ' (svg only — @resvg/resvg-js not installed)'}`);
