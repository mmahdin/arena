import { request } from '../../core/api/http.js';

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const route = (id, suffix = '') => {
  if (typeof id !== 'string' || !UUID.test(id)) throw new Error('Invalid conversation ID');
  return `/api/v1/conversations/${id}${suffix}`;
};
const items = (value, kind) => {
  if (!value || !Array.isArray(value.items)) throw new Error('Invalid conversation API response');
  if (kind === 'conversation' && !value.items.every((item) =>
    item && typeof item.id === 'string' && UUID.test(item.id) && typeof item.title === 'string' &&
    typeof item.createdAt === 'string' && typeof item.updatedAt === 'string')) throw new Error('Invalid conversation list');
  if (kind === 'message' && !value.items.every((item) =>
    item && typeof item.id === 'string' && UUID.test(item.id) && ['user', 'assistant'].includes(item.role) &&
    typeof item.content === 'string' && typeof item.createdAt === 'string')) throw new Error('Invalid message list');
  return value.items;
};

const validateMessageId = id => {
  if (typeof id !== 'string' || !UUID.test(id)) throw new Error('Invalid message ID');
  return id;
};
const savedItems = value => {
  if (!value || !Array.isArray(value.items) || !value.items.every(item =>
    item && typeof item.id === 'string' && UUID.test(item.id) &&
    typeof item.conversationId === 'string' && UUID.test(item.conversationId) &&
    ['user','assistant'].includes(item.role) && typeof item.content === 'string' &&
    typeof item.createdAt === 'string' && typeof item.savedAt === 'string')) {
    throw new Error('Invalid saved messages response');
  }
  return value.items;
};

export const conversationApi = {
  async bookmarks(signal) { return savedItems(await request('/api/v1/me/bookmarks', {signal})); },
  async saveBookmark(id, signal) { return request(`/api/v1/messages/${validateMessageId(id)}/bookmark`, { method: 'PUT', signal }); },
  async removeBookmark(id, signal) { return request(`/api/v1/messages/${validateMessageId(id)}/bookmark`, { method: 'DELETE', signal }); },
  async list(signal) { return items(await request('/api/v1/conversations', { signal }), 'conversation'); },
  async create(signal) {
    const value = await request('/api/v1/conversations', { method: 'POST', signal });
    if (!value || typeof value.id !== 'string' || !UUID.test(value.id)) throw new Error('Invalid created conversation');
    return value;
  },
  async messages(id, signal) { return items(await request(route(id, '/messages'), { signal }), 'message'); },
  async turn(id, clientTurnId, message, signal) {
    if (typeof clientTurnId !== 'string' || !UUID.test(clientTurnId)) throw new Error('Invalid turn ID');
    return request(route(id, '/turns'), { method: 'POST', body: { clientTurnId, message }, signal });
  },
  async rename(id, title, signal) {
    if (typeof title !== 'string' || !title.trim() || [...title.trim()].length > 60) throw new Error('Invalid conversation title');
    const value = await request(route(id), {method: 'PATCH', body: {title: title.trim()}, signal});
    if (!value || value.id !== id || typeof value.title !== 'string' ||
      typeof value.createdAt !== 'string' || typeof value.updatedAt !== 'string') throw new Error('Invalid renamed conversation');
    return value;
  },
  async remove(id, signal) { return request(route(id), { method: 'DELETE', signal }); },
};
