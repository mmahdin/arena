/**
 * Earthshine — the onboarding story, in one sky.
 *
 * The redesign keeps the whole journey in a single place: no room, no rug, no
 * architecture, so it belongs to no culture and no era. There is one subject
 * (the Moon), one variable (how readable the unlit — earthlit — part of it is)
 * and one warm light, rationed until the last scenes.
 *
 * The physics is the metaphor. Sunlight reflects off the Earth and gently
 * lights the part of the Moon that faces away from the Sun. The Moon has not
 * changed; what was always there has only become visible through reflection.
 * That is "revelation rather than illumination": Theia adds nothing and takes
 * nothing away. She is never drawn — only what she makes visible is.
 *
 * The rising readability of the ashen disc is an artistic narrative device, not
 * an astronomical simulation. The Moon itself is always drawn honestly: a thin
 * waning morning crescent with the earthlit disc beside it, never a bare
 * crescent-and-star emblem.
 *
 * Nothing here performs work: this module is copy plus art direction. Routing,
 * answers, persistence and consent stay in `src/data.js`, `src/screens/
 * Onboarding.jsx` and the profile/API layer.
 */

export const SCENE_COUNT = 8;

/** Plain, separate from the poetic copy: what the product actually is. */
export const AI_TRANSPARENCY = {
  lead: 'دربارهٔ تیا',
  text: 'تیا با هوش مصنوعی کار می‌کند؛ همراهی برای گفت‌وگو، نه جایگزین درمانگر حرفه‌ای.',
};

/** The privacy window is a symbol. Say so, once, in plain words. */
export const APERTURE_NOTE = 'این پنجره فقط نماد است؛ تنظیم واقعیِ اشتراک‌گذاری، کلید پایین است.';

/**
 * Eight scenes. `routes` is the part of the preserved 26-step flow each scene
 * carries; `earthshine` drives how readable the ashen disc is; `halo` is the
 * faint glare of the crescent; `stars`, `horizon`, `sun` and `venus` are the
 * only secondary elements the whole story is allowed to introduce.
 */
