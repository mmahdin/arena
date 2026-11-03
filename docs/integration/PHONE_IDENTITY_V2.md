# HAMRAH-INTEGRATION-02 v2 — synthetic phone-shaped entry

## Approved experience
- The existing Hamrah artwork, screen typography, components, and home/onboarding route structure are the visual source of truth. Add one local-only number-shaped input; **do not** offer a seeded list or login/signup selection.
- The only allowed team convention is an imaginary `09xxxxxxxxx` value (11 digits). Persian and Arabic numerals are normalized. This is not guaranteed telecom-unassigned; avoid real contact numbers entirely.
- Backend `POST /api/v1/dev/identity-session` accepts `{ "phone": "09000000001" }`, returns `{ "ok": true, "created": true|false }` and sets the HttpOnly server-signed revocable cookie. `created` is informational; always fetch `GET /api/v1/me` to decide route. A first-time profile has null `onboardingCompletedAt`; a returning profile with a timestamp opens Home.
- Unfinished profiles use existing Name → Approach → Ready screens and `PATCH /api/v1/me/profile` with `displayName`, `conversationStyle`; the other 20+ health/belief questions are NOT stored or transmitted.
- Refresh calls GET /me, missing or revoked cookie returns to the form, 503 keeps a retry state. Logout and account switch revoke the backend session and clear user-specific React state; late results must not recreate the old user. The frontend never stores/reads the cookie, raw number or chat in browser storage.

## Backend implementation
- Migration 0007 adds `dbo.dev_phone_identities` with binary(32) HMAC lookup key and user_id FK; no raw phone field.
- HMAC derives from stable DEV_SESSION_SECRET using a versioned domain prefix. Secret rotation makes old numeric lookup unrecoverable; do not rotate casually on an existing development DB.
- SERIALIZABLE transaction ensures first create, initial profile, mapping, and dev session are one atomic unit. Returning logins issue a new session; old presented cookie is revoked.
- Existing seeded `/dev/users` and `/dev/session` are preserved strictly for legacy internal smoke tests. The new UI never calls them.
- Not authentication: anyone with the fake identifier can take the account. No real users, no public exposure and no clinical safety claim.

## Apply / validate
1. Apply backend patch against Task07 + validation fixes, then verify/apply migration 0007 to isolated local MSSQL. Avoid resetting existing schema or data.
2. Apply frontend v2 patch to the packaged Hamrah UI **before** the rejected Task 02 patch.
3. Run `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py`, `cd apps/api && go test ./...`, local integration/acceptance, frontend `npm test`, `npm run build`, and real Playwright session E2E with both services active.
4. Verify create A, reenter A, create B, switch A→B with cookie revocation, onboarding, refresh and 401. Confirm no raw numbers or content in logs/DB or browser storage.
