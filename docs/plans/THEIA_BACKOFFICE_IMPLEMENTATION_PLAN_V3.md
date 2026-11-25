# Theia Back-Office - Implementation Plan v3 (updated backend)

**Status:** Approved target design, NOT implemented. **Reviewed repo:** `repo (1).zip` uploaded 2026-11-19. **Dependency:** `THEIA_AUTH_IMPLEMENTATION_PLAN_V3.md` is authoritative for Keycloak/PostgreSQL, Staff Login, sessions, cookie/CSRF, identity mapping, `RequireStaff` and developer-auth cutover. This document owns Admin use cases, Admin APIs/UI, audit of admin behavior, safe telemetry and diagnostics.

## A. Locked scope and product decisions

- Backend: keep the existing Go/Gin modular monolith + SQL Server 2022; new `internal/modules/admin` and guarded `/api/v1/admin/*` only. Frontend: separately deployed Back-Office web app; implement AFTER backend contracts and P0 APIs.
- MVP is **one fixed `admin` role**; all *implemented* Back-Office capabilities available to all active admins; no role editor, grants table or seven-persona RBAC in P0. This does not authorize implicit direct SQL or arbitrary safety/consent mutations; such operations must be implemented as explicit, audited application use cases. Admin can access synthetic diagnostic content intentionally, not by accidental default logging.
- Staff sign in via the shared Keycloak `theia-staff` realm (email/password, no MFA per team choice). Local loopback now; private VPN ingress mandatory before deployment to the team on a shared network.
- No dev-session identity is accepted as admin. Use isolated synthetic fixtures and Mock SMS for test users. Initial release is **synthetic-only**; production real-user sensitive content and clinical workflow gates remain separate.
- No new AI service, unrestricted transcripts in overview, user impersonation, direct AI risk/state mutation, unapproved clinical release, proactive dispatch, or unsupported permanent erasure.

## B. Updated repository findings - changes since v2

| Verified update | Revised Back-Office consequence |
|---|---|
| New `docs/ai-core/local-inspector.md` and `trace/local_ai_trace.v1.schema.json` | Inspector offers a useful local developer JSON view of actual pipeline nodes, edges, prompts, drafts and provider wire. It is NOT safe as a remote admin API or general telemetry source. Keep its files private and local; design the Back-Office inspector as a different metadata/durable read model. |
| `internal/app/composition/ai_logging.go` now attaches `loggingTracer`, not `nullTracer` | Remove outdated "runtime uses nullTracer" assumption. Reuse existing safe stage and degradation event vocabulary; no need to build an unrelated logger. |
| `platform/observability` + `httpx/logging.go` now return `X-Request-ID`, metadata-only diagnostics | Use request ID as a short-lived diagnostic entry point; do not claim cross-process durable lookup from logs. `turn_ref` is pseudonymous with process-local salt and cannot link across restarts. |
| `loggingTurn.AddUsage` remains a no-op; LLM `MemoryLedger` remains process-local | BO-04 still needs durable per-attempt model usage, stage/worker metrics and cost accounting. Local inspector totals do not solve production observability. |
| Multilingual product/safety tests and substantive draft `knowledge/` (18 skills, 7 protocols, 28 golden fixtures) | Extend synthetic Eval Viewer scenarios, language/resource coverage and golden provenance. Do not label lint/fixture presence as clinical approval or a successful model-run. |
| OpenAPI is still 0.3.0; SQL migration head still 0023; no admin routes | Contract-first BO work, additive migrations, initial module still required. |
| `NewAPI` and AI runtime bootstrap depend on `AllowDevLogin` | BO-01 blocks on AUTH-02. Auth must make user/admin/API/AI bootstrap independent of Dev Login before BO can function with it off. |
| Local Inspector requires loopback bind; current Compose API binds `0.0.0.0:8080` internally | Run sensitive full-payload inspector only on native loopback development setup, not as an exception to its gate in Compose or a shared admin server. |

**Status reality:** repo `knowledge/README.md` describes a *synthetic-only unsigned development draft*, not approved live clinical assets; `docs/operations/ai-core-wp16.md` remains internal-only. No live Auth, Admin UI or real-user deployment approval is implied.

## C. Module boundaries and source ownership

```text
Admin frontend (separate deployment, private when shared)
  -> same-origin /api/v1/admin/* through reverse proxy
Go/Gin
  -> auth.RequireStaff (from shared Auth module)
  -> internal/modules/admin
       domain/       fixed admin capability, audit, case state
       application/  admin use cases + safe query ports
       adapters/http/     validated handlers + pagination
       adapters/mssql/    bounded read projections, admin audit, telemetry
       module.go
  -> existing AI Core / product repositories via explicit application ports
  -> existing SQL Server + additive admin tables
```

