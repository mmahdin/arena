# Local Development and API Smoke Test

Prerequisites Docker/Compose. Environment is explicitly **not** Internet-facing.

```sh
cp .env.example .env
# Replace DEV_SESSION_SECRET with 32+ random bytes and MSSQL_SA_PASSWORD with a strong password; do not commit .env
docker compose up --build -d
curl -s localhost:8080/health/live
curl -s localhost:8080/health/ready
curl -s localhost:8080/api/v1/dev/users
```

Set `LLM_PROVIDER=mock` for default synthetic reply. `LLM_PROVIDER=openai-compatible` requires an approved server-side key and data-handling policy. Use fake discussion topics only. Apply migration `0005_dev_sessions.sql` before using the updated API (`make migrate` or migration container). The new session ID requires re-selecting a tester after upgrading. Logout and switching testers revoke their previous signed cookies in SQL Server. If SQL Server is unavailable, protected endpoints fail closed with `503 SESSION_UNAVAILABLE`.

The API's `Cookie` path is `/api/v1` and frontends should proxy API same-origin in local development. Set `APP_ALLOWED_ORIGIN` if needed, but do not rely on Origin as authentication.

To run native Go commands with root `.env` loaded literally (no shell evaluation):

```sh
make migrate
make api
# In another terminal:
make build
make integration-test
make smoke
```

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
