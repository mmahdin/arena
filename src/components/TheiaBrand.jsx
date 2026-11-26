import React from 'react';
import { archGeometry, dawnStarPath, crescentPath } from './theia/scenes.js';
import TheiaScene from './theia/TheiaScene.jsx';

/**
 * Theia visual identity.
 *
 * Iranian users meet an illustrated woman in nearly every wellness app, and a
 * Greek goddess rendered as a long-haired figure in a beige drape reads either
 * as imported fantasy or as one more lifestyle model. So Theia is not pictured
 * here. She is shown as what her myth is actually made of: a threshold you
 * stand in front of, and the light that appears in it.
 *
 * The mark is a two-centred Persian arch with a dawn star inside it — a shape
 * every Iranian reader has physically stood in, which says "you may enter"
 * without a single decorative flourish. It is the same geometry used in the
 * onboarding chapters, the app icon and the favicon.
 */

export const THEIA_MARK = '/brand/theia-mark.svg';
export const THEIA_LOGO_SRC = THEIA_MARK;

/**
 * Optical sizing. A true identity has two drawings of the same mark: the
 * hairline version for hero sizes, and a sturdier one for the 14–24px sizes it
 * lives at inside seals, chat bubbles and list rows. Rendering the hairline at
 * 16px turns it into a grey smudge, so the geometry and the stroke weight are
 * both tuned per size band.
 */
const MARK_SCALES = [
  { max: 32, geometry: { half: 52, spring: 146, d: 34, base: 198, starY: 148, starR: 26 }, stroke: 12 },
  { max: 64, geometry: { half: 48, spring: 148, d: 30, base: 196, starY: 146, starR: 21 }, stroke: 9 },
  { max: Infinity, geometry: { half: 44, spring: 152, d: 25, base: 194, starY: 140, starR: 15 }, stroke: 5.5 },
];

export function markScaleFor(size) {
  const numeric = typeof size === 'number' ? size : Number.parseFloat(size);
  const resolved = Number.isFinite(numeric) ? numeric : 24;
  return MARK_SCALES.find((scale) => resolved <= scale.max) || MARK_SCALES[MARK_SCALES.length - 1];
}

/** The instrument: threshold + star, drawn inline so it inherits colour. */
export function theiaMarkPaths(options = {}) {
  const { half, spring, d, base, starY, starR } = { ...markScaleFor(64).geometry, ...options };
  return {
    arch: archGeometry({ half, spring, d, base }).path,
    star: dawnStarPath(120, starY, starR, starR * 0.3),
  };
}

export function TheiaMark({ size = 24, luminous = false, className = '', title }) {
  const scale = markScaleFor(size);
  const { arch, star } = theiaMarkPaths(scale.geometry);
  return (
    <span
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      style={{ width: size, height: size }}
      role={title ? 'img' : undefined}
      aria-label={title || undefined}
      aria-hidden={title ? undefined : 'true'}
    >
      <svg viewBox="0 0 240 240" fill="none" focusable="false">
        <path d={arch} stroke="currentColor" strokeWidth={scale.stroke} strokeLinecap="round" strokeOpacity="0.9" />
        <path d={star} fill="currentColor" />
      </svg>
    </span>
  );
}

/** Same mark, kept under the old name for seals and signatures. */
export function TheiaSealIcon({ size = 11, className = '', title }) {
  return <TheiaMark size={size} className={className} title={title} />;
}

/**
 * Quiet empty state for Letters: a crescent above a single line of horizon.
 * The light is there, the letter has not arrived yet.
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
        d={crescentPath({ cx: 25, cy: 22, radius: 11, inner: 9.6, offset: 4.2 })}
        stroke="currentColor"
        strokeWidth="1.25"
        strokeLinejoin="round"
        strokeOpacity="0.85"
      />
      <path d="M8 36.5H40" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" strokeOpacity="0.45" />
      <path
        d={dawnStarPath(24, 31, 2.6, 0.8)}
        fill="currentColor"
        fillOpacity="0.7"
      />
    </svg>
  );
}

/** Home's invitation: the mark inside a slow dawn orbit. */
export function TheiaHomeArt() {
  const { arch, star } = theiaMarkPaths({ half: 34, spring: 148, d: 19, base: 182, starY: 136, starR: 11 });
  return (
    <span className="home-conversation-art" aria-hidden="true">
      <svg className="home-conversation-orbit" viewBox="0 0 100 100" fill="none" focusable="false">
        <path className="home-orbit-trace home-orbit-trace--first" d="M50 8C73 8 92 27 92 50S73 92 50 92 8 73 8 50 27 8 50 8Z" />
        <path className="home-orbit-trace home-orbit-trace--second" d="M50 16C69 16 84 31 84 50S69 84 50 84 16 69 16 50 31 16 50 16Z" />
        <path className="home-orbit-accent" d="M50 4v6M50 90v6M4 50h6M90 50h6" />
        <circle className="home-orbit-point home-orbit-point--sun" cx="50" cy="4" r="1.9" />
        <path className="home-orbit-crescent" d={crescentPath({ cx: 96, cy: 62, radius: 4.6, inner: 4, offset: 1.7 })} />
      </svg>
      <svg className="home-conversation-mark theia-home-emblem" viewBox="0 0 240 240" fill="none" focusable="false">
        <path d={arch} stroke="currentColor" strokeWidth="7" strokeLinecap="round" strokeOpacity="0.8" />
        <path d={star} fill="currentColor" />
      </svg>
    </span>
  );
}

/** A hairline of light — the separator that replaces decorative rules. */
export function TheiaHairline({ className = '' }) {
  return (
    <svg className={`theia-hairline ${className}`.trim()} viewBox="0 0 200 8" fill="none" aria-hidden="true" focusable="false">
      <path d="M0 4H86" stroke="currentColor" strokeWidth=".7" strokeOpacity=".4" />
      <path d="M114 4H200" stroke="currentColor" strokeWidth=".7" strokeOpacity=".4" />
      <path d={dawnStarPath(100, 4, 3.2, 1)} fill="currentColor" fillOpacity=".7" />
    </svg>
  );
}

export { TheiaScene };
