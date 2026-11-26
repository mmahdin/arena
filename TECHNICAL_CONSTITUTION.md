# Technical Constitution — new Hamrah React source

## Actual stack and ownership
- `react`/`react-dom` 19, Vite 6, JavaScript ES modules + `.jsx` (NO TypeScript requirement), `lucide-react`, CSS in `src/styles.css`.
- Three local Persian font packages imported by `src/main.jsx`: Estedad, Vazirmatn, Noto Naskh Arabic. `dir="rtl"` in `src/App.jsx`; confirm `lang="fa"` in index.
- Hash-route navigation in `src/App.jsx`; screens selected by the existing `FLOW` plus `EXTRA_ROUTES`. No React Router; do not assume file-based routing.
- `src/screens/Onboarding.jsx` holds the long illustrated intake; `src/screens/Main.jsx` holds Home/Survey/Day/History/Drawer/Chat/Insight; primitives/art in `src/components/UI.jsx` and fixtures in `src/data.js`.
- `src/styles.css` sets a max-430px mobile canvas with color tokens (`--light`, `--ink`, `--beige`, `--dark`, `--lavender`, `--green`, `--yellow`). Avoid swapping in an old Tailwind kit.
- Navigation/current profile/messages are local React state, not persisted or secured; current chat produces a delayed `setTimeout` reply. This code is **not yet connected**.

## Incremental extension, not replacement
- Keep all original runtime source files byte-identical in this packaging; subsequent feature work should preserve markup and CSS class names unless necessary for functionality.
- Add `src/core/api/http.js` for all relative `/api/v1` calls, and feature adapters in `src/features/{devsession,profile,conversation}` incrementally. Use `AbortController` / request generation gating to ignore old tester results; don't refactor entire visual tree in one pass.
- Use same-origin Vite proxy target `http://127.0.0.1:8080`, browser `http://127.0.0.1:5173` (avoid mixed localhost/127 cookie context). Only Dev Login sends `{userId}`; never use frontend IDs to authorize requests.
- Keep `credentials:'include'`. Server uses signed HttpOnly, MSSQL-revocable cookie. Do not expose raw session token to JS or persist chat/profile in browser storage.
- `POST /turns` is JSON, NOT streaming. Retain `clientTurnId` across timeout/retry and avoid duplicate optimistic bubbles; reconcile from GET messages after ambiguous failures.
- All UI navigation and synthetic display logic is frontend-owned. SQL and LLM keys, profile ownership, turn lease and AI safety are backend-owned.

## Current backend truth and drift
Backend is Go/Gin + Microsoft SQL Server 2022. This package carries a **Task07-derived snapshot** at `contracts/openapi/backend-openapi.snapshot.yaml`; fetch/sync the latest checked-out backend's `contracts/openapi/openapi.yaml` before building an adapter. Previous validation fixes were reported to be non-API. Do not invent voice, assessment, insight, review, saved-message, report or regenerate endpoints.

## Validation
- Foundation checks: `npm run check:foundation`; `npm test` runs Node structural/contract checks.
- Runtime: `npm ci`, `npm run build` and browser screenshots at 360/390/430px and desktop; verify smartphone keyboard, long Persian text, back navigation and `prefers-reduced-motion`.
- Full-stack: real Go + MSSQL + `LLM_PROVIDER=mock`, include session switch, profile gate, history persistence, 401/404/409/503, dropped-network stable-UUID retry and E2E tests; never claim mock API interception is real integration.
- No destructive MSSQL or unapproved deployment and no actual mental-health disclosures.
