import React from 'react';

/**
 * Theia's 2026 identity is intentionally faceless.
 *
 * TheiaSigil / TheiaMark remains the small shared brand emblem across the app
 * shell, Home, Letters, and Chat.
 *
 * TheiaInstrument carries the onboarding visual narrative: Earthshine and
 * Revelation rather than Illumination. Instead of geometric astrolabe rings or
 * decorative ticks, eight cohesive lunar and atmospheric compositions tell a
 * single story: nothing is added to the Moon — through reflection, what has
 * always been within gently becomes visible.
 */

const SIGIL_STAR = (cx, cy, radius) => {
  const k = radius * 0.085;
  const r = radius;
  return [
    `M${cx} ${cy - r}`,
    `Q${cx + k} ${cy - k} ${cx + r} ${cy}`,
    `Q${cx + k} ${cy + k} ${cx} ${cy + r}`,
    `Q${cx - k} ${cy + k} ${cx - r} ${cy}`,
    `Q${cx - k} ${cy - k} ${cx} ${cy - r}`,
    'Z',
  ].join(' ');
};

export function TheiaSigil({ size = 32, luminous = false, complete = false, className = '', title }) {
  const id = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const gold = `sigil-gold-${id}`;
  const glow = `sigil-glow-${id}`;
  const openRing = 'M19.4 54.7A26 26 0 1 1 44.6 54.7';
  return (
    <svg
      className={`theia-sigil ${luminous ? 'theia-sigil--luminous' : ''} ${className}`.trim()}
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
        <linearGradient id={gold} x1="8" y1="8" x2="56" y2="58" gradientUnits="userSpaceOnUse">
          <stop stopColor="#F7E7C6" />
          <stop offset=".46" stopColor="#E3C48C" />
          <stop offset="1" stopColor="#B98A4E" />
        </linearGradient>
        <filter id={glow} x="-40%" y="-40%" width="180%" height="180%">
          <feGaussianBlur stdDeviation="1.5" result="b" />
          <feMerge>
            <feMergeNode in="b" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>
      <g filter={luminous ? `url(#${glow})` : undefined}>
        <path
          d={complete ? 'M32 6A26 26 0 1 1 31.99 6' : openRing}
          stroke={`url(#${gold})`}
          strokeWidth={complete ? 2.6 : 2.2}
          strokeLinecap="round"
        />
        <g transform="translate(32 31) scale(0.74 1) translate(-32 -31)">
          <path d={SIGIL_STAR(32, 31, 11)} fill={`url(#${gold})`} opacity="0.97" />
        </g>
      </g>
    </svg>
  );
}

export function TheiaMark({ size = 24, luminous = false, complete = false, className = '' }) {
  return (
    <span
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      style={{ width: size, height: size }}
      aria-hidden="true"
    >
      <TheiaSigil size="100%" luminous={luminous} complete={complete} />
    </span>
  );
}

/* ------------------------------------------------------------------ */
/* Earthshine & Revelation: one unchanging Moon, revealed by light.   */
/* ------------------------------------------------------------------ */

const STATE_LABEL = {
  welcome: 'هلال ماه با تابش زمین: بخش تاریک ماه با بازتاب نور زمین به‌آرامی دیده می‌شود',
  night: 'شب: ماهِ کامل در سایهٔ آرام شب پیش از تابش نور',
  veil: 'حریم انتخابی: سایه‌ای که در میانه کنار می‌رود و بخشی از ماه را روشن می‌کند',
  sight: 'دیدن از راه بازتاب: نور زمین به سوی ماه برمی‌گردد و نقش‌های پنهانش را آشکار می‌کند',
  dawn: 'سپیده: هلال سحرگاهی بر فراز مهِ آرامِ پیش از طلوع',
  orbits: 'سه نور: خورشید، ماهِ زمین‌تاب و سپیده در یک آسمان پیوسته',
  pulse: 'یادآوری: بازتاب آرام هلال ماه در آبِ ساکن',
  seal: 'آشکارشدن کامل: همان ماهِ همیشگی، این‌بار یکسره دیده و روشن با رگه‌های طلاییِ درونی',
};

