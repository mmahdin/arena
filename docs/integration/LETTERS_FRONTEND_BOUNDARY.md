# Letters — frontend boundary and future backend integration

Date: 2026-11-04
Scope: frontend-only. No endpoint, schema, migration, or API contract was added or changed.

## Product intent

Hamrah has two distinct primary experiences:

- **Chat:** an ongoing, server-backed conversation with the AI companion.
- **Letters:** a slower reading experience for personalized insights/observations derived from meaningful conversational context.

Letters are not journaling, diary writing, daily reflection prompts, or a second chat. The active product no longer exposes the reflection card or diary-writing UI. Legacy `#/day` and `#/insight` hashes resolve to the Letters screen so old links do not reopen retired diary UI.

## Current frontend behavior

- Home envelope opens `#/letter`.
- The Letters screen has two tabs:
  - **نامه‌های من**: real personalized letters when a future backend adapter is connected.
  - **نمونه**: an explicitly illustrative sample letter.
- With no adapter connected, the personal tab shows an honest unavailable state. It does not pretend to load, generate, or personalize anything.
- The sample is labeled `نمونهٔ نمایشی` and the screen states that it was not created from the user's conversations.
- Loading, empty, error/retry, and ready states are implemented in the UI/store.
- A personalized letter may link to a related conversation through the existing conversation-open path when provenance is present.

Files:

- `src/features/letters/model.js` — normalization, provenance guard, sample content.
- `src/features/letters/store.js` — identity-scoped state machine with an optional future source adapter.
- `src/screens/Main.jsx` — `Letters` and `LetterPaper` UI.
- `src/App.jsx` — currently wires `createLetterStore(null)`.

## Future adapter interface

The store accepts an optional source object:

```js
createLetterStore({
  async list({ signal }) {
    // Call an approved, owner-scoped backend endpoint here.
    // Return an array of letter objects; throw ApiError on failure.
  },
}, { onUnauthorized });
```

To connect later:

1. Implement the approved backend contract and a thin transport adapter, e.g. `src/features/letters/api.js`.
2. Pass that adapter to `createLetterStore` in `src/App.jsx` instead of `null`.
3. Keep same-origin relative `/api/v1/...` requests, `credentials: 'include'`, AbortController fencing, and 401 session invalidation consistent with the conversation adapter.
4. Do not persist letters or provenance in localStorage/sessionStorage.
5. Do not generate personalized letters client-side or infer patterns from chat messages in the browser.

## Proposed letter data requirements

This is a frontend integration proposal, not an implemented API contract. Backend/product/safety approval is required before exposing personalized letters.

A personalized letter should include:

| Field | Requirement |
|---|---|
| `id` | Stable server-owned identifier. |
| `kind` | `'personalized'` for real letters. The frontend sample uses `'sample'` only. |
| `title` | Short, non-clinical title. |
| `salutation` | Optional greeting. |
| `paragraphs` | Non-empty array of thoughtful body paragraphs. |
| `question` | Optional self-awareness prompt. It must not open a diary editor. |
| `createdAt` | ISO timestamp owned by the server. |
| `sourceConversationIds` | Non-empty array of owner-scoped conversation UUIDs that actually support the letter. |

The frontend normalizer rejects a `personalized` letter without conversation provenance. This is intentional: a friendly tone or client-side flag is not evidence of personalization.

Backend responsibilities before real letters are shown:

- Derive letters only from meaningful, consented conversational context.
- Preserve provenance and owner isolation.
- Avoid fabricated personal observations, diagnosis, clinical scoring, or unsupported psychological claims.
- Provide safe empty/loading/error behavior and retryable failures.
- Define retention, deletion, and consent behavior separately from chat.
- Decide whether letters are generated asynchronously, on demand, or on a schedule. The current frontend only needs a list source; pagination/archive can be added later without changing the sample/demo boundary.

## Reflection/diary boundary

The previous daily-reflection frontend experience was removed from the active product by request. Existing modules remain for backend compatibility/history:

- `src/features/reflection/api.js`
- `src/features/reflection/store.js`
- `docs/integration/TASK08_DAILY_REFLECTIONS.md`

They are no longer imported by `App.jsx` or rendered by `Main.jsx`. No reflection data was deleted and no backend migration was modified. Do not reintroduce journaling or reflection prompts into Letters without a new product decision.

## Validation

- `npm test` covers letter normalization, provenance rejection, unavailable/loading/ready/error states, and removal of diary/fake controls.
- `npm run build` passes.
- Component-level browser QA verified sample labeling, personal-tab unavailable state, and absence of diary UI at 360/390/430px.
- Connected personalized-letter backend tests are **not run** because no backend adapter/endpoint exists in this patch.
