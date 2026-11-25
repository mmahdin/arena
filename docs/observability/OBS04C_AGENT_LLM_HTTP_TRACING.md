# OBS-04C — Agent, LLM & HTTP Tracing (local synthetic)

This patch builds on **OBS-04A and OBS-04B** and preserves the existing
`local_ai_trace.v1` Inspector schema and the user-visible AI response path.
There is no database migration, OpenAPI change, additional model invocation,
new dev identity or Phoenix replacement.

## How to run

Use the OBS-04A private loopback Phoenix runbook and synthetic Keycloak user:

```sh
python3 scripts/observability/run_native_with_phoenix.py --content
```

Full content is **disabled by default** and remains gated by local environment,
loopback HTTP and authenticated private loopback Phoenix ingestion. It must not
be used for real mental-health disclosures; the existing redactor is a
best-effort secret scrubber, not a substitute for data governance.

## Added trace data

- All existing `trace.Start` / `trace.Span.End` stage boundaries automatically
  add `theia.ai.stage.input` and `theia.ai.stage.output` events in content mode,
  including JEV, Unified, Orchestrator and Background Intelligence. Each event
  carries `theia.dev.stage.json`, `original_bytes` and explicit `omitted` flags.
  These values are the already-defined code-owned snapshots; callers are not
  invoked again for tracing. Metadata-only mode evaluates no lazy closures.
- Existing code-owned Inspector artifacts such as `rendered_prompt`,
  `prompt_composition`, gated decision, retry delay, attempt result and fallback
  selection can also appear as `theia.ai.stage.artifact` events without enabling
  the file-based Inspector. Streaming token-by-token `draft` and `chunk` are
  intentionally not duplicated into stage events.
- The shared LLM provider wrapper now creates full-input/output stage events
  for Structured, Embed and Stream calls when private OTel content is enabled,
  even when the optional Inspector is off. The exact post-serialization request
  and already-consumed provider response remain in `theia.dev.wire.*` on their
  physical request span (OBS-01B/A). Retry and per-attempt accounting remain
  the responsibility of the existing WP-04/OBS-01A boundaries.
- Each actual HTTP provider request receives its own `theia.ai.http.client`
  span with safe `http.request.method`, `url.path`, `http.response.status_code`,
  server-observed duration (via OTel span timing) and an explicit
  `theia.http.response_incomplete` flag for early-closed bodies. No provider
  hostname, full URL, cookies, authorization headers or error bodies are sent
  as HTTP metadata. The span remains open while the real consumer reads the
  JSON or SSE response; tracing never drains the body.

The JEV and Unified stages capture composed inputs and returned outputs.
The provider/HTTP layer records the final serialized request sent to the model. The two can differ after
model-specific parameter conversion; prefer the physical wire capture for
exact provider input. OBS-04B Retrieval's `selected_context` is not proof of
prompt inclusion until correlated with those final render/wire events.

For streaming, the provider stage output captures the end state; the
wire capture records bytes actually consumed. The provider may return malformed
or partial SSE; explicit `complete` and `omitted` fields must not be mistaken for
successful validation. Application stage outputs are the actual returned structs,
not hidden reasoning tokens or an independent causal explanation.

## Privacy and resource constraints

Only synthetic private opt-in runs export stage payloads. Unknown/bad JSON,
serialization failures and values above 1 MiB are explicitly omitted rather
than silently truncated. The shared regex/JSON-key redactor hides common
credentials and the private collector ingestion key; **arbitrary secrets hidden
inside free text cannot be guaranteed to be detected**. HTTP metadata excludes stack traces, request headers and provider error body strings.
OTel's per-span event limit and bounded exporter queue still apply; high-load
traces can be dropped, so completeness must be checked on the receiver.

## Verification

```sh
cd apps/api
go test ./internal/platform/telemetry ./internal/modules/ai/trace ./internal/modules/ai/llm ./internal/modules/ai/adapters/llmhttp
cd ../..
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests
```

Then execute **one synthetic Turn** through a live Keycloak/BFF/SQL/Phoenix
stack, inspect the same W3C `trace_id` in the Turn's Admin Trace References,
and verify JEV and Unified input/output, proposal resolution, each
physical LLM attempt, retries and the HTTP request status/timing. Confirm
no payload events with `THEIA_OTEL_CAPTURE_CONTENT=false`, and that a broken
OTLP receiver cannot change the response or cause a new model call.

**Rollback:** set `THEIA_OTEL_CAPTURE_CONTENT=false` (removes detailed payloads,
not structural metadata), or revert this OBS-04C patch. Existing Phoenix traces
are not deleted by a code rollback. OBS-04D will add portable AI/OpenInference
semantic labels for tool-specific rendering; this step uses Theia names.