/**
 * Exact spherical crescent path in canonical [-100..100] coordinates.
 * `phase` in (0, 1]: 0 = dark, 0.5 = half, 1 = full disc.
 * Sunlit limb sits on the +x side before group rotation.
 */
export function lunarLitPath(phase, r = 100) {
  const f = Math.max(0.002, Math.min(1, phase));
  if (f >= 0.999) {
    return `M0 ${-r}A${r} ${r} 0 1 1 0 ${r}A${r} ${r} 0 1 1 0 ${-r}Z`;
  }
  const rx = (r * Math.abs(1 - 2 * f)).toFixed(2);
  const sweep = f < 0.5 ? 0 : 1;
  return `M0 ${-r}A${r} ${r} 0 0 1 0 ${r}A${rx} ${r} 0 0 ${sweep} 0 ${-r}Z`;
}

/** Canonical lunar maria (basaltic seas) shared unchanged across every screen. */
const LUNAR_MARIA = [
  // Mare Imbrium
  'M-48 -58C-26 -72 2 -66 8 -46C14 -28 2 -12 -22 -14C-44 -16 -62 -36 -48 -58Z',
  // Oceanus Procellarum
  'M-72 -38C-56 -44 -44 -22 -46 2C-48 24 -34 38 -46 52C-58 64 -80 46 -86 18C-90 -8 -84 -32 -72 -38Z',
  // Mare Serenitatis
  'M12 -46C28 -54 46 -42 44 -24C42 -10 22 -8 12 -18C4 -26 4 -40 12 -46Z',
  // Mare Tranquillitatis & Fecunditatis
  'M16 -10C34 -18 58 -8 66 10C74 26 62 42 46 36C34 30 34 16 18 14C8 12 8 -2 16 -10Z',
  // Mare Crisium
  'M60 -30C68 -34 76 -26 74 -18C72 -12 62 -12 58 -18C55 -24 56 -28 60 -30Z',
  // Mare Nubium & Humorum
  'M-42 20C-26 14 -10 26 -12 44C-14 58 -32 66 -46 54C-56 44 -54 26 -42 20Z',
];

/** Subtle inner-gold mineral filaments nestled along the lunar ridges. */
const GOLD_VEINS = [
  'M-12 -44Q4 -34 10 -16Q16 -2 32 4',
  'M24 -12Q44 -6 56 12',
  'M-54 -14Q-42 8 -46 34',
  'M-34 46Q-14 52 8 42',
];

/**
 * Shared sculptural Moon in canonical coordinates scaled to `radius` at `(cx, cy)`.
 * The exact same lunar geography persists across every state; only the
 * angle and degree of direct light and reflected Earthshine change.
 */
