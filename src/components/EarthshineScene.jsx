import React from 'react';

/**
 * One atmospheric plate per chapter. The image is the narrative: a natural
 * Held Moon, an eastern horizon, and — only later — a lantern or first light.
 * The slow brightness adaptation is the product's only "animation" of Theia:
 * she is never drawn, the eye simply adjusts. Reduced-motion users see the
 * final state immediately.
 */
export default function EarthshineScene({ scene, className = '' }) {
  return (
    <div
      className={`es-backdrop ${className}`.trim()}
      data-es-scene={scene.id}
      role="img"
      aria-label={scene.alt}
    >
      <img
        className="es-plate"
        src={scene.image}
        alt=""
        draggable="false"
        decoding="async"
        fetchPriority={scene.id === 'welcome' ? 'high' : 'auto'}
        style={{ objectPosition: scene.focus }}
      />
      <div className="es-scrim" aria-hidden="true" />
      <div className="es-grain" aria-hidden="true" />
      <div className="es-vignette" aria-hidden="true" />
    </div>
  );
}
