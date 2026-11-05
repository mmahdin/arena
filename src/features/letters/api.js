import { request } from '../../core/api/http.js';

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** Owner identity comes only from the HttpOnly backend session cookie. */
export const letterApi = {
  async list({ signal } = {}) {
    const value = await request('/api/v1/me/letters', { signal });
    if (!value || !Array.isArray(value.items)) throw new Error('Invalid letters list response');
    return value.items;
  },
  async generate({ signal } = {}) {
    return request('/api/v1/me/letters', { method: 'POST', body: { consent: true }, signal });
  },
  async remove(id, { signal } = {}) {
    if (typeof id !== 'string' || !UUID.test(id)) throw new Error('Invalid letter id');
    return request(`/api/v1/me/letters/${id}`, { method: 'DELETE', signal });
  },
};
