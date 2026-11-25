# BO-05 — Admin Overview (read-only, metadata-only)

**Dependency:** BO-04 migration `0027` and BO-03 audit migration `0026` must be applied. BO-05 adds the additive SQL Server index migration `0028_admin_overview_turns_index.sql` **before** enabling the endpoint on a large dataset. This module does not change the AI Core turn lifecycle or write any model data.

## API and security

`GET /api/v1/admin/overview` requires the existing active Keycloak Staff BFF session (fixed `admin`) and valid Admin origin. Auth is NOT reimplemented. An authorized read is recorded in `admin_audit_events` using the allowlisted `admin.overview.read` action **before** any aggregate query executes. Invalid filters fail with `400 OVERVIEW_FILTER_INVALID` and do not query the DB. Missing/revoked Staff sessions get `401`; missing Auth returns `503`. Audit/storage failures fail closed (`503`) with no SQL exception details in the HTTP response. `Cache-Control: no-store` is set on success.

Optionally pass `from=<RFC3339>&to=<RFC3339>` (URL-encoded); both are required together, with inclusive lower/exclusive upper, max 31 days, no future upper bound. Default window is the past 24 hours. Duplicate/unknown query keys are rejected. `asOf` is the server-side collection time. Queries are bounded by a five-second request timeout.

## Exact semantics (do not relabel in the UI)

| Field | Interpretation |
| --- | --- |
| `turns.*` | **Current** statuses of `lifecycle_version=1` turns **created** within `[from,to)`. `excludedLegacy` counts v0 turns omitted from rates. A turn admitted yesterday and failed today is counted in yesterday's cohort if it is currently failed. These are NOT transitions per day. |
| `turns.failureRate` | Current failed / (current failed + completed) among admitted v1 turns. Cancelled/running/queued not in denominator; `null` if no terminal sample. |
| `turns.p95ElapsedMs` | SQL Server `PERCENTILE_CONT(0.95)` over valid **completed** v1 turns, from `created_at` to `completed_at` (includes queue time). `null` when no samples. NOT stage latency or model latency. `latencySamples` is reported. |
| `llm.attempts` | All **physical** provider attempts beginning in `[from,to)`, including retries, embedding, repairs and background/unattributed calls. Status is current as of query. |
| `llm.estimatedMicroUsd` | Sum of non-null configured-price estimates, in micro-USD; **not** an invoice. `null` if no priced attempts. Unpriced/unknown/start attempts are explicit; `costPartial` true when coverage incomplete or no attempts exist. |
| `llm.*Tokens` | Sum of provider-reported counters, `null` if none. These are partial when `unknownUsage` > 0. |
| `stages` | Best-effort sanitized stage observations recorded within `[from,to)`. `durationSamples` counts observations with a measured non-null duration; **no stage coverage percentage is asserted**, because trace writes may fail. |
| `jobs` | Current global queue state **at collection time**, not a historical time-series bounded by `[from,to)`. `dueNow` means queued with `run_after <= asOf`. |
| `workers` | Current worker heartbeat *liveness* (memory, reflection, scheduler); last heartbeat younger than 45 seconds with status running → `running`. Old heartbeat → `stale`, no heartbeat → `unknown`. Does not prove the worker processes jobs. Latest observed heartbeat is reported even when stale. |
| `warnings` | Stable quality/availability codes, never raw exception or user data. Any absence of telemetry is reported, not guessed as healthy. |

> The aggregates exclude user identifiers, phones, messages, prompts, raw trace payloads and local Inspector artifacts. The current first release remains synthetic-only; exposing this API to real-user workloads requires the separate operational/privacy release gates.

## Local validation

1. Apply migration 0028 using the normal SQL Server migration runner. Do not edit 0001–0027.
2. Run `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py`, `python3 -m unittest discover -s scripts/tests -v`.
3. From `apps/api`, run `go build ./...`, `go vet ./...`, `go test ./...` with your normal local module cache/network access.
4. For real SQL aggregation tests, **only** against an isolated disposable SQL Server with all migrations through 0028 applied, set `THEIA_DISPOSABLE_OVERVIEW_SQL_DSN` and `THEIA_DISPOSABLE_OVERVIEW_SQL_ACK=YES_DISPOSABLE_ONLY`. Run `go test ./internal/modules/admin/adapters/mssql -run TestBO05OverviewDisposableSQL -count=1 -v`. Never point this test at Staging or Production.
5. Sign in as active Staff and request `/api/v1/admin/overview`; verify `admin.overview.read` in audit. Repeat after staff deactivation, with a product User/Dev cookie, bad timestamps and DB/audit outage. Confirm no sensitive content is returned.

## Deploy and rollback

The query module only reads from `turns`, `ai_llm_attempts`, `ai_stage_events`, `ai_jobs` and `ai_worker_heartbeats`; it is not a telemetry writer or queue controller. Deploy after BO-04 telemetry writes have been validated. For rollback, restore the prior binary/route; leave migration 0028 intact (forward-only migration policy). Empty/partial metrics must remain `null`/warned in UI, not overwritten with fake zeros.
