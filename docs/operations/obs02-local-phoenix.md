# OBS-02 — Authenticated Phoenix for synthetic local AI tracing

**Status:** The private local container, initial admin login, authenticated
OTLP ingestion, trace persistence and Go metadata export were verified.
OBS-03 Staff trace references are planned.

## Trust boundary and architecture

```text
Developer browser --[127.0.0.1:6006, Phoenix login]--> Phoenix v22.0.1
       (Phoenix UI + authenticated OTLP/HTTP)              |
                      ^                                  SQLite
                      | Bearer system key             Docker named volume
                      |
Native Go API --[HTTP_ADDR=127.0.0.1:8080, opt-in local exporter]
          |
          +-- Existing SQL Server (durable Turns/attempts; NO wire bodies)
```

This is a **separate Compose project**. It does not create a second application
API, change Keycloak, expose a Phoenix HTTP endpoint on `app.localhost` or
`admin.localhost`, open port 4317, or proxy provider payloads through generic
telemetry collectors. The only published port is `127.0.0.1:6006:6006`.
Port binding alone is not adequate security on a compromised developer host:
Phoenix also requires local login and Bearer authentication for OTLP.

Phoenix uses **SQLite** under `/var/lib/phoenix` with the persistent Docker
`phoenix_data` volume. Do not run this variant on a shared or public server.
`PHOENIX_DEFAULT_RETENTION_POLICY_DAYS=7` is configured as the *default*
policy; confirm retention enforcement in the running version/project before
claiming traces are automatically erased. All payloads must be fictional.

Pinned vendor version: `arizephoenix/phoenix:version-22.0.1`. It is not `latest`.
Image updates require review of auth, migrations, retention and smoke tests.

## 1. Generate untracked local credentials

From the repository root:

```sh
python3 scripts/observability/init_local_phoenix_env.py
```

Creates `.env.observability` with mode `0600` and randomized Phoenix signing
secret and **nondefault** bootstrap admin password. Does not print or overwrite
credentials. Never commit or share this file. The committed
`.env.observability.example` contains names only (no working secrets).

`.env`, `.env.auth`, and `.env.observability` are ignored and were absent
from the Git index when checked.

The initial Phoenix admin is `admin@localhost`; use the password from the
owner-only `.env.observability` in your local editor/password manager, then
change it in the UI as appropriate. Because the first-start password is only
used when provisioning the user, editing the file later does not rotate an
existing user password. `PHOENIX_USE_SECURE_COOKIES=false` is acceptable
**only** because this deployment uses plain HTTP bound to loopback. Public or
shared deployment requires HTTPS, secure cookies, private ingress and a new
security review.

## 2. Start Phoenix independently of the Go/SQL stack

```sh
docker compose -p theia-obs02 --env-file .env.observability \
  -f compose.observability.dev.yaml --profile observability config --quiet

docker compose -p theia-obs02 --env-file .env.observability \
  -f compose.observability.dev.yaml --profile observability up -d phoenix

docker compose -p theia-obs02 --env-file .env.observability \
  -f compose.observability.dev.yaml --profile observability ps
```

Do not run `docker compose config` without `--quiet`: expanded service config
can display Phoenix credentials. Do not combine this file with `compose.yaml`
in the ordinary API stack. The native launcher posts to Phoenix on host loopback.

Open `http://127.0.0.1:6006` and sign in as `admin@localhost`. Verify that a
logged-out browser cannot see any trace or use Phoenix's API. `GET /healthz`
is intentionally public; it is not an authentication test.

Basic negative smoke (no ingestion key required):

```sh
python3 scripts/observability/smoke_phoenix.py --preflight
```

This checks health and requires unauthorized `GET /v1/projects` and
`POST /v1/traces` to return `401/403`. Any unexpected success is a **security
failure**: stop, inspect the Phoenix configuration, do not enable capture.
The smoke uses numeric loopback, no HTTP proxy and no redirect following.

## 3. Create and store a Phoenix API key without shell history

Inside the authenticated Phoenix UI, create a **system API key** in Settings.
Phoenix v22 uses Bearer authorization on `/v1/traces`. Copy the new key and
run:

```sh
python3 scripts/observability/init_local_phoenix_env.py --set-ingest-key
```

