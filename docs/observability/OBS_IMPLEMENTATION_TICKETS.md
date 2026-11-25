# OBS-00 decisions and the next backend implementation tickets

**OBS-02 local Phoenix ingestion and Go metadata export are verified. OBS-03 implements durable trace reference storage and Staff lookup. Later tickets remain planned.** Each ticket has one owner, bounded code scope, no unrelated product modifications, a complete patch, suggested conventional commit, and tests. The user asked for engineering observability first; Metabase/product analytics and management roles are intentionally deferred.

## Decision register

| Item | Decision for next PR |
| --- | --- |
| Full AI inspection | **Retain** local `trace.Inspector` with its v1 JSON schema and private offline Explorer. No file-serving endpoint. |
| Distributed traces | OpenTelemetry Go SDK and OTLP/HTTP exporter; metadata-only by default. |
| Full synthetic prompt/response | Opt-in local **synthetic-only**, directly to private authenticated Phoenix; redaction and caps required. |
| Source of truth for attempts/stages | Existing SQL `ai_llm_attempts`, `ai_stage_events`, jobs, finalizer and audits; do not duplicate raw text there. |
| Provider attempt identity | Reuse existing `llm.AttemptStart.ID`, not a second guessed attempt counter. |
| Turn–trace join | Add dedicated metadata-only mapping in OBS-03, not a raw trace table. |
| Async worker tracing | New trace per claimed job attempt; join by TurnID/JobID, optional actual SpanLink. |
| Admin trace access | Existing authenticated, fail-closed audit boundary; new approved trace reference route. No direct SQL or local file read. |
| Exporter failure | Never affects inference, billable request or durable finalization outcome. |
| First UI integration | Turn Detail → authenticated Phoenix link + existing metadata; raw in-app content later with reviewed authorization. |

## OBS-01A — metadata-first Go OTel instrumentation

**Owner:** Backend Go. **Dependencies:** OBS-00 design accepted, pinned SDK versions available, existing full Go suite passes locally.

**Code touchpoints:**

- **New** `apps/api/internal/platform/telemetry/`: OTel provider/exporter factory, Resource fields, bounded BatchSpanProcessor, explicit disabled no-op, lifecycle `Shutdown` with timeout; no payload logging and no tracing outside guarded configuration.
- `apps/api/internal/platform/config/config.go` + tests: `THEIA_OTEL_ENABLED`, `THEIA_OTEL_CAPTURE_CONTENT` gates. **Do not** allow a missing or invalid `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` in enabled mode.
- `apps/api/cmd/api/main.go`, `apps/api/internal/app/composition/api.go`, `ai_internal.go`: lifecycle wiring, graceful flush/shutdown, correct context propagation across User HTTP, orchestration and worker routines; no global mutable trace context.
- `apps/api/internal/modules/ai/application/orchestrator/orchestrator.go`: root/admission/prepare/finalizer spans, code-owned status and correct error classification. Prefer using existing `trace.Start` operation names as the canonical vocabulary without serializing private input for OTel-only metadata.
- `apps/api/internal/modules/ai/llm/attempt_telemetry.go`: **one client span per physical attempt** (including embed, schema repair, retry and streaming) keyed by durable AttemptStart.ID; do not change the Begin/Finish SQL ordering or wrapping semantics.
- `apps/api/internal/modules/ai/application/{memory,reflection}/worker.go`: new root per real claimed attempt; no invented parent context.
- `apps/api/go.mod`/`go.sum` with reviewed compatible Go OTel versions, no floating dependencies.

**Acceptance:** A synthetic Turn emits a W3C TraceID and a root/span tree with at least admission, core stages, physical provider attempts and finalization. A claimed background job emits a separate trace joinable by Turn/Job. Only metadata is exported. Disabled mode is demonstrably zero-content/no-export. When exporter is down, user-facing outcome and SQL attempt counts are unchanged.

**Suggested commit:** `feat(observability): instrument AI turns and LLM attempts with metadata-only OTel`

## OBS-01B — private synthetic full-capture extension (isolated exporter verified)

**Owner:** Backend Go + privacy/security review. **Depends on OBS-01A.**

**Touchpoints:** `config`, `llmhttp/provider.go`/`inspection.go`, OpenTelemetry attribute/event adapter, existing secret-redaction helpers. Capture actual **post-serialization wire request** and naturally consumed response; cover structured/stream/embed types and all attempts. Implement the opt-in local-synthetic guard described in `OBS01_OTEL_CONTRACT.md`; secret redaction, per-span byte limits, omission flags and tests for nested secrets and stream interruption. Do not duplicate full payload into SQL or general logs. Preserve Inspector local graph and byte-compatible fixtures.

**Acceptance:** In local synthetic mode, inspection of a successful and a failed Turn shows actual prompt and response content with attempt boundaries and no secrets. With content disabled, marker strings are absent from exported spans. Stage/Prod/public-listener startup refuses full capture. No extra LLM call, stream drain, change to retries, context budget or assistant visibility.

**Suggested commit:** `feat(observability): add gated synthetic AI payload tracing`

