> Phase 2 background sections are superseded by ADR 0016 and
> `docs/ai-core/background-intelligence.md`: periodic batches replace per-turn
> extraction; the Phase 1 synchronous pipeline remains intact.

> **2026-11-24 active runtime:** ADR 0015 and `docs/ai-core/unified-memory-centric.md` govern JEV knowledge selection, immediate dual-query retrieval, independent proposals and existing background workers. Runtime v4 has one conversation path; Safety remains within Unified.

# Theia AI Core

WP-16 wires the implemented WP-01–15 and Hardening modules into Gin through
`internal/app/composition`. User turns invoke Gateway/Orchestrator directly;
Conversation owns the single durable message/turn store. No prototype reply,
old prompt/safety gateway, or Daily Reflection dependency remains.

- `application/product` owns explicit authenticated HTTP use cases and consumes
  typed ports. `adapters/http` maps camelCase JSON to canonical core contracts;
  drafts and internal streaming sinks never leave the process.
- `NewAIInternal` builds Retrieval, Context/State, JEV, Unified, Protocol operations,
  Memory and Reflection/Scheduler. `NewAPI.Run` supervises the existing workers.
  No Brain/Generator pipeline or conversational classifier is constructed.
  Scheduler remains maintenance-only; user wake-ups/outbox delivery stay disabled.
- A released WP-03 bundle, WP-04 provider chains, explicit reviewed privacy
  mapping and managed encryption keys are required by the deployment factory.
  An absent configuration exposes no fallback AI: AI APIs/readiness return 503.
- User-initiated AI chat, retry and recovery require no separate chat-consent flag.
  Memory retrieval, reads, writes, reflection and background processing require no
  consent, including accounts with missing/false consent or legacy paused state.
  Controls always report `memoryPaused: false`; pause input is ignored for compatibility.
  Ownership, visibility, risk, evidence and transaction/version checks remain.
  Hidden retained sources still require their separate explicit permission;
  proactive execution and protocol consent are unchanged. Memory views/edits/corrections use WP-12.
- Migration 0023 adds account visibility metadata. Account hiding revokes sessions,
  fences AI work and retains all messages, memory, evidence, consent and legacy
  Daily Reflection data. No hard deletion or retention cleanup is implemented.

See [WP-16 report](../../../../../docs/quality/ai-core-wp16.md),
[HTTP contract](../../../../../contracts/openapi/openapi.yaml) and
[deployment/client migration](../../../../../docs/operations/ai-core-wp16.md).
Full encrypted runtime replay and reflection merge/split remain blocked on the
contracts identified by Hardening. All tests/fixtures are synthetic; passing
engineering checks is not clinical approval or authorization for real disclosures.

English knowledge supports English/Persian conversation through the saved
`PrefsView.Language`, independently of input `TurnView.Lang`. See the
[multilingual amendment](../../../../../docs/ai-core/multilingual-conversations.md)
for deterministic resolution and resource/review gates. English CBT/ACT knowledge
can be delivered in Persian by therapeutic intent; see the
[product-first amendment](../../../../../docs/ai-core/product-first-conversation.md)
for requests/consent, natural protocol progression and advisory style validation.
English and Persian draft content remains unreviewed and synthetic-only.

The opt-in [Local AI Core Inspector](../../../../../docs/ai-core/local-inspector.md)
extends the trace package with full local payloads, execution/data relationships,
provider wire requests and usage. It requires `AI_LOCAL_INSPECTOR=true`,
`APP_ENV=local` and a loopback listener; ordinary metadata traces remain unchanged.
The [graph contract](../../../../../docs/ai-core/inspector-graph.md) documents
explicit value bindings and nested indexes into existing rendered prompts, with
schema-valid full-turn fixtures running the real application services.

The offline [AI Execution Explorer](trace/explorer/README.md) opens locally
selected Inspector JSON files without a backend connection. When changing
trace payloads, schemas or AI Core instrumentation, follow the
[trace/Explorer agent instructions](trace/AGENTS.md) to keep the viewer and
synthetic examples compatible.

OBS-04C extends existing lazy Inspector stage boundaries to synthetic, private
OpenTelemetry input/output events and physical provider HTTP spans; the
`local_ai_trace.v1` schema and offline Explorer remain unchanged. See
[OBS-04C](../../../../../docs/observability/OBS04C_AGENT_LLM_HTTP_TRACING.md).
