import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
const root = new URL('../', import.meta.url);
const read = file => readFileSync(new URL(file, root), 'utf8');

test('Theia identity replaces the floral brand without changing the route flow', () => {
  const src = ['src/components/UI.jsx','src/screens/Onboarding.jsx','src/screens/Main.jsx','src/screens/Letters.jsx','src/features/devsession/DevLogin.jsx'].map(read).join('\n');
  assert.ok(!src.includes('Flower2'));
  for (const old of ['welcome-figures.png','flower-person-','privacy-flower.png','ready-daisy.png','checkins-handdrawn.svg']) assert.ok(!src.includes(old), old);
  assert.ok(src.includes('TheiaMark'));
  assert.ok(read('src/data.js').includes("'welcome', 'privacy', 'simple-intro'"));
  assert.ok(read('index.html').includes('lang="fa" dir="rtl"'));
  assert.ok(read('index.html').includes('<title>تیا — Theia</title>'));
});

test('original narrative assets and install icons are present', () => {
  for (const name of ['welcome','privacy','simple-intro','deeper-intro','know-intro','almost-intro','checkins','ready']) {
    const file = `public/assets/theia/${name}.svg`;
    assert.ok(existsSync(new URL(file, root)), file);
    assert.ok(read(file).includes('<svg'));
  }
  for (const file of ['public/icons/icon-192.png','public/icons/icon-512.png','public/icons/icon-maskable-512.png','public/icons/apple-touch-icon.png','public/manifest.webmanifest']) assert.ok(existsSync(new URL(file, root)), file);
  const manifest = JSON.parse(read('public/manifest.webmanifest'));
  assert.equal(manifest.lang, 'fa'); assert.equal(manifest.dir, 'rtl'); assert.equal(manifest.short_name, 'تیا');
});

test('legacy compatibility keys are not renamed by the visual rebrand', () => {
  assert.ok(read('src/features/profile/onboardingProgress.js').includes("FLOW_VERSION = 'hamrah-v2'"));
  assert.ok(read('src/features/letters/readingState.js').includes("PREFIX = 'hamrah:letter-seen:v1:'"));
  assert.ok(read('src/features/letters/discuss.js').includes('HAMRAH_LETTER_CONTEXT_V1'));
});
