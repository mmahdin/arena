# Figma Make Master Build Prompt — Premium Persian AI Psychology Companion (Frontend MVP)

## Role and outcome

Act as an exceptional senior product designer, interaction designer and frontend engineer. **Design AND build a fully navigable, high-fidelity mobile-first interactive frontend MVP** for an AI psychological companion and self-discovery application for Persian speakers.

The design must be extremely refined, beautiful, contemporary, minimalist and coherent. Use the attached 61 screenshots of The Path and the accompanying English design audit as your **visual reference**, never as the literal product specification.

**All the text displayed inside the application must be written in natural Persian (Farsi). All application layouts, navigation, form semantics and chat behavior must be authentically RTL.** This build prompt is English solely to ensure clear implementation instructions.

This is a **frontend-only** deliverable with mock data and a mock AI-response service. A real backend and live AI model will be connected later. Do not stop at wireframes, static renders, proposals or a design-only explanation. Create a navigable, interactive app prototype with functional demo flows.

## 1. What our MVP actually includes

Exactly five primary capabilities:

1. A gentle personalized onboarding flow that collects a limited initial user profile.
2. A communication-style picker showing distinctive AI response behaviors and **side-by-side or switchable examples to the same user message**.
3. Multiple independent text conversation threads, with **selective, contextual simulated memory** across conversations.
4. A self-understanding profile showing tentative observations, user preferences, recurring themes and user-editable memory items.
5. A library of self-discovery assessments, with at least one complete mock assessment and a mock results screen that can optionally feed the profile.

No actual licensed therapy, appointment booking, humans pretending to be AI therapists, fixed-duration sessions, camera-based emotion detection, automated diagnosis or sprawling marketing pages.

## 2. Design quality and visual inspiration

**Reference sources**
- `docs/THE_PATH_61_SCREEN_AUDIT_EN.md`: evidence-based notes on all 61 image files.
- `docs/REFERENCE_TO_MVP_MAPPING_EN.md`: which reference screens to consider for each **new** MVP screen.
- `screens/photo_61...photo_1...jpg`: original reference screenshots in provided descending sequence.
- `storyboards/`: compact overviews for rapidly finding relevant visual patterns.
- `data/reference_tokens_en.json`: approximate visual color and sizing guidance, not an official vendor export.

Capture the **design grammar**, not the trademarked visuals: quiet editorial headings, fine sans-serif copy, bold controlled whitespace, subtle atmospheric gradients, softly rounded cards, full-height immersive compositions, and clean fixed bottom actions. The Path is especially useful for the cadence of light onboarding pages, dark guide-selection gallery and chat, and curved atmospheric home headers.

### Original design direction

- Luxury-feeling, calm, emotionally intelligent, warm, modern, precise.
- Minimal, but **not** plain, hollow or generic.
- A distinctive brand feel generated through high-quality typography, organic abstract visuals and great composition rather than illustrations everywhere.
- Mostly warm off-white and charcoal neutrals; use accents sparingly and deliberately.
- Restrained soft shadows, subtle hairline borders, tasteful rounded corners, elegant hover/pressed/selected states.
- Emotional pause screens can have blurred lavender, pale coral, restrained olive, soft pink or blue-gray atmospheres. Do **not** place generic gradients on every card.
- One dominant action per screen. Supporting copy should usually be only one or two short lines.
- Use sophisticated, cohesive, abstract nonhuman style visuals. **Do not reuse The Path's people portraits, guide names, emblem, testimonials or marketing percentages.**

### Source-inspired palette (modifiable)

```text
Neutral canvas   #F2F2F2
Raised surface   #FFFFFF
Near-black ink   #1B1B1B
Secondary text   #747474
Soft border      #DADADA
Dark olive       #34431F
Muted olive      #9CA866
Soft lavender    #9198B6
Warm peach       #EF782A
Atmospheric pink #F6AEC5
```

These values were visually approximated for design inspiration. Use them as starting points and check contrast rather than assuming they are exact brand specs.

### Typography, Persian rendering, mobile rhythm

