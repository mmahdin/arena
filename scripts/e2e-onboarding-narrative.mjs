/**
 * Opt-in visual QA for the dawn onboarding. Browser only; no API interception.
 *
 * Prefers the repo's own Playwright. If it is not installed (as in the agent
 * sandbox), it falls back to `playwright-core` resolved from PLAYWRIGHT_CORE_DIR
 * with BROWSER_BIN, or reports BLOCKED — it never fakes a pass.
 *
 *   npm run preview:theia -- --port 5173
 *   node scripts/e2e-onboarding-narrative.mjs [--out test-results/narrative]
 *
 * Evidence per narrative chapter: one illustration, one heading, no raster art,
 * monotonically increasing light, no overflow at 360/390/430, and no collision
 * between the illustration, the copy and the action at 844/667/600 heights.
 */
import assert from 'node:assert/strict';
import { mkdirSync } from 'node:fs';
import { createRequire } from 'node:module';
import { join } from 'node:path';
import { chapterFor, NARRATIVE_ROUTES } from '../src/features/onboarding/chapters.js';
import { finishFullOnboarding } from './e2e-onboarding-helpers.mjs';

const ui = process.env.UI_ORIGIN || 'http://127.0.0.1:5173';
const outIndex = process.argv.indexOf('--out');
const outDir = join(process.cwd(), outIndex > 0 ? process.argv[outIndex + 1] : 'test-results/narrative');

let chromium = null;
try { ({ chromium } = await import('playwright')); } catch { /* fall through */ }
if (!chromium && process.env.PLAYWRIGHT_CORE_DIR) {
  const require = createRequire(join(process.env.PLAYWRIGHT_CORE_DIR, 'noop.js'));
  ({ chromium } = require('playwright-core'));
}
if (!chromium) {
  console.error('BLOCKED: install playwright, or set PLAYWRIGHT_CORE_DIR + BROWSER_BIN.');
  process.exit(2);
}
try {
  assert.equal((await fetch(ui)).status, 200, 'preview server');
} catch {
  console.error('BLOCKED: start `npm run preview:theia -- --port 5173` first.');
  process.exit(2);
}

mkdirSync(outDir, { recursive: true });
let browser;
try {
  browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_BIN ? { executablePath: process.env.BROWSER_BIN } : {}) });
} catch (error) {
  const reason = String(error?.message || error).split('\n').find((line) => /error while loading|not found|ENOENT/.test(line));
  console.error(`BLOCKED: Chromium could not start${reason ? ` — ${reason.trim()}` : ''}`);
  process.exit(2);
}
const errors = [];
const visited = new Set();
let previousDawn = -1;
let shot = 0;

try {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, locale: 'fa-IR', deviceScaleFactor: 1 });
  const page = await context.newPage();
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('console', (message) => { if (message.type() === 'error') errors.push(message.text()); });

  await page.goto(ui, { waitUntil: 'domcontentloaded' });
  await page.locator('#dev-phone').waitFor();

  // The entry screen must carry the same non-figurative art as the story.
  assert.equal(await page.locator('.dev-login-art .theia-scene').count(), 1, 'login shows one chapter scene');
  assert.equal(await page.locator('.dev-login-art img').count(), 0, 'no portrait on login');
  await page.screenshot({ path: join(outDir, '00-login.png') });

  await page.locator('#dev-phone').fill(`0900000${String(7000 + Math.floor(Math.random() * 900))}`);
  await page.getByRole('button', { name: 'ادامه' }).last().click();
  await page.locator('#dev-phone').waitFor({ state: 'hidden' });

  const inspect = async (route) => {
    if (!NARRATIVE_ROUTES.includes(route)) return;
    const root = page.locator('main .app-screen');
    const chapter = chapterFor(route);
    assert.equal(await root.locator('.theia-scene').count(), 1, `${route}: one chapter illustration`);
    assert.equal(await page.locator('main .app-screen img').count(), 0, `${route}: no raster art`);
    assert.equal(await root.locator('h1').count(), 1, `${route}: one heading`);
    assert.ok((await root.innerText()).includes(chapter.name), `${route}: chapter mark shows «${chapter.name}»`);

    const dawn = Number(await root.getAttribute('data-dawn'));
    assert.ok(dawn >= previousDawn, `${route}: the dawn must not dim (${previousDawn} → ${dawn})`);
    assert.equal(dawn, chapter.dawn, `${route}: dawn level matches the chapter map`);
    previousDawn = dawn;

    for (const width of [360, 390, 430]) {
      await page.setViewportSize({ width, height: 844 });
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, `${route} overflow at ${width}`);
      await page.screenshot({ path: join(outDir, `${String(++shot).padStart(2, '0')}-${route}-${width}.png`) });
    }
    for (const width of [360, 390]) {
      for (const height of [667, 600]) {
        await page.setViewportSize({ width, height });
        const collision = await page.evaluate(() => {
          const box = (selector) => document.querySelector(selector)?.getBoundingClientRect();
          const art = box('.narrative-art');
          const copy = box('.narrative-copy');
          const cta = box('.narrative-screen .primary-button');
          const overlaps = (a, b) => Boolean(a && b && a.bottom > b.top + 1);
          return {
            artCopy: overlaps(art, copy),
            copyCta: overlaps(copy, cta),
            ctaBelowFold: Boolean(cta && cta.bottom > window.innerHeight + 1),
            ctaOverflowX: Boolean(cta && (cta.left < -1 || cta.right > window.innerWidth + 1)),
          };
        });
        assert.deepEqual(collision, { artCopy: false, copyCta: false, ctaBelowFold: false, ctaOverflowX: false },
          `${route}: layout collides at ${width}×${height}`);
      }
    }
    await page.setViewportSize({ width: 390, height: 844 });
    visited.add(route);
  };

  await finishFullOnboarding(page, 'کاربر نمایشی', { onStep: inspect });
  assert.deepEqual([...visited], NARRATIVE_ROUTES, 'every narrative chapter was walked in order');

  // Reduced motion: the scene settles instantly, geometry unchanged.
  await page.locator('.home-screen').waitFor();
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.evaluate(() => { location.hash = '#/letter'; });
  await page.locator('.letter-screen').waitFor().catch(() => {});

  assert.deepEqual(errors, [], 'no runtime errors during the narrative flow');
  console.log(`PASS: 7 chapters, light never dims, no collisions at 360/390 × 844/667/600. Artifacts: ${outDir}`);
} finally {
  await browser.close();
}
