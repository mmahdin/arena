import assert from 'node:assert/strict';
import { test } from 'node:test';
import { buildOnboardingAnswers } from '../src/features/profile/onboardingAnswers.js';
import { devSessionApi } from '../src/features/devsession/api.js';
import { createSessionStore } from '../src/features/devsession/sessionStore.js';

test('only explicit valid selections enter onboarding context, including mood zero', () => {
  assert.deepEqual(buildOnboardingAnswers({
    age: '۲۵–۲۹', mood: 0, identity: 'تمایل ندارم بگم',
    concerns: ['کار یا درس', 'کار یا درس', 'injected'], focus: ['بهتر بخوابم'],
    interests: [], faith: 'injected', approach: 'هرچی پیش اومد',
    voice: 'بید', checkins: true, referral: 'تلگرام', healthSurvey: 'private', userId: 'other',
  }), {
    age: '۲۵–۲۹', mood: 0, identity: 'تمایل ندارم بگم', concerns: ['کار یا درس'],
    focus: ['بهتر بخوابم'], interests: [],
  });
  assert.deepEqual(buildOnboardingAnswers({ mood: 5, identity: 'غیردودویی', age: 'unknown' }), {});
});

test('final profile transport sends selected answers; settings edits omit them', async () => {
  const original = globalThis.fetch;
  const calls = [];
  const profile = { userId: 'synthetic-a', displayName: 'Synthetic', conversationStyle: 'balanced', onboardingCompletedAt: '2026-11-11T00:00:00Z' };
  globalThis.fetch = async (url, options) => {
    calls.push({ url, options });
    return new Response(JSON.stringify(profile), { status: 200 });
  };
  try {
    const answers = { mood: 0, concerns: ['کار یا درس'] };
    await devSessionApi.profile('Synthetic', 'balanced', undefined, answers);
    await devSessionApi.profile('Synthetic', 'balanced');
    assert.equal(calls[0].url, '/api/v1/me/profile');
    assert.equal(calls[0].options.credentials, 'include');
    assert.deepEqual(JSON.parse(calls[0].options.body).onboardingAnswers, answers);
    assert.equal(Object.hasOwn(JSON.parse(calls[1].options.body), 'onboardingAnswers'), false);
  } finally { globalThis.fetch = original; }
});

test('session store forwards answers and clears saved background on tester switch', async () => {
  let current = { userId: 'synthetic-a', displayName: '', conversationStyle: 'balanced', onboardingCompletedAt: null };
  let submitted;
  const store = createSessionStore({
    async me() { return current; },
    async profile(name, style, signal, answers) {
      submitted = answers;
      assert.equal(signal instanceof AbortSignal, true);
      current = { ...current, displayName: name, conversationStyle: style, onboardingAnswers: answers, onboardingCompletedAt: '2026-11-11T00:00:00Z' };
      return current;
    },
    async start() { current = { userId: 'synthetic-b', displayName: 'B', conversationStyle: 'listening', onboardingCompletedAt: null }; },
  });
  await store.bootstrap();
  const answers = { concerns: ['کار یا درس'] };
  assert.equal(await store.saveProfile('Synthetic', 'balanced', answers), true);
  assert.deepEqual(submitted, answers);
  assert.deepEqual(store.getState().profile.onboardingAnswers, answers);
  await store.login('09000000002');
  assert.equal(store.getState().profile.userId, 'synthetic-b');
  assert.equal(store.getState().profile.onboardingAnswers, undefined);
});
