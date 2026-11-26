import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';
import { EARTHSHINE_SCENES } from '../src/features/onboarding/earthshine.js';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

test('faceless Theia Earthshine assets, app icons and manifest exist', () => {
  const brandAssets = [
    'public/favicon.svg',
    'public/brand/theia-sigil.svg',
    'public/brand/theia-sigil-192.png',
    'public/brand/theia-sigil-512.png',
    'public/apple-touch-icon.png',
    'public/manifest.webmanifest',
    'public/brand/earthshine/01-welcome.webp',
    'public/brand/earthshine/02-privacy.webp',
    'public/brand/earthshine/03-night.webp',
    'public/brand/earthshine/04-seeing.webp',
    'public/brand/earthshine/05-dawn.webp',
    'public/brand/earthshine/06-three-lights.webp',
    'public/brand/earthshine/07-checkins.webp',
    'public/brand/earthshine/07-checkins-dim.webp',
    'public/brand/earthshine/08-final.webp',
  ];
  for (const file of brandAssets) {
    const url = new URL(file, root);
    assert.ok(existsSync(url), `Missing brand or Earthshine asset: ${file}`);
    assert.ok(statSync(url).size > 200, `Empty or invalid asset: ${file}`);
  }
  // Retired artwork stays in the repository as history, never as the live mark.
  for (const retired of ['public/brand/photo_2026-11-12_19-20-10.jpg', 'public/brand/theia-welcome.png', 'public/brand/theia-story-1.png']) {
    assert.ok(existsSync(new URL(retired, root)), `Retired asset should be preserved: ${retired}`);
  }
});

test('the Held Moon (ماهِ در آغوش) and Earthshine narrative replace retired figures', () => {
  const brand = read('src/components/TheiaBrand.jsx');
  const sigil = read('src/components/TheiaSigil.jsx');
  const scenes = read('src/components/EarthshineScene.jsx');
  assert.match(brand, /THEIA_LOGO_SRC = '\/brand\/theia-sigil\.svg'/);
  assert.match(sigil, /earthshine/);
  assert.match(scenes, /EARTHSHINE_SCENES/);

  // All 8 Earthshine story scenes are defined with accessible Persian labels and increasing/coherent legibility.
  const expectedScenes = ['welcome', 'privacy', 'night', 'seeing', 'dawn', 'three-lights', 'checkins', 'final'];
  assert.deepEqual(Object.keys(EARTHSHINE_SCENES), expectedScenes);
  assert.ok(EARTHSHINE_SCENES.welcome.earthshine < EARTHSHINE_SCENES.night.earthshine);
  assert.ok(EARTHSHINE_SCENES.night.earthshine < EARTHSHINE_SCENES.seeing.earthshine);
  assert.ok(EARTHSHINE_SCENES.seeing.earthshine < EARTHSHINE_SCENES.dawn.earthshine);
  assert.ok(EARTHSHINE_SCENES.dawn.earthshine < EARTHSHINE_SCENES['three-lights'].earthshine);

  const active = [
    read('src/components/UI.jsx'),
    read('src/components/EarthshineScene.jsx'),
    read('src/screens/Onboarding.jsx'),
    read('src/screens/Main.jsx'),
    read('src/screens/Letters.jsx'),
    read('src/features/devsession/DevLogin.jsx'),
  ].join('\n');

  for (const retired of ['photo_2026-11-12_19-20-10.jpg', 'theia-welcome.png', 'theia-story-1.png', 'theia-story-2.png', 'theia-story-3.png', 'theia-story-4.png', 'theia-privacy.png', 'theia-ready.png', 'theia-checkins.png']) {
    assert.ok(!active.includes(retired), `Retired female figure still referenced: ${retired}`);
  }

  const onboarding = read('src/screens/Onboarding.jsx');
  for (const scene of ['welcome', 'privacy', 'night', 'seeing', 'dawn', 'three-lights', 'checkins', 'final']) {
    assert.ok(
      onboarding.includes(`scene="${scene}"`) || onboarding.includes(`scene: '${scene}'`),
      `Onboarding missing Earthshine scene: ${scene}`,
    );
  }
});

test('Earthshine storyboard Persian copy, AI transparency, HTML head and manifest are wired', () => {
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

  // Verify the 8 Earthshine scene titles and key narrative copy
  const requiredOnboardingLines = [
    'تیا را نمی‌بینی؛ با تیا می‌بینی.',
    'آن‌چه از پیش در توست، روشن‌تر دیده شود.',
    'چه‌قدر بگویی، انتخاب توست.',
    'از تاریکی شروع می‌کنیم.',
    'چشم که عادت کند، چیزها کم‌کم پیدا می‌شوند.',
    'آن‌چه تاریک به نظر می‌رسد، خالی نیست.',
    'نوری که از زمین به آن می‌رسد، به چشم تو برمی‌گردد.',
    'سپیده یک‌باره نمی‌آید.',
    'گاهی نوری می‌آید و دوباره پس می‌رود؛ آن هم بخشی از راه است.',
    'سه نور؛ و هیچ‌کدام همیشگی نیست.',
    'اگر بخواهی، چراغ روشن می‌ماند.',
    'از این‌جا شروع می‌کنیم.',
    'آسمان همان است؛ فقط چشمت عادت کرده.',
    'تیا یک همراه هوش مصنوعی است و جایگزین روان‌درمانگر نیست.',
  ];
  for (const line of requiredOnboardingLines) {
    assert.ok(onboarding.includes(line), `Missing Earthshine copy line: ${line}`);
  }

  assert.match(main, /نامه‌های تیا/);
  assert.match(letters, /نامه‌های تیا/);
  assert.match(devLogin, /به تیا خوش آمدی/);
  assert.match(data, /چطور با تیا آشنا شدی؟/);
});

test('active UI keeps the Theia visual vocabulary', () => {
  const activeSource = [
    read('src/components/UI.jsx'),
    read('src/components/TheiaBrand.jsx'),
    read('src/components/TheiaSigil.jsx'),
    read('src/components/EarthshineScene.jsx'),
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
