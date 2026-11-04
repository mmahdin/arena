/**
 * Frontend-only contract for future, backend-authored personalized letters.
 * No request is made until the backend endpoint and privacy rules are approved.
 * Demonstration content must never be passed as a real user's letter.
 */
const BLOCK_TYPES = new Set(['heading', 'paragraph', 'question']);
const cleanText = value => typeof value === 'string' && value.trim() ? value.trim() : null;

export function normalizeLetter(value) {
  if (!value || typeof value !== 'object') return null;
  const id = cleanText(value.id);
  const title = cleanText(value.title);
  const createdAt = cleanText(value.createdAt);
  if (!id || !title || !createdAt || Number.isNaN(Date.parse(createdAt)) || !Array.isArray(value.blocks)) return null;
  const blocks = value.blocks.map(block => {
    if (!block || !BLOCK_TYPES.has(block.type)) return null;
    const text = cleanText(block.text);
    return text ? { type: block.type, text } : null;
  });
  if (blocks.some(block => !block) || !blocks.some(block => block.type === 'paragraph')) return null;
  return {
    id,
    title,
    createdAt,
    greeting: cleanText(value.greeting),
    blocks,
  };
}

export function formatLetterDate(value) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '' : new Intl.DateTimeFormat('fa-IR', { dateStyle: 'long' }).format(date);
}
