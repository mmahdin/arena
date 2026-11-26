import React from 'react';
import { ARCH, ART_COLORS, CONSTELLATION, GLYPH, SCENE_VIEWBOX, sceneForStage } from '../features/onboarding/artwork';

/**
 * Theia is not drawn as a woman, a statue or a deity. Her identity in this
 * product is a *room that light can enter*: a pointed arch, a horizon and one
 * source of light. These components are the only illustrations used by the
 * onboarding, and they share one geometry so the story stays continuous.
 *
 * Every scene is decorative; the surrounding copy carries all meaning for
 * assistive technology, so the SVG is `aria-hidden` unless a caller marks it
 * meaningful with `title`.
 */

let uid = 0;
const nextId = (prefix) => {
  uid += 1;
  return `${prefix}${uid}`;
};

/** The lines every scene shares; opacity carries the chapter's hour of night. */
function ArchFrame({ scene }) {
  return (
    <g fill="none" stroke={ART_COLORS.line} strokeLinecap="square">
      <path d={ARCH.outer} strokeWidth="1.1" opacity={scene.strokeOpacity} />
      {/* A second springing line makes the arch read as built masonry, not a blob. */}
      {scene.band && <path d={ARCH.band} strokeWidth="0.55" opacity={scene.strokeOpacity * 0.55} />}
      {scene.innerOpacity > 0 && (
        <path d={ARCH.middle} strokeWidth="0.9" opacity={scene.innerOpacity} />
      )}
      {scene.innerOpacity > 0.3 && (
        <path d={ARCH.inner} strokeWidth="0.7" opacity={scene.innerOpacity * 0.7} />
      )}
      <path d={ARCH.base} stroke={ART_COLORS.lineSoft} strokeWidth="0.7" opacity="0.45" />
      {scene.plinth && (
        <g stroke={ART_COLORS.lineSoft} strokeWidth="0.55" opacity="0.4">
          <path d={ARCH.floor} />
          <path d="M44 264 V288" opacity="0.5" />
          <path d="M196 264 V288" opacity="0.5" />
        </g>
      )}
    </g>
  );
}

const STAR_POINTS = [
  [52, 84], [188, 96], [72, 128], [206, 168], [44, 196], [160, 74], [96, 64], [186, 220], [128, 58],
];

function Stars({ scene }) {
  if (!scene.stars) return null;
  return (
    <g fill={ART_COLORS.moon} opacity={0.35 + scene.dawn * 0.4}>
      {STAR_POINTS.slice(0, scene.stars).map(([x, y], index) => (
        <circle key={`${x}-${y}`} cx={x} cy={y} r={index % 2 ? 0.9 : 1.3} />
      ))}
    </g>
  );
}

function Constellation({ placement = {} }) {
  const { offsetX = 0, offsetY = 0, scale = 1 } = placement;
  const point = ([x, y]) => `${(x * scale + offsetX).toFixed(1)} ${(y * scale + offsetY).toFixed(1)}`;
  return (
    <g className="theia-constellation">
      <g stroke={ART_COLORS.moon} strokeWidth="0.7" opacity="0.38">
        {CONSTELLATION.links.map(([a, b]) => (
          <path key={`${a}-${b}`} d={`M${point(CONSTELLATION.points[a])} L${point(CONSTELLATION.points[b])}`} />
        ))}
      </g>
      <g fill={ART_COLORS.moon} opacity="0.85">
        {CONSTELLATION.points.map(([x, y], index) => {
          const [px, py] = point([x, y]).split(' ').map(Number);
          return <circle key={`${x}-${y}`} cx={px} cy={py} r={index === 1 ? 1.9 : 1.3} />;
        })}
      </g>
    </g>
  );
}

