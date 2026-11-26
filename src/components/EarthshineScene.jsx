import React from 'react';

const SCENES = {
  welcome: { sky: ['#080d18', '#111b2b', '#202c3a'], moon: [292, 122, 39], ash: 0.12, stars: 8, roof: 0.88 },
  privacy: { sky: ['#111720', '#1d2935', '#29323a'], moon: [195, 154, 35], ash: 0.28, stars: 5, roof: 0.86 },
  night: { sky: ['#10172c', '#17223a', '#263348'], moon: [270, 108, 42], ash: 0.25, stars: 22, roof: 0.96 },
  seeing: { sky: ['#10182c', '#1b2940', '#354253'], moon: [268, 94, 52], ash: 0.53, stars: 12, roof: 0.98 },
  dawn: { sky: ['#182438', '#313b4d', '#90766c'], moon: [272, 82, 47], ash: 0.72, stars: 7, roof: 0.99 },
  three: { sky: ['#263044', '#59606a', '#c49b79'], moon: [274, 75, 43], ash: 1, stars: 4, roof: 1 },
  checkins: { sky: ['#253044', '#566171', '#b88f75'], moon: [278, 70, 40], ash: 1, stars: 3, roof: 1 },
  final: { sky: ['#657384', '#a3a5a0', '#e5c9a5'], moon: [292, 104, 38], ash: 1, stars: 0, roof: 1 },
};

const STAR_POINTS = [
  [33, 66], [78, 112], [114, 51], [142, 152], [182, 81], [221, 55], [345, 64], [313, 155],
  [44, 195], [105, 213], [164, 188], [225, 152], [350, 218], [306, 44], [255, 225], [64, 35],
  [134, 94], [23, 145], [190, 218], [330, 103], [89, 174], [236, 115],
];

