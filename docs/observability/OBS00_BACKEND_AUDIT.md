# OBS-00 — Backend observability audit (Theia)

**Status:** completed source audit and design handoff, **not** a runtime tracing implementation.
**Audited baseline:** `repo (2).zip`, with PR07–PR16 applied in order (BO-04..BO-10, AUTH-05..AUTH-08).
**Audience:** Go backend, DevOps, security/privacy reviewer, and Back-Office frontend agent.
**Next change:** OBS-01 OpenTelemetry instrumentation, implemented as a separate PR.

## Executive decision

**Reuse, do not replace, the existing AI Inspector and the BO-04 durable metadata telemetry.** This codebase already records a remarkably detailed, opt-in, local **full-payload** execution graph, including rendered prompts and actual HTTP wire bytes. The missing pieces for shared navigation and Phoenix are **OTLP spans, a durable Turn/Job ↔ trace index, and a guarded Admin trace-access workflow**. A naive new `PromptLog` table or unguarded `/admin/trace-file` endpoint would regress the existing privacy boundary.

Primary delivery goal: from a *synthetic* failed Turn, find the exact provider attempt and its actual rendered request, provider response (including failed/repaired attempts), execution durations, and linked asynchronous job attempts. Second goal: provide sanitized operational metrics and a future product analytics foundation without replicating private content into logs/SQL/Prometheus.

## 1. Verified inventory — what exists

| Capability | Evidence (relative to repository root) | Actual behavior |
| --- | --- | --- |
| Request and pseudonymous Turn correlation | `apps/api/internal/platform/observability/correlation.go:7-76`, `apps/api/internal/modules/ai/adapters/http/handler.go:148+` | Server-assigned request UUID and `turn_ref` are safe logger fields; raw user IDs/turn UUIDs are not ordinary log fields. These are **not W3C trace IDs**. |
| AI Core execution/orchestration | `apps/api/internal/modules/ai/application/orchestrator/orchestrator.go:103-235`, `:378+`, `:1024+` | Gateway admission, per-turn budget and lease, preparation, safety, routing, Brain, Generator, Validator and finalization are observed; duplicate/queued turns must not manufacture model calls. |
| Full-payload **local** Inspector | `apps/api/internal/modules/ai/trace/inspector.go:44-73`, `:198-255`; `docs/ai-core/local-inspector.md`; `docs/ai-core/inspector-graph.md` | Only `APP_ENV=local`, loopback `HTTP_ADDR`, and explicit `AI_LOCAL_INSPECTOR`; private 0700 directory, 0600 checkpoint JSON. Per invocation and worker claim. No HTTP API or background watcher. Schema `local_ai_trace.v1` and separate standalone offline Explorer. |
| Exact final LLM wire payload | `apps/api/internal/modules/ai/adapters/llmhttp/provider.go:99-139`, `:255+`; `apps/api/internal/modules/ai/adapters/llmhttp/inspection.go` | Instrumented after JSON serialization and schema/model adjustments; request body and HTTP response bytes actually read, including failed HTTP bodies/SSE. Only for `llmhttp`; other providers need their own transport integration. |
| Stage and prompt composition graph | `apps/api/internal/modules/ai/trace/inspector.go`, `graph.go`; `docs/ai-core/inspector-graph.md` | Nested real execution, explicit call-site bindings, prompt component references, retries, regeneration and skipped paths; `exact_json_value_match` is **not** proven causation. Capture already exists; do not create another competing graph schema. |
| Metadata-only port tracer and logs | `apps/api/internal/modules/ai/ports/ports.go` (`Tracer` / `TurnTrace`), `apps/api/internal/modules/ai/trace/recorder.go`, `apps/api/internal/app/composition/ai_logging.go` | Fixed vocabulary stage and sanitized error logs. No message/prompt bodies. `trace.NewRecorder` is a metadata test tool, **not** the full Inspector. |
| Durable physical LLM attempts | `apps/api/internal/modules/ai/llm/attempt_telemetry.go`, `apps/api/internal/modules/ai/adapters/mssql/telemetry.go:21-106`; migration `0027_ai_operational_telemetry.sql` | Begin before provider request and Finish once, including retries, embeddings, and unknown usage/cost; estimated micro-USD is not a paid invoice. Fails if journal unavailable; preserve accounting semantics. |
| Durable stage and worker metadata | `apps/api/internal/modules/ai/adapters/mssql/telemetry.go:113-221`, migration `0027_ai_operational_telemetry.sql` | Sanitized stage observations, heartbeats for memory/reflection/scheduler. Heartbeat is process **liveness**, not successful job processing. |
| Async worker execution | `apps/api/internal/modules/ai/application/memory/worker.go:72-110`, `.../reflection/worker.go:94-128`, `apps/api/internal/app/composition/ai_internal.go:217-285` | Each claimed memory/reflection attempt starts its own local Inspector JSON, associated with durable `turn_id`, `job_id` and attempt number. Do not falsely draw a continuous parent-child tree across async boundaries. |
| Admin operations and auditing | `apps/api/internal/modules/admin/application/inspector.go`, `.../adapters/http/inspector.go`, `.../domain/inspector.go`; `apps/api/internal/app/composition/api.go`; migrations `0026`–`0030` | Active Staff BFF, read-audit **before** data, bounded metadata-only Turn/Jobs/Workers, Overview, Evaluation import/detail, Bundles, Audit. Fail-closed on audit errors. No transcript or local trace-file API. |
| Strict local inspect gate | `apps/api/internal/platform/config/config.go:103-115`, `apps/api/internal/modules/ai/trace/inspector.go:44-74` | Dev Session/Auth settings do not relax Inspector's local + loopback gate. Container API bound to `0.0.0.0` cannot satisfy it by default. |
| User HTTP product route | `apps/api/internal/modules/ai/adapters/http/handler.go:148+`, `.../application/product/service.go`, `apps/api/internal/app/composition/api.go` | Owner derived from User middleware, synchronous final JSON response; internal provider SSE is **not** a public HTTP token stream. |

