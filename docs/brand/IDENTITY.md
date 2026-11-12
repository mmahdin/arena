# Theia (تیا) — The Aperture of First Light

**Status:** active visual identity · 2026-11-12
**Scope:** logo, app icon, brand marks, onboarding illustrations, home and letter
emblems. No product flow, layout, typography, copy or API behaviour changed.
**Board:** `docs/brand/theia-identity-board.png`
**Plates:** `docs/brand/plate-welcome.png` · `docs/brand/plate-ready.png` ·
`docs/brand/plate-pulse.png`
**Logo:** `docs/brand/logo-lockup.png`

---

## 1. The idea

Theia was previously represented by a woman holding a sphere of light. That
figure is retired. The brand now carries the same idea — sight, light, dawn,
discovery — without a character.

The identity is a **single object seen at different scales: an aperture.**

An aperture is the opening through which light enters (an eye, a lens, a
window, the gap between two curtains). It is also the opening through which a
person is seen. In this product, Theia is not someone who looks at you; Theia is
**the door left ajar, the window you look through.**

> «دریچه‌ای رو به آسمانِ درون»
> *An aperture onto the inner sky.*

Three elements carry the whole system:

| Element | Form | Meaning |
| --- | --- | --- |
| **The aperture** | A fine ring, always left open | A safe opening; it is never a closed target or an eye that watches |
| **The horizon** | A single hairline, measured with three or four ticks | The place where thought becomes visible; the pause before clarity |
| **First light** | One warm point, never centred | Attention, dawn, the beginning of seeing |

Nothing else. No figure, no crescent-and-star, no rays, no ornaments, no
generic AI gradients.

---

## 2. The mark

`TheiaMark` (`src/components/TheiaBrand.jsx`, also `public/brand/theia-icon.svg`)

- A ring opened by **58°** at the top, with rounded terminals.
- A **horizon** chord in the lower third.
- A **sun** of warm first light resting on the horizon, offset left of centre.
- A **brass arc** on the upper-left rim, as if light catches the edge of the
  aperture.

At 64 px it reads as a quiet window at dawn. At 32 px the horizon and highlight
remain. At 18–21 px (chat empty state, approach card, sign-in) the mark
simplifies to ring + light so it never turns into mush. A `mono` variant keeps
the wax-seal mark legible on the brass seal.

**Wordmark.** «تیا» in Noto Naskh Arabic, with `THEIA` letterspaced in Estedad.
The bilingual lockup is in `public/brand/theia-logo.svg`. The wordmark is
unchanged in typeface: the identity was re-imagined, not re-typeset.

**App icon and favicon.** The same aperture on warm near-black, full-bleed with
all geometry inside the PWA maskable safe zone.

---

## 3. Palette

The product's tokens are preserved. The identity adds a deeper brass for light
on paper, so gold never disappears on warm paper or beige.

| Token | Hex | Use |
| --- | --- | --- |
| Night | `#12110d` | Home, night plates, dark surfaces |
| Ink | `#23201b` | Type and hairlines on paper |
| Paper | `#f3f2ee` | Onboarding surfaces |
| Beige | `#cec4b8` | Check-ins panel, quiet fills |
| Brass | `#c99a5f` | Light on paper, seals, small marks |
| First light | `#f6cf98` | Light on night plates, app icon |
| Moon | `#d5d9f7` | The second, cooler light; stars |

Rules: one warm light per plate. No gradients as decoration; gradients only
describe light itself. Lavender is a whisper, never a second brand colour.

---

## 4. Typography (unchanged)

- UI: **Estedad Variable**, Persian. Secondary: Vazirmatn, Noto Naskh Arabic.
- Wordmark: **Noto Naskh Arabic**.
- No new weights, sizes or letter-spacing were introduced into the product UI.

---

## 5. The onboarding story

Eight plates, one continuous movement: **the aperture opens.** The ring's gap
widens from plate to plate; the light moves from a point to a horizon to a
risen sun. The user is not shown a finished picture at the door; the screens
tell the story in order.

