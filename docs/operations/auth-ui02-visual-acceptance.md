# AUTH-UI02: Tia SMS login visual acceptance and session smoke (local only)

## Scope

This patch follows AUTH-UI01 and modifies the **Keycloak User SMS theme** only.
It does not modify Keycloak's OTP verification, Go BFF/SQL sessions, React,
Staff Login, or the legacy Dev Session cutover. It adds a deterministic offline
preview/visual smoke plus an **opt-in, manually completed** live Keycloak/BFF
browser check. This is an internal synthetic-data implementation, not launch approval.

Changes to the theme:

- Placeholder, low-emphasis text, disabled resend label and footer use the same
  readable muted text token on the warm Tia background. The backdrop, branding,
  spacing, phone input and six-cell OTP design remain from AUTH-UI01.
- Resend countdown is based on a wall-clock deadline and rechecks on
  `visibilitychange` and `pageshow` so throttled background tabs do not show an
  artificially extended wait. An offscreen status announces readiness once.
  The server remains the **sole** authority for 60-second SMS cooldown,
  120-second OTP expiry, attempt limits and resend budgets.
- No tokens, phone numbers or OTP values are persisted or logged by the theme.
  There are no new remote assets or web fonts.

## Offline visual smoke (can run without Keycloak)

Generate sanitized, synthetic fixtures of phone/code and their error states:

```sh
python3 scripts/auth/preview_tia_login_ui.py --out /tmp/tia-login-preview
```

Optionally render actual Chromium screenshots and assert no horizontal overflow,
no JavaScript errors, loaded local Tia mark, form labels and OTP enhancement at
320x640, 360x740, 390x844, 430x932 and 1280x800:

```sh
# Requires Python playwright plus Chromium. Use your installed Chromium binary.
python3 scripts/auth/preview_tia_login_ui.py \
  --screenshots --chromium /usr/bin/chromium \
  --out /tmp/tia-login-preview
```

The output includes 20 PNGs plus `layout-report.json`. Screenshots are taken
from the **real theme CSS/JS and FTL source**, but a minimal fixture renderer
substitutes fake values. It is **not** a FreeMarker parser and does **not** prove
Keycloak can load the pages. These files contain no real phone/OTP data. Do not
publish them as verified screenshots of a running Keycloak server.

## Real local browser smoke (requires Keycloak + BFF + SQL)

Rebuild Keycloak after AUTH-UI02 because theme files are copied into its image:

```sh
docker compose --env-file .env --env-file .env.auth \
  -f compose.yaml -f compose.auth.yaml -f compose.auth-sms.yaml \
  --profile auth up -d --build keycloak
```

With the User browser edge reachable at `http://app.localhost:8080` and verified
SMS-only browser flow enabled, run from the **backend** root on a desktop with
a GUI and Python Playwright installed:

```sh
python3 scripts/auth/verify_tia_login_live.py \
  --origin http://app.localhost:8080 \
  --screenshots /tmp/tia-login-live
```

With a system browser, pass `--chromium /usr/bin/google-chrome` (or your
Chromium executable path) to use it with Python Playwright.

The browser opens at the actual `/auth/user/login` redirect. The script checks
that Keycloak serves the branded phone and OTP forms in RTL with loaded mark,
valid on-origin form actions, labelled fields and no mobile overflow. **The
operator enters a fictional/test phone and OTP manually**; the test never reads
SMS outbox files, stores OTP, displays tokens or takes a screenshot after any
input. Upon successful login, it verifies that the Go BFF issued an authenticated
User session with a CSRF token and HttpOnly session cookie. The tool intentionally
rejects nonlocal origins, cannot run silently without real services, and returns
nonzero when the login or session check was not completed.

Use frontend FE03's `npm run test:e2e:auth-bff` for the wider onboarding/chat/
logout acceptance matrix. In the running Keycloak also manually check:

1. Invalid phone, Persian/Arabic/ASCII digits and too-short phone input.
2. Wrong/expired OTP, 5 failed attempts, resend before/after cooldown, page
   background/resume, copy/paste and OS OTP autofill.
3. Change number invalidates the prior challenge; attempting to reuse that
   code fails without resetting rate-limit budgets.
4. Default `theia-staff` email/password login is unchanged; `theia-users` has
   exactly one REQUIRED SMS authenticator. No fallback Dev Session in the app.
5. At 320-430px and desktop widths check actual Keycloak-rendered pages, native
   keyboard focus, screen-reader feedback, JS-off fallback and reduced motion.

## Checks, compatibility and rollback

```sh
python3 -m unittest discover -s scripts/tests -v
node --check infra/keycloak/themes/theia-sms/login/resources/js/theia-login.js
node scripts/tests/auth_ui01_theme_browser.mjs
python3 scripts/check_structure.py
python3 scripts/check_contract.py
```

**No API change, schema migration or realm configuration change.** Rollback by
reverting this patch and rebuilding the SMS Keycloak image. AUTH-UI01 remains
intact. Live Keycloak/SQL verification is pending until the operator runs it;
offline Chromium screenshots do not satisfy that release gate.
