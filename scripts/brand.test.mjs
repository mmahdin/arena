import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { pebblePath, pebbleSamples, groundLine } from '../src/brand/geometry.js';
import { SCENES, PALETTES, companion, cairn, vessel, stone } from '../src/brand/scenes.js';

const root = new URL('../', import.meta.url);
const read = (file) => readFileSync(new URL(file, root), 'utf8');

test('the pebble quarry is deterministic across calls', () => {
  const spec = { x: 120, base: 200, w: 44, h: 62, lean: 18, seed: 1.3 };
  assert.equal(pebblePath(spec), pebblePath(spec));
  const a = pebbleSamples(spec);
  const b = pebbleSamples({ ...spec });
  assert.deepEqual(a, b);
  // no random source may leak into the geometry module (comments excepted)
  const code = read('src/brand/geometry.js').replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/.*$/gm, '');
  assert.doesNotMatch(code, /Math\.random/);
});

test('a stone always sits flat on its ground line', () => {
  const base = 240;
  const points = pebbleSamples({ x: 100, base, w: 40, h: 50, lean: 12, seed: 4.2 });
  const lowest = Math.max(...points.map(([, y]) => y));
  assert.ok(lowest <= base + 0.001, `stone dips below its base: ${lowest} > ${base}`);
  assert.ok(points.some(([, y]) => Math.abs(y - base) < 0.001), 'stone never touches its base');
});

test('every scene is well-formed layer data', () => {
  for (const [name, build] of Object.entries(SCENES)) {
    for (const paletteName of Object.keys(PALETTES)) {
      const scene = build(PALETTES[paletteName]);
      assert.match(scene.viewBox, /^0 0 \d+ \d+$/, `${name} viewBox`);
      assert.ok(scene.layers.length > 0, `${name} has layers`);
      for (const layer of scene.layers) {
        if (layer.t === 'circle') {
          assert.equal(typeof layer.cx, 'number');
          assert.equal(typeof layer.cy, 'number');
          assert.ok(layer.r > 0);
        } else {
          assert.equal(layer.t, 'path');
          assert.match(layer.d, /^M/, `${name} path data starts with M`);
          assert.ok(layer.fill === 'none' || /^#[0-9a-f]{6}$/i.test(layer.fill), `${name} fill`);
        }
      }
    }
  }
});

test('the ember is the only warm colour, and the rule of one holds', () => {
  const ember = PALETTES.light.ember;
  for (const name of ['welcome', 'name', 'ready', 'privacy']) {
    const scene = SCENES[name](PALETTES.light);
    const embers = scene.layers.filter((l) => l.t === 'circle' && l.fill === ember);
    assert.ok(embers.length >= 1, `${name} must carry one ember`);
    assert.ok(embers.length <= 2, `${name} should not scatter embers (${embers.length})`);
  }
  // no gradient / filter / opacity trickery in the mark
  const mark = SCENES.mark(PALETTES.light);
  for (const layer of mark.layers) {
    assert.ok(!('opacity' in layer) && !('gradient' in layer));
  }
  assert.equal(mark.layers.filter((l) => l.fill === PALETTES.light.ember).length, 1);
});

test('the mark is two stones and one ember, and nothing else', () => {
  const mark = SCENES.mark(PALETTES.light);
  assert.equal(mark.layers.length, 3);
  assert.deepEqual(mark.layers.map((l) => l.t), ['path', 'path', 'circle']);
});

test('characters never gain a face', () => {
  const poses = ['stand', 'walk', 'wave', 'offer', 'reach', 'hold', 'sit'];
  for (const pose of poses) {
    const layers = companion({ x: 80, base: 260, pose, palette: PALETTES.light });
    for (const layer of layers) {
      // a face would be a second small circle inside the body; only the ember may be round
      if (layer.t === 'circle') assert.ok(layer.r <= 10, `${pose} has a suspiciously large dot`);
    }
    assert.ok(layers.length >= 2, `${pose} renders something`);
  }
  const src = read('src/brand/scenes.js')
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/^\s*\/\/.*$/gm, '');
  assert.doesNotMatch(src, /\b(eye|eyes|face|mouth|smile)\b/i);
});

