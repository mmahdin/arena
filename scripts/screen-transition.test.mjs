import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { screenSurfaceForRoute } from '../src/features/appearance/screenSurface.js';

const styles = readFileSync(new URL('../src/styles.css', import.meta.url), 'utf8');
const app = readFileSync(new URL('../src/App.jsx', import.meta.url), 'utf8');

test('chat, home, history and all main dark screens use a dark underlying surface', () => {
  for (const route of ['home', 'chat', 'history', 'saved', 'survey', 'day', 'letter', 'insight', 'voice', 'mood']) {
    assert.equal(screenSurfaceForRoute(route), 'dark', route);
  }
  for (const route of ['welcome', 'menu', 'simple-intro', 'deeper-intro', 'know-intro', 'almost-intro']) {
    assert.equal(screenSurfaceForRoute(route), 'black', route);
  }
});

test('designer beige and light onboarding pages retain their surfaces', () => {
  for (const route of ['approach', 'interests']) assert.equal(screenSurfaceForRoute(route), 'beige');
  for (const route of ['name', 'privacy', 'ready', 'therapist', 'age', 'concerns', 'research']) {
    assert.equal(screenSurfaceForRoute(route), 'light', route);
  }
});

test('the route-controlled shell and transition never fade dark screens from transparent white', () => {
  assert.match(app, /data-screen-surface=\{active \? screenSurfaceForRoute\(route\) : 'light'\}/);
  assert.match(styles, /\.app-shell\[data-screen-surface="dark"\]\s*\{\s*--screen-surface:\s*var\(--dark\)/);
  assert.match(styles, /\.screen-transition\s*\{[^}]*background:\s*var\(--screen-surface\)/s);
  assert.match(styles, /@keyframes screen-in\s*\{\s*from\s*\{\s*opacity:\s*\.96/);
  assert.doesNotMatch(styles, /@keyframes screen-in\s*\{\s*from\s*\{\s*opacity:\s*0\s*;/);
  assert.match(styles, /@media\s*\(prefers-reduced-motion:\s*reduce\)\s*\{\s*\.screen-transition\s*\{\s*animation:\s*none/);
});
