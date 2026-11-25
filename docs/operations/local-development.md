> Current local entrypoint (AUTH-09 + Unified/Background Intelligence):
> Dev Session authentication is retired. Use Keycloak SMS through the User BFF.
> The older setup notes below are historical where they describe Dev Login.

## Run backend and frontend together

Keep `ai-therapist-backend` and `ai-therapist-frontend` as sibling directories.
With the existing private `.env`, `.env.auth`, SMS-enabled Keycloak image and
Go/Node/Docker dependencies configured, run from the backend root:

```sh
./theia-dev.sh
```

The launcher starts SQL Server, Keycloak/PostgreSQL, Go API (`127.0.0.1:18080`),
Vite (`127.0.0.1:5173`) and the same-origin Auth Edge (`127.0.0.1:8080`).
Open **http://app.localhost:8080**. Keep the terminal open; Ctrl+C stops the
native services and leaves the Docker dependencies running. Do not launch a
second copy while these ports are occupied. Service logs default to
`~/.local/state/theia-dev`; override with `THEIA_LOG_DIR` if needed.

On Linux, ensure these local aliases resolve before starting Go. If missing,
add them once with the host administrator's permission:

```sh
echo '127.0.0.1 app.localhost auth.localhost admin.localhost' | sudo tee -a /etc/hosts
```

The launcher does not run migrations. With the local database available, apply
new checked migrations before starting an updated API:

```sh
make migrate
```

This version requires migrations through `0042_background_intelligence.sql`.
Local mock SMS delivery writes OTPs inside the private Keycloak container; no
real SMS is sent. Use synthetic accounts and content only. See the
[SMS setup guide](auth-sms-otp-auth05.md) and
[live browser acceptance guide](../../scripts/auth/verify_tia_login_live.py).

---

# Local Development and API Smoke Test

Prerequisites Docker/Compose, Python 3.9+ and Make on Linux/macOS. Environment is explicitly **not** Internet-facing.

```sh
make env-sync
# Replace DEV_SESSION_SECRET with 32+ random bytes and MSSQL_SA_PASSWORD with a strong password; do not commit .env
make up
curl -s localhost:8080/health/live
curl -s localhost:8080/health/ready
curl -s localhost:8080/api/v1/dev/users
```

### Optional Keycloak / PostgreSQL identity infrastructure (AUTH-01)

The existing `docker compose up` continues to launch only the legacy API and
SQL Server. To run the new **local-only** Keycloak service, use the separate
[`Keycloak local setup guide`](keycloak-local.md) and the opt-in
`compose.auth.yaml` overlay. The legacy Dev Session stays available only for
synthetic testing until the cross-repository Auth cutover. No BFF login or SMS
OTP implementation is included in this infrastructure step.

Set `LLM_PROVIDER=mock` for default synthetic reply. `LLM_PROVIDER=openai-compatible` requires an approved server-side key and data-handling policy. Use fake discussion topics only. Apply migration `0005_dev_sessions.sql` before using the updated API (`make migrate` or migration container). The new session ID requires re-selecting a tester after upgrading. Logout and switching testers revoke their previous signed cookies in SQL Server. If SQL Server is unavailable, protected endpoints fail closed with `503 SESSION_UNAVAILABLE`.

The API's `Cookie` path is `/api/v1` and frontends should proxy API same-origin in local development. Set `APP_ALLOWED_ORIGIN` if needed, but do not rely on Origin as authentication.

### AUTH-02 transition: AI Core and API routes without Dev Login

`ALLOW_DEV_LOGIN=false` now disables the **legacy** `/api/v1/dev/*` endpoints
only. Existing product APIs still mount but return `401 SESSION_REQUIRED` until
a verified OIDC user guard is integrated. The reserved `/auth/user/*` and
`/auth/staff/*` routes return `503 AUTH_NOT_CONFIGURED` and do **not** create a
session. Setting `AI_CORE_CONFIG_FILE` loads the AI runtime independently of
Dev Login, and `/health/ready` checks the running core regardless of the
legacy flag. See [AUTH-02 route boundaries](auth-composition-transition.md)
for the exact behavior, local-draft/inspector safeguards and BFF host plan.

