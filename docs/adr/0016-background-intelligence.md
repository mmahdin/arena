# ADR 0016 — Bounded background intelligence and durable message coverage

Status: implemented in the internal synthetic build, 2026-11-24.

Phase 2 preserves the complete Phase 1 synchronous architecture. Background
models do not participate in admission, JEV, Unified or response finalization.
The existing `memory_write` and `reflection` queues and workers run the new
`application/background` service. Its two responsibilities share preparation,
validation, checkpoint recovery and the existing memory transaction. No agent,
selector service, new external dependency or queue is introduced.

Routine extraction is no longer scheduled by each turn. An independently timed
worker sweep reserves bounded batches of completed user messages. A per-message
ledger associates a source revision with a durable job and acknowledges it only
in the transaction that publishes a successful review. Empty valid reviews also
acknowledge their messages. Inference/provider/storage failures do not. Source
edits produce new job identities; exhausted work remains visibly unreviewed and
requires the existing explicit retry command. There is no 24-hour cutoff. Fair
owner scans and an idle flush allow partial batches to run without another user
turn. Pause stops claims; the foreground user lease takes priority.

Explicit Unified requests remain separate encrypted tasks. Their queries use the
existing hybrid retriever once per supplied query, without iterative retrieval.
Prepared input separates primary messages, structured memories, prior derived
understanding and operational records. Retrieved authorized user messages are
usable evidence; retrieved authorized memories are actual mutation targets. Up
to 24 targets replace the old five-neighbor contract. Scope is either the owner
or an explicit list of at most four owned threads.

Memory Intelligence owns reported facts/preferences/goals/events and relationship
entities. Reflection owns tentative hypotheses and formulations. The model chooses
semantic actions: add, confirm, revise, contradict, supersede, consolidate,
invalidate or noop. Go validates identity, ownership, primary evidence, policy,
bounds, source versions and concurrency. Scope/primary-lineage-aware exact identity deduplication
is operational; there is no semantic similarity merge, support-count promotion,
ProblemTags framework selection or psychological scoring in the active executor.

The authored framework cards and their two small complete documents are supplied
together in the same Reflection inference. The model selects zero/one/several
framework IDs and may omit formulations altogether. This avoids another selector
call and its cost/latency; it adds a bounded static input cost per Reflection call.
Authored field/status/evidence prerequisites remain operational validation.

Raw user evidence is required for substantive mutations. Assistant text, previous
hypotheses and formulations can provide context, never independent support. Derived
items retain inferred epistemic status, tentative text, uncertainty/alternatives
in the encrypted job result, primary evidence, contradiction links and archived
revisions. Model-selected proposed/supported status never promotes a hypothesis to
a fact or produces a calibrated clinical probability. The original Reflection risk
authorization boundary and all Unified Safety assets/instructions remain unchanged.

New results reuse `ai_memory_items`, evidence/version/batch records, formulations,
reflection runs and encrypted job checkpoints. Migration 0042 adds only coverage
and expiring recommendation records plus nullable formulation/reflection scope.
Corrections and explicit memory deletion invalidate dependent understanding without
erasing original records. Visibility-only conversation/raw-message hiding remains
separate from permission to use independently retained derived information.

A single transaction fences prepared revisions, stores accepted mutations and
rejections, appends formulation versions, writes scoped encrypted hints, acknowledges
coverage and completes the leased job. Stale checkpoints are cleared for a bounded
fresh retry; they cannot overwrite current data. A concurrently admitted foreground
turn defers background publication. The existing hourly/daily quotas and bounded
attempts continue to apply.

Hints carry provenance, evidence and source revisions. They expire within three
accepted turns/72 hours and can be invalidated. Loading them is optional and bounded;
a hint failure cannot reject a conversation. No background output can create another
background request. Inspector supports standalone opt-in job files when the original
turn trace is absent, including user-level analysis with no invented turn identity.

Old serialized `memory_job.v1`/`reflection.v1` artifacts and isolated regression
executors remain for pinned-job compatibility and authenticated controls. Their
standalone composition factories are removed. Old bundle hashes must be drained by
workers running their matching release, never silently reassigned to the new bundle.
See `docs/ai-core/background-intelligence.md` for contracts, limits and diagrams.

The active worker constructors do not instantiate legacy Memory Writer or
Reflection models. Legacy semantic maintenance is rejected by the new executor;
old versioned worker constructors remain only for their matching pinned releases
and isolated compatibility tests. Authenticated memory controls and index work
continue to reuse the existing operational services.
