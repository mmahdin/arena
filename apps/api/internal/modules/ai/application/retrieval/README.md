# AI Core — WP-09 Retrieval

Implements the deterministic **Retrieval Service** from the Final Architecture §3.4
and Contract Pack v1.2 §4.4 / §7.3. It is the *only* component that reads memory;
the Turn Brain (WP-10) receives a pre-assembled `[]MemoryView` and never queries
storage itself.

No LLM is used. The only model call is the embedding tier (WP-04), and its failure
degrades semantic ranking to lexical similarity instead of failing the turn.

## Layout

| Path | Contents |
| --- | --- |
| `contract/retrieval.go` | §4.3/§4.4 subset (`RetrievalRequest`, `RetrievalResult`, `MemoryView`, `RetrievalMeta`, `NeedsContext`, `TurnView`, `TemporalRef`) + retrieval read models. Verb-namespaced so the WP-01 superset can absorb it unchanged. |
| `domain/retrieval/` | Pure rules: `config.go` (tunables/validation), `rank.go` (scoring, dedupe, budget, novelty), `temporal.go` (time-window parser, recall verbs), `strategy.go` (strategy selection, anaphora/entity detection), `similarity.go` (cosine, lexical fallback). No SQL, no clock reads, no I/O. |
| `application/retrieval/` | `Service.Retrieve` — the §6.1 `Retriever` port implementation: strategy execution over consumer-owned ports, embedding integration, exclusion rules, result projection and `RetrievalMeta`. |
| `adapters/mssql/retrieval_repository.go` | SQL Server read model (§7.3): `RecentMemory`, `Preferences`, `MemoryByThread`, `MemoryByEntity`, `MemoryByIDs`, `MessageWindow`, `RecentMessages`, `Embeddings`, plus the float32 vector codec. |
| `adapters/embed/` | Embedding integration: `Deterministic` (offline, test/synthetic only) and `OpenAICompatible` (`/embeddings`, no SDK). |

## Retrieval strategies

| Strategy | Trigger | Method |
| --- | --- | --- |
| `temporal_exact` | Temporal expression **and** recall verb (or a resolved `TemporalRef` + forced strategy) | Owner-scoped `dbo.messages` window; returns **verbatim** messages, never summaries; sets `retrieval_meta.exact_match_found`. An empty window reports `false` — no semantic substitution. |
| `thread` | Focus thread present and the turn continues it, or explicitly requested | `ai_memory_items` linked through `OPENJSON(thread_ids_json)`. |
| `entity` | Kinship/relational mention ("my sister Lena") or second-hop `kind=entity` | `OPENJSON(entities_json)` on role/label. Alias mapping is **not** implemented (open decision §7.3). |
| `anaphora` | "same feeling as before", "یادت هست", … | Recent (14-day) items sharing a tag, thread or content term; adds the emotion/topic boost. With no thread and no theme the result sets `ambiguous_reference=true` so the Brain clarifies instead of guessing. |
| `hybrid` | Default | Bounded owner-scoped candidate pool + always-on preferences; cosine rerank in Go over `ai_embeddings`, lexical fallback for items with no (or incompatible) vector. |

A second hop (`NeedsContext`) runs **one** extra targeted strategy and the total
item cap rises from 5 to at most 7 (§4.3 `hard_cap_with_hop`).

## Ranking, dedupe and budget

```
score = 0.50·similarity + 0.20·recency_decay(21d half-life)
      + 0.15·importance + 0.15·thread_match   (+ anaphora emotion/topic boost)
```

* floor `min_score = 0.32`, top-K 5 (7 with a hop), 60 tokens per item, 400 tokens total;
* deterministic tie-break: score → kind priority (episode, fact, preference, goal,
  event, hypothesis, safety_event) → id, so an inferred item never displaces the
  user's own words at equal score;
* dedupe by `cluster_id` and by normalized text / near-duplicate overlap;
* novelty filter drops items whose text is already in the recent window;
* token estimation is a conservative character approximation (no tokenizer dependency).

## Access, provenance and deletion rules (enforced in code)

