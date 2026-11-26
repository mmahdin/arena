import React from 'react';

/**
 * Earthshine — one sky, drawn honestly.
 *
 * Each scene paints the same sky. Nothing travels and nothing sparkles: the
 * only motion is a slow fade on mount and a breathing luminance in the halo.
 *
 * The Moon is hand-authored SVG so it stays sharp at every size:
 *
 *  - a thin waning morning crescent, lit toward the lower left (the sunrise
 *    side), with a soft terminator so no hard vector edge reads as an outline;
 *  - the ashen disc is always present, never omitted: with it the image reads
 *    as the Moon, without it a lone crescent reads as a logo;
 *  - a few soft maria and faint highland patches keep the surface matte and
 *    slightly weathered instead of a flat gradient;
 *  - a faint halo exists only around the bright crescent, because the story is
 *    about glare: light that arrives from somewhere else.
 *
 * The disc is deliberately not to scale with any particular night sky. Its
 * increasing readability is the story's one variable, not an ephemeris.
 */

const MOON_RADIUS = 96;
const MOON_CENTER = 120;
const MOON_BOX = 240;
/** Unit vector toward the lit limb: lower-left, the direction of the sunrise. */
const LIMB = { x: -0.58, y: 0.81 };
/**
 * The shadow disc is almost concentric with the Moon and displaced toward the
 * far side, so the lit part left over is a thin lune: a waning crescent with
 * tapered horns, not a fat gibbous.
 */
const SHADOW_DISTANCE = MOON_RADIUS * 0.165;

/**
 * The Moon's seas, drawn as irregular closed shapes rather than ellipses: the
 * maria are weather-named terrain, not decoration. Each is blurred just enough
 * to sit on the sphere without a vector edge, and each is slightly shallower or
 * deeper so the surface reads as terrain rather than as a pattern.
 */
const MARIA = [
  // Imbrium, with its bay
  { d: 'M72 92C64 74 78 58 98 56C118 54 130 66 128 82C127 92 118 96 110 98C98 101 86 104 78 100C74 98 73 95 72 92Z', depth: 1 },
  // Serenitatis
  { d: 'M132 76C140 68 154 70 160 80C166 90 162 104 150 107C138 110 128 102 127 90C126 84 128 80 132 76Z', depth: 0.9 },
  // Tranquillitatis
  { d: 'M138 116C148 106 166 110 174 122C182 134 176 148 162 150C148 152 136 144 134 132C133 126 135 120 138 116Z', depth: 1 },
  // Crisium, foreshortened near the limb
  { d: 'M182 92C190 88 199 94 199 103C199 112 191 118 183 114C175 110 174 98 182 92Z', depth: 0.85 },
  // Procellarum, along the western limb
  { d: 'M42 112C46 98 60 94 70 100C78 105 80 116 76 130C72 144 64 156 54 154C44 152 36 132 38 120C39 116 40 114 42 112Z', depth: 0.88 },
  // Nubium
  { d: 'M86 148C92 140 108 140 116 148C124 156 122 170 110 174C98 178 86 170 84 160C83 155 84 151 86 148Z', depth: 0.92 },
  // Fecunditatis
  { d: 'M158 132C166 126 180 128 186 138C192 148 186 160 176 158C166 156 158 148 156 140C155 136 156 134 158 132Z', depth: 0.8 },
  // Vaporum, the small sea between the larger ones
  { d: 'M116 106C122 102 131 106 131 112C131 118 124 122 118 119C112 116 112 110 116 106Z', depth: 0.6 },
];

const STARS = [
  { x: 9, y: 40, r: 1, o: 0.55 },
  { x: 16, y: 66, r: 0.8, o: 0.4 },
  { x: 24, y: 17, r: 1.1, o: 0.5 },
  { x: 86, y: 24, r: 0.9, o: 0.46 },
  { x: 93, y: 46, r: 0.7, o: 0.36 },
  { x: 80, y: 12, r: 1.2, o: 0.6 },
  { x: 68, y: 90, r: 0.8, o: 0.32 },
];

