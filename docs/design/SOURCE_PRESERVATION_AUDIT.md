> **2026-11-07 update:** the user-approved Theia rebrand intentionally changes active brand copy/artwork and small brand styling. The original SHA-256 manifest remains historical; `check:source-baseline` is expected to differ for the rebranded files. Screen geometry and route flow are still preserved. See `THEIA_IDENTITY_2026_11.md`.

# Source preservation and design authority

- Source of truth: user-uploaded `ai-therapist-frontend(1).zip` (a new **همراه** UI, not the earlier رهراه gallery or TypeScript foundation).
- This addition **does not modify** `src/**`, `public/**` or `index.html`; the UI keeps its component tree, custom CSS, illustration assets, route order, typography, images and demo interactions.
- Only developer tooling/configuration can change (Vite host/proxy, npm check scripts, docs/tests). No visual or interactive feature was implemented by the repackaging.
- Run `npm run check:source-baseline` to verify byte-identical original sources against `docs/design/original-source-sha256.json`.
- `src/styles.css` is the styling source of truth. `FRONTEND_DESIGN_SPEC.md` describes the implementation as it exists rather than introducing a new look.
- The previous Rahrah screenshot set has **not** been imported; it is a different design baseline. No older theme should override this source.
