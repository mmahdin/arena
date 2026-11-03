import { request } from '../../core/api/http.js';

export function parsePrivacyPreferences(data) {
  if (!data || typeof data.conversationSharingEnabled !== 'boolean') {
    throw new Error('Invalid privacy preferences response');
  }
  return data.conversationSharingEnabled;
}

export const privacyPreferencesApi = {
  async read(signal) {
    return parsePrivacyPreferences(await request('/api/v1/me/privacy-preferences', { signal }));
  },
  async save(enabled, signal) {
    if (typeof enabled !== 'boolean') throw new Error('Invalid privacy choice');
    return parsePrivacyPreferences(await request('/api/v1/me/privacy-preferences', {
      method: 'PUT', body: { conversationSharingEnabled: enabled }, signal,
    }));
  },
};

// Preference metadata only. This store does not send or share chat messages.
export function createPrivacyPreferencesStore(api, { onUnauthorized = () => {} } = {}) {
  let owner = null;
  let revision = 0;
  let pending = null;
  let state = { enabled: true, loading: false, saving: false, loaded: false, error: false };
  const listeners = new Set();
  const publish = (changes) => {
    state = { ...state, ...changes };
    for (const listener of listeners) listener(state);
  };
  const begin = () => {
    revision++;
    pending?.abort();
    pending = new AbortController();
    const currentRevision = revision;
    const currentOwner = owner;
    return { signal: pending.signal, current: () => revision === currentRevision && owner === currentOwner && !pending?.signal.aborted };
  };
  const handleError = (error, req) => {
    if (!req.current()) return;
    publish({ loading: false, saving: false, error: true });
    if (error?.status === 401) onUnauthorized();
  };
  return {
    getState: () => state,
    subscribe(listener) { listeners.add(listener); return () => listeners.delete(listener); },
    activate(userId) {
      if (owner === userId) return;
      owner = userId;
      revision++;
      pending?.abort();
      publish({ enabled: true, loading: false, saving: false, loaded: false, error: false });
    },
    async load() {
      if (!owner) return false;
      const req = begin();
      publish({ loading: true, error: false });
      try {
        const enabled = await api.read(req.signal);
        if (!req.current()) return false;
        publish({ enabled, loading: false, loaded: true });
        return true;
      } catch (error) { handleError(error, req); return false; }
    },
    async toggle() {
      if (!owner || !state.loaded || state.loading || state.saving) return false;
      const req = begin();
      const next = !state.enabled;
      publish({ saving: true, error: false });
      try {
        const enabled = await api.save(next, req.signal);
        if (!req.current()) return false;
        publish({ enabled, saving: false });
        return true;
      } catch (error) { handleError(error, req); return false; }
    },
    reset() {
      owner = null;
      revision++;
      pending?.abort();
      publish({ enabled: true, loading: false, saving: false, loaded: false, error: false });
    },
  };
}
