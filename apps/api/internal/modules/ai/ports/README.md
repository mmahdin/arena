# AI Core consumer-owned ports (WP-01/02)

These are in-process interfaces from Contract Pack v1.2 §6. Composition wires storage through `NewAIPersistence`; AI runtime
components remain future work. They do not depend on `ReplyGenerator`, prototype prompts,
the existing provider adapter, or the unrelated Daily Reflection feature.

`ConversationStore` owns access to the single durable chat store. `Gateway` owns
admission and `Orchestrator.ProcessTurn` owns the new workflow. `TurnFinalizer` is
the adapter-level transaction boundary; no SQL transaction appears in these ports.
Admission's owner and conversation IDs must match its durable record and lease.
Finalizers must fence stale tokens on both completion and failure.

All I/O operations accept `context.Context` first. Small synchronous operations
(`Router.Select`, `PolicyGate.Gate`, stream guards, trace accumulation, clock and
immutable bundle metadata access) retain the specification's context-free shape.
`Stream.Recv`/`Close` and token sinks belong to the originating call's context;
the provider must unblock reads on cancellation and release resources on close.
Internal streaming does not expose SSE or unvalidated tokens to HTTP clients.

`contracttest.CheckLifecycle` and `CheckLeases` are reusable adapter suites. Factories
provision isolated synthetic users/conversations and deterministic failure/clock
controls. They cover user-first admission, completed/in-flight replay, changed-input
conflicts, cross-owner rejection, failed turns, completion failure, cancellation,
and per-user lease recovery/fencing. Both a test-only reference implementation and the WP-02 SQL Server adapter run
the suite. SQL-backed fixtures provide a durable `LeaseTurn` instead of leasing
nonexistent IDs. Prepared fixtures include valid state versions and objectives.
Additional integration tests cover atomic state/jobs/trace writes, SQL rollback,
concurrent adapter instances, final transaction failure and restart recovery.

`TurnRecovery` adds lease renewal, running/queued cancellation, explicit terminal
retry and an owner-scoped durable FIFO read. These are persistence operations;
the Gateway and background consumers are implemented in later work packages.

Deletion-related ports mean soft deletion of the explicit scope only; report
counts mean newly hidden records, never physical erasure. Derived records and
AI consent are not changed implicitly. Normal conversation access, result replay
and recovery exclude hidden conversations. See Contract Pack v1.2 §7.4 and ADR 0010.

## WP-04 model interface

`Client.Structured` and `Client.Stream` retain their v1.2 signatures. The new
`EmbeddingClient` is an additive, provider-neutral companion for the embedding
model tier, with explicit vector dimensions/model ID in returned values. See
`../llm/README.md`; the old `Generate` gateway remains transitional only until
WP-05 replaces the runtime and is never imported by the new implementation.
