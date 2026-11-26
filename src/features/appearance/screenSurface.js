/**
 * Which surface the shell paints *behind* the current screen. This keeps the
 * outgoing page from flashing a bright backdrop when moving into a dark route.
 * Presentation-only state; routing and profile permissions stay in App.
 *
 * The onboarding narrative chapters now live on the night surface, so the
 * transition between them feels like the same room changing light rather than
 * a series of unrelated cards. Question screens stay on light/beige, where the
 * reading contrast of long option lists is best.
 */
const DARK_ROUTES = new Set([
  'home', 'survey', 'day', 'letter', 'history', 'saved', 'settings', 'chat', 'insight',
  // Narrative chapters (see features/onboarding/chapters.js)
  'welcome', 'privacy', 'simple-intro', 'deeper-intro', 'know-intro', 'almost-intro',
  'ready', 'checkins', 'voice',
]);
const BLACK_ROUTES = new Set(['menu', 'research', 'experts']);
const BEIGE_ROUTES = new Set(['testimonials', 'interests', 'approach']);

export function screenSurfaceForRoute(route) {
  if (BLACK_ROUTES.has(route)) return 'black';
  if (DARK_ROUTES.has(route)) return 'dark';
  if (BEIGE_ROUTES.has(route)) return 'beige';
  return 'light';
}
