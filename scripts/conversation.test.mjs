import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createConversationStore } from '../src/features/conversation/store.js';
import { conversationApi } from '../src/features/conversation/api.js';
import { ApiError } from '../src/core/api/http.js';

const A = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const B = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';
const C = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc';
const T = 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee';
const entry = { id: C, title: 'گفتگوی آزمایشی', createdAt: '2026-11-02T10:00:00Z', updatedAt: '2026-11-02T10:00:00Z' };
const pair = [{ id: A, role: 'user', content: 'پیام ساختگی', createdAt: entry.createdAt }, { id: B, role: 'assistant', content: 'پاسخ تستی', createdAt: entry.updatedAt }];
function fixture(overrides = {}) {
  const calls = [];
  let stored = [];
  return {
    calls,
    async list() { calls.push('list'); return stored.length ? [entry] : []; },
    async create() { calls.push('create'); return entry; },
    async messages(id) { calls.push(['messages', id]); return stored; },
    async turn(id, turnId, text) { calls.push(['turn', id, turnId, text]); stored = pair; return { reply: 'پاسخ تستی' }; },
    async remove(id) { calls.push(['remove', id]); stored = []; },
    ...overrides,
  };
}

test('first send creates a conversation, stores a real pair and lists history', async () => {
  const api = fixture(); const store = createConversationStore(api, {newId: () => T});
  store.activate(A);
  assert.equal(store.startNew(), true);
  assert.equal(await store.send('  پیام ساختگی  '), true);
  assert.equal(store.getState().conversationId, C);
  assert.deepEqual(store.getState().messages, pair);
  assert.equal(store.getState().pending, null);
  assert.ok(api.calls.some((call) => Array.isArray(call) && call[2] === T && call[3] === 'پیام ساختگی'));
  await store.list();
  assert.equal(store.getState().conversations.length, 1);
  assert.equal(await store.open(C), true);
  assert.deepEqual(store.getState().messages, pair);
});

test('network/provider failure keeps exact turn ID and content for retry; no fabricated reply', async () => {
  let fail = true;
  const api = fixture({
    async turn(id, turnId, text) { api.calls.push(['turn', id, turnId, text]); if (fail) throw new ApiError(503,'AI_UNAVAILABLE'); return {reply:'پاسخ تستی'}; },
    async messages() { return pair; },
  });
  const store = createConversationStore(api,{newId:()=>T}); store.activate(A);
  assert.equal(await store.send('پیام ساختگی'), false);
  assert.equal(store.getState().messages.length, 0);
  assert.equal(store.getState().pending.clientTurnId, T);
  assert.equal(store.getState().messageError, 'AI_UNAVAILABLE');
  assert.equal(await store.send('پیام دوم'), false);
  fail = false;
  assert.equal(await store.retry(), true);
  assert.equal(store.getState().pending, null);
  assert.equal(store.getState().messages.length, 2);
  assert.deepEqual(api.calls.filter((c)=>Array.isArray(c) && c[0] === 'turn').map((c) => c.slice(1)), [[C,T,'پیام ساختگی'],[C,T,'پیام ساختگی']]);
});

test('same ID replay after lost response reconciles the server messages, not another optimistic pair', async () => {
  let tries = 0;
  const api = fixture({
    async turn() { tries++; if (tries === 1) throw new ApiError(0,'NETWORK_ERROR'); return {reply:'replayed'}; },
    async messages() { return pair; },
  });
  const store = createConversationStore(api,{newId:()=>T}); store.activate(A);
  await store.send('پیام ساختگی'); assert.equal(store.getState().messages.length,0);
  assert.equal(await store.retry(),true);
  assert.equal(store.getState().messages.length,2);
  assert.equal(tries,2);
});

test('409 pending reservation is retryable with unchanged clientTurnId', async () => {
  let calls=0;
  const api=fixture({async turn(){calls++; if(calls===1) throw new ApiError(409,'TURN_ALREADY_PROCESSING');},async messages(){return pair;}});
  const store=createConversationStore(api,{newId:()=>T});store.activate(A);
  await store.send('پیام ساختگی');assert.equal(store.getState().messageError,'TURN_ALREADY_PROCESSING');
  await store.retry();assert.equal(calls,2);assert.equal(store.getState().pending,null);
});

