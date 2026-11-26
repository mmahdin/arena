/**
 * The onboarding narrative.
 *
 * Theia is the Greek Titaness of divine light *and of sight* — her name is
 * built on thea, "a seeing". She is the mother of the sun (Helios), the moon
 * (Selene) and the dawn (Eos). Her myth is therefore not a story about
 * brightness as decoration; it is a story about how a person becomes able to
 * see their own life clearly, and in what order that becomes possible.
 *
 * That order is the onboarding. Night first — you can only make out edges.
 * Then dawn — the first line of light, which returns the shape of things.
 * Then sun — light that hides nothing, and judges nothing. Then sight — the
 * moment the light and the seeing are the same event.
 *
 * The copy never asks the reader to believe in a goddess, and never uses the
 * myth to promise a cure. The myth is used the way a good metaphor is used:
 * to give a name to a sequence the reader already recognises in themselves.
 * Iranian readers hold two symbolic languages at once — the Greek inheritance
 * they meet in world literature, and the Persian one they live inside — so the
 * imagery is deliberately built where those two languages agree: light, dawn,
 * an arch, a star.
 */

export const NARRATIVE = Object.freeze({
  welcome: {
    eyebrow: 'نور، پیش از آنکه دیده شود',
    title: 'اینجا جایی‌ست برای دیدنِ روشن‌تر.',
    body: 'تیا در اسطورهٔ یونان، الههٔ نور و دیدن است؛ مادر خورشید، ماه و سپیده‌دم. او را به شکل آدم نمی‌کشیم، چون چیزی که می‌خواهی ببینی خودِ او نیست: نوری‌ست که در روزهای سخت گم می‌شود.',
    action: 'شروع',
    caption: 'چهار فصل، چهار جور نور',
  },
  privacy: {
    eyebrow: 'پیش از ورود',
    title: 'آنچه اینجا می‌گویی، همان‌جا می‌ماند.',
    body: 'گفت‌وگوها و پاسخ‌های آشنایی روی سرور ذخیره می‌شن تا هر وقت برگشتی، از همان‌جا ادامه بدی. برای اینکه پاسخ‌ها به تو نزدیک‌تر باشه، متن گفت‌وگو در اختیار سرویس هوش مصنوعی قرار می‌گیره. هر وقت بخوای می‌تونی این اجازه رو پس بگیری.',
    action: 'ادامه',
    caption: 'نور، در مرزِ خودش',
    toggle: 'اجازهٔ استفاده از گفت‌وگوها',
  },
  simpleIntro: {
    eyebrow: 'فصل یکم · شب',
    title: 'در شب، فقط لبهٔ چیزها پیدا است.',
    body: 'قبل از اینکه چیزی روشن شود، باید بدانیم کجاییم. چند سؤال ساده می‌پرسم؛ نه برای قضاوت، فقط برای اینکه بفهمم از کجا با هم شروع کنیم.',
    myth: 'سِلنه، ماه — نوری که شکل نمی‌دهد، فقط نشان می‌دهد چیزی آنجاست.',
    // "ادامه" stays the single advance word for Assistive-tech and E2E stability;
    // the narrative voice lives in the title lines, not in the buttons.
    action: 'ادامه',
  },
  deeperIntro: {
    eyebrow: 'فصل دوم · سپیده',
    title: 'سپیده‌دم اولین چیزی‌ست که برمی‌گردد.',
    body: 'سؤال‌های بعدی کمی عمیق‌ترن. هرچه دقیق‌تر جواب بدی، این نور بهتر روی همان چیزی می‌افتد که این روزها سنگین‌تره.',
    myth: 'ائوس، سپیده‌دم — خطی از نور که به چشم فرصت می‌دهد دوباره ببیند.',
    action: 'ادامه',
  },
  knowIntro: {
    eyebrow: 'فصل سوم · خورشید',
    title: 'روشنایی، چیزی را پنهان نمی‌کند.',
    body: 'این بخش دربارهٔ چیزهایی‌ست که معمولاً دربارهٔ خودمان نمی‌گوییم. لازم نیست کامل یا مرتب باشد. خورشید برای دیدن است، نه برای قضاوت.',
    myth: 'هلیوس، خورشید — نوری که همه‌چیز را همان‌طور که هست نشان می‌دهد.',
    action: 'ادامه',
  },
  almostIntro: {
    eyebrow: 'فصل چهارم · دیدن',
    title: 'حالا نور، مالِ خودت است.',
    body: 'آخرین انتخاب‌ها: با چه صدایی حرف بزنم و چطور کنارت بمانم. بعد از این، دیگر تماشاگر نیستی؛ از اینجا به بعد با هم راه می‌رویم.',
    myth: 'تیا، نورِ بینا — لحظه‌ای که نور و دیدن یکی می‌شوند.',
    action: 'ادامه',
  },
  ready: {
    eyebrow: 'در آستانه',
    title: 'آماده‌ای؟',
    body: 'هر وقت که بخوای، می‌تونی برگردی و ادامه بدی. چیزی از تو عقب نمی‌مونه.',
    action: 'شروع گفت‌وگو',
    note: 'با شروع گفت‌وگو، پاسخ‌های آشنایی اولیه ذخیره می‌شن و برای شخصی‌سازی چت در اختیار سرویس هوش مصنوعی قرار می‌گیرن.',
  },
  checkins: {
    eyebrow: 'نوری که برمی‌گردد',
    title: 'گاهی خودم سراغت می‌آیم.',
    body: 'اگر بخوای، هر چند وقت یک‌بار حال‌ونت رو می‌پرسم؛ نه به‌عنوان یادآوری وظیفه، بلکه مثل نور صبح که هر روز برمی‌گرده. هر وقت خواستی خاموشش کن.',
    action: 'ادامه',
    caption: 'به ریتمِ خودت',
  },
  name: {
    eyebrow: 'قبل از شروع',
    title: 'چطور صدات کنم؟',
    body: 'با همین اسم صدات می‌کنم. هر وقت خواستی عوضش کنی، می‌تونی.',
    placeholder: 'اسمت',
    action: 'ادامه',
  },
});

