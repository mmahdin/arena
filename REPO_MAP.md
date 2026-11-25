# Repository Map

| Path | Owner / purpose |
|---|---|
| `MVP_SCOPE.md` | Product's binding internal MVP scope |
| `AGENTS.md`, `PRODUCT_CONSTITUTION.md`, `TECHNICAL_CONSTITUTION.md` | Agent instructions & non-negotiables |
| `apps/api/cmd/api` | API entrypoint / graceful shutdown |
| `apps/api/cmd/migrate` | SQL migration entrypoint |
| `apps/api/internal/app/composition` | Dependency injection + Gin route registry |
| `apps/api/internal/modules/user` | Profile and onboarding-completion state |
| `apps/api/internal/modules/conversation` | Conversation ownership, turn idempotency, history |
| `apps/api/internal/modules/ai` | Prompt policy, gateway adapter, mock and placeholder safety |
| `apps/api/internal/modules/ai/contract` | New WP-01 AI Core types and embedded strict LLM schemas; independent of the transitional prototype |
| `apps/api/internal/modules/ai/ports` | New AI Core consumer-owned interfaces and reusable adapter contract tests |
| `apps/api/internal/platform/config` | Env config + startup guardrails |
| `apps/api/internal/platform/database` | database/sql connection |
| `apps/api/internal/platform/httpx` | Machine errors / HTTP constraints |
| `apps/api/internal/platform/migrations` | Transactional checksummed migration runner |
| `apps/api/db/migrations` | SQL Server forward-only SQL + synthetic seed |
| `apps/api/tests/integration` | Opt-in live SQL Server tests with an isolated disposable database |
| `scripts/run_local.py` | Append missing `.env.example` defaults, load literal `.env` values and execute native Go commands |
| `contracts/openapi/openapi.yaml` | Authored HTTP contract; **no codegen implemented** |
| `docs/adr` | Architectural decision records, including approved-but-unimplemented Auth ADR 0012 and Back-Office ADR 0013 |
| `docs/plans` | Approved Auth/Back-Office v3 source-of-truth plans and backend-first PR runbook; **design documents only** |
| `apps/api/internal/modules/auth` | AUTH-03/04 Go BFF/OIDC, identity-to-UUID mapping, SQL sessions/CSRF, `RequireUser`/`RequireStaff`, Auth audit and staff provisioning |
| `apps/api/internal/modules/admin` | BO-01/02 guarded `/api/v1/admin/me` foundation; operational read models and Admin audit follow in BO-03+ |
| Keycloak + dedicated PostgreSQL (local infrastructure) | AUTH-01 two realms and OIDC clients; AUTH-05 SMS SPI + AUTH-06 explicit User BFF activation |
| `docs/operations` | Local start, quality checks, incident/limit notes |
| `.agents/skills/*/SKILL.md` | Named skill workflows for another coding agent |
| `scripts/check_structure.py` | Agent architecture static smoke check |
| `scripts/check_contract.py` | OpenAPI presence/basic YAML parse check |
| `scripts/new_migration.py` | Strictly increasing sequential migration file generator |
| `compose.yaml`, `Dockerfile` | Local loopback-bound SQL Server and API |

## Module rule

Use `module.go` as public constructor; put business logic in `application`, pure rules in `domain`, network handlers under `adapters/http` and persistence under `adapters/mssql`. AI Core uses its SQL Server adapters and provider-neutral WP-04 client; no prototype gateway remains.

## AI Core WP-02 persistence

- `apps/api/db/migrations/0017_ai_turn_lifecycle.sql` through `0019_ai_owner_integrity.sql`: additive lifecycle/state/owner/privacy foundation.
- `apps/api/internal/modules/conversation/adapters/mssql/lifecycle.go`: raw admission, FIFO, replay/recovery and finalization UoW.
- `apps/api/internal/modules/ai/adapters/mssql/`: state, leases and embedding storage codec.
- `apps/api/internal/app/composition/ai_persistence.go`: infrastructure composition, no runtime cutover.
- `apps/api/tests/integration/ai_lifecycle_test.go`: live SQL Server contract, migration, race and privacy tests.
- `docs/adr/0009-ai-core-sql-lifecycle.md`, `docs/quality/ai-core-wp02.md`: operational contract and verification.

- `db/migrations/0020_soft_delete_preservation.sql` (under `apps/api`): non-destructive metadata, active uniqueness, NO ACTION owner references and physical-delete guards.
- `docs/adr/0010-soft-delete-preservation.md`: binding WP-02 deletion correction; retention and visibility are independent of AI-use policy.
- `apps/api/tests/integration/soft_delete_test.go`: preservation, visibility, re-creation, migration, rollback and delete/finalize concurrency tests.

