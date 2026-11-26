import React from 'react';
import { TheiaGlyph } from './TheiaArt';

/**
 * Theia's identity mark.
 *
 * The former photographic silhouette (a woman's profile inside a black disc) is
 * retired. A mental-health product in Iran should not hand users the face of a
 * goddess to confide in, and a human portrait implies a clinician who is not
 * there. The mark is now the same non-figurative grammar as the onboarding art:
 * a pointed arch, a horizon, and one point of light where they meet — the
 * Persian *tāq* and the instant of dawn that Theia's name carries.
 *
 * Everything that used to place a picture now draws this glyph, so Home, the
 * drawer, chat, letters, login and onboarding share one coherent identity.
 */
export function TheiaMark({ size = 24, luminous = false, className = '', title }) {
  return (
    <span
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      style={{ width: size, height: size }}
    >
      <TheiaGlyph size="100%" className="theia-mark-glyph" title={title} />
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
 * Quiet celestial empty-inbox illustration for Letters ("هنوز نامهای نرسیده").
 * Same geometry as the mark, opened into a small horizon.
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
        d="M24 6.5C17.5 10.5 12.5 15.3 12.5 22.6V38.5H35.5V22.6C35.5 15.3 30.5 10.5 24 6.5Z"
        stroke="currentColor"
        strokeWidth="1.25"
        strokeLinejoin="round"
      />
      <path d="M9 30.5H39" stroke="currentColor" strokeWidth="1.1" strokeOpacity="0.7" strokeLinecap="round" />
      <circle cx="24" cy="30.5" r="2.6" fill="currentColor" fillOpacity="0.85" />
      <path
        d="M31 12.5A3.4 3.4 0 1 0 33.8 17.3A2.8 2.8 0 0 1 31 12.5Z"
        stroke="#d5d9f7"
        strokeOpacity="0.62"
        strokeWidth="1"
        strokeLinecap="round"
      />
    </svg>
  );
}
