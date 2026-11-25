# AUTH-07 — Browser, reverse-proxy and Android TWA handoff

**Scope:** backend/browser contracts and opt-in **local synthetic** smoke harness. The user web/PWA frontend, separate Staff frontend, verified Android TWA packaging, Keycloak browser-SSO logout (AUTH-08) and production/VPN deployment (AUTH-10) are **not** implemented by this patch. No SQL migration is required (head remains 0030).

## Exact origins and routing

| Surface | Local exact browser origin | Route prefix | Keycloak realm | Session cookie on local HTTP |
|---|---|---|---|---|
| User Web/PWA and Android TWA | `http://app.localhost:8080` | `/auth/user/*`, `/api/v1/me*`, `/api/v1/conversations*` and other protected product APIs | `theia-users` | `theia_user_session_local` |
| Staff back-office | `http://admin.localhost:8080` | `/auth/staff/*`, `/api/v1/admin/*` | `theia-staff` | `theia_staff_session_local` |
| Keycloak identity provider | `http://auth.localhost:8180` | `/realms/theia-users/*`, `/realms/theia-staff/*` | separate realms | Keycloak-owned cookies only |

These **HTTP** origins are permitted only in local development; never configure a real TWA to trust or use them. Outside `APP_ENV=local`, each app/staff/Keycloak origin **must be HTTPS**, with exact `AUTH_USER_ORIGIN`, `AUTH_STAFF_ORIGIN`, `AUTH_KEYCLOAK_BASE_URL` values and matching Keycloak confidential-client redirect URI allowlists. The public origin (scheme + host + port when non-default) must match what the browser actually sees. `app` and `admin` must not share an origin. OIDC callback URLs are fixed: `${AUTH_USER_ORIGIN}/auth/user/callback` and `${AUTH_STAFF_ORIGIN}/auth/staff/callback`.

For production/shared hosting, the **edge** terminates HTTPS and serves the frontend shell/assets on its exact hostname, while reverse-proxying `/auth/{user|staff}/*` and the corresponding `/api/v1/*` to the Go API **on the same browser origin**. The Go API is not a CORS/Bearer-token API for browsers. The frontend must always call relative paths, with `credentials: 'same-origin'`; never read `Set-Cookie`, use `Access-Control-Allow-Origin: *`, pass an OIDC ID/access token, or store a CSRF/session token in web storage. `AUTH_TRUSTED_PROXY_CIDRS` must contain only reviewed *immediate proxy* CIDRs, never `0.0.0.0/0` or `::/0`. The edge must **strip inbound** `X-Forwarded-*` headers and set exactly one `X-Forwarded-Proto` and `X-Forwarded-Host` matching the visible canonical host. It **must preserve the original Host** when forwarding to Go. A spoofed forwarded header cannot authorize a different Host.

Deployment sketch (adapt to your actual proxy; not a ready-to-deploy config):

```nginx
# Example within an HTTPS `server` for the exact user hostname.
# Exact app/server-name, TLS and upstream are infrastructure-owned.
location ^~ /auth/user/ {
  proxy_pass http://theia_go_api;
  proxy_set_header Host $http_host;
  proxy_set_header X-Forwarded-Proto $scheme;
  proxy_set_header X-Forwarded-Host $http_host;
}
location ^~ /api/v1/ {
  proxy_pass http://theia_go_api;
  proxy_set_header Host $http_host;
  proxy_set_header X-Forwarded-Proto $scheme;
  proxy_set_header X-Forwarded-Host $http_host;
}
# All other requests serve the user PWA shell and static assets.
# In the Staff HTTPS virtual host, route /auth/staff/ and /api/v1/admin/ only.
```

Do not forward the entire user API on the Staff hostname or vice versa. Cache neither `/auth/*` nor the private product/admin APIs (the API sets `Cache-Control: no-store`). The web server must render the app shell for `/` and valid same-origin frontend deep links; Go's bare API-only root currently returns 404 unless a frontend is mounted.

## Browser login and session contract (source: `contracts/openapi/openapi.yaml`)

