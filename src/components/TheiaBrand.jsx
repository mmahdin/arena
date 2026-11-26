import React from 'react';
import SvgTree from './SvgTree';
import { buildEmblem } from '../art/emblem';

/**
 * Theia's redesigned, non-human identity — «نشانهٔ تیا».
 *
 * Theia (Θεία), the Greek Titaness of sight and heavenly light, is the mother of
 * the Sun, the Moon and the Dawn; she is the force that lets the world be seen
 * and that gives gold its shine (Pindar, Isthmian 5). The former identity drew
 * her as a woman. The redesign renders her as pure geometry instead:
 *
 *   - the shamseh (شمسه) — the Persian sunburst medallion — for Helios,
 *   - a calm almond within the ring — the seeing eye — for sight itself,
 *   - one small still disc at the centre — the light and the gaze as one.
 *
 * The emblem is generated from src/art/emblem.js so the app, the favicon and
 * the install icons can never drift apart. It deliberately has no skin, no
 * face and no gender: Theia is radiance and witness, not a person.
 */

export function TheiaMark({ size = 24, luminous = false, className = '' }) {
  const numeric = typeof size === 'number' ? size : undefined;
  const cssSize = typeof size === 'number' ? undefined : size;
  return (
    <span
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      style={{ width: size, height: size }}
      aria-hidden="true"
    >
      <SvgTree
        tree={React.useMemo(() => buildEmblem({ luminous }), [luminous])}
        width={numeric ?? (cssSize ? '100%' : undefined)}
        height={numeric ?? (cssSize ? '100%' : undefined)}
        style={cssSize ? { width: '100%', height: '100%' } : undefined}
      />
    </span>
  );
}

export function TheiaSealIcon({ size = 11, className = '' }) {
  return <TheiaMark size={size} className={className} />;
}

/**
 * Home's breathing emblem. Same geometry, staged inside the halo rings the
 * Home button already paints behind it; the glow now travels with the rays.
 */
export function TheiaHomeArt() {
  return (
    <span className="home-conversation-art" aria-hidden="true">
      <TheiaMark size="100%" luminous className="home-conversation-mark theia-home-emblem" />
    </span>
  );
}

/**
 * Theia's sealed light — used by Privacy. A khatam octagram (the eight-pointed
 * star of Persian inlay geometry) around a calm almond eye whose pupil is a
 * four-pointed star: what you offer here is held by the light, never exhibited.
 */
export function TheiaSealedEye({ size = 108, className = '' }) {
  return (
    <svg
      className={`theia-sealed-eye ${className}`.trim()}
      width={size}
      height={size}
      viewBox="0 0 120 120"
      fill="none"
      aria-hidden="true"
      focusable="false"
    >
      {/* khatam octagram: two overlapping squares */}
      <rect x="31" y="31" width="58" height="58" stroke="#c39a54" strokeWidth="2.6" />
      <rect x="31" y="31" width="58" height="58" stroke="#c39a54" strokeWidth="2.6" transform="rotate(45 60 60)" />
      <circle cx="60" cy="60" r="26.5" stroke="#23201b" strokeWidth="2.4" />
      {/* the calm almond — seeing without staring */}
      <path
        d="M 41.5 60 Q 60 49.5 78.5 60 Q 60 70.5 41.5 60 Z"
        stroke="#23201b"
        strokeWidth="2.3"
        strokeLinejoin="round"
        strokeLinecap="round"
      />
      {/* a four-pointed star for the pupil: the light kept inside */}
      <path
        d="M 60 52.6 Q 61.35 58.65 67.4 60 Q 61.35 61.35 60 67.4 Q 58.65 61.35 52.6 60 Q 58.65 58.65 60 52.6 Z"
        fill="#c39a54"
      />
    </svg>
  );
}

/**
 * The check-in charm — a drop of light arriving over still water. The user is
 * not reminded; they are found by a small, precise light at the right moment.
 */
export function TheiaCheckinGlow({ size = 148, className = '' }) {
  return (
    <svg
      className={`theia-checkin-glow ${className}`.trim()}
      width={size}
      height={size * 0.78}
      viewBox="0 0 200 158"
      fill="none"
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <radialGradient id="checkin-drop" cx="0.5" cy="0.4" r="0.65">
          <stop offset="0" stopColor="#ffe9b0" />
          <stop offset="0.55" stopColor="#f2c078" />
          <stop offset="1" stopColor="#d8a76c" />
        </radialGradient>
      </defs>
      {/* the falling spark — held, not chased */}
      <path
        d="M 100 16 Q 103.2 33.8 121 37 Q 103.2 40.2 100 58 Q 96.8 40.2 79 37 Q 96.8 33.8 100 16 Z"
        fill="url(#checkin-drop)"
      />
      {[[100, 75, 1.65], [100, 89, 1.2]].map(([x, y, r]) => (
        <circle key={y} cx={x} cy={y} r={r} fill="#c39a54" opacity="0.85" />
      ))}
      {/* the two-point witness: sun-star and firouzeh */}
      <path d="M 46 42 Q 47 48.4 53.4 49.4 Q 47 50.4 46 56.8 Q 45 50.4 38.6 49.4 Q 45 48.4 46 42 Z" fill="#c39a54" opacity="0.85" />
      <circle cx="152" cy="33" r="2.5" fill="#3f8f84" opacity="0.9" />
      {/* the howz answering in front arcs */}
      <path d="M 22 114 A 78 12 0 0 0 178 114" stroke="#23201b" strokeWidth="2.05" strokeLinecap="round" opacity="0.8" />
      <path d="M 46 127 A 54 9 0 0 0 154 127" stroke="#23201b" strokeWidth="1.75" strokeLinecap="round" opacity="0.58" />
      <path d="M 72 139 A 30 5.6 0 0 0 128 139" stroke="#23201b" strokeWidth="1.45" strokeLinecap="round" opacity="0.4" />
      <ellipse cx="100" cy="149.5" rx="9.5" ry="2.1" fill="#c39a54" opacity="0.6" />
    </svg>
  );
}

/**
 * Quiet celestial empty-inbox illustration for Letters ("هنوز نامه‌ای نرسیده").
 * An envelope drifting under a firouzeh moon and a khatam spark — unchanged in
 * spirit, aligned to the new geometry.
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
        d="M29.6 4.4A5.2 5.2 0 1 0 34.2 10.7A4.15 4.15 0 0 1 29.6 4.4Z"
        stroke="#d5d9f7"
        strokeOpacity="0.65"
        strokeWidth="1.25"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {/* khatam spark — eight glints from one still point */}
      <path
        d="M20.6 5.2 Q21.1 8.15 24.1 8.7 Q21.1 9.25 20.6 12.2 Q20.1 9.25 17.1 8.7 Q20.1 8.15 20.6 5.2 Z"
        fill="#f6cf98"
        fillOpacity="0.82"
      />
    </svg>
  );
}
