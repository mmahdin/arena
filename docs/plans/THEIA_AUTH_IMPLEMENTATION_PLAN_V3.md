# Theia Authentication - Implementation Plan v3 (updated backend)

**Status:** Approved design; AUTH-01 local Keycloak infrastructure implemented, runtime Auth and Admin not yet implemented. **Reviewed repo:** `repo (1).zip`, uploaded 2026-11-19; 630 files versus 520 in the previous archive. **Authority:** This is the source of truth for login, identity, authentication guards, OIDC, sessions, staff provisioning, SMS OTP, TWA authentication, and developer-session retirement. The Back-Office plan consumes these capabilities; it must not recreate them.

## A. Locked decisions

| Concern | Decision |
|---|---|
| Identity provider | Pinned, separately deployed Keycloak; one installation, two realms `theia-users` and `theia-staff` |
| IdP storage | Dedicated PostgreSQL for Keycloak, separate from existing SQL Server 2022 |
| App (browser + Android TWA) | Mobile phone login/sign-up via SMS OTP; no password |
| Staff | Email + password, accounts provisioned by team, no public registration and no MFA for current private/local scope |
| Staff role | Single fixed `admin` with every *implemented* Back-Office capability; no RBAC editor/matrix in P0 |
| External SMS | Kavenegar via replaceable `SmsSender` adapter; Mock SMS in isolated local/CI |
| OTP owner | Keycloak Authentication SPI; Keycloak generates/verifies/consumes challenge; SMS vendor only delivers |
| App session owner | Go BFF (`internal/modules/auth`), opaque HTTP cookie; sessions + mapping in SQL Server |
| IdP browser session | Keycloak, using its PostgreSQL-backed persistence |
| Local deployment | Loopback-only app/API/Keycloak test surfaces; future shared internal Back-Office through VPN/private ingress |
| User app and staff UI | Separate hosts/origins and callbacks; same backend code, no shared user/staff cookie |
| Dev session | Keep only while migrating; remove route/middleware/config dependency at cross-repo cutover; never allow it in production |
| AI product | Synthetic-only until independent clinical, privacy and real-user readiness gates pass |

## B. What changed in the newest backend

1. **No production Auth has arrived:** `apps/api/internal/app/composition/api.go` still uses `devsecurity.Guard`; no Keycloak/OIDC/Auth module. The `contracts/openapi/openapi.yaml` remains `0.3.0`; migration head remains `0023_account_soft_delete.sql`.
2. **New AI setup coupling to remove in AUTH-02:** `NewAPI` returns after health when `!AllowDevLogin`; it also runs `loadProduct(...)` only when `AllowDevLogin && AICoreConfigFile != ""`. Separately, `config.Load` and `NewAPI` require `AllowDevLogin` for `AI_CORE_LOCAL_DRAFT`. The old plan identified only the first problem. Decouple product runtime loading and the local synthetic-draft gate before retiring Dev Login.
3. **New Local AI Core Inspector:** `docs/ai-core/local-inspector.md`, `internal/modules/ai/trace/inspector.go`, schema `local_ai_trace.v1`. Preserve local-only + loopback gate and private full-payload file policy after removing Dev Session; DO NOT turn on inspector in staging/production, serve it over an admin endpoint, or mix it into operational logs.
4. **New diagnostics:** `internal/platform/observability`, `internal/platform/httpx/logging.go`, `internal/app/composition/ai_logging.go` have safe request IDs, turn refs and metadata-only timing. Auth must keep request/correlation IDs but never log phone numbers, OTP, cookies, auth headers, tokens or arbitrary provider bodies.
5. **New development KB:** `knowledge/` is a synthetic-only, unsigned but substantive draft with multilingual tests. Removing Dev Login must NOT unintentionally enable it on shared listeners. Local draft needs a separate explicit local+loopback+synthetic-only gate.
6. **Build validation reviewed:** `python3 scripts/check_structure.py` and `python3 scripts/check_contract.py` passed on the new snapshot; Go tests could NOT run in the review environment because dependencies could not be fetched from proxy.golang.org. CI/local Go tests are still required.
7. **Governance documentation:** `AGENTS.md`, `README.md` and `.agents/skills/wellbeing-dev-session` still describe the synthetic-only Dev Login / no OIDC MVP. AUTH-00 must record the explicit new product decision in an ADR and update those instructions and any scope documents before implementing the new architecture. Do not silently contradict old guidance.

