import assert from 'node:assert/strict';
import test from 'node:test';
import { normalizeLetter, formatLetterDate } from '../src/features/letter/model.js';

const valid = {
  id: 'letter-1', title: 'یک عنوان واقعی', createdAt: '2026-11-04T10:00:00Z', greeting: 'سلام',
  blocks: [
    { type: 'paragraph', text: 'متن تأییدشدهٔ سرور.' },
    { type: 'heading', text: 'بخش کوتاه' },
    { type: 'question', text: 'یک پرسش برای فکر کردن؟' },
  ],
};

test('letter model accepts structured server content without inventing fields', () => {
  const letter = normalizeLetter(valid);
  assert.equal(letter.title, valid.title);
  assert.deepEqual(letter.blocks, valid.blocks);
  assert.match(formatLetterDate(letter.createdAt), /۱۴۰۵/);
});

test('empty and malformed letters produce an honest empty state', () => {
  for (const value of [null, undefined, {}, { ...valid, blocks: [] }, { ...valid, createdAt: 'not-a-date' },
    { ...valid, blocks: [{ type: 'diagnosis', text: 'unsupported' }] }, { ...valid, blocks: [{ type: 'heading', text: 'no paragraphs' }] }]) {
    assert.equal(normalizeLetter(value), null);
  }
});
