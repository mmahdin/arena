# AGENTS.md — Wellbeing Backend Agent Contract

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

- **Feature goal:** `Synthetic phone-shaped dev identity -> Onboarding -> Chat -> Saved conversations -> Return`. Reject scope expansion without an explicit product decision.
- API contracts are authored in `contracts/openapi/openapi.yaml` first. The current Gin adapters are handwritten starter code, **not generated**; codegen is a future explicit task. Do not claim otherwise.
- Module dependency direction: `domain` <- `application` <- `adapters`; `app/composition` wires implementations. Domain never imports Gin, database/sql or LLM SDK.
- `current user` comes from signed local-only session middleware, **never** from a client-provided query/body field. All conversation/messages SQL must filter by server-resolved user ownership.
- `ALLOW_DEV_LOGIN=true` is allowed only in `local`, `stage`, `staging` or `staging-v2` for synthetic tester sessions. Staging support is explicitly approved for deployment configuration compatibility; it is not real authentication or approval for real user data. Production and unknown environments must reject dev login.
- Use only synthetic data; do not log/store real mental health disclosures. No message bodies, prompts, tokens or secrets in logger, traces or analytics.
- AI prompt is `supportive-v3`; avoid claiming clinical care, diagnosis, human follow-up, emergency availability, or professional credentials. Keyword checks are not sufficient for real-world safety.
- No memory extraction, RAG, back office, OIDC or payments in MVP.
- Do not modify existing applied SQL migrations. DB engine is **Microsoft SQL Server 2022**. New migration = next `NNNN_name.sql`, transactional via runner. Use T-SQL and `datetime2(3)`/`SYSUTCDATETIME()`; never use PostgreSQL dialect.
- Read and apply `wellbeing-backend-module` for new backend features, `wellbeing-api-contract` for API changes, `wellbeing-database-migration` for DB changes, `wellbeing-permission-review` for ownership and privacy, `wellbeing-ai-prompt-safety` for model changes, `wellbeing-dev-session` for identity changes.
- Keep frontend and backend deployable separately; no imports from frontend source.
- Never secretly add automatic storage of inferred personality traits or cross-conversation memory.

## Definition of Done per PR

1. Behavior + scope summary, acceptance criteria and module owner.
2. Contract/schema migration with rollback/compatibility notes if touched.
3. Domain/application negative tests and cross-user isolation tests if relevant.
4. Sanitized errors and logs; no raw private payload in telemetry.
5. `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py`, `go test ./...` with environment details; no invented passing tests.
6. Update docs and report unfinished validation and risks.

## Safety stop

If a request involves public availability, real personal chat data, diagnosis, automated treatment, or emergency response: **do not treat this MVP foundation as approved**. Identify explicit new privacy/security/clinical review requirements first.

## Hamrah integration decision — identity v2
The approved local UI enters a **fictional phone-shaped test identifier**, not a seeded-user dropdown. `POST /api/v1/dev/identity-session` find-or-creates a test user and cookie; this is not OTP authentication. Accept `09xxxxxxxxx` phone-shaped identifiers for local synthetic testing only. This is not mobile authentication; do not use personal phone numbers. Preserve frontend design; legacy seeded-list routes remain only to keep previous automated tests working. See `apps/api/internal/modules/devsession/README.md`.

## HAMRAH-BACKEND-07: Saved Messages
Read `docs/integration/TASK07_SAVED_MESSAGES.md`. Do not remove the existing Chat menu or redesign the designer UI. Bookmarks use owner-checked message IDs and cascade with deleted conversations; local synthetic-only.
