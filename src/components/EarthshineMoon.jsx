import React from 'react';

/**
 * A small, deliberately astronomical visual language for onboarding. The Moon
 * never changes shape; only the visibility of its earthlit face changes.
 */
const SCENES = {
  welcome: { sky: 'night', radius: 54, earthshine: 0.045, halo: 0.22, surface: 0.07, stars: 0, horizon: false },
  privacy: { sky: 'night', radius: 54, earthshine: 0.08, halo: 0.12, surface: 0.1, stars: 0, horizon: false },
  night: { sky: 'ultramarine', radius: 58, earthshine: 0.2, halo: 0.1, surface: 0.18, stars: 8, horizon: false },
  seeing: { sky: 'deep', radius: 78, earthshine: 0.48, halo: 0.035, surface: 0.32, stars: 0, horizon: false },
  dawn: { sky: 'dawn', radius: 68, earthshine: 0.68, halo: 0.025, surface: 0.42, stars: 0, horizon: true },
  lights: { sky: 'pale', radius: 62, earthshine: 0.9, halo: 0.02, surface: 0.55, stars: 0, horizon: true },
  checkins: { sky: 'pale', radius: 58, earthshine: 0.92, halo: 0.015, surface: 0.56, stars: 0, horizon: true },
  final: { sky: 'twilight', radius: 54, earthshine: 0.98, halo: 0.01, surface: 0.6, stars: 0, horizon: true },
};

const STAR_POSITIONS = [
  [67, 73, 0.42], [104, 151, 0.28], [304, 60, 0.35], [328, 124, 0.25],
  [79, 208, 0.2], [283, 195, 0.31], [139, 47, 0.2], [346, 217, 0.18],
];

const SURFACE_MARKS = [
  'M155 105c7-7 17-9 23-3 5 5 4 12-2 16-8 5-20 2-22-5',
  'M214 116c8-4 17-2 19 4 2 6-4 12-12 12-7 0-11-7-7-12',
  'M178 145c6-5 13-4 16 1 3 5 0 10-6 11-8 1-13-6-10-12',
  'M220 151c5-3 10-1 11 3 1 4-3 7-8 7-5-1-6-6-3-10',
  'M151 132c4-3 8-2 9 1 1 3-2 6-5 6-4 0-6-4-4-7',
];

