# Age range picker — interaction redesign (2026-11-05)

## Problem
`age-screen` positioned each range with precomputed pixel offsets
(`--age-shift`, `[0, 58, 96, …]`) and changed `height`, `font-size` and
`background` on the selected row. Nothing was interpolated, so every swipe,
wheel notch or touch-end moved the whole column in one frame. The result read as
a list that re-stacks, not as a control that moves.

## Interaction now
`AgeScreen` in `src/screens/Onboarding.jsx` is one native scroll container
(`.age-wheel`) with fixed 58px rows, `scroll-snap-type: y mandatory`,
`scroll-snap-align: center` and half-viewport padding on both ends.

- **Input is the platform's own.** Touch momentum, iOS rubber-banding,
  trackpad and mouse-wheel all scroll the container directly; snap points land
  every gesture on exactly one range.
- **Mouse drag** is layered on top for desktop (`pointerType === 'mouse'`):
  snap and `scroll-behavior` are suspended for the duration of the drag so the
  wheel tracks the pointer 1:1, then released to settle.
- **Continuous painting.** A `requestAnimationFrame` loop reads each row's
  distance from the centre and writes only custom properties —
  `--age-scale`, `--age-fade`, `--age-tilt`. No layout, no font-size or height
  changes, so motion cannot tear.
- **Deterministic geometry.** Row height is fixed; the end padding is measured
  (`--age-pad`) instead of relying on percentage heights inside a scroll
  container; `ResizeObserver` re-centres when the viewport, keyboard or
  orientation changes.
- **Selection semantics.** The nearest row drives `aria-checked`, the ink
  colour and a `role="radio"` roving tabindex; `setAnswer('age', …)` is written
  once the wheel settles (130 ms idle), not per frame. The visual default
  remains a preview until the user actually touches the wheel, as before.

## Feedback
- The beige marker stays fixed at the centre (same 24px gutters, 58px height,
  17px radius and `--beige` fill as the previous selected row) and gives a
  2.6% scale pulse whenever a new range passes it. It is re-keyed, so the pulse
  restarts per detent.
- `navigator.vibrate(8)` fires per detent where the platform honours it
  (best-effort, guarded).
- Edge rows fade out through a mask and drop in scale/opacity, so the wheel has
  no hard cut at the top or bottom.

## Accessibility
- `role="radiogroup"` + `aria-orientation="vertical"`, `role="radio"` children
  with `aria-checked`, roving `tabIndex`.
- ↑/↓ move one range, PageUp/PageDown two, Home/End jump to the ends; focus
  follows selection with `preventScroll`, then the wheel animates to it.
- A polite live region announces the settled range.
- `prefers-reduced-motion`: scrolls become instant (`scroll-behavior: auto` via
  the existing global rule, `behavior: 'auto'` in `scrollTo`) and the tilt/scale
  transforms and marker pulse are disabled; snapping and the mask remain.

## Deliberately unchanged
- Sensitive intake data still stays in React memory only — nothing new is sent
  to the backend, and the answer is only committed on real interaction.
- Colours, type scale, gutters, radius and the light theme are the existing
  design tokens; no new palette or asset was introduced.

## Verification in this environment
- `npm run build`: PASSED.
- `npm test`: PASSED (91/91).
- `npm run check:foundation`: FAILED for a pre-existing reason — the checkout
  has no `.agents/skills` directory (`ENOENT`), unrelated to this change.
- Browser/Playwright verification: NOT RUN — no Chromium/Firefox binary is
  available in the sandbox and the browser download host is blocked. Component
  logic (mount centring, snap settle, keyboard, tap, drag suppression) was
  exercised against a jsdom render with mocked layout metrics.
