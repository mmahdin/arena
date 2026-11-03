# Hamrah visual identity — «هم‌سنگ»

**Version 1.0 · 2026-11-03 · supersedes the borrowed Ash-style flora in the illustration slots**

This document is the written record of the identity redesign. The visual brand book is
`docs/brand/Hamrah-Brand-Book.html` (self-contained, open it in any browser).

---

## 1. What was wrong before

The onboarding illustrations (`welcome-figures`, `flower-person-*`, `ready-daisy`,
`privacy-flower`) were flower-people with a single eye and walking legs. They were charming,
but they were **the Ash app's** charm: the same wavering ink contour, the same petal-head
silhouette, the same one-eyed face. A mental-health product cannot borrow its most emotional
asset from another product — it reads as a skin, not a self.

The identity also leaned on repeated stock gestures (a wax seal holding a `Flower2` icon, a
generic constellation, a squiggle speech bubble) that carried no shared meaning.

## 2. The idea: «هم‌سنگ» — the companion stone

> Two stones lean together. In the notch where they meet, an ember stays lit.

- **Stones** are patient. They do not hurry, judge, or leave. In mountain paths, cairns are
  stacked so the next traveller knows someone passed this way — a quiet, non-clinical way to
  say *you are not the first and not alone*.
- **Two, never one.** The mark is made of two unequal stones: the smaller leans on the taller.
  Dependence is the shape of the relationship, not a defect.
- **The notch is not empty.** Where the two stones meet, one warm dot sits. Attention is the
  product's core act; the ember is its only warm colour in any frame.
- **A flat base.** Both stones rest on the same ground line. Nothing in this identity floats.

The name is a coinage echoing «همراه» (companion) and «سنگ» (stone) — *ham-sang*, "stone-together".
It gives the design team a word to think with: every new asset must be *ham-sang*.

## 3. The system (three rules)

| Element | Rule |
| --- | --- |
| **Pebble** | One parametric superellipse with a leaning crown, a flattened base and a whisper of irregularity. Deterministic — the same stone every render. |
| **Ember** | Exactly one per frame. `#c1664c` on light, `#c9705a` on dark. It is the only warm colour allowed. |
| **Ground line** | A single clay stroke (`#c9c1b4`) that every scene rests on. Omitted only in the bare mark. |

The mark is the two pebbles plus the ember. The **characters** are the same pebble standing
upright with two small feet and an ember at its heart. The **scenes** are arrangements of
characters and loose stones on one ground line. The **vessel** is a wide stone with a carved
hollow that keeps the ember (used for privacy and the daily reflection).

## 4. Characters

A companion is: one upright pebble + two tapered legs + one ember. It has **no face** — no
eyes, no mouth, no tears. This is a product decision as much as an aesthetic one:

- It avoids implying diagnosis or emotional judgement.
- It keeps the product's tone quiet, which the product constitution requires.
- Feeling is carried by **posture**: leaning toward another stone means attention; the gap
  between feet means pace; a low, wide heap means rest.

Poses shipped: `stand`, `walk`, `wave`, `offer`, `reach`, `hold`, `sit`.
The `sit` pose is a resting heap with a small companion stone leaning against it.

## 5. Colour

Unchanged from the designer palette. One new rule:

> In any single frame, only one thing may be ember.

| Token | Hex | Role |
| --- | --- | --- |
| Ink | `#23201b` | Stone, text, primary button |
| Night | `#12110d` | Home, chat, drawer |
| Marble | `#f3f2ee` | Light surfaces |
| **Ember** | `#c1664c` | The one warm dot |
| Sand | `#cec4b8` | Beige interstitials, day panel |
| Mist | `#d5d9f7` | Quiet actions, speech |
| Sun | `#f6cf98` | Reflection card, letter |
| Moss | `#9eda9e` | Insight panel |

## 6. Typography

No new typefaces. **Estedad** stays the interface face; **Noto Naskh Arabic** keeps the
brand name, the home greeting and the tagline — exactly as the product already used it.

## 7. Motion

- Scene arrival: 4px rise, 200 ms — the product's existing page timing.
- The ember breathes: 4 s at rest, 1.6 s while the companion is thinking.
- Everything stops under `prefers-reduced-motion`.

## 8. Where it lives in the app

| Surface | Identity moment |
| --- | --- |
| Welcome | Two companions, one thread of attention, one ember |
| Name | A companion looks up at the ember it is about to be given |
| Intro screens | Stones laid, stones climbing, a path to a resting companion, a cairn |
| Privacy | A companion keeps watch beside the vessel that holds the ember |
| Ready | Two companions walking on, ember between their shoulders |
| Home | The wax seal of the daily letter carries the mark |
| Chat | The companion breathes in the header while it is thinking |
| Day | A resting vessel stone holding the day's ember |
| Insight | Stones joined by a dotted clay thread |
| Menu | Mark + wordmark, replacing `Flower2` |
| Dev login | Mark, name, tagline — the brand's first impression |

## 9. Source of truth

All geometry lives in code, not in exported images:

- `src/brand/geometry.js` — the parametric pebble, splines, ground line (no randomness)
- `src/brand/scenes.js` — every scene as plain layer data
- `src/brand/Scene.jsx` — the renderer used by the app
- `src/brand/Mark.jsx` — `Mark`, `MarkTile`, `Lockup`, `CompanionScene`
- `public/brand/*` — static exports generated from the same code
- `docs/brand/Hamrah-Brand-Book.html` — the visual book

Static exports are generated from the module, so the app, the book and any future print
artefact can never drift apart.

## 10. What was preserved

- The designer's screens, route order, CSS system, tokens, spacing and motion values.
- Every original illustration asset remains in `public/assets/` (nothing deleted) so the
  previous look can be compared or restored.
- All copy, including the copy flagged as unverified in `PRODUCT_CONSTITUTION.md`. This
  change is visual only and makes no claim about backend capability.

## 11. Regenerating the artefacts

Nothing in `public/brand/`, `public/favicon.svg` or the two book copies is hand-written.
After changing `src/brand/geometry.js` or `src/brand/scenes.js`:

```bash
node scripts/brand/build-assets.mjs   # marks, icons, share card, favicon
node scripts/brand/build-book.mjs     # public/brand/ + docs/brand/ copies of the book
```

`build-book.mjs` copies the three `@fontsource-variable` faces the app already depends on
(Estedad Arabic/Latin, Noto Naskh Arabic) next to the live book, and writes the archived copy
with those fonts inlined so it opens from disk and inside a sandboxed iframe with no network.

PNG re-rendering is optional and needs a rasterizer that is deliberately not a project
dependency:

```bash
npm i --no-save @resvg/resvg-js && node scripts/brand/build-assets.mjs
```

`scripts/brand.test.mjs` imports `svgAssets()` and asserts the committed SVGs and favicon are
byte-identical to what the geometry generates, so the exports cannot silently drift from the
code the app uses.

Dev-server host binding is unchanged (loopback only). Hosted preview sandboxes that need to
reach the server from another machine opt in per environment instead of loosening the repo
default:

```bash
HAMRAH_HOST=0.0.0.0 HAMRAH_ALLOWED_HOSTS=.e2b.app npm run dev
```
