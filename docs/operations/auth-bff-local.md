# AUTH-03 — Local OIDC/BFF infrastructure and usage

**Status:** implemented as a backend foundation, **not** a completed user or
staff sign-in release. User realm's BFF client is disabled until explicit AUTH-06 SMS-only flow activation and signed user-session proof. Staff accounts can now be locally provisioned through AUTH-04; see
`docs/operations/staff-auth-local.md` for the safe bootstrap/login procedure. Staff/admin
API routes are separate BO-01/02 work. Legacy Dev Login is not removed.

## One-time setup (development only)

1. Apply AUTH-00, AUTH-01, AUTH-02 and this AUTH-03 patch in order.
2. Keep the AUTH-01 Keycloak service bound to local loopback; use its dedicated
   PostgreSQL. Use SQL Server migration `0024_auth_identity_sessions.sql` for
   the Theia BFF, and check the migration runner result.
3. If `.env.auth` does not exist, run
   `python3 scripts/auth/init_local_keycloak_env.py`. If it already exists
   from AUTH-01, run `python3 scripts/auth/ensure_local_bff_key.py`. Both avoid
   printing secrets. Neither rotates existing Keycloak credentials.
4. Start Keycloak and the existing SQL Server via the documented local Compose
   setup. Verify the two realm discovery endpoints and SQL migration status.
5. To explicitly enable **only the BFF infrastructure**, set
   `AUTH_BFF_ENABLED=true` in the shell/Compose environment; keep
   `AUTH_USER_LOGIN_ENABLED=false` until AUTH-06 verification; see `auth-user-sms-auth06.md`. Staff Login can now be
   enabled in a local environment only, after AUTH-04 provisioning; see the
   Staff runbook. To keep the old API running exactly as before, leave
   `AUTH_BFF_ENABLED=false`.

For a Compose-based API, combine the **three** files when you opt into the BFF:

```sh
# NOTE: Compose expands secrets in `config`; do not print its full output.
docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-bff.yaml \
  --profile auth config --quiet

docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-bff.yaml \
  --profile auth up -d keycloak

# Run migrations first; then start/recreate API with desired environment flags.
# `AUTH_BFF_ENABLED` defaults false even in this overlay.
docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-bff.yaml \
  --profile auth up -d --build migrate api
```

For `make api`, set `AUTH_BFF_ENABLED=true` and the local SQL Server settings in
`.env`. The launcher fills missing `AUTH_BFF_KEY_BASE64`,
`THEIA_USERS_BFF_CLIENT_SECRET` and `THEIA_STAFF_BFF_CLIENT_SECRET` from
`.env.auth`. Exported values take precedence over `.env`, which takes
precedence over `.env.auth`. When running the Go binary directly, supply the
same settings via a managed environment loader. The local origins default to:

| Surface | Origin or issuer |
|---|---|
| Keycloak issuer base | `http://auth.localhost:8180` |
| User BFF | `http://app.localhost:8080` |
| Staff BFF | `http://admin.localhost:8080` |

The same Go listener may answer app/admin hosts, but Auth code checks exact
host, scheme, OIDC issuer/client and cookie name independently. Browser and
Docker must resolve all three `.localhost` names to the correct loopback /
Compose network destinations. HTTP is **local-only**; use reviewed HTTPS
front-door and `__Host-` cookies before nonlocal deployment.

## Session API contract for the web app

- `GET /auth/{user|staff}/login` redirects (`302`) to Keycloak, after creating
  a one-time encrypted login transaction with a 5-minute login cookie.
- `GET /auth/{user|staff}/callback` verifies OIDC state/cookie/PKCE/nonce/JWT;
  after resolving the active identity, stores a revocable SQL session, sets a
  host-only HttpOnly cookie and redirects (`303`) to the fixed `/` path.
- `GET /auth/{user|staff}/session` returns `401` if unauthenticated, else
  `{authenticated, principalType, principalId, expiresAt, csrfToken, role?}`.
  `role: admin` appears only for active pre-provisioned staff identities.
- All unsafe requests made with the new session, including
  `POST /auth/{kind}/logout`, need `X-CSRF-Token` from the same session response.
  Logout revokes **only the Theia session** in AUTH-03 and returns `204`.
- `401` means no/invalid session, `403` forbidden (e.g. unauthorized staff or
  CSRF denied), `400` incorrect host, `503` disabled/unavailable Auth.
- User API guard still uses legacy Dev Session when `ALLOW_DEV_LOGIN=true`;
  AUTH-06 can switch to real User Auth after opt-in activation, with `ALLOW_DEV_LOGIN=false`. Never enable both user login paths.
- No JS code should parse or store OIDC tokens. The browser uses cookies and
  sends a CSRF header for modifying requests.

## Limits and release gates

- **No real users or admin workflows yet.** Staff provisioning is available
  locally (AUTH-04), but SMS
  challenge verification (AUTH-05/06), key rotation, Keycloak SSO/backchannel
  logout, number recovery and production TLS/VPN/TWA tests are separate gates.
- The SQL login transaction contains AES-GCM-sealed PKCE/nonce, not plaintext;
  preserve `AUTH_BFF_KEY_BASE64` securely across restarts or in-flight login
  transactions become unusable. Existing sessions still validate without the
  key for the cookie hash, but CSRF token derivation uses that key.
- Initial accounts never reuse existing synthetic dev-phone identities. Local
  OIDC users get an explicit synthetic marker; real-user enablement requires
  separate clinical/privacy/release approval.
- The Keycloak-provided issuer URL must remain stable between browser and
  backend. Never disable signature verification to fix hostname problems.
- No decrypted AI traces, OTPs, phone numbers, bearer cookies, OAuth tokens,
  auth headers or client secrets belong in logs/audit payloads.

## Validation

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v
cd apps/api
go build ./...
go vet ./...
go test ./...
```

Integration testing also needs disposable **SQL Server 2022** and the actual
AUTH-01 Keycloak/PostgreSQL containers. Pure unit tests cannot establish the
migration's runtime validity, staff provisioning flow or real-browser Cookie
behavior. Neither a passing Go unit suite nor Docker startup alone authorizes
real-user rollout.
