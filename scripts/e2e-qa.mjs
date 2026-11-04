/** Live acceptance: Vite → Gin → MSSQL → mock AI. No route interception. */
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';

const ui = 'http://127.0.0.1:5173';
const artifacts = 'test-results/hamrah-qa';
await mkdir(artifacts, { recursive: true });
const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
try {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, locale: 'fa-IR', reducedMotion: 'reduce' });
  const page = await context.newPage();
  page.setDefaultTimeout(15000);
  const failures = [];
  const legacy = [];
  page.on('pageerror', () => failures.push('uncaught browser exception'));
  page.on('request', request => { if (request.url().endsWith('/dev/users')) legacy.push(true); });
  const call = (method, path, body) => page.evaluate(async ({method,path,body}) => {
    const response = await fetch(`/api/v1${path}`, { method, credentials: 'include', ...(body === undefined ? {} : {headers: {'Content-Type':'application/json'}, body: JSON.stringify(body)}) });
    return { status: response.status, body: response.status === 204 ? null : await response.json() };
  }, {method,path,body});
  const screenshot = name => page.screenshot({ path: `${artifacts}/${name}.png` });
  const navigate = async route => { await page.evaluate(route => { location.hash = `/${route}`; }, route); await page.locator(`.${route}-screen`).first().waitFor(); };
  const history = async () => {
    await navigate('home');
    await page.getByRole('button', {name:'بازکردن منو'}).click();
    await page.getByRole('button', {name:'تاریخچه',exact:true}).click();
    await page.locator('.history-screen').waitFor();
    await page.getByText('در حال بارگذاری گفت‌وگوها…').waitFor({state:'hidden'});
  };
  const leave = async label => {
    await navigate('home');
    await page.getByRole('button',{name:'بازکردن منو'}).click();
    await page.getByRole('button',{name:label,exact:true}).click();
    await page.locator('#dev-phone').waitFor({state:'visible'});
    await page.locator('#dev-phone').waitFor({state:'attached'});
    assert.equal((await call('GET','/me')).status,401);
  };
  async function enter(phone, fresh = false) {
    await page.locator('#dev-phone').fill(phone);
    const result = page.waitForResponse(r => r.url().endsWith('/dev/identity-session') && r.request().method() === 'POST');
    await page.getByRole('button',{name:'ادامه',exact:true}).click();
    const response = await result;
    assert.equal(response.status(),200);
    const login = await response.json();
    if (fresh) assert.equal(login.created,true,'fresh synthetic identity');
    await page.locator('#dev-phone').waitFor({state:'hidden'});
    const me = await call('GET','/me'); assert.equal(me.status,200);
    if (fresh) assert.equal(me.body.onboardingCompletedAt,null);
    if (!me.body.onboardingCompletedAt) {
      await page.reload();
      await finishFullOnboarding(page,'تستر ساختگی QA');
      assert.equal((await call('GET','/me')).body.userId,me.body.userId);
    }
    await page.locator('.home-screen').waitFor();
    return (await call('GET','/me')).body;
  }
  async function send(text, count) {
    await page.getByRole('textbox',{name:'متن پیام'}).fill(text);
    const sent = page.waitForResponse(r => r.url().endsWith('/turns') && r.request().method() === 'POST');
    await page.getByRole('button',{name:'فرستادن پیام'}).click();
    const response = await sent; assert.equal(response.status(),200);
    const request = response.request();
    const payload = request.postDataJSON();
    await page.locator('.message-assistant').nth(count-1).waitFor();
    await page.locator('.connected-pending').waitFor({state:'hidden'});
    return { path:new URL(request.url()).pathname.replace('/api/v1',''), payload, response:await response.json() };
  }
  await page.goto(ui);
  await page.locator('#dev-phone').waitFor();
  for (const width of [360,390,430,1280]) {
    await page.setViewportSize({width,height:844});
    await screenshot(`login-${width}`);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),true,'no horizontal viewport overflow');
  }
  await page.setViewportSize({width:390,height:844});
  const base = 1000 + Math.floor(Math.random()*7000);
  const phoneA = `0900000${base}`;
  const phoneB = `0900000${base+1}`;
  const a = await enter(phoneA,true);
  await page.reload(); await page.locator('.home-screen').waitFor();
  assert.equal((await call('GET','/me')).body.displayName,a.displayName);
  await history();
  await page.getByText('هنوز گفت‌وگویی نداری.',{exact:false}).waitFor();
  await navigate('home'); await page.locator('.chat-fab').click();
  const first = await send('پیام ساختگی اول برای آزمون یکپارچه',1);
  const id = first.path.split('/')[2];
  const replay = await call('POST',first.path,first.payload);
  assert.equal(replay.status,200); assert.deepEqual(replay.body,first.response);
  assert.equal((await call('GET',`/conversations/${id}/messages`)).body.items.length,2);
  assert.equal((await call('POST',first.path,{...first.payload,message:'متن تغییرکرده ساختگی'})).status,400);
  await send('پیام ساختگی دوم',2);
  await screenshot('chat-390');
  for (const width of [360,430,1280]) {
    await page.setViewportSize({width,height:844});
    const field = await page.getByRole('textbox',{name:'متن پیام'}).boundingBox();
    assert.ok(field.x>=0 && field.x+field.width<=width && field.y+field.height<=844,'composer visible');
    assert.equal(await page.evaluate(() => getComputedStyle(document.querySelector('.app-shell')).direction),'rtl');
    await screenshot(`chat-${width}`);
  }
  await page.setViewportSize({width:390,height:480});
  await page.getByRole('textbox',{name:'متن پیام'}).focus();
  const compact = await page.getByRole('textbox',{name:'متن پیام'}).boundingBox();
  assert.ok(compact.y+compact.height<=480,'composer visible with reduced keyboard-like viewport');
  await screenshot('chat-compact-height');
  await page.setViewportSize({width:390,height:844});
  // Second thread with the same real UI; opening an older one must restore its pairs.
  await page.getByRole('button',{name:'گفت‌وگوی جدید',exact:true}).click();
  const second = await send('گفتگوی ساختگی دوم برای تاریخچه',1);
  const id2 = second.path.split('/')[2]; assert.notEqual(id,id2);
  await history(); assert.equal(await page.locator('.history-row').count(),2);
  await screenshot('history-390');
  await page.locator('.history-row').nth(1).click();
  await page.getByText('پیام ساختگی اول برای آزمون یکپارچه',{exact:true}).waitFor();
  await send('ادامهٔ ساختگی گفتگوی قدیمی',3);
  assert.equal((await call('GET',`/conversations/${id}/messages`)).body.items.length,6);
  await page.reload(); await page.locator('.home-screen').waitFor();
  await history(); await page.locator('.history-row').first().click();
  await page.locator('.message-assistant').nth(2).waitFor();
  await leave('خروج');
  const returning = await enter(phoneA); assert.equal(returning.userId,a.userId);
  assert.equal(returning.onboardingCompletedAt,a.onboardingCompletedAt);
  await history(); assert.equal(await page.locator('.history-row').count(),2);
  // Genuine network failure, with no synthetic API response.
  await navigate('home'); await context.setOffline(true);
  await page.getByRole('button',{name:'بازکردن منو'}).click();
  await page.getByRole('button',{name:'تاریخچه',exact:true}).click();
  await page.locator('.history-screen [role="alert"]').first().waitFor();
  await context.setOffline(false);
  await page.locator('.history-screen [role="alert"]').filter({hasText:'دریافت یا حذف'}).getByRole('button').click();
  await page.locator('.saved-history-section [role="alert"] button').click();
  await page.waitForFunction(() => document.querySelectorAll('.history-screen [role="alert"]').length===0);
  assert.equal(await page.locator('.history-row').count(),2);
  await leave('تعویض کاربر آزمایشی');
  const b = await enter(phoneB,true); assert.notEqual(a.userId,b.userId);
  await history(); assert.equal(await page.locator('.history-row').count(),0);
  assert.equal((await call('GET',`/conversations/${id}/messages`)).status,404);
  await leave('تعویض کاربر آزمایشی'); await enter(phoneA);
  await history();
  // Delete with cancellation first, then real confirmation.
  page.once('dialog', dialog => dialog.dismiss());
  await page.getByRole('button',{name:/حذف/}).first().click();
  assert.equal((await call('GET','/conversations')).body.items.length,2);
  page.once('dialog', dialog => dialog.accept());
  await page.getByRole('button',{name:/حذف/}).first().click();
  await page.waitForFunction(async () => (await (await fetch('/api/v1/conversations')).json()).items.length===1);
  assert.equal((await call('GET',`/conversations/${id}/messages`)).status,404);
  // Delete the other thread while UI still has its cached history entry: real 404 UI.
  assert.equal((await call('DELETE',`/conversations/${id2}`)).status,204);
  await page.locator('.history-row').click();
  await page.getByText('این گفت‌وگو در دسترس نیست.').waitFor();
  await page.getByRole('button',{name:'گفت‌وگوی جدید',exact:true}).first().click();
  assert.equal(await page.locator('.message-user').count(),0);
  assert.equal((await call('GET','/conversations')).body.items.length,0);
  assert.equal(await page.evaluate(() => localStorage.length+sessionStorage.length),0);
  assert.deepEqual(legacy,[]); assert.deepEqual(failures,[]);
  await leave('خروج');
  console.log('PASS: fresh identity → onboarding/reload → two real chats → replay → older history/continue → logout/relogin → offline/retry → switch/isolation → confirmed deletion/404; RTL 360/390/430/1280.');
  await context.close();
} finally { await browser.close(); }
