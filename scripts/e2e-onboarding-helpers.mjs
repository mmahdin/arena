/**
 * Walk the complete original illustrated onboarding against the real running API.
 * UI interactions only: no interception, synthetic labels, no persisted questionnaire data.
 */
import assert from 'node:assert/strict';
import { QUESTIONS } from '../src/data.js';
import { ACTIVE_ONBOARDING_ROUTES } from '../src/features/profile/navigation.js';

export async function finishFullOnboarding(page, displayName = 'کاربر ساختگی', { onStep = () => {} } = {}) {
  const identity = await page.evaluate(async () => {
    const res = await fetch('/api/v1/me', { credentials: 'include' });
    if (res.status !== 200) throw new Error(`GET /me returned ${res.status}`);
    return res.json();
  });
  if (identity.onboardingCompletedAt) {
    await page.locator('.home-screen').waitFor();
    return false;
  }
  await page.waitForFunction(() => document.querySelector('main .app-screen') && location.hash.startsWith('#/'));
  // Account must remain incomplete while traversing all 24 original pages.
  const steps = ACTIVE_ONBOARDING_ROUTES;
  assert.equal(steps.length, 21);
  const resumed = await page.evaluate(() => location.hash.slice(2));
  const first = steps.indexOf(resumed);
  assert.ok(first >= 0, 'resume at a designer onboarding screen');
  for (let i = first; i < steps.length; i++) {
    const step = steps[i];
    await page.waitForFunction(route => window.location.hash === `#/${route}`, step);
    await onStep(step);
    if (step === 'name') {
      await page.locator('.name-screen #name').fill(displayName);
      await page.locator('.name-screen button[type="submit"]').click();
    } else if (step in QUESTIONS) {
      const question = page.locator(`.question-${step}`);
      await question.waitFor();
      if (step === 'interests') {
        await question.locator('.skip-button').click();
      } else if (QUESTIONS[step].type === 'single') {
        const privateAnswer = question.getByRole('button', { name: 'ترجیح می‌دهم نگویم' });
        if (await privateAnswer.count()) await privateAnswer.click();
        else await question.locator('.choice-button').first().click();
      } else {
        await question.locator('.choice-button').first().click();
        await question.locator('.primary-button').click();
      }
    } else {
      if (step === 'ready') {
        const beforeFinal = await page.evaluate(async () => (await fetch('/api/v1/me', { credentials: 'include' })).json());
        assert.equal(beforeFinal.onboardingCompletedAt, null, 'profile must not complete before final Ready');
      }
      await page.locator('main .app-screen .primary-button').click();
    }
    if (step !== 'ready') await page.waitForFunction(route => window.location.hash === `#/${route}`, steps[i + 1]);
  }
  // A resumed late step has no draft name: use the product's Ready recovery path.
  if (await page.locator('.name-screen').count()) {
    await page.locator('#name').fill(displayName);
    await page.locator('.name-screen button[type="submit"]').click();
    await page.locator('.approach-screen').waitFor();
    await page.locator('.approach-screen .primary-button').click();
    await page.locator('.ready-screen .primary-button').click();
  }
  await page.locator('.home-screen').waitFor();
  const after = await page.evaluate(async () => (await fetch('/api/v1/me', { credentials: 'include' })).json());
  assert.equal(after.userId, identity.userId);
  assert.equal(after.displayName, displayName);
  assert.ok(after.onboardingCompletedAt, 'only final Ready persisted onboarding completion');
  return true;
}
