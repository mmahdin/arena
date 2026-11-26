import { createWriteStream } from 'node:fs';
import { deflateSync } from 'node:zlib';

/**
 * Rasterises Theia's Earthshine Moon mark into PNG app icons without any image
 * dependency. Samples a rounded-square deep nocturnal sky, a 3D shaded earthlit
 * blue-gray lunar sphere, a soft elliptical terminator, and a cool-white waning
 * morning crescent with 4x supersampling.
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

const SKY_IN = [14, 23, 40];
const SKY_OUT = [6, 9, 17];
const ASHEN_LIGHT = [88, 110, 139];
const ASHEN_MID = [60, 80, 107];
const ASHEN_DARK = [39, 54, 75];
const CRESC_LIGHT = [247, 250, 254];
const CRESC_EDGE = [194, 210, 230];

const CORNER = 60;
const MOON_R = 80;
const TILT_RAD = (46 * Math.PI) / 180; // inverse rotation (+46°) into unrotated lunar coords

function roundedSquareCoverage(x, y) {
  const half = SIZE / 2;
  const dx = Math.max(Math.abs(x - half) - (half - CORNER), 0);
  const dy = Math.max(Math.abs(y - half) - (half - CORNER), 0);
  if (dx === 0 && dy === 0) return 1;
  return clamp01(CORNER - Math.hypot(dx, dy) + 0.5);
}

function samplePixel(x, y) {
  const cover = roundedSquareCoverage(x, y);
  if (cover <= 0) return [0, 0, 0, 0];

  const nx = (x - SIZE / 2) / (SIZE / 2);
  const ny = (y - SIZE * 0.42) / (SIZE * 0.72);
  let rgb = mix(SKY_IN, SKY_OUT, clamp01(Math.hypot(nx, ny)));

  const dx = x - 128;
  const dy = y - 128;
  const dist = Math.hypot(dx, dy);

  // Subtle crescent glare halo on the lower-left limb
  const hx = x - 102;
  const hy = y - 154;
  const haloT = clamp01(1 - Math.hypot(hx, hy) / 108);
  if (haloT > 0) {
    rgb = mix(rgb, [224, 235, 248], haloT * haloT * 0.14);
  }

  if (dist <= MOON_R + 0.6) {
    const discAlpha = clamp01(MOON_R + 0.5 - dist);
    // Ashen sphere shading
    const sphereD = clamp01(Math.hypot(dx - 22, dy + 22) / (MOON_R * 1.45));
    const ashen = sphereD < 0.55
      ? mix(ASHEN_LIGHT, ASHEN_MID, sphereD / 0.55)
      : mix(ASHEN_MID, ASHEN_DARK, (sphereD - 0.55) / 0.45);

    // Rotate into lunar frame where sunlit limb is on -X
    const lx = dx * Math.cos(TILT_RAD) - dy * Math.sin(TILT_RAD);
    const ly = dx * Math.sin(TILT_RAD) + dy * Math.cos(TILT_RAD);
    const yNorm = Math.min(1, Math.abs(ly) / MOON_R);
    const chord = Math.sqrt(Math.max(0, 1 - yNorm * yNorm)) * MOON_R;

    // Elliptical terminator x = -0.785 * chord, softened across penumbra
    const termX = -0.785 * chord;
    const penumbraWidth = Math.max(1.4, 0.11 * chord);
    const sunlit = clamp01((termX - lx) / penumbraWidth + 0.5);

    const limbProgress = chord > 0.1 ? clamp01((-lx - 0.76 * chord) / (0.24 * chord)) : 1;
    const crescColor = mix(CRESC_EDGE, CRESC_LIGHT, limbProgress);
    const moonColor = mix(ashen, crescColor, sunlit);
    rgb = mix(rgb, moonColor, discAlpha * 0.94);
  }

  return [rgb[0], rgb[1], rgb[2], cover * 255];
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