export function LunarSphere({
  ids,
  cx,
  cy,
  radius,
  tilt = -24,
  sunPhase = 0.11,
  earthshine = 0.5,
  goldReveal = 0.25,
  mariaStrength = 0.35,
  rimHighlight = true,
  tone = 'ink',
}) {
  const scale = (radius / 100).toFixed(4);
  const paper = tone === 'paper';
  const lit = lunarLitPath(sunPhase, 100);

  return (
    <g transform={`translate(${cx} ${cy}) scale(${scale}) rotate(${tilt})`}>
      <g clipPath={`url(#${ids.moonClip})`}>
        {/* 1. Shadowed sphere body */}
        <circle cx="0" cy="0" r="100" fill={`url(#${ids.moonDark})`} />

        {/* 2. Reflected Earthshine wash across the dark hemisphere */}
        {earthshine > 0 && (
          <circle
            cx="0"
            cy="0"
            r="100"
            fill={`url(#${ids.earthshineWash})`}
            opacity={earthshine}
          />
        )}

        {/* 3. Lunar maria — always present, gradually revealed by light */}
        <g opacity={mariaStrength} filter={`url(#${ids.mariaBlur})`}>
          {LUNAR_MARIA.map((d, i) => (
            <path key={i} d={d} fill={`url(#${ids.mariaFill})`} />
          ))}
        </g>

        {/* 4. Subtle bright lunar highlands around southern Tycho region */}
        <ellipse
          cx="-14"
          cy="64"
          rx="26"
          ry="16"
          fill={paper ? '#D9CFC0' : '#DCE7F6'}
          opacity={Math.min(0.16, earthshine * 0.18 + sunPhase * 0.08)}
          filter={`url(#${ids.mariaBlur})`}
        />

        {/* 5. Inherent gold filaments revealed where reflection reaches */}
        {goldReveal > 0 && (
          <g fill="none" stroke={`url(#${ids.goldVein})`} strokeLinecap="round">
            {GOLD_VEINS.map((d, i) => (
              <path
                key={i}
                d={d}
                strokeWidth={i < 2 ? '1.35' : '1.05'}
                opacity={goldReveal * (i % 2 === 0 ? 0.72 : 0.52)}
                filter={`url(#${ids.softSm})`}
              />
            ))}
            {GOLD_VEINS.map((d, i) => (
              <path
                key={`crisp-${i}`}
                d={d}
                strokeWidth={i < 2 ? '0.65' : '0.5'}
                opacity={goldReveal * 0.85}
              />
            ))}
          </g>
        )}

        {/* 6. Soft atmospheric penumbra along the solar terminator */}
        <path
          d={lit}
          fill={`url(#${ids.sunLit})`}
          opacity={sunPhase > 0.9 ? 0.55 : 0.42}
          filter={`url(#${ids.terminatorBlur})`}
        />

        {/* 7. Crisp sunlit crescent / revealed solar face */}
        <path
          d={lit}
          fill={`url(#${ids.sunLit})`}
          opacity={sunPhase > 0.9 ? 0.72 : 0.96}
        />

        {/* 8. Re-tint maria gently on full illumination so topography stays rich */}
        {sunPhase > 0.7 && (
          <g opacity={(sunPhase - 0.7) * 1.15} filter={`url(#${ids.mariaBlur})`}>
            {LUNAR_MARIA.map((d, i) => (
              <path key={`full-${i}`} d={d} fill={paper ? '#7D756A' : '#4C5361'} />
            ))}
          </g>
        )}

        {/* 9. Subtle spherical limb shading for genuine 3D depth */}
        <circle cx="0" cy="0" r="100" fill={`url(#${ids.sphereShade})`} />
      </g>

      {/* 10. Delicate sunlit rim specular & cool Earthshine edge whisper */}
      {rimHighlight && (
        <path
          d="M0 -99.5A99.5 99.5 0 0 1 0 99.5"
          stroke={paper ? '#FBF4E4' : '#FFF7E4'}
          strokeWidth={sunPhase < 0.15 ? '1.15' : '0.85'}
          strokeLinecap="round"
          opacity={sunPhase < 0.05 ? 0.48 : 0.82}
        />
      )}
      {earthshine > 0.2 && (
        <path
          d="M0 -99.5A99.5 99.5 0 0 0 0 99.5"
          stroke={paper ? '#8B98AA' : '#A8C1DF'}
          strokeWidth="0.75"
          strokeLinecap="round"
          opacity={earthshine * 0.38}
        />
      )}
    </g>
  );
}

