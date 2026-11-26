/** Match the designer's screen surfaces, including the dark chat shell.
 * Keep this as presentation-only state; routing and profile permissions stay in App.
 *
 * `night` is the onboarding identity surface: the four chapters of the light
 * narrative (night → dawn → sun → sight), plus the privacy, name, check-ins and
 * ready screens that sit between them. It exists so a dark page never fades in
 * from a light shell.
 */
const DARK_ROUTES = new Set(['home', 'survey', 'day', 'letter', 'history', 'saved', 'settings', 'chat', 'insight']);
const BLACK_ROUTES = new Set(['menu', 'research', 'experts', 'voice']);
const NIGHT_ROUTES = new Set([
  'welcome', 'privacy', 'name', 'checkins', 'ready',
  'simple-intro', 'deeper-intro', 'know-intro', 'almost-intro',
]);
const BEIGE_ROUTES = new Set(['testimonials', 'interests', 'approach']);

export function screenSurfaceForRoute(route) {
  if (NIGHT_ROUTES.has(route)) return 'night';
  if (BLACK_ROUTES.has(route)) return 'black';
  if (DARK_ROUTES.has(route)) return 'dark';
  if (BEIGE_ROUTES.has(route)) return 'beige';
  return 'light';
}
