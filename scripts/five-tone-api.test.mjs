import assert from 'node:assert/strict';
import { test } from 'node:test';
import { devSessionApi } from '../src/features/devsession/api.js';
import { APPROACH_TO_STYLE, approachForStyle, styleForApproach } from '../src/features/profile/preferences.js';

const userId = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const timestamp = '2026-11-02T00:00:00Z';
test('all five style values are accepted and submitted exactly without collapsing three into guidance', async () => {
  const original = globalThis.fetch;
  const observed = [];
  try {
    globalThis.fetch = async (path, request = {}) => {
      assert.equal(path, '/api/v1/me/profile');
      assert.equal(request.method, 'PATCH');
      assert.equal(request.credentials, 'include');
      const body = JSON.parse(request.body);
      observed.push(body.conversationStyle);
      return new Response(JSON.stringify({ userId, ...body, onboardingCompletedAt: timestamp }), { status: 200 });
    };
    for (const [approach, style] of Object.entries(APPROACH_TO_STYLE)) {
      const value = await devSessionApi.profile('Synthetic User', style);
      assert.equal(value.conversationStyle, style);
      assert.equal(approachForStyle(value.conversationStyle), approach);
    }
    assert.deepEqual(observed, Object.values(APPROACH_TO_STYLE));
    assert.equal(new Set(observed).size, 5);
  } finally { globalThis.fetch = original; }
});

test('old guidance profile response remains readable without losing the user', async () => {
  const original = globalThis.fetch;
  try {
    globalThis.fetch = async () => new Response(JSON.stringify({ userId, displayName: 'Synthetic User', conversationStyle: 'guidance', onboardingCompletedAt: timestamp }), { status: 200 });
    const profile = await devSessionApi.me();
    assert.equal(approachForStyle(profile.conversationStyle), 'راهنمایی آرام');
    assert.equal(styleForApproach(approachForStyle(profile.conversationStyle)), 'gentle_guidance');
  } finally { globalThis.fetch = original; }
});

test('unknown style response is rejected rather than silently resetting an existing preference', async () => {
  const original = globalThis.fetch;
  try {
    globalThis.fetch = async () => new Response(JSON.stringify({ userId, displayName: 'Synthetic User', conversationStyle: 'INJECT', onboardingCompletedAt: timestamp }), { status: 200 });
    await assert.rejects(devSessionApi.me(), /Invalid profile response/);
  } finally { globalThis.fetch = original; }
});
