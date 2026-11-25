# AUTH-06 — User SMS Login, OIDC identity and product-session integration

**Scope:** backend + local Keycloak User realm. This ticket does **not** authorize real personal/clinical data, public deployments, Android TWA releases, or removing Dev Session (AUTH-07/08/09/10 own those gates).

## Identity and security boundaries

- `theia-users` is the only allowed User OIDC issuer; Keycloak verifies phone and OTP. Neither the Go API nor the SMS provider validates OTPs.
- After **successful OTP verification**, the Keycloak SPI writes the *user-session note* `theia_sms_verified=sms-otp-v1`. The dedicated `oidc-usersessionmodel-note-mapper` maps that note into the **signed User ID token** (not the access token or browser). The Go OIDC verifier refuses User ID tokens without the exact verified claim, even when issuer, audience, signature and nonce otherwise pass. Staff login is unaffected and has no SMS requirement.
- The local-only `scripts/auth/activate_user_bff.py --apply` refuses unexpected realm browser flows, non-REQUIRED/password/alternative executions, untrusted clients, client-specific flow overrides, nonexact redirects or proof mappers. It installs the client mapper and enables the previously disabled confidential User BFF client; Go remains off until an operator explicitly enables it.
- Go User Login requires **both** `AUTH_USER_SMS_FLOW_VERIFIED=true` (operator attestation after inspecting Keycloak) and `AUTH_USER_LOGIN_ENABLED=true`. These flags do not replace the signed token verification or the local Mock SMS browser smoke test.
- `ResolveIdentity` in `apps/api/internal/modules/auth/adapters/mssql/repository.go` (from AUTH-03) resolves `(issuer,sub)` in a serializable SQL transaction, creates a **new** internal `users.id` and `user_profiles` only when the subject does not exist, and reuses that ID on returning visits. It does not look up or merge by phone, email or `dev_phone_identities`. Rejected/deleted accounts must not be recreated by re-login.
- `internal/platform/principal.UserID` is now the shared, provider-neutral request-context accessor. `RequireUser` verifies the **active SQL-backed session** before setting it; transitional `devsecurity.Guard` can still set it while explicitly enabled. All User, Conversation, AI, Letter, Privacy and Bookmark handlers use it rather than depending on the Dev Session module. Existing owner-scoped repository methods are unchanged.
- A new User has no onboarding completion timestamp. The frontend reads `GET /api/v1/me` to decide whether to show onboarding; the existing conversation/AI service still returns `409 ONBOARDING_REQUIRED` for gated operations. The OIDC callback returns `303 /` regardless of onboarding state; no new redirect target or login-callback query parameter is trusted.
- `/auth/user/session` returns the opaque Theia user principal ID, expiry and CSRF token, not OTPs or OIDC access/refresh/ID tokens. Product **mutations** require `X-CSRF-Token` and a same-origin request. User and Staff hosts/cookies remain separate.

## Step 1 — Install the updated SMS SPI and verify the realm

Work in a **dedicated, local synthetic environment** with a disposable Keycloak/PostgreSQL and SQL Server. Apply existing migrations through **0030**. Keep both `/auth` Staff and User BFF flags off while modifying Keycloak.

```sh
python3 scripts/auth/ensure_local_sms_key.py
python3 scripts/auth/ensure_local_bff_key.py

docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-sms.yaml \
  --profile auth up -d --build keycloak

# Only on an existing local disabled theia-users-bff client (AUTH-05):
python3 scripts/auth/configure_user_sms_flow.py --apply
# Inspect the REQUIRED-only flow and client, make email/name optional,
# and verify the signed proof mapper before enabling User BFF:
python3 scripts/auth/activate_user_bff.py --apply
python3 scripts/auth/activate_user_bff.py  # read-only recheck
```

**Existing User client already enabled?** `configure_user_sms_flow.py` is an AUTH-05 setup command that intentionally rejects enabled clients; use the read-only AUTH-06 activation check, and investigate any drift rather than repeatedly recreating the flow. Neither script is a general-purpose remote Admin tool; both use the loopback Keycloak instance and trusted `.env.auth` bootstrap credentials.

Verify manually in Keycloak Admin UI: `theia-users` has exactly one REQUIRED `theia-sms-otp` browser execution; a User BFF **client-level** `User Session Note` mapper named `theia-sms-verification` maps note `theia_sms_verified` to signed ID-token claim `theia_sms_verified`, with ID token enabled and access token disabled. There is no username/password, direct grant or alternate User authentication path. **Do not** add a static mapper with the same claim or enable registration/password recovery paths.

