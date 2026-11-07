/** Theia surface rhythm: dark story beats, parchment question screens, sand accents.
 * Presentation-only state; routing and profile permissions stay in App.
 */
const DARK_ROUTES = new Set(['home', 'chat', 'history', 'saved', 'survey', 'day', 'letter', 'insight', 'voice', 'mood', 'testimonials', 'experts']);
const BLACK_ROUTES = new Set(['welcome', 'menu', 'simple-intro', 'deeper-intro', 'know-intro', 'almost-intro']);
const BEIGE_ROUTES = new Set(['approach', 'interests']);

export function screenSurfaceForRoute(route) {
  if (BLACK_ROUTES.has(route)) return 'black';
  if (DARK_ROUTES.has(route)) return 'dark';
  if (BEIGE_ROUTES.has(route)) return 'beige';
  return 'light';
}
