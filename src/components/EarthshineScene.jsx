import React, { useId } from 'react';

/**
 * Earthshine — the imagery of Theia's onboarding, 2026-11.
 * ---------------------------------------------------------------------------
 * One story, told as a single pre-dawn from a single place under a single moon.
 *
 * Three real facts carry it:
 *   1. A thin crescent receives the least sunlight and the most Earthshine —
 *      the moment the sky seems to give the least it is handed the most.
 *   2. Glare hides the faint. The ashen disc appears only as the eye adapts.
 *   3. Earthshine comes back blue-grey, carrying the colour of the world it
 *      touched. The Moon's maria keep their 17th-century names for weather and
 *      states of mind: Serenity, Tranquility, Crises, Clouds, Cold.
 *
 * The camera never moves. A low parapet (بام), a kilim, a folded blanket, a
 * cypress beyond the wall, a plaster stair-head on the left and the eastern
 * horizon. Each scene changes only the light — so the place is learned as the
 * user's own window, and light itself becomes the progress bar.
 *
 * Theia is never drawn. She is the seeing: the patience of the eye adjusting,
 * the darkness that lets faint things show. Her signature is the ashen light.
 *
 * Everything below is hand-authored SVG — no raster art, no libraries — so the
 * sky stays crisp at any density and can carry slow, honest motion.
 */

/**
 * One canvas, phone-shaped, so nothing is ever cropped: 390 × 844 is the frame
 * the whole onboarding is composed in. The moon sits high and to the left — the
 * side that is free of the Persian caption, which is always set on the right.
 */
const VIEW = { w: 390, h: 844 };
const MOON = { x: 116, y: 176, r: 27 };
const HORIZON = 452; // where the plain meets the sky
const RIDGE = 494; // top edge of the parapet
const FLOOR = 538; // the roof floor, where the kilim begins

/* ------------------------------------------------------------------ */
/* Geometry helpers                                                    */
/* ------------------------------------------------------------------ */

const polar = (cx, cy, r, deg) => {
  const t = ((deg - 90) * Math.PI) / 180;
  return [cx + r * Math.cos(t), cy + r * Math.sin(t)];
};

/**
 * The crescent is the lune between a disc and an identical disc shifted up by
 * `shift`: an outer mask keeps only the lit rim. Its widest part is at the
 * bottom, so the bright sliver holds the unlit disc in its arms — the "Held
 * Moon" (ماهِ در آغوش) that the whole story is told around.
 */
function Crescent({ id, cx, cy, r, shift, fill, edge = null, edgeWidth = 0.6, edgeOpacity = 0.25 }) {
  return (
    <>
      <mask id={id} maskUnits="userSpaceOnUse" x={cx - r - 2} y={cy - r - 2} width={r * 2 + 4} height={r * 2 + 4}>
        <circle cx={cx} cy={cy} r={r} fill="#fff" />
        <circle cx={cx} cy={cy - shift} r={r} fill="#000" />
      </mask>
      <g mask={`url(#${id})`}>
        <circle cx={cx} cy={cy} r={r} fill={fill} />
        {edge && (
          <circle cx={cx} cy={cy} r={r} fill="none" stroke={edge} strokeWidth={edgeWidth} opacity={edgeOpacity} />
        )}
      </g>
    </>
  );
}

const f = (n) => Number(n.toFixed(2));

/* ------------------------------------------------------------------ */
/* Deterministic sky                                                   */
/* ------------------------------------------------------------------ */

