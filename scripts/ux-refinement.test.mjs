import assert from 'node:assert/strict';
import { test } from 'node:test';
import { readFileSync } from 'node:fs';
import { createConversationStore } from '../src/features/conversation/store.js';
import { createLetterStore } from '../src/features/letters/store.js';
import { LETTER_KIND, SAMPLE_LETTER, normalizeLetter, normalizeLetters } from '../src/features/letters/model.js';
import { APPROACHES, APPROACH_EXAMPLE_USER } from '../src/data.js';
import { ApiError } from '../src/core/api/http.js';

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8');
const A = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const B = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';
const C = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc';
const T = 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee';
const record = (id, title = 'گفت‌وگوی ساختگی') => ({ id, title, createdAt: '2026-11-04T10:00:00Z', updatedAt: '2026-11-04T10:00:00Z' });
const pair = [
  { id: A, role: 'user', content: 'پیام ساختگی', createdAt: '2026-11-04T10:00:00Z' },
  { id: B, role: 'assistant', content: 'پاسخ ساختگی', createdAt: '2026-11-04T10:00:01Z' },
];

function chatFixture(overrides = {}) {
  const calls = [];
  return {
    calls,
    async list() { calls.push('list'); return [record(C), record(B)]; },
    async create() { calls.push('create'); return record(C); },
    async messages(id) { calls.push(['messages', id]); return pair; },
    async turn(id, turnId, text) { calls.push(['turn', id, turnId, text]); return { reply: 'پاسخ ساختگی' }; },
    ...overrides,
  };
}

test('opening Chat restores the latest server conversation without creating a new one', async () => {
  const api = chatFixture();
  const store = createConversationStore(api);
  store.activate(A);
  assert.equal(await store.resumeLatest(), true);
  assert.equal(store.getState().conversationId, C);
  assert.deepEqual(store.getState().messages, pair);
  assert.ok(!api.calls.includes('create'));
  // An explicit new conversation is a real separate draft and is not auto-replaced.
  assert.equal(store.startNew(), true);
  assert.equal(store.getState().conversationId, null);
  assert.equal(await store.resumeLatest(), false);
  assert.equal(store.getState().conversationId, null);
});

test('conversation drafts stay scoped to each thread and survive navigation in memory', async () => {
  const store = createConversationStore(chatFixture());
  store.activate(A);
  store.startNew();
  store.setDraft('پیش‌نویس تازه');
  await store.open(B);
  assert.equal(store.getState().draft, '');
  store.setDraft('پیش‌نویس گفت‌وگوی B');
  await store.open(C);
  assert.equal(store.getState().draft, '');
  await store.open(B);
  assert.equal(store.getState().draft, 'پیش‌نویس گفت‌وگوی B');
  store.startNew();
  assert.equal(store.getState().draft, 'پیش‌نویس تازه');
  store.reset();
  assert.equal(store.getState().draft, '');
  assert.deepEqual(store.getState().drafts, {});
});

test('failed restore is retryable and never fabricates messages', async () => {
  let fail = true;
  const api = chatFixture({ async list() { if (fail) throw new ApiError(0, 'NETWORK_ERROR'); return [record(C)]; } });
  const store = createConversationStore(api);
  store.activate(A);
  assert.equal(await store.resumeLatest(), false);
  assert.equal(store.getState().messagesPhase, 'error');
  assert.deepEqual(store.getState().messages, []);
  fail = false;
  assert.equal(await store.resumeLatest(true), true);
  assert.equal(store.getState().conversationId, C);
  assert.deepEqual(store.getState().messages, pair);
});

test('sending clears the composer draft but keeps the exact pending turn on failure', async () => {
  const api = chatFixture({ async turn() { throw new ApiError(503, 'AI_UNAVAILABLE'); } });
  const store = createConversationStore(api, { newId: () => T });
  store.activate(A);
  store.setDraft('  پیام ساختگی  ');
  assert.equal(await store.send('پیام ساختگی'), false);
  assert.equal(store.getState().draft, '');
  assert.equal(store.getState().pending.clientTurnId, T);
  assert.equal(store.getState().pending.text, 'پیام ساختگی');
});

