# AUTH-05 — Keycloak passwordless SMS authenticator (LOCAL / synthetic only)

## Scope, deployment gate, and ownership

- This change **builds** a custom Java Keycloak `AuthenticatorFactory` (`theia-sms-otp`) for Keycloak **26.8.2** and a separate Persian/RTL Keycloak login theme.
- It is installed **only** with `compose.auth-sms.yaml`, layered on existing `compose.yaml` and `compose.auth.yaml`. `theia-staff` realm and Staff BFF remain unchanged.
- `theia-users-bff` stays **disabled** and Go `AUTH_USER_LOGIN_ENABLED=false` remains required. AUTH-06 must validate the complete realm + OIDC + SQL session journey before enabling real users. This PR is not permission to use real phone numbers or mental-health data.
- **No SQL Server migration.** OTP state is managed by Keycloak, using `SingleUseObjectProvider` for atomic single-use challenge consumption and globally shared `putIfAbsent` rate-limit slots. The PostgreSQL DB stays Keycloak-owned.
- The Keycloak startup JSON import is intentionally **unchanged**: an already imported realm is not updated by editing JSON. An explicit idempotent admin script selects the SMS-only browser flow *without* enabling the User client.

## Build and start

The Docker build uses Maven inside a multi-stage image. On a machine with Maven/JDK21 you can also run `mvn -f infra/keycloak/sms-spi/pom.xml verify`, but installation on the host is not required.

```sh
# From backend repo root; the existing .env and .env.auth must already exist.
python3 scripts/auth/ensure_local_sms_key.py

# Review the composed configuration WITHOUT printing secrets to shared logs.
docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-sms.yaml \
  --profile auth config --quiet

docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-sms.yaml \
  --profile auth up -d --build keycloak
```

The local overlay pins `THEIA_SMS_ENV=local` and `THEIA_SMS_TRANSPORT=mock` and does not pass Kavenegar credentials. The SPI always selects Mock in `local`, even when an inherited transport setting requests Kavenegar. Mock transport refuses to boot unless `KC_HOSTNAME=http://auth.localhost:8180`. Never use `start-dev`, mock SMS, localhost callbacks or HTTP in a public/shared installation.

The extension refuses to initialize if the 32-byte Base64 HMAC key is missing or the SMS adapter is invalid. If using Kavenegar in a separately approved test environment, set `THEIA_SMS_TRANSPORT=kavenegar`, `THEIA_SMS_KAVENEGAR_API_KEY` and `THEIA_SMS_KAVENEGAR_TEMPLATE` in private deployment secrets, plus the correct environment. Kavenegar uses HTTPS `verify/lookup` **as transport only**; Keycloak generates and checks the OTP. The provider response, key and token are never logged by this adapter.

## Install SMS-only flow on an existing local realm

```sh
# Print a no-op reminder (default). Requires no admin connection.
python3 scripts/auth/configure_user_sms_flow.py

# Explicitly update the existing theia-users realm; leave its BFF client disabled.
python3 scripts/auth/configure_user_sms_flow.py --apply
```

The script authenticates to local master/admin-cli with bootstrap credentials from `.env.auth`, confirms that the user BFF client is still disabled, ensures **exactly one REQUIRED** execution of `theia-sms-otp` in `theia-users-sms`, and binds `browserFlow` and `loginTheme`. It rejects a conflicting existing flow, absent provider, or unexpected client state. Never point the script at an external Keycloak; it only permits `http://auth.localhost:8180`.

Verify in Keycloak Admin Console for `theia-users`:

1. Authentication → Flows → `theia-users-sms`: one **Required** `Theia SMS OTP (passwordless)` execution, with no cookie/password alternative.
2. Realm Settings → Login → Browser flow: `theia-users-sms`; Login Theme: `theia-sms`.
3. Clients → `theia-users-bff`: **Disabled** (this is expected until AUTH-06).
4. The Staff realm still uses its original browser/password flow.

### Mock outbox / security

