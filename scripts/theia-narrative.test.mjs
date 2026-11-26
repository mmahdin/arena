import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync } from 'node:fs';
import {
  CHAPTERS, NARRATIVE, ROUTE_CHAPTER, chapterForRoute,
} from '../src/features/onboarding/theiaNarrative.js';
import {
  MOVEMENTS, PALETTE, SCENES, archGeometry, crescentPath, dawnStarPath, starField,
} from '../src/components/theia/scenes.js';
import { FLOW } from '../src/data.js';

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8');

test('the four chapters follow the myth in order and name their Greek source', () => {
  assert.deepEqual(CHAPTERS.map((chapter) => chapter.numeral), ['I', 'II', 'III', 'IV']);
  assert.deepEqual(CHAPTERS.map((chapter) => chapter.id), ['night', 'dawn', 'sun', 'sight']);
  assert.deepEqual(CHAPTERS.map((chapter) => chapter.myth), ['سِلنه', 'ائوس', 'هلیوس', 'تیا']);
  // Theia is the destination, not the opening: sight comes last.
  assert.equal(CHAPTERS.at(-1).id, 'sight');
  // The artwork set is exactly the movements plus the prologue/privacy/return/finale.
  assert.deepEqual(Object.keys(SCENES).sort(),
    ['dawn', 'night', 'ready', 'return', 'seal', 'sight', 'sun', 'welcome']);
  assert.deepEqual(MOVEMENTS.map((movement) => movement.scene), ['night', 'dawn', 'sun', 'sight']);
});

test('every chapter covers real onboarding routes and no route is orphaned', () => {
  const onboardable = FLOW.filter((route) => route !== 'home' && !['testimonials', 'research', 'experts'].includes(route));
  for (const route of onboardable) {
    assert.ok(route in ROUTE_CHAPTER, `${route} has no assigned chapter`);
  }
  for (const [route, id] of Object.entries(ROUTE_CHAPTER)) {
    if (id === null) continue;
    assert.ok(CHAPTERS.some((chapter) => chapter.id === id), `${route} points at unknown chapter ${id}`);
  }
  // The welcome and the privacy consent are prologue: outside the four chapters
  // on purpose, so nobody reads consent as part of a story.
  assert.equal(ROUTE_CHAPTER.welcome, null);
  assert.equal(ROUTE_CHAPTER.privacy, null);
  assert.equal(chapterForRoute('ready').id, 'sight');
  assert.equal(chapterForRoute('simple-intro').id, 'night');
  assert.equal(chapterForRoute('nope'), null);
});

test('narrative copy exists for every chapter and never promises a cure', () => {
  for (const key of ['welcome', 'privacy', 'simpleIntro', 'deeperIntro', 'knowIntro', 'almostIntro', 'ready', 'checkins', 'name']) {
    const copy = NARRATIVE[key];
    assert.ok(copy, `missing narrative copy: ${key}`);
    assert.ok(copy.title && copy.title.length > 4, `${key} needs a title`);
    assert.ok(copy.body && copy.body.length > 20, `${key} needs a body`);
  }
  const allCopy = Object.values(NARRATIVE).map((copy) => `${copy.title} ${copy.body} ${copy.myth || ''}`).join('\n');
  for (const forbidden of ['درمان قطعی', 'شفا', 'تضمین', 'همیشه خوب', '!!!', '😀', '❤']) {
    assert.ok(!allCopy.includes(forbidden), `narrative copy contains ${forbidden}`);
  }
  // The myth is offered as a metaphor, never as doctrine.
  assert.match(NARRATIVE.welcome.body, /اسطوره/);
  assert.doesNotMatch(allCopy, /خدا|الهه را بپرست|دعا/);
});

test('the privacy copy keeps the existing, accurate storage and AI disclosure', () => {
  const body = `${NARRATIVE.privacy.body} ${NARRATIVE.ready.note}`;
  assert.match(body, /سرور/);
  assert.match(body, /هوش مصنوعی/);
  assert.match(body, /پس بگیری/);
});

test('scene geometry is deterministic, closed and safe at small sizes', () => {
  const a = JSON.stringify(SCENES.night());
  const b = JSON.stringify(SCENES.night());
  assert.equal(a, b, 'scene output must not depend on Math.random');

  // Deterministic star field, and every star inside the 240x300 frame.
  assert.deepEqual(starField({ seed: 3 }), starField({ seed: 3 }));
  assert.notDeepEqual(starField({ seed: 3 }), starField({ seed: 4 }));
  for (const star of starField({ count: 60, seed: 9 })) {
    assert.ok(star.x >= 0 && star.x <= 240 && star.y >= 0 && star.y <= 300, 'star escaped the frame');
    assert.ok(star.r > 0.4 && star.r < 1.6, 'star radius out of scale');
    assert.ok(star.o > 0.1 && star.o <= 0.62, 'star opacity out of scale');
  }

  // The arch is a closed, two-centred path that actually narrows toward the apex.
  const arch = archGeometry({ half: 56, spring: 216, d: 32, base: 268 });
  assert.match(arch.path, /^M[\d.]+ 268L/);
  assert.ok(arch.apexY < arch.spring, 'apex must sit above the spring line');
  assert.ok(arch.radius > 56, 'a two-centred arch has a radius larger than its half-span');
  assert.equal(arch.left, 64);
  assert.equal(arch.right, 176);

  // A crescent must be a crescent: two arcs, and a span narrower than the disc.
  const crescent = crescentPath({ cx: 120, cy: 176, radius: 30, inner: 27.5, offset: 11 });
  assert.match(crescent, /^M[\d. -]+A30 30 0 1 0/);
  assert.match(crescent, /A27.5 27.5 0 0 1/);

  // The dawn star has eight alternating points regardless of scale.
  assert.equal((dawnStarPath(0, 0, 10).match(/L/g) || []).length, 7);
  assert.ok(dawnStarPath(0, 0, 10).endsWith('Z'));
});

