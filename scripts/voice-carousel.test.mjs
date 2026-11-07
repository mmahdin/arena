import assert from 'node:assert/strict';
import { test } from 'node:test';
import { resistVoiceDrag, voiceSwipeTarget } from '../src/features/onboarding/voiceCarousel.js';

const target = (index, dx, velocity = 0, step = 224, count = 6) =>
  voiceSwipeTarget({ index, dx, velocity, step, count });

test('a short drag returns to the current card; a real drag selects the next card', () => {
  assert.equal(target(2, -20), 2);
  assert.equal(target(2, -90), 3);
  assert.equal(target(2, 90), 1);
});

test('a quick flick commits; a slow, tiny touch never commits', () => {
  assert.equal(target(1, -26, -0.75), 2);
  assert.equal(target(1, 26, 0.75), 0);
  assert.equal(target(1, -10, -1.1), 1);
});

test('momentum is limited to two positions and edges never wrap', () => {
  assert.equal(target(2, -460, -1), 4);
  assert.equal(target(4, -460, -1), 5);
  assert.equal(target(0, 200, 1), 0);
  assert.equal(target(5, -200, -1), 5);
});

test('edge resistance is gradual and symmetric', () => {
  assert.equal(resistVoiceDrag(100, 0, 6), 22);
  assert.equal(resistVoiceDrag(-100, 5, 6), -22);
  assert.equal(resistVoiceDrag(100, 2, 6), 100);
  assert.equal(resistVoiceDrag(-1000, 5, 6), -48);
});
