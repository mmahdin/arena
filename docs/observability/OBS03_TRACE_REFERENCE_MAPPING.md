# OBS-03 — Durable OTel root references and audited Admin lookup

**Status:** Go build/tests and the disposable SQL Server upgrade, rerun and
pagination fixture passed. Live reconciliation of a product Turn or job with
Phoenix is pending. The Admin endpoint exposes only trace reference metadata.

## Why a separate SQL mapping exists

The local Inspector `trace_id` is a private file-based UUID, **not** a W3C
OpenTelemetry TraceID. Existing BO-04 SQL attempt/stage evidence is durable but
cannot join a specific root exported to Phoenix. OBS-03 adds a minimal
`dbo.ai_trace_refs` table: `turn_id`, root trace/span ID, kind, optional job ID
and claimed attempt, SQL insertion timestamp, and an opaque row ID. Nothing
from conversation content, user identity, OAuth tokens, prompts, responses,
model parameters, encrypted traces, Inspector paths, or Phoenix URL is stored.

Migration **0034** is forward-only SQL Server 2022 and adds foreign keys to
`turns`/`ai_jobs`, root-uniqueness and bounded Turn-ordered query indexes.
Only a real v1 admitted lease-owning Turn is recorded; for background work,
only an actually claimed memory/reflection job attempt is recorded. A job
reference must match its Turn in the INSERT predicate. Trace ID and span ID
are obtained from the **original code-owned OTel root**, even when the local
Inspector has created a nested child span.

### Write semantics

The writer uses a maximum 150 ms independent SQL timeout. Errors are swallowed
and never influence AI admission, LLM calls, finalization, or background job
outcomes. This provides **durable successful inserts**, not guaranteed delivery
under SQL outage, overload, restart or exporter failure. An absent mapping is
not evidence that a Turn did not execute; a persisted mapping does not prove
Phoenix received the span. No retry/backfill is invented, and there is no
automatic deletion or retention job. Retention and backup policy require a
separate approved decision. Existing soft-delete behavior is unchanged.

### Staff API

`GET /api/v1/admin/turns/{turnId}/trace-references`

- Requires the existing verified, active Staff BFF session on the Admin host.
- Entirely inside `RequireStaff`; no User cookie, synthetic Dev Session or
  spoofable header can grant access.
- Exact Turn UUID; optional `pageSize` (1–50, default 25) and opaque `cursor`.
  Unknown, empty or duplicate query parameters and invalid IDs are rejected.
- Appends `admin.turn.trace_refs.read` to immutable SQL audit **before** any
  Turn-existence or trace-reference SELECT. Audit write failure is 503 with no
  data read. Invalid/expired Staff yields 401, bad filters 400, absent v1 Turn
  404, internal reader/SQL failures 503. All errors are safe codes.
- Returns `{turnId, items, nextCursor}` with each item limited to
  `{id,kind,traceId,rootSpanId,jobId,jobAttempt,recordedAt}`. No raw messages,
  Inspector URLs or Phoenix credentials are returned. A v1 Turn without
  references returns an empty array; `nextCursor` is null.
- `Cache-Control: no-store`, `Referrer-Policy: no-referrer`; page size and SQL
  projection are fixed. It never queries Phoenix during an Admin read.

**Phoenix link rule:** the frontend may, after separate approved access,
construct a link to a fixed, trusted Phoenix UI origin from the validated
`traceId`. Do not form a URL from a user-controlled host/query or treat the
existence of a reference as proof that Phoenix stores it. Verify the exact
Phoenix UI route in a real OBS-02 deployment before making it clickable.
Phoenix authentication and content-view audit/authorization are separate from
this metadata-only Admin API. The API does not grant Phoenix login.

## Files touched

- `contracts/openapi/openapi.yaml` — contract-first path and strict schemas.
- `apps/api/db/migrations/0034_ai_trace_refs.sql` — metadata-only table/indexes.
- `apps/api/internal/modules/ai/trace/references.go` — ephemeral writer port,
  root-specific W3C join and no-failure propagation.
- `apps/api/internal/platform/telemetry/telemetry.go` — code-created OTel root
  context identifier independent of current child span or browser headers.
- `apps/api/internal/modules/ai/adapters/mssql/trace_refs.go` — fixed INSERT.
- `orchestrator.go` and the memory/reflection workers — real admission/claim
  recording points, leaving the existing Inspector and LLM code unchanged.
- `app/composition` — inject writer only with configured OTel/SQL; Staff guard
  for new Admin route.
- `admin/domain`, `admin/application`, `admin/adapters/{http,mssql}` —
  validated, audited, bounded read, safe status mapping.
- Static checks, Go unit/security tests, and opt-in disposable SQL test.

## Review and acceptance

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -q
(cd apps/api && go mod tidy) # on a connected machine, update go.sum from OBS-01
(cd apps/api && go build ./...)
(cd apps/api && go test ./...)
(cd apps/api && go test -race ./internal/modules/admin/... ./internal/modules/ai/trace/... ./internal/platform/telemetry/...)
```

For a fresh **disposable** SQL Server with all migrations through 0034:

```sh
THEIA_DISPOSABLE_TRACE_SQL_DSN='<disposable-db-dsn>' \
THEIA_DISPOSABLE_TRACE_SQL_ACK=YES_DISPOSABLE_ONLY \
  go test ./internal/modules/admin/adapters/mssql -run TestOBS03TraceReferenceDisposableSQL -v
```

Do not run this destructive fixture test against local shared, staging or prod.

The complete live acceptance gate still requires OBS-02 to have passed:
run one successful and one failed synthetic Turn, record each `turnId`, inspect
its OTel root trace in Phoenix, call the new Admin endpoint as Staff, and
reconcile the W3C IDs. Repeat for a claimed background job attempt. Then
verify anonymous/User/Dev cookies, wrong Admin host, inactive Staff and audit
outage all reject reads without leaking content. Never use real user text.
