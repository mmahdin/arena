# Letters backend integration — incremental patch

- `src/features/letters/api.js`: `GET/POST /api/v1/me/letters` and `DELETE /api/v1/me/letters/{id}`; same-origin cookie, no browser persistence.
- `App.jsx` wires `createLetterStore(letterApi)`; identity changes and 401 responses clear letters.
- User must explicitly click **با رضایت من، نامه‌ای بنویس** on personal tab. API `POST` includes `{ consent: true }` and runs only on button activation.
- Reader shows saved letters and an archive. A letter is deletable without deleting the conversation; source conversation deletion cascades to its letters.
- Current backend generates from the latest conversation containing >=3 user messages, one letter per source conversation. The generator needs `LLM_PROVIDER=openai-compatible`; mock mode returns an honest unavailable error. UI keeps sample separate.
- The diary/reflection UI stays removed; existing reflection API/data is retained.
- Before deployment apply migration `0015_personalized_letters.sql` through the regular backend migration runner. Do not touch applied migration checksums.
- Limit: no real-user/clinical rollout approval. No external database+LLM end-to-end test was possible in the offline patching environment.