Auth owns `auth_identities`, `staff_accounts`, `auth_sessions`, optional `auth_login_transactions`, and `auth_audit_events`. **Do not create** `admin_staff_identities` or `admin_role_grants`. Admin owns `admin_audit_events`, durable `ai_llm_usage_events`/safe trace metrics, optional worker heartbeat, `admin_eval_runs`, and P1 case/approval tables. Assign SQL migration numbers sequentially in merged PR order, AFTER Auth migrations; do not reserve `0024` in both streams. Reuse existing `turns`, `ai_jobs`, `ai_turn_traces` and AI read models; do not duplicate original messages/memories.

## D. P0 navigation and user workflows

| Screen | Primary job | Data rules |
|---|---|---|
| Overview | Know if AI, jobs and API are healthy | Durable status and timestamped freshness; do not invent stage P95 or costs |
| Test Users | Find synthetic user and state | No user impersonation; real-user search deferred |
| Turn Inspector | Find queued/failed turn; see statuses, rule codes and attempts | Durable turn ID and safe projections; separate deliberate, audited synthetic-content drilldown; no raw local Inspector file retrieval |
| Operations | Check memory/reflection workers, queue, leases and safe errors | Durable jobs/worker data; mark unavailable or stale |
| Evaluations | Inspect synthetic golden/test regressions, multilingual cases, blocked evidence | Validate `eval_report.v1` and hashes; no arbitrary commands or unverified content |
| Knowledge Bundles | Inspect current/candidate versions, signatures, reviewed status | Separate signed releases from local draft; no live hot edit |
| Audit | Find staff admin activity and sensitive reads | Append-only sanitized records; no private message text or credentials |

The first **backend vertical slice**: active staff session -> `GET /api/v1/admin/me` -> find failed *synthetic* turn -> read lifecycle/codes and optional explicitly requested synthetic content -> create audit record. The UI is delivered in the frontend phase, not required for initial backend PR completion.

## E. Proposed backend API (contract-first)

- `GET /api/v1/admin/me` - single admin identity, active state and session summary.
- `GET /api/v1/admin/overview?from=&to=` - bounded counts, freshness flags, safe metrics.
- `GET /api/v1/admin/test-users` - cursor-paged synthetic tester metadata.
- `GET /api/v1/admin/turns?status=&from=&to=&cursor=&pageSize=` and `GET /api/v1/admin/turns/{turnId}` - durable turn diagnosis and safe codes.
- `GET /api/v1/admin/turns/{turnId}/synthetic-content` - **only if needed**: dedicated no-cache, deliberate synthetic-only audited inspection, server verifies fixture/test marker (not an arbitrary client flag). Do not use this endpoint for real-user data or read the local inspector directory.
- `GET /api/v1/admin/jobs`, `GET /api/v1/admin/workers` - statuses, retries, freshness.
- `GET /api/v1/admin/evaluations`, `GET /api/v1/admin/evaluations/{runId}` - validated sanitized artifacts, failed vs blocked distinct.
- `GET /api/v1/admin/bundles` - signed/current/draft classification + provenance.
- `GET /api/v1/admin/audit` - page and filter staff audit.

Use existing `contracts/openapi/openapi.yaml` as authored contract, app handlers with typed request/response/Problem Details, fixed `requestId`, strict response field allowlists, bounded time windows and cursor pagination. Route mount must be independent of `AllowDevLogin`; all admin paths use `RequireStaff` before application calls. Never accept user ID or role from a client as an authorization claim.

## F. Revised BO tickets (backend-first, frontend later)

