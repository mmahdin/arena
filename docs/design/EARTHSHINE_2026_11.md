# Earthshine — the 2026-11 onboarding art direction

**Scope:** the eight narrated scenes of the pre-Home journey (all 26 routes in
`FLOW`), plus the sign-in entry. Questions, answer handling, server-owned
progress, consent and every network boundary are unchanged. This document
records the story, the visual system and how it is built; the earlier
ring-and-aperture identity is superseded for onboarding (the small brand mark in
Home, Letters and menus is untouched).

---

## 1. The idea

Theia is not only a goddess of light. What she offers is the power of light to
**reveal** what a thing already is. So the product principle is **revelation,
not illumination**: Theia adds nothing to a person and takes nothing away. She
helps what was always there become visible.

Earthshine is that idea in the sky. Sunlight reflects off the Earth, travels to
the Moon, and gently lights the part of the Moon the Sun does not reach. The
Moon has not changed. What was already there has simply become visible through
reflection.

Theia herself is never drawn. No figure, no mascot, no portrait, no rings. Her
presence is only ever what becomes visible.

## 2. One subject, one variable, one light

- **One subject:** the Moon, in the same place, at the same size, for the whole
  journey.
- **One variable:** how readable the unlit, earthlit part of the Moon is. That is
  the only thing that progresses.
- **One warm light:** everything is cool until the end. Warmth appears at the
  horizon in scene 5, as the sun's first rim in scene 6, as Venus in scene 7,
  and as a thin warm line in scene 8. Nothing else is warm.

There is no room, rug, architecture or ornament anywhere: a single sky belongs
to no culture and no era.

## 3. The visual system

**Frame.** One vertical composition. The Moon sits upper-centre with a generous
field of tone around it. From scene 5 one thin horizon line enters; the sea is
implied only by a slightly darker tone below it. Water is the oldest mirror and
is never described.

**Rendering.** Matte tonal gradients with fine film grain. The Moon is
hand-authored SVG: a real sphere with limb darkening, seven soft maria, faint
southern highlands and two young craters, and a blurred mask that gives the
terminator a soft edge instead of a vector line. No outlines, no glow filters, no
lens flares, no sparkles. The only effect allowed is a faint halo, and it sits
**outside** the Moon's silhouette so it never shines through the earthlit disc:
the halo is the story's idea of glare, light that arrives from somewhere else.

**Palette.** Sky from ink-navy to pale gray-blue; a cool-white crescent; a
blue-gray, slightly desaturated ashen disc; and one muted amber that only the
last scenes are allowed to use. In the pale scenes the unlit disc reads as a
quiet silhouette a shade below the sky, so the crescent stays the brightest
thing in the frame — as it does in the sky itself.

**Motion.** Slow cross-fades and breathing luminance only. Nothing travels,
nothing twinkles. `prefers-reduced-motion` removes the fade, the breathing and
the star fade-in.

**Guardrails.** The crescent is a waning morning crescent, lit toward the lower
left (the sunrise side), and the ashen disc is always present: without it a lone
crescent reads as a symbol, with it the image reads as the Moon. No
crescent-and-star emblem, no rays, no ornament. The rising readability is an
artistic narrative device, not an astronomical simulation.

## 4. The eight scenes

| # | Route(s) | Sky | Ashen disc | Added element | Warm light |
|---|----------|-----|-----------|---------------|------------|
| 1 | `welcome` | black | barely there | none | none |
| 2 | `privacy` | black, behind an opening | glimpsed | aperture | none |
| 3 | `simple-intro`, `name`, `age`, `identity`, `therapist` | ultramarine | ~20% | 7 stars | none |
| 4 | `deeper-intro`, `mood`, `concerns`, `focus`, `feelings` | deep blue | ~50% | none | none |
| 5 | `know-intro`, `faith`, `neurodivergent`, `relationships`, `interests`, `strengths`, `barriers`, `recharge` | blue to rose-gray | ~70% | horizon, false dawn | horizon only |
| 6 | `almost-intro`, `approach`, `voice` | pale blue | 100% | sun's first rim | horizon and sun |
| 7 | `checkins`, `referral` | pale blue | 100% | Venus | Venus |
| 8 | `ready` | pale twilight | 100% | none | thin warm line |

Copy per scene lives in `src/features/onboarding/earthshine.js`; the question
copy stays in `src/data.js` and is untouched.