## OBS-02 — deploy private Phoenix and validate OTLP export

**Owner:** DevOps/backend. **Depends on OBS-01A (OBS-01B for full content).**

**OBS-02 implementation status:** pinned authenticated localhost-only Compose, owner-only credential helper and CLI smoke are implemented. Docker/Go/Phoenix end-to-end and restart checks remain user-local acceptance gates. See `docs/operations/obs02-local-phoenix.md`.

**Touchpoints:** new opt-in `compose.observability.dev.yaml`, `.env.observability.example` (no secrets), operations runbook, pinned Phoenix image, named volume, private/auth-enabled web+collector, scoped Go API key, smoke tests. See `OBS02_PHOENIX_DEPLOYMENT_DESIGN.md`. Prefer Go native loopback for Dev full capture because local Inspector deliberately rejects wildcard binds.

**Acceptance:** Phoenix is private/authenticated, accepts real Go OTLP/HTTP traces, renders nested AI operations and attempt content, survives restart, refuses unauthenticated access, and records a failed synthetic execution without misleading completeness claims. No new ingress to the User app or public Admin origin.

**Suggested commit:** `infra(observability): add private Phoenix tracing for synthetic development`

## OBS-03 — durable trace refs and audited Admin lookup

**Owner:** Backend Go. **Depends on OBS-01 and OBS-02.**

**Touchpoints:**

- `apps/api/db/migrations/0034_ai_trace_refs.sql` is a forward-only SQL Server migration; `ai_trace_refs` metadata-only schema/indexes. Preserve all historical migrations.
- `apps/api/internal/modules/admin/domain`, `application`, `adapters/mssql`, `adapters/http` and `app/composition/api.go`: bounded `GET /api/v1/admin/turns/{turnId}/trace-references`, with Staff `RequireStaff`, fail-closed durable audit and safe errors.
- `contracts/openapi/openapi.yaml` first: input/response schema, cursor bounds and security; `scripts/check_contract.py` and tests.
- Runtime trace-ref writer near actual Turn admission/claimed job; no raw prompt/response or Inspector filesystem path stored in SQL.

**Acceptance:** An admin selects an existing Turn and receives only the correct W3C trace refs and corresponding known job attempts; non-Staff and wrong-origin requests fail, audit outage returns 503 before read, DB payloads contain no private text, pagination is bounded, Phoenix link destination is validated. Retention is policy-reviewed separately.

**Suggested commit:** `feat(admin): add audited AI trace reference lookup`

## FE-01 — independent Back-Office repo (owned by frontend agent)

**Dependency:** OBS-03 + existing Staff BFF/BO-10 Admin APIs. Implement desktop-first Staff login, Overview, Turn list/detail with attempt and stage timeline, Phoenix deep-link, and failure/empty/stale states. Then jobs/workers, Evaluation history/import, Bundle inventory and Audit from already implemented endpoints. The frontend agent delivers a **complete source repository**, not patches. The Go backend agent does not modify the User app or generate Back-Office UI here.

## MON-01A — private Prometheus/Grafana local slice

Private local metrics listener, bounded HTTP metrics, SQL-committed physical LLM attempt metrics, worker heartbeat metrics, Prometheus API-network-namespace scrape and provisioned authenticated Grafana dashboards. No additional SQL migration or Admin API. Go checks and the live local scrape and dashboard passed with a host-network Prometheus container and native Grafana. See `docs/observability/MON01A_PROMETHEUS_GRAFANA.md` for the standard Compose deployment.

## MON-01B / MON-02 — subsequent technical monitoring

Alloy + sanitized Loki (strict approved JSON log files only, not Docker daemon socket), operational alerts and incident runbooks, optional Tempo. Predefined label allowlists; never put Turn IDs, User IDs, raw prompts/responses or keys into metrics/logs. Existing BO-04 SQL attempt accounting is not replaced by Prometheus.

## PROD-01/02 and OPS-01 — later management workspace

Define event taxonomy now (`user_registered`, `onboarding_*`, `conversation_started`, `ai_turn_*`) but implement durable event capture, privacy-scoped analytics views, Metabase and management roles after the engineering acceptance gate. In particular, no live user transcript access, cross-user search or release/publish buttons are authorized by the current one-role Staff BFF contract.

## Required PR review checklist (every runtime ticket)

- Exact owner module, approved design scope and package dependency direction.
- No prompt/PII/credential/body content in ordinary `slog`, SQL operational tables or metrics.
- No new public route, user impersonation, weakened Auth/Inspector gate or secret in Compose.
- Durable attempt accounting and model calls unaffected by telemetry failure.
- Observed parent/child/link semantics reflect real execution and worker claim order.
- Required failure/negative tests and synthetic payload fixtures; full Go suite and targeted `-race` tests.
- Structure, OpenAPI and Python contract tests; disposable SQL integration test only when schema changes.
- Diff to previous code includes no unreviewed AI behavior or privacy-contract change.
- Every gap and unrun validation disclosed; no claim that "all prompts" are complete when a provider was not instrumented or an SSE ended early.
