/**
 * The words of the onboarding story.
 *
 * Theia (Θεία) is the Titaness of *divine light* — the radiance that existed
 * before the sun and the moon, and the root of *thea*, "sight". Her myth is not
 * about a person who counsels you; it is about the condition that makes seeing
 * possible. So the copy never speaks as a goddess, never promises healing, and
 * never names a diagnosis. It walks one idea across the screens: light does not
 * judge what it reveals, and a room is what makes that safe.
 *
 * Copy rules (keep these when editing):
 *  - one image per screen, taken from the chapter list in `chapters.js`;
 *  - concrete, short Persian sentences, no exclamation marks, no emoji;
 *  - every factual claim about storage or AI processing must stay literal and
 *    match the approved privacy wording — poetry sits beside disclosure, never
 *    instead of it.
 */

/** Required, approved disclosure. Do not soften or embellish. */
export const PRIVACY_DISCLOSURE =
  'گفت‌وگوها و پاسخ‌های آشنایی اولیه روی سرور ذخیره می‌شن. برای پاسخ متناسب با تو، اطلاعات مرتبط به سرویس هوش مصنوعی فرستاده می‌شن.';

export const READY_DISCLOSURE =
  'با شروع گفت‌وگو، پاسخ‌های آشنایی اولیه‌ات ذخیره می‌شن و برای شخصی‌سازی چت در اختیار سرویس هوش مصنوعی قرار می‌گیرن.';

/**
 * One entry per narrative screen. `title` and `lede` are the story; `note` is
 * optional quieter voice; `disclosure` is literal product truth.
 */
export const STORY = Object.freeze({
  welcome: {
    title: 'جایی برای دیدن، پیش از قضاوت',
    lede: 'تیا نام الههٔ نور در اسطورهٔ یونان است؛ نوری که پیش از خورشید و ماه بود.',
    note: 'این نوری نیست که دربارهٔ تو حکم بدهد. تیا ابزار گفت‌وگوست، نه روان‌درمانگر.',
    cta: 'شروع',
  },
  privacy: {
    title: 'پنجره، نه حفره',
    lede: 'مشبکِ این پنجره نور را رد می‌کند و درون را نگه می‌دارد.',
    note: 'می‌توانی هر وقت خواستی اجازهٔ استفاده از گفت‌وگوها را ببندی.',
    disclosure: PRIVACY_DISCLOSURE,
    cta: 'ادامه',
  },
  'simple-intro': {
    title: 'فقط چند سؤال ساده',
    lede: 'آن‌قدری که بفهمم این روزها کجای زندگی‌ات ایستاده‌ای.',
    cta: 'ادامه',
  },
  'deeper-intro': {
    title: 'کمی عمیق‌تر',
    lede: 'چند پرسش کوتاه، تا بدانم چه چیزی بیشتر سنگینی می‌کند.',
    note: 'هر سؤالی که نخواستی جواب بدهی، جواب نمی‌دهی.',
    cta: 'ادامه',
  },
  'know-intro': {
    title: 'خط‌ها که وصل شوند، شکل پیدا می‌شود',
    lede: 'بعضی چیزها فقط وقتی نام می‌گیرند که کسی بشنود.',
    cta: 'ادامه',
  },
  'almost-intro': {
    title: 'لحظه‌ای پیش از روشنایی کامل',
    lede: 'صدا و شیوهٔ حرف زدنم را انتخاب کن. بعد، در باز است.',
    cta: 'ادامه',
  },
  checkins: {
    title: 'یک یادآوری، درست وقتی لازم است',
    lede: 'تیا می‌تواند گاهی حالت را بپرسد. هر وقت بخواهی می‌توانی یادآوری‌ها را خاموش کنی.',
    cta: 'ادامه',
  },
  ready: {
    title: 'در باز است',
    lede: 'از اینجا به بعد، هر گفت‌وگو مالِ خودت است.',
    note: 'نور چیزی را عوض نمی‌کند؛ فقط دیده‌شدن را ممکن می‌کند.',
    disclosure: READY_DISCLOSURE,
    cta: 'شروع گفت‌وگو',
  },
});

/**
 * The line each chapter contributes to the story, shown on its chapter mark.
 * One sentence, no punctuation tricks: it is an inscription, not a slogan.
 */
export const CHAPTER_INSCRIPTION = Object.freeze({
  threshold: 'در بسته است؛ نور پشت آن ایستاده.',
  veil: 'پنجره اجازه می‌دهد ببینی، نه دیده شوی.',
  aperture: 'روزن باز می‌شود و هوا عوض می‌شود.',
  reflection: 'آنچه در آب می‌بینی، همان چیزی است که هست.',
  outline: 'نقطه‌ها بی‌معنا نیستند؛ فقط وصل نشده‌اند.',
  dawnline: 'سپیده با یک خط شروع می‌شود، نه با یک انفجار.',
  morning: 'دیگر لازم نیست منتظر نور بمانی.',
});

export function storyFor(route) {
  return STORY[route] || null;
}
