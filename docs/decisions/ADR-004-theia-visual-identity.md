# ADR-004: Theia visual identity without UI redesign

- **Status:** Accepted; 2026-11-07.
- **Decision:** Rename the product to **Theia / تیا** and replace Ash-inspired floral brand art with an original light/reflection identity. Preserve the existing JSX/CSS screen geometry, Persian RTL interface, route flow, typography, spacing, navigation and functionality.
- **Authority:** `docs/design/THEIA_IDENTITY_2026_11.md` supersedes only the floral/Hamrah brand-art portion of earlier design authority. ADR-001 still protects the underlying screen design and incremental integration rules.
- **Compatibility:** Do not rename persisted/protocol keys (`hamrah-v2`, letter context/storage prefixes) as part of this visual change.
- **Consequence:** Future agents must not restore floral mascots or treat the historical Hamrah name as the active product brand.
