// Browserless interaction regression for the Keycloak OTP enhancer. No packages required.
import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';
import assert from 'node:assert/strict';

const file = new URL('../../infra/keycloak/themes/theia-sms/login/resources/js/theia-login.js', import.meta.url);
const source = readFileSync(file, 'utf8');
const elements = new Map();
const intervals = new Map();
let nextInterval = 1;
const browserEvents = new Map();
const documentEvents = new Map();
let now = 1_700_000_000_000;
const subscribe = (target, type, cb) => target.set(type, [...(target.get(type) || []), cb]);
const emit = (target, type) => (target.get(type) || []).forEach((cb) => cb());

class Element {
  constructor() {
    this.handlers = new Map();
    this.attributes = new Map();
    this.classes = new Set();
    this.classList = {
      add: (v) => this.classes.add(v),
      toggle: (v, on) => on ? this.classes.add(v) : this.classes.delete(v),
    };
    this.value = '';
    this.textContent = '';
    this.disabled = false;
    this.hidden = false;
    this.dataset = {};
  }
  addEventListener(type, callback) { this.handlers.set(type, callback); }
  dispatch(type) { this.handlers.get(type)?.({ target: this }); }
  removeAttribute(name) { this.attributes.delete(name); }
  querySelectorAll(selector) { return selector === '.theia-otp__cell' ? cells : []; }
  querySelector() { return submitButton; }
  checkValidity() { return /^\d{6}$/.test(this.value); }
  focus() { this.dispatch('focus'); }
}
const cells = Array.from({ length: 6 }, () => new Element());
const submitButton = new Element();
for (const id of ['theia-code-form', 'theia-code', 'theia-otp', 'theia-resend-button', 'theia-resend-timer', 'theia-resend-ready', 'theia-auth-error']) {
  elements.set(id, new Element());
}
elements.get('theia-code').attributes.set('aria-invalid', 'true');
elements.get('theia-resend-button').dataset.waitSeconds = '41';
const sandbox = {
  document: {
    getElementById: (id) => elements.get(id) ?? null,
    querySelectorAll: () => [submitButton],
    addEventListener: (type, cb) => subscribe(documentEvents, type, cb),
  },
  window: {
    setInterval: (cb) => { const id = nextInterval++; intervals.set(id, cb); return id; },
    clearInterval: (id) => intervals.delete(id),
    addEventListener: (type, cb) => subscribe(browserEvents, type, cb),
  },
  Date: { now: () => now },
};
runInNewContext(source, sandbox, { filename: 'theia-login.js' });
const code = elements.get('theia-code');
const otp = elements.get('theia-otp');
const resend = elements.get('theia-resend-button');
const timer = elements.get('theia-resend-timer');
assert.equal(otp.classes.has('is-enhanced'), true);
assert.equal(cells[0].classes.has('is-current'), true);
assert.equal(resend.disabled, true);
assert.equal(timer.textContent, '۰۰:۴۱');
code.value = '۱۲٣۴٥۶';
code.dispatch('input');
assert.equal(code.value, '123456');
assert.deepEqual(cells.map((cell) => cell.textContent), ['1', '2', '3', '4', '5', '6']);
assert.equal(elements.get('theia-auth-error').hidden, true);
assert.equal(code.attributes.has('aria-invalid'), false);
code.dispatch('submit');
elements.get('theia-code-form').dispatch('submit');
assert.equal(submitButton.disabled, true);
now += 10_000;
emit(documentEvents, 'visibilitychange');
assert.equal(timer.textContent, '۰۰:۳۱');
now += 31_000;
for (const callback of [...intervals.values()]) callback();
assert.equal(resend.disabled, false);
assert.equal(timer.textContent, '');
assert.equal(intervals.size, 0);
assert.equal(elements.get('theia-resend-ready').textContent.length > 0, true);
emit(browserEvents, 'pageshow');
assert.equal(submitButton.disabled, false);
console.log('AUTH_UI01_DOM_INTERACTIONS_OK');
