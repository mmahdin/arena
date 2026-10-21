# Frontend Architecture

A small, boring, scalable React application. The goal of this phase is a foundation
that can absorb real workflows, more assessment forms and a live API without a
rewrite.

```
src/
├─ app/                 # composition root: providers, router, shell, global store
│  ├─ App.tsx           # routes only
│  └─ store.tsx         # workspace state + actions (context + reducer)
├─ components/
│  ├─ ui/               # design-system primitives (no domain knowledge)
│  ├─ domain/           # reusable, domain-aware compositions (VisitRow, CaseCard…)
│  └─ app/              # shell chrome (BottomNav, TopBar, Page, AppShell)
├─ features/            # one folder per screen / capability
│  ├─ home/ visits/ tasks/ reports/ cases/ calendar/ notifications/ profile/ design/
│  └─ NotFoundPage.tsx
├─ services/
│  └─ api.ts            # the ONLY data boundary (mock today, HTTP tomorrow)
├─ data/                # mock fixtures + demo clock (temporary)
├─ lib/                 # pure helpers: date, labels, selectors, phone, cn, calendar
├─ styles/              # tokens.css, base.css, utilities.css, patterns.css
└─ types/               # domain types = future API contract
```

## Data flow

```
mock fixtures (src/data)
      ↓
services/api.ts  ──►  AppDataProvider (src/app/store.tsx)  ──►  features
      ↑                        │
      └────── optimistic actions (setTaskStatus, saveReport, …)
```

* **Screens never import fixtures.** They read `useWorkspace()` and select with
  helpers from `src/lib/selectors.ts`.
* **All mutations go through `useAppActions()`**; each action updates local state
  immediately and calls the matching `api.*` function (a no-op today, an HTTP call
  later). This is where the real API hooks in — no component changes required.
* **Clock.** Relative labels ("۲ روز مانده") use the server-provided `now`
  (`useNow()`), never `Date.now()` inside components, so mock data and production
  behave identically and tests stay deterministic.

## Replacing mock data with a backend

1. Implement `services/api.ts` methods against real endpoints.
2. Map DTOs → the types in `src/types/index.ts` inside `api.ts` only.
3. Delete `src/data/*` once `loadWorkspace()` is server-backed.
4. Keep the store contract. If a screen needs a new mutation, add one action and one
   API method; do not add fetch calls in components.

## Adding a new screen

1. Create `src/features/<feature>/<Name>Page.tsx` (+ co-located CSS).
2. Compose from `Page` / `PageHeader`, primitives and domain components.
3. Register the route in `src/app/App.tsx`.
4. If it needs a bottom-nav entry, update `BottomNav.tsx` and `NAV_ROUTES`
   (keep 5 destinations max).
5. Screen-only components stay inside the feature folder; promote to
   `components/domain` when a second feature needs them.

## Adding a new assessment form

Forms are *data*, not UI. Extend the report templates:

```ts
// src/features/reports/ReportEditorPage.tsx
const templates: Record<ReportType, { id: string; label: string }[]> = { … }
```

Add a `ReportType` in `src/types`, a label in `src/lib/labels.ts`, and entries in
`templates`. The editor, autosave, progress calculation and detail view render any
template automatically. When forms outgrow a flat section list, replace `templates`
with a JSON schema in `src/data/forms/` and render `ReportEditorPage` from it — the
route, store and detail screen stay unchanged.

## Layout & responsiveness

`AppShell` renders a centered `460px` mobile canvas; on wide screens a quiet context
column appears beside it. Bottom navigation is fixed, centred on the same axis.
No desktop-specific breakpoints exist inside features — safe-area insets are respected
for notched devices.

## Conventions

* **Naming:** components `PascalCase.tsx`, hooks/utilities `camelCase.ts`, CSS classes `kebab-case`.
* **Styling:** CSS files imported next to the component, BEM-ish naming, values only
  from `src/styles/tokens.css`.
* **Imports:** `@/` alias for `src/`; no deep relative chains.
* **Types:** `strict` TypeScript, no `any` in feature code.
* **RTL:** use logical properties (`margin-inline-start`, `inset-inline-end`), never
  `left/right` for layout.
* **Illustrations:** inline SVG only (map preview, empty states) — no raster assets.

## Scripts

| Command | Purpose |
| --- | --- |
| `npm run dev` | Vite dev server on `0.0.0.0:5173` |
| `npm run build` | Type-check + production build to `dist/` |
| `npm run typecheck` | `tsc --noEmit` |
| `npm run preview` | Serve the production build |

Routing uses `HashRouter` so the built SPA works from any static host or file preview.
