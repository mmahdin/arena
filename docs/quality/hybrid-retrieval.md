# Hybrid retrieval / asynchronous embedding implementation report

Date: 2026-11-21. Owner: AI module. Scope: private synthetic build.

## Architecture and files

The unchanged user message produces one query embedding. The complete authorized,
visible, pre-trigger corpus feeds exhaustive cosine search and genuine Okapi BM25;
equal-vote RRF combines their ranked results before exact-text/identity deduplication
and context selection. Recent-context IDs are excluded by Orchestrator. Five-route
selection, temporal parsing, entity/anaphora heuristics and weighted relevance are
removed. Legacy contract hints are inert compatibility fields. Grounded Brain
recall uses provenance instead of the old temporal-exact flag.

- [Pipeline](../../apps/api/internal/modules/ai/application/retrieval/service.go),
  [BM25/cosine/RRF](../../apps/api/internal/modules/ai/application/retrieval/hybrid.go),
  [SQL corpus](../../apps/api/internal/modules/ai/adapters/mssql/retrieval.go).
- [Memory commit](../../apps/api/internal/modules/ai/application/memory/service.go),
  [transaction](../../apps/api/internal/modules/ai/adapters/mssql/memory_save.go),
  [worker](../../apps/api/internal/modules/ai/application/memory/worker.go),
  [index/backfill](../../apps/api/internal/modules/ai/adapters/mssql/search_index.go).
- [Decision / deployment](../adr/0014-hybrid-retrieval-index.md) and
  [source/permission audit](../../apps/api/internal/modules/ai/application/retrieval/README.md).

All memory kinds, original user/assistant messages, session summaries, thread
state, latest visible formulations, protocol inputs/summaries and open loops are
searchable. Original IDs, rowversions, dates, conversation/validity metadata,
epistemics, sensitivity and evidence counts survive retrieval. Other operational
and legacy stores were audited and their exclusions are documented. The original
evidence/version tables remain authoritative and untouched by indexing.

## Index generation and recovery

Validated writes save memory/evidence/versions/receipts and enqueue reference-only
jobs atomically. Source triggers also cover raw admission, assistant finalization
and all other indexed state writers. Neither commits nor maintenance call an
embedding provider. The existing supervised Memory Worker reads fresh authorized
content, calls the provider outside SQL transactions, then conditionally upserts
`ai_embeddings` and completes its lease in one short transaction.

Source/user-state revisions, visibility, owner, token and expiry checks prevent stale
publication. Incompatible/corrupt vectors do not block memory writes. Failures
retry with existing exponential delay/six-attempt defaults and explicit terminal
recovery. Denied/stale jobs skip model calls. Bounded backfill covers historical
records, model/dimension changes and invalidated indexes;
completed jobs do not prevent later rebuilding. Model-pinned backfill jobs cannot
be consumed by incompatible workers during a rolling deployment. Pending vectors
leave text available to BM25. Query-provider failure degrades retrieval.

## Migration / configuration

Additive [0033](../../apps/api/db/migrations/0033_hybrid_retrieval_index.sql) adds
source rowversions, the shared view/enqueue triggers, embedding revisions,
expanded owner validation and a covering owner/kind/dedupe job index. A seek-based
finalizer lookup prevents a reproduced cross-user job-scan deadlock. Scheduler
source DML now uses OUTPUT INTO to remain compatible with triggers.

`retrieval.v2` replaces route/weight configuration with context limits. Existing
runtime timeouts, model/dimension settings, ownership policies, cipher and worker
infrastructure are reused. No production environment variables, services or
dependencies were added. Stop workers and deploy migration/runtime/rebuilt signed
bundle together. Original records are retained; no destructive down migration.
The application database was not migrated during this work; live verification used
a separate loopback SQL Server 2022 container with disposable databases.

## Executed validation

| Check | Result |
| --- | --- |
| `python3 scripts/check_structure.py` | Passed |
| `python3 scripts/check_contract.py` | Passed; 66 routes unchanged |
| `go test ./...` | Passed; 53 test packages, live integration opt-ins skipped in this command |
| `go test -race` for Retrieval, Memory and AI MSSQL | All three passed |
| Full live SQL Server integration suite | 43 top-level tests passed; three pre-existing failures below |
| Refactor-focused SQL tests | Passed, including all seven source types' actual publication, encrypted content, provider-independent persistence, corrupt vectors, retry/token fencing, concurrent users, in-flight edits/revocation/deletion, backfill/model isolation, repeated rebuilding, rollback and migration preservation |
| Inspector real-service fixture regeneration / schema/graph tests | Passed; both full-turn examples refreshed, historical v1 inputs retained |
| Synthetic Persian/bilingual relevance with real provider embeddings | Passed coverage comparison; metrics below |
| `git diff --check` | Passed |

The unchanged committed code was separately exported to a temporary checkout and
tested on the same isolated SQL Server instance. These three failures reproduce
there and remain outside this refactor:

- `TestInternalAICoreSQLCompositionMultiturnEvalAndWorkers`: protocol-consent
  assertion after an existing Generator plan validation fallback.
- `TestWP11SQLValidatedDeliveryIsolationAndSoftDelete`: the fixture supplies
  cross-conversation messages as recent context; the verifier rejects them.
- `TestSoftDeleteUpgradeSQLServer`: existing unguarded-table assertion.

The baseline run also exhibited a separate `TestWP16HTTPFullCoreSQL` failure;
that test passed in the final modified-code run. No claim is made that the whole
live integration suite is green.

## Measured retrieval quality / limits

The benchmark uses 12 fabricated Persian snippets, ten Persian/English queries,
`text-embedding-3-small`, and 1,024 dimensions. Vectors were obtained from the
configured live provider; final repeat checks reused those synthetic vectors.
No application chat records were read or sent.

- Recall@1: **0.60**; Recall@3: **0.70**; Recall@5: **0.90**; MRR@5: **0.695**.
- Vector-only Recall@5: **0.90**. Fusion improves the sister/disagreement query
  from semantic rank 7 to fused rank 2, but weak lexical time/count matches push
  the exercise query out of the final five. Sleep and relocation queries retain
  weak semantic ranks. This is measured coverage, not evidence of broad ranking
  superiority; the dataset is small and synthetic.

V1 scans history exhaustively and computes BM25 in Go; CPU, memory and per-source
permission checks grow with history. Metadata-only source updates can regenerate
vectors. Very large derived records may exceed provider input limits and require
chunking later. ANN/inverted indexing, richer morphology, optional query rewriting
and specialized reranking remain deferred. No LLM query understanding, relevance
scoring, handwritten temporal interpretation or additional ranking weights were
introduced. Safety, ownership and hidden-source permission restrictions remain; ordinary
memory use no longer requires consent or an unpaused state. These engineering checks
do not validate clinical use.