| # | Screen | Plate | Persian | What happens | Ring gap |
| --- | --- | --- | --- | --- | --- |
| I | Welcome | The Window | دریچه | A night sky held in an aperture: a sliver of dawn on the horizon and one morning star. The promise, not the answer. | 26° |
| II | Simple intro | The Point | نقطه | In an empty night, one hairline finds a single point of light. Beginning. | — |
| III | Deeper intro | The Opening | گشودن | The aperture is visibly open; light gathers inside. | 70° |
| IV | Know intro | The Orbit | مدار | Orbits cross the aperture; light has a path now. | 90° |
| V | Almost intro | The Sky | آسمان | The frame becomes sky: a wide ellipse, many stars, a light near the horizon. | — |
| VI | Privacy | The Seal | مُهر | The aperture closes. The light is kept whole inside the seal. Privacy as protection, not surveillance. | closed |
| VII | Check-ins | The Pulse | نبض | Three slow ripples leave a drop of light — a reminder, never an alarm. | open |
| VIII | Ready | First Light | سپیده | The aperture is wide open and the sun has risen. | 106° |

Supporting art:

- **Home** — «دریچهٔ زنده» *The Living Aperture*: the seal at rest, with two
  slow orbits and a breathing sun. It replaces the old goddess medallion in the
  same DOM slot and keeps the same halo, label and hover behaviour.
- **Letters** — the envelope remains, signed with a small aperture and a sun.
- **Sign-in** — a smaller window with the same horizon.
- **Wax seal** — the aperture in monochrome on brass.

All illustrations are inline SVG (`TheiaWelcomeArt`, `TheiaPointArt`,
`TheiaOpeningArt`, `TheiaOrbitArt`, `TheiaSkyArt`, `TheiaPrivacyArt`,
`TheiaPulseArt`, `TheiaReadyArt`, `TheiaLoginArt`, `TheiaHomeArt`). They scale,
theme and animate without raster assets, and each carries its own unique
gradient ids so multiple plates can coexist on a page.

---

## 6. Motion

- Orbits turn once every **180 seconds**. The sky is not in a hurry.
- The sun breathes over 7 seconds; stars twinkle on staggered 6.5 second cycles.
- Check-in pulses travel outward on a 5.4 second cycle.
- All motion is CSS, GPU-friendly, and **completely disabled** under
  `prefers-reduced-motion: reduce`.
- Nothing flashes, bounces, or draws attention away from text.

---

## 7. Asset and component map

| Asset | Path |
| --- | --- |
| Logo lockup | `public/brand/theia-logo.svg` |
| App icon (source) | `public/brand/theia-icon.svg` |
| App icons (PNG) | `public/brand/theia-icon-192.png`, `theia-icon-512.png` |
| Favicon | `public/favicon.svg` |
| Apple touch icon | `public/apple-touch-icon.png` |
| Manifest | `public/manifest.webmanifest` |
| Components | `src/components/TheiaBrand.jsx` |
| Tokens, motion, art sizing | `src/styles.css` § *Theia visual identity system* |
| Identity board | `docs/brand/theia-identity-board.png` |

Old character PNGs (`theia-welcome.png`, `theia-story-1…4.png`,
`theia-privacy.png`, `theia-checkins.png`, `theia-ready.png`) were removed.
They must not be reinstated.

---

## 8. Constraints and preservation

- Onboarding structure, screen order, CSS layout, type scale, colours of the
  interface, and all copy outside the illustrations are unchanged.
- The illustration slots, aspect ratios and responsive behaviour are the
  existing ones; the SVG uses the same wrappers (`welcome-art`,
  `interstitial-art`, `privacy-art`, `checkins-art`, `ready-art`,
  `dev-login-art`).
- Every meaningful illustration has an accessible Persian label; purely
  atmospheric layers are `aria-hidden`.
- `aria-hidden`/focus behaviour of the decorative mark is unchanged.
- The identity is warm to dark surfaces and paper alike; it does not rely on a
  single background.

## 9. Validation

- `npm run build` — passed.
- `npm test` — 114/116 passed. The two failures are pre-existing in this
  checkout and unrelated to the identity: the stale `public/assets` file count
  (1 file present, test expects 9) and a missing voice-disclaimer string in the
  current onboarding source. Neither was introduced by this change.
- Browser-level screenshot QA could not run in this sandbox (no Chromium and no
  browser download access). The plates were rendered at their real aspect
  ratios from the actual components for review; live verification remains
  required on iOS Safari and Android Chrome at 360 / 390 / 430 px.
