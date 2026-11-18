# Theia AI Core — integration hardening (before WP-16)

**Scope:** focused changes on top of the user-supplied WP-01–WP-15 and integration baseline. **Status:** partial hardening, not full proactive/replay completion. No WP-16, API/HTTP, push-delivery, migration, schema, or clinical-content implementation. The baseline is the uploaded `ai therapist backend.zip`; changes are exported separately as `theia-ai-core-hardening.patch`.

Authoritative specifications reviewed: `docs/ai-core/Theia AI Core #U2014 Final Architecture Document.md` (particularly §§3.12–3.15, 4) and `docs/ai-core/Theia_AI_Core_Contract_Pack_v1.2.md` (particularly §§4.9, 7.5, 10–12). The existing `docs/quality/ai-core-integration.md` identifies the original integration gaps. The **binding 2026-11-16 soft-deletion amendment** in both architecture documents takes precedence over superseded destructive-retention language.

## 1. Root causes and disposition

| Issue | Root cause | Implemented disposition | Remaining block |
| --- | --- | --- | --- |
| Proactive wake-up execution | WP-14 persists authorized `wakeup.v1` in `dbo.ai_outbox`, but the WP-02 Gateway persists only a **real user-authored** message, the turn lifecycle requires `user_message_id`, and the Orchestrator refuses `PROACTIVE`. A synthetic `TurnRequest` would either fail admission or falsely create a user message. | Scheduler now actually dispatches due timers in its worker cycle; current authorization is rechecked transactionally, revocation and invalid references are permanently **skipped**, and the Orchestrator rejects wake-up requests **before** any admission side effects. Durable internal event dispatch is tested separately from user-facing messages. | **BLOCKED:** actual AI wake-up turn execution until versioned synthetic-turn admission, origin, state, finalization, and consent-at-execution contracts are approved. No user-facing delivery implemented. |
| Persisted runtime trace replay | WP-15 synthetic observation fixtures are complete, but normal `dbo.ai_turn_traces` stores `summary_json`, **state** `versions_json`, and no populated typed encrypted payload. That does not reconstruct the Safety/Brain/Generator/Validator inputs and outputs of a real turn. | Added a separately authorized, owner-scoped metadata-only SQL inspector plus a deterministic WP-15 replay-readiness report. It never reads prompt, message text, `payload_enc`, or unbounded trace content and always reports `replayable=false`. | **BLOCKED:** complete runtime replay requires an approved capture/artifact/access/retention and key-handling contract. Synthetic replay remains unchanged; no unrestricted logging enabled. |
| Reflection thread merge/split | `reflection.v1` `ThreadProposal` has `thread_id`, `phase`, `op`, `reason`, **no merge destination or partition boundaries**. A `ThreadOp` also lacks merge/split semantics; guessing targets could corrupt provenance and state. | Existing `RF-005` reject-and-retain path is preserved. Added a test demonstrating that neither a second candidate thread nor textual hints can cause a merge/split, while a valid `resolve` remains accepted. | **BLOCKED:** `reflection.v2` and state/SQL event contracts require approval, including target and evidence partition semantics. |
| Scheduler reliability/observability | `Scheduler.Run` previously invoked `Sweep` alone, suppressed per-cycle errors, and never called `DueTimers`. Persistent failures could go unnoticed, and `wakeup.v1` depended on an external manual `Due` call. | Added a bounded `Pulse` (Sweep → DueTimers), typed sanitized health counters and stage identifiers, configurable capped exponential retry, and a bounded persistent-failure return to `AIInternal.Run` supervision. Cancellation is not counted as a service failure. | Hosting/operations must consume `Run` errors and sample `Health()`. No notification-delivery metric is implied. |

## 2. Concrete changes

### 2.1 Wake-up dispatch and consent

