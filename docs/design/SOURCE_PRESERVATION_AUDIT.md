# Source preservation and design authority

## 2026-11-07 — Theia identity baseline (current)

- The product is now branded **Theia / تیا**. The identity, design system, illustration language and every screen were rebuilt for this brand; the previous floral (Ash-inspired) artwork is intentionally removed and must not return.
- `docs/design/original-source-sha256.json` now pins the **Theia** identity surface: entry HTML, icon set, manifest, design tokens (`src/styles.css`), the brand primitives (`src/components/Brand.jsx`) and the screens that compose them.
- Run `npm run check:source-baseline` to confirm those files are present; `npm run check:source-baseline -- --verify-original` compares their exact bytes.
- `src/styles.css` remains the single styling source of truth and `FRONTEND_DESIGN_SPEC.md` remains the interaction contract. `docs/design/THEIA_IDENTITY.md` documents the new visual system, tokens and component vocabulary.
- Behaviour, routes, backend integration and the onboarding question set are preserved; only presentation, brand language and the art direction changed.
- The earlier uploaded `ai-therapist-frontend(1).zip` (همراه) and the Rahrah gallery are historical references. No part of their floral iconography should be reintroduced.

## Historical note (pre-rebrand)

- Source was the user-uploaded `ai-therapist-frontend(1).zip` (the **همراه** UI). The repackaging step did not modify `src/**`, `public/**` or `index.html`; only developer tooling changed.
- That preservation guarantee applied to the repackaging task only and has been superseded by the 2026-11-07 rebrand above.
