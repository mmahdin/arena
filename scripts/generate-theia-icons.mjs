import { createWriteStream } from 'node:fs';
import { deflateSync } from 'node:zlib';

/**
 * Rasterises Theia's open-ring sigil into PNG app icons without any image
 * dependency. The sigil is defined by a handful of analytic shapes (rounded
 * square, annulus with a gap, four-point spark, ember dot), so coverage can be
 * sampled directly and clipped with 4x supersampling for clean edges.
 *
 * Usage: node scripts/generate-theia-icons.mjs
 */

const SIZE = 256; // logical design grid, matches public/favicon.svg
const SS = 4; // supersampling factor

const clamp01 = (value) => Math.min(1, Math.max(0, value));

function mix(a, b, t) {
  return [
    Math.round(a[0] + (b[0] - a[0]) * t),
    Math.round(a[1] + (b[1] - a[1]) * t),
    Math.round(a[2] + (b[2] - a[2]) * t),
  ];
}

const NIGHT_IN = [26, 29, 36];
const NIGHT_OUT = [10, 11, 13];
const GOLD_LIGHT = [247, 231, 198];
const GOLD_MID = [227, 196, 140];
const GOLD_DARK = [185, 138, 78];

/** Theia's four-point spark, sampled along its quadratic sides. */
function sparkPolygon(cx, cy, radius) {
  const k = radius * 0.085;
  const quad = (a, c, b, steps = 10) => {
    const points = [];
    for (let i = 0; i <= steps; i += 1) {
      const t = i / steps;
      const u = 1 - t;
      points.push([
        u * u * a[0] + 2 * u * t * c[0] + t * t * b[0],
        u * u * a[1] + 2 * u * t * c[1] + t * t * b[1],
      ]);
    }
    return points;
  };
  const top = [cx, cy - radius];
  const right = [cx + radius, cy];
  const bottom = [cx, cy + radius];
  const left = [cx - radius, cy];
  return [
    ...quad(top, [cx + k, cy - k], right),
    ...quad(right, [cx + k, cy + k], bottom),
    ...quad(bottom, [cx - k, cy + k], left),
    ...quad(left, [cx - k, cy - k], top),
  ];
}

function insidePolygon(poly, x, y) {
  let inside = false;
  for (let i = 0, j = poly.length - 1; i < poly.length; j = i, i += 1) {
    const [xi, yi] = poly[i];
    const [xj, yj] = poly[j];
    if ((yi > y) !== (yj > y) && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi) inside = !inside;
  }
  return inside;
}

// Design-space geometry (identical to public/favicon.svg, scaled to 256).
const RING = { cx: 128, cy: 128, r: 104, width: 8.8, gapDeg: 58 };
// A needle of light rather than a generic sparkle: taller than it is wide.
const SPARK = [
  ...sparkPolygon(128, 128, 44).map(([x, y]) => [128 + (x - 128) * 0.74, y]),
];
const CORNER = 60; // 15/64 * 256

function roundedSquareCoverage(x, y) {
  const inset = 0;
  const half = SIZE / 2;
  const dx = Math.max(Math.abs(x - half) - (half - CORNER - inset), 0);
  const dy = Math.max(Math.abs(y - half) - (half - CORNER - inset), 0);
  if (dx === 0 && dy === 0) return 1;
  const dist = Math.hypot(dx, dy);
  return clamp01(CORNER - dist + 0.5);
}

function goldAt(x, y) {
  // Diagonal sweep from the top-right (light) to the bottom-left (deep gold).
  const t = clamp01(((x - 24) / 208) * 0.55 + ((y - 18) / 220) * 0.45);
  return t < 0.46
    ? mix(GOLD_LIGHT, GOLD_MID, t / 0.46)
    : mix(GOLD_MID, GOLD_DARK, (t - 0.46) / 0.54);
}

