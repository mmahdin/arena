# HAMRAH-INTEGRATION-04 — Existing profile/preferences and designer UI routes

## Scope and design decision

The approved Hamrah designer UI remains the source of truth. **Do not remove visual routes, artwork or actions simply because no corresponding server endpoint exists.** The original 25-step onboarding source and all Main screens remain in place. The existing connected core (fictional local identity → short onboarding → Chat/History) is unchanged. Only a **completed** profile may open the other original routes; an unfinished profile cannot bypass the Go onboarding gate.

## Implemented in this slice

- The existing Menu → **ترجیحات** opens the *original ApproachScreen*. Its five original choices map to `listening`, `guidance` or `balanced`, and the selected value is persisted using the already-supported `PATCH /api/v1/me/profile`. Backend response remains authoritative. Saved style is restored on reopen.
- A small **نام نمایشی** entry in the existing drawer opens the *original NameScreen*. It uses the same endpoint and preserves the previous conversation style. Cancel restores the original profile value; failed saves leave persisted state unchanged.
- All original routes, including Day, Insight, Survey, Privacy, Research/Experts and other onboarding illustrations, remain visitable after profile completion. Nothing is deleted. No non-existent `/insights`, `/survey`, `/voice`, `/bookmark` etc. endpoints were invented.
- The Privacy screen retains its artwork and control, but no longer claims server conversations are stored only on the phone or that account deletion has been implemented. The optional-sharing control never transmits anything; it remains visible for a future supported flow.
- Visible developer terminology in the normal login and Ready screens is reduced. An unobtrusive internal-entry note remains because this flow is not phone ownership verification.

## Boundaries

- Other original screens still use **static designer content**; this is not personalized AI output or stored insights. The UI stays intact, and their content must undergo product and privacy review before real-person use. The Survey is only local ephemeral UI state and not a validated assessment. Do not persist sensitive answers in the database, network, analytics or localStorage.
- The existing message action menu and voice-selection UI stay available in source, but unsupported server features do not report fabricated success. Build future backend endpoints and migrate data only after a distinct scope decision.
- The only live profile fields remain `displayName` and `conversationStyle`; `onboardingCompletedAt` is preserved. Dev identity is internal synthetic only.
- No changes to Go, OpenAPI or applied SQL Server migrations were needed.

## Tests and local acceptance

Executed in this artifact environment:

- `npm test`: PASS, 27 Node tests (20 Task03 baseline + 7 new tests for route availability, profile validation/mapping and persistence/failure behavior).
- `npm run check:foundation`: PASS, 41/41.
- `node --check` of new `.js`/`.mjs`: PASS; `git diff --check`: PASS.
- `npm ci --offline --ignore-scripts --no-audit --no-fund`: BLOCKED, missing `yallist-3.1.1.tgz` in the environment's npm cache. Vite build/browser E2E therefore NOT RUN here.

On the **actual post-QA local checkout**, run `npm ci && npm test && npm run build && node scripts/e2e-profile-preferences.mjs` while Vite + Go + MSSQL are running. Browser test requires installed Playwright and Chrome (set `BROWSER_BIN` if needed). Confirm both previous Tasks02/03 browser suites continue to pass. Confirm the original connected source remains visually recognizable at 360/390/430px and desktop; this artifact environment could not compare screenshots.

**Patch baseline:** Task 03 with QA-01 runtime fixes. The user's independently added QA browser scripts and report are not overwritten. No new backend patch for this slice.