## C. Architecture & ownership

```text
Web/PWA/TWA app origin                    Separate Back-Office origin
  /auth/user/*   /api/v1/*                 /auth/staff/*   /api/v1/admin/*
            |                                   |
       HTTPS / same-origin BFF reverse-proxy + strict host/route binding
            |                 |
          Go/Gin Modular Monolith (Auth/BFF module in Go)
             |     OIDC Authorization Code + PKCE
             v
      Keycloak service (users realm + staff realm)
             |                 |
       PostgreSQL (IdP)    Custom SMS Authenticator (Java SPI)
                                    | SmsSender: Mock or Kavenegar

Go BFF -> SQL Server: auth_identities, staff_accounts, auth_sessions,
                       auth_login_transactions (if server-side state store),
                       auth_audit_events + existing users/profiles/AI data
```

**Auth owns:** Keycloak deployment, verified staff provisioning, OIDC client and redirect validation, login transaction state, signed-token validation, OTP SPI, SMS adapters, user identity resolution, session lifecycle, CSRF, `RequireUser`/`RequireStaff`, auth audit and Dev Session retirement.

**Back-Office owns:** `/api/v1/admin/*` contracts, administrative application services and data projections, admin UI, operational audit, telemetry/evaluation/bundle views. **AI Core** receives authenticated internal user UUID, not Keycloak tokens.

Prefer dedicated BFF endpoints (`/auth/user/login`, `/auth/user/callback`, `/auth/user/session`, `/auth/user/logout`; equivalent `/auth/staff/*`) and bind them to their expected external origin. A shared single Gin process must validate both the expected `Host`/forwarded host (trusted proxy only) and the expected realm/client; never permit a user callback on the staff host. Fixed exact redirect URI allowlists, no wildcard. The local approach may use fixed loopback ports with different cookie names; use TLS development hostnames before testing Secure cookies and TWA.

## D. Auth tables & session security

- `auth_identities`: `(issuer, sub)` unique, `principal_kind = user|staff`, immutable internal UUID; no phone/email as primary key. User UUID points to existing `dbo.users.id` and staff UUID to `staff_accounts.id`. Verify uniqueness on first-login races.
- `staff_accounts`: provisioned and activatable staff UUID, `active`, timestamps; fixed `admin` capability in application policy; no auto-create-on-login.
- `auth_sessions`: >=256-bit random bearer cookie; database stores hashed verifier (and optionally lookup selector), principal UUID/kind, issuer/sub, Keycloak `sid` if available, idle/absolute expiry, last seen, revocation and minimal device metadata. Independent user vs staff cookie names and host scope.
- `auth_login_transactions` (optional but recommended): short-lived, one-time `state`, `nonce`, PKCE verifier, trusted return target and intended realm/client/host; use TTL and atomic consume; alternatively prove equivalent replay-safe sealed cookie mechanism.
- `auth_audit_events`: sanitized append-only login/logout, failed/revoked session, staff provisioning. Back-Office creates separate `admin_audit_events` for privileged reads/actions.
- Keycloak Postgres migrations are owned by Keycloak, not by the Go SQL Server migrator. SQL Server migration numbers allocated sequentially on the integration branch, starting after `0023`; no duplicate `0024` between teams; never edit applied migrations.
- App cookie: `__Host-theia_user_session`; staff: `__Host-theia_staff_session`, each `HttpOnly; Secure; SameSite=Lax; Path=/` with no `Domain` in HTTPS environments. For loopback HTTP dev, document browser handling and never weaken nonlocal settings.
- Initial policy: user 7-day idle/30-day absolute; staff 30-minute idle/12-hour absolute; both revocable. Fail closed on SQL outages. Protect unsafe methods with Origin validation plus session-bound CSRF tokens and proper CSP/CORS.
- Avoid long-lived Keycloak refresh token storage by default; if needed, document the use case, encrypt server-side and rotate keys. Ensure local session revocation on Keycloak logout, disablement and backchannel events, with tested failure/recovery behavior and `sid` fallback.

