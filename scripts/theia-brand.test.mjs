import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

test('Theia aperture logo, app icons, manifest and onboarding plates exist', () => {
  const assets = [
    'public/favicon.svg',
    'public/apple-touch-icon.png',
    'public/manifest.webmanifest',
    'public/brand/theia-logo.svg',
    'public/brand/theia-icon.svg',
    'public/brand/theia-icon-192.png',
    'public/brand/theia-icon-512.png',
  ];
  for (const file of assets) {
    const url = new URL(file, root);
    assert.ok(existsSync(url), `Missing brand asset: ${file}`);
    assert.ok(statSync(url).size > 200, `Empty or invalid brand asset: ${file}`);
  }

  // The old character-led illustrations are retired.
  for (const legacy of ['theia-welcome.png', 'theia-privacy.png', 'theia-story-1.png', 'theia-story-2.png', 'theia-story-3.png', 'theia-story-4.png', 'theia-checkins.png', 'theia-ready.png']) {
    assert.ok(!existsSync(new URL(`public/brand/${legacy}`, root)), `Legacy character art still present: ${legacy}`);
  }
});

test('logo, favicon and icon share the open-aperture, horizon and first-light geometry', () => {
  for (const file of ['public/brand/theia-icon.svg', 'public/favicon.svg', 'public/brand/theia-logo.svg']) {
    const svg = read(file);
    assert.match(svg, /#f6cf98|#f3efe6/, `${file} should carry the warm light palette`);
  }
  const icon = read('public/brand/theia-icon.svg');
  assert.match(icon, /A38 38 0 1 1/, 'open aperture arc');
  assert.match(icon, /theiaSun/, 'sun gradient');
  assert.match(icon, /M30 80H96/, 'horizon');
});

test('HTML head, manifest and UI screens reflect the Persian RTL Theia (تیا) brand', () => {
  const html = read('index.html');
  assert.match(html, /lang="fa" dir="rtl"/);
  assert.match(html, /<title>تیا<\/title>/);
  assert.match(html, /href="\/favicon\.svg"/);
  assert.match(html, /href="\/apple-touch-icon\.png"/);
  assert.match(html, /href="\/manifest\.webmanifest"/);

  const manifest = JSON.parse(read('public/manifest.webmanifest'));
  assert.equal(manifest.short_name, 'تیا');
  assert.equal(manifest.dir, 'rtl');
  assert.equal(manifest.lang, 'fa');

  const onboarding = read('src/screens/Onboarding.jsx');
  const main = read('src/screens/Main.jsx');
  const letters = read('src/screens/Letters.jsx');
  const devLogin = read('src/features/devsession/DevLogin.jsx');
  const data = read('src/data.js');

  assert.match(onboarding, /با تیا آشنا شو/);
  assert.match(main, /نامه‌های تیا/);
  assert.match(letters, /نامه‌های تیا/);
  assert.match(devLogin, /به تیا خوش اومدی!/);
  assert.match(data, /چطور با تیا آشنا شدی؟/);

  // The onboarding story is told by inline SVG plates, not character PNGs.
  for (const plate of ['TheiaWelcomeArt', 'TheiaPrivacyArt', 'TheiaPointArt', 'TheiaOpeningArt', 'TheiaOrbitArt', 'TheiaSkyArt', 'TheiaPulseArt', 'TheiaReadyArt']) {
    assert.ok(onboarding.includes(plate), `Onboarding missing ${plate}`);
  }
  assert.ok(!onboarding.includes('/brand/theia-'), 'Onboarding still references retired character images');
});

test('active UI replaces flower/Ash icons and the old goddess emblem with the aperture system', () => {
  const activeSource = [
    read('src/components/UI.jsx'),
    read('src/components/TheiaBrand.jsx'),
    read('src/screens/Onboarding.jsx'),
    read('src/screens/Main.jsx'),
    read('src/screens/Letters.jsx'),
    read('src/features/devsession/DevLogin.jsx'),
  ].join('\n');

  for (const legacy of ['Flower2', 'flower-person-', 'welcome-figures.png', 'ready-daisy.png', 'privacy-flower.png', 'checkins-handdrawn.svg', 'goddess', 'TheiaHomeEmblem']) {
    assert.ok(!activeSource.includes(legacy), `Legacy reference still active: ${legacy}`);
  }
  assert.ok(activeSource.includes('TheiaHomeArt'));
  assert.ok(activeSource.includes('TheiaMark'));
  assert.ok(activeSource.includes('TheiaSealIcon'));
  assert.ok(activeSource.includes('TheiaLetterEmpty'));
});

test('brand motion is patient and respects reduced motion', () => {
  const css = read('src/styles.css');
  assert.match(css, /@keyframes theia-slow-turn/);
  assert.match(css, /prefers-reduced-motion: reduce/);
  assert.match(css, /\.theia-orbit-lines[\s\S]{0,200}animation/);
  const reduced = css.slice(css.indexOf('prefers-reduced-motion: reduce', css.indexOf('theia-slow-turn')));
  assert.match(reduced, /\.theia-orbit-lines/);
});
