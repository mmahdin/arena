# Personalized Letters & UI/UX Improvements — Backend Handoff

## Scope of This Frontend Update
- **Frontend-only changes:** No Go handlers, SQL Server migrations, or existing `/api/v1/*` schemas were altered.
- **Journaling removed from UI:** Home screen's Daily Reflection card (`یادداشت روزانه`) and the journaling editor/archive on `#/day` have been completely removed from the UI, and the frontend no longer issues requests to `/api/v1/me/daily-reflections*`. Existing backend daily-reflection routes and frontend adapter modules (`src/features/reflection/*`) are left intact so nothing breaks on the server.
- **Multi-conversation Chat UX:** Reuses the existing `/api/v1/conversations` (`GET`, `POST`, `PATCH`, `DELETE`), `/messages` (`GET`), `/turns` (`POST` with idempotent `clientTurnId`), and `/me/bookmarks` endpoints. Opening Chat now restores the user's most recent conversation (`chatStore.enterChat()`) rather than resetting to an empty conversation.
- **Personalized Letters UI & State Layer:** Prepared in `src/features/letter/model.js` and `src/features/letter/store.js` (`createLetterStore`) with full session/identity fencing and validation for both short and long letters, while defaulting to an honest awaiting state (`api = null`, zero fake API calls, zero fabricated insights).

---

## Required Future Backend Work

### 1. Personalized Letters API & Generation Pipeline (Required)
Personalized letters provide psychological insights, self-awareness observations, and recurring behavioral patterns synthesized from the user's **actual owned conversations** (respecting user consent / privacy preferences).

#### Proposed HTTP Endpoints (Same-Origin Cookie Auth via `devSession` / future auth)
1. **`GET /api/v1/me/letters`** — List recent personalized letters for the authenticated user (newest `createdAt` first, e.g. limit 50).
   ```json
   {
     "items": [
       {
         "id": "33333333-3333-4333-8333-333333333333",
         "title": "وقتی مراقب همه هستی، جز خستگیِ خودت",
         "excerpt": "توی گفت‌وگوهای اخیرت الگویی دیده می‌شه از مسئولیت‌پذیری زیاد...",
         "createdAt": "2026-11-04T09:00:00Z",
         "readAt": null,
         "themes": ["شناخت خود", "مرزهای شخصی"]
       }
     ]
   }
   ```
2. **`GET /api/v1/me/letters/{id}`** — Fetch full letter content (owner-scoped; `404` if missing or owned by another user).
   ```json
   {
     "id": "33333333-3333-4333-8333-333333333333",
     "title": "وقتی مراقب همه هستی، جز خستگیِ خودت",
     "excerpt": "توی گفت‌وگوهای اخیرت الگویی دیده می‌شه از مسئولیت‌پذیری زیاد...",
     "salutation": "برای نگار عزیز،",
     "quote": "جمله‌ای کوتاه یا مضمونی واقعی از گفت‌وگوهای خود کاربر (اختیاری)",
     "paragraphs": [
       "بند اول نامه...",
       "بند دوم نامه..."
     ],
     "signoff": "با مهر، همراه",
     "themes": ["شناخت خود", "مرزهای شخصی"],
     "createdAt": "2026-11-04T09:00:00Z",
     "readAt": null
   }
   ```
   *(Note: `parseLetterDetail` in `src/features/letter/model.js` accepts either `paragraphs: string[]` or `body: string` separated by blank lines, supporting both concise 1–2 paragraph letters and long multi-paragraph letters.)*
3. **`POST /api/v1/me/letters/{id}/read`** (or `PATCH /api/v1/me/letters/{id}`) — Idempotently mark a letter as read so Home's envelope wax-seal unread indicator updates accurately:
   ```json
   { "readAt": "2026-11-04T10:15:00Z" }
   ```
4. **Optional Feedback Endpoint — `PUT /api/v1/me/letters/{id}/feedback`:**
   - Body `{ "rating": "up" | "down" }` if product wants to persist letter helpfulness ratings (currently local UI state on real letters).

#### Backend Generation, Storage & Safety Considerations for Letters
- **Eligibility threshold:** Only generate a letter once the user has enough real conversation history (e.g., minimum N user turns across recent conversations) — never fabricate observations when conversation data is insufficient.
- **Safety & tone guardrails:** Prompt policy should enforce non-diagnostic, non-judgmental observational Persian prose (`supportive` policy boundary), avoiding clinical diagnoses or deterministic claims.
- **Database schema:** Owner-scoped `user_letters` table keyed by `(id UNIQUEIDENTIFIER, user_id UNIQUEIDENTIFIER)` with `title`, `excerpt`, `salutation`, `quote`, `body_json` / `paragraphs`, `themes_json`, `signoff`, `created_at`, `read_at`, and source conversation window metadata; cascade-deleted or revoked per account retention rules.
- **Frontend wiring step once ready:** Implement `letterApi` in `src/features/letter/api.js` using `request('/api/v1/me/letters...')` and pass `letterApi` instead of `null` to `createLetterStore(letterApi, ...)` in `src/App.jsx`.

---

### 2. Optional Backend Enhancements for Existing Features
1. **Voice Preference Persistence & TTS/Audio (Future / Optional):**
   - Onboarding's `VoiceScreen` (`بید`, `سرو`, `نسیم`, `آواز`, `سنگ`, `برگ`) currently keeps the selected voice in ephemeral onboarding state (`answers.voice`) as designed.
   - If voice synthesis or persistent voice selection is introduced later:
     - Add `voiceId` (`bid | sarv | nasim | avaz | sang | barg`) to `/api/v1/me` and `PATCH /api/v1/me/profile` (or `/api/v1/me/preferences`).
     - Provide static or generated voice preview audio samples for onboarding and TTS playback in Chat.
2. **Mental Health Questionnaire (`Survey` — PHQ/Check-in) Persistence (Future / Optional):**
   - `Survey` (`#/survey`) remains client-memory UI until clinical/privacy review defines whether questionnaire items, multi-question instruments (e.g. full PHQ-2/PHQ-9 or GAD-7), and longitudinal scoring should be persisted on the server.
3. **Check-in Reminders / Push Notifications (Future / Optional):**
   - `CheckinsScreen` (`#/checkins`) is informational during onboarding. Supporting real reminders later will require push subscription/notification preference endpoints and a scheduler.
4. **Multi-Conversation Quality-of-Life Improvements (Non-blocking):**
   - `GET /api/v1/conversations` already returns conversations ordered newest-first (`updatedAt` DESC), which powers restoring the most recent conversation and the in-chat switcher.
   - Optional future improvements:
     - Include `messageCount` or `lastMessagePreview` on `Conversation` summary items if richer previews are desired in History.
     - Optional `clientRequestId` idempotency key on `POST /api/v1/conversations` to prevent empty orphan threads if the network drops during initial conversation creation (already noted in `AGENTS.md`).
     - Cursor pagination once users exceed 100 conversations or 100 messages per thread.
