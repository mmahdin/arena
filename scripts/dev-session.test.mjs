import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createSessionStore } from '../src/features/devsession/sessionStore.js';
import { normalizeTestPhone } from '../src/features/devsession/phone.js';
import { ApiError, request } from '../src/core/api/http.js';

const A = { userId:'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', displayName:'آزمایش A', conversationStyle:'balanced', onboardingCompletedAt:'2026-11-01T00:00:00Z' };
const B = { userId:'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', displayName:'', conversationStyle:'balanced', onboardingCompletedAt:null };
function apiFixture() {
  let current = null;
  const revoked=[];
  const byPhone = new Map([['09000000001', A], ['09000000002', B]]);
  return {
    revoked,
    async me() { if (!current) throw new ApiError(401,'SESSION_INVALID'); return current; },
    async start(phone) { current = byPhone.get(phone); if (!current) throw new ApiError(400,'INVALID_DEV_PHONE'); return {ok:true,created:false}; },
    async end() { if (current) revoked.push(current.userId); current=null; },
    async profile(displayName,conversationStyle) { current = {...current,displayName,conversationStyle,onboardingCompletedAt:current.onboardingCompletedAt ?? new Date().toISOString()}; return current; },
  };
}
const run = async (store,method,...args) => { await store[method](...args); return store.getState(); };
test('synthetic mobile normalization ASCII Persian Arabic and invalid input', () => {
  for (const phone of ['09000000001','۰۹۰۰۰۰۰۰۰۰۱','٠٩٠٠٠٠٠٠٠٠١',' 09000000001 ']) assert.equal(normalizeTestPhone(phone),'09000000001');
  for (const phone of ['09123456789','۰۹۱۲۳۴۵۶۷۸۹','٠٩١٢٣٤٥٦٧٨٩']) assert.equal(normalizeTestPhone(phone),'09123456789');
  for (const phone of ['', '0900000000','090000000011','0900000-001','0900000a001']) assert.equal(normalizeTestPhone(phone),null);
});
test('fresh session, returning session and A -> B identity switch clear scoped state', async () => {
  const api=apiFixture();let cleared=0;
  const store=createSessionStore(api,{onInvalidate:()=>{cleared++;}});
  assert.equal((await run(store,'bootstrap')).phase,'anonymous');
  assert.equal((await run(store,'login','09000000001')).profile.userId,A.userId);
  assert.equal((await run(store,'bootstrap')).profile.userId,A.userId);
  assert.equal((await run(store,'leave')).phase,'anonymous');
  assert.equal((await run(store,'login','09000000002')).profile.userId,B.userId);
  assert.equal(store.getState().profile.onboardingCompletedAt,null);
  assert.equal(api.revoked.length,1);assert.equal(cleared,3);
});
test('minimal onboarding is persisted and failed save cannot claim completion', async () => {
  const api=apiFixture();const store=createSessionStore(api);
  await store.login('09000000002');
  assert.equal(await store.saveProfile('نام آزمایشی','listening'),true);
  assert.equal(store.getState().profile.conversationStyle,'listening');
  assert.ok(store.getState().profile.onboardingCompletedAt);
  const failing=createSessionStore({...api, profile:async()=>{throw new ApiError(400,'VALIDATION_ERROR');}});
  await failing.bootstrap();assert.equal(await failing.saveProfile('x','balanced'),false);
  assert.equal(failing.getState().phase,'authenticated');
});
test('missing or expired cookie is anonymous; database failure has recoverable error', async()=>{
  const anonymous=createSessionStore({me:async()=>{throw new ApiError(401,'SESSION_INVALID');}});
  assert.equal((await run(anonymous,'bootstrap')).phase,'anonymous');
  const offline=createSessionStore({me:async()=>{throw new ApiError(503,'SESSION_UNAVAILABLE');}});
  assert.equal((await run(offline,'bootstrap')).phase,'error');
});
test('logout failure does not appear successful and never retains stale profile UI',async()=>{
  const api=apiFixture();const store=createSessionStore({...api,end:async()=>{throw new ApiError(503,'DB_UNAVAILABLE');}});
  await store.login('09000000001');await store.leave();
  assert.equal(store.getState().phase,'error');assert.equal(store.getState().profile,null);
  assert.equal(store.getState().error.action,'logout');
});
test('late old login cannot overwrite a newer synthetic user',async()=>{
  let release;
  const wait=new Promise((r)=>{release=r;});
  const api=apiFixture();const store=createSessionStore({...api,
    start:async(phone)=>{if(phone==='09000000001')await wait;},me:async()=>B});
  const first=store.login('09000000001');
  await store.login('09000000002');release();await first;
  assert.equal(store.getState().profile.userId,B.userId);
});
test('HTTP API uses same-origin cookies and sanitizes Problem responses',async()=>{
  const original=globalThis.fetch;let observed;
  try {
    globalThis.fetch=async(url,init)=>{observed={url,...init};return new Response(JSON.stringify({code:'INVALID_DEV_PHONE',detail:'secret'}),{status:400});};
    await assert.rejects(request('/api/v1/dev/identity-session',{method:'POST',body:{phone:'09000000001'}}),(err)=>err.status===400 && err.code==='INVALID_DEV_PHONE' && !err.message.includes('secret'));
    assert.equal(observed.credentials,'include');assert.equal(observed.url,'/api/v1/dev/identity-session');
    assert.deepEqual(JSON.parse(observed.body),{phone:'09000000001'});
    await assert.rejects(request('https://invalid.example'),/Invalid API path/);
  } finally {globalThis.fetch=original;}
});