export function EarthshineMoon({ scene = 'welcome', aperture = 0.28, className = '', label }) {
  const details = SCENES[scene] || SCENES.welcome;
  const id = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const moonClip = `moon-clip-${id}`;
  const moonMask = `moon-mask-${id}`;
  const skyId = `sky-${id}`;
  const discId = `disc-${id}`;
  const r = details.radius;
  const apertureWidth = Math.max(0.1, Math.min(1, aperture)) * 136;
  const maskShift = Math.max(0, r * 0.58);
  const isDay = ['pale', 'twilight'].includes(details.sky);

  return (
    <svg
      className={`earthshine-art earthshine-art--${scene} ${className}`.trim()}
      viewBox="0 0 390 280"
      preserveAspectRatio="xMidYMid meet"
      role="img"
      aria-label={label || 'ماه با هلالی روشن و بخشی که نور بازتابیدهٔ زمین آن را آشکار می‌کند'}
      focusable="false"
    >
      <defs>
        <linearGradient id={`false-dawn-${id}`} x1="0" y1="240" x2="0" y2="150" gradientUnits="userSpaceOnUse">
          <stop stopColor="#AD8E99" stopOpacity=".16" />
          <stop offset=".55" stopColor="#AD8E99" stopOpacity=".055" />
          <stop offset="1" stopColor="#AD8E99" stopOpacity="0" />
        </linearGradient>
        <linearGradient id={skyId} x1="0" y1="0" x2="0" y2="1">
          {details.sky === 'night' && <><stop stopColor="#080B12" /><stop offset="1" stopColor="#0B111C" /></>}
          {details.sky === 'ultramarine' && <><stop stopColor="#10182B" /><stop offset="1" stopColor="#1A2942" /></>}
          {details.sky === 'deep' && <><stop stopColor="#142236" /><stop offset="1" stopColor="#1E3045" /></>}
          {details.sky === 'dawn' && <><stop stopColor="#142236" /><stop offset=".82" stopColor="#243446" /><stop offset="1" stopColor="#394652" /></>}
          {details.sky === 'pale' && <><stop stopColor="#879AAE" /><stop offset=".72" stopColor="#A8B6C0" /><stop offset="1" stopColor="#C2C5C2" /></>}
          {details.sky === 'twilight' && <><stop stopColor="#AAB8C2" /><stop offset=".73" stopColor="#C3C8C7" /><stop offset="1" stopColor="#D4C9BD" /></>}
        </linearGradient>
        <radialGradient id={discId} cx="36%" cy="30%" r="78%">
          <stop stopColor={isDay ? '#C9D3D8' : '#9AAAB9'} />
          <stop offset=".58" stopColor={isDay ? '#AAB8C2' : '#65788C'} />
          <stop offset="1" stopColor={isDay ? '#8797A5' : '#3B4D62'} />
        </radialGradient>
        <clipPath id={moonClip}><circle cx="195" cy="112" r={r} /></clipPath>
        <filter id={`terminator-soft-${id}`} x="-20%" y="-20%" width="140%" height="140%">
          <feGaussianBlur stdDeviation="1.05" />
        </filter>
        <mask id={moonMask} maskUnits="userSpaceOnUse" x={195 - r - 8} y={112 - r - 8} width={r * 2 + 16} height={r * 2 + 16}>
          <circle cx="195" cy="112" r={r + 6} fill="white" />
          <circle cx={195 + maskShift} cy="112" r={r + 2} fill="black" filter={`url(#terminator-soft-${id})`} />
        </mask>
      </defs>

      <rect width="390" height="280" fill={`url(#${skyId})`} />
      {scene === 'dawn' && <rect x="0" y="150" width="390" height="90" fill={`url(#false-dawn-${id})`} />}
      {details.horizon && <>
        <rect x="0" y="239" width="390" height="41" fill={isDay ? '#596F7D' : '#182633'} opacity=".14" />
        <path d="M0 239.5H390" stroke={scene === 'final' ? '#B99875' : '#C1B4B2'} strokeWidth={scene === 'final' ? '1' : '.65'} opacity={scene === 'final' ? '.78' : '.52'} />
      </>}

      {/* Earthlit face: the sphere is always present, but visibility is rationed. */}
      <circle cx="195" cy="112" r={r} fill={`url(#${discId})`} opacity={details.earthshine} />
      <g clipPath={`url(#${moonClip})`} opacity={details.surface}>
        {SURFACE_MARKS.map((d, index) => <path key={index} d={d} fill={isDay ? '#536678' : '#24364B'} opacity={index === 0 ? '.2' : '.14'} />)}
        <circle cx="174" cy="91" r="19" fill="none" stroke={isDay ? '#E4E8E7' : '#CAD3D8'} strokeWidth=".55" opacity=".12" />
        <circle cx="218" cy="137" r="14" fill="none" stroke={isDay ? '#E4E8E7' : '#CAD3D8'} strokeWidth=".45" opacity=".1" />
      </g>
      {/* A thin, soft-edged waning morning crescent; no outline and no emblem. */}
      {details.halo > 0 && <g mask={`url(#${moonMask})`}>
        <circle cx="195" cy="112" r={r + 5} fill="#E7ECEE" opacity={details.halo} />
      </g>}
      <g mask={`url(#${moonMask})`}>
        <circle cx="195" cy="112" r={r + 1} fill={isDay ? '#F0F1EB' : '#E8EDF0'} />
      </g>

      {details.stars > 0 && STAR_POSITIONS.slice(0, details.stars).map(([x, y, opacity], index) => (
        <circle key={index} cx={x} cy={y} r={index % 3 === 0 ? '.85' : '.58'} fill="#D6DEE3" opacity={opacity} />
      ))}

      {scene === 'lights' && <>
        {/* The sun is only just reaching the horizon: a short, quiet rim. */}
        <circle cx="300" cy="250" r="12" fill="#D9D4C6" opacity=".18" />
        <path d="M294 239.5Q300 236.5 306 239.5" fill="none" stroke="#E9E6D9" strokeWidth="1.1" strokeLinecap="round" opacity=".84" />
      </>}
      {scene === 'checkins' && <circle cx="284" cy="79" r="1.7" fill="#E9E6D9" opacity=".88" />}
      {scene === 'final' && <circle cx="284" cy="79" r="1.4" fill="#E2E3DE" opacity=".2" />}

      {scene === 'privacy' && <g aria-hidden="true">
        {/* The illustration plane is not connected to data controls. */}
        <path d={`M0 0H${195 - apertureWidth / 2}V280H0Z`} fill="#080B12" opacity=".96" />
        <path d={`M${195 + apertureWidth / 2} 0H390V280H${195 + apertureWidth / 2}Z`} fill="#080B12" opacity=".96" />
        <path d={`M${195 - apertureWidth / 2} 0V280M${195 + apertureWidth / 2} 0V280`} stroke="#A8B7C5" strokeWidth=".65" opacity=".58" />
      </g>}
    </svg>
  );
}
