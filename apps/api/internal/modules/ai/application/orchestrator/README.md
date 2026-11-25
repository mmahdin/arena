# Unified conversation orchestration

`NewUnified` is the only conversation constructor. See ADR 0015 and
`docs/ai-core/unified-memory-centric.md` for the architecture and contracts.

The authenticated gateway durably admits the original user message and acquires a
fenced user lease. Duplicate completed turns replay without model calls. Retrieval
starts immediately alongside state/persistent-context reads. The assembler builds
bounded `wctx.v3`; JEV selects cards; Go loads pinned full resources; Unified emits
`unified_turn.v2`. No Brain/Generator, router, regeneration or fallback path remains.
Safety remains inside Unified with the existing instructions and transition rules.

Independent optional proposals are validated and resolved in dependency order.
Protocol handling checks actual persisted consent, state and exact delivered steps.
Go enforces operational prerequisites, never psychological fit scores. Invalid
independent proposals are recorded; replies requiring rejected operations are withheld.
Storage conflicts and lease failures still prevent partial publication.

The existing Conversation finalizer commits the reply, accepted state, immutable
owner-scoped decisions, expiring recommendations and encrypted background tasks in
one transaction. Workers perform analytical retrieval/model calls after commit.
Routine memory review and its bounded periodic sweep do not require model requests.
Unified failure retains the original durable user-only turn. Authenticated retry,
FIFO recovery, cancellation and heartbeat fencing retain their existing semantics.

All components must use the same pinned bundle. Runtime `ai_runtime.v4` reserves
one JEV and one Unified generative call. Options expose only recent-message limit,
total deadline and lease renewal interval. Migration 0041 must precede deployment.

Run `go test ./internal/modules/ai/application/orchestrator` from `apps/api`.
SQL integration tests in `tests/integration` exercise actual composition, replay,
atomic rollback, owner isolation, background execution and stale research fences
against disposable synthetic SQL Server databases.
