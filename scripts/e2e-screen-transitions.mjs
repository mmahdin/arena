/** UI transition regression against real local Vite + Go/MSSQL. No API interception. */
import assert from 'node:assert/strict';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';

let chromium;
try { ({ chromium } = await import('playwright')); }
catch { console.error('BLOCKED: Playwright is required for transition browser QA.'); process.exit(2); }
for (const url of ['http://127.0.0.1:5173', 'http://127.0.0.1:8080/health/ready']) {
  try { assert.equal((await fetch(url)).status, 200); }
  catch { console.error('BLOCKED: start Vite and the Go/MSSQL stack first.'); process.exit(2); }
}

const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
try {
  for (const width of [360, 390, 430]) {
    const context = await browser.newContext({ viewport: { width, height: 844 }, locale: 'fa-IR' });
    const page = await context.newPage();
    const errors = [];
    page.on('pageerror', (error) => errors.push(error.message));
    await page.goto('http://127.0.0.1:5173');
    await page.locator('#dev-phone').fill('09000000998');
    await page.getByRole('button', { name: 'ادامه' }).last().click();
    await page.locator('#dev-phone').waitFor({ state: 'hidden' });
    await finishFullOnboarding(page, 'کاربر نمایشی');
    await page.locator('.home-screen').waitFor();

    await page.locator('.chat-fab').click();
    await page.locator('.chat-screen').waitFor();
    assert.equal(await page.locator('.app-shell').getAttribute('data-screen-surface'), 'dark');
    const css = await page.evaluate(() => {
      const shell = document.querySelector('.app-shell');
      const transition = document.querySelector('.screen-transition');
      return { shell: getComputedStyle(shell).backgroundColor, layer: getComputedStyle(transition).backgroundColor };
    });
    assert.equal(css.shell, 'rgb(18, 17, 13)', 'dark chat must not expose light shell');
    assert.equal(css.layer, css.shell, 'animated wrapper and page backdrop must match');

    await page.locator('.chat-header .back-button').click();
    await page.locator('.home-screen').waitFor();
    assert.equal(await page.locator('.app-shell').getAttribute('data-screen-surface'), 'dark');
    await page.locator('.home-menu-button').click();
    await page.locator('.drawer-panel').waitFor();
    assert.equal(await page.locator('.app-shell').getAttribute('data-screen-surface'), 'black');
    await page.getByRole('button', { name: 'تاریخچه' }).click();
    await page.locator('.history-screen').waitFor();
    assert.equal(await page.locator('.app-shell').getAttribute('data-screen-surface'), 'dark');
    await page.emulateMedia({ reducedMotion: 'reduce' });
    assert.equal(await page.locator('.screen-transition').evaluate(node => getComputedStyle(node).animationName), 'none');
    assert.deepEqual(errors, [], 'no runtime errors during navigation');
    await context.close();
  }
  console.log('PASS: chat/home/menu/history surfaces at 360/390/430 and reduced motion.');
} finally { await browser.close(); }
