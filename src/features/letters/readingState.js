/**
 * Device-local read receipts for personalized letters. The API does not expose
 * read status yet; only opaque letter ids are saved, never letter content.
 */
const PREFIX = 'hamrah:letter-seen:v1:';
const MAX_SAVED_IDS = 300;
const keyFor = userId => `${PREFIX}${encodeURIComponent(userId)}`;
const defaultStorage = () => {
  try { return globalThis.localStorage; } catch { return null; }
};

export function loadReadLetterIds(userId, storage = defaultStorage()) {
  if (typeof userId !== 'string' || !userId || !storage) return [];
  try {
    const stored = JSON.parse(storage.getItem(keyFor(userId)) || '[]');
    return Array.isArray(stored)
      ? [...new Set(stored.filter(id => typeof id === 'string' && id.length > 0 && id.length <= 120))].slice(-MAX_SAVED_IDS)
      : [];
  } catch { return []; }
}

export function markReadLetter(userId, letterId, storage = defaultStorage()) {
  const current = loadReadLetterIds(userId, storage);
  if (typeof userId !== 'string' || !userId || typeof letterId !== 'string' || !letterId) return current;
  if (current.includes(letterId)) return current;
  const next = [...current, letterId].slice(-MAX_SAVED_IDS);
  try { storage?.setItem(keyFor(userId), JSON.stringify(next)); } catch { /* private mode or unavailable storage */ }
  return next;
}
