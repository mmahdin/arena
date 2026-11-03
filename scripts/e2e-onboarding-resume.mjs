/** Live E2E regression: persisted route cursor survives refresh and tester re-entry. */
import assert from 'node:assert/strict';
import { randomInt } from 'node:crypto';
import { FLOW } from '../src/data.js';
let chromium;
try { ({ chromium } = await import('playwright')); }
catch { console.error('BLOCKED: Playwright is unavailable.'); process.exit(2); }
const ui = 'http://127.0.0.1:5173';
const api = 'http://127.0.0.1:8080';
for (const url of [ui, `${api}/health/ready`]) {
  try { assert.equal((await fetch(url)).status, 200); }
  catch { console.error('BLOCKED: start Vite, Go API and MSSQL first.'); process.exit(2); }
}
const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
try {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, locale: 'fa-IR' });
  const page = await context.newPage();
  await page.goto(ui);
  const phone = `0900000${String(randomInt(1000, 10000)).padStart(4, '0')}`;
  await page.locator('#dev-phone').fill(phone);
  await page.getByRole('button', { name: 'ادامه' }).last().click();
  await page.locator('.welcome-screen').waitFor();
  for (const step of ['welcome']) {
    await page.waitForFunction(expected => location.hash === `#/${expected}`, step);
    await page.locator('main .app-screen .primary-button').click();
  }
  await page.waitForFunction(() => location.hash === '#/name');
  let data = await page.evaluate(async () => (await fetch('/api/v1/me/onboarding-progress',{credentials:'include'})).json());
  assert.equal(data.furthestStep, 4);
  await page.reload();
  await page.locator('.name-screen').waitFor();
  assert.equal(await page.evaluate(() => location.hash), '#/name');
  await page.evaluate(() => { location.hash = '/ready'; });
  await page.waitForFunction(() => location.hash === '#/name');
  data = await page.evaluate(async () => (await fetch('/api/v1/me/onboarding-progress',{credentials:'include'})).json());
  assert.equal(data.furthestStep, 4);
  // Re-entry under the same identifier must restore the same server-owned cursor.
  await page.evaluate(async () => { await fetch('/api/v1/dev/session', {method:'DELETE',credentials:'include'}); });
  await page.reload();
  await page.locator('#dev-phone').fill(phone);
  await page.getByRole('button', {name:'ادامه'}).last().click();
  await page.locator('.name-screen').waitFor();
  assert.equal(await page.evaluate(() => location.hash), '#/name');
  // The request never transmits questionnaire answers. This test uses only a synthetic account.
  assert.equal(FLOW[4],'name');
  await context.close();
  console.log('PASS: persisted cursor resumes after refresh and re-entry; deep-link bypass denied.');
} finally { await browser.close(); }
