import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

test('Theia logo, app icons, manifest and connected onboarding story assets exist', () => {
  const assets = [
    'public/favicon.svg',
    'public/apple-touch-icon.png',
    'public/manifest.webmanifest',
    'public/brand/theia-logo.svg',
    'public/brand/theia-icon-192.png',
    'public/brand/theia-icon-512.png',
    'public/brand/theia-welcome.svg',
    'public/brand/theia-privacy.svg',
    'public/brand/theia-story-1.svg',
    'public/brand/theia-story-2.svg',
    'public/brand/theia-story-3.svg',
    'public/brand/theia-story-4.svg',
    'public/brand/theia-checkins.svg',
    'public/brand/theia-ready.svg',
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

  // Ensure all four chapters of the night-to-dawn onboarding story are wired in sequence
  for (const step of ['theia-welcome.svg', 'theia-privacy.svg', 'theia-story-1.svg', 'theia-story-2.svg', 'theia-story-3.svg', 'theia-story-4.svg', 'theia-checkins.svg', 'theia-ready.svg']) {
    assert.ok(onboarding.includes(`/brand/${step}`), `Onboarding missing ${step}`);
  }
});

test('active UI replaces all flower/Ash icons and illustrations with Theia visuals', () => {
  const activeSource = [
    read('src/components/UI.jsx'),
    read('src/components/TheiaBrand.jsx'),
    read('src/screens/Onboarding.jsx'),
    read('src/screens/Main.jsx'),
    read('src/screens/Letters.jsx'),
    read('src/features/devsession/DevLogin.jsx'),
  ].join('\n');

  for (const legacy of ['Flower2', 'flower-person-', 'welcome-figures.png', 'ready-daisy.png', 'privacy-flower.png', 'checkins-handdrawn.svg']) {
    assert.ok(!activeSource.includes(legacy), `Legacy floral reference still active: ${legacy}`);
  }
  assert.ok(activeSource.includes('TheiaHomeArt'));
  assert.ok(activeSource.includes('TheiaMark'));
  assert.ok(activeSource.includes('TheiaSealIcon'));
  assert.ok(activeSource.includes('TheiaLetterEmpty'));
  assert.ok(!activeSource.includes('Goddess'));
  assert.ok(!read('src/screens/Onboarding.jsx').includes('تیا در'));
  assert.ok(read('src/components/TheiaBrand.jsx').includes('home-conversation-mark'));
  assert.ok(read('public/favicon.svg').includes('#e8b979'));
});
