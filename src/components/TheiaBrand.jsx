import React from 'react';

/** The user-approved logo. Keep the supplied artwork and colors intact. */
export const THEIA_LOGO_SRC = '/brand/photo_2026-11-12_19-20-10.jpg';

export function TheiaMark({ size = 24, luminous = false, className = '' }) {
  return (
    <span
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      style={{ width: size, height: size }}
      aria-hidden="true"
    >
      <img src={THEIA_LOGO_SRC} width="1254" height="1254" alt="" draggable="false" />
    </span>
  );
}

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
        d="M9 17.5C9 15.57 10.57 14 12.5 14H35.5C37.43 14 39 15.57 39 17.5V33.5C39 35.43 37.43 37 35.5 37H12.5C10.57 37 9 35.43 9 33.5V17.5Z"
        stroke="currentColor"
        strokeWidth="1.35"
      />
      <path
        d="M10.5 16L24 26.2L37.5 16"
        stroke="currentColor"
        strokeWidth="1.35"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M28.5 5.2A5.2 5.2 0 1 0 32.8 11.4A4.1 4.1 0 0 1 28.5 5.2Z"
        stroke="#d5d9f7"
        strokeOpacity="0.65"
        strokeWidth="1.25"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M21 6.5L21.65 8.35L23.5 9L21.65 9.65L21 11.5L20.35 9.65L18.5 9L20.35 8.35L21 6.5Z"
        fill="#f6cf98"
        fillOpacity="0.78"
      />
    </svg>
  );
}
