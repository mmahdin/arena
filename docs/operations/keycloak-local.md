# Local Keycloak + PostgreSQL (AUTH-01)

**Scope:** local, synthetic-only identity infrastructure. This is NOT a production
configuration. `start-dev` disables important production protections; never
expose its listener, admin console, or PostgreSQL to the internet. Shared team
hosting needs a separate TLS/private-VPN deployment design (AUTH-10).

This is an opt-in Compose **overlay**, not a replacement for the existing SQL
Server/API stack. The Keycloak database is PostgreSQL and is completely
separate from SQL Server. Keycloak manages its own schema; do not run the Go
SQL Server migration runner on PostgreSQL.

## Start, using freshly generated local credentials

Prerequisites: Docker with the Compose V2 plugin, Python 3, and a local `.env`
for the existing stack. The auth overlay pins Keycloak `26.8.2` and PostgreSQL
`17.12` to reviewed patch versions.

```sh
# In backend repo root (existing local DB/API secrets remain in .env):
test -f .env || cp .env.example .env
python3 scripts/auth/init_local_keycloak_env.py

# Always include BOTH compose files and BOTH env files for this overlay.
docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml --profile auth config --quiet

docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml --profile auth up -d keycloak

docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml --profile auth ps
```

The generator creates `.env.auth` **once**, permissions `0600`, with four
independent cryptographically random secrets. It never prints or overwrites
credentials. `.env.auth.example` shows only variable names; `.env.auth` is
Git-ignored. Do not paste the output of `docker compose config` into tickets:
Compose expands secrets in its output. Keep `.env.auth` backed up as a local
secret; losing/mismatching the DB password can prevent connecting to the
existing PostgreSQL volume. Do not track `.env` either.

## Verify readiness and realm discovery

The Compose readiness check checks PostgreSQL with `pg_isready` and Keycloak
on its **unpublished** management port (`9000`). Keycloak's public test HTTP
listener is bound **only** on the host loopback address at port `8180`.

```sh
# Should respond with one issuer for each independent realm.
curl -fsS http://auth.localhost:8180/realms/theia-users/.well-known/openid-configuration
curl -fsS http://auth.localhost:8180/realms/theia-staff/.well-known/openid-configuration

# Admin console, LOCAL only:
# http://auth.localhost:8180/admin/
# Credentials are in .env.auth; never commit or publish them.
```

If `.localhost` subdomains do not resolve on your machine, add these names to
its **local loopback** hosts configuration: `auth.localhost`, `app.localhost`,
`admin.localhost`. All three must resolve to `127.0.0.1`. Browser and API
container will eventually use the same issuer URL; the Compose network alias
`auth.localhost` resolves to the Keycloak container on port `8180` for a
containerized Go BFF. Do not change the issuer/hostname independently for
browser vs API: JWT issuer validation depends on an exact match.

After first boot, check in the local console that the two realms and clients
exist; no real users or staff accounts are imported:

| Realm | BFF client | State in AUTH-01 | Exact callback allowlist |
|---|---|---|---|
| `theia-users` | `theia-users-bff` | **Disabled** until SMS SPI (AUTH-05) | `http://app.localhost:8080/auth/user/callback` |
| `theia-staff` | `theia-staff-bff` | Enabled; **no staff is provisioned** until AUTH-04 | `http://admin.localhost:8080/auth/staff/callback` |

Both clients are confidential, use distinct secrets interpolated into the
realm imports at startup, and require Authorization Code + PKCE S256. No
implicit/password grant, wildcard redirects, public staff sign-up, or seeded
credentials. `http://*.localhost` callbacks are **local design targets**, not
working API endpoints yet. AUTH-02/03 will implement the BFF endpoints; AUTH-04
will provision staff; AUTH-05 will install the audited SMS Authenticator and
enable the user client. A regular Keycloak login screen is **not** SMS OTP.

For local integration, `app.localhost:8080` and `admin.localhost:8080` are two
hostnames pointing to the existing Go listener once the corresponding BFF
routes exist. An eventual frontend origin/reverse proxy must maintain those
strict host and callback boundaries, or update the explicit Keycloak client
allowlists through a reviewed configuration change. For Web/TWA and shared
hosting, use HTTPS and do not reuse these HTTP development callbacks.

## Persistence and change management

- PostgreSQL data persists in the Compose volume `theia_keycloak_pg`. Its port
  is never published to the host. Keycloak schema changes are Keycloak-owned.
- Startup `--import-realm` **skips realms that already exist**. Editing imported
  JSON files or rotating a client secret in `.env.auth` does **not** update an
  existing realm in the database. Treat subsequent realm changes as explicit,
  reviewed admin configuration operations (or a disposable test-environment
  rebuild); do not rely on restart to overwrite configuration.
- The bootstrap admin account is intended for local Keycloak administration;
  it is **not** a Theia staff identity and cannot access `/api/v1/admin/*`.
- If you need to restart: run the same Compose command with `restart keycloak`
  or `up -d keycloak`; the PostgreSQL volume is retained.
- To stop only these two containers, use the same Compose prefix followed by
  `stop keycloak keycloak-postgres`.
- Never use `down -v` as a troubleshooting step: it destroys databases,
  including the **existing SQL Server** volume when both files are loaded.
- For later deployments, provision secrets via a managed secret store, plan
  PostgreSQL backup/restore, use optimized Keycloak with TLS, and review
  session/logout semantics in ADR 0012. This local overlay is not deployable
  as-is in staging/production.

## Troubleshooting and verification limits

- `config --quiet` fails on a required variable: ensure `.env.auth` exists and
  pass `--env-file` flags before both `-f` files as shown.
- Database is healthy but Keycloak not ready: check Keycloak logs via the same
  Compose prefix plus `logs --tail=100 keycloak`; the readiness probe permits
  a longer first boot.
- Realm client secret changes not visible: the initial realm import is
  deliberately non-destructive, see the persistence rule above.
- Browser displays an unexpected issuer: verify Keycloak's fixed hostname and
  local `.localhost` host resolution; don't disable issuer validation.
- Run local config tests with `python3 -m unittest discover -s scripts/tests -v`.
- Docker runtime/realm import verification requires a workstation or CI runner
  with Docker. Static JSON tests and Python structural checks alone do not
  establish that Keycloak actually booted or successfully imported clients.

Official references:
[Keycloak container](https://www.keycloak.org/server/containers),
[health checks](https://www.keycloak.org/server/health),
[realm startup import](https://www.keycloak.org/server/importExport),
[hostname](https://www.keycloak.org/server/hostname).
