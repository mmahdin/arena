import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';
import { AI_DISCLOSURE, NARRATIVE, SCENES } from '../src/features/onboarding/earthshine.js';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

test('faceless Theia identity assets, app icons and manifest exist', () => {
  const assets = [
    'public/favicon.svg',
    'public/brand/theia-sigil.svg',
    'public/brand/theia-sigil-192.png',
    'public/brand/theia-sigil-512.png',
    'public/apple-touch-icon.png',
    'public/manifest.webmanifest',
  ];
  for (const file of assets) {
    const url = new URL(file, root);
    assert.ok(existsSync(url), `Missing brand asset: ${file}`);
    assert.ok(statSync(url).size > 200, `Empty or invalid brand asset: ${file}`);
  }
  for (const retired of ['public/brand/photo_2026-11-12_19-20-10.jpg', 'public/brand/theia-welcome.png', 'public/brand/theia-story-1.png']) {
    assert.ok(existsSync(new URL(retired, root)), `Retired asset should be preserved: ${retired}`);
  }
});

test('the eight Earthshine plates and the natural held Moon are present', () => {
  assert.equal(SCENES.length, 8);
  const files = [
    'public/brand/earthshine/held-moon.jpg',
    ...SCENES.map((scene) => `public${scene.image}`),
  ];
  for (const file of files) {
    const url = new URL(file, root);
    assert.ok(existsSync(url), `Missing Earthshine plate: ${file}`);
    assert.ok(statSync(url).size > 200, `Empty or invalid Earthshine plate: ${file}`);
  }
  for (const scene of SCENES) {
    assert.ok(scene.alt && scene.alt.length > 20, `Missing Persian description: ${scene.id}`);
    assert.ok(scene.legibility >= 0 && scene.legibility <= 1, `Invalid legibility: ${scene.id}`);
  }
  assert.equal(SCENES[0].legibility, 0.04);
  assert.equal(SCENES[2].legibility, 0.2);
  assert.equal(SCENES[3].legibility, 0.5);
  assert.equal(SCENES[4].legibility, 0.7);
  assert.equal(SCENES[5].legibility, 1);
});

test('onboarding shows the natural Moon, never Theia as an object or ring instrument', () => {
  const brand = read('src/components/TheiaBrand.jsx');
  const sigil = read('src/components/TheiaSigil.jsx');
  const onboarding = read('src/screens/Onboarding.jsx');
  const devLogin = read('src/features/devsession/DevLogin.jsx');
  assert.match(brand, /THEIA_LOGO_SRC = '\/brand\/theia-sigil\.svg'/);
  assert.match(sigil, /openRing/);
  assert.match(sigil, /State|STATE_LABEL/);

  assert.match(onboarding, /<EarthshineScene scene=\{scene\}/);
  assert.match(devLogin, /<EarthshineScene scene=\{scene\}/);
  assert.doesNotMatch(onboarding, /TheiaInstrument|TheiaSigil|TheiaMark/);
  assert.doesNotMatch(devLogin, /TheiaInstrument|TheiaSigil|TheiaMark/);
  // Two legacy expert portraits and one lunar-surface detail for the mood step.
  assert.equal((onboarding.match(/<img/g) || []).length, 3);

  const active = [
    read('src/components/UI.jsx'),
    onboarding,
    read('src/screens/Main.jsx'),
    read('src/screens/Letters.jsx'),
    devLogin,
  ].join('\n');
  for (const retired of ['photo_2026-11-12_19-20-10.jpg', 'theia-welcome.png', 'theia-story-1.png', 'theia-story-2.png', 'theia-story-3.png', 'theia-story-4.png', 'theia-privacy.png', 'theia-ready.png', 'theia-checkins.png']) {
    assert.ok(!active.includes(retired), `Retired female figure still referenced: ${retired}`);
  }
});

test('HTML head, manifest and entry copy reflect the Persian RTL Theia (تیا) brand', () => {
  const html = read('index.html');
  assert.match(html, /lang="fa" dir="rtl"/);
  assert.match(html, /<title>تیا<\/title>/);
  assert.match(html, /href="\/favicon\.svg\?v=20261126"/);
  assert.match(html, /href="\/brand\/theia-sigil-192\.png"/);
  assert.match(html, /href="\/manifest\.webmanifest"/);
  assert.match(html, /name="theme-color" content="#0A0B0D"/);
  assert.match(html, /rel="preload" as="image" href="\/brand\/earthshine\/01-welcome\.jpg"/);

  const manifest = JSON.parse(read('public/manifest.webmanifest'));
  assert.equal(manifest.short_name, 'تیا');
  assert.equal(manifest.dir, 'rtl');
  assert.equal(manifest.lang, 'fa');
  assert.ok(manifest.icons.some((icon) => icon.src.includes('theia-sigil-512.png') && icon.sizes === '512x512'));
  assert.equal(manifest.background_color, '#0A0B0D');

  const main = read('src/screens/Main.jsx');
  const letters = read('src/screens/Letters.jsx');
  const devLogin = read('src/features/devsession/DevLogin.jsx');
  const data = read('src/data.js');

  assert.equal(NARRATIVE.welcome.title, 'تیا را نمی\u200cبینی؛ با تیا می\u200cبینی.');
  assert.match(NARRATIVE.welcome.body, /آن\u200cچه از پیش در توست/);
  assert.match(AI_DISCLOSURE, /هوش مصنوعی/);
  assert.match(AI_DISCLOSURE, /درمانگر نیست/);
  assert.match(NARRATIVE.privacy.body, /روی سرور ذخیره می\u200cشوند/);
  assert.match(NARRATIVE.privacy.body, /سرویس هوش مصنوعی فرستاده می\u200cشود/);
  assert.match(NARRATIVE.ready.body, /شخصی\u200cسازی گفت\u200cوگو/);
  assert.match(main, /نامه\u200cهای تیا/);
  assert.match(letters, /نامه\u200cهای تیا/);
  assert.match(devLogin, /به تیا خوش آمدی/);
  assert.match(data, /چطور با تیا آشنا شدی؟/);
});

test('active UI keeps the shared Theia visual vocabulary outside onboarding', () => {
  const activeSource = [
    read('src/components/UI.jsx'),
    read('src/components/TheiaBrand.jsx'),
    read('src/components/TheiaSigil.jsx'),
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
