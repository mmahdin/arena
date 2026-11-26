import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { screenSurfaceForRoute } from '../src/features/appearance/screenSurface.js';
import { FLOW } from '../src/data.js';

const styles = readFileSync(new URL('../src/styles.css', import.meta.url), 'utf8');
const app = readFileSync(new URL('../src/App.jsx', import.meta.url), 'utf8');

test('chat, home, history and all main dark screens use a dark underlying surface', () => {
  for (const route of ['home', 'chat', 'history', 'survey', 'day', 'letter', 'insight']) {
    assert.equal(screenSurfaceForRoute(route), 'dark', route);
  }
  for (const route of ['menu', 'research', 'experts']) {
    assert.equal(screenSurfaceForRoute(route), 'black', route);
  }
});

test('Earthshine keeps one ink surface across the whole pre-dawn onboarding', () => {
  for (const route of FLOW.filter((item) => item !== 'home')) {
    assert.equal(screenSurfaceForRoute(route), 'ink', route);
  }
  assert.equal(screenSurfaceForRoute('testimonials'), 'beige');
});

test('the route-controlled shell and transition never fade dark screens from transparent white', () => {
  assert.match(app, /data-screen-surface=\{active \? screenSurfaceForRoute\(route\) : 'ink'\}/);
  const onboardingCss = readFileSync(new URL('../src/onboarding.css', import.meta.url), 'utf8');
  assert.match(onboardingCss, /\.app-shell\[data-screen-surface="ink"\]\s*\{\s*--screen-surface:\s*#0a0b0d/);
  assert.match(onboardingCss, /\.app-shell\[data-screen-surface="paper"\]\s*\{\s*--screen-surface:\s*#f2ede4/);
  const earthshineCss = readFileSync(new URL('../src/earthshine.css', import.meta.url), 'utf8');
  assert.match(earthshineCss, /\.app-shell\[data-screen-surface='paper'\]/);
  assert.match(styles, /\.app-shell\[data-screen-surface="dark"\]\s*\{\s*--screen-surface:\s*var\(--dark\)/);
  assert.match(styles, /\.screen-transition\s*\{[^}]*background:\s*var\(--screen-surface\)/s);
  assert.match(styles, /@keyframes screen-in\s*\{\s*from\s*\{\s*opacity:\s*\.96/);
  assert.doesNotMatch(styles, /@keyframes screen-in\s*\{\s*from\s*\{\s*opacity:\s*0\s*;/);
  assert.match(styles, /@media\s*\(prefers-reduced-motion:\s*reduce\)\s*\{\s*\.screen-transition\s*\{\s*animation:\s*none/);
});
