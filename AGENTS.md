# AGENTS.md — Wellbeing Backend Agent Contract

## Background intelligence amendment (2026-11-24)

ADR 0016 and `docs/ai-core/background-intelligence.md` define the owner-requested
Phase 2. Keep Phase 1 synchronous inference intact. Use the existing queues and
shared Background Intelligence executor, periodic message batches with durable
coverage, explicit scoped analysis, primary-evidence validation, archived revisions
and optional expiring hints. Memory Intelligence owns reported memory; Reflection
owns inferred hypotheses/formulations. No support-count promotion or ProblemTags
framework router in the active executor. Migration 0042 is additive. Preserve the
existing Reflection risk boundary and all Unified Safety logic/instructions/assets.
Use synthetic data only; this does not approve deployment or clinical claims.

## Unified conversation and JEV amendment (2026-11-24)

The owner explicitly requested the integrated architecture in ADR 0015 and
`docs/ai-core/unified-memory-centric.md`. Active runtime is `ai_runtime.v4`,
`wctx.v3`, and `unified_turn.v2`: immediate dual-query retrieval → JEV card
selection → Unified → independent proposal resolution → atomic finalization.
Do not restore Brain/Generator paths or LLM query rewriting. Use existing
Memory/Reflection workers for bounded post-finalization analysis and recent-message
review. Migration 0041 stores immutable owner-scoped decisions and carry-over.
Safety stays inside Unified; existing Safety logic, instructions and policies
are unchanged by this amendment. Historical contracts/assets remain readable.


## Approved next-stage scope: Authentication and private Back-Office (2026-11-19)

The owner has **explicitly authorized** implementing the shared Keycloak/OIDC/SMS
Auth/BFF module and a private Back-Office, superseding the original MVP's
"no OIDC / no Back-Office" exclusion **for these two planned workstreams only**.
This is a design/implementation authorization, **not** a claim that real Auth or
Admin APIs are already present, and **not** approval to handle real mental-health
data or launch publicly. See ADRs `0012` and `0013` and `docs/plans/README.md`.

- **Auth source of truth:** `docs/plans/THEIA_AUTH_IMPLEMENTATION_PLAN_V3.md`.
  Keycloak runs separately with dedicated PostgreSQL; Go Auth/BFF keeps
  revocable application sessions and identity mappings in SQL Server. Users
  log in via SMS OTP; provisioned staff use email/password. Staff and users
  have separate Keycloak realms, callbacks and host-only cookies.
- **Back-Office source of truth:** `docs/plans/THEIA_BACKOFFICE_IMPLEMENTATION_PLAN_V3.md`.
  Its `admin` module consumes the shared Auth guard/session, never duplicates
  login logic. P0 has one fixed **full-access admin** for implemented Admin
  features; no RBAC editor. Explicit, audited use cases still govern sensitive
  data access and operations; local Inspector files stay private/local.
- **Delivery order:** `docs/plans/THEIA_BACKEND_FIRST_IMPLEMENTATION_RUNBOOK.md`.
  Backend contracts and services first; frontend Web/PWA, Admin and TWA next;
  remove Dev Session only after cross-repo cutover. Local/CI may use isolated
  synthetic fixtures and Mock SMS; team-shared hosting requires private VPN
  and TLS. No MFA is required for the current local/private staff flow.
- For these workstreams, historical clauses in `MVP_SCOPE.md`, constitutions,
  and older module/agent guides that prohibit OIDC or Admin are historical.
  **All synthetic-only, privacy, safety, no-clinical-claim and no-public-launch
  restrictions remain binding.**

## Approved onboarding personalization amendment (2026-11-11)

The user explicitly requested using onboarding information throughout chat. This
supersedes older name/style-only and ephemeral-only answer restrictions below for
the internal synthetic build. Persist only explicit allowlisted `onboardingAnswers`
on final Ready via the owner-scoped profile API; no inferred traits, memory extraction
or historical backfill. Keep privacy/Ready copy accurate. Migration 0016 is additive;
prompt version is `supportive-v5`. Preserve existing `MODEL_*`/`LLM_PROVIDER` naming;
staging OpenAI model is explicitly `gpt-5.6-sol`. Never commit API keys.

**Start here.** This repository is an INTERNAL synthetic-data prototype derived from Yaran Clinic architecture. Do **not** expose it on a public interface or use real mental-health disclosures.

## Read in order before changes

1. `MVP_SCOPE.md` — agreed scope, user journey, exclusion list, acceptance criteria.
2. `PRODUCT_CONSTITUTION.md` — safety, privacy and product behavior boundaries.
3. `TECHNICAL_CONSTITUTION.md` — architecture, DB/API and dependency rules.
4. `REPO_MAP.md` — code ownership and entrypoints.
5. Owning module `README.md`; closest HTTP/DB adapter and tests.
6. Relevant `.agents/skills/<skill>/SKILL.md` with supporting `references/`.

