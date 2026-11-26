# ADR-001: new Hamrah designer source of truth

- **Status:** Accepted for this frontend handoff; 2026-11-02.
- **Decision:** `ai-therapist-frontend(1).zip` is the UI/product-design authority. Keep its React19 + JSX + Vite6 + custom CSS structure, visual assets and route intent. Previous Rahrah designer frontend, former TypeScript foundation and historical docs are references only and must not overwrite it.
- **Consequence:** Go/MSSQL Backend remains technical truth for supported API, but visual features lacking backend require deliberate new contract decisions, not hidden fake calls.
- **Scope distinction:** internal synthetic MVP can ship selected flows first without deleting other designed UI routes; clinical/privacy/research statements require verification or gating.
