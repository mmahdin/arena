/**
 * Isolated browser regression for the real ApproachScreen component.
 * Starts its own synthetic Vite fixture; no API, tester session or backend is
 * involved. Real persistence remains covered by e2e-five-tones/profile-preferences.
 */
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';
import { APPROACHES, APPROACH_EXAMPLE_USER, FLOW } from '../src/data.js';
import { startApproachPreview } from './approach-preview.mjs';

const server = await startApproachPreview({ host: '127.0.0.1', port: Number(process.env.APPROACH_TEST_PORT || 5174) });
const port = server.httpServer.address().port;
const base = `http://127.0.0.1:${port}`;
const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
await mkdir('test-results/approach', { recursive: true });
const label = (index) => APPROACHES[index].label || APPROACHES[index].name;
const reply = (index) => (APPROACHES[index].exampleResponse || APPROACHES[index].response).trim();
const nextRoute = FLOW[FLOW.indexOf('approach') + 1];

try {
  for (const [width, height] of [[360, 844], [390, 844], [430, 844], [1280, 900], [360, 640], [390, 568]]) {
    const context = await browser.newContext({ viewport: { width, height }, hasTouch: true, locale: 'fa-IR' });
    const page = await context.newPage();
    const errors = [];
    const apiCalls = [];
    page.on('pageerror', error => errors.push(error.message));
    page.on('request', request => { if (request.url().includes('/api/v1/')) apiCalls.push(request.url()); });
    await page.goto(base);
    await page.locator('.approach-screen').waitFor();
    await page.evaluate(() => document.fonts.ready);
    await page.locator('.screen-transition').evaluate(node => Promise.all(node.getAnimations().map(animation => animation.finished)));
    const stage = page.locator('.approach-stage');
    const styleName = page.locator('#approach-style');
    const choose = async index => {
      await stage.focus();
      await page.keyboard.press('Home');
      for (let step = 0; step < index; step++) await page.keyboard.press('ArrowLeft');
    };
    const selected = async index => {
      assert.equal(await styleName.textContent(), label(index));
      assert.equal(await page.locator('.approach-position span').count(), APPROACHES.length);
      assert.equal(await page.locator('.approach-position .is-active').count(), 1);
      assert.equal(await page.locator('.approach-position span').nth(index).getAttribute('class'), 'is-active');
      assert.equal((await page.locator('.approach-reply.is-active .approach-response').textContent()).trim(), reply(index));
      assert.equal(await page.locator('.approach-reply.is-active').count(), 1);
      assert.equal(await page.locator('.approach-reply[aria-hidden="false"]').count(), 1);
    };
    await selected(2);
    assert.equal(await page.getByRole('combobox').count(), 0, 'style selection should not use a dropdown');
    assert.equal(await page.locator('.approach-picker select, .approach-picker p').count(), 0);
    assert.equal(await page.locator('.approach-direction').count(), 2);
    assert.equal(await page.locator('#approach-gesture-hint').count(), 0);
    const previous = page.getByRole('button', { name: 'سبک قبلی', exact: true });
    const next = page.getByRole('button', { name: 'سبک بعدی', exact: true });
    for (const button of [previous, next]) {
      const box = await button.boundingBox();
      assert.ok(box.width >= 44 && box.height >= 44);
      assert.equal(await button.evaluate(node => getComputedStyle(node).backgroundColor), 'rgba(0, 0, 0, 0)');
    }
    await next.click();
    await selected(3);
    await previous.click();
    await selected(2);
    // Style selection must be directly above the chat preview, not detached
    // down by the bottom actions; the label stays readable through a swipe.
    assert.ok(await page.evaluate(() => {
      const picker = document.querySelector('.approach-picker').getBoundingClientRect();
      const prompt = document.querySelector('.approach-message--user').getBoundingClientRect();
      const reply = document.querySelector('.approach-replies').getBoundingClientRect();
      const actions = document.querySelector('.approach-actions').getBoundingClientRect();
      return picker.bottom < prompt.top && prompt.bottom < reply.top && reply.bottom < actions.top;
    }), 'style selector should sit immediately above the example conversation');
    assert.equal((await page.locator('.approach-message--user').textContent()).trim(), APPROACH_EXAMPLE_USER);
    // Pagination belongs below the conversation; the preview stays borderless.
    assert.ok(await page.evaluate(() => {
      const stage = document.querySelector('.approach-stage');
      const reply = document.querySelector('.approach-replies').getBoundingClientRect();
      const dots = document.querySelector('.approach-position').getBoundingClientRect();
      const style = getComputedStyle(stage);
      return dots.top > reply.bottom && dots.bottom <= stage.getBoundingClientRect().bottom
        && style.backgroundColor === 'rgba(0, 0, 0, 0)'
        && parseFloat(style.borderTopWidth) === 0;
    }), 'the dots should sit under the conversation without an outer card');
    for (const removed of ['.approach-tabs', '.approach-preview-badge', '.approach-swipe-hint', '.approach-author', '.approach-description', '.approach-change-note', '.approach-example-label', '.approach-step', '.approach-picker .pager-dots']) {
      assert.equal(await page.locator(removed).count(), 0, `${removed} should not return`);
    }

    // The prompt, reply slot, picker and actions must not move on any frame.
    const offsets = await page.evaluate(async () => {
      const nodes = ['.approach-picker', '.approach-message--user', '.approach-replies', '.approach-actions'].map(selector => document.querySelector(selector));
      const before = nodes.map(node => node.getBoundingClientRect().toJSON());
      const stage = document.querySelector('.approach-stage');
      const choose = index => {
        stage.dispatchEvent(new KeyboardEvent('keydown', { key: 'Home', bubbles: true }));
        for (let step = 0; step < index; step++) stage.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }));
      };
      const samples = [];
      for (const index of [0, 4, 1, 3, 2]) {
        choose(index);
        const start = performance.now();
        while (performance.now() - start < 360) {
          await new Promise(requestAnimationFrame);
          nodes.forEach((node, i) => {
            const box = node.getBoundingClientRect();
            samples.push(Math.abs(box.x - before[i].x), Math.abs(box.y - before[i].y), Math.abs(box.width - before[i].width), Math.abs(box.height - before[i].height));
          });
        }
      }
      return samples;
    });
    assert.ok(Math.max(...offsets) < 0.5, `layout jumps at ${width}×${height}`);
    await selected(2);
    await page.evaluate(() => {
      const stage = document.querySelector('.approach-stage');
      const choose = index => {
        stage.dispatchEvent(new KeyboardEvent('keydown', { key: 'Home', bubbles: true }));
        for (let step = 0; step < index; step++) stage.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }));
      };
      for (const index of [4, 0, 3, 2]) {
        choose(index);
      }
    });
    await selected(2);
    await page.waitForTimeout(380);
    await selected(2);

    await choose(1);
    await selected(1);
    await stage.focus();
    await page.keyboard.press('ArrowLeft');
    await selected(2);
    await page.keyboard.press('Home');
    await selected(0);
    assert.equal(await previous.isDisabled(), true);
    await page.keyboard.press('ArrowRight');
    await selected(0);
    await page.keyboard.press('End');
    await selected(4);
    assert.equal(await next.isDisabled(), true);
    await page.keyboard.press('ArrowLeft');
    await selected(4);
    await choose(2);

    const dotCenter = () => page.locator('.approach-position .is-active').evaluate(node => {
      const box = node.getBoundingClientRect();
      return box.x + box.width / 2;
    });
    const initialDotCenter = await dotCenter();
    // Adjacent dots must occupy the same physical side as adjacent replies,
    // independently of the surrounding Persian reading direction.
    assert.ok(await page.evaluate(() => {
      const dots = document.querySelectorAll('.approach-position span');
      const replies = document.querySelectorAll('.approach-reply');
      return dots[3].getBoundingClientRect().x > dots[2].getBoundingClientRect().x
        && replies[3].getBoundingClientRect().x > replies[2].getBoundingClientRect().x;
    }));
    const box = await stage.boundingBox();
    const x = box.x + box.width / 2;
    const y = box.y + box.height / 2;
    await page.mouse.move(x, y);
    await page.mouse.down();
    await page.mouse.move(x - 100, y, { steps: 8 });
    // During the actual drag, both panels must travel with the pointer.
    // A mere text swap after pointerup must never satisfy this regression.
    await selected(2);
    const inFlight = await page.evaluate(() => {
      const viewport = document.querySelector('.approach-replies').getBoundingClientRect();
      const current = document.querySelector('.approach-reply.is-active').getBoundingClientRect();
      const incoming = document.querySelectorAll('.approach-reply')[3].getBoundingClientRect();
      return {
        currentShift: current.left - viewport.left,
        incomingShift: incoming.left - viewport.left,
        viewportWidth: viewport.width,
        incomingVisible: incoming.left < viewport.right - 2 && incoming.right > viewport.left,
      };
    });
    assert.ok(Math.abs(inFlight.currentShift + 100) < 12, 'current reply should follow drag 1:1');
    assert.ok(Math.abs(inFlight.incomingShift - (inFlight.viewportWidth - 100)) < 12, 'neighbor should follow the same drag');
    assert.ok(inFlight.incomingVisible, 'neighbor must become visible before release');
    await page.mouse.up();
    await selected(3);
    assert.ok(await dotCenter() > initialDotCenter, 'the incoming right-hand reply must activate the dot on the right');
    await page.mouse.move(x, y);
    await page.mouse.down();
    await page.mouse.move(x + 100, y, { steps: 8 });
    await page.mouse.up();
    await selected(2);
    assert.ok(Math.abs(await dotCenter() - initialDotCenter) < 1, 'swiping back must restore the original dot position');

    const cdp = await context.newCDPSession(page);
    const touch = async (dx, dy = 0, cancel = false) => {
      await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y }] });
      for (let step = 1; step <= 5; step++) await cdp.send('Input.dispatchTouchEvent', {
        type: 'touchMove', touchPoints: [{ x: x + dx * step / 5, y: y + dy * step / 5 }],
      });
      await cdp.send('Input.dispatchTouchEvent', { type: cancel ? 'touchCancel' : 'touchEnd', touchPoints: [] });
      await page.waitForTimeout(380);
    };
    await touch(100);
    await selected(1);
    await touch(-100);
    await selected(2);
    await touch(-10);
    await selected(2);
    await touch(5, 100);
    await selected(2);
    await touch(-100, 0, true);
    await selected(2);
    assert.equal(await page.locator('.approach-reply.is-active').evaluate(node => node.style.getPropertyValue('--reply-drag')), '0px');

    // Short viewports and long Persian text remain scrollable, never clipped.
    const screen = page.locator('.approach-screen');
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth));
    await page.locator('.approach-actions').scrollIntoViewIfNeeded();
    const screenBox = await screen.boundingBox();
    const actionsBox = await page.locator('.approach-actions').boundingBox();
    assert.ok(actionsBox.y + actionsBox.height <= screenBox.y + screenBox.height + 1);
    await page.locator('.approach-reply.is-active .approach-response').evaluate(node => {
      node.textContent = 'این یک متن آزمایشی بلند فارسی برای بررسی شکستن خط، فاصله‌گذاری و دسترس‌پذیری دکمه‌ها است. '.repeat(8);
    });
    await page.locator('.approach-actions').scrollIntoViewIfNeeded();
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth));
    assert.ok(await page.locator('.approach-reply.is-active .approach-response').evaluate(node => node.scrollHeight <= node.clientHeight + 1));
    await page.reload();
    await page.locator('.approach-screen').waitFor();
    await page.evaluate(() => document.fonts.ready);
    await page.waitForTimeout(380);
    await page.screenshot({ path: `test-results/approach/${width}x${height}.png` });

    await page.emulateMedia({ reducedMotion: 'reduce' });
    await choose(4);
    await selected(4);
    const motion = await page.locator('.approach-reply.is-active').evaluate(node => ({ duration: getComputedStyle(node).transitionDuration, transform: getComputedStyle(node).transform }));
    assert.ok(motion.duration.split(',').every(value => parseFloat(value) < 0.001));
    assert.equal(motion.transform, 'none');

    // Onboarding callbacks still advance, skip and go back without saving early.
    await page.emulateMedia({ reducedMotion: 'no-preference' });
    await choose(1);
    await page.locator('.approach-actions .primary-button').click();
    await page.getByRole('status').filter({ hasText: `مسیر بعدی ${nextRoute}` }).waitFor();
    await page.locator('.approach-actions .skip-button').click();
    await page.getByRole('status').filter({ hasText: `مسیر بعدی ${nextRoute}` }).waitFor();
    await page.locator('.approach-screen > .back-button').click();
    await page.getByRole('status').filter({ hasText: 'بازگشت' }).waitFor();

    // Completed-profile mode keeps explicit save/cancel and a real busy state.
    await page.goto(`${base}/scripts/fixtures/approach-preview.html?edit=1`);
    await page.locator('.approach-screen').waitFor();
    await page.getByRole('button', { name: 'ذخیرهٔ سبک گفت‌وگو', exact: true }).click();
    assert.equal(await page.locator('.approach-actions .primary-button').isDisabled(), true);
    await page.getByRole('status').filter({ hasText: 'ذخیرهٔ سروری انجام نشد' }).waitFor();
    assert.equal(await page.locator('.approach-actions .primary-button').isDisabled(), false);
    await page.locator('.approach-actions .skip-button').click();
    await page.getByRole('status').filter({ hasText: 'بازگشت' }).waitFor();
    assert.deepEqual(apiCalls, [], 'the isolated preview must never call the backend');
    assert.deepEqual(errors, []);
    await context.close();
  }
  console.log('PASS: plain style name and unboxed directional controls, no visible instructions or dropdown, stable geometry, rapid selection, RTL keyboard, mouse/touch swipe, cancel, reduced motion, short/long text, onboarding and edit callbacks at 360/390/430/desktop and 568/640px heights.');
} finally {
  await browser.close();
  await server.close();
}
