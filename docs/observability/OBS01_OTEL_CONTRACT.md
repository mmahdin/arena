> **OBS-04A update:** The original OBS-01B eligibility design below predates
> AUTH-09 and the opt-in local API capture implementation. OBS-04A removes the
> Dev Session and Local Inspector dependency, expands event/value capacity,
> but still requires loopback-only local API + authenticated private Phoenix.
> See [OBS04A_FULL_TRACING_FOUNDATION.md](OBS04A_FULL_TRACING_FOUNDATION.md)
> for the current implementation. This file remains historical design context.

# OBS-01 — Proposed OpenTelemetry + GenAI trace contract

**Status:** OBS-01A metadata and OBS-01B synthetic content export passed local OTLP/HTTP checks. After AUTH-09, content export is available through isolated synthetic tests. This document remains the **target** for OBS-02/03 integrations. See [`OBS01A_METADATA_TRACING.md`](OBS01A_METADATA_TRACING.md) and [`OBS01B_SYNTHETIC_CONTENT.md`](OBS01B_SYNTHETIC_CONTENT.md).

## 1. Invariants and SDK boundary

- Existing `ai/trace` local Inspector and schema (`local_ai_trace.v1`) remain intact. It is the source of highest-fidelity private local execution evidence; **its `trace_id` is a UUID**, not an OTel TraceID.
- Existing `ports.Tracer` and SQL stage/attempt journaling remain the durable, sanitized operational source. OTel cannot alter Turn admission, billing, LLM retry, finalizer semantics or transaction boundaries.
- New OTel SDK ownership belongs in `internal/platform/telemetry` (new). Start it from `cmd/api` or `app/composition` with explicit opt-in configuration; inject the tracer into AI orchestration/LLM and worker composition. Avoid OTel imports in AI Core `contract`, pure domain code or SQL models.
- Use `service.name=theia-api`, `service.version`, and `deployment.environment.name` Resource attributes. Keep code-owned span names, stage/status and model aliases **bounded**; do not synthesize a Prometheus label from Turn/User/Message IDs.
- The API starts a root trace from an authenticated request/process execution. Do not blindly trust `traceparent` supplied by an untrusted public client or send internal `traceparent`, cookies or credentials to an arbitrary model provider. Accept/propagate W3C context only at explicitly trusted internal hops.
- Root/child OTel span context must survive `context.WithDeadline` and safe worker cancellation. **Async claimed jobs start a new trace**, correlated by durable Turn/Job IDs. Use an OTel Span Link only when actual prior SpanContext was persisted and validated.
- Export is asynchronous and bounded (batch size, memory limits, retry limit, timeout). Export failures must never turn a successful inference into a failed/duplicated model call. A configured but unreachable exporter reports telemetry **unavailable**, not "no AI activity".

## 2. IDs, cardinality and privacy

| Field | Source | Where allowed | Important note |
| --- | --- | --- | --- |
| `trace_id` (W3C) | OTel SDK, 32 hex | Phoenix, sampled OTel spans, **new** SQL `ai_trace_refs` mapping and audited Admin trace references | Not the local Inspector UUID |
| `span_id` | OTel SDK, 16 hex | Trace backend | Child-parent hierarchy only for actual synchronous scope |
| `theia.turn_id` | Actual `Gateway.Admit` turn UUID | **Traces** and SQL `ai_trace_refs` only | Never in Prometheus labels or routine logs |
| `theia.job_id`, `theia.job_attempt` | SQL-claimed job | Trace spans / SQL ref mapping only | Attempt-specific root trace; not an invented parent-child relationship |
| `theia.llm_attempt_id` | `llm.AttemptStart.ID` (durable BO-04 row ID) | Trace span + SQL relationship | Enables joining each billable physical provider call |
| `theia.inspector_invocation_id` | Existing local Inspector document UUID | **Local** private debug or controlled trace mapping; absent otherwise | Do not expose filesystem paths |
| `request_id` | Safe server-assigned request UUID | Logs + optional trace attributes | Never forward arbitrary browser request ID as authority |
| `turn_ref` | Pseudonymous hash of Turn UUID | Ordinary logs | Existing safe log correlation; do not reverse or conflate with trace_id |
| `theia.prompt.version`, `theia.bundle.version` | Pinned runtime/manifest | Traces | Small, code-owned version labels; omit raw bundle contents |

The **new** SQL `ai_trace_refs` table is proposed for OBS-03, **not** an OBS-01 requirement. It should contain only Turn/Job/attempt correlation metadata, OTel TraceID, span ID, capture level and event times; no text, prompt, raw response, IP, email, phone, user ID or full JSON trace. Next migration after the reviewed baseline `0031` must be allocated on the integration branch (expected `0032`, verify before authoring). Include proper indexes on `(turn_id, started_at)` and optional `(job_id, attempt)`. Metadata retention is a distinct reviewed policy; do not casually delete owner data or alter previous migrations.

