import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');
const scene = read('src/components/EarthshineScene.jsx');
const onboarding = read('src/screens/Onboarding.jsx');
const css = read('src/onboarding.css');

test('Earthshine: one locked camera, one canvas, one place', () => {
  // The whole journey is composed in a single phone-shaped frame, so nothing is
  // ever cropped and the camera never moves.
  assert.match(scene, /const VIEW = \{ w: 390, h: 844 \}/);
  assert.match(scene, /preserveAspectRatio="xMidYMid slice"/);
  // the roof's geometry is declared once, in named constants
  for (const key of ['MOON', 'HORIZON', 'RIDGE', 'FLOOR']) {
    assert.match(scene, new RegExp(`const ${key} = \\{?\\s*(x|\\d)`), `${key} must be a named constant`);
  }
  // and there is exactly one moon in the sky, shared by every world
  assert.equal((scene.match(/^function HeldMoon\(/gm) || []).length, 1);
});

test('Earthshine: the moon is a thin waning crescent holding an ashen disc', () => {
  // The crescent is a lune — a disc masked by an identical disc shifted up — so
  // the bright sliver always holds the unlit face in its arms.
  assert.match(scene, /function Crescent\(/);
  assert.match(scene, /const shift = Math\.max\(0\.8, r \* crescent\)/);
  assert.ok(scene.includes('The Held Moon'));
  // the seas are the moon's real named maria, and they are always in the
  // geometry: only their visibility changes as the eye adapts
  for (const sea of ['Serenity', 'Tranquility', 'Crises', 'Clouds', 'Cold']) {
    assert.ok(scene.includes(sea), `Missing named sea ${sea}`);
  }
  assert.match(onboarding, /دریای بحران‌ها|آرامش|آسودگی/);
});

test('Earthshine: legibility of the disc is the progress bar', () => {
  // The narrative device: the ashen disc becomes readable as the story advances.
  assert.match(scene, /export function legibilityFor/);
  assert.match(scene, /export const LEGIBILITY/);
  assert.match(scene, /export function MoonProgress/);
  // every world declares an `ash` value, and they must ascend
  const ashes = [...scene.matchAll(/moon: \{[^}]*ash: ([0-9.]+)/g)].map((m) => Number(m[1]));
  assert.ok(ashes.length >= 5, 'each world should set the disc legibility');
  assert.equal(ashes[0], Math.min(...ashes), 'the journey starts on the faintest disc');
  assert.ok(Math.max(...ashes) >= 0.9, 'the disc must become fully readable');
});

test('Earthshine: warm light is rationed to one lantern and the last ray', () => {
  // Only the check-ins world lights the lantern, and only the final frame lets
  // the sun's beam onto the rug.
  const worlds = scene.slice(scene.indexOf('const WORLDS = {'));
  const lantern = [...worlds.matchAll(/lantern: ([0-9.]+)/g)].map((m) => Number(m[1]));
  assert.ok(lantern.length >= 1 && lantern.length <= 2, 'at most two frames may show the lantern');
  assert.equal((worlds.match(/ray: [0-9.]+/g) || []).length, 1, 'only one frame carries the first ray');
  assert.equal((worlds.match(/door: true/g) || []).length, 1, 'only the final frame opens the stair door');
  // the crescent stays cool: no warm tone on the moon itself
  assert.doesNotMatch(scene, /crescentTone[^\]]*#f[0-9a-f]e?[0-9a-f]?b?[0-9a-f]?\]/i);
});

test('Earthshine: eight stills, each with a label for assistive tech', () => {
  const labels = scene.slice(scene.indexOf('const SCENE_LABELS'), scene.indexOf('const SCENE_META'));
  for (const sceneKey of ['welcome', 'privacy', 'night', 'seeing', 'dawn', 'three', 'checkins', 'final']) {
    assert.ok(labels.includes(`${sceneKey}:`), `Missing still label: ${sceneKey}`);
  }
  assert.match(scene, /aria-label=\{description\}/);
  assert.match(scene, /role="img"/);
});

test('Earthshine: the sheet honours reduced motion and keeps touch targets', () => {
  assert.match(css, /@media \(prefers-reduced-motion: reduce\)/);
  assert.match(css, /\.es-flame[\s\S]{0,200}animation: es-flicker/);
  // the primary action is a 56px target with a 16px radius, on both surfaces
  assert.match(css, /\.ob-footer \.primary-button[\s\S]{0,900}height: 56px/);
  assert.match(css, /height: 56px;[\s\S]{0,200}border-radius: 16px/);
  assert.match(css, /\.back-button[\s\S]{0,300}width: 44px/);
});
