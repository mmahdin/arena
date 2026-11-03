/** Real local browser/API regression; no chat calls or saved preference changes. */
import assert from 'node:assert/strict';
import { chromium } from 'playwright';
import { mkdir } from 'node:fs/promises';
import { APPROACHES } from '../src/data.js';

const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
await mkdir('test-results/approach', { recursive: true });
try {
  for (const width of [360, 390, 430, 1280]) {
    const context = await browser.newContext({ viewport: { width, height: 844 }, hasTouch: true, locale: 'fa-IR' });
    const page = await context.newPage();
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    await page.goto('http://127.0.0.1:5173/index.staff.html#/approach');
    await page.locator('#dev-phone').fill('09000000967');
    await page.getByRole('button', { name: 'ادامه' }).last().click();
    await page.locator('.home-screen').waitFor();
    await page.locator('.home-menu-button').click();
    await page.getByRole('button', { name: 'ترجیحات' }).click();
    const stage = page.locator('.approach-stage');
    await stage.waitFor();
    await page.locator('.screen-transition').evaluate(async node => Promise.all(node.getAnimations().map(a => a.finished)));
    const selected = async index => {
      assert.equal(await page.getByRole('button', { name: `گزینه ${index + 1}`, exact: true }).getAttribute('aria-current'), 'true');
      assert.equal((await page.locator('.approach-response').textContent()).trim(), APPROACHES[index].response.trim());
    };
    // Sample every animation frame: changing content must never displace the card.
    const offsets = await page.evaluate(async () => {
      const card = document.querySelector('.approach-card');
      const dots = [...document.querySelectorAll('.approach-screen .pager-dots button')];
      const samples = [];
      for (const index of [0, 4, 1, 3, 2]) {
        dots[index].click();
        const start = performance.now();
        while (performance.now() - start < 300) {
          await new Promise(requestAnimationFrame);
          const bounds = card.getBoundingClientRect();
          const stage = document.querySelector('.approach-stage').getBoundingClientRect();
          samples.push(Math.abs(bounds.x + bounds.width / 2 - stage.x - stage.width / 2));
          if (card !== document.querySelector('.approach-card')) throw new Error('card remounted');
        }
      }
      return samples;
    });
    assert.ok(Math.max(...offsets) < 1, `card jumps off center at ${width}px`);
    await selected(2);
    // Fast changes resolve to the latest choice without queued animation state.
    await page.evaluate(() => {
      const dots = document.querySelectorAll('.approach-screen .pager-dots button');
      for (const index of [4, 0, 3, 2]) dots[index].click();
    });
    await selected(2);
    await page.getByRole('button', { name: 'گزینه 2', exact: true }).click();
    await selected(1);
    await stage.focus();
    await page.keyboard.press('ArrowLeft');
    await selected(2);
    await page.keyboard.press('Home');
    await page.keyboard.press('ArrowRight');
    await selected(0);
    await page.keyboard.press('End');
    await page.keyboard.press('ArrowLeft');
    await selected(4);
    await page.getByRole('button', { name: 'سبک قبلی', exact: true }).click();
    await selected(3);
    await page.getByRole('button', { name: 'سبک بعدی', exact: true }).click();
    await selected(4);
    await page.getByRole('button', { name: 'گزینه 3', exact: true }).click();
    const box = await stage.boundingBox();
    const x = box.x + box.width / 2;
    const y = box.y + 160;
    await page.mouse.move(x, y);
    await page.mouse.down();
    await page.mouse.move(x - 100, y, { steps: 8 });
    await page.mouse.up();
    await selected(3);
    const cdp = await context.newCDPSession(page);
    const touch = async (dx, dy = 0, cancel = false) => {
      await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y }] });
      for (let step = 1; step <= 5; step++) await cdp.send('Input.dispatchTouchEvent', {
        type: 'touchMove', touchPoints: [{ x: x + dx * step / 5, y: y + dy * step / 5 }],
      });
      await cdp.send('Input.dispatchTouchEvent', { type: cancel ? 'touchCancel' : 'touchEnd', touchPoints: [] });
    };
    await touch(100);
    await selected(2);
    await touch(-10); // tap/small movement is not a swipe
    await selected(2);
    await touch(5, 100); // vertical movement is not a selection
    await selected(2);
    await touch(-100, 0, true);
    await selected(2);
    await page.waitForTimeout(300);
    assert.equal(await page.locator('.approach-card').evaluate(node => node.style.getPropertyValue('--approach-drag')), '0px');
    await page.screenshot({ path: `test-results/approach/${width}.png` });
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.getByRole('button', { name: 'گزینه 5', exact: true }).click();
    await selected(4);
    assert.ok(await page.locator('.approach-response').evaluate(node => parseFloat(getComputedStyle(node).animationDuration) < 0.001));
    assert.deepEqual(errors, []);
    await context.close();
  }
  console.log('PASS: centered card throughout transitions, rapid selection, keyboard, side buttons, mouse drag, touch swipe/cancel, and reduced motion at 360/390/430/1280px.');
} finally { await browser.close(); }
