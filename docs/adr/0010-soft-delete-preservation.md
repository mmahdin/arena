# ADR 0010: Preserve data with user-initiated soft deletion

Status: accepted for the WP-02 correction, 2026-11-16.
Supersedes ADR 0009's purge/cascade behavior and earlier table-retirement gates.
Authority: the user's correction and the amended Contract Pack v1.2 §7.4–7.5.

## Decision

All current user deletion endpoints change visibility only: conversations,
bookmarks, Daily Reflection diary entries, and personalized letters. Logout
continues to revoke the session without deleting its row. No account/memory
HTTP API or permanent-erasure mechanism is added.

Migration `0020_soft_delete_preservation.sql` adds nullable `deleted_at`,
`deleted_by` and `deletion_reason` to those four resource tables, raw messages,
AI memory items and formulations. A null timestamp means visible. Deletion uses
SQL Server UTC time, the server-resolved owner UUID and `user_request`; the check
constraint requires consistent metadata. Repeated deletion retains the timestamp.
Metadata on messages/memories/formulations prepares storage for later explicitly
scoped APIs; this correction does not introduce those APIs.

Conversation deletion marks only the owned conversation. Messages and bookmarks
are hidden through the parent visibility predicate, retaining their own original
rows and metadata. Independently stored letters, memories, analyses, evidence,
embeddings, state projections, protocol inputs, events, jobs, outbox, traces,
tombstones and preferences remain intact. No derivative is automatically marked,
invalidated, regenerated or removed. A letter remains independently visible until
its own delete operation is invoked. Wrong-owner access still returns 404.

The conversation owner row lock serializes deletion against WP-02 admission and
finalization and the transitional prototype claim/completion paths. Deletion
expires only a lease belonging to that conversation; queues skip hidden parents.
Late completions, duplicate result replay, admission, rename and history reads
cannot expose the hidden conversation. Other conversations and their workers are
unaffected. Lease release/reacquisition now updates the control row rather than
removing it; a released expiry is at least one millisecond in the past to avoid
`datetime2(3)` rounding briefly leaving it active. Prototype failure recovery
retains its reservation and expires the lease, preserving same-key retry.

A new diary entry, bookmark or letter after deletion receives a new physical row.
Filtered unique indexes enforce one active record for the logical key, retaining
all deleted revisions. Diary save/delete locks the owner before changing the
active key, avoiding insertion races across the new surrogate primary key.
Editing an existing visible diary entry retains its existing update semantics.

## Preservation is independent of AI use

No consent, `memory_paused`, proactive preference, evidence count, risk state,
suppression list or semantic memory status is changed by conversation deletion.
The retained records neither authorize AI reuse nor establish a new reuse ban.
This correction implements normal resource visibility and processing boundaries;
it does not implement retrieval, Memory Writer, Reflection or Scheduler policy.
Those future components need separately approved use eligibility. The current
letter-generation use case selects visible source conversations and messages;
previously generated letters retain their independent visibility.

There is no expiry cleanup or legal-erasure implementation. Historical trace
expiry metadata remains stored; new traces have a null expiry. Permanent deletion,
legal erasure, backup disposal and retention schedules are a separate future
policy. Product/API documentation must describe hiding and retention accurately.

## Migration and deployment

Do not change applied migrations `0001–0019`. Apply `0020` through the existing
checksummed transactional runner, with writers stopped during the schema change.
It adds metadata without changing original payloads, backfills surrogate UUIDs
for diary/bookmark rows, replaces active uniqueness, and recreates every existing
cascade FK with its original ordered columns and validated `NO ACTION` references.
Ownership constraints remain enforced. It drops the obsolete purge procedure and
replaces the conditional raw-delete guard with physical DELETE guards on all
existing persisted data tables, including AI state and leases. The migration
ledger is excluded. Future resource migrations must preserve the same invariant.

These guards reject application/raw DELETE statements, not administrative DDL
or a privileged operator disabling safeguards. No application code uses TRUNCATE,
DROP TABLE or content-clearing as an alternative. Test teardown drops only the
suite's isolated disposable databases. All existing data tables, including
`daily_reflections`, remain; feature code retirement never authorizes data loss.

Old binaries cannot safely run after this schema change: their purge/delete paths
will fail, and old writers do not account for filtered active uniqueness. Deploy
the schema and corrected binary together. Roll forward with a reviewed migration;
there is no destructive down migration or automatic restore/undelete endpoint.
This correction cannot reconstruct data physically erased before it was deployed.

## Verification

See [the correction report](../quality/ai-core-soft-delete.md). Integration tests
use a dedicated SQL Server 2022 container and synthetic disposable databases.
They compare stored original/derived data, test visibility and re-creation,
ownership, blocked physical deletes, full migration/upgrade integrity, rollback,
lease recovery, independent conversations and concurrent delete/finalize ordering.
