import React from 'react';

/**
 * Core symbolic mark of Theia (تیا) — Goddess of Sight, Light, Sun, Moon & Dawn.
 * Unites Selene's crescent cradle, Helios's rising solar arc, and Eos's four-point
 * star of clarity in a calm geometric-organic form legible from 10px to 48px.
 */
export function TheiaMark({ size = 24, luminous = false, className = '' }) {
  const crescentStroke = luminous ? '#d9dcf7' : 'currentColor';
  const dawnStroke = luminous ? '#f6cf98' : 'currentColor';
  const starFill = luminous ? '#f6cf98' : 'currentColor';

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()}
      aria-hidden="true"
      focusable="false"
    >
      {/* Graceful outer crescent moon cradling inner vision */}
      <path
        d="M21.8 5.4A11.2 11.2 0 1 0 26.6 20.6A8.9 8.9 0 0 1 21.8 5.4Z"
        stroke={crescentStroke}
        strokeWidth="1.55"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {/* Subtle rising solar horizon arc */}
      <path
        d="M9.6 20.2C11.4 17.6 14.3 16.2 17.5 16.4"
        stroke={dawnStroke}
        strokeWidth="1.35"
        strokeLinecap="round"
        opacity={luminous ? '0.88' : '0.68'}
      />
      {/* Four-point star of first light & clarity */}
      <path
        d="M20.5 7.6L21.55 10.65L24.6 11.7L21.55 12.75L20.5 15.8L19.45 12.75L16.4 11.7L19.45 10.65L20.5 7.6Z"
        fill={starFill}
      />
    </svg>
  );
}

/**
 * Compact 4-point dawn-and-crescent seal used on tiny badges (10px–14px)
 * such as the unread letter wax seal on Home.
 */
export function TheiaSealIcon({ size = 11, className = '' }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <path
        d="M8 1.6L9.45 6.05L13.9 7.5L9.45 8.95L8 13.4L6.55 8.95L2.1 7.5L6.55 6.05L8 1.6Z"
        fill="currentColor"
      />
    </svg>
  );
}

/**
 * Centerpiece Goddess & Celestial Orbit emblem for the Home screen ("بیا حرف بزنیم").
 * Preserves the exact 240x240 orbital geometry and calm breathing motion of Home
 * while expressing Theia herself framed by the Sun, Crescent Moon, and Dawn rays.
 */
