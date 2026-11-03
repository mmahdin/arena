# Hamrah — Backend completion & frontend connection plan

**Design decision (locked):** Use `ai-therapist-frontend(1).zip` as the **source of truth for product UI and intended journey**. Preserve the Persian «همراه» screens, 25-step onboarding appearance, illustrations, theme, hash navigation and chat design. The old Rahrah/Wellbeing frontend must not replace this implementation.

**Engineering baseline:** user-reported backend Tasks 02–07 passed Go/MSSQL/HTTP acceptance on 2026-11-02 with validation fixes for migration 0006, lease UUID and provider deadline test. This turn includes only the **new UI ZIP**; the actual newly validated backend checkout is not attached. Confirm its source/OpenAPI before proposing code changes. Current API is JSON, no SSE. Keep internal/local/synthetic-only restriction.

## Important product principle
The new UI defines what users should *see and eventually be able to do*. A visual control or copy is **not** evidence that the existing backend offers that capability. Build first the connected core, then extend backend feature-by-feature for approved design capabilities. Avoid simplifying the aesthetic to fit the old backend, but avoid faking unsupported behavior. Sensitive intake, therapy claims and surveys require separate safety/privacy review.

## Phase 0 — Lock current design and baseline (frontend first)
- Use current source SHA manifest `docs/design/original-source-sha256.json`, screenshots at 360/390/430px and desktop, `npm ci`, `npm run build`, `npm test`.
- Preserve `src/screens/Onboarding.jsx`/`Main.jsx`, `src/components/UI.jsx`, `src/data.js`, `src/styles.css`, and `/public/assets/*`.
- Audit local-only demo content: named clinicians, university association, 6000 ratings, phone-only storage, account deletion and clinical question wording. Preserve screen composition but gate/replace unverified assertions before connected testing.
- Acceptance: baseline build and screenshots are recorded; no accidental visual rewrite.

## Phase 1 — Verify backend contract and environments (both repos)
- Use actual validated backend branch, ensure migration ledger 0001–0006 valid, `make acceptance-backend` and `make acceptance-http` succeed on local MSSQL, `LLM_PROVIDER=mock`.
- Compare **backend** `contracts/openapi/openapi.yaml` with frontend `contracts/openapi/backend-openapi.snapshot.yaml`; update the snapshot only if source differs.
- Confirm Vite proxy `/api/v1` → `127.0.0.1:8080`, server and Vite loopback-only, Cookie Path `/api/v1` and consistent hostname. Frontend should not touch database or LLM key.
- Acceptance: 200 live/ready, no API drift and no lost existing synthetic data. No backend feature work unless an actual mismatch is found.

## Phase 2 — Real Dev Session in current UI
- Add a small, visually native test-user selection state **before** `welcome`/connected flow; do not turn the prototype into Google/Apple/email login.
- Wire `POST /dev/identity-session` `{phone}` for fictional `0900000XXXX` values, `GET /me`, `DELETE /dev/session` using `credentials:'include'` and relative paths. All user-scoped state is derived from server session.
- On 401: clear UI profile/messages/draft/turn state and show chooser; on switch/logout: revoke cookie and invalidate in-flight queries (epoch or abort controller).
- Acceptance: A→B→A, page reload, revoked/expired sessions, no cross-user UI leakage; source look unchanged.

## Phase 3 — Minimal profile + route policy
- Bind current `name` screen to `displayName` and `approach` screen to `conversationStyle`. Proposed map: listening=`همدلی و شنیدن`; guidance=`راهنمایی آرام`/`به من یاد بده`/`حل مسئله با هم`; balanced=`گفت‌وگوی آزاد`. Confirm mapping with product owner; use safe default for skip.
- `PATCH /me/profile`, gate chat until onboarding completed, returning tester goes to home, not forced through 25 steps each login.
- **Product approval needed:** decide how the full long onboarding is represented in connected build. Recommendation for current internal slice: visual/design-gallery or skippable optional preview, with **no transmission or persistence** of age, therapy history, religion, neurodivergence, relationships, fears or health answers. If mandatory and persisted is desired, treat as new backend feature with risk review, not an ad-hoc extension of profile PATCH.
- Acceptance: profile saves and persists across reload; no backend request contains unapproved intake answers.

## Phase 4 — Connect real chat without changing visual design
- In `src/screens/Main.jsx` chat, replace delayed fixed reply with `POST /conversations` then `POST /conversations/{id}/turns`; load `GET /conversations/{id}/messages`. Keep dark bubble layout, paragraph rendering, suggestion chips and composer.
- Generate UUID with `crypto.randomUUID` for each user turn; preserve **same UUID and exact input** for Retry, even if timeout or network response was lost. Fetch persisted messages before applying an ambiguous retry result. Do not append duplicate optimistic user bubbles.
- Handle `401` (tester), `404` (removed/foreign thread), `409` (onboarding required or turn pending), `503` (AI unavailable), errors and cancellation. No fake assistant completion or pseudo streaming.
- Acceptance: send → real Mock AI response persisted atomically → reload → continue; network-drop recovery with exactly one pair in MSSQL.

