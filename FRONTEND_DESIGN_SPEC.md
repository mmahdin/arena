# Frontend design and interaction contract — همراه (current authority)

> **2026-11-12 brand amendment — Theia identity.** The character-led Theia
> illustration (and every goddess portrait) is retired. The active visual
> identity for the logo, app icon, wax seal, home emblem and all onboarding
> illustrations is the **Aperture of First Light** system documented in
> `docs/brand/IDENTITY.md`: an open ring, a horizon and a single point of warm
> light, with eight onboarding plates that open the aperture from Welcome to
> Ready. UI layouts, screen order, typography, type scale, copy and CSS
> geometry are unchanged; only the brand/illustration layer was replaced.
> Illustrations are inline SVG in `src/components/TheiaBrand.jsx`. Do not
> reintroduce character art or the old crescent/profile emblem.

> 2026-11-06 approach design: use the `ui-ux-pro-max` Minimalism & Swiss Style direction with low density. Show one heading, the current style name between two subtle, unboxed chevrons, one shared example prompt and one authored reply. The style name is plain text; swipe the example left/right, tap a chevron or use the arrow keys to change it. There is no dropdown. Back, Continue and quiet Skip/Cancel remain. The chevrons have 44px hit areas with no visible container; style lists, dropdowns, written gesture instructions, subtitles, counters, descriptions and footer notes are removed. Five small, noninteractive position dots sit below the style name, ordered to match the physical slide positions (LTR), with one elongated active mark synchronized to swipe, arrows and keyboard. Preserve the five original style keys, mapping, default, swipe/keyboard behavior, stable reply slot, reduced motion, beige palette and local typography. See `docs/design/APPROACH_REDESIGN_2026_11.md`.

> 2026-11-04 product update: the active Home no longer preserves the reflection card or diary-writing UI. The envelope opens Letters, and Chat is a separate continuous conversation experience. Older reflection-preservation notes below are historical; see `docs/integration/UX_REFINEMENT_2026_11.md` and `docs/integration/LETTERS_FRONTEND_BOUNDARY.md`.

## Authority / preservation
**This uploaded 2026-11-02 new `ai-therapist-frontend(1).zip` is the definitive visual product reference.** It replaces the earlier Rahrah design and the older TypeScript Wellbeing prototype as a visual source. No broad redesign is requested; the 2026-11-06 approach-screen update above is the scoped exception. Make each new functional state feel native to this exact UI; do not migrate from another app.

## Visual language observed in source
- Persian friendly, soft organic floral figures; deliberate contrast between light neutral onboarding and almost-black home/chat screens.
- Max-width **430px** centered mobile viewport; page transitions ~180ms in CSS; RTL throughout. Keep mobile portrait hierarchy intact when adding loader/errors.
- Fonts: Estedad for primary, Vazirmatn and Noto Naskh Arabic installed locally.
- Main CSS tokens: `--light:#f3f2ee`, `--ink:#23201b`, `--beige:#cec4b8`, `--dark:#12110d`, `--lavender:#d5d9f7`, `--yellow:#f6cf98`, `--green:#9eda9e`.
- Components `StatusBar`, `BackButton`, `PrimaryButton`, `ChoiceButton`, `PageHeader`, `PagerDots`, `VoiceWave` etc. are already implemented; reuse rather than replacing them.

## Screen behavior map (design preserved, functionality gated by current scope)
1. **Welcome:** continue to designed onboarding. Add a Dev Tester picker in the same visual vocabulary, before connected user data. No email/signup façade.
2. **Testimonials / Research / Experts:** preserve cards, typography and imagery; do not claim 6,000 reviews, 14 papers, affiliations, clinicians, publications or outcomes without verification. Prefer disabled gallery preview or truthful copy of identical length.
3. **Name:** save displayName when connected; avoid hard-coded default `محمدمهدی` before loading `/me`; validate required and backend length constraints.
4. **Privacy:** keep illustration and toggle geometry, but do not state chats live on phone or account deletion works. Sharing control must be disabled/off until an actual audited service and consent model exists.
5. **Intake questions:** routes `therapist`, `age`, `concerns`, `focus`, `feelings`, `neurodivergent`, `faith`, `interests`, `relationships`, `strengths`, `barriers` currently store choices in component state only; these are *sensitive* design-only fields. Do not use them as backend profile fields or LLM prompt. For core MVP choose a safe short path or explicit design-gallery mode; document owner decision before changing navigation.
6. **Voice and Approach:** preserve Voice swipeable cards and wave art; Voice remains visual only, **not actual TTS/model voice**. Approach uses the 2026-11-06 minimal chat preview: one shared prompt, five concise authored replies, RTL swipe/keyboard selection with a plain style name and subtle unboxed directional controls, and reduced-motion support. Approach names (`گفت‌وگوی آزاد`, `همدلی و شنیدن`, `به من یاد بده`, `راهنمایی آرام`, `حل مسئله با هم`) still map explicitly to the five persistent `conversationStyle` values in `preferences.js`; legacy `guidance` remains readable. Do not conflate voice with style.
7. **Home:** preserve greeting, animated envelope and reflection card. The letter/day reflections currently static; either clearly label demo or plan new domain endpoints after approval.
8. **History:** replace one hard-coded row and weekly score with `GET /conversations`, display stable list. Don't imply weekly progress insights from conversation counts alone.
9. **Chat:** preserve dark bubbles, paragraph spacing, suggestions, action menu and composer. Replace the `setTimeout(...850ms)` mock response with a single JSON `/turns` request in connected mode; use loading state, exact `clientTurnId` replay and guard duplicate send.
10. **Message actions:** copy is local; browser speech synthesis may read locally, but label honestly. Bookmark, regenerate and report currently fake; disable or add *approved* API work, never show a fake success toast.
11. **Insight / Survey / Day:** remain design-intent screens. The current insight text and health-questionnaire response are static. Before use with real connected testing require new product/safety/back-end contracts and real metrics provenance.
12. **Menu / Logout:** logout must revoke the backend Dev Session; clear all local tester-state and in-flight requests; never just reset visual fields.

## Responsive + accessible states
- Compare at width 360 / 390 / 430 and desktop; check safe area, 100dvh, keyboard open, dynamic scrolling and long mixed Persian/English text.
- For all connected actions support `loading`, `empty`, `pending`, `error`, `retry`, `disabled`, `permission` and delete confirmation, without changing page silhouettes.
- Announce errors in a live region; provide `aria-label` for icon buttons; focus-visible and reduced-motion. Keep screen order consistent under hash back/forward navigation.
- With 401 show the fictional mobile-shaped test identifier form; 404 show unavailable conversation; 409 onboarding/pending turn; 503 network/provider unavailable with retry. Never infer another tester's conversation.

## Visual QA acceptance
- Default routes and imagery match original at 390px and desktop, before/after API adapters.
- No original CSS classes/asset references deleted by integration unless explicitly approved.
- RTL keyboards and screen-reader labels work. Chat is not pretending to stream tokens.
- No active real-user screen has unsupported clinical, research or privacy guarantees.
- Every visible connected action either performs its advertised behavior or clearly says unavailable; demo-only art remains intact.
