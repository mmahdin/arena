import { FLOW, LEGACY_INFO_ROUTES } from '../../data.js';

export const MAIN_ROUTES = Object.freeze(['home', 'survey', 'history', 'menu', 'chat', 'letter']);
export const VALID_ROUTES = new Set([...FLOW, ...LEGACY_INFO_ROUTES, ...MAIN_ROUTES]);
export const ONBOARDING_ROUTES = Object.freeze(FLOW.filter((route) => route !== 'home'));
export const ACTIVE_ONBOARDING_ROUTES = ONBOARDING_ROUTES;
export const EXCLUDED_ONBOARDING_ROUTES = LEGACY_INFO_ROUTES;
export const ONBOARDING_SET = new Set(ONBOARDING_ROUTES);

export function visibleOnboardingRoute(route) {
  return ONBOARDING_SET.has(route) ? route : null;
}
export function previousOnboardingRoute(route) {
  const index = ONBOARDING_ROUTES.indexOf(route);
  return index > 0 ? ONBOARDING_ROUTES[index - 1] : null;
}

export function canOpenRoute(route, onboardingComplete) {
  return VALID_ROUTES.has(route) && (onboardingComplete || ONBOARDING_SET.has(route));
}
export function initialRoute(onboardingComplete) {
  return onboardingComplete ? 'home' : ONBOARDING_ROUTES[0];
}
export function canVisitOnboarding(route, furthestIndex) {
  const index = ONBOARDING_ROUTES.indexOf(route);
  return index >= 0 && index <= furthestIndex;
}
// Never trust a hash deep link to advance the server-backed cursor.
export function onboardingMove(from, to, furthestIndex) {
  const current = ONBOARDING_ROUTES.indexOf(from);
  const target = ONBOARDING_ROUTES.indexOf(to);
  if (current < 0 || target < 0) return null;
  if (target > furthestIndex && target !== current + 1) return null;
  return Math.max(furthestIndex, target);
}
