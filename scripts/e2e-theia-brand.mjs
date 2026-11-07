/** Visual-brand QA against the isolated Theia preview. No backend or intercepted API. */
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';

const url = process.env.THEIA_PREVIEW_URL || 'http://127.0.0.1:5173';
const artifacts = 'test-results/theia-brand';
await mkdir(artifacts, { recursive: true });
try { assert.equal((await fetch(`${url}/scripts/fixtures/theia-preview.html`)).status, 200); }
catch { console.error('BLOCKED: start `npm run preview:theia` first.'); process.exit(2); }

const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
const routes = ['welcome','privacy','simple-intro','name','age','deeper-intro','mood','know-intro','almost-intro','approach','voice','checkins','ready','home','menu','chat','letter','history','settings'];
try {
  const errors = [];
  for (const width of [360, 390, 430]) {
    const context = await browser.newContext({ viewport: { width, height: 844 }, locale: 'fa-IR', reducedMotion: 'reduce' });
    const page = await context.newPage();
    page.on('pageerror', error => errors.push(error.message));
    page.on('console', message => { if (message.type() === 'error') errors.push(message.text()); });
    for (const route of routes) {
      await page.goto(`${url}/scripts/fixtures/theia-preview.html#/${route}`);
      await page.locator('.app-shell').waitFor();
      await page.waitForTimeout(80);
      assert.equal(await page.evaluate(() => getComputedStyle(document.querySelector('.app-shell')).direction), 'rtl', `${route} RTL`);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, `${route} horizontal overflow`);
      const broken = await page.evaluate(() => [...document.images].filter(image => !image.complete || image.naturalWidth === 0).map(image => image.src));
      assert.deepEqual(broken, [], `${route} broken artwork`);
      assert.equal(await page.evaluate(() => document.querySelectorAll('img[src*="flower"], img[src*="daisy"], img[src*="welcome-figures"]').length), 0, `${route} legacy artwork`);
      if (width === 390) await page.screenshot({ path: `${artifacts}/${route}.png` });
    }
    await context.close();
  }
  const desktopContext = await browser.newContext({ viewport: { width: 1280, height: 900 }, locale: 'fa-IR', reducedMotion: 'reduce' });
  const desktopPage = await desktopContext.newPage();
  await desktopPage.goto(`${url}/scripts/fixtures/theia-preview.html#/welcome`);
  await desktopPage.screenshot({ path: `${artifacts}/desktop.png` });
  await desktopContext.close();
  // Interaction smoke: the rebrand must not disturb the existing controls.
  const smokeContext = await browser.newContext({ viewport: { width: 390, height: 844 }, locale: 'fa-IR', reducedMotion: 'reduce' });
  const smokePage = await smokeContext.newPage();
  smokePage.on('pageerror', error => errors.push(error.message));
  await smokePage.goto(`${url}/scripts/fixtures/theia-preview.html#/welcome`);
  await smokePage.getByRole('button', { name: 'شروع' }).click();
  await smokePage.locator('.privacy-screen').waitFor();
  await smokePage.goto(`${url}/scripts/fixtures/theia-preview.html#/approach`);
  const styleBefore = await smokePage.locator('#approach-style').textContent();
  await smokePage.locator('.approach-stage').focus();
  await smokePage.keyboard.press('ArrowLeft');
  assert.notEqual(await smokePage.locator('#approach-style').textContent(), styleBefore, 'approach keyboard selection');
  await smokePage.goto(`${url}/scripts/fixtures/theia-preview.html#/voice`);
  await smokePage.locator('.voice-carousel').focus();
  await smokePage.keyboard.press('ArrowLeft');
  assert.equal(await smokePage.locator('.voice-card.is-active').count(), 1, 'voice keyboard selection');
  await smokePage.goto(`${url}/scripts/fixtures/theia-preview.html#/mood`);
  await smokePage.locator('.mood-slider').fill('4');
  assert.equal(await smokePage.locator('.mood-slider').inputValue(), '4', 'mood slider');
  await smokePage.goto(`${url}/scripts/fixtures/theia-preview.html#/home`);
  await smokePage.getByRole('button', { name: 'شروع گفت‌وگوی تازه' }).click();
  await smokePage.locator('.chat-screen').waitFor();
  await smokeContext.close();
  assert.deepEqual(errors, [], 'no runtime or asset errors');
  console.log(`PASS: Theia brand/RTL/artwork at 360/390/430/desktop across ${routes.length} screens, plus interaction smoke.`);
} finally { await browser.close(); }
