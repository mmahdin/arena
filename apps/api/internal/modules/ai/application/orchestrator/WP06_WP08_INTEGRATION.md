# WP-06 + WP-08 integration into WP-05 (integration-only)

`NewIntegrated(dependencies, wp05Options, IntegratedOptions)` opts into the
actual WP-06 Safety Pre-Gate, FSM and Crisis Planner and the WP-08 Context
Assembler and StateCommitter. The original `New` remains available for the
isolated WP-05 walking skeleton. Production composition is intentionally not
modified here: a separate integration owner must wire the *existing* WP-02
ConversationStore, Gateway, TurnRecovery and TurnFinalizer, a WP-08 SQL
`contextstate.Reader`, a release-verified WP-03 BundleRegistry and loaded WP-06
Assets + L1 Classifier. The later-phase Brain, Generator and Validator are still
required as injected ports; this integration does not implement any of them.

## Execution and persistence

1. **Admit durably and fence** the worker with the WP-02 user-scoped lease.
   Completed/in-flight duplicates never invoke safety or any model.
2. **Pin the WP-03 bundle** after admission, and verify its immutable version
   and prompt hashes equal the installed Safety assets. Unreviewed/synthetic
   bundles fail closed in the production setting. Propagate the authenticated
   user/turn/bundle as `safety.WithCallMeta` for the WP-04 L1 classifier.
3. **Read owned persisted state** and check session/user versions, ownership,
   session identity and closure. Slice recent messages at the admitted user
   message, preventing later queued content from entering L0/L1 history.
4. **Assess Safety (L0/L1)** alongside optional retrieval; assemble the real
   WP-08 WorkingContext and re-check its snapshot versions. Apply the pure
   WP-06 FSM using the persisted risk; default-clearance evidence is empty,
   so no speculative de-escalation occurs. Route conservatively. High/imminent
   risk uses the bundled CrisisPlanner and skips the normal Brain.
5. **Run the restricted two-stage pipeline** for normal turns. A valid Brain
   raise can only increase risk; re-run the FSM and re-route to buffered CRISIS
   before generation. This patch does *not* authorize interventions, protocols,
   memory, clinical skills, new threads, or psychotherapy decisions.
6. **Validate the response or its pinned fallback**, then build a StateDelta
   from admission versions. Allow only existing-focus-thread evidence-based
   inferred slots and limited session fields; the WP-08 committer validates
   every optional proposal. A risk transition must survive validation and
   carry exactly one matching metadata-only `risk_level_changed` event.
7. **Complete once** via the existing WP-02 finalizer, wrapped by WP-08's
   `ValidatedFinalizer`. Assistant, session/thread/protocol/user state, trace,
   and outbox event remain in *one* fenced SQL transaction. The small
   `applyUser` integration updates `risk_follow_up_until` in that same unit of
   work from the matched event, never in a second writer. Low-risk transitions
   are not required to invent a follow-up deadline.
8. Failed work uses WP-05's existing `Fail`, idempotent replay, explicit
   `RetryFailed` and `RecoverNext` on the **persisted** request. No state or
   assistant message is independently committed, and no data is hard-deleted.

## Known limitations / clinical and infrastructure gates

- **Do not deploy this patch alone.** WP-03's bundled clinical assets are
  synthetic. Release verification, clinician-approved per-locale lexicons,
  L1 prompts, crisis scripts, vetted contacts and fallback copy are required.
  Synthetic loading is allowed solely for test fixtures. When the active bundle
  rotates, Safety assets/classifier must be re-instantiated for that version;
  mismatch fails closed instead of mixing policies mid-turn.
- The real WP-07 output Validator is **not** implemented here. WP-05's injected
  validator must enforce crisis content, required elements, and allowed
  resources; the test validator intentionally tests orchestration mechanics,
  not clinical safety. Human escalation and legal/jurisdiction decisions
  remain clinical/product/legal review topics.
- Verified de-escalation remains **blocked rather than guessed**: the present
  WP-08 StateCommitter rejects a lowering transition without authoritative
  evidence, and the WP-02/08 risk projection does not persist/reconstruct
  `LastSignalAt`. This integration never fabricates proof or changes WP-08
  contracts. A later, reviewed migration/contract and evidence provenance are
  needed to support safe de-escalation.
- WP-06's pure FSM may extend `FollowUpUntil` for repeated signals *without*
  a level change. The present `StateDelta` only transports a risk transition,
  so refresh-only extensions cannot yet be finalized; extending that contract
  must be coordinated with the state owner. Existing deadlines are preserved
  monotonically in SQL, never shortened on model inference.
- WP-07, WP-09, WP-10 and later modules are not implemented. `NewIntegrated`
  is opt-in; changing HTTP composition was out of scope due to parallel work.
- Database behavior must be checked against SQL Server in CI/staging. This
  environment has no DB and could not download all real Go dependencies; local
  Go tests with temporary dependency stand-ins are **not** equivalent to
  production compilation against those dependencies. The stand-ins are outside
  the delivered patch.

## Targeted verification (with real Go dependencies)

```sh
cd apps/api
go test -race ./internal/modules/ai/application/orchestrator ./internal/modules/ai/safety ./internal/modules/ai/application/assembler ./internal/modules/ai/application/state ./internal/modules/ai/adapters/mssql
go vet ./internal/modules/ai/application/orchestrator ./internal/modules/ai/safety ./internal/modules/ai/application/assembler ./internal/modules/ai/application/state ./internal/modules/ai/adapters/mssql
```
