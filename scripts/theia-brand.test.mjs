import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

test('Theia emblem, launch tiles, manifest and onboarding artwork exist', () => {
  const assets = [
    'public/favicon.svg',
    'public/apple-touch-icon.png',
    'public/brand/theia-emblem.svg',
    'public/brand/theia-logo.svg',
    'public/brand/theia-icon-192.png',
    'public/brand/theia-icon-512.png',
    'public/manifest.webmanifest',
  ];
  for (const file of assets) {
    const url = new URL(file, root);
    assert.ok(existsSync(url), `Missing brand asset: ${file}`);
    assert.ok(statSync(url).size > 200, `Empty or invalid brand asset: ${file}`);
  }
});

test('HTML head, manifest and UI screens reflect the Persian RTL Theia (تیا) brand', () => {
  const html = read('index.html');
  assert.match(html, /lang="fa" dir="rtl"/);
  assert.match(html, /<title>تیا<\/title>/);
  assert.match(html, /href="\/favicon\.svg\?v=20261126"/);
  assert.match(html, /href="\/apple-touch-icon\.png\?v=20261126"/);
  assert.match(html, /href="\/manifest\.webmanifest"/);
  assert.ok(!html.includes('photo_2026-11-12'), 'index.html still references the retired photo logo');

  const manifest = JSON.parse(read('public/manifest.webmanifest'));
  assert.equal(manifest.short_name, 'تیا');
  assert.equal(manifest.dir, 'rtl');
  assert.equal(manifest.lang, 'fa');
  const manifestIconSources = manifest.icons.map((icon) => icon.src).join(' ');
  assert.ok(!manifestIconSources.includes('photo_2026-11-12'), 'manifest still references the retired photo logo');
  assert.ok(manifestIconSources.includes('/brand/theia-icon-512.png'), 'manifest missing 512px install icon');

  const onboarding = read('src/screens/Onboarding.jsx');
  const main = read('src/screens/Main.jsx');
  const letters = read('src/screens/Letters.jsx');
  const devLogin = read('src/features/devsession/DevLogin.jsx');
  const data = read('src/data.js');

  assert.match(onboarding, /خودت رو با نور ببین/);
  assert.match(main, /نامه‌های تیا/);
  assert.match(letters, /نامه‌های تیا/);
  assert.match(devLogin, /به تیا خوش اومدی!/);
  assert.match(data, /چطور با تیا آشنا شدی؟/);

  // The five chapters of the night-to-dawn narrative must all render the shared scene
  assert.ok(onboarding.includes('DawnScene'), 'Onboarding does not render the shared DawnScene');
  for (const step of ['welcome', 'simple-intro', 'deeper-intro', 'know-intro', 'almost-intro']) {
    assert.ok(onboarding.includes(step), `Onboarding missing dawn chapter: ${step}`);
  }
  assert.doesNotMatch(onboarding, /theia-welcome\.png|theia-story-\d|theia-ready\.png|theia-privacy\.png|theia-checkins\.png/);
});

test('active UI uses the generated, non-human Theia identity', () => {
  const activeSource = [
    read('src/components/UI.jsx'),
    read('src/components/TheiaBrand.jsx'),
    read('src/components/DawnScene.jsx'),
    read('src/components/SvgTree.jsx'),
    read('src/art/emblem.js'),
    read('src/art/dawnScene.js'),
    read('src/screens/Onboarding.jsx'),
    read('src/screens/Main.jsx'),
    read('src/screens/Letters.jsx'),
    read('src/features/devsession/DevLogin.jsx'),
  ].join('\n');

  for (const legacy of [
    'Flower2', 'flower-person-', 'welcome-figures.png', 'ready-daisy.png', 'privacy-flower.png',
    'checkins-handdrawn.svg', 'theia-welcome.png', 'theia-ready.png', 'theia-story-',
    'theia-privacy.png', 'theia-checkins.png', 'photo_2026-11-12', 'THEIA_LOGO_SRC',
  ]) {
    assert.ok(!activeSource.includes(legacy), `Retired artistic identity still active: ${legacy}`);
  }
  for (const required of [
    'TheiaHomeArt', 'TheiaMark', 'TheiaSealIcon', 'TheiaLetterEmpty', 'TheiaSealedEye', 'TheiaCheckinGlow',
    'buildEmblem', 'buildDawnScene', 'DawnProgress',
  ]) {
    assert.ok(activeSource.includes(required), `New identity element missing: ${required}`);
  }
});
