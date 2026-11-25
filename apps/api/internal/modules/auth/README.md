# Auth/BFF module (AUTH-03/04)

This module is the **only** backend owner of OIDC login transactions, verified
Keycloak identity mapping, and revocable opaque web sessions. It is deliberately
independent of AI Core and Admin read models. PostgreSQL is **Keycloak-owned**;
all Theia auth tables are in SQL Server migration `0024_auth_identity_sessions.sql`.

## Boundaries

- `oidc/`: fixed-issuer discovery, Authorization Code + PKCE S256, confidential
  client token exchange, RS256 ID-token verification using only that issuer's
  configured JWKS (no `alg=none`, `jku` or user-provided URLs). The ID token is
  verified for issuer, audience/authorized party, nonce, and time claims.
- `service.go`: random 256-bit state, nonce and verifier; AES-GCM-sealed durable
  one-time login transaction; opaque 256-bit session bearer and HMAC-derived CSRF.
- `adapters/mssql/`: serializable issuer/subject first-login mapping, staff
  allowlist lookup, SQL-backed session/active-principal checks, atomic login/
  logout audit writes; no raw user phone or OAuth token persistence.
- `adapters/http/`: host- and scheme-bound routes, independent cookies and
  `RequireUser`/`RequireStaff`. The staff identity is not derived from headers.

The **legacy synthetic** `/api/v1/dev/*` and dev context helper remain until
AUTH-09 cutover. AUTH-06 provides `internal/platform/principal` as a neutral, verified
user identity adapter used by both real and transitional Dev Session guards.

## Deliberate gates

`AUTH_BFF_ENABLED=false` by default: old reserved handlers stay `503`.
When set true, OIDC discovery must succeed or API startup fails closed.
`AUTH_USER_LOGIN_ENABLED=false` by default. AUTH-06 permits opt-in User
login only with an reviewed SMS-only realm and verified, signed OTP claim. `AUTH_STAFF_LOGIN_ENABLED=false` by default. AUTH-04 adds a local-only
`staffctl` CLI for explicitly provisioning an existing Keycloak staff user;
see `docs/operations/staff-auth-local.md`. No initial staff account is seeded. Staff who authenticate at
Keycloak but have no **active pre-provisioned** `auth_identities` ->
`staff_accounts` mapping are denied.

AUTH-08 adds explicit browser SSO handoff, signed Keycloak back-channel notifications
and all-Theia-session revocation. The original `/logout` remains app-only for
compatibility. Phone recovery and remote Keycloak global logout are separate
security/release features; see the AUTH-08 section below.

## Security and operational invariants

- Separate realm/client/host/cookie for users and staff. No implicit/password
  grant and no OAuth token in browser local storage, responses or logs.
- OAuth callbacks require the same-origin short-lived login cookie, unguessable
  state, single-use transaction, PKCE, nonce and verified signed ID token.
- HTTPS cookies are `__Host-...` (`Secure`, `HttpOnly`, `SameSite=Lax`, `Path=/`,
  no `Domain`). HTTP names ending `_local` work **only** in APP_ENV=local on
  `.localhost`/loopback hosts. Do not deploy them on a public interface.
- Unsafe methods on authenticated user/staff surfaces require `X-CSRF-Token`
  from the corresponding `/auth/{kind}/session` response. Matching allowed
  `Origin` and Fetch Metadata checks also apply. No CORS token exposure.
- Application sessions are hashed in SQL, with user idle 7d/absolute 30d,
  staff idle 30m/absolute 12h, explicit revocation, and principal eligibility
  rechecked every request. DB error never becomes an authenticated request.
- Only explicitly trusted proxy CIDRs may provide `X-Forwarded-Proto`; no
  automatic trust of client-supplied forwarded hosts. The deployment proxy
  design and real HTTPS/TWA validation are AUTH-07/10 gates.
- This is **not** authorization for real mental-health users or unreviewed
  clinical material. Keep synthetic fixtures, Mock SMS and local inspector
  under their existing isolation rules.

## Required integration tests before real use

Go syntax and stdlib-only OIDC/config tests run without third-party packages;
run full Go modules and SQL integration in a workstation/CI with dependencies:

- Keycloak two-realm discovery/RS256 verification and rotated signing keys;
- SQL Server 2022 migration plus concurrent first-login and race-safe one-time
  state consumption; active/deactivated staff and hidden users;
- browser callback state/host/CSRF/expired cookie/revoked session paths;
- real HTTPS reverse-proxy and Android TWA in later release gate;
- ensure AUTH-05 SMS SPI is installed before enabling user login;
- audit correctness with forced SQL rollback/failure injection.