### No installed external distributed observability was found

Search of runtime source, `apps/api/go.mod`, Compose configuration and operational docs found no OpenTelemetry Go packages, OTLP exporter, Prometheus metric endpoint, Phoenix collector, Grafana, Loki or Tempo configuration in this audited snapshot. No evidence of a W3C `trace_id` stored with Admin Turns was found. Treat all of these as **proposed**, not existing services.

### The critical existing data split (must be preserved)

1. **SQL Server is durable product/operational truth**: Turn/admission/finalization, `ai_llm_attempts`, `ai_stage_events`, `ai_worker_heartbeats`, jobs, and Admin audits. Contains **no full local Inspector payloads**.
2. **Local Inspector is full-payload development evidence**, stored as private JSON files with its own local UUID `trace_id` (NOT the future 32-hex-character OpenTelemetry TraceID); it is not an authenticated API or server-wide trace backend.
3. **Sanitized logs are diagnostics**: structural errors, safe codes and pseudoreferences only. Do not copy `context` inputs or raw error strings into `slog`.

## 2. Actual execution flow and instrumentation locations

```text
GET/POST user API (Gin, RequireUser, request_id)
  -> ai/adapters/http.Handler.Turn
  -> ai/application/product.Service.Process
  -> ai/application/orchestrator.Service.ProcessTurn
       -> Gateway.Admit: persisted user msg + turn / duplicate detection
       -> pin bundle and read state/context
       -> Safety + Retrieval (parallel where applicable)
       -> Assembler -> Router
       -> Brain (unless fast/crisis route)
            -> ports / llm.Client -> meteredProvider -> llmhttp.Provider.send
       -> Protocol + Generator (internal guarded stream)
            -> ports / llm.Client -> meteredProvider -> llmhttp.Provider.send
       -> Validator; may regenerate / retry
       -> Finalizer.Complete: durable assistant msg + state + job/outbox
  -> HTTP JSON result, no unvalidated drafts

Independent claimed background work:
  -> Memory Worker.RunOnce / Reflection Worker.RunOnce
       -> new execution attempt, new local Inspector file
       -> model calls through llm.Client + llmhttp
       -> state validation and persistence/retries

Durable metadata observers (already active):
  -> loggingTracer / TelemetryStore.RecordStage
  -> llm.AttemptJournal Begin/Finish -> ai_llm_attempts
  -> WorkerPulse -> ai_worker_heartbeats
```

**Best hook locations in OBS-01** (keep a single source for each fact):

- **API request/root:** `platform/httpx` middleware and `orchestrator.ProcessTurn` only after the real User boundary. Never trace raw cookies/Authorization headers or URL query secrets.
- **Stage spans:** `orchestrator.prepare`, Safety, Retrieval, Assembler, Router, Brain, Generator, Validator, Finalizer. Use the **already named** `trace.Start` boundaries for naming/provenance; do not delete/rewrite that local graph. Implement the OTel adapter independently and never call Inspector's lazy `input()` just to emit metadata.
- **Each physical provider attempt:** `llm/meteredProvider` around one actual request, with durable `AttemptStart.ID` as the correlation key. Count retries/schema repairs/embeddings separately. The attempt must remain recorded even when validation or parsing later fails.
- **Actual provider wire content:** `llmhttp.Provider.send` **after** final `json.Marshal` and model/schema adjustments; response only from bytes naturally consumed. Do not drain streaming bodies, do not expose API key headers and do not claim synthetic structured input equals exact wire form.
- **Workers:** `memory.Worker.RunOnce` and `reflection.Worker.RunOnce` **after durable claim**. Create a new root trace for each execution attempt; join by persisted `turn_id`, `job_id`, attempt. Use an OTel Span Link only if the actual parent context was deliberately persisted; otherwise do not invent parentage.
- **Durable joins:** SQL `ai_llm_attempts.id` and `turn_id`; stage records and jobs also relate to `turn_id`. The planned `ai_trace_refs` index will map Go/OTel traces without storing content.

