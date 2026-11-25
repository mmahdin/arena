# Theia - Backend-First Implementation Runbook

**Companion plans:** `THEIA_AUTH_IMPLEMENTATION_PLAN_V3.md` (identity/session source of truth) and `THEIA_BACKOFFICE_IMPLEMENTATION_PLAN_V3.md` (Admin operations source of truth). **Reviewed backend:** `repo (1).zip` 2026-11-19. **Status:** guide only; source code has not been modified.

## 1. Practical sequence / dependencies

```text
BACKEND REPO FIRST
  PR-00: AUTH-00 + BO-00                Architecture ADRs + repo guidance update
      |
  PR-01: AUTH-01                        Keycloak + PostgreSQL + reproducible realms
      |
  PR-02: AUTH-02                        Decouple routes, AI startup, draft/inspector from Dev Login
      |
  PR-03: AUTH-03                        Go OIDC/BFF, SQL sessions, identity mappings, guards
      |
  PR-04: AUTH-04                      Staff login + controlled local provisioning
  PR-04b: BO-01 + BO-02               Admin API contracts + protected /admin/me
      |
  PR-05: AUTH-05                        Keycloak Java SMS SPI + Mock/Kavenegar adapter
      |
  PR-06: AUTH-06                        User Mock SMS sign-in, UUID mapping, owner guard
      |
  PR-07: BO-03 + thin BO-06             Admin action audit + failed synthetic turn detail API
      |
  PR-08: BO-04 + BO-05 + rest BO-06     Durable observability + overview/jobs/workers APIs
      |
  PR-09: BO-08 + BO-09                  Eval imports and signed/draft bundle metadata APIs
      |
  PR-10: AUTH-07/08 + BO backend tests  Contract freeze, browser test harness, security
      |
FRONTEND REPOS (after backend contract freeze)
  FE-01: existing user web/PWA          Begin login/session; Keycloak theme for phone+OTP
  FE-02: separate admin frontend        Staff login -> Turn Inspector -> dashboard -> etc.
  FE-03: Android TWA                    Origins/assetlinks/real device/login/logout/OTP
      |
CROSS-REPO CUTOVER
  PR-11: AUTH-09                        Remove Dev Session + old frontend selectors,
                                        replace fixtures/smoke, update docs/contracts
  PR-12: BO-10                          Final local synthetic-only integrated E2E
      |
FUTURE (not part of P0)
  AUTH-10: VPN/shared-hosting and real-user release gates
  BO-11..BO-15: safety review, governance, privacy, analytics and release gates
```

**Parallelism:** PR-05/06 may overlap BO-03..09 after PR-04b. Do not force all BO backend work to wait for SMS OTP. Contracts and E2E test harnesses can be done before any product UI. **Dev Session removal cannot be completed entirely during backend-only work if the existing frontend still depends on `/api/v1/dev/identity-session`.** Coordinate the final cutover with FE-01/FE-02.

## 2. First developer iteration - exactly what to do

### PR-00 (docs/decisions; zero runtime behavior changes)

- Read `AGENTS.md`, `MVP_SCOPE.md`, `PRODUCT_CONSTITUTION.md`, `TECHNICAL_CONSTITUTION.md`, `REPO_MAP.md` and modules' READMEs; apply the local `.agents/skills/wellbeing-*` guidelines.
- Add new ADR for approved Keycloak + Postgres / two realms / SMS, BFF, separate app/staff sessions and fixed full-access admin for existing Back-Office features; add separate admin ADR or section with data boundaries.
- Explicitly supersede stale `AGENTS.md`/README statements that OIDC, back-office or production identity are forbidden in the original synthetic MVP. Keep restriction on **real user mental-health content** until separately approved.
- Mark `THEIA_AUTH_IMPLEMENTATION_PLAN_V3.md` as owner for Auth, and `THEIA_BACKOFFICE_IMPLEMENTATION_PLAN_V3.md` as owner for Admin. Set new-migration ownership policy and local/CI/production environment table.
- Verify `python3 scripts/check_structure.py` and `python3 scripts/check_contract.py` stay green.

