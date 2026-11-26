# Earthshine — onboarding imagery, 2026-11

**Direction:** *Revelation rather than Illumination.*

**Scope:** imagery and visual storytelling only. Every word of onboarding copy,
every route, every answer, the chapter seam, all interaction logic, the five
conversation styles, privacy copy and the sign-in entry text are unchanged.
`docs/design/earthshine-onboarding-copy-baseline.txt` is a verbatim snapshot of
all 88 Persian strings in `src/screens/Onboarding.jsx` from the base commit;
`scripts/earthshine-art.test.mjs` fails if a single one moves.

---

## 1. The idea

In Greek myth Theia is not merely the goddess of light. Brilliance, preciousness
and the inherent worth of things were attributed to her: gold was gold *because*
of her. Light, in that reading, does not manufacture value. It makes an already
existing value visible.

That is the psychological stance of the product. Theia does not add something
to a person, change who they are, or fix them. It helps them notice aspects of
themselves that have always existed but have remained unseen.

**Earthshine** is the exact natural image of that stance. When the Moon is a thin
crescent, the seemingly dark part of its disc is sometimes faintly visible:
sunlight has struck the Earth, been reflected back into the night, and landed on
the side of the Moon the Sun never reached. The Moon has not changed. Nothing
has been added to it. What was always there has simply become visible through
reflection.

> Through reflection, we begin to see what has always been within us.

The previous onboarding used an astrolabe-like open ring, a central spark,
ticks and orbits, and closed the ring on the last screen. It was geometry about
light. This direction replaces it with the real phenomenon: a moon that is
never altered, only revealed. There are no rings, no orbits, no rays, no
sparks, no decorative diagrams and no human figure anywhere.

## 2. One rule for the whole artwork

**There is one photograph of one moon in every screen.** Only the amount of
reflected light changes.

That constraint is the story. A conventional onboarding would show a different
illustration per chapter — a seed, a sunrise, a completed circle — and would
therefore imply that something is being built or added. Here the same lunar
disc, the same maria and the same slender sunlit crescent appear in chapter ۱
and chapter ۶. The final screen is not a brighter or bigger moon. It is the
same moon, finally legible.

Three plates, cropped from the same composition:

| Plate | Used for | Why |
| --- | --- | --- |
| `earthshine-moon.jpg` | seven of eight states | the crescent, the ashen disc and a warm glow from below the frame |
| `earthshine-reflection.jpg` | `orbits` | the same moon with the dark limb of the Earth at the bottom — the light is shown to have come from somewhere |
| `earthshine-moon-paper.jpg` | `tone="paper"` (privacy, checkins) | the same drawing as an ink-and-wash plate on warm ivory, so it belongs to the two paper screens |

All three are 900×900 JPEGs, ~210 KB combined, kept in `public/brand/earthshine/`.
They are photographic/editorial plates, not screenshots and not icon sets.

## 3. The narrative, chapter by chapter

The chapter names and all copy stay as they were. The imagery now answers them
literally instead of decorating them.

| Chapter | State | What the plate does |
| --- | --- | --- |
| ۱ · آغاز | `welcome` | The crescent is clear, the dark disc only half-legible. On mount, a veil of darkness lifts over ~2.8s: the first act of the product is *revealing*, not lighting. |
| ۱ · حریم | `veil` (paper) | A soft shadow parts from both sides, leaving a narrow opening over the moon. The opening is chosen by the person, not taken. No door, no lock, no curtain illustration. |
| ۲ · شب | `night` | Almost nothing is visible. The disc is still there, dimmed to a ghost; not-knowing is the starting point, not a defect. |
| ۳ · دیدن | `sight` | The earthshine is at reading strength: maria and crater shading become legible. Nothing new has appeared — the eye has adjusted. |
| ۴ · سپیده | `dawn` | A warm reflected glow rises slowly from below the frame over 18s. It never announces itself; it is gradual enough that you barely notice it starting. |
| ۵ · سه نور | `orbits` | The frame lowers to include the dark, barely lit limb of the Earth. Sunlight (off-frame) → Earth → Moon: the three lights are a *relationship*, not three decorative orbs. |
| ۵ · یادآوری | `pulse` (paper) | The same earthlight breathes gently. Returning light, not an expanding ring. |
| ۶ · حلقه | `seal` | The whole disc is quietly visible in the same reflected light. The circle completes because it was always a circle — not because an emblem was closed. |

The sign-in entry that precedes welcome uses the same `welcome` plate, so the
journey begins and ends with one object.

## 4. Implementation

- `src/components/EarthshineInstrument.jsx` — the plate, the eight readings
  (`light` / `saturation` / `warmth` / `shadow`), the Persian accessible
  descriptions and the hairline crescent glyph.
- `src/components/TheiaSigil.jsx` — keeps the small open-ring *product mark*
  used at 11–24px in Home, Letters and chat, and re-exports the instrument under
  its existing name. A logo may be an emblem; a story about seeing must be a
  real thing being seen. The two are deliberately separate.
- `src/screens/Onboarding.jsx` — only the import line and the two tiny marks in
  the chrome / sample reply changed (`TheiaSigil` → `EarthshineGlyph`). All
  strings, including the old `label` props, are byte-identical; the artwork now
  owns its own accessible names and `label` remains a fallback for a future
  state.
- `src/onboarding.css` — the astrolabe keyframes (`ti-drift`, `ti-rays`,
  `ti-pulse`, `ti-sight-ring`) are replaced by an Earthshine surface: a soft
  radial mask that dissolves the plate into ink or paper, a shadow that thins,
  an earthlight that arrives, a parted fold for privacy, and a very light grain.
- `src/features/devsession/DevLogin.jsx` is untouched; it picks up the new
  plate through the same export.

`TheiaInstrument` renders no `<svg>`, no `<path>`, no `<circle>` and no stroke.
It is one plate plus four absolutely-positioned light layers. There is no
JavaScript animation, no canvas, no new dependency and no network request beyond
the three static images.

## 5. Accessibility and motion

- The plate is `role="img"` with a Persian description of what is *happening to
  the light* in that chapter; decorative layers are `aria-hidden`.
- The chrome glyph is `aria-hidden`; chapter progress is still announced once
  and updated politely by the existing seam.
- No text is embedded in the images, so RTL and font loading are unaffected.
- All motion is CSS opacity/filter only and is disabled under
  `prefers-reduced-motion`: the plate then appears immediately at its final
  reading, with the same meaning.
- Paper-tone plates use an ivory veil and a warm wash instead of black shadow,
  so the two light screens do not carry a dark rectangle.

## 6. Verification

- `npm run build` — passes.
- `npm test` — 115 tests: 114 pass, with the single pre-existing
  `foundation.test.mjs` failure (`designer routes and visual assets remain
  present` expects nine files in `public/assets/`, only one was ever committed)
  also present on the base commit.
- `scripts/earthshine-art.test.mjs` — asserts the verbatim copy baseline, the
  three plates, all eight states, the absence of drawn geometry in the
  instrument and the reduced-motion rules.
- Browser/visual QA at 360 / 390 / 430px is **not recorded**: this sandbox has
  no Chromium, so the crops, masks and plate weights were verified by
  rasterising the real assets at the intended sizes. Run
  `npm run preview:theia` and the QA matrix before release.
