## Approved onboarding personalization amendment (2026-11-11)

The user explicitly requested using onboarding information throughout chat. This
supersedes older name/style-only and ephemeral-only restrictions below for the
internal synthetic build. Send only explicit allowlisted `onboardingAnswers` on
final Ready, with accurate server-storage/AI-processing copy. Drafts remain in React
memory. Ordinary profile edits omit answers to preserve them. Gender options are
exactly زن، مرد، تمایل ندارم بگم. No inferred traits or browser storage is added.
Deploy backend migration 0016 and the matching API before this frontend.

## UX refinement product boundary (2026-11-04)
The active product now has two primary experiences: **Chat** and **Letters**. Home's reflection card and diary-writing UI were removed by user request; do not reintroduce journaling/reflection prompts into Letters or Home without a new product decision. Legacy `#/day` and `#/insight` resolve to Letters. Opening Chat from Home must preserve/resume conversation context and must not auto-create a new conversation. Letters are frontend-only until an approved backend adapter is wired; sample content must remain explicitly illustrative, and personalized letters require server-owned conversation provenance. See `docs/integration/LETTERS_FRONTEND_BOUNDARY.md` and `docs/integration/UX_REFINEMENT_2026_11.md`. Existing reflection API/store modules and backend data remain untouched.

## Temporary onboarding exclusions
`testimonials`, `research`, and `experts` are temporarily excluded from onboarding by user request. Keep their components/assets and the canonical `FLOW`/hamrah-v1 ordinals intact. `EXCLUDED_ONBOARDING_ROUTES` in `src/features/profile/navigation.js` controls the visible flow (21 screens). Welcome goes to Name, whose original persisted ordinal remains4. Advance excluded ordinals sequentially through the existing API; never renumber saved progress or shorten the remaining intake. Research remains available from the completed-user About menu.

## HAMRAH-INTEGRATION-05 — Full onboarding restoration
The original designer's complete 24-step pre-Home onboarding is reachable again for new users. `src/features/profile/navigation.js` defines the allowed progression; `src/screens/Onboarding.jsx` retains original screens. **Never reintroduce the short `name → approach → ready` path.** Saving `displayName` + `conversationStyle` happens on final `ready` only, and sensitive answers remain ephemeral. Incomplete refresh now resumes at a server-owned non-sensitive route cursor (Task06). Sensitive answers remain ephemeral; see `docs/integration/TASK06_ONBOARDING_RESUME.md`.

---

## HAMRAH-INTEGRATION-03 — Connected Chat and History
- This incremental frontend patch **requires HAMRAH-INTEGRATION-02 v2** (fictional phone login). Never apply the rejected seeded-user Task 02 patch.
- Chat and History use the existing backend contract; no new API, MSSQL migration, schema or provider feature was added.
- `src/features/conversation/api.js` is the sole conversations transport and `store.js` scopes messages/turns by the current `/me` user UUID. It clears on tester switch/logout/401 and fences stale asynchronous responses.
- `src/screens/Main.jsx` now renders **server** message content, real history, delete-confirmation and explicit failed-turn Retry; no canned AI response, generated insight or fake saved-message action.
- After 409/503/network errors retry **the same** `clientTurnId` and text. Use `GET /messages` after success to reconcile stored message pairs. A lost response from the initial conversation **creation** can still leave an empty orphan: create itself is not idempotent, record as a future backend improvement.
- Run `npm test`, `npm run check:foundation`, `npm run build`, and opt-in `npm run test:e2e:chat` with real local Go/MSSQL Mock AI. Only `npm test` and structure can be claimed as passed in this sandbox.

---

# AGENTS — Hamrah (همراه) frontend, new designer-source edition

**Main rule: the attached `ai-therapist-frontend(1).zip` is the authoritative product UI design.** Preserve the actual React/JSX screens, `src/styles.css`, route flow and original illustrations. Do NOT import or rebuild the earlier Rahrah/Wellbeing frontend, and do not silently turn the project into TypeScript, Tailwind or React Router.

## Read in this order
1. `MVP_SCOPE.md` (what is ready for the currently approved *internal* build).
2. `PRODUCT_CONSTITUTION.md` (identity, sensitive claims, data limits).
3. `TECHNICAL_CONSTITUTION.md` (actual JSX/CSS/Vite architecture and rules).
4. `FRONTEND_DESIGN_SPEC.md` and `docs/design/SOURCE_PRESERVATION_AUDIT.md` (frozen new visual baseline).
5. `REPO_MAP.md` (routes/screens/source ownership).
6. `docs/integration/UI_BACKEND_GAP_MATRIX.md`, `docs/integration/BACKEND_CONTRACT.md`, then `docs/roadmap/BACKEND_FRONTEND_INTEGRATION_PLAN.md`.
7. Appropriate `.agents/skills/<name>/SKILL.md` and its linked references before modifying that area.
8. `VALIDATION_REPORT.md` and `NEXT_AGENT_TASKS.md`.

