import { request } from '../../core/api/http.js';

const DATE = /^\d{4}-\d{2}-\d{2}$/;

export function validReflectionDate(date) {
  if (typeof date !== 'string' || !DATE.test(date)) return false;
  const parsed = new Date(`${date}T12:00:00Z`);
  return !Number.isNaN(parsed.getTime()) && parsed.toISOString().slice(0, 10) === date;
}

export function localReflectionDate(now = new Date()) {
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export function parseReflection(data) {
  if (!data || !validReflectionDate(data.date) || typeof data.text !== 'string' ||
      typeof data.createdAt !== 'string' || typeof data.updatedAt !== 'string') {
    throw new Error('Invalid reflection response');
  }
  return { date: data.date, text: data.text, createdAt: data.createdAt, updatedAt: data.updatedAt };
}

function path(date) {
  if (!validReflectionDate(date)) throw new Error('Invalid reflection date');
  return `/api/v1/me/daily-reflections/${date}`;
}

export const reflectionApi = {
  async list(signal) {
    const result = await request('/api/v1/me/daily-reflections', { signal });
    if (!Array.isArray(result?.items)) throw new Error('Invalid reflection list');
    return result.items.map(parseReflection);
  },
  async read(date, signal) {
    return parseReflection(await request(path(date), { signal }));
  },
  async save(date, text, signal) {
    return parseReflection(await request(path(date), { method: 'PUT', body: { text }, signal }));
  },
  async remove(date, signal) {
    return request(path(date), { method: 'DELETE', signal });
  },
};