To run native Go commands with root `.env` loaded literally (no shell evaluation):

```sh
make migrate
make api
# In another terminal:
make build
make integration-test
make smoke
```

Every `run_local.py` launch (including `make api`, `make migrate` and integration
targets) synchronizes `.env` with `.env.example` before loading it. `make up`
synchronizes before invoking Compose. Missing keys are appended with the template
defaults; existing values, empty assignments, custom keys, secrets and comments
are preserved. Concurrent launches serialize updates. If `.env` is absent, it is
created with private `0600` permissions. `make env-sync` does just this setup step.
New credentials need a one-time value; changed defaults and removed keys never
overwrite or delete existing settings. Exported shell variables take precedence
for the launched command and do not get written into `.env`.

`make api` also enables the local AI inspector inside the API process, overriding
the `.env` inspector flag for this command. It requires a local environment and
loopback listener, and writes traces to `apps/api/.local/ai-inspector/` by default.
Use `AI_LOCAL_INSPECTOR=false make api` for an API without inspection. Direct
`python3 scripts/run_local.py go run ./cmd/api` uses the inspector setting from
`.env` or the exported environment. See the [inspector guide](../ai-core/local-inspector.md).

Run `make test-scripts` to verify synchronization, preservation, concurrent
launches and Makefile behavior without starting Go, SQL Server or a model.

### Stop and clean synthetic state

```sh
docker compose down
# DESTRUCTIVE: remove local database volume
docker compose down -v
```

### Failure triage

- `ALLOW_DEV_LOGIN` error: only local/staging environments may enable it, with a signing secret of at least 32 bytes. Production remains disabled.
- Session rejected: check `DEV_SESSION_SECRET`, cookie path, local API proxy, reload session after server secret rotation.
- Database errors: check mssql health and migration container logs.
- API responds `503`: check provider credentials, model reachability and status.
- Model safety: mock works without external API; crisis keyword filtering is a testing placeholder and is not clinically reliable.

For SQL Server local development, the migration service connects to `master` only to create the `wellbeing` database (if needed); subsequent migrations use the `wellbeing` database. The SQL Server 2022 Developer container may require at least 2 GiB of memory. The SQL connection uses `encrypt=disable` exclusively inside this local Docker setup. If your SA password has reserved URL characters, URL-encode it in the connection string or use a URL-safe strong password.

### Verified dependency/network fallback

`apps/api/go.sum` pins the downloaded Go modules. Docker uses BuildKit module/compiler caches and verifies checksums. If proxy.golang.org returns a regional 403, run:

```sh
docker compose build --build-arg GOPROXY=https://goproxy.io,direct
docker compose up --no-build -d
```

Do not start a native API and Compose API simultaneously: both use `127.0.0.1:8080`. The delivered stack uses Compose.

### Paired frontend

```sh
cd ../wellbeing-front
npm ci
npm run dev
# http://127.0.0.1:5173
```

The local `.env` files already contain generated secrets in this workspace and are ignored; do not overwrite them on routine restarts. Keep `LLM_PROVIDER=mock`. Fresh migration seed: tester 1 has one synthetic thread, tester 2 has an empty incomplete profile, tester 3 has two synthetic threads. Live browser tests complete tester 2’s profile and clean up their test conversation.

### Migration compatibility

`0001` and `0002` are unchanged. `0003` backfills and sequences message order; deploy it before the new API. `0004` adds three deterministic synthetic demo conversations and six messages. The checksummed runner applies each once, transactionally. Both fresh installation and upgrade from `0002` were exercised. No down migration: preserve the volume and use a forward fix; a code rollback can keep the extra column/default/index. Do not run destructive volume cleanup to roll back an upgrade.
