import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { EARTHSHINE_SCENES, crescentPath } from '../src/features/onboarding/earthshineModel.js';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');

test('Earthshine scenes implement one Moon, progressive earthshine readability, and rationed horizon warmth', () => {
  const order = ['welcome', 'night', 'seeing', 'dawn', 'threeLights', 'checkins', 'final'];
  let previousEarthshine = 0;
  for (const key of order) {
    const scene = EARTHSHINE_SCENES[key];
    assert.ok(scene, `Missing scene config: ${key}`);
    assert.ok(scene.earthshine >= previousEarthshine, `${key} earthshine (${scene.earthshine}) should not regress below ${previousEarthshine}`);
    previousEarthshine = scene.earthshine;
  }

  // Horizon line enters for the first time in Scene 5 (dawn) and stays through Scene 8.
  for (const noHorizon of ['welcome', 'privacy', 'night', 'seeing']) {
    assert.equal(EARTHSHINE_SCENES[noHorizon].horizon, false, `${noHorizon} must not draw a horizon`);
    assert.equal(Boolean(EARTHSHINE_SCENES[noHorizon].falseDawn), false, `${noHorizon} must withhold warm dawn light`);
    assert.equal(Boolean(EARTHSHINE_SCENES[noHorizon].solarRim), false, `${noHorizon} must withhold solar rim`);
  }
  for (const withHorizon of ['dawn', 'threeLights', 'checkins', 'final']) {
    assert.equal(EARTHSHINE_SCENES[withHorizon].horizon, true, `${withHorizon} must draw the thin horizon`);
  }

  // Stars appear only in Night (<= 8 faint stars), never as emblem constellations.
  assert.ok(EARTHSHINE_SCENES.night.stars.length <= 8 && EARTHSHINE_SCENES.night.stars.length >= 6);
  assert.equal(EARTHSHINE_SCENES.welcome.stars.length, 0);

  // Scene 8 (final) mirrors Scene 1 (welcome) Moon position & scale with reversed civil-twilight values.
  assert.equal(EARTHSHINE_SCENES.final.moonCy, EARTHSHINE_SCENES.welcome.moonCy);
  assert.equal(EARTHSHINE_SCENES.final.moonR, EARTHSHINE_SCENES.welcome.moonR);
  assert.equal(EARTHSHINE_SCENES.final.reversed, true);

  // Venus is placed well below and laterally offset from the Moon so it never forms a crescent-and-star emblem.
  assert.ok(EARTHSHINE_SCENES.checkins.venus > 0.8);
  assert.ok(EARTHSHINE_SCENES.final.venus > 0 && EARTHSHINE_SCENES.final.venus < EARTHSHINE_SCENES.checkins.venus);

  // Astronomical two-arc crescent path closes cleanly.
  const d = crescentPath(180, 120, 34, 0.22);
  assert.match(d, /^M .+ A .+ A .+ Z$/);
});

test('Onboarding renders verbatim Persian copy for all eight Earthshine scenes plus plain AI transparency', () => {
  const onboarding = read('src/screens/Onboarding.jsx');

  const requiredCopy = [
    // 1. Welcome
    'تیا را نمی‌بینی؛ با تیا می‌بینی.',
    'در اسطوره‌ها، تیا الههٔ نور و دیدن بود. این‌جا چهره‌ای ندارد؛ همراهی است که کمک می‌کند آن‌چه از پیش در توست، روشن‌تر دیده شود.',
    // 2. Privacy
    'چه‌قدر بگویی، انتخاب توست.',
    'فقط تو تعیین می‌کنی چه‌قدر از خودت بگویی. برای اینکه پاسخ‌ها به تو نزدیک‌تر باشند، گفت‌وگوها و پاسخ‌های آشنایی اولیه روی سرور ذخیره می‌شوند و اطلاعات مرتبط به سرویس هوش مصنوعی فرستاده می‌شود.',
    // 3. Night
    'از تاریکی شروع می‌کنیم.',
    'نه چون چیزی خراب است؛ چون هر دیدنی از ندیدن شروع می‌شود. لازم نیست از پیش بدانی در تو چه می‌گذرد؛ چشم که عادت کند، چیزها کم‌کم پیدا می‌شوند.',
    // 4. Seeing
    'آن‌چه تاریک به نظر می‌رسد، خالی نیست.',
    'بخشِ تاریکِ ماه هم کمی روشن است: نوری که از زمین به آن می‌رسد، به چشم تو برمی‌گردد. تیا چیزی به تو اضافه نمی‌کند و چیزی از تو نمی‌گیرد؛ فقط کمک می‌کند آن‌چه هست واضح‌تر دیده شود، حتی وقتی دیدنش آسان نیست.',
    // 5. Dawn
    'سپیده یک‌باره نمی‌آید.',
    'کم‌کم روشن می‌شود، آن‌قدر آرام که اول متوجهش نمی‌شوی. گاهی نوری می‌آید و دوباره پس می‌رود؛ آن هم بخشی از راه است. حال آدم هم اغلب همین‌طور عوض می‌شود.',
    // 6. Three Lights
    'سه نور؛ و هیچ‌کدام همیشگی نیست.',
    'خورشید، ماه و سپیده، هر سه فرزندانِ تیا بودند: سه حالتِ یک نور. حال‌های تو هم همین‌طورند: می‌آیند و می‌روند، و هیچ‌کدام تمامِ تو نیست.',
    // 7. Check-ins
    'اگر بخواهی، یک نور روشن می‌ماند.',
    'تیا می‌تواند گاهی سراغ حالت را بگیرد، بی‌اصرار. هر وقت بخواهی می‌توانی یادآوری‌ها را خاموش کنی.',
    // 8. Final Chapter
    'از این‌جا شروع می‌کنیم.',
    'آسمان همان است؛ فقط چشمت عادت کرده. پاسخ‌های آشنایی اولیه‌ات ذخیره می‌شوند و برای شخصی‌سازی گفت‌وگو در اختیار سرویس هوش مصنوعی قرار می‌گیرند.',
    // Plain AI companion transparency
    'تیا یک همراه مبتنی بر هوش مصنوعی است و جایگزین روان‌درمانگر یا متخصص سلامت روان نیست.',
  ];

  for (const line of requiredCopy) {
    assert.ok(onboarding.includes(line), `Missing exact Persian copy: ${line}`);
  }
});

test('Privacy symbolic aperture stays decoupled from backend consent switch, and Check-ins Venus fades slowly', () => {
  const onboarding = read('src/screens/Onboarding.jsx');
  const css = read('src/onboarding.css');

  // Symbolic aperture slider has explicit symbolic label and never calls onTogglePrivacy.
  assert.match(onboarding, /aria-label="تنظیم نمادین گشودگی دریچه در تصویر"/);
  assert.match(onboarding, /onClick=\{onTogglePrivacy\}/);

  // Venus fades slowly (~1.5s) rather than disappearing abruptly when reminders are switched off,
  // while respecting prefers-reduced-motion.
  assert.match(onboarding, /venusVisible=\{remindersOn\}/);
  assert.match(css, /\.es-venus\s*\{\s*transition:\s*opacity\s+1500ms/);
  assert.match(css, /@media\s*\(prefers-reduced-motion:\s*reduce\)/);
});
