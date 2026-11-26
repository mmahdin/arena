import React from 'react';

/**
 * Earthshine — the onboarding imagery.
 *
 * When the Moon is a thin crescent, the part that the Sun does not reach is
 * sometimes still faintly visible: sunlight has struck the Earth, been
 * reflected back into the night, and landed on the Moon's dark side. The Moon
 * itself has not changed. Nothing has been added to it. What was always there
 * has simply become visible through reflection.
 *
 * That is the whole visual argument of this onboarding, so the artwork obeys
 * one rule: there is ONE photograph of ONE moon in every screen. Only the
 * amount of reflected light changes between chapters. No new object is ever
 * introduced to represent growth, insight or progress — no rings closing, no
 * orbits, no sparks, no rays, no decorative geometry. The last chapter is not
 * a brighter Moon; it is the same Moon, finally legible.
 *
 * Three plates, all cropped from the same composition:
 *   - the moon with its earthshine-lit disc and a warm glow from below;
 *   - the same moon with the dark limb of the Earth at the bottom of the frame,
 *     which makes the *source* of the reflection visible for the three-lights
 *     chapter (sunlight → Earth → Moon);
 *   - the same drawing as an ink-and-wash plate for the two ivory screens.
 *
 * The photograph supplies the texture and the light. CSS supplies only the
 * gradual revelation: a veil of darkness that lifts, a warm earthlight that
 * breathes, and a soft vignette so the plate dissolves into the surface rather
 * than sitting on it as a card. Every animation is disabled for reduced motion.
 */

const MOON_PLATE = '/brand/earthshine/earthshine-moon.jpg';
const REFLECTION_PLATE = '/brand/earthshine/earthshine-reflection.jpg';
const PAPER_PLATE = '/brand/earthshine/earthshine-moon-paper.jpg';

/** Descriptions belong to the artwork, not to the surrounding copy. */
export const STATE_LABEL = {
  welcome: 'هلال باریک ماه؛ بخش تاریک قرص بهآرامی در نور بازتابیدهٔ زمین پیداست',
  night: 'شبی تقریباً تاریک؛ از قرص ماه فقط اثری بسیار کم‌نور دیده می‌شود',
  veil: 'سایه‌ای نرم از دو سو کنار می‌رود و قرص ماه از میان شکاف آن پیدا می‌شود',
  sight: 'قرص تاریک ماه در نور زمینتاب واضح می‌شود؛ دریاها و دهانه‌هایش پیدایند',
  dawn: 'گرمی کمفروغی از پایین قاب بالا میآید و آرامآرام ماه را نشان میدهد',
  orbits: 'لبهٔ تاریک زمین در پایین قاب و ماه بالای آن؛ نور خورشید از زمین به ماه بازتابیده است',
  pulse: 'نور زمینتاب بهآرامی بالا و پایین میرود، بیآنکه چیزی به ماه افزوده شود',
  seal: 'تمام قرص ماه، آرام و کامل، در همان نور بازتابیده دیده میشود',
};

/**
 * A chapter is a reading of the SAME plate, never a different illustration.
 * `light` is how much of what was already there has become visible; `warmth`
 * is the reflected light rising from the unseen Earth; `shadow` is the
 * not-knowing that has not lifted yet.
 */
const STATES = {
  welcome: { plate: 'moon', light: 0.88, saturation: 0.92, warmth: 0.16, shadow: 0.14 },
  night:   { plate: 'moon', light: 0.44, saturation: 0.64, warmth: 0.05, shadow: 0.48 },
  veil:    { plate: 'moon', light: 0.98, saturation: 0.78, warmth: 0.10, shadow: 0.18 },
  sight:   { plate: 'moon', light: 1.04, saturation: 0.96, warmth: 0.20, shadow: 0.02 },
  dawn:    { plate: 'moon', light: 1.06, saturation: 1,    warmth: 0.32, shadow: 0 },
  orbits:  { plate: 'reflection', light: 1.02, saturation: 0.94, warmth: 0.26, shadow: 0.04 },
  pulse:   { plate: 'moon', light: 1,    saturation: 0.88, warmth: 0.22, shadow: 0.08 },
  seal:    { plate: 'moon', light: 1.08, saturation: 1,    warmth: 0.28, shadow: 0 },
};

/**
 * Kept under the old component name so every onboarding call site, the
 * sign-in entry and the existing structure checks stay exactly as they were.
 * The `label` prop remains as a caller-supplied fallback for a future state;
 * the eight chapters take their accessible names from the artwork itself, so
 * no onboarding text has to change to describe the new imagery.
 */
export function TheiaInstrument({ state = 'welcome', tone = 'ink', className = '', label }) {
  const key = Object.prototype.hasOwnProperty.call(STATES, state) ? state : 'welcome';
  const chapter = STATES[key];
  const paper = tone === 'paper';
  const plate = paper
    ? PAPER_PLATE
    : chapter.plate === 'reflection' ? REFLECTION_PLATE : MOON_PLATE;

  return (
    <div
      className={`earthshine theia-instrument earthshine--${key} earthshine--tone-${paper ? 'paper' : 'ink'} ${className}`.trim()}
      role="img"
      aria-label={STATE_LABEL[key] || label}
      data-earthshine-state={key}
      style={{
        '--es-image': `url("${plate}")`,
        '--es-light': chapter.light,
        '--es-saturation': chapter.saturation,
        '--es-warmth': chapter.warmth,
        '--es-shadow': chapter.shadow,
      }}
    >
      <span className="earthshine__plate" aria-hidden="true" />
      <span className="earthshine__glow" aria-hidden="true" />
      <span className="earthshine__shadow" aria-hidden="true" />
      <span className="earthshine__fold" aria-hidden="true" />
      <span className="earthshine__grain" aria-hidden="true" />
    </div>
  );
}

/**
 * A hairline crescent for the onboarding chrome and the sample reply mark.
 * Unlike the old open-ring sigil it is not an emblem or a diagram: it is the
 * same object the story is about, reduced to two overlapping circles.
 */
export function EarthshineGlyph({ size = 19, className = '' }) {
  const id = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const gold = `es-gold-${id}`;
  const crescent = `es-crescent-${id}`;
  return (
    <svg
      className={`earthshine-glyph ${className}`.trim()}
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <linearGradient id={gold} x1="11" y1="5" x2="27" y2="28" gradientUnits="userSpaceOnUse">
          <stop stopColor="#F8EBCE" />
          <stop offset="0.5" stopColor="#E3C48C" />
          <stop offset="1" stopColor="#B0824A" />
        </linearGradient>
        <mask id={crescent}>
          <rect width="32" height="32" fill="#000" />
          <circle cx="16" cy="16" r="10.5" fill="#fff" />
          {/* Shifted just enough to leave a slender lit edge on the right. */}
          <circle cx="12.2" cy="15.4" r="10.5" fill="#000" />
        </mask>
      </defs>
      {/* The unlit disc is present too — faintly, as earthshine would show it. */}
      <circle cx="16" cy="16" r="10.5" fill="#B9C5CE" opacity="0.18" />
      <circle cx="16" cy="16" r="10.5" fill={`url(#${gold})`} mask={`url(#${crescent})`} />
    </svg>
  );
}
