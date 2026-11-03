/** Full cross-feature acceptance. All successful APIs are real Vite→Go→MSSQL→Mock AI. */
import assert from 'node:assert/strict';
import { randomInt } from 'node:crypto';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';
import { FLOW } from '../src/data.js';
import { ACTIVE_ONBOARDING_ROUTES } from '../src/features/profile/navigation.js';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';
const directory='test-results/hamrah-qa02';await mkdir(directory,{recursive:true});
const browser=await chromium.launch({headless:true,...(process.env.BROWSER_BIN?{executablePath:process.env.BROWSER_BIN}:{})});
let context,page;
const requestShapes=[];const errors=[];
const open=async()=>{
 context=await browser.newContext({viewport:{width:390,height:844},locale:'fa-IR',reducedMotion:'reduce'});
 page=await context.newPage();page.setDefaultTimeout(15000);
 page.on('pageerror',()=>errors.push('uncaught browser exception'));
 page.on('request',r=>{if(r.method()==='PUT'&&r.url().endsWith('/onboarding-progress'))requestShapes.push(Object.keys(r.postDataJSON()));});
 await page.goto('http://127.0.0.1:5173');await page.locator('#dev-phone').waitFor();
};
const call=(method,path,body)=>page.evaluate(async({method,path,body})=>{
 const r=await fetch(`/api/v1${path}`,{method,credentials:'include',...(body===undefined?{}:{headers:{'Content-Type':'application/json'},body:JSON.stringify(body)})});
 return {status:r.status,body:r.status===204?null:await r.json()};
},{method,path,body});
const enter=async phone=>{await page.locator('#dev-phone').fill(phone);await page.getByRole('button',{name:'ادامه',exact:true}).click();await page.locator('#dev-phone').waitFor({state:'hidden'});};
const route=async target=>{await page.evaluate(target=>{location.hash=`/${target}`;},target);};
const history=async()=>{await route('home');await page.locator('.home-menu-button').click();await page.getByRole('button',{name:'تاریخچه',exact:true}).click();await page.locator('.history-screen').waitFor();await page.getByText('در حال دریافت گفتگوها…').waitFor({state:'hidden'});};
const leave=async()=>{await route('home');await page.locator('.home-menu-button').click();await page.getByRole('button',{name:'خروج',exact:true}).click();await page.locator('#dev-phone').waitFor();};
const shots=new Set(['welcome','name','approach','ready']);
const seen=new Set();
async function inspect(step){
 seen.add(step);
 const index=FLOW.indexOf(step);
 for(const width of [360,390,430]){
  await page.setViewportSize({width,height:844});
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true,`${step} overflow ${width}`);
  if(shots.has(step))await page.screenshot({path:`${directory}/${step}-${width}.png`});
 }
 await page.setViewportSize({width:390,height:844});
 // Exercise each available Back control, then revisit the already-reached screen.
 const back=page.locator('main .app-screen .back-button');
 if(index>0&&await back.count()){
  await back.click();await page.waitForFunction(expected=>location.hash===`#/${expected}`,ACTIVE_ONBOARDING_ROUTES[ACTIVE_ONBOARDING_ROUTES.indexOf(step)-1]);
  await route(step);await page.waitForFunction(expected=>location.hash===`#/${expected}`,step);
 }
 const skip=page.locator('main .app-screen .skip-button');
 if(await skip.count()){
  await skip.click();await page.waitForFunction(expected=>location.hash===`#/${expected}`,ACTIVE_ONBOARDING_ROUTES[ACTIVE_ONBOARDING_ROUTES.indexOf(step)+1]);
  await route(step);await page.waitForFunction(expected=>location.hash===`#/${expected}`,step);
 }
 if(step==='name'){
  await page.locator('#name').fill('');
  assert.equal(await page.locator('.name-screen button[type="submit"]').isDisabled(),true);
 }
}
try{
 await open();
 const n=randomInt(1000,8998);const a=`0900000${n}`,b=`0900000${n+1}`;
 await enter(a);await page.locator('.welcome-screen').waitFor();
 const owner=(await call('GET','/me')).body.userId;
 assert.equal((await call('GET','/me/onboarding-progress')).body.furthestStep,0);
 assert.equal((await call('PUT','/me/onboarding-progress',{furthestStep:5})).status,409);
 assert.equal((await call('POST','/conversations')).status,409);
 await route('home');await page.waitForFunction(()=>location.hash==='#/welcome');
 const stop=new Error('intentional interruption');
 try{await finishFullOnboarding(page,'نام ساختگی',{onStep:async step=>{await inspect(step);if(FLOW.indexOf(step)===10)throw stop;}});}catch(error){if(error!==stop)throw error;}
 const cursor=(await call('GET','/me/onboarding-progress')).body.furthestStep;assert.equal(cursor,10);
 await page.reload();await page.waitForFunction(expected=>location.hash===`#/${expected}`,FLOW[cursor]);
 await call('DELETE','/dev/session');await context.close();
 await open();await enter(a);await page.waitForFunction(expected=>location.hash===`#/${expected}`,FLOW[cursor]);
 assert.equal((await call('GET','/me')).body.userId,owner);
 await finishFullOnboarding(page,'نام بازیابی‌شده',{onStep:inspect});
 assert.equal(seen.size,21,'all enabled pre-Home screens visited');
 for(const width of [360,390,430]){await page.setViewportSize({width,height:844});await page.screenshot({path:`${directory}/home-${width}.png`});}
 await page.setViewportSize({width:390,height:844});
 await page.locator('.chat-fab').click();
 for(let i=1;i<=2;i++){
  await page.getByRole('textbox',{name:'متن پیام'}).fill(`پیام ساختگی یکپارچه شماره ${i}`);
  await page.getByRole('button',{name:'فرستادن پیام'}).click();
  await page.locator('.message-assistant').nth(i-1).waitFor();await page.locator('.connected-pending').waitFor({state:'hidden'});
 }
 const id=(await call('GET','/conversations')).body.items[0].id;
 const messages=(await call('GET',`/conversations/${id}/messages`)).body.items;assert.equal(messages.length,4);
 const mid=messages[1].id;
 async function toggle(label){await page.locator('.message-assistant .message-text').first().click({button:'right'});await page.getByRole('menuitem',{name:label,exact:true}).click();await page.locator('.message-actions').waitFor({state:'hidden'});}
 await page.locator('.message-assistant .message-text').first().click({button:'right'});
 await page.screenshot({path:`${directory}/chat-actions-390.png`});
 await page.getByRole('menuitem',{name:'ذخیره پیام',exact:true}).click();await page.locator('.message-actions').waitFor({state:'hidden'});
 await history();await page.locator('.saved-history-entry').waitFor();
 await page.screenshot({path:`${directory}/history-saved-390.png`});
 await page.locator('.saved-history-entry').click();await page.locator('.message-assistant').nth(1).waitFor();
 await page.reload();await page.locator('.home-screen').waitFor();await history();await page.locator('.saved-history-entry').click();await page.locator('.message-assistant').nth(1).waitFor();
 await toggle('برداشتن از ذخیره‌شده‌ها');await history();await page.getByText('هنوز پیامی ذخیره نکرده‌ای.').waitFor();
 await page.locator('.history-row').click();await page.locator('.message-assistant').first().waitFor();await toggle('ذخیره پیام');
 await leave();await enter(a);await page.locator('.home-screen').waitFor();await history();await page.locator('.saved-history-entry').waitFor();
 await leave();await enter(b);await page.locator('.welcome-screen').waitFor();
 assert.equal((await call('GET','/me/onboarding-progress')).body.furthestStep,0);
 await finishFullOnboarding(page,'حساب ساختگی دوم');
 assert.equal((await call('GET',`/conversations/${id}/messages`)).status,404);
 assert.equal((await call('PUT',`/messages/${mid}/bookmark`)).status,404);
 assert.equal((await call('GET','/me/bookmarks')).body.items.length,0);
 await history();assert.equal(await page.locator('.saved-history-entry').count(),0);
 await leave();await enter(a);await page.locator('.home-screen').waitFor();
 await page.locator('.home-menu-button').click();await page.getByRole('button',{name:'ترجیحات',exact:true}).click();await page.locator('.approach-screen').waitFor();await page.screenshot({path:`${directory}/preferences-390.png`});
 await page.locator('.approach-screen .back-button').click();await route('home');
 await history();page.once('dialog',d=>d.accept());await page.getByRole('button',{name:/حذف/}).first().click();
 await page.locator('.saved-history-entry').waitFor({state:'hidden'});
 assert.equal((await call('GET','/me/bookmarks')).body.items.length,0);
 assert.equal((await call('GET',`/conversations/${id}/messages`)).status,404);
 assert.equal(await page.evaluate(()=>localStorage.length+sessionStorage.length),0);
 assert.ok(requestShapes.length>20);assert.ok(requestShapes.every(keys=>keys.length===1&&keys[0]==='furthestStep'));
 assert.deepEqual(errors,[]);await leave();
 console.log('PASS: 21-screen mobile audit, mid-flow refresh/new-browser re-entry, Ready name recovery, two turns, save/history/refresh/unsave/re-save/logout/relogin, B onboarding/isolation, A deletion/cascade; real APIs, no interception.');
}finally{await context?.close();await browser.close();}