test('the palette stays a night-and-gold system with legible contrast', () => {
  const hex = (value) => value.replace('#', '').match(/../g).map((pair) => parseInt(pair, 16));
  const luminance = (value) => {
    const [r, g, b] = hex(value).map((channel) => {
      const c = channel / 255;
      return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
    });
    return 0.2126 * r + 0.7152 * g + 0.0722 * b;
  };
  const ratio = (a, b) => {
    const [light, dark] = [luminance(a), luminance(b)].sort((x, y) => y - x);
    return (light + 0.05) / (dark + 0.05);
  };
  // Body copy is cream on night; the CTA is ink on gold-light. Both must clear
  // WCAG AA for normal text, because this is a health product read at 2am.
  assert.ok(ratio(PALETTE.cream, PALETTE.night) > 12, 'cream on night must be very high contrast');
  assert.ok(ratio('#201a12', PALETTE.goldLight) > 9, 'CTA label on gold must be high contrast');
  assert.ok(ratio(PALETTE.gold, PALETTE.night) > 5, 'gold on night must clear AA');
  // Nothing in the identity is a bright pastel surface.
  for (const value of Object.values(PALETTE)) {
    assert.ok(luminance(value) < 0.95);
  }
});

test('onboarding renders the narrative, not figure illustrations', () => {
  const screen = read('src/screens/Onboarding.jsx');
  for (const token of ['TheiaScene', 'ChapterRail', 'ChapterHead', 'NARRATIVE', 'chapterForRoute']) {
    assert.ok(screen.includes(token), `Onboarding is missing ${token}`);
  }
  for (const dead of ['<img src="/brand/theia-welcome.png"', 'theia-story-', 'theia-ready.png', 'theia-checkins.png']) {
    assert.ok(!screen.includes(dead), `Onboarding still references ${dead}`);
  }
  // Privacy consent must remain a real, labelled switch on its own screen.
  assert.match(screen, /role="switch"/);
  assert.match(screen, /aria-checked=\{sharing\}/);
});

test('the stylesheet defines the night surface, chapter rail and reduced motion', () => {
  const css = read('src/styles.css');
  assert.match(css, /\.app-shell\[data-screen-surface="night"\]/);
  assert.match(css, /\.chapter-rail \{/);
  assert.match(css, /\.theia-scene\.is-animated \.theia-star/);
  assert.match(css, /@media \(prefers-reduced-motion: reduce\)[\s\S]{0,400}\.theia-aurora/);
  // Motion is a single vocabulary: no ad-hoc keyframes on chapter screens.
  const keyframes = [...css.matchAll(/@keyframes ([a-z-]+)/g)].map((match) => match[1]);
  for (const name of ['theia-star-breathe', 'theia-glow-breathe', 'theia-drift', 'theia-rise', 'theia-ripple']) {
    assert.ok(keyframes.includes(name), `missing keyframes: ${name}`);
  }
});

test('the narrative smoke script is wired and exercises every chapter', () => {
  const pkg = JSON.parse(read('package.json'));
  assert.equal(pkg.scripts['test:narrative'], 'node scripts/onboarding-narrative-smoke.mjs');
  assert.match(pkg.scripts['brand:build'], /build-brand-assets\.mjs/);
  assert.match(pkg.scripts['brand:build'], /render-theia-scenes\.mjs --write-brand/);
  const smoke = read('scripts/onboarding-narrative-smoke.mjs');
  for (const marker of ['chapter-screen--', 'theia-scene', 'chapter-rail', 'noteRoute', 'privacy-screen [role="switch"]']) {
    assert.ok(smoke.includes(marker), `smoke test lost ${marker}`);
  }
  assert.match(smoke, /JSDOM/);
});

test('generated brand assets are committed and reproducible from geometry', () => {
  const brandIcons = readdirSync(new URL('../public/brand/icons', import.meta.url));
  for (const file of ['theia-icon-192.png', 'theia-icon-512.png', 'theia-icon-maskable.png', 'icon.svg']) {
    assert.ok(brandIcons.includes(file), `missing generated icon ${file}`);
  }
  assert.match(read('public/favicon.svg'), /<svg[\s\S]*320 320|<svg/);
  const buildScript = read('scripts/build-brand-assets.mjs');
  assert.match(buildScript, /from '\.\.\/src\/components\/theia\/scenes\.js'/);
  const reviewScript = read('scripts/render-theia-scenes.mjs');
  assert.match(reviewScript, /SCENES/);
  assert.match(reviewScript, /--write-brand/);
});
