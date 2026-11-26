## After Task05 — Full illustrated onboarding restored

Task05 restores all 24 pre-Home designer stages for new profiles. **Do not reduce the path to name/approach/ready again.** Preserve all screens and buttons even when backend features remain pending. Task06 adds server-owned step-only resume (no questionnaire answers). Next: approve any further non-sensitive draft preferences separately, then bookmarks, daily reflections/insights, advanced chat and voice. Read `docs/integration/TASK05_FULL_ONBOARDING.md`.

---

# Current handoff — HAMRAH-INTEGRATION-03 (real Chat and History)

1. Apply the **Task 02 v2 Backend + Frontend** first; verify migration 0007 and a working fictional phone session. Do not apply the rejected Task 02 seeded-user patch.
2. Apply the **incremental Task 03 frontend patch**. Backend API already supports conversations/history/JSON turns; no Task 03 backend patch or schema change.
3. On the local machine run `npm ci`, `npm test`, `npm run check:foundation`, `npm run build`; run `npm run test:e2e:chat` against running Vite + Go + MSSQL + Mock AI (no Playwright route mocking). Also run backend `make acceptance-backend` and inspect OpenAPI drift.
4. Verify two fictional accounts are isolated, create/send/reload/continue/delete works, 409 and 503 retry identical `clientTurnId`/body, and wrong-owner conversations give 404.
5. **Only then** prioritize further features: profile preferences, long history pagination, bookmarks/regenerate, approved safe reflection features. Visual designs for survey/insight/voice remain disabled and must not be fabricated.
6. For any user-visible scope outside this project, do not collect real numbers or mental-health information. Treat the prototype as synthetic internal-only.

---

# Current Hamrah handoff — after dynamic synthetic identity v2

**Status:** Task 02 v2 code patch implemented, pending live build/SQL Server/browser acceptance. Do not apply previous seeded-user-picker Task 02 patch or reuse its identity interface.

1. Apply backend `hamrah-integration-02-v2-backend.patch` to validated Task07+fixes backend, then migrate `0007`. Retain the stable `DEV_SESSION_SECRET`; inspect migration ledger first. Run Go/MSSQL and HTTP tests.
2. Apply frontend `hamrah-integration-02-v2-frontend.patch` to documented latest Hamrah source (without previous Task 02 patch). `npm ci`, `npm test`, `npm run build` and `npm run test:e2e:session` against Go/MSSQL. Use only synthetic `0900000XXXX` identifiers, never actual phone numbers.
3. The Hamrah UI remains source of truth. Profile first save takes the existing name, approach, ready path, no intimate questionnaire answers are persisted.
4. Once browser acceptance passes, next slice: real JSON chat, conversation history and persistent reload. Maintain `clientTurnId` idempotency, JSON turn contract, profile cookie gate and no fabricated responses.
5. Existing extra onboarding screens, therapist/research testimonials, daily insight and voice remain design-only, not enabled clinical features.

---

# Ordered next agent tasks — source-preserving Hamrah UI

## Start: verification of this packaged designer baseline
1. Read AGENTS, MVP_SCOPE, FRONTEND_DESIGN_SPEC, integration docs and relevant local skills.
2. `npm ci && npm run check:foundation && npm test && npm run build`. Capture 390px/desktop screenshots; record tool limitations truthfully. Confirm no original source bytes changed unintentionally.
3. Obtain current validated backend checkout; compare its OpenAPI to `contracts/openapi/backend-openapi.snapshot.yaml`. Task07 + three backend validation fixes were reported as passing Go/MSSQL; these test results are external evidence, not executed here.

## Vertical slices
4. **Tester/session bootstrap:** implement `/dev/users` list, `/dev/session` POST/DELETE and `/me`; show a visually coherent dev-only picker; clear per-user state on switch and reject delayed responses.
5. **Minimal profile:** bind `NameScreen`, add backend `conversationStyle` choice mapped from `APPROACHES`, PATCH `/me/profile` and use onboardingCompletedAt for returning testers. Make long/intimate questions design-gallery only pending scope/safety decision.
6. **Real chat:** create conversation and post `/turns` with stable `clientTurnId`; GET `/messages`; remove fixed `setTimeout` reply in connected mode. Preserve chat layout. Implement pending/errors/retry and model-unavailable UX.
7. **History and lifecycle:** list/open/continue/delete conversations, reload preservation, return user, logout and cookie-expiry recovery. Hide fake weekly metrics if not supported.
8. **Test E2E:** Playwright against real Vite proxy → Go/Gin → MSSQL → mock LLM, no API route mocking. Verify mobile RTL, switching, duplicate retry, 401/404/409/503, deletion, no PII leakage.

## Product-led backend expansion (after Core MVP and explicit approval)
9. Create a decision record for intake fields, consent/storage & retention, age limits, voice, insight/survey, message actions/regenerate; see `docs/roadmap/BACKEND_FRONTEND_INTEGRATION_PLAN.md`.
10. For each approved design capability, update backend OpenAPI → SQL migration if needed → Go module/ownership checks → frontend adapter/UX → tests, one slice at a time.
11. Test real LLM with approved key, rate/cost guards, synthetic prompts, and explicit safety review; do not publicly launch on this prototype.

## After Task 07 (Saved Messages)
- Task 07 supports existing Chat bookmark action and History saved-message section. See `docs/integration/TASK07_SAVED_MESSAGES.md`.
- Next: define approved Daily Reflections / Insights contracts and content-policy decisions before collecting new sensitive data. Retain all existing UI routes and components.
