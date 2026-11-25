# ADR 0013 — Private Back-Office boundary and operational access

- **Status:** accepted for implementation; no Admin API or UI is implemented by this ADR
- **Date:** 2026-11-19
- **Owner:** Back-Office/Admin module
- **Dependencies:** ADR 0012 (Auth and staff sessions)
- **Authoritative implementation plan:** `docs/plans/THEIA_BACKOFFICE_IMPLEMENTATION_PLAN_V3.md`

## Context

The AI Core now has turn state, workers, synthetic evaluations and bundle artifacts that require internal operations and debugging. The original MVP excluded an admin console. The team explicitly approved a private Back-Office with one full-access `admin` role for now, rather than a speculative multi-role permission framework.

## Decision

1. Build a **separately deployed admin web frontend** against `/api/v1/admin/*` in the **existing Go/Gin modular monolith** (`internal/modules/admin`). Do not create an Admin backend microservice, query SQL from the browser, or expose unbounded AI Core ports.
2. All Admin APIs consume the **shared Auth module's** verified staff session and `RequireStaff`, backed by pre-provisioned active `staff_accounts`. A valid user-realm identity, Dev Session, inactive staff account or forged role cannot enter the Admin API.
3. P0 has **one fixed `admin` role with access to every implemented Back-Office capability**. No RBAC/grants table, role editor, organization system or multi-role screen gating. This decision does not create unpublished arbitrary SQL, unsafe AI mutation, account impersonation or clinical-approval endpoints; new operations must first be explicitly implemented and tested.
4. Expose aggregate operational overview, synthetic-test-user lookup, bounded turn/job/worker diagnostics, evaluation history, Knowledge Bundle metadata and admin audit. Use durable safe read projections, not raw debug logs or inferred values.
5. Authorized admins may deliberately inspect **synthetic** test content via a specific, audited, no-cache endpoint after the server verifies test provenance. Do not automatically load transcripts on overview pages, expose local Inspector JSON, or present real-user private content while the product is synthetic-only.
6. Record Admin **action and sensitive-read** audit separately from Auth login/session audit. Keep actor, operation, resource ID, reason, outcome and timestamp; never log content, phone numbers, credentials or tokens.
7. Local development requires loopback-only access; no VPN implementation is needed now. **Before shared team hosting**, use VPN/private ingress and TLS. No-MFA staff auth is a conscious local/private decision, not equivalent to a public security approval.
8. Preserve AI Core risk/consent/memory invariants, signed Knowledge Bundle promotion and soft-delete semantics. Safety review, privacy cases, regulated retention/erasure and real-user deployment remain separately gated P1/production programs.

## Module and data ownership

- **Auth module:** Keycloak clients, OIDC flows, `auth_identities`, `staff_accounts`, `auth_sessions`, `auth_audit_events`, guards and session lifecycle.
- **Admin module:** `/api/v1/admin/*`, admin use cases, explicit sanitized projections, `admin_audit_events`, persisted operational metrics, evaluation/bundle views and later approved governance case tables.
- **AI Core:** authoritative turn/job/risk/memory data and existing owner-scoped operations. Admin reads them through approved application ports or safe SQL projections; no implicit write bypass.
- **SQL migration ownership:** migrations are sequential and checksummed. Assign next number when each PR merges, with Auth tables preceding Admin tables; never reserve the same next migration in parallel branches.

## Alternatives considered

- **Full RBAC in MVP:** deferred because the current staff is the owning team and the requested policy is a single full-access admin role.
- **Reuse Keycloak Admin Console as Back-Office:** rejected; identity administration is not AI operations.
- **Unrestricted transcript/DB browser:** rejected because it bypasses use-case auditing and the synthetic/real-user boundary.

## Verification and release boundaries

First backend acceptance path: authenticated active staff -> `GET /api/v1/admin/me` -> metadata-only failed synthetic turn detail -> audited access. Reject wrong realm and Dev cookies, prove isolation and absence of secrets in error/log output, then add monitoring and evaluation views. No deployment or clinical safety claims arise from a functioning admin console. Use `docs/plans/THEIA_BACKEND_FIRST_IMPLEMENTATION_RUNBOOK.md` for PR order and frontend handoff.
