import React from 'react';
import { TheiaMark, TheiaSigil } from './TheiaSigil';

/**
 * Single source for Theia's faceless identity mark: the Held Moon (ماهِ در آغوش),
 * a waning pre-dawn crescent cradling its cool blue-gray Earthshine sphere.
 *
 * Retired female-portrait assets remain in `public/brand/` strictly as history
 * and are never referenced in the UI, HTML head or install manifest.
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
      <TheiaMark size="100%" luminous earthshine={0.86} className="home-conversation-mark theia-home-emblem" />
    </span>
  );
}

/**
 * Quiet nocturnal empty-inbox illustration for Letters ("هنوز نامه‌ای نرسیده") —
 * an envelope beneath the Held Moon (crescent + ashen sphere, no star).
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
      {/* Held Moon: Earthshine sphere + lower-right waning crescent, never beside a star */}
      <circle cx="24" cy="8.2" r="4.3" fill="#6E84A3" fillOpacity="0.28" stroke="#8EA5C2" strokeOpacity="0.42" strokeWidth="0.7" />
      <path
        d="M27.8 6.2A4.3 4.3 0 0 1 21.8 11.9A4.05 4.05 0 0 0 27.8 6.2Z"
        fill="#E8EFFA"
        fillOpacity="0.88"
      />
    </svg>
  );
}
