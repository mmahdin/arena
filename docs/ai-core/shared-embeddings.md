# Shared embedding computations

`application/embedding.Service` implements the existing `ports.EmbeddingClient`.
`NewAIInternal` constructs one instance and injects that same pointer into hybrid
Retrieval and the Memory Service used by its indexing worker. Background Memory
Intelligence and Reflection retrieval use that same Retrieval instance. All seven
search source kinds therefore share computations without moving indexing or
changing query construction, ranking, memory inference, or source authorization.

The service wraps the configured component client. The existing client still
owns provider adapters, retries/fallback, quotas, circuit breakers, attempts,
usage and costs. Generative clients and prompts are unchanged. No SQL migration,
persistent cache, batch embedding, Redis, or additional model call is introduced.

## Identity and bounds

The key is owner UUID, primary provider/model/dimensions, SHA-256 of the **exact**
provider input, and a digest of the ordered compatible model chain and profile
revision. Providers/endpoints are immutable for the process lifetime. No Turn,
source, session, conversation, job, bundle or consumer identity enters the key.
There is no extra normalization. Runtime resolves its actual component settings;
substituted clients must supply an explicit profile in tests.

Only nonzero finite vectors of the resolved model/dimension are published.
Fallback results can satisfy current waiters but are never cached under the
primary identity. Unidentifiable results from ambiguous chains fail validation
or remain uncached. Cached vectors and delivered responses are copied. Errors,
cancellations, invalid vectors and invalidated executions are not cached.

```yaml
embeddingCache:
  ttl_seconds: 900
  max_entries: 4096
  max_in_flight: 64
```

These optional `ai_runtime.v4` settings receive the same defaults when omitted.
Values are validated and unknown fields rejected. There is one mutex for cache
lookup, registration, publication and invalidation. Expiration is lazy, reads
update LRU order, and insertion evicts expired entries before LRU eviction.
New distinct keys at the computation limit return a bounded capacity error;
cache hits and existing joins remain available. There is no internal wait queue,
periodic cache goroutine or unbounded per-user generation map.

## Independent execution and privacy

Each caller retains its original context and request timeout. A miss starts one
execution with the resolved embedding component timeout, preserving the initiating
request's metadata and trace scope while removing its cancellation/deadline. The
absolute `Budget.TotalDeadline` is rebound to the execution deadline; its other
limits and the existing client checks remain. A caller can stop waiting without
cancelling other consumers. Even if all callers leave, a successful computation
can populate the cache until its bounded execution finishes.

Application shutdown cancels the shared lifecycle and joins service-owned
executions before the host closes SQL. Providers must honor the existing context
contract. A provider panic is converted to a failed computation so an independent
execution goroutine cannot terminate the process or leave a stuck flight.

Successful account hiding and controls that revoke a consent flag (or send the
legacy pause request) invalidate only that user's cache. This does not restore
legacy consent/pause gates or change memory access policy. Invalidation removes
entries and cancels/marks matching in-flight computations; they cannot repopulate
or deliver invalidated vectors. Durable data and soft-delete behavior are retained.
Authorization and source/version checks still run before embedding and before
SQL publication, including on a hit. Conversation-sharing preferences are unrelated
to AI memory permission and do not silently change it.

## Inspector and accounting

Each consumer has an `embedding.Embed` wait node. Events distinguish
`embedding.cache_hit`, `embedding.cache_miss`, `embedding.inflight_join`,
`embedding.compute`, `embedding.wait_timeout`, `embedding.compute_failed`, and
`embedding.cache_store`. Additive optional `node.embedding` metadata contains a
computation UUID, requested vector identity, hashed input, consumer, outcome,
wait/execution durations and source computation attempt count. Origin trace,
Turn and node IDs correlate reuse across files without merging Turn graphs.

The compute span registers before the waiter can leave and holds the original
Turn inspection until provider completion. The physical provider subtree appears
once, under that computation, even after the response finishes. Same-Turn jobs
join that document through existing Turn/job correlation. Another Turn records
only reuse metadata. Reused responses report zero usage/cost; the initiating
successful miss can return its original usage, while accounting remains solely
at the actual configured client/provider boundary. A source attempt count on a
reuse node is historical correlation, never a new charge.

The offline Explorer displays shared events and origin links separately from
physical provider calls. It does not fetch another trace or connect to a backend.
Generated synthetic examples are `shared_embedding_origin.json` and
`shared_embedding_reuse.json`; prior v1 examples remain valid.

## Operational limitations

Deduplication is process-local. Cache eviction, expiry, invalidation and restart
can require a later recomputation. Runtime model/provider configuration is
immutable; a changed endpoint/model needs a new process/service or a distinct
profile revision. No cross-user reuse is allowed. Durable indexing jobs and
`ai_embeddings` remain the recovery/source-of-truth mechanisms. A service cache
never substitutes for eligibility, policy, lease or source-revision validation.

## Current Retrieval policy

BM25 is temporarily disconnected from active query ranking and failure fallback.
Current/contextual/background embeddings continue through this same shared service.
A failed semantic query is reported as degraded; another valid query or existing
independent metadata signal can still supply results within the original budget.
See the Retrieval module README. This does not change cache identity, invalidation,
provider fallback, accounting, indexing jobs, or lexical index maintenance.
