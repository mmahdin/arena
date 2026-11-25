# OBS-04D — Portable AI Semantic Attributes (Phoenix first, Langfuse next)

OBS-04D builds on OBS-04A/B/C. **No change to AI routing, ranking, prompt composition,
HTTP requests, token accounting, SQL schema, API contracts or the offline Inspector.**
This milestone annotates **existing** spans; it does not generate synthetic spans or
call models twice. Phoenix remains the active OTLP receiver. Dual export to
Langfuse is a separate OBS-05A milestone, not implemented here.

## Semantics on actual spans

| Existing OTel span (examples) | `openinference.span.kind` | Meaning |
| --- | --- | --- |
| `theia.ai.turn` | `AGENT` | Root of the active AI Turn; never an extra model call |
| `theia.ai.memory_job`, `theia.ai.reflection_job` | `CHAIN` | Independent asynchronous roots |
| `theia.ai.application.retrieval.service.Retrieve`, `theia.ai.retrieval.repository.Scan` | `RETRIEVER` | Authorized memory/document lookup |
| `theia.ai.retrieval.rank` | `RERANKER` | Deterministic RRF fusion/selection, **not** a learned reranker |
| `theia.ai.jev.Select`, `theia.ai.unified.Infer` | `AGENT` | Knowledge selection and unified conversational inference |
| `theia.ai.assembler.Build`, assembler stage | `PROMPT` | Prompt preparation, not provider inference |
| Standalone Safety / Validator spans | `GUARDRAIL` | Historical operations. Active conversational safety runs within Unified |
| Other `theia.ai.*` stages | `CHAIN` | Internal orchestration steps |
| `gen_ai.client` with `chat` | `LLM` | **One actual physical provider attempt**, including retries |
| `gen_ai.client` with `embeddings` | `EMBEDDING` | Actual embedding request, not a chat generation |
| `theia.ai.http.client` | *(none)* | Ordinary HTTP transport child, **not** a second LLM span |

Provider attempts keep the existing `gen_ai.operation.name`,
`gen_ai.provider.name`, `gen_ai.request.model`, and known
`gen_ai.usage.input_tokens`/`output_tokens` OTel attributes. OBS-04D adds
OpenInference `llm.provider`, `llm.model_name`, `llm.token_count.prompt`,
`llm.token_count.completion`, `llm.cost.total` (USD **estimated** from the
pre-existing `theia.llm.cost_micro_usd`) to chat attempts. Embedding attempts
get `embedding.model_name`, **not** the LLM usage aliases. Unknown provider
usage stays **unknown**, never zero; total cost is emitted only when the
existing accounting layer marks it `estimated`. Do not sum parent/child tokens.

## Opt-in I/O mapped from existing, already-redacted evidence

With `THEIA_OTEL_CAPTURE_CONTENT=true` on the approved **local/private synthetic**
configuration only:

- OBS-04C Stage input/output events also populate OpenInference `input.value`,
  `output.value` with `input.mime_type` / `output.mime_type = application/json`.
  These are the already-collected, bounded, credential-redacted stage snapshots.
- OBS-04B RAG `query` and `selected_context` events populate the corresponding
  `input.value` and `output.value` on their **existing** retrieval span.
  Selected results additionally export flattened
  `retrieval.documents.<index>.document.id|content|score` (up to 16, with
  `theia.dev.retrieval.semantic_documents_omitted` count when capped).
  **Active vector/thread/date rankings, RRF and selection decisions remain in the
  existing `theia.rag.*` events**, so no evidence is silently replaced.
  Selected retrieval output is **not proof of inclusion in a final JEV or Unified prompt**.
- OBS-04A physical HTTP wire capture also supplies `input.value` and a
  **complete, non-rejected** `output.value` to the corresponding `gen_ai.client`
  span using a private Go context key. It survives intermediate provider spans.
  The HTTP child retains `theia.dev.wire.*` byte hashes and completion flags.
  Complete SSE captures have MIME `text/event-stream`, not JSON. Missing,
  omitted or interrupted output is **not** represented as a complete generation.

We deliberately do **not** populate `gen_ai.input.messages` / `gen_ai.output.messages`
with raw provider JSON: the OTel GenAI message schema is **not** the same as
OpenAI-compatible request/response JSON. Translating that schema correctly is
future compatibility work; falsely claiming conformance would be worse than
exporting a redacted JSON string via OpenInference.

## Privacy and compatibility

Metadata-only mode **does not serialize input/output**. No raw vectors,
unapproved error bodies, API keys, bearer tokens or cookies are introduced to
metadata. Full-content data is protected by OBS-04A's local/loopback/credential
validation and existing redaction/1 MiB omission rules. This is **not** a
production or real-patient content-capture approval. Redaction is best-effort;
use synthetic data and keep the receiver private.

No new dependencies, env flags, migrations or BFF permissions are required.
A span may be retained in the OTel SDK exporter queue, and content can be
omitted by size or dropped by transport; receiver/UI ingestion is not assumed.
OpenInference display handling differs by Phoenix/Langfuse version, so the next
phase should compare the *same W3C trace IDs* in both services.

Reference schemas:

- OpenInference: https://arize-ai.github.io/openinference/spec/semantic_conventions.html
- OpenTelemetry GenAI: https://opentelemetry.io/docs/specs/semconv/registry/attributes/gen-ai/

## Verification

```sh
cd apps/api
go test ./internal/platform/telemetry ./internal/modules/ai/trace ./internal/modules/ai/llm ./internal/modules/ai/adapters/llmhttp
cd ../..
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests
```

With an authorized local synthetic BFF account and Phoenix running:

1. Capture a synthetic Turn exercising Retrieval, JEV, Unified
   and at least one LLM attempt, optionally with a retry/embedding attempt.
2. Check the same actual trace tree and `openinference.span.kind` values,
   `retrieval.documents.*`, redacted `input.value` / `output.value` and reported
   token counts. Verify that provider HTTP is not a second LLM invocation.
3. Retry with `THEIA_OTEL_CAPTURE_CONTENT=false`; no `input.value`,
   `output.value` or `retrieval.documents.*` content should appear.
4. Test early-closed streaming, provider rejection, oversized body and OTLP
   receiver down. No incomplete output should appear as complete and no AI
   behavior or retry count should change.

**Rollback:** revert this OBS-04D patch. OBS-04C full-content events and all
previous instrumentation survive; no SQL or stored trace reference migration
is necessary. Existing exports are not deleted.
