# Shared embedding computation

One service implements `ports.EmbeddingClient` for every production embedding
consumer. It owns bounded TTL/LRU reuse, in-flight coordination, defensive copies,
execution lifetime and per-user invalidation. It owns no memory, retrieval or SQL
policy. `ports.EmbeddingProfileResolver` exposes the configured vector identity
before lookup; the existing client still executes, meters and retries providers.

See `docs/ai-core/shared-embeddings.md` for runtime settings, lifecycle, Inspector,
privacy boundaries and limitations. Unit/concurrency tests use deterministic
providers; integration tests exercise real Retrieval and Memory Worker SQL paths.
