## Onboarding chat context amendment — 2026-11-11

Final Ready now sends explicit allowlisted questionnaire selections as
`onboardingAnswers` with name/style. Backend migration 0016 stores them on the
session-owned profile, and `supportive-v5` uses them on every new chat turn, including
after reload. Draft answers remain ephemeral; prior discarded answers cannot be
recovered. Gender has three options: زن، مرد، تمایل ندارم بگم. Privacy/Ready disclose
server storage and AI processing. This supersedes earlier ephemeral-only intake
notes below; it adds no inferred traits, clinical assessment or public release.

## Current implementation status — UX refinement (2026-11-04) supersedes older product-shape notes
- **Connected:** fictional phone-based Dev Session; full illustrated onboarding cursor; real Chat create/send/retry, message history, History list/open/continue/rename/delete, and saved messages through the existing Go/MSSQL contract.
- **Conversation continuity:** opening Chat from Home no longer creates a conversation. The current in-memory thread is preserved; after reload the frontend restores the latest server-owned thread via existing `GET /conversations` + `GET /messages`. New Conversation is explicit and separate.
- **Letters:** Home envelope opens a frontend-only letter experience. Personalized letters are **unavailable** until a future approved adapter is wired; the UI shows an explicitly labeled illustrative sample plus loading/empty/error states. No letter endpoint, persistence, or personal inference is implemented. See [Letters frontend boundary](LETTERS_FRONTEND_BOUNDARY.md).
- **Reflection/diary removed from active product:** Home reflection card and diary-writing UI are removed. Legacy `#/day` and `#/insight` hashes resolve to Letters. Existing reflection API/store modules and backend data are left untouched; they are no longer wired into App.
- **Still design-only/gated:** personalized letters, survey scoring, clinical intake persistence, voice/TTS, regeneration/report, provider-quality claims, account deletion and metrics.
- **Missing backend contract for future review:** pagination (list/messages cap 100), idempotent conversation creation on a lost response, personalized-letter generation/provenance, and advanced message actions. Do not add these without separate scope and testing.
- Backend OpenAPI is the runtime schema source; frontend snapshot is a reference. This refinement adds no endpoint, schema, migration, or API contract change.

---

# Actual Hamrah UI vs current Backend: gap matrix

This matrix is based on **the uploaded JSX source**, not an imagined frontend. UI source is authoritative for visual/product intent; backend OpenAPI determines supported server behavior. See `BACKEND_CONTRACT.md`.

