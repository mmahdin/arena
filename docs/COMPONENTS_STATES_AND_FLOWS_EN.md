# Original MVP Component & Interaction Blueprint

**Objective:** frontend-only, responsive, Persian RTL, mock-data-driven prototype; future backend integration should not require redesigning all screens. The labels here are developer/AI-design instructions in English; the displayed application is Persian.

## Foundations

- **Mobile breakpoints:** 360, 390, 430 CSS pixels; ensure usable vertical layouts at short viewports. Respect safe areas and on-screen keyboard.
- **Text system:** Estedad display and Vazirmatn UI are proposed. Enable font fallback without clipped glyphs or fake tracking. Test RTL mixed numerals/English words.
- **Colors:** `data/reference_tokens_en.json` contains source-inspired approximate values; invent a coherent original brand treatment with neutral canvas, charcoal ink, restrained olive and occasional atmospheric color.
- **Spacing:** suggested 4/8/12/16/20/24/32/40/48/64px. Reserve visual quiet space rather than filling every gap.
- **Icons:** few, consistent outlined symbols. In RTL, reposition navigation arrows and choice indicators intentionally, not via arbitrary `scaleX(-1)` on all icons.
- **Touch and accessibility:** actionable controls at least 44px; target readable contrast, noncolor selection indicators, accessible names, reduced motion, and large Persian text.

## Component map

| Component name | Variants / states | Used on | Design reference |
|---|---|---|---|
| `AppShell` | onboarding, signed-in, dark immersive | all | 56, 22, 4 |
| `HeaderBar` | back, close, title, greeting, optional menu | onboarding/chat/settings | 56, 54, 9, 4 |
| `BottomTabBar` | 4 items; inactive, active, tapped | signed-in app | NEW design; source 1 for secondary navigation only |
| `PrimaryCTA` | dark on light, white on dark, loading, disabled | all flows | 57, 52, 22, 9 |
| `OutlineButton` | secondary, disabled, pressed | skip/secondary | 54, 42, 8 |
| `OnboardingStepper` | active, completed, skipped | onboarding | 52, 28 |
| `QuestionShell` | single question, optional, scrollable | onboarding/tests | 52, 41 |
| `RadioChoiceCard` | idle, selected, disabled, long-label, focus | onboarding/tests | 45, 52, 41 |
| `MultiSelectChip` | idle, selected, pressed | onboarding/tags | NEW, inspired by simple source pills |
| `FreeResponseField` | empty, focused, filled, error, keyboard-open | onboarding | 47, 44, 42 |
| `AmbientInterlude` | lavender, coral, blue-gray, olive | welcome/reflection | 57, 36, 30 |
| `ReflectionNote` | neutral, tentative, user-disputed | onboarding/profile | 46, 43, 39 |
| `StyleCard` | 5 styles, active, inactive, pressed | style catalog | 21–10, abstract redraw only |
| `StyleResponsePreview` | style A–E, compact, expanded | style compare | NEW feature, borrowed quality bar from 9 |
| `ChatPreviewRow` | unread(optional), recent, archived(optional) | chat list/home | 3 (adapted) |
| `ChatBubble` | user, assistant, long multiline, sending, failed | chat | 4 |
| `MessageComposer` | idle, typed, multiline, disabled, sending, keyboard-open | chat | 5, 4 without mandatory mic |
| `TypingState` | thinking, retry/error | chat | 4 |
| `ContextMemoryIndicator` | hidden, relevant memory used, inspect | chat | NEW privacy-minded feature |
| `PersonalInsightCard` | tentative, accepted, disputed, editable | profile | 46, 2 as hierarchy only |
| `MemoryItemRow` | active, editing, deleted, source detail | profile/settings | NEW |
| `AssessmentCard` | new, in-progress, completed | assessment library | 3 as card hierarchy only |
| `AssessmentQuestion` | unanswered, selected, summary, back | assessment | 28, 27, 24, 25 |
| `AssessmentResult` | mock description, save/unsave | assessment/profile | 2 and 46 as indirect references |
| `ConfirmationSheet` | delete memory, discard, confirm | memory/settings | NEW |
| `StatePlaceholder` | empty, loading, error, retry | all | 22 empty state, 2 processing idea |

## User flow 1 — onboarding and start

```text
Welcome
  → First-step explanation
  → Name (optional)
  → Primary goal (single choice; optional)
  → Support preference (single or multi-choice; optional)
  → Optional free text
  → Short reflective pause
  → Choose communication style
  → Preview all styles on same input
  → Confirm style
  → Home / start first chat
```