test('letter model requires provenance for personalized content and labels samples honestly', () => {
  assert.equal(SAMPLE_LETTER.kind, LETTER_KIND.SAMPLE);
  assert.deepEqual(SAMPLE_LETTER.sourceConversationIds, []);
  assert.throws(() => normalizeLetter({
    id: 'x', kind: 'personalized', title: 'عنوان', paragraphs: ['متن'], createdAt: '2026-11-04T00:00:00Z', sourceConversationIds: [],
  }), /provenance/);
  const personal = normalizeLetter({
    id: 'x', kind: 'personalized', title: 'عنوان', paragraphs: ['متن'], createdAt: '2026-11-04T00:00:00Z', sourceConversationIds: [C],
  });
  assert.deepEqual(personal.sourceConversationIds, [C]);
  assert.equal(normalizeLetters([personal]).length, 1);
});

test('letter store is unavailable without an adapter and supports future loading/ready/error states', async () => {
  const unavailable = createLetterStore(null);
  unavailable.activate(A);
  assert.equal(unavailable.getState().phase, 'unavailable');
  assert.equal(await unavailable.load(), false);

  const personal = normalizeLetter({
    id: 'personal-1', kind: 'personalized', title: 'عنوان', paragraphs: ['متن'], createdAt: '2026-11-04T00:00:00Z', sourceConversationIds: [C],
  });
  let finish;
  const waiting = new Promise((resolve) => { finish = resolve; });
  const ready = createLetterStore({ async list() { await waiting; return [personal]; } });
  ready.activate(A);
  assert.equal(ready.getState().phase, 'loading');
  finish();
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(ready.getState().phase, 'ready');
  assert.equal(ready.getState().letters.length, 1);

  const sampleSource = createLetterStore({ async list() { return [SAMPLE_LETTER]; } });
  sampleSource.activate(A);
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(sampleSource.getState().phase, 'error', 'backend source must not feed demo samples into personal letters');

  const failing = createLetterStore({ async list() { throw new ApiError(503, 'SERVICE_UNAVAILABLE'); } });
  failing.activate(A);
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(failing.getState().phase, 'error');
  assert.equal(failing.getState().error, 'SERVICE_UNAVAILABLE');
});

test('frontend refinement removes diary/fake controls and keeps backend boundaries', () => {
  const app = read('../src/App.jsx');
  const main = read('../src/screens/Main.jsx');
  const onboarding = read('../src/screens/Onboarding.jsx');
  assert.ok(!app.includes("if (target === 'chat' && route !== 'history' && route !== 'chat') chatStore.startNew();"));
  assert.ok(app.includes('chatStore.resumeLatest()'));
  assert.ok(/createLetterStore\(letterApi/.test(app));
  assert.ok(!main.includes('daily-reflection-form'));
  assert.ok(!main.includes('ساخت دوباره'));
  assert.ok(!main.includes('گزارش پیام'));
  assert.ok(!main.includes('پیام صوتی'));
  assert.ok(main.includes('نمونهٔ نمایشی'));
  assert.ok(onboarding.includes('پخش صوتی نداریم'));
  assert.ok(onboarding.includes('APPROACH_EXAMPLE_USER'));
  assert.ok(onboarding.includes("'--tick-index'"));
  assert.ok(main.includes('type="radio"'));
});

test('style preview keeps the five original options and one shared user prompt', () => {
  assert.deepEqual(APPROACHES.map((item) => item.name), [
    'گفت‌وگوی آزاد', 'همدلی و شنیدن', 'به من یاد بده', 'راهنمایی آرام', 'حل مسئله با هم',
  ]);
  assert.ok(APPROACH_EXAMPLE_USER.trim().length > 0);
  for (const approach of APPROACHES) {
    assert.ok(approach.exampleResponse.trim().length > 0, approach.name);
  }
  assert.equal(new Set(APPROACHES.map((item) => item.exampleResponse)).size, 5);
});