## 3. Span tree and attribute budget

Proposed **minimum root and children**; all optional branches must reflect actual execution rather than a fabricated fixed DAG:

```text
theia.ai.turn [SERVER or INTERNAL, root for actual turn processing]
  theia.ai.admission
  theia.ai.prepare
    theia.ai.safety
    theia.ai.retrieval
    theia.ai.assembler
    theia.ai.route
    theia.ai.brain (when executed)
      gen_ai.client structured attempt #N [physical provider request]
        http.client provider /chat/completions
    theia.ai.protocol (when executed)
    theia.ai.generator
      gen_ai.client stream attempt #N
        http.client provider /chat/completions
    theia.ai.validator
    theia.ai.regeneration (only if executed; may contain new attempt spans)
  theia.ai.finalizer

Independent background trace:
theia.ai.memory_job | theia.ai.reflection_job [claimed attempt root]
  ... actual memory/reflector/embedding and model calls ...
```

Source of truth for actual boundaries: `orchestrator.ProcessTurn`, current `trace.Start` nodes, `meteredProvider` and `llmhttp.Provider.send`. Implement incrementally, prefer a shared metadata-only span hook that preserves `trace.Start`'s lazy payload semantics and existing local Inspector contract. Do not JSON-snapshot full `input()` closures when only OTel **metadata** is enabled. It is acceptable for the first OBS-01 PR to instrument key stages explicitly, then add finer-grained spans without altering execution.

| Semantic field | Value / producer | Capture policy |
| --- | --- | --- |
| `gen_ai.operation.name` | `chat`/`embeddings` or operation matching pinned semantic conventions | Metadata, always when tracing on |
| `gen_ai.provider.name` | Allowlisted provider adapter name | Metadata |
| `gen_ai.request.model` / `gen_ai.response.model` | Actual selected/requested model; safe allowlisted label | Metadata |
| `gen_ai.usage.input_tokens`, `gen_ai.usage.output_tokens` | **Only when provider usage known** | Metadata, don't turn unknown into 0 |
| `theia.llm_attempt_id`, `theia.llm.kind`, `theia.llm.tier` | Durable attempt event / fixed enum | Trace attributes (high cardinal IDs **not metric labels**) |
| `theia.llm.cost_micro_usd`, `theia.llm.cost_status` | Configured-price estimate / unknown/unpriced | Metadata; not invoice spend |
| `theia.stage.status`, `theia.stage.rule_id`, `theia.stage.degraded` | Approved stage/status/rule vocabulary | Metadata |
| `error.type`, OTel span error status | Sanitized machine failure kind | **Never** `err.Error()`, HTTP provider rejection body, or SQL text |
| `gen_ai.input.messages`, `gen_ai.output.messages` | Actual messages reaching/returning from the provider | **Dev synthetic full-capture only**; never default |
| `theia.dev.wire.request_json`, `theia.dev.wire.response_body` | Actual bytes after final JSON serialization/bytes consumed by transport | **Dev synthetic full-capture only**, bounded and secret-redacted |
| `theia.dev.content.sha256`, `theia.dev.content.bytes`, `theia.dev.content.truncated` | Capture integrity and truncation evidence | Dev only; do not mark truncated as complete |
| `theia.dev.stream.complete` | SSE naturally reached final `[DONE]`/EOF | Dev only; never drain provider to invent missing data |

Use a deliberately pinned OpenTelemetry GenAI semantic-convention version at implementation time, plus OpenInference kind hints (`LLM`, `CHAIN`, `RETRIEVER`, `TOOL`) where compatible with Phoenix. The GenAI convention registry explicitly warns that message attributes can include PII. Avoid duplicating identical content under multiple attributes when it increases cost/privacy risk. Exact *transport bytes* and decoded message representations are different evidence types; label them separately. Do not emit embeddings/vector contents into generic OTel spans.

Do not infer prompt versions/knowledge versions that were not pinned during execution. Trace times from OTel are per-operation durations; do not sum parent and nested child durations or count a provider retry as a separate user Turn.

## 4. Explicit content-capture state machine (proposed new config)

Configuration keys used by OBS-01A and OBS-01B:

```dotenv
THEIA_OTEL_ENABLED=false
THEIA_OTEL_CAPTURE_CONTENT=false
THEIA_OTEL_SYNTHETIC_DATA_CONFIRMED=false
# THEIA_OTEL_CONTENT_API_KEY=<private/untracked Phoenix key; set only with content mode>
OTEL_SERVICE_NAME=theia-api
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://127.0.0.1:6006/v1/traces
```

