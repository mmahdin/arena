import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

/** All Persian string literals in the onboarding screen, in stable order. */
const copyOf = (source) => [...source.matchAll(/(['"`])((?:[^\\`'"\n]|\\.)*?)\1/g)]
  .map((match) => match[2])
  .filter((text) => /[\u0600-\u06FF]/.test(text))
  .sort();

test('every onboarding word is unchanged: imagery only', () => {
  const baseline = read('docs/design/earthshine-onboarding-copy-baseline.txt').trim().split('\n');
  assert.deepEqual(copyOf(read('src/screens/Onboarding.jsx')), baseline,
    'Onboarding copy must be byte-identical to the pre-Earthshine baseline');
  // The sign-in entry copy is part of the same journey and is also untouched.
  assert.match(read('src/features/devsession/DevLogin.jsx'), /به تیا خوش آمدی/);
});

test('the Earthshine plates are present, small and referenced by the artwork', () => {
  const plates = [
    'public/brand/earthshine/earthshine-moon.jpg',
    'public/brand/earthshine/earthshine-reflection.jpg',
    'public/brand/earthshine/earthshine-moon-paper.jpg',
  ];
  for (const plate of plates) {
    const url = new URL(plate, root);
    assert.ok(existsSync(url), `Missing Earthshine plate: ${plate}`);
    assert.ok(statSync(url).size > 20_000, `Plate looks empty: ${plate}`);
    assert.ok(statSync(url).size < 300_000, `Plate is too heavy for onboarding: ${plate}`);
  }
  const art = read('src/components/EarthshineInstrument.jsx');
  for (const plate of plates) {
    assert.ok(art.includes(`/${plate.replace('public/', '')}`), `Unused plate: ${plate}`);
  }
});

test('one moon is re-read in eight states; only the reflected light changes', () => {
  const art = read('src/components/EarthshineInstrument.jsx');
  const states = ['welcome', 'night', 'veil', 'sight', 'dawn', 'orbits', 'pulse', 'seal'];
  for (const state of states) {
    assert.match(art, new RegExp(`^\\s*${state}:\\s*\\{`, 'm'), `Missing chapter state ${state}`);
    assert.match(art, new RegExp(`^\\s*${state}:\\s*'`, 'm'), `Missing description for ${state}`);
  }
  // The reflection plate is the one chapter that reveals where the light came
  // from; every other chapter uses the same moon plate.
  assert.match(art, /orbits:\s*\{\s*plate:\s*'reflection'/);
  assert.equal((art.match(/plate: 'moon'/g) || []).length, 7);
  // The instrument itself draws no geometry at all: no rings, no ticks, no
  // orbits, no rays. It only re-reads one plate with more or less light.
  const instrument = art.slice(art.indexOf('export function TheiaInstrument'), art.indexOf('export function EarthshineGlyph'));
  for (const geometry of ['<svg', '<circle', '<path', '<line', 'stroke=']) {
    assert.ok(!instrument.includes(geometry), `Instrument must not draw ${geometry}`);
  }
  // The only vector drawing is the hairline crescent used in the chrome.
  assert.match(art, /export function EarthshineGlyph/);
});

test('onboarding, chrome and sign-in all use the Earthshine artwork', () => {
  const onboarding = read('src/screens/Onboarding.jsx');
  const sigil = read('src/components/TheiaSigil.jsx');
  const login = read('src/features/devsession/DevLogin.jsx');
  const css = read('src/onboarding.css');

  assert.match(sigil, /export \{ EarthshineGlyph, TheiaInstrument, STATE_LABEL \}/);
  assert.match(onboarding, /<TheiaInstrument state=\{state\} tone=\{tone\}/);
  assert.match(onboarding, /<EarthshineGlyph size=\{19\} \/>/);
  assert.match(onboarding, /<EarthshineGlyph size=\{18\} \/>/);
  assert.match(login, /<TheiaInstrument state="welcome"/);

  for (const state of ['night', 'sight', 'dawn', 'orbits']) {
    assert.ok(onboarding.includes(`state: '${state}'`), `Chapter ${state} must remain wired`);
  }
  assert.match(css, /\.earthshine--orbits \.earthshine__plate/);
  assert.match(css, /@keyframes es-reveal/);
  assert.match(css, /@media \(prefers-reduced-motion: reduce\)/);
  // The old astrolabe motion is gone with the old geometry.
  assert.ok(!css.includes('ti-drift') && !css.includes('ti-rays'));
});
