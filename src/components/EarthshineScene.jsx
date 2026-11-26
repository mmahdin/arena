import React, { useEffect, useMemo } from 'react';
import { EARTHSHINE_SCENES } from '../features/onboarding/earthshine';

export { EARTHSHINE_SCENES };

const preloadedUrls = new Set();
function preloadImage(url) {
  if (!url || typeof Image === 'undefined' || preloadedUrls.has(url)) return;
  preloadedUrls.add(url);
  const img = new Image();
  img.decoding = 'async';
  img.src = url;
}

export function EarthshineScene({
  scene = 'welcome',
  open = true,
  lanternLit = true,
  compact = false,
  className = '',
  label,
}) {
  const meta = EARTHSHINE_SCENES[scene] || EARTHSHINE_SCENES.welcome;

  useEffect(() => {
    preloadImage(meta.next);
    if (meta.dimSrc) preloadImage(meta.dimSrc);
  }, [meta.next, meta.dimSrc]);

  const isPrivacy = meta.id === 'privacy';
  const isCheckins = meta.id === 'checkins';

  return (
    <figure
      className={[
        'earthshine-scene',
        `earthshine-scene--${meta.id}`,
        compact ? 'earthshine-scene--compact' : '',
        isPrivacy && !open ? 'is-shutter-narrow' : '',
        isCheckins && !lanternLit ? 'is-lantern-dim' : '',
        className,
      ].filter(Boolean).join(' ')}
      role="img"
      aria-label={label || meta.label}
    >
      <div className="earthshine-scene__viewport">
        {isCheckins && meta.dimSrc ? (
          <>
            <img
              className="earthshine-scene__img earthshine-scene__img--base"
              src={meta.dimSrc}
              alt=""
              aria-hidden="true"
              decoding="async"
              style={{ objectPosition: meta.focus }}
            />
            <img
              className="earthshine-scene__img earthshine-scene__img--lit"
              src={meta.src}
              alt=""
              aria-hidden="true"
              decoding="async"
              style={{
                objectPosition: meta.focus,
                opacity: lanternLit ? 1 : 0,
              }}
            />
          </>
        ) : (
          <img
            className="earthshine-scene__img"
            src={meta.src}
            alt=""
            aria-hidden="true"
            decoding="async"
            style={{ objectPosition: meta.focus }}
          />
        )}

        {/* Privacy screen: shutter aperture responds gently to user consent */}
        {isPrivacy && (
          <div className="earthshine-shutters" aria-hidden="true">
            <span className="earthshine-shutters__leaf earthshine-shutters__leaf--left" />
            <span className="earthshine-shutters__leaf earthshine-shutters__leaf--right" />
          </div>
        )}

        {/* Seamless atmospheric vignette so the rooftop window breathes with the page */}
        <span className="earthshine-scene__matte" aria-hidden="true" />
      </div>
    </figure>
  );
}

/**
 * Mood screen artwork for Chapter 3 (دیدن — Seeing): emotion drawn as lunar-sea
 * terrain and moonlit horizon contours rather than a binary switch or icon.
 * Evolves smoothly across levels 0..4 from heavy, restless night swells to a
 * settled, legible Earthshine horizon.
 */
