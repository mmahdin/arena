import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';

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
  // Retired artwork stays in the repository as history, never as the live mark.
  for (const retired of ['public/brand/photo_2026-11-12_19-20-10.jpg', 'public/brand/theia-welcome.png', 'public/brand/theia-story-1.png']) {
    assert.ok(existsSync(new URL(retired, root)), `Retired asset should be preserved: ${retired}`);
  }
});

test('the vector Moon carries the onboarding identity without reviving retired portraits', () => {
  const brand = read('src/components/TheiaBrand.jsx');
  const sigil = read('src/components/TheiaSigil.jsx');
  assert.match(brand, /THEIA_LOGO_SRC = '\/brand\/theia-sigil\.svg'/);
  assert.match(sigil, /openRing/);
  assert.match(sigil, /State|STATE_LABEL/);
  const active = [
    read('src/components/UI.jsx'),
    read('src/screens/Onboarding.jsx'),
    read('src/screens/Main.jsx'),
    read('src/screens/Letters.jsx'),
    read('src/features/devsession/DevLogin.jsx'),
  ].join('\n');
  for (const retired of ['photo_2026-11-12_19-20-10.jpg', 'theia-welcome.png', 'theia-story-1.png', 'theia-story-2.png', 'theia-story-3.png', 'theia-story-4.png', 'theia-privacy.png', 'theia-ready.png', 'theia-checkins.png']) {
    assert.ok(!active.includes(retired), `Retired female figure still referenced: ${retired}`);
  }
  // The onboarding subject is the Moon alone; the old astrolabe/ring artwork is not used.
  const onboarding = read('src/screens/Onboarding.jsx');
  const moon = read('src/components/EarthshineMoon.jsx');
  assert.match(onboarding, /import \{ EarthshineMoon \}/);
  assert.doesNotMatch(onboarding, /TheiaInstrument|TheiaSigil/);
  for (const scene of ['welcome', 'privacy', 'night', 'seeing', 'dawn', 'lights', 'checkins', 'final']) {
    assert.ok(moon.includes(`${scene}: {`), `Missing Earthshine scene ${scene}`);
  }
  assert.match(moon, /maskShift/);
  assert.match(moon, /horizon: true/);
  assert.match(onboarding, /این تصویر نمادین است؛ تغییر آن بر ذخیره‌سازی یا پردازش اطلاعاتت اثری ندارد/);
  assert.match(onboarding, /تیا همراهی مبتنی بر هوش مصنوعی است، نه جایگزینِ درمانگر حرفه‌ای/);
  assert.match(onboarding, /آن‌چه تاریک به نظر می‌رسد، خالی نیست/);
  assert.match(onboarding, /سه نور؛ و هیچ‌کدام همیشگی نیست/);
  // Only two excluded expert portraits remain in legacy content; story art is vector.
  assert.equal((onboarding.match(/<img/g) || []).length, 2, 'Earthshine story screens must stay vector-only');
});

test('HTML head, manifest and entry copy reflect the Persian RTL Theia (تیا) brand', () => {
  const html = read('index.html');
  assert.match(html, /lang="fa" dir="rtl"/);
  assert.match(html, /<title>تیا<\/title>/);
  assert.match(html, /href="\/favicon\.svg\?v=20261126"/);
  assert.match(html, /href="\/brand\/theia-sigil-192\.png"/);
  assert.match(html, /href="\/manifest\.webmanifest"/);
  assert.match(html, /name="theme-color" content="#0A0B0D"/);

  const manifest = JSON.parse(read('public/manifest.webmanifest'));
  assert.equal(manifest.short_name, 'تیا');
  assert.equal(manifest.dir, 'rtl');
  assert.equal(manifest.lang, 'fa');
  assert.ok(manifest.icons.some((icon) => icon.src.includes('theia-sigil-512.png') && icon.sizes === '512x512'));
  assert.equal(manifest.background_color, '#0A0B0D');

  const onboarding = read('src/screens/Onboarding.jsx');
  const main = read('src/screens/Main.jsx');
  const letters = read('src/screens/Letters.jsx');
  const devLogin = read('src/features/devsession/DevLogin.jsx');
  const data = read('src/data.js');

  assert.match(onboarding, /تیا را نمی‌بینی؛ با تیا می‌بینی/);
  assert.match(main, /نامه‌های تیا/);
  assert.match(letters, /نامه‌های تیا/);
  assert.match(devLogin, /به تیا خوش آمدی/);
  assert.match(data, /چطور با تیا آشنا شدی؟/);

  // Each authored Earthshine beat is connected to its existing onboarding route.
  for (const state of ['night', 'seeing', 'dawn', 'lights']) {
    assert.ok(onboarding.includes(`state: '${state}'`), `Onboarding missing ${state} scene`);
  }
  assert.match(onboarding, /state="welcome"/);
  assert.match(onboarding, /state="privacy"/);
  assert.match(onboarding, /state="checkins"/);
  assert.match(onboarding, /state="final"/);
});

test('active UI keeps the Theia visual vocabulary', () => {
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
