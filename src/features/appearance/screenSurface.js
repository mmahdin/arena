/** Match each route's actual surface, including the 2026 onboarding palette.
 * Keep this as presentation-only state; routing and profile permissions stay in App.
 */
const DARK_ROUTES = new Set(['home', 'survey', 'day', 'letter', 'history', 'saved', 'settings', 'chat', 'insight']);
const BLACK_ROUTES = new Set(['menu', 'research', 'experts']);
// The onboarding story alternates night ink and ivory paper.
const INK_ROUTES = new Set(['welcome', 'simple-intro', 'deeper-intro', 'know-intro', 'almost-intro', 'voice', 'approach', 'ready']);
const PAPER_ROUTES = new Set([
  'privacy', 'name', 'age', 'identity', 'therapist', 'mood', 'concerns', 'focus', 'feelings',
  'faith', 'neurodivergent', 'relationships', 'interests', 'strengths', 'barriers',
  'recharge', 'checkins', 'referral',
]);
const BEIGE_ROUTES = new Set(['testimonials']);

export function screenSurfaceForRoute(route) {
  if (INK_ROUTES.has(route)) return 'ink';
  if (BLACK_ROUTES.has(route)) return 'black';
  if (PAPER_ROUTES.has(route)) return 'paper';
  if (BEIGE_ROUTES.has(route)) return 'beige';
  if (DARK_ROUTES.has(route)) return 'dark';
  return 'light';
}