## Hard rules

- **Original feature goal:** `Synthetic phone-shaped dev identity -> Onboarding -> Chat -> Saved conversations -> Return`. New Auth and private Back-Office are explicitly approved by ADRs 0012/0013; all other expansion still needs a product decision.
- API contracts are authored in `contracts/openapi/openapi.yaml` first. The current Gin adapters are handwritten starter code, **not generated**; codegen is a future explicit task. Do not claim otherwise.
- Module dependency direction: `domain` <- `application` <- `adapters`; `app/composition` wires implementations. Domain never imports Gin, database/sql or LLM SDK.
- **Current implementation:** `current user` comes from signed local-only Dev Session middleware. **New Auth target:** server-verified BFF session and `RequireUser`/`RequireStaff`; never from a client-provided query/body field. All owner-scoped SQL continues to enforce server-resolved ownership.
- AUTH-09 permanently retired the synthetic Dev Session authentication, even in local/staging. `ALLOW_DEV_LOGIN=true` now fails startup. Use the Keycloak User BFF.
- Use only synthetic data; do not log/store real mental health disclosures. No message bodies, prompts, tokens or secrets in logger, traces or analytics.
- AI prompt is `supportive-v3`; avoid claiming clinical care, diagnosis, human follow-up, emergency availability, or professional credentials. Keyword checks are not sufficient for real-world safety.
- The **original MVP** excluded memory extraction/RAG, Back-Office, OIDC and payments. AI Core is governed by Contract Pack v1.2/ADR 0011; Auth and private Back-Office are now approved under ADRs 0012/0013. Do not treat this as approval for payments or other unrelated features.
- Do not modify existing applied SQL migrations. DB engine is **Microsoft SQL Server 2022**. New migration = next `NNNN_name.sql`, transactional via runner. Use T-SQL and `datetime2(3)`/`SYSUTCDATETIME()`; never use PostgreSQL dialect.
- Read and apply `wellbeing-backend-module` for new backend features, `wellbeing-api-contract` for API changes, `wellbeing-database-migration` for DB changes, `wellbeing-permission-review` for ownership and privacy, `wellbeing-ai-prompt-safety` for model changes, `wellbeing-dev-session` for identity changes.
- Keep frontend and backend deployable separately; no imports from frontend source.
- Never secretly add automatic storage of inferred personality traits or cross-conversation memory.

## Definition of Done per PR

1. Behavior + scope summary, acceptance criteria and module owner.
2. Contract/schema migration with rollback/compatibility notes if touched; never allocate the same next migration number concurrently to Auth and Admin.
3. Domain/application negative tests and cross-user isolation tests if relevant.
4. Sanitized errors and logs; no raw private payload in telemetry.
5. `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py`, `go test ./...` with environment details; no invented passing tests.
6. Update docs and report unfinished validation and risks.

## Safety stop

If a request involves public availability, real personal chat data, diagnosis, automated treatment, or emergency response: **do not treat this MVP foundation as approved**. Identify explicit new privacy/security/clinical review requirements first.

## Hamrah integration decision — identity v2
AUTH-09: User login is Keycloak SMS-only via Go BFF. No synthetic phone-shaped test identifier or `/api/v1/dev/*` exists; do not restore a dev authentication bypass.

## HAMRAH-BACKEND-07: Saved Messages
Read `docs/integration/TASK07_SAVED_MESSAGES.md`. Do not remove the existing Chat menu or redesign the designer UI. Bookmarks use owner-checked message IDs and inherit soft-deleted conversation visibility while retaining all records (ADR 0010); local synthetic-only.

## WP-02 soft-delete correction (2026-11-16)

All user deletion is soft-only. Preserve original and derived data; do not purge,
cascade physical deletes, clear content or run retention cleanup. Conversation
deletion must not invalidate derived memories/analyses or change consent. Normal
queries hide deleted resources. Retention/visibility and AI-use permissions are
separate; no implicit reuse policy is authorized. Permanent/legal erasure is a
separate future policy. Use migration 0020 and ADR 0010; keep prior migrations immutable.

## Theia AI Core Inspector / offline Execution Explorer

When changing AI Core tracing or any producer payload consumed by the local
Inspector, review `apps/api/internal/modules/ai/trace/AGENTS.md`. Keep its
standalone `trace/explorer/index.html`, authoritative JSON Schema and synthetic
fixtures compatible. Check real browser interactions when the viewer changes;
no backend connection or automatic file watcher is authorized. Full-payload
traces remain opt-in, local-only and synthetic/test-data-only.
