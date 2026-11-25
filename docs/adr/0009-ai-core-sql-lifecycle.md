# ADR 0009: AI Core SQL persistence and turn lifecycle (WP-02)

Status: implemented for synthetic tests; no AI runtime cutover.
Deletion behavior is corrected by [ADR 0010](0010-soft-delete-preservation.md).
Authority: Final Architecture and Contract Pack v1.2 §§2.3, 4.1, 6.1–6.2,
7, 9.2, 10.2, 12.3 and WP-02 in §13.1. This extends ADR 0008.

## Boundaries

Conversation owns `dbo.conversations`, `dbo.turns`, `dbo.messages` and the short
admission/finalization transactions. Its MSSQL `TurnStore` implements the AI
consumer-owned storage, finalizer, serialization and recovery interfaces. AI's
MSSQL `StateStore` owns SQL for `ai_*` data. Composition injects this infrastructure
participant through `NewAIPersistence`; `sql.Tx` is restricted to adapters.
Domain/application contracts never expose a transaction or database driver.

No model, prompt, Gateway, Orchestrator, worker, retrieval algorithm, semantic
state FSM, HTTP route or OpenAPI behavior is implemented here. The persistence
adapter accepts an already validated `PreparedTurn`; WP-05/06/07/08/13 must supply
the pipeline, safety/output validation and semantic state/protocol policies.
The new persistence path imports no prototype service, prompt or safety policy.
Prototype replacement remains bounded by WP-04/05/16 as recorded in ADR 0008.

## Admission, concurrency and recovery

- Normalize raw user text to NFC; reject invalid UTF-8, blank input, invalid IDs,
  unknown timezone and input beyond the existing SQL UTF-16 storage bound.
  This foundation admits user turns; scheduled wakeups remain WP-14.
- `(user_id, client_turn_id)` is the idempotency scope. A server-generated turn ID
  is distinct from the client key. Hash the normalized request including its
  conversation and semantic metadata; ignore client timestamp changes on retry.
  Store request metadata without duplicating raw text, enabling `Next` after restart.
- Lock the owner's existing `users` row with `UPDLOCK,HOLDLOCK` at the start of
  every mutation. Check conversation ownership inside that short transaction.
  Persist the turn and its sole USER message before returning admission.
- Use FIFO across all conversations of one user, ordered by a durable identity
  column. Every queued message retains its own turn and raw message ID; there is
  no coalescing. Only the oldest queued/running turn can acquire a lease.
- Lease time comes from SQL Server. A fresh random token fences every acquisition;
  duplicates never receive a live worker's token. The configured duration must be
  1 second–1 hour (composition uses 2 minutes); renew before expiry. There is no
  transaction held during inference. Different users can proceed independently.
- `Next(user)` returns the oldest recoverable request. Call `AdmitUserMessage` to
  claim it and refresh session/user versions. An abandoned running turn remains
  durable and becomes reclaimable at expiry. `Acquire` requires an existing head
  turn: it never invents a turn or drops queued input.
- `Fail`/`Cancel` require the current unexpired token; terminal failure retains
  the USER message. `CancelQueued` only changes a queued turn. A terminal duplicate
  is replayed, never implicitly restarted; `Retry(user, clientTurnID)` is the
  explicit recovery operation. The future caller must authorize it. Failed
  terminal updates can be repeated without changing their result.

## Atomic finalization

`Complete` checks owner, conversation, user-message identity and lease; inserts
only the validated ASSISTANT row; applies session/thread/protocol/user patches
with optimistic versions; persists open loops/follow-ups, state events, jobs,
turn context, trace summary and outbox; marks the turn complete and releases the
lease. It checks token/expiry again at release and returns success only after
`Commit` succeeds. Any error rolls back all these changes, leaving admission intact.
A completed retry returns the same stored assistant and result metadata.

ST-001 versions, reference ownership, evidence roles, enum/count checks and
protocol consent references are persistence guards. The user version and lease
serialize secondary-thread operations. Clinical transition legality, summary
budgeting, session closure/resumption, support checks and policy-specific repair
belong to WP-08/13. The finalizer does not silently repair an optimistic conflict;
the future orchestrator must reload/re-evaluate before retrying.

State deltas retain restricted user-derived data in `ai_turn_events`; ordinary
trace summaries and outbox envelopes contain IDs/version/route/objective metadata,
not message text. Domain-event detail is available in the restricted delta;
outbox envelopes reference its turn. Jobs are durable inserts only, deduplicated
by owner/kind/key while queued/running. A reused key with different payload fails
completion. Follow-up proposals without proactive consent are persisted as skipped.
There are no workers, external deliveries, retention jobs or prompt capture.

Embedding storage codec v1 is finite, normalized IEEE754 float32, little endian,
with separate version/model/dimensions metadata. Decoding rejects mismatches and
non-unit vectors before any future ranking. No vector search is implemented.

## Migration and deployment safeguards

Apply immutable migrations `0017_ai_turn_lifecycle`, `0018_ai_core_state`,
`0019_ai_owner_integrity` using the existing runner. Stop writers for expansion;
backfill/index creation takes schema locks. The runner serializes migrations,
verifies old checksums and makes each file transactional. If an old turn contains
multiple messages of the same role, the unique index intentionally fails and
rolls back `0017`; investigate the data rather than deleting duplicates.

`0017` preserves historical records as lifecycle version 0 and maps their client
keys from existing turn IDs. Existing processing/completed checks remain valid
for those rows. Version 1 uses queued/running/completed/failed/cancelled and a
per-user lease instead of the old per-turn lease. Old migrations, raw messages,
message ordering and historical prompt metadata are not rewritten or deleted.
`0018` adds the §7.2 AI table families; `0019` adds composite owner references,
polymorphic-reference checks, consent guards and deletion safeguards.

This expansion does not switch the HTTP pipeline. Do not run prototype and new
turn processors concurrently for the same users. WP-05 must retire the old
processing entrypoint before enabling the new one. After migration 0020,
do not roll back to an old binary with destructive deletion or purge paths. Roll
forward with a corrective migration or restore an approved backup into an
isolated environment; the runner has no destructive down migrations.

`dbo.daily_reflections` is intentionally retained. The amended v1.2
§7.1 requires preservation of its table and all original rows; no drop-table
migration is authorized in this phase. WP-16 owns the existing
route/code retirement. This table is never consumed by AI Reflection. The admin-
maintained `migration-log.md` is not advanced by this change.

## Privacy and deletion

Migration 0020 replaces all physical delete cascades with validated `NO ACTION`
references and blocks accidental DELETE operations. Conversation deletion marks
only its visibility and expires its processing lease. Every original and derived
record remains intact; other conversations and AI consent are unchanged. Normal
history/bookmark views hide the selected conversation through owner/visibility
joins. Independent derived records retain their own visibility. No permanent
deletion, retention cleanup or implicit AI-use policy is implemented. See ADR 0010
for migration, concurrency and query details.
