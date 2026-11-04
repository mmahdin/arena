import assert from 'node:assert/strict';
import { test } from 'node:test';
import { canOpenRoute, canVisitOnboarding, initialRoute, onboardingMove, ONBOARDING_ROUTES, ACTIVE_ONBOARDING_ROUTES, EXCLUDED_ONBOARDING_ROUTES, visibleOnboardingRoute, previousOnboardingRoute, VALID_ROUTES } from '../src/features/profile/navigation.js';
import { FLOW, QUESTIONS } from '../src/data.js';
import { APPROACH_TO_STYLE, PROFILE_STYLES, approachForStyle, styleForApproach, validateDisplayName } from '../src/features/profile/preferences.js';
import { createSessionStore } from '../src/features/devsession/sessionStore.js';

const CURRENT = { userId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', displayName: 'کاربر', conversationStyle: 'balanced', onboardingCompletedAt: '2026-11-02T09:30:00Z' };

test('active designed routes remain reachable for completed profiles', () => {
  for (const route of ['welcome', 'testimonials', 'research', 'experts', 'privacy', 'therapist', 'age', 'concerns', 'voice', 'approach', 'survey', 'letter', 'chat', 'history', 'menu', 'home']) {
    assert.ok(VALID_ROUTES.has(route));
    assert.equal(canOpenRoute(route, true), true, route);
  }
  assert.equal(canOpenRoute('nonexistent', true), false);
});

test('unfinished profiles visit the exact 26 v2 steps and exclude legacy info pages', () => {
  assert.deepEqual(ONBOARDING_ROUTES, FLOW.slice(0, -1));
  assert.equal(ONBOARDING_ROUTES.length, 26);
  for (const route of ACTIVE_ONBOARDING_ROUTES) assert.equal(canOpenRoute(route, false), true, route);
  for (const route of EXCLUDED_ONBOARDING_ROUTES) assert.equal(canOpenRoute(route, false), false, route);
  assert.equal(ACTIVE_ONBOARDING_ROUTES.length, 26);
  for (const route of ['home', 'chat', 'history', 'survey', 'day', 'insight', 'letter', 'menu']) {
    assert.equal(canOpenRoute(route, false), false, route);
  }
  assert.equal(initialRoute(false), 'welcome');
  assert.equal(initialRoute(true), 'home');
});

test('every step advances in the designer order; back/revisit works but deep-link bypass fails', () => {
  let furthest = 0;
  assert.equal(canVisitOnboarding('name', furthest), false);
  assert.equal(onboardingMove('welcome', 'ready', furthest), null);
  assert.equal(onboardingMove('welcome', 'home', furthest), null);
  assert.equal(onboardingMove('welcome', 'testimonials', furthest), null);
  furthest = onboardingMove('welcome', 'privacy', furthest);
  assert.equal(furthest, 1);
  for (let i = 1; i < ACTIVE_ONBOARDING_ROUTES.length - 1; i++) {
    furthest = onboardingMove(ACTIVE_ONBOARDING_ROUTES[i], ACTIVE_ONBOARDING_ROUTES[i + 1], furthest);
    assert.equal(furthest, ONBOARDING_ROUTES.indexOf(ACTIVE_ONBOARDING_ROUTES[i + 1]));
    assert.equal(canVisitOnboarding(ACTIVE_ONBOARDING_ROUTES[i + 1], furthest), true);
  }
  assert.equal(onboardingMove('ready', 'home', furthest), null, 'Home requires server profile completion');
  assert.equal(onboardingMove('ready', 'approach', furthest), furthest);
  assert.equal(onboardingMove('approach', 'ready', furthest), furthest);
  assert.equal(canVisitOnboarding('home', furthest), false);
  assert.equal(canVisitOnboarding('invalid-route', furthest), false);
});

test('all original onboarding questions and visual route entrypoints remain wired', async () => {
  const { readFile } = await import('node:fs/promises');
  const source = await readFile(new URL('../src/screens/Onboarding.jsx', import.meta.url), 'utf8');
  for (const key of Object.keys(QUESTIONS)) assert.ok(source.includes('QUESTIONS[route]'), `Missing question engine: ${key}`);
  for (const section of ['function Welcome(', 'function Testimonials(', 'function Research(', 'function Experts(', 'function NameScreen(', 'function Interstitial(', 'function Privacy(', 'function AgeScreen(', 'function QuestionScreen(', 'function MoodScreen(', 'function CheckinsScreen(', 'function VoiceScreen(', 'function ApproachScreen(', 'function Ready(']) {
    assert.ok(source.includes(section), section);
  }
  const app = await readFile(new URL('../src/App.jsx', import.meta.url), 'utf8');
  assert.ok(!app.includes('MINIMAL_ONBOARDING'));
  assert.ok(!app.includes("route === 'name' && target === 'simple-intro'"));
  assert.ok(app.includes("route === 'ready' && target === 'home'"));
  assert.ok(app.includes('onboardingMove('));
});

test('all five original approaches round-trip with independent persistent values', () => {
  const choices = ['گفت‌وگوی آزاد', 'همدلی و شنیدن', 'به من یاد بده', 'راهنمایی آرام', 'حل مسئله با هم'];
  const values = choices.map(styleForApproach);
  assert.equal(new Set(values).size, 5);
  assert.deepEqual(Object.keys(APPROACH_TO_STYLE), choices);
  for (let i = 0; i < choices.length; i++) {
    assert.ok(PROFILE_STYLES.includes(values[i]));
    assert.equal(approachForStyle(values[i]), choices[i]);
  }
  assert.equal(approachForStyle('guidance'), 'راهنمایی آرام');
  assert.equal(approachForStyle('unknown'), 'گفت‌وگوی آزاد');
});

test('display name editing validates length and trims input before persistence', () => {
  assert.equal(validateDisplayName('  سارا  '), 'سارا');
  assert.equal(validateDisplayName(''), null);
  assert.equal(validateDisplayName('   '), null);
  assert.equal(validateDisplayName('x'.repeat(61)), null);
  assert.equal(validateDisplayName('x'.repeat(60)), 'x'.repeat(60));
});

test('editing completed profile persists existing onboarding timestamp and changes only owned fields', async () => {
  let latest = { ...CURRENT };
  const calls = [];
  const api = {
    async me() { return latest; },
    async profile(displayName, conversationStyle) {
      calls.push([displayName, conversationStyle]);
      latest = { ...latest, displayName, conversationStyle };
      return latest;
    },
  };
  const store = createSessionStore(api);
  await store.bootstrap();
  assert.equal(await store.saveProfile('کاربر تازه', 'balanced'), true);
  assert.equal(await store.saveProfile('کاربر تازه', 'guidance'), true);
  assert.deepEqual(calls, [['کاربر تازه', 'balanced'], ['کاربر تازه', 'guidance']]);
  assert.equal(store.getState().profile.onboardingCompletedAt, CURRENT.onboardingCompletedAt);
  assert.equal(store.getState().profile.userId, CURRENT.userId);
});

test('failed preference save restores previously persisted profile', async () => {
  const store = createSessionStore({
    async me() { return CURRENT; },
    async profile() { throw new Error('connection lost'); },
  });
  await store.bootstrap();
  assert.equal(await store.saveProfile('new', 'listening'), false);
  assert.equal(store.getState().phase, 'authenticated');
  assert.deepEqual(store.getState().profile, CURRENT);
});

test('main screen keeps real actions and removes journaling and fabricated insights', async () => {
  const { readFile } = await import('node:fs/promises');
  const screen = await readFile(new URL('../src/screens/Main.jsx', import.meta.url), 'utf8');
  for (const control of ['ذخیره پیام', 'فرستادن پیام', 'function LetterScreen(', 'function Survey(', 'function ConversationSheet(']) {
    assert.ok(screen.includes(control), `Missing active control: ${control}`);
  }
  for (const removed of ['daily-reflection-form', 'reflection-card', 'function DayDetail(', 'function Insight(', 'ساخت دوباره']) {
    assert.ok(!screen.includes(removed), `Removed user-facing feature returned: ${removed}`);
  }
});

test('confirming the default visible approach records that choice before advancing', async () => {
  const { readFile } = await import('node:fs/promises');
  const source = await readFile(new URL('../src/screens/Onboarding.jsx', import.meta.url), 'utf8');
  assert.ok(source.includes("setAnswer('approach', approach.name); nextFrom('approach', navigate);"),
    'the default visible teaching choice must not silently persist as balanced');
});

test('excluded screens resolve forward on resume and Name returns directly to Welcome', () => {
  for (const route of EXCLUDED_ONBOARDING_ROUTES) {
    assert.equal(visibleOnboardingRoute(route), null);
    assert.equal(canVisitOnboarding(route, 25), false);
    assert.ok(VALID_ROUTES.has(route), 'original page remains available outside onboarding');
  }
  assert.equal(previousOnboardingRoute('name'), 'simple-intro');
  assert.equal(previousOnboardingRoute('simple-intro'), 'privacy');
  assert.equal(ONBOARDING_ROUTES.indexOf('name'), 3);
  assert.equal(ONBOARDING_ROUTES.indexOf('ready'), 25);
});
