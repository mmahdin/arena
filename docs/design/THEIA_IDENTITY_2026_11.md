# Theia identity — "the light and the seeing of it"

Decision date: 2026-11-26 · Status: implemented · Supersedes the 2026-11-12
female-profile logo decision.

## The problem with the old identity

The previous identity drew Theia as a long-haired woman in a beige drape,
holding a small sun or a crescent, and used a photographic female profile as
the product's logo.

For an Iranian audience that fails in three ways at once.

1. **It is a borrowed body, not a borrowed idea.** Iranian wellness and
   lifestyle apps are saturated with illustrated women — calm faces, flowing
   hair, pastel drapery. A Greek goddess rendered in that same vocabulary reads
   as either imported fantasy or as one more lifestyle model. It does not read
   as *Theia*.
2. **It assigns a gender to the therapist.** The product is a mental-health
   companion used by women and men, and the onboarding asks the reader how they
   identify. Depicting the companion as a woman contradicts that neutrality in
   the very first screen.
3. **It is decorative where it should be structural.** A figure is close to
   unusable at 16px in a seal, a signature, a chat bubble or a favicon. The
   identity had already fragmented into three different drawings.

## What the myth actually gives us

Theia (Θεία) is a Titaness. Her domains in Hesiod and Pindar are *divine
light* and *sight* — her name is built on **thea**, "a seeing" — and she is the
mother of **Helios** (sun), **Selene** (moon) and **Eos** (dawn). Light, in her
myth, is not decoration: it is the precondition for seeing, and seeing is the
precondition for acting well.

That is precisely the psychological arc of the product: a person arrives unable
to see their situation clearly, and the work is to make it visible without
being judged for it. So we do not illustrate the goddess. We illustrate the
thing her myth is made of.

## The concept

**A threshold, and the light that appears in it.**

The threshold is a two-centred Persian arch — the *tāq* of a mihrab, an iwan, a
doorway in a bazaar. Hairline only, no ornament. It was chosen for an Iranian
reader specifically:

- it is a shape they have physically stood in front of, in a mosque doorway, a
  caravanserai, a grandmother's house;
- it means "you may enter" and "you are received" without making a religious
  claim, which matters in a product that asks about faith;
- it lends the product the dignity of architecture instead of the intimacy of a
  body, which is the correct register for a mental-health tool;
- it is *the same shape* as the Greek post-and-lintel idea of a threshold, so
  the Greek myth and the Persian visual culture meet in one line rather than
  one being translated into the other.

Each chapter keeps that arch and changes only the light inside it.

| Movement | Myth | Light | What it says to the reader |
| --- | --- | --- | --- |
| **I · شب / Night** | Selene | a crescent above a dark horizon; stars; the moon only lends edges | "In the dark you can only find the edges of things. That is not failure; that is where everyone starts." |
| **II · سپیده / Dawn** | Eos | a warm line at the horizon, sky graded from night-blue to gold, the sun's arc just surfacing | "The first line of light returns the shape of things. Naming what is wrong is the first usable step." |
| **III · خورشید / Sun** | Helios | rays, a star at the centre, nothing hidden | "Full light hides nothing — and it does not judge what it shows. You can look at this without being condemned for it." |
| **IV · دیدن / Sight** | Theia | the arch is filled from the floor up; light is on the reader's side of the threshold | "The light and the seeing are the same event. From here, you are not an audience." |

The arch is drawn as a **hairline** throughout. Nothing is filled with flat
colour; light is always a gradient that fades before it reaches the edges of the
frame. This is what keeps the work from becoming decorative: the restraint is
the statement.

## Where it appears

| Surface | Expression |
| --- | --- |
| Welcome | The arch almost empty, one dawn star inside, a slow aurora behind. Four chapters named in a single quiet rail. |
| Privacy | The same arch, closed; the light contained by a boundary. "Light inside its own border." |
| Name | No figure. Only the aurora and one gold underline that warms on focus. |
| Chapter interstitials (4) | One scene each, plus the Greek anchor in small Naskh and one line connecting myth to the questions that follow. |
| Questions | The chapter rail only — colour and active numeral change per chapter. No illustration competes with reading. |
| Check-ins | Concentric rings that return on a slow pulse: light that comes back on its own. |
| Ready | The arch opens onto a smaller inner arch — a gate — with the bloom on the near side. |
| Home | The mark inside the existing orbit, now drawn as an inline SVG rather than a photo. |
| Letters, seals, signatures, drawer, favicon, app icon | The same mark, one geometry, generated. |

## Craft decisions

- **Geometry, not hand-drawing.** `src/components/theia/scenes.js` returns scene
  data; `TheiaScene` renders it; `scripts/render-theia-scenes.mjs` renders the
  same data for review; `scripts/build-brand-assets.mjs` renders the icons. The
  icon, the favicon and the welcome screen cannot drift.
- **Persian type for the inscribed voice.** Chapter titles and myth lines use
  Noto Naskh Arabic; interface copy stays Estedad. Naskh carries the "written
  down, therefore true" register without any calligraphic pastiche.
- **Night is a first-class theme**, not a dark variant: `screenSurfaceForRoute`
  returns `night` for the onboarding narrative so a dark page never fades in
  from a light shell.
- **Motion is one vocabulary**, used sparingly: breathing light, a star that
  surfaces, a horizon that settles, ripples that return. Everything stops under
  `prefers-reduced-motion`.
- **Copy never promises a cure**, never asks the reader to believe in a
  goddess, and keeps the existing honest disclosure about server storage and AI
  processing exactly where the privacy screen had it.

## Copy register

Warm, adult, second person, no exclamation marks, no diminutives, no emoji in
the narrative. The myth is used the way a good metaphor is used — to name a
sequence the reader already recognises — not as authority. Lines were written to
survive being read aloud by a tired person at 2 a.m.

## Files

- `src/components/theia/scenes.js` — geometry, palette, movements, scene specs
- `src/components/theia/TheiaScene.jsx` — React renderer
- `src/components/TheiaBrand.jsx` — mark, seal, hairline, home art, empty state
- `src/features/onboarding/theiaNarrative.js` — copy, chapters, route→chapter map
- `src/screens/Onboarding.jsx` — Welcome, Privacy, Name, chapters, Check-ins, Ready
- `src/styles.css` — `/* Theia identity */` section
- `scripts/render-theia-scenes.mjs` — review sheet (`artifacts/theia-scenes`)
- `scripts/build-brand-assets.mjs` — `public/brand/**`, `public/favicon.svg`

## Open items

- The narrative copy is Persian-only, like the product; if an English build
  appears, the four chapter names and myth anchors need a written translation
  rather than a transliteration.
- `public/assets/` still holds the 2020-era floral artwork used by legacy
  screens (research/experts). Those screens are excluded from onboarding and
  still carry unverified institutional claims; they were intentionally left
  untouched here.
