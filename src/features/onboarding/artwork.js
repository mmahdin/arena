/**
 * Theia's visual grammar, as geometry rather than a portrait.
 *
 * Every onboarding scene is the same room at a different hour: a pointed arch
 * (the Persian *tāq*), a horizon, and a single source of light. Nothing here is
 * figurative — no face, no body, no deity. Theia is *the condition of seeing*,
 * so the imagery shows light arriving into an enclosed, protected space. That is
 * also the therapeutic promise: a bounded place where what was in shadow can be
 * looked at without being exposed.
 *
 * The geometry is kept as plain data so the React renderer and the design
 * preview script produce identical drawings.
 */

export const SCENE_VIEWBOX = '0 0 240 300';

/** Pointed arch, drawn once and reused at three insets. Large, middle, small. */
export const ARCH = Object.freeze({
  outer: 'M44 288 V152 C44 96 86 68 120 38 C154 68 196 96 196 152 V288',
  band: 'M50 288 V150 C50 99 89 72 120 45 C151 72 190 99 190 150 V288',
  middle: 'M60 288 V156 C60 112 94 88 120 62 C146 88 180 112 180 156 V288',
  inner: 'M76 288 V160 C76 128 100 110 120 90 C140 110 164 128 164 160 V288',
  base: 'M28 288 H212',
  floor: 'M28 264 H212',
});

/** Night-to-dawn palette, shared with CSS custom properties. */
export const ART_COLORS = Object.freeze({
  night: '#0b0c11',
  nightLift: '#11131a',
  line: '#c9b48c',
  lineSoft: '#8d8471',
  dawn: '#edc28d',
  dawnBright: '#f8e6c6',
  moon: '#c2ccdf',
});

/**
 * Seven scenes. `dawn` (0–1) drives glow size and opacity; the booleans add
 * architectural detail so each chapter reads as a distinct hour of one night.
 */
export const SCENES = Object.freeze([
  {
    id: 'threshold',
    dawn: 0.04,
    plinth: true,
    band: false,
    // A closed door: only the hairline of the arch, and light that has not
    // arrived yet — one point, exactly on the threshold.
    horizonY: 232,
    disc: { y: 232, r: 3.2 },
    glow: 26,
    strokeOpacity: 0.34,
    innerOpacity: 0,
    stars: 0,
    moon: false,
    lattice: false,
    water: false,
    beams: false,
    rays: false,
    openDoor: false,
    doorSeam: true,
  },
  {
    id: 'veil',
    plinth: true,
    band: false,
    dawn: 0.12,
    // Privacy: a lattice screen (*mashrabiya*). You can see out; the world
    // cannot see in.
    horizonY: 208,
    disc: { y: 208, r: 4.6 },
    glow: 40,
    strokeOpacity: 0.42,
    innerOpacity: 0.16,
    stars: 3,
    moon: false,
    lattice: true,
    water: false,
    beams: false,
    rays: false,
    openDoor: false,
  },
  {
    id: 'aperture',
    plinth: true,
    band: false,
    dawn: 0.24,
    // The oculus: the first deliberate opening. A thin shaft of light.
    horizonY: 208,
    disc: { y: 208, r: 7 },
    glow: 58,
    strokeOpacity: 0.5,
    innerOpacity: 0.24,
    stars: 4,
    moon: true,
    lattice: false,
    water: false,
    beams: true,
    rays: false,
    openDoor: false,
  },
  {
    id: 'reflection',
    plinth: true,
    band: true,
    dawn: 0.38,
    // Reflection: light on still water. What is seen is also what is mirrored.
    horizonY: 208,
    disc: { y: 208, r: 15 },
    glow: 92,
    strokeOpacity: 0.56,
    innerOpacity: 0.32,
    stars: 5,
    moon: true,
    lattice: false,
    water: true,
    beams: false,
    rays: false,
    openDoor: false,
  },
  {
    id: 'outline',
    plinth: true,
    band: true,
    dawn: 0.54,
    // Outline: constellation lines. Scattered points become a shape once
    // connected — the work of naming what you feel.
    horizonY: 208,
    disc: { y: 208, r: 21 },
    glow: 120,
    strokeOpacity: 0.62,
    innerOpacity: 0.4,
    stars: 7,
    moon: false,
    lattice: false,
    water: false,
    beams: true,
    rays: false,
    openDoor: false,
    constellation: { offsetX: 26, offsetY: -18, scale: 0.62 },
  },
  {
    id: 'dawnline',
    plinth: true,
    band: true,
    dawn: 0.72,
    // The half-sun on the horizon line, before it clears the wall.
    horizonY: 208,
    disc: { y: 208, r: 30 },
    glow: 168,
    strokeOpacity: 0.68,
    innerOpacity: 0.5,
    stars: 3,
    moon: false,
    lattice: false,
    water: false,
    beams: true,
    rays: true,
    openDoor: false,
    halfSun: true,
  },
  {
    id: 'morning',
    plinth: true,
    band: true,
    dawn: 1,
    // The door stands open. Light is no longer an event, it is the room.
    horizonY: 232,
    disc: { y: 150, r: 22 },
    glow: 210,
    strokeOpacity: 0.78,
    innerOpacity: 0.62,
    stars: 0,
    moon: false,
    lattice: false,
    water: false,
    beams: true,
    rays: true,
    openDoor: true,
  },
]);

export function sceneForStage(stage) {
  const index = Number.isInteger(stage) ? Math.max(0, Math.min(SCENES.length - 1, stage)) : 0;
  return SCENES[index];
}

/** Theia glyph: an arch, a horizon, and a single point of light on the line. */
export const GLYPH = Object.freeze({
  viewBox: '0 0 100 100',
  arch: 'M50 12 C34 22 22 33 22 52 V84 H78 V52 C78 33 66 22 50 12 Z',
  horizon: 'M12 64 H88',
  disc: { x: 50, y: 64, r: 5.4 },
  halo: { x: 50, y: 64, r: 17 },
});

/** Loose constellation drawn only in the "outline" chapter. */
export const CONSTELLATION = Object.freeze({
  points: [
    [92, 150], [118, 132], [150, 146], [172, 124], [142, 176], [108, 186], [78, 196],
  ],
  links: [[0, 1], [1, 2], [2, 3], [1, 4], [4, 5], [5, 6], [0, 5]],
});
