/**
 * DOM smoke test for the redesigned onboarding.
 *
 * Runs the real App against the in-memory synthetic API used by the sandbox
 * preview, in jsdom, and walks the whole 26-step journey. It is a layout-free
 * regression guard for behaviour: the narrative screens render, the chapter
 * rail advances with the story, every question still works, and nothing
 * crashes on a missing scene.
 *
 * Usage: node scripts/onboarding-narrative-smoke.mjs
 */
let JSDOM;
try {
  ({ JSDOM } = await import('jsdom'));
} catch {
  console.error('BLOCKED: jsdom is not installed (npm i --no-save jsdom).');
  process.exit(2);
}

const dom = new JSDOM('<!doctype html><html lang="fa" dir="rtl"><body><div id="root"></div></body></html>', {
  url: 'http://127.0.0.1:5173/',
  pretendToBeVisual: true,
});
const { window } = dom;
globalThis.window = window;
globalThis.document = window.document;
Object.defineProperty(globalThis, 'navigator', { value: window.navigator, configurable: true });
globalThis.location = window.location;
globalThis.history = window.history;
globalThis.HTMLElement = window.HTMLElement;
globalThis.Element = window.Element;
globalThis.Node = window.Node;
globalThis.getComputedStyle = window.getComputedStyle;
globalThis.requestAnimationFrame = (cb) => window.setTimeout(() => cb(Date.now()), 0);
globalThis.cancelAnimationFrame = (id) => window.clearTimeout(id);
globalThis.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} };
globalThis.matchMedia = () => ({ matches: false, addEventListener() {}, removeEventListener() {}, addListener() {}, removeListener() {} });
window.matchMedia = globalThis.matchMedia;
window.scrollTo = () => {};
// jsdom implements neither element scrolling nor scrollIntoView.
window.Element.prototype.scrollTo = function scrollTo() {};
window.Element.prototype.scrollIntoView = function scrollIntoView() {};
window.HTMLElement.prototype.scrollTo = function scrollTo() {};
Object.defineProperty(window.Element.prototype, 'scrollHeight', { configurable: true, get() { return 0; } });
window.requestAnimationFrame = globalThis.requestAnimationFrame;
window.cancelAnimationFrame = globalThis.cancelAnimationFrame;
window.ResizeObserver = globalThis.ResizeObserver;

// React 19's act() refuses to run unless the environment opts in before import.
window.IS_REACT_ACT_ENVIRONMENT = true;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
process.env.NODE_ENV = process.env.NODE_ENV || 'test';

const { createServer } = await import('vite');
const react = (await import('@vitejs/plugin-react')).default;

const server = await createServer({
  configFile: false,
  root: process.cwd(),
  plugins: [react()],
  server: { middlewareMode: true },
  appType: 'custom',
  logLevel: 'error',
});

// Bring the preview's synthetic API in as a connect-style middleware and drive
// it with a tiny fetch shim, so the App talks to the same contract as in the
// sandbox preview without needing a socket.
const { previewApi } = await server.ssrLoadModule('/scripts/theia-preview.mjs').catch(() => ({}));
if (!previewApi) {
  await server.close();
  console.error('BLOCKED: the preview API is not exported for tests.');
  process.exit(2);
}

const queue = [previewApi];
const fakeReqRes = async (url, init = {}) => {
  const cookie = globalThis.__cookie || '';
  const req = {
    url, method: init.method || 'GET',
    headers: { cookie, ...(init.headers || {}) },
    on(event, handler) {
      if (event === 'data' && init.body) handler(Buffer.from(init.body));
      if (event === 'end') handler();
      return this;
    },
  };
  let status = 200;
  const headers = {};
  let payload = '';
  const res = {
    writeHead(code, extra = {}) { status = code; Object.assign(headers, extra); return this; },
    setHeader(name, value) { headers[name] = value; },
    end(body = '') { payload = body; },
  };
  await new Promise((resolve) => {
    const next = () => resolve();
    const middleware = queue[0];
    middleware(req, res, next).then(resolve).catch(resolve);
  });
  if (headers['Set-Cookie']) globalThis.__cookie = String(headers['Set-Cookie']).split(';')[0];
  return {
    status,
    ok: status >= 200 && status < 300,
    headers: { get: (name) => headers[name] ?? headers[name.toLowerCase()] ?? null },
    async json() { return payload ? JSON.parse(payload) : null; },
    async text() { return payload; },
  };
};
globalThis.fetch = (url, init) => fakeReqRes(String(url), init);

const React = await import('react');
const { act } = React;
const ReactDOM = await import('react-dom/client');
const App = (await server.ssrLoadModule('/src/App.jsx')).default;

const root = window.document.getElementById('root');
let client;
window.IS_REACT_ACT_ENVIRONMENT = true;
client = ReactDOM.createRoot(root);
await act(async () => { client.render(React.createElement(App)); });

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
// React's own act() is the only reliable way to drain work in jsdom.
const flush = async () => {
  for (let i = 0; i < 8; i += 1) {
    await act(async () => { await sleep(15); });
  }
};
await flush();