- `application/scheduler/scheduler.go`: new `Pulse(ctx)` runs `Sweep(...,100)` then `DueTimers(...,100)`; the existing SQL dispatch path atomically checks user ownership, opt-in, safety, quiet hours, follow-up reference validity, and the 24-hour cap under the owner-scoped transaction. Successful dispatch means **`follow_up_due` internal outbox insert**, not user delivery. `Pulse` returns only the internal dispatch count.
- `adapters/mssql/scheduler.go`: explicit revocation after scheduling marks the timer `status='skipped', skip_reason='consent_revoked'`; no earlier timer is silently revived on later re-opt-in. An ineligible thread/protocol reference becomes `status='skipped', skip_reason='reference_ineligible'`. Both are owner-constrained `UPDATE`s within the existing transaction; **no `DELETE`** and no content erasure. Transient failures (e.g., privacy policy temporarily denies dispatch or quiet hours) remain scheduled for later reassessment.
- `application/orchestrator/orchestrator.go`: `OriginWakeUp` or any `WakeUpInfo` is rejected *before* `Gateway.Admit` runs, including if the incoming origin is disguised as a user turn. The existing route-level prohibition remains defense in depth. A new unit test verifies there is no fabricated user message or provider call.
- This does **not** turn an outbox event into a synthetic user message, fire a notification, bypass Safety/Brain/Generator/Validator, or activate the PROACTIVE route. Nothing currently executes a proactive model turn; **safe refusal is intentional** until authorization is specified.

**Exact contract decision needed to execute proactive AI turns:** approve a versioned `wakeup_turn.v1` or equivalent internal origin/event contract with (1) owner ID, follow-up ID, event ID and one idempotency key; (2) validated reason and provenance, and a non-user synthetic input; (3) consent/safety checks at *claim*, *model-inference start* and *finalization* including revocation/fencing; (4) a persisted assistant-only or explicit system-origin turn representation that does not require or invent a user-authored `dbo.messages` row; (5) transactional state/trace/outbox semantics, retry and crash recovery; (6) explicit disposition on stale/hidden conversations, quiet hours, and daily caps. This requires an approved additive lifecycle migration and port changes. Do not silently reinterpret `user_message_id` as a fake message or dispatch prevalidated text to users. Delivery transport, if later desired, is a distinct WP-16/product boundary.

### 2.2 Runtime trace replay readiness

- `eval/runtime.go`: `RuntimeTraceCoverage` reports safe, typed blocker codes for missing bundle version, missing model/prompt versions, missing or unverified encrypted payload, unverified typed stage evidence, and absent approved replay policy. The report cannot contain user identifiers, prompts, content, or model drafts.
- `adapters/mssql/trace_replay.go`: `RuntimeTraceInspector` requires a non-nil explicit `TraceInspectionPolicy`, authenticated owner and turn IDs, and an SQL join over `ai_turn_traces`, `turns`, and **visible** owner conversations. The SQL query selects only three derived presence flags from existing metadata; it does not retrieve raw JSON, ciphertext, or messages. Denied, cross-owner, hidden, and nonexistent resources expose only the same generic replay-denied error. No new external route exists.
- Tests check denial without a policy, cross-user isolation, soft-deleted conversation visibility with trace retention, and that even nonempty ciphertext is **not** proof of replay correctness.

**Exact policy decision needed:** approve a versioned, encrypted **per-stage replay evidence** format for pinned prompts/assets/models, typed Brain decisions and plans, Safety/Validator results, selected context IDs with authorized snapshots, transactional state revisions and error/repair decisions; specify authenticated capture/use consent, access audit, per-owner isolation, encryption/AAD/key rotation, storage bounds, replay-safe redaction, and retention (including backup/legal handling). **Contract Pack v1.2 §7.5 currently preserves records; trace `expires_at` is metadata, not permission to purge.** Do not invent an expiry, decrypt retained payloads in the inspector, or enable default raw prompt logging. Deterministic replay should be gated on verified typed evidence, never on a ciphertext-presence flag.

### 2.3 Reflection merge/split — proposed compatible contract

**No approved schema was modified.** A possible *future* `reflection.v2` proposal is:

