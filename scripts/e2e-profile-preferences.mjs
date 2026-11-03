/**
 * Optional real-browser acceptance: Vite -> Go/Gin -> MSSQL using a fictional local identity.
 * Never intercepts /api/v1. Keeps the designer's other routes accessible.
 * Needs local Vite:5173 and API:8080 plus installed Playwright/Chrome.
 */
import assert from 'node:assert/strict';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';

let chromium;
try { ({ chromium } = await import('playwright')); }
catch { console.error('BLOCKED: install Playwright before the browser acceptance run.'); process.exit(2); }
for (const url of ['http://127.0.0.1:8080/health/ready', 'http://127.0.0.1:5173']) {
  try { assert.equal((await fetch(url)).status, 200); }
  catch { console.error('BLOCKED: start local Go/MSSQL and Vite first.'); process.exit(2); }
}

const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
try {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, locale: 'fa-IR' });
  const page = await context.newPage();
  const me = async () => page.evaluate(async () => {
    const response = await fetch('/api/v1/me', { credentials: 'include' });
    assertNoError(response.status);
    return response.json();
    function assertNoError(status) { if (status !== 200) throw new Error(`GET /me ${status}`); }
  });
  const menu = async () => {
    await page.locator('.home-menu-button').click();
    await page.locator('.drawer-panel').waitFor();
  };
  await page.goto('http://127.0.0.1:5173');
  await page.locator('#dev-phone').fill('09000000957');
  await page.getByRole('button', { name: 'ادامه' }).last().click();
  await page.locator('#dev-phone').waitFor({ state: 'hidden' });
  let profile = await me();
  await finishFullOnboarding(page, 'کاربر نمونه');
  profile = await me();
  const original = { ...profile };
  const newStyle = profile.conversationStyle === 'listening' ? 'balanced' : 'listening';
  const newStyleIndex = newStyle === 'listening' ? 2 : 1;
  await menu();
  await page.getByRole('button', { name: 'ترجیحات' }).click();
  await page.locator('.approach-screen').waitFor();
  await page.getByRole('button', { name: `گزینه ${newStyleIndex}` }).click();
  await page.getByRole('button', { name: 'ذخیرهٔ سبک گفت‌وگو' }).click();
  await page.locator('.drawer-panel').waitFor();
  let saved = await me();
  assert.equal(saved.conversationStyle, newStyle);
  assert.equal(saved.onboardingCompletedAt, original.onboardingCompletedAt);
  assert.equal(saved.userId, original.userId);

  await page.getByRole('button', { name: 'نام نمایشی' }).click();
  await page.locator('.name-screen #name').fill('نام جدید همراه');
  await page.getByRole('button', { name: 'ذخیره نام' }).click();
  await page.locator('.drawer-panel').waitFor();
  saved = await me();
  assert.equal(saved.displayName, 'نام جدید همراه');
  await page.reload();
  await page.locator('.home-screen').waitFor();
  assert.equal((await me()).displayName, saved.displayName);
  assert.equal((await me()).conversationStyle, newStyle);

  await menu();
  await page.getByRole('button', { name: 'حساب و داده‌ها' }).click();
  await page.locator('.privacy-screen').waitFor();
  await page.locator('.privacy-screen .back-button').click();
  await page.locator('.drawer-panel').waitFor();
  await page.getByRole('button', { name: 'درباره' }).click();
  await page.locator('.research-screen').waitFor();
  await page.locator('.research-screen .back-button').click();
  await page.locator('.drawer-panel').waitFor();
  await page.getByRole('button', { name: 'پرسش‌نامه‌ی سلامت' }).click();
  await page.locator('.survey-screen').waitFor();
  await page.locator('.survey-screen .back-button').click();
  await page.locator('.drawer-panel').waitFor();
  await page.locator('.drawer-backdrop').click();
  await page.getByRole('button', { name: 'بازکردن نامه' }).click();
  await page.locator('.day-screen').waitFor();
  await page.locator('.letter-card').click();
  await page.locator('.insight-screen').waitFor();
  await page.locator('.insight-screen .back-button').click();
  await page.locator('.day-screen').waitFor();
  await page.locator('.day-screen .back-button').click();
  await page.locator('.home-screen').waitFor();

  // Restore previous test-account settings without creating new server features.
  const restored = await page.evaluate(async (data) => {
    const response = await fetch('/api/v1/me/profile', {
      method: 'PATCH', credentials: 'include', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ displayName: data.displayName, conversationStyle: data.conversationStyle }),
    });
    return response.status;
  }, original);
  assert.equal(restored, 200);
  await context.close();
  console.log('PASS: real preferences persistence, reload, original UI routes and safe restoration.');
} finally { await browser.close(); }