Paste the key into the hidden prompt. It is stored only in the owner-only
`.env.observability`, not in a command argument, terminal echo or log. The
helper rejects replacing an existing key; to rotate, revoke the old key in
Phoenix and deliberately update the local file. Never paste the key into a
chat, screenshot, `.env.example`, frontend variable or git commit.

**Caution:** a Phoenix system API key is not an ingestion-only capability;
local access to the key can grant wider Phoenix API access. This first release
uses it only on an isolated synthetic development host. For a team-shared
system, investigate scoped service credentials and independent user roles.

Perform a real **authenticated ingestion and readback** smoke:

```sh
python3 scripts/observability/smoke_phoenix.py --ingest
```

The script sends one 100%-synthetic metadata-only OTLP/protobuf span and
checks that it can be retrieved from the `theia-obs02-smoke` project via an
authenticated Phoenix REST API. It prints an opaque 32-character TraceID, **no
secret and no prompt/response content**. A `200/202` OTLP response alone is
not considered sufficient; query readback must succeed.

Verify persistence and that the authentication configuration survived restart:

```sh
docker compose -p theia-obs02 --env-file .env.observability \
  -f compose.observability.dev.yaml --profile observability restart phoenix

python3 scripts/observability/smoke_phoenix.py \
  --after-restart <TRACE_ID_FROM_INGEST>
```

## 4. Run the native Theia Go API with authenticated metadata tracing

AUTH-09 retired the Dev Session identity path. OBS-04A independently enables
full provider wire capture for a loopback-only local API. The native launcher
defaults to metadata-only and adds `--content` for an explicit test-only opt-in.
Both modes retain the Keycloak/BFF configuration supplied by the environment.

Ensure local SQL Server is running and migrations are current. When BFF is
enabled, the native Go process must also resolve the configured Keycloak issuer.
Start the service with:

```sh
python3 scripts/observability/run_native_with_phoenix.py
# For fictional test accounts/messages ONLY; exports full provider wire bodies:
python3 scripts/observability/run_native_with_phoenix.py --content
```

The launcher checks health, denial of unauthenticated access, and the system
key's access before starting Go. It passes the ingestion key through the process
environment. Go restricts this key to numeric loopback HTTP on port 6006 and
uses a direct connection with redirects disabled.

With your local AI runtime configured, exercise a synthetic AI Turn and inspect
the spans in Phoenix's default project. The default mode shows execution metadata without provider bodies;
`--content` additionally exports scrubbed, bounded request and response bytes. The smoke script uses a separate
`theia-obs02-smoke` project.

The local test operator must never enter real private or clinical disclosures.
Full content does not grant extra identity permissions or Staff API access.
OBS-03 already supplies the audited Turn-to-TraceID mapping for Back Office.

## 5. Shutdown, backup and failure handling

```sh
docker compose -p theia-obs02 --env-file .env.observability \
  -f compose.observability.dev.yaml --profile observability stop phoenix
```

`down` (without `-v`) removes the container/network but preserves the named
SQLite volume. **Never use `down -v`** unless deliberately disposing of all
fictional test traces, accounts and keys. Docker host bind must remain
`127.0.0.1:6006` and no private prompt/response should be transmitted to any
remote Collector endpoint. If the service is unavailable, the Go tracing
exporter may drop spans; this is **not** evidence that no Turn ran. Use SQL
attempt records as the durable operational source of truth.

For a future shared service, use a real secrets manager, Postgres, TLS, scoped
access, explicit retention/backup/restore and a fresh privacy review. This
local Compose project is not a production or clinical deployment.

## Verified vs. pending

- Verified Compose startup and loopback binding, initial admin login, anonymous
  read/write denial, authenticated OTLP readback and persistence after restart.
- Verified a synthetic span from the Go exporter with no prompt marker or
  ingestion key in the trace. Native API liveness returned HTTP 200.
- A complete AI Turn and provider retry trace remain to be tested with the
  AI runtime configured.

Docs: <https://arize.com/docs/phoenix/deployment/authentication>,
<https://arize.com/docs/phoenix/self-hosting/deploying-phoenix>,
<https://github.com/Arize-ai/phoenix/releases/tag/arize-phoenix-v22.0.1>.
