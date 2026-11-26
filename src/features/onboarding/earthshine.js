/**
 * Earthshine — The Last Hours of the Night.
 *
 * One pre-dawn, under one waning crescent. The eight plates are not a set of
 * unrelated illustrations: they are successive readings of the same held Moon,
 * the same eastern horizon and the same cool night. The "progress bar" is the
 * legibility of the ashen disc, from almost invisible to fully readable, while
 * warm light is rationed to one small lantern and the final sunrise.
 *
 * Theia is never an object in the frame. The imagery is the seeing: a natural
 * Moon whose unlit disc becomes visible as the eye adapts. Routing, answers,
 * server progress and privacy controls live elsewhere and are unchanged here.
 */

const ROOT = '/brand/earthshine';

export const AI_DISCLOSURE =
  'تیا یک همراه گفت‌وگوی مبتنی بر هوش مصنوعی است؛ درمانگر نیست و جایگزین کمک تخصصی محسوب نمی‌شود.';

export const SCENES = [
  {
    id: 'welcome',
    name: 'آغاز',
    numeral: '۱',
    image: `${ROOT}/01-welcome.jpg`,
    alt: 'آسمانی نزدیک به سیاه پیش از سپیده؛ هلال باریکی در شرق و قرص خاکستری ماه که تقریباً دیده نمی‌شود.',
    routes: ['welcome'],
    legibility: 0.04,
    focus: '50% 42%',
    sky: '#05080d',
    scrimTop: 'rgb(3 7 12 / 2%)',
    scrimMid: 'rgb(3 7 12 / 30%)',
    scrimBottom: 'rgb(3 7 12 / 88%)',
  },
  {
    id: 'privacy',
    name: 'حریم خصوصی',
    numeral: '۲',
    image: `${ROOT}/02-privacy.jpg`,
    alt: 'همان آسمان از پشت یک روزنهٔ نیمه‌باز؛ هلال ماه از شکافی که خودت تعیین می‌کنی دیده می‌شود.',
    routes: ['privacy'],
    legibility: 0.1,
    focus: '50% 38%',
    sky: '#080c12',
    scrimTop: 'rgb(3 7 12 / 8%)',
    scrimMid: 'rgb(3 7 12 / 40%)',
    scrimBottom: 'rgb(3 7 12 / 90%)',
  },
  {
    id: 'night',
    name: 'شب',
    numeral: '۳',
    image: `${ROOT}/03-night.jpg`,
    alt: 'آسمان ژرف لاجوردی پر از ستاره‌های کم‌نور، هلالی کمی بالاتر و نقطهٔ گرم بسیار دوری بر افق.',
    routes: ['simple-intro', 'name', 'age', 'identity', 'therapist'],
    legibility: 0.2,
    focus: '50% 42%',
    sky: '#0a1220',
    scrimTop: 'rgb(4 9 17 / 4%)',
    scrimMid: 'rgb(4 9 17 / 38%)',
    scrimBottom: 'rgb(3 7 12 / 90%)',
  },
  {
    id: 'seeing',
    name: 'دیدن',
    numeral: '۴',
    image: `${ROOT}/04-seeing.jpg`,
    alt: 'ماه نزدیک‌تر از همیشه؛ هلال سفید سرد، قرص خاکستری آبی و دریاواره‌های تیرهٔ سطح ماه کم‌کم خوانا می‌شوند.',
    routes: ['deeper-intro', 'mood', 'concerns', 'focus', 'feelings'],
    legibility: 0.5,
    focus: '50% 40%',
    sky: '#09101a',
    scrimTop: 'rgb(4 9 17 / 4%)',
    scrimMid: 'rgb(4 9 17 / 38%)',
    scrimBottom: 'rgb(3 7 12 / 90%)',
  },
  {
    id: 'dawn',
    name: 'سپیده',
    numeral: '۵',
    image: `${ROOT}/05-dawn.jpg`,
    alt: 'نوار باریک صورتی‌خاکستری و طلایی بر افق، سپیدهٔ کاذب محو شونده و جزئیات ساحل که از تاریکی بیرون می‌آیند.',
    routes: ['know-intro', 'faith', 'neurodivergent', 'relationships', 'interests', 'strengths', 'barriers', 'recharge'],
    legibility: 0.7,
    focus: '50% 44%',
    sky: '#101826',
    scrimTop: 'rgb(5 10 18 / 8%)',
    scrimMid: 'rgb(5 10 18 / 46%)',
    scrimBottom: 'rgb(3 7 12 / 92%)',
  },
  {
    id: 'three-lights',
    name: 'سه نور',
    numeral: '۶',
    image: `${ROOT}/06-three-lights.jpg`,
    alt: 'عریض‌ترین قاب شب: نخستین لبهٔ خورشید بر افق، سپیدهٔ گسترده و ماهی با قرص خاکستری کاملاً خوانا، هر سه با هم.',
    routes: ['almost-intro', 'approach', 'voice'],
    legibility: 1,
    focus: '50% 44%',
    sky: '#111b2a',
    scrimTop: 'rgb(5 10 18 / 8%)',
    scrimMid: 'rgb(5 10 18 / 48%)',
    scrimBottom: 'rgb(3 7 12 / 92%)',
  },
  {
    id: 'checkins',
    name: 'یادآوری',
    numeral: '۷',
    image: `${ROOT}/07-checkins.jpg`,
    alt: 'فانوس کوچکی با شعلهٔ کهربایی در پیش‌زمینه، تنها نور گرم ساخت دست انسان؛ ماه همچنان بالای آسمان است.',
    routes: ['checkins', 'referral'],
    legibility: 1,
    focus: '50% 46%',
    sky: '#121c2a',
    scrimTop: 'rgb(5 10 18 / 8%)',
    scrimMid: 'rgb(5 10 18 / 48%)',
    scrimBottom: 'rgb(3 7 12 / 92%)',
  },
  {
    id: 'first-light',
    name: 'آغاز مسیر',
    numeral: '۸',
    image: `${ROOT}/08-first-light.jpg`,
    alt: 'بازگشت به قاب آغازین، این‌بار در روشنایی آغازین صبح؛ افق باز، فانوسی تقریباً بی‌نیاز و ماهی که هنوز دیده می‌شود.',
    routes: ['ready'],
    legibility: 1,
    focus: '50% 44%',
    sky: '#26313d',
    scrimTop: 'rgb(8 12 18 / 6%)',
    scrimMid: 'rgb(6 10 16 / 46%)',
    scrimBottom: 'rgb(4 7 11 / 90%)',
  },
];

