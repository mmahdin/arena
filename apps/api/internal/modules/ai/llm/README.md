# WP-04 — Provider-neutral LLM Client Layer

Authority: *Final Architecture Document*, §§6.1–6.2 and 9.7, and *Contract Pack v1.2*, §§6.3, 8.5, 12.3 and 13.1. This package is the **new** model-client implementation for the Go modular monolith. It does not call, wrap, or fall back to `ai/application.Service`, `ai/adapters/openai.Gateway`, `ai/adapters/mock`, `ai/policy` or the old conversation turn workflow. No HTTP turn endpoint is changed or published here: the in-process Orchestrator connection belongs to WP-05.

## Consumer contracts and provider adapters

- `ports.Client.Structured` accepts the pinned, post-injection JSON Schema, schema name, system blocks, chat messages, model component, temperature, limits, metadata and idempotency key. Every result (including provider-native JSON-schema responses) is checked with `contract/schema.CompileInline`: draft 2020-12, strict JSON with no duplicate keys/trailing data, asserted formats, and no external resource access. The client charges reported usage even when the output fails schema validation.
- Native JSON Schema is selected by model capability. Non-native providers receive the full, pinned JSON Schema as a system instruction in JSON-object mode; on a schema mismatch, exactly **one** JSON repair on the **same model** may follow, without new reasoning or interpretation. An invalid second result returns `ErrInvalid`, leaving Brain's conservative plan fallback to WP-10. Invalid native-schema output fails without a repair. No other provider is tried for a semantic/schema failure.
- `ports.Client.Stream` is an internal consumer stream, **not** SSE to HTTP. It retries transport/5xx/429 errors or hops to the next model only before the first emitted token. After a token, any failure is terminal; later output validation and template fallback are owned by WP-07/WP-11. The first-token watchdog has a separate per-attempt context, while the overall call obeys its deadline and any tighter `contract.Budget.TotalDeadline`. Clean EOF requires a terminal finish signal and usage; missing/incomplete upstream output is rejected. `Close` cancels blocked reads and closes the provider body.
- `ports.EmbeddingClient` is an additive companion interface to the **unchanged** v1.2 `ports.Client`. It uses only the embedding tier and validates the configured dimension and finite float32 values; the model ID accompanies returned vectors. Vector normalization/storage/indexing and matching embedding models on retrieval are WP-09, not WP-04.
- `adapters/llmhttp` is a separate HTTPS, redirect-disabled OpenAI Chat Completions/Embeddings-compatible transport with native schema/JSON mode, bounded JSON and SSE bodies, usage parsing and sanitized status errors. It does not use the legacy OpenAI adapter or a vendor SDK. `Block.Cache` is advisory: the compatible API supports automatic provider prompt caching where available, but this adapter does not assert explicit cache-control support.
- `adapters/fake` is an **independent** scriptable provider double for Structured, Stream and Embed. Its call history contains model/operation/mode, never prompts, response bodies, vectors, secrets or user IDs.

## Model configuration, budgets and observation

`llm.New(config, providers, options)` receives a validated `Config`. The supported runtime file `config/ai-runtime.yaml` uses **component keys** (`safety`, `brain`, `generator`, `validator`, `memory_writer`, `reflection`, `embedding`), not shared `small`/`strong`/`classifier` tiers. Each component defines an ordered `chain` of provider/model choices plus timeout, temperature and output ceiling. The first entry is the primary model; further entries are explicit fallbacks. Embeddings have a single model/dimension to preserve vector compatibility. Missing or unexpected components fail closed.

Operational deadlines, retries, circuit breaker and quotas are defined in the same runtime file, independent of the signed Psychology Knowledge Bundle. Historical tier-only `llm.Config` values remain supported for isolated client tests, but cannot be loaded by `ai_runtime.v2`. No model identity is pinned to a Knowledge artifact.


Requests honor stage timeout, a configurable hard maximum, caller cancellation and optional `Budget.TotalDeadline`. All provider invocations (including retry/repair) are checked against the configured per-user daily quota and per-request `MaxLLMCalls` when present; the Orchestrator must still enforce **cross-stage, turn-wide** call counts in WP-05. Jittered retry/backoff honors `Retry-After` and declines to sleep beyond the overall deadline. A per-provider/model rolling-window breaker opens on transient failures; half-open allows a single probe. Permanently invalid requests are not retried.

A `UsageLedger` charges the tokens, cache hits, micro-USD costs and latency of **every successful provider response**, including abandoned repair responses or a usage-bearing partial stream. Each model's prices must be explicitly supplied and kept current: **zero price means configured as zero** and is not proof of free external usage. Costs are rounded conservatively at micro-USD precision. `MemoryLedger` is thread-safe, keeps UTC per-user/day aggregates without deleting historical counters, and supplies development/test quotas. It is **not a durable multi-instance quota mechanism**, and check-then-charge is not a hard pre-reservation against concurrent in-flight overspend. WP-05 must wire a durable usage/quota ledger for production enforcement if required. Upstreams that fail without usage cannot yield exact billable costs; this limitation must be monitored. `Metrics.Observe` receives only safe metadata, latency/failure/fallback tags and per-attempt charged usage; **no** prompts, schemas, tokens-as-text or user identifiers.

`IdempotencyKey` remains in the provider-neutral request; the generic compatible transport intentionally does not transmit a nonstandard HTTP header that a target provider might not honor. Cross-turn idempotency is enforced by the WP-02 persistence layer, and external request-level deduplication is provider-specific and **not guaranteed** by WP-04.

## Failure ownership / limits

`FailureTransport`, `FailureServer`, and `FailureRateLimit` are retryable. Permanent HTTP statuses, invalid provider responses and schema failure are not transparently retried. A terminal stream error after tokens is not a validated assistant reply; the future Generator/Validator must discard buffered unsafe output. There are no static psychological fallbacks, clinical protocols, new memory writes, database mutations, purges or retention logic in this work package. Data and traces already stored under WP-02's soft-delete policy remain untouched; user-triggered deletion cannot hard-delete source or derived data.

## Tests

```sh
cd apps/api
go test ./internal/modules/ai/llm/... ./internal/modules/ai/adapters/llmhttp/... ./internal/modules/ai/adapters/fake/...
go test -race ./internal/modules/ai/llm/... ./internal/modules/ai/adapters/llmhttp/... ./internal/modules/ai/adapters/fake/...
```

The scripted tests cover schema rejection and bounded repair, native schema, usage prices and daily quotas, retry and non-retry conditions, circuit breaker/half-open recovery, per-call deadline/cancellation, first-token timeout, pre-token fallback, no retry after the first token, incomplete stream handling, embedding vectors and metadata-safe events. The HTTPS `httptest` adapter suite verifies request shapes, SSE usage/finish and truncated-stream failures, embeddings, error classification, no redirects and unsafe HTTP configurations.

## BO-04 durable provider attempts

`Options.Attempts` optionally wraps each underlying provider with an
`AttemptJournal`. Production AI runtime wires it to SQL Server so every
physical provider request is durably started before dispatch and finished at
most once, including retries, repair, embedding and streaming. This is **not**
a billing-authoritative ledger or distributed quota system. Unknown usage
and missing configured prices remain explicitly unknown/unpriced. Local
synthetic tests without `Options.Attempts` retain the existing fake ledger.
See `docs/operations/ai-telemetry-bo04.md`.

OBS-04C: existing shared provider boundaries emit opt-in private synthetic
OTel input/output events even with the file-based Inspector disabled. No
additional LLM requests are made for tracing. See the OBS-04C runbook.
