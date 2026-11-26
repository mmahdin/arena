/** Match each route's actual surface, including the 2026 onboarding palette.
 * Keep this as presentation-only state; routing and profile permissions stay in App.
 */
const DARK_ROUTES = new Set(['home', 'survey', 'day', 'letter', 'history', 'saved', 'settings', 'chat', 'insight']);
const BLACK_ROUTES = new Set(['menu', 'research', 'experts']);
// Earthshine: one sky, eight scenes. The surface behind each transition is the
// top tone of that scene's sky, so the sky never fades out of the wrong night.
const INK_ROUTES = new Set(['welcome', 'privacy']);
const NIGHT_ROUTES = new Set([
  'simple-intro', 'name', 'age', 'identity', 'therapist',
  'deeper-intro', 'mood', 'concerns', 'focus', 'feelings',
]);
const DAWN_ROUTES = new Set([
  'know-intro', 'faith', 'neurodivergent', 'relationships', 'interests',
  'strengths', 'barriers', 'recharge',
]);
const PALE_ROUTES = new Set(['almost-intro', 'approach', 'voice', 'checkins', 'referral', 'ready']);
const BEIGE_ROUTES = new Set(['testimonials']);

export function screenSurfaceForRoute(route) {
  if (INK_ROUTES.has(route)) return 'ink';
  if (NIGHT_ROUTES.has(route)) return 'night';
  if (DAWN_ROUTES.has(route)) return 'dawn';
  if (PALE_ROUTES.has(route)) return 'pale';
  if (BLACK_ROUTES.has(route)) return 'black';
  if (BEIGE_ROUTES.has(route)) return 'beige';
  if (DARK_ROUTES.has(route)) return 'dark';
  return 'light';
}
