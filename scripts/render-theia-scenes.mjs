/**
 * Renderer for the Theia light scenes.
 *
 * The shipped SVG artwork is generated from `src/components/theia/scenes.js`,
 * so this script renders exactly the same geometry — no screenshots, no drift.
 * It is used for design review, for the generated PWA icons and for the
 * static PNG fallbacks in `public/brand/scenes/`.
 *
 * Usage:
 *   node scripts/render-theia-scenes.mjs               # review sheet -> artifacts/theia-scenes
 *   node scripts/render-theia-scenes.mjs --write-brand # also write public/brand/scenes/*.svg
 */
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { PALETTE, SCENES } from '../src/components/theia/scenes.js';

const here = fileURLToPath(new URL('.', import.meta.url));
const root = resolve(here, '..');
const writeBrand = process.argv.includes('--write-brand');
const outDir = resolve(root, process.argv.slice(2).find((arg) => !arg.startsWith('--')) || 'artifacts/theia-scenes');

const makeColor = (ids) => (value) => {
  if (!value) return 'none';
  if (PALETTE[value]) return PALETTE[value];
  return ids.has(value) ? `url(#${value})` : value;
};
const attrs = (pairs) => Object.entries(pairs)
  .filter(([, value]) => value !== undefined && value !== null && value !== '')
  .map(([key, value]) => `${key}="${typeof value === 'number' ? Math.round(value * 100) / 100 : value}"`)
  .join(' ');

function renderElement(el, color) {
  const dash = el.dash ? ` stroke-dasharray="${el.dash}"` : '';
  const cls = el.className ? ` class="${el.className}"` : '';
  switch (el.t) {
    case 'clip':
      return `<clipPath id="${el.id}">${el.children.map((child) => renderElement(child, color)).join('')}</clipPath>`;
    case 'circle':
      return `<circle${cls} ${attrs({ cx: el.cx, cy: el.cy, r: el.r, fill: color(el.fill), stroke: color(el.stroke), 'stroke-width': el.strokeWidth, opacity: el.opacity })}${dash} />`;
    case 'rect':
      return `<rect${cls} ${attrs({ x: el.x, y: el.y, width: el.width, height: el.height, fill: color(el.fill) })} />`;
    case 'line':
      return `<line${cls} ${attrs({ x1: el.x1, y1: el.y1, x2: el.x2, y2: el.y2, stroke: color(el.stroke), 'stroke-width': el.strokeWidth, opacity: el.opacity, 'stroke-linecap': 'round' })}${dash} />`;
    case 'group':
      return `<g${cls}${el.clip ? ` clip-path="url(#${el.clip})"` : ''}>${el.children.map((child) => renderElement(child, color)).join('')}</g>`;
    default:
      return `<path${cls} ${attrs({ d: el.d, fill: color(el.fill), stroke: color(el.stroke), 'stroke-width': el.strokeWidth, opacity: el.opacity, 'stroke-linecap': 'round', 'stroke-linejoin': 'round' })}${dash} />`;
  }
}

function renderDefs(defs = []) {
  const color = (value) => (value && PALETTE[value]) || value || 'none';
  return `<defs>${defs.map((def) => {
    const stops = def.stops
      .map(([offset, c, o]) => `<stop offset="${offset * 100}%" stop-color="${color(c)}" stop-opacity="${o}" />`)
      .join('');
    if (def.type === 'radial') return `<radialGradient id="${def.id}">${stops}</radialGradient>`;
    const coords = def.from
      ? ` gradientUnits="userSpaceOnUse" x1="${def.from[0]}" y1="${def.from[1]}" x2="${def.to[0]}" y2="${def.to[1]}"`
      : ' x1="0" y1="0" x2="0" y2="1"';
    return `<linearGradient id="${def.id}"${coords}>${stops}</linearGradient>`;
  }).join('')}</defs>`;
}

export function sceneToSvg(name, { background = PALETTE.ink, width, height } = {}) {
  const scene = SCENES[name]();
  const color = makeColor(new Set(scene.defs.map((def) => def.id)));
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="${scene.viewBox}" fill="none"${width ? ` width="${width}"` : ''}${height ? ` height="${height}"` : ''} role="img" aria-label="تیا — ${name}">`
    + renderDefs(scene.defs)
    + (background ? `<rect x="0" y="0" width="240" height="300" fill="${background}" />` : '')
    + scene.elements.map((element) => renderElement(element, color)).join('')
    + '</svg>';
}

const names = Object.keys(SCENES);

if (writeBrand) {
  const brandDir = join(root, 'public/brand/scenes');
  mkdirSync(brandDir, { recursive: true });
  for (const name of names) {
    writeFileSync(join(brandDir, `${name}.svg`), `${sceneToSvg(name)}\n`);
  }
  console.log(`Wrote ${names.length} scene SVGs to public/brand/scenes`);
}

mkdirSync(outDir, { recursive: true });
for (const name of names) {
  writeFileSync(join(outDir, `${name}.svg`), sceneToSvg(name));
}

const cell = 250;
const columns = 4;
const rows = Math.ceil(names.length / columns);
const sheet = [
  `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${columns * cell} ${rows * (cell + 30)}" width="${columns * cell}" height="${rows * (cell + 30)}">`,
  `<rect width="100%" height="100%" fill="${PALETTE.ink}" />`,
];
names.forEach((name, index) => {
  const x = (index % columns) * cell;
  const y = Math.floor(index / columns) * (cell + 30);
  const scene = SCENES[name]();
  sheet.push(`<svg x="${x + 6}" y="${y + 4}" width="${cell - 12}" height="${cell - 12}" viewBox="${scene.viewBox}">`);
  sheet.push(renderDefs(scene.defs));
  const color = makeColor(new Set(scene.defs.map((def) => def.id)));
  sheet.push(scene.elements.map((element) => renderElement(element, color)).join(''));
  sheet.push('</svg>');
  sheet.push(`<text x="${x + cell / 2}" y="${y + cell + 14}" text-anchor="middle" font-family="sans-serif" font-size="10" letter-spacing="2.5" fill="#8d8579">${name.toUpperCase()}</text>`);
});
sheet.push('</svg>');
writeFileSync(join(outDir, 'contact-sheet.svg'), sheet.join('\n'));
console.log(`Rendered ${names.length} scenes + contact sheet into ${outDir}`);