export const CHAPTERS = SCENES.map(({ name, routes }) => ({ name, routes }));

export const NARRATIVE = {
  welcome: {
    chapter: 'آغاز',
    title: 'تیا را نمی‌بینی؛ با تیا می‌بینی.',
    body: 'در اسطوره‌ها، تیا الههٔ نور و دیدن بود. این‌جا چهره‌ای ندارد؛ همراهی است که کمک می‌کند آن‌چه از پیش در توست، روشن‌تر دیده شود.',
    disclosure: AI_DISCLOSURE,
  },
  privacy: {
    chapter: 'حریم خصوصی',
    title: 'چه‌قدر بگویی، انتخاب توست.',
    body: 'فقط تو تعیین می‌کنی چه‌قدر از خودت بگویی. برای اینکه پاسخ‌ها به تو نزدیک‌تر باشند، گفت‌وگوها و پاسخ‌های آشنایی اولیه روی سرور ذخیره می‌شوند و اطلاعات مرتبط به سرویس هوش مصنوعی فرستاده می‌شود.',
  },
  'simple-intro': {
    chapter: 'شب',
    title: 'از تاریکی شروع می‌کنیم.',
    body: 'نه چون چیزی خراب است؛ چون هر دیدنی از ندیدن شروع می‌شود. لازم نیست از پیش بدانی در تو چه می‌گذرد؛ چشم که عادت کند، چیزها کم‌کم پیدا می‌شوند.',
  },
  'deeper-intro': {
    chapter: 'دیدن',
    title: 'آن‌چه تاریک به نظر می‌رسد، خالی نیست.',
    body: 'بخشِ تاریکِ ماه هم کمی روشن است: نوری که از زمین به آن می‌رسد، به چشم تو برمی‌گردد. تیا چیزی به تو اضافه نمی‌کند و چیزی از تو نمی‌گیرد؛ فقط کمک می‌کند آن‌چه هست واضح‌تر دیده شود، حتی وقتی دیدنش آسان نیست.',
  },
  'know-intro': {
    chapter: 'سپیده',
    title: 'سپیده یک‌باره نمی‌آید.',
    body: 'کم‌کم روشن می‌شود، آن‌قدر آرام که اول متوجهش نمی‌شوی. گاهی نوری می‌آید و دوباره پس می‌رود؛ آن هم بخشی از راه است. حال آدم هم اغلب همین‌طور عوض می‌شود.',
  },
  'almost-intro': {
    chapter: 'سه نور',
    title: 'سه نور؛ و هیچ‌کدام همیشگی نیست.',
    body: 'خورشید، ماه و سپیده، هر سه فرزندانِ تیا بودند: سه حالتِ یک نور. حال‌های تو هم همین‌طورند: می‌آیند و می‌روند، و هیچ‌کدام تمامِ تو نیست.',
  },
  checkins: {
    chapter: 'یادآوری',
    title: 'اگر بخواهی، چراغ روشن می‌ماند.',
    body: 'تیا می‌تواند گاهی سراغ حالت را بگیرد، بی‌اصرار. هر وقت بخواهی می‌توانی یادآوری‌ها را خاموش کنی.',
  },
  ready: {
    chapter: 'آغاز مسیر',
    title: 'از این‌جا شروع می‌کنیم.',
    body: 'آسمان همان است؛ فقط چشمت عادت کرده. پاسخ‌های آشنایی اولیه‌ات ذخیره می‌شوند و برای شخصی‌سازی گفت‌وگو در اختیار سرویس هوش مصنوعی قرار می‌گیرند.',
  },
};

export const NARRATIVE_ROUTES = new Set(Object.keys(NARRATIVE));

/** The lunar seas stay a quiet, hidden vocabulary — never a gamified map. */
export const MOOD_SEAS = [
  { name: 'دریای بحران‌ها', x: -7, y: 3 },
  { name: 'دریای آرامش', x: 6, y: -4 },
  { name: 'دریای آسایش', x: -2, y: -8 },
  { name: 'دریای ابرها', x: 8, y: 7 },
  { name: 'دریای سرما', x: -5, y: -2 },
];

export function sceneForRoute(route) {
  return SCENES.find((scene) => scene.routes.includes(route)) || SCENES[0];
}

export function sceneIndexForRoute(route) {
  const index = SCENES.findIndex((scene) => scene.routes.includes(route));
  return index < 0 ? 0 : index;
}

export function sceneStyle(scene) {
  return {
    '--es-image': `url("${scene.image}")`,
    '--es-focus': scene.focus,
    '--es-sky': scene.sky,
    '--es-legibility': String(scene.legibility),
    '--es-segment-width': `${(8 + Number(scene.legibility) * 13).toFixed(1)}px`,
    '--es-scrim-top': scene.scrimTop,
    '--es-scrim-mid': scene.scrimMid,
    '--es-scrim-bottom': scene.scrimBottom,
  };
}
