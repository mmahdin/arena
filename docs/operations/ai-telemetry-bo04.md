# BO-04 — Durable AI operational telemetry (synthetic-only)

Implementation baseline: updated backend uploaded on 2026-11-19; migration `0027_ai_operational_telemetry.sql`.

## Data ownership and privacy

The AI Core retains its original conversation, turn and job tables. BO-04 adds **three separate, sanitized SQL Server tables**:

| Table | What is stored | Explicitly excluded |
| --- | --- | --- |
| `dbo.ai_llm_attempts` | One UUID per **physical** Structured/Stream/Embed provider attempt; optional turn UUID, configured provider/model labels and price rate snapshot, timestamps, terminal status, nullable token counts and nullable estimated micro-USD cost | User ID, phone, prompts, completions, response JSON, provider error body, API keys |
| `dbo.ai_stage_events` | Allowlisted AI Core stage/status, observed duration, bounded count and attempt, approved rule code and synthetic turn UUID | Arbitrary `Record` values, local Inspector JSON, model output, internal exception text |
| `dbo.ai_worker_heartbeats` | Per-run random UUID, worker kind, last-seen timestamp and running/stopped/failed state | Worker job payload, private state, process secrets |

`ai_turn_traces` remains separate; the **local full-payload Inspector must not be exposed** by BO-04. Admin HTTP APIs are **not** added here; BO-05/06 own bounded reads, protected by Staff Auth and Admin Audit.

## Behavioral guarantees and limitations

- The optional `llm.AttemptJournal` is wired into **real deployment-runtime clients** in `loadProduct`, not fake client tests by default. A begin row is inserted *before* the provider request, then a terminal UPDATE with `WHERE status='started'` occurs after a physical provider call. Completion is idempotent; a crash or finalization failure leaves the attempt **started/unknown**, not free. A provider request is not sent when a required Begin write fails. A failed Finish returns a sanitized error, never a successful provider result.
- Retries, JSON schema repair, fallback calls, embeddings and streaming provider attempts are each individually metered. `succeeded` means the **provider** request completed, not that the overall AI turn passed schema/safety validation.
- Only explicit nonzero token usage is treated as known. `unknown` cost is NULL (including interrupted/failed attempts without counters). `unpriced` means usage was received but no complete configured price was available. `estimated` is calculated from the **configured, snapshotted** input/output/cached micro-USD-per-million rates. It is not an invoice, a hard quota, or verified provider billing. Do not sum NULL costs as zero or claim completeness for missing usage.
- Existing `llm.MemoryLedger` remains the process-local **soft quota** mechanism. BO-04 is an observability sink, **not** a distributed reservation/limit. Hard quota enforcement and provider invoice reconciliation are separate tasks.
- Stage writes reuse `loggingTracer` approved stage vocabulary and `trace.ApprovedRule`. Write failure is logged with a fixed safe warning and may create incomplete event coverage; use reported freshness and coverage instead of inventing a stage P95. A zero/unavailable stage duration stays NULL.
- Heartbeats reflect **process liveness**, not job progress, queue speed or worker correctness. Consider them stale after 45 seconds without an update (15-second cadence). A crash may leave the last state as `running`; freshness is required. Workers update their terminal state on graceful exit.
- No full-payload trace, arbitrary message inspection, backfill, cleanup, physical deletion, or sensitive export is introduced. The existing soft-delete retention policy remains unchanged.

## Deploy and test

1. Apply **migration 0027 before enabling the new backend**. Do not edit applied migrations 0001–0026. SQL Server 2022 only; Keycloak PostgreSQL remains unaffected.
2. Confirm `APP_ENV=local`, synthetic runtime assets, and local/private listener. Configure AI Core as before. No new credentials or services are needed.
3. Run `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py`, `python3 -m unittest discover -s scripts/tests -v`, then from `apps/api` run `go build ./...`, `go vet ./...`, and `go test ./...`.
4. For the SQL test, use a **local disposable SQL Server** and opt in with `WELLBEING_INTEGRATION=true APP_ENV=local`, setting `DATABASE_URL` to a local SQL Server URL, then `go test ./tests/integration -run TestBO04DurableTelemetrySQL -count=1 -v`. The test creates/drops a random isolated database; it must **never** target staging or production.
5. Run a synthetic AI turn including an LLM retry and verify: one attempt row per provider call; `started` rows only for interrupted writes; telemetry values contain no message bodies; stage/worker timestamps refresh; a worker crash yields stale/failed status, not permanent green.
6. Failure drills: disable SQL briefly and confirm no new provider call happens on Begin failure; leave a begun row after an interrupted Finish; check no duplicate billed row on completion replay; verify no raw exception content in SQL or logs.

## Upgrade and rollback

This is additive schema plus opt-in integration at the real AI runtime. Rolling back the binary stops new writes; the tables remain for diagnostics. **Do not DROP / TRUNCATE data as a rollback**. Deploying a binary with the BO-04 writer against a database without migration 0027 can cause provider calls to fail closed; use the repo's migration job before deploy.

**Known gap:** physical provider usage could remain unknown after an upstream timeout, partial stream, process termination or missing provider counters. These records must not be reported as zero cost. Stage write failures can omit events; exact stage-latency and spending statements need coverage flags in BO-05. Worker heartbeat reflects only the in-process loops; BO-06 will join with existing `ai_jobs` statuses and retry metadata.
