# Frontend UX refresh — November 2026

## Implemented without API changes

- Voice choices use swipe, keyboard, compact previous/next controls, and selectable pager dots.
- Mood and reminder headers follow the shared onboarding rhythm. The mood thumb and ticks share one five-position track.
- Conversation style keeps one user message fixed while the companion response, selected label, and persisted five-style preference change together.
- The questionnaire gives the two-week context primary hierarchy and presents responses as one consistent scale. Content and local selection behavior are unchanged.
- Daily Reflections/journaling and the old static Insight screen are no longer routed or loaded. The existing `src/features/reflection/**` transport/store and backend APIs remain untouched.
- Home contains the letter space and conversation entry only; the Daily Reflect card has no replacement feature.
- Chat resumes the latest conversation on ordinary entry. A new thread is a local empty state until the first send; existing create/turn/messages/list/rename/delete/bookmark contracts are unchanged. History, switching, and creation are available together in the chat sheet.
- Chat remains JSON/non-streaming. Retry preserves the exact `clientTurnId` and text.

## Personalized letter frontend contract (not connected yet)

`src/features/letter/model.js` validates a future server-authored object. `App.jsx` currently supplies `null`, so Home and the letter reader show an honest empty state. There is no demo letter, generated insight, fabricated pattern, or frontend request to an invented endpoint.

Proposed response shape for a future `GET /api/v1/me/letter` (route requires backend/product approval):

```json
{
  "id": "server-letter-id",
  "title": "Backend-authored title",
  "createdAt": "2026-11-04T10:00:00Z",
  "greeting": "Optional greeting",
  "blocks": [
    { "type": "paragraph", "text": "A grounded paragraph" },
    { "type": "heading", "text": "Optional short section" },
    { "type": "paragraph", "text": "More writing" },
    { "type": "question", "text": "An optional reflective question?" }
  ]
}
```

`204 No Content` (or an explicitly agreed empty response) should map to `null`. The frontend accepts `heading`, `paragraph`, and `question` blocks only and requires at least one paragraph. It does not interpret diagnoses, scores, citations, or evidence.

## Backend work required next phase

1. Add an authenticated, owner-scoped personalized-letter read contract after privacy and product approval. Define empty, loading/error, and unread/read behavior if a notification badge is desired. The current red menu badge is only the existing visual state and does not claim an unread letter.
2. Generate letters only from authorized, genuine conversation evidence. Include safeguards against inferred/fabricated facts and psychological diagnoses; define retention, consent, regeneration cadence, and deletion behavior with the conversation data lifecycle.
3. Return the structured content above (or version the contract before integration). The backend owns all personal claims and reflective questions.
4. No backend change is required for the refreshed onboarding controls, conversation-style persistence, chat switching, or removal of user-facing journaling. Daily Reflection APIs may remain deployed but unused.

## Existing non-blocking API limitations

Conversation list/messages remain capped at 100 without pagination, and conversation creation itself is not idempotent after a lost create response. These existing limitations are unchanged by this frontend phase and should be addressed separately if scale/reliability requirements demand it.
