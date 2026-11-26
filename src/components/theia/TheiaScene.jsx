import React from 'react';
import { PALETTE, SCENES } from './scenes.js';

// Palette keys become hex; scene-local gradient ids become url(#id); anything
// else (including 'none') is passed through untouched.
const makeColor = (ids) => (value) => {
  if (!value) return 'none';
  if (PALETTE[value]) return PALETTE[value];
  return ids.has(value) ? `url(#${value})` : value;
};

const paletteColor = (value) => (value && PALETTE[value]) || value || 'none';

function SceneDefs({ defs = [] }) {
  const color = paletteColor;
  return (
    <defs>
      {defs.map((def) => (def.type === 'radial' ? (
        <radialGradient key={def.id} id={def.id}>
          {def.stops.map(([offset, c, o], index) => (
            <stop key={index} offset={`${offset * 100}%`} stopColor={color(c)} stopOpacity={o} />
          ))}
        </radialGradient>
      ) : (
        <linearGradient
          key={def.id}
          id={def.id}
          gradientUnits={def.from ? 'userSpaceOnUse' : undefined}
          x1={def.from ? def.from[0] : 0}
          y1={def.from ? def.from[1] : 0}
          x2={def.to ? def.to[0] : 0}
          y2={def.to ? def.to[1] : 1}
        >
          {def.stops.map(([offset, c, o], index) => (
            <stop key={index} offset={`${offset * 100}%`} stopColor={color(c)} stopOpacity={o} />
          ))}
        </linearGradient>
      )))}
    </defs>
  );
}

function SceneElement({ element: el, index, color }) {
  const style = el.delay ? { '--theia-delay': `${el.delay}s` } : undefined;
  const common = { className: el.className, style };
  const dashed = el.dash ? { strokeDasharray: el.dash } : {};
  switch (el.t) {
    case 'clip':
      return (
        <clipPath id={el.id} key={el.key || index}>
          {el.children.map((child, i) => <SceneElement key={i} element={child} index={i} color={color} />)}
        </clipPath>
      );
    case 'circle':
      return <circle key={el.key || index} {...common} cx={el.cx} cy={el.cy} r={el.r} fill={color(el.fill)} stroke={color(el.stroke)} strokeWidth={el.strokeWidth} opacity={el.opacity} {...dashed} />;
    case 'rect':
      return <rect key={el.key || index} {...common} x={el.x} y={el.y} width={el.width} height={el.height} fill={color(el.fill)} />;
    case 'line':
      return <line key={el.key || index} {...common} x1={el.x1} y1={el.y1} x2={el.x2} y2={el.y2} stroke={color(el.stroke)} strokeWidth={el.strokeWidth} opacity={el.opacity} strokeLinecap="round" {...dashed} />;
    case 'group':
      return (
        <g key={el.key || index} {...common} clipPath={el.clip ? `url(#${el.clip})` : undefined}>
          {el.children.map((child, i) => <SceneElement key={i} element={child} index={i} color={color} />)}
        </g>
      );
    case 'path':
    default:
      return <path key={el.key || index} {...common} d={el.d} fill={color(el.fill)} stroke={color(el.stroke)} strokeWidth={el.strokeWidth} strokeLinecap="round" strokeLinejoin="round" opacity={el.opacity} {...dashed} />;
  }
}

/**
 * One chapter of the light narrative. Generated from the geometry in
 * scenes.js: sharp at any size, animatable with CSS, and re-renderable for
 * design review without a browser.
 */
export function TheiaScene({ name, className = '', label, animated = true }) {
  const build = SCENES[name];
  if (!build) return null;
  const scene = build();
  const classes = ['theia-scene', `theia-scene--${name}`, animated ? 'is-animated' : '', className]
    .filter(Boolean).join(' ');
  return (
    <svg
      className={classes}
      viewBox={scene.viewBox}
      fill="none"
      role={label ? 'img' : undefined}
      aria-label={label || undefined}
      aria-hidden={label ? undefined : 'true'}
      focusable="false"
      preserveAspectRatio="xMidYMid meet"
    >
      <SceneDefs defs={scene.defs} />
      {scene.elements.map((element, index) => <SceneElement key={index} element={element} index={index} color={makeColor(new Set(scene.defs.map((def) => def.id)))} />)}
    </svg>
  );
}

export default TheiaScene;