```json
{
  "schema_version": "reflection.v2",
  "thread_proposals": [
    {"op": "merge", "source_thread_id": "<UUID>", "target_thread_id": "<UUID>", "evidence_ids": ["<UUID>"], "reason": "reviewed evidence reference"},
    {"op": "split", "source_thread_id": "<UUID>", "partitions": [{"partition_key": "p1", "evidence_ids": ["<UUID>"]}], "reason": "reviewed evidence reference"}
  ]
}
```

These are **proposed field names, not valid current contracts or clinical conclusions**. Before approval, define destination/new-thread creation strategy, explicit assignment of existing thread slots/messages/hypotheses to partitions, evidence provenance and owner verification, disjointness/overlap policy, ambiguity rejection, stable idempotency keys, state transition and expected-version rules, audit event(s), and reversible or superseding thread relationships. Never guess target IDs, move/delete historical evidence, or silently drop links.

**Compatibility:** keep v1 parser/schema unchanged and reject v1 merge/split with `RF-005`; register v2 as a separate negotiated bundle schema only after review; update reflection result and StateDelta/ThreadOp representations together; add forward-only SQL changes to represent merge/link and split lineage, with all original records retained. Validate legacy v1 fixtures and existing resolve/dormant paths throughout rollout.

### 2.4 Scheduler operational behavior

- `RunOptions` allows callers to configure interval, retry base, retry cap and maximum consecutive failures; `New` remains backward compatible with defaults (`1 min`, `2 s`, `1 min`, six failures). Invalid or unbounded attempt counts are rejected at construction.
- `Health()` returns a mutex-protected snapshot: total cycles, total failed cycles, internal outbox event count, consecutive failures, last stage (`sweep` or `due`), a known public error code, and timestamp markers. No raw database errors, IDs, user text, or secrets are recorded.
- After a failure, `Run` backs off exponentially until the configured cap. After the configured consecutive-failure limit it returns sanitized `ErrUnhealthy` with stage/code/count; the existing `AIInternal.Run` worker supervisor propagates this failure and cancels/join workers. Context cancellation exits cleanly without a fabricated health failure. A successful cycle resets the streak.
- **Metric limitation:** if `DueTimers` partially dispatches events and then returns an error, `EventsDispatched` does not count those partially committed events; the outbox remains the source of truth. The metric measures *successful Pulse returns only* and cannot be interpreted as exact delivered notifications or comprehensive partial-success accounting. An operational observer must also monitor the durable outbox backlog.

## 3. Tests and verification actually executed

**Environment:** this isolated session has **Go 1.23.2**, but the Go module dependency cache is incomplete, external DNS resolution fails for `proxy.golang.org`, and no `DATABASE_URL`/SQL Server 2022 instance is configured. To avoid modifying dependencies or introducing unreviewed vendors, Go invocations were attempted with `GOPROXY=off GOFLAGS=-mod=readonly`. All attempted Go commands exited `1` during **module lookup**, before compilation or test execution; this is not evidence that hardening unit tests fail or pass. The earlier integration report's Go 1.25.4 and SQL Server results were obtained in a different environment and are **not** new verification of this patch.

| Check | Result in this session | Notes |
| --- | --- | --- |
| `go test ./...` | **BLOCKED / not executed** | Missing external modules, lookup disabled due to unavailable network. Exit 1 before tests. |
| `go build ./...` | **BLOCKED / not compiled** | Same module resolution error, exit 1. |
| `go vet ./...` | **BLOCKED / not analyzed** | Same module resolution error, exit 1. |
| `go test -race ./internal/modules/ai/... ./internal/app/composition ./tests/integration` | **BLOCKED / not executed** | Same module resolution error, exit 1. |
| `go test -race ./internal/modules/ai/eval -count=1 -v` | **BLOCKED / not executed** | WP-15 package dependencies missing, exit 1. |
| `WELLBEING_INTEGRATION=true APP_ENV=local go test ./tests/integration -count=1` | **BLOCKED / not executed** | Dependencies missing and local SQL Server connection unavailable, exit 1. |
| `go run ./cmd/aicore-eval -golden ... -observations ...` | **BLOCKED / no scenarios run** | WP-15 recorded evaluation and comparison unavailable, exit 1. |
| Go parser over repository | **PASS** | 301 `.go` files parsed, zero syntax errors (standard-library `go/parser`). **Not** type-checking. |
| `python3 scripts/check_contract.py` | **PASS** | Existing OpenAPI: 30 method/route pairs match Gin composition. |
| `python3 scripts/check_structure.py` | **FAIL (baseline archive gap)** | 12 required `.agents/skills/.../SKILL.md` and `agents/openai.yaml` files are absent from uploaded baseline, unrelated to hardening. Not fabricated in patch. |
| `gofmt -l` on changed/new Go files | **PASS** | No formatting differences. |
| `git diff --check`, clean patch application | **PASS** | See patch export checks; no WP-01–WP-15 code included. |

