# AI Core SQL Server persistence (WP-02)

`StateStore` is the AI-owned infrastructure participant in Conversation's UoW.
`composition.NewAIPersistence` supplies it to the Conversation `TurnStore`.
It never owns an inference transaction, starts a worker or calls a provider.

Inputs are an authenticated `Admission` and a validated `PreparedTurn`. Writes
include optimistic state projections, restricted delta events, durable jobs,
metadata-only trace/outbox, and a fenced expiring user lease. A failed statement,
version conflict or failed final transaction causes full rollback; the admitted
user message remains durable. Callers must reload/re-evaluate stale state.

The SQL schema uses composite owner keys, JSON/check constraints, filtered unique
indexes and ownership triggers for polymorphic references. Migration 0020 replaces
delete cascades with validated `NO ACTION` references and rejects physical DELETE.
Conversation soft deletion preserves all AI data and consent; it never purges,
invalidates or recursively hides derived records. No implicit AI-reuse policy or
permanent-erasure mechanism is introduced. New traces have no default expiry.
Highly-sensitive memory requires ciphertext; encryption/KMS is a separate gate.
See [ADR 0010](../../../../../../../docs/adr/0010-soft-delete-preservation.md).

`EncodeEmbedding`/`DecodeEmbedding` implement storage codec v1 only: little-endian
normalized finite float32 vectors, model/dimension/version checks, no retrieval.

See [ADR 0009](../../../../../../../docs/adr/0009-ai-core-sql-lifecycle.md) for
queue/replay/recovery, migration deployment, privacy scope and deferred work.
`tests/integration/ai_lifecycle_test.go` runs the WP-01 port harness and additional
SQL Server 2022 migration, transaction, concurrency and privacy tests. The package
unit test covers the vector codec; Conversation's unit test covers canonical
request identity and input bounds. No semantic/clinical FSM is implemented here.
