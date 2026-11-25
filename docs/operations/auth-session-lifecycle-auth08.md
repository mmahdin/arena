# AUTH-08 — Keycloak SSO and Theia session lifecycle

**Scope:** backend + local Keycloak client configuration, synthetic/test-only. Depends on AUTH-07, migrations through **0031**, Keycloak 26.8.2, and the two existing confidential BFF clients. **No frontend production release, Dev Session removal, remote-Keycloak global sign-out, phone recovery, or real-user launch is included.**

## 1. Two distinct session owners

Theia owns independent, hashed, SQL-backed browser session tokens. Keycloak owns SSO cookies and the OIDC user session. Clearing the Theia cookie cannot clear the Keycloak cookie. The legacy `POST /auth/{kind}/logout` retains its previous **204 app-only** contract for clients already relying on it.

AUTH-08 adds the following routes for each of `user` and `staff`:

| Method | Route | Security and behavior |
| --- | --- | --- |
| `POST` | `/auth/{kind}/logout/sso` | Host-bound Theia cookie + `X-CSRF-Token` + Origin checks. Atomically revokes current Theia session and returns `200` JSON `logoutUrl` (server-owned, no input redirect) |
| `POST` | `/auth/{kind}/logout-all` | Same CSRF checks. Revokes **all existing Theia sessions for the verified principal**, including other browsers/devices, and returns the same browser SSO handoff |
| `POST` | `/auth/{kind}/backchannel-logout` | **No browser cookie or CSRF.** Private server-to-server signed `logout_token` from the configured Keycloak realm; validates RS256, issuer, audience, event, jti, iat, exp, no nonce, sub/sid; serializable replay protection; revokes matching Theia sessions |

The `logout/sso` or `logout-all` JSON response has exactly these fields:

```json
{
  "logoutUrl": "http://auth.localhost:8180/realms/theia-users/protocol/openid-connect/logout?client_id=theia-users-bff&post_logout_redirect_uri=http%3A%2F%2Fapp.localhost%3A8080%2F",
  "theiaSessionsRevoked": "current",
  "keycloakSSOStatus": "browser_action_required"
}
```

The example URL is local only. **The browser must perform a top-level navigation to the returned URL after a `200`.** Because the app intentionally does not store an ID token for use as `id_token_hint`, Keycloak may show a logout confirmation. **Never tell the user Keycloak SSO ended solely because the Theia API returned `200`.** Once Keycloak completes logout, it can send signed back-channel notices for matching realm client sessions. A remote device may still have its own valid Keycloak SSO session after `/logout-all`, enabling future reauthentication. Therefore do not label `/logout-all` as global Keycloak logout or “sign out of all identity provider devices.”

For Web/PWA/TWA, use `fetch` with same-origin credentials and the session's CSRF token, then top-level navigate. Do not use an iframe or background `fetch` for Keycloak SSO logout. Do not persist `logoutUrl`, OAuth tokens, Keycloak cookies, or OTP in local storage. If the browser closes before navigating, Theia is logged out but Keycloak may not be. If the Keycloak logout page fails, app sessions remain revoked.

## 2. Apply migration and configure *existing* Keycloak instances

Apply `apps/api/db/migrations/0031_auth_backchannel_logout.sql` using the normal forward-only SQL Server migration runner. It adds only **hashed replay event keys**, **hashed logout fences**, and a **per-principal logout-all watermark**. Neither JWTs, OTPs, raw Keycloak SIDs/JTIs, nor personal text appear in the new tables.

Newly imported local realms contain back-channel and valid post-logout redirect settings. **Realm import does not upgrade an existing PostgreSQL realm.** After starting Keycloak and the API on their shared Compose network:

```bash
python3 scripts/auth/configure_keycloak_logout.py
python3 scripts/auth/configure_keycloak_logout.py --apply
python3 scripts/auth/configure_keycloak_logout.py
```

This checks both fixed clients and changes only their logout-specific settings. It never enables the User BFF client, changes SMS/password authentication, or creates/deactivates an account. Local Keycloak calls the Go API at `http://api:8080/auth/user/backchannel-logout` or `/auth/staff/backchannel-logout` over the **internal Docker network**, not through the app's browser origin. No public ingress route is needed or recommended. Keycloak must be able to resolve `api` on the same Compose network. The configured logout redirect allowlist is exactly `http://app.localhost:8080/` or `http://admin.localhost:8080/`. If the Keycloak client configuration drifts, stop and inspect before reapplying.

