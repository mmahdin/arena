# Repo map — Kenar designer source

| Source | Actual behavior in the uploaded ZIP | Future integration owner |
| --- | --- | --- |
| `src/main.jsx` | React StrictMode bootstrap, locally loaded Persian fonts, CSS | Do not break |
| `src/App.jsx` | Hash `#/route` router, local name/answers/messages, toast, reset | Add session bootstrap + adapter orchestration carefully |
| `src/screens/Onboarding.jsx` | Welcome/Reviews/Research/Experts/Name/Privacy/Questions/Age/Voice/Approach/Ready | Keep 25-step visual design; gate unsupported intake, map name/approach to profile |
| `src/screens/Main.jsx` | Home/Survey/Day/History/Menu/Chat/Insight; fixed delayed chat reply | Replace mock send, history and user actions incrementally |
| `src/components/UI.jsx` | Original buttons, icons, envelope artwork, lettering, wax seal | Reuse existing primitives |
| `src/components/Kenar.jsx` | Kenar identity system: character, horizon/echo, scenes, mark, glyph, wordmark, companion | Do not add floral or Ash-era symbols; extend here |
| `src/data.js` | `FLOW`, question/voice/approach options, reviews and sample messages | DEMO data, not verified facts or server truth |
| `src/styles.css` | Responsive mobile shell, theme/color tokens, visual treatment | Source of truth for visual system |
| `public/assets/` | Legacy Ash-era floral PNGs (now unreferenced) + 2 expert portraits | Keep as design artifacts; portraits still used by the Experts screen; do not re-reference the florals |
| `public/brand/` | Kenar mark, app icon set, favicons, lockups, web manifest | Regenerate from `src/components/Kenar.jsx` if the drawing changes |
| `docs/design/KENAR_VISUAL_IDENTITY.md` | Brand identity authority: concept, character, mark, palette additions, motion, voice, application map | Read before any visual change |
| `vite.config.js` | React/Vite dev/preview config, localhost API proxy (handoff addition) | Do not expose local API to LAN |
| `contracts/openapi/` | Task07-derived backend contract snapshot | Sync from actual server before wire-up |
| `docs/integration/` | Exact UI/API gaps and integration behavior | Consult before new endpoint work |
| `.agents/skills/` | 7 locally scoped coding-agent workflows | Read by matching feature |
| `scripts/check-foundation.mjs` | No-dependencies static audit | Not a substitute for browser/backend tests |
| `docs/design/original-source-sha256.json` | SHA-256 manifest of original UI source assets | Verify no unintended loss |
| `scripts/brand-preview.mjs` | Backend-free identity gallery (real screens, synthetic props) | `npm run preview:brand` |

## Source screens and routes
Onboarding order: `welcome → testimonials → research → experts → name → simple-intro → privacy → deeper-intro → therapist → age → concerns → focus → feelings → neurodivergent → faith → know-intro → interests → relationships → strengths → barriers → almost-intro → voice → approach → ready → home`.

Secondary routes: `survey`, `day`, `history`, `menu`, `chat`, `insight`. Hash router uses no login guard yet; history and insights contain hardcoded examples; outgoing replies are hardcoded in a `setTimeout` in `Main.jsx`.
