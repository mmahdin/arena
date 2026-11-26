import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createConversationStore } from '../src/features/conversation/store.js';

const read = path => readFileSync(new URL(path, import.meta.url), 'utf8');
const id1 = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const id2 = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';
const record = (id, title) => ({id, title, createdAt:'2026-11-02T10:00:00Z', updatedAt:'2026-11-02T10:00:00Z'});
const tick = () => new Promise(resolve => setImmediate(resolve));

test('rename persists server response and stays after history reload', async () => {
  const rows = [record(id1,'گفت‌وگوی جدید')];
  const api = {
    list: async () => rows.map(x => ({...x})),
    rename: async (id,title) => { const row = rows.find(x => x.id === id); row.title = title; return {...row}; },
  };
  const store = createConversationStore(api);
  store.activate('A');
  await store.list();
  assert.equal((await store.rename(id1,'برنامه آخر هفته')),true);
  assert.equal(store.getState().conversations[0].title,'برنامه آخر هفته');
  await store.list();
  assert.equal(store.getState().conversations[0].title,'برنامه آخر هفته');
});

test('rename response from a previous user cannot populate the next user history', async () => {
  let done;
  const api = { rename: () => new Promise(resolve => { done = resolve; }), list: async () => [record(id2,'حساب ب')] };
  const store = createConversationStore(api);
  store.activate('A');
  // Existing conversation is required to start an edit.
  const originalList = api.list;
  api.list = async () => [record(id1,'عنوان اولیه')];
  await store.list();
  api.list = originalList;
  const pending = store.rename(id1,'عنوان حساب الف');
  await tick();
  store.activate('B');
  await store.list();
  done(record(id1,'عنوان حساب الف'));
  assert.equal(await pending,false);
  assert.deepEqual(store.getState().conversations,[record(id2,'حساب ب')]);
});

test('invalid names do not invoke the API', async () => {
  let calls = 0;
  const store = createConversationStore({rename: () => {calls++; return Promise.resolve({});}});
  store.activate('A');
  assert.equal(await store.rename(id1,' '),false);
  assert.equal(await store.rename(id1,'x'.repeat(61)),false);
  assert.equal(calls,0);
});

test('history screen retains original navigation and offers inline rename controls', () => {
  const source = read('../src/screens/Main.jsx');
  assert.match(source,/onOpenConversation\(item\.id\)/);
  assert.match(source,/onDeleteConversation\(item\.id\)/);
  assert.match(source,/onRenameConversation/);
  assert.match(source,/تغییر نام/);
  assert.match(source,/ذخیره نام/);
});

test('rename API uses PATCH, same-origin cookies and verifies server reply', async () => {
  const { conversationApi } = await import('../src/features/conversation/api.js');
  const saved = globalThis.fetch;
  const observed = [];
  try {
    globalThis.fetch = async (path, options) => {
      observed.push({path,options});
      return new Response(JSON.stringify(record(id1,'عنوان دلخواه')),{status:200,headers:{'Content-Type':'application/json'}});
    };
    const updated = await conversationApi.rename(id1,' عنوان دلخواه ');
    assert.equal(updated.title,'عنوان دلخواه');
    assert.equal(observed[0].path,`/api/v1/conversations/${id1}`);
    assert.equal(observed[0].options.method,'PATCH');
    assert.equal(observed[0].options.credentials,'include');
    assert.deepEqual(JSON.parse(observed[0].options.body),{title:'عنوان دلخواه'});
    await assert.rejects(conversationApi.rename(id1,''),/Invalid conversation title/);
    assert.equal(observed.length,1);
  } finally { globalThis.fetch = saved; }
});
