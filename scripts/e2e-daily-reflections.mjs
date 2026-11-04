/** Connected browser test: Vite -> Go -> SQL Server, no intercepted successful API calls. */
import assert from 'node:assert/strict';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';
let chromium;
try { ({ chromium } = await import('playwright')); }
catch { console.error('BLOCKED: Playwright dependency missing.'); process.exit(2); }
const host='http://127.0.0.1:5173';
for(const url of [host,'http://127.0.0.1:8080/health/ready']){
 try {assert.equal((await fetch(url)).status,200);}
 catch { console.error('BLOCKED: Vite and local API/MSSQL must be running.');process.exit(2); }
}
const browser=await chromium.launch({headless:true,...(process.env.BROWSER_BIN?{executablePath:process.env.BROWSER_BIN}:{})});
try{
 const context=await browser.newContext({viewport:{width:390,height:844},locale:'fa-IR'});
 const page=await context.newPage();
 const date=new Date();
 const day=`${date.getFullYear()}-${String(date.getMonth()+1).padStart(2,'0')}-${String(date.getDate()).padStart(2,'0')}`;
 const text='این یک متن ساختگی برای آزمون بازتاب روزانه است.';
 const waitForText=async(expected)=>{
   for(let i=0;i<40;i++){ if(await page.locator('#daily-reflection-text').inputValue()===expected)return; await page.waitForTimeout(100); }
   assert.equal(await page.locator('#daily-reflection-text').inputValue(),expected);
 };
 const updated='این نسخهٔ ویرایش‌شدهٔ ساختگی است.';
 const api=async(path, method='GET')=>page.evaluate(async({path,method})=>{
   const response=await fetch(path,{credentials:'include',method});
   return {status:response.status,body:response.status===204?null:await response.json().catch(()=>null)};
 },{path,method});
 async function login(number){
   await page.locator('#dev-phone').fill(number);
   await page.getByRole('button',{name:'ادامه'}).last().click();
   await page.locator('#dev-phone').waitFor({state:'hidden'});
   await finishFullOnboarding(page,'کاربر ساختگی');
   await page.locator('.home-screen').waitFor();
 }
 async function visitDay(){await page.locator('.reflection-card').click();await page.locator('.day-screen').waitFor();await page.locator('#daily-reflection-text').waitFor();}
 await page.goto(host);await page.locator('#dev-phone').waitFor();
 await login('09000000881');await visitDay();
 await page.locator('#daily-reflection-text').fill(text);
 await page.getByRole('button',{name:'ذخیره'}).click();
 await page.getByText('یادداشتت ذخیره شد.').waitFor();
 assert.equal((await api(`/api/v1/me/daily-reflections/${day}`)).body.text,text);
 await page.reload();await page.locator('.home-screen').waitFor();await visitDay();
 await waitForText(text);
 await page.locator('#daily-reflection-text').fill(updated);
 await page.getByRole('button',{name:'ذخیره'}).click();
 await page.getByText('یادداشتت ذخیره شد.').waitFor();
 assert.equal((await api(`/api/v1/me/daily-reflections`)).body.items.filter(x=>x.date===day).length,1);
 await page.locator('.day-art-panel .back-button').click();
 await page.locator('.home-menu-button').click();await page.getByRole('button',{name:'تعویض کاربر آزمایشی'}).click();
 await page.locator('#dev-phone').waitFor();await login('09000000882');await visitDay();
 assert.equal((await api(`/api/v1/me/daily-reflections/${day}`)).status,404);
 await page.locator('.day-art-panel .back-button').click();
 await page.locator('.home-menu-button').click();await page.getByRole('button',{name:'تعویض کاربر آزمایشی'}).click();
 await page.locator('#dev-phone').waitFor();await login('09000000881');await visitDay();
 await waitForText(updated);
 page.once('dialog',dialog=>dialog.accept());await page.getByRole('button',{name:'حذف'}).click();
 assert.equal((await api(`/api/v1/me/daily-reflections/${day}`)).status,404);
 console.log('PASS: Daily Reflection save/edit/refresh/relogin/A-B isolation/delete over real MSSQL.');
 await context.close();
} finally {await browser.close();}
