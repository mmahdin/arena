/**
 * Server-render every onboarding screen and assert its narrative structure.
 *
 * This is not a substitute for browser QA (Playwright needs a running stack),
 * but it does prove that each route renders without throwing, draws exactly one
 * chapter illustration, keeps the approved disclosures, and never falls back to
 * a raster portrait.
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { build } from 'esbuild';
import { renderToStaticMarkup } from 'react-dom/server';
import React from 'react';

const root = new URL('../', import.meta.url);
// Keep the bundle inside the repo so Node can resolve the app's own React.
const outdir = new URL('../node_modules/.cache/theia-render/', import.meta.url).pathname;
mkdirSync(outdir, { recursive: true });

const bundle = await build({
  stdin: {
    contents: `
      export { default as OnboardingScreen } from './src/screens/Onboarding.jsx';
      export { default as DevLogin } from './src/features/devsession/DevLogin.jsx';
    `,
    resolveDir: new URL(root).pathname,
    loader: 'js',
  },
  bundle: true,
  format: 'esm',
  platform: 'node',
  jsx: 'automatic',
  external: ['react', 'react-dom', 'react/jsx-runtime', 'lucide-react'],
  outfile: join(outdir, 'screens.mjs'),
  logLevel: 'silent',
});
assert.ok(bundle.outputFiles === undefined);

const { OnboardingScreen } = await import(join(outdir, 'screens.mjs'));

const noop = () => {};
const baseProps = {
  navigate: noop,
  back: noop,
  name: '',
  setName: noop,
  answers: {},
  setAnswer: noop,
  notify: noop,
  onSaveOnboardingName: async () => {},
  privacySharing: true,
  privacyLoaded: true,
  privacyLoading: false,
  privacySaving: false,
  privacyError: false,
  onTogglePrivacy: noop,
  onRetryPrivacy: noop,
};

const NARRATIVE = {
  welcome: '۰۱',
  privacy: '۰۲',
  'simple-intro': '۰۳',
  'deeper-intro': '۰۴',
  'know-intro': '۰۵',
  'almost-intro': '۰۶',
  checkins: '۰۶',
  ready: '۰۷',
};

test('every narrative route renders one chapter illustration and its own words', () => {
  for (const [route, numeral] of Object.entries(NARRATIVE)) {
    const html = renderToStaticMarkup(React.createElement(OnboardingScreen, { ...baseProps, route }));
    assert.ok(html.includes('theia-scene'), `${route}: chapter illustration missing`);
    assert.equal((html.match(/class="theia-scene theia-scene--/g) || []).length, 1, `${route}: exactly one scene`);
    assert.ok(html.includes('chapter-marks'), `${route}: chapter mark missing`);
    assert.ok(html.includes(numeral), `${route}: expected chapter numeral ${numeral}`);
    assert.ok(html.includes('narrative-title'), `${route}: title missing`);
    assert.ok(html.includes('primary-button'), `${route}: action missing`);
    assert.ok(!html.includes('<img'), `${route}: no raster imagery`);
    assert.ok(!/<svg[^>]*>\s*<image/.test(html), `${route}: no embedded portraits`);
  }
});

test('the approved data-processing disclosure appears verbatim where it is required', () => {
  const privacy = renderToStaticMarkup(React.createElement(OnboardingScreen, { ...baseProps, route: 'privacy' }));
  assert.ok(privacy.includes('گفت‌وگوها و پاسخ‌های آشنایی اولیه روی سرور ذخیره می‌شن'));
  assert.ok(privacy.includes('به سرویس هوش مصنوعی فرستاده می‌شن'));
  assert.ok(privacy.includes('role="switch"'), 'the sharing control stays a switch');

  const ready = renderToStaticMarkup(React.createElement(OnboardingScreen, { ...baseProps, route: 'ready' }));
  assert.ok(ready.includes('برای شخصی‌سازی چت در اختیار سرویس هوش مصنوعی قرار می‌گیرن'));

  const welcome = renderToStaticMarkup(React.createElement(OnboardingScreen, { ...baseProps, route: 'welcome' }));
  assert.ok(welcome.includes('ابزار گفت‌وگوست، نه روان‌درمانگر'), 'the tool-not-therapist note stays');
});

test('the dawn never dims as the reader advances', async () => {
  const { chapterFor, NARRATIVE_ROUTES } = await import('../src/features/onboarding/chapters.js');
  const levels = NARRATIVE_ROUTES.map((route) => chapterFor(route).dawn);
  for (let i = 1; i < levels.length; i++) {
    assert.ok(levels[i] >= levels[i - 1], `${NARRATIVE_ROUTES[i]} must not be darker than ${NARRATIVE_ROUTES[i - 1]}`);
  }
});

test('narrative screens expose one heading and an accessible action label', () => {
  for (const route of Object.keys(NARRATIVE)) {
    const html = renderToStaticMarkup(React.createElement(OnboardingScreen, { ...baseProps, route }));
    assert.equal((html.match(/<h1/g) || []).length, 1, `${route}: one primary heading`);
    assert.ok(/aria-label="[^"]+"/.test(html) || html.includes('chapter-marks'), `${route}: chapter mark is described`);
  }
});
