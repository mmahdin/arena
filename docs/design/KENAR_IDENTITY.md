# کنار — Visual identity

> Status: **current visual authority (2026-11-08).** It replaces the Ash-inspired
> floral identity that shipped with the original designer source. It does **not**
> change layouts, navigation, spacing, component structure, interaction or flow.

## 1. Why this identity exists

The product is called **کنار**.

In Persian the word carries two inseparable meanings:

1. **کنارِ کسی بودن** — being beside someone: companionship, presence, someone
   who is simply there.
2. **کنار** — the shore, the edge where two things meet.

Both meanings describe exactly what this product is for. It is not a clinician
above you and not a stranger in front of you. It is someone who sits **beside**
you and stays. The identity had to make that sentence visible before a user reads
a single word.

So the identity has **one subject, and only one**:

> **Two presences, seated side by side, on a single unbroken shore line.**

Nothing else belongs in the picture. No flowers, no mascot waving at the camera,
no decorative symbols borrowed from another product. If a new illustration can be
described without two presences and a shore, it is not a Kenar illustration.

## 2. The presence (Kenar's "character")

Kenar is a **presence**, not a mascot.

- **The warm form (clay).** Kenar itself: a seated figure seen from behind, small,
  round, wrapped — the silhouette of someone sitting with their knees drawn up
  under a blanket. It never faces the viewer, so the user never has to meet a
  stare. It is *beside*, which is the whole point.
- **The ink form.** You. Same shape, larger, drawn in the surface's ink. Two
  figures of the same family is what makes them read as companions rather than a
  character and an audience.
- **The pebble.** The smallest trace of the presence, used where a full figure
  would be too loud (badges, seals, unread marks, favicon detail). Kenar sets a
  pebble down where you will sit: *«جات رو کنارم نگه داشتم.»*
- **The shore line.** The single, slightly wavering line the whole system rests
  on. It is the ground, the horizon and the promise of continuity. In the
  logomark, in every scene and under the wordmark, the line is always unbroken
  and always runs past both figures.

Kenar is never sad, never celebratory, never "cute". It is **warm, patient and
quiet** — the emotional register of someone who has sat with you before.

## 3. The logomark, wordmark and icon

**Logomark** (`src/brand/kenar.jsx` → `KenarMark`): two presences on one shore.
It is a miniature of the product's core image, so the icon is not a symbol *about*
the brand — it is the brand's sentence. It stays readable at 16 px because it
reduces to two dots on a line.

**Wordmark**: the name set in the already-installed **Noto Naskh Arabic**, the
editorial voice of the product, with the shore line drawn underneath and a single
clay pebble closing the word. Naskh makes the name feel written rather than
manufactured.

**App icon**: `public/assets/kenar-icon.svg` (+ `kenar-icon-512.png`,
`kenar-touch-icon.png`). Night ground, cream figure, clay figure, one shore line,
and a dusk disc behind them. It is the same drawing as the logomark, not a second
concept.

## 4. Visual grammar

| Element | Rule |
| --- | --- |
| `--k-fg` ink | You, the line, the light on a dark screen |
| `--k-accent` clay `#B9775A` | Kenar, warmth, wax seals, unread marks |
| shore line | Unbroken, never dashed, never enclosed in a frame |
| dusk disc | A soft light on the horizon; one per scene, never a second sun |
| listening rings | Ripples around the pair, open or full circles; never a rainbow |
| grain | 5% paper noise over large illustrations so flat colour feels made, not rendered |

All primitives are CSS-variable driven, so the same JSX drawing is paper-and-ink
on the onboarding and lamplight-on-night on Home, Chat and Letters.

## 5. Palette

| Token | Value | Role |
| --- | --- | --- |
| `--light` | `#F4F1EA` | paper (onboarding, important surfaces) |
| `--ink` | `#221F19` | text, the user's presence |
| `--dark` | `#14120E` | night (Home, Chat, Letters) |
| `--beige` | `#D8CAB8` | sand (approach, age marker, reminder panel) |
| `--clay` | `#B9775A` | **signature**: Kenar, seals, unread marks |
| `--dusk` (`--lavender`) | `#CEC8E0` | interactive accent, unchanged role |
| `--yellow` | `#EEC596` | lantern (letters) |
| `--green` | `#A6D3A8` | success only |

