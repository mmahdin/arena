# BO-10 — Backend Security & Integration Hardening

**Scope:** backend-only guard/audit/privacy hardening for the existing AUTH-03/04
and BO-01..09 API. This PR contains no frontend, SMS authenticator, IdP logout,
production release permission, new role, or database migration. The deployment
remains an internal **synthetic-data-only** system; admin privileges do not
authorize real mental-health content, impersonation, or unrestricted SQL.

## Security invariants to preserve

1. The entire `/api/v1/admin/*` group inherits the same `RequireStaff` BFF
   middleware. `AUTH_BFF_ENABLED=false` fails all Admin endpoints with
   `503 AUTH_NOT_CONFIGURED`, irrespective of `ALLOW_DEV_LOGIN` or spoofed
   `X-Staff-ID`/`Authorization` headers. Anonymous, user-realm, synthetic dev,
   expired and revoked sessions cannot access Admin. The exact Admin host and
   scheme must match the configured Staff origin.
2. A current, active **pre-provisioned** SQL Staff identity is revalidated on
   every authenticated request. All implemented Admin features use the same
   single fixed `admin` role; no permission matrix or additional login is added.
3. Every privileged **read** and synthetic evaluation **import** appends an
   allowlisted `admin_audit_events` record *before* accessing its data. Its
   actor UUID comes from verified Staff, request ID from the server middleware.
   An audit-write failure returns `503 ADMIN_AUDIT_UNAVAILABLE`, not results.
   No request content, OIDC subjects, cookies, user phone, prompt, or SQL error
   text belongs in this event.
4. The sole Admin mutation, `POST /evaluations/import`, requires Staff session,
   session-bound `X-CSRF-Token`, matching browser `Origin` and a strictly
   validated synthetic `eval_report.v1` + SHA-256 of the exact uploaded bytes.
   The hash is integrity evidence, **not** runner attestation or clinical signoff.
5. The Inspector/Overview expose bounded **metadata-only** projections. The
   reader now converts malformed historical/direct-SQL labels into `unknown`
   or `unclassified` before serialization. `SafeCode` is a defensive *syntax*
   filter, not a guarantee against intentional code-shaped private values;
   deploy controls must still prohibit putting PII in telemetry columns.
6. OIDC callbacks accept exactly one `state` and one `code`, reject any `error`
   parameter, require their single-use login cookie and still verify PKCE,
   nonce, issuer, audience and signed token. The existing user BFF runtime
   remains disabled until the SMS-only Keycloak login path is integrated.

## Automated checks

