# Source preservation and design authority

> **2026-11-08 — intentional divergence.** The Kenar identity pass changes
> `src/**`, `public/assets/**` and `index.html` on purpose: the Ash-inspired
> floral identity was replaced. `npm run check:source-baseline` therefore no
> longer matches and the floral entries in `original-source-sha256.json` are kept
> only as history. Current visual authority is `docs/design/KENAR_IDENTITY.md`.

- Source of truth: user-uploaded `ai-therapist-frontend(1).zip` (a new **همراه** UI, not the earlier رهراه gallery or TypeScript foundation).
- This addition **does not modify** `src/**`, `public/**` or `index.html`; the UI keeps its component tree, custom CSS, illustration assets, route order, typography, images and demo interactions.
- Only developer tooling/configuration can change (Vite host/proxy, npm check scripts, docs/tests). No visual or interactive feature was implemented by the repackaging.
- Run `npm run check:source-baseline` to verify byte-identical original sources against `docs/design/original-source-sha256.json`.
- `src/styles.css` is the styling source of truth. `FRONTEND_DESIGN_SPEC.md` describes the implementation as it exists rather than introducing a new look.
- The previous Rahrah screenshot set has **not** been imported; it is a different design baseline. No older theme should override this source.
