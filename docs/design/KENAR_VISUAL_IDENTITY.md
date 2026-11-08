# Kenar (کنار) — visual identity

> **Scope.** This document replaces the Ash-inspired floral identity with an original
> Kenar identity. It changes *artwork, colour, symbols and character*; it does **not**
> change layout geometry, navigation, interaction, component structure, spacing or
> flow order. The only flow-adjacent edits are copy lines that named the old brand
> (see "Copy changes").

## The word, and why it is the whole identity

In Persian, **کنار** is not "next to" as a coordinate. It is the place you *stay*:
*کنارِ کسی بودن* — to be at someone's side. The word already holds the product's
promise, so the identity is built from the word rather than decorated around it.

Two senses are used, and they are the same picture:

| sense | Persian | what it means here |
| --- | --- | --- |
| beside a person | کنارِ کسی بودن | presence, companionship, staying |
| the edge of water | کنارِ رود / کنارِ دریا | the shore — the line where things come to rest |

Both resolve into one image: **two river stones resting side by side on a single
shore line.**

## The mark

Two stones, one shore. The larger stone is Kenar; the smaller is you. Neither is
centred, neither is above the other — they are *beside*. The line beneath them is
simultaneously the shore and the sentence that holds them.

- Built from one idea, not a decorative motif: it reads at 20 px as a silhouette.
- No flower, no petal, no stem, no face in the logo. The mark is a place, not a mascot.
- The small stone sits lower and closer to the shore, which keeps the pair from
  reading as "a big thing and its accessory".

Files: `public/assets/brand/kenar-mark*.svg` (colour / mono / reversed),
`kenar-icon.svg` + `kenar-icon-maskable.svg` (app icon, 36 px corner radius at 160,
22.5 % at runtime), and a React component `KenarMark` in `src/components/UI.jsx`
that inherits `currentColor`.

## The companion — "سنگِ صبور", the patient stone

The character is not a new species invented for an app. It comes from a Persian
idea people already carry: **سنگِ صبور** — the patient stone by the river that you
sit with, talk to, and leave your grief with, until it softens. That story is
already an image of *being beside someone*, so Kenar's character is that stone,
and it is the one sitting next to *you*.

Design rules:

- **Closed, unhurried eyes.** Never wide, never looking at the user. Kenar looks
  *with* you, not *at* you.
- **A body like a river stone**: wide flat base, soft shoulder, asymmetric weight.
  It always rests on a line; it is never floating.
- **No limbs, no mouth-open expressions, no gender.** Emotion is carried by the
  eyes and by proximity to the second stone.
- **The smaller stone is presence, not a second character.** It never speaks, never
  has a full face, and never appears alone. It exists so the pair reads as *beside*.
- **Scale means intimacy.** Distance between the two stones is the story: they
  begin apart and end touching.

## The shore line

A single hand-drawn-weight curve under every scene. It is the recurring horizon of
the identity — the thing that makes otherwise separate screens feel like one place.
It is drawn as `currentColor` so it flips automatically between paper and night.

## Colour

Replacing the lavender / mint / butter-yellow accents of the Ash-era UI.

| token | hex | role |
| --- | --- | --- |
| `--light` paper | `#f7f3e9` | daylight surfaces (onboarding) |
| `--sand` | `#d8c7a6` | stone body |
| `--sand-deep` | `#bfa87e` | the smaller stone |
| `--ochre` | `#8a6849` | stone outline, focus ring |
| `--clay` | `#c37c52` | the single warm accent (ripples, seals, unread) |
| `--sage` | `#a6c1ab` | growth / on-states |
| `--ink` pine | `#1e2a23` | type, mark, primary buttons |
| `--dark` night | `#101510` | home, chat, letters |

Rules: paper and pine carry the layout; sand is only ever the stone; clay appears
**once per screen at most**; sage is reserved for "yes / on" states.

## Story arc across onboarding

The existing 26-step flow is unchanged. Only the artwork on the four interstitial
beats and the welcome/ready screens now tells one continuous story:

| step | art | story |
| --- | --- | --- |
| welcome | `kenar/welcome-figures.svg` | a stone arrives at the shore; Kenar is already there, a step away |
| privacy | `kenar/privacy-ring.svg` | the stone inside an unbroken, dotted ring |
| simple-intro | `kenar/companion-small.svg` | Kenar alone — "let me ask a few simple things" |
| deeper-intro | `kenar/companion-medium.svg` | the second stone appears, still apart |
| know-intro | `kenar/companion-large.svg` | they are close |
| almost-intro | `kenar/companion-tall.svg` | side by side |
| checkins | `kenar/checkins-rhythm.svg` | one stone, upright, on a flat shore |
| ready | `kenar/ready-companions.svg` | the two walk the shore together |

Each scene is authored on a canvas matching the real CSS slot
(`.welcome-art` 346×491, `.art-small` 306×288, `.art-medium` 306×314,
`.art-large` 306×373, `.art-tall` 314×491, `.ready-art` 334×470,
`.checkins-art` 342×295, `.privacy-art` 102×116) and every stone is positioned by
a fraction of the frame width, so no composition can run off an edge no matter how
the Persian copy above it reflows on a small screen.

The chat hand-off greeting changed from a leaf emoji to
`من کنارتم؛ هر جا که بخوای ازش شروع کنیم.` — the first line Kenar ever says is
the promise of the word.

## Where the old identity was removed

| old | new |
| --- | --- |
| `welcome-figures.png`, `flower-person-*.png`, `ready-daisy.png`, `privacy-flower.png`, `checkins-handdrawn.svg` | `public/assets/kenar/*` — each scene re-authored at its slot's true aspect ratio (see the table below) |
| `<Flower2>` on home, chat empty state, letter seal, unread badge, style-preview reply | `<KenarMark>` |
| floral wax seal | wax seal pressed with the Kenar mark |
| `🌿` in the onboarding greeting | no emoji; the promise in words |
| astral-flower constellation art | `ConstellationArt` = two stones as points of light on a shore |
| spiral "insight" doodle | `InsightSpiral` = ripples on water |

The old PNGs are deleted rather than left in place, so no stale Ash artwork can
reappear.

## Copy changes (brand name only)

`همراه` → `کنار` in: welcome title/description, testimonials + research headings,
the referral question, the letters heading and signature, the drawer brand, the
mail aria-labels, the dev-login title, the letter-context prompt sent to the model,
and `index.html` (title, description, theme colour, icon links).

No other wording, layout or control was touched.

## What deliberately did **not** change

Route order (`FLOW`), `ONBOARDING_ROUTES`, the age wheel, the mood slider, the
approach swipe preview, the voice carousel, chat behaviour, bookmarks, letters,
privacy toggles, settings, all hit areas, all ARIA and all reduced-motion rules.
Geometry was preserved; only colour tokens, artwork, and glyph substitutions moved.

## Files

- Generator: `scripts/brand/build-kenar-assets.mjs` (deterministic, no network).
- Brand sheet: `docs/design/kenar-brand-sheet.svg` / `.png`.
- Assets: `public/assets/brand/`, `public/assets/kenar/`.
- React: `KenarMark`, `KenarWordmark`, `ShoreStoneArt`, `ConstellationArt`,
  `InsightSpiral` in `src/components/UI.jsx`.
- Styles: the `KENAR IDENTITY` block at the end of `src/styles.css`.
