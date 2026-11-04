import { request } from '../../core/api/http.js';
import { ONBOARDING_ROUTES } from './navigation.js';

/** Only a route index is sent. Sensitive questionnaire answers stay in React memory. */
export const FLOW_VERSION = 'hamrah-v2';
export function validateProgress(data) {
  if (!data || data.flowVersion !== FLOW_VERSION ||
      !Number.isInteger(data.furthestStep) || data.furthestStep < 0 ||
      data.furthestStep >= ONBOARDING_ROUTES.length) {
    throw new Error('Unsupported onboarding progress response');
  }
  return data.furthestStep;
}
export const onboardingProgressApi = {
  // Persist sequential server ordinals. Data remains client-only; no answers are sent.
  async advanceTo(step, fromStep, signal, onProgress = () => {}) {
    validateProgress({ furthestStep: step, flowVersion: FLOW_VERSION });
    validateProgress({ furthestStep: fromStep, flowVersion: FLOW_VERSION });
    let saved = fromStep;
    while (saved < step) {
      signal?.throwIfAborted();
      const requested = saved + 1;
      saved = await onboardingProgressApi.advance(requested, signal);
      signal?.throwIfAborted();
      if (saved < requested) throw new Error('Onboarding progress did not advance');
      onProgress(saved);
    }
    return saved;
  },
  async read(signal) {
    return validateProgress(await request('/api/v1/me/onboarding-progress', { signal }));
  },
  async advance(step, signal) {
    if (!Number.isInteger(step) || step < 0 || step >= ONBOARDING_ROUTES.length) {
      throw new Error('Invalid onboarding step');
    }
    return validateProgress(await request('/api/v1/me/onboarding-progress', {
      method: 'PUT', body: { furthestStep: step }, signal,
    }));
  },
};
