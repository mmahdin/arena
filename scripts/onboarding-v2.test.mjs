import assert from 'node:assert/strict';
import { test } from 'node:test';
import { FLOW, QUESTIONS, LEGACY_INFO_ROUTES } from '../src/data.js';
import { ONBOARDING_ROUTES, onboardingMove, previousOnboardingRoute, canVisitOnboarding } from '../src/features/profile/navigation.js';
import { createOnboardingGreeting } from '../src/features/profile/onboardingGreeting.js';

test('v2 has exactly the requested 26 pages and a final direct-chat handoff', () => {
  assert.deepEqual(FLOW, [
    'welcome', 'privacy', 'simple-intro', 'name', 'age', 'identity',
    'therapist', 'deeper-intro', 'mood', 'concerns', 'focus', 'feelings',
    'know-intro', 'faith', 'neurodivergent', 'relationships', 'interests',
    'strengths', 'barriers', 'recharge', 'almost-intro', 'approach',
    'voice', 'checkins', 'referral', 'ready', 'home',
  ]);
  assert.equal(ONBOARDING_ROUTES.length, 26);
  assert.equal(previousOnboardingRoute('ready'), 'referral');
  for (const oldPage of LEGACY_INFO_ROUTES) assert.ok(!ONBOARDING_ROUTES.includes(oldPage));
});

test('v2 respects consent-sensitive options and allows no restorative selections', () => {
  assert.deepEqual(QUESTIONS.identity.options, ['زن', 'مرد', 'تمایل ندارم بگم']);
  assert.equal(QUESTIONS.recharge.options.length, 6);
  assert.equal(QUESTIONS.recharge.allowEmpty, true);
  assert.deepEqual(QUESTIONS.referral.options, ['اینستاگرام', 'چت‌بات هوش مصنوعی', 'تلگرام', 'بله']);
});

test('no route can skip a new step or enter private chat before completion', () => {
  for (let i = 0; i < ONBOARDING_ROUTES.length - 1; i++) {
    assert.equal(onboardingMove(ONBOARDING_ROUTES[i], ONBOARDING_ROUTES[i + 1], i), i + 1);
    assert.equal(canVisitOnboarding(ONBOARDING_ROUTES[i + 1], i), false);
    if (i < ONBOARDING_ROUTES.length - 2) {
      assert.equal(onboardingMove(ONBOARDING_ROUTES[i], ONBOARDING_ROUTES[i + 2], i), null);
    }
  }
  assert.equal(onboardingMove('ready', 'home', 25), null);
});

test('only minimal one-time greeting uses selection, not hidden structured data', () => {
  const text = createOnboardingGreeting('آوا', { mood: 0, concerns: ['کار یا درس'], focus: ['بهتر بخوابم'] });
  assert.match(text, /آوا/);
  assert.match(text, /سخت/);
  assert.match(text, /کار یا درس/);
  assert.match(text, /بهتر بخوابم/);
  assert.doesNotMatch(createOnboardingGreeting('آوا', { identity: 'زن', faith: 'مهم' }), /زن|مهم/);
});

test('name is persisted on its own onboarding step, not requested again on ready', async () => {
  const { readFile } = await import('node:fs/promises');
  const app = await readFile(new URL('../src/App.jsx', import.meta.url), 'utf8');
  const screens = await readFile(new URL('../src/screens/Onboarding.jsx', import.meta.url), 'utf8');
  assert.match(app, /devSessionApi\.saveOnboardingName\(displayName\)/);
  assert.match(screens, /onSaveOnboardingName/);
  assert.doesNotMatch(app, /recoveringAtReady/);
  assert.doesNotMatch(app, /replaceRoute\('name'\);[\s\S]{0,80}setRoute\('name'\);/);
});

test('name-only API saves the selected name without completing onboarding', async () => {
  const { devSessionApi } = await import('../src/features/devsession/api.js');
  const original = globalThis.fetch;
  const calls = [];
  globalThis.fetch = async (url, options) => {
    calls.push({ url, options });
    return new Response(JSON.stringify({
      userId: 'synthetic-user', displayName: 'مهسا', conversationStyle: 'balanced', onboardingCompletedAt: null,
    }), { status: 200 });
  };
  try {
    const p = await devSessionApi.saveOnboardingName('مهسا');
    assert.equal(p.displayName, 'مهسا');
    assert.equal(p.onboardingCompletedAt, null);
    assert.equal(calls[0].url, '/api/v1/me/onboarding-name');
    assert.equal(calls[0].options.method, 'PATCH');
    assert.deepEqual(JSON.parse(calls[0].options.body), { displayName: 'مهسا' });
    assert.equal(calls[0].options.credentials, 'include');
  } finally {
    globalThis.fetch = original;
  }
});