Full local deployment notes: `docs/operations/auth-bff-local.md`.

## AUTH-04 Staff account lifecycle

A local operator creates a named, individual staff user in the Keycloak
`theia-staff` realm, then uses `/app/staffctl -action=provision -subject=<uuid>
-confirm` to bind that issuer/subject to an active SQL `staff_accounts` record.
The mapping is idempotent and never creates Keycloak credentials. Staff `admin`
capabilities are fixed; granular roles and a role management UI are out of P0.
`-action=deactivate` revokes all Theia sessions atomically; Keycloak user
disablement is a separate operator task. SQL audit events identify the target
without pretending that the affected user performed the provisioning action.
The additional SQL migration is `0025_staff_provisioning_audit.sql`.

Staff BFF routes `/auth/staff/login|callback|session|logout` were introduced in
AUTH-03. With this ticket, the operator can enable them via
`AUTH_BFF_ENABLED=true` and `AUTH_STAFF_LOGIN_ENABLED=true` in the local env.
`/api/v1/admin/*` is **not yet implemented** (BO-01/02 owns the Admin API).
Keycloak browser/logout sync is implemented as an opt-in AUTH-08 integration.

## AUTH-06 User login

The Keycloak SMS SPI (AUTH-05) writes `theia_sms_verified=sms-otp-v1` only after successful OTP verification. AUTH-06 installs the dedicated User Session Note client mapper (via an operator-reviewed local activation script) and **requires** that proof in the verified, RSA-signed User realm ID token. Without it, login fails even if other OIDC claims are valid. `AUTH_USER_SMS_FLOW_VERIFIED` is an explicit operator attestation, not a substitute for token proof.

The existing AUTH-03 serializable identity resolver performs first-login creation and returning-user mapping keyed by `(issuer,sub)`; `is_test` is true in local and accounts are never automatically linked to synthetic Dev phone identities. `internal/platform/principal.UserID` replaces product-handler references to `devsecurity.UserID`, and AUTH-09 retired the former Dev Session test compatibility. The frontend uses `/auth/user/session` and `/api/v1/me` for login/onboarding state; no new auth bypass routes. See `docs/operations/auth-user-sms-auth06.md`.

## AUTH-07 browser/PWA/TWA handoff

User and Staff BFF login routes accept an optional URL-encoded `return_to` that
must be a *same-origin path* (not an arbitrary URL). The path is validated and
AES-GCM-sealed into the existing one-time OIDC transaction; the callback returns
`303` to that stored path, never to a callback-provided URL. Missing `return_to`
continues to default to `/`. Existing PKCE/nonce/SMS/Staff checks are unchanged.

A local-only opt-in smoke harness is available at `/__auth-smoke/user` and
`/__auth-smoke/staff` behind `AUTH_BROWSER_SMOKE_ENABLED=true`. It is not an
alternate authenticator, has no writable state except normal CSRF-protected
logout, and cannot be enabled on public origins. The **actual frontend** must
proxy `/auth/` and `/api/v1/` through the same browser origin, maintain exact
Host/`X-Forwarded-Proto`/cookie contracts, and use relative URLs with
`credentials: 'same-origin'`. Android Asset Links are served by the HTTPS
frontend/edge, not automatically by this API.

See `docs/operations/auth-browser-twa-auth07.md` for the browser flow,
reverse-proxy contract, Android signing/assetlinks validator and live smoke
checklist. Keycloak SSO logout handoff and back-channel are implemented in AUTH-08; remote global SSO logout is not.

## AUTH-08 Keycloak SSO and cross-device session lifecycle

`POST /auth/{kind}/logout` remains app-only `204`. For the frontend's normal
logout use `POST /auth/{kind}/logout/sso` (session cookie + CSRF, `200` fixed
`logoutUrl` handoff); the browser must navigate and may see Keycloak's logout
confirmation. `POST /auth/{kind}/logout-all` revokes all existing **Theia**
sessions for the verified principal, not independent Keycloak SSO sessions on
remote devices. Keycloak back-channel POSTs reach `/auth/{kind}/backchannel-logout`
and are verified by issuer-pinned RS256 and serialized replay/fence writes.
Migration `0031` is required. Existing local Keycloak PostgreSQL realms need
`scripts/auth/configure_keycloak_logout.py --apply`; imports do not overwrite
existing realms. See `docs/operations/auth-session-lifecycle-auth08.md`.
