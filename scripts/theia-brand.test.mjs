import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');
const exists = (path) => existsSync(new URL(path, root));

test('Theia identity assets exist: outlined logo, symbol, icons, favicon and manifest', () => {
  const assets = [
    'public/favicon.svg',
    'public/apple-touch-icon.png',
    'public/manifest.webmanifest',
    'public/brand/theia-logo.svg',
    'public/brand/theia-logo-light.svg',
    'public/brand/theia-mark.svg',
    'public/brand/theia-mark-light.svg',
    'public/brand/theia-icon.svg',
    'public/brand/theia-icon-192.png',
    'public/brand/theia-icon-512.png',
  ];
  for (const file of assets) {
    const url = new URL(file, root);
    assert.ok(existsSync(url), `Missing brand asset: ${file}`);
    assert.ok(statSync(url).size > 200, `Empty or invalid brand asset: ${file}`);
  }

  // The wordmark ships as outlines: the logo never depends on an installed font.
  for (const file of ['public/brand/theia-logo.svg', 'public/brand/theia-logo-light.svg']) {
    const svg = read(file);
    assert.ok(!svg.includes('<text'), `${file} must not rely on live text/fonts`);
    assert.match(svg, /aria-label="Theia — تیا"/);
    assert.match(svg, /<path /);
  }
});

test('the previous goddess/female-character illustrations are removed, not reskinned', () => {
  const retired = [
    'public/brand/theia-welcome.png',
    'public/brand/theia-privacy.png',
    'public/brand/theia-story-1.png',
    'public/brand/theia-story-2.png',
    'public/brand/theia-story-3.png',
    'public/brand/theia-story-4.png',
    'public/brand/theia-checkins.png',
    'public/brand/theia-ready.png',
  ];
  for (const file of retired) {
    assert.ok(!exists(file), `Retired character artwork still present: ${file}`);
  }

  const activeSource = [
    read('src/components/TheiaBrand.jsx'),
    read('src/screens/Onboarding.jsx'),
    read('src/screens/Main.jsx'),
    read('src/screens/Letters.jsx'),
    read('src/features/devsession/DevLogin.jsx'),
    read('src/components/UI.jsx'),
  ].join('\n');

  for (const legacy of ['theia-welcome.png', 'theia-story-', 'theia-ready.png', 'theia-checkins.png', 'theia-privacy.png']) {
    assert.ok(!activeSource.includes(legacy), `Legacy illustration still referenced: ${legacy}`);
  }
  for (const legacy of ['Flower2', 'flower-person-', 'welcome-figures.png', 'ready-daisy.png', 'privacy-flower.png', 'checkins-handdrawn.svg', 'الههٔ']) {
    assert.ok(!activeSource.includes(legacy), `Legacy visual reference still active: ${legacy}`);
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
  assert.equal(manifest.background_color, '#12110d');

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
});

test('onboarding is a single visual narrative: one drawing whose opening closes toward Ready', () => {
  const brand = read('src/components/TheiaBrand.jsx');
  const onboarding = read('src/screens/Onboarding.jsx');

  // Every authored chapter is wired to a screen.
  for (const variant of ['welcome', 'simple', 'deeper', 'know', 'almost', 'privacy', 'checkins', 'ready']) {
    assert.match(brand, new RegExp(`function\\s+\\w+Scene`), 'scene components must exist');
  }
  assert.match(brand, /export const THEIA_CHAPTERS/);
  assert.match(brand, /export function TheiaScene/);
  for (const variant of ['welcome', 'privacy', 'checkins', 'ready']) {
    assert.ok(onboarding.includes(`variant="${variant}"`), `Onboarding missing ${variant} scene`);
  }
  for (const scene of ['simple', 'deeper', 'know', 'almost']) {
    assert.ok(onboarding.includes(`scene: '${scene}'`), `Onboarding story missing ${scene} chapter`);
  }

  // Changes the goddess/female character on every onboarding artwork screen.
  assert.ok(!onboarding.includes('TheiaGoddess'));
  assert.ok(!onboarding.includes('/brand/'), 'Onboarding must use authored scenes, not raster brand files');

  // The narrative is testable: the circle narrows page by page and closes at Ready,
  // then opens again on Home where the conversation begins.
  const gapFor = (key) => {
    const block = brand.match(new RegExp(`${key}: \\{[^}]*\\}`));
    assert.ok(block, `Missing chapter: ${key}`);
    return Number(block[0].match(/gap: (\d+)/)[1]);
  };
  const sequence = ['welcome', 'simple', 'deeper', 'know', 'almost'];
  const gaps = sequence.map(gapFor);
  for (let index = 1; index < gaps.length; index += 1) {
    assert.ok(gaps[index] < gaps[index - 1], `The opening must narrow: ${sequence[index - 1]} → ${sequence[index]}`);
  }
  assert.equal(gapFor('ready'), 0, 'The circle closes on Ready');
  assert.ok(gapFor('home') > gapFor('ready'), 'Home opens the circle again');
  assert.equal(gapFor('privacy'), 0, 'Privacy is a closed boundary');
});

test('active UI uses the new Theia symbol family everywhere the old figure appeared', () => {
  const brand = read('src/components/TheiaBrand.jsx');
  const ui = read('src/components/UI.jsx');
  const letters = read('src/screens/Letters.jsx');
  const main = read('src/screens/Main.jsx');
  const devLogin = read('src/features/devsession/DevLogin.jsx');

  for (const symbol of ['TheiaMark', 'TheiaSealIcon', 'TheiaHomeArt', 'TheiaLetterEmpty', 'TheiaScene']) {
    assert.ok(brand.includes(`export function ${symbol}`), `Missing symbol: ${symbol}`);
  }
  assert.ok(ui.includes('TheiaMark'), 'Wax seal must use the Theia mark');
  assert.ok(letters.includes('TheiaLetterEmpty') && letters.includes('TheiaMark'));
  assert.ok(main.includes('TheiaHomeArt') && main.includes('TheiaSealIcon'));
  assert.ok(devLogin.includes('TheiaScene'));

  // The open-circle system, not the retired goddess/crescent figure.
  assert.ok(!brand.includes('Goddess'));
  assert.ok(!brand.includes('home-orbit-crescent'));
});

test('brand palette tokens and the open-circle motion system are defined in CSS', () => {
  const css = read('src/styles.css');
  for (const token of ['--theia-night', '--theia-paper', '--theia-ivory', '--theia-dawn', '--theia-ember', '--theia-gold', '--theia-dusk']) {
    assert.ok(css.includes(token), `Missing brand token ${token}`);
  }
  assert.match(css, /Theia brand: «دایرهٔ باز»/);
  assert.match(css, /\.theia-art::after/);
  assert.match(css, /prefers-reduced-motion[\s\S]*theia-/);
});
