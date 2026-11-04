/**
 * Frontend-only letter domain model.
 *
 * There is deliberately no HTTP transport in this module. A future backend
 * adapter can supply normalized, server-owned personalized letters; until then
 * the UI shows an explicitly illustrative sample and an honest unavailable state.
 */
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const MAX_PARAGRAPHS = 12;
const MAX_PARAGRAPH_LENGTH = 2000;

export const LETTER_KIND = Object.freeze({
  PERSONALIZED: 'personalized',
  SAMPLE: 'sample',
});

function cleanText(value, max = 240) {
  if (typeof value !== 'string') return null;
  const text = value.trim();
  return text.length > 0 && [...text].length <= max ? text : null;
}

function validTimestamp(value) {
  return typeof value === 'string' && !Number.isNaN(new Date(value).getTime());
}

/**
 * Proposed integration shape (not an API contract implemented by this patch):
 * {
 *   id, kind: 'personalized', title, salutation?, paragraphs: string[],
 *   question?, createdAt, sourceConversationIds: uuid[]
 * }
 *
 * Personalized letters must carry server-proven conversation provenance. The
 * frontend will not render a letter as personalized merely because it has a
 * friendly tone or because a client-side flag was set.
 */
export function normalizeLetter(value) {
  if (!value || typeof value !== 'object') throw new Error('Invalid letter response');
  const id = cleanText(value.id, 120);
  const title = cleanText(value.title, 160);
  const kind = value.kind;
  if (!id || !title || ![LETTER_KIND.PERSONALIZED, LETTER_KIND.SAMPLE].includes(kind)) {
    throw new Error('Invalid letter response');
  }
  if (!Array.isArray(value.paragraphs) || value.paragraphs.length === 0 || value.paragraphs.length > MAX_PARAGRAPHS) {
    throw new Error('Invalid letter paragraphs');
  }
  const paragraphs = value.paragraphs.map((paragraph) => {
    const text = cleanText(paragraph, MAX_PARAGRAPH_LENGTH);
    if (!text) throw new Error('Invalid letter paragraph');
    return text;
  });
  if (!validTimestamp(value.createdAt)) throw new Error('Invalid letter timestamp');

  const sourceConversationIds = Array.isArray(value.sourceConversationIds)
    ? [...new Set(value.sourceConversationIds)]
    : [];
  if (!sourceConversationIds.every((sourceId) => typeof sourceId === 'string' && UUID.test(sourceId))) {
    throw new Error('Invalid letter provenance');
  }
  if (kind === LETTER_KIND.PERSONALIZED && sourceConversationIds.length === 0) {
    throw new Error('Personalized letter requires conversation provenance');
  }

  const salutation = value.salutation == null ? null : cleanText(value.salutation, 120);
  const question = value.question == null ? null : cleanText(value.question, 600);
  if (value.salutation != null && !salutation) throw new Error('Invalid letter salutation');
  if (value.question != null && !question) throw new Error('Invalid letter question');

  return Object.freeze({
    id,
    kind,
    title,
    salutation,
    paragraphs: Object.freeze(paragraphs),
    question,
    createdAt: value.createdAt,
    sourceConversationIds: Object.freeze(sourceConversationIds),
  });
}

export function normalizeLetters(value) {
  if (!Array.isArray(value)) throw new Error('Invalid letter list');
  return value.map(normalizeLetter);
}

/** Explicitly illustrative. It is not derived from any user's conversations. */
export const SAMPLE_LETTER = normalizeLetter({
  id: 'sample-letter',
  kind: LETTER_KIND.SAMPLE,
  title: 'یک مکث، پیش از پاسخ',
  salutation: 'دوست من،',
  paragraphs: [
    'گاهی وقتی کسی چیزی می‌گوید، پیش از آنکه فرصت کنیم بفهمیم چه حسی داریم، ذهنمان دنبال جواب می‌گردد.',
    'می‌شود برای یک لحظه بین شنیدن و پاسخ دادن فاصله انداخت؛ نه برای پیدا کردن حرف درست، بلکه برای فهمیدن اینکه آن حرف چه حسی در ما ایجاد کرده است.',
    'این مکث کوچک قرار نیست چیزی را ثابت کند. فقط فضایی است برای اینکه واکنش بعدی‌ات از آگاهی بیاید، نه از عجله.',
  ],
  question: 'آخرین بار که پیش از جواب دادن مکث کردی، چه چیزی را بهتر شنیدی؟',
  createdAt: '2026-11-04T00:00:00.000Z',
  sourceConversationIds: [],
});
