import { localReflectionDate, validReflectionDate } from './api.js';

const blank = () => ({
  userId: null, list: [], listPhase: 'idle', listError: null,
  openedDate: null, entry: null, dayPhase: 'idle', dayError: null,
  saving: false,
});

// All server results are fenced by the current identity and request generation.
export function createReflectionStore(api, { onUnauthorized = () => {} } = {}) {
  let state = blank();
  let epoch = 0;
  let requestId = 0;
  let listRevision = 0;
  const listeners = new Set();
  const pending = new Set();
  const getState = () => state;
  const subscribe = (fn) => { listeners.add(fn); return () => listeners.delete(fn); };
  const publish = (patch) => { state = { ...state, ...patch }; for (const fn of listeners) fn(state); };
  const abort = () => { for (const ctl of pending) ctl.abort(); pending.clear(); };
  const reset = () => { epoch++; requestId++; listRevision++; abort(); publish(blank()); };
  const activate = (userId) => {
    if ((userId || null) === state.userId) return;
    reset();
    publish({ userId: userId || null });
  };
  const scope = () => {
    const version = epoch;
    const controller = new AbortController();
    pending.add(controller);
    return { signal: controller.signal, current: () => epoch === version && !controller.signal.aborted,
      done: () => pending.delete(controller) };
  };
  const failed = (error, req, kind) => {
    if (!req.current()) return;
    if (error?.status === 401) { reset(); onUnauthorized(); return; }
    publish(kind === 'list' ? { listPhase: 'error', listError: error?.code || 'NETWORK_ERROR' }
      : { dayPhase: 'error', dayError: error?.code || 'NETWORK_ERROR' });
  };
  async function list() {
    if (!state.userId) return false;
    const req = scope(); const revision = listRevision; publish({ listPhase: 'loading', listError: null });
    try {
      const entries = await api.list(req.signal);
      if (!req.current() || revision !== listRevision) return false;
      publish({ list: entries, listPhase: 'ready', listError: null }); return true;
    } catch (e) { failed(e, req, 'list'); return false; }
    finally { req.done(); }
  }
  async function open(date = localReflectionDate()) {
    if (!state.userId || !validReflectionDate(date)) return false;
    // Abort any older read when the user selects a different calendar day.
    const req = scope(); const id = ++requestId;
    publish({ openedDate: date, entry: null, dayPhase: 'loading', dayError: null });
    try {
      const entry = await api.read(date, req.signal);
      if (!req.current() || id !== requestId) return false;
      publish({ entry, dayPhase: 'ready', dayError: null }); return true;
    } catch (e) {
      if (!req.current() || id !== requestId) return false;
      if (e?.status === 404) { publish({ entry: null, dayPhase: 'ready', dayError: null }); return true; }
      failed(e, req, 'day'); return false;
    } finally { req.done(); }
  }
  async function save(date, text) {
    if (!state.userId || state.saving || !validReflectionDate(date) || date !== state.openedDate) return false;
    const clean = typeof text === 'string' ? text.trim() : '';
    if (!clean || clean.length > 4000 || /[\p{Cc}\p{Cf}]/u.test(clean.replace(/[\n\t\u200c\u200d]/gu, ''))) {
      publish({ dayError: 'VALIDATION_ERROR' }); return false;
    }
    const req = scope(); const id = requestId;
    publish({ saving: true, dayError: null });
    try {
      const entry = await api.save(date, clean, req.signal);
      if (!req.current() || id !== requestId) return false;
      listRevision++;
      publish({ entry, dayPhase: 'ready', listPhase: 'ready', list: [entry, ...state.list.filter(x => x.date !== date)].sort((a,b) => b.date.localeCompare(a.date)).slice(0, 30) });
      return true;
    } catch (e) {
      if (req.current()) {
        if (e?.status === 401) { reset(); onUnauthorized(); }
        else publish({ dayError: e?.code || 'NETWORK_ERROR' });
      }
      return false;
    }
    finally { if (req.current()) publish({ saving: false }); req.done(); }
  }
  async function remove(date) {
    if (!state.userId || state.saving || date !== state.openedDate) return false;
    const req = scope(); const id = requestId;
    publish({ saving: true, dayError: null });
    try {
      await api.remove(date, req.signal);
      if (!req.current() || id !== requestId) return false;
      listRevision++;
      publish({ entry: null, dayPhase: 'ready', listPhase: 'ready', list: state.list.filter(x => x.date !== date) });
      return true;
    } catch (e) {
      if (req.current()) {
        if (e?.status === 401) { reset(); onUnauthorized(); }
        else publish({ dayError: e?.code || 'NETWORK_ERROR' });
      }
      return false;
    }
    finally { if (req.current()) publish({ saving: false }); req.done(); }
  }
  return { getState, subscribe, reset, activate, list, open, save, remove, cancel: abort };
}
