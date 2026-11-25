# ADR 0012 — Shared identity and browser sessions for Theia

- **Status:** accepted for implementation; no runtime Auth implementation is implied
- **Date:** 2026-11-19
- **Owners:** Authentication/platform (not Back-Office)
- **Authoritative implementation plan:** `docs/plans/THEIA_AUTH_IMPLEMENTATION_PLAN_V3.md`

## Context

The current signed Dev Session is a **synthetic identity selector**, not verified authentication. Existing Go/Gin routes and parts of AI startup are still coupled to `ALLOW_DEV_LOGIN`. The approved product direction requires real login for the web/PWA/Android TWA app and separately authenticated internal staff, without duplicating authentication inside the Back-Office module. Existing user UUIDs, SQL Server application data and AI Core owner-scoping must remain stable.

## Decision

1. Run a **separately deployed Keycloak service** with its **own PostgreSQL database**. Keep the application/AI Core in the existing Go modular monolith and SQL Server 2022; do not create a custom Auth microservice.
2. Use one Keycloak installation and separate realms: `theia-users` (passwordless phone + SMS OTP) and `theia-staff` (provisioned staff, email + password). There is no staff self-registration. MFA is not required for the approved local/private prototype; reassess before any public exposure.
3. Implement SMS authentication using a reviewed **Keycloak Authentication SPI**. Keycloak owns OTP generation, verification, expiry, single use and abuse controls; Kavenegar only delivers SMS through a replaceable `SmsSender`. Use `MockSmsSender` **only** in isolated local/CI tests, never as a public authentication bypass.
4. Implement one shared Go **Auth/BFF module**, responsible for OIDC Authorization Code + PKCE, state/nonce/issuer/audience/redirect verification, identity mapping, user/staff guards, CSRF, logout/revocation and sanitized Auth audit. The Back-Office consumes `RequireStaff`; it must not implement its own OIDC client or session store.
5. Use **opaque host-only HttpOnly Secure cookies** and revocable **SQL Server application sessions**; only hashed bearer verifiers are persisted. Keycloak retains its own IdP sessions in PostgreSQL. Staff and user session cookies and callback origins must be distinct, with fail-closed issuer/client/host verification and trusted reverse-proxy boundaries.
6. Map immutable `(issuer, sub)` to an existing stable `users.id` or provisioned `staff_accounts.id`. A phone number is not an application primary key. Staff login never auto-provisions staff. User identity creation happens only after proven OTP verification.
7. The web/PWA and Android **TWA share the browser-based flow**. Do not put OAuth tokens into frontend localStorage, JavaScript state or Android-native storage. Validate host-only cookies, OIDC redirects and Digital Asset Links on real devices before release.
8. **Transition instead of abrupt deletion:** decouple API routing, AI Core bootstrap, local synthetic Knowledge Bundle and Local Inspector from `ALLOW_DEV_LOGIN` before replacing the existing dev guard. Remove `/api/v1/dev/*`, the signer/config, and dependent frontend/tests at the **cross-repository cutover**, not in the infrastructure PR.

## Environment and deployment boundaries

| Environment | Identity and test policy | Exposure |
| --- | --- | --- |
| Local development | Keycloak + PostgreSQL + SQL Server; synthetic users; isolated Mock SMS; existing Dev Session temporarily during migration | Loopback only; no real mental-health content |
| CI / disposable integration | Disposable Keycloak/PostgreSQL + SQL Server and deterministic synthetic fixtures; mock SMS transport; negative auth tests | Isolated jobs, no public test auth endpoints |
| Team internal shared deployment (future) | Staff Keycloak login, fixed `admin`; verified user Auth; synthetic-only unless additional release gates pass | Private VPN/ingress; TLS, secure cookies, audited staff activity |
| Real-user/public (not authorized by this ADR) | Requires independent clinical, privacy, legal, retention/erasure, threat-model and security approval | No automatic promotion from local/internal |

## Security and implementation requirements

- Host-bound, exact callback redirect allowlists, no wildcard URLs. Bind OIDC transactions to the initiating browser/session; reject replay, wrong realm, wrong host, wrong audience, and tampered state.
- OTP: six digits and initially two-minute expiry, no more than five attempts per challenge, at least 60-second resend delay, replay prevention, multidimensional anti-abuse and cost controls; validate these defaults with implementation tests.
- Cookie sessions: independent user/staff names; short staff idle lifetime; SQL-backed revocation; fail closed on storage outage; protect unsafe methods against CSRF. Do not log SMS codes, phones, tokens, cookies, passwords or private prompts.
- Use additive SQL Server migrations only, assigned **in actual merge order** after current head; Keycloak schema is managed by Keycloak in PostgreSQL, not the Go SQL migration runner.
- Auth is **not** approval to process real mental-health data. The restrictions in ADR 0010/0011 and AI Core privacy/safety release gates still apply.

## Alternatives considered

- **Custom Auth inside Go:** rejected because password/IdP/session and OTP protocol maintenance would expand security ownership.
- **A second custom Go Auth service:** rejected as unnecessary operational complexity.
- **Keycloak holding only the application session:** rejected for this browser BFF; Theia requires immediate application-level session revocation, owner-scoping and staff state enforcement.
- **Reuse Dev Session for staff:** rejected; synthetic selection does not establish identity.

## Consequences and validation

Keycloak and PostgreSQL add deployment, backup and availability responsibilities; custom SMS SPI is a security-critical Java component requiring integration tests and version pinning. Verify wrong-realm/cross-user access, failed/expired OTP, concurrent first login, callback replay, logout, revocation, browser/TWA cookie handling, and AI functionality with `ALLOW_DEV_LOGIN=false` before cutover. Follow the ticket order and rollback gates in `docs/plans/THEIA_BACKEND_FIRST_IMPLEMENTATION_RUNBOOK.md`.