Local mock code delivery writes `Local mock SMS OTP: <code>` to the local Keycloak log, without a phone number. Read it with `docker logs -f ai-therapist-backend-keycloak-1`. The owner-requested local debugging output is restricted to the loopback-only Mock transport; Kavenegar does not log OTPs. Keep these synthetic development logs local.

The existing private outbox remains available for local tooling: one owner-only file per send in `/tmp/theia-sms-outbox/*.otp`, with file permissions `0600` and directory permissions `0700`. It is not an HTTP endpoint and must not be mounted as a public volume. Never send a mock OTP to a real phone.

Because the User client is intentionally disabled in AUTH-05, the browser login journey is exercised by the **AUTH-06 integration harness**, not by manually enabling a client in this PR. Do not enable `AUTH_USER_LOGIN_ENABLED` or flip the client to `enabled=true` until that end-to-end gate has passed.

## OTP and security guarantees

| Property | Behavior |
|---|---|
| Phone | Strict Iranian mobile only; canonical `+989XXXXXXXXX`; Persian/Arabic digits accepted |
| Challenge | CSPRNG-generated six digits; HMAC-SHA256 digest bound to random 192-bit challenge ID; 120 seconds |
| Attempts | Max five total distributed attempts per challenge, including bad-format entries |
| Resend | New code after 60 seconds; previous challenge consumed; provider failure leaves no valid replacement |
| Rate limits | Atomic: 5 sends/phone/hour, 20 sends/remote-IP/hour, 200 sends globally/hour, and phone cooldown. Partial budget reservations are not refunded |
| Replay | `SingleUseObjectProvider.remove` permits only one successful consumer, even across nodes |
| Identity | New Keycloak user is created only **after successful OTP**; username canonical phone. Go UUID mapping is AUTH-06 |
| Privacy | No OTP, HMAC key, IP or full phone in persistent challenge/rate-limit keys; private phone stored only in Keycloak auth-session note and local mock outbox |
| Errors | Generic messages; no account existence disclosure or SMS API secret exposure |

Important limitations for AUTH-06 and deployment: authenticating the same phone in two concurrent first-login attempts must be verified with **real Keycloak PostgreSQL uniqueness/rollback semantics**, not just unit tests. Keycloak restart should invalidate in-flight sessions safely; do not claim OTP continuity across restart. `getRemoteAddr()` is the trusted socket peer; behind a reverse proxy, formally configure Keycloak trusted proxy handling before relying on per-client IP limits. SMS is subject to SIM swap/recycled-number risk; phone-change and recovery are policy tasks, not assumed solved. A global hourly cap is a safety backstop, not a substitute for account/payment/provider-level abuse prevention.

## Tests and verification

```sh
# No network/Maven required for repository checks
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v

# Requires Maven dependency access or cached artifacts; runs Keycloak-26.8.2 SPI compile and JUnit tests
mvn -f infra/keycloak/sms-spi/pom.xml verify

# Requires Docker + real Keycloak + PostgreSQL; only after --apply above:
# verify provider registration, REQUIRED-only execution, disabled user client and staff realm unchanged.
```

Before AUTH-06, run disposable Keycloak integration tests for: wrong/expired/5th guess, resend, SMS transport failure, two concurrent correct verifications, bad phone input, per-phone/IP/global limits, Keycloak restart, first-account race, and missing/wrong environment variables. The Java JUnit tests cover logic-level scenarios, but **do not** replace live Keycloak boot, realm import, browser theme, or PostgreSQL integration. No SMS content or real private information should be stored in telemetry, Admin Audit, or AI traces.

## Rollback

- Keep `theia-users-bff` disabled and `AUTH_USER_LOGIN_ENABLED=false`; the existing API/Staff login remain operational.
- For local test rollback only, restore the user realm's previous browser flow from Keycloak Admin Console after confirming client stays disabled; stop using the SMS overlay.
- Do not erase Keycloak PostgreSQL or production identities as a rollback operation.