function samplePixel(x, y) {
  let bg = [0, 0, 0, 0];
  const cover = roundedSquareCoverage(x, y);
  if (cover > 0) {
    const nx = (x - SIZE / 2) / (SIZE / 2);
    const ny = (y - SIZE * 0.4) / (SIZE * 0.7);
    const radial = clamp01(Math.hypot(nx, ny));
    bg = [...mix(NIGHT_IN, NIGHT_OUT, radial), cover];
  }
  const paint = (color, alpha) => {
    if (alpha <= 0) return;
    const a = clamp01(alpha);
    const dstA = bg[3] / 255;
    const outA = a + dstA * (1 - a);
    const blend = (index) => (color[index] * a + bg[index] * dstA * (1 - a)) / (outA || 1);
    bg = [blend(0), blend(1), blend(2), outA * 255];
  };

  const dx = x - RING.cx;
  const dy = y - RING.cy;
  const dist = Math.hypot(dx, dy);
  const half = RING.gapDeg / 2;
  const fromTop = (((Math.atan2(dy, dx) * 180) / Math.PI) + 90 + 360) % 360;
  const inSector = fromTop <= 180 - half || fromTop >= 180 + half;
  const inBand = Math.abs(dist - RING.r) <= RING.width / 2;
  let alpha = inSector && inBand ? 1 : 0;
  for (const side of [-1, 1]) {
    const theta = ((side * half - 90) * Math.PI) / 180;
    const capX = RING.cx + RING.r * Math.cos(theta);
    const capY = RING.cy + RING.r * Math.sin(theta);
    if (Math.hypot(x - capX, y - capY) <= RING.width / 2) alpha = 1;
  }
  if (alpha > 0) {
    // A soft inner shading keeps the ring from looking flat at small sizes.
    const shade = 0.88 + 0.12 * clamp01((RING.r + RING.width / 2 - dist) / RING.width);
    paint(goldAt(x, y).map((c) => Math.round(c * shade)), 1);
  }
  if (insidePolygon(SPARK, x, y)) paint(goldAt(x, y), 1);
  return bg;
}

function renderIcon(size) {
  const data = Buffer.alloc(size * size * 4);
  const scale = SIZE / size;
  for (let py = 0; py < size; py += 1) {
    for (let px = 0; px < size; px += 1) {
      let r = 0; let g = 0; let b = 0; let a = 0;
      for (let sy = 0; sy < SS; sy += 1) {
        for (let sx = 0; sx < SS; sx += 1) {
          const x = (px + (sx + 0.5) / SS) * scale;
          const y = (py + (sy + 0.5) / SS) * scale;
          const [rr, gg, bb, aa] = samplePixel(x, y);
          r += rr * aa; g += gg * aa; b += bb * aa; a += aa;
        }
      }
      const total = SS * SS;
      const alpha = a / total;
      const offset = (py * size + px) * 4;
      if (alpha > 0) {
        data[offset] = Math.round(r / a);
        data[offset + 1] = Math.round(g / a);
        data[offset + 2] = Math.round(b / a);
        data[offset + 3] = Math.round(alpha);
      }
    }
  }
  return encodePng(size, size, data);
}

function crc32(buffer) {
  let crc = 0xFFFFFFFF;
  for (const byte of buffer) {
    crc ^= byte;
    for (let i = 0; i < 8; i += 1) crc = (crc >>> 1) ^ (0xEDB88320 & -(crc & 1));
  }
  return (crc ^ 0xFFFFFFFF) >>> 0;
}

function chunk(type, body) {
  const length = Buffer.alloc(4);
  length.writeUInt32BE(body.length);
  const typeAndBody = Buffer.concat([Buffer.from(type, 'ascii'), body]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(typeAndBody));
  return Buffer.concat([length, typeAndBody, crc]);
}

function encodePng(width, height, rgba) {
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8; // bit depth
  header[9] = 6; // truecolour with alpha
  const stride = width * 4;
  const raw = Buffer.alloc((stride + 1) * height);
  for (let y = 0; y < height; y += 1) {
    raw[y * (stride + 1)] = 0; // no filter
    rgba.copy(raw, y * (stride + 1) + 1, y * stride, (y + 1) * stride);
  }
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A]),
    chunk('IHDR', header),
    chunk('IDAT', deflateSync(raw, { level: 9 })),
    chunk('IEND', Buffer.alloc(0)),
  ]);
}

const targets = [
  ['public/brand/theia-sigil-192.png', 192],
  ['public/brand/theia-sigil-512.png', 512],
  ['public/apple-touch-icon.png', 180],
  ['public/brand/theia-sigil-1024.png', 1024],
];

for (const [path, size] of targets) {
  const bytes = renderIcon(size);
  await new Promise((resolve, reject) => {
    const stream = createWriteStream(path);
    stream.on('error', reject);
    stream.on('finish', resolve);
    stream.end(bytes);
  });
  console.log(`wrote ${path} (${size}px, ${bytes.length} bytes)`);
}