- Preferred Persian heading font: **Estedad**, weights around 500–600.
- Preferred Persian functional text font: **Vazirmatn**, weights around 400–600.
- Use properly licensed accessible substitutes only if those cannot be loaded.
- Persian display headings should have confident visual scale and comfortable line height. Do not add artificial character tracking to Arabic-script text.
- Typical title 28–34 CSS px, readable body 15–17 px, labels 12–14 px. Test accessibility scaling.
- Use a responsive 360 / 390 / 430 CSS-px mobile layout, designed primarily for 390px.
- Baseline gutters ~20–24px; spacing scale 4 / 8 / 12 / 16 / 20 / 24 / 32 / 40 / 48px.
- Option rows >=48px high, buttons approx 50–52px high, touch targets >=44x44px.
- Respect notches/safe area and fixed composer/button regions. Provide sufficient bottom scroll padding.

Use a native RTL foundation: `dir="rtl"`, right-aligned content, correct arrow direction, right-origin navigation, well-handled mixed-script fields, RTL text selection and responsive Persian multiline wrapping. The composer must behave properly when typing Persian or a Latin product name in the same message.

## 3. App navigation and screen map

Set a coherent four-tab mobile bottom navigation:

- **خانه** — Home
- **گفت‌وگوها** — Conversations
- **شناخت من** — Self-understanding
- **آزمون‌ها** — Assessments

Keep secondary settings and privacy/memory controls in a simple contextual profile/settings entry, not an additional main tab. Navigation should be consistent and styled beautifully with a meaningful selected state.

Build the following screens / states:

1. Brand welcome / entry.
2. Short onboarding introduction.
3. Onboarding questions: name, goal, response preference; include examples of a single-choice screen, optional free text and skip behavior.
4. Optional reflective pause between onboarding and app (short and non-diagnostic).
5. AI communication-style catalog (five options).
6. Same-prompt style comparison preview.
7. Selected style detail / confirmation (only if it improves the selection experience).
8. Home: empty and populated states.
9. Conversation list: empty and populated states.
10. Active text chat and starting a new chat; thinking, error and ongoing message states.
11. Self-understanding profile with tentative insights.
12. Memory items list, editable/correctable/removable controls, and sensitive-use explanation.
13. Assessment library with not-started/in-progress/completed states.
14. One complete sample assessment with question progress.
15. Assessment results, with optional save-to-profile toggle/action.
16. Basic settings/privacy surface, and appropriate dialogs/confirmation states.

This is a **product prototype**, not 61 separate copies of the reference screens. Feature completeness and premium interaction design are more important than screen count.

## 4. Onboarding — friendly, low-friction, adaptable

The true onboarding questionnaire is not finalized. We will research the appropriate psychological content and adaptive branching separately. **Do not claim the mock questionnaire is a clinically validated instrument.**

Create an elegant reusable engine for 4–6 sample onboarding steps, demonstrating:

- A short preferred-name text field (`دوست داری چی صدات کنیم؟`).
- One single-choice question about the user's main purpose (`این روزها بیشتر دوست داری روی چی تمرکز کنی؟`).
- A preference question about how they like to receive support (`چه مدل گفت‌وگویی بیشتر بهت کمک می‌کنه؟`).
- Optional free-text context (`اگه دوست داری، یکم بیشتر برام بگو.`).
- A short transition/reflective interlude using cautious language.

Keep the flow modular and data-driven: each question has stable ID, label, format, options, optional flag, and any conditional visibility metadata. A stepper should be subtle. Preserve entered answers across Previous / Next. Show a discreet Skip for optional/sensitive items. Selected options should be unambiguous, with border, radio and contrast (not color only). Do not force lengthy disclosure just to enter the app.

**Must demonstrate keyboard-open behavior** with CTA visible, not covered by on-screen keyboard. Handle long Persian choices. Avoid hardcoding unvalidated clinical questions from reference screenshots.

## 5. AI communication-style catalog — signature experience

After onboarding, present **five behavior-based communication styles**. They are NOT separate human therapists and should not be represented by someone else's real photos. Use compelling original abstract art / atmospheric graphic identities with visual differentiation and consistent component structure.

| Working UI name | Communication behavior |
|---|---|
| **همدل** | Acknowledges feelings, supports without empty flattery, then offers helpful reflection. |
| **تحلیل‌گر** | Breaks the situation into structured alternatives and cause/effect reasoning. |
| **صریح** | Responds quickly and directly with clear, actionable points. |
| **شنونده** | Listens, reflects carefully, asks thoughtful open questions instead of rushing to solutions. |
| **مثال‌محور** | Makes abstract ideas understandable through concrete everyday examples and comparisons. |

