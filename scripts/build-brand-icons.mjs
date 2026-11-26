/**
 * Regenerates the launch icons from the single source of truth in src/art/emblem.js.
 * Usage: node scripts/build-brand-icons.mjs   (requires the optional peer "sharp")
 * The checked-in icons under public/ are the output; re-run after any emblem change.
 */
import { writeFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import sharp from 'sharp';
import { buildEmblemTile, buildEmblem, emblemToSVGString } from '../src/art/emblem.js';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const out = (path) => resolve(root, path);

const tile = buildEmblemTile({ cornerRadius: 0 }); // iOS/maskable supply their own mask
const faviconTile = buildEmblemTile({ cornerRadius: 13 });

const tileSVG = (tree) => emblemToSVGString(tree, {});

writeFileSync(out('public/brand/theia-logo.svg'), emblemToSVGString(buildEmblemTile({ cornerRadius: 0 }), {}));
writeFileSync(out('public/favicon.svg'), tileSVG(faviconTile));
writeFileSync(out('public/brand/theia-emblem.svg'), emblemToSVGString(buildEmblem({ luminous: true }), { title: 'Theia — نشانهٔ تیا' }));

for (const [file, size] of [['public/brand/theia-icon-192.png', 192], ['public/brand/theia-icon-512.png', 512], ['public/apple-touch-icon.png', 180]]) {
  await sharp(Buffer.from(tileSVG(tile)), { density: 300 }).resize(size, size).png().toFile(out(file));
  console.log('wrote', file);
}
console.log('wrote public/favicon.svg, public/brand/theia-logo.svg, public/brand/theia-emblem.svg');
