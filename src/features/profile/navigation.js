import { FLOW } from '../../data.js';

export const MAIN_ROUTES = Object.freeze(['home', 'survey', 'day', 'history', 'menu', 'chat', 'insight']);
export const VALID_ROUTES = new Set([...FLOW, ...MAIN_ROUTES]);
export const ONBOARDING_ROUTES = Object.freeze(FLOW.filter((route) => route !== 'home'));
// Keep the original 24 ordinals immutable for hamrah-v1 server cursors.
export const EXCLUDED_ONBOARDING_ROUTES = Object.freeze(['testimonials', 'research', 'experts']);
export const ACTIVE_ONBOARDING_ROUTES = Object.freeze(ONBOARDING_ROUTES.filter(route => !EXCLUDED_ONBOARDING_ROUTES.includes(route)));
export const ONBOARDING_SET = new Set(ACTIVE_ONBOARDING_ROUTES);

export function visibleOnboardingRoute(route) {
  const index = ONBOARDING_ROUTES.indexOf(route);
  if (index < 0) return route;
  return ONBOARDING_ROUTES.slice(index).find(step => ONBOARDING_SET.has(step));
}
export function previousOnboardingRoute(route) {
  const index = ONBOARDING_ROUTES.indexOf(route);
  return ONBOARDING_ROUTES.slice(0, Math.max(0, index)).filter(step => ONBOARDING_SET.has(step)).at(-1) ?? null;
}

/** Incomplete profiles see enabled onboarding screens, not protected main routes. */
export function canOpenRoute(route, onboardingComplete) {
  return VALID_ROUTES.has(route) && (onboardingComplete || ONBOARDING_SET.has(route));
}

export function initialRoute(onboardingComplete) {
  return onboardingComplete ? 'home' : ONBOARDING_ROUTES[0];
}

/** Prevent deep links from bypassing the unfinished local onboarding journey. */
export function canVisitOnboarding(route, furthestIndex) {
  const index = ONBOARDING_ROUTES.indexOf(route);
  return ONBOARDING_SET.has(route) && index >= 0 && index <= furthestIndex;
}

/** Only the next step in the original order may advance the journey. Revisits are allowed. */
export function onboardingMove(from, to, furthestIndex) {
  const current = ONBOARDING_ROUTES.indexOf(from);
  const target = ONBOARDING_ROUTES.indexOf(to);
  if (current < 0 || target < 0 || !ONBOARDING_SET.has(from) || !ONBOARDING_SET.has(to)) return null;
  const nextVisible = visibleOnboardingRoute(ONBOARDING_ROUTES[current + 1]);
  if (target > furthestIndex && to !== nextVisible) return null;
  return Math.max(furthestIndex, target);
}