**Done:** reviewers agree on scope and no ambiguous auth owner/parallel session implementation remains.

### PR-01 (Keycloak runtime and local development)

- Add explicit Keycloak/PostgreSQL services and health checks; pin Keycloak version and config, persist IdP DB, keep external ports loopback only. Preserve existing SQL Server. Prefer an opt-in Compose profile so old DB/API workflows are not accidentally broken.
- Define two realms, two confidential OIDC BFF clients, exact redirect URIs, no wildcard hosts or self-signup for staff. SMS realm can start with a test-only authentication setup until the SPI is ready; do not claim production phone auth works.
- Add safe env template and secret provisioning notes. Document what frontend/test harness will see; do not commit credentials.
- Test realm boot/import and failure/restart behavior. Do not deploy with public IdP admin console.

**Done:** Keycloak, PostgreSQL, realms and clients can be started/recreated deterministically in local test environment.

### PR-02 (THE blocking backend refactor)

- In `apps/api/internal/app/composition/api.go`: remove `if !cfg.AllowDevLogin { return api, nil }` from the *real* API mounting path; dev-only routes should be an optional isolated group during transition.
- Decouple `loadProduct(...)` in composition from `AllowDevLogin`; AI runtime and `/health/ready` should depend on actual configured/loaded core and its health, not synthetic login flag.
- In `internal/platform/config/config.go` and composition: change `AI_CORE_LOCAL_DRAFT` gating to explicit local + loopback + synthetic-only + runtime-config checks **without requiring Dev Login**. Keep local inspector strict local+loopback+private directory, even in post-Dev-Session setup.
- Define trusted reverse-proxy host/routing/cookie model and add exact Auth BFF paths to OpenAPI. Keep all existing `/me` and conversation contracts unchanged while refactoring composition.
- Add combination tests: dev flag on/off; AI config present/missing; local draft permitted/denied; inspector enabled/denied; user/staff route mount independent of dev mode.

**Done:** users and AI Core no longer disappear when Dev Login is off; rejected unsafe configurations remain rejected; tests do not require a live auth provider yet.

### PR-03 onward (small vertical PRs)

- PR-03: Auth tables, OIDC transaction/issuer/PKCE/nonce/host checks, session hash/cookie/CSRF, `RequireUser/RequireStaff`, lifecycle and auth audit; disposable SQL integration and negative tests.
- PR-04: Provision one staff, staff login roundtrip and guarded `GET /api/v1/admin/me`; a user-realm identity or dev cookie must not pass. No RBAC matrix.
- PR-05: Keycloak SMS Java SPI + MockSmsSender + KavenegarSender; tests verify challenge atomicity, cooldowns, provider failure and restart behavior.
- PR-06: signed-in test user gets one immutable Theia UUID, onboarding works and all owner-scoped APIs reject other-user data.
- PR-07: admin audit + read-only synthetic turn vertical slice; no local trace file browser or user impersonation.
- PR-08/09: durable safe telemetry/status APIs and validated Eval/Bundle APIs; no fabricated dashboard metrics or unreviewed bundle promotion.
- PR-10: freeze OpenAPI, errors, session lifecycle and frontend URL contract; do browser-based E2E with disposable IdP+DB and minimal harness, without waiting for frontend styling.

## 3. Repository conventions & test commands

