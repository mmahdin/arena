import React from 'react';

/** Theia's signature: a point of attention framed by a quiet, opening lens. */
export function TheiaMark({ size = 24, luminous = false, className = '' }) {
  return (
    <svg width={size} height={size} viewBox="0 0 48 48" fill="none"
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      aria-hidden="true" focusable="false">
      <path d="M10 33a14 14 0 0 0 28 0" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
      <path d="M16 33a8 8 0 0 0 16 0" stroke="currentColor" strokeOpacity=".38" strokeWidth="1.1" strokeLinecap="round" />
      <path d="M6 35h13m10 0h13" stroke="currentColor" strokeOpacity=".64" strokeWidth="1.2" strokeLinecap="round" />
      <circle cx="24" cy="25" r="2.7" fill={luminous ? '#e8bd82' : 'currentColor'} />
      <path d="M24 19v-3" stroke={luminous ? '#e8bd82' : 'currentColor'} strokeWidth="1.1" strokeLinecap="round" />
    </svg>
  );
}

/** Small signature point used on the letter seal. */
export function TheiaSealIcon({ size = 11, className = '' }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={className} aria-hidden="true" focusable="false">
      <path d="M2 9a6 6 0 0 1 9.4-4.8M14 7a6 6 0 0 1-9.4 4.8" stroke="currentColor" strokeWidth="1.15" strokeLinecap="round" />
      <circle cx="8" cy="8" r="1.45" fill="currentColor" />
    </svg>
  );
}

/** An abstract, gently breathing focus point for the conversation doorway. */
export function TheiaHomeArt() {
  return (
    <span className="home-conversation-art" aria-hidden="true">
      <svg className="home-conversation-orbit" viewBox="0 0 240 240" fill="none" focusable="false">
        <path className="home-orbit-trace home-orbit-trace--first" d="M190 55a82 82 0 0 0-140 62" />
        <path className="home-orbit-trace home-orbit-trace--second" d="M51 144a82 82 0 0 0 140 42" />
        <path className="home-orbit-accent" d="M66 77a73 73 0 0 1 42-23" />
        <circle className="home-orbit-point home-orbit-point--sun" cx="120" cy="38" r="2.8" />
        <circle className="home-orbit-point home-orbit-point--sun" cx="196" cy="154" r="2" opacity=".58" />
      </svg>
      <svg className="home-conversation-mark theia-home-emblem" width="86" height="86" viewBox="0 0 96 96" fill="none" focusable="false">
        <circle cx="48" cy="48" r="38" stroke="#d5d9f7" strokeOpacity=".18" strokeWidth="1" />
        <path d="M23 61a25 25 0 0 0 50 0" stroke="#e6e1d7" strokeWidth="1.7" strokeLinecap="round" />
        <path d="M34 61a14 14 0 0 0 28 0" stroke="#d5d9f7" strokeOpacity=".35" strokeWidth="1.1" strokeLinecap="round" />
        <path d="M14 65h26m16 0h26" stroke="#aaa394" strokeOpacity=".82" strokeWidth="1.2" strokeLinecap="round" />
        <circle cx="48" cy="49" r="4" fill="#e8bd82" />
        <path d="M48 40v-6" stroke="#e8bd82" strokeWidth="1.3" strokeLinecap="round" />
      </svg>
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
      <path d="M18 11a6 6 0 0 1 9.4-4.8M30 9a6 6 0 0 1-9.4 4.8" stroke="#d5d9f7" strokeOpacity=".72" strokeWidth="1.15" strokeLinecap="round" />
      <circle cx="24" cy="9" r="1.25" fill="#e8bd82" />
    </svg>
  );
}
