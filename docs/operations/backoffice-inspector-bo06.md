# BO-06 — Turn Inspector, jobs and worker heartbeat APIs

## Scope

**Backend-only, metadata-only**, using the shared Keycloak Staff BFF guard and BO-03 mandatory audit. The fixed `admin` role may use every endpoint. No impersonation, user auth changes, SQL writes to AI state, job retry/override, local Inspector payloads, conversation text, raw trace, private JSON or job payload access. This is **not complete AI trace replay**. It is safe operational evidence, including partial/unknown telemetry.

## Endpoints and contracts

- `GET /api/v1/admin/turns`: v1 turn state by *createdAt* in `[from,to)`. Default 24h, max 31 days; `status` optional; `pageSize` 1–100 (default 25); opaque `cursor`. Keyset order `createdAt DESC, id DESC`. Error codes are bounded, code-shaped labels (malformed historical values become null); they are not arbitrary exception text. Legacy turn v0 rows excluded, consistent with BO-05.
- `GET /api/v1/admin/turns/{turnId}`: exact UUID lookup, `404` if missing or legacy. Returns current lifecycle state, at most 50 recent stage observations, 50 LLM provider attempts and 50 linked background jobs, with truncation flags. Timestamps are UTC. Does **not** include `ai_turn_events.delta_json`, `ai_turn_context`, `ai_turn_traces.payload_enc` or `request_meta_json`/`result_meta_json`. Warnings state that sampling can be incomplete, not authoritative full-trace reconstruction. An arbitrary request UUID is audited even when the turn does not exist.
- `GET /api/v1/admin/jobs`: metadata for jobs *created* in `[from,to)`, using same pagination; optional `status`, `kind` from a fixed list. Due times and current status may be later than the creation window. No `user_id`, `dedupe_key`, `locked_by`, lease token, encrypted result or `payload_json`.
- `GET /api/v1/admin/workers`: last seven days of liveness heartbeats from `ai_worker_heartbeats`, top 100 with truncation warning. Liveness: recent ≤45 seconds and `running` → running; old → stale. A running heartbeat does not prove successful queue processing. No hostname, worker IP or user identifiers.

All four endpoints use **verified active Staff on the Admin host** and emit a server-assigned request UUID in `admin_audit_events` **before** touching inspector data. The `admin.turns.read`, `admin.turn.detail.read`, `admin.jobs.read`, and `admin.workers.read` actions map to existing `turn`, `job`, or `worker` resource kinds (already constrained by migration `0026`). Fail closed with `503 ADMIN_AUDIT_UNAVAILABLE` when audit write fails. Unexpected SQL errors return `503 ADMIN_INSPECTOR_UNAVAILABLE` without raw exception text. Missing/expired session → `401`; invalid query or UUID → `400`. All success responses set `Cache-Control: no-store`.

`from/to` must be specified together as RFC3339 timestamps with timezone and be ≤31 days apart, upper bound not in future; repeated/unknown parameters rejected. For stable pages, reuse the *same explicit from/to and filters* with each cursor. The default moving 24-hour window is convenient for the first page but is not a permanent snapshot across repeated requests.

## Database and performance

Migration `0029_admin_inspector_jobs_index.sql` adds only a keyed `ai_jobs(created_at DESC,id DESC)` index. The list/detail queries use existing BO-05 `turns_admin_created_idx` and BO-04 stage, usage, and heartbeat indexes. Existing AI writes and worker leases are unchanged. Filtered keyset queries read at most `pageSize+1` rows; details read ≤51 per secondary table. Worker list reads ≤101 and excludes heartbeats older than 7 days. All repository queries use positional SQL parameters; no arbitrary SQL input.

## Verification

1. Apply 0029 using the existing SQL Server migration runner **before** deploying BO-06.
2. Run `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py` and `python3 -m unittest discover -s scripts/tests -v`.
3. From `apps/api`, run `go build ./...`, `go vet ./...` and `go test ./...` against the real Go dependencies.
4. On an explicitly disposable migrated SQL Server only, set `THEIA_DISPOSABLE_INSPECTOR_SQL_DSN` and `THEIA_DISPOSABLE_INSPECTOR_SQL_ACK=YES_DISPOSABLE_ONLY`, then run `go test ./internal/modules/admin/adapters/mssql -run TestBO06InspectorDisposableSQL -count=1 -v`. Never use Staging/Production for a fixture test.
5. With Keycloak/Staff BFF enabled, verify real Staff success and audited reads, stale worker warning, no leaked message/trace text, rejected User/Dev cookies, wrong-host rejection, invalid filters, expired Staff session, and audit DB outage.

No frontend work is included. A separate Admin UI ticket will consume the OpenAPI contracts. Roll back application binary/routes if required; keep the additive migration in place under forward-only migration rules.