| Mode | App environment | Content | Export target | Startup behavior |
| --- | --- | --- | --- | --- |
| Disabled | Any | No OTel spans and no payload serialization | None | Safe default |
| Metadata traces | Local/Stage/Prod under approved policy | **No message text** | Approved OTLP service | Bounded spans; exporter errors cannot break product |
| Dev synthetic full content | **Only local**, loopback-listening Go API + synthetic-only runtime/data | Final wire/prompt/response only after redaction | Dedicated, loopback/private authenticated Phoenix endpoint | Startup **reject** if synthetic eligibility, private endpoint or explicit flag is missing |
| Full content requested in Stage/Prod or public listener | Stage/Prod/unknown/public bind | Never | Never | Fail startup; do not silently downgrade to unsafe capture |

**Guard requirement:** `THEIA_OTEL_CAPTURE_CONTENT=true` must require `THEIA_OTEL_ENABLED=true`, `APP_ENV=local`, a proven local loopback bind (`LocalDraftListenerAllowed`), deliberate synthetic-only runtime/fixture gate (for example the existing reviewed `AI_CORE_LOCAL_DRAFT` subject to its own safeguards), an explicitly trusted Phoenix endpoint, and credentials/secrets registered for redaction. Merely setting `APP_ENV=local` is insufficient. Do not weaken `AI_LOCAL_INSPECTOR` guards to support a Docker `0.0.0.0` service.

Apply content scrubbing **before export**, not only in Phoenix UI: omit Authorization headers, cookie/session identifiers, `MODEL_*`/`AI_*` secrets, sensitive nested keys, raw SQL args, unrestricted errors and arbitrary credentials. Test a secret embedded in a provider **response** and in a **nested tool result**, not just request headers. Hard cap payload per span/event; record omitted/truncated status and byte lengths. If total capture is truncated, direct developers to the existing local Inspector and visibly say "partial", not "complete".

For OBS-01/02 use **one isolated Phoenix traces pipeline** for Dev full content. Do not fan out those same full-content OTLP traces to a general collector, Loki, Tempo, Sentry, Prometheus or analytics. Metrics/logs later need a **distinct sanitized pipeline**. Exporter failure must not change LLM retry/finalization/accounting outcome.

## 5. Proposed test gates (required in implementation PR)

1. **Disabled:** no snapshot serialization, exporter connection, extra provider call, config secret read, or change to existing local `trace.Start` behavior. Existing `trace` graph/schema fixtures remain byte-compatible.
2. **Metadata-only:** instrument a synthetic request containing unique marker strings across user input, system prompt, memory, response and tool output. Assert no marker appears in any exported span event, attributes, log, metric or SQL metadata DTO.
3. **Full synthetic local:** one normal message captures the exact provider-bound JSON (after JSON-mode schema injection), the returned provider body, usage and rendered prompt; a deliberate retry/repair appears as two or more distinct physical attempt spans with attempt UUIDs.
4. **Provider failure:** capture safe failure classification, status, and **redacted** provider rejection payload in private trace. SQL `ai_llm_attempts` remains the cost/usage truth. Unknown usage stays unknown.
5. **Internal streaming:** preserve only bytes naturally read, incomplete state on interrupted SSE, no added token drain or unvalidated partial reply to HTTP.
6. **Async job:** new trace per claimed memory/reflection attempt, correlated through durable Turn/Job IDs and attempt; no fabricated causal links to unrelated traces.
7. **Security:** reject full-content capture in Stage/Prod, on wildcard/public listener, on non-synthetic runtime, with untrusted destination, or missing secret-redaction configuration. Verify Phoenix auth and no public listener.
8. **Failure isolation:** collector down, trace exporter timeout, buffer full, disk full, delayed SQL commit and concurrent Turns do not change AI outcome, quota, retry or user response; telemetry is marked unavailable.
9. **No leakage:** inspect sample `slog`, error reporting, HTTP request middleware, service stdout and planned general collector for prompt/message/OTP/API key absence; validate user-owner/admin guard invariants.
10. **Existing regression suite:** `go test ./...`, targeted `-race` AI/trace/composition/admin, original trace schema fixtures and offline Inspector Explorer; disposable SQL Server tests if DB mapping changes.

See [`OBS_IMPLEMENTATION_TICKETS.md`](OBS_IMPLEMENTATION_TICKETS.md) for file-by-file execution order.

## Official standards to confirm at implementation time

- OpenTelemetry GenAI attribute registry: https://opentelemetry.io/docs/specs/semconv/registry/attributes/gen-ai/
- OpenTelemetry Go exporters: https://opentelemetry.io/docs/languages/go/exporters/
- OTLP HTTP endpoint syntax: https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/
- Phoenix self-host authentication: https://arize.com/docs/phoenix/deployment/authentication

These URLs are guidance; pin and test exact SDK/image versions in OBS-01/02 rather than accepting floating defaults.