function palette(scene) {
  if (scene.tone === 'day') {
    return {
      // Civil twilight, not daylight. In a pale sky the unlit disc reads as a
      // quiet silhouette a shade below the sky, so the crescent stays the
      // brightest thing in the frame — as it does in the sky itself.
      discCore: '#93A7BA',
      discMid: '#8A9EB2',
      discEdge: '#7A8FA3',
      discOpacity: 0.42 + 0.13 * scene.earthshine,
      maria: '#6C8095',
      mariaOpacity: 0.34 * scene.earthshine,
      highland: '#FFFFFF',
      crescent1: '#FFFFFF',
      crescent2: '#F6FAFD',
      crescent3: '#DDE8F2',
    };
  }
  return {
    discCore: '#A4B5CB',
    discMid: '#8B9CB3',
    discEdge: '#66748B',
    // Scene 1 is only "a darkness very slightly different from the sky"; the
    // disc becomes legible by scene 4 and fully readable by scene 6.
    discOpacity: 0.045 + 0.58 * scene.earthshine,
    maria: '#3B4760',
    mariaOpacity: 0.5 * scene.earthshine,
    highland: '#EBF1F8',
    crescent1: '#F8FBFF',
    crescent2: '#E3ECF6',
    crescent3: '#C3D1E1',
  };
}

function Moon({ scene }) {
  const uid = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const p = palette(scene);
  const ids = {
    disc: `moon-disc-${uid}`,
    crescent: `moon-lit-${uid}`,
    soft: `moon-soft-${uid}`,
    clip: `moon-clip-${uid}`,
    mask: `moon-mask-${uid}`,
    grain: `moon-grain-${uid}`,
    sea: `moon-sea-${uid}`,
    haze: `moon-haze-${uid}`,
  };
  const shadowX = MOON_CENTER - LIMB.x * SHADOW_DISTANCE;
  const shadowY = MOON_CENTER - LIMB.y * SHADOW_DISTANCE;
  return (
    <svg
      className="ob-moon"
      viewBox={`0 0 ${MOON_BOX} ${MOON_BOX}`}
      fill="none"
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        {/* Limb-darkened sphere, not a flat circle. */}
        <radialGradient id={ids.disc} cx="46%" cy="44%" r="78%">
          <stop offset="0" stopColor={p.discCore} />
          <stop offset="0.82" stopColor={p.discMid} />
          <stop offset="1" stopColor={p.discEdge} />
        </radialGradient>
        {/* Brightness falls off across the thin crescent, toward the terminator. */}
        <linearGradient id={ids.crescent} x1="0" y1="1" x2="1" y2="0">
          <stop offset="0.2" stopColor={p.crescent1} />
          <stop offset="0.31" stopColor={p.crescent2} />
          <stop offset="0.44" stopColor={p.crescent3} />
          <stop offset="0.62" stopColor={p.crescent3} stopOpacity="0.55" />
        </linearGradient>
        <filter id={ids.soft} x="-40%" y="-40%" width="180%" height="180%">
          <feGaussianBlur stdDeviation="1.35" />
        </filter>
        <filter id={ids.grain} x="0" y="0" width="100%" height="100%">
          <feTurbulence type="fractalNoise" baseFrequency="0.92" numOctaves="3" stitchTiles="stitch" />
          <feColorMatrix type="saturate" values="0" />
        </filter>
        <filter id={ids.sea} x="-30%" y="-30%" width="160%" height="160%">
          <feGaussianBlur stdDeviation="1.4" />
        </filter>
        {/* A pale sky softens the limb: no vector-hard edge in daylight. */}
        <filter id={ids.haze} x="-20%" y="-20%" width="140%" height="140%">
          <feGaussianBlur stdDeviation="0.6" />
        </filter>
        <clipPath id={ids.clip}>
          <circle cx={MOON_CENTER} cy={MOON_CENTER} r={MOON_RADIUS} />
        </clipPath>
        {/* White disc minus a blurred, offset disc: a soft-terminator crescent. */}
        <mask id={ids.mask} maskUnits="userSpaceOnUse" x="-80" y="-80" width="400" height="400">
          <rect x="-80" y="-80" width="400" height="400" fill="#000" />
          <circle cx={MOON_CENTER} cy={MOON_CENTER} r={MOON_RADIUS} fill="#fff" />
          <circle cx={shadowX} cy={shadowY} r={MOON_RADIUS * 1.004} fill="#000" filter={`url(#${ids.soft})`} />
        </mask>
      </defs>

      {/* Earthshine: the part the Sun does not reach, lit by the Earth. */}
      <g opacity={p.discOpacity} filter={scene.tone === 'day' ? `url(#${ids.haze})` : undefined}>
        <circle cx={MOON_CENTER} cy={MOON_CENTER} r={MOON_RADIUS} fill={`url(#${ids.disc})`} />
        <g clipPath={`url(#${ids.clip})`}>
          <g fill={p.maria} filter={`url(#${ids.sea})`}>
            {MARIA.map((sea) => (
              <path key={sea.d.slice(0, 14)} d={sea.d} opacity={p.mariaOpacity * sea.depth} />
            ))}
          </g>
          {/* Faint southern highlands and two young craters: texture, not scenery. */}
          <g fill={p.highland} filter={`url(#${ids.sea})`}>
            <circle cx="128" cy="184" r="15" opacity={p.mariaOpacity * 0.3} />
            <circle cx="128" cy="184" r="3.4" opacity={p.mariaOpacity * 0.42} />
            <circle cx="99" cy="118" r="2.2" opacity={p.mariaOpacity * 0.3} />
          </g>
          {/* Matte grain so the disc never looks like smooth CSS. */}
          <rect
            x="24"
            y="24"
            width="192"
            height="192"
            filter={`url(#${ids.grain})`}
            opacity="0.05"
            style={{ mixBlendMode: 'soft-light' }}
          />
        </g>
      </g>

      {/* The crescent. The brightest thing in every scene. */}
      <circle
        cx={MOON_CENTER}
        cy={MOON_CENTER}
        r={MOON_RADIUS}
        fill={`url(#${ids.crescent})`}
        mask={`url(#${ids.mask})`}
      />
    </svg>
  );
}

