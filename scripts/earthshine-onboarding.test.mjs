import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { FLOW, QUESTIONS } from '../src/data.js';
import { ONBOARDING_ROUTES } from '../src/features/profile/navigation.js';
import {
  AI_DISCLOSURE,
  CHAPTERS,
  NARRATIVE,
  NARRATIVE_ROUTES,
  SCENES,
  sceneForRoute,
} from '../src/features/onboarding/earthshine.js';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

const EXPECTED_COPY = {
  welcome: {
    title: 'تیا را نمی\u200cبینی؛ با تیا می\u200cبینی.',
    body: 'در اسطوره\u200cها، تیا الههٔ نور و دیدن بود. این\u200cجا چهره\u200cای ندارد؛ همراهی است که کمک می\u200cکند آن\u200cچه از پیش در توست، روشن\u200cتر دیده شود.',
  },
  privacy: {
    title: 'چه\u200cقدر بگویی، انتخاب توست.',
    body: 'فقط تو تعیین می\u200cکنی چه\u200cقدر از خودت بگویی. برای اینکه پاسخ\u200cها به تو نزدیک\u200cتر باشند، گفت\u200cوگوها و پاسخ\u200cهای آشنایی اولیه روی سرور ذخیره می\u200cشوند و اطلاعات مرتبط به سرویس هوش مصنوعی فرستاده می\u200cشود.',
  },
  'simple-intro': {
    title: 'از تاریکی شروع می\u200cکنیم.',
    body: 'نه چون چیزی خراب است؛ چون هر دیدنی از ندیدن شروع می\u200cشود. لازم نیست از پیش بدانی در تو چه می\u200cگذرد؛ چشم که عادت کند، چیزها کم\u200cکم پیدا می\u200cشوند.',
  },
  'deeper-intro': {
    title: 'آن\u200cچه تاریک به نظر می\u200cرسد، خالی نیست.',
    body: 'بخشِ تاریکِ ماه هم کمی روشن است: نوری که از زمین به آن می\u200cرسد، به چشم تو برمی\u200cگردد. تیا چیزی به تو اضافه نمی\u200cکند و چیزی از تو نمی\u200cگیرد؛ فقط کمک می\u200cکند آن\u200cچه هست واضح\u200cتر دیده شود، حتی وقتی دیدنش آسان نیست.',
  },
  'know-intro': {
    title: 'سپیده یک\u200cباره نمی\u200cآید.',
    body: 'کم\u200cکم روشن می\u200cشود، آن\u200cقدر آرام که اول متوجهش نمی\u200cشوی. گاهی نوری می\u200cآید و دوباره پس می\u200cرود؛ آن هم بخشی از راه است. حال آدم هم اغلب همین\u200cطور عوض می\u200cشود.',
  },
  'almost-intro': {
    title: 'سه نور؛ و هیچ\u200cکدام همیشگی نیست.',
    body: 'خورشید، ماه و سپیده، هر سه فرزندانِ تیا بودند: سه حالتِ یک نور. حال\u200cهای تو هم همین\u200cطورند: می\u200cآیند و می\u200cروند، و هیچ\u200cکدام تمامِ تو نیست.',
  },
  checkins: {
    title: 'اگر بخواهی، چراغ روشن می\u200cماند.',
    body: 'تیا می\u200cتواند گاهی سراغ حالت را بگیرد، بی\u200cاصرار. هر وقت بخواهی می\u200cتوانی یادآوری\u200cها را خاموش کنی.',
  },
  ready: {
    title: 'از این\u200cجا شروع می\u200cکنیم.',
    body: 'آسمان همان است؛ فقط چشمت عادت کرده. پاسخ\u200cهای آشنایی اولیه\u200cات ذخیره می\u200cشوند و برای شخصی\u200cسازی گفت\u200cوگو در اختیار سرویس هوش مصنوعی قرار می\u200cگیرند.',
  },
};

test('the storyboard copy is implemented verbatim, with one plain AI disclosure', () => {
  assert.equal(AI_DISCLOSURE, 'تیا یک همراه گفت\u200cوگوی مبتنی بر هوش مصنوعی است؛ درمانگر نیست و جایگزین کمک تخصصی محسوب نمی\u200cشود.');
  assert.equal(Object.keys(NARRATIVE).length, 8);
  assert.equal(NARRATIVE_ROUTES.size, 8);
  for (const [route, expected] of Object.entries(EXPECTED_COPY)) {
    assert.equal(NARRATIVE[route].title, expected.title, `${route} title`);
    assert.equal(NARRATIVE[route].body, expected.body, `${route} body`);
  }
  assert.equal(NARRATIVE.welcome.disclosure, AI_DISCLOSURE);
  for (const route of Object.keys(NARRATIVE)) {
    if (route !== 'welcome') assert.equal(NARRATIVE[route].disclosure, undefined, `${route} must not repeat the disclosure`);
  }
});

