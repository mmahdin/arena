import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import {
  AI_TRANSPARENCY,
  APERTURE_NOTE,
  EARTHSHINE_ROUTES,
  SCENES,
  SCENE_BY_ROUTE,
  SCENE_COUNT,
  sceneById,
  sceneForRoute,
  toneForRoute,
} from '../src/features/onboarding/earthshine.js';
import { FLOW, QUESTIONS } from '../src/data.js';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

test('eight scenes cover the preserved flow once, in the original order', () => {
  assert.equal(SCENES.length, 8);
  assert.equal(SCENE_COUNT, 8);
  assert.deepEqual(EARTHSHINE_ROUTES, FLOW.filter((route) => route !== 'home'));
  assert.equal(new Set(EARTHSHINE_ROUTES).size, EARTHSHINE_ROUTES.length);
  for (const route of Object.keys(SCENE_BY_ROUTE)) {
    assert.ok(FLOW.includes(route), `${route} is not part of the flow`);
  }
  // Every question and every form step belongs to exactly one scene.
  for (const route of FLOW) {
    if (route === 'home') continue;
    assert.ok(sceneForRoute(route), `No scene carries ${route}`);
  }
  for (const route of Object.keys(QUESTIONS)) {
    assert.ok(sceneForRoute(route), `Question ${route} lost its scene`);
  }
  assert.equal(sceneById(1).routes[0], 'welcome');
  assert.equal(sceneById(8).routes[0], 'ready');
  assert.equal(toneForRoute('welcome'), 'night');
  assert.equal(toneForRoute('ready'), 'day');
});

test('the eight titles and the plain disclosure carry the requested copy', () => {
  const titles = SCENES.map((scene) => scene.title);
  assert.deepEqual(titles, [
    'تیا را نمی‌بینی؛ با تیا می‌بینی.',
    'چه‌قدر بگویی، انتخاب توست.',
    'از تاریکی شروع می‌کنیم.',
    'آن‌چه تاریک به نظر می‌رسد، خالی نیست.',
    'سپیده یک‌باره نمی‌آید.',
    'سه نور؛ و هیچ‌کدام همیشگی نیست.',
    'اگر بخواهی، یک نور روشن می‌ماند.',
    'از این‌جا شروع می‌کنیم.',
  ]);
  for (const scene of SCENES) {
    assert.ok(scene.title.length > 4, `Scene ${scene.id} lost its title`);
    assert.ok(scene.body.length > 40, `Scene ${scene.id} lost its body`);
    assert.ok(!/[\u0000-\u0008]/.test(scene.body));
  }
  // Earthshine is the argument of the fourth scene, in its own words.
  assert.match(sceneById(4).body, /نوری که از زمین به آن می‌رسد/);
  assert.match(sceneById(4).body, /چیزی به تو اضافه نمی‌کند/);
  // AI transparency is plain and separate from the poetry.
  assert.match(AI_TRANSPARENCY.text, /هوش مصنوعی/);
  assert.match(AI_TRANSPARENCY.text, /درمانگر/);
  assert.match(AI_TRANSPARENCY.text, /جایگزین/);
  assert.ok(AI_TRANSPARENCY.text.length < 120);
  const welcome = read('src/screens/Onboarding.jsx');
  assert.match(welcome, /<Disclosure \/>/);
  assert.match(welcome, /AI_TRANSPARENCY\.text/);
});

test('one subject, one variable, one rationed light', () => {
  // Readability of the ashen disc only ever increases, and only reaches full in
  // the last three scenes.
  const disc = SCENES.map((scene) => scene.earthshine);
  assert.ok(disc[0] <= 0.06, 'the first scene must barely show the disc');
  assert.equal(disc[7], 1);
  for (let i = 1; i < disc.length; i++) {
    assert.ok(disc[i] >= disc[i - 1], `earthshine must not fall back at scene ${i + 1}`);
  }
  // No secondary element appears before its scene: no horizon, sun or Venus in
  // the first four scenes.
  for (const scene of SCENES.slice(0, 4)) {
    assert.equal(scene.horizon, false, `Scene ${scene.id} added a horizon too early`);
    assert.equal(scene.sun, false);
    assert.equal(scene.venus, 0);
  }
  assert.equal(SCENES[5].sun, true, 'the sun may break at scene 6 only');
  assert.ok(SCENES[6].venus > 0 && SCENES[7].venus > 0);
  for (const scene of SCENES) {
    assert.ok(scene.stars <= 8, `Scene ${scene.id} has too many stars`);
  }
  // The last scene returns to the first composition, moon for moon.
  assert.equal(SCENES[7].moon, SCENES[0].moon);
  assert.ok(sceneById(4).moon > sceneById(1).moon, 'scene 4 is the close-up');
  // Warmth is rationed: the story is cool until scene 5.
  assert.deepEqual(SCENES.slice(0, 4).map((scene) => scene.tone), ['night', 'night', 'night', 'night']);
  assert.deepEqual(SCENES.slice(5).map((scene) => scene.tone), ['day', 'day', 'day']);
});

