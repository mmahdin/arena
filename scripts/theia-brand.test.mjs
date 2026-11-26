import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

/**
 * Theia's identity is now non-human by design: a Persian threshold arch and a
 * dawn star, generated from geometry. These tests protect both halves of that
 * decision — the shipped assets exist, and no screen quietly falls back to
 * depicting her as a woman.
 */

test('the identity ships as generated geometry, not figure illustrations', () => {
  const assets = [
    'public/favicon.svg',
    'public/brand/theia-mark.svg',
    'public/brand/theia-logo.svg',
    'public/brand/icons/theia-icon-192.png',
    'public/brand/icons/theia-icon-512.png',
    'public/brand/icons/theia-icon-maskable.png',
    'public/apple-touch-icon.png',
  ];
  for (const file of assets) {
    const url = new URL(file, root);
    assert.ok(existsSync(url), `Missing brand asset: ${file}`);
    assert.ok(statSync(url).size > 200, `Empty or invalid brand asset: ${file}`);
  }

  // The retired female-profile logo and storybook figures must stay retired.
  for (const retired of [
    'public/brand/photo_2026-11-12_19-20-10.jpg',
    'public/brand/theia-welcome.png',
    'public/brand/theia-privacy.png',
    'public/brand/theia-ready.png',
    'public/brand/theia-story-1.png',
    'public/brand/theia-story-2.png',
    'public/brand/theia-story-3.png',
    'public/brand/theia-story-4.png',
    'public/brand/theia-checkins.png',
  ]) {
    assert.ok(!existsSync(new URL(retired, root)), `Retired figure asset is still shipped: ${retired}`);
  }
});

test('HTML head, manifest and screens carry the Persian RTL Theia (تیا) brand', () => {
  const html = read('index.html');
  assert.match(html, /lang="fa" dir="rtl"/);
  assert.match(html, /<title>تیا<\/title>/);
  assert.match(html, /href="\/favicon\.svg\?v=20261126"/);
  assert.match(html, /href="\/apple-touch-icon\.png\?v=20261126"/);
  assert.match(html, /href="\/manifest\.webmanifest"/);
  assert.doesNotMatch(html, /photo_2026-11-12/);

  const manifest = JSON.parse(read('public/manifest.webmanifest'));
  assert.equal(manifest.short_name, 'تیا');
  assert.equal(manifest.dir, 'rtl');
  assert.equal(manifest.lang, 'fa');
  assert.ok(manifest.icons.some((icon) => icon.purpose === 'maskable'));
  assert.ok(manifest.icons.every((icon) => !icon.src.includes('photo_2026')));

  const onboarding = read('src/screens/Onboarding.jsx');
  const main = read('src/screens/Main.jsx');
  const letters = read('src/screens/Letters.jsx');
  const devLogin = read('src/features/devsession/DevLogin.jsx');
  const data = read('src/data.js');

  assert.match(main, /نامه‌های تیا/);
  assert.match(letters, /نامه‌های تیا/);
  assert.match(devLogin, /به تیا خوش اومدی!/);
  assert.match(data, /چطور با تیا آشنا شدی؟/);

  // The narrative chapters are all rendered from the generated scene set.
  for (const scene of ['welcome', 'seal', 'night', 'dawn', 'sun', 'sight', 'return', 'ready']) {
    assert.ok(onboarding.includes(`"${scene}"`) || onboarding.includes(`'${scene}'`),
      `Onboarding never references the ${scene} scene`);
  }
  assert.match(onboarding, /TheiaScene/);
});

test('active UI never falls back to the retired flower/figure artwork', () => {
  const activeSource = [
    read('src/components/UI.jsx'),
    read('src/components/TheiaBrand.jsx'),
    read('src/screens/Onboarding.jsx'),
    read('src/screens/Main.jsx'),
    read('src/screens/Letters.jsx'),
    read('src/features/devsession/DevLogin.jsx'),
  ].join('\n');

  for (const legacy of [
    'Flower2', 'flower-person-', 'welcome-figures.png', 'ready-daisy.png',
    'privacy-flower.png', 'checkins-handdrawn.svg',
    'theia-welcome.png', 'theia-story-', 'theia-ready.png', 'theia-privacy.png', 'theia-checkins.png',
    'photo_2026-11-12',
  ]) {
    assert.ok(!activeSource.includes(legacy), `Legacy artwork still active: ${legacy}`);
  }
  assert.ok(activeSource.includes('TheiaHomeArt'));
  assert.ok(activeSource.includes('TheiaMark'));
  assert.ok(activeSource.includes('TheiaSealIcon'));
  assert.ok(activeSource.includes('TheiaLetterEmpty'));
  assert.ok(activeSource.includes('TheiaScene'));
});

test('the identity is documented as non-human and mythically grounded', () => {
  const spec = read('FRONTEND_DESIGN_SPEC.md');
  assert.match(spec, /Theia is not depicted as a person/);
  assert.match(spec, /Selene|سِلنه/);
  assert.match(spec, /mihrab|طاق|threshold/i);
});