From the repository root (normal local Go module cache required):

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v
(cd apps/api && go build ./...)
(cd apps/api && go vet ./...)
(cd apps/api && go test ./...)
(cd apps/api && go test -race ./internal/modules/admin/... ./internal/modules/auth/adapters/http ./internal/app/composition)
```

The BO-10 additions include:

- `composition/admin_security_test.go`: enumerate the *actual* Gin Admin
  routes and reject every route when BFF is disabled, with Dev Login on/off.
  Adding an Admin endpoint requires updating this explicit release gate.
- `admin/adapters/http/security_matrix_test.go`: exercise all ten Admin GET
  paths with the real Auth guard + in-memory stores: anonymous, user, dev,
  wrong host, active and inactive Staff, audit outage and request IDs. Test
  that import cannot bypass Origin/CSRF or audit. No test invokes an LLM.
- `admin/domain/inspector_security_test.go`: allowed/unsafe metadata labels.
- `auth/adapters/http/callback_security_test.go`: repeated OIDC state/code,
  unexpected error parameter; no session is issued on rejected callbacks.
- `scripts/tests/test_bo10_admin_security_contract.py`: all 11 authored Admin
  operations declare `staffSession`, 401/503 error responses, guarded
  composition registration and lack of raw private columns in operational
  SELECT projections. These are regression checks, not formal proofs.

### Disposable SQL Server follow-up (not run automatically)

Use a **fresh, disposable SQL Server 2022 database**, not your normal
staging/production database. Apply all checked migrations `0001` through
`0030` via the checksummed runner. Record the `schema_migrations` entries
and verify the application boots with telemetry (0027), overview (0028),
inspector (0029), evaluation history (0030), and admin audit (0026).
Do not edit migrations to make tests pass. Run these tests using the explicit
safety flags documented in their respective runbooks:

- `TestAuth04StaffProvisioningSQL` — deactivation and non-resurrection of sessions
- `TestAuditSQLDisposableDatabase` — append-only audit, inactive Staff
- `TestBO04DurableTelemetrySQL` — attempt idempotency, unknown usage, worker state
- `TestBO05OverviewDisposableSQL` — real aggregation, null/partial counts
- `TestBO06InspectorDisposableSQL` — pagination, redaction and stale heartbeats
- `TestBO08EvaluationsDisposableSQL` — strict imported evaluation history

The BO-06 disposable SQL test now additionally inserts invalid historical
labels in `ai_stage_events.stage` and `ai_llm_attempts.model` and asserts the
Admin reader returns `unclassified` instead. Do **not** run tests that insert
fixtures or call cleanup SQL against persistent or shared data.

### Real Keycloak + browser + host routing follow-up

For a trusted *local synthetic* environment, with Staff BFF and SQL ready:

1. Provision an active test Staff account through `staffctl`; open the
   Admin origin, follow `/auth/staff/login`, confirm Keycloak realm
   `theia-staff`, return to `/auth/staff/session` and check fixed `admin`.
   Confirm login with a user-realm identity or a Dev Session cookie fails.
2. Load every GET Admin endpoint. Check `Cache-Control: no-store`, an
   `X-Request-ID`, correct audit `action_code`, matching request ID and Staff
   UUID. Verify no response contains private synthetic fixture text, prompt,
   raw trace, backend filesystem path or authorization secret.
3. Retry an Admin request after Staff deactivation and after app logout;
   it must return 401. Logout itself requires session CSRF token. Re-activation
   must not resurrect a revoked browser session.
4. With a valid Staff session, call evaluation import with missing/wrong
   `Origin`, missing CSRF token, wrong report hash and invalid extra keys;
   none may produce a new Admin evaluation record. A valid synthetic report
   may import only after an audit append.
5. Stop/deny the audit database write in a **disposable** environment. All
   Admin reads and import must return 503 and no private data or raw SQL error.
6. Test reverse-proxy `Host`/scheme configuration and `TrustedProxyCIDRs`
   with your actual proxy. These gates are not satisfied by a fake Gin router.

**Infrastructure backup gate:** before any deployment to a persistent test
instance, have the operations owner demonstrate a restorable SQL Server backup
and a separate Keycloak PostgreSQL backup in disposable infrastructure. The
reviewer must record restore commands, recovery outcome and recovery time in
the deployment runbook. No backup/restore was performed by this patch.

## Frontend and real-data release blockers

- BO-07 Staff web frontend and real-browser E2E are **not included** here;
  this backend patch is not evidence that the Admin UI works.
- AUTH-05/06 SMS authenticator and user frontend integration are outstanding.
  AUTH-08 Keycloak-wide/backchannel logout and key rotation also remain
  release gates; deleting Dev Session is AUTH-09 after frontend cutover.
- Shared Staff hosting needs private/VPN ingress, reviewed HTTPS/cookies,
  proxy configuration and explicit security testing. The no-MFA MVP decision
  does not authorize a publicly reachable Staff Login.
- No real clinical data, proactive clinical response, safety-review
  decisions or medical deployment is authorized by these tests.

## Deployment and rollback

No schema change: deploy this binary only after existing migrations through
`0030` are present. It is compatible with earlier synthetic records; it
preserves old audit and telemetry tables and does not rewrite historical
labels. If behavior regresses, roll back the application binary. Do not
remove prior migrations/data. Continue using metadata-only endpoints rather
than exposing the full local Inspector or writing an unrestricted SQL viewer.