/** A single east-facing Persian rooftop, revisited as the eye adjusts before dawn. */
export default function EarthshineScene({ scene = 'welcome', label }) {
  const s = SCENES[scene] || SCENES.welcome;
  const [mx, my, r] = s.moon;
  const shutters = scene === 'privacy';
  const dawn = ['dawn', 'three', 'checkins', 'final'].includes(scene);
  const lantern = scene === 'checkins' || scene === 'final';
  const final = scene === 'final';
  const starCount = s.stars;
  const starPoints = STAR_POINTS.slice(0, starCount);
  const labelText = label || 'آسمانِ سپیده‌دم، ماهِ هلال با نورِ زمین و بامِ آرام';
  return (
    <svg className={`earthshine-scene earthshine-scene--${scene}`} viewBox="0 0 390 470" role="img" aria-label={labelText} preserveAspectRatio="xMidYMid slice">
      <defs>
        <linearGradient id="es-sky" x1="0" y1="0" x2="0" y2="1">
          <stop stopColor={s.sky[0]} /><stop offset=".65" stopColor={s.sky[1]} /><stop offset="1" stopColor={s.sky[2]} />
        </linearGradient>
        <linearGradient id="es-horizon" x1="0" y1="0" x2="1" y2="0">
          <stop stopColor="#b89083" stopOpacity="0"/><stop offset=".5" stopColor={final ? '#f2d5ad' : '#d5a38a'} stopOpacity={dawn ? '.76' : '0'}/><stop offset="1" stopColor="#b89083" stopOpacity="0"/>
        </linearGradient>
        <radialGradient id="es-earth" cx="38%" cy="34%" r="72%">
          <stop stopColor="#a8b8c0"/><stop offset=".58" stopColor="#778895"/><stop offset="1" stopColor="#46535e"/>
        </radialGradient>
        <linearGradient id="es-crescent" x1="0" y1="0" x2="1" y2="1">
          <stop stopColor="#fff8e8"/><stop offset=".48" stopColor="#e5e9e8"/><stop offset="1" stopColor="#aebbc2"/>
        </linearGradient>
        <radialGradient id="es-moon-wash"><stop stopColor="#dce8ee" stopOpacity=".13"/><stop offset="1" stopColor="#dce8ee" stopOpacity="0"/></radialGradient>
        <radialGradient id="es-lantern"><stop stopColor="#f7cc83" stopOpacity=".42"/><stop offset="1" stopColor="#f7cc83" stopOpacity="0"/></radialGradient>
        <linearGradient id="es-rug" x1="0" y1="0" x2="1" y2="1"><stop stopColor="#746253"/><stop offset="1" stopColor="#403d3b"/></linearGradient>
        <linearGradient id="es-ray" x1="0" y1="0" x2="0" y2="1"><stop stopColor="#f6d6a7" stopOpacity=".75"/><stop offset="1" stopColor="#f6d6a7" stopOpacity="0"/></linearGradient>
        <mask id="es-crescent-mask"><rect width="390" height="470" fill="black"/><circle cx={mx} cy={my} r={r} fill="white"/><circle cx={mx + r * .43} cy={my - r * .08} r={r * .92} fill="black"/></mask>
        <filter id="es-grain" x="0" y="0" width="100%" height="100%"><feTurbulence type="fractalNoise" baseFrequency=".78" numOctaves="2" stitchTiles="stitch"/><feColorMatrix type="saturate" values="0"/><feComponentTransfer><feFuncA type="table" tableValues="0 .11"/></feComponentTransfer><feBlend in="SourceGraphic" mode="soft-light"/></filter>
        <filter id="es-soft"><feGaussianBlur stdDeviation="15"/></filter>
      </defs>
      <rect width="390" height="470" fill="url(#es-sky)"/>
      <rect y="260" width="390" height="115" fill="url(#es-horizon)"/>
      {starPoints.map(([x, y], i) => <circle key={`${x}-${y}`} cx={x} cy={y} r={i % 5 === 0 ? 1.05 : .62} fill="#e3e8e8" opacity={.18 + (i % 4) * .1}/>) }
      {dawn && <path d="M190 334 L235 236 Q249 218 258 238 L290 334Z" fill="url(#es-horizon)" opacity=".18"/>}
      <circle cx={mx} cy={my} r={r * 2.15} fill="url(#es-moon-wash)" opacity={scene === 'welcome' ? .45 : .26}/>
      <circle cx={mx} cy={my} r={r} fill="url(#es-earth)" opacity={s.ash}/>
      <g opacity={s.ash * .24} fill="#384955">
        <path d={`M${mx-r*.49} ${my-r*.16}q${r*.2}-${r*.25} ${r*.43}-${r*.1}t${r*.04} ${r*.4}q-${r*.27} ${r*.1}-${r*.47}-.3Z`}/>
        <path d={`M${mx+r*.09} ${my+r*.08}q${r*.14}-${r*.24} ${r*.39}-.12t-${r*.08} ${r*.4}q-${r*.18} ${r*.16}-${r*.31}-.28Z`}/>
        <path d={`M${mx-r*.38} ${my+r*.35}q${r*.18}-${r*.18} ${r*.38}-.04t${r*.17} ${r*.28}q-${r*.25} ${r*.18}-${r*.55}-.24Z`}/>
      </g>
      <circle cx={mx} cy={my} r={r} fill="url(#es-crescent)" mask="url(#es-crescent-mask)"/>
      <path d={`M${mx-r*.85} ${my-r*.12} Q${mx-r*.79} ${my-r*.92} ${mx-r*.12} ${my-r*.98}`} fill="none" stroke="#f5f3ec" strokeWidth=".75" opacity=".28"/>

      {/* A low, lived-in horizon: walls and cypress emerge as dawn advances. */}
      {dawn && <g fill="#222b33" opacity=".84">
        <path d="M0 324h35v-23h18v23h30v-36h27v36h13v-19h30v19h28v-31h23v31h35v-26h31v26h25v-41h20v41h25v-16h30v16h25v46H0Z"/>
        <path d="M62 318q-14-42 0-70 15 28 2 70Z"/><path d="M63 281q-18-21-15-34 15 7 18 29 6-23 17-28 1 17-15 35Z"/>
        <path d="M322 313q-12-35 0-57 13 22 1 57Z"/><path d="M323 281q-14-17-12-29 12 6 15 24 5-20 14-23 1 14-13 29Z"/>
        {[18, 92, 149, 207, 286, 354].map((x, i) => <rect key={x} x={x} y={i % 2 ? 307 : 316} width="4" height="4" rx=".6" fill="#e2b780" opacity={i % 3 === 0 ? .8 : .45}/>) }
      </g>}
      {!dawn && <g fill="#17202a" opacity=".72"><path d="M0 326h44v-21h24v21h50v-16h30v16h47v-25h31v25h47v-18h32v18h85v48H0Z"/><path d="M278 318q-14-42 0-68 15 27 2 68Z"/></g>}
      <path d="M0 348 Q190 340 390 348 V470 H0Z" fill={final ? '#4a4a45' : '#171d23'}/>
      <path d="M0 351 Q190 343 390 351" fill="none" stroke="#9ba0a0" strokeOpacity=".2" strokeWidth="2"/>
      {/* The roof rug is the trace of someone who has been sitting here. */}
      <path d="M72 378 Q190 368 319 380 L343 454 Q195 444 47 455Z" fill="url(#es-rug)" stroke="#b29b7c" strokeOpacity=".42" strokeWidth="1.2"/>
      <path d="M82 386 Q195 378 308 387 L326 442 Q190 435 63 443Z" fill="none" stroke="#c2aa88" strokeOpacity=".22" strokeWidth="1"/>
      <path d="M91 394 Q194 386 300 395 M72 432 Q190 425 318 434" fill="none" stroke="#d2b994" strokeOpacity=".16" strokeWidth=".8"/>
      <path d="M110 411 Q190 403 277 412" fill="none" stroke="#d2b994" strokeOpacity=".14" strokeWidth=".7"/>
      {lantern && <g>
        <ellipse cx="319" cy="390" rx="49" ry="42" fill="url(#es-lantern)"/>
        <path d="M306 383h25l-3 19h-19Z" fill="#3a332c" stroke="#a58a67" strokeWidth="1"/>
        <path d="M309 386h19v11h-19Z" fill="#edb865" opacity=".83"/>
        <path d="M319 388q-6 6 0 8 6-3 0-8Z" fill="#fff0c3"/>
        <path d="M310 382q9-8 18 0 M314 381v-4h10v4" fill="none" stroke="#867253" strokeWidth="1.4"/>
      </g>}
      {final && <path d="M209 470 L236 351 L252 351 L288 470Z" fill="url(#es-ray)" opacity=".34"/>}
      {/* Stair door is only visible at the threshold, open toward ordinary life. */}
      {final && <g><path d="M14 349h44v-74H14Z" fill="#292c2c" stroke="#918d83" strokeOpacity=".55"/><path d="M18 345v-65l31 8v57Z" fill="#55534d" stroke="#c1ae90" strokeOpacity=".6"/><circle cx="43" cy="316" r="1.7" fill="#d1b17c"/></g>}
      {shutters && <g>
        <rect width="390" height="470" fill="#171719" opacity=".22"/>
        <path d="M0 0h112v470H0Z" fill="#493d33"/><path d="M278 0h112v470H278Z" fill="#493d33"/>
        {[22,42,62,82,102].map((x) => <path key={`l${x}`} d={`M${x} 0v470`} stroke="#201f20" strokeOpacity=".37" strokeWidth="3"/>)}
        {[300,320,340,360,380].map((x) => <path key={`r${x}`} d={`M${x} 0v470`} stroke="#201f20" strokeOpacity=".37" strokeWidth="3"/>)}
        <path d="M0 0h390v24H0Z M0 446h390v24H0Z" fill="#302a26"/>
        <path d="M110 0v470 M280 0v470" stroke="#8f775e" strokeOpacity=".28" strokeWidth="2"/>
        <rect x="112" width="166" height="470" fill="none" stroke="#b49a78" strokeOpacity=".17"/>
        <path d="M0 0h390v12H0Z" fill="#72604b" opacity=".35"/>
      </g>}
      <rect width="390" height="470" fill="transparent" filter="url(#es-grain)" opacity=".35" pointerEvents="none"/>
    </svg>
  );
}
