# Theia identity — 2026-11-07

## Decision
The product is now **Theia / تیا**. This is a scoped visual-identity update, not a UI redesign. The existing Persian RTL screens, route order, component geometry, typography, spacing, navigation, interaction model and backend boundaries remain authoritative.

## Brand idea
Theia is interpreted through **light, reflection and clarity**, not literal mythology. The original mark is two reflective halves opened by a single thread of light. It avoids suns, eyes, Greek columns, laurel wreaths and floral mascots.

- Primary surfaces stay the existing neutrals: `--light`, `--ink`, `--beige`, `--dark`.
- Brand accent: `--theia-gold: #d7b98a`; glow: `--theia-glow: #f4e3bd`.
- The gold is used for brand marks, seals, unread-letter light and small ambient accents. Functional selection/focus colors are not broadly repainted.
- The Persian wordmark uses the existing local typography in product UI. Standalone outlined lockups are in `public/brand/`.

## Onboarding narrative
A small point of light moves from first encounter to clarity:

1. `welcome.svg` — two quiet forms meet; a point of light appears between them.
2. `privacy.svg` — the light rests inside protective frames.
3. `simple-intro.svg` — the light approaches its first narrow opening.
4. `deeper-intro.svg` — the light meets a mirror and finds a soft reflection.
5. `know-intro.svg` — two curved mirrors hold the light; an aperture for understanding.
6. `almost-intro.svg` — the light rises through a narrow path toward an open horizon.
7. `checkins.svg` — a gentle reminder returns as soft waves and a small letter.
8. `ready.svg` — the light arrives at the Theia mark; reflection becomes clarity.

The illustrations are original, resolution-independent SVGs. They reuse the existing artwork containers and object-fit behavior; no screen layout was rebuilt.

## Assets
- `public/brand/theia-mark*.svg` — primary, light and monochrome marks.
- `public/brand/theia-lockup-*.svg` — Persian/English lockups.
- `public/brand/theia-app-icon*.svg` — standard and maskable icon sources.
- `public/icons/` — generated PNG/favicon/apple-touch icons (`npm run brand:icons`).
- `public/assets/theia/` — onboarding narrative artwork.
- `src/components/TheiaMark.jsx` — small inline brand mark replacing `Flower2`.

Legacy floral assets remain on disk only for the historical source-preservation manifest; active screens no longer reference them.

## Compatibility
Internal protocol/storage identifiers such as `hamrah-v2`, `hamrah:letter-seen:v1:` and `HAMRAH_LETTER_CONTEXT_V1` are intentionally unchanged so existing profiles, progress and letters remain readable. The rebrand changes presentation and user-facing brand copy, not data contracts.

## Preview and QA
- `npm run preview:theia` serves an isolated design fixture using the real screens and local synthetic state. It makes no API calls.
- `npm run test:e2e:brand` checks RTL, horizontal overflow, broken/legacy artwork and runtime errors at 360/390/430px and desktop across 19 screens.
- Connected Go/MSSQL E2E remains separate and requires the real local backend.
