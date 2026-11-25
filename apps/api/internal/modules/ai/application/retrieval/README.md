> **2026-11-23 active runtime:** See `docs/ai-core/unified-memory-centric.md`. Production uses one unified conversational model with immediate retrieval and independent baseline context. Brain/Generator and classifier runtime descriptions below are historical; Memory Writer/Reflection and persistence remain active.

# Semantic memory retrieval (BM25 temporarily disabled)

One pipeline implements `ports.Retriever`:

```
current user message → query embedding
complete authorized corpus → current/contextual cosine ranks → RRF → deduplicate → context budget
```

The initial query is exactly `Turn.Text`; a distinct contextual query includes
bounded earlier dialogue. Both use the same Shared Embedding Service as indexing
workers. No query LLM, embedding call or new service is added by disabling BM25.
Legacy strategy hints remain accepted as inert compatibility fields.

Current-query and contextual-query BM25 are temporarily disconnected from the
active service. They do not run, add RRF votes, provide a failure fallback, or emit
rank artifacts. There is no new feature flag or runtime setting. The isolated
algorithm, language normalization, numerical tests and SQL lexical index remain
available for a later explicit code change.

`Repository.Scan` still visits the complete visible, earlier corpus for the verified
owner. SQL has no `TOP`, memory-kind restriction, importance pool or lexical
prefilter. Active scans request no query term frequencies. Eligible vectors must
match model, dimensions and current source revision. Current/contextual cosine
rankings and the existing independent focus-thread/temporal metadata votes feed
unchanged equal-vote RRF (`sum(1/(60+rank))`), without a per-branch cutoff. Final
selection follows RRF rank order, preserving exact source-identity deduplication,
configured context budgets and authorized direct evidence expansion. Text overlap
and shared source types do not penalize or discard distinct memories.

A failed, invalid or incompatible query embedding contributes no semantic ranks
and is recorded in per-query/top-level failed strategies. A valid surviving query
continues to work within the original total deadline. If every query fails, the
result is degraded: only an independently valid thread/date signal can supply
candidates; without one, the result is empty. Pending/missing/stale/invalid source
vectors cannot create semantic matches from lexical text. Existing metadata or
direct evidence expansion can still supply a permitted source under their own
rules. A successful query with no eligible source matches returns an empty result;
this is distinct from a failed query embedding.

Inspector emits only actually available `vector_ranks` /
`contextual_vector_ranks` plus merged RRF/selection artifacts. Rank previews stay
bounded to five references without truncating rankings. Historical v1 files with
BM25 artifacts remain readable by the offline Explorer; no schema change is needed.

## Sources and permission

Migration `0033_hybrid_retrieval_index.sql` provides `dbo.ai_search_sources`:

| Source | Searchable content | Visibility / epistemics |
| --- | --- | --- |
| `ai_memory_items` | All memory kinds, including safety events; encrypted content through the approved cipher | Active/proposed/supported, visible, evidenced; original epistemics and sensitivity retained |
| `messages` | Original user **and assistant** text across owned conversations | Visible message and parent; assistant prose stays `system_observed` |
| `ai_sessions` | Nonempty rolling summaries | Visible owned conversation; `system_observed` |
| `ai_threads` | Topic, goal and structured slots | Owner scoped; `system_observed` |
| `ai_formulations` | Paragraph and fields | Latest visible version per thread; `inferred` |
| `ai_protocol_instances` | Protocol identity, captured inputs and summary | Owner scoped; `system_observed` |
| `ai_open_loops` | Open loop text | Open, owner scoped; `system_observed` |

`CanSearch` precedes reading; `CanUse` precedes protected decryption and ranking.
A missing/failed policy fails closed. Explicit denial skips that source.
`ProductUsePolicy` checks active-account ownership and visibility without reading
consent or honoring legacy memory pause. This applies to raw messages and derived
state alike. Highly sensitive items still require an explicit sensitivity request;
normal generation retains its stricter sensitivity boundary. Deleted conversations
hide raw/session sources while independently retained memories keep their own
visibility. Hidden retained evidence still requires its separate explicit permission.

Source kind, UUID, rowversion, original creation time, conversation ID where
applicable, validity date, epistemics, sensitivity and evidence count survive into
context and Inspector. Evidence links and historical versions remain in their
original tables. Brain/Generator recheck source content, metadata, revision,
chronology, visibility and authorization before disclosure. No inferred summary
or assistant response becomes a reported user claim.

