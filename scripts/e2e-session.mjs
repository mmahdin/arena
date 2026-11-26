/**
 * Opt-in REAL browser acceptance test. Requires a local running Vite server, Go API
 * and SQL Server with seeded synthetic testers. Does not intercept/mock any APIs.
 * Install Playwright separately in your local test environment; this does not modify
 * the source package's pinned frontend dependencies.
 */
import assert from 'node:assert/strict';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';

let chromium;
try { ({ chromium } = await import('playwright')); }
catch { console.error('BLOCKED: install playwright in your local test environment before executing this real-browser test.'); process.exit(2); }

const api = 'http://127.0.0.1:8080';
const ui = 'http://127.0.0.1:5173';
try {
  const ready = await fetch(`${api}/health/ready`);
  assert.equal(ready.status, 200, 'Go/MSSQL readiness');
  const frontend = await fetch(ui);
  assert.equal(frontend.status, 200, 'Vite frontend readiness');
} catch (error) {
  console.error('BLOCKED: start the real Go/MSSQL API and Vite on loopback first.');
  process.exit(2);
}

const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
try {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, locale: 'fa-IR' });
  const page = await context.newPage();
  const phoneField = page.locator('#dev-phone');
  const enter = async (phone) => { await phoneField.fill(phone); await page.getByRole('button', { name: 'ادامه' }).last().click(); };
  await page.goto(ui, { waitUntil: 'domcontentloaded' });
  await phoneField.waitFor();
  await enter('09000000001');
  await phoneField.waitFor({ state: 'hidden' });
  const firstProfile = await page.evaluate(async () => {
    const response = await fetch('/api/v1/me', { credentials: 'include' });
    return { status: response.status, data: await response.json() };
  });
  assert.equal(firstProfile.status, 200);
  assert.ok(firstProfile.data.userId);
  const firstUser = firstProfile.data.userId;

  await finishFullOnboarding(page, 'تستر ساختگی');
  await page.locator('.home-screen').waitFor();
  await page.reload();
  await page.locator('.home-screen').waitFor();
  await page.getByRole('button', { name: 'بازکردن منو' }).click();
  await page.getByRole('button', { name: 'تعویض کاربر آزمایشی' }).click();
  await phoneField.waitFor();
  const revoked = await page.evaluate(async () => (await fetch('/api/v1/me', { credentials: 'include' })).status);
  assert.equal(revoked, 401, 'logout must revoke server session');
  await enter('09000000002');
  const secondProfile = await page.evaluate(async () => {
    const response = await fetch('/api/v1/me', { credentials: 'include' });
    return { status: response.status, data: await response.json() };
  });
  assert.equal(secondProfile.status, 200);
  assert.notEqual(secondProfile.data.userId, firstUser, 'B must not inherit A identity');
  if (secondProfile.data.onboardingCompletedAt) await page.locator('.home-screen').waitFor();
  else await page.locator('.welcome-screen').waitFor();
  await page.reload();
  const persisted = await page.evaluate(async () => (await fetch('/api/v1/me', { credentials: 'include' })).json());
  assert.equal(persisted.userId, secondProfile.data.userId);
  await context.clearCookies();
  await page.reload();
  await phoneField.waitFor();
  console.log('PASS: real browser + Go/MSSQL synthetic identity entry, profile gate, reload, switch, revocation and missing cookie.');
  await context.close();
} finally { await browser.close(); }
