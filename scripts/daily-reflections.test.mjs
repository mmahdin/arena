import assert from 'node:assert/strict';
import { test } from 'node:test';
import { reflectionApi, parseReflection, validReflectionDate, localReflectionDate } from '../src/features/reflection/api.js';
import { createReflectionStore } from '../src/features/reflection/store.js';

const A='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const B='bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';
const day='2026-11-02';
const now='2026-11-02T09:00:00Z';
const entry=(date=day,text='یک متن ساختگی')=>({date,text,createdAt:now,updatedAt:now});
const fixture=()=>{
  const values=new Map();
  let user=A;
  const api={
    async list(){return [...(values.get(user)?.values() || [])]},
    async read(date){const value=values.get(user)?.get(date); if(!value)throw {status:404,code:'RESOURCE_NOT_FOUND'};return value;},
    async save(date,text){if(!values.has(user))values.set(user,new Map());const value=entry(date,text);values.get(user).set(date,value);return value;},
    async remove(date){values.get(user)?.delete(date)},
  };
  return {api,setUser:(next)=>{user=next;},values};
};

test('strict calendar dates, local calendar date, response validation',()=>{
  assert.equal(validReflectionDate('2026-02-28'),true);
  for(const value of ['2026-02-30','2026-13-01','2026-1-02','today','2026-01-01/abc']) assert.equal(validReflectionDate(value),false);
  assert.equal(localReflectionDate(new Date(2026,10,2,23,30)),day);
  assert.deepEqual(parseReflection(entry()),entry());
  assert.throws(()=>parseReflection({date:day,text:3}));
});

test('save, load, update, delete, refresh via API and switch isolates user data',async()=>{
  const {api,setUser}=fixture();
  const s=createReflectionStore(api);
  s.activate(A);
  await s.open(day);assert.equal(s.getState().entry,null);
  assert.equal(await s.save(day,'یک متن ساختگی'),true);
  assert.equal(await s.save(day,'نسخه دوم'),true);
  assert.equal(s.getState().entry.text,'نسخه دوم');
  await s.list();assert.equal(s.getState().list.length,1);
  const later=createReflectionStore(api);later.activate(A);await later.open(day);
  assert.equal(later.getState().entry.text,'نسخه دوم');
  setUser(B);later.activate(B);await later.list();await later.open(day);
  assert.equal(later.getState().entry,null);assert.equal(later.getState().list.length,0);
  setUser(A);later.activate(A);await later.open(day);
  assert.equal(later.getState().entry.text,'نسخه دوم');
  assert.equal(await later.remove(day),true);
  assert.equal(await later.remove(day),true);
  await later.list();assert.equal(later.getState().list.length,0);
});

test('failed save retains drafted text and supports same-text retry',async()=>{
  let attempts=0;
  const s=createReflectionStore({
    async read(){throw {status:404}},
    async save(date,text){attempts++;if(attempts===1)throw {status:503,code:'DB_UNAVAILABLE'};return entry(date,text)},
  });
  s.activate(A);await s.open(day);
  assert.equal(await s.save(day,'چیزی که نوشتم'),false);
  assert.equal(s.getState().dayError,'DB_UNAVAILABLE');
  assert.equal(s.getState().dayPhase,'ready');
  assert.equal(await s.save(day,'چیزی که نوشتم'),true);
  assert.equal(s.getState().entry.text,'چیزی که نوشتم');
});

test('late read/save/list response cannot contaminate a new identity',async()=>{
  let resolveRead, resolveList, resolveSave;
  const s=createReflectionStore({
    async read(){return new Promise(resolve=>resolveRead=resolve)},
    async list(){return new Promise(resolve=>resolveList=resolve)},
    async save(){return new Promise(resolve=>resolveSave=resolve)},
  });
  s.activate(A);
  const read=s.open(day),list=s.list();
  s.activate(B);
  resolveRead(entry(day,'private'));resolveList([entry(day,'private')]);
  await Promise.all([read,list]);
  assert.equal(s.getState().userId,B);assert.equal(s.getState().entry,null);assert.deepEqual(s.getState().list,[]);
  const newRead=s.open(day);resolveRead(entry());await newRead;
  const save=s.save(day,'synthetic');s.reset();resolveSave(entry());await save;
  assert.equal(s.getState().userId,null);assert.equal(s.getState().entry,null);
});

test('invalid body is blocked before calling API',async()=>{
  let called=0;
  const s=createReflectionStore({async read(){throw {status:404}},async save(){called++}});
  s.activate(A);await s.open(day);
  assert.equal(await s.save(day,'  '),false);
  assert.equal(await s.save(day,'x'.repeat(4001)),false);
  assert.equal(await s.save(day,'hello\u202E'),false);
  assert.equal(await s.save(day,'hello\u0085'),false);
  assert.equal(await s.save(day,'hello\u200B'),false);
  assert.equal(called,0);
});

test('Persian half-spaces and joiners are accepted', async()=>{
  const {api}=fixture();
  const s=createReflectionStore(api);
  s.activate(A); await s.open(day);
  assert.equal(await s.save(day,'نسخهٔ ویرایش‌شده با نیم\u200cفاصله و پیوند\u200dدهنده'),true);
});

test('same-origin HTTP client uses cookies, PUT/DELETE, does not persist browser tokens',async()=>{
  const original=globalThis.fetch;
  const calls=[];
  try {
    globalThis.fetch=async (url,options)=>{
      calls.push({url,...options});
      if(options.method==='DELETE')return new Response(null,{status:204});
      if(options.method==='PUT')return new Response(JSON.stringify(entry()),{status:200});
      if(url.endsWith(day))return new Response(JSON.stringify(entry()),{status:200});
      return new Response(JSON.stringify({items:[entry()]}),{status:200});
    };
    assert.equal((await reflectionApi.list()).length,1);
    assert.equal((await reflectionApi.read(day)).date,day);
    assert.equal((await reflectionApi.save(day,'یک متن ساختگی')).text,'یک متن ساختگی');
    await reflectionApi.remove(day);
    assert.equal(calls.length,4);
    for(const call of calls){assert.equal(call.credentials,'include');assert.match(call.url,/^\/api\/v1\/me\/daily-reflections/);}
    assert.equal(calls[2].method,'PUT');assert.deepEqual(JSON.parse(calls[2].body),{text:'یک متن ساختگی'});
    assert.equal(calls[3].method,'DELETE');
    await assert.rejects(reflectionApi.read('2026-02-30'),/Invalid reflection date/);
  } finally {globalThis.fetch=original;}
});

test('late list cannot replace a newly saved reflection',async()=>{
  let finishList;
  const s=createReflectionStore({
    async list(){return new Promise(resolve=>finishList=resolve)},
    async read(){throw {status:404}},
    async save(date,text){return entry(date,text)},
  });
  s.activate(A);
  const pendingList=s.list();await s.open(day);
  assert.equal(await s.save(day,'newest'),true);
  finishList([]);await pendingList;
  assert.equal(s.getState().list.length,1);
  assert.equal(s.getState().list[0].text,'newest');
  assert.equal(s.getState().listPhase,'ready');
});

test('unknown date and old day selection remain read-only without mutation',async()=>{
  const s=createReflectionStore({async read(){throw {status:404}}});
  s.activate(A);assert.equal(await s.open(day),true);
  assert.equal(s.getState().dayPhase,'ready');
  assert.equal(s.getState().entry,null);
  assert.equal(await s.open('2026-02-30'),false);
  assert.equal(s.getState().openedDate,day);
});
