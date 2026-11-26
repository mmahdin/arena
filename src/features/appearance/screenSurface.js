/** Match each route's actual surface, including the 2026 Earthshine palette.
 * Keep this as presentation-only state; routing and profile permissions stay in App.
 */
import { FLOW } from '../../data.js';

const DARK_ROUTES = new Set(['home', 'survey', 'day', 'letter', 'history', 'saved', 'settings', 'chat', 'insight']);
const BLACK_ROUTES = new Set(['menu', 'research', 'experts']);
// Every onboarding plate belongs to the same pre-dawn night, from near-black
// ink to civil twilight. No screen falls back to the old ivory paper surface.
const INK_ROUTES = new Set(FLOW.filter((route) => route !== 'home'));
const BEIGE_ROUTES = new Set(['testimonials']);

export function screenSurfaceForRoute(route) {
  if (INK_ROUTES.has(route)) return 'ink';
  if (BLACK_ROUTES.has(route)) return 'black';
  if (BEIGE_ROUTES.has(route)) return 'beige';
  if (DARK_ROUTES.has(route)) return 'dark';
  return 'light';
}
