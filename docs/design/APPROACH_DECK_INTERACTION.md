# Conversation-style picker — deck redesign (2026-11-05)

## Problem
`approach-screen` stacked five chips, a badge, a user bubble, an author label, a
clamped response, a swipe hint and a long CTA. Choosing a style only swapped the
text inside one static card (`--approach-drag` was clamped to ±48px, and the
content re-mounted with a 260ms fade), so a swipe read as replacement rather
than travel, and the screen as a pile of controls.

## Interaction now
`ApproachScreen` in `src/screens/Onboarding.jsx` renders all five styles on one
track (`.approach-track`) inside a clipped stage. The track is translated by a
single value, `--approach-x`, and nothing else carries layout state.

- **The deck follows the finger.** Pointer drag (touch, pen and mouse) moves the
  track 1:1, with rubber-band resistance past the first and last card. A drag is
  only claimed once the gesture is clearly horizontal, so a tap still selects.
- **Velocity-aware settle.** On release the deck eases to a target in a
  frame-rate independent loop (~250 ms, no overshoot). A flick carries exactly
  one style past the one the drag started from and never rolls back a longer
  drag; a slower release settles on the nearest card.
- **Depth painted from position.** Every card derives `--approach-focus`,
  `--approach-scale`, `--approach-opacity` and `--approach-lift` from its live
  distance to the centre, so neighbours peek in at ~30–50px, recede and lift,
  and the incoming card's response fades up as it arrives.
- **Meaningful reveal.** The centred card brightens to a light surface with a
  soft shadow; its sample reply re-mounts once and rises in over 300 ms, and a
  brief haptic tick (`navigator.vibrate(8)`) marks the settle where supported.
- **Live selection.** While the finger is down the nearest card drives
  `aria-checked`, the dots and the CTA label, and writes the answer only when it
  changes. Deliberate jumps (dots, keyboard, wheel) animate straight to their
  target without flickering through the styles they pass.

## Layout, less clutter
Header (question + one line of expectation) → one quiet shared-prompt line →
the deck → dots → CTA. The five chips, the card badge, the user bubble, the
author label and the swipe hint are gone.

The five styles all answer the same sample prompt, so it is shown **once** above
the deck (`approach-context`, labelled «پیام نمونه») instead of being repeated
inside five cards; each card shows the style name, its one-line personality
(`response`) and the reply (`exampleResponse`) labelled «پاسخ».

## Accessibility
- `role="radiogroup"` + `aria-orientation="horizontal"`, cards are `role="radio"`
  with `aria-checked`, roving `tabIndex` and `aria-labelledby` to the style name.
- RTL keyboard order: `ArrowLeft` next, `ArrowRight` previous, `Home`/`End` jump;
  focus follows the selection. Wheel/trackpad steps one style with a 240ms
  cooldown. Dots remain the labelled fallback (`PagerDots`), with an enlarged
  hit area.
- `aria-describedby` announces the drag/keyboard affordance and a polite live
  region announces the settled style.
- `prefers-reduced-motion`: selection becomes instant (`behavior` is bypassed in
  JS), and card/sample transitions and animations are disabled.

## Deliberately unchanged
- `answers.approach` still stores the style **name**, and the final save still
  sends `onSaveApproach(approach.name)` — no new backend field.
- The `profileEdit` path (from the menu) keeps its Save/Cancel buttons and its
  "saving" state; only the description copy differs.
- Palette, type scale, radius and the beige theme are existing tokens.

## Responsive notes
Cards are `min(100% - 84px, 300px)` wide (peek ≈ 31–51px) and capped at 392px
tall. Below 740px height the type tightens and the sample clamps to three lines;
below 660px height the sample is dropped so name + personality line always fit;
below 374px width the cards narrow and the sample type shrinks.

## Verification in this environment
- `npm run build`: PASSED.
- `npm test`: PASSED (91/91), including the suites that pin the five styles and
  the shared sample prompt.
- `npm run check:foundation`: FAILED for a pre-existing reason — the checkout has
  no `.agents/skills` directory (`ENOENT`), unrelated to this change.
- Browser/Playwright: NOT RUN — no browser binary is available in the sandbox and
  the download host is blocked. The deck was exercised against a jsdom render
  with mocked layout metrics (mount centring, drag follow, flick settle, tap,
  dots, wheel, RTL keyboard order, vertical-gesture rejection, ARIA state).
- `scripts/e2e-approach-transitions.mjs` was already stale before this change
  (it asserts `.approach-tab`, `aria-current` dots, prev/next buttons and a
  `.approach-response` string that the current source does not render). It needs
  a rewrite for the deck and was **not** updated or run here.
- `npm run check:source-baseline`: FAILED by design — it compares against the
  recorded original sources, and these files are the intentional edit.
- Deliverable: `approach-screen-redesign.patch`, verified with `git apply
  --check` and then applied to a clean worktree of the base commit (`ff86bbb`),
  where `npm run build` and `npm test` (91/91) both pass on the patch alone.
