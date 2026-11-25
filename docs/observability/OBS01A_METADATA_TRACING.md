# OBS-01A — Metadata-only OpenTelemetry backend instrumentation

**Implementation status:** Go runtime implemented and verified against a local OTLP/HTTP test receiver.
OBS-01B content capture, OBS-02 authenticated Phoenix deployment, OBS-03 SQL trace
references/Admin lookup, Grafana/Prometheus and UI integration are **not** part
of this change. Existing BO-04 durable SQL usage and the optional private local
Inspector are preserved; Inspector's UUID is not a W3C OTel TraceID.

## What OBS-01A does

- Uses the Go OpenTelemetry API/SDK and OTLP/HTTP exporter (**pinned v1.38.0**, the last release compatible with the repository's Go 1.23 baseline).
- Defaults to **disabled**. When on, a bounded BatchSpanProcessor exports
  metadata-only root traces for actual AI Turns, child spans for existing code-owned
  stages (admission, preparation, selection, generator/finalizer, etc.) and one
  `gen_ai.client` span for each *durable* physical Structured/Embedding/Streaming
  provider attempt. Usage and cost statuses come from the existing SQL journal;
  unknown usage is not converted to zero.
- Each **claimed** memory/reflection job gets its own independent root with
  Turn UUID, Job UUID and claimed attempt. A worker is never represented as a
  child of an earlier completed HTTP request. The context is propagated
  through the existing Go worker goroutines and request handlers.
- Excludes inbound public `traceparent`/baggage extraction. No global tracer
  context, new public route, SQL migration or automatic propagation of
  internal tracing headers to arbitrary provider URLs.
- Keeps the **local Inspector**'s serialized graph and exact wire-payload
  capture path unchanged. Turning on OTel alone never invokes its lazy
  input/output closures or installs its provider-body capture wrapper.
- OTel spans never contain raw prompts, messages, provider payloads, model
  response bodies, token secrets, user identity, exception text, or SQL errors.
  Allowed attributes are explicitly enumerated in `platform/telemetry`.

## Config

Disabled default (requires no dependencies or running collector):

```dotenv
THEIA_OTEL_ENABLED=false
THEIA_OTEL_CAPTURE_CONTENT=false
OTEL_SERVICE_NAME=theia-api
OTEL_SERVICE_VERSION=dev
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=
```

Opt in on **local host** after deploying an approved OTLP receiver:

```dotenv
APP_ENV=local
THEIA_OTEL_ENABLED=true
THEIA_OTEL_CAPTURE_CONTENT=false
OTEL_SERVICE_NAME=theia-api
OTEL_SERVICE_VERSION=dev
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://127.0.0.1:6006/v1/traces
```

`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` must be a full `/v1/traces` URL, without
credentials, query, or fragment. HTTP is accepted only on local loopback;
all other destinations require HTTPS. The exporter must not receive sensitive
payloads. OBS-04A supersedes the prior API rejection: `THEIA_OTEL_CAPTURE_CONTENT=true`
now works with a loopback-only local API and authenticated Phoenix receiver.
See [OBS-04A](OBS04A_FULL_TRACING_FOUNDATION.md). The existing
`AI_LOCAL_INSPECTOR` switch is separate and remains protected by its own local
loopback guard. For native Go + Docker collector, make sure `127.0.0.1:6006`
resolves to the intended host collector; the OBS-02 design separately controls
Phoenix authentication and network access.

## Span schema examples (no content)

```text
theia.ai.turn [root]
  theia.ai.orchestrator.ProcessTurn (only when local Inspector is enabled)
  theia.ai.gateway.Admit
  theia.ai.application.orchestrator.orchestrator.prepare
    theia.ai.assembler.Build
    theia.ai.router.Select
    ... only actual executed code-owned stages ...
    gen_ai.client [theia.llm_attempt_id, kind/tier, provider/model, status, usage/cost status]
  theia.ai.finalizer.Complete

# An independent trace after a durable background claim:
theia.ai.memory_job [turn_id,job_id,job_attempt]
  theia.ai.application.memory.writer.Propose
  ...
```

**No SQL `ai_trace_refs` mapping exists yet.** The OTel TraceID is visible
only through the authenticated tracing backend. OBS-03 will implement durable
correlation and an audited Admin lookup. It must not reuse or expose the
private Inspector's UUID/file paths.

## Failure and security semantics

- SQL `BeginAttempt` happens **before** provider calls and the OTel attempt span.
  SQL `FinishAttempt` is still the authoritative persistence step. Telemetry
  failure never cancels or duplicates a provider call, though a missing SQL
  journal entry still fails closed as before.
- The exporter queues at most 512 spans, batches up to 64, and cannot block
  a model request on a network export. Shutdown has an independent bounded
  flush after the HTTP server and workers stop. An unreachable collector can
  lose spans; it must not be mistaken for zero AI activity.
- Span links and W3C propagation across *trusted* out-of-process hops are
  intentionally deferred until the corresponding durable link data and trust
  boundaries are implemented. No Turn/User/Job IDs are Prometheus labels.
- Any complete prompt/response capture belongs to OBS-01B with explicit local
  synthetic eligibility, redaction, size bounds and private destination.

## Verification

With network access, from `apps/api` run:

```sh
go mod download                # fetch pinned dependencies using go.sum
GOPROXY=off go test ./internal/platform/telemetry ./internal/modules/ai/trace ./internal/modules/ai/llm ./internal/platform/config
GOPROXY=off go test ./...
GOPROXY=off go test -race ./internal/platform/telemetry ./internal/modules/ai/trace ./internal/modules/ai/llm
```

From the backend repository root:

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -q
```

After OBS-02 adds a private, authenticated OTLP receiver, run a synthetic
completed and failed Turn and confirm real provider attempts, stages, W3C
TraceIDs, separate claimed-job roots and **absence** of prompt/response text.
Do not claim live Phoenix integration from this code-only milestone.

The local OTLP/HTTP smoke verified that a synthetic span reached a loopback
receiver with its trace metadata and without the supplied prompt or raw error
text. This check did not exercise Phoenix or a live AI turn.
