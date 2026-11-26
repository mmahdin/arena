/** Match each route's actual surface, including the Earthshine onboarding palette.
 * Keep this as presentation-only state; routing and profile permissions stay in App.
 */
const DARK_ROUTES = new Set(['home', 'survey', 'day', 'letter', 'history', 'saved', 'settings', 'chat', 'insight']);
const BLACK_ROUTES = new Set(['menu', 'research', 'experts']);
// All eight Earthshine story scenes plus nocturnal voice/style previews sit on ink;
// questionnaire intake steps rest on calm limestone paper.
const INK_ROUTES = new Set([
  'welcome', 'privacy', 'simple-intro', 'deeper-intro', 'know-intro',
  'almost-intro', 'voice', 'approach', 'checkins', 'ready',
]);
const PAPER_ROUTES = new Set([
  'name', 'age', 'identity', 'therapist', 'mood', 'concerns', 'focus', 'feelings',
  'faith', 'neurodivergent', 'relationships', 'interests', 'strengths', 'barriers',
  'recharge', 'referral',
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
