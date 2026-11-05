import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createLetterStore } from '../src/features/letters/store.js';
import { letterApi } from '../src/features/letters/api.js';

const a = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const b = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';
const c = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc';
const letter = {
  id: b, kind: 'personalized', title: 'آگاهی از یک واکنش', salutation: null,
  paragraphs: ['گاهی پیش از پاسخ دادن مکث می‌کنیم.', 'در گفت‌وگویت این امکان را بررسی کردی.'],
  question: null, createdAt: '2026-11-04T00:00:00Z', sourceConversationIds: [c],
};

test('authenticated letter adapter uses owner-scoped JSON endpoints with explicit generation consent', async () => {
  const saved = globalThis.fetch;
  const calls = [];
  globalThis.fetch = async (url, init) => {
    calls.push({ url, init });
    if (init.method === 'POST') return { status: 200, ok: true, json: async () => letter };
    if (init.method === 'DELETE') return { status: 204, ok: true };
    return { status: 200, ok: true, json: async () => ({ items: [letter] }) };
  };
  try {
    assert.deepEqual(await letterApi.list(), [letter]);
    assert.equal((await letterApi.generate()).kind, 'personalized');
    await letterApi.remove(b);
    assert.deepEqual(calls.map(x => x.url), ['/api/v1/me/letters', '/api/v1/me/letters', `/api/v1/me/letters/${b}`]);
    assert.equal(calls[0].init.credentials, 'include');
    assert.deepEqual(JSON.parse(calls[1].init.body), { consent: true });
    assert.equal(calls[2].init.method, 'DELETE');
    await assert.rejects(letterApi.remove('not-a-uuid'), /Invalid letter id/);
  } finally { globalThis.fetch = saved; }
});

test('letter store creates, preserves, deletes owner-scoped letters and clears when switching identity', async () => {
  const store = createLetterStore({
    async list() { return []; }, async generate() { return letter; }, async remove() { return null; },
  });
  store.activate(a);
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(store.getState().phase, 'ready');
  assert.equal(await store.generate(), true);
  assert.equal(store.getState().letters[0].id, b);
  assert.equal(await store.remove(b), true);
  assert.equal(store.getState().letters.length, 0);
  store.activate(c);
  assert.equal(store.getState().letters.length, 0);
  store.reset();
  assert.equal(store.getState().userId, null);
});

test('generate never creates a fake letter on provider failure', async () => {
  const store = createLetterStore({
    async list() { return []; },
    async generate() { throw Object.assign(new Error('offline'), { code: 'LETTER_GENERATION_UNAVAILABLE', status: 503 }); },
  });
  store.activate(a);
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(await store.generate(), false);
  assert.equal(store.getState().letters.length, 0);
  assert.equal(store.getState().generationError, 'LETTER_GENERATION_UNAVAILABLE');
});

test('backend integration remains separate from removed diary UI', () => {
  const screen = readFileSync(new URL('../src/screens/Main.jsx', import.meta.url), 'utf8');
  assert.match(screen, /با رضایت من، نامه‌ای بنویس/);
  assert.match(screen, /حذف این نامه/);
  assert.doesNotMatch(screen, /daily-reflection-form/);
});