| Ticket | What to implement | Dependency / done condition |
|---|---|---|
| BO-00 | Admin ADR; update stale scope/README references; fixed admin/no-RBAC, synthetic data classification, no-impersonation and safe reads; reference Auth v3 | AUTH-00 decisions; no duplicate Auth implementation |
| BO-01 | Create Admin module and OpenAPI admin endpoints, wire guarded routes through new composition | AUTH-02; contract tests show fail-closed when Auth disabled |
| BO-02 | **Integrate** `RequireStaff` + active Staff checks; no Keycloak client, no SQL session and no staff identity duplication | AUTH-04; wrong realm, dev session, inactive staff denied; active admin allowed all implemented BO features |
| BO-03 | Append-only `admin_audit_events`, cross-table correlation and sensitive read auditing | BO-02; no message/phone/token leakage in records |
| BO-04 | Enhance existing logging/trace interfaces with *separate durable sanitized telemetry* and per-attempt LLM usage idempotency, worker health, unknown cost fields | Existing `loggingTracer` reused; stage metrics and usage survive restarts; no local payload JSON in admin |
| BO-05 | Overview APIs; safe filters, SQL indexes, counts and freshness | BO-04 where needed; no fake/stale claims |
| BO-06 | Turn Inspector/jobs/workers read APIs and optional synthetic-only audited content drilldown | BO-01/03; can inspect failure without direct SQL/AI state mutation |
| BO-07 | **Frontend-only ticket after API freeze:** Admin app shell, login redirect, 4 operational screens, eval/bundles/audit UI | BO-02/05/06/08/09 backend available; test with controlled fixtures |
| BO-08 | Backend eval artifact import/read API; use existing WP-15 sanitized report; include multilingual/new 28-fixture coverage | BO-01/03; hash/source verified, blocked != failed |
| BO-09 | Backend Knowledge Bundle/release manifest metadata endpoints | BO-01/03; local draft != signed reviewed release, no hot editing |
| BO-10 | Backend hardening + frontend E2E when frontend available; JWT/CSRF/SQL/trace privacy/cross-actor tests, backup/runbooks | BO-02..09, frontend login integration and AUTH-09 cutover |
| BO-11 | P1 safety-review cases and decisions (synthetic first), audited | Explicit policy/operational scope; no implicit human emergency response |
| BO-12 | P1 release approvals, test gating and signed bundle pipeline | BO-08/09 and separate sign-off policy |
| BO-13 | P1 privacy case workflows, truthful soft-hidden/erase status | Approved privacy/retention/rights policy; no invented physical erasure |
| BO-14 | P2 privacy-reviewed longitudinal/product quality metrics | BO-04, privacy/analytics scope approval |
| BO-15 | Joint production readiness gates excluding Auth implementation itself | AUTH-10 + BO-10 and privacy/clinical/safety decisions |

**Specific v2 corrections:** the old claim that the AI Core uses a `nullTracer` in its deployed runtime is no longer accurate; the logging tracer is now wired. Preserve the private Local Inspector as a local developer tool. Split BO-07 (UI work) from backend API tickets so backend completion does not depend on the missing frontend repository. Keep a single fixed `admin`, but never bypass audit/explicit use cases.

## G. Backend acceptance gates

1. Active staff via Keycloak can call every implemented Admin endpoint; user realm, dev cookie, unsigned/forged token, inactive staff or expired session cannot; 401 vs 403 correctly defined.
2. Synthetic turn status and job errors reconcile to SQL while any associated content is accessed only via explicit audited request. Neither `local_ai_trace.v1` files nor raw prompt/wire bodies enter admin metrics/HTTP/logs by default.
3. Operational summaries use durable events rather than parsing stdout/stderr; `request_id` is useful for one request, `turn_ref` is not presented as a permanent cross-instance join key.
4. Durable cost reports distinguish actual provider usage, unknown usage, and estimates; no assumed invoice accuracy. Model attempts must remain unique across retry/failover.
5. Eval viewer distinguishes synthetic fixture success, lint, model run, failure and missing/blocked evidence. Draft KB is not called clinically approved.
6. Admin audit is append-only and separate from Auth audit. Privileged reads are logged without payloads.
7. Real-user sensitive content/proactive actions/clinical governance stay outside internal synthetic-only P0; a single admin role does not bypass release safety gates.
8. Repository checks (`check_structure`, `check_contract`, Go build/vet/test), SQL Server integration tests, API contract tests, test fixture seeding, and security/negative tests pass before claiming completed work.

## H. Frontend handoff (after backend work)

- Back-Office frontend is a separate web deployment/repo; use `admin` origin, `/auth/staff/login` redirect, `/auth/staff/session` for session state, same-origin `/api/v1/admin/*` and logout endpoint; do not read Keycloak tokens in JavaScript.
- Screens in order: Admin shell/login/error/expired session -> Overview -> Test Users & Turn Inspector -> Operations -> Evaluations -> Bundles -> Audit. Each screen has loading, empty, stale, server error and offline states with keyboard-accessible tables and bounded filters.
- Content drilldown should say "Synthetic test data" and require a deliberate click, even for a full-access admin, with audit on the server. This is not RBAC.
- Do not build role editor, global transcript search, local inspector file browser, unmanaged memory editing, unsupported crisis monitoring, or raw SQL console.
- Since current upload is backend-only, frontend framework, routes, component conventions and test scripts must be verified against the future frontend repository before coding the UI.

**Sources:** new `docs/ai-core/local-inspector.md`, `docs/observability/LOGGING.md`, `docs/operations/LOGGING.md`, `apps/api/internal/app/composition/{api.go,ai_logging.go,ai_runtime.go}`, `apps/api/internal/platform/observability`, `apps/api/internal/modules/ai/{trace,llm}`, `knowledge/README.md`, `docs/operations/ai-core-wp16.md`, existing `THEIA_BACKOFFICE_IMPLEMENTATION_PLAN_FINAL.md` v2, and shared Auth v3.
