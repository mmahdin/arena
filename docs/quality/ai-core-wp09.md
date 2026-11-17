# WP-09 — Retrieval (integration and verification)

Scope: WP-09 only. Reference: `docs/ai-core/Theia_AI_Core_Contract_Pack_v1.2.md`
§§4.4, 4.5, 6.4, 7.2, 7.3, 7.4, 8.5, 10 (BR-016/019/023, MC-011), 12.1, 13.1–13.2,
and the Final Architecture Document §3.4 and §4.4. WP-01 contracts, WP-02
persistence, WP-03 bundles, WP-04 embeddings and WP-05 orchestrator are reused
unchanged. This patch adds no second message store, no clinical logic, no model
call outside the existing WP-04 client, and no hard delete.

## Owned packages and authority

- `modules/ai/application/retrieval`: the retrieval service and its policy,
  temporal parser, lexical extraction, scoring/ranking and store ports.
- `modules/ai/adapters/mssql/retrieval.go`: read-only projection of WP-02
  `messages`, `conversations`, `turns`, `ai_threads`, `ai_memory_items`,
  `ai_embeddings` and `ai_user_state`. It contains no `INSERT`/`UPDATE`/`DELETE`.
- `modules/ai/adapters/embedding/embedding.go`: bridges the existing WP-04
  `ports.EmbeddingClient` to the retrieval `Embedder` port. It adds no provider
  SDK and no retry loop of its own.
- `internal/app/composition/ai_retrieval.go`: explicit construction helper; not
  called from the running API path in this patch.

### Invariants

- **Owner-first reads.** Every SQL statement filters `user_id = @p1` first and
  joins parents on the same owner. The service re-validates ownership in Go, so
  a misbehaving adapter still cannot leak another user's memory.
- **Soft delete is preserved.** `deleted_at IS NULL` is applied to the memory
  row and to its parent conversation/turn. Retrieval hides rows; it never purges
  or rewrites them.
- **Exact recall is not semantic.** A temporal request returns verbatim messages
  from the local-time window. An empty window sets `exact_match_found=false` and
  is *not* filled by a similar memory (Architecture §4.4 scenario 2).
- **No fabricated similarity.** Similarity is used only when a stored vector
  matched the configured model id, codec version and dimensions. Otherwise the
  component contributes zero and the score falls honestly below `min_score`.
- **Deterministic routing.** Temporal → thread → anaphora → entity → hybrid,
  deduplicated; a caller-supplied strategy list is honoured when valid.
- **Fatal vs degraded.** `ErrForbidden`/`ErrValidation` abort the request.
  Adapter/embedding failures degrade it with `meta.degraded=true` and partial
  items. Total failure returns `ErrRetrievalDegraded` with no items, so an outage
  is never presented as "no memories exist".
- **Bounded work.** Per-strategy and total timeouts from the bundle, capped
  candidate pools, escaped `LIKE` with 3–40 character terms, `TOP(@limit)`, a
  per-item token cap and a total token budget.

## Measured verification (this sandbox)

| Package | Assertions passing | Failures |
| --- | --- | --- |
| `application/retrieval` | 96 | 0 |
| `adapters/embedding` | 16 | 0 |
| `adapters/mssql` (pure-Go query/guard tests) | 26 | 0 |
| `app/composition` | 8 | 0 |
| `tests/integration` (WP-09 SQL Server) | 6 functions | skipped without a database |

77 WP-09 test functions in total. `go vet` is clean on all five packages and the
suite passes under `-race -count=3`.

The repository's full suite still has one **pre-existing** failure,
`TestFailedUserOnlyReplayAndExplicitRetry` in `application/orchestrator`. It
reproduces on the base commit with WP-09 reverted and is unrelated to this work.

## What was deliberately not built

1. **Novelty filter.** `novelty_filter.skip_if_in_recent_messages` requires the
   recent-history set, which the assembler owns. Adding a second read of recent
   messages here would duplicate WP-08's authority. Deduplication by cluster and
   text still bounds repeats.
2. **Business Time / person aliases.** Anaphora resolution to a named person is
   `[OPEN]` in v1.2. Capitalized names are deliberately never promoted to entity
   roles, so the service cannot invent a relationship the user never stored.
3. **Inactive strategies.** Memory *corrections*, supersession graphs and
   cross-thread summarization belong to later work packages; the reader only
   consumes what WP-02 already persists.
4. **Vector index.** There is no ANN index by design (no pgvector, no external
   service). Cosine reranking is linear over a bounded pool and must not be
   marketed as millisecond ANN search at clinic scale.
5. **Runtime activation.** `composition.NewRetriever` is available but not wired
   into the live gateway in this patch. WP-05's `prepare` does not yet forward
   admission time, recent text or a turn-wide bundle version, so enabling it
   today would change turn-level behavior that belongs to a coordinated WP-05
   change. Until then the orchestrator's existing "no retriever ⇒ degraded
   memory" path remains authoritative.

## Explicit gaps and blockers

- **No live SQL Server** was available in this sandbox. All SQL statements are
  reviewed and unit-tested for construction, parameterization and guards, but
  have **not** been executed against SQL Server 2022 here. The six integration
  tests require `WELLBEING_INTEGRATION=true`, `DATABASE_URL` and
  `APP_ENV=local` and are skipped by default. Do not claim database verification
  until they run in CI.
- **Synthetic bundle only.** Tests use inline `retrieval.v1`/`budgets.v1`
  fixtures that mirror the pinned bundle shape. No clinic-approved bundle was
  provided, so threshold tuning (especially `min_score`) is unvalidated against
  real recall quality.
- **Embedding model identity** must be configured to match the vectors already
  stored by WP-04. A mismatch is ignored rather than mis-scored, which means a
  misconfiguration silently reduces recall quality instead of failing loudly.
  Deployment must assert the model id and dimensions at startup.
- **User time zone** comes from `ai_user_state.tz`. Users without a state row
  are treated as UTC, so "yesterday" can be off by one day for a user who has
  not completed onboarding. This is a conservative default, not a correctness
  proof.

## Verification commands

```sh
cd apps/api
go vet ./internal/modules/ai/application/retrieval ./internal/modules/ai/adapters/mssql \
       ./internal/modules/ai/adapters/embedding ./internal/app/composition ./tests/integration
go test ./internal/modules/ai/application/retrieval ./internal/modules/ai/adapters/mssql \
       ./internal/modules/ai/adapters/embedding ./internal/app/composition
go test -race ./internal/modules/ai/application/retrieval
# With a disposable SQL Server 2022:
WELLBEING_INTEGRATION=true APP_ENV=local DATABASE_URL='sqlserver://…' \
  go test ./tests/integration -run TestRetrieval -v
```
