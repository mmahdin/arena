# تیا — Onboarding redesign: Earthshine (زمین‌تاب), 2026-11

**Scope:** the entire pre-Home journey (26 routes in `FLOW`), plus the sign-in
entry that precedes it. Routing, server-owned progress, answer storage, and
every network boundary remain unchanged.

---

## 1. Core idea — *Earthshine: The Last Hours of the Night*

The story is told as **one pre-dawn, from one fixed place, under one moon**:
a quiet Iranian rooftop (`بام`) facing east, with a low mud-plaster parapet
(`جان‌پناه`), a woven Persian rug, a rooftop stairhead (`خرپشته`), and a thin
waning morning crescent rising in the last hours before sunrise.

Three real astronomical and psychological facts carry the story:

1. **Least sunlight, most Earthshine.** When the Moon is a thin crescent, it
   receives the least direct sunlight and the most light reflected from Earth.
   When the sky seems to give you the least, you are handed the most from your
   own lived world.
2. **Glare hides the faint.** The bright crescent dazzles the eye and makes the
   rest of the Moon look emptier than it is. As the eye adapts with patience,
   the quieter ashen sphere becomes readable.
3. **Earthshine returns tinted by what it touched.** It comes back cool
   ocean-blue-gray, revealing the Moon's dark maria — named in the 17th century
   for weather and states of mind (*Serenitatis, Tranquillitatis, Crisium,
   Nubium, Frigoris*).

## 2. The cast & cultural guardrails

- **Theia** is never drawn as a person, face, or geometric ornament. She is the
  *seeing*: the patience of dark adaptation and the ashen light itself.
- **The Held Moon (`ماهِ در آغوش`):** a tilted lower-right waning morning
  crescent cradling its spherical Earthshine disc in its arms.
  - Always a *waning* morning crescent rising in the east, never an evening new
    moon.
  - The spherical ashen disc is always present so it never reads as a flag or
    religious icon.
  - Never isolated beside a star.
- **No human figure** appears anywhere. Only quiet traces of presence: a rug, a
  folded wool blanket, wooden shutters, a small brass-and-glass lantern, and at
  first light, an open rooftop stairhead door pointing back toward ordinary life.

## 3. Continuity & progress map

The story's progress bar in `.ob-chrome` is the **legibility of the Held Moon's
ashen disc** (`TheiaSigil` `earthshine` prop from `0.08` to `1.00`) paired with
a six-chapter horizon thread. Warm light is strictly rationed: cool white and
cool blue-gray dominate the night chapters; soft horizon warmth enters only at
Dawn and Three Lights; localized amber appears only for the foreground lantern
on Check-ins and the single quiet morning ray on the Final screen.

| Scene | Route | Chapter | Moment | Ashen disc | Warm light |
| --- | --- | --- | --- | --- | --- |
| 1 · Welcome | `welcome` | آغاز | Deepest pre-dawn dark | barely perceptible (~8%) | none |
| 2 · Privacy | `privacy` | آغاز (حریم خصوصی) | Interior room through shutters | glimpsed (~14%) | none (warm-dark plaster room) |
| 3 · Night | `simple-intro` | شب | Ultramarine starlight, blanket on rug | ~20% | one far pinpoint on plain |
| 4 · Seeing | `deeper-intro` | دیدن | Intimate telephoto Moon & maria | ~50% | none |
| 5 · Dawn | `know-intro` | سپیده | Horizon band, false dawn (`صبح کاذب`), cypress | ~70% | horizon + 2 distant windows |
| 6 · Three Lights | `almost-intro` | سه نور | Sun's first rim, Dawn, Held Moon | 100% | horizon + sun's rim |
| 7 · Check-ins | `checkins` | سه نور (یادآوری) | Foreground parapet lantern | readable (~86%) | localized brass lantern |
| 8 · Final | `ready` | آغاز مسیر | Civil twilight, open stair door, ray on rug | readable (~80%) | first morning ray on rug |

## 4. Screen-level interactions & boundaries

- **Welcome (`welcome`):** states the mythic premise directly (*«تیا را نمی‌بینی؛
  با تیا می‌بینی.»*) and includes a plain, non-poetic AI transparency line:
  *«تیا یک همراه هوش مصنوعی است و جایگزین روان‌درمانگر نیست.»*
- **Privacy (`privacy`):** seen from inside a quiet plaster room through two
  wooden shutter leaves; toggling `اجازهٔ استفاده از گفت‌وگوها` subtly widens or
  narrows the shutter aperture while preserving the exact server-storage and
  AI-processing disclosure and `/api/v1/me/privacy-preferences` contract.
- **Mood (`mood`):** `EarthshineMoodTerrain` draws emotion as nocturnal
  lunar-sea contours and moonlit horizon lines across all five slider positions
  rather than a decorative icon.
- **Check-ins (`checkins`):** switching the lantern toggle off gently dims the
  flame to a calm ember (`07-checkins-dim.webp`) rather than extinguishing it,
  with explicit honest copy that push notifications are not yet active.
- **Intake, Voice, Approach & Ready:** `FLOW` order, ordinals, server cursor,
  ephemeral draft answers, allowlisted `onboardingAnswers` on final Ready, five
  conversation styles, and six voice cards remain completely intact.

## 5. Implementation map

- `src/components/EarthshineScene.jsx` — the eight Earthshine scene plates,
  preloading, interactive shutter aperture, interactive lantern cross-dissolve,
  and `EarthshineMoodTerrain`.
- `public/brand/earthshine/01-welcome.webp` … `08-final.webp` (+
  `07-checkins-dim.webp`) — cohesive nocturnal-to-civil-twilight rooftop plates.
- `src/components/TheiaSigil.jsx` & `src/components/TheiaBrand.jsx` — the Held
  Moon (`ماهِ در آغوش`) vector mark with variable Earthshine legibility.
- `scripts/generate-theia-icons.mjs` — dependency-free supersampled PNG icon
  generator for `theia-sigil-{192,512}.png` and `apple-touch-icon.png`.
- `src/screens/Onboarding.jsx` & `src/onboarding.css` — Earthshine chapters,
  storyboards, rationed color tokens, and responsive layouts.
- `src/features/devsession/DevLogin.jsx` — sign-in entry in the same pre-dawn
  rooftop world.
- `scripts/theia-brand.test.mjs` — automated checks for assets, copy, faceless
  guardrails, and retired portraits.