**Title and body (as shipped).** Scene 1 «تیا را نمی‌بینی؛ با تیا می‌بینی.» ·
Scene 2 «چه‌قدر بگویی، انتخاب توست.» · Scene 3 «از تاریکی شروع می‌کنیم.» ·
Scene 4 «آن‌چه تاریک به نظر می‌رسد، خالی نیست.» · Scene 5 «سپیده یک‌باره
نمی‌آید.» · Scene 6 «سه نور؛ و هیچ‌کدام همیشگی نیست.» · Scene 7 «اگر بخواهی، یک
نور روشن می‌ماند.» · Scene 8 «از این‌جا شروع می‌کنیم.»

## 5. Honesty on the privacy screen

The window is an **aperture, not a vault**: a flat dark plane over the sky with
one vertical opening whose width belongs to the user. It is deliberately
symbolic, and the screen says so in plain words under the slider («این پنجره فقط
نماد است؛ تنظیم واقعیِ اشتراک‌گذاری، کلید پایین است.»).

The one real control on the screen is still the existing sharing switch, still
wired to `privacyPreferencesApi` through `onTogglePrivacy` / `onRetryPrivacy`,
still `role="switch"`, still disabled until the preference has loaded. The
symbol never writes a preference; the switch never pretends to control imagery.

**AI transparency.** Welcome carries one plain sentence, set apart from the
poetry by a hairline and in body type: «تیا با هوش مصنوعی کار می‌کند؛ همراهی
برای گفت‌وگو، نه جایگزین درمانگر حرفه‌ای.»

## 6. Implementation map

- `src/features/onboarding/earthshine.js` — the eight scenes: routes, copy, tone
  and art direction values, plus the AI-transparency and aperture notes. Pure
  data; no behaviour, no network.
- `src/components/TheiaSky.jsx` — `TheiaSky` paints one scene (sky field, stars,
  Moon, false dawn, sun rim, Venus, horizon, grain veil); `TheiaMoon` renders the
  Moon alone for the sign-in entry. All SVG, all hand-authored.
- `src/screens/Onboarding.jsx` — new `Shell` (status inset, chrome, sky band,
  flow, footer), eight-scene seam, scene copy, the privacy window and slider.
  Age wheel, mood slider, voice carousel, approach preview and every question
  engine are the preserved implementations, restyled only.
- `src/onboarding.css` — the Earthshine sheet. It owns `--ob-sky-zone`,
  `--ob-sky-paint` and `--ob-sea` per scene, the night/day tone tokens that every
  control reads, and the sky/apparatus rules. `styles.css` remains the source of
  truth for Home, Chat, Letters and menus.
- `src/features/appearance/screenSurface.js` — routes map to `ink` → `night` →
  `dawn` → `pale`, so the surface behind a transition matches the sky's top tone.
- `src/screens/../devsession/DevLogin.jsx` — the sign-in entry opens on scene 1.
- Retired: the ring/aperture instrument (`TheiaInstrument`) and the hand-drawn
  mood scribble. Both were second subjects; the sky is now the only artwork.

## 7. Verification

- `npm test` — 122 tests, including `scripts/earthshine.test.mjs`: the eight
  scenes cover the 26-step flow exactly once and in order; earthshine never
  decreases and only reaches full at scene 6; no horizon, sun or Venus appears
  early; the last scene returns to the first composition; the aperture note is
  symbolic and never drives the preference; AI transparency is plain and
  separate.
- `npm run check:foundation` and `npm run build` pass. (`check-foundation` reads
  `.agents/skills`, which is not part of this checkout, so it stops on the
  skills directory in this environment; that is a repository gap, not this
  change.)
- Every one of the 26 onboarding routes was server-rendered with stub props to
  catch runtime errors, and each scene was rasterised from the real component to
  check the crescent, the terminator, the ashen disc, the maria, the halo and the
  day palette.

**Known limits.** No browser is available in this environment, so final visual
QA at 360 / 390 / 430px, keyboard order and reduced motion still need a manual
pass in `npm run preview:theia`. Scene 7's copy promises that reminders can be
turned off; that setting does not exist yet, so the screen deliberately shows no
fake toggle. The legacy `testimonials/research/experts` pages keep their earlier
unverified claims and broken portraits and remain outside the flow.
