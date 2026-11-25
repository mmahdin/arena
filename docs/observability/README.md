# Theia Engineering Observability

OBS-01A metadata tracing and OBS-01B synthetic wire-content capture passed local OTLP/HTTP checks. OBS-04A enables opt-in local full-content provider wire capture after AUTH-09; it does not reintroduce Dev Session. Shared environments remain metadata-only. OBS-02 adds an opt-in private Phoenix instance with authentication and a local ingestion check. OBS-03 adds durable trace IDs and an audited Staff lookup.

1. [`OBS00_BACKEND_AUDIT.md`](OBS00_BACKEND_AUDIT.md) — what the backend already captures, real execution path and missing pieces.
2. [`OBS01_OTEL_CONTRACT.md`](OBS01_OTEL_CONTRACT.md) — OpenTelemetry span schema, correlation, privacy gates and test design.
3. [`OBS01A_METADATA_TRACING.md`](OBS01A_METADATA_TRACING.md) — implemented metadata-only tracing, configuration, limitations and validation checklist.
4. [`OBS01B_SYNTHETIC_CONTENT.md`](OBS01B_SYNTHETIC_CONTENT.md) — implemented local-synthetic private capture controls, omission semantics, tests and rollout gates.
5. [`OBS02_PHOENIX_DEPLOYMENT_DESIGN.md`](OBS02_PHOENIX_DEPLOYMENT_DESIGN.md) — private Phoenix topology and reviewed Admin integration approach.
6. [`OBS_IMPLEMENTATION_TICKETS.md`](OBS_IMPLEMENTATION_TICKETS.md) — exact implementation order and acceptance gates for the next backend PRs.

7. [Local Phoenix runbook](../operations/obs02-local-phoenix.md) — deployment, authenticated ingestion and persistence checks.

Existing full-inspection reference: [`../ai-core/local-inspector.md`](../ai-core/local-inspector.md) and [`../ai-core/inspector-graph.md`](../ai-core/inspector-graph.md). The full-payload local Inspector remains local/loopback-only and never serves files over an HTTP route.

8. [OBS-03 trace reference mapping](OBS03_TRACE_REFERENCE_MAPPING.md) — migration and Staff endpoint.

9. [MON-01A private monitoring](MON01A_PROMETHEUS_GRAFANA.md) — Prometheus metrics and Grafana.

10. [MON-01B private logging](MON01B_ALLOY_LOKI.md) — sanitized log export, Alloy, Loki, and Grafana Explore.

11. [MON-02 operational alerting](MON02_ALERTING.md) — Prometheus rules, Alertmanager, and local notification testing.

12. [OBS-04A Full Tracing Foundation](OBS04A_FULL_TRACING_FOUNDATION.md) — local API content opt-in, SDK limits, receiver safety, and verified/unverified checks.

13. [OBS-04B Deep RAG Tracing](OBS04B_DEEP_RAG_TRACING.md) — authorized candidate source text, full hybrid rank lists, RRF decisions and returned context in local synthetic OTel events; Local Inspector unchanged.

- [OBS-04C Agent, LLM & HTTP tracing](OBS04C_AGENT_LLM_HTTP_TRACING.md) — local synthetic stage input/output and actual HTTP attempt spans.

- [OBS-04D Portable AI Semantic Attributes](OBS04D_SEMANTIC_ATTRIBUTES.md) — OpenInference span kinds, sanitized input/output, selected RAG documents and physical LLM usage aliases for future Phoenix/Langfuse comparison.

- [OBS-05A Dual OTLP Export](OBS05A_DUAL_OTLP_EXPORT.md) — local bearer-authenticated Collector fans out the same synthetic spans to Phoenix + Langfuse v4; includes private credentials, readback smoke and rollback.

## OBS-05B — Actual trace comparison

Use [`OBS05B_PHOENIX_LANGFUSE_COMPARISON.md`](OBS05B_PHOENIX_LANGFUSE_COMPARISON.md)
for an authenticated same-TraceID dual readback and a 10-span synthetic fixture.
Complete [`OBS05B_MANUAL_SCORECARD.md`](OBS05B_MANUAL_SCORECARD.md) before choosing
Phoenix or Langfuse. Automated parity is not a UI review or a migration decision.
