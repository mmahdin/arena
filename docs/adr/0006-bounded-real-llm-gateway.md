# ADR 0006 — Bounded, optional external LLM gateway

Status: Accepted for **internal synthetic MVP only** (Task 05).

## Context

Task 04 guarantees a fenced, idempotent turn in SQL Server, but the existing provider adapter had no output token limit, allowed implicit HTTP redirects, and had minimal reply validation. Short-term history had a message-count limit but no text-size budget.

## Decision

- Keep `LLM_PROVIDER=mock` by default. Opt into `openai-compatible` only with a configured HTTPS endpoint, model name, and secret injected on the server.
- Submit `system + bounded recent conversation + current message` through the existing application port. No inferred memory, RAG, diagnosis, analytics or new client API. Budget recent history by Unicode characters; message count remains limited by the SQL adapter. No sentence/word truncation.
- Version the revised conversation prompt as `supportive-v2`; older turns keep their prompt version. Treat profile and history as untrusted content; make all guidance explicitly non-clinical.
- Send `max_tokens`, limit response body size to 256 KiB, limit resulting assistant text to 4000 code points, reject incomplete/filtered/empty choices. Refuse redirects; do not include upstream error text in application or HTTP errors.
- Explicitly bound provider timeout to 90 seconds, below the fixed 120-second turn lease. Client Retry with the same UUID can resume after failures; never auto-retry the provider (cost/duplication side effects).

## Consequences and limitations

- Existing `POST /turns` JSON contract, DB schema and SQL migrations are unchanged. Upstream provider errors remain `503 AI_UNAVAILABLE`.
- The prompt and heuristic safety checks are **not** validated for clinical use. This integration is not approval to process real mental-health content.
- Character budgeting is not provider tokenization. `max_tokens` is a common Chat Completions-compatible setting, not universal for every model. Account-level quotas, rate limits, cost caps, structured safety evaluation, and real-provider integration tests are outstanding.
- The 120-second lease does not cover unbounded DB congestion. Production timeouts, tracing without sensitive text, and multi-turn concurrency need separate review.
