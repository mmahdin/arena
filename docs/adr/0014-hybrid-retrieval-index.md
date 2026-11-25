# ADR 0014 — Unified hybrid retrieval and rebuildable embeddings

- **Status:** accepted and implemented for the private synthetic build
- **Date:** 2026-11-21
- **Owner:** AI module; Conversation retains raw message persistence ownership
- **Authority:** explicit user request to replace the retrieval architecture

## Decision

Use one query path: the unchanged current user message, mandatory semantic
search, genuine Okapi BM25, equal-vote Reciprocal Rank Fusion, then deduplication
and bounded context selection. Do not interpret query intent using handwritten
keywords or calendar rules. Do not add a query LLM, reranker, weighted relevance
formula or separate search service. Existing bounded Brain context retries call
the same pipeline with current context exclusions; they do not rewrite queries.

SQL Server 2022's optional full-text subsystem is absent from the tested image.
Microsoft documents BM25-based FREETEXTTABLE ranking, but other full-text rankers
are different algorithms. Implement BM25 directly in Go with identical English/
Persian analysis for every source and with statistics over the authorized corpus.
Cosine search considers every compatible current vector, with no preselection by
importance, recency, memory type or query terms. RRF consumes both complete
rankings; only final context selection is limited to the requested result depth.
The independent source projection, query boundary, pure rankers, fusion and selection support optional future stages without changing
persistence. They do not implement those future stages now.

Index all seven narrative source types documented in the
[Retrieval README](../../apps/api/internal/modules/ai/application/retrieval/README.md).
Keep actual authorship/epistemics and source IDs, versions, timestamps,
conversation/validity metadata and original evidence. Operational receipts,
control/scheduler metadata, already referenced records and separate legacy product
stores are not silently promoted into additional personal memory authorities.

Treat embeddings as a rebuildable index. Necessary memory validation/reconciliation
precedes a short transaction that saves content/evidence/versions/receipts and
queues a source reference atomically. Source-table triggers cover every writer,
including raw turn admission and assistant finalization. Models and encryption
backends never run in that transaction. The existing supervised Memory Worker
loads current authorized content and generates/publishes vectors afterward.

Every source has a rowversion; vectors bind to that revision. Publishing rechecks
source visibility/revision, account/consent revision and durable lease token/expiry
under SQL locks. Retrieval ignores old/incompatible vectors and final disclosure
rechecks the original source. Deletion changes visibility only; retained derived
memories have independent permissions. Pending vectors do not prevent BM25 use.
Failed jobs retry independently, and bounded, idempotent backfill covers missing
vectors, model changes and re-enabled consent. Original records are never purged.

## Deployment and compatibility

Apply additive migration `0033_hybrid_retrieval_index.sql`; previous migrations
are immutable. It adds rowversions, the search view, enqueue triggers, an index
revision column, expanded polymorphic owner validation and an owner/kind/dedupe
job lookup index. Existing embedding codec and uniqueness are reused. Old
vector-bearing job payloads are retained with cancelled status; backfill rebuilds
current sources. Triggered scheduler DML uses OUTPUT INTO. No new HTTP endpoints,
services, dependencies or production environment variables are introduced.

Stop existing workers before upgrading the migration, runtime and knowledge
bundle together. The bundle config is now `retrieval.v2`; signed releases must be
rebuilt/re-signed. Existing model/dimension/consent/cipher/worker settings and the
runtime `per_strategy_ms` timeout name are retained. There is no destructive down
migration. Roll forward or stop workers and restore a compatible runtime/bundle
as an explicit operator rollback; preserve all original and derived records.

## Consequences

Memory commits and maintenance work during an embedding-provider outage. Exact
text deduplication remains synchronous; semantic deduplication can use vectors
once available. A provider crash may repeat an embedding call, while publication
is idempotent. A query embedding outage degrades retrieval instead of presenting
keyword-only results as successful hybrid search.

Exhaustive search/BM25 use application CPU and memory proportional to owner
history. Policy checks incur source round trips. Any source update, including
metadata-only updates, changes its rowversion and can schedule regeneration.
Large histories and oversized derived records will need measured optimization;
ANN/inverted indexing, chunking, morphology-aware analysis, query rewriting and a
specialized reranker are explicitly deferred. This change makes no clinical or
real-user privacy-readiness claim.
