import { LETTER_KIND, normalizeLetter, normalizeLetters } from './model.js';

/**
 * Identity-scoped, server-backed letter state with an optional source for tests.
 *
 * `source` is the server transport: list/generate/remove. Passing null still
 * supports a truthful unavailable state in isolated tests.
 */
export function createLetterStore(source = null, { onUnauthorized = () => {} } = {}) {
  const initial = () => ({
    userId: null,
    phase: source ? 'idle' : 'unavailable',
    letters: [],
    error: null,
    generating: false,
    generationError: null,
    deletingId: null,
    deletionError: null,
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
  async function generate() {
    if (!state.userId || !source?.generate || state.generating) return false;
    const req = scope();
    const requestVersion = ++version;
    publish({ generating: true, generationError: null });
    try {
      // Invoked only from a user-facing explicit-consent button.
      const letter = normalizeLetter(await source.generate({ signal: req.signal }));
      if (letter.kind !== LETTER_KIND.PERSONALIZED) throw new Error('Invalid generated letter');
      if (!req.current() || requestVersion !== version) return false;
      publish({
        phase: 'ready',
        letters: [letter, ...state.letters.filter(item => item.id !== letter.id)],
        error: null,
        generationError: null,
      });
      return true;
    } catch (error) {
      if (req.current()) {
        if (error?.status === 401) { reset(); onUnauthorized(); }
        else publish({ generationError: error?.code ?? 'INVALID_RESPONSE' });
      }
      return false;
    } finally {
      req.done();
      if (req.current()) publish({ generating: false });
    }
  }
  async function remove(id) {
    if (!state.userId || !source?.remove || state.deletingId) return false;
    const req = scope();
    const requestVersion = ++version;
    publish({ deletingId: id, deletionError: null });
    try {
      await source.remove(id, { signal: req.signal });
      if (!req.current() || requestVersion !== version) return false;
      publish({ letters: state.letters.filter(item => item.id !== id), deletionError: null });
      return true;
    } catch (error) {
      if (req.current()) {
        if (error?.status === 401) { reset(); onUnauthorized(); }
        else publish({ deletionError: error?.code ?? 'NETWORK_ERROR' });
      }
      return false;
    } finally {
      req.done();
      if (req.current()) publish({ deletingId: null });
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
    generate,
    remove,
    reset,
    cancel() { epoch++; version++; for (const controller of controllers) controller.abort(); controllers.clear(); },
  };
}
