import assert from 'node:assert/strict';
import { test } from 'node:test';
import { readFileSync } from 'node:fs';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { createServer } from 'vite';
import { APPROACHES, APPROACH_USER_MESSAGE, VOICES } from '../src/data.js';
import { createConversationStore } from '../src/features/conversation/store.js';
import { createLetterStore } from '../src/features/letter/store.js';
import { LETTER_FORMAT_PREVIEWS, parseLetterDetail, parseLetterSummary } from '../src/features/letter/model.js';

const USER_A = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
const USER_B = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';
const CONV_OLD = '11111111-1111-4111-8111-111111111111';
const CONV_RECENT = '22222222-2222-4222-8222-222222222222';
const LETTER_ID = '33333333-3333-4333-8333-333333333333';

test('enterChat restores the most recent conversation instead of creating a blank chat', async () => {
  const older = {
    id: CONV_OLD,
    title: 'گفت‌وگوی قدیمی‌تر',
    createdAt: '2026-11-01T08:00:00Z',
    updatedAt: '2026-11-01T09:00:00Z',
  };
  const recent = {
    id: CONV_RECENT,
    title: 'آخرین گفت‌وگوی من',
    createdAt: '2026-11-03T10:00:00Z',
    updatedAt: '2026-11-03T15:30:00Z',
  };
  const recentMessages = [
    { id: USER_A, role: 'user', content: 'سلام همراه', createdAt: recent.createdAt },
    { id: USER_B, role: 'assistant', content: 'سلام، خوشحالم برگشتی.', createdAt: recent.updatedAt },
  ];
  const calls = [];
  const store = createConversationStore({
    async list() { calls.push('list'); return [older, recent]; },
    async messages(id) { calls.push(['messages', id]); return id === CONV_RECENT ? recentMessages : []; },
  });

  store.activate(USER_A);
  assert.equal(await store.enterChat(), true);
  assert.equal(store.getState().conversationId, CONV_RECENT);
  assert.deepEqual(store.getState().messages, recentMessages);
  assert.equal(store.getState().isDraftNew, false);
  assert.deepEqual(calls, ['list', ['messages', CONV_RECENT]]);

  // Re-entering Chat while a conversation is already open keeps it active without wiping state.
  assert.equal(await store.enterChat(), true);
  assert.equal(store.getState().conversationId, CONV_RECENT);
  assert.equal(calls.length, 2);
});

test('explicit startNew preserves a fresh conversation draft on enterChat', async () => {
  const recent = {
    id: CONV_RECENT,
    title: 'گفت‌وگوی قبلی',
    createdAt: '2026-11-03T10:00:00Z',
    updatedAt: '2026-11-03T15:30:00Z',
  };
  const store = createConversationStore({
    async list() { return [recent]; },
    async messages() { throw new Error('should not load old messages when explicit draft is active'); },
  });

  store.activate(USER_A);
  assert.equal(store.startNew(), true);
  assert.equal(store.getState().isDraftNew, true);
  assert.equal(await store.enterChat(), true);
  assert.equal(store.getState().conversationId, null);
  assert.deepEqual(store.getState().messages, []);
});

test('enterChat with zero saved conversations settles cleanly into empty ready state', async () => {
  const store = createConversationStore({
    async list() { return []; },
    async messages() { throw new Error('unexpected messages call'); },
  });
  store.activate(USER_A);
  assert.equal(await store.enterChat(), true);
  assert.equal(store.getState().conversationId, null);
  assert.equal(store.getState().messagesPhase, 'ready');
  assert.deepEqual(store.getState().messages, []);
});

test('switching identity during in-flight enterChat fences stale conversation restore', async () => {
  let resolveList;
  const pendingList = new Promise((resolve) => { resolveList = resolve; });
  const store = createConversationStore({
    async list() { return pendingList; },
    async messages() { return [{ id: USER_A, role: 'user', content: 'خصوصی', createdAt: '2026-11-03T10:00:00Z' }]; },
  });

  store.activate(USER_A);
  const entering = store.enterChat();
  store.activate(USER_B);
  resolveList([{ id: CONV_RECENT, title: 'خصوصی', createdAt: '2026-11-03T10:00:00Z', updatedAt: '2026-11-03T10:00:00Z' }]);
  assert.equal(await entering, false);
  assert.equal(store.getState().userId, USER_B);
  assert.equal(store.getState().conversationId, null);
  assert.deepEqual(store.getState().conversations, []);
});

