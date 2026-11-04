> **Historical reference (v1).** This document describes an earlier onboarding flow. The active implementation is **v2**; see [ONBOARDING_V2.md](ONBOARDING_V2.md). Do not use the route indices below for current development.

# HAMRAH-INTEGRATION-06 — Resume onboarding without changing the designer UI

**Build on Task05 + Tasks02 v2/03/04 and QA fixes.** The full 24 illustrated pre-Home screens, all assets and all Main screens stay in place. This patch changes only the route cursor, API snapshot and repeatable tests.

When a fresh synthetic account enters, GET `/me/onboarding-progress` restores the backend's `{furthestStep,flowVersion}` before any step renders. Entering the next *new* step performs a PUT with **only** `{furthestStep: nextIndex}` before navigation. Going backwards and revisiting a previously unlocked page are local and do not decrease stored progress. A lost PUT response can be retried with the same ordinal without creating a duplicate or losing progress. Login/refresh/logout/switch abort stale calls; foreign users receive only their own cursor.

### Privacy and retained design

No sensitive questionnaire answers, age, therapy, religion, neurodivergence, relationships, interests, voice choice or chat content enter the progress HTTP request or localStorage. They remain in React memory as before and are discarded after completion/logout. The cursor is not a clinical record. Do not add telemetry or persistence of questionnaire contents without separate approval. Only the existing final `PATCH /me/profile` writes **displayName + conversationStyle** and sets the onboarding completion timestamp. Every designer page remains navigable, and other designed features are neither deleted nor called implemented without an endpoint.

**Important:** Names and style selections are not draft-persisted; these inputs must be reentered if the browser was closed before completing Ready. The app resumes at the saved screen; if name is missing at Ready, it redirects to the existing Name then Approach screens, returns to Ready and lets the user finish. No additional UI screens are introduced.

### Verification

- `npm test` includes route-order, API whitelist, flow version and privacy tests; `npm run check:foundation` checks baseline structure.
- Run `npm ci && npm run build` in the user's dependency-enabled environment.
- With Go + MSSQL + Mock AI and Vite running: `npm run test:e2e:resume` and `node scripts/e2e-full-onboarding.mjs`; then rerun `test:e2e:session`, `test:e2e:chat`, and any local QA suites.
- Expected scenario: new fictional phone → Welcome → navigate to Name → refresh → Name (not Welcome) → deep-link to Ready denied → logout/reenter with same number → Name; return all the way to Ready, save profile and revisit Home. A different fictional number must start at Welcome.
- The original `check:source-baseline` SHA equality necessarily fails on intentionally modified integration files; it is not a claim that code hasn't changed. Visually compare original screens at 360/390/430 widths.

### Patch environment result

- `npm test`: 33 PASS; `npm run check:foundation`: 41/41 PASS; `node --check`: PASS.
- Backend contract checker: 15 routes PASS; structure checker 41 PASS.
- Go full unit tests: BLOCKED because Gin/uuid/MSSQL modules are not cached in this authoring container.
- Vite build and live SQL/Playwright E2E: NOT RUN in this container. A passing local real-stack validation is required before marking Task06 complete.