## E. Required user/staff flows

**User (web/PWA/TWA):** app login action -> same-origin BFF authorization transaction -> redirect `theia-users` -> Keycloak-hosted phone+OTP theme -> normalize Iran mobile to canonical E.164 -> rate-limit -> create secure OTP challenge -> Mock/Kavenegar delivery -> user input -> atomic single-use verification -> resolve/create Keycloak identity -> OIDC code callback to Go BFF -> validate state/PKCE/nonce/issuer/audience -> resolve/create `auth_identities` and internal UUID transactionally -> SQL session -> host-only cookie -> existing `/api/v1/me`, onboarding/home.

**Staff:** admin login action -> BFF transaction -> `theia-staff` email/password -> code callback -> validate claims -> find pre-provisioned active `staff_accounts` -> SQL admin session + staff cookie -> `RequireStaff` -> `/api/v1/admin/me` -> Back-Office. Staff and user realms/cookies are mutually rejected. No public staff signup; first admin provisioning via a controlled setup command/runbook.

**OTP baseline:** 6 digits, 2-minute lifetime, <=5 guesses per challenge, >=60-second resend delay, invalidate prior challenge on resend, multi-dimensional per-phone/IP limits + SMS cost caps, constant-time verifier, CSPRNG, no OTP/content logs. Prove concurrency, resend, provider-accepted-but-not-delivered, Keycloak restart and provider-outage behavior. Provider should only transport the OTP. WebOTP format tied to Keycloak OTP page origin if vendor/template supports it; manual entry always works. SMS is phishing/SIM-swap susceptible; before real-user launch define number change, recovery, forced re-auth for sensitive operations and recycled-number/account takeover mitigation.

**Android TWA:** same browser-origin Session, no Android-native JWT store. Validate callback, login relaunch, logout, clearing cookies, Chrome/default browser changes, site verification and Digital Asset Links, optional Keycloak multi-origin verification. Device install/uninstall is NOT server-side logout. Physical device HTTPS smoke tests before TWA release.

## F. Test identity strategy after Dev Session removal

Use seeded **synthetic fixtures** for users, accounts, turns, AI data and worker scenarios; `MockSmsSender` in local/CI with an internal test outbox/test harness. Tests MUST exercise the real user OIDC path and SQL session; never provide `/dev/session`-like bypass in deployed application. Fake IdP for unit tests only; disposable Keycloak/Postgres + SQL Server for integration/E2E. Mock SMS transport not reachable via production routes/images. For local KB draft tests, keep local/loopback-only + explicitly `synthetic_only`, independently of `AllowDevLogin`. Optional Local Inspector remains local and does not become a replacement login or Admin API.

## G. Backend-first delivery tickets (revised)

