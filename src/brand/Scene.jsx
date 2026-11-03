import React from 'react';

/**
 * Renders Hamrah identity scenes.
 *
 * Layer data comes from `brand/scenes.js`, the same source that generates the
 * brand documentation and print assets, so the app and the identity can never
 * drift apart. Everything is decorative: the accessible name lives on the
 * surrounding element, or is supplied through `label`.
 */
function Layer({ layer }) {
  if (layer.t === 'circle') {
    return <circle cx={layer.cx} cy={layer.cy} r={layer.r} fill={layer.fill} />;
  }
  return (
    <path
      d={layer.d}
      fill={layer.fill}
      fillRule={layer.rule}
      stroke={layer.stroke}
      strokeWidth={layer.width}
      strokeLinecap={layer.cap}
      strokeLinejoin={layer.cap === 'round' ? 'round' : undefined}
      strokeDasharray={layer.dash}
    />
  );
}

export default function Scene({ scene, className = '', label, style }) {
  const decorative = !label;
  return (
    <svg
      className={`brand-scene ${className}`.trim()}
      viewBox={scene.viewBox}
      role={decorative ? 'presentation' : 'img'}
      aria-hidden={decorative ? 'true' : undefined}
      aria-label={label}
      focusable="false"
      style={style}
    >
      {scene.layers.map((layer, index) => <Layer key={index} layer={layer} />)}
    </svg>
  );
}