function mulberry32(seed) {
  let a = seed;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/**
 * One sky, generated once. Every scene shows the same stars in the same places,
 * only brighter or fainter — so dawn really is the same sky being read again.
 * The disc around the moon is left empty: a crescent is never put beside a star.
 */
const SKY = (() => {
  const rand = mulberry32(0x5e1e11a);
  const stars = [];
  let guard = 0;
  while (stars.length < 190 && guard < 6000) {
    guard += 1;
    const x = 6 + rand() * 378;
    const y = 8 + Math.pow(rand(), 0.9) * 430;
    if (Math.hypot(x - MOON.x, y - MOON.y) < 96) continue;
    const weight = Math.pow(rand(), 1.7); // 0 = brightest
    stars.push({
      x: f(x),
      y: f(y),
      r: f(0.32 + (1 - weight) * 0.92),
      o: f(0.14 + (1 - weight) * 0.66),
      weight,
    });
  }
  stars.sort((a, b) => b.weight - a.weight); // fainter last: drawn first
  return stars;
})();

/* ------------------------------------------------------------------ */
/* Gradient helpers                                                    */
/* ------------------------------------------------------------------ */

const stopsOf = (stops) => stops.map(([offset, color, opacity]) => (
  <stop key={`${offset}-${color}`} offset={offset} stopColor={color} stopOpacity={opacity ?? 1} />
));

function Linear({ id, stops, x1 = 0, y1 = 0, x2 = 0, y2 = 1 }) {
  return <linearGradient id={id} x1={x1} y1={y1} x2={x2} y2={y2}>{stopsOf(stops)}</linearGradient>;
}

function Radial({ id, stops, cx = '50%', cy = '50%', r = '50%' }) {
  return <radialGradient id={id} cx={cx} cy={cy} r={r}>{stopsOf(stops)}</radialGradient>;
}

/* ------------------------------------------------------------------ */
/* The sky                                                             */
/* ------------------------------------------------------------------ */

function Starfield({ tint = '#cdd9ec', count = 60, opacity = 0.8, avoid = null }) {
  const visible = avoid
    ? SKY.filter((star) => Math.hypot(star.x - avoid.x, star.y - avoid.y) > avoid.r)
    : SKY;
  const shown = visible.slice(0, count);
  return (
    <g fill={tint} opacity={opacity}>
      {shown.map((star, index) => (
        <circle key={index} cx={star.x} cy={star.y} r={star.r} opacity={star.o} />
      ))}
      {shown.slice(-2).map((star, index) => (
        <circle key={`halo-${index}`} cx={star.x} cy={star.y} r={star.r * 2.1} opacity={star.o * 0.16} />
      ))}
    </g>
  );
}

/** Deterministic craters: positions in disc-relative units, seeded once. */
const CRATERS = (() => {
  const rand = mulberry32(0x1a73);
  const list = [];
  let guard = 0;
  // Craters gather in the bright southern highlands and thin out in the seas.
  while (list.length < 74 && guard < 2600) {
    guard += 1;
    const a = rand() * Math.PI * 2;
    const d = Math.sqrt(rand()) * 0.92;
    const x = Math.cos(a) * d;
    const y = Math.sin(a) * d + 0.12;
    if (y < -0.35 && rand() < 0.72) continue; // the northern seas are smoother
    list.push({ x, y, s: 0.01 + Math.pow(rand(), 3.2) * 0.062, o: 0.2 + rand() * 0.34 });
  }
  list.push({ x: 0.06, y: 0.2, s: 0.036, o: 0.34 }); // Copernicus
  list.push({ x: -0.18, y: 0.42, s: 0.022, o: 0.36 }); // Tycho
  list.push({ x: 0.62, y: -0.52, s: 0.026, o: 0.4 }); // Aristarchus
  return list;
})();

/**
 * A mare drawn as one ragged coastline rather than stacked ellipses: the radius
 * breathes with three harmonics, and the closed path is smoothed through the
 * midpoints so the edge stays confident and irregular like the real seas.
 */
function blobPath(seed, x, y, rx, ry, rot) {
  const rand = mulberry32(seed);
  const ph = [rand() * 6.283, rand() * 6.283, rand() * 6.283];
  const points = 30;
  const cos = Math.cos(rot);
  const sin = Math.sin(rot);
  const raw = [];
  for (let i = 0; i < points; i += 1) {
    const t = (i / points) * Math.PI * 2;
    const k = 1
      + 0.17 * Math.sin(t * 3 + ph[0])
      + 0.09 * Math.sin(t * 5 + ph[1])
      + 0.05 * Math.sin(t * 8 + ph[2]);
    const px = Math.cos(t) * rx * k;
    const py = Math.sin(t) * ry * k;
    raw.push([x + px * cos - py * sin, y + px * sin + py * cos]);
  }
  // Smooth through midpoints: quadratic curves whose control points are the raw
  // vertices, so the coastline reads as a shore instead of a polygon.
  const mid = (a, b) => [f((a[0] + b[0]) / 2), f((a[1] + b[1]) / 2)];
  const start = mid(raw[points - 1], raw[0]);
  let d = `M${start[0]} ${start[1]}`;
  for (let i = 0; i < points; i += 1) {
    const ctrl = raw[i];
    const next = mid(raw[i], raw[(i + 1) % points]);
    d += `Q${f(ctrl[0])} ${f(ctrl[1])} ${next[0]} ${next[1]}`;
  }
  return `${d}Z`;
}

/* Unit-disc coordinates: x/y offsets, rx/ry radii, rotation, opacity. */
/**
 * The five seas the story names — دریای آرامش، آسودگی، بحران‌ها، ابرها، سرما —
 * plus the two that frame the near limb. Deliberately few: a real face of the
 * moon, not a splatter. Sizes are fractions of the disc radius and are dressed
 * with a soft edge so they read as lava plains, not cut-out blobs.
 */
const MARIA = [
  { seed: 11, x: 0.16, y: -0.34, rx: 0.29, ry: 0.23, rot: -0.3, o: 0.74 }, // Serenity آسودگی
  { seed: 12, x: 0.24, y: 0.06, rx: 0.32, ry: 0.26, rot: 0.14, o: 0.8 }, // Tranquility آرامش
  { seed: 13, x: 0.58, y: -0.26, rx: 0.13, ry: 0.12, rot: 0.4, o: 0.6 }, // Crises بحران‌ها
  { seed: 17, x: -0.06, y: 0.44, rx: 0.19, ry: 0.12, rot: 0.2, o: 0.52 }, // Clouds ابرها
  { seed: 18, x: -0.02, y: -0.66, rx: 0.33, ry: 0.1, rot: 0.05, o: 0.5 }, // Cold سرما
  { seed: 15, x: -0.44, y: -0.28, rx: 0.2, ry: 0.18, rot: 0.5, o: 0.6 }, // Imbrium
  { seed: 16, x: -0.5, y: 0.16, rx: 0.22, ry: 0.28, rot: -0.1, o: 0.62 }, // Procellarum
].map((mare) => ({ ...mare, d: blobPath(mare.seed, mare.x, mare.y, mare.rx, mare.ry, mare.rot) }));

/* Minor seas and lakes scattered between the majors, from a fixed seed. */
const MINOR_SEAS = (() => {
  const rand = mulberry32(0x4c1a);
  return Array.from({ length: 16 }, (_, index) => {
    const a = rand() * Math.PI * 2;
    const d = 0.18 + Math.sqrt(rand()) * 0.62;
    return {
      seed: 100 + index,
      x: f(Math.cos(a) * d),
      y: f(Math.sin(a) * d),
      rx: f(0.03 + rand() * 0.09),
      ry: f(0.026 + rand() * 0.07),
      rot: rand() * 3,
      o: f(0.3 + rand() * 0.4),
    };
  });
})().map((mare) => ({ ...mare, d: blobPath(mare.seed, mare.x, mare.y, mare.rx, mare.ry, mare.rot) }));

/* Bright ray material around Tycho, and the pale highland south. */
const RAYS = [
  [-0.18, 0.42, -0.62, 0.9, 0.05],
  [-0.18, 0.42, 0.26, 0.86, 0.04],
  [-0.18, 0.42, -0.86, 0.34, 0.035],
  [-0.18, 0.42, 0.7, 0.62, 0.03],
  [-0.18, 0.42, -0.5, 0.96, 0.03],
];

/* Dust: soft, very faint patches that keep the highlands from reading as flat. */
const MOTTLE = (() => {
  const rand = mulberry32(0x9d21);
  return Array.from({ length: 52 }, () => {
    const a = rand() * Math.PI * 2;
    const d = Math.sqrt(rand()) * 0.86;
    return {
      x: f(Math.cos(a) * d),
      y: f(Math.sin(a) * d),
      rx: f(0.018 + rand() * 0.055),
      ry: f(0.014 + rand() * 0.045),
      o: f(0.03 + rand() * 0.08),
      light: rand() > 0.66,
    };
  });
})();

/**
 * The Held Moon. `ash` is the legibility of the ashen disc — the story's real
 * progress bar — never a switch. Maria, craters and limb shading are always in
 * the geometry; only their visibility changes as the eye adapts.
 */
function HeldMoon({
  pid,
  cx = MOON.x,
  cy = MOON.y,
  r = MOON.r,
  ash = 0.1,
  crescent = 0.115,
  tilt = -12,
  halo = 0.36,
  glow = 0.14,
  ashTone = '#8fa3c0',
  crescentTone = ['#fbfdff', '#dbe6f5', '#b3c5dc'],
  maria = 1,
}) {
  const shift = Math.max(0.8, r * crescent);
  const body = `m-${pid}-body`;
  const limb = `m-${pid}-limb`;
  const hilands = `m-${pid}-hilands`;
  const cresSoft = `m-${pid}-cressoft`;
  const mareBlur = `m-${pid}-mareblur`;
  const clip = `m-${pid}-clip`;
  const cres = `m-${pid}-cres`;
  const cresGlow = `m-${pid}-cresglow`;
  const bloom = `m-${pid}-bloom`;
  const [bx, by] = polar(cx, cy, r * 0.82, 176);

  return (
    <g className="es-moon">
      <defs>
        {/* the regolith is retroreflective: the disc is brightest towards its
            centre and darkens at the limb, exactly as earthshine photographs */}
        <radialGradient id={body} cx="46%" cy="55%" r="62%">
          <stop offset="0" stopColor={ashTone} stopOpacity="0.62" />
          <stop offset="0.55" stopColor={ashTone} stopOpacity="0.44" />
          <stop offset="0.86" stopColor={ashTone} stopOpacity="0.22" />
          <stop offset="1" stopColor={ashTone} stopOpacity="0.05" />
        </radialGradient>
        <radialGradient id={hilands} cx="42%" cy="40%" r="66%">
          <stop offset="0" stopColor="#e6eefb" stopOpacity="0.36" />
          <stop offset="0.6" stopColor="#cfe0f5" stopOpacity="0.14" />
          <stop offset="1" stopColor="#a8bdd8" stopOpacity="0" />
        </radialGradient>
        <linearGradient id={limb} x1={cx - r} y1={cy} x2={cx + r} y2={cy} gradientUnits="userSpaceOnUse">
          <stop offset="0" stopColor="#e8f1ff" stopOpacity="0.34" />
          <stop offset="0.5" stopColor="#e8f1ff" stopOpacity="0" />
        </linearGradient>
        <linearGradient id={cres} x1="0" y1="1" x2="0.35" y2="0">
          <stop offset="0" stopColor={crescentTone[2]} />
          <stop offset="0.28" stopColor={crescentTone[1]} />
          <stop offset="0.62" stopColor={crescentTone[0]} />
          <stop offset="1" stopColor={crescentTone[1]} />
        </linearGradient>
        <radialGradient id={cresGlow} cx="50%" cy="50%" r="50%">
          <stop offset="0" stopColor="#dce9fb" stopOpacity="0.55" />
          <stop offset="0.5" stopColor="#b9cde8" stopOpacity="0.16" />
          <stop offset="1" stopColor="#8ea8c9" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={bloom} cx="50%" cy="50%" r="50%">
          <stop offset="0" stopColor="#a8bcd6" stopOpacity="0.12" />
          <stop offset="0.45" stopColor="#7f98ba" stopOpacity="0.045" />
          <stop offset="1" stopColor="#6c86a8" stopOpacity="0" />
        </radialGradient>
        <filter id={cresSoft} x="-30%" y="-30%" width="160%" height="160%">
          <feGaussianBlur stdDeviation={Math.max(0.24, r * 0.011)} />
        </filter>
        <filter id={mareBlur} x="-70%" y="-70%" width="240%" height="240%">
          <feGaussianBlur stdDeviation="0.042" />
        </filter>
        <filter id={`${mareBlur}-fine`} x="-40%" y="-40%" width="180%" height="180%">
          <feGaussianBlur stdDeviation="0.008" />
        </filter>
        <clipPath id={clip}><circle cx={cx} cy={cy} r={r} /></clipPath>
      </defs>

      {/* the sky is slightly milky around the moon, the way air scatters */}
      <circle cx={cx} cy={cy} r={r * 2.4} fill={`url(#${bloom})`} opacity={halo * 0.85} />

      <g className="es-adapt">
        <circle cx={cx} cy={cy} r={r} fill={`url(#${body})`} opacity={ash} />
        <g clipPath={`url(#${clip})`} opacity={ash}>
          {/* the named seas: Serenity, Tranquility, Crises, Fecundity,
              Imbrium, Procellarum, Clouds, Cold — terrain, never decoration */}
          <g transform={`translate(${cx} ${cy}) scale(${r})`}>
            {MARIA.map((mare) => (
              <path key={mare.seed} d={mare.d} fill="#0d1724" opacity={mare.o * maria * 0.92} filter={`url(#${mareBlur})`} />
            ))}
            {MINOR_SEAS.map((mare) => (
              <path key={mare.seed} d={mare.d} fill="#101a27" opacity={mare.o * maria * 0.6} filter={`url(#${mareBlur})`} />
            ))}
            {/* Tycho's ejecta: a soft bright field, never drawn as spokes */}
            {/* Tycho, on the bright southern highlands: a halo of ejecta with a
                single small point at its centre, never drawn as spokes */}
            <ellipse cx="-0.1" cy="0.52" rx="0.34" ry="0.3" fill="#eef4ff" opacity={0.16 * maria} filter={`url(#${mareBlur})`} />
            <ellipse cx="-0.02" cy="0.78" rx="0.42" ry="0.16" fill="#eef4ff" opacity={0.12 * maria} filter={`url(#${mareBlur})`} />
            <ellipse cx="-0.14" cy="0.5" rx="0.035" ry="0.032" fill="#f2f7ff" opacity={0.2 * maria} filter={`url(#${mareBlur}-fine)`} />
            {MOTTLE.map((dot, index) => (
              <ellipse
                key={index}
                cx={dot.x} cy={dot.y} rx={dot.rx} ry={dot.ry}
                fill={dot.light ? '#e9f1fd' : '#0e1622'}
                opacity={dot.o * maria}
                filter={`url(#${mareBlur})`}
              />
            ))}

          </g>
          {/* craters: a dark pit and its lit eastern wall */}
          <g opacity={0.4 * maria}>
            {CRATERS.map((c, index) => {
              const x = cx + c.x * r;
              const y = cy + c.y * r;
              const s = c.s * r;
              return (
                <g key={index}>
                  <ellipse cx={x} cy={y} rx={s} ry={s * 0.92} fill="#0f1722" opacity={c.o * 0.5} />
                  <path d={`M${x + s * 0.1} ${y - s * 0.9} a ${s} ${s * 0.92} 0 0 1 ${s * 0.72} ${s * 0.62}`} fill="none" stroke="#e6eefb" strokeWidth={Math.max(0.18, s * 0.3)} opacity={c.o * 0.34} />
                </g>
              );
            })}
          </g>
          {/* the limb falls away into the sky instead of ending in a drawn line */}
          <circle cx={cx} cy={cy} r={r} fill={`url(#${hilands})`} opacity={0.5 * ash} />
          <circle cx={cx} cy={cy} r={r} fill={`url(#${limb})`} opacity={0.85} />
        </g>
      </g>

      {/* the bright sliver: loud, but only an edge */}
      <g transform={`rotate(${tilt} ${cx} ${cy})`}>
        <circle cx={bx} cy={by} r={r * 1.7} fill={`url(#${cresGlow})`} opacity={halo} />
        <g className="es-crescent" filter={`url(#${cresSoft})`}>
          <Crescent
            id={`${pid}-cres`}
            cx={cx} cy={cy} r={r} shift={shift}
            fill={`url(#${cres})`}
            edge={crescentTone[0]} edgeWidth={Math.max(0.3, r * 0.045)} edgeOpacity={glow * 0.55}
          />
        </g>
      </g>
    </g>
  );
}

/* ------------------------------------------------------------------ */
/* The place                                                           */
/* ------------------------------------------------------------------ */

/**
 * The plain. Four distances, each one a little darker than the sky behind it:
 * far hills, the town on the valley floor, a near ridge, and the low garden wall
 * behind the cypresses. Flat roofs and a wall — a place where people live,
 * without ever becoming a skyline.
 */
const TOWN = (() => {
  const rand = mulberry32(0x7a11);
  const houses = [];
  const domes = [];
  const walls = [];
  // Four clusters of adjoining flat-roofed houses — a village seen from a roof,
  // not a row of isolated boxes on a line.
  const clusters = [[-14, 76], [70, 86], [168, 84], [258, 74], [336, 70]];
  for (const [from, span] of clusters) {
    let x = from;
    while (x < from + span) {
      const w = 9 + rand() * 16;
      const h = 3 + Math.pow(rand(), 1.5) * 6;
      const y = HORIZON - 3.4;
      houses.push({ x: f(x), w: f(w), h: f(h), y });
      // a parapet notch on some roofs, a dome on others
      if (rand() > 0.72) {
        domes.push({ x: f(x + w * 0.5), y: f(y - h), r: f(Math.min(w * 0.42, 5.2)) });
      }
      x += w - rand() * 1.5; // houses lean on each other
    }
    // a compound wall running out from the cluster
    const wallStart = from + span - 6;
    walls.push({ x: f(wallStart), w: f(14 + rand() * 26), h: f(2.2 + rand() * 1.4), y: HORIZON + 0.6 });
  }
  return { houses, domes, walls };
})();

function Plain({ pid, land = 1, lights = [], lightColor = '#e8b57d', glint = 0.4, town = 1 }) {
  const far = `plain-far-${pid}`;
  const mid = `plain-mid-${pid}`;
  const hazeId = `plain-haze-${pid}`;
  const H = HORIZON;
  // one continuous crest: the nearest edge of the village, everything on it
  const crest = `M0 ${H - 3}C 34 ${H - 10} 68 ${H - 5} 102 ${H - 8}C 140 ${H - 11.4} 176 ${H - 4} 212 ${H - 7}C 252 ${H - 10.4} 292 ${H - 4} 326 ${H - 7.4}C 354 ${H - 10} 374 ${H - 6} 390 ${H - 8}`;
  return (
    <g>
      <defs>
        <linearGradient id={far} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#22304a" stopOpacity="0.34" />
          <stop offset="0.46" stopColor="#16202f" stopOpacity="0.5" />
          <stop offset="1" stopColor="#0b1220" stopOpacity="0.62" />
        </linearGradient>
        <linearGradient id={mid} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#0a1019" stopOpacity="1" />
          <stop offset="1" stopColor="#04060b" stopOpacity="1" />
        </linearGradient>
        <linearGradient id={hazeId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#d9c3c0" stopOpacity="0" />
          <stop offset="0.42" stopColor="#d9c3c0" stopOpacity="0.12" />
          <stop offset="0.68" stopColor="#8c8ea0" stopOpacity="0.05" />
          <stop offset="1" stopColor="#3c4457" stopOpacity="0" />
        </linearGradient>
      </defs>

      {/* far hills, only just separable from the sky */}
      <path
        d={`M0 ${H - 16}C 30 ${H - 23} 62 ${H - 19} 94 ${H - 22}C 132 ${H - 26} 170 ${H - 19} 206 ${H - 21}C 250 ${H - 24} 290 ${H - 18} 328 ${H - 22}C 356 ${H - 25} 378 ${H - 21} 390 ${H - 23}L390 ${H + 10}L0 ${H + 10}Z`}
        fill={`url(#${far})`}
        opacity={land}
      />

      {/* the ground itself: one solid landmass, dark, with the village standing on it */}
      <g opacity={land * town}>
        <path d={`${crest}L390 ${VIEW.h}L0 ${VIEW.h}Z`} fill={`url(#${mid})`} />
        {/* the village, all of it, on that crest */}
        <g fill="#03050a">
          {TOWN.walls.map((wall, index) => (
            <rect key={`w-${index}`} x={wall.x} y={wall.y} width={wall.w} height={wall.h + 3} />
          ))}
          {TOWN.houses.map((house, index) => (
            <rect key={`h-${index}`} x={house.x} y={house.y} width={house.w} height={house.h + 3} />
          ))}
          {TOWN.domes.map((dome, index) => (
            <path key={`d-${index}`} d={`M${dome.x - dome.r} ${dome.y}A${dome.r} ${dome.r * 0.92} 0 0 1 ${dome.x + dome.r} ${dome.y}Z`} />
          ))}
          {/* a windcatcher and the garden wall */}
          <path d={`M178 ${H - 7} L178 ${H - 16} L182.4 ${H - 16} L182.4 ${H - 7} Z M182.4 ${H - 15} L187 ${H - 15} L187 ${H - 7} L182.4 ${H - 7} Z`} />
          <path d={`M64 ${H - 5} L96 ${H - 5.6} L96 ${H - 2} L64 ${H - 1.6} Z`} opacity="0.9" />
          {/* a lone cypress far off on the plain, thin as a hair */}
          <path d={`M150 ${H + 2} C 149.2 ${H - 5} 149.6 ${H - 12} 150.4 ${H - 18} C 151.4 ${H - 12} 151.8 ${H - 5} 151.6 ${H + 2} Z`} />
        </g>
      </g>

      {/* haze over the foot of the hills, so distance reads as air */}
      <rect x="0" y={H - 18} width={VIEW.w} height="30" fill={`url(#${hazeId})`} opacity={land} />

      {/* the horizon itself: one hairline of light, never a glowing bar */}
      <path
        d={`M0 ${H + 0.4}C 60 ${H - 0.6} 140 ${H + 0.8} 210 ${H - 0.2}C 280 ${H - 1.2} 340 ${H + 0.6} 390 ${H - 0.4}`}
        stroke="#d3e0ef" strokeWidth="0.45" fill="none" opacity={glint * 0.28}
      />

      {lights.map((light, index) => (
        <g key={index}>
          <circle cx={light.x} cy={light.y} r={light.r * 6} fill={lightColor} opacity={light.o * 0.07} />
          <circle cx={light.x} cy={light.y} r={light.r * 2.4} fill={lightColor} opacity={light.o * 0.13} />
          <circle cx={light.x} cy={light.y} r={light.r} fill={lightColor} opacity={light.o} />
        </g>
      ))}
    </g>
  );
}

/**
 * The parapet (جان‌پناه). Hand-plastered, so its top edge is a drawn line rather
 * than a rectangle, and only the part under the brightest sky catches any light.
 */
function Parapet({ pid, edge, wall, rim, rimOpacity = 0.5, faceGlow = 0.0 }) {
  const topId = `parapet-top-${pid}`;
  const faceId = `parapet-face-${pid}`;
  const topSurface = `parapet-surface-${pid}`;
  const R = RIDGE;
  // hand-plastered: the lip is a drawn line, not a rectangle
  const top = `M-4 ${R}C 42 ${R - 1.1} 78 ${R + 0.9} 122 ${R - 0.2}C 168 ${R - 1.3} 206 ${R + 0.7} 248 ${R - 0.4}C 292 ${R - 1.4} 336 ${R + 0.6} 394 ${R - 0.6}`;
  return (
    <g>
      <defs>
        <linearGradient id={topId} x1="0" y1="0" x2="1" y2="0">
          <stop offset="0" stopColor={rim} stopOpacity="0.16" />
          <stop offset="0.4" stopColor={rim} stopOpacity="0.52" />
          <stop offset="0.66" stopColor={rim} stopOpacity="1" />
          <stop offset="1" stopColor={rim} stopOpacity="0.34" />
        </linearGradient>
        <linearGradient id={faceId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor={wall} stopOpacity="1" />
          <stop offset="0.16" stopColor={edge} stopOpacity="0.78" />
          <stop offset="0.52" stopColor={wall} stopOpacity="1" />
          <stop offset="1" stopColor="#020407" stopOpacity="1" />
        </linearGradient>
        <linearGradient id={topSurface} x1="0" y1="0" x2="1" y2="0">
          <stop offset="0" stopColor={rim} stopOpacity="0.06" />
          <stop offset="0.38" stopColor={rim} stopOpacity="0.13" />
          <stop offset="0.68" stopColor={rim} stopOpacity="0.22" />
          <stop offset="1" stopColor={rim} stopOpacity="0.1" />
        </linearGradient>
      </defs>
      {/* the top of the parapet is a horizontal surface: it holds the most sky */}
      <path
        d={`M-4 ${R - 5}C 42 ${R - 6.1} 78 ${R - 4.1} 122 ${R - 5.2}C 168 ${R - 6.3} 206 ${R - 4.3} 248 ${R - 5.4}C 292 ${R - 6.4} 336 ${R - 4.4} 394 ${R - 5.6}L394 ${R + 0.6}C 336 ${R + 1.8} 292 ${R - 0.2} 248 ${R + 0.8}C 206 ${R + 1.8} 168 ${R - 0.2} 122 ${R + 0.8}C 78 ${R + 1.8} 42 ${R - 0.2} -4 ${R + 0.8}Z`}
        fill={`url(#${topSurface})`} opacity={0.5 + faceGlow * 0.34}
      />
      <path d={`${top}L394 ${FLOOR}L-4 ${FLOOR}Z`} fill={`url(#${faceId})`} />
      {faceGlow > 0 && <path d={`${top}L394 ${R + 26} L-4 ${R + 26}Z`} fill={rim} opacity={faceGlow * 0.1} />}
      <path d={top} fill="none" stroke={`url(#${topId})`} strokeWidth="1.5" strokeLinecap="round" opacity={rimOpacity} />
      <path d={top} fill="none" stroke={rim} strokeWidth="3" strokeLinecap="round" opacity={rimOpacity * 0.13} />
      <path
        d={`M-4 ${R + 2.6}C 42 ${R + 1.5} 78 ${R + 3.5} 122 ${R + 2.4}C 168 ${R + 1.3} 206 ${R + 3.3} 248 ${R + 2.2}C 292 ${R + 1.2} 336 ${R + 3.2} 394 ${R + 2}`}
        fill="none" stroke="#000" strokeWidth="0.7" opacity="0.55"
      />
      <path d={`M228 ${R + 14} C 244 ${R + 12.2} 258 ${R + 14.4} 274 ${R + 13}`} fill="none" stroke="#000" strokeWidth="0.8" opacity="0.24" />
      {/* trowel marks: broad, faint, and never parallel */}
      <g stroke="#8ea3bd" fill="none" opacity={0.05 + faceGlow * 0.06}>
        <path d={`M22 ${R + 16} C 84 ${R + 12} 150 ${R + 22} 220 ${R + 17}`} strokeWidth="1.6" />
        <path d={`M120 ${R + 30} C 190 ${R + 26} 260 ${R + 34} 330 ${R + 29}`} strokeWidth="1.2" />
        <path d={`M-4 ${R + 38} C 60 ${R + 34} 130 ${R + 42} 196 ${R + 37}`} strokeWidth="0.9" />
      </g>
      <path d={`M58 ${R + 30} C 74 ${R + 28.4} 90 ${R + 30.8} 104 ${R + 29.4}`} fill="none" stroke="#0c1018" strokeWidth="0.7" opacity="0.36" />
    </g>
  );
}

/**
 * A kilim lying on the roof: a shallow perspective, an inner border, woven
 * diamonds in rows, and one small off-centre motif that the weaver invented.
 */
function Kilim({ pid, field, border, motif, motifOpacity = 0.5, blanket = 0, blanketTone = '#4a5364' }) {
  /* The roof floor is plaster (کاه‌گل). A kilim is laid across it, its far edge
     parallel to the parapet, running off the bottom of the frame: we are standing
     on the rug's edge, looking east. Nothing here is a rise in the ground. */
  const floor = `floor-${pid}`;
  const rugGrad = `rug-grad-${pid}`;
  const weave = `rug-weave-${pid}`;
  const clip = `rug-clip-${pid}`;
  const shade = `floor-shade-${pid}`;

  const Y0 = FLOOR + 58;   // far edge of the rug (a hair of plaster shows above it)
  const Y1 = FLOOR + 66;   // the woven border band
  const Y2 = FLOOR + 130;  // the field, below the border
  const B = 902;           // bottom of the frame, past the phone's own edge

  const rug = `M-40 ${Y0} C 80 ${Y0 - 7} 310 ${Y0 - 7} 430 ${Y0} L430 ${B} L-40 ${B} Z`;
  const fieldPath = `M-30 ${Y1} C 80 ${Y1 - 7} 310 ${Y1 - 7} 420 ${Y1} L420 ${B} L-30 ${B} Z`;
  const guardPath = `M-24 ${Y2} C 80 ${Y2 - 6} 310 ${Y2 - 6} 412 ${Y2} L412 ${B} L-24 ${B} Z`;

  return (
    <g>
      <defs>
        <linearGradient id={floor} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#1a2130" />
          <stop offset="0.5" stopColor="#10151f" />
          <stop offset="1" stopColor="#06080e" />
        </linearGradient>
        <linearGradient id={rugGrad} x1="0.2" y1="0" x2="0.8" y2="1">
          <stop offset="0" stopColor={field[0]} />
          <stop offset="0.34" stopColor={field[1]} />
          <stop offset="1" stopColor={field[0]} />
        </linearGradient>
        <linearGradient id={shade} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#000000" stopOpacity="0.5" />
          <stop offset="0.3" stopColor="#000000" stopOpacity="0.16" />
          <stop offset="1" stopColor="#000000" stopOpacity="0.44" />
        </linearGradient>
        <clipPath id={clip}><path d={rug} /></clipPath>
        <pattern id={weave} width="2.6" height="2.6" patternUnits="userSpaceOnUse">
          <path d="M0 2.6 L2.6 0" stroke={motif} strokeWidth="0.34" opacity="0.3" />
        </pattern>
      </defs>

      {/* the plaster floor between the parapet and the rug */}
      <rect x="-10" y={FLOOR + 2} width={VIEW.w + 20} height={VIEW.h - FLOOR - 2} fill={`url(#${floor})`} />
      <rect x="-10" y={FLOOR + 2} width={VIEW.w + 20} height="9" fill={border} opacity="0.1" />
      <path d={`M-10 ${FLOOR + 9} C 90 ${FLOOR + 5} 300 ${FLOOR + 13} ${VIEW.w + 10} ${FLOOR + 7}`}
            fill="none" stroke="#94a8c4" strokeWidth="0.5" opacity="0.1" />

      <path d={rug} fill={`url(#${rugGrad})`} />

      <g clipPath={`url(#${clip})`}>
        <rect x="-40" y={Y0 - 10} width="480" height={B - Y0 + 20} fill={`url(#${weave})`} opacity="0.6" />

        {/* two woven rails with a guard stripe between them */}
        <path d={fieldPath} fill="none" stroke={border} strokeWidth="12" opacity="0.44" />
        <path d={guardPath} fill="none" stroke={motif} strokeWidth="0.9" opacity={motifOpacity * 0.5} />
        <path d={guardPath} fill="none" stroke={border} strokeWidth="4" opacity="0.22" />

        {/* the field: hooked diamonds, wider as they come nearer the eye */}
        {[0.1, 0.36, 0.62, 0.88, 1.14].map((t, row) => {
          const y = Y2 + 14 + t * 62;
          const half = 306 - t * 10;
          const count = 7 + row;
          return (
            <g key={t} opacity={motifOpacity * (0.28 + t * 0.3)}>
              {Array.from({ length: count }, (_, i) => {
                const x = 240 - half + ((2 * half) * (i + 0.5)) / count + (row % 2 ? 18 : -12);
                const sz = 2.6 + t * 4.2;
                return (
                  <path
                    key={i}
                    d={`M${f(x)} ${f(y - sz * 1.15)}L${f(x + sz * 0.7)} ${f(y)}L${f(x)} ${f(y + sz * 1.15)}L${f(x - sz * 0.7)} ${f(y)}Z`}
                    fill="none" stroke={motif} strokeWidth="0.6"
                  />
                );
              })}
            </g>
          );
        })}

        {/* the one motif the weaver invented, off-centre */}
        <path d={`M300 ${Y2 + 96} l13 -15 l13 15 l-13 15 Z M313 ${Y2 + 81} l0 -12 M326 ${Y2 + 96} l14 0`} fill="none" stroke={motif} strokeWidth="0.95" opacity={motifOpacity * 0.9} />
        <path d={`M150 ${Y2 + 172} l11 -13 l11 13 l-11 13 Z M161 ${Y2 + 159} l0 -11`} fill="none" stroke={motif} strokeWidth="0.8" opacity={motifOpacity * 0.6} />

        <rect x="-40" y={Y0 - 10} width="480" height={B - Y0 + 20} fill={`url(#${shade})`} />
      </g>

      {/* the fringe: warp threads standing out along the far edge — this is what
          tells the eye it is a rug and not a rise in the ground */}
      <g opacity={motifOpacity * 0.5}>
        {Array.from({ length: 62 }, (_, i) => {
          const x = -24 + i * 7.2;
          const t = (x - 195) / 235;
          const y = Y0 + 1 + Math.abs(t) * 6;
          return <path key={i} d={`M${f(x)} ${f(y)} L${f(x + 0.4)} ${f(y - 9)}`} stroke={motif} strokeWidth="0.55" />;
        })}
      </g>
      <path d={rug} fill="none" stroke={border} strokeWidth="1.8" opacity="0.45" />

      {blanket > 0 && (
        <g opacity={blanket}>
          {/* a wool blanket, folded and left where somebody was sitting */}
          <path d={`M128 ${Y2 + 118} C 172 ${Y2 + 104} 236 ${Y2 + 114} 282 ${Y2 + 130} C 318 ${Y2 + 142} 334 ${Y2 + 168} 324 ${Y2 + 190} C 312 ${Y2 + 212} 264 ${Y2 + 214} 218 ${Y2 + 206} C 164 ${Y2 + 196} 124 ${Y2 + 174} 108 ${Y2 + 152} C 94 ${Y2 + 132} 106 ${Y2 + 124} 128 ${Y2 + 118} Z`} fill={blanketTone} />
          <path d={`M142 ${Y2 + 140} C 192 ${Y2 + 160} 246 ${Y2 + 176} 302 ${Y2 + 188}`} fill="none" stroke="#0d1119" strokeWidth="2" opacity="0.44" />
          <path d={`M132 ${Y2 + 168} C 182 ${Y2 + 188} 236 ${Y2 + 202} 292 ${Y2 + 210}`} fill="none" stroke="#0d1119" strokeWidth="1.5" opacity="0.26" />
          <path d={`M156 ${Y2 + 122} C 202 ${Y2 + 136} 252 ${Y2 + 150} 298 ${Y2 + 160}`} fill="none" stroke="#c9d3e2" strokeWidth="0.9" opacity="0.13" />
        </g>
      )}
    </g>
  );
}

/**
 * The stair-head: the little plastered hut over the stairs to the street. Its
 * doorway is dark through most of the night; at first light the door stands open.
 */
function StairHead({ pid, tone, doorTone, open = false, warm = 0 }) {
  /* The stair-head (سرپله): the little plastered hut over the stairs down to the
     street. It stands on the roof's north edge, so we see its long lit face and
     a foreshortened side — never a black box. */
  const faceId = `hut-face-${pid}`;
  const sideId = `hut-side-${pid}`;
  const openId = `hut-open-${pid}`;
  const top = RIDGE - 30;
  // the wall runs off the left edge of the frame
  const front = `M-10 ${top}C 16 ${top - 1.1} 54 ${top + 0.4} 78 ${top - 0.3}C 86 ${top - 0.6} 92 ${top - 0.2} 96 ${top}L96 ${FLOOR}L-10 ${FLOOR}Z`;
  const side = `M96 ${top} L124 ${top + 7} L124 ${FLOOR} L96 ${FLOOR} Z`;
  const doorX = 26;
  const doorW = 32;
  const doorTop = RIDGE + 66;
  const door = `M${doorX} ${FLOOR} L${doorX} ${doorTop + 2} Q${doorX} ${doorTop} ${doorX + 2} ${doorTop} L${doorX + doorW - 2} ${doorTop} Q${doorX + doorW} ${doorTop} ${doorX + doorW} ${doorTop + 2} L${doorX + doorW} ${FLOOR} Z`;
  return (
    <g>
      <defs>
        <linearGradient id={faceId} x1="0" y1="0" x2="0.9" y2="0.2">
          <stop offset="0" stopColor={tone} />
          <stop offset="0.62" stopColor={tone} />
          <stop offset="1" stopColor="#020306" />
        </linearGradient>
        <linearGradient id={sideId} x1="0" y1="0" x2="1" y2="0">
          <stop offset="0" stopColor="#020306" />
          <stop offset="0.7" stopColor="#070a10" />
          <stop offset="1" stopColor="#0b0f16" />
        </linearGradient>
        <linearGradient id={openId} x1="0" y1="1" x2="0" y2="0">
          <stop offset="0" stopColor="#d79059" stopOpacity="0.62" />
          <stop offset="0.42" stopColor="#7a4a2c" stopOpacity="0.26" />
          <stop offset="1" stopColor="#0d0805" stopOpacity="0.95" />
        </linearGradient>
      </defs>

      <path d={side} fill={`url(#${sideId})`} />
      <path d={front} fill={`url(#${faceId})`} />
      {/* the roof edge of the hut, catching the same pale line as the parapet */}
      <path d={`M-10 ${top}C 16 ${top - 1.1} 54 ${top + 0.4} 78 ${top - 0.3}C 86 ${top - 0.6} 92 ${top - 0.2} 96 ${top}L124 ${top + 7}`}
            fill="none" stroke="#b9cbe2" strokeWidth="1.1" opacity="0.2" strokeLinecap="round" />
      <path d={`M96 ${top} L96 ${FLOOR}`} stroke="#000" strokeWidth="1.1" opacity="0.4" />
      <path d={`M124 ${top + 7} L124 ${FLOOR}`} stroke="#8ea3bd" strokeWidth="0.5" opacity="0.1" />

      {/* plaster detail: one course of bricks near the base, one old stain */}
      <path d={`M-10 ${FLOOR - 12} L96 ${FLOOR - 12}`} stroke="#8ea3bd" strokeWidth="0.5" opacity="0.06" />
      <path d={`M12 ${top + 22} C 26 ${top + 16} 36 ${top + 26} 50 ${top + 20}`} fill="none" stroke="#8ea3bd" strokeWidth="0.6" opacity="0.09" />
      <path d={`M64 ${top + 40} C 72 ${top + 35} 80 ${top + 42} 88 ${top + 36}`} fill="none" stroke="#000" strokeWidth="0.7" opacity="0.22" />

      <path d={door} fill={doorTone} />
      <path d={`M${doorX} ${FLOOR} L${doorX} ${doorTop + 2} Q${doorX} ${doorTop} ${doorX + 2} ${doorTop} L${doorX + doorW - 2} ${doorTop} Q${doorX + doorW} ${doorTop} ${doorX + doorW} ${doorTop + 2} L${doorX + doorW} ${FLOOR}`}
            fill="none" stroke="#5b4a38" strokeWidth="1" opacity="0.5" />

      {open ? (
        <g>
          <path d={door} fill={`url(#${openId})`} />
          {/* two steps going down, lit from below */}
          <path d={`M${doorX + 3} ${FLOOR - 6} L${doorX + doorW - 3} ${FLOOR - 6}`} stroke="#f0b985" strokeWidth="0.9" opacity="0.4" />
          <path d={`M${doorX + 6} ${FLOOR - 22} L${doorX + doorW - 6} ${FLOOR - 22}`} stroke="#f0b985" strokeWidth="0.7" opacity="0.24" />
          <path d={`M${doorX + 9} ${FLOOR - 38} L${doorX + doorW - 9} ${FLOOR - 38}`} stroke="#f0b985" strokeWidth="0.6" opacity="0.13" />
          {/* a warm wash spilling out over the plaster and the roof edge */}
          {warm > 0 && (
            <g opacity={warm}>
              <path d={`M${doorX - 4} ${FLOOR} L${doorX + doorW + 4} ${FLOOR} L${doorX + doorW + 30} ${FLOOR + 30} L${doorX - 26} ${FLOOR + 30} Z`} fill="#e2a468" opacity="0.1" />
              <path d={`M${doorX} ${doorTop} L${doorX + doorW} ${doorTop} L${doorX + doorW + 16} ${doorTop - 20} L${doorX - 14} ${doorTop - 20} Z`} fill="#e2a468" opacity="0.05" />
            </g>
          )}
        </g>
      ) : (
        <path d={`M${doorX + doorW * 0.55} ${FLOOR} L${doorX + doorW * 0.55} ${doorTop + 2}`} stroke="#000" strokeWidth="0.9" opacity="0.4" />
      )}
    </g>
  );
}

/**
 * Cypress (سرو), the tree of the Persian garden: a dark flame that never quite
 * repeats itself. A grove of different heights behind the neighbour's low wall,
 * so the right edge has a place with depth rather than one decorative tree.
 */
function Cypress({ tone, glow = 0, pid = 'c' }) {
  const soft = `cypress-soft-${pid}`;
  const tree = (x, base, height, w, lean = 0) => {
    const top = base - height;
    const h = base - top;
    return (
      <path
        key={`${x}-${height}`}
        d={`M${x} ${top}
            C ${x + w * 0.42} ${top + h * 0.16} ${x + w * 0.92} ${top + h * 0.46} ${x + w * 0.62} ${base - h * 0.18}
            C ${x + w * 0.46} ${base - h * 0.04} ${x + w * 0.2} ${base} ${x} ${base}
            C ${x - w * 0.22} ${base} ${x - w * 0.48} ${base - h * 0.03} ${x - w * 0.64} ${base - h * 0.2}
            C ${x - w * 0.98} ${top + h * 0.44} ${x - w * 0.4} ${top + h * 0.16} ${x} ${top} Z`}
        fill={tone}
        transform={lean ? `rotate(${lean} ${x} ${base})` : undefined}
      />
    );
  };
  return (
    <g>
      <defs>
        <filter id={soft} x="-40%" y="-20%" width="180%" height="140%">
          <feGaussianBlur stdDeviation="9" />
        </filter>
      </defs>
      {glow > 0 && (
        <g opacity={glow * 0.4} filter={`url(#${soft})`}>
          <ellipse cx="330" cy={RIDGE - 10} rx="96" ry="46" fill="#e8b78b" opacity="0.055" />
        </g>
      )}
      {/* the neighbour's low wall, then the grove behind it */}
      <path d={`M262 ${RIDGE + 1} L394 ${RIDGE - 3} L394 ${RIDGE + 16} L262 ${RIDGE + 22} Z`} fill="#03050a" opacity="0.95" />
      <path d={`M262 ${RIDGE + 1} L394 ${RIDGE - 3}`} stroke="#93a8c2" strokeWidth="0.5" opacity="0.18" />
      {tree(284, RIDGE + 3, 74, 6.0, -0.6)}
      {tree(297, RIDGE + 3, 128, 7.2, 0.5)}
      {tree(310, RIDGE + 4, 186, 9.0, -0.3)}
      {tree(323, RIDGE + 4, 138, 7.8, 0.7)}
      {tree(336, RIDGE + 5, 224, 9.8, -0.2)}
      {tree(349, RIDGE + 5, 158, 8.2, 0.4)}
      {tree(362, RIDGE + 6, 196, 9.0, -0.4)}
      {tree(375, RIDGE + 6, 110, 6.8, 0.6)}
      {tree(386, RIDGE + 6, 146, 7.2, -0.3)}
    </g>
  );
}

/**
 * The moon's light is the only cool light in the story; the lantern's is the
 * only warm human light. It lights a patch of plaster and wool, never the sky.
 */
function Lantern({ level = 1, x = 264, y = RIDGE + 4 }) {
  if (level <= 0) return null;
  const ox = x - 0.5;
  const pid = `lantern-${Math.round(x)}`;
  const s = 2.05; // the lantern is small on the wall, but must read on a phone
  return (
    <g className="es-lantern" opacity={level} transform={`translate(${ox} ${y}) scale(${s}) translate(${-ox} ${-y})`}>
      <defs>
        <radialGradient id={`${pid}-pool`} cx="50%" cy="50%" r="50%">
          <stop offset="0" stopColor="#efa969" stopOpacity="0.36" />
          <stop offset="0.4" stopColor="#d4914f" stopOpacity="0.15" />
          <stop offset="1" stopColor="#b8773f" stopOpacity="0" />
        </radialGradient>
        <linearGradient id={`${pid}-brass`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#4a3722" />
          <stop offset="0.5" stopColor="#2a1e14" />
          <stop offset="1" stopColor="#150f09" />
        </linearGradient>
      </defs>

      {/* light on the wall behind, and the pool it throws on the parapet top */}
      <ellipse cx={ox} cy={y - 12} rx="34" ry="24" fill={`url(#${pid}-pool)`} />
      <ellipse cx={ox} cy={y + 1.2} rx="26" ry="4.2" fill={`url(#${pid}-pool)`} opacity="0.95" />
      <path d={`M${ox - 12} ${y - 1} Q${ox} ${y + 6} ${ox + 12} ${y - 1}`} fill="none" stroke="#c98a4d" strokeWidth="0.7" opacity="0.3" />

      {/* the brass body, the glass, and a real flame inside it */}
      <path d={`M${ox - 4.4} ${y - 1.4}L${ox - 4} ${y - 12}Q${ox} ${y - 13.8} ${ox + 4} ${y - 12}L${ox + 4.4} ${y - 1.4}Z`} fill={`url(#${pid}-brass)`} />
      <path d={`M${ox - 2.7} ${y - 3.2}L${ox - 2.4} ${y - 10.8}Q${ox} ${y - 11.8} ${ox + 2.4} ${y - 10.8}L${ox + 2.7} ${y - 3.2}Z`} fill="#eda95f" opacity="0.55" />
      <path d={`M${ox - 1.7} ${y - 4.4}L${ox - 1.5} ${y - 10}Q${ox} ${y - 10.7} ${ox + 1.5} ${y - 10}L${ox + 1.7} ${y - 4.4}Z`} fill="#ffdcab" opacity="0.75" />
      <path className="es-flame" d={`M${ox} ${y - 10.6}C ${ox - 2} ${y - 9} ${ox - 1.6} ${y - 6} ${ox} ${y - 4.8}C ${ox + 1.6} ${y - 6} ${ox + 2} ${y - 9} ${ox} ${y - 10.6}Z`} fill="#fff6e2" />
      <path d={`M${ox - 4.4} ${y - 12}L${ox + 4.4} ${y - 12}L${ox + 3.2} ${y - 14}L${ox - 3.2} ${y - 14}Z`} fill={`url(#${pid}-brass)`} />
      <path d={`M${ox - 4.6} ${y - 1.4}L${ox + 4.6} ${y - 1.4}L${ox + 4.2} ${y + 0.6}L${ox - 4.2} ${y + 0.6}Z`} fill={`url(#${pid}-brass)`} />
      <path d={`M${ox - 4.4} ${y - 12}L${ox - 4.4} ${y - 1.4}M${ox + 4.4} ${y - 12}L${ox + 4.4} ${y - 1.4}`} stroke="#2a1d13" strokeWidth="1.1" strokeLinecap="round" />
      <path d={`M${ox - 3} ${y - 15}Q${ox} ${y - 19} ${ox + 3} ${y - 15}`} fill="none" stroke="#2a1d13" strokeWidth="0.95" strokeLinecap="round" />
      <path d={`M${ox - 2.7} ${y - 10.8}Q${ox} ${y - 11.8} ${ox + 2.7} ${y - 10.8}`} fill="none" stroke="#ffe1b3" strokeWidth="0.5" opacity="0.75" />
    </g>
  );
}

/**
 * Air, not noise: one fractal-turbulence layer at very low opacity laid over the
 * whole frame, so the sky reads as photographed rather than plotted.
 */
function Atmosphere({ pid, opacity = 0.16 }) {
  const filterId = `grain-${pid}`;
  return (
    <g pointerEvents="none">
      <defs>
        <filter id={filterId} x="0" y="0" width="100%" height="100%">
          <feTurbulence type="fractalNoise" baseFrequency="0.82" numOctaves="2" seed="7" stitchTiles="stitch" />
          <feColorMatrix type="saturate" values="0" />
        </filter>
      </defs>
      <rect x="0" y="0" width={VIEW.w} height={VIEW.h} filter={`url(#${filterId})`} opacity={opacity} className="es-grain" />
    </g>
  );
}

function RoofWorld({ pid, world }) {
  const { sky, haze = null, stars = { count: 60, opacity: 0.75 }, moon = {}, plain = {}, roof = {}, lantern = 0, door = false, ray = 0, veil = null } = world;
  const skyId = `sky-${pid}`;
  const glowId = `glow-${pid}`;
  const dawnId = `dawn-${pid}`;
  const rayId = `ray-${pid}`;
  const sunId = `sun-${pid}`;
  const vigId = `vig-${pid}`;
  const zodId = `zod-${pid}`;
  return (
    <>
      <defs>
        <Linear id={skyId} stops={sky} />
        {haze && <Radial id={glowId} stops={haze.stops} cx="50%" cy="50%" r="50%" />}
        {world.falseDawn > 0 && (
          <Linear
            id={dawnId}
            x1="0" y1="0" x2="0" y2="1"
            stops={[[0, '#c7d5e8', 0], [0.5, '#ccd9ec', 0.42], [1, '#dedcd3', 0.72]]}
          />
        )}
        {ray > 0 && (
          <Linear
            id={rayId} x1="1" y1="0" x2="0" y2="1"
            stops={[[0, '#ffd9a2', 0.5], [0.42, '#f6bd7c', 0.22], [1, '#f0a860', 0]]}
          />
        )}
        <Radial
          id={sunId}
          stops={[[0, '#ffe6bd', 0.85], [0.4, '#f3b982', 0.34], [1, '#e0955f', 0]]}
        />
        <Radial id={vigId} stops={[[0.5, '#000000', 0], [1, '#000000', 0.55]]} cx="50%" cy="46%" r="72%" />
        <filter id={zodId} x="-60%" y="-20%" width="220%" height="140%">
          <feGaussianBlur stdDeviation="7" />
        </filter>
      </defs>

      <rect x="0" y="0" width={VIEW.w} height={VIEW.h} fill={`url(#${skyId})`} />
      <Starfield count={stars.count} opacity={stars.opacity} tint={stars.tint || '#cdd9ec'} />

      {haze && (
        <ellipse cx={haze.x ?? 200} cy={haze.y ?? HORIZON + 4} rx={haze.rx ?? 250} ry={haze.ry ?? 96} fill={`url(#${glowId})`} />
      )}

      {world.falseDawn > 0 && (
        <g opacity={world.falseDawn} filter={`url(#${zodId})`}>
          <path
            d={`M${(world.dawnX ?? 214) - 13} ${HORIZON + 3}
                C ${(world.dawnX ?? 214) - 9} ${HORIZON - 40} ${(world.dawnX ?? 214) - 4} ${HORIZON - 86} ${(world.dawnX ?? 214) + 2} ${HORIZON - 128}
                L${(world.dawnX ?? 214) + 15} ${HORIZON - 126}
                C ${(world.dawnX ?? 214) + 13} ${HORIZON - 84} ${(world.dawnX ?? 214) + 17} ${HORIZON - 38} ${(world.dawnX ?? 214) + 26} ${HORIZON + 3} Z`}
            fill={`url(#${dawnId})`}
            opacity="0.26"
          />
          <path
            d={`M${(world.dawnX ?? 214) - 40} ${HORIZON + 3} L${(world.dawnX ?? 214) + 56} ${HORIZON + 3} L${(world.dawnX ?? 214) + 22} ${HORIZON - 46} L${(world.dawnX ?? 214) + 4} ${HORIZON - 46} Z`}
            fill={`url(#${dawnId})`}
            opacity="0.16"
          />
        </g>
      )}

      <Plain
        pid={pid}
        land={plain.land ?? 1}
        town={plain.town ?? 1}
        lights={plain.lights ?? []}
        glint={plain.glint ?? 0.4}
      />

      {world.sunRim && (
        <g opacity={world.sunRim.o ?? 1}>
          {/* the first rim of sun: the top of the disc, and nothing but the top */}
          <g filter={`url(#${zodId})`}>
            <ellipse cx={world.sunRim.x} cy={HORIZON + 30} rx="26" ry="26" fill={`url(#${sunId})`} />
            <ellipse cx={world.sunRim.x} cy={HORIZON + 2} rx="120" ry="9" fill="#f6c48c" opacity="0.2" />
          </g>
          {/* the lit limb: the only part of the disc above the land */}
          {/* the disc itself: only its crown clears the land, and that crown is
              brighter at its centre than at its shoulders */}
          <path
            d={`M${world.sunRim.x - 22} ${HORIZON - 0.6}A22 22 0 0 1 ${world.sunRim.x + 22} ${HORIZON - 0.6}Z`}
            fill="#ffe7bc"
          />
          <path
            d={`M${world.sunRim.x - 15} ${HORIZON - 0.6}A15 15 0 0 1 ${world.sunRim.x + 15} ${HORIZON - 0.6}Z`}
            fill="#fffaf0" opacity="0.9"
          />
          <path
            d={`M${world.sunRim.x - 22} ${HORIZON - 0.6}A22 22 0 0 1 ${world.sunRim.x + 22} ${HORIZON - 0.6}`}
            fill="none" stroke="#ffc98a" strokeWidth="1.4" strokeLinecap="round" opacity="0.55"
          />
          {/* and the mirrored smear of that rim on the land below */}
          <ellipse cx={world.sunRim.x} cy={HORIZON + 3} rx="34" ry="4" fill="#ffd9a2" opacity="0.16" filter={`url(#${zodId})`} />
        </g>
      )}

      <Cypress tone={roof.tree} glow={world.treeGlow ?? 0} pid={pid} />

      {world.moon !== null && <HeldMoon pid={pid} {...moon} />}

      {veil}

      <StairHead pid={pid} tone={roof.hut} doorTone={roof.hutDoor} open={door} warm={world.doorWarm ?? 0} />
      <Parapet pid={pid} edge={roof.edge} wall={roof.wall} rim={roof.rim} rimOpacity={roof.rimOpacity} faceGlow={world.faceGlow ?? 0} />
      {ray > 0 && (
        <g opacity={ray} filter={`url(#${zodId})`}>
          {/* one low beam, arriving from the sunrise side of the frame and
              landing on the rug where somebody sat */}
          <path d={`M394 ${HORIZON + 6} L394 ${HORIZON + 22} L${120} ${FLOOR + 96} L${40} ${FLOOR + 96} Z`} fill={`url(#${rayId})`} opacity="0.85" />
          <path d={`M394 ${HORIZON + 9} L394 ${HORIZON + 15} L${168} ${FLOOR + 84} L${110} ${FLOOR + 84} Z`} fill="#ffd9a2" opacity="0.15" />
          {/* and its edge catching the parapet's top, which is what sells the angle */}
          <path d={`M${214} ${RIDGE + 0.6} L394 ${RIDGE - 1} L394 ${RIDGE + 1.6} L${206} ${RIDGE + 3} Z`} fill="#ffd9a2" opacity="0.4" />
        </g>
      )}
      <Kilim
        pid={pid}
        field={roof.rug}
        border={roof.rugBorder}
        motif={roof.motif}
        motifOpacity={roof.motifOpacity}
        blanket={roof.blanket ?? 0}
        blanketTone={roof.blanketTone}
      />
      <Lantern level={lantern} y={RIDGE - 1} x={world.lanternX ?? 268} />
      <rect x="0" y="0" width={VIEW.w} height={VIEW.h} fill={`url(#${vigId})`} opacity="0.55" />
      <Atmosphere pid={pid} opacity={0.16} />
    </>
  );
}

/**
 * The interior of scene 2: rough plaster, two weathered shutter leaves parted by
 * a vertical of sky. A window, not a vault — you choose how much passes through.
 */
function ShutterWorld({ pid, ash = 0.12 }) {
  const wall = `wall-${pid}`;
  const reveal = `reveal-${pid}`;
  const gapSky = `gapsky-${pid}`;
  const wood = `wood-${pid}`;
  const woodLit = `woodlit-${pid}`;
  const sill = `sill-${pid}`;
  const vig = 'vig';
  const crack = 'M52 120 C60 168 46 206 56 252 C64 292 52 330 62 372';
  const crack2 = 'M330 706 C324 740 334 768 326 796';
  return (
    <>
      <defs>
        <Linear id={wall} x1="0" y1="0" x2="0.35" y2="1" stops={[[0, '#232a35'], [0.45, '#161c26'], [1, '#0b0f16']]} />
        <Linear id={reveal} x1="1" y1="0" x2="0" y2="0" stops={[[0, '#2b241c'], [0.42, '#241d16'], [1, '#171209']]} />
        <Linear id={gapSky} x1="0" y1="0" x2="0" y2="1" stops={[[0, '#05070e'], [0.55, '#080d17'], [0.86, '#0d1523'], [1, '#121b2b']]} />
        <Linear id={wood} x1="0" y1="0" x2="1" y2="0.2" stops={[[0, '#2a2b2c'], [0.34, '#3a3833'], [0.62, '#242523'], [1, '#171817']]} />
        <Linear id={woodLit} x1="0" y1="0" x2="1" y2="0" stops={[[0, '#4a4a45'], [0.18, '#5c5a51'], [0.52, '#2d2e2c'], [1, '#1a1b1a']]} />
        <Linear id={sill} x1="0" y1="0" x2="0" y2="1" stops={[[0, '#3c2d20'], [0.35, '#2a2016'], [1, '#15100a']]} />
        <Radial id={`${vig}-${pid}`} stops={[[0.48, '#000000', 0], [1, '#000000', 0.6]]} cx="50%" cy="48%" r="70%" />
      </defs>

      <rect x="0" y="0" width={VIEW.w} height={VIEW.h} fill={`url(#${wall})`} />
      {/* plaster tooth: a few long, almost invisible unevennesses */}
      <g stroke="#000" fill="none" opacity="0.5">
        <path d={crack} strokeWidth="0.7" opacity="0.35" />
        <path d="M56 252 C66 262 60 274 68 284" strokeWidth="0.5" opacity="0.25" />
        <path d="M28 470 C 44 462 58 472 72 464" strokeWidth="0.5" opacity="0.16" />
        <path d={crack2} strokeWidth="0.6" opacity="0.22" />
        <path d="M40 330 C56 326 70 332 84 328" strokeWidth="0.5" opacity="0.18" />
      </g>
      <g fill="#c9b79b" opacity="0.024">
        <ellipse cx="26" cy="760" rx="60" ry="48" />
        <ellipse cx="356" cy="252" rx="44" ry="72" />
      </g>
      <g stroke="#000" fill="none" opacity="0.26">
        <path d="M18 232 C 36 224 46 234 62 228" strokeWidth="0.5" />
        <path d="M320 740 C 336 730 348 742 366 734" strokeWidth="0.5" />
      </g>

      {/* the opening, with a rough plaster reveal that catches the sky */}
      <path d="M74 168 L316 164 L318 664 L74 668 Z" fill="#241d16" />
      <path d="M80 173 L310 169 L312 659 L80 663 Z" fill="#1a1510" />
      <rect x="84" y="172" width="226" height="488" fill={`url(#${gapSky})`} />
      <path d="M96 174 H302" stroke="#2a3a53" strokeWidth="0.7" opacity="0.5" />
      <Starfield count={26} opacity={0.5} tint="#b9c9e2" avoid={{ x: 166, y: 320, r: 86 }} />
      {/* the horizon, seen through the window: the town is out there */}
      <g opacity="0.7">
        <path d={`M84 556 C 120 552 190 558 310 553`} fill="none" stroke="#22344d" strokeWidth="1.2" />
        <rect x="84" y="553" width="226" height="40" fill="#070c15" opacity="0.9" />
        <path d="M120 553 L120 548 L132 548 L132 553 Z M150 553 L150 546 L166 546 L166 553 Z M200 553 L200 549 L212 549 L212 553 Z M250 553 L250 544 L268 544 L268 553 Z" fill="#050a12" />
      </g>

      {/* the leaves: left open wide, right drawn nearly shut — the gap is the choice */}
      <g>
        {/* right leaf, swung almost shut — it is the gap that is chosen */}
        <path d="M190 172 L304 167 L304 656 L190 660 Z" fill={`url(#${wood})`} />
        <g opacity="0.5" stroke="#120c08" strokeWidth="0.8">
          {[212, 234, 256, 278].map((x) => <path key={x} d={`M${x} 170 L${x} 658`} />)}
        </g>
        <path d="M190 172 L304 167 L304 656 L190 660 Z" fill="none" stroke="#0a0705" strokeWidth="1.3" />
        {/* the lit inner edge: the sky is the only light in this room */}
        <path d="M190 172 L190 660" stroke="#93aac6" strokeWidth="1.2" opacity="0.52" />
        <path d="M192.6 176 L192.6 656" stroke="#c7b394" strokeWidth="0.6" opacity="0.2" />
        <circle cx="248" cy="392" r="5.2" fill="#0d0906" opacity="0.75" />
        <circle cx="248" cy="392" r="3.1" fill="none" stroke="#5d4834" strokeWidth="0.9" opacity="0.8" />

        {/* left leaf, opened wide and away, seen almost edge-on */}
        <path d="M84 174 L142 180 L142 690 L84 686 Z" fill={`url(#${woodLit})`} />
        <g opacity="0.36" stroke="#100b07" strokeWidth="0.7">
          {[101, 115, 129].map((x) => <path key={x} d={`M${x} 176 L${x} 688`} />)}
        </g>
        <path d="M142 180 L142 690" stroke="#8ba2be" strokeWidth="1.1" opacity="0.44" />
        <path d="M84 174 L142 180 L142 690 L84 686 Z" fill="none" stroke="#0a0705" strokeWidth="1.2" />
        <circle cx="116" cy="404" r="4.6" fill="#0d0906" opacity="0.7" />
        <circle cx="116" cy="404" r="2.7" fill="none" stroke="#5d4834" strokeWidth="0.8" opacity="0.75" />
      </g>

      {/* the crescent exactly in the vertical of sky */}
      <g clipPath={`url(#es-gap-${pid})`}>
        <HeldMoon
          pid={`${pid}-in`}
          cx={162}
          cy={318}
          r={13.4}
          ash={ash}
          crescent={0.13}
          tilt={-11}
          halo={0.4}
          glow={0.2}
          ashTone="#8fa3c0"
        />
      </g>
      <defs>
        <clipPath id={`es-gap-${pid}`}><rect x="144" y="172" width="44" height="488" /></clipPath>
      </defs>

      {/* the lintel: a heavier beam over the opening, with a shadow beneath it */}
      <path d="M44 150 L346 145 L354 172 L38 177 Z" fill={`url(#${sill})`} />
      <path d="M44 150 L346 145" stroke="#8fa6c2" strokeWidth="1" opacity="0.24" />
      <path d="M38 177 L354 172 L354 180 L38 185 Z" fill="#060910" opacity="0.6" />
      {/* the sill: the one horizontal, catching a thin cool edge */}
      <path d="M56 658 L332 654 L342 676 L46 680 Z" fill={`url(#${sill})`} />
      <path d="M56 658 L332 654" stroke="#c3d2e6" strokeWidth="0.9" opacity="0.34" />
      <path d="M46 680 L342 676" stroke="#000" strokeWidth="1.1" opacity="0.45" />
      <ellipse cx="190" cy="664" rx="96" ry="14" fill="#9fb4cd" opacity="0.05" />
      <rect x="0" y="0" width={VIEW.w} height={VIEW.h} fill={`url(#${vig}-${pid})`} opacity="0.62" />
      <Atmosphere pid={pid} opacity={0.2} />
    </>
  );
}

/**
 * Scene 4, the intimate frame: the same sky through a longer lens. The disc is
 * legible, its maria are terrain, and the bright arc is still the loudest thing.
 */
function CloseWorld({ pid }) {
  const sky = `sky-${pid}`;
  const vig = 'vig';
  const ashGlow = `ashglow-${pid}`;
  return (
    <>
      <defs>
        <Linear id={sky} x1="0" y1="0" x2="0" y2="1" stops={[[0, '#04070f'], [0.4, '#070d18'], [0.72, '#0a1322'], [1, '#0f1a2b']]} />
        <Radial id={ashGlow} stops={[[0, '#4d6285', 0.34], [0.55, '#33445c', 0.1], [1, '#1c2736', 0]]} />
        <Radial id={`${vig}-${pid}`} stops={[[0.46, '#000000', 0], [1, '#000000', 0.55]]} cx="50%" cy="46%" r="72%" />
      </defs>
      <rect x="0" y="0" width={VIEW.w} height={VIEW.h} fill={`url(#${sky})`} />
      <Starfield count={104} opacity={0.55} tint="#c6d3e8" avoid={{ x: 195, y: 372, r: 132 }} />
      <ellipse cx="195" cy="372" rx="176" ry="176" fill={`url(#${ashGlow})`} />

      {/* the wall of the parapet at the bottom of the frame: we are standing at
          it, close enough that the moon fills the sky above it */}
      <path d="M0 690 C 70 682 130 692 200 686 C 272 680 330 690 390 683 L390 844 L0 844 Z" fill="#04060b" />
      <path d="M0 690 C 70 682 130 692 200 686 C 272 680 330 690 390 683" fill="none" stroke="#8ea3bd" strokeWidth="0.8" opacity="0.2" />
      <path d="M0 702 C 70 694 130 704 200 698 C 272 692 330 702 390 695" fill="none" stroke="#000" strokeWidth="1" opacity="0.4" />
      {/* one dark cypress tip, out of focus, at the left edge — scale */}
      <path d="M52 700 C 56 660 62 630 70 600 C 76 632 82 664 86 700 Z" fill="#03050a" opacity="0.8" />
      <path d="M92 702 C 96 674 100 652 106 628 C 111 654 116 678 119 702 Z" fill="#03050a" opacity="0.6" />

      <HeldMoon
        pid={pid}
        cx={195}
        cy={372}
        r={112}
        ash={0.62}
        crescent={0.078}
        tilt={-13}
        halo={0.46}
        glow={0.18}
        ashTone="#8ba0bd"
        maria={1.25}
      />
      <rect x="0" y="0" width={VIEW.w} height={VIEW.h} fill={`url(#${vig}-${pid})`} opacity="0.6" />
      <Atmosphere pid={pid} opacity="0.12" />
    </>
  );
}

/* ------------------------------------------------------------------ */
/* Scene registry                                                      */
/* ------------------------------------------------------------------ */

const NIGHT = {
  edge: '#0a0d14',
  wall: '#05070c',
  rim: '#8a9cb6',
  hut: '#070a10',
  hutDoor: '#020407',
  tree: '#04060a',
  rug: ['#1d2534', '#131a26'],
  rugBorder: '#313d52',
  motif: '#7d8daa',
  motifOpacity: 0.34,
  blanketTone: '#2b3342',
  rimOpacity: 0.3,
};

const WORLDS = {
  /* 1 · Welcome — the deepest dark: a bright sliver, a held dark, and almost
     nothing else. Depth is carried by the horizon, not by detail. */
  welcome: {
    sky: [[0, '#02040a'], [0.32, '#04070f'], [0.6, '#070c15'], [0.82, '#0a1220'], [1, '#0d1727']],
    stars: { count: 44, opacity: 0.55 },
    haze: {
      x: 210, y: HORIZON + 26, rx: 240, ry: 76,
      stops: [[0, '#41597c', 0.16], [0.42, '#2c3e57', 0.07], [1, '#1b2637', 0]],
    },
    moon: { ash: 0.05, crescent: 0.11, halo: 0.34, glow: 0.16 },
    plain: { land: 1, town: 0.85, glint: 0.14 },
    roof: {
      ...NIGHT, rimOpacity: 0.16, rim: '#61748f', motifOpacity: 0.11,
      rug: ['#161d29', '#0e141e'],
    },
  },

  /* 3 · Night — deep ultramarine, more stars than first seem to be there, one
     warm light far away on the plain, and a blanket left on the kilim. */
  night: {
    sky: [[0, '#03060d'], [0.26, '#061020'], [0.56, '#0a1a32'], [0.82, '#0f2543'], [1, '#162c4c']],
    stars: { count: 140, opacity: 0.95 },
    haze: {
      x: 196, y: HORIZON + 22, rx: 246, ry: 88,
      stops: [[0, '#3a5678', 0.2], [0.4, '#283c58', 0.09], [1, '#17202f', 0]],
    },
    moon: { ash: 0.22, crescent: 0.105, halo: 0.34, glow: 0.14 },
    plain: {
      land: 1, town: 1, glint: 0.5,
      lights: [{ x: 118, y: HORIZON + 15, r: 0.85, o: 0.4 }],
    },
    roof: { ...NIGHT, rim: '#8ba1bf', rimOpacity: 0.3, rug: ['#1b2331', '#101722'], blanket: 0.95 },
  },

  /* 5 · Dawn — the horizon band is the main event; the false dawn rises and
     fades above it, and the place begins to appear. Warmth stays on the horizon. */
  dawn: {
    sky: [[0, '#050a15'], [0.2, '#081120'], [0.42, '#101d33'], [0.6, '#22304a'], [0.76, '#3f4659'], [0.9, '#6f5f60'], [1, '#93705c']],
    stars: { count: 30, opacity: 0.34 },
    haze: {
      x: 206, y: HORIZON + 3, rx: 268, ry: 104,
      stops: [[0, '#f0bc90', 0.66], [0.28, '#cf9475', 0.44], [0.62, '#7b6470', 0.18], [1, '#3a3b4d', 0]],
    },
    falseDawn: 1,
    dawnX: 206,
    moon: { ash: 0.72, crescent: 0.1, halo: 0.24, glow: 0.1, crescentTone: ['#fdf9f2', '#e9e1da', '#c2b9bd'] },
    plain: {
      land: 1, town: 1, glint: 0.85,
      lights: [
        { x: 96, y: HORIZON + 13, r: 0.8, o: 0.34 },
        { x: 264, y: HORIZON + 17, r: 0.75, o: 0.28 },
        { x: 348, y: HORIZON + 14, r: 0.8, o: 0.22 },
      ],
    },
    roof: {
      ...NIGHT,
      edge: '#14171f', wall: '#080b12',
      rim: '#e2bb92', rimOpacity: 0.62,
      rug: ['#43333c', '#1d1824'], rugBorder: '#553f45', motif: '#b89a89', motifOpacity: 0.4,
      blanketTone: '#4b4350', tree: '#08080c',
    },
    faceGlow: 0.5,
    treeGlow: 1,
  },

  /* 6 · Three lights — sun rim, dawn band and the fully readable moon, all in
     one frame; balanced, luminous, still shadowed at the top. */
  three: {
    sky: [[0, '#071020'], [0.22, '#0f1e35'], [0.44, '#24304c'], [0.62, '#43475e'], [0.78, '#6f6168'], [0.9, '#a37a63'], [1, '#cf9469']],
    stars: { count: 12, opacity: 0.28 },
    haze: {
      x: 196, y: HORIZON + 5, rx: 290, ry: 124,
      stops: [[0, '#f9c992', 0.78], [0.26, '#e3a377', 0.5], [0.58, '#8a6a6e', 0.2], [1, '#3f4052', 0]],
    },
    sunRim: { x: 196, o: 1 },
    moon: {
      ash: 1, crescent: 0.09, halo: 0.18, glow: 0.09,
      ashTone: '#93a9c8', crescentTone: ['#fdf8ef', '#ece2d6', '#c3b7b2'],
    },
    plain: {
      land: 1, town: 1, glint: 1,
      lights: [{ x: 308, y: HORIZON + 15, r: 0.75, o: 0.3 }, { x: 66, y: HORIZON + 16, r: 0.75, o: 0.24 }],
    },
    roof: {
      ...NIGHT,
      edge: '#171a22', wall: '#090c13',
      rim: '#f0c79c', rimOpacity: 0.66,
      rug: ['#553a40', '#241d2c'], rugBorder: '#684b46', motif: '#cfa88a', motifOpacity: 0.5,
      blanketTone: '#5b5560', tree: '#08080c',
    },
    faceGlow: 0.7,
    treeGlow: 1,
  },

  /* 7 · Check-ins — the same frame a little later; a small brass lantern is
     the only thing added to the world since the story began. */
  checkins: {
    sky: [[0, '#0a1526'], [0.24, '#132138'], [0.46, '#26324c'], [0.64, '#414559'], [0.79, '#6b5d63'], [0.91, '#96705f'], [1, '#b48264']],
    stars: { count: 9, opacity: 0.24 },
    haze: {
      x: 150, y: HORIZON + 6, rx: 250, ry: 108,
      stops: [[0, '#efb98a', 0.6], [0.28, '#c9906f', 0.38], [0.62, '#75616a', 0.16], [1, '#3d3e4e', 0]],
    },
    moon: { ash: 1, crescent: 0.09, halo: 0.16, glow: 0.09, crescentTone: ['#fdf8ef', '#ece2d6', '#c3b7b2'] },
    plain: {
      land: 1, town: 1, glint: 0.9,
      lights: [{ x: 84, y: HORIZON + 14, r: 0.75, o: 0.26 }],
    },
    roof: {
      ...NIGHT,
      edge: '#15181f', wall: '#080b12',
      rim: '#dcb492', rimOpacity: 0.52,
      rug: ['#42323a', '#1c1724'], rugBorder: '#544045', motif: '#b8997f', motifOpacity: 0.44,
      blanketTone: '#4c4752', tree: '#08080c',
    },
    faceGlow: 0.4,
    lantern: 1,
    treeGlow: 0.8,
  },

  /* 8 · Final — the welcome frame again at first light: the sun's ray reaches
     the rug where somebody was sitting, the door stands open, the moon stays. */
  final: {
    sky: [[0, '#1a2742'], [0.2, '#2e3d5c'], [0.4, '#4d5a7c'], [0.58, '#7b8298'], [0.74, '#a99e9f'], [0.88, '#d3ac93'], [1, '#efc39a']],
    stars: { count: 5, opacity: 0.16 },
    haze: {
      x: 214, y: HORIZON + 8, rx: 310, ry: 142,
      stops: [[0, '#ffe0b0', 0.86], [0.24, '#eebc95', 0.54], [0.55, '#ab9098', 0.22], [1, '#5d5f75', 0]],
    },
    moon: {
      ash: 0.5, crescent: 0.088, halo: 0.1, glow: 0.06,
      ashTone: '#7e8da5',
      crescentTone: ['#fffaef', '#f0e4d6', '#c6b8b0'],
    },
    plain: { land: 1, town: 1, glint: 0.75, lights: [{ x: 302, y: HORIZON + 14, r: 0.75, o: 0.2 }] },
    roof: {
      ...NIGHT,
      edge: '#1d2029', wall: '#0a0d14',
      rim: '#ffd9a8', rimOpacity: 0.78,
      rug: ['#6d4744', '#2c2130'], rugBorder: '#8a5a49', motif: '#dfb191', motifOpacity: 0.62,
      blanketTone: '#7d7684', tree: '#07070b',
    },
    faceGlow: 1,
    lantern: 0.5,
    lanternX: 268,
    door: true,
    doorWarm: 0.8,
    ray: 0.9,
    treeGlow: 0.6,
  },
};

/**
 * The backdrop for the mood scale. It is the same roof, but the moon has left the
 * sky: on that screen the moon comes forward and the user answers with it, so two
 * moons would be a contradiction, not an echo.
 */
WORLDS.mood = {
  ...WORLDS.night,
  moon: null,
  stars: { count: 120, opacity: 0.85 },
  plain: { ...WORLDS.night.plain, glint: 0.34 },
};

const SCENE_LABELS = {
  welcome: 'شب پیش از سپیده؛ ماهِ باریک در آسمان و بامی خالی با قالیچه‌ای که کسی رویش نشسته بوده.',
  privacy: 'همان آسمان، از پشت پنجره‌ای با دو لت چوبی که کمی باز مانده و ماه در شکاف آن پیدا است.',
  night: 'آسمان پرستارهٔ نیمه‌شب روی بام؛ ماه کمی بالاتر آمده و پتویی روی قالیچه افتاده است.',
  seeing: 'نمای نزدیک ماهِ در آغوش: هلالِ روشن، قرصِ خاکستری‌اش با دریاهای تاریک، و ستاره‌ها دور آن.',
  dawn: 'سپیده در افق: نواری از طلایی و گل‌بهی، سپیدهٔ کاذب، و سرو و خانه‌های دور که کم‌کم پیدا می‌شوند.',
  three: 'سه نور با هم: خورشید تازه از افق سر زده، سپیده در کنارش و ماه با قرص خاکستری کامل بالای آن‌ها.',
  checkins: 'فانوسی کوچک روی جانپناهِ بام روشن است؛ نور گرمش فقط دیوار و قالیچه را روشن می‌کند.',
  final: 'اولین نور صبح روی قالیچه افتاده، درِ پلکان باز است و ماه هنوز در آسمان روشن مانده.',
  mood: 'همان بام زیر آسمان؛ ماه از آسمان کنار رفته تا خودش ترازوی حال باشد.',
};

const SCENE_META = {
  welcome: { kind: 'roof', world: 'welcome', ash: 0.055 },
  privacy: { kind: 'shutter', ash: 0.12 },
  night: { kind: 'roof', world: 'night', ash: 0.2 },
  seeing: { kind: 'close', ash: 0.52 },
  dawn: { kind: 'roof', world: 'dawn', ash: 0.68 },
  three: { kind: 'roof', world: 'three', ash: 1 },
  checkins: { kind: 'roof', world: 'checkins', ash: 1 },
  final: { kind: 'roof', world: 'final', ash: 0.55 },
  mood: { kind: 'roof', world: 'mood', ash: 0 },
};

/* ------------------------------------------------------------------ */
/* Public: the scene                                                   */
/* ------------------------------------------------------------------ */

export function EarthshineScene({ scene = 'welcome', variant = 'bleed', className = '', label }) {
  const raw = useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const pid = `es${raw}${scene}`;
  const meta = SCENE_META[scene] || SCENE_META.welcome;
  const description = label || SCENE_LABELS[scene] || SCENE_LABELS.welcome;

  const art = meta.kind === 'shutter'
    ? <ShutterWorld pid={pid} ash={meta.ash} />
    : meta.kind === 'close'
      ? <CloseWorld pid={pid} />
      : <RoofWorld pid={pid} world={WORLDS[meta.world]} />;

  return (
    <div className={`es-scene es-scene--${variant} es-scene--${scene} ${className}`.trim()} data-scene={scene}>
      <svg
        className="es-scene__svg"
        viewBox={`0 0 ${VIEW.w} ${VIEW.h}`}
        preserveAspectRatio="xMidYMid slice"
        role="img"
        aria-label={description}
        focusable="false"
      >
        {art}
      </svg>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/* Public: progress as the legibility of the ashen disc                */
/* ------------------------------------------------------------------ */

export const LEGIBILITY = {
  welcome: 0.06, privacy: 0.1,
  'simple-intro': 0.2, name: 0.21, age: 0.22, identity: 0.23, therapist: 0.24,
  'deeper-intro': 0.5, mood: 0.54, concerns: 0.56, focus: 0.58, feelings: 0.6,
  'know-intro': 0.7, faith: 0.72, neurodivergent: 0.74, relationships: 0.76,
  interests: 0.78, strengths: 0.8, barriers: 0.82, recharge: 0.84,
  'almost-intro': 1, approach: 1, voice: 1, checkins: 1, referral: 1, ready: 1,
};

export function legibilityFor(route) {
  return LEGIBILITY[route] ?? 0.2;
}

/**
 * A 24px moon that carries the whole story's progress: the travelled arc, and
 * the ashen disc brightening from barely visible to fully readable.
 */
export function MoonProgress({ route, progress = 0, className = '' }) {
  const routeKey = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const ash = legibilityFor(route);
  const r = 9.4;
  const circumference = 2 * Math.PI * r;
  const swept = Math.max(0.02, Math.min(1, progress));
  return (
    <span className={`ob-moon-progress ${className}`.trim()} aria-hidden="true">
      <svg viewBox="0 0 30 30" width="26" height="26" focusable="false">
        <circle cx="15" cy="15" r={r} fill="none" stroke="currentColor" strokeWidth="0.7" opacity="0.18" />
        <circle
          cx="15" cy="15" r={r} fill="none" stroke="currentColor" strokeWidth="1.1"
          strokeLinecap="round" opacity="0.62"
          strokeDasharray={`${(circumference * swept).toFixed(2)} ${circumference.toFixed(2)}`}
          transform="rotate(-90 15 15)"
        />
        <g transform="rotate(-12 19 14.4)" opacity="0.95">
          <circle cx="19" cy="14.4" r="4.3" fill="#9fb3cf" opacity={0.1 + ash * 0.62} />
          <Crescent id={`${routeKey}-cres`} cx={19} cy={14.4} r={4.3} shift={1.35} fill="currentColor" />
        </g>
      </svg>
    </span>
  );
}


/**
 * The held moon as a single drawing, for screens that need the moon itself at
 * large size (the mood scale). Same geometry, same palette and same legibility
 * rules as the stills: `ash` is the only thing that changes.
 */
export function HeldMoonFigure({
  ash = 0.5, size = 300, tilt = -13, cx = 150, cy = 138, r = 96,
  seas = [], activeIndex = -1, craters = 26, className = '', label,
}) {
  const raw = useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const pid = `hm${raw}`;
  const disc = `${pid}-disc`;
  const bloom = `${pid}-bloom`;
  const tycho = `${pid}-tycho`;
  const limb = `${pid}-limb`;
  const cres = `${pid}-cres`;
  const cresGlow = `${pid}-cresglow`;
  const soft = `${pid}-soft`;
  const clip = `${pid}-clip`;
  const height = Math.round((size * 286) / 300);

  const randCraters = React.useMemo(() => {
    const out = [];
    let seed = 0x2f1d;
    const rand = () => {
      seed = (seed * 16807) % 2147483647;
      return seed / 2147483647;
    };
    for (let i = 0; i < craters; i += 1) {
      const a = rand() * Math.PI * 2;
      const rad = Math.sqrt(rand()) * 0.82;
      out.push({ x: Math.cos(a) * rad, y: Math.sin(a) * rad, r: 0.012 + Math.pow(rand(), 2.4) * 0.058, o: 0.16 + rand() * 0.34 });
    }
    return out;
  }, [craters]);

  return (
    <svg className={className} viewBox="0 0 300 286" width={size} height={height}
         role="img" aria-label={label || 'ماهِ در آغوش'} focusable="false">
      <defs>
        <radialGradient id={disc} cx="42%" cy="40%" r="66%">
          <stop offset="0" stopColor="#a8bcd8" stopOpacity={ash * 0.72} />
          <stop offset="0.62" stopColor="#8ba0bd" stopOpacity={ash * 0.5} />
          <stop offset="0.9" stopColor="#6f86a4" stopOpacity={ash * 0.26} />
          <stop offset="1" stopColor="#5d748f" stopOpacity={ash * 0.08} />
        </radialGradient>
        <radialGradient id={bloom} cx="50%" cy="50%" r="50%">
          <stop offset="0" stopColor="#8fa6c4" stopOpacity={0.05 + ash * 0.06} />
          <stop offset="0.5" stopColor="#6d84a4" stopOpacity="0.018" />
          <stop offset="1" stopColor="#5e748f" stopOpacity="0" />
        </radialGradient>
        <radialGradient id={tycho} cx="50%" cy="50%" r="50%">
          <stop offset="0" stopColor="#eef4ff" stopOpacity={0.3 * ash} />
          <stop offset="1" stopColor="#eef4ff" stopOpacity="0" />
        </radialGradient>
        <linearGradient id={limb} x1="0" y1="0" x2="1" y2="0">
          <stop offset="0" stopColor="#e8f1ff" stopOpacity="0.3" />
          <stop offset="0.55" stopColor="#e8f1ff" stopOpacity="0" />
        </linearGradient>
        <linearGradient id={cres} x1="0" y1="1" x2="0.35" y2="0">
          <stop offset="0" stopColor="#b9cbe2" />
          <stop offset="0.3" stopColor="#dde8f7" />
          <stop offset="0.66" stopColor="#fbfdff" />
          <stop offset="1" stopColor="#dde8f7" />
        </linearGradient>
        <radialGradient id={cresGlow} cx="50%" cy="50%" r="50%">
          <stop offset="0" stopColor="#dce9fb" stopOpacity="0.42" />
          <stop offset="0.5" stopColor="#b9cde8" stopOpacity="0.12" />
          <stop offset="1" stopColor="#8ea8c9" stopOpacity="0" />
        </radialGradient>
        <filter id={soft} x="-40%" y="-40%" width="180%" height="180%">
          <feGaussianBlur stdDeviation="4.2" />
        </filter>
        <clipPath id={clip}><circle cx={cx} cy={cy} r={r} /></clipPath>
      </defs>

      <circle cx={cx} cy={cy} r={r * 2.1} fill={`url(#${bloom})`} />
      <circle cx={cx} cy={cy} r={r} fill={`url(#${disc})`} />

      <g clipPath={`url(#${clip})`}>
        <g transform={`translate(${cx} ${cy}) rotate(${tilt})`}>
          {seas.map((sea, index) => {
            const isActive = index === activeIndex;
            return (
              <g key={sea.key || index}>
                <path
                  d={sea.d}
                  fill={isActive ? '#dfe9fa' : '#33445c'}
                  fillOpacity={isActive ? 0.13 + ash * 0.1 : 0.2 + ash * 0.14}
                  filter={`url(#${soft})`}
                />
                {isActive && (
                  <path className="mood-sea-marker" d={sea.d} fill="none" stroke="#eaf2ff"
                        strokeOpacity="0.4" strokeWidth="1" filter={`url(#${soft})`} />
                )}
              </g>
            );
          })}
          {randCraters.map((c, index) => {
            const px = c.x * r;
            const py = c.y * r;
            const cr = c.r * r;
            return (
              <g key={index} opacity={c.o * (0.4 + ash * 0.6)}>
                <ellipse cx={px} cy={py} rx={cr} ry={cr * 0.92} fill="#0e1622" opacity="0.5" />
                <path d={`M${px + cr * 0.1} ${py - cr * 0.9} a ${cr} ${cr * 0.92} 0 0 1 ${cr * 0.72} ${cr * 0.62}`}
                      fill="none" stroke="#e6eefb" strokeWidth={Math.max(0.3, cr * 0.26)} opacity="0.4" />
              </g>
            );
          })}
        </g>
        <ellipse cx={cx - 0.16 * r} cy={cy + 0.5 * r} rx={0.34 * r} ry={0.3 * r} fill={`url(#${tycho})`} />
        <circle cx={cx} cy={cy} r={r} fill={`url(#${limb})`} opacity="0.7" />
      </g>

      <g transform={`rotate(${tilt} ${cx} ${cy})`}>
        <circle cx={cx + r * 0.62} cy={cy + r * 0.66} r={r * 1.42} fill={`url(#${cresGlow})`} opacity={0.22 + ash * 0.16} />
        <path
          d={`M${cx - r * 0.62} ${cy + r * 0.78} A${r} ${r} 0 0 1 ${cx + r * 0.34} ${cy - r * 0.94} A${r * 0.96} ${r * 0.96} 0 0 0 ${cx - r * 0.62} ${cy + r * 0.78} Z`}
          fill={`url(#${cres})`} opacity="0.95"
        />
        <path
          d={`M${cx - r * 0.62} ${cy + r * 0.78} A${r} ${r} 0 0 1 ${cx + r * 0.34} ${cy - r * 0.94}`}
          fill="none" stroke="#ffffff" strokeOpacity="0.5" strokeWidth="0.7"
        />
      </g>
      <circle cx={cx} cy={cy} r={r} fill="none" stroke="#b7cbe4" strokeOpacity={0.16 + ash * 0.14} strokeWidth="0.8" />
    </svg>
  );
}

/** The same moon, still and small — the assistant's mark inside the preview. */
export function HeldMoonGlyph({ size = 18, className = '' }) {
  return (
    <svg
      viewBox="0 0 24 24" width={size} height={size}
      className={`held-moon-glyph ${className}`.trim()}
      aria-hidden="true" focusable="false"
    >
      <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" strokeWidth="0.9" opacity="0.28" />
      <g transform="rotate(-14 12.6 12.4)" opacity="0.92">
        <circle cx="12.6" cy="12.4" r="5.6" fill="currentColor" opacity="0.16" />
        <Crescent id="held-glyph-cres" cx={12.6} cy={12.4} r={5.6} shift={1.7} fill="currentColor" />
      </g>
    </svg>
  );
}
