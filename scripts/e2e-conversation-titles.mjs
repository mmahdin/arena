/** Opt-in, connected browser check: real Vite/Go/MSSQL with local mock AI. */
import assert from 'node:assert/strict';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';
let chromium;
try { ({ chromium } = await import('playwright')); }
catch { console.error('BLOCKED: Playwright must be installed.'); process.exit(2); }
const host = 'http://127.0.0.1:5173';
for (const url of [host, 'http://127.0.0.1:8080/health/ready']) {
  try { assert.equal((await fetch(url)).status, 200); }
  catch { console.error('BLOCKED: Start Vite and local Go/MSSQL first.'); process.exit(2); }
}
const browser = await chromium.launch({headless:true, ...(process.env.BROWSER_BIN ? {executablePath:process.env.BROWSER_BIN} : {})});
try {
  const context = await browser.newContext({viewport:{width:390,height:844},locale:'fa-IR'});
  const page = await context.newPage();
  const api = (path, method = 'GET', body) => page.evaluate(async ({path,method,body}) => {
    const response = await fetch(path,{method,credentials:'include',...(body?{headers:{'Content-Type':'application/json'},body:JSON.stringify(body)}:{})});
    return {status:response.status,body:response.status===204?null:await response.json().catch(()=>null)};
  },{path,method,body});
  async function login(phone) {
    await page.locator('#dev-phone').fill(phone);
    await page.getByRole('button',{name:'ادامه'}).last().click();
    await page.locator('#dev-phone').waitFor({state:'hidden'});
    await finishFullOnboarding(page, 'تستر ساختگی');
    await page.locator('.home-screen').waitFor();
  }
  async function history() {
    await page.locator('.home-menu-button').click();
    await page.getByRole('button',{name:'تاریخچه'}).click();
    await page.locator('.history-screen .history-row').first().waitFor();
  }
  async function switchTester() {
    await page.locator('.home-menu-button').click();
    await page.getByRole('button',{name:'تعویض کاربر آزمایشی'}).click();
    await page.locator('#dev-phone').waitFor();
  }
  const phoneA='09000000891',phoneB='09000000892';
  const text=`برنامه‌ریزی برای آخر هفته. ${Date.now()} یک متن ساختگی.`;
  await page.goto(host); await page.locator('#dev-phone').waitFor(); await login(phoneA);
  await page.locator('.chat-fab').click();
  await page.getByRole('textbox',{name:'متن پیام'}).fill(text);
  await page.getByRole('button',{name:'فرستادن پیام'}).click();
  await page.locator('.message-assistant').first().waitFor({timeout:40000});
  const listing = await api('/api/v1/conversations');
  assert.equal(listing.status,200);
  const conversation = listing.body.items[0];
  assert.equal(conversation.title,'برنامه‌ریزی برای آخر هفته');
  await page.locator('.chat-header .back-button').click();
  await history();
  await page.getByRole('button',{name:`تغییر نام ${conversation.title}`}).click();
  const input = page.getByLabel('تغییر نام گفت‌وگو');
  await input.fill('عنوان دلخواه من');
  await page.getByRole('button',{name:'ذخیره نام'}).click();
  await page.getByText('عنوان دلخواه من').waitFor();
  assert.equal((await api('/api/v1/conversations')).body.items.find(x=>x.id===conversation.id).title,'عنوان دلخواه من');
  await page.reload(); await page.locator('.home-screen').waitFor();
  await history(); await page.getByText('عنوان دلخواه من').waitFor();
  await page.locator('.history-row').filter({hasText:'عنوان دلخواه من'}).click();
  await page.getByRole('textbox',{name:'متن پیام'}).fill('پیام دوم صرفاً ساختگی');
  await page.getByRole('button',{name:'فرستادن پیام'}).click();
  await page.locator('.message-assistant').nth(1).waitFor({timeout:40000});
  assert.equal((await api('/api/v1/conversations')).body.items.find(x=>x.id===conversation.id).title,'عنوان دلخواه من');
  await page.locator('.chat-header .back-button').click();
  if (await page.locator('.history-screen').count()) await page.locator('.history-screen .back-button').click();
  await page.locator('.home-screen').waitFor();
  await switchTester(); await login(phoneB);
  assert.equal((await api(`/api/v1/conversations/${conversation.id}`, 'PATCH', {title:'نشت داده'})).status,404);
  const bList = await api('/api/v1/conversations');
  assert.ok(!bList.body.items.some(x=>x.id===conversation.id));
  await switchTester(); await login(phoneA);
  assert.equal((await api('/api/v1/conversations')).body.items.find(x=>x.id===conversation.id).title,'عنوان دلخواه من');
  console.log('PASS: auto-title/rename/reload/follow-up/A-B ownership, real Go + MSSQL.');
  await context.close();
} finally { await browser.close(); }
