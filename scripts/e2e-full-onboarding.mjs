/** Live E2E: a fresh synthetic account traverses every original designer screen. */
import assert from 'node:assert/strict';
import { randomInt } from 'node:crypto';
import { ACTIVE_ONBOARDING_ROUTES } from '../src/features/profile/navigation.js';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';
let chromium;
try { ({ chromium } = await import('playwright')); }
catch { console.error('BLOCKED: Playwright is unavailable.'); process.exit(2); }
const api = 'http://127.0.0.1:8080';
const ui = 'http://127.0.0.1:5173';
for (const url of [`${api}/health/ready`, ui]) {
  try { assert.equal((await fetch(url)).status, 200); }
  catch { console.error('BLOCKED: start local Vite, Go API, and MSSQL before E2E.'); process.exit(2); }
}
const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
try {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, locale: 'fa-IR' });
  const page = await context.newPage();
  await page.goto(ui);
  const phone = `0900000${String(randomInt(1000, 10000)).padStart(4, '0')}`;
  await page.locator('#dev-phone').fill(phone);
  await page.getByRole('button', { name: 'ادامه' }).last().click();
  await page.locator('#dev-phone').waitFor({ state: 'hidden' });
  const before = await page.evaluate(async () => (await fetch('/api/v1/me', { credentials: 'include' })).json());
  assert.equal(before.onboardingCompletedAt, null, 'requires a not-yet-completed synthetic identifier');
  await page.locator('.welcome-screen').waitFor();
  // Direct URL changes must not bypass the onboarding flow.
  await page.evaluate(() => { window.location.hash = '/ready'; });
  await page.waitForFunction(() => window.location.hash === '#/welcome');
  await page.evaluate(() => { window.location.hash = '/chat'; });
  await page.waitForFunction(() => window.location.hash === '#/welcome');
  await page.locator('.welcome-screen .primary-button').click();
  await page.locator('.privacy-screen').waitFor();
  await page.locator('.privacy-screen .back-button').click();
  await page.locator('.welcome-screen').waitFor();
  await page.reload();
  await page.locator('.privacy-screen').waitFor(); // resume highest server-saved step after refresh
  const restoredCursor = await page.evaluate(async () => (await fetch('/api/v1/me/onboarding-progress', { credentials: 'include' })).json());
  assert.equal(restoredCursor.furthestStep, 1);
  await page.locator('.privacy-screen .back-button').click();
  await page.locator('.welcome-screen').waitFor();
  const profilePayloadShapes = [];
  page.on('request', request => {
    if (request.method() === 'PATCH' && new URL(request.url()).pathname === '/api/v1/me/profile') {
      const payload = request.postDataJSON();
      profilePayloadShapes.push(Object.keys(payload).sort());
    }
  });
  const visited = [];
  await finishFullOnboarding(page, 'کاربر همراه', { onStep: async route => visited.push(route) });
  assert.deepEqual(visited, ACTIVE_ONBOARDING_ROUTES);
  assert.deepEqual(profilePayloadShapes, [['conversationStyle', 'displayName']], 'only approved fields are submitted once on Ready');
  await page.reload();
  await page.locator('.home-screen').waitFor();
  await context.close();
  console.log('PASS: 26 enabled steps, original screens retained + server cursor resume + final completion → Chat.');
} finally { await browser.close(); }
