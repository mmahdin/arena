import { createWriteStream } from 'node:fs';
import { deflateSync } from 'node:zlib';

/**
 * Rasterises Theia's Held Moon (ماهِ در آغوش) sigil into PNG app icons without
 * any external image dependency: a deep pre-dawn night rounded square holding
 * the cool blue-gray Earthshine sphere (with soft atmospheric lunar maria)
 * cradled by a slender cool-white lower-right waning morning crescent, sampled
 * at 4x supersampling.
 *
 * Usage: node scripts/generate-theia-icons.mjs
 */

const SIZE = 256;
const SS = 4;

const clamp01 = (value) => Math.min(1, Math.max(0, value));

function mix(a, b, t) {
  return [
    Math.round(a[0] + (b[0] - a[0]) * t),
    Math.round(a[1] + (b[1] - a[1]) * t),
    Math.round(a[2] + (b[2] - a[2]) * t),
  ];
}

const SKY_IN = [18, 25, 40];
const SKY_OUT = [7, 9, 15];
const ASHEN_LIGHT = [124, 148, 180];
const ASHEN_MID = [73, 92, 120];
const ASHEN_DARK = [32, 43, 61];
const MARIA_DARK = [22, 32, 48];
const CRESCENT_BRIGHT = [246, 249, 254];
const CRESCENT_MID = [220, 231, 245];
const CRESCENT_COOL = [159, 181, 207];

const CORNER = 60;
const MOON = { cx: 128, cy: 128, r: 86 };
const CUT = { cx: 118.0, cy: 117.2, r: 83.9 };

const MARIA_WASHES = [
  { cx: 98, cy: 106, rx: 48, ry: 40, rot: (-24 * Math.PI) / 180, strength: 0.56 },
  { cx: 150, cy: 114, rx: 40, ry: 33, rot: (16 * Math.PI) / 180, strength: 0.48 },
  { cx: 116, cy: 156, rx: 40, ry: 28, rot: (8 * Math.PI) / 180, strength: 0.44 },
];

function roundedSquareCoverage(x, y) {
  const half = SIZE / 2;
  const dx = Math.max(Math.abs(x - half) - (half - CORNER), 0);
  const dy = Math.max(Math.abs(y - half) - (half - CORNER), 0);
  if (dx === 0 && dy === 0) return 1;
  return clamp01(CORNER - Math.hypot(dx, dy) + 0.5);
}

function mariaShade(x, y) {
  let acc = 0;
  for (const m of MARIA_WASHES) {
    const dx = x - m.cx;
    const dy = y - m.cy;
    const cos = Math.cos(-m.rot);
    const sin = Math.sin(-m.rot);
    const rx = (dx * cos - dy * sin) / m.rx;
    const ry = (dx * sin + dy * cos) / m.ry;
    const d2 = rx * rx + ry * ry;
    if (d2 < 1.8) {
      acc += m.strength * Math.exp(-d2 * 1.9);
    }
  }
  return clamp01(acc);
}

function samplePixel(x, y) {
  let bg = [0, 0, 0, 0];
  const cover = roundedSquareCoverage(x, y);
  if (cover <= 0) return bg;

  const nx = (x - SIZE * 0.5) / (SIZE * 0.55);
  const ny = (y - SIZE * 0.38) / (SIZE * 0.68);
  bg = [...mix(SKY_IN, SKY_OUT, clamp01(Math.hypot(nx, ny))), cover * 255];

  const paint = (color, alpha) => {
    const a = clamp01(alpha);
    if (a <= 0) return;
    const dstA = bg[3] / 255;
    const outA = a + dstA * (1 - a);
    bg = [
      (color[0] * a + bg[0] * dstA * (1 - a)) / (outA || 1),
      (color[1] * a + bg[1] * dstA * (1 - a)) / (outA || 1),
      (color[2] * a + bg[2] * dstA * (1 - a)) / (outA || 1),
      outA * 255,
    ];
  };

  // Subtle optical glare around the lower-right sunlit limb
  const hDist = Math.hypot(x - 144, y - 145);
  if (hDist < 102) {
    const halo = Math.pow(clamp01(1 - hDist / 102), 2.6) * 0.11;
    paint([214, 228, 248], halo);
  }

  const mDist = Math.hypot(x - MOON.cx, y - MOON.cy);
  const sphereCov = clamp01(MOON.r - mDist + 0.5);
  if (sphereCov > 0) {
    const gx = (x - (MOON.cx + MOON.r * 0.32)) / (MOON.r * 1.48);
    const gy = (y - (MOON.cy + MOON.r * 0.36)) / (MOON.r * 1.48);
    const gt = clamp01(Math.hypot(gx, gy));
    let ashen = gt < 0.48
      ? mix(ASHEN_LIGHT, ASHEN_MID, gt / 0.48)
      : mix(ASHEN_MID, ASHEN_DARK, (gt - 0.48) / 0.52);

    const maria = mariaShade(x, y);
    if (maria > 0) ashen = mix(ashen, MARIA_DARK, maria);
    paint(ashen, sphereCov * 0.85);

    const rimBand = 1 - Math.abs(mDist - (MOON.r - 0.9)) / 1.25;
    if (rimBand > 0) paint([143, 166, 196], clamp01(rimBand) * 0.30 * sphereCov);

    const cDist = Math.hypot(x - CUT.cx, y - CUT.cy);
    const innerHole = clamp01(CUT.r - cDist + 0.5);
    const crescentCov = sphereCov * (1 - innerHole);
    if (crescentCov > 0) {
      const ct = clamp01(((194 - x) * 0.5 + (198 - y) * 0.5) / 132);
      const col = ct < 0.60
        ? mix(CRESCENT_BRIGHT, CRESCENT_MID, ct / 0.60)
        : mix(CRESCENT_MID, CRESCENT_COOL, (ct - 0.60) / 0.40);
      paint(col, crescentCov);
    }
  }

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
  header[8] = 8;
  header[9] = 6;
  const stride = width * 4;
  const raw = Buffer.alloc((stride + 1) * height);
  for (let y = 0; y < height; y += 1) {
    raw[y * (stride + 1)] = 0;
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
