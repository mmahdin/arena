/**
 * Design-QA spec extraction: server-renders the real onboarding screens and
 * emits a small JSON description (scene, surface, text blocks, footer) that
 * scripts/qa-screens.py composes into phone-sized stills.
 *
 * This exists because the sandbox has no usable browser; it is a review aid,
 * not a test of layout.
 */
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';

const root = resolve(import.meta.dirname, '..');
const out = resolve(root, '.design');
mkdirSync(out, { recursive: true });

writeFileSync(resolve(out, 'qa-entry.jsx'), `
import { renderToStaticMarkup } from 'react-dom/server';
import OnboardingScreen from '../src/screens/Onboarding';
export const routes = __ROUTES__;
export function markup(route, answers) {
  return renderToStaticMarkup(
    <OnboardingScreen route={route} navigate={() => {}} back={() => {}}
      answers={answers} setAnswer={() => {}} notify={() => {}} name=""
      setName={() => {}} privacySharing privacyLoaded
      onTogglePrivacy={() => {}} />,
  );
}
`);

const routes = process.argv.slice(2).length ? process.argv.slice(2) : [
  'welcome', 'privacy', 'simple-intro', 'name', 'age', 'identity', 'therapist', 'mood',
  'concerns', 'deeper-intro', 'know-intro', 'interests', 'relationships', 'recharge',
  'almost-intro', 'approach', 'voice', 'checkins', 'referral', 'ready',
];
writeFileSync(resolve(out, 'qa-entry.jsx'),
  (await import('node:fs')).readFileSync(resolve(out, 'qa-entry.jsx'), 'utf8')
    .replace('__ROUTES__', JSON.stringify(routes)));

const { build } = await import('esbuild');
await build({
  entryPoints: [resolve(out, 'qa-entry.jsx')],
  bundle: true, format: 'esm', platform: 'node',
  outfile: resolve(out, 'qa.mjs'),
  jsx: 'automatic',
  external: ['react', 'react-dom'],
  banner: {
    js: "import { createRequire as __cr } from 'node:module'; const require = __cr(import.meta.url);",
  },
  loader: { '.css': 'empty' }, logLevel: 'silent',
});

const mod = await import(resolve(out, 'qa.mjs') + `?t=${Date.now()}`);
const strip = (html) => html.replace(/<[^>]+>/g, '').replace(/&#x27;/g, "'").replace(/&quot;/g, '"').replace(/&amp;/g, '&').trim();

const specs = [];
for (const route of routes) {
  const html = mod.markup(route, { mood: 2, age: '۲۵–۲۹' });
  const scene = (html.match(/data-ob-scene="([a-z]+)"/) || [])[1] || 'welcome';
  const surface = /ob-paper/.test(html) ? 'paper' : 'night';
  const frame = (html.match(/ob-frame-([a-z]+)/) || [])[1] || 'band';
  const grab = (cls) => {
    const re = new RegExp(`<([a-z0-9]+) class="${cls}"[^>]*>([\\s\\S]*?)</\\1>`);
    const m = html.match(re);
    return m ? strip(m[2]) : '';
  };
  const chapter = grab('ob-chapter');
  const epigraph = grab('ob-epigraph');
  const title = (html.match(/<h1>([\s\S]*?)<\/h1>/) || [])[1];
  const lede = grab('ob-lede');
  const disclosure = grab('ob-disclosure');
  const footnote = grab('mood-footnote');
  const moodLabel = (html.match(/<h2 aria-live="polite">([\s\S]*?)<\/h2>/) || [])[1];
  const moodSea = (html.match(/class="mood-sea-name"[^>]*>([\s\S]*?)<\/p>/) || [])[1];
  const moodSvg = html.match(/<svg class="mood-moon"[\s\S]*?<\/svg>/);
  const moodMoon = moodSvg ? moodSvg[0].replace(/^<svg[^>]*>/, '').replace(/<\/svg>$/, '') : '';
  const moodLevelMatch = html.match(/aria-valuetext="[^"]*"/);
  const buttons = [];
  const choiceRe = /class="choice-button[^"]*"[\s\S]*?<span>([\s\S]*?)<\/span>/g;
  let choice;
  while ((choice = choiceRe.exec(html))) buttons.push(strip(choice[1]));
  const primary = (html.match(/class="primary-button[^"]*"[\s\S]*?>([^<]+)<\/button>/) || [])[1];
  const skip = (html.match(/class="skip-button"[^>]*>([^<]+)</) || [])[1];
  const privacy = (html.match(/class="privacy-card"[\s\S]*?<span>([^<]+)<\/span>/) || [])[1];
  const namePlaceholder = (html.match(/placeholder="([^"]+)"/) || [])[1];
  const ageOptions = [];
  const ageRe = /class="age-option[^"]*"[^>]*>([^<]+)</g;
  let age;
  while ((age = ageRe.exec(html))) ageOptions.push(strip(age[1]));
  specs.push({
    route, scene, surface, frame,
    chapter: strip(chapter || ''),
    epigraph: strip(epigraph || ''),
    title: strip(title || ''),
    lede: strip(lede || ''),
    disclosure: strip(disclosure || ''),
    footnote: strip(footnote || ''),
    moodLabel: strip(moodLabel || ''),
    moodSea: strip(moodSea || ''),
    moodMoon,
    moodLevel: 2,
    choices: buttons,
    primary: primary ? strip(primary) : '',
    skip: skip ? strip(skip) : '',
    privacy: privacy ? strip(privacy) : '',
    privacyCard: /privacy-card/.test(html),
    namePlaceholder: namePlaceholder ? strip(namePlaceholder) : '',
    ageOptions,
  });
  console.log('spec', route, scene, surface, frame);
}
writeFileSync(resolve(out, 'spec.json'), JSON.stringify(specs, null, 2));
console.log('wrote .design/spec.json');
