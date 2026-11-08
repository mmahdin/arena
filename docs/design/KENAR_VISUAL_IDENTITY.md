# کنار — Visual identity system

> **Status:** active identity, applied across the existing product.
> **Scope:** visual identity only. Layouts, navigation, spacing, interactions and component
> structure of the designer source are preserved; only the brand marks, illustrations,
> symbols and the product's own name change.

---

## 1. The name is the idea

**کنار** is a small word with three meanings that all point at the same feeling:

| sense | Persian | what it means for the product |
| --- | --- | --- |
| beside | «کنارِ تو» | presence without pressure — someone sitting next to you, not in front of you |
| the shore, the edge | «کنارِ رود»، «کنارِ صفحه» | the quiet margin of a page, the place where words come to rest |
| coming to terms | «با خودم کنار آمدن» | the work of the product: making peace with yourself, slowly |

The identity is built from exactly that: **one long line (the shore / the conversation)** and
**one small presence that sits on it, beside your words**.

There is no flower, no petal, no mascot holding a sign, no star, no heart. The brand does not
decorate the page; it keeps it company.

---

## 2. The character — «هم‌نشین»

Kenar's presence is called the **هم‌نشین** (the one who sits beside). It is deliberately not a
person, not an animal and not a plant:

- a **soft sand-coloured dome** — the shape of a shoulder, a hill on a horizon, a gathered blanket;
- a **small curled tuft** on top — the one asymmetrical detail that makes it feel alive;
- **two ink dots and a quiet smile** — drawn, never modelled; it looks at the reader, not at the camera;
- it has **no arms or legs** — it does not do anything to you. It *stays*.

It always sits **on the line**, at the edge of the composition. It is never centred in a hero
bubble, never waving, never winking. Its only gestures are the ones the product actually offers:

| pose | when | meaning |
| --- | --- | --- |
| `sit` | first meeting, home | «همین‌جام» |
| `lean` | deeper onboarding | «بهت گوش می‌دم» |
| `rest` (eyes closed) | the long intake | trust; nothing is demanded |
| `voice` (sound arcs) | before voice selection | «حالا صدای من رو هم بشنو» |
| `arrive` / `ready` | welcome → ready | the relationship beginning |

The tuft is the character's fingerprint. The silhouette must stay recognisable at 16 px, so the
logo and favicon drop the tuft and keep only: **dome + horizon + face**.

---

## 3. The mark

The **seal** is the companion reduced to a logo: a sand dome with two dots and a small smile,
sitting on a hand-drawn horizon. The horizon runs wider than the body, so the mark reads as
"a presence *beside* something", not as an avatar in a circle.