test('scenes that use stone helpers still rest on one ground line', () => {
  for (const [name, build] of Object.entries(SCENES)) {
    if (name === 'mark' || name === 'insight') continue;
    const scene = build(PALETTES.light);
    const grounds = scene.layers.filter((l) => l.fill === 'none' && l.stroke === PALETTES.light.ground);
    assert.ok(grounds.length <= 1, `${name} draws ${grounds.length} ground lines`);
  }
  assert.match(groundLine({ x1: 0, x2: 100, y: 50 }), /^M/);
});

test('the committed brand assets are exactly what the geometry generates', async () => {
  const { svgAssets } = await import('./brand/build-assets.mjs');
  const assets = svgAssets();
  for (const file of ['hamrah-mark.svg', 'hamrah-mark-reverse.svg', 'hamrah-mark-mono.svg', 'hamrah-mark-ember.svg', 'hamrah-icon.svg', 'hamrah-icon-maskable.svg', 'hamrah-share-card.svg']) {
    assert.equal(read(`public/brand/${file}`), assets[file], `${file} has drifted from src/brand — run node scripts/brand/build-assets.mjs`);
  }
  assert.equal(read('public/favicon.svg'), assets.favicon, 'favicon.svg has drifted from src/brand');
});

test('brand assets and the visual identity book ship with the app', () => {
  for (const file of [
    'public/brand/hamrah-mark.svg',
    'public/brand/hamrah-mark-reverse.svg',
    'public/brand/hamrah-mark-mono.svg',
    'public/brand/hamrah-mark-ember.svg',
    'public/brand/hamrah-icon.svg',
    'public/brand/hamrah-icon-512.png',
    'public/brand/hamrah-icon-192.png',
    'public/brand/hamrah-apple-touch.png',
    'public/favicon.svg',
    'public/manifest.webmanifest',
    'docs/brand/Hamrah-Brand-Book.html',
    'docs/brand/IDENTITY.md',
  ]) {
    assert.ok(existsSync(new URL(file, root)), `${file} is missing`);
  }
  const book = read('docs/brand/Hamrah-Brand-Book.html');
  assert.match(book, /هم‌سنگ/);
  assert.match(book, /<svg/);
  assert.ok(book.length > 20000, 'the book embeds its illustrations');
});

test('the app renders the identity and keeps the legacy artwork for comparison', () => {
  for (const file of ['src/screens/Onboarding.jsx', 'src/screens/Main.jsx']) {
    assert.match(read(file), /brand\/Scene/);
  }
  assert.match(read('src/screens/Onboarding.jsx'), /SCENES\.welcome|SCENES\[/);
  assert.match(read('src/features/devsession/DevLogin.jsx'), /Lockup|Mark/);
  assert.match(read('src/components/UI.jsx'), /brand\/Mark/);
  // the borrowed flora must not be referenced by the app any more
  for (const asset of ['welcome-figures', 'flower-person', 'ready-daisy', 'privacy-flower']) {
    assert.doesNotMatch(read('src/screens/Onboarding.jsx'), new RegExp(asset), `${asset} still referenced`);
  }
  // ...but every original file is preserved on disk
  assert.equal(readdirSync(new URL('public/assets/', root)).length, 9);
});

test('the brand is declared in the shell and documented for the next agent', () => {
  assert.match(read('index.html'), /favicon\.svg/);
  assert.match(read('index.html'), /manifest\.webmanifest/);
  assert.match(read('AGENTS.md'), /هم‌سنگ/);
  assert.match(read('FRONTEND_DESIGN_SPEC.md'), /هم‌سنگ/);
  assert.match(read('docs/brand/IDENTITY.md'), /companion stone/);
  // exactly one ember rule is documented where designers will look
  assert.match(read('docs/brand/IDENTITY.md'), /only one thing may be ember/i);
});
