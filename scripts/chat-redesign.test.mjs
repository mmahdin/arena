import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const main = readFileSync(new URL('../src/screens/Main.jsx', import.meta.url), 'utf8');
const css = readFileSync(new URL('../src/styles.css', import.meta.url), 'utf8');
const chat = main.slice(main.indexOf('function Chat('), main.indexOf('export default function MainScreen'));

test('chat header has one history entry point and no subtitle', () => {
  assert.equal((chat.match(/setSheetOpen\(true\)/g) || []).length, 1);
  assert.ok(!chat.includes('chat-history-button'));
  assert.ok(!chat.includes('chat-subtitle'));
  assert.ok(!main.includes('برای ادامهٔ همین گفت‌وگو'));
});

test('composer: Enter sends, Shift+Enter keeps the new line, no instruction text', () => {
  assert.match(chat, /event\.key !== 'Enter' \|\| event\.shiftKey/);
  assert.match(chat, /isComposing/);
  assert.ok(!chat.includes('composer-hint'));
  assert.ok(!main.includes('Ctrl+Enter'));
  assert.ok(!css.includes('.composer-hint'));
});

test('assistant messages carry no repeated author label', () => {
  assert.ok(!chat.includes('message-author'));
  assert.ok(!css.includes('.message-author'));
});

test('message actions are inline tools, not a bottom sheet or three-dot menu', () => {
  assert.ok(main.includes('function MessageTools('));
  assert.ok(!main.includes('function MessageActions('));
  assert.ok(!main.includes('MoreHorizontal'));
  assert.ok(!chat.includes('onContextMenu'));
  for (const label of ['کپی پیام', 'ذخیره پیام', 'حذف از ذخیره‌شده‌ها', 'خواندن با صدای دستگاه']) assert.ok(main.includes(label), label);
  assert.match(css, /\.message-tools \{[^}]*pointer-events: none/);
});

test('motion respects reduced-motion preference', () => {
  const block = css.slice(css.indexOf('@media (prefers-reduced-motion: reduce)'));
  assert.ok(block.includes('.message-block.is-fresh'));
  assert.ok(block.includes('.message-tool--save.is-saved svg'));
});
