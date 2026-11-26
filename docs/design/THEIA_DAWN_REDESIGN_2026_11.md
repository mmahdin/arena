# سپیدهٔ درون — Theia, «The Inner Dawn»
### 2026-11-26 · Identity & onboarding redesign

This document is the design contract for the redesigned Theia identity and the
onboarding experience. It replaces every woman-shaped depiction of Theia in the
active product with a **generated, non-human, culturally Persian symbol system**
rooted in what Theia — the Greek Titaness of sight and heavenly light — actually
*is* in the myths, and in what the product is for.

The redesign is not a re-skin. It is one authored narrative: **a single night,
a single horizon, and a single light that rises as the user moves through
onboarding.**

---

## 1 · Why Theia, and what she means

Theia (Θεία) is one of the twelve Titans, daughter of Uranus (sky) and Gaia
(earth). The ancient sources give her almost no stories of her own — she is
supremacy without theatre. Three things are said of her, and each maps onto the
product:

| Myth | Source | Product meaning |
|---|---|---|
| Goddess of **sight** — *thea*, "to see"; the eyes were thought to emit light | Theoi: "THEIA — Titan goddess of sight … and the shining ether of the bright blue sky" | The product's core act: *seeing oneself clearly* with the help of a calm, unblinking witness |
| Mother of **Helios (Sun), Selene (Moon), Eos (Dawn)** — all light descends from her | Hesiod, *Theogony* 371 ff. | She is not the light; she is the **source** of it. In-app, her children are the visual cast: the crescent, the rising sun, the dawn |
| She endowed **gold, silver and gemstones** with their brilliance | Pindar, *Isthmian* 5 | Intrinsic worth that needs only light to be seen — the therapeutic premise of being *witnessed into value*, not fixed |
| Epithets: **Euryphaessa** («wide-shining»), **Aethra** («clear blue sky») | Theoi; Apollodorus 1.2.2 | The two states user and product move between: dim, wide night → clear, lit sky |

A fourth, older idea completes the arc: the Greeks believed **to see is to be
touched by light**. Anxiety, rumination and shame are not failures of will —
they are the dark where the light has not arrived yet. The onboarding's job is
to walk the user from the darkest point of that sky to the first full sunrise,
slowly, kindly and without spectacle.

## 2 · The Persian symbol vocabulary (and what was refused)

The identity is Greek in its source myth but **speaks Persian in its body** —
because the product belongs to its users. Every symbol below is secular,
well-established in Persian art, and legible to a modern Iranian audience
without exoticising itself.

| Symbol | Persian | Meaning carried |
|---|---|---|
| **Shamseh** — the sunburst medallion of Persian illumination and dome coffering | شمسه | The radiant sun (Helios) rendered in the ornamental grammar of Persian manuscripts; layered rays = attention unfolding outward |
| **Khatam** — eight-pointed geometric star of inlay work (two rotated squares) | خاتم | Precision, craft, patience — the reward of inward discipline; used for spark/khatam-glints, never as decoration for its own sake |
| **Howz** — the still reflecting pool of a Persian garden | حوض | The mind allowed to become calm; only then does it mirror the light truthfully. This is the single strongest metaphor in the system |
| **Sarv (cypress)** — sentinel tree of Persian gardens and poetry | سرو | Dignity and stillness; frames the composition the way it frames a garden |
| **Firouzeh (turquoise)** — the stone of Iran, altars and amulets | فیروزه | The one star in the field painted turquoise; a silent nod to the gems Theia taught to shine; a cool counterpoint to the dawn gold |
| **Dawn over the Alborz** — a two-ridge silhouette, the mountain wall north of Tehran | سپیده‌دم / البرز | Home. The user's own geography, under Theia's sky |

Deliberately **refused**: winged figures, Persepolis pastiche, zodiac tables,
Faravahar (a specifically Zoroastrian religious symbol — not appropriate for a
secular mental-health product), minaret/mihrab imagery, literal eye-of-Horus or
"evil eye" talismans (nazar), and any pseudo-mystic calligraphy. The product
touches suffering; it must be dignified, never decorative about it. The tone is
Sufi-adjacent in *restraint* only — ink, gold, one turquoise — never in claim.

## 3 · The identity system

### 3.1 The emblem — «نشانهٔ تیا» (`src/art/emblem.js`)

One geometry, three readings, no face:

1. **The shamseh** — 16 rays in 8 long/8 short khatam rhythm, from r=17.4 to
   r=27.2, round caps. It says *Helios* before it says anything else.