const el = (selector) => window.document.querySelector(selector);
const all = (selector) => [...window.document.querySelectorAll(selector)];
const fail = (message) => { console.error(`FAIL: ${message}`); process.exitCode = 1; };
const ok = (condition, message) => { if (!condition) fail(message); else console.log(`ok — ${message}`); };
const route = () => window.location.hash.replace(/^#\/?/, '');
const visited = new Set();
const noteRoute = () => {
  const current = route();
  if (visited.has(current)) return;
  visited.add(current);
  const chapterScreen = el('[class*="chapter-screen--"]');
  if (chapterScreen) {
    const id = [...chapterScreen.classList].find((name) => name.startsWith('chapter-screen--')).split('--')[1];
    const name = [...window.document.querySelectorAll('.chapter-rail .is-active')].map((n) => n.textContent).join('');
    ok(el('.theia-scene') && name.length > 0, `chapter ${id} renders its scene and rail (${name})`);
  }
};

const click = async (node) => {
  if (!node) return;
  await act(async () => {
    node.dispatchEvent(new window.MouseEvent('click', { bubbles: true }));
    await sleep(10);
  });
  await flush();
};

ok(el('#dev-phone'), 'login renders for a fresh visitor');
await act(async () => {
  const field = el('#dev-phone');
  const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
  setter.call(field, '09000000001');
  field.dispatchEvent(new window.Event('input', { bubbles: true }));
  await sleep(10);
});
await flush();
const enter = all('.dev-login-panel button').find((node) => node.textContent.includes('ادامه'));
await click(enter);
await flush();

ok(route() === 'welcome', 'new user lands on Welcome');
noteRoute();
ok(el('.welcome-screen'), 'Welcome uses the narrative shell');
ok(el('.welcome-screen .theia-scene'), 'Welcome renders generated scene art');
ok(all('.chapter-rail li').length === 4, 'Welcome shows the four-chapter rail');
ok(el('.welcome-title').textContent.includes('دیدن'), 'Welcome headline carries the seeing theme');

const advance = async (expected) => {
  const button = el('main .app-screen .primary-button');
  if (!button) { fail(`no advance button on ${route()}`); return; }
  await click(button);
  noteRoute();
  if (expected && route() !== expected) fail(`expected #/${expected}, got #/${route()}`);
};

await advance('privacy');
ok(el('.privacy-screen'), 'Privacy follows Welcome');
ok(el('.privacy-screen [role="switch"]'), 'Privacy keeps its consent switch');
await advance('simple-intro');
ok(el('.chapter-screen--night'), 'Simple intro opens chapter I (night)');
ok(el('.chapter-eyebrow').textContent.includes('شب'), 'chapter I names the night');
await advance('name');
ok(el('.name-screen'), 'Name follows chapter I');
const nameInput = el('.name-screen #name');
await act(async () => {
  const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
  setter.call(nameInput, 'آوا');
  nameInput.dispatchEvent(new window.Event('input', { bubbles: true }));
  await sleep(10);
});
await flush();
await click(el('.name-screen button[type="submit"]'));
ok(route() === 'age', 'name is persisted and the journey continues');

// Walk the rest of the intake with the same interactions the E2E helper uses.
const { QUESTIONS } = await server.ssrLoadModule('/src/data.js');
const { ACTIVE_ONBOARDING_ROUTES } = await server.ssrLoadModule('/src/features/profile/navigation.js');
let guard = 0;
while (!['home', 'chat'].includes(route()) && guard < 60) {
  guard += 1;
  const current = route();
  noteRoute();
  if (current === 'age') { await advance('identity'); continue; }
  if (current === 'mood') { await advance('concerns'); continue; }
  if (QUESTIONS[current]) {
    const first = el(`.question-${current} .choice-button`);
    if (!first) { fail(`no choices on ${current}`); break; }
    await click(first);
    // Single-choice questions auto-advance; multi-choice ones need the CTA.
    if (route() === current && QUESTIONS[current].type === 'multi') {
      const skip = el(`.question-${current} .skip-button`);
      await click(skip || el(`.question-${current} .primary-button`));
    }
    if (route() === current) { await click(el(`.question-${current} .primary-button`)); }
    if (route() === current) await sleep(250);
    if (route() === current) { fail(`could not leave question ${current}`); break; }
    continue;
  }
  if (current === 'approach') {
    await click(el('.approach-screen .primary-button'));
    continue;
  }
  await advance();
}
ok(route() === 'chat' || route() === 'home', `onboarding completes into the product (${route()})`);
for (const chapterScreen of ['simple-intro', 'deeper-intro', 'know-intro', 'almost-intro']) {
  ok(visited.has(chapterScreen), `the journey passes through chapter screen ${chapterScreen}`);
}
ok(visited.has('checkins') && visited.has('ready'), 'the journey ends on check-ins and the finale');
ok(!window.document.body.textContent.includes('undefined'), 'no undefined leaked into the copy');

await server.close();
client.unmount?.();
if (!process.exitCode) console.log('PASS: onboarding narrative smoke');
