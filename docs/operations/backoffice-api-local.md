# Local Back-Office API — BO-01/BO-02

This PR introduces **only** a guarded self-identification endpoint; the
Back-Office UI, administrative audit log, diagnostics and other read models
are future tickets. No additional database migration is needed.

## Prerequisites

1. Apply and validate AUTH-03 SQL migration `0024_auth_identity_sessions.sql`
   and AUTH-04 migration `0025_staff_provisioning_audit.sql`.
2. Start the opt-in Keycloak/PostgreSQL infrastructure and create a named
   account in the `theia-staff` realm (not the Keycloak `master` realm).
3. Provision its UUID with `staffctl` as documented in
   `docs/operations/staff-auth-local.md`.
4. Enable `AUTH_BFF_ENABLED=true`, `AUTH_STAFF_LOGIN_ENABLED=true`, and leave
   `AUTH_USER_LOGIN_ENABLED=false` until SMS authentication is complete.
5. The local Staff browser origin must be exactly the configured
   `AUTH_STAFF_ORIGIN` (typically `http://admin.localhost:8080`); browser
   cookies are host-only, not shared with `app.localhost`.

## Verify

- Visit `/auth/staff/login` on the Staff origin and sign in with Keycloak.
- `/auth/staff/session` returns an active Staff session and a CSRF token.
- `GET /api/v1/admin/me` on the **same origin** returns:

```json
{
  "staffId": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  "role": "admin",
  "active": true,
  "expiresAt": "2030-01-01T00:00:00Z"
}
```

These are example values only. The endpoint contains no passwords, OTPs or
Keycloak tokens. Response headers include `Cache-Control: no-store`.

### Negative cases

- No BFF configuration or Staff Login disabled: `503 AUTH_NOT_CONFIGURED`.
- No Staff cookie, revoked or inactive account: `401`.
- Wrong host (for example `app.localhost`): `400 AUTH_ORIGIN_MISMATCH`.
- User-realm or Dev Session cookie: cannot authenticate an Admin API request.
- Supplying `X-Staff-ID`, `userId`, `role` or other caller-controlled identity
  data never creates an authenticated Staff principal.
- Any new Admin write method must inherit the existing BFF CSRF guard and
  requires its own purpose-specific application authorization and audit.

`/api/v1/admin/me` must remain independent of Dev Session settings; do not
restore temporary Dev Login-based Staff access. Before shared-network use,
require the planned private VPN and TLS setup even though local loopback does
not require VPN.
