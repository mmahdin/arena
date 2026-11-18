# WP-09 Retrieval (Contract Pack v1.2)

This package implements `ports.Retriever` without changing the WP-01–WP-08 contracts. It uses the current, pinned WP-03 `config/retrieval.yaml`, WP-04 `ports.EmbeddingClient`, WP-02 SQL Server memory/vector tables, and WP-08 focus-thread state. The SQL Server adapter is `adapters/mssql.RetrievalRepository`; `composition.NewAIRetrieval` provides explicit dependency assembly. Orchestrator passes the admitted trigger ID, focus thread, excluded current-context IDs, and the pinned knowledge bundle.

## Mandatory integration dependency

**Provide an approved `UsePolicy` implementation before enabling this service.** WP-02's storage visibility, `memory_paused`, evidence links and soft-delete state are not, by themselves, grants to use personal data for AI processing. `New` rejects a missing policy. `CanSearch` runs before any data read; `CanUse` checks each returned candidate and denies the entire request if authorization is uncertain. No permissive default or inferred consent is supplied. Inject the resulting `ports.Retriever` into the Orchestrator's dependencies. An approved production policy and its source of truth must be supplied by the application owner.

The SQL adapter only reads existing tables (`ai_user_state`, `ai_memory_items`, `ai_memory_evidence`, `ai_embeddings`, `messages`, `conversations`) and uses the existing WP-02 embedding codec. It performs bounded owner/trigger-scoped reads and checks parent/message visibility. It never updates or deletes records; it does not require a migration, Postgres, pgvector or a new service. Encrypted `content_enc` values cannot be read without an approved decryptor and separate authorization, so this version intentionally excludes them.

## Behavior and limits

Temporal lookup uses calendar intervals in the user's persisted timezone and checks raw, prior, user-role messages. Missing exact evidence is reported as missing, never substituted with semantically similar content. Thread, entity, anaphora and hybrid routes use bounded candidate pools, WP-04 embeddings where available, lexical relevance and thread/recency/importance weights from the bundle; vectors with missing or incompatible dimensions/models are ignored. Sensitive and stale memory, unsupported hypotheses, missing provenance, duplicate clusters and already-included messages are filtered. Per-item and total retrieval budgets are enforced. Any authorization or storage failure returns an empty, degraded result.

The architecture's AI-use authorization model remains an explicit unresolved production dependency, not something this patch can define without a product decision. In addition, SQL Server 2022 has no vector index in the provided schema: hybrid ranking runs over a bounded owner-scoped pool rather than promising exhaustive approximate nearest-neighbor search. Novelty checks may read recent historical text under `CanSearch`, so the approved policy must cover this use of recent context as well.

## Tests

From `apps/api`:

```
go test ./internal/modules/ai/application/retrieval ./internal/modules/ai/adapters/mssql ./internal/modules/ai/application/orchestrator
```

Optional SQL Server 2022 integration fixture (requires all migrations, a **disposable database**, and explicit opt-in):

```
THEIA_WP09_SQLSERVER_DSN='sqlserver://...' THEIA_WP09_SQLSERVER_ALLOW_FIXTURES=1 \
  go test ./internal/modules/ai/adapters/mssql -run '^TestWP09SQLServerIntegration$' -v
```

The integration fixture creates only synthetic records inside a transaction and rolls back the transaction, without executing any SQL `DELETE`. When the DSN and opt-in are absent, the test skips.
