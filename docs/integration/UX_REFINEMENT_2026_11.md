# Hamrah frontend UX refinement — 2026-11-04

Frontend-only refinement preserving Hamrah's Persian RTL identity, palette, typography, floral/envelope artwork, and mobile-first shell. No backend code, schema, migration, or API contract was changed.

## Screen changes

### Voice selection

- Replaced the single ambiguous wave button with a swipeable deck of voice cards.
- Active card is centered, scaled, highlighted, and checked; inactive neighbors remain visible as affordances.
- Dots are labeled with voice names and remain selectable.
- Keyboard arrows/Home/End work on the carousel.
- Copy explicitly says there is no audio playback. No fake play button or TTS promise was added.

### Mood selection

- Heading moved upward to align with the onboarding rhythm.
- Slider ticks are positioned inside the native range thumb travel (`thumb / 2` inset), so the circular thumb centers exactly over each tick.
- Alignment was verified programmatically for all five values at mobile widths.
- Existing mood labels, notes, and answer behavior are preserved.

### Reminder screen

- Heading/supporting text moved upward with the same rhythm as Mood.
- Illustration and overall composition preserved.

### Conversation style selection

- Replaced text-replacement card with a realistic chat preview.
- The same illustrative user message is shown for every style.
- Each of the five original styles has a distinct authored assistant response.
- Visible style tabs make options discoverable; active tab is checked and announced.
- Swipe and keyboard navigation remain available.
- Preview is labeled as a sample and never sends a message or calls the conversation API.
- Original style names, mapping, skip/edit/save behavior preserved.

### Psychological questionnaire

- Context is larger and more readable; the actual question remains the primary focus.
- Added supportive spacing and native radio semantics with clear selected state.
- Existing question, answer options, local answer behavior, and automatic return home preserved.
- No scoring, persistence, or clinical claim was added.

### Home

- Removed the bottom Reflection card and diary entry point.
- Envelope now opens Letters, not diary UI.
- Home hint distinguishes illustrative sample from future personal letters.
- Chat CTA is a single clear entry point and shows the active conversation title when available.
- Removed unsupported notification dot from the menu button.

### Letters

- New `#/letter` route with sample/personal tabs.
- Sample content is explicitly illustrative and not falsely personalized.
- Personal tab supports unavailable, loading, empty, error/retry, and ready states.
- Personalized letters require server-provided conversation provenance before the frontend will render them as personal.
- Legacy `#/day` and `#/insight` redirect to Letters.
- See `LETTERS_FRONTEND_BOUNDARY.md` for future backend integration.

### Chat

- Opening Chat from Home no longer calls `startNew()`.
- Current in-memory conversation is preserved across navigation.
- After reload, Chat restores the latest server-owned conversation using existing list/messages APIs; it does not silently create a new one.
- New Conversation moved into a quiet conversation sheet opened from the header title/history control.
- Sheet shows active conversation, recent conversations, empty/loading/error states, and a management link to History.
- Switching away from an unsent failed turn requires confirmation; switching while a turn is sending is blocked.
- Composer drafts are kept in memory per conversation/new draft and cleared on send. No browser persistence was added.
- Removed nonfunctional attachment and voice-input buttons.
- Removed fake regenerate/report message actions; kept real copy, device speech, and bookmark actions.
- Added assistant author labels, clearer bubble structure, timestamps, message-action affordance, loading skeleton, warm empty state, jump-to-latest, and bounded multiline composer.
- Visual viewport height is tracked for mobile keyboard behavior; safe-area insets are respected.

## Backend boundaries

- Existing conversation creation, history, messaging, switching, rename/delete, and bookmarks remain connected to the existing backend contract.
- Existing reflection API/store modules and backend data are untouched, but the diary UI is no longer active.
- Letters have no invented endpoint or fake persistence. `createLetterStore(null)` keeps personalized letters unavailable until an approved adapter is supplied.

## Validation

- `npm test`: PASS (87 tests).
- `npm run build`: PASS.
- Component-level browser QA: PASS at 360/390/430px for home, letters, chat, history, survey, voice, approach, mood, reminder, menu; plus interaction checks for voice selection, style tabs, mood tick alignment, conversation sheet/new/resume, composer, message actions, and survey radio selection.
- `npm run check:foundation`: BLOCKED by pre-existing missing `.agents/skills` directory in this checkout; not caused by this refinement.
- Connected Go/MSSQL E2E suites: NOT RUN in this sandbox (no backend services). Run them against the real local stack before release, especially chat/history/titles/composer/transitions and onboarding.

## Known follow-ups

- Wire an approved personalized-letter backend adapter and provenance contract.
- Re-run connected E2E suites after backend availability; update any legacy selectors that still assume the retired reflection card or old top-level New Conversation button.
- Restore/replace the missing `.agents/skills` handoff assets if foundation verification is required in CI.
