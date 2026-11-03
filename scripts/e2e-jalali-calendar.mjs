import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';
import { gregorianDateOnlyToJalali, jalaliDateToGregorianDateOnly, toFa } from '../src/components/date/jalaliDate.js';
import { localReflectionDate } from '../src/features/reflection/api.js';

const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
await mkdir('test-results/jalali', { recursive: true });
try {
  for (const width of [360, 390, 430, 1280]) {
    const context = await browser.newContext({ viewport: { width, height: 844 }, locale: 'fa-IR', timezoneId: 'Asia/Tehran' });
    const page = await context.newPage();
    const errors = [];
    const reads = [];
    page.on('pageerror', error => errors.push(error.message));
    page.on('request', request => { if (request.method() === 'GET' && /daily-reflections\/\d/.test(request.url())) reads.push(request.url()); });
    await page.goto('http://127.0.0.1:5173/index.staff.html#/day');
    await page.locator('#dev-phone').fill('09000000967');
    await page.getByRole('button', { name: 'ادامه' }).last().click();
    await page.locator('.home-screen').waitFor();
    await page.locator('.reflection-card').click();
    const trigger = page.getByRole('button', { name: 'انتخاب تاریخ تأمل از تقویم' });
    await trigger.click();
    const dialog = page.getByRole('dialog');
    await dialog.waitFor();
    const rect = await dialog.boundingBox();
    assert.ok(rect.x >= 0 && rect.x + rect.width <= width && rect.y >= 0 && rect.y + rect.height <= 844);
    assert.equal(await page.getByRole('button', { name: 'ماه بعد', exact: true }).isDisabled(), true);
    await page.getByRole('button', { name: /^انتخاب سال،/ }).click();
    await page.getByRole('group', { name: 'انتخاب سال شمسی' }).waitFor();
    assert.equal(await page.getByRole('button', { name: '۱۲ سال بعد' }).isDisabled(), true);
    await page.getByRole('button', { name: '۱۲ سال قبل' }).click();
    await page.getByRole('button', { name: '۱۲ سال بعد' }).click();
    const today = gregorianDateOnlyToJalali(localReflectionDate());
    await page.getByRole('button', { name: `سال ${toFa(today.slice(0, 4))}`, exact: true }).click();
    await page.getByRole('button', { name: 'ماه قبل', exact: true }).click();
    await page.screenshot({ path: `test-results/jalali/${width}.png` });
    for (let i = 0; i < 40; i++) {
      await page.keyboard.press('Tab');
      assert.equal(await dialog.evaluate(node => node.contains(document.activeElement)), true, 'focus escaped modal');
    }
    await page.keyboard.press('Escape');
    await dialog.waitFor({ state: 'hidden' });
    assert.equal(await trigger.evaluate(node => node === document.activeElement), true);
    await trigger.click();
    await page.getByRole('button', { name: 'ماه قبل', exact: true }).click();
    const firstDay = dialog.locator('.jdf-day:not(:disabled)').first();
    await firstDay.click();
    await dialog.waitFor({ state: 'hidden' });
    const parts = await Promise.all(['year', 'month', 'day'].map(part => page.locator(`#daily-reflection-date-${part}`).inputValue()));
    const iso = jalaliDateToGregorianDateOnly(parts.join('/'));
    assert.ok(iso && iso < localReflectionDate());
    await page.waitForFunction(() => !document.querySelector('#daily-reflection-text')?.disabled);
    assert.ok(reads.some(url => url.endsWith(`/daily-reflections/${iso}`)));
    const count = reads.length;
    await page.locator('#daily-reflection-date-month').fill('۱۳');
    await page.locator('#daily-reflection-date-day').focus();
    assert.equal(await page.locator('#daily-reflection-date-year').evaluate(node => node.validity.valid), false);
    assert.equal(reads.length, count, 'invalid date reached the API');
    await trigger.click();
    await page.getByRole('button', { name: 'انتخاب امروز', exact: true }).click();
    if (width === 390) {
      await page.locator('#daily-reflection-date-year').fill('۱۴۰۴');
      await page.locator('#daily-reflection-date-month').fill('۰۱');
      await page.locator('#daily-reflection-date-day').fill('۰۱');
      await page.waitForFunction(() => !document.querySelector('#daily-reflection-text')?.disabled);
      assert.ok(reads.some(url => url.endsWith('/daily-reflections/2025-03-21')), 'Persian typing did not request Nowruz');
      await page.locator('#daily-reflection-text').fill('پیش‌نویس ساختگی ذخیره‌نشده');
      await trigger.click();
      page.once('dialog', dialog => dialog.dismiss());
      await page.getByRole('button', { name: 'انتخاب امروز', exact: true }).click();
      assert.equal(await page.locator('#daily-reflection-date-year').inputValue(), '۱۴۰۴');
      assert.equal(await page.locator('#daily-reflection-text').inputValue(), 'پیش‌نویس ساختگی ذخیره‌نشده');
    }
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await trigger.click();
    await page.getByRole('button', { name: 'بستن تقویم' }).click();
    assert.deepEqual(errors, []);
    await context.close();
  }
  console.log('PASS: Persian calendar month/year navigation, ISO API dates, validation, focus trap/restore, keyboard and mobile/desktop layout.');
} finally { await browser.close(); }