/** Shared SVG <defs> builder keyed per component instance so IDs never collide. */
export function useEarthshineDefs(prefix, tone = 'ink') {
  const uid = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const id = (name) => `${prefix}-${name}-${uid}`;
  const ids = {
    moonClip: id('mc'),
    moonDark: id('md'),
    earthshineWash: id('ew'),
    mariaFill: id('mf'),
    sunLit: id('sl'),
    goldVein: id('gv'),
    sphereShade: id('ss'),
    earthLimb: id('el'),
    earthBeam: id('eb'),
    dawnMist: id('dm'),
    veilLeft: id('vl'),
    veilRight: id('vr'),
    waterPlane: id('wp'),
    corona: id('co'),
    helios: id('he'),
    eos: id('eo'),
    mariaBlur: id('mb'),
    terminatorBlur: id('tb'),
    softSm: id('sm'),
    softLg: id('lg'),
    waterClip: id('wc'),
  };

  const paper = tone === 'paper';

  const defs = (
    <defs>
      <clipPath id={ids.moonClip}>
        <circle cx="0" cy="0" r="100" />
      </clipPath>
      <clipPath id={ids.waterClip}>
        <rect x="20" y="196" width="320" height="140" />
      </clipPath>

      {/* Deep shadowed lunar surface */}
      <radialGradient id={ids.moonDark} cx="35%" cy="38%" r="78%">
        <stop offset="0%" stopColor={paper ? '#3C3C40' : '#181C26'} />
        <stop offset="65%" stopColor={paper ? '#29282B' : '#0E1016'} />
        <stop offset="100%" stopColor={paper ? '#1E1D1F' : '#090A0E'} />
      </radialGradient>

      {/* Cool silver-blue reflected Earthshine across the dark face */}
      <radialGradient id={ids.earthshineWash} cx="28%" cy="46%" r="76%">
        <stop offset="0%" stopColor={paper ? '#B6C3D4' : '#A9C2E2'} stopOpacity="0.95" />
        <stop offset="52%" stopColor={paper ? '#7D8B9F' : '#657D9F'} stopOpacity="0.62" />
        <stop offset="100%" stopColor={paper ? '#485263' : '#253145'} stopOpacity="0.18" />
      </radialGradient>

      {/* Organic lunar basalt seas */}
      <radialGradient id={ids.mariaFill} cx="50%" cy="50%" r="60%">
        <stop offset="0%" stopColor={paper ? '#202229' : '#090C12'} stopOpacity="0.92" />
        <stop offset="100%" stopColor={paper ? '#343844' : '#131924'} stopOpacity="0.45" />
      </radialGradient>

      {/* Direct solar illumination — warm lunar cream & gold */}
      <linearGradient id={ids.sunLit} x1="-20" y1="-90" x2="100" y2="85" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stopColor={paper ? '#FBF3DF' : '#FFF8E8'} />
        <stop offset="48%" stopColor={paper ? '#E3C084' : '#ECCE97'} />
        <stop offset="100%" stopColor={paper ? '#BA8848' : '#C29153'} />
      </linearGradient>

      {/* Inner gold revealed inside the shadow */}
      <linearGradient id={ids.goldVein} x1="-70" y1="-60" x2="70" y2="60" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stopColor={paper ? '#D29F56' : '#F4DFB4'} />
        <stop offset="55%" stopColor={paper ? '#B47E3A' : '#D9B679'} />
        <stop offset="100%" stopColor={paper ? '#8F5E24' : '#A97B41'} />
      </linearGradient>

      {/* Spherical limb darkening so the Moon reads as a three-dimensional orb */}
      <radialGradient id={ids.sphereShade} cx="46%" cy="46%" r="54%">
        <stop offset="68%" stopColor="#000" stopOpacity="0" />
        <stop offset="92%" stopColor="#050608" stopOpacity={paper ? '0.18' : '0.32'} />
        <stop offset="100%" stopColor="#050608" stopOpacity={paper ? '0.32' : '0.52'} />
      </radialGradient>

      {/* Soft atmospheric corona around the Moon */}
      <radialGradient id={ids.corona} cx="50%" cy="50%" r="50%">
        <stop offset="0%" stopColor={paper ? '#D8B37B' : '#96B2D6'} stopOpacity={paper ? '0.28' : '0.26'} />
        <stop offset="55%" stopColor={paper ? '#E7CFA5' : '#D9B679'} stopOpacity={paper ? '0.12' : '0.10'} />
        <stop offset="100%" stopColor={paper ? '#F2EDE4' : '#0A0B0D'} stopOpacity="0" />
      </radialGradient>

      {/* Earth atmospheric arc in `sight` */}
      <radialGradient id={ids.earthLimb} cx="46%" cy="18%" r="76%">
        <stop offset="0%" stopColor="#C8E1F4" stopOpacity="0.92" />
        <stop offset="22%" stopColor="#6E9AC2" stopOpacity="0.58" />
        <stop offset="58%" stopColor="#1A2C44" stopOpacity="0.32" />
        <stop offset="100%" stopColor="#0A0B0D" stopOpacity="0" />
      </radialGradient>

      {/* Reflected light stream from Earth to Moon */}
      <linearGradient id={ids.earthBeam} x1="88" y1="278" x2="220" y2="118" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stopColor="#A6CBE8" stopOpacity="0.24" />
        <stop offset="48%" stopColor="#94B6D8" stopOpacity="0.11" />
        <stop offset="100%" stopColor="#94B6D8" stopOpacity="0" />
      </linearGradient>

      {/* Pre-dawn horizon mist */}
      <linearGradient id={ids.dawnMist} x1="180" y1="208" x2="180" y2="322" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stopColor="#E4B884" stopOpacity="0" />
        <stop offset="42%" stopColor="#D79F68" stopOpacity="0.28" />
        <stop offset="68%" stopColor="#F0CF9B" stopOpacity="0.48" />
        <stop offset="100%" stopColor="#0A0B0D" stopOpacity="0" />
      </linearGradient>

      {/* Parted twilight veils for `veil` (Privacy on paper) */}
      <linearGradient id={ids.veilLeft} x1="62" y1="180" x2="178" y2="180" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stopColor="#F2EDE4" stopOpacity="0" />
        <stop offset="55%" stopColor="#262421" stopOpacity="0.82" />
        <stop offset="92%" stopColor="#1B1917" stopOpacity="0.94" />
        <stop offset="100%" stopColor="#1B1917" stopOpacity="0" />
      </linearGradient>
      <linearGradient id={ids.veilRight} x1="298" y1="180" x2="182" y2="180" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stopColor="#F2EDE4" stopOpacity="0" />
        <stop offset="55%" stopColor="#262421" stopOpacity="0.82" />
        <stop offset="92%" stopColor="#1B1917" stopOpacity="0.94" />
        <stop offset="100%" stopColor="#1B1917" stopOpacity="0" />
      </linearGradient>

      {/* Still water surface for `pulse` (Checkins on paper) */}
      <linearGradient id={ids.waterPlane} x1="180" y1="196" x2="180" y2="318" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stopColor="#2E323B" stopOpacity="0.22" />
        <stop offset="35%" stopColor="#464E5D" stopOpacity="0.14" />
        <stop offset="100%" stopColor="#F2EDE4" stopOpacity="0" />
      </linearGradient>

      {/* Helios (diffused solar glow) & Eos (dawn mist) for `orbits` */}
      <radialGradient id={ids.helios} cx="50%" cy="50%" r="50%">
        <stop offset="0%" stopColor="#FFF6DF" stopOpacity="0.95" />
        <stop offset="18%" stopColor="#F2D194" stopOpacity="0.62" />
        <stop offset="52%" stopColor="#C89354" stopOpacity="0.18" />
        <stop offset="100%" stopColor="#0A0B0D" stopOpacity="0" />
      </radialGradient>
      <linearGradient id={ids.eos} x1="180" y1="224" x2="180" y2="334" gradientUnits="userSpaceOnUse">
        <stop offset="0%" stopColor="#E6B19A" stopOpacity="0" />
        <stop offset="46%" stopColor="#DCA58E" stopOpacity="0.32" />
        <stop offset="74%" stopColor="#ECC59B" stopOpacity="0.36" />
        <stop offset="100%" stopColor="#0A0B0D" stopOpacity="0" />
      </linearGradient>

      <filter id={ids.mariaBlur} x="-25%" y="-25%" width="150%" height="150%">
        <feGaussianBlur stdDeviation="4.5" />
      </filter>
      <filter id={ids.terminatorBlur} x="-20%" y="-20%" width="140%" height="140%">
        <feGaussianBlur stdDeviation="5.5" />
      </filter>
      <filter id={ids.softSm} x="-35%" y="-35%" width="170%" height="170%">
        <feGaussianBlur stdDeviation="2.4" />
      </filter>
      <filter id={ids.softLg} x="-40%" y="-40%" width="180%" height="180%">
        <feGaussianBlur stdDeviation="11" />
      </filter>
    </defs>
  );

  return { ids, defs };
}

