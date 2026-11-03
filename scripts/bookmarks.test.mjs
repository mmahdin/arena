import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createConversationStore } from '../src/features/conversation/store.js';
import { conversationApi } from '../src/features/conversation/api.js';
import { ApiError } from '../src/core/api/http.js';

const A='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const B='bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';
const C='cccccccc-cccc-4ccc-8ccc-cccccccccccc';
const time='2026-11-02T10:00:00Z';
const message={ id:C, conversationId:A, role:'assistant', content:'یک متن ساختگی', createdAt:time, savedAt:time };
const make = (overrides={}) => {
  let saved=[];
  return {
    async bookmarks(){return saved;},
    async saveBookmark(id){saved=[{...message,id}];return {saved:true};},
    async removeBookmark(){saved=[];return null;},
    ...overrides,
  };
};

test('saved message toggle stores and removes only server-returned rows', async()=>{
  const store=createConversationStore(make());store.activate(A);
  assert.equal(await store.listBookmarks(),true);
  assert.equal(store.getState().bookmarkPhase,'ready');
  assert.equal(await store.toggleBookmark(message),true);
  assert.deepEqual(store.getState().bookmarks,[message]);
  assert.equal(await store.toggleBookmark(message),true);
  assert.deepEqual(store.getState().bookmarks,[]);
});

test('bookmark failure never creates a false saved result and remains retryable', async()=>{
  let fail=true;
  const store=createConversationStore(make({async saveBookmark(){if(fail) throw new ApiError(503,'DB_UNAVAILABLE');return {saved:true};}}));
  store.activate(A);await store.listBookmarks();
  assert.equal(await store.toggleBookmark(message),false);
  assert.deepEqual(store.getState().bookmarks,[]);
  assert.equal(store.getState().bookmarkError,'DB_UNAVAILABLE');
  fail=false;
  assert.equal(await store.listBookmarks(),true);
  assert.equal(await store.toggleBookmark(message),true);
});

test('bookmark list from old account cannot appear after tester switch',async()=>{
  let release;
  const wait = new Promise(resolve=>release=resolve);
  const store=createConversationStore(make({async bookmarks(){await wait;return [message];}}));
  store.activate(A);const pending=store.listBookmarks();
  store.activate(B);release();await pending;
  assert.equal(store.getState().userId,B);
  assert.deepEqual(store.getState().bookmarks,[]);
});

test('logout fences in-flight bookmark update',async()=>{
  let release;
  const wait = new Promise(resolve=>release=resolve);
  const store=createConversationStore(make({async saveBookmark(){await wait;return {saved:true};}}));
  store.activate(A);await store.listBookmarks();const pending=store.toggleBookmark(message);
  store.reset();release();await pending;
  assert.equal(store.getState().userId,null);
  assert.deepEqual(store.getState().bookmarks,[]);
});

test('401 when saving bookmarks clears identity-scoped state',async()=>{
  let unauth=0;
  const store=createConversationStore(make({async saveBookmark(){throw new ApiError(401,'SESSION_INVALID');}}),{onUnauthorized:()=>unauth++});
  store.activate(A);await store.listBookmarks();await store.toggleBookmark(message);
  assert.equal(unauth,1);assert.equal(store.getState().userId,null);
});

test('bookmark transport uses same-origin cookie requests and validates messages',async()=>{
  const seen=[];const before=globalThis.fetch;
  try {
    globalThis.fetch=async (path, opts)=>{
      seen.push([path,opts.method,opts.credentials]);
      if(opts.method==='DELETE')return new Response(null,{status:204});
      if(opts.method==='PUT')return new Response(JSON.stringify({saved:true}),{status:200});
      return new Response(JSON.stringify({items:[message]}),{status:200});
    };
    assert.deepEqual(await conversationApi.bookmarks(),[message]);
    assert.deepEqual(await conversationApi.saveBookmark(C),{saved:true});
    assert.equal(await conversationApi.removeBookmark(C),null);
    assert.deepEqual(seen,[['/api/v1/me/bookmarks','GET','include'],[`/api/v1/messages/${C}/bookmark`,'PUT','include'],[`/api/v1/messages/${C}/bookmark`,'DELETE','include']]);
    await assert.rejects(conversationApi.saveBookmark('invalid'),/Invalid message ID/);
  } finally { globalThis.fetch=before; }
});


test('deleting a conversation removes its saved-message row without waiting for a refresh', async()=>{
  const store=createConversationStore(make({
    async bookmarks(){return [message];},
    async remove(){return null;},
  }));
  store.activate(A);await store.listBookmarks();
  assert.equal(store.getState().bookmarks.length,1);
  assert.equal(await store.remove(message.conversationId),true);
  assert.deepEqual(store.getState().bookmarks,[]);
});

test('history refresh during a save cannot leave the bookmark action permanently busy', async()=>{
 let release; const wait=new Promise(resolve=>release=resolve);
 const store=createConversationStore(make({async saveBookmark(){await wait;}}));
 store.activate(A); await store.listBookmarks();
 const saving=store.toggleBookmark(message);
 await store.listBookmarks();
 release(); await saving;
 assert.equal(store.getState().bookmarkBusyId,null);
 assert.equal(await store.toggleBookmark(message),true);
});

test('a bookmark list begun before deletion cannot resurrect deleted saved entries', async()=>{
 let release; let reads=0; const wait=new Promise(resolve=>release=resolve);
 const store=createConversationStore(make({async bookmarks(){if(++reads>1)await wait;return [message];},async remove(){}}));
 store.activate(A);await store.listBookmarks();
 const listing=store.listBookmarks();await store.remove(message.conversationId);
 release();await listing;
 assert.deepEqual(store.getState().bookmarks,[]);
});