Show real progress; preserve answers on Back. Do not demand optional sensitive input. The exact question count and clinical content may change. Never label a fabricated questionnaire as a validated psychological assessment.

## User flow 2 — switchable, independent chat threads

```text
Home or Conversations
  → New Chat / Select Existing Chat
  → Chat view
  → Compose Persian text
  → Local mocked assistant 'thinking'
  → Mock assistant message in selected style
  → Return to Conversations (thread preserved)
  → Another thread (same user, separate message history)
```

The style used for current replies is explicit and can be changed. The thread list is a first-class feature, not a collection of artificial session report cards.

## User flow 3 — simulate selective shared memory

```text
Conversation A: user discusses work stress
  → (Demo-only) memory item created or selected
  → Memory manager shows source and editable text
Conversation B: user asks about a career decision
  → Local mock service chooses a relevant memory item
  → Reply may transparently refer to earlier work stress
  → User can inspect, correct, or remove the item
Conversation C: user asks unrelated question
  → No memory reference
```

**Do not automatically paste all prior messages into every thread.** Source/accuracy controls and user expectations matter. For a frontend demo, this can be deterministic, but must not claim genuine embedding-based retrieval or real storage security.

## User flow 4 — self-understanding

```text
Profile
  → See goals, chosen style, topics, tentative reflections
  → Tap an insight to see its source
  → Accept / Mark inaccurate / Edit
  → Memory section
  → Edit or delete stored mock memory
  → Updates reflected in profile and in later mock replies
```

Prefer nuanced natural-language insights. No invented diagnosis or numerical personality scores.

## User flow 5 — self-discovery assessments

```text
Assessment Library
  → Choose illustrative self-discovery assessment
  → Intro / purpose
  → Question 1 ... Question N with Back/Next
  → Completion
  → Descriptive mock result, clearly non-diagnostic
  → Save result into self-understanding profile (optional)
  → Profile updated
```

Include in-progress and completed library states. Use a source-aware result row if saved into user memory.

## Required empty / populated / interruption states

| Area | Empty | Populated / success | Interrupted / alternative |
|---|---|---|---|
| Home | no conversations, new-chat CTA | recent threads and greeting | loading/retry optional |
| Onboarding | unanswered question | selected answer | long Persian text, keyboard, skip, back |
| Style picker | no choice yet | one of five selected | switch to different style, compare previews |
| Chat | no messages | ongoing thread | thinking, failed send, retry, multiline input |
| Memory | no saved personal memory | source-labeled memory rows | correction, deletion, confirm, no contextual use |
| Profile | few insights | several tentative insights | disputed/edited insight |
| Assessments | no completed tests | completed result | in-progress, back, non-diagnostic disclaimer |
| Settings | default preferences | changed preferences | confirm/reset demo data |

## Mock services / future backend boundaries

Useful frontend interfaces (pseudocode names, not actual backend endpoints):

- `getCurrentUser()`
- `getOnboardingQuestions()` / `saveOnboardingAnswers()`
- `listCommunicationStyles()` / `setCommunicationStyle()`
- `listConversations()` / `createConversation()` / `getMessages()` / `sendMessageMock()`
- `listMemoryItems()` / `updateMemoryItem()` / `deleteMemoryItem()`
- `getPersonalInsights()` / `markInsightInaccurate()`
- `listAssessments()` / `saveAssessmentProgress()` / `completeAssessmentMock()`

All these functions should operate on frontend mock data for now. Do not invent live authentication or secure backend storage.

## UI content discipline

- Keep navigation and labels short, natural and approachable; no legal wall of text in normal interaction.
- Use first person sparingly for AI and use tentative wording when describing the user.
- Avoid automatically praising, endorsing harmful interpretations, or implying insight certainty.
- On dark/chat screens, typography and dark-on-dark controls must meet accessibility expectations.
- Do not let a fixed bottom CTA overlap the long guide description, options list, assessment or keyboard.

## Hand-off acceptance checklist

- [ ] Every main button works and routes to a meaningful state.
- [ ] All UI copy is Persian; debug text and English design instructions aren't visible in the app.
- [ ] Five style choices have visibly different preview responses for **one identical user prompt**.
- [ ] Multiple conversation threads persist within the demo and remain distinct.
- [ ] Memory behavior is selective, simulated, and user-controllable.
- [ ] Personal insights have tentative language and edit/disagree mechanisms.
- [ ] One assessment works from the first question through results and optionally the profile.
- [ ] All central components support 360, 390 and 430px widths, scroll and RTL keyboard states.
- [ ] No original people photos, original logomark, exact source guide names, unsourced percentages or timed-session gating have been adopted.
