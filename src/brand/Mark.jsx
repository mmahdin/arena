import React from 'react';
import { SCENES, PALETTES, companionScene } from './scenes.js';
import Scene from './Scene.jsx';

/**
 * The Hamrah mark — «همسنگ»: two stones leaning together with an ember in the
 * crook between them. One mark, three polarities, no gradients.
 */
export function Mark({ tone = 'ink', size = 40, label = 'نشان همراه', className = '', breathing = false, thinking = false }) {
  const palette = tone === 'reverse'
    ? { ...PALETTES.light, stone: '#f3f2ee' }
    : tone === 'ember'
      ? { ...PALETTES.light, stone: '#f3f2ee', ember: '#12110d' }
      : { ...PALETTES.light, ember: '#b4553d' };
  return (
    <Scene
      scene={SCENES.mark(palette)}
      className={`brand-mark brand-mark--${tone}${breathing ? ' is-breathing' : ''}${thinking ? ' is-thinking' : ''} ${className}`.trim()}
      style={{ width: size, height: size }}
      label={label}
    />
  );
}

/** The mark inside the product's own app-icon silhouette. */
export function MarkTile({ size = 56, className = '', breathing = false }) {
  const palette = { ...PALETTES.light, stone: '#f3f2ee', ember: '#f6cf98' };
  return (
    <span className={`brand-tile${breathing ? ' is-breathing' : ''} ${className}`.trim()} style={{ width: size, height: size }} aria-hidden="true">
      <span className="brand-tile__field" />
      <Scene scene={SCENES.mark(palette)} />
    </span>
  );
}

/**
 * The lockup: mark + «همراه» + tagline.
 * The wordmark is live text in the product's own Persian face, so it stays
 * crisp, selectable and legible to screen readers.
 */
export function Lockup({ tone = 'ink', tagline = 'با هم، آرام‌تر', size = 'md', className = '' }) {
  return (
    <span className={`brand-lockup brand-lockup--${tone} brand-lockup--${size} ${className}`.trim()}>
      <Mark tone={tone} size={size === 'sm' ? 26 : 38} label={null} />
      <span className="brand-lockup__text">
        <span className="brand-lockup__name">همراه</span>
        {tagline && <span className="brand-lockup__tagline">{tagline}</span>}
      </span>
    </span>
  );
}

/** A single companion in a square frame — used as an avatar or small spot art. */
export function CompanionScene({ tone = 'ink', className = '', label, pose = 'stand' }) {
  const palette = tone === 'reverse'
    ? { ...PALETTES.light, stone: '#f3f2ee', ground: '#3d3934' }
    : PALETTES.light;
  return <Scene scene={companionScene(palette, { pose })} className={className} label={label} />;
}

export { SCENES, PALETTES };
