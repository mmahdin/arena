# Letters redesign — read-only inbox and reader (2026-11)

Supersedes the UI description in `LETTERS_FRONTEND_BOUNDARY.md` and `LETTERS_BACKEND_CONNECTED.md`.
Store, API adapter, model, provenance rules and device-local read receipts are unchanged.

## Product rules
- Letters are only received. The screen has no compose, "write me a letter" or sample tab.
  `letterApi.generate` / `store.generate` remain in the code but nothing in the UI calls them.
- One route (`#/letter`) with two internal views: **inbox** and **reader**. Back in the reader returns to the inbox; back in the inbox returns Home. `Esc` closes the reader.
- Opening a letter marks it read (device-local ids only, as before).
- The end of every letter offers one quiet action, **دربارهٔ این نامه حرف بزنیم**. It starts a new conversation
  (`chatStore.startNew()`) with an empty composer and a dismissible letter reply chip. When the
  person sends their first written message, the client includes the letter title and text (trimmed
  to the 4000-character chat API limit) in the outgoing message, while the UI renders only a small
  reference and the person's own words. The backend still has no separate letter-context parameter;
  this text-based context is persisted in the first chat turn and remains available to later turns.
- Deleting received letters is not exposed in the interface.

## Files
- `src/screens/Letters.jsx` — new screen (`Inbox`, `LetterReader`).
- `src/features/letters/discuss.js` — discussion draft builder.
- `src/App.jsx` — `onDiscussLetter`; removed `onCreateLetter` and generation props from shared props.
- `src/styles.css` — old `.letter-*` screen rules replaced by the "Letters: a quiet inbox…" block.
- Tests: `scripts/letters-redesign.test.mjs` (new); `letters-integration`, `ux-refinement`, `profile-navigation` assertions updated.
