# WP-12 Memory System

Owner: `modules/ai`. Authority: Final Architecture §§3.10–3.11/7 and Contract
Pack v1.2 §§5/10.5/11.4/WP-12. This package implements the new Memory Writer,
deterministic committer, explicit user controls and in-process workers. It does
not call the experimental responder, Daily Reflection, Protocol Engine,
psychological Reflector, Scheduler, HTTP adapters or later work packages.

## Composition and activation

Apply migration `0021_ai_memory_system.sql` through the existing migration
runner after 0020. Call `composition.NewAIMemory` with the same immutable WP-03
snapshot as Safety, Brain, Generator, Validator and Retrieval; the existing
WP-04 client/embedding client; WP-09's embedding model/dimension; an explicit
`memory.Policy`; a deployment-managed `Cipher`; a technical `FeedbackPolicy`;
and a unique worker ID. Inject `Committer` as `orchestrator.Dependencies.Memory`.
Run a bounded number of `Worker.Run(ctx)` loops under application shutdown.
No HTTP route or existing experimental runtime is activated by this factory.

The policy is mandatory and deliberately has no permissive production default.
It receives current stored consent, actual owner-scoped evidence, the proposal,
source authority and existing memory metadata. Its reviewed implementation must
authorize collection, disclosure/use, storage, volunteered sensitive content,
information-schema collection/confirmation conditions and third-party data
minimization. `ExplicitChange` and `Endorse` require an attested user action;
model reason tags are insufficient. Hidden raw sources require the additional
explicit `UseRetainedSource` permission; future-turn evidence is always refused.
Visibility alone neither authorizes reuse nor establishes a permanent reuse ban.

All shipping `knowledge/` assets remain synthetic placeholders. Production
loading rejects synthetic bundles and placeholder Writer prompts. Clinicians
must approve extraction criteria, the Writer prompt, locale hedge/clinical-label
lists and goldens. Privacy/security owners must approve consent keys, use policy,
provider disclosure, key custody/rotation and the technical feedback settings.
The test-only bundle and consent key are fabricated engineering fixtures.

## Processing and recovery

The Orchestrator reads only consent and the current source when scheduling. It
encrypts bounded analysis/claims/tags and attaches a job to the existing final
assistant/state transaction. FAST, crisis and template fallbacks do not schedule
extraction. A scheduling-preparation outage degrades the accepted reply; a later
explicit consolidation can recover persisted sources. A known correction must
commit synchronously before generation; a correction failure prevents delivery.
The corrected source is removed from permitted context/hypotheses/formulation.

The separate small-tier Writer assembles §11.4's three cached system blocks and
four XML-escaped dynamic blocks. Message/item UUIDs are attributes, roles and
epistemics remain explicit, neighbors are capped at five and selected using
cached WP-09 embeddings or lexical similarity. Limits are 800/250/500/150 dynamic
tokens, 3,000 total estimated input tokens and 500 output tokens. WP-04 performs
structured generation/repair; WP-01 strictly decodes `memwriter.v1`. Batches have
at most eight operations, six evidence messages per operation, 320 characters
and forty words per item. WP-08-validated support/contradiction signals reserve
schema capacity and join the same atomic memory batch; they do not author prose
or turn a hypothesis into a reported fact.

MC-001–013 run before persistence, including before candidate text can reach an
embedding provider. Assistant claims are context evidence and never increase
reported support. Hypothesis confidence/status use distinct supporting sources
and sessions. Changed identified attributes take precedence over cosine dedupe;
ambiguous changes contest both records and create `confirm_fact` loops. Older
asynchronous proposals cannot overwrite newer confirmed values. Sensitivity
downgrades are refused. User edits retain priority and get a separate immutable
user-confirmation event; old message evidence remains context.

All provider and encryption work happens outside SQL transactions. Candidate
similarity uses WP-04 embeddings; publication into the retrieval index is a
separate post-commit job with fresh consent/content/version checks. Existing
vectors are retained. Embedding/provider failures retry the checkpointed batch.
Memory, evidence, operation receipts, batch receipt and job completion share one
short owner-locked SQL transaction. The same durable user lock as WP-02 plus
`memory_revision`, user/item versions and lease tokens fence concurrent work.
Receipts enforce `(turn_id, normalized_op_hash)` and command/batch idempotency;
changing a reused operation ID's body returns a conflict.

Two-minute database leases, bounded attempts, per-user extraction quotas and
encrypted checkpoints support crash recovery. A process stops work before its
lease expires; stale processes cannot commit, acknowledge or reset reclaimed
jobs. `RetryJob` explicitly recovers terminal failures while preserving cumulative
attempts and checkpoints. No raw model/provider/SQL errors enter job error codes,
metrics or the audit outbox. Keep old encryption keys available: ciphertext is
owner/purpose bound, and retained historical data must remain decryptable.

## Controls, maintenance and retention

Authenticated adapters should supply `memory.WithMeta` using the server-resolved
user, a stable operation UUID, the pinned bundle and optional originating turn.
`List`, `Edit`, `Endorse`, `Correct`, `Pin`, `SetControls`, `Suppress`, `Delete` and
`RetryJob` are application commands for future owning HTTP adapters, not new
routes. Explicit preferences update the existing WP-08 user state. Pausing or
revoking collection does not disable corrections/deletion. Suppression requires
its own explicit command and is never inferred from deletion.

`Maintain` performs deterministic evidence-preserving dedupe/cluster merges,
staleness and hot-set caps. Accepted-turn retrieval feedback updates use time and
reduces importance for repeatedly retrieved, never referenced items using the
explicit technical configuration. `ScheduleMaintenance` queues the same work.
`Consolidate` is an explicit bounded Writer pass over 2–12 source IDs, emitting
ordinary episode memory proposals. WP-14's session-close psychological summaries,
formulations, trigger engine and Scheduler remain unimplemented.

Deletion marks UTC timestamp, authenticated actor and `user_request`; repeated
deletion retains the first metadata. It never removes or clears content, evidence,
vectors, historical versions, events, jobs, traces, consent or derived records.
Conversation hiding does not invalidate independently eligible memories/index
jobs. Every new data table has a physical-delete guard and ownership FKs use
`NO ACTION`. Original versions are archived before mutations. No purge, cascade,
retention cleanup, key destruction or destructive rollback is included.

Deploy by expansion; rollback disables the new factory/workers and retains the
added columns/tables. Do not drop or truncate them. `List`/snapshot processing is
bounded at 1,000 visible records and fails with a budget error beyond that cap;
paginated public memory APIs belong to WP-16.