## Active WP-16 product integration

- `internal/app/composition/{api,ai_product,ai_runtime}.go` under `apps/api`: Gin,
  explicit reviewed deployment dependencies and supervised worker lifetime.
- `modules/ai/application/product`, `modules/ai/adapters/http`: new public use cases
  and typed HTTP boundary; OpenAPI 0.3.0 is authoritative.
- `0023_account_soft_delete.sql`: additive user visibility; all data retained.
- Prototype AI/one-shot providers and top-level `modules/reflection` are retired.
  The old route/prompt descriptions above are historical, not active behavior.

- `internal/modules/admin/adapters/mssql/inspector.go` and `docs/operations/backoffice-inspector-bo06.md`: BO-06 audited, metadata-only turn/job/worker reads.

- `internal/modules/admin/{domain,application,adapters/http,adapters/mssql}/evaluations.go` and `docs/operations/backoffice-evaluations-bo08.md`: BO-08 synthetic WP-15 report import, immutable history, baseline comparison and audited reads.

- **BO-09** read-only Knowledge Bundle inventory: `internal/app/composition/ai_bundle_inventory.go` captures the validated pinned snapshot and up to five explicit operator candidate directories at AI startup; `internal/modules/admin` exposes metadata through audited `GET /api/v1/admin/bundles`. No live release catalog, edits, promotion, or unsigned production fallback.

- **BO-10** hardening: `internal/app/composition/admin_security_test.go` locks the actual guard boundary; `internal/modules/admin/adapters/http/security_matrix_test.go` verifies all Admin read/import paths, CSRF, origin and audit fail-closed behavior; `internal/modules/admin/adapters/mssql/inspector.go` sanitizes unexpected historical SQL labels; `scripts/tests/test_bo10_admin_security_contract.py` catches contract/payload-query drift. See `docs/operations/backoffice-hardening-bo10.md`.

- **AUTH-05** Java Keycloak SMS SPI: `infra/keycloak/sms-spi/` (Keycloak 26.8.2 provider + Mock/Kavenegar transport, no Go API routes), `infra/keycloak/themes/theia-sms/` (Persian phone/OTP UI), `infra/keycloak/Dockerfile.sms` and `compose.auth-sms.yaml` (opt-in local image); `scripts/auth/ensure_local_sms_key.py` and `configure_user_sms_flow.py` (non-destructive activation). AUTH-06 provides the explicit User BFF activation gate; disabled by default. See `docs/operations/auth-sms-otp-auth05.md`.

- **AUTH-06** User OIDC/Keycloak SMS attestation: `scripts/auth/activate_user_bff.py`, signed User Session Note claim verification in `internal/modules/auth/oidc`, provider-neutral `internal/platform/principal`, `docs/operations/auth-user-sms-auth06.md` and an opt-in disposable SQL identity test. No new database migration; User Login remains opt-in.


- **AUTH-08 session lifecycle:** `apps/api/internal/modules/auth/lifecycle.go`,
  `oidc/logout.go`, `adapters/mssql/lifecycle.go`, migration
  `0031_auth_backchannel_logout.sql`; `scripts/auth/configure_keycloak_logout.py`
  configures the local Keycloak clients. The HTTP endpoints use signed OP
  notifications and explicit browser navigation for SSO logout; see
  `docs/operations/auth-session-lifecycle-auth08.md`.

## OBS-00 observability audit and OTel/Phoenix next-stage design

- `docs/observability/OBS00_BACKEND_AUDIT.md` — source-verified inventory of existing local AI Inspector, BO-04 SQL metadata, Admin read guards and missing external tracing; no runtime changes.
- `docs/observability/OBS01_OTEL_CONTRACT.md` — planned W3C trace/GenAI span contract, correlation, Dev-only payload gate, secure export and negative tests.
- `docs/observability/OBS02_PHOENIX_DEPLOYMENT_DESIGN.md` — proposed authenticated local Phoenix deployment and Admin trace-link boundary; not yet installed.
- `docs/observability/OBS_IMPLEMENTATION_TICKETS.md` — ordered backend tickets OBS-01A, OBS-01B, OBS-02, OBS-03 and later frontend/monitoring phases.

The local full-content Inspector is already implemented under `internal/modules/ai/trace`; its `trace_id` is not a W3C OTel TraceID. No OpenTelemetry exporter, Phoenix deployment, or Admin trace-content endpoint is implemented by OBS-00.

## Active unified conversation and JEV (2026-11-24)

