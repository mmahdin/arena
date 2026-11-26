/**
 * Theia onboarding narrative — seven chapters of a single dawn.
 *
 * Theia (Θεία) is the Titaness of divine light: not a person who speaks, but the
 * condition in which anything can be seen at all. Her name is bound to *thea*,
 * "sight". Persian architecture carries the same idea without a figure: a dark
 * chamber, a pointed arch, and one aperture that lets the sky in. This file maps
 * every onboarding route to a beat of that quiet story, so the room slowly gains
 * light from the first screen to Ready.
 *
 * Ordering rule: chapter index (and therefore ambient light) must never go
 * backwards along FLOW. `assertChapterContinuity` encodes that promise for tests.
 */
import { FLOW } from '../../data.js';

/** Seven beats. `stage` drives the illustration, `dawn` (0–1) the ambient light. */
export const CHAPTERS = Object.freeze([
  {
    id: 'threshold',
    index: 0,
    numeral: '۰۱',
    name: 'آستانه',
    stage: 0,
    dawn: 0.04,
  },
  {
    id: 'veil',
    index: 1,
    numeral: '۰۲',
    name: 'پنجره',
    stage: 1,
    dawn: 0.12,
  },
  {
    id: 'aperture',
    index: 2,
    numeral: '۰۳',
    name: 'روزن',
    stage: 2,
    dawn: 0.24,
  },
  {
    id: 'reflection',
    index: 3,
    numeral: '۰۴',
    name: 'بازتاب',
    stage: 3,
    dawn: 0.38,
  },
  {
    id: 'outline',
    index: 4,
    numeral: '۰۵',
    name: 'طرح',
    stage: 4,
    dawn: 0.54,
  },
  {
    id: 'dawnline',
    index: 5,
    numeral: '۰۶',
    name: 'سپیده',
    stage: 5,
    dawn: 0.72,
  },
  {
    id: 'morning',
    index: 6,
    numeral: '۰۷',
    name: 'بامداد',
    stage: 6,
    dawn: 1,
  },
]);

/** The screens that carry a chapter illustration and a line of the story. */
export const NARRATIVE_ROUTES = Object.freeze([
  'welcome',
  'privacy',
  'simple-intro',
  'deeper-intro',
  'know-intro',
  'almost-intro',
  'checkins',
  'ready',
]);

const STAGE_BY_ROUTE = Object.freeze({
  welcome: 0,
  privacy: 1,
  'simple-intro': 2,
  name: 2,
  age: 2,
  identity: 2,
  therapist: 2,
  'deeper-intro': 3,
  mood: 3,
  concerns: 3,
  focus: 3,
  feelings: 3,
  'know-intro': 4,
  faith: 4,
  neurodivergent: 4,
  relationships: 4,
  interests: 4,
  strengths: 4,
  barriers: 4,
  recharge: 4,
  'almost-intro': 5,
  approach: 5,
  voice: 5,
  checkins: 5,
  referral: 5,
  ready: 6,
});

/** Chapter beat for a route; unknown routes fall back to the first beat. */
export function chapterFor(route) {
  const stage = STAGE_BY_ROUTE[route];
  return stage === undefined ? CHAPTERS[0] : CHAPTERS[stage];
}

/** Ambient light (0–1) for a route, used by the shell background. */
export function dawnFor(route) {
  return chapterFor(route).dawn;
}

export function isNarrativeRoute(route) {
  return NARRATIVE_ROUTES.includes(route);
}

/** True when the story never dims while walking FLOW forwards. */
export function assertChapterContinuity(flow = FLOW) {
  let previous = -1;
  const ordered = [];
  for (const route of flow) {
    if (!(route in STAGE_BY_ROUTE)) continue;
    const section = chapterFor(route);
    if (section.index < previous) return { ok: false, route, sections: ordered };
    previous = section.index;
    if (ordered.at(-1) !== section.id) ordered.push(section.id);
  }
  return { ok: true, sections: ordered };
}

/** Persian numerals, kept local so the narrative never depends on Intl output. */
export function faNumeral(value) {
  return String(value).replace(/\d/g, (digit) => '۰۱۲۳۴۵۶۷۸۹'[Number(digit)]);
}
