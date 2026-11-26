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

test('the faceless identity: ring brand mark, Moon story art, no figure anywhere', () => {
  const brand = read('src/components/TheiaBrand.jsx');
  const sigil = read('src/components/TheiaSigil.jsx');
  const onboarding = read('src/screens/Onboarding.jsx');
  const sky = read('src/components/TheiaSky.jsx');
  assert.match(brand, /THEIA_LOGO_SRC = '\/brand\/theia-sigil\.svg'/);
  assert.match(sigil, /openRing/);
  // The 2026-11 Earthshine pass retires the ring/aperture instrument from the
  // story: the Moon is the only subject of the onboarding artwork now.
  assert.ok(!sigil.includes('TheiaInstrument'), 'the ring instrument is retired');
  assert.ok(!onboarding.includes('TheiaInstrument'), 'onboarding artwork must be the Moon, not the ring');
  assert.match(onboarding, /<TheiaSky scene=\{scene\}>/);
  assert.match(sky, /export function TheiaMoon/);
  const active = [
    read('src/components/UI.jsx'),
    onboarding,
    read('src/screens/Main.jsx'),
    read('src/screens/Letters.jsx'),
    read('src/features/devsession/DevLogin.jsx'),
  ].join('\n');
  for (const retired of ['photo_2026-11-12_19-20-10.jpg', 'theia-welcome.png', 'theia-story-1.png', 'theia-story-2.png', 'theia-story-3.png', 'theia-story-4.png', 'theia-privacy.png', 'theia-ready.png', 'theia-checkins.png']) {
    assert.ok(!active.includes(retired), `Retired female figure still referenced: ${retired}`);
  }
  // No illustration may depict a person: the story is told by the sky.
  assert.equal((onboarding.match(/<img/g) || []).length, 2, 'Theia story screens must stay vector-only');
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
  const story = read('src/features/onboarding/earthshine.js');
  const main = read('src/screens/Main.jsx');
  const letters = read('src/screens/Letters.jsx');
  const devLogin = read('src/features/devsession/DevLogin.jsx');
  const data = read('src/data.js');

  assert.match(story, /تیا را نمی‌بینی؛ با تیا می‌بینی/);
  assert.match(onboarding, /AI_TRANSPARENCY/);
  assert.match(main, /نامه‌های تیا/);
  assert.match(letters, /نامه‌های تیا/);
  assert.match(devLogin, /به تیا خوش آمدی/);
  assert.match(data, /چطور با تیا آشنا شدی؟/);

  // The eight scenes are wired in sequence through one sky.
  for (const name of ['آغاز', 'حریم خصوصی', 'شب', 'دیدن', 'سپیده', 'سه نور', 'یادآوری', 'آغاز مسیر']) {
    assert.ok(story.includes(`name: '${name}'`), `Missing scene ${name}`);
  }
  assert.match(onboarding, /SCENE_COUNT/);
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
