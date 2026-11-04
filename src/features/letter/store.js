import { parseLetterDetail, parseLetterSummary } from './model.js';

const initial = () => ({
  userId: null,
  backendIntegrated: false,
  listPhase: 'idle',
  detailPhase: 'idle',
  items: [],
  activeLetter: null,
  error: null,
});

/**
 * Identity-scoped store for Personalized Letters.
 * Until backend letter generation endpoints are enabled (`api === null`),
 * it cleanly reports an empty, non-fabricated state without calling fake URLs.
 */
export function createLetterStore(api = null, { onUnauthorized = () => {} } = {}) {
  let state = initial();
  let epoch = 0;
  let listVersion = 0;
  let detailVersion = 0;
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
    epoch += 1;
    listVersion += 1;
    detailVersion += 1;
    for (const controller of controllers) controller.abort();
    controllers.clear();
    publish(initial());
  }

  function activate(userId) {
    const normalized = userId || null;
    if (state.userId === normalized) return;
    reset();
    if (normalized) {
      publish({
        userId: normalized,
        backendIntegrated: Boolean(api),
        listPhase: api ? 'idle' : 'ready',
      });
    }
  }

  async function loadList() {
    if (!state.userId) return false;
    if (!api) {
      publish({ backendIntegrated: false, listPhase: 'ready', items: [], error: null });
      return true;
    }
    const req = scope();
    const version = ++listVersion;
    publish({ backendIntegrated: true, listPhase: 'loading', error: null });
    try {
      const rawItems = await api.list(req.signal);
      if (!req.current() || version !== listVersion) return false;
      const items = Array.isArray(rawItems) ? rawItems.map(parseLetterSummary) : [];
      publish({ listPhase: 'ready', items, error: null });
      return true;
    } catch (error) {
      if (!req.current() || version !== listVersion || error?.name === 'AbortError') return false;
      if (error?.status === 401) {
        reset();
        onUnauthorized();
        return false;
      }
      publish({ listPhase: 'error', error: error?.code ?? 'NETWORK_ERROR' });
      return false;
    } finally {
      req.done();
    }
  }

  async function openLetter(id) {
    if (!state.userId || !id) return false;
    if (!api) return false;
    const req = scope();
    const version = ++detailVersion;
    publish({ detailPhase: 'loading', error: null });
    try {
      const detail = parseLetterDetail(await api.read(id, req.signal));
      if (!req.current() || version !== detailVersion) return false;
      publish({ activeLetter: detail, detailPhase: 'ready', error: null });
      return true;
    } catch (error) {
      if (!req.current() || version !== detailVersion || error?.name === 'AbortError') return false;
      if (error?.status === 401) {
        reset();
        onUnauthorized();
        return false;
      }
      publish({ detailPhase: 'error', error: error?.status === 404 ? 'RESOURCE_NOT_FOUND' : error?.code ?? 'NETWORK_ERROR' });
      return false;
    } finally {
      req.done();
    }
  }

  async function markRead(id) {
    if (!state.userId || !id || !api?.markRead) return false;
    const req = scope();
    try {
      const result = await api.markRead(id, req.signal);
      if (!req.current()) return false;
      const readAt = typeof result?.readAt === 'string' ? result.readAt : new Date().toISOString();
      publish({
        items: state.items.map((item) => (item.id === id ? { ...item, readAt } : item)),
        activeLetter: state.activeLetter?.id === id ? { ...state.activeLetter, readAt } : state.activeLetter,
      });
      return true;
    } catch (error) {
      if (req.current() && error?.status === 401) {
        reset();
        onUnauthorized();
      }
      return false;
    } finally {
      req.done();
    }
  }

  function closeLetter() {
    detailVersion += 1;
    publish({ activeLetter: null, detailPhase: 'idle', error: null });
  }

  return {
    getState: () => state,
    subscribe(listener) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    activate,
    reset,
    loadList,
    openLetter,
    markRead,
    closeLetter,
    cancel() {
      epoch += 1;
      for (const controller of controllers) controller.abort();
      controllers.clear();
    },
  };
}
