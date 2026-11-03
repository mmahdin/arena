import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createPrivacyPreferencesStore, parsePrivacyPreferences, privacyPreferencesApi } from '../src/features/profile/privacyPreferences.js';

const A = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const B = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';

test('default is on, persistence survives reload and is scoped per user', async () => {
  const values = new Map();
  let currentUser = A;
  const api = {
    async read() { return values.get(currentUser) ?? true; },
    async save(enabled) { values.set(currentUser, enabled); return enabled; },
  };
  const store = createPrivacyPreferencesStore(api);
  store.activate(A);
  assert.equal(store.getState().enabled, true);
  await store.load();
  assert.equal(store.getState().loaded, true);
  assert.equal(await store.toggle(), true);
  assert.equal(store.getState().enabled, false);
  const later = createPrivacyPreferencesStore(api);
  later.activate(A);
  await later.load();
  assert.equal(later.getState().enabled, false);
  currentUser = B;
  later.activate(B);
  await later.load();
  assert.equal(later.getState().enabled, true);
  currentUser = A;
  later.activate(A);
  await later.load();
  assert.equal(later.getState().enabled, false);
});

test('failure retains known value and retry succeeds', async () => {
  let failed = false;
  const store = createPrivacyPreferencesStore({
    async read() { return true; },
    async save(enabled) { if (!failed) { failed=true; throw new Error('offline'); } return enabled; },
  });
  store.activate(A);
  await store.load();
  assert.equal(await store.toggle(), false);
  assert.equal(store.getState().enabled, true);
  assert.equal(store.getState().error, true);
  assert.equal(await store.toggle(), true);
  assert.equal(store.getState().enabled, false);
});

test('outdated in-flight preference response never crosses user scope', async () => {
  let release;
  const waiting = new Promise(resolve => { release = resolve; });
  const store = createPrivacyPreferencesStore({
    async read() { await waiting; return false; },
  });
  store.activate(A);
  const request = store.load();
  store.activate(B);
  release();
  await request;
  assert.equal(store.getState().enabled, true);
  assert.equal(store.getState().loaded, false);
  store.reset();
  assert.equal(store.getState().enabled, true);
});

test('strict API response and cookie-backed same-origin transport', async () => {
  assert.equal(parsePrivacyPreferences({conversationSharingEnabled:false}),false);
  assert.throws(() => parsePrivacyPreferences({conversationSharingEnabled:'false'}));
  const previous = globalThis.fetch;
  try {
    let received;
    globalThis.fetch = async (url, options) => {
      received = {url,...options};
      return new Response(JSON.stringify({conversationSharingEnabled:false}),{status:200});
    };
    assert.equal(await privacyPreferencesApi.save(false),false);
    assert.equal(received.url,'/api/v1/me/privacy-preferences');
    assert.equal(received.credentials,'include');
    assert.equal(received.method,'PUT');
    assert.deepEqual(JSON.parse(received.body),{conversationSharingEnabled:false});
  } finally { globalThis.fetch=previous; }
});
