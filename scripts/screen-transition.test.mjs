import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { screenSurfaceForRoute } from '../src/features/appearance/screenSurface.js';

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

test('2026 onboarding alternates night ink and ivory paper surfaces', () => {
  for (const route of ['welcome', 'simple-intro', 'deeper-intro', 'know-intro', 'almost-intro', 'voice', 'approach', 'ready']) {
    assert.equal(screenSurfaceForRoute(route), 'ink', route);
  }
  for (const route of ['privacy', 'name', 'age', 'therapist', 'interests', 'mood', 'checkins', 'referral', 'concerns']) {
    assert.equal(screenSurfaceForRoute(route), 'paper', route);
  }
  assert.equal(screenSurfaceForRoute('testimonials'), 'beige');
});

test('the route-controlled shell and transition never fade dark screens from transparent white', () => {
  assert.match(app, /data-screen-surface=\{active \? screenSurfaceForRoute\(route\) : 'ink'\}/);
  const onboardingCss = readFileSync(new URL('../src/onboarding.css', import.meta.url), 'utf8');
  assert.match(onboardingCss, /\.app-shell\[data-screen-surface="ink"\]\s*\{\s*--screen-surface:\s*#0a0b0d/);
  assert.match(onboardingCss, /\.app-shell\[data-screen-surface="paper"\]\s*\{\s*--screen-surface:\s*#f2ede4/);
  assert.match(styles, /\.app-shell\[data-screen-surface="dark"\]\s*\{\s*--screen-surface:\s*var\(--dark\)/);
  assert.match(styles, /\.screen-transition\s*\{[^}]*background:\s*var\(--screen-surface\)/s);
  assert.match(styles, /@keyframes screen-in\s*\{\s*from\s*\{\s*opacity:\s*\.96/);
  assert.doesNotMatch(styles, /@keyframes screen-in\s*\{\s*from\s*\{\s*opacity:\s*0\s*;/);
  assert.match(styles, /@media\s*\(prefers-reduced-motion:\s*reduce\)\s*\{\s*\.screen-transition\s*\{\s*animation:\s*none/);
});
