import { LETTER_KIND, normalizeLetters } from './model.js';

/**
 * Backend-ready letter state without inventing an endpoint.
 *
 * `source` is an optional future adapter: { list({ signal }): Promise<Letter[]> }.
 * Passing null keeps the product honest: personalized letters are unavailable,
 * while the UI can still show the separately labeled illustrative sample.
 */
export function createLetterStore(source = null, { onUnauthorized = () => {} } = {}) {
  const initial = () => ({
    userId: null,
    phase: source ? 'idle' : 'unavailable',
    letters: [],
    error: null,
  });
  let state = initial();
  let epoch = 0;
  let version = 0;
  const controllers = new Set();
  const listeners = new Set();
  const publish = (patch) => {
    state = { ...state, ...patch };
    for (const listener of listeners) listener(state);
  };
  const scope = () => {
    const ticket = epoch;
    const controller = new AbortController();
    controllers.add(controller);
    return {
      signal: controller.signal,
      current: () => ticket === epoch && !controller.signal.aborted && state.userId !== null,
      done: () => controllers.delete(controller),
    };
  };
  function reset() {
    epoch++;
    version++;
    for (const controller of controllers) controller.abort();
    controllers.clear();
    publish(initial());
  }
  function activate(userId) {
    if (state.userId === userId) return;
    reset();
    if (userId) {
      publish({ userId });
      if (source) void load();
    }
  }
  async function load() {
    if (!state.userId) return false;
    if (!source) {
      publish({ phase: 'unavailable', letters: [], error: null });
      return false;
    }
    const req = scope();
    const requestVersion = ++version;
    publish({ phase: 'loading', error: null });
    try {
      const letters = normalizeLetters(await source.list({ signal: req.signal }));
      if (letters.some((letter) => letter.kind !== LETTER_KIND.PERSONALIZED)) {
        throw new Error('Letter source must return personalized letters only');
      }
      if (req.current() && requestVersion === version) {
        publish({ phase: 'ready', letters, error: null });
        return true;
      }
      return false;
    } catch (error) {
      if (req.current() && requestVersion === version) {
        if (error?.status === 401) {
          reset();
          onUnauthorized();
        } else {
          publish({ phase: 'error', error: error?.code ?? 'NETWORK_ERROR' });
        }
      }
      return false;
    } finally {
      req.done();
    }
  }
  return {
    getState: () => state,
    subscribe(listener) { listeners.add(listener); return () => listeners.delete(listener); },
    activate,
    load,
    reset,
    cancel() { epoch++; version++; for (const controller of controllers) controller.abort(); controllers.clear(); },
  };
}