- **App icon:** the same dome, bigger, on `--kenar-night` (#12110d) with a warm halo and a second,
  fainter horizon line below — the two-line "beside" motif.
- **Small sizes (16–32 px):** the icon uses a bolder face and drops the echo line, so the eyes and
  smile still survive the raster.
- **Wordmark:** «کنار» in **Estedad 700**, tightened to −1.6 px, with the mark at the right
  (RTL start) and the two-line motif beneath. The name and the presence sit on the same horizon.
- The favicon is the icon without the second line, at 32/16 px.

Nothing in the identity uses a perfect circle, a perfect arc or a straight line. Every contour
carries a deterministic 0.5–2 px wobble (a low-frequency sine applied along the path), so the
brand looks inked by hand at every size while staying pixel-crisp and resolution independent.

---

## 4. The two-line motif

The second graphic signature is **two lines that never cross**:

- the long line is the conversation;
- the short, fainter line beneath is your own place beside it.

It appears as the horizon under the character, the rule under the wordmark, the wax seal's
ground, the privacy loop, the echo under the letter signature, and the faint "far shores" in the
onboarding illustrations. It replaces every flower and decorative squiggle from the previous
identity.

---

## 5. Colour

The existing product palette is kept. Kenar only *adds* three values that belong to the character:

| token | light | dark | use |
| --- | --- | --- | --- |
| `--kenar-sand` | `#e9d0ae` | `#e9d0ae` | the companion's body, the horizon on the icon |
| `--kenar-clay` | `#b9744f` | `#b9744f` | the blush, seal accents, the beginning dot |
| `--kenar-shore` | `#a89e90` | `#7e776c` | fainter second line, distant shores |

Ink stays `--ink` (#23201b) on light and paper (#efe8db) on dark, inherited through
`currentColor`, so the illustrations theme themselves instead of shipping two raster files.

The existing tokens (`--light`, `--beige`, `--dark`, `--lavender`, `--yellow`, `--green`) and every
screen surface (`light` / `beige` / `dark` / `black`) are untouched.

---

## 6. Typography

The product already owns a strong Persian type system and it stays:

- **Estedad** — interface, headings, the wordmark;
- **Vazirmatn** — fallback / long-form;
- **Noto Naskh Arabic** — letters, the diary-like serif moments (home greeting, letter body).

The identity does not add a fourth typeface. Its personality comes from the drawing, the copy
and the two-line rhythm — not from a new font.

---

## 7. Motion

Motion is used to make the presence feel alive, never to decorate:

1. **The horizon draws itself** (1 s, dash offset), then
2. **the companion settles** onto it (0.7 s, 8 px rise + micro-scale), and only then
3. **its first words fade in** (the welcome caption «سلام. من کنارم.»).
4. On Home and Chat it **breathes** slowly (6.5 s) and **blinks** every ~8.5 s.

All of it is disabled under `prefers-reduced-motion: reduce`, and no state depends on an
animation completing.

---

## 8. Voice

The character speaks in the first person, in the product's existing warm Iranian Persian, and
never as a clinician:

- «سلام. من کنارم.»
- «همین‌جا می‌مونم؛ هر وقت آماده بودی.»
- «لازم نیست همه‌چیز رو یه‌بار بگی.»
- «داری برام آشنا می‌شی.»
- «از این به بعد، کنارتم.»

Short sentences, no exclamation marks except the one existing «من آماده‌ام!», no promises about
outcomes, no clinical claims. The copy carries the story; the illustrations never spell it out.

---

## 9. The onboarding story (in the existing flow)

The onboarding was not restructured. The relationship is told inside the screens that already
existed, so the user still walks the same 26 steps:

| step | what the person sees | story beat |
| --- | --- | --- |
| `welcome` | the companion arrives on the horizon, first words appear | meeting |
| `simple-intro` | small, waiting, with room around it | "I'll stay" |
| `deeper-intro` | leaning in | "no need to say everything at once" |
| `know-intro` | eyes closed, resting | trust |
| `almost-intro` | sound arcs | "now hear my voice" |
| `privacy` | inside a closed loop | "your words stay here" |
| `checkins` | soft reminder ripples | "I'll check in sometimes" |
| `ready` | sitting beside the line that now runs forward | «از این به بعد، کنارتم» |
| first chat | greeting: «…، خوش اومدی. من کنارم.» | the relationship continues |

No new routes, no new ordinals, no new required input, no change to the server-owned progress
cursor or the sensitive-intake boundary.

---

## 10. Where the identity is applied

| surface | before | after |
| --- | --- | --- |
| Browser chrome | title «همراه», no icon | title «کنار», favicon, manifest, theme |
| Welcome | flower pair PNG | `KenarScene variant="welcome"` + first words |
| Onboarding interstitials ×4 | flower people PNGs | four poses on the same horizon |
| Privacy | flower + lock | the companion inside a closed line |
| Check-ins | hand-drawn phone SVG | reminder ripples |
| Voice/approach avatar | lucide `Flower2` | `KenarMark` |
| Ready | daisy PNG | `KenarScene variant="ready"` |
| Home centre | `Flower2` in the orbit | the companion sitting inside the orbit |
| Home letter badge | `Flower2` | `KenarGlyph` |
| Envelope wax seal | `Flower2` | `KenarMark` pressed in wax |
| Chat empty state | `Flower2` | `KenarMark` |
| Drawer brand | plain «همراه» text | `KenarWordmark` + two-line rule |
| Letters | «نامه‌های همراه», signature «همراه» | «نامه‌های کنار», signature «کنار» + glyph |
| Login | daisy PNG + flower | `KenarScene variant="login"` + mark |
| First chat greeting | «…خوش اومدی. 🌿» | «…خوش اومدی. من کنارم.» |

The previous floral PNGs remain in `public/assets/` untouched (the source-preservation rule),
but nothing references them any more. Portraits of experts are photography, not Ash artwork, and
are unchanged.

---

## 11. Accessibility

- All illustrations are `aria-hidden` decoration; the only text that carries meaning is real text
  (the welcome caption, headings, descriptions).
- The mark exposes no colour-only meaning; contrast of ink on sand ≥ 4.5:1 in both themes.
- Motion respects `prefers-reduced-motion` (draw, settle, blink, breathe all stop).
- Nothing about the layout, hit areas, focus order or ARIA semantics of the existing screens was
  changed; new elements are non-interactive.

---

## 12. Files

```
src/components/Kenar.jsx              the whole system: mark, glyph, wordmark, scenes, companion
src/styles.css                        "Kenar identity system" section at the end
public/brand/kenar-mark*.svg          48 px seal, light + dark
public/brand/kenar-icon.svg|*.png     app icon (512/384/192/180/152/120)
public/brand/favicon.svg|*.png        bold small-size favicon (64/48/32/16)
public/brand/kenar-lockup-*.svg|png   horizontal lockup, light + dark
public/brand/manifest.webmanifest     PWA name, icons, colours
docs/design/kenar-identity-sheet.png  presentation sheet (brand at a glance)
scripts/fixtures/brand-preview.*      isolated, synthetic preview of the identity in the real screens
```

## 13. Rules of thumb

**Do**
- keep the companion low, wide, asymmetric and quiet;
- let the line go past the edge of the frame;
- place the character at the margin, never in the optical centre of a whole screen;
- use the clay dot sparingly (blush, seal, the start of a line);
- let empty space read as calm, not as an unfinished layout.

**Don't**
- add petals, leaves, hearts, stars, sparkles or rainbows;
- give the companion arms, legs, clothes or props;
- animate it continuously near text, or make it talk in more than one quiet line per screen;
- put the mark in a perfect circle bubble (the horizon is the container);
- reintroduce the Ash flower figures, `Flower2` icons or the old floral PNGs.

---

## 14. Why this is Kenar and not "Ash with a new name"

Ash's identity was *botanical*: bloom-headed characters, petals, daisies, a flower pressed in wax.
Kenar's identity is *positional*: a presence at the edge of the page, a horizon, two lines that run
side by side. The character has no species — it is a shape that sits. The name, the illustrations,
the logo, the onboarding story and the copy all describe the same single idea
(**someone stays beside you**), which is why the product reads as designed under this name from
the start rather than renamed.
