/** Same-origin JSON transport. The server owns identity and its HttpOnly cookie. */
export class ApiError extends Error {
  constructor(status, code = 'HTTP_ERROR') {
    super(status === 0 ? 'NETWORK_ERROR' : code);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

export async function request(path, { method = 'GET', body, signal } = {}) {
  if (!/^\/api\/v1\/[a-z0-9/-]+$/i.test(path)) throw new Error('Invalid API path');
  let response;
  try {
    response = await fetch(path, {
      method,
      credentials: 'include',
      cache: 'no-store',
      signal,
      headers: body === undefined ? { Accept: 'application/json' } : {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
  } catch (error) {
    if (error?.name === 'AbortError') throw error;
    throw new ApiError(0, 'NETWORK_ERROR');
  }
  if (response.status === 204) return null;
  if (!response.ok) {
    // Do not forward arbitrary server text, stack traces, URLs or credentials to the UI.
    const problem = await response.json().catch(() => null);
    const code = typeof problem?.code === 'string' && /^[A-Z][A-Z0-9_]{0,63}$/.test(problem.code)
      ? problem.code : 'HTTP_ERROR';
    throw new ApiError(response.status, code);
  }
  try {
    return await response.json();
  } catch {
    throw new ApiError(0, 'INVALID_RESPONSE');
  }
}

export function isUnauthorized(error) {
  return error instanceof ApiError && error.status === 401;
}
