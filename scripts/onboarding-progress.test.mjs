import assert from 'node:assert/strict';
import { test } from 'node:test';
import { FLOW } from '../src/data.js';
import { FLOW_VERSION, validateProgress, onboardingProgressApi } from '../src/features/profile/onboardingProgress.js';

const ok = (n) => ({ furthestStep: n, flowVersion: FLOW_VERSION });

test('progress cursor aligns exactly to new 26-screen designer flow, with no Home index', () => {
  assert.equal(FLOW.length, 27);
  assert.equal(FLOW[25], 'ready');
  assert.equal(FLOW[26], 'home');
  for (let i = 0; i < 26; i++) assert.equal(validateProgress(ok(i)), i);
});

test('invalid / changed flow version cannot misroute a returning user', () => {
  for (const value of [null, {}, { furthestStep: 23 }, ok(26), ok(-1), ok(1.5), ok('9'), { ...ok(2), flowVersion: 'unexpected-v2' }]) {
    assert.throws(() => validateProgress(value));
  }
});

test('progress API uses cookies and only an integer index, never questionnaire answers', async () => {
  const original = globalThis.fetch;
  const calls = [];
  globalThis.fetch = async (url, options) => {
    calls.push({ url, options });
    return { ok: true, status: 200, async json() { return ok(calls.length); } };
  };
  try {
    assert.equal(await onboardingProgressApi.read(), 1);
    assert.equal(await onboardingProgressApi.advance(2), 2);
    assert.deepEqual(calls.map(c => c.url), ['/api/v1/me/onboarding-progress', '/api/v1/me/onboarding-progress']);
    assert.deepEqual(calls.map(c => c.options.method), ['GET', 'PUT']);
    assert.ok(calls.every(c => c.options.credentials === 'include'));
    assert.deepEqual(JSON.parse(calls[1].options.body), { furthestStep: 2 });
    await assert.rejects(onboardingProgressApi.advance(26));
    assert.equal(calls.length, 2);
  } finally { globalThis.fetch = original; }
});

test('UI fetches the session-owned cursor and retains all original screens', async () => {
  const { readFile } = await import('node:fs/promises');
  const app = await readFile(new URL('../src/App.jsx', import.meta.url), 'utf8');
  for (const required of ['onboardingProgressApi.read(', 'onboardingProgressApi.advanceTo(', "progressPhase !== 'ready'", 'identityVersion.current !== version', 'ONBOARDING_ROUTES[furthest]', 'setAnswers({})']) {
    assert.ok(app.includes(required), `missing cursor guard: ${required}`);
  }
  assert.ok(!app.includes('localStorage.setItem'));
});


test('sequential PUTs, preserve acknowledged progress and are safely retryable', async () => {
  const original = globalThis.fetch; const calls = []; const acknowledged = [];
  let fail = true;
  globalThis.fetch = async (_, options) => {
    const { furthestStep } = JSON.parse(options.body); calls.push(furthestStep);
    if (furthestStep === 3 && fail) throw new Error('lost network');
    return new Response(JSON.stringify(ok(furthestStep)), { status: 200 });
  };
  try {
    await assert.rejects(onboardingProgressApi.advanceTo(4, 0, undefined, step => acknowledged.push(step)));
    assert.deepEqual(acknowledged, [1, 2]);
    fail = false;
    assert.equal(await onboardingProgressApi.advanceTo(4, acknowledged.at(-1)), 4);
    assert.deepEqual(calls, [1, 2, 3, 3, 4]);
    for (const saved of [1, 2, 3]) assert.equal(await onboardingProgressApi.advanceTo(4, saved), 4);
    const controller = new AbortController(); controller.abort();
    const before = calls.length;
    await assert.rejects(onboardingProgressApi.advanceTo(4, 0, controller.signal), { name: 'AbortError' });
    assert.equal(calls.length, before);
  } finally { globalThis.fetch = original; }
});