test('the sky is drawn honestly: crescent, ashen disc, grain, no glow filter', () => {
  const sky = read('src/components/TheiaSky.jsx');
  // The unlit part of the Moon is always present, and it is the variable.
  assert.match(sky, /earthshine/);
  assert.match(sky, /discOpacity/);
  assert.match(sky, /MARIA/);
  // Waning morning crescent with an offset, blurred shadow — a soft terminator.
  assert.match(sky, /SHADOW_DISTANCE/);
  assert.match(sky, /feGaussianBlur/);
  assert.match(sky, /mixBlendMode: 'soft-light'/);
  assert.match(sky, /feTurbulence/);
  // Astronomical honesty, not a symbol: no crescent-and-star emblem, no rays,
  // no lens flare, no drop shadow.
  assert.ok(!sky.includes('feDropShadow'), 'no glow filters are allowed');
  assert.ok(!/twinkle|sparkle-animation|star-pulse/i.test(sky));
  // The Moon is the single subject; no person, no architecture, no props.
  assert.ok(!sky.includes('<img'), 'the sky stays vector-only');
  assert.ok(!/building|window frame|rug|person|mascot/i.test(sky));
  // Faint halo is allowed, once, around the bright crescent.
  assert.match(sky, /ob-sky__glare/);
});

test('privacy stays honest: the opening is a symbol, the switch is the control', () => {
  const onboarding = read('src/screens/Onboarding.jsx');
  assert.match(onboarding, /APERTURE_NOTE/);
  assert.match(APERTURE_NOTE, /نماد/);
  assert.ok(!/تصمیم می‌گیرد|کنترل می‌کند|ذخیره نمی‌شود/.test(APERTURE_NOTE));
  // The aperture is local component state; the real preference still comes from
  // the backend store and is written through the existing callback.
  assert.match(onboarding, /const \[aperture, setAperture\] = useState/);
  assert.match(onboarding, /role="switch"/);
  assert.match(onboarding, /onClick=\{onTogglePrivacy\}/);
  assert.match(onboarding, /onRetryPrivacy/);
  assert.match(onboarding, /privacySaving/);
  assert.ok(!/onTogglePrivacy\(aperture/.test(onboarding), 'the symbol must not drive the preference');
});

test('the onboarding surfaces one sky per screen and no retired ring art', () => {
  const onboarding = read('src/screens/Onboarding.jsx');
  const css = read('src/onboarding.css');
  assert.ok(!onboarding.includes('TheiaInstrument'));
  assert.match(onboarding, /<TheiaSky scene=\{scene\}>/);
  assert.match(onboarding, /<SceneLabel scene=\{scene\}/);
  // Eight painted skies, four legacy surfaces, and the horizon only from scene 5.
  for (let id = 1; id <= 8; id++) {
    assert.ok(css.includes(`.ob-sky[data-scene="${id}"]`), `No sky paint for scene ${id}`);
  }
  assert.match(css, /\.ob-sky\[data-scene="5"\] \.ob-sky__horizon/);
  assert.ok(!css.includes('.ob-sky[data-scene="1"] .ob-sky__horizon'));
  assert.ok(!css.includes('theia-instrument'), 'retired ring styles must not return');
  assert.match(css, /@media \(prefers-reduced-motion: reduce\)/);
  assert.match(css, /\.ob-moon,[\s\S]{0,120}animation: none !important/);
});

test('aperture and moon copy never imply clinical claims', () => {
  const story = SCENES.map((scene) => `${scene.title} ${scene.body}`).join(' ');
  for (const claim of ['درمان می‌کند', 'شفا', 'تشخیص قطعی', 'بهبود تضمینی']) {
    assert.ok(!story.includes(claim), `Clinical claim found: ${claim}`);
  }
  assert.match(AI_TRANSPARENCY.text, /نه جایگزین درمانگر/);
});
