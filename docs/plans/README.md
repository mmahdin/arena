# Approved next-stage implementation plans (not implemented)

Read these in order:

1. [`THEIA_AUTH_IMPLEMENTATION_PLAN_V3.md`](THEIA_AUTH_IMPLEMENTATION_PLAN_V3.md) — **source of truth for identity, Keycloak, SMS OTP, staff login, OIDC/BFF, application sessions, guards and Dev Session retirement**. See ADR 0012.
2. [`THEIA_BACKOFFICE_IMPLEMENTATION_PLAN_V3.md`](THEIA_BACKOFFICE_IMPLEMENTATION_PLAN_V3.md) — **source of truth for Admin API/operations, staff-facing screens, audit, AI monitoring and evaluation**. Depends on the shared Auth capabilities. See ADR 0013.
3. [`THEIA_BACKEND_FIRST_IMPLEMENTATION_RUNBOOK.md`](THEIA_BACKEND_FIRST_IMPLEMENTATION_RUNBOOK.md) — ordered PR/checklist, backend-first execution, frontend handoff and cross-repository cutover.

**Status:** AUTH-01 provides opt-in *local-only Keycloak/PostgreSQL infrastructure*. AUTH-02 separates AI startup and protected API routing from Dev Login, but reserves login endpoints as **503-only** placeholders. AUTH-03 adds opt-in OIDC/BFF foundations, identity mapping, SQL sessions and user/staff guards; the staff login feature flag remains off by default but AUTH-04 adds a local-only staff provisioning CLI and permits the operator to turn it on explicitly. User SMS login, Admin APIs and UI remain unimplemented. See `docs/operations/keycloak-local.md`, `docs/operations/auth-composition-transition.md`, `docs/operations/auth-bff-local.md` and `docs/operations/staff-auth-local.md`.

**Conflict rule:** ADR 0012 and the Auth plan govern all login/identity/session decisions; ADR 0013 and the Back-Office plan govern administrative features. Current OpenAPI, code and migrations still document what is implemented **today**. The original 2026-11-02 `MVP_SCOPE.md` describes the historical synthetic-only prototype; new ADRs explicitly authorize the next implementation phase without authorizing real-user mental-health deployment.

**Scope:** Development and CI use synthetic fixtures and isolated Mock SMS. Shared staff deployment requires private VPN/TLS controls; full real-user deployment remains subject to separate clinical/privacy/security/legal release gates.

**Migration note:** Allocate SQL Server migration numbers on the integration branch in merge order. The current repository head is `0023_account_soft_delete.sql`; do not reserve `0024` independently in both Auth and Admin branches. Do not edit historical migrations.
