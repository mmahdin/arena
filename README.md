# Wellbeing Companion — Backend Foundation

> **Internal-only / synthetic-data prototype. NOT safe for a public mental-health product, real patient data, or open internet exposure.**

This repository is a standalone Go modular monolith **derived from the architectural patterns of Yaran Clinic**, not a clone of its clinical domain or identity infrastructure. It accompanies [`MVP_SCOPE.md`](MVP_SCOPE.md).

## What is included

- Go + Gin HTTP API with application composition root, health endpoints and centralized machine errors.
- Microsoft SQL Server 2022 storage (`database/sql` + `go-mssqldb`), sequential/checksummed T-SQL migration runner, three *synthetic* seed users and three clearly marked synthetic demo conversations.
- Explicitly enabled local/staging signed-cookie Test User Switcher (`/api/v1/dev/*`); no real authentication.
- User profile and onboarding save (`GET /me`, `PATCH /me/profile`).
- Conversation CRUD, message history, JSON AI turns with retry idempotency.
- AI gateway interface; deterministic mock provider by default; optional OpenAI-compatible Chat Completions provider through server-only environment variables.
- Versioned `supportive-v3` prompt and a deliberately **rudimentary**, non-clinical safety fallback.
- OpenAPI 3.1 **source spec** (not yet a generated server; see [Known gaps](#known-gaps)).
- Six agent skills under `.agents/skills`, including **`wellbeing-backend-module`** designed for this product.
- `AGENTS.md`, constitution docs, module docs, ADR and check scripts.

## Repository layout

See [`REPO_MAP.md`](REPO_MAP.md). Backend Go module is nested at `apps/api`.

## Quick start (local only)

Prerequisites: Docker + Compose plugin. Go 1.23+ only if running the binaries outside Docker. The SQL Server container requires sufficient memory (at least ~2 GB available to SQL Server).

```sh
cp .env.example .env
# IMPORTANT: replace DEV_SESSION_SECRET with 32+ unpredictable characters and set a strong MSSQL_SA_PASSWORD.
# Example: python3 -c 'import secrets; print(secrets.token_hex(32))'
docker compose up --build -d
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

Default mode `LLM_PROVIDER=mock` responds with an explicitly marked deterministic synthetic reply. To use an external LLM, set `LLM_PROVIDER=openai-compatible`, `MODEL_API_KEY`, and `MODEL_NAME` **only after verifying provider data handling**. Never put keys or real sensitive chat content in the repo. Restart the API. `LLM_TIMEOUT_SECONDS` (default 35, maximum 90) is bounded below the 120-second turn lease; `MODEL_MAX_OUTPUT_TOKENS` (default 700) and `MAX_CONTEXT_CHARS` (default 16000) bound external model usage. External provider requests are not retried automatically and all provider errors are sanitized as `503 AI_UNAVAILABLE`. This path has **only fake-HTTPS unit testing here**; do not send real or sensitive messages. See [AI module](apps/api/internal/modules/ai/README.md) and [ADR-0006](docs/adr/0006-bounded-real-llm-gateway.md).

> `compose.yaml` binds the host's API and SQL Server ports to **127.0.0.1**. A Docker container or device outside this host cannot access it without additional networking. For a frontend, use a same-origin dev proxy to `/api/v1` or configure a suitable same-site test setup; avoid public port forwarding.

### Smoke test via cookie jar

```sh
curl -s http://127.0.0.1:8080/api/v1/dev/users
curl -sS -c /tmp/wellbeing-cookie.txt -H 'Content-Type: application/json' \
  -d '{"userId":"11111111-1111-4111-8111-111111111111"}' \
  http://127.0.0.1:8080/api/v1/dev/session
curl -sS -b /tmp/wellbeing-cookie.txt http://127.0.0.1:8080/api/v1/me
curl -sS -b /tmp/wellbeing-cookie.txt -H 'Content-Type: application/json' \
  -X POST -d '{}' http://127.0.0.1:8080/api/v1/conversations
```

Use the returned conversation UUID and a **new** UUID for `clientTurnId` in `POST /api/v1/conversations/{id}/turns`. The test users are `tester-1` (completed onboarding), `tester-2` (not completed), and `tester-3` (completed). Tester 1 has one synthetic demo thread; tester 3 has two. Tester 2 starts without history and with onboarding incomplete. No real/private content is seeded. Browser smoke tests complete tester 2’s profile on their first run.

### Direct Go commands

```sh
cd apps/api
go mod download            # install versions pinned in go.mod/go.sum
go test ./...
go run ./cmd/migrate        # DATABASE_URL, MIGRATIONS_DIR=db/migrations
go run ./cmd/api            # configuration from shell environment; .env not auto-loaded
```

For native commands that load the root `.env` safely, use `make migrate` and `make api` from this directory. Do not shell-source `.env`: the SQL URL contains shell metacharacters. `make build` writes binaries under ignored `bin/`.

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

Staging configuration follows the Yaran variable naming convention:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ai-therapist-backend-configmap
  namespace: ramona
data:
  APP_ENV: staging
  BUSINESS_TIMEZONE: Asia/Tehran
  DATABASE_ENCRYPT: disable
  DATABASE_HOST: db-test.alibaba.local
  DATABASE_NAME: YaranClinic
  DATABASE_PORT: "1433"
  DATABASE_USERNAME: yaranclinic
  FRONTEND_ORIGIN: https://ai-therapist-frontend.alibaba.stg.agidp.ir
  HTTP_ADDR: ":8080"
  LOG_LEVEL: warn
  ALLOW_DEV_LOGIN: "true"
  COOKIE_SECURE: "true"
  LLM_PROVIDER: openai-compatible
  MODEL_API_BASE_URL: https://api.openai.com/v1
  MODEL_NAME: gpt-5.6-sol
  MIGRATIONS_DIR: /app/migrations
  MSSQL_AUTO_CREATE_DATABASE: "false"
```

`ai-therapist-backend-secret` must contain `MODEL_API_KEY`, `DATABASE_PASSWORD` and an independent
random `DEV_SESSION_SECRET` of at least 32 bytes. Keep the signing secret stable
across restarts and replicas. No credentials belong in the ConfigMap or Git.

The `AI_OPENAI_*` names used in other services are not read here. Map the base
origin to `MODEL_API_BASE_URL` **including `/v1`**, the API key to `MODEL_API_KEY`,
and the default model to `MODEL_NAME`. Set `LLM_PROVIDER=openai-compatible`
explicitly; providing a key alone does not override `mock`. This app uses one
model directly, without an alias table or automatic upstream retries. Do not put
the API key in frontend/Vite variables. Recreate/restart the API deployment after
updating its ConfigMap/Secret so the running process receives the changes.

For a quick check after signing into staging, run in the browser console:

```js
await fetch('/api/v1/me/ai-status', { credentials: 'include' }).then(r => r.json())
```

Expected external configuration: `provider: "openai-compatible"`,
`model: "gpt-5.6-sol"`, `mock: false`, `promptVersion: "supportive-v5"`.
This endpoint is authenticated and returns configuration only; it does not call
OpenAI or verify credentials/quota/network. Send a short synthetic ordinary chat
message and verify the request in the OpenAI project's usage to confirm an actual
external call. A configured external-provider failure returns `503 AI_UNAVAILABLE`;
there is no fallback to Mock. Explicit crisis routing can bypass the provider.
Before this endpoint is deployed, the current mock adapter's ordinary reply ends
with `(پاسخ آزمایشی Mock)`; `/health/ready` checks SQL Server, not OpenAI.

Deploy migration `0016_onboarding_answers.sql` before the new API, then deploy the
frontend. Explicit onboarding answers are saved on final Ready and read afresh
for each chat turn, including after reload. Existing completed users have no
answers to recover; previously discarded selections cannot be backfilled.
Draft selections before Ready still do not survive a page reload. See
[`docs/integration/ONBOARDING_CHAT_CONTEXT.md`](docs/integration/ONBOARDING_CHAT_CONTEXT.md).

`FRONTEND_ORIGIN` takes precedence over legacy `APP_ALLOWED_ORIGIN`. HTTPS origins
default to secure cookies; an explicit `COOKIE_SECURE` value overrides this.
`LOG_LEVEL` configures the API's JSON startup/error logs, and `BUSINESS_TIMEZONE`
controls their timestamps; database timestamps remain UTC. Timezone data is
embedded so `Asia/Tehran` works in the Alpine runtime image.

`ALLOW_DEV_LOGIN=true` is required explicitly for test routes. Supported test
environments are `local`, `stage`, `staging`, and `staging-v2`; production and
unknown environments reject the flag. This is synthetic tester identity, not
Keycloak or verified mobile authentication. Yaran's `KEYCLOAK_*`, OAuth redirect,
and OAuth session TTL settings are not used by this application. The existing
dev-session absolute lifetime remains 12 hours; no idle-session policy is added.

The migration binary consumes the same database fields without requiring AI
credentials or a dev-session signing secret. Loading configuration does not
create a database or apply migrations. Auto-creation remains restricted to
explicit local bootstrap via `MSSQL_AUTO_CREATE_DATABASE=true`.

## Agent orientation

1. Read [`AGENTS.md`](AGENTS.md) -> [`MVP_SCOPE.md`](MVP_SCOPE.md) -> [`PRODUCT_CONSTITUTION.md`](PRODUCT_CONSTITUTION.md) -> [`TECHNICAL_CONSTITUTION.md`](TECHNICAL_CONSTITUTION.md) -> [`REPO_MAP.md`](REPO_MAP.md).
2. Read the owning module README and relevant `.agents/skills/<name>/SKILL.md`.
3. Update OpenAPI source before changing public API. Generated handlers/types are a **planned task**, not falsely asserted as existing.
4. Use new migrations; do not edit applied migrations. Run `python3 scripts/new_migration.py your_change`.
5. Run `make validate` in an environment with downloaded dependencies, SQL Server and optionally model provider configured.

**Database decision:** SQL Server 2022 is authoritative (ADR-0002). The original PostgreSQL proposal in ADR-0001 is superseded. SQL Server scripts are not backwards-compatible with any earlier local PostgreSQL volume.

## Known gaps / deliberate deferrals

- No OIDC, Keycloak, production login, back office, analytics, profile inference, long-term memory, streaming SSE, or Worker.
- **Dev session is identity selection, not authentication.** Anyone with local access can assume any synthetic test identity. Not for real user data.
- Current Tasks 02–07 pass live SQL Server and HTTP acceptance, including concurrent retries and fenced lease recovery. Concurrent distinct turns in a single thread remain deferred. See the current run in `VALIDATION_REPORT.md`.
- Prompt keyword safety checks can miss nuanced crises and have false positives; **not clinically validated**. Must be replaced/reviewed before any real-user test.
- API handlers are handwritten Gin adapters today; **generated server/client code is not present**. Introduce codegen + drift CI as a separate feature before public delivery.
- Dependency versions and checksums are pinned in `apps/api/go.mod` and `apps/api/go.sum`. Docker builds verify modules and reuse download/compiler caches.
- Conversation history is plaintext in a **local isolated development database**; never put real sensitive data into it. At-rest encryption, retention, deletion/export, mature consent and real authorization are future release gates.
- Idempotency uses fenced 120-second SQL Server leases. A retry with the same turn ID and payload can reclaim an expired reservation; stale workers cannot commit the successor lease. Recovery is request-driven; there is no background reconciler.
- Message list is limited to latest 100. No pagination UI contract yet.
- Rate limiting and cost caps are not yet enforced. Model timeouts exist; before external model testing, add usage limits and safe retries.

## Validation

- `python3 scripts/smoke_api.py` after starting Docker Compose: login, cross-user isolation, conversation and retry smoke test.

- `python3 scripts/check_structure.py`
- `python3 scripts/check_contract.py` (does not replace OpenAPI schema lint)
- `make test` (uses pinned modules)
- `make integration-test` (local SQL Server, creates/removes an isolated test database)
- `docker compose up --build -d` + `python3 scripts/smoke_api.py` (local mock + SQL Server)

This is a **foundation with a thin testable path**, not a finished healthcare or authentication system.


### Dev Session Task 02 update

The session selector remains internal-only. Migration `0005_dev_sessions.sql` adds revocable SQL Server dev sessions: a browser logout or test-user switch invalidates the old cookie on the server; existing cookies require a new test login after the upgrade. Run migration before starting the updated API. See `docs/adr/0003-revocable-dev-sessions.md` and `VALIDATION_REPORT.md` for tests and limitations.

### Task 03: Onboarding & profile completion

`GET /api/v1/me` reports `onboardingCompletedAt` (null for new synthetic testers). `PATCH /api/v1/me/profile` accepts **both** a display name and `listening|guidance|balanced`; the first valid update completes onboarding. Edits preserve that initial timestamp, including under concurrent SQL writes. `POST /api/v1/conversations` and `POST /api/v1/conversations/{id}/turns` return `409 ONBOARDING_REQUIRED` before completion (foreign conversation IDs still return 404). No new migration or frontend request schema change. See `docs/adr/0004-onboarding-profile-and-chat-gate.md`. Run Go, SQL Server integration and smoke checks before considering Task 03 verified.

### Task 04 — Conversation durability and retries

Migration `0006_turn_recovery_lease.sql` adds fenced 120-second processing leases. After a crashed AI request, retry with the **same** `clientTurnId` and unchanged message: while a lease is active the API returns `409 TURN_ALREADY_PROCESSING`; once it expires a retry takes over safely. Completed replies are replayed without regenerating or duplicating messages. A changed payload under the same ID is `400 VALIDATION_ERROR`. Existing JSON chat APIs remain unchanged. **Stop the old API binary, apply the migration, and start the upgraded binary** (no mixed-version runtime). Run `make migrate`, `make test`, `make integration-test`, `make smoke` and frontend live E2E.

### Task 05 — Real LLM gateway, prompt and context

`LLM_PROVIDER=mock` remains the default. For a locally approved synthetic-only test, configure `LLM_PROVIDER=openai-compatible`, `MODEL_NAME`, `MODEL_API_KEY`, optional `MODEL_API_BASE_URL` and the bounded timeout/context/output settings in `.env`; restart the API. `supportive-v2` includes style-specific Persian guidance while the database preserves prompt versions from older replies. The JSON API and MSSQL schema have **not** changed. Fake-HTTPS adapter tests cannot replace a live, approved provider check or clinical/privacy/safety review.

### Task 06 — AI safety and reliability (synthetic only)

The `supportive-v3` prompt adds explicit professional/medication/dependency boundaries. Server-side phrase checks normalize common Persian spelling/spacing variations, route explicit crisis phrases to a fixed response without calling the provider, and replace a small set of unsafe model statements. Canceled requests and malformed model outputs are rejected instead of being saved. The API and MSSQL schema are unchanged; no migration is required. **This is not validated crisis detection or clinical safety screening.** See `docs/quality/ai-safety-evaluation.md` and `docs/adr/0007-synthetic-ai-safety-reliability.md`. Run Go tests, MSSQL integration and smoke checks in your local environment.

### Task 07 — local MVP acceptance

Run `make acceptance-http` for an isolated SQL Server + Gin HTTP golden-flow test, or `make acceptance-backend` to run all backend gates including the live mock smoke. See `docs/quality/mvp-acceptance.md` for the full evidence matrix, browser limitations and internal-only caveats.

### Hamrah local synthetic phone entry

After applying migration `0007_dev_phone_identities.sql`, the Hamrah client uses `POST /api/v1/dev/identity-session` with a *fake* `0900000XXXX` identifier. No OTP, SMS, phone verification, or real phone data is supported. This is **not authentication**. Prior seeded dev endpoints are retained only for legacy local tests; do not show a user list in Hamrah.