/** The Moon alone, for the sign-in entry that precedes the story. */
export function TheiaMoon({ scene }) {
  const box = Math.round(scene.moon * 1.25);
  return (
    <div
      className="ob-moon-standalone"
      style={{ width: `min(${box}px, ${((box / 390) * 100).toFixed(2)}vw)`, '--ob-halo': scene.halo }}
    >
      <span className="ob-sky__glare" />
      <Moon scene={scene} />
    </div>
  );
}

/**
 * The sky for one scene. `children` renders above the Moon (the privacy window).
 * Layout of the sky column — its height, the tone under it — lives in CSS so
 * the light can shift between scenes without re-measuring anything.
 */
export function TheiaSky({ scene, children = null }) {
  const box = Math.round(scene.moon * 1.25);
  const stars = Math.round(scene.stars);
  const moonStyle = {
    width: `min(${box}px, ${((box / 390) * 100).toFixed(2)}vw, calc(var(--ob-sky-zone) * 0.82))`,
    '--ob-halo': scene.halo,
  };
  return (
    <div className="ob-sky" data-scene={scene.id} data-tone={scene.tone} aria-hidden="true">
      <div className="ob-sky__field" />
      {scene.falseDawn && <span className="ob-sky__dawn" />}
      {stars > 0 && (
        <div className="ob-sky__stars" style={{ '--ob-star-strength': Math.min(1, scene.stars) }}>
          {STARS.slice(0, stars).map((star) => (
            <span
              key={`${star.x}-${star.y}`}
              className="ob-star"
              style={{
                left: `${star.x}%`,
                top: `${star.y}%`,
                width: `${star.r * 2}px`,
                height: `${star.r * 2}px`,
                '--ob-star-opacity': star.o,
              }}
            />
          ))}
        </div>
      )}
      <div className="ob-sky__moon" style={moonStyle}>
        <span className="ob-sky__glare" />
        <Moon scene={scene} />
      </div>
      {scene.venus > 0 && <span className="ob-sky__venus" style={{ '--ob-venus': scene.venus }} />}
      {scene.sun && <span className="ob-sky__sun" />}
      <span className="ob-sky__veil" />
      {scene.horizon && <span className="ob-sky__horizon" />}
      {children}
    </div>
  );
}
