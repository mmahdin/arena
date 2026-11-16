# Technical Constitution — Wellbeing MVP Backend

## Baseline and trace to Yaran Clinic

Adapted from Yaran: Go/Gin, modular monolith, module-owned domain/application/HTTP/DB adapters, composition root, contract-first policy, explicit SQL migrations, same-origin cookie sessions and error contracts. **Not reused:** patient/therapist/branch domain, OIDC, enterprise authz, worker/outbox, financial rules.

## Ownership and imports

```text
cmd/api -> internal/app/composition
composition -> module public constructors + platform
modules/<feature>/adapters/http -> modules/<feature>/application
modules/<feature>/adapters/mssql -> modules/<feature>/application and domain
modules/<feature>/application -> modules/<feature>/domain and consumer-owned ports
modules/<feature>/domain -> standard library + UUID type only
```

No shared `global/repository/handler` folders. Feature-to-feature use must go via a small application port implemented in composition. Concrete example: `conversation/application.ProfileReader` is implemented by a composition adapter that calls `user/application.Service`.

## Runtime and data

- Go module at `apps/api`, Go 1.23 baseline; Gin 1.10, go-mssqldb v1.7.2 + database/sql, SQL Server 2022.
- JSON routes under `/api/v1`, camelCase fields, UUID IDs, stable Problem Details codes.
- API version source: `contracts/openapi/openapi.yaml`; handwritten HTTP adapters today, codegen scheduled.
- SQL schema with foreign keys, checks, indexes, timestamps UTC; sequential checksummed forward-only migrations. Migration runner owns each DB transaction and a SQL Server `sp_getapplock` application lock.
- `users.id` stable internal UUID — future OIDC will map `sub -> users.id`, not replace all existing primary keys.
- All protected queries scope to server-resolved user ID. Foreign conversation ID must return non-enumerating 404.
- Client UUID `clientTurnId` ensures idempotent turns and is scoped by conversation + actor; never trust an ID for ownership.
- Network LLM calls MUST NOT execute inside a SQL transaction. User and assistant messages commit together only after safe output.
- `mock` AI for local synthetic testing only; the OpenAI-compatible adapter receives server-only env config.
- No SSE in starter; JSON endpoint exists. Do not claim streaming is implemented.

## Private-data and AI constraints

- No sensitive input in logs, errors, exception messages, metrics labels or CI artifacts.
- Do not add external telemetry automatically.
- Prompt version and model provider are config-driven. Message text supplied to LLM is untrusted; never let it control backend secrets, other users, permissions, or role scope.
- Simple keyword precheck and output phrase checks are **placeholder safety routing** and unsafe as a substitute for evaluated mental-health safeguards.
- The database stores synthetic conversational text in plaintext for internal tests only. Privacy-hardening is an explicit gate before real content.

## Deployment restrictions

- `ALLOW_DEV_LOGIN=true` explicitly enables synthetic tester sessions in `local`, `stage`, `staging` or `staging-v2`. Production/unknown environments reject it. Compose publishes API and DB on host loopback.
- Never open the local development server to the public Internet. If needed for team-only testing behind VPN/private ingress, add proper perimeter protection first.
- Without the explicit dev-login flag, only health routes are mounted. No production auth exists yet.
- The API and migration binary accept split `DATABASE_*` settings. Staging connection values come from its ConfigMap and Secret; local settings remain on the independent `wellbeing` database. `DATABASE_URL` is retained for existing local environments when no split fields are set. No staging database credentials or addresses are compiled into the application.

## Operational checks

`make arch-check`, `make contract-check`, `make test`, `docker compose up --build -d` (with real dependencies), database migration checksums, manual cross-user smoke test.
