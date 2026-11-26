# Earthshine — The Last Hours of the Night

**Scope:** the entire pre-Home onboarding and the sign-in entry that precedes it.
**Status:** implemented, 2026-11-26. This document records the creative and
technical decisions; the implementation lives in `src/features/onboarding/earthshine.js`,
`src/components/EarthshineScene.jsx`, `src/screens/Onboarding.jsx` and
`src/earthshine.css`.

---

## 1. The thesis

The whole story is one pre-dawn under one thin waning crescent. When the Moon
seems to give the least light, the Earth — nearly full as seen from the Moon —
returns the most. The unlit disc is therefore not empty: it is lit by the world
the user is standing on, and it becomes visible as the eye adapts.

Three natural phenomena carry the narrative:

- **Earthshine** is strongest on a thin crescent. The quiet thesis: the moment
  the sky seems to give least is the moment your own world gives most.
- **Glare hides the faint.** The bright crescent dazzles the eye; the ashen disc
  appears with patience. A loud feeling can hide quieter ones without erasing
  them.
- **Returned light is changed light.** Earthshine comes back cool blue-grey,
  carrying oceans and clouds. Reflection in conversation does the same.

The Moon's dark seas — Serenity, Tranquility, Crises, Clouds, Cold — remain a
hidden vocabulary in `MOOD_SEAS`, not a gamified dashboard. The Mood step pans
across a real lunar surface rather than drawing a decorative ring.

## 2. The cast and the rule of absence

| Presence | What it is | What it is not |
| --- | --- | --- |
| The Sun | the first rim at the horizon, given and asking nothing | a face or deity |
| The Earth | never shown; the user is standing on it | a globe in the frame |
| The Held Moon | a natural waning crescent holding a textured ashen disc | a crescent icon or geometric shape |
| The ashen light | what was already there, made visible | a glow effect |
| Theia | the seeing: patience, adaptation, the dark that lets faint things show | a body, ring, eye, logo or assistant portrait in the frame |

The old open-ring instrument remains in `TheiaSigil.jsx` for non-onboarding
product surfaces and as repository history. It is not rendered anywhere in the
onboarding story or sign-in entry. The assistant mark on the conversation-style
preview is now a hairline, not a sigil.

## 3. The visual world

Eight painted plates share one eastern horizon, one natural Moon, cool
near-black/ultramarine skies and fine astronomical-photography grain. They are
deliberately composed with enough shadowed lower area for Persian RTL copy.
Framing changes between scenes — sparse sky, a window aperture, stars, an
intimate lunar close-up, a false-dawn cone, the widest three-light frame, a
lantern foreground, and a civil-twilight echo of Welcome.

Warm light is rationed: cool white crescent, cool blue-grey ashen disc, one tiny
distant pinprick in Night, horizon warmth only from Dawn onward, one small
lantern in Check-ins, and first light in the final scene. No region-specific
architecture, traditional furnishings, cultural ornaments, people, text or
arbitrary glowing shapes appear in the artwork.

| Scene | Route | Moment | Ashen disc | Warm light |
| --- | --- | --- | --- | --- |
| ۱ · آغاز | `welcome` | deepest dark | barely perceptible | none |
| ۲ · حریم خصوصی | `privacy` | same sky through an aperture | glimpsed | none |
| ۳ · شب | `simple-intro` + name/age/identity/therapy | deep ultramarine, stars out | ~20% | one far light |
| ۴ · دیدن | `deeper-intro` + mood/concerns/focus/feelings | late night, Moon larger | ~50% | none |
| ۵ · سپیده | `know-intro` + surroundings questions | first horizon light, false dawn | ~70% | horizon only |
| ۶ · سه نور | `almost-intro` + approach/voice | just before sunrise | 100% | horizon + sun's rim |
| ۷ · یادآوری | `checkins` + referral | slightly later | readable | lantern |
| ۸ · آغاز مسیر | `ready` | first light | readable | warmth reaching foreground |

## 4. Copy

The eight storyboard titles and bodies are implemented verbatim in Persian in
`NARRATIVE`. The factual privacy sentence is kept intact on Privacy; the final
chapter keeps the same server-storage / AI-personalization meaning. One plain
line outside the poetic layer is added on Welcome:

> تیا یک همراه گفت‌وگوی مبتنی بر هوش مصنوعی است؛ درمانگر نیست و جایگزین کمک تخصصی محسوب نمی‌شود.

The Name step no longer refers to the retired ring. All other question copy,
options, limits, skip behaviour and keyboard/pointer interactions remain intact.

## 5. Implementation boundaries

Unchanged on purpose:

- `FLOW` order and all 26 onboarding routes; server-owned progress cursor.
- The full question set in `src/data.js` and the `QUESTIONS[route]` engine.
- Name persistence, age wheel physics, mood slider, voice carousel, approach
  swipe/keyboard behaviour, privacy toggle and its API store.
- Final `ready → home` profile completion with `displayName`,
  `conversationStyle` and allowlisted `onboardingAnswers`.
- Legacy `testimonials`, `research` and `experts` routes from the menu.

Changed:

- Every onboarding shell is now one ink surface (`screenSurfaceForRoute`).
- `EarthshineScene` renders the plate, a scene-specific scrim, fine grain and a
  slow 2.4s brightness adaptation — the only motion of Theia. Reduced-motion
  users see the final state immediately.
- `ChapterSeam` is eight hairline segments; the current segment widens with the
  scene's ashen-disc legibility. No dots, rings or glows.
- `MoodSeas` uses `held-moon.jpg` and the hidden sea names instead of the old
  spiral/ring scribble.
- `DevLogin` opens on the same Welcome plate, with no figure or instrument.

## 6. Assets

`public/brand/earthshine/`:

- `held-moon.jpg` — the natural lunar texture anchor and Mood-step surface.
- `01-welcome.jpg` … `08-first-light.jpg` — the eight scene plates.
- `01-welcome.jpg` is preloaded from `index.html` for the first paint.

All plates are small JPEGs (under ~100 KB each) and are committed as design
assets. The retired figure artwork in `public/brand/` remains in the repository
but is not referenced by the active UI.

## 7. Accessibility and verification

- Each backdrop is a single `role="img"` with a Persian description; the inner
  `<img>` is decorative and `alt=""`.
- Copy sits above a scene-specific dark scrim with text shadow; controls have a
  cool high-contrast focus ring.
- The adaptation animation and all transitions are disabled under
  `prefers-reduced-motion`.
- `npm test`, `npm run build`, and `npm run check:foundation` pass. The
  Earthshine regression test verifies verbatim copy, all eight scene mappings,
  AI transparency, no retired instrument language, and preserved
  questions/backend wiring.
- `npm run preview:theia` serves the full flow with the repository's synthetic
  API for visual review.
