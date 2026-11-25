# WP-08 — Context & State (integration and verification)

Scope: WP-08 only; reference is `docs/ai-core/Theia_AI_Core_Contract_Pack_v1.2.md`, especially §§4.3, 6.1–6.2, 8.6, 10.2, and 13.1. WP-01 contracts, WP-02 SQL Unit of Work, WP-03 versioned artifacts and WP-05 orchestrator remain unchanged. This patch does **not** introduce another message store, clinical logic or model calls.

## Owned packages and authority

- `modules/ai/application/contextstate`: immutable-per-operation WP-03 policy handle, strict typed configuration decoding, `Snapshot`/`Reader` interfaces for owner-scoped AI state and optimistic versions.
- `modules/ai/adapters/mssql/context_reader.go`: read-only projection of WP-02 `ai_sessions`, `ai_user_state`, `ai_threads`, `ai_protocol_instances`, `ai_open_loops`, `ai_follow_ups` and active `ai_formulations`. `UserEvidence` checks user-owned, **not-soft-deleted** user messages. `ExistingHypotheses` checks active user-owned hypothesis IDs without reading text. It never runs `DELETE`, updates rows, or builds derived state from experimental AI code.
- `modules/ai/application/assembler`: implements the existing `ports.ContextAssembler`; owns the deterministic `wctx.v1` assembly, recent history cap, current message separation, risk max, session/resumption and protocol views, focus-thread slots, confirmed retrieval projection, bounded tokens, eligibility and Information Needs. Unknown problem slots remain explicitly `unknown`; synthetic WP-03 data cannot enable protocol eligibility. Context does not retrieve memory itself.
- `modules/ai/application/state`: implements the existing `ports.StateCommitter`; validates/repairs proposed deltas *before* persistence. The `ValidatedFinalizer` wraps WP-02's existing `ports.TurnFinalizer` and delegates the **single** assistant/state/jobs/trace transaction to WP-02. It bypasses repeat validation on completed-turn replay and validates owner/turn admission before first completion.

### Invariants

- Only an owner-scoped, admitted running turn is assembled. The trigger is excluded from `recent`; queued later turns must not leak into earlier context. All memory views arrive from the caller-supplied `RetrievalResult` (WP-09); degraded/paused memory is not exposed.
- A `StateDelta` is deep-copied, checked against live user/session and the exact thread/protocol versions before forwarding. Invalid optional changes are dropped/clamped with stable rule codes; stale session/user/thread/protocol versions are conflicts, not discarded.
- Thread transitions obey the phase FSM; `understanding_sufficient` requires the WP-03 readiness rule and verifiable thread-level user-turn evidence. A speculative thread-turn count is *never* invented. At most three projected active threads are allowed; automatic pausing will not orphan a focused thread.
- Slot updates require existing user-owned evidence; hypothesis signals additionally require active owner-scoped hypothesis rows and unpaused memory. Existing related-thread IDs are preserved on an unrelated update, because WP-02's current SQL writes `related_thread_ids_json` unconditionally.
- Risk cannot be reduced by the AI-only delta; a valid rise is preserved. Elevated or higher risk pauses an active protocol within the same state delta. A protocol cannot activate without a user-owned consent message and timestamp, and permitted step/status transitions are validated against its pinned WP-03 graph.
- Open loops deduplicate by `(kind, ref_id)`; follow-up scheduling needs explicit opt-in, a valid due time, no quiet-hour conflict, and a 24-hour gap. Cancellation is owner-checked. No WP-08 code physically deletes anything.
- On replay, the wrapped finalizer delegates to WP-02's idempotent completed result. `Reconcile` only audits finalizations and reports inconsistencies; it never reconstructs missing user data by guesswork.

## Wiring requested for a separate integration PR (not included)

Inside the existing composition root, construct `contextReader, err := mssql.NewContextReader(db)` (AI MSSQL adapter), then `assembler.New(contextReader, conversationStore, wp03Registry)` and `state.New(contextReader, wp03Registry)`. Construct `state.NewValidatedFinalizer(existingWP02Finalizer, stateCommitter)` and inject **only** these new implementations through WP-05's existing `ports.ContextAssembler` and `ports.TurnFinalizer` dependencies. Keep the same WP-02 Conversation transaction participant, user lease, authentication and `dbo.messages`. Do not replace the transaction or insert a second state writer. Runtime activation of WP-03 synthetic bundles is development-only.

**WP-05 integration constraints:** Its current `prepare` intentionally ignores the `ports.StateSnapshot` returned by `Build`, overwrites `wc.Turn` with the minimal identifiers, builds a minimal `StateDelta` from admission versions and emits no pinned bundle version in `PreparedTurn.Trace`. A separate, coordinated WP-05 change is needed for a rich context turn view, actual per-thread/protocol patch version propagation, trace of validation rejections, and a stable *turn-wide* knowledge-bundle version. A finalizer wrapper cannot solve these without changing those contracts or internals. Do not silently route this path to the experimental pipeline.

## Explicit gaps / safely blocked work

1. **Persisted comfort budget:** WP-01 `UserState.ComfortBudget` has no corresponding WP-02 `ai_user_state` column. The SQL read projection deliberately leaves it at zero; therefore `InfoNeeds` is conservatively suppressed with the current database, though deterministic tests using a known positive budget pass. A reviewed forward-only schema/owner write contract is needed in a later integration change; no migration is bundled here.
2. **Per-thread turn counts:** WP-02 persists `ai_sessions.turn_count`, but has no authenticated per-thread message association. The SQL reader returns unknown counts as zero; transitions requiring `min_user_turns_in_thread` remain blocked until an explicit thread-turn association exists. Never infer thread turns from all session turns.
3. **Focus clearing:** WP-02's `applySession` uses `COALESCE(@focus,focus_thread_id)` and lacks a clear-focus patch. WP-08 rejects transitions that would orphan a focus; explicit clear-focus semantics require a separate WP-02 contract/SQL change.
4. **Clinical and future WPs:** Lexical/semantic verification of slot support, clinical risk de-escalation, memory content/corrections, protocol decision-making, protocol timeout scheduler (ST-023), proactive delivery and trace/event review are deliberately not implemented. Some rules are conservatively rejected or recorded as deviations until their owning work packages exist.
5. **Production readiness:** No SQL Server instance or clinic-approved Knowledge Bundle was provided. All SQL and clinical integration claims remain unverified. The WP-03 bundle in tests is synthetic-only and not a release candidate.

## Verification

Run from `apps/api` with the repository's **real** pinned Go dependencies in CI:

```sh
go test ./internal/modules/ai/application/contextstate ./internal/modules/ai/application/assembler ./internal/modules/ai/application/state ./internal/modules/ai/adapters/mssql
go test -race ./internal/modules/ai/application/contextstate ./internal/modules/ai/application/assembler ./internal/modules/ai/application/state ./internal/modules/ai/adapters/mssql
go vet ./internal/modules/ai/application/contextstate ./internal/modules/ai/application/assembler ./internal/modules/ai/application/state ./internal/modules/ai/adapters/mssql
```

Sandbox checks use a temporary **external** offline UUID/PyYAML module bridge because pinned dependencies cannot be fetched here. That bridge is not included in the deliverable and is not a substitute for real dependency verification. Check SQL Server 2022 execution, consistency under concurrent turns, WP-05 integration and the SQL transaction in normal CI.
