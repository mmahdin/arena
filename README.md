> **AUTH-09 CURRENT:** Tia authenticates only through Keycloak User SMS + Go BFF. `/api/v1/dev/*` and its legacy cookie have been permanently removed. `ALLOW_DEV_LOGIN=true` is rejected. Historic Dev Session migrations and ADRs remain for checksum/audit history, not runtime support. See [AUTH-09 runbook](docs/operations/auth09-dev-session-retirement.md).

# Wellbeing Companion — Backend Foundation

> **Internal-only / synthetic-data prototype. NOT safe for a public mental-health product, real patient data, or open internet exposure.**

This repository is a standalone Go modular monolith **derived from the architectural patterns of Yaran Clinic**, not a clone of its clinical domain or identity infrastructure. It accompanies [`MVP_SCOPE.md`](MVP_SCOPE.md).

> **Approved backend Auth/Back-Office implementation:** shared Keycloak
> (dedicated PostgreSQL, user SMS OTP and provisioned staff login), Go Auth/BFF
> with SQL Server app sessions, and a private Back-Office Admin module. Follow
> [Auth plan v3](docs/plans/THEIA_AUTH_IMPLEMENTATION_PLAN_V3.md),
> [Back-Office plan v3](docs/plans/THEIA_BACKOFFICE_IMPLEMENTATION_PLAN_V3.md)
> and the [backend-first runbook](docs/plans/THEIA_BACKEND_FIRST_IMPLEMENTATION_RUNBOOK.md).
> ADRs [0012](docs/adr/0012-shared-identity-and-sessions.md) and
> [0013](docs/adr/0013-private-backoffice-boundary.md) supersede the
> corresponding **historical MVP exclusions**, not the synthetic-only release
> restrictions. `README.md` quick-start/dev endpoints below describe the
> **historical** earlier milestones; AUTH-09 supersedes Dev Session instructions.


> **AUTH-01 through AUTH-06 (local-only, opt-in):** Keycloak
> `26.8.2` + PostgreSQL `17.12` with separate user/staff realms; Go OIDC/BFF
> and SQL sessions are implemented, and staff can now be allowlisted through
> the local `staffctl` command. Staff Login remains disabled until explicitly
> enabled; no seeded staff account is present. The AUTH-05 SMS authenticator
> and Persian Keycloak theme are now available as an **opt-in local SPI image**;
> AUTH-06 adds local User BFF activation and mandatory signed SMS proof.
> User Login remains **disabled by default**. The Back-Office backend
> APIs are implemented, while the separate Admin UI is still pending. See
> [Keycloak setup](docs/operations/keycloak-local.md),
> [BFF setup](docs/operations/auth-bff-local.md) and
> [Staff provisioning](docs/operations/staff-auth-local.md) and
> [SMS OTP SPI setup](docs/operations/auth-sms-otp-auth05.md).

## What is included

