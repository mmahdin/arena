/**
 * Builds the Theia brand assets from the same geometry the UI renders.
 *
 * The previous logo was a photographic female profile; Theia is now a light
 * instrument (threshold + dawn star), so every shipped icon is regenerated from
 * `src/components/theia/scenes.js` instead of being hand-drawn twice.
 *
 * Usage: node scripts/build-brand-assets.mjs
 */
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { PALETTE, archGeometry, dawnStarPath } from '../src/components/theia/scenes.js';

const root = resolve(fileURLToPath(new URL('.', import.meta.url)), '..');
const brand = join(root, 'public/brand');
const icons = join(brand, 'icons');
mkdirSync(icons, { recursive: true });

const C = PALETTE;

/** The compact mark: a threshold with a dawn star standing inside it. */
export function markSvg({ size = 512, padding = 0.14, background = true } = {}) {
  const s = 1 - padding * 2;
  // Icons are read at 48-512px, so they use the middle optical band: a
  // sturdier stroke than the hero hairline, without the extra weight the
  // 16px band needs.
  const arch = archGeometry({
    cx: 120,
    half: 48 * s,
    spring: 148,
    d: 30 * s,
    base: 196,
  });
  const stroke = Math.max(8, 9 / s);
  const star = dawnStarPath(120, 146, 21 * s, 21 * s * 0.3);
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 240 240" width="${size}" height="${size}" fill="none">`
    + (background ? '<rect width="240" height="240" fill="url(#bg)"/>' : '')
    + '<defs>'
    + '<radialGradient id="bg" cx="50%" cy="52%" r="60%">'
    + `<stop offset="0" stop-color="${C.lapis}"/><stop offset="0.6" stop-color="${C.night}"/><stop offset="1" stop-color="${C.ink}"/>`
    + '</radialGradient>'
    + '<linearGradient id="rim" gradientUnits="userSpaceOnUse" x1="0" y1="84" x2="0" y2="194">'
    + `<stop offset="0" stop-color="${C.goldLight}" stop-opacity="0.95"/><stop offset="1" stop-color="${C.gold}" stop-opacity="0.45"/>`
    + '</linearGradient>'
    + `<radialGradient id="halo"><stop offset="0" stop-color="${C.goldLight}" stop-opacity="0.42"/><stop offset="0.55" stop-color="${C.gold}" stop-opacity="0.12"/><stop offset="1" stop-color="${C.gold}" stop-opacity="0"/></radialGradient>`
    + '</defs>'
    + '<circle cx="120" cy="146" r="62" fill="url(#halo)"/>'
    + `<path d="${arch.path}" stroke="url(#rim)" stroke-width="${stroke.toFixed(2)}" stroke-linecap="round"/>`
    + `<path d="${star}" fill="${C.cream}"/>`
    + '</svg>';
}

/** Quiet variant without the plate, for dark surfaces and documentation. */
export function wordmarkSvg() {
  const arch = archGeometry({ cx: 120, half: 48, spring: 148, d: 30, base: 196 });
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 240 240" width="240" height="240" fill="none">`
    + `<defs><radialGradient id="h"><stop offset="0" stop-color="${C.goldLight}" stop-opacity="0.3"/><stop offset="1" stop-color="${C.gold}" stop-opacity="0"/></radialGradient></defs>`
    + '<circle cx="120" cy="146" r="66" fill="url(#h)"/>'
    + `<path d="${arch.path}" stroke="${C.gold}" stroke-width="9" stroke-linecap="round"/>`
    + `<path d="${dawnStarPath(120, 146, 21, 6.3)}" fill="${C.cream}"/>`
    + '</svg>';
}

writeFileSync(join(brand, 'theia-mark.svg'), `${markSvg({ size: 512 })}\n`);
writeFileSync(join(brand, 'theia-logo.svg'), `${wordmarkSvg()}\n`);
writeFileSync(join(root, 'public/favicon.svg'), `${markSvg({ size: 64 })}\n`);
writeFileSync(join(icons, 'icon.svg'), `${markSvg({ size: 512 })}\n`);

// PNG rasterisation is optional: the sandbox has sharp, but the repository
// should stay installable without it.
let sharp = null;
try {
  ({ default: sharp } = await import('sharp'));
} catch {
  console.warn('sharp is not installed — skipping PNG icon rasterisation');
}
if (sharp) {
  const png = (svg, size) => sharp(Buffer.from(svg), { density: 400 }).resize(size, size).png({ compressionLevel: 9 });
  await png(markSvg({ size: 512 }), 512).toFile(join(icons, 'theia-icon-512.png'));
  await png(markSvg({ size: 512 }), 192).toFile(join(icons, 'theia-icon-192.png'));
  await png(markSvg({ size: 512 }), 180).toFile(join(root, 'public/apple-touch-icon.png'));
  await png(markSvg({ size: 512 }), 32).toFile(join(icons, 'favicon-32.png'));
  await png(markSvg({ size: 512, padding: 0.22 }), 512).toFile(join(icons, 'theia-icon-maskable.png'));
}

console.log('Theia brand assets rebuilt from scene geometry');