| Existing UI (file/route) | Current UI action/data | Backend capability today | Next action and scope decision |
|---|---|---|---|
| `App.jsx` / all routes | Hash routing, no session guard; default fictional name; transient answers/messages | server-signed dev sessions and profile exist | add test-user gate, bootstrap `/me`, guard connected routes, preserve UI shell |
| `Onboarding.jsx` / `welcome` | Start illustrated multi-step flow | no user account signup; dev users available | add internal selector matching UI visual design, not production login |
| `testimonials` | 6,000+ reviews and named reviews | none | preserve card layout; unverified marketing evidence cannot be shown as facts |
| `research`, `experts` | 14 articles, universities, named doctors/credentials | none | preserve compositions/art; remove or gate assertions pending substantiation/legal review |
| `name` | default name in App state, input | `PATCH /me/profile` displayName | load actual tester profile; save validated value |
| `privacy` | locally toggled `shareAnonymized`, says data stored on phone/delete account | chat stored on server MSSQL; no account-delete/consent API | remove false privacy claims, default opt-in off/gate until real policy exists |
| `therapist`, `age`, `concerns`, `focus`, `feelings`, `neurodivergent`, `faith`, `interests`, `relationships`, `strengths`, `barriers` | local answers only, some health/religion data | no approved fields or safe collection pipeline | keep design references; short connected onboarding or approved extension with privacy review |
| `voice` | swipeable voice deck, active card + dots, explicit no-playback note | no audio/voice service | keep visual selection only; do not suggest TTS/model voice or fake playback |
| `approach` | minimal open chat preview: one shared prompt, five concise replies, swipe/arrow/dot/keyboard picker | five persistent style values | preserve options/mapping; preview is illustrative and never sends a turn; see `docs/design/APPROACH_REDESIGN_2026_11.md` |
| `ready`, `home` | greeting, envelope opens Letters, single Chat CTA resumes current thread | `/me` + conversations; no personalized letters | keep sample/unavailable letter states honest; future letter adapter required |
| `survey` | readable context + primary question, native radio options, local response, automatic navigation | no assessment or clinical scoring | demo/design only pending explicit clinical/safety/scope review |
| `day` | legacy hash redirects to Letters; diary UI removed | reflection API exists but is not wired | do not restore diary entry point without product approval; preserve existing data/backend |
| `history` | real conversation list, active marker, rename/delete, saved messages, quiet new action | `GET /conversations`, `GET /messages`, PATCH/DELETE | continue using existing contract; no weekly progress inference |
| `chat` | real messages, conversation sheet, explicit new thread, resume latest after reload, bounded composer | `POST /conversations`; `/turns` JSON; GET `/messages` | protect retry UUID/text, pending-turn switch confirmation, no auto-create on Home entry |
| `chat` empty state | warm empty state; no predefined reflection prompts | none | do not reintroduce diary/reflection starters without approval |
| message actions: copy | clipboard API | frontend-only | keep, accessible/error feedback |
| message actions: read aloud | Web Speech synthesis when available | no backend voice | optional browser-only feature with truthful UX; no model-generated speech promise |
| message actions: save | real bookmark toggle/list through existing API | bookmarks exist in current frontend contract | keep server-confirmed state; no fake saved result |
| message actions: regenerate/report | removed from active UI | no regenerate/report endpoints | reintroduce only after approved backend contract |
| `insight` | legacy hash redirects to Letters | no insight or feedback endpoint | no AI inference claims; sample is explicitly illustrative |
| menu: logout | local React reset | `DELETE /dev/session` | revoke server session, clear UI scoped state and outstanding requests |
| menu: preferences | navigates to approach | `PATCH /me/profile` | connected profile setting with explicit mapping |
| menu: support/emergency | link-copy and placeholder UI | no safety service | don't promise live dispatch or fabricated helpline |

## Primary blockers / decision log

UX refinement update (2026-11-04): Home's reflection card and active diary-writing UI were removed by product request. The envelope now opens Letters, a backend-ready frontend experience with an explicitly labeled sample and honest unavailable/loading/empty/error states. Chat entry preserves conversation continuity and exposes New Conversation through a quiet conversation sheet. Existing reflection backend/API modules are untouched and existing user data is not deleted. See [Letters frontend boundary](LETTERS_FRONTEND_BOUNDARY.md) and [UX refinement notes](UX_REFINEMENT_2026_11.md).

Historical Task08 update (2026-11-02): Home's reflection card previously opened `day`, connected to owner-scoped `GET/PUT/DELETE /me/daily-reflections/{date}` and `GET /me/daily-reflections`. Explicitly authored entries persist in MSSQL; drafts stay in React memory. That frontend entry point is now retired, but the backend capability and data remain. See [Task08](TASK08_DAILY_REFLECTIONS.md).

1. **Live core integration blocker (frontend):** no fetch/API client/session guard/real Chat/History implementation, despite the polished UI.
2. **Privacy/product blocker:** if the entire long intake is required in the connected flow, it needs field-level necessity, consent, minimization and storage rules and possibly backend migrations. This is *not* a reason to redesign the screen.
3. **Capability choices:** voice, personalized insights/letters, assessments, message regeneration/bookmarks/feedback have UI design but no backend specification. New backend endpoints only after explicit prioritization and approval.
4. **Claim accuracy blocker:** review/statistics, research/affiliations, clinician credentials, phone-storage/delete-account assurances must be verified or replaced before public/real use.
5. **Data consistency blocker:** all current demo messages and `History` content are local hardcoded assets; connected mode must never show them as another tester's saved data.
