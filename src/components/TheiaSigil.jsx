import React from 'react';
import { crescentPath, EarthshineScene } from './Earthshine';

/**
 * Theia's faceless identity mark: Earthshine.
 *
 * Theia is never depicted as a person, mascot, or decorative astrolabe ring.
 * Her presence is expressed through revelation — the waning morning crescent
 * alongside its blue-gray earthlit disc, visible through reflected light.
 * Keeping the ashen disc always present ensures the mark reads as the Moon,
 * never as a standalone crescent emblem.
 */

export function TheiaSigil({ size = 32, luminous = false, complete = false, className = '', title }) {
  const id = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const clipId = `sigil-clip-${id}`;
  const ashenId = `sigil-ashen-${id}`;
  const haloId = `sigil-halo-${id}`;
  const cx = 32;
  const cy = 32;
  const r = 22;

  return (
    <svg
      className={`theia-sigil ${luminous ? 'theia-sigil--luminous' : ''} ${className}`.trim()}
      width={size}
      height={size}
      viewBox="0 0 64 64"
      fill="none"
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : 'true'}
      aria-label={title}
      focusable="false"
    >
      <defs>
        <clipPath id={clipId}>
          <circle cx={cx} cy={cy} r={r} />
        </clipPath>
        <radialGradient id={ashenId} cx="62%" cy="56%" r="72%" fx="74%" fy="62%">
          <stop offset="0%" stopColor={complete ? '#798FA9' : '#6A809C'} />
          <stop offset="58%" stopColor={complete ? '#5B708B' : '#4C607B'} />
          <stop offset="100%" stopColor={complete ? '#42556E' : '#34455D'} />
        </radialGradient>
        <radialGradient id={haloId} cx="46" cy="39" r="30" gradientUnits="userSpaceOnUse">
          <stop offset="0" stopColor="#EEF3FA" stopOpacity="0.22" />
          <stop offset="1" stopColor="#EEF3FA" stopOpacity="0" />
        </radialGradient>
      </defs>

      {luminous && <circle cx="44" cy="38" r="28" fill={`url(#${haloId})`} />}

      <circle cx={cx} cy={cy} r={r} fill={`url(#${ashenId})`} opacity={complete ? '0.94' : '0.84'} />
      <g clipPath={`url(#${clipId})`}>
        <path d={crescentPath(cx, cy, r * 1.005, 0.255, 23)} fill="#C5D4E8" opacity="0.32" />
        <path d={crescentPath(cx, cy, r, 0.21, 23)} fill="#EEF3FA" />
      </g>
    </svg>
  );
}

export function TheiaMark({ size = 24, luminous = false, complete = false, className = '' }) {
  return (
    <span
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      style={{ width: size, height: size }}
      aria-hidden="true"
    >
      <TheiaSigil size="100%" luminous={luminous} complete={complete} />
    </span>
  );
}

export { EarthshineScene };
