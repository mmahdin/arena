/**
 * Design QA for the Earthshine onboarding: renders each scene at the exact
 * still geometry the screens use, so composition can be reviewed without a
 * browser. (Layout and type are checked in the running preview.)
 *
 * Usage: node scripts/render-screens.mjs [scene ...]
 */
import { mkdirSync, writeFileSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { resolve } from 'node:path';

const root = resolve(import.meta.dirname, '..');
const out = resolve(root, '.design/screens');
mkdirSync(out, { recursive: true });

const scenes = process.argv.slice(2).length
  ? process.argv.slice(2)
  : ['welcome', 'privacy', 'night', 'seeing', 'dawn', 'three', 'checkins', 'final', 'mood'];

writeFileSync(resolve(root, '.design/screens-entry.jsx'), `
import { renderToStaticMarkup } from 'react-dom/server';
import { EarthshineScene } from '../src/components/EarthshineScene';
export const scenes = ${JSON.stringify(scenes)};
export function markup(scene) { return renderToStaticMarkup(<EarthshineScene scene={scene} />); }
`);

const { build } = await import('esbuild');
await build({
  entryPoints: [resolve(root, '.design/screens-entry.jsx')],
  bundle: true, format: 'esm', platform: 'node',
  outfile: resolve(root, '.design/screens.mjs'),
  jsx: 'automatic', external: ['react', 'react-dom'],
  logLevel: 'silent',
});

const mod = await import(resolve(root, '.design/screens.mjs') + `?t=${Date.now()}`);
for (const scene of mod.scenes) {
  const html = mod.markup(scene);
  const svg = html.slice(html.indexOf('<svg'), html.indexOf('</svg>') + 6);
  writeFileSync(resolve(out, `${scene}.svg`), svg);
  execFileSync('python3', [resolve(root, 'scripts/svg-to-png.py'), resolve(out, `${scene}.svg`), resolve(out, `${scene}.png`)]);
  console.log('rendered', scene);
}
