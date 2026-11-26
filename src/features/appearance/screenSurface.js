/**
 * Match each route's underlying surface so screen transitions never flash
 * an incompatible background behind outgoing/incoming frames.
 * Keep this as presentation-only state; routing and profile permissions stay in App.
 */
const DARK_ROUTES = new Set(['home', 'survey', 'day', 'letter', 'history', 'saved', 'settings', 'chat', 'insight']);
const BLACK_ROUTES = new Set(['menu', 'research', 'experts']);

// Earthshine onboarding progresses across one continuous cool night-to-morning sky,
// reversing tonal values into pale civil twilight only on the final Ready threshold.
const INK_ROUTES = new Set(['welcome', 'privacy']);
const NIGHT_ROUTES = new Set(['simple-intro', 'name', 'age', 'identity', 'therapist']);
const SEEING_ROUTES = new Set(['deeper-intro', 'mood', 'concerns', 'focus', 'feelings']);
const DAWN_ROUTES = new Set([
  'know-intro', 'faith', 'neurodivergent', 'relationships', 'interests',
  'strengths', 'barriers', 'recharge',
]);
const MORNING_ROUTES = new Set(['almost-intro', 'approach', 'voice', 'checkins', 'referral']);
const TWILIGHT_ROUTES = new Set(['ready']);
const BEIGE_ROUTES = new Set(['testimonials']);

export function screenSurfaceForRoute(route) {
  if (INK_ROUTES.has(route)) return 'ink';
  if (NIGHT_ROUTES.has(route)) return 'night';
  if (SEEING_ROUTES.has(route)) return 'seeing';
  if (DAWN_ROUTES.has(route)) return 'dawn';
  if (MORNING_ROUTES.has(route)) return 'morning';
  if (TWILIGHT_ROUTES.has(route)) return 'twilight';
  if (BLACK_ROUTES.has(route)) return 'black';
  if (BEIGE_ROUTES.has(route)) return 'beige';
  if (DARK_ROUTES.has(route)) return 'dark';
  return 'light';
}
