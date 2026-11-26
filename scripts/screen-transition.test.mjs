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

test('Earthshine onboarding progresses continuously from night ink to civil twilight', () => {
  for (const route of ['welcome', 'privacy']) {
    assert.equal(screenSurfaceForRoute(route), 'ink', route);
  }
  for (const route of ['simple-intro', 'name', 'age', 'identity', 'therapist']) {
    assert.equal(screenSurfaceForRoute(route), 'night', route);
  }
  for (const route of ['deeper-intro', 'mood', 'concerns', 'focus', 'feelings']) {
    assert.equal(screenSurfaceForRoute(route), 'seeing', route);
  }
  for (const route of ['know-intro', 'faith', 'neurodivergent', 'relationships', 'interests', 'strengths', 'barriers', 'recharge']) {
    assert.equal(screenSurfaceForRoute(route), 'dawn', route);
  }
  for (const route of ['almost-intro', 'approach', 'voice', 'checkins', 'referral']) {
    assert.equal(screenSurfaceForRoute(route), 'morning', route);
  }
  assert.equal(screenSurfaceForRoute('ready'), 'twilight');
  assert.equal(screenSurfaceForRoute('testimonials'), 'beige');
});

test('the route-controlled shell and transition never fade dark screens from transparent white', () => {
  assert.match(app, /data-screen-surface=\{active \? screenSurfaceForRoute\(route\) : 'ink'\}/);
  const onboardingCss = readFileSync(new URL('../src/onboarding.css', import.meta.url), 'utf8');
  for (const surface of ['ink', 'night', 'seeing', 'dawn', 'morning', 'twilight']) {
    assert.match(onboardingCss, new RegExp(`\\.app-shell\\[data-screen-surface="${surface}"\\]\\s*\\{\\s*--screen-surface:`));
  }
  assert.match(styles, /\.app-shell\[data-screen-surface="dark"\]\s*\{\s*--screen-surface:\s*var\(--dark\)/);
  assert.match(styles, /\.screen-transition\s*\{[^}]*background:\s*var\(--screen-surface\)/s);
  assert.match(styles, /@keyframes screen-in\s*\{\s*from\s*\{\s*opacity:\s*\.96/);
  assert.doesNotMatch(styles, /@keyframes screen-in\s*\{\s*from\s*\{\s*opacity:\s*0\s*;/);
  assert.match(styles, /@media\s*\(prefers-reduced-motion:\s*reduce\)\s*\{\s*\.screen-transition\s*\{\s*animation:\s*none/);
});
