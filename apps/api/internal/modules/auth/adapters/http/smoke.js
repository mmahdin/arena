// Local synthetic-only smoke: no OAuth tokens, storage, cross-origin fetches,
// credentials in URLs, or raw user/clinical response data are inspected.
'use strict';
(() => {
  const kind = document.body.dataset.kind;
  if (kind !== 'user' && kind !== 'staff') return;
  const status = document.getElementById('status');
  const result = document.getElementById('result');
  const logout = document.getElementById('logout');
  const ssoLogout = document.getElementById('ssoLogout');
  let csrfToken = '';
  const sessionPath = `/auth/${kind}/session`;
  const protectedPath = kind === 'staff' ? '/api/v1/admin/me' : '/api/v1/me';

  const request = (path, options = {}) => fetch(path, {
    credentials: 'same-origin', cache: 'no-store', redirect: 'error', ...options,
  });
  const show = (message, summary) => {
    status.textContent = message;
    result.textContent = summary;
  };

  async function refresh() {
    csrfToken = '';
    logout.disabled = true;
    ssoLogout.disabled = true;
    try {
      const response = await request(sessionPath);
      if (response.status === 401) {
        show('Not authenticated. Use Keycloak login.', 'Session: 401');
        return;
      }
      if (!response.ok) {
        show('Session unavailable.', `Session HTTP ${response.status}`);
        return;
      }
      const session = await response.json();
      if (session.authenticated !== true || session.principalType !== kind ||
          typeof session.csrfToken !== 'string' || !session.csrfToken) {
        show('Invalid session contract.', 'Session returned unexpected fields');
        return;
      }
      csrfToken = session.csrfToken; // in memory only; never displayed
      logout.disabled = false;
      ssoLogout.disabled = false;
      const protectedResponse = await request(protectedPath);
      show('Authenticated.',
        `Session: 200\nProtected API: ${protectedResponse.status}\nRealm: ${kind}\nExpires: ${String(session.expiresAt || 'unknown')}`);
    } catch {
      show('Network or same-origin failure.', 'Check the reverse proxy and browser network panel.');
    }
  }

  document.getElementById('refresh').addEventListener('click', refresh);
  logout.addEventListener('click', async () => {
    if (!csrfToken) return;
    try {
      const response = await request(`/auth/${kind}/logout`, {
        method: 'POST', headers: {'X-CSRF-Token': csrfToken},
      });
      csrfToken = '';
      logout.disabled = true;
      ssoLogout.disabled = true;
      show(response.status === 204 ? 'Theia session revoked.' : 'Logout failed.',
        `App-only logout HTTP ${response.status}. Keycloak SSO remains active.`);
    } catch {
      show('Logout network failure.', 'The application session may still be active.');
    }
  });
  ssoLogout.addEventListener('click', async () => {
    if (!csrfToken) return;
    try {
      const response = await request(`/auth/${kind}/logout/sso`, {
        method: 'POST', headers: {'X-CSRF-Token': csrfToken},
      });
      if (!response.ok) { show('SSO handoff failed.', `HTTP ${response.status}`); return; }
      const body = await response.json();
      const next = new URL(body.logoutUrl);
      // This is the local-only smoke harness: the browser must go directly to
      // the exact local Keycloak host, never arbitrary data from the response.
      if (next.hostname !== 'auth.localhost' ||
          !next.pathname.endsWith('/protocol/openid-connect/logout') ||
          next.searchParams.get('client_id') !== `theia-${kind === 'user' ? 'users' : 'staff'}-bff`) {
        show('Unexpected Keycloak logout target.', 'Refusing cross-origin navigation.');
        return;
      }
      csrfToken = '';
      ssoLogout.disabled = true;
      logout.disabled = true;
      window.location.assign(next.toString());
    } catch {
      show('SSO handoff failed.', 'Application session may already be revoked.');
    }
  });
  refresh();
})();
