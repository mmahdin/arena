# AUTH-09 — Retire Dev Session (Tia User + Go BFF)

The real Keycloak SMS/User BFF flow was validated in the owner's disposable local environment before AUTH-09. This patch removes the synthetic cookie, routes, runtime signer, identity repository, settings, and Dev Session frontend paths. Staff login and administrative APIs are unchanged.

## Deployment order

1. Apply backend AUTH-09 patch after AUTH-08 + AUTH-UI02. Build and run backend tests.
2. **Back up the local database if keeping test data matters.** Execute migration `0032_retire_dev_identity.sql` through the existing migration runner, not manually against shared/prod data. It drops only `dbo.dev_sessions` and `dbo.dev_phone_identities`. Historical migrations `0005` and `0007` remain immutable to preserve checksums. Existing `users`, `user_profiles`, conversations, and `auth_*` session tables remain intact. The old fake-phone identity association does not transfer to Keycloak; the user confirmed all prior data were disposable local fixtures.
3. Remove `DEV_SESSION_SECRET` and `ALLOW_DEV_LOGIN` from local `.env` and `.env.auth`. The backend rejects `ALLOW_DEV_LOGIN=true` rather than silently reenabling it. Set real BFF flags:

```
APP_ENV=local
AUTH_BFF_ENABLED=true
AUTH_USER_LOGIN_ENABLED=true
AUTH_USER_SMS_FLOW_VERIFIED=true
```

4. Apply frontend AUTH-09 patch after FE03 and run `npm ci`, `npm test`, `npm run build`.
5. Keep both frontend and backend on their expected User origin (e.g. `http://app.localhost:8080`). Keycloak stays on the configured exact auth origin. Use the existing `docs/operations/auth-ui02-visual-acceptance.md` and frontend `docs/integration/AUTH_BFF_LIVE_ACCEPTANCE_FE03.md` for the live test.

## Acceptance on a disposable local environment

- `GET/POST/DELETE /api/v1/dev/*` return 404 or 405, and never set an authentication cookie.
- Old `wellbeing_dev_session` cookie cannot authenticate `/api/v1/me` or any owner-scoped route; no legacy signing secret is accepted.
- Login at `/auth/user/login` navigates to the Tia-branded Keycloak phone and OTP pages, then returns via the BFF callback; session and product requests authenticate via `__Host-theia_user_session` over HTTPS (or local HTTP session cookie).
- Refresh, chat, onboarding resume, CSRF checks, expiry, logout/sso and logout-all work as in FE03.
- Hiding a user account invalidates their real `auth_sessions`; data retention unchanged.
- Staff `/auth/staff/session` and `/api/v1/admin/me` continue to require an active staff session; no user cookie authenticates Staff.
- Check migration output: `OBJECT_ID('dbo.dev_sessions','U')` and `OBJECT_ID('dbo.dev_phone_identities','U')` must be NULL; `dbo.auth_sessions` must exist.

## Tests & limits

Run `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py`, `python3 -m unittest discover -s scripts/tests`, and `cd apps/api && go test ./...` on an environment with project dependencies. SQL integration requires disposable DB and explicit `WELLBEING_INTEGRATION=true`. Previous synthetic dev-cookie HTTP integration suites are removed; the real browser acceptance suite (`npm run test:e2e:auth-bff` in the user frontend) is the supported end-to-end route, while repository-level unit/security tests remain. Historical ADRs/migration history are retained as archival documents, not runtime functionality.