test('Personalized Letter store performs zero fake network calls when backend is unintegrated and validates short/long letters when connected', async () => {
  const unintegrated = createLetterStore(null);
  unintegrated.activate(USER_A);
  assert.equal(await unintegrated.loadList(), true);
  assert.equal(unintegrated.getState().backendIntegrated, false);
  assert.deepEqual(unintegrated.getState().items, []);
  assert.equal(unintegrated.getState().activeLetter, null);

  const shortPayload = {
    id: LETTER_ID,
    title: 'نامهٔ کوتاه',
    excerpt: 'خلاصهٔ کوتاه',
    createdAt: '2026-11-04T09:00:00Z',
    readAt: null,
    paragraphs: ['پاراگراف اول کوتاه.'],
  };
  const longPayload = {
    ...shortPayload,
    themes: ['شناخت خود', 'الگوی رفتاری'],
    quote: 'جمله‌ای از گفت‌وگوهای واقعی کاربر',
    paragraphs: ['بند اول.', 'بند دوم.', 'بند سوم.', 'بند چهارم.'],
  };
  assert.equal(parseLetterSummary(shortPayload).title, 'نامهٔ کوتاه');
  assert.equal(parseLetterDetail(shortPayload).paragraphs.length, 1);
  assert.equal(parseLetterDetail(longPayload).paragraphs.length, 4);
  assert.equal(LETTER_FORMAT_PREVIEWS.short.isSamplePreview, true);
  assert.equal(LETTER_FORMAT_PREVIEWS.long.isSamplePreview, true);

  let unauthorized = 0;
  const connected = createLetterStore({
    async list() { return [shortPayload]; },
    async read() { return longPayload; },
    async markRead() { return { readAt: '2026-11-04T10:00:00Z' }; },
  }, { onUnauthorized: () => { unauthorized += 1; } });

  connected.activate(USER_A);
  assert.equal(await connected.loadList(), true);
  assert.equal(connected.getState().items.length, 1);
  assert.equal(await connected.openLetter(LETTER_ID), true);
  assert.equal(connected.getState().activeLetter.paragraphs.length, 4);
  assert.equal(await connected.markRead(LETTER_ID), true);
  assert.equal(connected.getState().items[0].readAt, '2026-11-04T10:00:00Z');
  connected.reset();
  assert.equal(connected.getState().activeLetter, null);
  assert.equal(unauthorized, 0);
});

