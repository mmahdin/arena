/** Separate controlled network failures; successes always reach the actual API. */
import assert from 'node:assert/strict';
import { randomInt } from 'node:crypto';
import { chromium } from 'playwright';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';
const browser=await chromium.launch({headless:true,...(process.env.BROWSER_BIN?{executablePath:process.env.BROWSER_BIN}:{})});
try{
 const context=await browser.newContext({viewport:{width:390,height:844}});
 const page=await context.newPage();page.setDefaultTimeout(15000);
 const call=(method,path)=>page.evaluate(async({method,path})=>{const r=await fetch(`/api/v1${path}`,{method,credentials:'include'});return {status:r.status,body:r.status===204?null:await r.json()};},{method,path});
 const phone=`0900000${randomInt(1000,9999)}`;
 await page.goto('http://127.0.0.1:5173');
 const progress='**/api/v1/me/onboarding-progress';
 await page.route(progress,r=>r.abort('internetdisconnected'));
 await page.locator('#dev-phone').fill(phone);await page.getByRole('button',{name:'ادامه',exact:true}).click();
 await page.getByText('ادامهٔ مسیر بارگذاری نشد.').waitFor();
 await page.unroute(progress);await page.getByRole('button',{name:'تلاش دوباره',exact:true}).click();await page.locator('.welcome-screen').waitFor();
 await page.route(progress,r=>r.request().method()==='PUT'?r.abort('timedout'):r.continue());
 await page.locator('.welcome-screen .primary-button').click();
 await page.getByText('ثبت مرحله انجام نشد؛ دوباره ادامه را انتخاب کن.').waitFor();
 assert.equal((await call('GET','/me/onboarding-progress')).body.furthestStep,0);
 await page.unroute(progress);await page.locator('.welcome-screen .primary-button').click();await page.locator('.name-screen').waitFor();
 // Restore Welcome locally and use every real remaining screen.
 await page.locator('.name-screen .back-button').click();
 await finishFullOnboarding(page,'حساب خطای ساختگی');
 await page.locator('.chat-fab').click();await page.getByRole('textbox',{name:'متن پیام'}).fill('پیام ساختگی آزمون قطع شبکه');await page.getByRole('button',{name:'فرستادن پیام'}).click();await page.locator('.message-assistant').waitFor();
 await page.locator('.connected-pending').waitFor({state:'hidden'});
 const id=(await call('GET','/conversations')).body.items[0].id;
 const bookmarks='**/api/v1/messages/*/bookmark';
 await page.route(bookmarks,r=>r.abort('internetdisconnected'));
 await page.getByRole('button',{name:'ذخیره پیام',exact:true}).first().click();
 await page.getByText('وضعیت پیام‌های ذخیره‌شده دریافت نشد.').waitFor();
 assert.equal((await call('GET','/me/bookmarks')).body.items.length,0);
 await page.unroute(bookmarks);await page.getByRole('button',{name:'تلاش دوباره',exact:true}).click();await page.getByText('وضعیت پیام‌های ذخیره‌شده دریافت نشد.').waitFor({state:'hidden'});
 await page.getByRole('button',{name:'ذخیره پیام',exact:true}).first().click();await page.getByRole('button',{name:'حذف از ذخیره‌شده‌ها'}).first().waitFor();
 assert.equal((await call('GET','/me/bookmarks')).body.items.length,1);
 await call('DELETE',`/conversations/${id}`);
 assert.equal((await call('GET','/me/bookmarks')).body.items.length,0);
 await call('DELETE','/dev/session');await context.close();
 console.log('PASS: real progress read/write network failure and retry; real bookmark save failure with no false success, Retry and cascade.');
}finally{await browser.close();}