- `modules/ai/application/unified` under `apps/api/internal`: one typed conversational inference, including conversational safety and reply/state intents.
- `application/orchestrator.NewUnified` is the sole production composition path; legacy constructors and stage ports have been removed.
- `application/assembler/layers.go`, `adapters/mssql/persistent.go`, and `app/composition/ai_persistent_profile.go`: independent baseline, active state, retrieved evidence and chronological history.
- Migrations `0039`/`0040`: bounded context selection indexes and versioned hashed lexical metadata on existing SQL Server infrastructure.
- `docs/ai-core/unified-memory-centric.md`: authoritative active lifecycle/contracts/limits; earlier architecture and Contract Pack flow descriptions are superseded.

## OBS-01A — opt-in metadata-only OpenTelemetry (source implemented)

- `apps/api/internal/platform/telemetry/`: pinned Go OTel SDK + private OTLP/HTTP
  exporter, bounded batches, attribute allowlist, no global tracer context.
- `apps/api/internal/platform/config/config.go` and `cmd/api/main.go`: disabled
  by default; explicit endpoint/HTTPS validation; independent bounded shutdown.
- `apps/api/internal/app/composition/{api,ai_product}.go`: inject tracer into
  actual HTTP request and worker contexts; untrusted client traceparent ignored.
- `apps/api/internal/modules/ai/trace/{inspector,graph}.go` and
  `llm/attempt_telemetry.go`: trace roots, code-owned stages and **physical**
  LLM attempts without serializing local Inspector payloads; independent claimed
  worker jobs. Local Inspector JSON schema/Explorer stay unchanged.
- `docs/observability/OBS01A_METADATA_TRACING.md`: config, privacy scope and
  exact remaining validation. No Phoenix, SQL trace refs or Admin trace lookup.

Build note: OTel modules are pinned to v1.38.0 (Go 1.23-compatible), but this
patch's offline authoring environment cannot resolve their Go sums. The target
checkout must run `go mod tidy` with network and commit the generated `go.sum`
before Go CI can validate the merged change.

## OBS-01B private local synthetic content capture

- `internal/platform/telemetry/content.go`: bounded, private-only wire capture,
  credential scrubber, completeness/omission flags; never ordinary metadata fields.
- `internal/platform/config/config.go`: multi-factor synthetic-only guard,
  loopback Phoenix receiver and server-held ingestion key; BFF User mode excluded.
- `modules/ai/adapters/llmhttp/{provider.go,content_capture.go}`: capture exact
  post-serialization wire requests and only naturally consumed JSON/SSE replies.
- `docs/observability/OBS01B_SYNTHETIC_CONTENT.md`: runbook and tests.
- Phoenix itself is **not** installed; OBS-02 is still required for live traces.

## OBS-02 — Phoenix synthetic development telemetry deployment

- `compose.observability.dev.yaml` — independent Compose project, Phoenix 22.0.1, loopback-only port, authenticated OTLP/HTTP and persistent SQLite volume.
- `.env.observability.example` — variable names only; generated `.env.observability` is excluded from Git.
- `scripts/observability/init_local_phoenix_env.py` — random owner-only Phoenix credentials, hidden ingestion-key entry.
- `scripts/observability/smoke_phoenix.py` — reject anonymous reads/ingest, authenticate synthetic OTLP/protobuf, verify query/readback/restart.
- `scripts/observability/run_native_with_phoenix.py` — native loopback synthetic Go runner honoring OBS-01B gates.
- `docs/operations/obs02-local-phoenix.md` — setup, provider/trace acceptance and limitations.

OBS-02 does not add Admin trace-reference endpoints, public proxy routes, role management, SQL migrations, or Grafana/Prometheus.

### OBS-03 — W3C trace-reference mapping and guarded Admin lookup

- `apps/api/db/migrations/0034_ai_trace_refs.sql`: append-only root correlation
  metadata; Turn/Job links and bounded lookup index, no content.
- `apps/api/internal/modules/ai/trace/references.go` and
  `apps/api/internal/modules/ai/adapters/mssql/trace_refs.go`: actual admitted
  Turn and claimed job root references, best-effort bounded SQL persistence.
- `apps/api/internal/modules/admin/{domain,application,adapters}`:
  audit-before-read `GET /api/v1/admin/turns/{turnId}/trace-references`.
- `docs/observability/OBS03_TRACE_REFERENCE_MAPPING.md` and
  `docs/operations/obs03-admin-trace-lookup.md`: contract, privacy limits, and
  live deployment/verification tasks. No Phoenix URL or content is exposed.

## MON-01A — private Prometheus/Grafana local monitoring

