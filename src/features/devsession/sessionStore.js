/**
 * Single identity state machine: asynchronous responses are generation-fenced and
 * aborted when users switch. No cookie, user profile or intake answers are persisted.
 */
export function createSessionStore(api, { onInvalidate = () => {} } = {}) {
  let state = { phase: 'loading', profile: null, error: null };
  let generation = 0;
  let controller = null;
  const listeners = new Set();

  const publish = (next) => {
    state = { ...state, ...next };
    for (const listener of listeners) listener(state);
  };
  const begin = (phase) => {
    generation++;
    controller?.abort();
    controller = new AbortController();
    publish({ phase, error: null });
    const ticket = generation;
    return { signal: controller.signal, current: () => ticket === generation && !controller.signal.aborted };
  };
  const fail = (err, req, action) => {
    if (!req.current()) return;
    publish({ phase: 'error', profile: null, error: { status: err?.status ?? 0, action } });
  };

  async function bootstrap() {
    const req = begin('loading');
    try {
      const profile = await api.me(req.signal);
      if (req.current()) publish({ phase: 'authenticated', profile });
    } catch (error) {
      if (!req.current()) return;
      if (error?.status !== 401) return fail(error, req, 'bootstrap');
      if (req.current()) publish({ phase: 'anonymous', profile: null, error: null });
    }
  }

  async function login(phone) {
    const req = begin('working');
    onInvalidate();
    publish({ profile: null });
    try {
      await api.start(phone, req.signal);
      if (!req.current()) return;
      const profile = await api.me(req.signal);
      if (req.current()) publish({ phase: 'authenticated', profile });
    } catch (error) { fail(error, req, 'login'); }
  }

  async function leave() {
    const req = begin('working');
    onInvalidate();
    publish({ profile: null });
    try {
      await api.end(req.signal);
      if (!req.current()) return;
      if (req.current()) publish({ phase: 'anonymous', profile: null, error: null });
    } catch (error) { fail(error, req, 'logout'); }
  }

  async function saveProfile(displayName, conversationStyle) {
    if (state.phase !== 'authenticated') return false;
    // Keep identity while saving; only switch/logout can revoke this operation.
    const prior = state.profile;
    const req = begin('saving-profile');
    try {
      const profile = await api.profile(displayName, conversationStyle, req.signal);
      if (req.current()) publish({ phase: 'authenticated', profile, error: null });
      return req.current();
    } catch (error) {
      if (req.current()) {
        if (error?.status === 401) {
          onInvalidate();
          publish({ phase: 'error', profile: null, error: { status: 401, action: 'bootstrap' } });
        } else {
          publish({ phase: 'authenticated', profile: prior, error: { status: error?.status ?? 0, action: 'profile' } });
        }
      }
      return false;
    }
  }

  return {
    getState: () => state,
    subscribe(listener) { listeners.add(listener); return () => listeners.delete(listener); },
    bootstrap, login, leave, saveProfile,
    cancel() { generation++; controller?.abort(); },
  };
}
