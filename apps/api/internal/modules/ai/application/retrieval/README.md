# Retrieval (WP-09)

Deterministic, read-only memory retrieval for the Theia AI Core. It answers one
question per turn: *which few prior memories or messages are relevant enough to
show the Brain?* It never writes, never summarizes, and never invents content.

Authority: `docs/ai-core/Theia_AI_Core_Contract_Pack_v1.2.md` §§4.4, 7.3, 8.5,
10 (BR-016/019/023, MC-011) and Architecture §3.4. Configuration comes from the
pinned Knowledge Bundle (`config/retrieval.yaml`, `config/budgets.yaml`), never
from environment variables or code defaults.

## Layout

| File | Responsibility |
| --- | --- |
| `policy.go` | `LoadConfig` (pinned, strict, fail-closed validation), `WeekStart` |
| `temporal.go` | `TemporalParser`: half-open local-time windows, `hasRecallVerb` |
| `lexical.go` | closed-list keyword/role/emotion extraction, `overlappingTerms` |
| `score.go` | `candidate`, weighted score, recency decay, cosine, rank/dedupe/budget |
| `store.go` | `Store`/`Embedder` ports, query/row structs, `ErrNotVisible` |
| `service.go` | `Service.Retrieve`, router, five strategies, degradation policy |

Adjacent packages: `adapters/mssql/retrieval.go` (SQL Server reads),
`adapters/embedding/embedding.go` (WP-04 client bridge).

## Strategies

Routing is deterministic and ordered by specificity, so an exact question is
never answered with a semantic near-match:

1. **temporal** — exact recall of raw messages in a local-time window. The
   window rows *are* the answer (Architecture §4.4 scenario 2). An empty window
   reports `exact_match_found=false` and no other strategy may substitute.
2. **thread** — memory linked to the focus thread (or the top active one).
3. **anaphora** — recent emotion/topic-tagged memory for "the same feeling".
4. **entity** — memory whose stored entity role matches a role the user named.
5. **hybrid** — lexically pre-filtered candidates, cosine-reranked in Go.

## Scoring

```text
score = 0.50·similarity + 0.20·recency + 0.15·importance + 0.15·thread_match
```

Components are normalized to `[0,1]` and compared against `defaults.min_score`.
A missing component contributes **zero**; weights are never renormalized, which
is what prevents over-retrieval when embeddings are absent. `similarity` is the
larger of a real stored-vector cosine and a structural match from an exact
lookup (entity role, resolved thread link, temporal hit). No similarity is ever
manufactured from text heuristics.

Recency decays with a 21-day half-life. Temporal hits bypass `min_score`
because their relevance is the window itself.

## Guarantees

- **Owner first.** Every read is scoped by the server-resolved `user_id`, then
  re-checked in Go. A user can never receive another user's row.
- **Soft delete only.** Adapters filter `deleted_at IS NULL` on the row and its
  parents. Nothing is removed; retrieval hides.
- **Fail closed.** An unreadable bundle is an error, not a default config.
- **Fatal vs degraded.** Ownership and validation errors abort the request.
  Infrastructure errors degrade it: partial items plus `meta.degraded=true`.
  If every strategy fails the service returns `ErrRetrievalDegraded` with no
  items — it never reports an empty result as a confident "nothing found".
- **Bounded.** Per-strategy and total timeouts, capped candidate pools, escaped
  `LIKE`, `TOP(@limit)`, and a hard item/token budget.

## Wiring

```go
retriever, err := composition.NewRetriever(composition.RetrievalOptions{
    DB:             db,
    Bundles:        wp03Registry,
    Embedding:      embeddingClient,           // optional
    EmbeddingModel: embedding.ModelID{Model: "…", Dimensions: 1536},
})
```

Inject the result as WP-05's `orchestrator.Dependencies.Retriever`. When no
retriever is injected the orchestrator already degrades memory, so wiring this
is additive.

## Known gaps

- `novelty_filter` (drop a memory already visible in the last 8 messages) is not
  applied: the assembler owns recent history and the service does not read it.
  The duplicate is bounded by dedupe but not by novelty.
- Anaphora person/alias resolution (`[OPEN]` in v1.2) is not attempted; names
  never become entity roles.
- Cosine reranking is linear over a bounded candidate pool. There is no ANN
  index, so this must not be presented as millisecond vector search at scale.
- No live SQL Server was available in the development sandbox; the integration
  suite compiles and skips without `WELLBEING_INTEGRATION=true`.
