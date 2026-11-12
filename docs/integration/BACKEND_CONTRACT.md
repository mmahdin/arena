## 2026-11-11 profile extension

`PATCH /api/v1/me/profile` optionally accepts `onboardingAnswers`; omission preserves
saved selections and `{}` clears them. GET `/me` returns them when present. Final
Ready submits only explicit allowlisted choices; name/style settings edits omit
the field. Migration 0016 must precede the new API and frontend. Authenticated
`GET /api/v1/me/ai-status` reports configured provider/model/mock/promptVersion only;
it does not verify provider connectivity. The backend OpenAPI and synchronized
snapshot specify field bounds and schemas. Earlier name/style-only restrictions
below are superseded for internal synthetic onboarding personalization.

## Task 03 — verified contract usage (no new endpoints)
- Frontend `src/features/conversation/api.js` implements list/create/read messages/delete/submit turn using the existing `GET/POST /api/v1/conversations`, `GET /api/v1/conversations/{id}/messages`, `DELETE /api/v1/conversations/{id}`, and `POST /api/v1/conversations/{id}/turns`.
- `store.js` uses `credentials:'include'` via `src/core/api/http.js`, body `{clientTurnId, message}`, and the same UUID/text on any retry. `401` invalidates session; 404 inaccessible thread; 409 in-progress turn; 503/provider issues are recoverable. Browser never calls LLM/MSSQL directly.
- Server's 100-item limits and lack of conversation-creation idempotency are explicitly documented. No streamed response, voice, insight, bookmark or regenerate endpoint is implied.
- No backend changes are required for this vertical slice. Run the real local Go/MSSQL + Mock AI acceptance test to establish runtime compatibility.

---

# Backend API contract — available validated Task07-era snapshot

**Authority:** an up-to-date backend checkout's `contracts/openapi/openapi.yaml`, not this snapshot. This frontend-only upload has no newer backend ZIP, so the bundled YAML is copied from the prior Task07-derived frontend reference. User reported subsequent MSSQL migration/lease/deadline fixes with unchanged public HTTP routes. Recheck the actual server when integrating.

| HTTP | Relative path | Use on Hamrah UI |
|---|---|---|
| POST | `/api/v1/dev/identity-session` | `{phone: "09000000001"}` (fictional only) → `{ok:true,created:boolean}` and signed cookie; new user gets incomplete profile |
| GET | `/api/v1/dev/users` | legacy test route; **NOT used by Hamrah UI** |
| POST | `/api/v1/dev/session` | legacy seeded-tester session route; **NOT used by Hamrah UI** |
| DELETE | `/api/v1/dev/session` | logout; backend revokes session |
| GET | `/api/v1/me` | profile identity and onboardingCompletedAt |
| PATCH | `/api/v1/me/profile` | `{displayName,conversationStyle}` where style is `listening`, `guidance`, `balanced` |
| GET | `/api/v1/conversations` | `{items:[{id,title,createdAt,updatedAt}]}` list, max 100 |
| POST | `/api/v1/conversations` | create new owned conversation; server onboarding gate |
| GET | `/api/v1/conversations/{id}/messages` | `{items:[{id,role,content,createdAt}]}`; last 100 ordered |
| DELETE | `/api/v1/conversations/{id}` | delete owned thread; confirmation in UI |
| POST | `/api/v1/conversations/{id}/turns` | `{clientTurnId,message}` → JSON `{userMessageId,assistantMessageId,reply,promptVersion}` |
| GET | `/health/live`, `/health/ready` | backend service health |

## Network + cookies
- Use `fetch('/api/v1/...', {credentials:'include'})` through Vite proxy at `127.0.0.1:5173` → API `127.0.0.1:8080`. Don't mix hostnames. Cookie is server-issued `HttpOnly`, signed and MSSQL-revocable, no JS access.
- Hamrah local tester identity is resolved via `/dev/identity-session`; all other user-scoped requests derive ownership on the backend from session. No auth tokens in local/session storage.
- Errors: `401` invalid session; `404` missing/not-owned conversation; `409 ONBOARDING_REQUIRED` / `TURN_ALREADY_PROCESSING`; `503` AI/service failure; `application/problem+json` with `code` and `status`.
- Retry the **same `clientTurnId` and exact text** after a timeout/dropped reply; reconcile persisted records by fetching messages. Never assume request failure means DB did not commit.
- No SSE, voice transport, survey, insights, bookmarks, regeneration, report-feedback, daily letter or review endpoint exists in this contract.

## Onboarding UI → profile mapping
- Name screen → displayName, normalized to server constraints.
- Approach screen is not directly persisted. Proposed mapping requires a *product approval* (not yet coded):
  - `همدلی و شنیدن` → `listening`;
  - `راهنمایی آرام`, `حل مسئله با هم`, `به من یاد بده` → `guidance`;
  - `گفت‌وگوی آزاد` → `balanced`.
- Voice choice is purely a presentation option, not conversationStyle and not a backend model voice. No other questionnaire response should be sent through profile API.

## Drift and acceptance
When backend checkout is available, diff actual OpenAPI with snapshot before coding. Run backend contract check and update snapshot only after confirming contract changes. Run connected HTTP tests using **actual** Go/MSSQL and `LLM_PROVIDER=mock`; browser mock interception alone is not acceptance.
