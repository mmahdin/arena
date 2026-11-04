/** Match the existing designer's screen surfaces, including the dark chat shell.
 * Keep this as presentation-only state; routing and profile permissions stay in App.
 */
const DARK_ROUTES = new Set(['home', 'survey', 'day', 'letter', 'history', 'chat', 'insight']);
const BLACK_ROUTES = new Set(['menu', 'research', 'experts', 'simple-intro', 'deeper-intro', 'know-intro', 'almost-intro', 'voice']);
const BEIGE_ROUTES = new Set(['testimonials', 'interests', 'approach']);

export function screenSurfaceForRoute(route) {
  if (BLACK_ROUTES.has(route)) return 'black';
  if (DARK_ROUTES.has(route)) return 'dark';
  if (BEIGE_ROUTES.has(route)) return 'beige';
  return 'light';
}