export function TheiaInstrument({ state = 'welcome', tone = 'ink', className = '', label }) {
  const { ids, defs } = useEarthshineDefs('ti', tone);

  return (
    <svg
      className={`theia-instrument theia-instrument--${state} theia-instrument--tone-${tone} ${className}`.trim()}
      viewBox="0 0 360 360"
      fill="none"
      role="img"
      aria-label={label || STATE_LABEL[state] || STATE_LABEL.welcome}
      focusable="false"
    >
      {defs}

      {/* WELCOME: Earthshine crescent — cool reflected light revealing unseen basins & inner gold */}
      {state === 'welcome' && (
        <g>
          <circle cx="180" cy="174" r="146" fill={`url(#${ids.corona})`} className="ti-earthshine-breathe" />
          <ellipse cx="212" cy="198" rx="92" ry="74" fill="#F2D49B" opacity="0.06" filter={`url(#${ids.softLg})`} />
          <g fill="#DCE6F5">
            <circle cx="72" cy="96" r="1.1" opacity="0.34" />
            <circle cx="294" cy="108" r="0.9" opacity="0.28" />
            <circle cx="84" cy="252" r="0.9" opacity="0.22" />
          </g>
          <LunarSphere
            ids={ids}
            cx={180}
            cy={174}
            radius={88}
            tilt={-26}
            sunPhase={0.11}
            earthshine={0.54}
            goldReveal={0.26}
            mariaStrength={0.38}
            tone={tone}
          />
        </g>
      )}

      {/* NIGHT: The whole Moon resting unseen in quiet shadow before reflection */}
      {state === 'night' && (
        <g>
          <circle cx="180" cy="174" r="126" fill="#526585" opacity="0.04" filter={`url(#${ids.softLg})`} />
          <LunarSphere
            ids={ids}
            cx={180}
            cy={174}
            radius={90}
            tilt={-28}
            sunPhase={0.02}
            earthshine={0.12}
            goldReveal={0.03}
            mariaStrength={0.14}
            tone={tone}
          />
        </g>
      )}

      {/* VEIL (Privacy on paper): Parted twilight shadow opening softly by choice */}
      {state === 'veil' && (
        <g>
          <circle cx="180" cy="178" r="118" fill={`url(#${ids.corona})`} opacity="0.85" />
          <LunarSphere
            ids={ids}
            cx={180}
            cy={178}
            radius={74}
            tilt={-18}
            sunPhase={0.18}
            earthshine={0.48}
            goldReveal={0.36}
            mariaStrength={0.34}
            tone={tone}
          />
          {/* Soft parted shadow curtains that open gently at the center */}
          <path
            d="M54 78Q148 178 54 282L160 282Q144 178 160 78Z"
            fill={`url(#${ids.veilLeft})`}
            filter={`url(#${ids.softLg})`}
          />
          <path
            d="M306 78Q212 178 306 282L200 282Q216 178 200 78Z"
            fill={`url(#${ids.veilRight})`}
            filter={`url(#${ids.softLg})`}
          />
          <path
            d="M158 96Q144 178 158 260"
            stroke="#C79A5C"
            strokeWidth="0.9"
            opacity="0.34"
            filter={`url(#${ids.softSm})`}
          />
          <path
            d="M202 96Q216 178 202 260"
            stroke="#C79A5C"
            strokeWidth="0.9"
            opacity="0.34"
            filter={`url(#${ids.softSm})`}
          />
        </g>
      )}

      {/* SIGHT: Earth's sunlit atmosphere reflecting gentle blue-silver light onto the dark Moon */}
      {state === 'sight' && (
        <g>
          {/* Diffused Earthshine cone traveling from Earth's curved limb up to the Moon */}
          <path
            d="M44 268Q114 214 156 122L254 188Q174 264 114 316Z"
            fill={`url(#${ids.earthBeam})`}
            filter={`url(#${ids.softLg})`}
            className="ti-earthshine-breathe"
          />
          {/* Distant stars */}
          <circle cx="96" cy="84" r="1.05" fill="#DCE6F5" opacity="0.34" />
          <circle cx="302" cy="88" r="0.9" fill="#DCE6F5" opacity="0.26" />

          {/* Moon bathed in reflected Earthlight */}
          <circle cx="212" cy="134" r="106" fill={`url(#${ids.corona})`} opacity="0.92" />
          <LunarSphere
            ids={ids}
            cx={212}
            cy={134}
            radius={68}
            tilt={-34}
            sunPhase={0.09}
            earthshine={0.72}
            goldReveal={0.44}
            mariaStrength={0.44}
            tone={tone}
          />

          {/* Earth's curved atmospheric horizon in the lower-left foreground */}
          <g>
            <circle cx="34" cy="482" r="246" fill={`url(#${ids.earthLimb})`} />
            <path
              d="M28 286A246 246 0 0 1 208 320"
              stroke="#BBE0FA"
              strokeWidth="2.4"
              strokeLinecap="round"
              opacity="0.58"
              filter={`url(#${ids.softSm})`}
            />
            <path
              d="M32 284A246 246 0 0 1 198 314"
              stroke="#E6F4FF"
              strokeWidth="0.9"
              strokeLinecap="round"
              opacity="0.72"
            />
          </g>
        </g>
      )}

      {/* DAWN: Waning Earthlit crescent suspended peacefully above rising pre-dawn mist */}
      {state === 'dawn' && (
        <g>
          <circle cx="180" cy="140" r="106" fill={`url(#${ids.corona})`} opacity="0.82" />
          <LunarSphere
            ids={ids}
            cx={180}
            cy={140}
            radius={62}
            tilt={154}
            sunPhase={0.14}
            earthshine={0.58}
            goldReveal={0.34}
            mariaStrength={0.4}
            tone={tone}
          />
          {/* Gradual layers of dawn mist — light arriving without announcement */}
          <ellipse cx="180" cy="274" rx="148" ry="44" fill={`url(#${ids.dawnMist})`} filter={`url(#${ids.softLg})`} />
          <ellipse cx="180" cy="264" rx="112" ry="18" fill="#F2D09A" opacity="0.22" filter={`url(#${ids.softLg})`} />
          <path
            d="M52 268Q180 258 308 268"
            stroke="#F6DEC0"
            strokeWidth="1.2"
            strokeLinecap="round"
            opacity="0.34"
            filter={`url(#${ids.softSm})`}
          />
        </g>
      )}

      {/* ORBITS (Three Lights): Helios, Selene, and Eos united by flowing light — no rings */}
      {state === 'orbits' && (
        <g>
          {/* Eos — pale rose-gold dawn lifting gently across the lower horizon */}
          <ellipse cx="180" cy="284" rx="148" ry="46" fill={`url(#${ids.eos})`} filter={`url(#${ids.softLg})`} />
          <path
            d="M58 282Q180 272 302 282"
            stroke="#ECC5A2"
            strokeWidth="1.1"
            strokeLinecap="round"
            opacity="0.36"
            filter={`url(#${ids.softSm})`}
          />

          {/* Helios — diffused solar radiance upper-right casting light toward Selene */}
          <g className="ti-earthshine-breathe">
            <circle cx="264" cy="86" r="78" fill={`url(#${ids.helios})`} />
            <circle cx="264" cy="86" r="13" fill="#FFF4D8" opacity="0.88" filter={`url(#${ids.softSm})`} />
            <circle cx="264" cy="86" r="6.5" fill="#FFFDF6" opacity="0.96" />
          </g>

          {/* Subtle diagonal wash of sunlight reaching Selene */}
          <path
            d="M254 96L156 174"
            stroke="#F4D7A2"
            strokeWidth="48"
            strokeLinecap="round"
            opacity="0.06"
            filter={`url(#${ids.softLg})`}
          />

          {/* Selene — the Moon catching Helios's crescent while holding Earthshine */}
          <LunarSphere
            ids={ids}
            cx={162}
            cy={178}
            radius={54}
            tilt={-42}
            sunPhase={0.16}
            earthshine={0.58}
            goldReveal={0.34}
            mariaStrength={0.42}
            tone={tone}
          />
        </g>
      )}

      {/* PULSE (Checkins on paper): Crescent reflected quietly on still water */}
      {state === 'pulse' && (
        <g>
          <circle cx="180" cy="116" r="78" fill={`url(#${ids.corona})`} opacity="0.72" />
          <LunarSphere
            ids={ids}
            cx={180}
            cy={116}
            radius={43}
            tilt={-22}
            sunPhase={0.15}
            earthshine={0.48}
            goldReveal={0.32}
            mariaStrength={0.36}
            tone={tone}
          />

          {/* Still water horizon */}
          <path d="M44 196H316" stroke="#8C8375" strokeWidth="0.7" opacity="0.38" />
          <rect x="34" y="196" width="292" height="122" fill={`url(#${ids.waterPlane})`} />

          {/* Inverted lunar reflection dissolving gently in water */}
          <g clipPath={`url(#${ids.waterClip})`} opacity="0.34" filter={`url(#${ids.softSm})`}>
            <g transform="translate(0 392) scale(1 -0.82)">
              <LunarSphere
                ids={ids}
                cx={180}
                cy={116}
                radius={43}
                tilt={-22}
                sunPhase={0.15}
                earthshine={0.4}
                goldReveal={0.25}
                mariaStrength={0.25}
                rimHighlight={false}
                tone={tone}
              />
            </g>
          </g>

          {/* Gentle breathing surface ripples */}
          <ellipse
            cx="180"
            cy="244"
            rx="44"
            ry="6"
            stroke="#A87E45"
            strokeWidth="0.9"
            opacity="0.45"
            className="ti-ripple"
          />
          <ellipse
            cx="180"
            cy="244"
            rx="68"
            ry="9"
            stroke="#847C70"
            strokeWidth="0.7"
            opacity="0.32"
            className="ti-ripple ti-ripple--late"
          />
        </g>
      )}

      {/* SEAL (Ready): The exact same Moon, wholly revealed in serene pearl & inner gold */}
      {state === 'seal' && (
        <g>
          <circle cx="180" cy="174" r="148" fill={`url(#${ids.corona})`} opacity="0.95" className="ti-earthshine-breathe" />
          <circle cx="180" cy="174" r="108" fill="#F2D8A6" opacity="0.16" filter={`url(#${ids.softLg})`} />
          <LunarSphere
            ids={ids}
            cx={180}
            cy={174}
            radius={90}
            tilt={-26}
            sunPhase={1}
            earthshine={0.78}
            goldReveal={0.78}
            mariaStrength={0.42}
            rimHighlight={false}
            tone={tone}
          />
          {/* Gentle luminous rim of completion */}
          <circle
            cx="180"
            cy="174"
            r="90.5"
            stroke="#F7E5BF"
            strokeWidth="1.2"
            opacity="0.64"
            filter={`url(#${ids.softSm})`}
          />
        </g>
      )}
    </svg>
  );
}