1. Link the **top-level browser navigation** to `GET /auth/user/login` (or `/auth/staff/login`). Optional `?return_to=%2Fconversations%2F...` is a **relative same-origin path** (max 1024 bytes); invalid, protocol-relative, encoded slash/backslash, dot-segment, fragment and duplicate values fail with `400 AUTH_RETURN_PATH_INVALID`. User login also requires the signed Keycloak SMS proof gate (AUTH-06).
2. The BFF sets a 5-minute HttpOnly OIDC login cookie, performs `302` to Keycloak's realm. The browser completes Keycloak phone/SMS OTP (User) or Email/Password (Staff). The frontend must **not build an OTP form** unless explicitly replacing the Keycloak theme in a later approved design.
3. Keycloak redirects to fixed BFF `/auth/{kind}/callback`. The BFF verifies the login cookie + state + PKCE + signed ID token + nonce + issuer + audience (plus User SMS proof). It consumes the SQL transaction **exactly once**, issues the host-only app session cookie and responds `303 Location: <sealed return_to>`, default `/`. A callback `return_to` parameter cannot change this destination.
4. On SPA boot and after a successful callback, `GET /auth/{kind}/session` with same-origin cookies. `200`: `{ authenticated: true, principalType: "user" | "staff", principalId: "<uuid>", expiresAt: "<UTC timestamp>", csrfToken: "<opaque>" }`, plus `role: "admin"` on Staff. The response has `Cache-Control: no-store` and `Vary: Cookie`; never log, persist, or display the CSRF value. `401`: unauthenticated/expired/revoked → show login. `503`: Auth or SQL temporarily unavailable → retry/error state **not** an authenticated user. `400`: mismatched origin/host, fix proxy configuration, not a credential issue.
5. For User, `GET /api/v1/me` reads onboarding status; the frontend selects Onboarding or the main experience. For Staff, `GET /api/v1/admin/me` confirms Back-Office access. Neither `principalId` nor role may come from a caller-supplied header.
6. For all **mutating** protected APIs and `POST /auth/{kind}/logout`, send `X-CSRF-Token` with the current session's CSRF value, same-origin credentials, and ordinary browser `Origin`/Fetch Metadata. Never retry a `403 CSRF_*` blindly; refresh session when appropriate. The backend enforces this on both protected User and Staff routes.
7. `POST /auth/{kind}/logout` returns `204`, revokes the *Theia* SQL-backed session and clears its cookie. Existing Keycloak SSO may still issue another login without an OTP/password until AUTH-08 SSO logout is implemented and tested. UI must not claim global logout.

**Cookie contract:** HTTPS uses `__Host-theia_user_session` / `__Host-theia_staff_session`, `Secure`, `HttpOnly`, `SameSite=Lax`, `Path=/`, *no Domain*. HTTP local development uses `_local` names on separate `.localhost` hosts. A PWA or TWA does not receive/handle a bearer token; cookies remain browser-managed.

**Failure/UI states to hand off to frontend:** 401 session expired/revoked; 403 wrong CSRF/Staff inactive; 409 onboarding required on gated product operations; 503 IdP/SQL unavailable; OTP delivery/expiry/rate limit failures are **Keycloak theme** states; no OTP codes or PII in query strings or logs. Never automatically switch to Dev Session as a fallback.

## Opt-in same-origin browser smoke harness

The Go API serves a *static*, restrictive-CSP diagnostic page only when `AUTH_BROWSER_SMOKE_ENABLED=true`, `AUTH_BFF_ENABLED=true`, `APP_ENV=local` and both configured browser origins are loopback/.localhost. The pages are absent otherwise. It cannot grant an identity: it exercises the normal Keycloak/BFF/Staff guards. The script only shows HTTP status, principal kind and session expiry; the CSRF token is used in memory for logout, never displayed. The test is **not** a frontend and is never a real-user release gate.

With the existing `.env` / `.env.auth` setup from AUTH-01 through AUTH-06, explicitly set in `.env` for a **User** smoke test:

```dotenv
ALLOW_DEV_LOGIN=false
AUTH_BFF_ENABLED=true
AUTH_USER_LOGIN_ENABLED=true
AUTH_USER_SMS_FLOW_VERIFIED=true
AUTH_STAFF_LOGIN_ENABLED=false
AUTH_BROWSER_SMOKE_ENABLED=true
```

Ensure the AUTH-05 SPI is built/installed, the local Mock SMS transport is selected, the realm SMS-only flow and signed note mapper are configured, the User BFF client is enabled via `scripts/auth/activate_user_bff.py`, and SQL Server migrations through 0030 were applied. The optional `compose.auth-bff.yaml` now overrides the base `ALLOW_DEV_LOGIN` setting using `${ALLOW_DEV_LOGIN:-true}` so `.env` can disable it. Keep secrets in `.env.auth`, never paste its expanded Compose configuration.