- `apps/api/internal/platform/metrics/`: official Prometheus client with a per-process registry, fixed-label HTTP/physical LLM attempt/worker heartbeat observations. Never record raw URLs, user/turn IDs or content.
- `apps/api/cmd/api/main.go`: opt-in private loopback **separate** metric listener; not mounted in Gin/OpenAPI.
- `apps/api/internal/app/composition/monitoring.go`: SQL-backed LLM attempt and heartbeat observers (no billing or worker lifecycle change).
- `compose.monitoring.dev.yaml`, `infra/monitoring/`: local private Prometheus scrape + authenticated Grafana, with provisioned engineering dashboard.
- `scripts/monitoring/` and `docs/observability/MON01A_PROMETHEUS_GRAFANA.md`: credential initializer, smoke test and runbook. Loki/Alloy remains MON-01B.


## MON-01B — private sanitized Loki/Alloy operational logging

- `apps/api/internal/platform/observability/safe_loki.go`: opt-in local-only
  allowlisted JSONL exporter, bounded rotation, typed metadata; no raw logger
  shipping, request bodies, prompts, responses, user IDs or secrets.
- `apps/api/cmd/api/main.go`, `internal/platform/config/config.go`: safe export
  initialization and environment/path gating. No change to normal stderr logs.
- `compose.logging.dev.yaml`: optional overlay layered with base + MON-01A,
  non-root API/Alloy/Loki and separate private named volumes, no exposed ports.
- `infra/monitoring/{alloy,loki}/` + Grafana Loki data source/log dashboard:
  private file-tail -> Loki and visual exploration; no Docker socket mounts.
- `docs/observability/MON01B_ALLOY_LOKI.md`: runbook, limitations, tests.

- MON-02: `compose.alerting.dev.yaml`, `infra/monitoring/alerts/`, `infra/monitoring/alertmanager/`, `infra/monitoring/prometheus.alerting.yml`, and `docs/operations/mon02-alerting.md` own local synthetic Prometheus-rule evaluation and a single Alertmanager-to-Mailpit notification path. `theia_worker_expected` appears only when AI Core is configured.


## PROD-01 — synthetic-only transactional product events

- `apps/api/db/migrations/0035_product_events.sql`: fixed allowlist event ledger with UTC time, unique `(event_name,subject_id)`, synthetic-only daily aggregate view; no new API.
- `apps/api/internal/platform/productevents/`: typed event vocabulary and transaction-bound SQL writer, no arbitrary event properties or payloads.
- First-party SQL adapters for Keycloak synthetic user creation, onboarding progress/completion, conversation creation, v1 AI Turn admission/complete/first failure, and first saved bookmark insert emit facts *inside* their business transactions.
- `docs/analytics/PROD01_PRODUCT_EVENTS.md` and `docs/operations/prod01-product-events.md`: precise event meanings, local SQL acceptance, retention/security limitations, PROD-02 read-only analytics boundary.
- Real users are **not** tracked, no historical backfill, no new admin/user API, no DAU/retention claims. Migrate through **0036 before** deploying this code.

### PROD-02 synthetic product analytics

- `apps/api/db/migrations/0037_product_reporting_views.sql`: aggregate reporting views and a view-only SQL role.
- `apps/api/cmd/analytics-reader/`: local SQL login bootstrap.
- `compose.analytics.dev.yaml`: loopback Metabase with a separate PostgreSQL application database.
- `scripts/analytics/`: local credentials, reader setup, and dashboard provisioning.
- `docs/analytics/PROD02_METRICS.md` and `docs/operations/prod02-metabase.md`: metric definitions and operations.

- `application/knowledge`, `knowledge/cards.yaml`, `prompts/jev.system.tmpl`: bounded card-based JEV selection and pinned complete resource loading.
- `contract/schema/unified_turn.v2.json`, `application/orchestrator/proposals.go`: independently validated reply/state/background/hint sections and dependencies.
- `application/retrieval/research.go`, existing Memory/Reflection workers: post-finalization analytical retrieval with source/version fences.
- `0041_unified_turn_decisions.sql`: immutable owner-scoped decision/carry-over receipts committed with the response and jobs.
- `docs/adr/0015-unified-knowledge-selection.md` and `docs/quality/unified-refactor-2026-11-24.md`: decisions, rollout and exact validation evidence.

### Phase 2 background intelligence

- `application/background`: shared Memory Intelligence/Reflection contracts, bounded
  preparation/inference, independent operational validation and encrypted checkpoints.
- `adapters/mssql/intelligence_*.go`: owner-scoped batches, real retrieved targets,
  atomic memory/formulation/hint/coverage persistence on existing SQL transactions.
- `background_hints.go`: optional provenance-bound recommendations for future turns.
- Migration 0042 and ADR 0016 supersede per-turn routine Memory Writer scheduling.
