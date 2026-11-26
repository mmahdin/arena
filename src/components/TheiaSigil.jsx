import React from 'react';

/**
 * Theia's small product mark is intentionally faceless.
 *
 * The open ring + spark remains the identity used at 11–24px across Home,
 * Letters and chat. It is a signature, not the onboarding story. The
 * onboarding imagery now follows the Earthshine direction — a single moon
 * whose dark side becomes visible through reflected light — and lives in
 * `EarthshineInstrument.jsx`. The two are deliberately separate: a logo may
 * be an emblem, but a story about seeing must be a real thing being seen.
 *
 * The mark is hand-authored SVG so it stays sharp at 11px.
 */

const SIGIL_STAR = (cx, cy, radius) => {
  const k = radius * 0.085;
  const r = radius;
  return [
    `M${cx} ${cy - r}`,
    `Q${cx + k} ${cy - k} ${cx + r} ${cy}`,
    `Q${cx + k} ${cy + k} ${cx} ${cy + r}`,
    `Q${cx - k} ${cy + k} ${cx - r} ${cy}`,
    `Q${cx - k} ${cy - k} ${cx} ${cy - r}`,
    'Z',
  ].join(' ');
};

export function TheiaSigil({ size = 32, luminous = false, complete = false, className = '', title }) {
  const id = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const gold = `sigil-gold-${id}`;
  const glow = `sigil-glow-${id}`;
  // The ring stops short of closing at the bottom. `complete` closes it.
  const openRing = 'M19.4 54.7A26 26 0 1 1 44.6 54.7';
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
        <linearGradient id={gold} x1="8" y1="8" x2="56" y2="58" gradientUnits="userSpaceOnUse">
          <stop stopColor="#F7E7C6" />
          <stop offset=".46" stopColor="#E3C48C" />
          <stop offset="1" stopColor="#B98A4E" />
        </linearGradient>
        <filter id={glow} x="-40%" y="-40%" width="180%" height="180%">
          <feGaussianBlur stdDeviation="1.5" result="b" />
          <feMerge>
            <feMergeNode in="b" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>
      <g filter={luminous ? `url(#${glow})` : undefined}>
        <path
          d={complete ? 'M32 6A26 26 0 1 1 31.99 6' : openRing}
          stroke={`url(#${gold})`}
          strokeWidth={complete ? 2.6 : 2.2}
          strokeLinecap="round"
        />
        <g transform="translate(32 31) scale(0.74 1) translate(-32 -31)">
          <path d={SIGIL_STAR(32, 31, 11)} fill={`url(#${gold})`} opacity="0.97" />
        </g>
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

/* The onboarding instrument keeps its old export name and call sites. The
   Earthshine states and their descriptions are owned by the artwork module. */
export { EarthshineGlyph, TheiaInstrument, STATE_LABEL } from './EarthshineInstrument.jsx';
