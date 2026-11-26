import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

test('Theia logo, app icons, manifest and abstract onboarding artwork exist', () => {
  const assets = [
    'public/favicon.svg',
    'public/brand/photo_2026-11-12_19-20-10.jpg',
    'public/manifest.webmanifest',
    'public/brand/theia-logo.svg',
    'src/components/TheiaLightArt.jsx',
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
  assert.match(html, /href="\/favicon\.svg\?v=20261112"/);
  assert.match(html, /href="\/brand\/photo_2026-11-12_19-20-10\.jpg"/);
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

  assert.match(onboarding, /برای دیدنِ حالِ خودت، لازم نیست عجله کنی/);
  assert.match(onboarding, /اسطوره‌های یونان/);
  assert.match(main, /نامه‌های تیا/);
  assert.match(letters, /نامه‌های تیا/);
  assert.match(devLogin, /به تیا خوش اومدی!/);
  assert.match(data, /چطور با تیا آشنا شدی؟/);

  // The four narrative beats now share a person-free, in-product light artwork.
  const lightArt = read('src/components/TheiaLightArt.jsx');
  assert.match(onboarding, /TheiaLightArt scene="welcome"/);
  assert.match(onboarding, /scene: 'pause'/);
  assert.match(onboarding, /scene: 'reflect'/);
  assert.match(onboarding, /scene: 'dawn'/);
  assert.match(onboarding, /scene="privacy"/);
  assert.match(onboarding, /scene="ready"/);
  assert.match(devLogin, /TheiaLightArt scene="welcome"/);
  assert.match(lightArt, /className={`theia-light-art/);
  for (const legacy of ['theia-welcome.png', 'theia-privacy.png', 'theia-story-1.png', 'theia-story-2.png', 'theia-story-3.png', 'theia-story-4.png', 'theia-checkins.png', 'theia-ready.png']) {
    assert.ok(!onboarding.includes(legacy), `Old figurative onboarding art still active: ${legacy}`);
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
});