test('Onboarding and Main JSX screens SSR-render improved UX states cleanly with journaling removed', async () => {
  const mainSource = readFileSync(new URL('../src/screens/Main.jsx', import.meta.url), 'utf8');
  const appSource = readFileSync(new URL('../src/App.jsx', import.meta.url), 'utf8');
  assert.doesNotMatch(mainSource, /یادداشت روزانه|یادداشت‌های قبلی|daily-reflection-form/);
  assert.doesNotMatch(appSource, /reflectionStore/);

  const vite = await createServer({
    server: { middlewareMode: true },
    appType: 'custom',
    logLevel: 'silent',
  });

  try {
    const { default: OnboardingScreen } = await vite.ssrLoadModule('/src/screens/Onboarding.jsx');
    const { default: MainScreen } = await vite.ssrLoadModule('/src/screens/Main.jsx');

    const baseProps = {
      navigate: () => {},
      back: () => {},
      name: 'نگار',
      setName: () => {},
      answers: { mood: 0, focus: ['بهتر بخوابم'] },
      setAnswer: () => {},
      notify: () => {},
    };

    const voiceHtml = renderToStaticMarkup(React.createElement(OnboardingScreen, { ...baseProps, route: 'voice' }));
    assert.match(voiceHtml, /voice-peek-card--prev/);
    assert.match(voiceHtml, /voice-peek-card--next/);
    for (const voice of VOICES) {
      assert.ok(voiceHtml.includes(voice.name), `voice option visible: ${voice.name}`);
    }

    const moodHtml = renderToStaticMarkup(React.createElement(OnboardingScreen, { ...baseProps, route: 'mood' }));
    assert.match(moodHtml, /left:0%/);
    assert.match(moodHtml, /left:25%/);
    assert.match(moodHtml, /left:50%/);
    assert.match(moodHtml, /left:75%/);
    assert.match(moodHtml, /left:100%/);
    assert.match(moodHtml, /mood-indicator/);

    const checkinsHtml = renderToStaticMarkup(React.createElement(OnboardingScreen, { ...baseProps, route: 'checkins' }));
    assert.match(checkinsHtml, /checkins-screen/);
    assert.match(checkinsHtml, /یه یادآوری، درست وقتی لازمه/);

    const approachHtml = renderToStaticMarkup(React.createElement(OnboardingScreen, { ...baseProps, route: 'approach' }));
    assert.ok(approachHtml.includes(APPROACH_USER_MESSAGE));
    for (const approach of APPROACHES) {
      assert.ok(approachHtml.includes(approach.name), `style tab visible: ${approach.name}`);
    }
    assert.ok(approachHtml.includes(APPROACHES[2].response));

    const focusHtml = renderToStaticMarkup(React.createElement(OnboardingScreen, { ...baseProps, route: 'focus' }));
    assert.match(focusHtml, /question-selection-meta/);

    const homeHtml = renderToStaticMarkup(React.createElement(MainScreen, {
      route: 'home',
      navigate: () => {},
      name: 'نگار',
      letters: [],
    }));
    assert.doesNotMatch(homeHtml, /reflection-card/);
    assert.match(homeHtml, /envelope-art/);
    assert.match(homeHtml, /chat-fab/);

    const awaitingLetterHtml = renderToStaticMarkup(React.createElement(MainScreen, {
      route: 'day',
      back: () => {},
      navigate: () => {},
      name: 'نگار',
      letters: [],
      activeLetter: null,
    }));
    assert.match(awaitingLetterHtml, /letter-sheet--awaiting/);
    assert.doesNotMatch(awaitingLetterHtml, /آخرش من عذرخواهی می‌کنم/);

    const realLetterHtml = renderToStaticMarkup(React.createElement(MainScreen, {
      route: 'day',
      back: () => {},
      navigate: () => {},
      name: 'نگار',
      letters: [{ id: LETTER_ID, title: 'نگاه تازه', excerpt: 'خلاصه', createdAt: '2026-11-04T09:00:00Z', readAt: null, themes: ['خودآگاهی'] }],
      activeLetter: {
        id: LETTER_ID,
        title: 'نگاه تازه',
        excerpt: 'خلاصه',
        createdAt: '2026-11-04T09:00:00Z',
        readAt: null,
        themes: ['خودآگاهی'],
        paragraphs: ['بند اول نامه.', 'بند دوم نامه.'],
        signoff: 'با مهر، همراه',
      },
    }));
    assert.match(realLetterHtml, /نگاه تازه/);
    assert.match(realLetterHtml, /بند اول نامه\./);

    const surveyHtml = renderToStaticMarkup(React.createElement(MainScreen, {
      route: 'survey',
      back: () => {},
      navigate: () => {},
      answers: { healthSurvey: 'چند روز' },
      setAnswer: () => {},
      notify: () => {},
    }));
    assert.match(surveyHtml, /survey-intro/);
    assert.match(surveyHtml, /توی دو هفتهٔ گذشته/);

    const chatHtml = renderToStaticMarkup(React.createElement(MainScreen, {
      route: 'chat',
      navigate: () => {},
      conversationId: CONV_RECENT,
      conversations: [{ id: CONV_RECENT, title: 'گفت‌وگو دربارهٔ کار', createdAt: '2026-11-04T08:00:00Z', updatedAt: '2026-11-04T09:00:00Z' }],
      messages: [
        { id: USER_A, role: 'user', content: 'کارهام سنگین شده.', createdAt: '2026-11-04T08:00:00Z' },
        { id: USER_B, role: 'assistant', content: 'می‌خوای از مهم‌ترینش شروع کنیم؟', createdAt: '2026-11-04T08:01:00Z' },
      ],
      bookmarks: [],
      bookmarkPhase: 'ready',
      onNewConversation: () => true,
    }));
    assert.match(chatHtml, /chat-thread-switcher/);
    assert.match(chatHtml, /گفت‌وگو دربارهٔ کار/);
    assert.match(chatHtml, /assistant-presence/);
  } finally {
    await vite.close();
  }
});
