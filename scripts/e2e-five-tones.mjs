/** Real browser -> Vite -> Gin -> SQL Server acceptance. Synthetic local accounts only. */
import assert from 'node:assert/strict';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';
import { APPROACHES } from '../src/data.js';
import { APPROACH_TO_STYLE } from '../src/features/profile/preferences.js';

let chromium;
try { ({ chromium } = await import('playwright')); }
catch { console.error('BLOCKED: Playwright not available.'); process.exit(2); }
for (const url of ['http://127.0.0.1:8080/health/ready', 'http://127.0.0.1:5173']) {
  try { assert.equal((await fetch(url)).status, 200); }
  catch { console.error('BLOCKED: start the local Go/MSSQL and Vite servers.'); process.exit(2); }
}
const browser = await chromium.launch({headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {})});
try {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, locale: 'fa-IR' });
  const page = await context.newPage();
  const me = () => page.evaluate(async () => {
    const response = await fetch('/api/v1/me', { credentials: 'include' });
    if (response.status !== 200) throw new Error('GET /me failed');
    return response.json();
  });
  const openPreferences = async () => {
    await page.locator('.home-menu-button').click();
    await page.locator('.drawer-panel').waitFor();
    await page.getByRole('button', { name: 'ترجیحات' }).click();
    await page.locator('.approach-screen').waitFor();
  };
  await page.goto('http://127.0.0.1:5173');
  await page.locator('#dev-phone').fill('09000000967');
  await page.getByRole('button', { name: 'ادامه' }).last().click();
  await page.locator('#dev-phone').waitFor({ state: 'hidden' });
  const first = await me();
  if (!first.onboardingCompletedAt) await finishFullOnboarding(page, 'کاربر ساختگی');
  await page.locator('.home-screen').waitFor();
  const original = await me();
  try {
    for (let i = 0; i < APPROACHES.length; i++) {
      const choice = APPROACHES[i];
      const style = APPROACH_TO_STYLE[choice.name];
      await openPreferences();
      await page.getByRole('button', { name: `گزینه ${i + 1}` }).click();
      await page.getByRole('button', { name: 'ذخیرهٔ سبک گفت‌وگو' }).click();
      await page.locator('.drawer-panel').waitFor();
      assert.equal((await me()).conversationStyle, style, `${choice.name} was not stored`);
      await page.reload();
      await page.locator('.home-screen').waitFor();
      assert.equal((await me()).conversationStyle, style, 'style did not persist after refresh');
      await openPreferences();
      assert.equal((await page.locator('.approach-response').textContent()).trim(), choice.response.trim(), `${choice.name} UI did not restore`);
      await page.locator('.approach-screen .back-button').click();
      await page.locator('.drawer-panel').waitFor();
      await page.locator('.drawer-backdrop').click({ position: { x: 8, y: 400 } });
    }
  } finally {
    const status = await page.evaluate(async (profile) => {
      const response = await fetch('/api/v1/me/profile', {
        method: 'PATCH', credentials: 'include', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ displayName: profile.displayName, conversationStyle: profile.conversationStyle }),
      });
      return response.status;
    }, original);
    assert.equal(status, 200, 'restoring original preference failed');
  }
  await context.close();
  console.log('PASS: all five original UI approaches persist distinctly through the real API/SQL Server and restore after refresh.');
} finally { await browser.close(); }
