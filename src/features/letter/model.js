const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function isNonEmptyString(value, maxLength = 20000) {
  return typeof value === 'string' && value.trim().length > 0 && [...value].length <= maxLength;
}

function isIsoTimestamp(value) {
  return typeof value === 'string' && !Number.isNaN(Date.parse(value));
}

/**
 * Validates a summary item returned by a future `GET /api/v1/me/letters` endpoint.
 */
export function parseLetterSummary(raw) {
  if (!raw || typeof raw !== 'object') throw new Error('Invalid letter summary');
  if (typeof raw.id !== 'string' || !UUID.test(raw.id)) throw new Error('Invalid letter ID');
  if (!isNonEmptyString(raw.title, 160)) throw new Error('Invalid letter title');
  if (typeof raw.excerpt !== 'string' || [...raw.excerpt].length > 500) throw new Error('Invalid letter excerpt');
  if (!isIsoTimestamp(raw.createdAt)) throw new Error('Invalid letter createdAt');
  if (raw.readAt !== null && raw.readAt !== undefined && !isIsoTimestamp(raw.readAt)) {
    throw new Error('Invalid letter readAt');
  }
  const themes = Array.isArray(raw.themes)
    ? raw.themes.filter((item) => isNonEmptyString(item, 48)).slice(0, 5)
    : [];
  return {
    id: raw.id,
    title: raw.title.trim(),
    excerpt: raw.excerpt.trim(),
    createdAt: raw.createdAt,
    readAt: raw.readAt ?? null,
    themes,
  };
}

/**
 * Validates a full letter payload returned by a future `GET /api/v1/me/letters/{id}` endpoint.
 * Supports both concise short letters (1-2 paragraphs) and long multi-section letters.
 */
export function parseLetterDetail(raw) {
  const summary = parseLetterSummary(raw);
  const paragraphs = Array.isArray(raw.paragraphs)
    ? raw.paragraphs.filter((item) => isNonEmptyString(item, 4000)).map((item) => item.trim())
    : isNonEmptyString(raw.body, 24000)
      ? raw.body.split(/\n\s*\n/).map((item) => item.trim()).filter(Boolean)
      : [];
  if (paragraphs.length === 0) throw new Error('Invalid letter paragraphs');
  const quote = isNonEmptyString(raw.quote, 600) ? raw.quote.trim() : null;
  const salutation = isNonEmptyString(raw.salutation, 120) ? raw.salutation.trim() : null;
  const signoff = isNonEmptyString(raw.signoff, 120) ? raw.signoff.trim() : 'با مهر، همراه';
  return {
    ...summary,
    salutation,
    quote,
    paragraphs,
    signoff,
  };
}

/**
 * Explicitly non-personalized layout previews used ONLY when the user toggles
 * design preview mode to inspect short vs. long letter typography.
 * Never persisted or presented as derived from user conversations.
 */
export const LETTER_FORMAT_PREVIEWS = Object.freeze({
  short: Object.freeze({
    id: 'preview-short',
    isSamplePreview: true,
    title: 'مکثی کوتاه میان روزهای شلوغ',
    themes: Object.freeze(['نمونهٔ کوتاه', 'خودآگاهی']),
    paragraphs: Object.freeze([
      'گاهی وقتی همه‌چیز با هم جلو می‌آد، ذهن به‌جای استراحت کردن، مدام دنبال یک راه‌حل فوری می‌گرده؛ درحالی‌که خستگیِ زیرِ این شلوغی فقط می‌خواد اول دیده بشه.',
      'شاید این هفته لازم نباشه همهٔ جواب‌ها رو از قبل بدونی. همین که وسطِ فشار، چند دقیقه بایستی و از خودت بپرسی «الان واقعاً به چی نیاز دارم؟» خودش یک قدمِ جدی و مهربونانه با خودته.',
    ]),
    signoff: 'با مهر، همراه',
  }),
  long: Object.freeze({
    id: 'preview-long',
    isSamplePreview: true,
    title: 'وقتی مراقب همه هستی، جز خستگیِ خودت',
    themes: Object.freeze(['نمونهٔ بلند', 'الگوهای رفتاری', 'مرزهای شخصی']),
    quote: 'انگار همیشه اول مطمئن می‌شم بقیه آرومن، بعد تازه یادم می‌افته خودم چقدر خسته‌ام.',
    paragraphs: Object.freeze([
      'توی روایت‌های روزمره، الگویی هست که خیلی وقت‌ها بی‌صدا تکرار می‌شه: مسئولیت‌پذیریِ زیاد در برابر حالِ دیگران، و سخت‌گیریِ پنهان نسبت به نیازهای خودمون.',
      'وقتی به این عادت کرده باشیم که همیشه «نفرِ آرام و منطقیِ جمع» باشیم، طبیعی می‌شه که خستگی یا دلخوریِ خودمون رو عقب بندازیم تا فضای رابطه متشنج نشه. در کوتاه‌مدت، این کار آرامش ظاهری می‌سازه؛ اما در بلندمدت، حسِ دیده‌نشدن و تنهایی رو بیشتر می‌کنه.',
      'نکتهٔ جالب اینجاست که مراقبت از دیگران یکی از نقطه‌های قوتِ ارزشمندِ توست — چیزی نیست که لازم باشه کنار گذاشته بشه. مسئله فقط اینه که خودِ تو هم باید داخلِ دایرهٔ همین مراقبت جا بگیری، نه بیرونش.',
      'توی روزهای آینده، اگر دیدی داری خیلی سریع می‌گی «اشکالی نداره» درحالی‌که تهِ دلت سنگینی هست، فقط یک لحظه مکث کن. لازم نیست یک‌دفعه همه‌چیز رو تغییر بدی؛ فقط به اون حسِ اولیه اجازه بده بدون سانسور برای خودت روشن بشه.',
    ]),
    signoff: 'کنارت هستم، همراه',
  }),
});