From the backend repo root:

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v
(cd apps/api && go build ./...)
(cd apps/api && go vet ./...)
(cd apps/api && go test ./...)
```

For integration, use disposable SQL Server and Keycloak/PostgreSQL; follow repository setup (`compose.yaml`, `docs/operations/local-development.md`, `docs/operations/ai-core-wp16.md`). Use `python3 scripts/new_migration.py <description>` for each SQL Server migration, allocating numbers in actual merge order; leave `0001`-`0023` unchanged. Use `go test ./tests/integration` only once the DB/IdP fixtures are ready. **Review environment:** Python structure/contract checks passed; Go tests could not fetch modules due network restrictions and remain unverified.

## 4. Frontend phase: required API handoff and sequence

**The current upload contains backend only.** Do not assume a frontend framework/router or existing login components; inspect the frontend repo before implementation.

1. **User web/PWA**: replace fake-phone Dev identity selection with `/auth/user/login` redirect. Keycloak theme owns the mobile entry and OTP screen; the app owns entry point, loading/error/expired session, session fetch and onboarding redirect. Never keep JWT/Keycloak codes in browser app storage. Use cookies/CSRF via same-origin proxy.
2. **Staff web app**: separate UI origin; `/auth/staff/login` -> verified staff Session; use new `/api/v1/admin/*` read models. Build screens progressively: login/session/expired -> failed turn -> overview/jobs -> evaluation/bundle -> audit. No role management UI in MVP.
3. **TWA**: because it uses browser cookies, the Web login flow should remain consistent. Configure HTTPS and `assetlinks.json`; if Keycloak origin needs full-screen TWA, add verified multiple origins. Validate on physical Android device and with alternate default browser.
4. **Cross-repo E2E & cutover**: update existing frontend to stop calling `/api/v1/dev/*`; run Owner / Cross-Realm / OTP / Session / Logout smoke; **then** remove Dev Session routes/code/config and tests or migrate them to fixture harness, and update OpenAPI. Keep feature flags/rollback plan bounded and fail closed in every deployed environment.

### Frontend contract handoff checklist

- Exact app/admin/auth origins and redirect URL allowlists; callback routes.
- `/auth/user|staff/session`, `/auth/*/login`, `/auth/*/logout` fields and error/status definitions.
- OTP theme design requirements: E.164 normalization for Iranian phones, Persian/Latin digits, resend/expiry/errors, WebOTP origin-binding and manual fallback.
- 401 vs 403 vs 503 behavior, CSRF/header contract, credentialed same-origin calls, session expiry and inactive staff handling.
- Admin OpenAPI `me`, turns, jobs, telemetry, evaluations, bundles and audit schemas; fixture samples and data freshness semantics.
- TWA origin/assetlinks and offline/installation behavior.

## 5. Major risks / release gating

- **AI bootstrap and Dev Login coupling** is the main technical blocker. Fix before any cutover.
- **Local Inspector is privacy-sensitive** and must stay local private-files-only. It is not a production admin monitor.
- **Backend-first does NOT mean early Dev Session deletion**: frontend and CI migration must finish first.
- **Keycloak SMS SPI** is custom security-sensitive code: review/restart/atomicity and anti-abuse tests are mandatory.
- **Single full-access admin** is a product decision for the internal prototype; admin operations still use audit and deliberate sensitive views, and public real-user deployment is not approved. VPN is required before shared staff hosting in no-MFA configuration.
- **No published frontend repository is available in this task**. The frontend sequence is a contract-level plan until the actual repo is inspected.

## 6. Ready-to-paste backend coding agent starter prompt

> Work only on the Theia backend repository. Read AGENTS.md, PRODUCT_CONSTITUTION.md, TECHNICAL_CONSTITUTION.md, REPO_MAP.md and relevant local `.agents/skills` first. Treat `THEIA_AUTH_IMPLEMENTATION_PLAN_V3.md` as identity/session source of truth, and `THEIA_BACKOFFICE_IMPLEMENTATION_PLAN_V3.md` as Admin source of truth. Start with **PR-00 only**: author ADRs and reconcile stale no-OIDC/no-back-office repository guidance with the user's approved plan, while preserving synthetic-only and privacy/clinical release gates. Do not write runtime Auth or Admin code in this PR. List changed files and run structure and contract checks; report exactly which tests ran. Stop after PR-00 for review.

After PR-00 is merged, replace **PR-00 only** with the next PR from section 1, keeping each change reviewable and tested. Never combine Keycloak infrastructure, custom OTP, the new session system, and the whole dashboard into one PR.
