# HAMRAH-INTEGRATION-05 — Restore the original complete onboarding

**Design rule:** The newer Hamrah designer UI remains the source of truth. Task05 restores **all 24 illustrated onboarding screens before Home** without removing Main screens, illustrations, chat actions, history, menu or profile editors. Existing completed accounts still open Home; fresh accounts start at `welcome` and proceed in the original `FLOW` order.

## Flow and guards

`welcome → testimonials → research → experts → name → simple-intro → privacy → deeper-intro → therapist → age → concerns → focus → feelings → neurodivergent → faith → know-intro → interests → relationships → strengths → barriers → almost-intro → voice → approach → ready → home`

- New incomplete users cannot open Home, Chat, History, Survey, Menu or Insight by a direct `#/...` URL.
- A direct URL to an **unvisited** onboarding step is redirected to the furthest visited step; navigating back to visited steps and forward in the normal order works.
- Finishing `ready` calls the existing `PATCH /api/v1/me/profile` once with **displayName and conversationStyle only**. If saving fails, the user remains on `ready` with an error/retry. Backend `onboardingCompletedAt` stays authoritative.
- On a finished account, `name` and `approach` still work as Menu → existing profile editors. Other designed pages remain reachable.
- All questionnaire inputs (including health, religious/spiritual, neurodivergence and age) remain **ephemeral React memory only**. They are not sent in HTTP requests or LLM context, persisted in storage, or logged; answers are cleared after final completion, tester switch or logout. The Voice selection is likewise local until separately approved.
- Task05 originally restarted unfinished users at Welcome; **Task06 supersedes this** with a backend-owned ordinal cursor. Read `TASK06_ONBOARDING_RESUME.md`. Questionnaire answers remain ephemeral; do not invent localStorage persistence.
- Existing testimonials/research/clinician names and design claims remain original visual content; they have **not** been independently verified or approved for public/clinical use. Review copy/affiliations before external release. Preserve visual structures while doing so.

## Validation and known checkout differences

- `npm test` checks the exact `FLOW`, all original route components and progression/deep-link rules. `npm run check:foundation` checks all source structure and skills.
- `node scripts/e2e-full-onboarding.mjs` is a live browser journey (Vite → Go/Gin → MSSQL) that traverses the entire design; it checks the profile remains incomplete before Ready and verifies completion after Ready. Local services and Playwright are required; no `/api/v1` interception.
- Existing `scripts/e2e-session.mjs`, `scripts/e2e-chat-history.mjs` and `scripts/e2e-profile-preferences.mjs` now call `finishFullOnboarding` rather than expecting the rejected three-step shortcut. **If your local QA agent added additional E2E scripts (`e2e-qa.mjs`, `e2e-faults.mjs`), update their setup helpers to import this helper too**. Those extra files were not present in the supplied Task04 patch baseline, so this patch does not replace them.
- This patch is incremental on **Task04 applied after Task02 v2 + Task03 + QA runtime fixes**; not on the rejected Task02 seeded-user picker. Backend API and migrations need no changes in Task05.

## Manual verification

Run the local backend with MSSQL and `LLM_PROVIDER=mock`, Vite and Playwright. Check navigation at 360/390/430px, backwards steps, optional skips, errors/retry on final profile save, refresh incomplete versus complete, history persistence, user A/B isolation and responsive keyboard layouts. Use only disposable, synthetic identifiers and questionnaire choices. Verify original design assets have not been deleted.

## Artifact checks executed (this patch authoring environment)

- PASS — `npm test`: 29 tests, including full-route preservation and deep-link progression checks.
- PASS — `npm run check:foundation`: 41/41.
- PASS — `node --check` of changed/new `.mjs` and `.js` scripts.
- PASS — `git diff --check` and reapplying the patch onto a clean copy of Task04 (verified before delivery).
- BLOCKED — `npm ci --offline`: the environment lacks cached `yallist-3.1.1.tgz`.
- NOT RUN — full Vite build or Playwright + Go/MSSQL live E2E here. These are required local QA gates, not passed by Node tests alone.

Keep existing checkout `VALIDATION_REPORT.md` and QA-01 documents intact; the QA agent should append its actual local E2E evidence after running the restored flow.
