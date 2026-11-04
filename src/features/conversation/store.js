function pickMostRecentConversation(conversations) {
  if (!Array.isArray(conversations) || conversations.length === 0) return null;
  let best = conversations[0];
  let bestTime = Date.parse(best?.updatedAt ?? '');
  for (let i = 1; i < conversations.length; i += 1) {
    const candidate = conversations[i];
    const time = Date.parse(candidate?.updatedAt ?? '');
    if (!Number.isNaN(time) && (Number.isNaN(bestTime) || time > bestTime)) {
      best = candidate;
      bestTime = time;
    }
  }
  return best;
}

/** Backend-owned chat state. Never persist tester conversations in browser storage. */
export function createConversationStore(api, { onUnauthorized = () => {}, newId = () => crypto.randomUUID() } = {}) {
  const initial = () => ({
    userId: null, conversations: [], historyPhase: 'idle', historyError: null,
    conversationId: null, messages: [], messagesPhase: 'idle', messageError: null,
    pending: null, sending: false, deletingId: null, renamingId: null,
    bookmarks: [], bookmarkPhase: 'idle', bookmarkError: null, bookmarkBusyId: null,
    isDraftNew: false,
  });
  let state = initial();
  let epoch = 0;
  let openVersion = 0;
  let historyVersion = 0;
  let bookmarkVersion = 0;
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
  const handleError = (error, req, phase) => {
    if (!req.current()) return;
    if (error?.name === 'AbortError') return;
    if (error?.status === 401) {
      reset();
      onUnauthorized();
      return;
    }
    if (phase === 'history') publish({ historyPhase: 'error', historyError: error?.code ?? 'NETWORK_ERROR' });
    else publish({ messagesPhase: 'error', messageError: error?.status === 404 ? 'RESOURCE_NOT_FOUND' : error?.code ?? 'NETWORK_ERROR' });
  };
  function reset() {
    epoch++;
    openVersion++;
    historyVersion++;
    bookmarkVersion++;
    for (const controller of controllers) controller.abort();
    controllers.clear();
    publish(initial());
  }
  function activate(userId) {
    if (state.userId === userId) return;
    reset();
    if (userId) publish({ userId });
  }
  async function list() {
    if (!state.userId) return;
    const req = scope();
    const version = ++historyVersion;
    publish({ historyPhase: 'loading', historyError: null });
    try {
      const conversations = await api.list(req.signal);
      if (req.current() && version === historyVersion) publish({ historyPhase: 'ready', conversations });
    } catch (error) {
      if (version === historyVersion) handleError(error, req, 'history');
    } finally { req.done(); }
  }
  async function listBookmarks() {
    if (!state.userId || state.bookmarkBusyId) return false;
    const req = scope();
    const version = ++bookmarkVersion;
    publish({ bookmarkPhase: 'loading', bookmarkError: null });
    try {
      const bookmarks = await api.bookmarks(req.signal);
      if (req.current() && version === bookmarkVersion) publish({ bookmarks, bookmarkPhase: 'ready', bookmarkError: null });
      return req.current() && version === bookmarkVersion;
    } catch (error) {
      if (req.current() && version === bookmarkVersion) {
        if (error?.status === 401) handleError(error, req, 'history');
        else publish({ bookmarkPhase: 'error', bookmarkError: error?.code ?? 'NETWORK_ERROR' });
      }
      return false;
    } finally { req.done(); }
  }
  async function toggleBookmark(message) {
    if (!state.userId || !message?.id || state.bookmarkBusyId || state.bookmarkPhase !== 'ready') return false;
    const req = scope();
    const id = message.id;
    const already = state.bookmarks.some(item => item.id === id);
    const version = ++bookmarkVersion;
    publish({ bookmarkBusyId: id, bookmarkError: null });
    try {
      if (already) await api.removeBookmark(id, req.signal);
      else await api.saveBookmark(id, req.signal);
      if (!req.current() || version !== bookmarkVersion) return false;
      // Reload from backend; never rely on a fabricated saved result.
      const bookmarks = await api.bookmarks(req.signal);
      if (!req.current() || version !== bookmarkVersion) return false;
      publish({ bookmarks, bookmarkPhase: 'ready', bookmarkError: null });
      return true;
    } catch (error) {
      if (req.current() && version === bookmarkVersion) {
        if (error?.status === 401) handleError(error, req, 'history');
        else publish({ bookmarkPhase: 'error', bookmarkError: error?.status === 404 ? 'RESOURCE_NOT_FOUND' : error?.code ?? 'NETWORK_ERROR' });
      }
      return false;
    } finally { if (req.current()) publish({ bookmarkBusyId: null }); req.done(); }
  }
  function startNew() {
    if (!state.userId || state.sending) return false;
    openVersion++;
    publish({ conversationId: null, messages: [], messagesPhase: 'ready', messageError: null, pending: null, isDraftNew: true });
    return true;
  }
  async function open(id) {
    if (!state.userId || state.sending) return false;
    const req = scope();
    const version = ++openVersion;
    publish({ conversationId: id, messages: [], messagesPhase: 'loading', messageError: null, pending: null, isDraftNew: false });
    try {
      const messages = await api.messages(id, req.signal);
      if (req.current() && version === openVersion) publish({ messages, messagesPhase: 'ready' });
      return req.current() && version === openVersion;
    } catch (error) {
      if (version === openVersion) handleError(error, req, 'messages');
      return false;
    } finally { req.done(); }
  }
  /**
   * Restores the user's most recent conversation when entering Chat unless a
   * conversation is already active or the user explicitly started a new draft.
   */
  async function enterChat() {
    if (!state.userId || state.sending || state.pending) return false;
    if (state.conversationId || state.isDraftNew) {
      if (state.historyPhase === 'idle' && typeof api.list === 'function') void list();
      return true;
    }
    if (state.historyPhase === 'ready') {
      const recent = pickMostRecentConversation(state.conversations);
      if (recent) return open(recent.id);
      publish({ conversationId: null, messages: [], messagesPhase: 'ready', messageError: null });
      return true;
    }
    if (typeof api.list !== 'function') {
      publish({ conversationId: null, messages: [], messagesPhase: 'ready', messageError: null });
      return true;
    }
    const req = scope();
    const currentOpenVersion = ++openVersion;
    const currentHistoryVersion = ++historyVersion;
    publish({
      historyPhase: 'loading',
      historyError: null,
      messagesPhase: 'loading',
      messageError: null,
    });
    try {
      const conversations = await api.list(req.signal);
      if (!req.current()) return false;
      if (currentHistoryVersion === historyVersion) {
        publish({ historyPhase: 'ready', conversations, historyError: null });
      }
      if (currentOpenVersion !== openVersion) return false;
      const recent = pickMostRecentConversation(conversations);
      if (!recent) {
        publish({ conversationId: null, messages: [], messagesPhase: 'ready', messageError: null, isDraftNew: false });
        return true;
      }
      publish({ conversationId: recent.id, messages: [], messagesPhase: 'loading', messageError: null, isDraftNew: false });
      const messages = await api.messages(recent.id, req.signal);
      if (!req.current() || currentOpenVersion !== openVersion) return false;
      publish({ conversationId: recent.id, messages, messagesPhase: 'ready', messageError: null });
      return true;
    } catch (error) {
      if (currentHistoryVersion === historyVersion && state.historyPhase === 'loading') {
        handleError(error, req, 'history');
      }
      if (currentOpenVersion === openVersion) {
        handleError(error, req, 'messages');
      }
      return false;
    } finally {
      req.done();
    }
  }
  async function performTurn(turn) {
    if (!state.userId || state.sending) return false;
    const req = scope();
    const version = openVersion;
    publish({ sending: true, messagesPhase: 'ready', messageError: null, pending: turn });
    const current = () => req.current() && version === openVersion;
    try {
      let conversationId = turn.conversationId;
      if (!conversationId) {
        const created = await api.create(req.signal);
        if (!current()) return false;
        conversationId = created.id;
        turn = { ...turn, conversationId };
        publish({ conversationId, pending: turn, isDraftNew: false });
      }
      // Always reuse the identical clientTurnId + text, including after a lost response.
      await api.turn(conversationId, turn.clientTurnId, turn.text, req.signal);
      if (!current()) return false;
      const messages = await api.messages(conversationId, req.signal);
      if (!current()) return false;
      publish({ messages, pending: null, messagesPhase: 'ready', messageError: null, isDraftNew: false });
      // Refresh titles/timestamps, but never delay a successful send on list availability.
      void list();
      return true;
    } catch (error) {
      if (current()) handleError(error, req, 'messages');
      return false;
    } finally {
      if (current()) publish({ sending: false });
      req.done();
    }
  }
  async function send(text) {
    const clean = text.trim();
    if (!clean || [...clean].length > 4000 || state.sending || state.pending || state.messagesPhase === 'loading' || !state.userId) return false;
    return performTurn({ clientTurnId: newId(), text: clean, conversationId: state.conversationId });
  }
  async function retry() {
    if (!state.pending || state.sending || state.messageError === 'RESOURCE_NOT_FOUND') return false;
    return performTurn(state.pending);
  }
  async function rename(id, title) {
    const clean = typeof title === 'string' ? title.trim() : '';
    if (!state.userId || state.renamingId || state.deletingId || !clean ||
      [...clean].length > 60 || /[\u0000-\u001f\u007f]/.test(clean) ||
      !state.conversations.some(item => item.id === id)) return false;
    const req = scope();
    publish({ renamingId: id });
    try {
      const updated = await api.rename(id, clean, req.signal);
      if (!req.current()) return false;
      // Ignore any earlier list started before the rename was confirmed.
      historyVersion++;
      publish({
        conversations: state.conversations.map(item => item.id === id ? updated : item),
        historyPhase: 'ready', historyError: null,
      });
      return true;
    } catch (error) {
      if (req.current() && error?.status === 401) handleError(error, req, 'history');
      return false;
    } finally {
      if (req.current()) publish({ renamingId: null });
      req.done();
    }
  }
  async function remove(id) {
    if (!state.userId || state.sending || state.deletingId) return false;
    const req = scope();
    publish({ deletingId: id, historyError: null });
    try {
      await api.remove(id, req.signal);
      if (!req.current()) return false;
      bookmarkVersion++;
      publish({ bookmarkPhase: 'ready', conversations: state.conversations.filter((conversation) => conversation.id !== id),
        bookmarks: state.bookmarks.filter((item) => item.conversationId !== id) });
      if (state.conversationId === id) startNew();
      return true;
    } catch (error) {
      if (req.current() && error?.status === 404) {
        bookmarkVersion++;
        publish({ bookmarkPhase: 'ready', conversations: state.conversations.filter((conversation) => conversation.id !== id),
          bookmarks: state.bookmarks.filter((item) => item.conversationId !== id), historyError: 'RESOURCE_NOT_FOUND' });
        if (state.conversationId === id) startNew();
      } else handleError(error, req, 'history');
      return false;
    } finally {
      if (req.current()) publish({ deletingId: null });
      req.done();
    }
  }
  return {
    getState: () => state,
    subscribe(listener) { listeners.add(listener); return () => listeners.delete(listener); },
    activate, reset, list, open, enterChat, startNew, send, retry, remove, rename, listBookmarks, toggleBookmark,
    cancel() { epoch++; for (const controller of controllers) controller.abort(); controllers.clear(); },
  };
}