test('eight scenes cover every onboarding route exactly once and progress the ashen disc', () => {
  assert.equal(SCENES.length, 8);
  assert.equal(CHAPTERS.length, 8);
  assert.deepEqual(CHAPTERS.map((chapter) => chapter.name), SCENES.map((scene) => scene.name));
  assert.deepEqual(SCENES.map((scene) => scene.id), [
    'welcome', 'privacy', 'night', 'seeing', 'dawn', 'three-lights', 'checkins', 'first-light',
  ]);
  assert.deepEqual(SCENES.map((scene) => scene.legibility), [0.04, 0.1, 0.2, 0.5, 0.7, 1, 1, 1]);
  for (const route of ONBOARDING_ROUTES) {
    const owners = SCENES.filter((scene) => scene.routes.includes(route));
    assert.equal(owners.length, 1, `${route} must belong to exactly one scene`);
  }
  assert.equal(sceneForRoute('ready').id, 'first-light');
  assert.equal(sceneForRoute('referral').id, 'checkins');
  assert.equal(sceneForRoute('voice').id, 'three-lights');
  assert.equal(sceneForRoute('testimonials').id, 'welcome', 'legacy routes fall back to the first plate');
  for (const scene of SCENES) {
    assert.ok(existsSync(new URL(`public${scene.image}`, root)), `Missing plate: ${scene.image}`);
  }
});

test('the new visual system is wired without the retired ring instrument', () => {
  const onboarding = read('src/screens/Onboarding.jsx');
  const devLogin = read('src/features/devsession/DevLogin.jsx');
  const earthshine = read('src/features/onboarding/earthshine.js');
  const css = read('src/earthshine.css');
  const main = read('src/main.jsx');
  const html = read('index.html');

  assert.match(onboarding, /EarthshineScene/);
  assert.match(devLogin, /EarthshineScene/);
  assert.match(main, /import '\.\/earthshine\.css'/);
  assert.match(html, /01-welcome\.jpg/);
  assert.match(css, /@keyframes es-adapt/);
  assert.match(css, /prefers-reduced-motion: reduce/);
  for (const retired of [
    'TheiaInstrument', 'TheiaSigil', 'TheiaMark', 'حلقه کامل می\u200cشود',
    'حلقهٔ باز تیا', 'تیا نور را نمی\u200cسازد', 'حریم تو، مثل این حلقه است',
  ]) {
    assert.ok(!onboarding.includes(retired), `Retired onboarding language remains: ${retired}`);
  }
  assert.ok(!earthshine.includes('قفل') && !earthshine.includes('گنبد'), 'No vault, lock or region-specific architecture language');
  assert.doesNotMatch(devLogin, /TheiaInstrument|TheiaMark|TheiaSigil/);
});

test('all questions, answers, privacy controls and backend completion remain intact', () => {
  const onboarding = read('src/screens/Onboarding.jsx');
  const app = read('src/App.jsx');
  assert.equal(FLOW.length, 27);
  assert.equal(FLOW[0], 'welcome');
  assert.equal(FLOW.at(-1), 'home');
  assert.equal(ONBOARDING_ROUTES.length, 26);
  for (const key of Object.keys(QUESTIONS)) {
    assert.ok(FLOW.includes(key), `Question route missing from FLOW: ${key}`);
    assert.match(onboarding, /QUESTIONS\[route\]/);
  }
  for (const section of [
    'function Welcome(', 'function Testimonials(', 'function Research(', 'function Experts(',
    'function NameScreen(', 'function Interstitial(', 'function Privacy(', 'function AgeScreen(',
    'function QuestionScreen(', 'function MoodScreen(', 'function CheckinsScreen(',
    'function VoiceScreen(', 'function ApproachScreen(', 'function Ready(',
  ]) {
    assert.ok(onboarding.includes(section), section);
  }
  assert.match(onboarding, /onTogglePrivacy/);
  assert.match(onboarding, /onRetryPrivacy/);
  assert.match(app, /onboardingMove\(/);
  assert.match(app, /finishProfile/);
  assert.match(app, /advanceTo\(/);
  assert.match(app, /buildOnboardingAnswers\(answers\)/);
  assert.match(app, /route === 'ready' && target === 'home'/);
});