| Ticket | Backend deliverable / dependencies | Done when |
|---|---|---|
| AUTH-00 | ADR, threat model, scope amendment to `AGENTS.md`/README, two-realms/session/hosting contract; local-vs-internal-vs-live environment matrix | Old no-OIDC instructions formally superseded; decisions recorded |
| AUTH-01 | Add pinned Keycloak/PostgreSQL local Compose profile/config/realm import, secrets and health checks; local HTTPS proxy strategy; do not expose admin console publicly | Keycloak boots, two realm/client configurations reproducible, backup documented |
| AUTH-02 | **Expand:** refactor `composition/api.go`, config, readiness and AI runtime bootstrap to remove ALL Dev Login coupling (route mounting, `loadProduct`, `AI_CORE_LOCAL_DRAFT`), preserve local Inspector security; add OIDC/auth contracts to OpenAPI | Disabling `ALLOW_DEV_LOGIN` does not remove app routes/AI Core; local draft can run safely on loopback without dev bypass; unsafe draft config fails elsewhere |
| AUTH-03 | Go Auth/BFF module and SQL tables, OIDC transaction/PKCE callback, host binding, token checks, session cookie + CSRF, active/owner guards, logout/revocation and auth audit | User/staff OIDC contract/middleware tests and negative issuer/host/CSRF/expiry tests pass |
| AUTH-04 | Keycloak staff realm/login/password policy + deterministic initial staff provisioning + `/auth/staff/*` + `/auth/staff/session`; BO-01 owns `/api/v1/admin/me` | Active admin can login/logout on local origin, inactive or wrong realm denied |
| AUTH-05 | Keycloak Java SMS SPI, rate limits, Keycloak theme, `SmsSender` Mock/Kavenegar adapters and Keycloak integration tests | Challenge tested for replay, expiry, failures and abuse; no secret/phone leakage |
| AUTH-06 | User phone flow + first-login/returning-account mapping, onboarding gate; API `RequireUser` replaces dev guard in new route tree | User authenticated via Mock SMS can access only own profile/chat; duplicates denied |
| AUTH-07 | Backend support for web/TWA callback/Session and reverse-proxy origin contracts; formal frontend handoff sheet | Exact redirects/cookies/headers documented; same-origin browser smoke via test page; frontend work remains later |
| AUTH-08 | Security+life-cycle: logout propagation, password/phone recovery design, CSRF/XSS/CSP, account-disable, rate-limit, provider/DB restarts, secrets + backup tests | Security gates pass on disposable integrated environment |
| AUTH-09 | **Cross-repo cutover (NOT backend-only):** after frontend user login & admin app are migrated, remove `/api/v1/dev/*`, dev signer/middleware, Dev cookie/config/DB dependencies, update API schemas/tests/docs and CI | No legacy login route/middleware in deployed build; end-to-end web + admin + TWA tests pass |
| AUTH-10 | Shared/internal/deployment and real-user readiness gate: VPN for staff (not local), TLS, approved privacy/clinical/safety/retention controls, real SMS smoke, operational runbooks | No public/real-user claim solely from completing Auth code |

**Ticket ownership:** AUTH-00..08 are primarily backend/infrastructure/Keycloak work. AUTH-09 is a synchronized backend + frontend change. Frontend is not required to implement the earlier backend PRs except for minimal test harnesses.

## H. Mandatory regression and PR checks

- Preserve `docs/ai-core/local-inspector.md` restrictions (`APP_ENV=local`, loopback bind, private files) under both dev-auth-enabled transition and dev-auth-removed final state; no inspector HTTP API.
- Tests: correct/wrong realm, wrong issuer/audience, cross-user and cross-staff cookies, deactivated staff, state replay, callback host mismatch, CSRF, logout/revocation, SQL rollback, two concurrent first logins, OTP resend/rate limits and provider timeout.
- Verify `APP_ENV`/`ALLOW_DEV_LOGIN` and `AI_CORE_CONFIG_FILE` combinations; AI runtime and `/health/ready` reflect actual loaded core regardless of dev flag.
- Use `python3 scripts/check_structure.py`; `python3 scripts/check_contract.py`; from `apps/api`: `go build ./...`, `go vet ./...`, `go test ./...`; plus SQL Server and Keycloak/Postgres disposable integration tests, realm import and Java SPI tests. Go deps require network/cache.
- New migrations: `python3 scripts/new_migration.py <name>` (check current head first); no old migration edits. Do not claim checks pass unless executed.

## I. Backend-to-frontend contract freeze

Give the frontend team: (1) exact user/staff login and logout URLs, (2) session introspection endpoints and their JSON error states, (3) cookie/CSRF contract, (4) onboarding redirect behavior, (5) generic account/OTP UI states and which screens live in the Keycloak theme, (6) admin OpenAPI and fixture responses, (7) TWA origin and Digital Asset Links requirements, (8) error/maintenance handling for expired sessions, OTP vendor outages and IdP downtime. Frontend should not parse or persist OIDC tokens.

**Source pointers:** current `apps/api/internal/app/composition/api.go`, `apps/api/internal/platform/config/config.go`, `apps/api/internal/app/composition/ai_runtime.go`, `apps/api/internal/app/composition/ai_logging.go`, `apps/api/internal/platform/observability`, `docs/ai-core/local-inspector.md`, `knowledge/README.md`, `apps/api/db/migrations/0023_account_soft_delete.sql`, and previous `THEIA_AUTH_IMPLEMENTATION_PLAN_FINAL.md` (v2).
