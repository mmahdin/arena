# Source preservation and design authority

- Source of truth: the user-uploaded `ai-therapist-frontend(1).zip` — the **کنار / Kenar** UI.
  It descends from the original همراه upload, and the 2026-11-08 rebrand replaced that
  product's Ash-inspired floral artwork and brand name while keeping its structure.
- **Preserved by policy:** the component tree, layouts, navigation, route order,
  spacing, hit areas, interaction behaviour, typography and all demo interactions.
  `FRONTEND_DESIGN_SPEC.md` describes the implementation as it exists rather than
  introducing a new look.
- **Intentionally changed (brand work only):** illustration assets, colour tokens,
  the logo/app-icon/mark components, decorative glyphs, the product name and the
  copy that named the old brand. Full rationale and file-by-file map:
  `docs/design/KENAR_VISUAL_IDENTITY.md`.
- Run `npm run check:source-baseline` to verify the current committed bytes against
  `docs/design/original-source-sha256.json`. That manifest tracks the reviewed state
  of every source file and every shipped brand asset, so drift is always explicit.
- `src/styles.css` is still the styling source of truth. The identity changes live in
  its `:root` token block and the single appended `KENAR IDENTITY` section; no existing
  geometry rule was rewritten.
- The previous Rahrah screenshot set has **not** been imported; it is a different
  design baseline. No older theme should override this source.
