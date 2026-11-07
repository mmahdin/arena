import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { buildLetterReplyMessage, parseLetterReplyMessage } from '../src/features/letters/discuss.js';

const letter = { title: 'نامه یک', paragraphs: ['متن یک', 'متن دو'], question: 'نظرت؟' };
test('reply reference survives message round-trip through API text and hides body in UI', () => {
  const text = buildLetterReplyMessage('این قسمت برام مهمه', letter);
  assert.match(text, /متن یک/);
  assert.match(text, /متن دو/);
  assert.match(text, /نظرت/);
  assert.deepEqual(parseLetterReplyMessage(text), {
    text: 'این قسمت برام مهمه', reference: { title: 'نامه یک' },
  });
});
test('composer and sent chat messages show the compact reference while plain message stays plain', () => {
  const chat = readFileSync(new URL('../src/screens/Main.jsx', import.meta.url), 'utf8');
  assert.match(chat, /letterReply && <LetterReference/);
  assert.match(chat, /parseLetterReplyMessage\(message.content\)/);
  assert.match(chat, /onDismissLetterReply/);
  assert.doesNotMatch(chat, /void onSendMessage\(clean\);[\s\S]{0,100}setDraft\(''\)/);
});
