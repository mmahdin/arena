/**
 * Pure astronomical geometry and storyboard scene definitions for Theia's
 * Earthshine onboarding ("One subject, one variable, one warm light").
 */

export function clampAperture(value) {
  const num = Number(value);
  if (!Number.isFinite(num)) return 0.22;
  return Math.max(0.08, Math.min(0.92, num));
}

/**
 * Astronomically faithful waning crescent path (lit on the lower-right / eastern
 * morning limb, facing a rising Sun below the right-hand horizon).
 *
 * `phase` in (0, 0.45): fraction of radius separating the outer circular limb
 * from the inner elliptical terminator arc, rotated by `tiltDeg`.
 */
export function crescentPath(cx, cy, r, phase = 0.21, tiltDeg = 23) {
  const rad = (tiltDeg * Math.PI) / 180;
  const sin = Math.sin(rad);
  const cos = Math.cos(rad);
  const tx = cx + r * sin;
  const ty = cy - r * cos;
  const bx = cx - r * sin;
  const by = cy + r * cos;
  const innerRx = Math.max(0.5, r * (1 - phase * 2));
  const f = (n) => n.toFixed(2);
  return [
    `M ${f(tx)} ${f(ty)}`,
    `A ${f(r)} ${f(r)} 0 0 1 ${f(bx)} ${f(by)}`,
    `A ${f(innerRx)} ${f(r)} ${tiltDeg} 0 0 ${f(tx)} ${f(ty)}`,
    'Z',
  ].join(' ');
}

const NIGHT_STARS = [
  { x: 0.16, y: 0.12, r: 0.9, opacity: 0.56, delay: '0.2s' },
  { x: 0.81, y: 0.16, r: 0.85, opacity: 0.48, delay: '0.9s' },
  { x: 0.27, y: 0.41, r: 0.72, opacity: 0.38, delay: '1.5s' },
  { x: 0.73, y: 0.37, r: 1.0, opacity: 0.58, delay: '0.6s' },
  { x: 0.12, y: 0.57, r: 0.78, opacity: 0.34, delay: '2.1s' },
  { x: 0.86, y: 0.52, r: 0.75, opacity: 0.40, delay: '1.2s' },
  { x: 0.59, y: 0.11, r: 0.68, opacity: 0.32, delay: '1.8s' },
];