## 3. Gaps and risks ranked for the next milestone

| Severity | Gap | Required remediation / no-regression test |
| --- | --- | --- |
| P0 | No OTel spans or exporter, so Phoenix receives nothing. | Introduce a separately configurable metadata-only OTel pipeline with safe no-op mode and exporter failure isolation. |
| P0 | Full Dev content exists only as private Inspector files. No protected Admin trace retrieval. | Start with a separate, authenticated Phoenix UI; do **not** serve inspector directories. Later add audited metadata-only trace refs and narrowly scoped content access if needed. |
| P0 | `local_ai_trace.v1.trace_id` is **not** OTel TraceID. | Use explicit `inspector_invocation_id` vs `otel_trace_id` fields; never rename/alias them. |
| P0 | An unguarded exporter can leak prompts, memories, profile text or credentials to general telemetry. | Metadata-only by default; explicit local-synthetic full-capture gate, payload size limit, secret redaction, dedicated Phoenix destination; fail closed when misconfigured. |
| P0 | Existing Admin role is a single `admin` and current read audit does not cover external Phoenix UI. | Private Phoenix authentication, narrow access; approval of link and later audited content-proxy API. Current Staff session alone cannot authorize external dashboards. |
| P1 | Async workers have separate attempts and no durable W3C TraceID/Span Link. | Add DB trace-ref mapping and annotate real causal links only. |
| P1 | No standard cross-service trace propagation. | Add W3C IDs for internal trusted hops; don't blindly trust arbitrary inbound context or propagate private headers to arbitrary LLM hosts. |
| P1 | Local Inspector's synchronous per-operation fsync changes latency and is bound to loopback. | Preserve this invariant; don't run latency benchmarks with full capture on or widen listener restrictions to fit Docker. |
| P1 | Payload capture can be too large or incomplete when streaming is interrupted. | Byte bound + `captured_bytes`, `truncated`, `stream_complete`, and hash. Never display truncated as exact complete. |
| P1 | SQL costs/usage can be unknown or unpriced. | Preserve nullable fields and configured-price estimate; no fabricated zeros or invoice claims. |
| P2 | Metrics, logs collector and external product analytics are absent. | Add later after trace E2E, with aggregated labels and explicit data-contract definitions. |

## 4. Non-goals of OBS-01

- Replacing or changing the `local_ai_trace.v1` schema and offline Explorer without explicit need.
- Duplicating every prompt and response in SQL Server, app logs, Loki, Sentry or Prometheus.
- Adding user impersonation, arbitrary SQL, real-user transcript search or general Admin access to local files.
- Changing AI inference ordering, safety policies, response streaming semantics, idempotency, provider timeouts or retry decisions.
- Changing Keycloak User/Staff authentication, Dev Session eligibility or real-user rollout scope.
- Claiming traces always cover unavailable/missing stages; absent work must remain absent.

## 5. Verification during OBS-00

Reconstructed patch chain applied cleanly over `repo (2).zip`. Ran:

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -q
```

**Result:** structure/SQL dialect checks passed; OpenAPI/route check passed (66 pairs); 59 Python tests passed. A targeted offline Go test of `ai/trace`, `ai/application/orchestrator` and `admin/application` **could not run** because required Go modules were not cached and `GOPROXY=off` forbids downloads. This is an environment limitation, not a Go test failure. No live SQL Server/Keycloak/Phoenix/E2E test was run. No runtime code, DB migration or OpenAPI endpoint is changed in OBS-00.

**Source hierarchy:** running code/OpenAPI > recent operations docs > older implementation proposals. The prior `docs/plans/README.md` contains historical status text; do not treat its "unimplemented" claims as the status of this reconstructed BO-10 / AUTH-08 snapshot.

**Next reading:** [`OBS01_OTEL_CONTRACT.md`](OBS01_OTEL_CONTRACT.md), [`OBS02_PHOENIX_DEPLOYMENT_DESIGN.md`](OBS02_PHOENIX_DEPLOYMENT_DESIGN.md), and [`OBS_IMPLEMENTATION_TICKETS.md`](OBS_IMPLEMENTATION_TICKETS.md).
