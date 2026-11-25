# WP-12 Memory System verification

## Scope and integration

Implemented against the Final Architecture Document and Contract Pack v1.2:
the small-tier structured Memory Writer; MC-001–013 reconciliation and counted
hypothesis transitions; immutable provenance/evidence; dedupe, contest, supersede,
correction and explicit endorsement; preferences, consent, pause, pinning,
suppression and soft-delete controls; transactional receipts/version history;
leased asynchronous extraction, indexing, feedback and maintenance; bounded
episode-memory consolidation; encryption and owner isolation.

The patch is a delta against the workspace captured before WP-12, including its
already implemented WP-10 and WP-11 files. It does not repackage those modules.
Shared Orchestrator changes are only the WP-12 handoff, correction boundary,
bundle check and test dependency injection. No legacy responder is imported,
no HTTP cutover is made, and WP-13/14/15/16 remain outside this work.

Migration 0021 extends existing memory/user/job rows, permits actor-bound memory
events outside chat while retaining owner FKs, and adds receipt/history/feedback
tables with `NO ACTION` FKs and physical-delete guards. It never clears, purges,
physically deletes or recursively hides source or derived data. Forward deployment
preserves existing data; rollback disables the new composition/workers and retains
all schema/data. Applied migrations 0001–0020 are unchanged.

Use `composition.NewAIMemory`, inject its committer into `NewIntegrated`, and run
its worker under graceful shutdown. Required dependencies are the pinned reviewed
bundle, existing WP-04 client and embedder, SQL Server 2022, the same embedding
model/dimension as WP-09, managed encryption keys, an explicit consent/use policy
and technical retrieval-feedback settings. The host can explicitly call
`Consolidate` or `ScheduleMaintenance`; no WP-14 trigger engine is supplied.
Future authenticated memory adapters must pass server-resolved identity and stable
operation IDs with `memory.WithMeta`. See the module README for the full contract.

## Executed checks

Environment: local Linux workspace, Go 1.25.4, existing local SQL Server 2022
container. Live SQL tests used isolated synthetic databases with the repository's
existing integration fixtures. No live LLM or real disclosures were used.

| Check | Result |
| --- | --- |
| `go test ./...` | Passed; SQL opt-in cases skip in the ordinary run |
| `go test -race ./internal/modules/ai/... ./internal/app/composition ./tests/integration` | Passed |
| `go build ./...` | Passed |
| `go vet ./...` | Passed |
| `python3 scripts/check_structure.py` | Passed |
| `python3 scripts/check_contract.py` | Passed; no new routes/contracts |
| Live SQL: `go test -race ./tests/integration -run '^(TestWP12SQL\|TestSoftDeleteUpgradeSQLServer)' -count=1 -v` | Passed |
| Full live SQL integration package | All tests passed except the pre-existing WP-05 test below |
| Untouched pre-WP-12 snapshot: `TestWP05SQLServerFailureAndOriginalRequestRecovery` | Same failure reproduced |

The full live package is therefore **not entirely green**:
`TestWP05SQLServerFailureAndOriginalRequestRecovery` fails at
`ai_wp05_test.go:189` with `terminal failure was not idempotent`. This is the old
provider-call-count expectation around regeneration and reproduces without any
WP-12 changes. It was deliberately not changed in this WP-12-only patch.

Unit coverage includes evidence ownership/identity and hidden/future evidence,
explicit retained-source eligibility, assistant-vs-user epistemics, approved
tags/hedges and clinical-label rejection, pause/storage/use denial, suppression,
sensitivity, content/token bounds, schema failure, provider failure, prompt
escaping and provenance handles, semantic and canonical dedupe, incompatible
entities, changed attributes despite a high cosine, older asynchronous updates,
independent-session confidence/FSM behavior, user-edit precedence, maintenance,
signal operation capacity and encryption owner/purpose/tamper/rotation checks.

Live SQL coverage includes fresh/repeated and data-preserving upgrade migrations,
atomic accepted-turn job scheduling, extraction checkpoints, post-commit indexing,
durable replay, reported/hypothesis evidence and independent sessions, separate
reported endorsement with the original hypothesis retained, user corrections and
cooldowns, encrypted highly sensitive items, owner controls, consent revoked with
pending work, explicit suppression, message and memory soft-delete retention,
blocked physical deletion, concurrent identical operations, stale/reclaimed job
leases, transaction rollback and terminal retry using the encrypted checkpoint
without another extraction call, bounded consolidation, leased staleness,
retrieved-but-unused feedback, quota/bundle fencing and preserved derived memory
eligibility after its conversation is hidden. The existing broad SQL lifecycle,
privacy and soft-delete suites also pass.

Orchestrator integration tests prove synchronous correction happens before any
generation, disputed content is excluded, correction failure prevents delivery,
jobs join the existing finalizer transaction and a scheduling-preparation outage
degrades a valid reply without publishing an unsafe memory proposal.

## Release blockers and limits

Production activation remains blocked on clinician-reviewed extraction criteria,
Writer instructions, locale assets and memory quality goldens, plus privacy/security
approval of consent keys, collection/confirmation conditions, volunteered sensitive
content and third-party minimization, provider disclosure and key management.
The mandatory policy receives actual evidence and proposal metadata; the code does
not invent a clinical collection policy or assume permission from visibility.
Shipping Knowledge Bundle assets are still inert synthetic placeholders. Test
fixtures demonstrate engineering behavior, not clinical validity or live-model
quality. No clinician-facing diagnostic or treatment content was added.

Snapshots/user memory reads are bounded at 1,000 visible items and return a budget
error beyond that bound; public pagination belongs to WP-16. Candidate similarity
is computed outside SQL transactions and index publication is retried after the
memory transaction. Provider outages may delay memory while keeping the original
turn/checkpoint available. An extraction scheduling-preparation outage can require
a later explicit bounded sweep. Episode consolidation here is an ordinary bounded
memory proposal; WP-14's psychological session summaries/formulations and triggers
remain unimplemented. No erasure, retention cleanup or destructive schema rollback
is authorized or included.
