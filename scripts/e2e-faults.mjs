/** Separate network fault injection; no fabricated success, only real Mock AI for successful turns. */
import assert from 'node:assert/strict';
import { chromium } from 'playwright';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';
const browser = await chromium.launch({headless:true,...(process.env.BROWSER_BIN?{executablePath:process.env.BROWSER_BIN}:{})});
try {
  const context=await browser.newContext({viewport:{width:390,height:844},locale:'fa-IR'});
  const page=await context.newPage(); page.setDefaultTimeout(15000);
  const call=(method,path,body)=>page.evaluate(async({method,path,body})=>{
    const response=await fetch(`/api/v1${path}`,{method,credentials:'include',...(body===undefined?{}:{headers:{'Content-Type':'application/json'},body:JSON.stringify(body)})});
    return {status:response.status,body:response.status===204?null:await response.json()};
  },{method,path,body});
  await page.goto('http://127.0.0.1:5173');
  await page.locator('#dev-phone').fill('09000000991');
  await page.getByRole('button',{name:'ادامه',exact:true}).click();
  await page.locator('#dev-phone').waitFor({state:'hidden'});
  const me=await call('GET','/me');
  await finishFullOnboarding(page,'تستر خطای ساختگی');
  await page.locator('.home-screen').waitFor();await page.locator('.chat-fab').click();
  const attempts=[];let mode='provider-failure';let id;
  await page.route('**/api/v1/conversations/*/turns',async route=>{
    const request=route.request();attempts.push(request.postDataJSON());
    id=new URL(request.url()).pathname.split('/')[4];
    if(mode==='provider-failure'){
      // Controlled unsuccessful transport response; successful turns still use real Mock AI.
      await route.fulfill({status:503,contentType:'application/problem+json',body:JSON.stringify({status:503,code:'AI_UNAVAILABLE'})});
    }else if(mode==='lost-response'){
      const response=await route.fetch();assert.equal(response.status(),200);
      await route.abort('timedout'); // DB committed; browser cannot know the final result.
    }else await route.continue();
  });
  await page.getByRole('textbox',{name:'متن پیام'}).fill('پیام ساختگی برای قطع سرویس');
  await page.getByRole('button',{name:'فرستادن پیام'}).click();
  await page.getByRole('button',{name:'تلاش دوباره با همان پیام'}).waitFor();
  assert.equal((await call('GET',`/conversations/${id}/messages`)).body.items.length,0);
  assert.equal(await page.locator('.message-assistant').count(),0);
  mode='normal';
  await page.getByRole('button',{name:'تلاش دوباره با همان پیام'}).click();
  await page.locator('.message-assistant').first().waitFor();
  await page.locator('.connected-pending').waitFor({state:'hidden'});
  assert.deepEqual(attempts[0],attempts[1]);
  assert.equal((await call('GET',`/conversations/${id}/messages`)).body.items.length,2);
  mode='lost-response';
  await page.getByRole('textbox',{name:'متن پیام'}).fill('پیام ساختگی با پاسخ گم‌شده');
  await page.getByRole('button',{name:'فرستادن پیام'}).click();
  await page.getByRole('button',{name:'تلاش دوباره با همان پیام'}).waitFor();
  assert.equal((await call('GET',`/conversations/${id}/messages`)).body.items.length,4);
  mode='normal';await page.getByRole('button',{name:'تلاش دوباره با همان پیام'}).click();
  await page.locator('.message-assistant').nth(1).waitFor();
  await page.locator('.connected-pending').waitFor({state:'hidden'});
  assert.deepEqual(attempts[2],attempts[3]);
  assert.equal((await call('GET',`/conversations/${id}/messages`)).body.items.length,4);
  // Live deletion then turn: terminal 404 must offer a new thread, not retry forever.
  assert.equal((await call('DELETE',`/conversations/${id}`)).status,204);
  await page.getByRole('textbox',{name:'متن پیام'}).fill('پیام ساختگی پس از حذف گفتگو');
  await page.getByRole('button',{name:'فرستادن پیام'}).click();
  await page.getByText('این گفت‌وگو در دسترس نیست.').waitFor();
  assert.equal(await page.getByRole('button',{name:'تلاش دوباره با همان پیام'}).count(),0);
  page.once('dialog',dialog=>dialog.accept());
  await page.getByRole('button',{name:'گفت‌وگوی جدید',exact:true}).first().click();
  await page.locator('.connected-pending').waitFor({state:'hidden'});
  // Real revoked session: the next private request must return to login.
  await call('DELETE','/dev/session');
  await page.evaluate(()=>{location.hash='/history';});
  await page.locator('#dev-phone').waitFor();
  assert.equal(await page.locator('.message-assistant').count(),0);
  await context.setOffline(true);
  await page.locator('#dev-phone').fill('09000000992');
  await page.getByRole('button',{name:'ادامه',exact:true}).click();
  await page.locator('.dev-login-error[role="alert"]').first().waitFor();
  assert.equal(await page.locator('.home-screen').count(),0);
  await context.setOffline(false);
  await page.getByRole('button',{name:/تلاش دوباره/}).click();
  await page.getByRole('button',{name:'ادامه',exact:true}).waitFor();
  console.log('PASS: injected 503/no partial pair → same-ID retry; committed response lost → same-ID replay/exactly two pairs; terminal deleted-thread 404; revoked-session 401; offline login remains unauthenticated.');
  await context.close();
}finally{await browser.close();}
