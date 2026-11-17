# AI module — AI Core transition (internal synthetic tests only)

WP-01 introduces independent [contracts](contract/README.md), strict embedded
schemas, and [consumer-owned ports](ports/README.md) from Contract Pack v1.2.
These packages are not wired into the running application. No new AI behavior,
model call, storage, or HTTP contract is introduced by WP-01.

The implementation described below is a **transitional disposable prototype**,
not the foundation or fallback for AI Core. Its bounded removal targets are WP-04
(provider replacement), WP-05 (turn workflow/composition replacement), and WP-16
(obsolete HTTP surfaces). Daily Reflection retirement is tracked separately under
WP-02/WP-16. See [ADR 0008](../../../../../docs/adr/0008-ai-core-contract-foundation.md)
for the dependency inventory and scope boundary. No new contract package imports it.

## WP-02 persistence

[SQL Server persistence](adapters/mssql/README.md) and composition
`NewAIPersistence` implement durable user-first admission, per-user fenced FIFO
leases, recovery and atomic assistant/state/jobs/trace finalization. Migrations
0017–0019 preserve existing raw data and add AI-owned tables. Migration 0020
corrects deletion to soft-only preservation and removes destructive cascades/purges.
Conversation deletion leaves all AI-derived content and consent unchanged; see ADR 0010. This foundation is
not yet connected to the HTTP AI runtime. See ADR 0009 and
[verification](../../../../../docs/quality/ai-core-wp02.md).

## WP-04 provider-neutral client layer

[`llm/`](llm/README.md) implements a NEW structured/streaming LLM client and
embedding companion port from Contract Pack v1.2 §6.3. Independent
`adapters/llmhttp` and `adapters/fake` provide HTTPS Chat Completions-compatible
transport and a deterministic test double; neither imports the old AI module.
Model-tier chains, timeout/retry/breakers, usage quotas/cost and strict post-parse
schema checks live in the new layer. It is deliberately **not** wired into the
running experimental turn route; replacing that workflow is WP-05, so the old
provider has not been silently installed as a fallback. See
[WP-04 verification](../../../../../docs/quality/ai-core-wp04.md).

## Historical prototype behavior

- `policy/prompt.go`: versioned `supportive-v5`, five distinct conversational styles (plus legacy `guidance` alias) and explicitly **untrusted** profile preferences and onboarding answers. Uses relevant stated concerns, goals, interests, strengths, barriers and other explicit background on every turn; current messages/corrections take precedence. Declined disclosures are omitted at the profile port; mood is labeled a past snapshot, never a current assessment. Stored completed turns retain their original prompt version; do not rewrite prior records.
- `policy/safety.go`: normalized Persian/English deterministic phrase heuristics, **NOT clinical screening**. Explicitly matched crisis phrases bypass the provider; matched unsafe model claims are replaced by a static safe fallback. They can still miss indirect intent, unsafe paraphrases and context/negation.
- `application`: formats system + most recent bounded chronological history + new user input; history is a contiguous suffix limited by `MAX_CONTEXT_CHARS` (default 16000 Unicode characters). Repository also limits it by `MAX_CONTEXT_MESSAGES` (default 20). Limits are **character approximations, not token accounting**. The new message is added once, and past conversations are never searched.
- `adapters/mock`: deterministic safe synthetic reply, default.
- `adapters/openai`: server-side HTTPS Chat Completions-compatible request. Requires explicitly configured `MODEL_NAME` and `MODEL_API_KEY`; uses `max_completion_tokens` without temperature for GPT-5/6 and o-series, legacy `max_tokens` for older compatible models. GPT-5.6 uses `reasoning_effort=none` to reserve the small output budget for visible chat. Disallows HTTP redirects, rejects empty/truncated/oversized/filtered responses and returns sanitized errors. Only synthetic selections and synthetic chat are approved.
- A `POST /turns` failing to reach the provider returns `503 AI_UNAVAILABLE`; the current turn lease is released. Client must retry using the **same clientTurnId and identical message**, never auto-create a fresh ID after timeout. Successful turn still saves both messages atomically; no streaming.
- `LLM_TIMEOUT_SECONDS` is 5–90 (default 35), less than the SQL Server turn lease (120 seconds). `MODEL_MAX_OUTPUT_TOKENS` is 128–2048 (default 700), `MAX_CONTEXT_CHARS` is 2000–30000. Application also rejects empty output, malformed UTF-8, unexpected control characters or replies longer than 4000 Unicode code points, and rejects a reply after cancellation even if its gateway ignores context.
- `GET /api/v1/me/ai-status` reports provider/model/mock/promptVersion to the authenticated tester; it does not probe the provider. No API key or endpoint credentials are returned. Provider-specific usage and billing are NOT implemented.
- Safety evaluation and data handling remain blockers for real user data, public exposure or clinical claims. Cost budgets, operational rate limiting, strong output safety and provider-specific observability remain future work. Never log raw prompts, model output, tokens, headers or credentials.

## Test matrix

`go test ./internal/modules/ai/... ./internal/platform/config` exercises synthetic prompt constraints, context window selection, fake HTTPS Chat Completions payload/status/redirect/deadline and output safety. Full `go test ./...` and MSSQL integration should run in the user's local stack; provider smoke needs an explicitly approved non-sensitive test endpoint.

## Task 06 — Additional safety checks

- `supportive-v3` adds explicit rules for medication/diagnosis boundaries, delusion reinforcement, dependency and emergency-service claims. Old `promptVersion` values remain unchanged in SQL Server.
- Both pre-generation crisis routing and post-generation output checking run in application/policy on the server. The checker is phrase-based, intentionally incomplete, and **must not be used as actual mental-health risk screening**.
- A canceled request never accepts a late model reply. Provider failures, invalid output or expired context use generic errors; there is no automatic provider retry. The client keeps the same `clientTurnId` for a user-requested retry.
- See [`docs/quality/ai-safety-evaluation.md`](../../../../../docs/quality/ai-safety-evaluation.md) for synthetic test matrix, known false negatives, and clinical release blockers. No DB migration and no HTTP contract change.