2. **The calm almond** — an eye drawn with two arcs, never a stare; the lids are
   level, unhurried. It says *thea*, sight, being witnessed.
3. **The still pupil** — a small filled disc at the geometric centre: the light
   and the gaze are one thing. It says *Euryphaessa*, the wide-shining.

Usage:
- **Luminous** (`luminous` prop): gold radial gradient (#ffe9b8 → #f2c078 →
  #c08a44) on near-black grounds — Home, drawer, chat marks, install tiles.
- **Monochrome** (default): `currentColor` — inherits the surrounding ink on
  light grounds (e.g. approach-style swatch, wax seals on envelopes).
- Favicon/install icons: the emblem on a #12110d tile with a 22%-opacity warm
  halo, emblem at 72% scale (maskable-safe), generated by
  `scripts/build-brand-icons.mjs`. **Never hand-edit the PNG/SVG outputs** —
  edit `src/art/emblem.js` and re-run the script.

### 3.2 The narrative scene — «سپیدهٔ درون» (`src/art/dawnScene.js`)

A parametric SVG landscape in **0 0 340 680** (portrait-first). One `progress`
value in [0, 1] drives every channel: sky palette, star opacity, moon dimming,
sun altitude, ray opacity, horizon hairline, and the howz's reflection. It is
deterministic (seeded stars, mulberry32) and shared verbatim between the app
and the QA rasterizer (`scripts/render-brand-qa.mjs`).

Draw order (back to front): sky gradient → seeded starfield (with one turquoise
firouzeh star at index 17 and a long-lingering gold morning star) → Selene's
crescent → glow → shamseh rays → sun disc → far ridge → near ridge → horizon
hairline → the howz (water gradient, sun-path streak, reflection ellipse, three
ripple rings) → two khatam glints (only p > 0.55) → cypress sentinels → corner
vignette.

Both ridges read as the Alborz wall seen from the Tehran plateau; the pool is
the howz; the composition is a Persian garden at the moment the night lifts.

### 3.3 The rising hairline (`DawnProgress`)

Every onboarding screen carries a 2 px top hairline whose gold gradient widens
with `FLOW` position. Decorative (`aria-hidden`), theme-aware
(`#a9853f → #f6cf98` on light, `#8a5f2a → #ffdf9e` on dark), and it collapses to
instant under `prefers-reduced-motion`.

## 4 · The narrative arc, screen by screen

The onboarding sequence becomes the arc of one night becoming morning. Chapters
are labelled in Persian eyebrows (`.page-eyebrow`) — small, quiet, set in
Estedad — never as gamified "steps".

| # | Route | Chapter eyebrow | Progress | What the eye sees | Copy intent |
|---|---|---|---|---|---|
| 0 | `welcome` | «تیا · Θεία» | 0.06 | Full night; stars dense; crescent bright; only the morning star foretells the turn; horizon holds a breath | "خودت رو با نور ببین" — *See yourself with light.* The promise: here you are met, not fixed |
| 1 | `privacy` | — | (off-axis) | **The sealed eye**: an almond within a khatam octagram, a star for a pupil — light held inside | "این‌جا امن است." What you offer stays held, never exhibited |
| 2 | `simple-intro` | «سپیدهٔ اول» | 0.30 | The ridge ruled by a hairline of gold; sun still below; stars thinning; first warmth on the far peaks | "اول چند سؤال ساده بپرسم." Every answer is a small lamp lit in the dark |
| 3 | `deeper-intro` | «صبح‌هنگام» | 0.52 | Sun's crown clearing the ridge; glow pooling on the howz; rays at quarter-strength | "حالا بیشتر دربارهٔ خودت بگو." The work of being seen begins |
| 4 | `know-intro` | «سپیده‌دم» | 0.72 | Sun full above the ridge; reflection standing on the water | "بذار بیشتر بشناسمت." Almost nothing left unlit; a few honest answers remain |
| 5 | `almost-intro` | «طلوعِ نزدیک» | 0.88 | High sun; layered rays; the pool carrying the light; stars gone except a faint turquoise breath | "دیگه چیزی نمونده!" — the voice and style of the companion are chosen as the morning completes |
| 6 | `ready` | «طلوع» | 1.00 | Full radiance; the shamseh open; the howz a sheet of gold; corner cypress silhouettes | "صبحِ تو شروع شد" — *Your morning has begun.* CTA: «شروع گفت‌وگو» |

Question screens between chapters have no artwork of their own; they advance
the hairline instead — restraint keeps the scenes special and the form calm.
The `checkins` chapter is a single small sigil (`TheiaCheckinGlow`): one drop
of light arriving over still water — *a reminder as a found moment, not an
alarm*. The Dev Tester login shows the stage-0 night scene as a framed print,
linking identity entry to the same story.

Copy rules honoured throughout: warm, direct colloquial Persian (as the product
already speaks), no exclamation marks except where the moment earns one,
honesty preserved exactly (server-side storage, AI-processing disclosure on the
privacy and ready screens is unchanged in substance).

## 5 · Motion & accessibility

- `prefers-reduced-motion: reduce` freezes the ray rotation (140 s linear,
  `transform-box: fill-box`), the dust-in animations and the hairline transition.
- The scene is deterministic — no React randomness, no layout shift, no images
  to decode beyond fonts.
- All scenes are presentational (`aria-hidden` behind a labelled wrapper) or
  carry `role="img"` with a Persian description; decorative sparks are unlabeled.
- Keyboard paths, touch targets, 360/390/430 px layouts unchanged; dark scenes
  keep ≥ 4.5:1 text contrast (Estedad, #f4f3ef / #d8d4ca on #05070d).

## 6 · Tokens

| Token | Value | Role |
|---|---|---|
| `--theia-dawn` | `#f6cf98` | Dawn gold — rays, pupil, primary light |
| `--theia-moon` | `#d5d9f7` | Selene lilac — crescent, points, soft highlights |
| `--theia-gold` | `#d8a76c` | Accent gold — borders, small seals, eyebrow |
| `--theia-firouzeh` | (inline `#8fd6cb` / `#3f8f84`) | Turquoise — the one gem |
| `--theia-night` | `#05070d` | Full-bleed scene ground |

Legacy literal `#f6cf98` / `#d5d9f7` / `#d8a76c` occurrences elsewhere in CSS
were left intact deliberately (letters, chat, seals) — they are the same hues.

## 7 · File map

**New**
- `src/art/emblem.js` — tree builders for the emblem + dark tile
- `src/art/dawnScene.js` — tree builder for the six-movement scene + serializer
- `src/components/SvgTree.jsx` — tree→React renderer
- `src/components/DawnScene.jsx` — React wrapper + `DAWN_STAGES`
- `scripts/build-brand-icons.mjs` — regenerates favicon/touch/install icons (needs `sharp`, optional peer)
- `scripts/render-brand-qa.mjs` — rasters every stage to `/tmp/dawn-qa` for visual QA
- `public/brand/theia-emblem.svg` — standalone luminous emblem
- this file

**Rewritten**
- `src/components/TheiaBrand.jsx` — new emblem-based marks, `TheiaSealedEye`, `TheiaCheckinGlow`, restyled `TheiaLetterEmpty`
- `src/screens/Onboarding.jsx` — new chapters, copy, progression
- `src/screens/DevLogin.jsx` (via `DawnScene`), `src/components/UI.jsx` (`PageHeader` eyebrow + `DawnProgress`)
- `src/styles.css` — art rules, dark stages, hairline, eyebrow, mark styles
- `index.html`, `public/manifest.webmanifest`, `public/favicon.svg`,
  `public/brand/theia-logo.svg`, `public/apple-touch-icon.png`,
  `public/brand/theia-icon-{192,512}.png` — new launch identity
- `scripts/theia-brand.test.mjs` — brand contract tests updated

**Retired (kept on disk as historical assets, no longer referenced)**
- `public/brand/photo_2026-11-12_19-20-10.jpg` — the woman-shaped JPEG logo
- `public/brand/theia-welcome.png`, `theia-story-{1..4}.png`, `theia-ready.png`,
  `theia-privacy.png`, `theia-checkins.png` — woman-shaped illustration set
- `public/brand/theia-logo.svg` (old JPEG-embedding) — overwritten by the new tile

`FRONTEND_DESIGN_SPEC.md` carries the superseding 2026-11-26 decision note at
the top. `VALIDATION_REPORT.md` remains the historical record of the previous
logo decision and was intentionally left unedited.

## 8 · Honest notes

- No clinical claims were added or removed. The privacy and ready screens keep
  the existing server-storage / AI-processing disclosures verbatim in substance.
- All artwork is original vector geometry produced for this product — no stock,
  no AI-raster, no borrowed iconography.
- The QA loop for the scenes is `node scripts/render-brand-qa.mjs` and visual
  inspection of `/tmp/dawn-qa/*.png`; the interactive flow is verified by
  `npm test`, `npm run build` and `npm run check:foundation`.