/** One chapter illustration. `stage` 0–6 walks from closed door to open morning. */
export function TheiaScene({ stage = 0, className = '', title }) {
  const scene = sceneForStage(stage);
  const id = nextId('theia-scene');
  const glowId = `${id}-glow`;
  const shaftId = `${id}-shaft`;
  const waterId = `${id}-water`;
  const rays = scene.rays
    ? [0, 30, 60, 90, 120, 150].map((angle) => {
      const rad = (angle * Math.PI) / 180;
      const inner = scene.disc.r + 10;
      const outer = scene.disc.r + 24;
      return {
        key: angle,
        d: `M${(120 + Math.cos(rad) * inner).toFixed(1)} ${(scene.horizonY - Math.sin(rad) * inner).toFixed(1)} L${(120 + Math.cos(rad) * outer).toFixed(1)} ${(scene.horizonY - Math.sin(rad) * outer).toFixed(1)}`,
      };
    })
    : [];

  return (
    <svg
      className={`theia-scene theia-scene--${scene.id} ${className}`.trim()}
      viewBox={SCENE_VIEWBOX}
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : 'true'}
      focusable="false"
    >
      {title && <title>{title}</title>}
      <defs>
        <radialGradient id={glowId}>
          <stop offset="0" stopColor={ART_COLORS.dawnBright} stopOpacity="0.62" />
          <stop offset="0.42" stopColor={ART_COLORS.dawn} stopOpacity="0.18" />
          <stop offset="1" stopColor={ART_COLORS.dawn} stopOpacity="0" />
        </radialGradient>
        <linearGradient id={shaftId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor={ART_COLORS.dawn} stopOpacity="0.16" />
          <stop offset="1" stopColor={ART_COLORS.dawn} stopOpacity="0" />
        </linearGradient>
        <linearGradient id={waterId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor={ART_COLORS.dawn} stopOpacity="0.22" />
          <stop offset="1" stopColor={ART_COLORS.dawn} stopOpacity="0" />
        </linearGradient>
      </defs>

      <Stars scene={scene} />
      <circle className="theia-scene-glow" cx="120" cy={scene.disc.y} r={scene.glow} fill={`url(#${glowId})`} />

      <ArchFrame scene={scene} />

      {scene.beams && (
        <path
          className="theia-scene-shaft"
          d="M44 288 V152 C44 96 86 68 120 38 C154 68 196 96 196 152 V288 Z"
          fill={`url(#${shaftId})`}
          opacity={0.4 + scene.dawn * 0.6}
        />
      )}

      {scene.doorSeam && (
        <g className="theia-scene-seam" stroke={ART_COLORS.lineSoft} strokeWidth="0.6" opacity="0.5">
          <path d="M120 76 V288" />
          <path d="M44 236 H196" />
        </g>
      )}

      <path d={`M18 ${scene.horizonY} H222`} stroke={ART_COLORS.line} strokeWidth="0.85" opacity={0.35 + scene.dawn * 0.55} />
      <path className="theia-scene-horizon-glint" d={`M104 ${scene.horizonY} H136`} stroke={ART_COLORS.dawnBright} strokeWidth="1.4" opacity={0.25 + scene.dawn * 0.5} strokeLinecap="round" />

      {scene.water && (
        <g className="theia-scene-water">
          <path d={`M44 ${scene.horizonY} H196 V288 H44 Z`} fill={`url(#${waterId})`} />
          <g stroke={ART_COLORS.lineSoft} strokeWidth="0.6" opacity="0.5">
            <path d={`M60 ${scene.horizonY + 8} H180`} />
            <path d={`M72 ${scene.horizonY + 19} H168`} />
            <path d={`M88 ${scene.horizonY + 32} H152`} />
          </g>
        </g>
      )}

      {scene.halfSun ? (
        <path
          className="theia-scene-disc"
          d={`M${120 - scene.disc.r} ${scene.horizonY} A${scene.disc.r} ${scene.disc.r} 0 0 1 ${120 + scene.disc.r} ${scene.horizonY} Z`}
          fill={ART_COLORS.dawnBright}
          opacity="0.95"
        />
      ) : (
        <circle className="theia-scene-disc" cx="120" cy={scene.disc.y} r={scene.disc.r} fill={ART_COLORS.dawnBright} opacity="0.95" />
      )}

      {rays.length > 0 && (
        <g stroke={ART_COLORS.dawn} strokeWidth="0.7" opacity="0.5">
          {rays.map((ray) => <path key={ray.key} d={ray.d} />)}
        </g>
      )}

      {scene.moon && (
        <path
          className="theia-scene-moon"
          d="M176 92 A11 11 0 1 0 176 114 A12.6 12.6 0 0 1 176 92 Z"
          fill={ART_COLORS.moon}
          opacity="0.68"
        />
      )}

      {scene.lattice && (
        // A *mashrabiya* screen: many slim openings in a carved panel, spanning
        // the arch's width as architecture rather than a floating grid.
        <g className="theia-scene-lattice" stroke={ART_COLORS.lineSoft} strokeWidth="0.55" opacity="0.6">
          <path d={`M52 152 H188 V${scene.horizonY} H52 Z`} opacity="0.7" />
          {Array.from({ length: 11 }, (_, index) => (
            <path key={`v${index}`} d={`M${60 + index * 12} 152 V${scene.horizonY}`} opacity={index % 2 ? 0.55 : 0.8} />
          ))}
          {[0, 1, 2, 3].map((index) => (
            <path key={`h${index}`} d={`M52 ${163 + index * 14} H188`} opacity="0.5" />
          ))}
          <path d={`M52 ${scene.horizonY - 4} H188`} opacity="0.9" />
        </g>
      )}

      {scene.constellation && <Constellation placement={scene.constellation} />}

      {scene.disc.y < scene.horizonY - scene.disc.r - 6 && (
        <g stroke={ART_COLORS.lineSoft} strokeWidth="0.5" opacity="0.45">
          <path d={`M120 ${scene.disc.y + scene.disc.r + 9} V${scene.horizonY}`} strokeDasharray="2 5" />
        </g>
      )}

      {scene.openDoor && (
        <path
          className="theia-scene-door"
          d="M76 288 V160 C76 128 100 110 120 90 C140 110 164 128 164 160 V288 Z"
          fill={`url(#${shaftId})`}
          stroke={ART_COLORS.line}
          strokeWidth="0.8"
        />
      )}
    </svg>
  );
}

/**
 * The compact mark. Same grammar as the scenes, legible at 16px: an arch, a
 * horizon and the point of light where they meet. It replaces the photographic
 * face used before, so login, drawer, letters and chat carry one identity.
 */
export function TheiaGlyph({ size = 24, className = '', title }) {
  return (
    <svg
      className={`theia-glyph ${className}`.trim()}
      width={size}
      height={size}
      viewBox={GLYPH.viewBox}
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : 'true'}
      focusable="false"
    >
      {title && <title>{title}</title>}
      <path className="theia-glyph-halo" d={GLYPH.halo && `M${GLYPH.disc.x} ${GLYPH.disc.y - GLYPH.halo.r} A${GLYPH.halo.r} ${GLYPH.halo.r} 0 1 0 ${GLYPH.disc.x} ${GLYPH.disc.y + GLYPH.halo.r} A${GLYPH.halo.r} ${GLYPH.halo.r} 0 1 0 ${GLYPH.disc.x} ${GLYPH.disc.y - GLYPH.halo.r} Z`} fill="currentColor" opacity="0.14" />
      <path d={GLYPH.arch} fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinejoin="round" />
      <path d={GLYPH.horizon} fill="none" stroke="currentColor" strokeWidth="1.5" opacity="0.65" strokeLinecap="round" />
      <circle className="theia-glyph-disc" cx={GLYPH.disc.x} cy={GLYPH.disc.y} r={GLYPH.disc.r} fill="currentColor" />
    </svg>
  );
}

/**
 * Three-step chapter indicator used on narrative screens. It shows where the
 * reader is in the dawn, not a page count: the marks fill as light arrives.
 */
export function ChapterMarks({ chapter, className = '' }) {
  return (
    <span className={`chapter-marks ${className}`.trim()} role="img" aria-label={`${chapter.name}، پردهٔ ${chapter.numeral} از ۰۷`}>
      <span className="chapter-marks-numeral" aria-hidden="true">{chapter.numeral}</span>
      <span className="chapter-marks-rule" aria-hidden="true" />
      <span className="chapter-marks-name" aria-hidden="true">{chapter.name}</span>
    </span>
  );
}
