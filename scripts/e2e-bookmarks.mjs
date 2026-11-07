/** Browser → Vite proxy → actual Go API → SQL Server; no network intercepts. Synthetic-only test. */
import assert from 'node:assert/strict';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';
let chromium;
try { ({chromium} = await import('playwright')); }
catch { console.error('BLOCKED: install Playwright to run this opt-in browser test.'); process.exit(2); }
const ui='http://127.0.0.1:5173', api='http://127.0.0.1:8080';
for(const url of [ui,`${api}/health/ready`]) {
 try { assert.equal((await fetch(url)).status,200); }
 catch { console.error('BLOCKED: start Vite and Go/MSSQL services.'); process.exit(2); }
}
const browser=await chromium.launch({headless:true,...(process.env.BROWSER_BIN?{executablePath:process.env.BROWSER_BIN}:{})});
try {
 const context=await browser.newContext({viewport:{width:390,height:844},locale:'fa-IR'});
 const page=await context.newPage();
 const call=async(path,method='GET')=>page.evaluate(async({path,method})=>{
  const r=await fetch(path,{method,credentials:'include'});
  return {status:r.status,body:r.status===204?null:await r.json().catch(()=>null)};
 },{path,method});
 async function login(phone){
  await page.locator('#dev-phone').fill(phone);
  await page.getByRole('button',{name:'ادامه'}).last().click();
  await page.locator('#dev-phone').waitFor({state:'hidden'});
  await finishFullOnboarding(page,'کاربر ساختگی');
  await page.locator('.home-screen').waitFor();
 }
 async function history(){
  await page.locator('.home-screen .home-menu-button').click();
  await page.getByRole('button',{name:'تاریخچه'}).click();
  await page.locator('.history-screen').waitFor();
 }
 await page.goto(ui);await page.locator('#dev-phone').waitFor();
 await login('09000000761');
 await page.locator('.chat-fab').click();
 await page.getByRole('textbox',{name:'متن پیام'}).fill('یک پیام ساختگی برای ذخیره کردن');
 await page.getByRole('button',{name:'فرستادن پیام'}).click();
 await page.locator('.chat-screen .message-assistant').first().waitFor({timeout:40000});
 const convs=await call('/api/v1/conversations');assert.equal(convs.status,200);
 const id=convs.body.items[0].id;
 const messages=await call(`/api/v1/conversations/${id}/messages`);
 assert.equal(messages.body.items.length,2);const savedId=messages.body.items[1].id;
 await page.locator('.message-assistant').first().hover();
 await page.getByRole('button',{name:'ذخیره پیام'}).first().click();
 const saved=await call('/api/v1/me/bookmarks');assert.equal(saved.status,200);
 assert.ok(saved.body.items.some(item=>item.id===savedId));
 await page.reload();await page.locator('.home-screen').waitFor();
 await history();
 await page.locator('.saved-history-entry').first().waitFor();
 await page.locator('.saved-history-entry').first().click();
 await page.locator('.chat-screen .message-assistant').first().waitFor();
 await page.locator('.message-assistant').first().hover();
 await page.getByRole('button',{name:'حذف از ذخیره‌شده‌ها'}).first().click();
 const removed=await call('/api/v1/me/bookmarks');assert.equal(removed.status,200);
 assert.ok(!removed.body.items.some(item=>item.id===savedId));
 // Save again, switch account, and verify the new session sees no foreign saved message.
 await page.locator('.message-assistant').first().hover();
 await page.getByRole('button',{name:'ذخیره پیام'}).first().click();
 await page.locator('.chat-header .back-button').click();
 await page.locator('.home-menu-button').click();
 await page.getByRole('button',{name:'تعویض کاربر آزمایشی'}).click();
 await page.locator('#dev-phone').waitFor();
 await login('09000000762');
 assert.ok(!(await call('/api/v1/me/bookmarks')).body.items.some(item=>item.id===savedId));
 assert.equal((await call(`/api/v1/messages/${savedId}/bookmark`,'PUT')).status,404);
 // A returns and deletes the conversation; bookmark must cascade away.
 await page.locator('.home-menu-button').click();
 await page.getByRole('button',{name:'تعویض کاربر آزمایشی'}).click();
 await page.locator('#dev-phone').waitFor();await login('09000000761');
 await history();
 page.once('dialog',dialog=>dialog.accept());
 await page.getByRole('button',{name:/حذف/}).first().click();
 const after=await call('/api/v1/me/bookmarks');assert.equal(after.status,200);
 assert.ok(!after.body.items.some(item=>item.id===savedId));
 console.log('PASS: saved/unsaved, refresh, history, return, A/B isolation and cascade delete via real Go/MSSQL.');
 await context.close();
} finally {await browser.close();}
