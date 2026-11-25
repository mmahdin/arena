# Admin module — BO-01/BO-03

The Back-Office is a separate HTTP namespace within the **existing Go/Gin
modular monolith**, not a new authentication server or an AI Core service.

`GET /api/v1/admin/me` is the initial Admin endpoint. It projects only the
verified Staff UUID, fixed `admin` role, active state and absolute session
expiry. It intentionally does not expose Keycloak `sub`, email, raw sessions,
OAuth tokens, internal user accounts, or private conversation content.

## Auth ownership and route safety

- `apps/api/internal/modules/auth` exclusively owns Keycloak/OIDC login,
  identity mapping, SQL application sessions, cookies, CSRF and logout.
- All `/api/v1/admin/*` routes use the **route-group middleware**
  `authhttp.Handler.RequireStaff()` before Admin handlers.
- A staff account must be explicitly provisioned and active in SQL Server;
  deactivation immediately denies further requests even with an old cookie.
- **No BFF configured / Staff Login disabled:** the mounted Admin endpoint
  fails closed with `503 AUTH_NOT_CONFIGURED`.
- **Missing, expired, or revoked Staff session:** `401`. Wrong Host/Origin:
  `400`. Product User or Dev Session cookies never authenticate Staff.
- The single fixed `admin` role has access to all *implemented* Back-Office
  capabilities. No `admin_role_grants`, RBAC matrix or per-screen permissions
  are introduced in P0.
- Actual sensitive Admin use cases will require explicit, audited operations
  as they are implemented; an Admin session does not bypass consent/state
  invariants or enable arbitrary AI/database mutation.

See `docs/operations/backoffice-api-local.md`, `docs/operations/backoffice-audit.md`
and the approved Back-Office v3 plan. BO-04 adds durable telemetry, BO-06
adds safe turn/job inspection.

## BO-03 audit boundary

Admin reads are audited **before** returning data. `GET /admin/me` and the
new `GET /admin/audit` use a durable, fail-closed, allowlisted event recorder.
The backend obtains actor and request UUIDs from validated middleware, never
from browser headers or request bodies. The admin audit contains no raw user
content, session secrets or contact identifiers. This first phase does not
pretend to provide cryptographic immutability or persist unauthenticated denials.

## BO-06 inspector

`GET /api/v1/admin/turns`, `/turns/{turnId}`, `/jobs` and `/workers`
provide bounded, audited, metadata-only operational diagnostics. They never
read private message text, job payloads or full local Inspector traces.
See `docs/operations/backoffice-inspector-bo06.md` for exact pagination,
truncation, freshness and failure semantics.

## BO-05 overview

`GET /api/v1/admin/overview` is a separate, audited, metadata-only aggregate
read. It is backed by the BO-04 durable telemetry tables and v1 turns and
job states. It does not expose the local full-payload Inspector. Read
`docs/operations/backoffice-overview-bo05.md` for exact count, P95,
unknown-cost and heartbeat semantics; the UI must not treat missing data as a
healthy zero. The single fixed `admin` can use this capability, but Staff
session and durable read-audit requirements still apply.

## BO-08 evaluations

The guarded evaluation API imports strict, sanitized WP-15 `eval_report.v1`
files and provides audited run list/detail. A submitted digest verifies byte
integrity only; runner source, per-fixture execution and clinical approval
remain explicitly **unattested**. `blocked` never becomes `passed` by absence
of evidence. The current 28 reference fixtures are **unreviewed** and their
manifest is checked in Python tests. Import, immutable SQL storage and
read models are documented in `docs/operations/backoffice-evaluations-bo08.md`.

### BO-09 — Knowledge Bundle inventory

`GET /api/v1/admin/bundles` returns the immutable startup metadata of the
*actually pinned* AI runtime bundle and up to five deployment-configured,
read-only candidates. All requests require active Staff auth and a durable audit
write before response. No artifact bytes, clinical reviewer IDs, local filesystem
paths, signatures, prompts, or safety policy text reach this API. A matching
Ed25519 signature verifies integrity against the configured key; it does not
independently attest the reviewer or establish clinical authorization. Candidate
errors contain code-only diagnostics, and candidates are never activated.

## BO-10 backend hardening

Every implemented read route must continue requiring the shared active Staff
session and a durable append-only audit BEFORE reading privileged data. The
BO-10 security matrix exercises all Admin GET routes with real Go guards and
test stores; the actual Gin composition has a separate route-enumeration test
for the Auth-disabled and transitional Dev Session modes. The evaluation
import additionally requires a session-bound CSRF header and exact Origin.
The inspector treats unexpected historical SQL label values as `unknown` or
`unclassified`, never as arbitrary text. No migration or new admin role is
introduced. Full Keycloak/SQL/desktop E2E remains a separate deployment gate.
See `docs/operations/backoffice-hardening-bo10.md`.

### OBS-03 trace-reference metadata (not content)

`GET /api/v1/admin/turns/{turnId}/trace-references` is an authenticated,
pre-read-audited, paginated query for actual OpenTelemetry **root** IDs written
after a real Turn admission or worker claim. The action is
`admin.turn.trace_refs.read`. The endpoint returns no Phoenix URL, no prompt,
response, message, local Inspector file identifier or user ID. Missing mappings
are normal when tracing is off or persistence/export is unavailable; the
record itself does not prove Phoenix received the span. See
`docs/observability/OBS03_TRACE_REFERENCE_MAPPING.md`.
