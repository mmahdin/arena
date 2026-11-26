/**
 * Opt-in browser + real Go/MSSQL + Mock AI E2E; NO route interception/mock.
 * Requires localhost Vite:5173 and API:8080 with dev phone login enabled.
 * Uses ONLY fictional 0900000XXXX identities and synthetic messages.
 */
import assert from 'node:assert/strict';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';

let chromium;
try { ({ chromium } = await import('playwright')); }
catch { console.error('BLOCKED: Playwright is not installed in this test environment.'); process.exit(2); }

const ui = 'http://127.0.0.1:5173';
const api = 'http://127.0.0.1:8080';
for (const url of [`${api}/health/ready`, ui]) {
  try { const response = await fetch(url); assert.equal(response.status, 200, `${url} should be healthy`); }
  catch { console.error('BLOCKED: Start the local Go/MSSQL and Vite servers first.'); process.exit(2); }
}

const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
try {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, locale: 'fa-IR' });
  const page = await context.newPage();
  const apiGET = (path) => page.evaluate(async (path) => {
    const response = await fetch(path, { credentials: 'include' });
    return {status:response.status,body:await response.json().catch(()=>null)};
  }, path);
  async function enter(phone) {
    await page.locator('#dev-phone').fill(phone);
    await page.getByRole('button', { name: 'ادامه' }).last().click();
    await page.locator('#dev-phone').waitFor({state:'hidden'});
    const me = await apiGET('/api/v1/me');
    assert.equal(me.status,200);
    await finishFullOnboarding(page, 'تستر ساختگی');
    await page.locator('.home-screen').waitFor();
    return (await apiGET('/api/v1/me')).body.userId;
  }
  async function switchTester() {
    await page.locator('.home-screen .home-menu-button').click();
    await page.getByRole('button', { name: 'تعویض کاربر آزمایشی' }).click();
    await page.locator('#dev-phone').waitFor();
  }
  await page.goto(ui, {waitUntil:'domcontentloaded'});
  await page.locator('#dev-phone').waitFor();
  const ownerA = await enter('09000000841');
  const text = `پیام صرفاً آزمایشی ${Date.now()}`;
  await page.locator('.chat-fab').click();
  await page.locator('.chat-screen').waitFor();
  await page.getByRole('textbox', { name: 'متن پیام' }).fill(text);
  await page.getByRole('button', { name: 'فرستادن پیام' }).click();
  await page.locator('.message-assistant').first().waitFor({timeout:40000});
  assert.equal(await page.locator('.message-user').count(),1);
  const list = await apiGET('/api/v1/conversations');
  assert.equal(list.status,200);assert.ok(list.body.items.length);
  const id = list.body.items[0].id;
  const pair = await apiGET(`/api/v1/conversations/${id}/messages`);
  assert.equal(pair.status,200);assert.equal(pair.body.items.length,2);
  assert.equal(pair.body.items[0].content,text);
  await page.reload();await page.locator('.home-screen').waitFor();
  await page.locator('.home-menu-button').click();
  await page.getByRole('button',{name:'تاریخچه'}).click();
  await page.locator('.history-screen .history-row').first().click();
  await page.locator('.chat-screen .message-assistant').first().waitFor();
  await page.getByRole('textbox',{name:'متن پیام'}).fill('ادامهٔ مکالمه صرفاً برای تست');
  await page.getByRole('button',{name:'فرستادن پیام'}).click();
  await page.locator('.message-assistant').nth(1).waitFor({timeout:40000});
  const continued = await apiGET(`/api/v1/conversations/${id}/messages`);
  assert.equal(continued.body.items.length,4);
  await page.locator('.chat-header .back-button').click();
  await switchTester();
  const ownerB = await enter('09000000842');
  assert.notEqual(ownerA,ownerB);
  const forbidden = await apiGET(`/api/v1/conversations/${id}/messages`);
  assert.equal(forbidden.status,404);
  await switchTester();await enter('09000000841');
  await page.locator('.home-menu-button').click();await page.getByRole('button',{name:'تاریخچه'}).click();
  await page.locator('.history-screen .history-row').first().waitFor();
  page.once('dialog', dialog => dialog.accept());
  await page.getByRole('button',{name:/حذف/}).first().click();
  const deleted = await apiGET(`/api/v1/conversations/${id}/messages`);
  assert.equal(deleted.status,404);
  console.log('PASS: Hamrah browser → Go/MSSQL Mock AI; create/send/history/reload/continue/user isolation/delete.');
  await context.close();
} finally { await browser.close(); }
