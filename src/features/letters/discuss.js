/**
 * The chat API accepts only a plain text turn, not attachment metadata.
 * When a person writes their first message in response to a letter, attach
 * the letter as *context for the model* in that one outgoing turn. The UI
 * decodes the context as a compact reply reference in conversation history.
 * It is never placed in the editable composer draft.
 */
export const DISCUSSION_MAX_LENGTH = 4000;
const MARKER = '\n\n[[HAMRAH_LETTER_CONTEXT_V1]]\n';
const END = '\n[[/HAMRAH_LETTER_CONTEXT_V1]]';
const length = (value) => Array.from(value).length;
const clip = (value, max) => length(value) <= max
  ? value : `${Array.from(value).slice(0, Math.max(0, max - 1)).join('').trimEnd()}…`;

export function isLetterReference(letter) {
  return Boolean(letter && typeof letter.title === 'string' && letter.title.trim() &&
    Array.isArray(letter.paragraphs) && letter.paragraphs.some((p) => typeof p === 'string' && p.trim()));
}

/** Return null if the user's own text plus useful letter context won't fit. */
export function buildLetterReplyMessage(userText, letter, maxLength = DISCUSSION_MAX_LENGTH) {
  if (!isLetterReference(letter) || typeof userText !== 'string') return null;
  const user = userText.trim();
  if (!user) return null;
  const title = clip(letter.title.replace(/[\r\n\u0000-\u001f]/g, ' ').trim(), 100);
  const paragraphs = letter.paragraphs.filter((p) => typeof p === 'string' && p.trim());
  const body = [...paragraphs, ...(typeof letter.question === 'string' && letter.question.trim() ? [letter.question] : [])].join('\n\n');
  const contextStart = `این نوشته نامه‌ای است که همراه برای من فرستاده. پاسخ خودت را با توجه به محتوای آن و پیام من بده.\nعنوان: ${title}\nمتن نامه:\n`;
  const capacity = maxLength - length(user + MARKER + END + contextStart);
  // Leave enough of the letter to be useful; never silently send an ungrounded reference.
  if (capacity < 120) return null;
  const full = `${user}${MARKER}${contextStart}${clip(body, capacity)}${END}`;
  return length(full) <= maxLength ? full : null;
}

/** Render the original user's words and a short reply chip, not the long context. */
export function parseLetterReplyMessage(content) {
  if (typeof content !== 'string') return { text: '', reference: null };
  const start = content.lastIndexOf(MARKER);
  if (start < 1 || !content.endsWith(END)) return { text: content, reference: null };
  const context = content.slice(start + MARKER.length, -END.length);
  const match = context.match(/(?:^|\n)عنوان: ([^\n]+)\nمتن نامه:\n/);
  if (!match) return { text: content, reference: null };
  return { text: content.slice(0, start), reference: { title: match[1] } };
}