export function EarthshineMoodTerrain({ level = 2 }) {
  const id = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const safeLevel = Math.max(0, Math.min(4, Number.isInteger(level) ? level : 2));

  const contours = useMemo(() => {
    const rows = 9;
    const width = 308;
    const xStart = 16;
    const turbulence = [18, 12.5, 8.2, 4.4, 2.1][safeLevel];
    const chop = [3.6, 2.7, 1.9, 1.25, 0.85][safeLevel];
    const spacing = [11.5, 12.5, 13.5, 14.5, 15.5][safeLevel];
    const baseY = 148;

    return Array.from({ length: rows }, (_, r) => {
      const y0 = baseY + r * spacing;
      const steps = 48;
      const pts = [];
      for (let i = 0; i <= steps; i += 1) {
        const t = i / steps;
        const x = xStart + t * width;
        const edgeTaper = Math.sin(t * Math.PI);
        const primary = Math.sin(t * Math.PI * chop * 2 + r * 0.72) * turbulence * 0.62;
        const secondary = Math.cos(t * Math.PI * (chop * 3.4 + 0.7) - r * 1.15) * turbulence * 0.38;
        const micro = safeLevel <= 1 ? Math.sin(t * Math.PI * 13 + r * 2.1) * (2.6 - safeLevel * 1.1) : 0;
        const y = y0 + (primary + secondary + micro) * (0.28 + 0.72 * edgeTaper);
        pts.push(`${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`);
      }
      return {
        d: pts.join(' '),
        opacity: (0.22 + (1 - r / rows) * 0.56).toFixed(3),
        strokeWidth: r === 0 ? '1.55' : (1.25 - r * 0.06).toFixed(2),
      };
    });
  }, [safeLevel]);

  // Earthshine legibility & sky calm increase gently from level 0 to 4
  const moonY = [88, 84, 79, 75, 72][safeLevel];
  const ashenOpacity = [0.18, 0.30, 0.46, 0.64, 0.82][safeLevel];
  const veilOpacity = [0.46, 0.30, 0.16, 0.06, 0][safeLevel];
  const reflectionCount = safeLevel + 2;
  const strokeColor = ['#3E495C', '#3A475B', '#34455C', '#2E425C', '#283E5A'][safeLevel];

  return (
    <svg className="mood-scribble earthshine-mood-terrain" viewBox="0 0 340 288" aria-hidden="true" focusable="false">
      <defs>
        <radialGradient id={`mood-wash-${id}`} cx="50%" cy="40%" r="54%">
          <stop offset="0%" stopColor="#6D84A3" stopOpacity={0.12 + safeLevel * 0.035} />
          <stop offset="68%" stopColor="#8EA3BE" stopOpacity={0.05 + safeLevel * 0.015} />
          <stop offset="100%" stopColor="#8EA3BE" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={`mood-ashen-${id}`} cx="66%" cy="68%" r="72%">
          <stop offset="0%" stopColor="#6B83A4" />
          <stop offset="56%" stopColor="#425570" />
          <stop offset="100%" stopColor="#263347" />
        </radialGradient>
        <mask id={`mood-crescent-${id}`}>
          <rect width="340" height="288" fill="#000" />
          <circle cx="170" cy={moonY} r="25" fill="#fff" />
          <circle cx="167.1" cy={moonY - 3.1} r="24.35" fill="#000" />
        </mask>
      </defs>

      {/* Subtle atmospheric wash on limestone paper */}
      <ellipse cx="170" cy="138" rx="148" ry="116" fill={`url(#mood-wash-${id})`} />

      {/* The Held Moon above the terrain */}
      <g>
        <circle cx="170" cy={moonY} r="25" fill={`url(#mood-ashen-${id})`} opacity={ashenOpacity} />
        <circle cx="170" cy={moonY} r="24.7" fill="none" stroke="#4D6280" strokeWidth="0.65" opacity={0.18 + safeLevel * 0.08} />
        <circle cx="170" cy={moonY} r="25" fill="#1E293B" opacity={0.78 + safeLevel * 0.05} mask={`url(#mood-crescent-${id})`} />
      </g>

      {/* Cloud/haze veil across the Moon when days feel heavy (levels 0–2) */}
      {veilOpacity > 0 && (
        <g stroke="#556173" strokeLinecap="round" opacity={veilOpacity}>
          <path d={`M118 ${moonY - 6} C144 ${moonY - 12}, 192 ${moonY - 2}, 224 ${moonY - 9}`} strokeWidth="1.4" fill="none" />
          <path d={`M126 ${moonY + 9} C156 ${moonY + 4}, 188 ${moonY + 14}, 216 ${moonY + 7}`} strokeWidth="1.15" fill="none" />
        </g>
      )}

      {/* Engraved lunar-sea / nocturnal tide contours */}
      <g fill="none" stroke={strokeColor} strokeLinecap="round" strokeLinejoin="round">
        {contours.map((line, index) => (
          <path key={index} d={line.d} strokeWidth={line.strokeWidth} opacity={line.opacity} />
        ))}
      </g>

      {/* Quiet Earthshine shimmer paths settling on the water/horizon as clarity grows */}
      <g stroke="#5E789B" strokeLinecap="round" opacity={0.22 + safeLevel * 0.12}>
        {Array.from({ length: reflectionCount }, (_, idx) => {
          const ry = 154 + idx * 14;
          const half = Math.max(8, 28 - idx * 4.2 + safeLevel * 3.5);
          const drift = (idx % 2 === 0 ? 1 : -1) * (4 - safeLevel * 0.7);
          return (
            <line
              key={idx}
              x1={(170 - half + drift).toFixed(1)}
              y1={ry}
              x2={(170 + half + drift).toFixed(1)}
              y2={ry}
              strokeWidth="1.35"
            />
          );
        })}
      </g>
    </svg>
  );
}