The previous tokens were kept by name and moved by only a few degrees, so every
existing component keeps its contrast, hierarchy and identity — the product simply
becomes warmer.

## 6. Typography

- **Estedad** — the interface voice (unchanged).
- **Noto Naskh Arabic** — the emotional voice: the wordmark, Home greeting,
  letter bodies, drawer brand, letter signatures.

The two already existed in the app. The identity formalises the split: Estedad
speaks *to* you, Naskh is the voice that *sits with* you.

## 7. Onboarding: the story

The onboarding flow is untouched — same 21 visible screens, same order, same
questions, same buttons. The story is told inside the frames that already existed,
so nothing is added to the user's path.

**One story, six frames** (`PresenceScene`):

| Frame | Screen | What is drawn | Copy |
| --- | --- | --- | --- |
| 0 | Welcome | Kenar alone at the edge of the shore | «با کنار آشنا شو» / «کنار یعنی کسی همین نزدیکی، کنارت…» |
| 1 | `simple-intro` | Kenar closer; a pebble marks your place | «جات رو کنارم نگه داشتم…» |
| 2 | `deeper-intro` | You arrive and sit; still a gap | «حالا که نشستی، آرومآروم بیشتر بگو…» |
| 3 | `know-intro` | Nearer, both present | «کمکم نزدیکتر میشیم… کسی قضاوتت نمیکنه.» |
| 4 | `almost-intro` | Shoulder to shoulder | «حالا دیگه شونهبهشونه نشستیم…» |
| 5 | `ready` | The full frame, largest | «من کنارتم.» / «هر وقت خواستی حرف بزنی، همینجام.» |

The camera slowly moves closer across the six frames, which is why trust and
proximity feel like the same thing. The user's own arrival *is* the story; no
narrative text is required to explain it.

## 8. Where the identity is applied

| Surface | Identity element |
| --- | --- |
| Welcome / 4 interstitials / Ready | six-frame presence story |
| Privacy | a wide pebble with a keyhole, and someone sitting beside it |
| Check-ins | one small bell under an arch, the pair sitting beneath it |
| Home | two presences inside listening ripples, inside the existing halo |
| Chat (empty) | one seated presence, no demands |
| Chat/History/Letters header, menu | Kenar wordmark, `KenarMark` |
| Envelope | the wax seal now carries the logomark in relief |
| Chat empty / approach replies | logomark in monochrome, 20 px |
| Letters | «نامههای کنار», signature «کنار» |
| Dev login | presence scene + logomark |
| Browser chrome | `kenar-icon.svg`, apple touch icon, title «کنار» |

## 9. Copy voice

- Second person, short sentences, no diagnosis, no promises, no marketing claims.
- Kenar speaks about **place**, not about **helping as a service**:
  «همینجام»، «کنارتم»، «جات محفوظه».
- Never use the vocabulary of flowers, growth, bloom or petals — that identity is
  retired, and the new one is about **proximity**, not flora.

## 10. Motion

- Home: listening ripples breathe slowly (7 s) while the halo breathes (7 s).
- Presence figures drift 2 px on hover; no bouncing, no spring, no confetti.
- Every animation is disabled under `prefers-reduced-motion`.

## 11. Files

```
src/brand/kenar.jsx                     primitives, mark, wordmark, scenes
src/styles.css  (KENAR identity layer)  tokens, k-* primitives, wordmark, grain
public/assets/kenar-icon.svg            favicon / icon source
public/assets/kenar-icon-512.png        app icon
public/assets/kenar-touch-icon.png      iOS icon
docs/design/kenar/kenar-identity-board.* identity board (this document, visual)
scripts/identity-preview.mjs            synthetic, server-free preview of every screen
```

Retired (Ash-inspired, deleted): `welcome-figures.png`, `flower-person-*.png`,
`privacy-flower.png`, `ready-daisy.png`, `checkins-handdrawn.svg`.

## 12. See it

```bash
npm run preview:identity   # http://localhost:5174 — every screen, fixture data only
```

The preview mounts the real screens with synthetic content; it never authenticates
a tester, calls the API or writes data. Use the route selector to walk the story
frame by frame (`welcome` → `ready`) and then the product (`home`, `chat`,
`letter`, `menu`).