```sh
docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-sms.yaml -f compose.auth-bff.yaml \
  --profile auth up -d --build migrate api keycloak
# Open in a local browser:
# http://app.localhost:8080/__auth-smoke/user
```

Use **synthetic phone and private Mock SMS outbox only**. Press “Start Keycloak login”: it returns through the validated local `return_to` to the smoke page, avoiding the API-only root 404. Use “Check session and protected API” and “Revoke application session”. Verify `401` after logout, and separate User/Staff host isolation. To smoke-test Staff instead, enable Staff login, provision one explicit synthetic Staff identity using AUTH-04, and open `http://admin.localhost:8080/__auth-smoke/staff`. A disabled realm returns 503; a wrong host returns 400. Turn the smoke flag off afterward; remove it entirely in shared environments.

## Android TWA / Digital Asset Links (frontend release preparation only)

- The **TWA launches the exact public HTTPS User web origin**, not the Keycloak origin, Admin origin, a localhost callback, or a custom in-app browser with separate cookie storage. Login navigates to Keycloak in the user's browser context and returns to the **same** User-origin OIDC callback and session cookie. Test on real signed Android builds; desktop/PWA smoke does not establish Android behavior.
- The User frontend/edge must serve the Android Digital Asset Links file at `https://<user-origin>/.well-known/assetlinks.json`, status 200 and `Content-Type: application/json`, containing the actual signed Android package ID and SHA-256 signing certificate fingerprint. For Play App Signing, use the **app-signing certificate** corresponding to the installed release, not merely the upload key. The API backend intentionally **does not auto-publish** a sample association file. The Admin and Keycloak hosts must not serve an association for the User TWA.
- Before frontend release, run the local validator (with **real** package/fingerprint and the planned external HTTPS User origin):

```sh
python3 scripts/auth/validate_twa_assetlinks.py \
  --file path/to/actual/assetlinks.json \
  --package ir.example.theia \
  --fingerprint 'AA:BB:...:32-BYTES' \
  --origin https://app.example.org
```

The fingerprint in that example is a **placeholder** and must be replaced with the real 32-byte colon-delimited SHA-256 value; the validator rejects the placeholder. Validate the actual served file using `curl -i https://app.example.org/.well-known/assetlinks.json`, then perform Chrome/TWA digital asset link verification and login/logout deep-link tests on the signed release build. Wrong signing keys, environment host drift or a changed package name break TWA verification. For multiple release signing certificates, enumerate only the intended keys.

## Test checklist and limits

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v
(cd apps/api && go build ./... && go vet ./... && go test ./...)
```

Go tests include return-path forgery/replay, local-only smoke access, cache headers and trusted-proxy spoofing. Full live testing still requires disposable Keycloak/PostgreSQL, SQL Server, actual Chrome redirect/cookie behavior and the signed Android TWA. **AUTH-08** owns Keycloak SSO/backchannel logout, cross-session revocation and security lifecycle. **AUTH-09** retires Dev Session only after both frontends are migrated. **AUTH-10** owns TLS/private ingress/VPN and real-user safety/retention approvals.

Official TWA references: [Chrome TWA and Digital Asset Links](https://developer.chrome.com/docs/android/trusted-web-activity/android-for-web-devs), [Google Digital Asset Links format](https://developers.google.com/identity/credential-sharing/digital-asset-links), and [Android App Links signing certificates](https://developer.android.com/training/app-links/configure-assetlinks).


### AUTH-08 addendum — SSO handoff after backend cutover

The AUTH-07 smoke page now also offers an SSO logout button. Frontends should
use `POST /auth/{kind}/logout/sso` with `X-CSRF-Token`, then navigate the **top-level
browser** to the returned fixed `logoutUrl`. This API does not automatically
clear the Keycloak browser cookie; user confirmation may be required. Existing
`POST /auth/{kind}/logout` remains app-only. `POST /auth/{kind}/logout-all`
revokes Theia sessions on other devices but not their separate Keycloak SSO
sessions. See `docs/operations/auth-session-lifecycle-auth08.md` for the
back-channel, client registration and full safety checklist.
