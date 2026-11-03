## Current implementation status — Task 03 supersedes the historical design-gap observations below
- **Connected:** fictional phone-based Dev Session; short name/style onboarding; real Chat create/send/retry and message history; History list/open/continue/delete through Go/MSSQL.
- **Existing UI is still source of truth:** no redesign of original brand, palette, images or hash-route navigation. The designer's demo History weekly progress and fake chat replies are not shown in connected mode.
- **Still design-only/gated:** daily letters, insight, survey, clinical intake, voice, saved messages, regeneration, provider-quality claims, preference editing in menu, account deletion and metrics.
- **Missing backend contract for future review:** pagination (list/messages cap 100), idempotent *conversation creation* on a lost response, advanced message actions and approved insight content. Do not add these without separate scope and testing.
- Backend OpenAPI is the runtime schema source; frontend snapshot is a reference. This implementation adds no endpoint or migration.

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
| `voice` | swipe choice cards; button suggests voice preview | no audio/voice service | leave visual, do not suggest actual speech model or permission |
| `approach` | 5 style choices | backend 3 enum styles | approved mapping proposed in contract; save only enum |
| `ready`, `home` | entry, greeting, static reflection card and envelope | `/me` + conversations; no reflections/daily letters | show profile and real chat CTA; keep envelope as non-data visual or clearly demo |
| `survey` | 2-week anhedonia question, local response, automatic navigation | no assessment or clinical scoring | demo/design only pending explicit clinical/safety/scope review |
| `day` | static daily letter and insight card | no daily summaries/insight | design intent, do not imply stored/personalized insights |
| `history` | one hard-coded week row + weekly progress | `GET /conversations` list and `GET /messages` | replace list data, persist current convo, defer weekly score unless modeled |
| `chat` | `INITIAL_MESSAGES` + delayed canned `setTimeout` response | `POST /conversations`; `/turns` JSON; GET `/messages` | replace fake reply in connected mode, protect retry UUID and response ordering |
| `chat` suggestion buttons | predefined prompt chips | submit text through `/turns` | send selected text as user turn, not separate endpoint |
| message actions: copy | clipboard API | frontend-only | keep, accessible/error feedback |
| message actions: read aloud | Web Speech synthesis when available | no backend voice | optional browser-only feature with truthful UX; no model-generated speech promise |
| message actions: save/regenerate/report | local toast or fake replacement | no bookmark/regenerate/report endpoints | disable honest state; separately approve contract and storage |
| `insight` | fixed statement + helpful vote | no insight or feedback endpoint | preserve visuals, mark demo, no AI inference claims |
| menu: logout | local React reset | `DELETE /dev/session` | revoke server session, clear UI scoped state and outstanding requests |
| menu: preferences | navigates to approach | `PATCH /me/profile` | connected profile setting with explicit mapping |
| menu: support/emergency | link-copy and placeholder UI | no safety service | don't promise live dispatch or fabricated helpline |

## Primary blockers / decision log

Task08 update (2026-11-02): Home's reflection card now opens `day`, connected to owner-scoped `GET/PUT/DELETE /me/daily-reflections/{date}` and `GET /me/daily-reflections`. Explicitly authored entries persist in MSSQL; drafts stay in React memory. Save/edit/reload/relogin/user isolation/delete were verified through the real local API and database. The existing letter/Insight remains outside this feature; no AI-generated daily summary is implemented. See [Task08](TASK08_DAILY_REFLECTIONS.md).

1. **Live core integration blocker (frontend):** no fetch/API client/session guard/real Chat/History implementation, despite the polished UI.
2. **Privacy/product blocker:** if the entire long intake is required in the connected flow, it needs field-level necessity, consent, minimization and storage rules and possibly backend migrations. This is *not* a reason to redesign the screen.
3. **Capability choices:** voice, personalized insights/letters, assessments, message regeneration/bookmarks/feedback have UI design but no backend specification. New backend endpoints only after explicit prioritization and approval.
4. **Claim accuracy blocker:** review/statistics, research/affiliations, clinician credentials, phone-storage/delete-account assurances must be verified or replaced before public/real use.
5. **Data consistency blocker:** all current demo messages and `History` content are local hardcoded assets; connected mode must never show them as another tester's saved data.