- Go + Gin HTTP API with application composition root, health endpoints and centralized machine errors.
- Microsoft SQL Server 2022 storage (`database/sql` + `go-mssqldb`), sequential/checksummed T-SQL migration runner, three *synthetic* seed users and three clearly marked synthetic demo conversations.
- User profile and onboarding save (`GET /me`, `PATCH /me/profile`).
- Conversation CRUD, message history, JSON AI turns with retry idempotency.
- The complete non-proactive Theia AI Core, with structured Brain, plan-bound generation, layered safety, provenance-carrying memory, Protocols, Reflection and supervised SQL workers.
- Explicit consent and privacy APIs; retained-data soft deletion only. No default mock/prototype reply or Daily Reflection feature remains.
- OpenAPI 3.1 **source spec** (not yet a generated server; see [Known gaps](#known-gaps)).
- Five agent skills under `.agents/skills`, including **`wellbeing-backend-module`** designed for this product.
- `AGENTS.md`, constitution docs, module docs, ADR and check scripts.

## Repository layout

See [`REPO_MAP.md`](REPO_MAP.md). Backend Go module is nested at `apps/api`.

## Quick start (local only)

Prerequisites: Docker + Compose plugin, Python 3.9+ and Make for setup commands on Linux/macOS. Go 1.23+ only if running the binaries outside Docker. The SQL Server container requires sufficient memory (at least ~2 GB available to SQL Server).

```sh
make env-sync                # creates .env or appends only missing template keys
# IMPORTANT: set a strong MSSQL_SA_PASSWORD and configure Keycloak SMS/BFF per the AUTH-09 runbook.
make up
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

WP-16 replaces the experimental AI workflow. Configure `AI_CORE_CONFIG_FILE` with
released bundle verification keys, WP-04 model/provider chains, managed memory
keys and an explicitly reviewed privacy mapping. See
[deployment and HTTP migration](docs/operations/ai-core-wp16.md). Without this
configuration, protected AI routes and readiness return 503; there is no mock or
legacy fallback. The repository's placeholder bundle cannot be released as real
clinical content.

> `compose.yaml` binds the host's API and SQL Server ports to **127.0.0.1**. A Docker container or device outside this host cannot access it without additional networking. For a frontend, use a same-origin dev proxy to `/api/v1` or configure a suitable same-site test setup; avoid public port forwarding.

### Live login smoke test (Keycloak + BFF)

The legacy `/api/v1/dev/*` endpoints and cookie-based test login were removed in AUTH-09. After configuring the SMS SPI, Go BFF, reverse proxy and local SQL Server, run this **interactive**, local-only smoke check:

```sh
make smoke
```

It opens the Tia-branded Keycloak login and verifies the real browser session. See [AUTH-09 runbook](docs/operations/auth09-dev-session-retirement.md) and [AUTH-UI02](docs/operations/auth-ui02-visual-acceptance.md). An active User BFF, valid OTP and loopback origin are prerequisites.

### Direct Go commands

```sh
cd apps/api
go mod download            # install versions pinned in go.mod/go.sum
go test ./...
go run ./cmd/migrate        # DATABASE_URL, MIGRATIONS_DIR=db/migrations
go run ./cmd/api            # configuration from shell environment; .env not auto-loaded
```

For native commands that load the root `.env` safely, use `make migrate` and `make api` from this directory. The launcher first adds missing `.env.example` keys to `.env`; existing values (including empty values), secrets and comments stay intact. Exported environment variables override the loaded values. `make up` also syncs before starting Compose; run `make env-sync` to sync without starting anything. New template defaults are copied literally, so new credentials still need to be configured once. Changes to defaults and removed template keys do not replace or delete existing local settings.

`make api` starts the API with the [local AI inspector](docs/ai-core/local-inspector.md) enabled in that process, even if `.env` says `AI_LOCAL_INSPECTOR=false`. Use `AI_LOCAL_INSPECTOR=false make api` to disable it. It runs inside the API and requires `APP_ENV=local` and a loopback `HTTP_ADDR`. Direct `python3 scripts/run_local.py go run ./cmd/api` follows `.env` and exported values instead of enabling inspection itself.

Do not shell-source `.env`: the SQL URL contains shell metacharacters. `make build` writes binaries under ignored `bin/`.

If the default Go proxy returns a regional 403 during Docker build, the verified fallback used here is:

```sh
docker compose build --build-arg GOPROXY=https://goproxy.io,direct
docker compose up --no-build -d
```

## Staging environment configuration

The API and `/app/migrate` share the same database configuration loader. Set
`DATABASE_HOST`, `DATABASE_PORT` (default `1433`), `DATABASE_USERNAME`,
`DATABASE_NAME`, and `DATABASE_ENCRYPT` (`disable`, `false`, or `true`) in the
ConfigMap, and `DATABASE_PASSWORD` in the Secret. Credentials are URL-encoded
by the application; store the original password in the Secret.

When any split database field is set, split configuration takes precedence over
`DATABASE_URL`, and incomplete settings fail startup without exposing secrets.
Existing local `.env` files and Compose can continue to use `DATABASE_URL` when
no split settings are present. The application has no built-in staging host,
database name, username, or password. Keep the local `.env` on its independent
loopback `wellbeing` database; do not copy the staging connection into it.

AI deployment configuration is described in [the WP-16 guide](docs/operations/ai-core-wp16.md).
Keep `MODEL_*` secrets server-side. The old one-model environment settings do not
activate the new core, and the old AI-status/prompt-version route has been removed.
Run all migrations through the current migration head (0032 in AUTH-09) before deploying the new binary. Model/network
availability is not inferred from readiness; it checks database/core worker life.
Only synthetic test identities are supported; no public deployment is authorized.

`FRONTEND_ORIGIN` takes precedence over legacy `APP_ALLOWED_ORIGIN`. HTTPS origins
default to secure cookies; an explicit `COOKIE_SECURE` value overrides this.
`LOG_LEVEL` configures the API's JSON startup/error logs, and `BUSINESS_TIMEZONE`
controls their timestamps; database timestamps remain UTC. Timezone data is
embedded so `Asia/Tehran` works in the Alpine runtime image.

Dev Session authentication has been retired. The backend rejects `ALLOW_DEV_LOGIN=true` in every environment; remove the obsolete flag and `DEV_SESSION_SECRET` from local `.env`. For real User Login, activate the verified SMS Keycloak realm and configure `AUTH_BFF_ENABLED`, `AUTH_USER_LOGIN_ENABLED` and `AUTH_USER_SMS_FLOW_VERIFIED` per the BFF runbook. Staff Login uses a separate realm and never shares the User cookie.

The migration binary consumes the same database fields without requiring AI
credentials or a dev-session signing secret. Loading configuration does not
create a database or apply migrations. Auto-creation remains restricted to
explicit local bootstrap via `MSSQL_AUTO_CREATE_DATABASE=true`.
New local databases use unique data/log filenames in SQL Server's default
directories, so recreation does not collide with files belonging to a renamed
database or left behind by an earlier database. Existing databases and files
are preserved.

## Agent orientation

1. Read [`AGENTS.md`](AGENTS.md) -> [`MVP_SCOPE.md`](MVP_SCOPE.md) -> [`PRODUCT_CONSTITUTION.md`](PRODUCT_CONSTITUTION.md) -> [`TECHNICAL_CONSTITUTION.md`](TECHNICAL_CONSTITUTION.md) -> [`REPO_MAP.md`](REPO_MAP.md).
2. Read the owning module README and relevant `.agents/skills/<name>/SKILL.md`.
3. Update OpenAPI source before changing public API. Generated handlers/types are a **planned task**, not falsely asserted as existing.
4. Use new migrations; do not edit applied migrations. Run `python3 scripts/new_migration.py your_change`.
5. Run `make validate` in an environment with downloaded dependencies, SQL Server and optionally model provider configured.

**Database decision:** SQL Server 2022 is authoritative (ADR-0002). The original PostgreSQL proposal in ADR-0001 is superseded. SQL Server scripts are not backwards-compatible with any earlier local PostgreSQL volume.

## Known gaps / deliberate deferrals

- **Current implementation:** OIDC/BFF, local Staff provisioning and SMS-authenticated User login are opt-in; the Back-Office backend APIs exist, while the separate frontend UI and live integration/security release gates remain pending. ADRs 0012/0013 authorize this staged work. The historical exclusion list for additional features is superseded only where explicit newer ADRs or AI Core Contract Pack v1.2 authorize them.
- **Dev session is identity selection, not authentication.** Anyone with local access can assume any synthetic test identity. Not for real user data.
- Current Tasks 02–07 pass live SQL Server and HTTP acceptance, including concurrent retries and fenced lease recovery. Concurrent distinct turns in a single thread remain deferred. See the current run in `VALIDATION_REPORT.md`.
- Prompt keyword safety checks can miss nuanced crises and have false positives; **not clinically validated**. Must be replaced/reviewed before any real-user test.
- API handlers are handwritten Gin adapters today; **generated server/client code is not present**. Introduce codegen + drift CI as a separate feature before public delivery.
- Dependency versions and checksums are pinned in `apps/api/go.mod` and `apps/api/go.sum`. Docker builds verify modules and reuse download/compiler caches.
- Conversation history is plaintext in a **local isolated development database**; never put real sensitive data into it. At-rest encryption, retention, deletion/export, mature consent and real authorization are future release gates.
- Idempotency uses fenced 120-second SQL Server leases. A retry with the same turn ID and payload can reclaim an expired reservation; stale workers cannot commit the successor lease. Recovery is request-driven; there is no background reconciler.
- Message list is limited to latest 100. No pagination UI contract yet.
- New user-turn rate limits are enforced atomically in SQL from the reviewed bundle. WP-04 usage limits remain process-local and are not a cross-instance financial reservation guarantee.

## Validation

- `python3 scripts/smoke_api.py` after starting Docker Compose: login, cross-user isolation, conversation and retry smoke test.

- `python3 scripts/check_structure.py`
- `python3 scripts/check_contract.py` (does not replace OpenAPI schema lint)
- `make test` (uses pinned modules)
- `make integration-test` (local SQL Server, creates/removes an isolated test database)
- `docker compose up --build -d` + configured synthetic-only `scripts/smoke_api.py` (see WP-16 guide)

This is a **foundation with a thin testable path**, not a finished healthcare or authentication system.


### Dev Session Task 02 update

[ARCHIVED milestone] Migration `0005_dev_sessions.sql` previously introduced local dev sessions; AUTH-09 removes the runtime and drops the tables via `0032`. The historical migration remains immutable.

### Task 03: Onboarding & profile completion

`GET /api/v1/me` reports `onboardingCompletedAt` (null for new synthetic testers). `PATCH /api/v1/me/profile` accepts **both** a display name and `listening|guidance|balanced`; the first valid update completes onboarding. Edits preserve that initial timestamp, including under concurrent SQL writes. `POST /api/v1/conversations` and `POST /api/v1/conversations/{id}/turns` return `409 ONBOARDING_REQUIRED` before completion (foreign conversation IDs still return 404). No new migration or frontend request schema change. See `docs/adr/0004-onboarding-profile-and-chat-gate.md`. Run Go, SQL Server integration and smoke checks before considering Task 03 verified.

### Active AI Core product boundary

WP-16 replaces the earlier AI tasks and JSON turn shape. See the
[operation/client guide](docs/operations/ai-core-wp16.md) and
[verification report](docs/quality/ai-core-wp16.md). Old AI prompts, phrase safety,
mock replies and Daily Reflection routes are retired; all stored data remains.
The SQL HTTP acceptance suite injects the actual full core with explicit synthetic
provider fixtures. Running a deployment smoke requires a reviewed configuration,
explicit test-account consent and an authorized synthetic provider environment.

### Hamrah local synthetic phone entry

[ARCHIVED milestone] The former `/api/v1/dev/identity-session` synthetic phone flow was retired by AUTH-09. Use `/auth/user/login` for actual SMS-based Keycloak authentication.

## Troubleshooting and logs

See [API logging and error correlation](docs/operations/LOGGING.md) for structured JSON logs, `LOG_LEVEL`, `LOG_FILE`, request IDs, AI Core stage tracing, and privacy constraints.

### New AUTH-03/04 BFF and local Staff Login (opt-in)

The backend now contains an independent Keycloak OIDC/BFF module with SQL Server
identity mapping and revocable application sessions. This does **not** replace
Dev Login yet or enable user SMS login by default. Staff Sign-In can be
activated locally only after creating a named account in the Keycloak Staff
Realm and allowlisting its Keycloak subject in SQL via `staffctl`. The Keycloak
PostgreSQL service remains separate; see
[`docs/operations/auth-bff-local.md`](docs/operations/auth-bff-local.md) and
[`docs/operations/staff-auth-local.md`](docs/operations/staff-auth-local.md).

### AUTH-05 SMS OTP SPI (opt-in, user login still disabled)

Use `compose.auth-sms.yaml` with the original Keycloak overlays to build the
passwordless User realm authenticator. See
[`docs/operations/auth-sms-otp-auth05.md`](docs/operations/auth-sms-otp-auth05.md)
for local secret generation, non-destructive realm setup, Mock/Kavenegar
transport boundaries, and test instructions. This does **not** enable
`theia-users-bff`, `AUTH_USER_LOGIN_ENABLED`, or real-user access.

### AUTH-06 User SMS sign-in (opt-in, local synthetic only)

The User BFF can be enabled after installing the Keycloak User Session Note mapper, verifying the single REQUIRED SMS browser flow, and checking the local Mock SMS browser journey. It requires `AUTH_USER_LOGIN_ENABLED=true`, `AUTH_USER_SMS_FLOW_VERIFIED=true` and `ALLOW_DEV_LOGIN=false` (or unset); a signed `theia_sms_verified=sms-otp-v1` claim is also mandatory. Product handlers now consume a provider-neutral verified User ID. SQL Server identity mapping is stable and never merges by phone. See [AUTH-06 runbook](docs/operations/auth-user-sms-auth06.md). This is not real-user or TWA release approval.

AUTH-07 browser integration handoff: see
[`docs/operations/auth-browser-twa-auth07.md`](docs/operations/auth-browser-twa-auth07.md).
The backend supports sealed same-origin `return_to` redirects and an opt-in,
local-only synthetic browser smoke harness; no user/admin frontend or Android
TWA package is created by this ticket.


**AUTH-08 Keycloak logout lifecycle:** real User/Staff BFF login now supports an
explicit SSO logout handoff, realm-verified back-channel notifications and
all-Theia-session revocation (SQL Server migration `0031`). For local Keycloak
volumes, run `python3 scripts/auth/configure_keycloak_logout.py --apply` after
migrating. The previous `/logout` is still an app-only 204; use `/logout/sso`
for browser SSO handoff. Follow `docs/operations/auth-session-lifecycle-auth08.md`.

### PROD-02 — Local synthetic Metabase analytics

After PROD-01 migrations through 0036, apply `0037_product_reporting_views.sql` and
follow [the local analytics runbook](docs/operations/prod02-metabase.md).
Metabase is a private optional Compose overlay with dedicated PostgreSQL
application state and a SQL Server **aggregate-view-only** reporting login;
it must never receive SQL Server `sa` credentials or real-user chat content.
Metric definitions are in [PROD02_METRICS.md](docs/analytics/PROD02_METRICS.md).
