import { Resvg } from '@resvg/resvg-js';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';

const root = resolve(import.meta.dirname, '..');
const out = join(root, 'public', 'icons');
mkdirSync(out, { recursive: true });

const render = (file, size, name) => {
  const svg = readFileSync(join(root, 'public', 'brand', file), 'utf8');
  const png = new Resvg(svg, { fitTo: { mode: 'width', value: size } }).render().asPng();
  writeFileSync(join(out, name), png);
  console.log(`${name} (${size}×${size})`);
};

render('theia-app-icon.svg', 192, 'icon-192.png');
render('theia-app-icon.svg', 512, 'icon-512.png');
render('theia-app-icon.svg', 1024, 'icon-1024.png');
render('theia-app-icon-maskable.svg', 512, 'icon-maskable-512.png');
render('theia-app-icon.svg', 180, 'apple-touch-icon.png');
render('theia-app-icon.svg', 32, 'favicon-32.png');
