import React from 'react';
import { TheiaMark, TheiaSigil } from './TheiaSigil';

/**
 * Single source for Theia's identity mark (Earthshine).
 *
 * Theia is never represented as a person, mascot, or decorative ornament.
 * The shared mark is the waning morning crescent with its ever-present
 * earthlit blue-gray disc (`public/brand/theia-sigil.svg`). Retired portrait
 * artwork remains in `public/brand/` solely for repository history.
 */
export const THEIA_LOGO_SRC = '/brand/theia-sigil.svg';

export { TheiaMark, TheiaSigil };

export function TheiaSealIcon({ size = 11, className = '' }) {
  return <TheiaMark size={size} className={className} />;
}

/** Keep Home's existing button geometry and gentle breathing motion. */
export function TheiaHomeArt() {
  return (
    <span className="home-conversation-art" aria-hidden="true">
      <TheiaMark size="100%" luminous className="home-conversation-mark theia-home-emblem" />
    </span>
  );
}

/**
 * Quiet celestial empty-inbox illustration for Letters ("هنوز نامه‌ای نرسیده").
 * Uses the Earthshine Moon (crescent + soft ashen disc) with no star emblem.
 */
export function TheiaLetterEmpty({ size = 44 }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 48 48"
      fill="none"
      className="theia-letter-empty"
      aria-hidden="true"
      focusable="false"
    >
      <path
        d="M9 18.5C9 16.57 10.57 15 12.5 15H35.5C37.43 15 39 16.57 39 18.5V34.5C39 36.43 37.43 38 35.5 38H12.5C10.57 38 9 36.43 9 34.5V18.5Z"
        stroke="currentColor"
        strokeWidth="1.35"
      />
      <path
        d="M10.5 17L24 27.2L37.5 17"
        stroke="currentColor"
        strokeWidth="1.35"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <g transform="translate(24 7.5)">
        <circle cx="0" cy="0" r="4.6" fill="#475B76" fillOpacity="0.45" />
        <g transform="rotate(-46)">
          <path
            d="M 0 -4.6 A 4.6 4.6 0 0 0 0 4.6 A 3.55 4.6 0 0 1 0 -4.6 Z"
            fill="#E7EFFA"
            fillOpacity="0.9"
          />
        </g>
      </g>
    </svg>
  );
}
