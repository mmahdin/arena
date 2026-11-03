/** Composer regression in the real local UI; no message or model requests. */
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';

const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
await mkdir('test-results/composer', { recursive: true });
try {
  for (const width of [360, 390, 430, 1280]) {
    const page = await browser.newPage({ viewport: { width, height: 844 }, locale: 'fa-IR' });
    const errors = [];
    const turns = [];
    page.on('pageerror', error => errors.push(error.message));
    page.on('request', request => { if (request.url().endsWith('/turns')) turns.push(request.url()); });
    await page.goto('http://127.0.0.1:5173');
    await page.locator('#dev-phone').fill('09000000967');
    await page.getByRole('button', { name: 'ادامه', exact: true }).last().click();
    await page.locator('.home-screen').waitFor();
    await page.locator('.chat-fab').click();
    const field = page.getByRole('textbox', { name: 'متن پیام' });
    await field.waitFor();
    const small = (await field.boundingBox()).height;
    const text = 'این یک متن بلند ساختگی برای بررسی شکستن خط در ورودی پیام است. '.repeat(8);
    await field.fill(text);
    const large = await field.boundingBox();
    assert.ok(large.height > small && large.height <= 140);
    assert.equal(await field.evaluate(node => node.scrollWidth <= node.clientWidth + 1), true, 'horizontal overflow');
    assert.equal(await field.evaluate(node => node.scrollHeight > node.clientHeight), true, 'long text must scroll vertically at the cap');
    await field.fill('متن‌بلندبدونفاصله'.repeat(35));
    assert.equal(await field.evaluate(node => node.scrollWidth <= node.clientWidth + 1), true);
    await field.fill('خط اول');
    await field.press('Enter');
    await field.pressSequentially('خط دوم');
    assert.equal(await field.inputValue(), 'خط اول\nخط دوم');
    await field.press('Shift+Enter');
    await field.pressSequentially('خط سوم');
    assert.equal(await field.inputValue(), 'خط اول\nخط دوم\nخط سوم');
    await field.fill(text);
    await page.waitForFunction(() => {
      const messages = document.querySelector('.messages-scroll').getBoundingClientRect();
      const composer = document.querySelector('.composer-wrap').getBoundingClientRect();
      return messages.bottom <= composer.top + 1;
    });
    await page.screenshot({ path: `test-results/composer/${width}.png` });
    await page.setViewportSize({ width: 360, height: 640 });
    assert.equal(await field.inputValue(), text);
    await field.fill('');
    assert.equal((await field.boundingBox()).height, small, 'clearing must shrink the field');
    assert.deepEqual(turns, [], 'typing or Enter unexpectedly submitted');
    assert.deepEqual(errors, []);
    await page.close();
  }
  console.log('PASS: Persian/continuous text wraps, bounded auto-height, vertical scrolling, Enter newlines, resize/clear and unobscured message area at mobile/desktop widths. No model calls.');
} finally { await browser.close(); }