Use short meaningful descriptions, attractive selection cards, and an easily understood selection state. Do not invent exaggerated clinical qualifications for the communication styles.

### Interactive example comparison (must actually work)

Have the user choose a sample scenario, or default to:

`این روزها برای تصمیم‌گیری خیلی گیج می‌شم.`

Provide a tab, swipe, or compact compare interface. When switching styles, display a **different, coherent, genuinely representative** response to the **same** prompt. This is essential for the user to discover which voice they prefer. Avoid responses that differ only in a greeting or adjective. Implement all five previews from mock data. Allow committing to the chosen style (`این سبک رو انتخاب می‌کنم`).

Style can be changed later, including during a chat. Visually distinguish selected communication style from the content subject of a conversation.

## 6. Conversation experience — multiple threads, continuous chat

Create a clean, expressive, text-first conversation experience.

- List distinct chat threads with title, one-line preview, optional last-updated metadata and a clear New Chat button.
- New chats begin as clean independent threads. Users can switch, return and continue chatting without losing previously entered messages.
- Chat bubbles must be easy to distinguish and appropriately placed for RTL.
- The message composer remains accessible above the keyboard; support multiline Persian input, send, disabled-send and errors.
- Include assistant thinking animation or low-key `در حال فکر کردن…` status and simulated delay, without fake tool calls or pretending to reach a model.
- Mock responses should take into account current selected communication style and the conversation's general topic; they are scripted/local, not generated by a live AI.
- Include empty state, active state, loading, network-style mock error (with Retry), and a simple conversation options menu.

**Do not add a session countdown, 5–30 minute picker, forced session endings, paid guide locks, audio calls or video observation.** Those are present in the reference, not in our MVP.

### Contextual cross-chat memory — visibly simulated, responsibly controlled

The future backend will selectively use relevant shared context across independent threads. Demonstrate that concept in local mock data, not as uncontrolled context leakage.

- Keep mock memory items separate from raw chat history. A memory item should have an ID, short description, scope/source, editable value, and a toggle or delete action.
- When showing a relevant example, a later career-decision chat may briefly reference an earlier work-stress observation; another unrelated thread should NOT do so.
- Include a subtle, optional indication when a reply was influenced by remembered context, with a way to inspect the source memory.
- Users must be able to correct/remove an item and see its effect on mock subsequent replies.
- No real retrieval model, embedding store, secret collection or backend persistence is required in this phase.
- If using localStorage, call it **demo-only local storage**, never production secure memory.

## 7. Self-understanding profile — `شناخت من`

This is a reflective personal profile, not a social-media profile or medical diagnosis dashboard.

Populate an elegant layout with mock data derived from onboarding and simulated conversations:

- Current self-reported goals.
- Chosen communication style.
- Recurring themes/topics.
- Tentative observations (phrased with `به نظر می‌رسه…`, not overconfident labels).
- Recently completed assessments / helpful insights.
- Memory controls: view, edit, correct, delete.

Examples must be compassionate and specific. Do not show arbitrary percent scores, personality badges, mental-health diagnoses or fabricated behavioral analytics. Where useful, show discreet source labels such as `از گفت‌وگوها` / `از پاسخ‌های اولیه` / `از آزمون‌ها`. Users can mark an observation as inaccurate (`این برداشت دقیق نیست`).

Design empty, populated and disagreement/correction states. Carefully demonstrate that these observations are provisional and user-controlled.

## 8. Self-discovery assessments — `آزمون‌ها`

The assessment library must support browse / start / resume / complete / view results. For this MVP, **all questions and results are illustrative mock content, not clinical instruments.** The screenshots include some items resembling common standardized questionnaires; do not copy them as a validated diagnostic implementation and do not invent scoring validity.

Create 3–4 elegant sample assessment cards, for instance:

- `سبک گفت‌وگو` (communication preferences)
- `روش تصمیم‌گیری` (decision habits)
- `آگاهی هیجانی` (emotional self-awareness)
- `عادت‌های مدیریت استرس` (stress coping patterns)

Include a plausible approximate duration as **demo data**, not a real validation claim.

Build one **complete** short example (e.g. 4–6 questions) with step indicator, answer options, Back / Next, completion and outcome. The results should express **descriptive, tentative non-diagnostic observations** instead of clinical risk levels. Provide an explicit optional action to add an assessment insight to the user's personal profile and memory; reflect that saved item on the profile screen. Implement in-progress/completed statuses.

