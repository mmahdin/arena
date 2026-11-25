# AUTH-UI01 - Tia branded Keycloak User login (local/synthetic)

## Scope and ownership

This change is **Keycloak User Login Theme + a small User-only SMS SPI action**, not a React login component or a change to Go BFF/OIDC/SQL. It is intended for the existing `theia-users` realm and `theia-users-bff` confidential client, which must still run the single REQUIRED SMS authenticator. Staff retains its separate Keycloak realm/theme and email/password flow. No schema migration and no new HTTP/API endpoints.

The implementation uses the **existing approved Tia mark** copied from `ai-therapist-frontend/public/brand/photo_2026-11-12_19-20-10.jpg` (the frontend's `TheiaMark` source), and follows Tia's `#f3f2ee` surface, `#23201b` text and mobile-first RTL type direction. No external font/CDN/resource or new font file is bundled; the configured Estedad/Vazirmatn family gracefully falls back to system Arabic-capable fonts. The screenshot used in the React app is user-provided project branding and remains project-owned.

## Screen behavior

| Page | Behavior |
|---|---|
| Phone | Labelled telephone input, Persian-friendly numeric keyboard, autofill support, input preservation on invalid phone / rate-limit, sanitized generic error; POST `smsAction=request` to `url.loginAction` |
| OTP | One actual input (`autocomplete=one-time-code`) + six decorative visual cells when JS loads; paste and Persian/Arabic digit normalization; native fallback when JS is blocked; 120-second server-valid OTP; POST `smsAction=verify` |
| Resend | Server-derived remaining cooldown (up to 60 seconds) initializes a visible timer; JavaScript disables the client button until ready, but the actual globally atomic rate-limits still live in `ChallengeService`; POST `smsAction=resend` |
| Change number | Dedicated POST `smsAction=change` invalidates the previous challenge in the shared atomic Keycloak store, clears the phone/challenge/timer auth notes, and returns to a prefilled editable phone field. The old OTP **cannot** be used after this action. Sender budgets/cooldown are **not reset**. |
| Errors | Invalid phone, invalid code, expiration, attempt exhaustion, resend cooldown and SMS send failure; server controls all security decisions, not JavaScript. |

**Changing a phone before login is not changing a linked account's phone number.** Account recovery or reassignment of a phone from an existing verified identity is explicitly out of scope and remains a separate security/product policy task. Do not add a new registration, password or alternate browser execution to make the UI work.

The FreeMarker pages are rendered as complete standalone HTML by their local `theia-layout.ftl`, with a separate scoped CSS/JS resource. Ordinary Keycloak error and account pages inherit the parent theme. The SMS forms post to Keycloak's `url.loginAction`. The branded User logout confirmation posts to `url.logoutConfirmAction` with Keycloak's `session_code`. After confirmation, Keycloak completes browser SSO logout and redirects to the configured app URL. Styles are not injected into React. All user-provided values are HTML-escaped.

## Apply and run locally

Requires backend through AUTH-08 and frontend through FE03. From the backend root after applying this patch:

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v

# Requires a reachable Maven repository or cached artifacts, plus JDK 21:
mvn -f infra/keycloak/sms-spi/pom.xml verify

# Rebuild and restart the User SMS-enabled Keycloak image (theme is copied into image):
docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-sms.yaml \
  --profile auth up -d --build keycloak
```

Ensure the **existing** `theia-users` realm still has `loginTheme=theia-sms` and REQUIRED-only `theia-sms-otp` in browser flow. Realm JSON imports do not update an existing Keycloak realm. For a previously configured SMS realm, do not rerun the initializer if it would disable an already activated client. Verify in Keycloak Admin Console instead. The Staff realm must not change.

Start the already-configured Go BFF, SQL Server and Tia frontend on its correct **same-origin** browser host, for example `http://app.localhost:8080`. Begin at `/auth/user/login` using the real browser (not `fetch`). Go remains responsible for OIDC callback/PKCE and host-only Theia cookie; Keycloak remains responsible for OTP.

For local testing with Mock SMS, inspect codes **only** inside Keycloak's owner-only `/tmp/theia-sms-outbox/` files. Do not serve the outbox through HTTP, logging or JS. Use fake phone-shaped identifiers; no real user data or clinical launch is authorized.

## Acceptance matrix (real Keycloak + browser, pending until run)

1. At 320/360/390/430 px and desktop width, both pages load the Tia mark, have no horizontal overflow, readable RTL, visible focus, screen-reader input labels, and usable keyboard/submit controls; check JS-disabled fallback and reduced-motion settings.
2. Submit invalid phone; input is preserved, escaped and an understandable alert appears. Valid Persian, Arabic and ASCII digit formats are accepted by the *server*.
3. Send a code, enter it from the OS autofill or paste; only six normalized digits are sent. Wrong code shows an error and preserves current cooldown; the correct code logs in once.
4. Attempt resend before 60 seconds both normally and by manually enabling the client button: **server** rejects it. Resend after 60 seconds invalidates the old OTP and sends a new one. Expired codes (120 seconds) and five failed attempts are rejected.
5. Press change-number after sending OTP: previous OTP must become unusable and a different phone can request a fresh code, while the old phone's rate-limit budget remains in force. The action must not create a user or set `theia_sms_verified`.
6. Successful OTP creates/uses the same immutable BFF user identity as before; `GET /auth/user/session` and `/api/v1/me` work, and the existing Tia onboarding and chat still load. `theia_sms_verified` must be present only after successful verification.
7. Verify resource responses are same-origin, theme version is visible after container rebuild (clear browser cache if needed), and staff email/password login still uses its original theme. No OTP/session tokens appear in browser URLs, client logs or local storage.

## Limitations and rollback

This environment verified static contracts and a standalone Java cancellation test, **not** live Keycloak FreeMarker rendering or Maven Keycloak integration. Run the real visual/browser acceptance matrix before declaring this production-ready. For a local rollback, revert only this patch and rebuild the SMS Keycloak image. No Keycloak realm, SMS flow, BFF or DB migration changes are required.