1. **Owner isolation** — every port method takes the server-resolved `userID`; the
   SQL filters on it, the adapter drops rows whose `user_id` differs, and the service
   re-checks. Three independent layers; a single mistake cannot leak data.
2. **Soft delete only** — nothing in this package deletes a row. `ai_memory_items.deleted_at`
   and `ai_message_suppressions` mark removals; retrieval excludes flagged rows and
   never rewrites/removes them.
3. **Provenance** — items with `evidenceN < 1` are never retrieved (§2.5 #3).
4. **Status** — `rejected` and `superseded` items are excluded; `safety_event` items are
   reserved for the safety pipeline.
5. **Hypotheses** — only `proposed`/`supported`, and only when stored as `inferred`
   (§7.5). A hypothesis stored as `reported` is treated as invalid and dropped.
6. **Sensitivity** — `highly_sensitive` items are excluded from default retrieval;
   the caller may widen this explicitly through `RetrievalRequest.IncludeSensitivity`
   (WP-08 owns the policy decision; WP-09 only refuses to widen it silently).
7. **Degradation** — any store/timeout failure yields zero items with
   `meta.degraded=true` (FB-002/BR-023), never a partial set presented as complete.

## Configuration

`domain.Defaults()` mirrors `config/retrieval.yaml` (§8.5) exactly and
`Tunables.Normalize()` rejects unsafe combinations at construction (weights not
summing to 1, `per_strategy > total` timeout, unbounded candidate pool, hypothesis
statuses other than proposed/supported, …).

Environment (see `internal/platform/config`):

| Variable | Default | Meaning |
| --- | --- | --- |
| `EMBEDDING_PROVIDER` | `deterministic` | `deterministic` (offline) or `openai-compatible` |
| `EMBEDDING_MODEL` | `deterministic-test-embed` | Model id stored alongside every vector |
| `EMBEDDING_DIMENSIONS` | `128` | 8..8192; a network model's real dimension must be set here (OPEN-4) |

The default is offline on purpose: the repository has no approved embedding model or
dimension yet.

## Tests

| Package | What is covered |
| --- | --- |
| `domain/retrieval` | Formula and weights, recency decay/half-life, anaphora boost window, ordering/tie-breaks, cluster+text dedupe, top-K and token budget, truncation, novelty filter, temporal parser (en/fa, weeks, weekdays, explicit dates, unresolved input), recall verbs, anaphora/entity detection, cosine error cases, config validation. |
| `application/retrieval` | Ranking end to end, embeddings vs lexical fallback, dimension/model mismatch, missing/corrupt embeddings, exclusion matrix (soft-deleted, superseded, rejected, no evidence, highly sensitive, hypothesis statuses), cross-user isolation (store + service level), temporal verbatim recall, empty-window `exact_match_found=false`, thread/entity/anaphora strategies, second hop and hard cap, per-strategy failure degradation, cancellation, dedupe, token budget, score floor, meta reporting. |
| `adapters/mssql` | Vector codec (length/dim/NaN), JSON column tolerance, limits, predicates; **SQL Server integration** (`WELLBEING_INTEGRATION=true`) for owner scoping on every method, soft-delete retention. |
| `adapters/embed` | Deterministic stability/normalization/limits, provider request/response shape, misordered indices, malformed payloads, error sanitization. |
| `tests/integration` | Full SQL Server scenario: migration up + idempotent re-apply, two synthetic users, every exclusion rule, cluster dedupe, exact/temporal recall, suppressions, corrupt embeddings, preferences core set and novelty window. |

## Known gaps (not silently implemented)

* Person alias resolution ("my sister" → "Lena") — open product decision (§7.3).
* SQL Server Full-Text Search is **not** used. `ai_embeddings` has no ANN index, so
  hybrid recall is bounded-pool + in-Go cosine, exactly as §7.3 permits; latency at
  scale is untested and must be benchmarked before any claim (OPEN-4).
* Persian temporal coverage is pattern-based and tested for the listed forms only;
  it is not a natural-language date parser.
* `embedded`/provider embeddings are untested against a live provider (no approved endpoint).
* Retrieval is not yet invoked by an orchestrator: WP-05/WP-10 own `ProcessTurn`.