export const EARTHSHINE_SCENES = {
  welcome: {
    key: 'welcome',
    label: 'آسمان شب و هلال باریک ماه پیش از سپیده',
    skyStops: [
      [0, '#05070D'],
      [0.58, '#080C15'],
      [1, '#0A0F1A'],
    ],
    lowerSurface: '#0A0F1A',
    moonCx: 0.5,
    moonCy: 0.305,
    moonR: 0.088,
    phase: 0.19,
    tiltDeg: 23,
    earthshine: 0.085,
    halo: 0.11,
    maria: 0.16,
    reversed: false,
    stars: [],
    horizon: false,
  },
  privacy: {
    key: 'privacy',
    label: 'دریچهٔ باریک رو به هلال ماه در آسمان شب',
    skyStops: [
      [0, '#070D19'],
      [0.55, '#0C172B'],
      [1, '#0F1D34'],
    ],
    lowerSurface: '#05070C',
    shutterColor: '#04060A',
    moonCx: 0.5,
    moonCy: 0.36,
    moonR: 0.098,
    phase: 0.20,
    tiltDeg: 23,
    earthshine: 0.20,
    halo: 0.095,
    maria: 0.25,
    reversed: false,
    stars: [],
    horizon: false,
    shutter: true,
  },
  night: {
    key: 'night',
    label: 'آسمان لاجوردی شب با چند ستارهٔ کم‌سو و طرح کمرنگ قرص ماه',
    skyStops: [
      [0, '#060C1B'],
      [0.52, '#0C1832'],
      [1, '#11213F'],
    ],
    lowerSurface: '#11213F',
    moonCx: 0.5,
    moonCy: 0.31,
    moonR: 0.094,
    phase: 0.205,
    tiltDeg: 23,
    earthshine: 0.22,
    halo: 0.095,
    maria: 0.30,
    reversed: false,
    stars: NIGHT_STARS,
    horizon: false,
  },
  seeing: {
    key: 'seeing',
    label: 'نمای نزدیک ماه با زمین‌تاب روشن‌تر و دریاهای سطح ماه',
    skyStops: [
      [0, '#081224'],
      [0.54, '#0F1F3B'],
      [1, '#152948'],
    ],
    lowerSurface: '#152948',
    moonCx: 0.5,
    moonCy: 0.37,
    moonR: 0.195,
    phase: 0.215,
    tiltDeg: 23,
    earthshine: 0.55,
    halo: 0.055,
    maria: 0.72,
    reversed: false,
    stars: [],
    horizon: false,
  },
  dawn: {
    key: 'dawn',
    label: 'نخستین خط افق و روشنای آرام صبح کاذب زیر ماه',
    skyStops: [
      [0, '#0B162A'],
      [0.50, '#162640'],
      [0.74, '#2A344B'],
      [0.82, '#6E6166'],
      [1, '#6E6166'],
    ],
    horizonY: 0.82,
    seaTop: '#0D1626',
    seaBottom: '#0A1220',
    lowerSurface: '#0A1220',
    horizonLine: 'rgba(186, 162, 158, 0.36)',
    falseDawn: true,
    moonCx: 0.5,
    moonCy: 0.295,
    moonR: 0.098,
    phase: 0.215,
    tiltDeg: 23,
    earthshine: 0.74,
    halo: 0.048,
    maria: 0.76,
    reversed: false,
    stars: [],
    horizon: true,
  },
  threeLights: {
    key: 'threeLights',
    label: 'سه نور: ماه کامل در زمین‌تاب، روشنای سپیده، و نخستین بارقهٔ خورشید بر افق',
    skyStops: [
      [0, '#1A2D49'],
      [0.44, '#2E4668'],
      [0.70, '#586D88'],
      [0.805, '#B59B82'],
      [0.84, '#DEBA8E'],
      [1, '#DEBA8E'],
    ],
    horizonY: 0.84,
    seaTop: '#132033',
    seaBottom: '#0E1929',
    lowerSurface: '#0E1929',
    horizonLine: 'rgba(236, 198, 148, 0.62)',
    solarRim: { x: 0.62 },
    moonCx: 0.5,
    moonCy: 0.265,
    moonR: 0.082,
    phase: 0.215,
    tiltDeg: 23,
    earthshine: 1.0,
    halo: 0.038,
    maria: 0.86,
    reversed: false,
    stars: [],
    horizon: true,
  },
  checkins: {
    key: 'checkins',
    label: 'آسمان بامداد با قرص کامل ماه و نقطهٔ آرام زهره در افق',
    skyStops: [
      [0, '#223652'],
      [0.48, '#3B5474'],
      [0.74, '#667B95'],
      [0.83, '#A99B8E'],
      [1, '#A99B8E'],
    ],
    horizonY: 0.83,
    seaTop: '#17263B',
    seaBottom: '#121E30',
    lowerSurface: '#121E30',
    horizonLine: 'rgba(214, 189, 162, 0.42)',
    venus: 0.96,
    venusPos: { x: 0.68, y: 0.56 },
    moonCx: 0.5,
    moonCy: 0.29,
    moonR: 0.092,
    phase: 0.215,
    tiltDeg: 23,
    earthshine: 1.0,
    halo: 0.035,
    maria: 0.86,
    reversed: false,
    stars: [],
    horizon: true,
  },
  final: {
    key: 'final',
    label: 'گرگ‌ومیش روشن صبحگاهی؛ همان ماهِ آغاز در آسمانی روشن',
    skyStops: [
      [0, '#A5B7CC'],
      [0.48, '#BDCDDD'],
      [0.75, '#D5DCE2'],
      [0.83, '#E7D5C4'],
      [1, '#E7D5C4'],
    ],
    horizonY: 0.83,
    seaTop: '#AEBCCC',
    seaBottom: '#BCC7D4',
    lowerSurface: '#BCC7D4',
    horizonLine: 'rgba(208, 158, 108, 0.58)',
    venus: 0.44,
    venusPos: { x: 0.68, y: 0.56 },
    moonCx: 0.5,
    moonCy: 0.305,
    moonR: 0.088,
    phase: 0.19,
    tiltDeg: 23,
    earthshine: 1.0,
    halo: 0.02,
    maria: 0.82,
    reversed: true,
    stars: [],
    horizon: true,
  },
};