test('switching user aborts stale messages and prevents cross-user history',async()=>{
  let finish;
  const old=new Promise((resolve)=>{finish=resolve;});
  const api=fixture({async messages(){await old;return pair;},async list(){return [entry];}});
  const store=createConversationStore(api);store.activate(A);
  const opening=store.open(C);
  store.activate(B);
  finish();await opening;
  assert.equal(store.getState().userId,B);
  assert.equal(store.getState().conversationId,null);
  assert.deepEqual(store.getState().messages,[]);
  assert.deepEqual(store.getState().conversations,[]);
});

test('logout during in-flight turn drops old completion and no stale messages survive',async()=>{
  let finish;
  const old=new Promise((resolve)=>{finish=resolve;});
  const api=fixture({async turn(){await old;}, async messages(){return pair;}});
  const store=createConversationStore(api,{newId:()=>T});store.activate(A);
  const send=store.send('پیام ساختگی');
  await new Promise((resolve)=>setImmediate(resolve));
  store.reset();finish();await send;
  assert.equal(store.getState().userId,null);
  assert.equal(store.getState().messages.length,0);
  assert.equal(store.getState().pending,null);
});

test('401 invalidates scoped state and requests a session refresh',async()=>{
  let calls=0;
  const store=createConversationStore(fixture({async list(){throw new ApiError(401,'SESSION_INVALID');}}),{onUnauthorized:()=>calls++});
  store.activate(A);await store.list();
  assert.equal(store.getState().userId,null);assert.equal(calls,1);
});

test('deletion and 404 are safe; no foreign conversation content is displayed',async()=>{
  const api=fixture({async messages(){throw new ApiError(404,'RESOURCE_NOT_FOUND');}});
  const store=createConversationStore(api);store.activate(A);
  assert.equal(await store.open(C),false);
  assert.equal(store.getState().messageError,'RESOURCE_NOT_FOUND');
  await store.list();
  assert.equal(await store.remove(C),true);
  assert.deepEqual(store.getState().conversations,[]);
});

test('conversation HTTP client uses same-origin, cookies and exact JSON contract',async()=>{
  const old=globalThis.fetch; const seen=[];
  try {
    globalThis.fetch=async(path,opts)=>{
      seen.push({path,...opts});
      if(path.endsWith('/turns'))return new Response(JSON.stringify({userMessageId:A,assistantMessageId:B,reply:'پاسخ تستی',promptVersion:'supportive-v3'}),{status:200});
      if(opts.method==='DELETE')return new Response(null,{status:204});
      if(opts.method==='POST')return new Response(JSON.stringify(entry),{status:201});
      return new Response(JSON.stringify({items:[]}),{status:200});
    };
    await conversationApi.list();await conversationApi.create();await conversationApi.messages(C);
    await conversationApi.turn(C,T,'پیام ساختگی');await conversationApi.remove(C);
    assert.deepEqual(seen.map((x)=>x.path),['/api/v1/conversations','/api/v1/conversations',`/api/v1/conversations/${C}/messages`,`/api/v1/conversations/${C}/turns`,`/api/v1/conversations/${C}`]);
    assert.equal(seen.every(x=>x.credentials==='include'),true);
    assert.deepEqual(JSON.parse(seen[3].body),{clientTurnId:T,message:'پیام ساختگی'});
    await assert.rejects(conversationApi.messages('bad-id'),/Invalid conversation ID/);
  } finally {globalThis.fetch=old;}
});

test('already deleted active conversation clears messages and fences an outstanding open', async () => {
  let finish;
  const waiting = new Promise(resolve => { finish = resolve; });
  const store = createConversationStore(fixture({
    async messages() { await waiting; return pair; },
    async remove() { throw new ApiError(404, 'RESOURCE_NOT_FOUND'); },
  }));
  store.activate(A);
  const opening = store.open(C);
  await store.remove(C);
  finish(); await opening;
  assert.equal(store.getState().conversationId, null);
  assert.deepEqual(store.getState().messages, []);
});

test('real backend 404 is terminal for turn retry and normalized for the UI', async () => {
  let attempts = 0;
  const store = createConversationStore(fixture({ async turn() { attempts++; throw new ApiError(404, 'RESOURCE_NOT_FOUND'); } }), { newId: () => T });
  store.activate(A);
  await store.send('پیام ساختگی');
  assert.equal(store.getState().messageError, 'RESOURCE_NOT_FOUND');
  assert.equal(await store.retry(), false);
  assert.equal(attempts, 1);
  assert.equal(store.startNew(), true);
  assert.equal(store.getState().pending, null);
});