/**
 * The four chapters, used by the chapter rail on the interstitial screens.
 * `scene` names match the generated artwork in components/theia/scenes.js.
 */
export const CHAPTERS = Object.freeze([
  { id: 'night', numeral: 'I', name: 'شب', myth: 'سِلنه', scene: 'night' },
  { id: 'dawn', numeral: 'II', name: 'سپیده', myth: 'ائوس', scene: 'dawn' },
  { id: 'sun', numeral: 'III', name: 'خورشید', myth: 'هلیوس', scene: 'sun' },
  { id: 'sight', numeral: 'IV', name: 'دیدن', myth: 'تیا', scene: 'sight' },
]);

/** Which chapter a given onboarding route belongs to, for the quiet progress rail. */
export const ROUTE_CHAPTER = Object.freeze({
  welcome: null,
  privacy: null,
  'simple-intro': 'night',
  name: 'night',
  age: 'night',
  identity: 'night',
  therapist: 'night',
  'deeper-intro': 'dawn',
  mood: 'dawn',
  concerns: 'dawn',
  focus: 'dawn',
  feelings: 'dawn',
  'know-intro': 'sun',
  faith: 'sun',
  neurodivergent: 'sun',
  relationships: 'sun',
  interests: 'sun',
  strengths: 'sun',
  barriers: 'sun',
  recharge: 'sun',
  'almost-intro': 'sight',
  approach: 'sight',
  voice: 'sight',
  checkins: 'sight',
  referral: 'sight',
  ready: 'sight',
});

export function chapterForRoute(route) {
  const id = ROUTE_CHAPTER[route];
  return id ? CHAPTERS.find((chapter) => chapter.id === id) || null : null;
}
