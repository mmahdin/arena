# BO-03: Administrative audit (local and private Back-Office)

This document is about **Admin actions**, not Keycloak login or Auth Session events.

## Storage and APIs

- Apply the forward-only SQL Server migration `0026_admin_audit_events.sql` **after** 0024 and 0025.
- Requires an active, pre-provisioned Staff session via `RequireStaff` on the configured admin origin.
- `GET /api/v1/admin/me` and `GET /api/v1/admin/audit` both record `authorized` metadata before returning private Admin data. If that record cannot be stored, they return a sanitized `503 ADMIN_AUDIT_UNAVAILABLE` rather than unlogged data.
- `GET /api/v1/admin/audit` accepts a default 24-hour interval or explicit `from` and `to` RFC3339 timestamps (maximum span 31 days), `pageSize=1..100`, optional `cursor`, `actionCode`, `actorStaffId`. Results are newest-first with SQL-backed keyset pagination; `to` is exclusive. Unknown or repeated query parameters return `400 AUDIT_FILTER_INVALID`.
- No Admin API creates, changes or deletes audit entries on behalf of the browser. The `AuditStore` port exposes only `Append` and bounded `List`.
- Every event contains only actor staff UUID, action/resource codes, optional opaque resource UUID, outcome code, a **server-generated** request UUID and a server-generated UTC timestamp. Do not store contact data, raw messages, prompts, trace bodies, IP addresses, user-agent, tokens, secrets, query strings or client-supplied reason text.
- Future Admin endpoints should call `RecordAuthorizedRead` **before** returning sensitive data (and add an explicit action to the allowlist). A distinct, authenticated, audited administrative command is required for any future write; a single `admin` role does not imply arbitrary database mutation.

## Tamper resistance and limitations

- The application repository deliberately has no update/delete methods. Migration 0026 also installs an `AFTER UPDATE,DELETE` trigger that rejects ordinary mutation attempts. This is **not a cryptographically immutable ledger**: DB owners/admins can disable triggers, backups can be overwritten, and privileged operators may access the DB directly.
- For team/production deployments, use a least-privilege SQL login with `INSERT` + bounded `SELECT` on this table, no `UPDATE`, `DELETE`, `TRUNCATE`, or schema alteration rights. Define separate retention, secured backup and restore, and optional forwarding to protected external logging before production.
- Unauthenticated attempts fail at Auth middleware before an Admin principal is known. Existing structured HTTP logs record route/status/request ID without sensitive content; these are not durable Admin actor events. A future security monitoring phase may persist anonymous denial counts separately with abuse controls.
- An `authorized` event means a verified staff request was permitted and recorded **before** the subsequent query; it is not evidence that the query or HTTP response completed successfully. Full command outcomes and denied-action records must be added when those corresponding actions are implemented.

## Validate locally

1. Apply the migration using the existing SQL Server migration runner; never run it against real-user production without backups.
2. Enable the Staff Auth/BFF settings (AUTH-04), provision an active staff account and log in on the admin origin.
3. `GET /api/v1/admin/me` returns `200` and inserts `admin.me.read`.
4. `GET /api/v1/admin/audit` records `admin.audit.read` before listing. The newly written event may appear only on the next request because the current query uses a fixed upper time bound.
5. Verify expired/revoked staff, user/Dev cookies and wrong host cannot access the API.
6. Verify `pageSize=101`, invalid cursor, invalid action code, >31-day date range, extra parameters return `400` without SQL access.
7. Verify updates/deletes to the table fail using a disposable database and restricted app credentials; inspect the event payload for sensitive values.
8. Test SQL outage: privileged reads should fail closed with `503`, not return unaudited content.

No Go dependencies are added by BO-03. The next BO-04/06 tickets should reuse the validated audit port, not recreate Auth or add generic raw-content logging.