**Tests added but not executed:** Orchestrator rejection before admission; Scheduler Pulse/Sweep/Due ordering and bounded loops; recovery, sanitized health and cancellation; runtime replay-readiness denial with/without ciphertext; SQL owner-policy and soft-delete isolation; SQL Pulse idempotency and no synthetic user turn; SQL reference invalidation; retained and skip-on-revocation SQL test; Reflection RF-005 rejection with valid resolve preserved. No assertions, SQL transactions, clinical evaluations or race tests from these new files are claimed to have run.

**Reproduction:** From `apps/api`, with project dependencies downloaded and local SQL Server 2022 and an explicitly authorized **local test** `DATABASE_URL`, run all commands above. Then run the WP-15 CLI and regression comparison as documented in `docs/quality/ai-core-integration.md`. No environment secrets, real transcripts or user-derived records were exported into this report.

## 4. Migrations, compatibility, privacy, and remaining decisions

- **Migrations:** none implemented. All changed SQL DML uses existing `dbo.ai_follow_ups.status='skipped'` allowed by the existing constraint, and the existing `skip_reason` column. No schema edits, destructive statements, changed policy defaults or off-box services.
- **Caller behavior:** `scheduler.New()` and `Due()` are retained. `Run()` now also invokes due-dispatch each cycle, may surface `ErrUnhealthy`, and is expected to be supervised/monitored. `Health()` is a new in-process inspectable counter API. Outbox consumers must **not** equate `status='sent'` to a user notification. Explicit opt-out permanently skips the old timer; re-opt-in requires a **new** follow-up ID. These are intentional behavior changes and are tested for regression.
- **Security:** owner-scoped SQL authorization, current opt-in, eligibility and transactional idempotency remain in the SQL adapter. The trace inspector's policy and SQL join both must allow metadata access; access to retained trace contents is **not** granted. Do not expose the inspector directly in an unreviewed API. No raw prompt logging, unencrypted trace export or clinical labeling was introduced.
- **Data preservation:** every user-derived message, memory, state, event, trace and follow-up remains retained; no hard deletes, no clearing content or destructive retention job. A skipped timer is an audit-visible state change, not deletion.
- **Open approvals:** (A) synthetic turn/admission/finalizer contract and consent race semantics, (B) replay capture/privacy, encryption, explicit usage/retention policy, (C) reflection.v2 thread relationship and partition contract, (D) operational health/alerting configuration for host, plus pre-existing reviewed clinical assets and policies. The missing baseline `.agents/skills` files separately block the repository structure check.

## 5. WP-16 readiness — explicit decision

**NOT READY for an unconditional WP-16 handoff declaring the internal AI Core complete.** Scheduler reliability, consent-revocation behavior, trace-inspection boundaries and reflection fail-closed invariants are improved in this patch, but the **actual proactive AI turn** and **full runtime replay** remain blocked on approved contracts/policies; reflection merge/split remains intentionally disabled. More importantly, Go compilation, tests, live SQL, races and WP-15 regression checks **could not be performed in this environment**, so this patch is **not verified for merge or release**. The approved non-proactive user-turn pipeline may be a viable future WP-16 integration target only after independent re-verification, review of the pending decisions, and explicit scoping. **WP-16 was not implemented.**
