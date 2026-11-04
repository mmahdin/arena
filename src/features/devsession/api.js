import { request } from '../../core/api/http.js';
import { PROFILE_STYLES } from '../profile/preferences.js';

export const devSessionApi = {
  start(phone, signal) {
    return request('/api/v1/dev/identity-session', { method: 'POST', body: { phone }, signal });
  },
  end(signal) {
    return request('/api/v1/dev/session', { method: 'DELETE', signal });
  },
  async me(signal) {
    const result = await request('/api/v1/me', { signal });
    if (!result || typeof result.userId !== 'string' || typeof result.displayName !== 'string' ||
        !PROFILE_STYLES.includes(result.conversationStyle) ||
        !Object.hasOwn(result, 'onboardingCompletedAt')) {
      throw new Error('Invalid profile response');
    }
    return result;
  },
  async saveOnboardingName(displayName, signal) {
    const result = await request('/api/v1/me/onboarding-name', {
      method: 'PATCH', body: { displayName }, signal,
    });
    if (!result || typeof result.userId !== 'string' || result.displayName !== displayName ||
        result.onboardingCompletedAt !== null) {
      throw new Error('Invalid onboarding name response');
    }
    return result;
  },
  async profile(displayName, conversationStyle, signal) {
    const result = await request('/api/v1/me/profile', {
      method: 'PATCH', body: { displayName, conversationStyle }, signal,
    });
    if (!result || typeof result.userId !== 'string' || typeof result.displayName !== 'string' ||
        !PROFILE_STYLES.includes(result.conversationStyle) ||
        !result.onboardingCompletedAt) throw new Error('Invalid updated profile response');
    return result;
  },
};
