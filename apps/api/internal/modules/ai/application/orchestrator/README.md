# WP-05 — Gateway and Orchestrator walking skeleton

This directory contains the in-process `ports.Orchestrator` implementation for the
**new** AI Core. It is deliberately not connected to the old
`conversation.Service.Turn` / experimental reply pipeline. No external RPC and
no additional database schema are required.

## Durable lifecycle

1. The application composition boundary supplies the **authenticated** user ID
   and normalized request to `ProcessTurn`. `gateway.New(store, recovery)`
   delegates ownership checks and admission to the existing Conversation-owned
   WP-02 SQL Server `TurnStore`. Admission commits the original user message and
   durable turn **before** model inference; a per-user fenced lease serializes
   turns across conversations. Duplicate in-flight/queued/finalized admissions
   return their persisted status, not a second model invocation.
2. The Orchestrator reads visible recent messages, assesses safety and optionally
   retrieves context concurrently, assembles a bounded working context, and
   applies its deterministic `Router`. For a normal turn the supplied Brain port
   is invoked; trivial fast turns skip Brain. High-risk turns require an
   externally supplied Crisis Planner. Wake-up origins cannot be admitted by
   WP-02 yet, so proactive output is deliberately disabled.
3. A supplied Generator writes into an **internal** guarded token buffer, never
   directly to an HTTP stream. The supplied Validator must accept the complete
   reply. A fallback is allowed only when provided by an independently reviewed
   `Options.Fallback` or the injected Crisis Planner, and that fallback must
   also pass final validation. With neither source, the turn fails safely;
   WP-05 contains **no** psychological scripts or approved fallback content.
4. Only after validation, `TurnFinalizer.Complete` atomically stores the assistant
   reply, minimal state delta, redacted trace summary, domain outbox event and
   lease release. `Fail` and `Cancel` retain the original user message; they do
   not insert an assistant reply. No AI stage holds an open SQL transaction.
5. After a crash or lease expiry, an authorized worker can call
   `RecoverNext(ctx, authenticatedUserID)`. It loads the **original** persisted
   request. `RetryFailed(ctx, authenticatedUserID, clientTurnID)` is a distinct,
   explicit operation that first requeues a terminal unsuccessful turn. Simple
   duplicate requests never requeue failures or cancelled turns.

`trace.NewRecorder()` is a test/development metadata-only tracer; it does not
store prompts, message text, model output or exceptions. The durable WP-02
finalizer separately persists the minimal trace summary. A production encrypted
trace/retention backend and full Bundle/Policy integration belong to later WPs.

## Wiring boundaries (not changed by this patch)

The later composition/API work package should inject:

- The *same* WP-02 Conversation `TurnStore` instance as
  `ports.ConversationStore`, `ports.TurnFinalizer` and `ports.TurnRecovery`, plus
  `gateway.New(turnStore, turnStore)` for `ports.Gateway`. That store must be
  constructed using WP-02's AI-owned SQL state participant, on the same DB.
- The real WP-04 LLM client into WP-10 Brain and WP-11 Generator adapters when
  those packages exist. Until WP-06/07/08/09/10/11 are implemented, inject
  **test-only** Safety, Context, Brain, Generator and Validator adapters in tests
  only. Do not expose the skeleton as a clinical production endpoint.
- An explicit `RouterConfig` and independently reviewed fallback texts when
  available; none is loaded from the concurrent unfinished WP-03 bundle.
- An appropriately provisioned lease TTL: the default total turn budget is 45s;
  configure the WP-02 lease TTL **longer** than the turn budget and database
  commit time, or enable `RenewInterval` with `ports.TurnRecovery`. The
  Orchestrator cancels work if renewal fails and fences out stale writers.

HTTP/auth, OpenAPI, deploy-time composition, legacy endpoint retirement,
clinical safety gating, memory, protocols, active state logic and streaming
transport are deliberately out of scope. The `UserID` passed into `ProcessTurn`
**must be derived from the authenticated request**, not trusted from user JSON.

## Verification

- Unit/contract tests: `go test -race ./internal/modules/ai/application/gateway ./internal/modules/ai/application/orchestrator ./internal/modules/ai/trace` from `apps/api`.
- Optional SQL Server integration: `WELLBEING_INTEGRATION=true go test ./tests/integration -run '^TestWP05' -count=1` with the existing test SQL Server environment. These tests use real WP-02 lifecycle SQL and real WP-04 client, with scripted provider and non-clinical stage adapters.
- No migration, shared contract, KB bundle or product-route changes are needed for this WP.