For staging/production use real TLS/strict ingress and configure the Keycloak Admin client **back-channel logout URL to a private, reachable API service address** and post-logout redirect to an exact approved HTTPS frontend root. Do not copy the `api:8080` address or loopback redirect into deployed realms. Keep User/Staff clients, realms and origin hosts separate. The API verifies *signed* notifications, not IP address alone; private ingress and rate limiting also matter.

## 3. Back-channel security, race handling and retention

The callback accepts a single bounded `application/x-www-form-urlencoded` `logout_token` field. It verifies the JWT using **only** the configured Keycloak issuer's JWKS; no token-provided key URLs, algorithm downgrade, browser-provided actor ID, or cross-realm sid are accepted. Valid duplicate JWTs are safe `200` no-ops. Validation failures return `400`; storage/JWKS outages return `503` (not an acknowledged successful logout). Do not log the token, raw `sub`, `sid` or `jti` in HTTP middleware, proxies, tracing or error telemetry.

SQL Server applies replay consumption, logout fence creation, matching-session revocation and the minimal audit event in **one transaction**. The fixed kind+issuer+SID is matched, with subject as an additional check when present; a subject-only OP notification revokes all matching subject sessions. Fences prevent a callback *started before* the Keycloak logout from creating a new Theia session afterward; the `logout-all` watermark provides the analogous principal-wide protection. Fresh login transactions begun after those events are permitted (and still require Keycloak's user OTP or Staff authentication).

The replay and fence tables contain no raw identity-provider IDs; they have expiration indexes. Do **not** truncate them during normal operations. An operator-approved, separately audited cleanup may eventually delete replay events whose `expires_at` is in the past and fences whose `expires_at` is in the past; keep fences until their declared 31-day period to cover the longest Theia session. Replays after the signed JWT has expired are rejected by the verifier anyway.

## 4. Local verification matrix (must pass before relying on SSO propagation)

1. Apply migrations through `0031`; run `go build ./...`, `go vet ./...` and `go test ./...` from `apps/api`; run the repository Python/contract suite. Run disposable SQL tests with `THEIA_DISPOSABLE_AUTH08_SQL_DSN` and `THEIA_DISPOSABLE_AUTH08_SQL_ACK=YES_DESTROY_AFTER_TEST` (**never** on shared SQL Server).
2. Verify both BFF clients' actual Keycloak settings via `configure_keycloak_logout.py`; verify that Keycloak can POST to `http://api:8080`. Start on synthetic users/Mock SMS only. Staff uses a named, provisioned test account.
3. Browser A and B sign in as the same synthetic User/Staff. Check `/auth/{kind}/session` returns `200` on both. Browser A `POST /logout-all` with its CSRF: both Theia cookies must return `401` on their next protected request, while unrelated principal/realm sessions remain valid. Simultaneous/late callback from before logout must be rejected.
4. Browser A logs in again and invokes `/logout/sso`; verify `200`, cleared host-only cookie, fixed `logoutUrl`, Keycloak browser confirmation (if present), final redirect to the exact approved frontend origin, and `401` on stale Theia cookies. Test both realms independently.
5. Logout the user directly from the Keycloak Account console or admin console. Confirm a signed Keycloak back-channel notification reaches the internal API, and its `sid` matches the stored ID-token `sid`; matching Theia sessions become `401`. Check behavior for a subject-only notification if your OP configuration uses it.
6. Invalid signature, wrong issuer/audience/realm, missing jti/event, stale/future iat, expired JWT, nonce present, wrong `Content-Type`, oversized body and duplicated form parameters must not revoke any session. A repeated valid event is an idempotent success. Simulate SQL/JWKS failure: it must not return a successful logout notification response.
7. Inspect traces/proxy logs: no OTP, JWT, session cookie, access/refresh token or full back-channel request bodies. Confirm logout-all cannot affect another Theia principal, and existing Admin authorization/audit routes still require Staff.

## 5. What remains a release blocker

This patch does **not** implement account recovery or changing a phone number. The future recovery process must explicitly handle recycled numbers, SIM-swap risk, account dispute, and revocation of prior sessions; never silently bind a verified phone to another existing user. The full live Keycloak/PostgreSQL + SQL Server + browser/TWA matrix, staff private ingress/VPN, TLS, secret rotation, recovery controls, and approved privacy/clinical release checks remain independent gates. **Do not remove Dev Session (AUTH-09) until both frontend origins use the new session flow and E2E cutover is complete.**

References: OpenID Connect Back-Channel Logout 1.0 (errata set 1), Keycloak 26 Server Administration Guide (RP-Initiated and Back-Channel Logout).