## Step 2 — Local browser smoke with Mock SMS

Set in `.env` or the approved environment loader (do not export secrets in the shell history):

```dotenv
APP_ENV=local
ALLOW_DEV_LOGIN=false
AUTH_BFF_ENABLED=true
AUTH_STAFF_LOGIN_ENABLED=false
AUTH_USER_LOGIN_ENABLED=true
AUTH_USER_SMS_FLOW_VERIFIED=true
```

Use your existing `.env.auth` for the separate BFF key and both OIDC client secrets. Keep `THEIA_SMS_ENV=local`, `THEIA_SMS_TRANSPORT=mock`; **never** use a real user's phone number or real wellbeing data. Recreate the API using the auth-bff overlay and SMS-enabled Keycloak:

```sh
docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-sms.yaml -f compose.auth-bff.yaml \
  --profile auth up -d --build migrate api
```

1. In a clean local browser session, open `http://app.localhost:8080/auth/user/login` and enter a **synthetic test phone**; Keycloak presents its Persian RTL OTP challenge.
2. Read the test OTP from the local Keycloak log (`Local mock SMS OTP: <code>`) or the private container outbox `/tmp/theia-sms-outbox/` (AUTH-05 documentation); keep local OTP logs private and never expose the outbox through the browser/API or telemetry.
3. Enter the OTP. A valid challenge redirects to the app via `/auth/user/callback`, issues an HttpOnly user cookie, and returns `303 /`.
4. Open `http://app.localhost:8080/auth/user/session`: expect `authenticated=true`, `principalType=user` and a UUID. The response contains `csrfToken` for mutation requests.
5. Open `http://app.localhost:8080/api/v1/me`: expect the **same** UUID and incomplete onboarding for a new identity. Complete the existing onboarding flow using the User cookie and CSRF header; then confirm a conversation can be created. A second browser login for the **same Keycloak subject** must return the same internal UUID and existing onboarding profile.
6. Authenticate a second synthetic phone to a **different** Keycloak subject. Check that its User ID is different and it cannot read, edit, delete or retry the first account's conversations/messages/AI controls/bookmarks/memories.
7. Confirm User cookie on `admin.localhost:8080` fails, Staff cookie on User endpoints fails, Dev Session cookie on User endpoints fails, and missing/wrong `X-CSRF-Token` for mutations returns `403`.
8. `POST /auth/user/logout` with CSRF revokes the **local Theia session** and clears its cookie. Keycloak browser SSO logout and backchannel revocation remain AUTH-08.

## Failure modes, rollback and release restrictions

- If `AUTH_USER_SMS_FLOW_VERIFIED` is unset, **User BFF refuses startup** when User login is enabled. A missing/incorrect signed SMS proof causes callback `401 AUTH_CALLBACK_INVALID`; no internal user/session is minted.
- User realm, client or SPI drift: disable `AUTH_USER_LOGIN_ENABLED` first; this leaves Staff login operational. Disable the User client in Keycloak, then investigate rather than bypassing the signed-claim requirement.
- SQL Server unavailable: user resolution and session minting fail closed. OIDC state cannot be replayed; the user must start over after recovery.
- Do **not** claim production readiness from a passing isolated test. Real Keycloak 26.8.2 SPI compile/runtime, SMS vendor, browser, multi-replica, concurrency, session revocation, privacy and clinical release gates still need independent checks. See AUTH-07/08/10.
- Never migrate synthetic `dev_phone_identities` to real accounts by matching phone-like values, and do not remove the legacy Dev Session implementation before the frontend cutover in AUTH-09.

## Validation

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v
(cd apps/api && go build ./... && go vet ./... && go test ./...)
mvn -f infra/keycloak/sms-spi/pom.xml verify
```

The optional destructive-disposable integration test `TestAUTH06FirstLoginAndReturningUserDisposableSQL` requires a throwaway SQL Server 2022 instance with migrations through 0030 and explicit opt-in:

```sh
THEIA_DISPOSABLE_USER_SQL_DSN='sqlserver://...DISPOSABLE_DATABASE...' \
THEIA_DISPOSABLE_USER_SQL_ACK=YES_DESTROY_AFTER_TEST \
  sh -c 'cd apps/api && go test ./internal/modules/auth/adapters/mssql -run TestAUTH06FirstLoginAndReturningUserDisposableSQL -v'
```

The test writes synthetic records and deletes an account; destroy the disposable DB afterward. It is skipped by default. The actual Keycloak+PostgreSQL+SQL Server browser journey must also be executed locally before you turn on User login even for a shared test installation.