export const SCENES = [
  {
    id: 1,
    name: 'آغاز',
    tone: 'night',
    routes: ['welcome'],
    earthshine: 0.04,
    halo: 0.6,
    stars: 0,
    moon: 136,
    horizon: false,
    sun: false,
    venus: 0,
    title: 'تیا را نمی‌بینی؛ با تیا می‌بینی.',
    body: 'در اسطوره‌ها، تیا الههٔ نور و دیدن بود. این‌جا چهره‌ای ندارد؛ همراهی است که کمک می‌کند آن‌چه از پیش در توست، روشن‌تر دیده شود.',
  },
  {
    id: 2,
    name: 'حریم خصوصی',
    tone: 'night',
    routes: ['privacy'],
    earthshine: 0.05,
    halo: 0.55,
    stars: 0,
    moon: 132,
    horizon: false,
    sun: false,
    venus: 0,
    aperture: true,
    title: 'چه‌قدر بگویی، انتخاب توست.',
    body: 'فقط تو تعیین می‌کنی چه‌قدر از خودت بگویی. برای اینکه پاسخ‌ها به تو نزدیک‌تر باشند، گفت‌وگوها و پاسخ‌های آشنایی اولیه روی سرور ذخیره می‌شوند و اطلاعات مرتبط به سرویس هوش مصنوعی فرستاده می‌شود.',
  },
  {
    id: 3,
    name: 'شب',
    tone: 'night',
    routes: ['simple-intro', 'name', 'age', 'identity', 'therapist'],
    earthshine: 0.2,
    halo: 0.44,
    stars: 7,
    moon: 140,
    horizon: false,
    sun: false,
    venus: 0,
    title: 'از تاریکی شروع می‌کنیم.',
    body: 'نه چون چیزی خراب است؛ چون هر دیدنی از ندیدن شروع می‌شود. لازم نیست از پیش بدانی در تو چه می‌گذرد؛ چشم که عادت کند، چیزها کم‌کم پیدا می‌شوند.',
  },
  {
    id: 4,
    name: 'دیدن',
    tone: 'night',
    routes: ['deeper-intro', 'mood', 'concerns', 'focus', 'feelings'],
    earthshine: 0.5,
    halo: 0.3,
    stars: 3,
    moon: 176,
    horizon: false,
    sun: false,
    venus: 0,
    title: 'آن‌چه تاریک به نظر می‌رسد، خالی نیست.',
    body: 'بخشِ تاریکِ ماه هم کمی روشن است: نوری که از زمین به آن می‌رسد، به چشم تو برمی‌گردد. تیا چیزی به تو اضافه نمی‌کند و چیزی از تو نمی‌گیرد؛ فقط کمک می‌کند آن‌چه هست واضح‌تر دیده شود، حتی وقتی دیدنش آسان نیست.',
  },
  {
    id: 5,
    name: 'سپیده',
    tone: 'night',
    routes: ['know-intro', 'faith', 'neurodivergent', 'relationships', 'interests', 'strengths', 'barriers', 'recharge'],
    earthshine: 0.7,
    halo: 0.32,
    stars: 2,
    moon: 156,
    horizon: true,
    falseDawn: true,
    sun: false,
    venus: 0,
    title: 'سپیده یک‌باره نمی‌آید.',
    body: 'کم‌کم روشن می‌شود، آن‌قدر آرام که اول متوجهش نمی‌شوی. گاهی نوری می‌آید و دوباره پس می‌رود؛ آن هم بخشی از راه است. حال آدم هم اغلب همین‌طور عوض می‌شود.',
  },
  {
    id: 6,
    name: 'سه نور',
    tone: 'day',
    routes: ['almost-intro', 'approach', 'voice'],
    earthshine: 1,
    halo: 0.24,
    stars: 0.6,
    moon: 160,
    horizon: true,
    sun: true,
    venus: 0,
    title: 'سه نور؛ و هیچ‌کدام همیشگی نیست.',
    body: 'خورشید، ماه و سپیده، هر سه فرزندانِ تیا بودند: سه حالتِ یک نور. حال‌های تو هم همین‌طورند: می‌آیند و می‌روند، و هیچ‌کدام تمامِ تو نیست.',
  },
  {
    id: 7,
    name: 'یادآوری',
    tone: 'day',
    routes: ['checkins', 'referral'],
    earthshine: 1,
    halo: 0.2,
    stars: 0,
    moon: 148,
    horizon: true,
    sun: false,
    venus: 1,
    title: 'اگر بخواهی، یک نور روشن می‌ماند.',
    body: 'تیا می‌تواند گاهی سراغ حالت را بگیرد، بی‌اصرار. هر وقت بخواهی می‌توانی یادآوری‌ها را خاموش کنی.',
  },
  {
    id: 8,
    name: 'آغاز مسیر',
    tone: 'day',
    routes: ['ready'],
    earthshine: 1,
    halo: 0.12,
    stars: 0,
    moon: 136,
    horizon: true,
    sun: false,
    venus: 0.4,
    title: 'از این‌جا شروع می‌کنیم.',
    body: 'آسمان همان است؛ فقط چشمت عادت کرده. پاسخ‌های آشنایی اولیه‌ات ذخیره می‌شوند و برای شخصی‌سازی گفت‌وگو در اختیار سرویس هوش مصنوعی قرار می‌گیرند.',
  },
];

const BY_ROUTE = new Map();
for (const scene of SCENES) {
  for (const route of scene.routes) {
    if (BY_ROUTE.has(route)) throw new Error(`Route ${route} belongs to two Earthshine scenes`);
    BY_ROUTE.set(route, scene);
  }
}

export const SCENE_BY_ROUTE = Object.fromEntries(BY_ROUTE);

/** Routes that belong to the narrated onboarding (legacy info pages are not). */
export const EARTHSHINE_ROUTES = [...BY_ROUTE.keys()];

export function sceneForRoute(route) {
  return BY_ROUTE.get(route) || null;
}

export function sceneById(id) {
  return SCENES.find((scene) => scene.id === id) || null;
}

/** The first scene on or before `route`, used to tone the shell during entry. */
export function toneForRoute(route) {
  return sceneForRoute(route)?.tone || 'night';
}
