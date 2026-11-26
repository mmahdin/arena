import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';
import { SCENES, ARCH, GLYPH } from '../src/features/onboarding/artwork.js';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

test('app icon, manifest and installable branding still exist', () => {
  const assets = [
    'public/favicon.svg',
    'public/manifest.webmanifest',
    'public/brand/theia-logo.svg',
    'public/brand/theia-icon-192.png',
    'public/brand/theia-icon-512.png',
    'public/brand/theia-touch-icon.png',
  ];
  for (const file of assets) {
    const url = new URL(file, root);
    assert.ok(existsSync(url), `Missing brand asset: ${file}`);
    assert.ok(statSync(url).size > 200, `Empty or invalid brand asset: ${file}`);
  }
});

test('HTML head and manifest use the night surface and the new mark', () => {
  const html = read('index.html');
  assert.match(html, /theme-color" content="#0b0c11"/);
  assert.match(html, /apple-touch-icon" href="\/brand\/theia-touch-icon\.png"/);
  assert.doesNotMatch(html, /photo_2026-11-12/);
  const manifest = JSON.parse(read('public/manifest.webmanifest'));
  assert.equal(manifest.theme_color, '#0b0c11');
  assert.equal(manifest.background_color, '#0b0c11');
  assert.ok(manifest.icons.some((icon) => icon.src.endsWith('theia-icon-512.png')));
});

test('the identity is non-figurative: no portrait or goddess illustration ships in the UI', () => {
  // The retired photographic portrait must not be referenced anywhere in the app.
  const activeSource = [
    read('src/components/UI.jsx'),
    read('src/components/TheiaBrand.jsx'),
    read('src/components/TheiaArt.jsx'),
    read('src/screens/Onboarding.jsx'),
    read('src/screens/Main.jsx'),
    read('src/screens/Letters.jsx'),
    read('src/features/devsession/DevLogin.jsx'),
  ].join('\n');

  for (const retired of [
    'photo_2026-11-12_19-20-10',
    '/brand/theia-welcome.png',
    '/brand/theia-story-1.png',
    '/brand/theia-story-2.png',
    '/brand/theia-story-3.png',
    '/brand/theia-story-4.png',
    '/brand/theia-ready.png',
    '/brand/theia-privacy.png',
    '/brand/theia-checkins.png',
  ]) {
    assert.ok(!activeSource.includes(retired), `Retired figurative asset still referenced: ${retired}`);
  }
  // The illustration system is drawn from shared geometry, not images.
  assert.match(activeSource, /TheiaScene/);
  assert.match(activeSource, /TheiaGlyph/);
  assert.ok(!/theia-(welcome|story|ready|privacy|checkins)/.test(read('src/screens/Onboarding.jsx')));
});

test('the mark is an arch, a horizon and one point of light', () => {
  assert.match(GLYPH.arch, /^M50 12 C/);
  assert.match(GLYPH.horizon, /^M12 64 H88$/);
  assert.equal(GLYPH.disc.x, 50);
  assert.equal(GLYPH.disc.y, 64, 'the light must sit on the horizon');
  assert.ok(GLYPH.disc.y > 40 && GLYPH.disc.y < 84, 'the light sits inside the arch');
});

test('the narrative has seven chapters whose light never goes backwards', async () => {
  const { CHAPTERS, assertChapterContinuity, chapterFor, dawnFor } = await import('../src/features/onboarding/chapters.js');
  assert.equal(CHAPTERS.length, 7);
  assert.equal(SCENES.length, 7);
  assert.deepEqual(CHAPTERS.map((chapter) => chapter.stage), [0, 1, 2, 3, 4, 5, 6]);
  for (let i = 1; i < CHAPTERS.length; i++) {
    assert.ok(CHAPTERS[i].dawn > CHAPTERS[i - 1].dawn, `chapter ${CHAPTERS[i].id} must be lighter`);
  }
  const continuity = assertChapterContinuity();
  assert.equal(continuity.ok, true, 'the story must not dim while walking FLOW forward');
  assert.deepEqual(continuity.sections, CHAPTERS.map((chapter) => chapter.id));
  assert.equal(chapterFor('welcome').id, 'threshold');
  assert.equal(chapterFor('ready').id, 'morning');
  assert.equal(dawnFor('ready'), 1);
});

test('every narrative route has one story beat and one honest disclosure where needed', async () => {
  const { STORY, PRIVACY_DISCLOSURE, READY_DISCLOSURE } = await import('../src/features/onboarding/story.js');
  const { NARRATIVE_ROUTES } = await import('../src/features/onboarding/chapters.js');
  assert.deepEqual([...NARRATIVE_ROUTES].sort(), Object.keys(STORY).sort());
  for (const route of NARRATIVE_ROUTES) {
    const beat = STORY[route];
    assert.ok(beat.title.trim().length > 0, route);
    assert.ok(beat.lede.trim().length > 0, route);
    assert.ok(beat.cta.trim().length > 0, route);
    assert.ok(!beat.title.includes('!'), `${route}: no exclamation marks`);
  }
  // The approved data-processing disclosure is preserved word for word.
  assert.equal(STORY.privacy.disclosure, PRIVACY_DISCLOSURE);
  assert.equal(STORY.ready.disclosure, READY_DISCLOSURE);
  assert.match(PRIVACY_DISCLOSURE, /روی سرور ذخیره/);
  assert.match(READY_DISCLOSURE, /سرویس هوش مصنوعی/);
});

test('the narrative copy does not promise clinical outcomes or speak as a deity', async () => {
  const { STORY, CHAPTER_INSCRIPTION } = await import('../src/features/onboarding/story.js');
  const all = [...Object.values(STORY).flatMap((beat) => [beat.title, beat.lede, beat.note || '']), ...Object.values(CHAPTER_INSCRIPTION)].join('\n');
  for (const forbidden of ['درمان می‌کنم', 'شفا', 'تشخیص', 'تضمین', 'من الهه', 'روان‌درمانگر تو']) {
    assert.ok(!all.includes(forbidden), `clinical or devotional claim leaked into copy: ${forbidden}`);
  }
  // The product says plainly that it is a tool, not a therapist.
  assert.match(STORY.welcome.note, /ابزار گفت‌وگو/);
});

test('the onboarding renders chapters instead of portrait images', () => {
  const onboarding = read('src/screens/Onboarding.jsx');
  assert.match(onboarding, /<TheiaScene stage=\{chapter\.stage\}/);
  assert.match(onboarding, /<ChapterMarks chapter=\{chapter\}/);
  // Every narrative route is rendered through the shared chapter screen, or
  // with an explicit chapter lookup when it keeps its own layout.
  assert.match(onboarding, /const INTERSTITIAL_ROUTES = new Set\(\['simple-intro', 'deeper-intro', 'know-intro', 'almost-intro'\]\)/);
  for (const beat of ['welcome', 'privacy', 'checkins', 'ready']) {
    assert.ok(onboarding.includes(`route="${beat}"`), `missing chapter wiring: ${beat}`);
  }
  assert.ok(onboarding.includes("chapterFor('name')"));
  assert.ok(onboarding.includes("chapterFor('voice')"));
  assert.ok(onboarding.includes("chapterFor('approach')"));
  // No <img> illustration remains in the onboarding.
  assert.ok(!/welcome-art[\s\S]{0,200}<img/.test(onboarding));
  assert.ok(!onboarding.includes('interstitial-art'));
});

test('the art is architectural and avoids figurative primitives', () => {
  assert.ok(ARCH.outer.startsWith('M44 288 V152'));
  for (const scene of SCENES) {
    assert.ok(scene.dawn >= 0 && scene.dawn <= 1, scene.id);
    assert.ok(scene.disc.y <= 300 && scene.disc.y >= 0, scene.id);
    assert.ok(scene.horizonY > 150, `${scene.id}: the horizon stays in the lower half`);
  }
  const art = read('src/components/TheiaArt.jsx');
  assert.ok(!/<image\b/.test(art), 'no raster portraits inside the art system');
  assert.match(art, /aria-hidden/);
});
