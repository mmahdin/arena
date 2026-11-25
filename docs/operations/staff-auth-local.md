# AUTH-04 — Local Staff Login and Provisioning

**Scope:** backend-only local Keycloak Staff login with a fixed `admin` capability.
This is **not** the Back-Office UI or `/api/v1/admin/*` API (BO-01/02), and it
is not approval for deployment to public networks or real user data.

## Preflight

1. Apply AUTH-00, AUTH-01, AUTH-02 and AUTH-03 before this patch.
2. Start SQL Server and Keycloak/PostgreSQL from the existing Compose overlays.
   Run SQL Server migrations **through `0025_staff_provisioning_audit.sql`**.
3. The Keycloak realm `theia-staff` has registration disabled, uses manually
   provisioned email/password credentials, 14+ character passwords, brute-force
   protection, and no MFA per the current **local/private-only** decision. The
   `theia-users` BFF client remains **disabled** until the SMS Authenticator.
4. `AUTH_BFF_ENABLED=true` and `AUTH_STAFF_LOGIN_ENABLED=true` must be explicitly
   enabled for the API process. Leave `AUTH_USER_LOGIN_ENABLED=false`.
   `ALLOW_DEV_LOGIN=true` remains transitional for synthetic user flows until
   the cross-repository AUTH-09 cutover; it does **not** grant Staff access.

## Create the first Staff account (no seeded user or shared password)

1. Open `http://auth.localhost:8180/admin/` with the separately generated
   Keycloak **bootstrap administrator** credentials in your private `.env.auth`.
   That account is for managing Keycloak itself, NOT for entering Theia Back-Office.
2. Select the **`theia-staff`** realm (NOT `master` or `theia-users`). In
   **Users → Add user**, create a named staff account with its individual email.
   Set a unique password of at least 14 characters under **Credentials**.
   Keep it enabled. Do not create an imported realm-wide default staff password.
3. Inspect that new user's **Keycloak user ID** in the user details. Copy the
   opaque UUID only (NOT the email, password, client secret, or access token).
4. Run the local bootstrap tool to bind that already-existing user to Theia's
   staff allowlist. **Never point it at a production database.** Both the tool
   and its Keycloak issuer are restricted to local hosts and `APP_ENV=local`.
   The tool is idempotent: repeating a successful provision preserves the same
   `staff_accounts.id` without recording a second creation event.

### Option A: Docker Compose (recommended if API and SQL are in Compose)

The Auth overlay only adds environment to `api`; no account is seeded until
this command is executed. Use the three Compose files and both private env
files. Put `AUTH_BFF_ENABLED=true` and `AUTH_STAFF_LOGIN_ENABLED=true` in your
local `.env` or export them in the terminal before recreating the API.

```sh
# All commands from the backend repository root.
python3 scripts/auth/init_local_keycloak_env.py
# If already created, the initializer never overwrites .env.auth.
python3 scripts/auth/ensure_local_bff_key.py

docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-bff.yaml \
  --profile auth up -d keycloak

docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-bff.yaml \
  --profile auth up -d --build migrate

docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-bff.yaml \
  --profile auth build api

# Replace UUID with the Keycloak *staff realm user ID* from step 3.
docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-bff.yaml \
  --profile auth run --rm --no-deps api \
  /app/staffctl -action=provision -subject=00000000-0000-4000-8000-000000000001 -confirm

# Recreate API with both BFF/Staff flags explicitly enabled.
docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-bff.yaml \
  --profile auth up -d --build api
```

**Important:** The example UUID is a placeholder; use the real Keycloak Staff
User ID. Do not paste the secret-bearing result of `docker compose config` into
logs or bug reports.

### Option B: Go binary on the host

If you run Go directly, supply `APP_ENV=local`, a loopback-only `DATABASE_URL`,
and the same `AUTH_KEYCLOAK_BASE_URL` as the API. From `apps/api`:

```sh
go run ./cmd/migrate
APP_ENV=local go run ./cmd/staffctl -action=provision \
  -subject=00000000-0000-4000-8000-000000000001 -confirm
```

This assumes your secret and SQL connection variables are already set through a
trusted env loader. Never commit a `.env` containing credentials.

