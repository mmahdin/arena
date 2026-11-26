import test from 'node:test';
import assert from 'node:assert/strict';
import { buildLetterReplyMessage, parseLetterReplyMessage, DISCUSSION_MAX_LENGTH } from '../src/features/letters/discuss.js';
import { readFileSync } from 'node:fs';

const letter = {
  id: 'a-letter', title: 'یک مکث', paragraphs: ['بند اول.', 'بند دوم.'], question: 'چه چیزی را شنیدی؟',
};

test('letter reference is not inserted into composer and real letter context is sent only with first user turn', () => {
  const app = readFileSync(new URL('../src/App.jsx', import.meta.url), 'utf8');
  assert.match(app, /chatStore\.setDraft\(''\)/);
  assert.match(app, /buildLetterReplyMessage\(text, letterReply\)/);
  assert.doesNotMatch(app, /chatStore\.setDraft\(draft\)/);
  const message = buildLetterReplyMessage('به نظرت از کجا شروع کنم؟', letter);
  assert.ok(message.includes('بند اول.') && message.includes('بند دوم.') && message.includes('چه چیزی را شنیدی؟'));
  assert.match(message, /به نظرت از کجا شروع کنم؟/);
  const view = parseLetterReplyMessage(message);
  assert.equal(view.text, 'به نظرت از کجا شروع کنم؟');
  assert.equal(view.reference.title, 'یک مکث');
});

test('long letters are clipped without clipping the user message or exceeding API limits', () => {
  const long = { title: 'ع'.repeat(130), paragraphs: Array.from({ length: 12 }, () => 'م'.repeat(2000)), question: 'س'.repeat(600) };
  const message = buildLetterReplyMessage('من دربارهٔ این نامه سؤال دارم.', long);
  assert.ok(Array.from(message).length <= DISCUSSION_MAX_LENGTH);
  assert.equal(parseLetterReplyMessage(message).text, 'من دربارهٔ این نامه سؤال دارم.');
  assert.ok(message.includes('متن نامه:'));
});

test('invalid letters, empty texts and messages leaving no room for letter context are refused', () => {
  assert.equal(buildLetterReplyMessage('سلام', null), null);
  assert.equal(buildLetterReplyMessage('', letter), null);
  assert.equal(buildLetterReplyMessage('x'.repeat(3900), letter), null);
});

test('ordinary user messages never show a false reference', () => {
  const ordinary = 'سلام\n\nامروز چطوری؟';
  assert.deepEqual(parseLetterReplyMessage(ordinary), { text: ordinary, reference: null });
});

test('letters cannot be deleted from the inbox or reader interface', () => {
  const screen = readFileSync(new URL('../src/screens/Letters.jsx', import.meta.url), 'utf8');
  const app = readFileSync(new URL('../src/App.jsx', import.meta.url), 'utf8');
  assert.doesNotMatch(screen, /onDelete|حذف نامه|letter-forget/);
  assert.doesNotMatch(app, /onDeleteLetter:/);
});