## 9. Functional mock-data requirement and architecture

The whole prototype must be runnable/navigable with coherent data. Use `data/mock_app_data_fa.json` as a starter content fixture if available. You may adjust content for natural Persian, but keep the same feature coverage.

If Figma Make is generating code, prefer React + TypeScript where available. Use clean component boundaries, theme tokens, consistent models and a replaceable mock service / repository abstraction rather than embedding data directly in JSX.

Suggested types: `User`, `OnboardingQuestion`, `OnboardingAnswer`, `CommunicationStyle`, `Conversation`, `Message`, `MemoryItem`, `PersonalInsight`, `Assessment`, `AssessmentQuestion`, `AssessmentAttempt`, `AssessmentResult`.

Use local state to support the user journey and optional localStorage for demo continuity. No database, API keys, authentication provider, deployment, external embeddings or real model integration is necessary. Keep mock operations self-contained and label them as demo behavior in code or documentation.

**Core interactions that must all work:** start onboarding; answer/skip/back; choose a style; compare all five responses to one prompt; change the chosen style; create a chat; send a message and get a mock reply; switch between chats; observe one deliberately relevant memory reference; review/edit/delete memory; browse assessments; complete one assessment; save result to self-profile; navigate four tabs; use settings; exit dialogs.

Do not add decorative buttons that do nothing.

## 10. Component inventory and interaction states

Design a proper reusable design system (components, variants and coherent typography/spacing):

- App shell, RTL top bar, bottom tab navigation, back/close actions.
- Standard CTA/secondary/tertiary buttons and pills, including disabled/loading.
- Onboarding stepper; single-select, multi-select and free-text question blocks.
- Keyboard-aware text/number entry; optionally helper/validation states.
- Atmospheric full-screen interlude; short editorial reflection.
- Abstract style card; style-detail; behavior traits; response-example preview / comparison.
- Chat list item, chat bubbles, composer, reply loading and failed/retry state.
- Personal insight card, origin label, helpful/not-accurate controls, memory editor.
- Assessment list card, question choices, progress and results card.
- Modal/bottom sheet, toast, privacy confirmation, empty state.

Define tap/pressed, selected, focused, disabled, loading, empty, error, long-Persian-text and keyboard-visible states. Use appropriate minimum contrast and reduced-motion support.

## 11. Trust, clarity, and safety

- The product is an AI companion/self-reflection tool, **not** a licensed human therapist, diagnosis service or replacement for emergency treatment.
- Never claim a real private encrypted backend already exists; this is a local demo.
- Sensitive inputs must be optional and only needed as appropriate. Make remembered context visible and adjustable.
- Avoid extreme claims about reading emotions, knowing intentions, measuring personality with certainty, or guaranteed outcomes.
- In an emergency-related demo state, provide a visible general path to immediate/local human help. Do not blindly present a US-only crisis number to Persian-language users worldwide.
- No facial recognition/camera-emotion inference, no user photo surveillance and no mandatory voice permissions.

## 12. Acceptance gate — finish only when all pass

1. All screens and simulated system copy in the product are in fluent Persian, with true RTL behavior.
2. Premium design quality: thoughtful grid, hierarchy, negative space, organic but restrained accents, no generic dashboard clutter.
3. The complete main user journey works as an interactive frontend prototype.
4. The five communication styles feel visually distinctive **and actually respond differently to the same mock message**.
5. Multiple chats remain separate; relevant mock cross-chat memory is demonstrated only when justified, and controls work.
6. Self-understanding profile is populated with tentative insights, edit/reject actions, and clear source cues.
7. One mock assessment is fully completable and produces a simulated result that optionally updates the profile.
8. 360 / 390 / 430px devices are usable. Fixed bottom UI does not obscure questions, chat composer or the keyboard.
9. All core controls are functional; loading/error/empty/selected states exist.
10. Components and mock services can later be replaced or wired to backend APIs without redesigning every screen.
11. The Path is inspiration only: no copied brand assets, human portraits, original guides, testimonials, metrics or timed-session flow.

**Build the frontend now. If external screenshots or docs are not actually accessible in your execution context, explicitly state which references you could not inspect; still implement the full MVP based on the written constraints.**
