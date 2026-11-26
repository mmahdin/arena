# Five Conversation Tones — frontend validation record

**Scope:** one-to-one Approach -> `conversationStyle`, legacy `guidance` reading, same original UI and existing `PATCH /me/profile` API. Onboarding now saves the shown default card on confirmation, even if the user did not swipe.

## Executed here

- `npm run check:foundation`: PASS (41/41).
- `npm test`: PASS (51 Node tests, including API transport, all five mappings, legacy value and default choice regression).
- `node --check scripts/e2e-five-tones.mjs`: PASS.
- Backend and frontend OpenAPI snapshots: synchronized byte for byte in this delivery.
- `npm run build`: BLOCKED — Vite executable unavailable because node dependencies are not installed in this container.
- Live Vite -> Go -> MSSQL browser test: NOT RUN here.

## Local acceptance

1. Apply Backend patch and run migration 0011 first, then apply Frontend patch.
2. Run `npm ci && npm test && npm run check:foundation && npm run build`.
3. Start Vite, Go/Gin, Microsoft SQL Server with `LLM_PROVIDER=mock`, and run `npm run test:e2e:tones`. This navigates the original Approach screen for each choice, saves through the real API, reloads, verifies the preserved choice, and restores the original preference.
4. Run existing session, full onboarding, profile preferences, chat and bookmarks browser regressions. Check no pages or visual assets were altered.
5. Inspect a synthetic local provider request/Go prompt unit tests to verify per-style directives; do not expect different response text from the deterministic mock gateway.

The user-selected tone affects newly generated turns after save, not previously committed or idempotently replayed turns.
