import React from 'react';

export function TheiaLightMark({ size = 18, className = '' }) {
  return (
    <svg
      className={`theia-light-mark ${className}`.trim()}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
      focusable="false"
    >
      <circle cx="12" cy="12" r="10.25" stroke="currentColor" strokeOpacity=".5" strokeWidth=".8" />
      <circle cx="12" cy="10" r="4.1" fill="#F6CF98" />
      <path d="M3.5 13.3h17M7 16.4c1.45 1.15 3.13 1.7 5 1.7s3.55-.55 5-1.7" stroke="currentColor" strokeOpacity=".72" strokeWidth=".8" strokeLinecap="round" />
    </svg>
  );
}

/**
 * Theia's onboarding presence is light, not a person: a quiet aperture at the
 * horizon. The rings borrow the balance of Persian geometric drawing without
 * copying sacred, national, or architectural motifs.
 */
export default function TheiaLightArt({ scene = 'welcome', className = '' }) {
  return (
    <svg
      className={`theia-light-art theia-light-art--${scene} ${className}`.trim()}
      viewBox="0 0 400 430"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <radialGradient id="theia-halo" cx="0" cy="0" r="1" gradientTransform="matrix(0 155 -155 0 200 208)" gradientUnits="userSpaceOnUse">
          <stop stopColor="#F6CF98" stopOpacity=".30" />
          <stop offset=".52" stopColor="#E5C9A3" stopOpacity=".12" />
          <stop offset="1" stopColor="#E5C9A3" stopOpacity="0" />
        </radialGradient>
        <radialGradient id="theia-core" cx="0" cy="0" r="1" gradientTransform="matrix(82 115 -113 81 174 154)" gradientUnits="userSpaceOnUse">
          <stop stopColor="#FFF4D8" />
          <stop offset=".42" stopColor="#F6CF98" />
          <stop offset="1" stopColor="#B48352" />
        </radialGradient>
        <linearGradient id="theia-shadow" x1="200" y1="231" x2="200" y2="366" gradientUnits="userSpaceOnUse">
          <stop stopColor="#26241F" stopOpacity=".96" />
          <stop offset="1" stopColor="#171612" />
        </linearGradient>
        <linearGradient id="theia-ray" x1="200" y1="174" x2="200" y2="375" gradientUnits="userSpaceOnUse">
          <stop stopColor="#F6CF98" stopOpacity=".42" />
          <stop offset="1" stopColor="#F6CF98" stopOpacity="0" />
        </linearGradient>
        <filter id="theia-blur" x="43" y="45" width="314" height="314" colorInterpolationFilters="sRGB" filterUnits="userSpaceOnUse">
          <feGaussianBlur stdDeviation="25" />
        </filter>
      </defs>

      <circle className="theia-light-halo" cx="200" cy="208" r="150" fill="url(#theia-halo)" />
      <circle className="theia-light-bloom" cx="200" cy="205" r="94" fill="#F6CF98" fillOpacity=".24" filter="url(#theia-blur)" />

      {/* Quiet measured orbit lines: a lens for attention, not a celestial chart. */}
      <path d="M60 259C69 151 128 75 200 75C272 75 331 151 340 259" className="theia-orbit theia-orbit--outer" />
      <path d="M89 258C96 171 142 117 200 117C258 117 304 171 311 258" className="theia-orbit theia-orbit--inner" />
      <path d="M123 253C129 198 157 163 200 163C243 163 271 198 277 253" className="theia-orbit theia-orbit--near" />

      {/* A luminous body held at the edge of visibility. No face, figure, or mascot. */}
      <circle cx="200" cy="205" r="70" fill="url(#theia-core)" className="theia-light-core" />
      <circle cx="200" cy="205" r="70" className="theia-core-outline" />
      <path d="M200 106V77M112 137L91 116M288 137L309 116" className="theia-ray-line" />
      <circle cx="200" cy="74" r="2" className="theia-point theia-point--top" />
      <circle cx="82" cy="266" r="2.5" className="theia-point theia-point--left" />
      <circle cx="318" cy="266" r="2.5" className="theia-point theia-point--right" />

      {/* Horizon and its reflected line: the moment of noticing, before certainty. */}
      <path d="M54 267H346V430H54Z" fill="url(#theia-shadow)" className="theia-shadow-land" />
      <path d="M54 267H346" className="theia-horizon" />
      <path d="M93 279H307" className="theia-reflection" />
      <path d="M139 292C159 301 181 305 200 305C219 305 241 301 261 292" className="theia-reflection theia-reflection--arc" />
      <path d="M200 270V366" stroke="url(#theia-ray)" strokeWidth="1.2" />
      <path d="M91 268C117 314 155 340 200 340C245 340 283 314 309 268" className="theia-field-edge" />
      <path d="M122 268C140 300 168 318 200 318C232 318 260 300 278 268" className="theia-field-edge theia-field-edge--inner" />

      <g className="theia-scene-privacy" aria-hidden="true">
        <circle cx="200" cy="205" r="84" className="theia-privacy-ring" />
        <path d="M127 256C145 272 171 282 200 282C229 282 255 272 273 256" className="theia-privacy-arc" />
      </g>
      <g className="theia-scene-pause" aria-hidden="true">
        <path d="M154 353H246" className="theia-scene-line" />
        <circle cx="200" cy="353" r="2.5" className="theia-point" />
      </g>
      <g className="theia-scene-reflect" aria-hidden="true">
        <path d="M165 321C177 328 189 332 200 332C211 332 223 328 235 321" className="theia-scene-line" />
        <path d="M176 337C184 341 192 343 200 343C208 343 216 341 224 337" className="theia-scene-line theia-scene-line--soft" />
      </g>
      <g className="theia-scene-dawn" aria-hidden="true">
        <path d="M200 65V48M167 75L158 59M233 75L242 59" className="theia-dawn-ray" />
        <circle cx="200" cy="43" r="2" className="theia-point" />
      </g>
      <g className="theia-scene-ready" aria-hidden="true">
        <path d="M74 360C111 382 153 394 200 394C247 394 289 382 326 360" className="theia-ready-orbit" />
        <circle cx="200" cy="394" r="3" className="theia-point" />
      </g>
    </svg>
  );
}
