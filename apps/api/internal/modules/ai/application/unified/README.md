# Unified conversational model

The active conversational inference service consumes `wctx.v3`, Core Policies and
only the complete psychological resources selected by JEV. It verifies source
access and pinned content, invokes `ComponentUnified` once with `MaxAttempts=1`,
and decodes `unified_turn.v2` into reply, independent state proposals, background
requests and future hints. It performs no state or memory writes.

Safety, understanding, response composition, semantic suitability and conversational
choices remain inside this model. Existing Safety instructions/policies are frozen
for this refactor. Structural validation is separate from psychological reasoning.
JEV only selects resources; there is no Brain, Generator, classifier, regeneration
or translation execution path. Provider failures retain the existing failed turn
and authenticated retry/recovery behavior.

See [the active architecture](../../../../../../../../docs/ai-core/unified-memory-centric.md),
`service_test.go` and the opt-in `live_eval_test.go`. Mock/schema tests are structural
engineering evidence; live quality and Safety scenarios require separate evaluation.

Provider transport and sampling capabilities are configured in
`config/ai-runtime.yaml`. Unified uses GPT-5.6 Sol's native `json_schema` format,
with `reasoning_effort: none` and the configured temperature sent on the wire.
The schema is sent once, inside `response_format`, rather than in a system message.
`json_schema_strict: false` preserves v2's optional fields, variable-key protocol
inputs and independent proposal validation. This is native schema guidance, not
constrained decoding: the backend still validates the envelope, each independent
section, references, evidence and proposal dependencies. Other native schemas
retain strict mode unless explicitly configured otherwise. Providers configured
with `native_json_schema: false` keep JSON-object mode with one schema message;
no capability probe or additional inference is introduced.

The schema shares identical UUID, proposal-ID and dependency rules and omits
prompt prose. The prompt describes only the active v2 envelope; asynchronous
analysis uses `background_requests`, not legacy memory/reflection signal fields.
Core Policies and selected resource YAML remain intact and are each supplied once.
Existing assembler identity deduplication and protocol lifecycle resolution remain
the source of context/lifecycle behavior.

Inspector captures the configured model capabilities, effective client request
and exact provider JSON using its existing nodes. The generic HTML viewer supports
both native-schema and historical JSON-object traces without changes.
