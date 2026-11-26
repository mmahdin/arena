/** Match each route's actual surface, including the 2026 onboarding palette.
 * Keep this as presentation-only state; routing and profile permissions stay in App.
 */
const DARK_ROUTES = new Set(['home', 'survey', 'day', 'letter', 'history', 'saved', 'settings', 'chat', 'insight']);
const BLACK_ROUTES = new Set(['menu', 'research', 'experts']);
// The onboarding holds one cool sky from Welcome through the final pale twilight.
const INK_ROUTES = new Set([
  'welcome', 'privacy', 'simple-intro', 'name', 'age', 'identity', 'therapist',
  'deeper-intro', 'mood', 'concerns', 'focus', 'feelings', 'know-intro', 'faith',
  'neurodivergent', 'relationships', 'interests', 'strengths', 'barriers', 'recharge',
  'almost-intro', 'approach', 'voice', 'checkins', 'referral',
]);
const DAWN_ROUTES = new Set(['ready']);
const BEIGE_ROUTES = new Set(['testimonials']);

export function screenSurfaceForRoute(route) {
  if (INK_ROUTES.has(route)) return 'ink';
  if (DAWN_ROUTES.has(route)) return 'dawn';
  if (BLACK_ROUTES.has(route)) return 'black';
  if (BEIGE_ROUTES.has(route)) return 'beige';
  if (DARK_ROUTES.has(route)) return 'dark';
  return 'light';
}
