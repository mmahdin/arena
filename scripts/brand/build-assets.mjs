/*
 * Generates every static piece of the identity from src/brand/scenes.js.
 *
 *   node scripts/brand/build-assets.mjs            # write public/brand/** + favicon
 *
 * SVG output needs nothing but Node. PNG output needs @resvg/resvg-js, which is
 * deliberately not a dependency — the shipped PNGs are committed, and only a change
 * to the mark needs them re-rendered:
 *
 *   npm i --no-save @resvg/resvg-js && node scripts/brand/build-assets.mjs
 *
 * `brand.test.mjs` imports `svgAssets()` and compares it with the committed files,
 * so the app, the book and the icon can never quietly drift from the geometry.
 * Never hand-edit the files this script writes.
 */
import { SCENES, PALETTES } from '../../src/brand/scenes.js';
import fs from 'node:fs';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

const root = join(import.meta.dirname, '..', '..');
const OUT = join(root, 'public/brand');

const NIGHT = '#12110d';
const MARBLE = '#f3f2ee';

const layer = (x) => (x.t === 'circle'
  ? `<circle cx="${x.cx}" cy="${x.cy}" r="${x.r}" fill="${x.fill}"/>`
  : `<path d="${x.d}" fill="${x.fill}"${x.stroke ? ` stroke="${x.stroke}" stroke-width="${x.width}" stroke-linecap="${x.cap}"` : ''}${x.rule ? ` fill-rule="${x.rule}"` : ''}${x.dash ? ` stroke-dasharray="${x.dash}"` : ''}/>`);

const mark = (palette) => SCENES.mark(palette).layers.map(layer).join('');

/** The mark in its four legal treatments, plus the icon family. All pure SVG text. */
export function svgAssets() {
  const ink = PALETTES.light;
  const reverse = { ...PALETTES.light, stone: MARBLE, ember: '#c9705a' };
  const mono = { ...PALETTES.light, stone: '#23201b', ember: '#23201b' };
  const emberField = { ...PALETTES.light, stone: '#c1664c', ember: MARBLE };

  const bare = (palette) => {
    const [, , w, h] = SCENES.mark(palette).viewBox.split(' ').map(Number);
    return `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 240 240">${mark(palette)}</svg>`;
  };

  // The mark is nudged a hair above the centre of a square tile, which is how it looks
  // optically centred rather than measurably centred.
  const iconMark = `<g transform="translate(96,88) scale(1.333)">${mark(reverse)}</g>`;
  const icon = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><rect width="512" height="512" rx="124" fill="${NIGHT}"/>${iconMark}</svg>`;
  const appleTouch = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><rect width="512" height="512" fill="${NIGHT}"/>${iconMark}</svg>`;
  const maskable = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><rect width="512" height="512" fill="${NIGHT}"/><g transform="translate(128,112) scale(1.0667)">${mark(reverse)}</g></svg>`;
  const favicon = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><rect width="64" height="64" rx="15" fill="${NIGHT}"/><g transform="translate(12,11) scale(0.1667)">${mark(reverse)}</g></svg>`;

  // Share card: type is shaped by the browser/CMS that displays it, so it stays vector.
  const shareCard = `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">
<rect width="1200" height="630" fill="${MARBLE}"/>
<g transform="translate(80,150) scale(1.1)">${mark(ink)}</g>
<text x="1120" y="300" font-family="Noto Naskh Arabic, serif" font-size="150" fill="#23201b" text-anchor="end" direction="rtl">همراه</text>
<text x="1120" y="380" font-family="Estedad, sans-serif" font-size="42" fill="#6f6a61" text-anchor="end" direction="rtl">با هم، آرام‌تر</text>
</svg>`;

  return {
    'hamrah-mark.svg': bare(ink),
    'hamrah-mark-reverse.svg': bare(reverse),
    'hamrah-mark-mono.svg': bare(mono),
    'hamrah-mark-ember.svg': bare(emberField),
    'hamrah-icon.svg': icon,
    'hamrah-icon-maskable.svg': maskable,
    'hamrah-share-card.svg': shareCard,
    favicon,
    // rasters are described, not embedded, so the test can check them without a renderer
    rasters: [
      [icon, 512, 'hamrah-icon-512.png'],
      [icon, 192, 'hamrah-icon-192.png'],
      [icon, 32, 'hamrah-icon-32.png'],
      [bare(ink), 512, 'hamrah-mark-512.png'],
      [bare(reverse), 512, 'hamrah-mark-reverse-512.png'],
      [appleTouch, 180, 'hamrah-apple-touch.png'],
    ],
  };
}

async function cli() {
  const assets = svgAssets();
  for (const file of ['hamrah-mark.svg', 'hamrah-mark-reverse.svg', 'hamrah-mark-mono.svg', 'hamrah-mark-ember.svg', 'hamrah-icon.svg', 'hamrah-icon-maskable.svg', 'hamrah-share-card.svg']) {
    fs.writeFileSync(join(OUT, file), assets[file]);
  }
  fs.writeFileSync(join(root, 'public/favicon.svg'), assets.favicon);

  let Resvg = null;
  try {
    ({ Resvg } = await import('@resvg/resvg-js'));
  } catch {
    console.warn('! @resvg/resvg-js not installed — PNGs left alone (SVGs are up to date)');
  }
  if (Resvg) {
    for (const [svg, size, file] of assets.rasters) {
      const png = new Resvg(svg, { fitTo: { mode: 'width', value: size } }).render().asPng();
      fs.writeFileSync(join(OUT, file), png);
    }
  }
  console.log('brand assets written to public/brand');
}

if (import.meta.url === pathToFileURL(process.argv[1]).href) await cli();
