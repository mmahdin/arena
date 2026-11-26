import sharp from 'sharp';
import { mkdirSync } from 'node:fs';
import { buildDawnScene, treeToSVGString } from '../src/art/dawnScene.js';
import { buildEmblem, emblemToSVGString } from '../src/art/emblem.js';

mkdirSync('/tmp/dawn-qa', { recursive: true });
const stages = { welcome: 0.06, 'simple-intro': 0.3, 'deeper-intro': 0.52, 'know-intro': 0.72, 'almost-intro': 0.88, ready: 1 };
for (const [name, p] of Object.entries(stages)) {
  const svg = treeToSVGString(buildDawnScene(p), { size: 680 });
  await sharp(Buffer.from(svg), { density: 96 }).png().toFile(`/tmp/dawn-qa/scene-${name}.png`);
}
for (const [label, opts] of [['luminous', { luminous: true }], ['mono-dark-bg', {}]]) {
  const svg = emblemToSVGString(buildEmblem(opts), { size: 512, background: '#12110d' });
  await sharp(Buffer.from(svg), { density: 96 }).png().toFile(`/tmp/dawn-qa/emblem-${label}.png`);
}
const tile = emblemToSVGString(buildEmblem({ luminous: true }), { size: 512, background: '#12110d', title: 'Theia' });
await sharp(Buffer.from(tile), { density: 96 }).png().toFile('/tmp/dawn-qa/emblem-tile.png');
console.log('rendered');