## Browser smoke test

1. Visit `http://admin.localhost:8080/auth/staff/login` in a browser.
   The BFF should redirect to Keycloak `theia-staff`.
2. Sign in with the **staff** email/password from Keycloak, not with the
   Keycloak bootstrap administrator account. OIDC callback must validate the
   issuer, audience, nonce, PKCE verifier, state, and correct host.
3. The BFF callback redirects back to `http://admin.localhost:8080/` with a
   host-only `theia_staff_session_local` **HttpOnly** cookie. There is no
   Back-Office frontend yet, so `/` may return 404: this is expected.
4. Open `http://admin.localhost:8080/auth/staff/session`. The JSON response
   should show `authenticated:true`, `principalType:"staff"`, and
   `role:"admin"` plus a CSRF token. The response must **not** include a
   Keycloak token, password, or SMS OTP.
5. POST `/auth/staff/logout` **from the same host**, with that CSRF token as
   `X-CSRF-Token` and a matching `Origin` header. Expect `204`; a subsequent
   `/auth/staff/session` must return `401`. This revokes the Theia SQL session;
   Keycloak SSO/logout propagation will be added in AUTH-08.
6. Sign in with a *different* Keycloak staff user who was **not** allowlisted
   through `staffctl`: the OIDC callback must reject them with **403** and
   `STAFF_NOT_AUTHORIZED`. A normal `theia-users` login/cookie cannot become
   a Staff session. `AUTH_USER_LOGIN_ENABLED` remains off.

**Diagnostics:** `503 AUTH_NOT_CONFIGURED` means the staff flag or BFF isn't
active. `400 AUTH_ORIGIN_MISMATCH` usually means the browser used
`localhost:8080` instead of **`admin.localhost:8080`** (or the reverse proxy
is misconfigured). `403 STAFF_NOT_AUTHORIZED` usually means issuer/subject
mapping is absent, inactive, or wrong. Never relax issuer/host checks to fix
these errors.

## Disable and reactivate Staff

To remove access, **deactivate in SQL first**, then disable the corresponding
Keycloak user through the admin console. Disabling via `staffctl` atomically
sets `staff_accounts.active=0`, revokes all its active Theia sessions, and
records an append-only audit event. An already-issued browser Cookie becomes
invalid immediately on the next request.

```sh
# Within the same local docker compose run command as above, replace the tail:
/app/staffctl -action=status     -subject=<keycloak-user-uuid>
/app/staffctl -action=deactivate -subject=<keycloak-user-uuid> -confirm
/app/staffctl -action=reactivate -subject=<keycloak-user-uuid> -confirm
```

Reactivation must be deliberate: first check the person's Keycloak status and
operational authorization, then reactivate in SQL. **Old sessions remain
revoked**, even after reactivation. Keycloak re-enable/disable is a separate
operator action; the BFF does not yet subscribe to Keycloak account disable
and backchannel logout events (AUTH-08).

Provisioning audit records the affected staff account but **does not** falsely
attribute the action to the staff member; local CLI operator attribution must
be handled by your local access/operations controls. Auth audit is different
from the future Back-Office `admin_audit_events` (BO-03).

## Security and deployment requirements

- Do not use this CLI against a nonlocal database, shared environment or
  production. It requires explicit `-confirm` on mutations. A future admin
  workflow will have authenticated operator identity and purpose/audit fields.
- Never publish either Staff Login or Keycloak Admin Console on the public
  internet without the separate private ingress/VPN, TLS and security gates.
  No MFA is configured at the team's request for the current local scope;
  the risk must be revisited if the environment changes.
- Staff capability is the fixed role `admin`, with access to every **implemented
  administrative feature** once BO-01/02 installs its separate API. It does not
  grant access via `/api/v1/me/*` or turn staff into product users.
- Run Go and SQL integration checks on an isolated disposable SQL Server 2022
  database; no caller-owned mental-health data is used for these tests.

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v
cd apps/api
go build ./...
go vet ./...
go test ./...
# Separately, with a disposable local SQL Server and safe local DATABASE_URL:
# WELLBEING_INTEGRATION=true APP_ENV=local go test ./tests/integration -run TestAuth04StaffProvisioningSQL -v
```