## Product decision hierarchy
- **Source of truth for look, brand, interaction layout, and intended user journey:** this *new* designer JSX source, under the Persian brand **همراه**.
- **Source of truth for what the currently validated backend actually supports:** current backend `contracts/openapi/openapi.yaml` and Go/MSSQL implementation; the frontend OpenAPI snapshot is only a reference copy.
- **Current testing restrictions:** internal / local / synthetic data only; don't imply the UI is clinically safe or publicly released. When a designed flow asks for something the backend does not support, **log a product/backend gap**, preserve the design, do not fake connected behavior or invent endpoints.
- **Unsupported claims are not product facts:** testimonials, 6000 ratings, 14 research articles/university associations, named clinicians, 'saved on phone' and 'delete account' copy are unverified or contradict the local prototype. Do not display as real to any real user; correct or gate before connecting testers while preserving layout.

## Present implementation (do not assume more)
- React 19 + **JavaScript/JSX**, Vite 6, lucide-react and custom CSS; no React Router, Tailwind or typed API client.
- `src/App.jsx`: `#/route` hash navigation and temporary local synthetic session via backend cookie. Chat/history still unconnected and must not display fake AI replies.
- `src/screens/Onboarding.jsx`: 25-route journey, mostly *locally held answers*. Some ask health / belief / age / therapy history. These data **must not silently become server-collected**.
- `src/screens/Main.jsx`: Home, health-related survey, a static day card, static History, menu, mock chat and static AI Insight. Chat's `setTimeout` reply is a demo, not AI.
- `src/data.js`: demo questions, voice choices, approach choices, sample conversation and unverified review copy.
- CSS/fontrendering and `public/assets/**` are design artifacts; preserve.

## Implementation rules
1. Freeze the source (manifest + screenshots on real device when available). Add API/state logic in `src/core/api/*.js` and feature folders as needed, then wire screens one at a time. Do not mass-replace JSX/class names.
2. Browser always calls relative same-origin `/api/v1/*` through localhost Vite proxy, `credentials:'include'`. No direct LLM/MSSQL access, cookies or secrets in JS, URL, localStorage/sessionStorage or logs.
3. Hamrah v2 uses a **phone-shaped internal tester identifier** (`09xxxxxxxxx`), never a dropdown of seeded testers. Signed HttpOnly revocable session lives on backend. On tester switching/logout, clear all tester-scoped React data and reject stale pending responses.
4. Currently supported backend profile: displayName + `listening|guidance|balanced` + onboardingCompletedAt. Map the design's approach choices explicitly and review ambiguities; do **not** add 20 sensitive intake attributes to PATCH `/me/profile`.
5. Chat is **JSON/non-streaming**. New turn has `clientTurnId` UUID, same UUID **and exact text** on retry. No fabricated reply when the request fails. GET messages/list and server 401/404/409/503 govern UI behavior.
6. Timed sessions, surveys/insight, voice generation, bookmarks, regeneration, analytics, clinical questionnaires and privacy sharing are UI designs but **not implemented backend capabilities**. Preserve appearance and mark them demo/disabled until approved and implemented.
7. Update `docs/integration/UI_BACKEND_GAP_MATRIX.md` whenever a screen is connected. Verify visual regression at 360/390/430px, desktop, keyboard and reduced motion.
8. `npm run check:foundation`, `npm run check:source-baseline` (only before intentional source changes), `npm test`, `npm run build` and connected Playwright should be reported individually as PASSED / FAILED / BLOCKED / NOT RUN. Do not equate source audit to runtime verification.
9. Do not run schema or deployment changes on production/shared databases. Only synthetic local tests; backend currently MSSQL 2022. No public access/auth or real patient data.

## HAMRAH-INTEGRATION-02 v2
The previous seeded-user-picker frontend patch was rejected and is NOT a prerequisite. Use `POST /api/v1/dev/identity-session` `{phone}` with synthetic `09xxxxxxxxx` values only. The UI must not call `/dev/users`. `GET /me` decides returning Home vs incomplete Onboarding. This v2 frontend preserves designer components/assets but gates nonintegrated/sensitive routes. See `docs/integration/PHONE_IDENTITY_V2.md`.
