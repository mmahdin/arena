# ADR-0001 — Internal MVP Foundation

- **Status:** superseded in its database choice by ADR-0002; other decisions remain
- **Date:** 2026-11-02
- **Context:** The team wants to verify the core chat and mobile experience, following Yaran Clinic's code organization without inheriting its operational complexity.
- **Decision:** Use Go/Gin + PostgreSQL + separate modular monolith. Dev user selection via signed session cookie is allowed *only* for local synthetic tests. Add OpenAI-compatible provider interface, default mock, and one versioned supportive prompt. Auth and safety hardening are deliberately deferred to real-user release gate.
- **Alternatives:** (1) copy Yaran with SQL Server/Keycloak — over-scoped, (2) public sign-up/OIDC in first slice — unnecessary, (3) all-in-one app service with no modules — undermines future maintainability.
- **Consequences:** Lightweight API, stable user IDs, easier migration to OIDC; current version cannot safely host private real mental-health data.
- **Follow-up:** OpenAPI generation, real credentials, crisis policy validated by professional, privacy retention, tests against SQL Server, automated OpenAPI lint, SSE if UX requires it.
