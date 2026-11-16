# ADR 0003 — Revocable developer sessions for internal tester switching

**Status:** Accepted for internal-only MVP (2026-11-02).

## Context

HMAC-only stateless dev cookies were valid until the 12-hour expiry even after a browser logout. Clearing the cookie did not invalidate a previously copied token. The MVP requires tester switching and clear ownership separation without implementing OIDC.

## Decision

Keep the existing local-only HMAC-signed HttpOnly cookie and stable internal user IDs. Add random session UUID (`sid`) to the signed payload, and store its association with the eligible test user and expiration in `dbo.dev_sessions` in SQL Server. On protected routes, check signature, expiry, DB session status and eligibility. Revoke the old record when switching testers and on logout. If SQL storage is unavailable, return a non-sensitive 503 instead of allowing access.

## Consequences

- Requires migration `0005_dev_sessions.sql` applied before the updated API starts.
- Pre-change cookies will need a fresh Dev Login. This is deliberate, no impact to seeded profiles or conversations.
- One SQL read per protected request is acceptable for the internal MVP. Production auth and scalable session management are out of scope.
- The user selector remains **not real authentication**: anyone who can reach it could select any test identity. Keep synthetic-only traffic on private loopback.
- Expired/revoked records may later need a retention cleanup task; not required for the short-lived internal prototype.