## Phase 5 — History, profile, and truthful UI state
- Replace static History row with `GET /conversations`; open thread → messages; new thread; confirm/delete via `DELETE /conversations/{id}`. Backend currently returns max 100 entries/messages, so make limitations explicit rather than silently pretending unlimited history.
- Connect menu logout and profile style updates. Preserve Home envelope/reflection art, but do not present static daily letters/insights as newly AI-generated personalized facts.
- Action menu truth table: copy works frontend-only; browser speech synthesis is device-dependent; save/regenerate/report require new API if functional; otherwise disabled/clear demo label.
- Acceptance: returning tester sees their own history, delete works, switching users immediately clears old history.

## Phase 6 — Real end-to-end acceptance
- Browser Playwright against real Vite proxy → Go Gin API → MSSQL → Mock AI (do NOT intercept `/api/v1` requests in these tests).
- Fresh tester → profile → chat → reply → refresh → history → continuation → same-ID retry → owner isolation → logout; confirm 401/404/409/503, API down, offline/dropped request, input keyboard focus, RTL layout, mobile 360/390/430 and desktop.
- Run backend `make acceptance-backend`, frontend `npm ci && npm test && npm run build` plus browser suite. Check logs never contain real/sensitive content, cookie and API keys never exposed.
- Acceptance: both repos show PASSED reports from **the same checkout/date**; internal-synthetic MVP is complete. **Do not claim public/clinical readiness.**

## Phase 7 — Backend product expansion from the new UI (explicit individual decisions)
After the core MVP works, pick desired capabilities based on this design; ship one independent vertical slice at a time (product definition → OpenAPI change → MSSQL T-SQL forward migration if necessary → Go modules/data ownership → frontend adapters → E2E). Proposed priority:

| Priority | New UI capability and current gap | Backend implication + guardrail |
|---|---|---|
| P1 | Approach preference richer than three styles | start with explicit UI-to-enum mapping; expand enum only if behavior differences are approved and tested |
| P1 | Multiple chats, searchable/long history, timestamps and weekly grouping | improve existing list/messages pagination (>100), stable cursor/order; avoid unverified weekly wellbeing score |
| P2 | Saved messages, per-message feedback and reporting | new owned message metadata/feedback endpoints, rate caps and SQL schema if needed; distinguish private UI bookmark from safety report |
| P2 | Regenerate response action | new explicit regeneration semantics and idempotency (never rewrite historic conversation silently) |
| P2 | Daily letters, reflection and 'Insight' | new scoped reflection/insight data model and provenance; avoid clinical inference and invented personal memory |
| Decision required | 25-step intake and optional sensitive data | clarify necessity, consent, field retention/deletion, age policy and whether answers reach LLM; no collection by default |
| Decision required | Voice picker and voice input/output | choose device-only accessibility vs actual speech provider; microphone permissions, secure media processing and cost before enabling |
| Decision required | Depression/anhedonia survey | no clinical screening/PHQ-like scoring without clinical, legal, UX and crisis review; do not persist now |
| Decision required | Privacy sharing and account deletion controls | define data retention/export/delete and truthful consent, real authentication before external testers |
| Decision required | Testimonials, doctor/institution affiliations | require verified claims/evidence and proper permissions; not a backend coding task by itself |

## Real LLM mode (separate controlled gate)
- Only after core E2E passes with mock. Set server-only credentials and approved compatible provider; rate/cost caps and operational logging without prompts. Run synthetic Persian safety test matrix and timeout/503 retry under real provider; no real mental-health disclosures, no public traffic and no clinical efficacy claims.

## Next concrete agent task
**HAMRAH-INTEGRATION-01: establish current UI build + Dev Session API bootstrap (without visual redesign).** Read `AGENTS.md`, `FRONTEND_DESIGN_SPEC.md`, `docs/integration/UI_BACKEND_GAP_MATRIX.md`, and backend live OpenAPI. Capture UI screenshots, add session picker matching existing shell, implement user switch/logout, test real Go API over Vite proxy. Deliver a small diff with tests, updated docs and no changed artwork/CSS unless essential. Avoid starting new backend tables until a verified gap demands one.

## Readiness status at handoff
- Frontend: UI designer prototype, source preserved; not API connected.
- Backend: previously reported PASS WITH LIMITATIONS for Mock AI + MSSQL after Tasks 02–07; **not independently rerun in this turn**.
- Combined product: NOT RUN end-to-end, no real-provider approval or clinical validation.
