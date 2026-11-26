import React from 'react';

/**
 * Theia's faceless identity — The Held Moon (ماهِ در آغوش) & Earthshine (زمین‌تاب).
 *
 * Theia is never drawn as a person or a geometric diagram. Her signature is
 * Earthshine itself: a slender cool-white waning morning crescent cradling the
 * spherical Moon, whose dark side is softly lit in cool ocean-blue-gray by
 * light returning from the Earth. Three cultural and narrative rules hold:
 *   1. It is a tilted pre-dawn waning crescent (lit on the lower-right limb),
 *      holding the sphere in its arms.
 *   2. The ashen disc is always present, so it never reads as a bare icon.
 *   3. No star sits beside or inside the crescent.
 */

export function TheiaSigil({
  size = 32,
  luminous = false,
  complete = false,
  earthshine = 0.78,
  className = '',
  title,
}) {
  const id = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const ashenGrad = `es-ashen-${id}`;
  const mareWashA = `es-mare-a-${id}`;
  const mareWashB = `es-mare-b-${id}`;
  const crescentGrad = `es-crescent-${id}`;
  const clipId = `es-sphere-${id}`;
  const maskId = `es-crescent-mask-${id}`;
  const glowId = `es-glow-${id}`;

  const legibility = Math.max(0.06, Math.min(1, Number.isFinite(earthshine) ? earthshine : 0.78));
  const discOpacity = 0.12 + legibility * 0.80;
  const mariaOpacity = 0.10 + legibility * 0.42;
  const rimOpacity = 0.10 + legibility * 0.34;

  return (
    <svg
      className={`theia-sigil ${luminous ? 'theia-sigil--luminous' : ''} ${complete ? 'theia-sigil--complete' : ''} ${className}`.trim()}
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
        <radialGradient id={ashenGrad} cx="68%" cy="70%" r="76%">
          <stop offset="0%" stopColor="#7C94B4" />
          <stop offset="48%" stopColor="#495C78" />
          <stop offset="100%" stopColor="#202B3D" />
        </radialGradient>
        <radialGradient id={mareWashA} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#182233" stopOpacity="0.62" />
          <stop offset="100%" stopColor="#182233" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={mareWashB} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#1A2538" stopOpacity="0.52" />
          <stop offset="100%" stopColor="#1A2538" stopOpacity="0" />
        </radialGradient>
        <linearGradient id={crescentGrad} x1="48" y1="50" x2="17" y2="17" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stopColor={complete ? '#FBF6EC' : '#F6F9FE'} />
          <stop offset="60%" stopColor={complete ? '#E9E2D6' : '#DCE7F5'} />
          <stop offset="100%" stopColor="#9FB5CF" />
        </linearGradient>
        <clipPath id={clipId}>
          <circle cx="32" cy="32" r="23" />
        </clipPath>
        <mask id={maskId}>
          <rect width="64" height="64" fill="#000" />
          <circle cx="32" cy="32" r="23" fill="#fff" />
          <circle cx="29.35" cy="29.15" r="22.45" fill="#000" />
        </mask>
        <filter id={glowId} x="-35%" y="-35%" width="170%" height="170%">
          <feGaussianBlur stdDeviation="1.6" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>

      {/* Subtle optical glare whisper around the sunlit lower-right crescent */}
      {luminous && (
        <circle cx="35.2" cy="35.4" r="24" fill="#DCE8F8" opacity={0.10} filter={`url(#${glowId})`} />
      )}

      {/* Ashen sphere lit by Earthshine, with soft atmospheric lunar maria */}
      <g clipPath={`url(#${clipId})`} opacity={discOpacity}>
        <circle cx="32" cy="32" r="23" fill={`url(#${ashenGrad})`} />
        <g opacity={mariaOpacity}>
          <ellipse cx="24" cy="26" rx="13" ry="11" transform="rotate(-24 24 26)" fill={`url(#${mareWashA})`} />
          <ellipse cx="38" cy="28" rx="11" ry="9" transform="rotate(16 38 28)" fill={`url(#${mareWashB})`} />
          <ellipse cx="29" cy="40" rx="11" ry="7.5" transform="rotate(8 29 40)" fill={`url(#${mareWashB})`} />
        </g>
      </g>

      {/* Delicate spherical limb of the ashen disc */}
      <circle
        cx="32"
        cy="32"
        r="22.75"
        stroke="#8FA6C4"
        strokeWidth="0.55"
        opacity={rimOpacity}
      />

      {/* Slender waning morning crescent cradling the sphere from below-right */}
      <g filter={luminous ? `url(#${glowId})` : undefined}>
        <circle
          cx="32"
          cy="32"
          r="23"
          fill={`url(#${crescentGrad})`}
          mask={`url(#${maskId})`}
        />
      </g>
    </svg>
  );
}

export function TheiaMark({ size = 24, luminous = false, complete = false, earthshine = 0.78, className = '' }) {
  return (
    <span
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      style={{ width: size, height: size }}
      aria-hidden="true"
    >
      <TheiaSigil size="100%" luminous={luminous} complete={complete} earthshine={earthshine} />
    </span>
  );
}