The source audit also considered `ai_reflection_runs` (encrypted processing
receipts whose accepted outputs live in the indexed canonical tables),
`ai_follow_ups` (scheduling codes/references), events, jobs, traces, outbox,
tombstones and cooldowns (operational/control metadata). These are not independent
narrative memories. Bookmarks reference already indexed messages. Current profile/
onboarding preferences already enter context directly. Legacy Daily Reflections
and personalized letters are separate product stores with no approved AI reuse
mapping and are not silently added to this index.

## Retained BM25 and language infrastructure

The tested SQL Server 2022 image reports `IsFullTextInstalled=0`. Microsoft's
[ranking documentation](https://learn.microsoft.com/en-us/sql/relational-databases/search/limit-search-results-with-rank?view=sql-server-ver16)
distinguishes BM25-based `FREETEXTTABLE` from other full-text rankers. The retained,
currently inactive Go implementation uses actual Okapi BM25: term frequency saturation (`k1=1.2`), document
length normalization (`b=.75`), and positive Robertson IDF over the authorized
corpus. No full-text installation, language pack, external search service, or new
dependency is required. The numerical formula has a direct regression test.

The retained keyword algorithm and indexing utilities use NFKC normalization, Persian/Arabic yeh/kaf
folding, digit folding, diacritic removal, joiner token boundaries and ordinary
English/Persian lexical stop words. Negations are retained; there is no stemming,
translation, handwritten temporal interpretation, or phrase intent detection.
Semantic cross-language quality depends on the configured multilingual embedding
model. The embedding query itself remains unchanged.

## Index lifecycle and configuration

See [Memory](../memory/README.md) and [ADR 0014](../../../../../../../docs/adr/0014-hybrid-retrieval-index.md).
Memory commits never generate vectors. Source triggers atomically enqueue
reference-only `memory_embed` jobs for every insert/update. The existing supervised
Memory Worker reads current, permitted text, calls the embedding provider outside
transactions, and conditionally upserts `ai_embeddings`. Only matching source
revisions/models/dimensions are semantically searchable. Lexical publication and
all existing rows/migrations remain intact, but unindexed text is no longer a
keyword fallback. A query-embedding failure is explicitly degraded and never
activates BM25.

`knowledge/config/retrieval.yaml` is now `retrieval.v2`: context limits only.
Existing runtime `retrievalTimeouts.per_strategy_ms` is retained as the query
embedding timeout for deployment compatibility. Model, dimensions, consent,
encryption and worker configuration reuse the existing deployment settings.
Released bundles must be rebuilt/re-signed with V2 config. No HTTP API changes.

## Verification and limits

Run `go test ./internal/modules/ai/application/retrieval` for formula, fusion,
source/type coverage, isolation, failures, budgets and a 2,001-record recall test.
`THEIA_LIVE_EMBEDDINGS=1` opts into the synthetic Persian/bilingual provider
benchmark using `MODEL_API_BASE_URL` / `MODEL_API_KEY`. It prints per-query ranks,
Recall@1/3/5 and MRR, and compares selected semantic results with raw cosine rankings and an isolated
BM25 reference (not a production retrieval branch).
`THEIA_RELEVANCE_CACHE` optionally reuses synthetic benchmark vectors locally.
It never reads application chats.

Live SQL coverage is `WELLBEING_INTEGRATION=true APP_ENV=local go test
./tests/integration -run 'TestHybridIndexSQL|TestWP12SQL|TestWP10SQL|TestAICore'` with
a local `DATABASE_URL`; tests create disposable databases. The test requires SQL
Server 2022 and never migrates the configured application database.

The active service exhaustively scans an owner's corpus and computes cosine ranks. CPU,
memory and policy round trips grow with history; deadlines degrade the complete
request instead of silently truncating candidates. ANN/inverted indexing, chunking
very large derived records, morphology-aware analyzers, query rewriting and a
specialized reranker remain future work. The small synthetic benchmark is an
engineering check, not a broad retrieval-quality or clinical validation claim.

## OBS-04B — Detailed synthetic OpenTelemetry tracing

With the private local `THEIA_OTEL_CAPTURE_CONTENT=true` gate, OTel records
deterministic queries, the authorized candidate scan, active vector/thread/date
rankings, RRF, selection decisions and the returned context after evidence
expansion. BM25 stays disabled. The offline Inspector keeps its Top-5 previews
and existing artifacts. Later JEV and Unified stages record what actually reaches
the model. Oversized events include an omission reason.
See [OBS-04B](../../../../../../../docs/observability/OBS04B_DEEP_RAG_TRACING.md).
