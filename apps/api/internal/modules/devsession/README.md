# Devsession Module — internal test identity only

**Not real authentication.** Synthetic users only. Mounted only when `ALLOW_DEV_LOGIN=true` and `APP_ENV` is `local`, `stage`, `staging` or `staging-v2`. Staging uses the same signed/revocable test sessions; production still rejects dev login.

- `GET /api/v1/dev/users`: list synthetic eligible users.
- `POST /api/v1/dev/session`: select test user, create fresh SQL Server `dbo.dev_sessions` record, issue signed HttpOnly SameSite=Lax cookie (12h). If the caller presents a valid previous cookie, that session is revoked after the new session is persisted.
- `DELETE /api/v1/dev/session`: revoke currently presented valid session in SQL Server and expire browser cookie. Repeated logout is idempotent. A copied older cookie cannot be reused after logout/switch.
- `platform/devsecurity.Guard`: verify cookie signature + expiry, then check `dev_sessions` is not revoked or expired and that its user is still an eligible test user. A missing/invalid/revoked cookie returns 401; unavailable session DB returns 503 (fail-closed).
- `adapters/mssql`: implements test user selection and persistent session storage. No login exists for real accounts or production instances.
- The internal user UUID never comes from a client-provided ownership field. Profile, conversation, message and turn persistence are scoped to this internal UUID at SQL boundaries.

## Migration / compatibility

Apply `0005_dev_sessions.sql` **before** deploying the updated API. Previously issued cookies cannot authenticate against the new session table, intentionally requiring re-selection of a test user. Existing user IDs, conversations and messages remain unchanged. Do not edit prior migrations; this new migration adds only the test-session table and its index.

## Test requirements

Test missing/forged/expired/revoked cookies, switching testers, logout replay, unknown/non-test IDs, deletion of test-user eligibility, SQL storage failure, wrong-owner conversation read/turn/delete and no dev endpoints outside local mode. See `.agents/skills/wellbeing-dev-session/references/session-test-matrix.md`.

**Internal synthetic-data use only; a user switcher is not a security boundary for real identities.**

## Hamrah phone-shaped synthetic entry (v2)

- `POST /api/v1/dev/identity-session` body `{ "phone": "09000000001" }`: **local-only** get-or-create test identity, issue signed revocable HttpOnly cookie. Existing identifier returns `created:false`; new identifier returns `created:true` with an incomplete profile. The backend never asks for OTP or confirms ownership.
- Accept any Iranian `09xxxxxxxxx`-shaped identifier (including Persian/Arabic digits). This is not OTP and provides no proof of number ownership: local synthetic-only testing; do not use a real person's number. Error `400 INVALID_DEV_PHONE` for other inputs. Do not log raw input.
- Migration `0007_dev_phone_identities.sql` stores a unique HMAC-SHA256 digest of the normalized string and a UUID reference. Existing `users`/`user_profiles` and old dev sessions remain valid. **Preserve `DEV_SESSION_SECRET` across local restarts**: it derives these identifiers as well as signing cookies; rotating it makes prior mappings unreachable by the typed number, without deleting existing rows.
- Concurrent calls use serializable locking to avoid duplicate users and create the profile + session in one transaction. Legacy `/dev/users` and `/dev/session` remain for older internal smoke suites only; the Hamrah UI must never enumerate users.
- This is **identity selection without authentication**. Anyone knowing a synthetic identifier can assume that tester. No public exposure, real phone numbers, or real mental-health data.