export function TheiaHomeArt() {
  return (
    <span className="home-conversation-art" aria-hidden="true">
      <svg className="home-conversation-orbit" viewBox="0 0 240 240" fill="none" focusable="false">
        {/* Outer orbital traces of Sun & Moon */}
        <path
          className="home-orbit-trace home-orbit-trace--first"
          d="M187 48C157 19 113 21 81 38C39 61 21 105 35 148C45 182 73 205 107 211"
        />
        <path
          className="home-orbit-trace home-orbit-trace--second"
          d="M58 177C83 205 127 219 163 202C200 185 222 151 211 109C205 88 195 75 184 65"
        />
        {/* Golden Dawn horizon shimmer */}
        <path
          className="home-orbit-accent"
          d="M58 78C76 56 101 46 126 47"
        />
        {/* Solar pearl (Helios) on lower orbital node */}
        <circle className="home-orbit-point home-orbit-point--sun" cx="111" cy="211" r="3.2" />
        {/* Delicate crescent moon (Selene) on upper-right orbital node */}
        <path
          className="home-orbit-crescent"
          d="M184.5 52.5A6.2 6.2 0 1 0 190.2 60.8A4.9 4.9 0 0 1 184.5 52.5Z"
        />
      </svg>

      {/* Central Theia Goddess-Crescent-Dawn Medallion */}
      <svg
        className="home-conversation-mark theia-home-emblem"
        width="86"
        height="86"
        viewBox="0 0 120 120"
        fill="none"
        focusable="false"
      >
        {/* Subtle lower dawn rays (Eos / Helios) */}
        <g stroke="#f6cf98" strokeLinecap="round" opacity="0.58">
          <line x1="60" y1="96" x2="60" y2="108" strokeWidth="1.35" />
          <line x1="46" y1="93.5" x2="41.5" y2="104" strokeWidth="1.2" />
          <line x1="74" y1="93.5" x2="78.5" y2="104" strokeWidth="1.2" />
          <line x1="34" y1="86.5" x2="26.5" y2="94.5" strokeWidth="1.05" opacity="0.75" />
          <line x1="86" y1="86.5" x2="93.5" y2="94.5" strokeWidth="1.05" opacity="0.75" />
          <line x1="25.5" y1="75.5" x2="16.5" y2="80" strokeWidth="0.95" opacity="0.5" />
          <line x1="94.5" y1="75.5" x2="103.5" y2="80" strokeWidth="0.95" opacity="0.5" />
        </g>

        {/* Outer celestial halo circle */}
        <circle cx="60" cy="56" r="35" stroke="#d5d9f7" strokeOpacity="0.26" strokeWidth="1.1" />

        {/* Luminous Crescent Moon forming Theia's veil & sanctuary */}
        <path
          d="M64 21.5C82.8 23.5 96.5 40.3 94.4 59.1C92.3 77.9 75.5 91.5 56.7 89.5C43.2 88 32.1 79.1 27.5 66.8C33.4 76.2 44.3 82.1 56.4 81.2C72.8 80 85.2 65.8 84 49.4C83.1 36.8 75.1 26.2 64 21.5Z"
          stroke="#e4e6fa"
          strokeWidth="2.1"
          strokeLinejoin="round"
          fill="#d5d9f7"
          fillOpacity="0.08"
        />

        {/* Serene minimalist profile of Theia nested within the crescent */}
        <path
          d="M53.5 24.5C46.5 28.8 43.2 35.8 43.8 43.2C44.1 46.2 41.2 49.8 39.2 52.8C38.4 54 39.2 55.1 40.9 55.6C40.1 57.4 40.5 59.1 42.1 59.8C41.4 61.5 42.2 63.4 44.2 64.2C46.8 65.2 51.2 63.5 54.8 63.8C58.8 64.2 61.2 68.5 59.5 74.8C58.2 79.5 53.5 83.2 47.8 85.2"
          stroke="#f3efe8"
          strokeWidth="2.05"
          strokeLinecap="round"
          strokeLinejoin="round"
        />

        {/* Flowing hair / celestial stream line */}
        <path
          d="M53.5 24.5C50.8 33.2 55.5 41.2 63.8 47.5C71.8 53.6 74.8 63.5 70.2 75.8C68.1 81.3 63.8 85.8 58.2 88.2"
          stroke="#d5d9f7"
          strokeOpacity="0.82"
          strokeWidth="1.65"
          strokeLinecap="round"
        />

        {/* Delicate closed meditative eyelash */}
        <path
          d="M42.8 44.4C41.1 44.8 39.6 44.5 38.5 43.6"
          stroke="#f3efe8"
          strokeWidth="1.35"
          strokeLinecap="round"
        />

        {/* Four-point Star of Sight & Dawn Clarity */}
        <path
          d="M65.5 31L67.3 36.2L72.5 38L67.3 39.8L65.5 45L63.7 39.8L58.5 38L63.7 36.2L65.5 31Z"
          fill="#f6cf98"
        />
      </svg>
    </span>
  );
}

/**
 * Quiet celestial empty-inbox illustration for Letters ("هنوز نامه‌ای نرسیده").
 */
export function TheiaLetterEmpty({ size = 44 }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 48 48"
      fill="none"
      className="theia-letter-empty"
      aria-hidden="true"
      focusable="false"
    >
      <path
        d="M9 17.5C9 15.57 10.57 14 12.5 14H35.5C37.43 14 39 15.57 39 17.5V33.5C39 35.43 37.43 37 35.5 37H12.5C10.57 37 9 35.43 9 33.5V17.5Z"
        stroke="currentColor"
        strokeWidth="1.35"
      />
      <path
        d="M10.5 16L24 26.2L37.5 16"
        stroke="currentColor"
        strokeWidth="1.35"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M28.5 5.2A5.2 5.2 0 1 0 32.8 11.4A4.1 4.1 0 0 1 28.5 5.2Z"
        stroke="#d5d9f7"
        strokeOpacity="0.65"
        strokeWidth="1.25"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M21 6.5L21.65 8.35L23.5 9L21.65 9.65L21 11.5L20.35 9.65L18.5 9L20.35 8.35L21 6.5Z"
        fill="#f6cf98"
        fillOpacity="0.78"
      />
    </svg>
  );
}
