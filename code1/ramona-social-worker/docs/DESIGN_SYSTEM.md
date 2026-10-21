# Ramona — Design System (Social Worker App)

This document is the contract for the visual language. Every screen and component
must use the tokens and patterns described here. A living version is available in
the app at **`/#/design`**.

---

## 1. Principles

| Principle | What it means in practice |
| --- | --- |
| **Calm competence** | Warm paper surfaces, one restrained green, generous line-height. The product is used during home visits; it must never feel loud or urgent. |
| **Attention first** | The top of every root screen answers "what needs me now?" before anything else. |
| **One primary action per screen** | Detail screens expose a single sticky primary action; everything else is secondary or in an overflow. |
| **Mobile-native, not shrunk** | Layouts are designed at 390×844 first, use thumb-reachable bottom actions and sheets instead of dialogs. |
| **Status is never colour-only** | Every state pairs colour with text and/or an icon. |
| **Confidentiality is visible** | Restricted-data surfaces carry a short, quiet privacy note. |

---

## 2. Colour

The palette is deliberately small. Green communicates trust and growth; clay is
reserved for *attention* (never for decoration); neutrals are warm so long reading
sessions on reports stay comfortable.

### Brand — botanical green

| Token | Value | Use |
| --- | --- | --- |
| `--brand-25` | `#f4f9f6` | Tinted sections (report link, unread notification) |
| `--brand-50` | `#e9f3ee` | Soft surfaces, icon tiles, active nav pill |
| `--brand-100` | `#d2e8df` | Borders on tinted surfaces |
| `--brand-200` | `#a8d0c1` | Outline badges, rails |
| `--brand-500` | `#17715c` | Progress fills, chart strokes |
| `--brand-600` | `#0f5d4b` | **Primary** — buttons, links, active states |
| `--brand-700` | `#0b4a3b` | Text on soft green |
| `--brand-800` | `#09392e` | Emphasis text |
| `--brand-900` | `#072c24` | Deep contrast |

### Accent — clay `--accent-500 #b85c38`

Only for *"needs attention but not an error"*: today's deadlines, unread dots,
priority "بالا". Never used for large fills.

### Functional tones

| Tone | Solid | Soft bg | Meaning |
| --- | --- | --- | --- |
| success | `--success-500` | `--success-50` | performed / approved / stable |
| warning | `--warning-500` | `--warning-50` | returned for revision / needs follow-up |
| danger | `--danger-500` | `--danger-50` | overdue / critical / destructive |
| info | `--info-500` | `--info-50` | scheduled, neutral-in-progress |
| accent | `--accent-500` | `--accent-50` | time-sensitive |
| neutral | `--neutral-500` | `--neutral-100` | draft, canceled, closed |

### Surfaces & text

| Token | Role |
| --- | --- |
| `--color-canvas` | App background (`#f6f3ec`, warm paper) |
| `--color-surface` | Cards, sheets, inputs |
| `--color-surface-muted` | Nested blocks inside cards |
| `--color-surface-sunken` | Segmented-control track, wells |
| `--color-text` / `-secondary` / `-tertiary` | 3-level hierarchy only |
| `--color-border` / `-strong` | 1px hairlines |

**Contrast rules:** body text ≥ 7:1 on canvas; tertiary text (`--neutral-500`) is
reserved for meta lines ≥ 11px and never for interactive labels.

---

## 3. Typography

Font: **Vazirmatn** (variable, self-hosted in `/public/fonts`). Persian numerals are
used everywhere via `lib/date.ts` helpers (`fa()`, `formatNumber()`).

| Token | Size | Weight | Usage |
| --- | --- | --- | --- |
| `--text-3xl` | 28px | 700 | Page titles (one per screen) |
| `--text-2xl` | 22px | 700 | Detail hero titles |
| `--text-xl` | 18px | 600 | Sheet titles, item titles |
| `--text-lg` | 16px | 600 | Section headers |
| `--text-md` | 14px | 400 | **Body default** |
| `--text-base` | 15px | 400 | Long-form report reading |
| `--text-sm` | 13px | 400/500 | Meta, labels, list subtitles |
| `--text-xs` | 12px | 500 | Badges, timestamps |
| `--text-2xs` | 11px | 600 | Counts inside chips |

Line heights: `--leading-tight 1.4` (titles), `--leading-snug 1.55` (UI copy),
`--leading-normal 1.75` (body), `--leading-relaxed 1.9` (report documents).

Numerals use `font-variant-numeric: tabular-nums` via `.num`.

---

## 4. Spacing, radius, elevation

- **Spacing** — 4px base scale (`--space-1` … `--space-16`). Screen gutter is
  `--app-gutter: 18px`. Section rhythm is `24px` (`--space-6`).
- **Radius** — `xs 8` · `sm 10` · `md 14` · `lg 18` · `xl 22` · `2xl 28` · `full`.
  Cards use `lg`, inputs/buttons `md`, pills `full`.
- **Elevation** — four warm-tinted shadows. Cards are border-first, shadow-second:
  `border: 1px solid var(--color-border)` + `--shadow-xs`. Only floating layers
  (sheets, toasts, nav) use `--shadow-lg` / `--shadow-nav`.
- **Touch targets** — minimum 44×44px (`--tap-min`); icon buttons are 40px with
  44px hit area through padding where space allows.

---

## 5. Components

All primitives live in `src/components/ui`, domain components in
`src/components/domain`. Each has a co-located CSS file and is exported by name.

### UI primitives

| Component | Notes |
| --- | --- |
| `Button` | variants `primary / secondary / soft / ghost / danger`, sizes `sm / md / lg`, `loading`, `fullWidth`, icons. |
| `IconButton` | variants `plain / surface / soft` — always requires `label`. |
| `Card` | `padding`, `tone` (`default / muted / brand / inverse`), `bordered`. |
| `Badge` | `tone` × `variant (soft/outline/solid)`, optional `dot` / `icon`. |
| `Avatar` | sizes `xs–xl`, `circle / squircle`, deterministic tone from name. |
| `Field` + `Input`/`Textarea`/`Select` | label, hint, error, counter; focus ring on wrapper. |
| `Sheet` | bottom sheet for micro-tasks; focus trap, Escape, backdrop, footer slot. |
| `Toast` (`useToast`) | 1–3 stacked, auto-dismiss, optional action (undo). |
| `SegmentedControl` | in-page *navigation between views* (2–4 items). |
| `FilterChips` | in-page *filtering* (scrollable, multi options). |
| `Checkbox`, `Switch`, `ProgressBar`, `Alert`, `EmptyState`, `Skeleton`, `SearchInput`, `SectionHeader`. |

### Domain components

`VisitRow`, `VisitTimeline`, `TaskRow`, `CaseCard`, `ReportCard`, `MonitoringCard`
(+ `CycleDots`), `LocationCard`, `PersonRow` / `InfoRow`, and badge mappers in
`Badges.tsx` (`VisitStatusBadge`, `ReportStatusBadge`, `PriorityBadge`, …).

**Rule:** labels, tones and icons for a domain status are defined **once** in
`src/lib/labels.ts` and consumed through these mappers. Never inline a status colour.

---

## 6. Interaction patterns

| Need | Pattern |
| --- | --- |
| Record a visit outcome | Sticky `action-bar` → `Sheet` with 3 explicit outcomes |
| Complete a task | Inline `Checkbox` in the list + undo `Toast` |
| Filter a list | `FilterChips` directly under the screen title |
| Switch between two related lists | `SegmentedControl` (Tasks ⇄ Reports, Cases ⇄ Monitoring) |
| View a record | Dedicated route with `TopBar`, not a modal |
| Read long content | Report document surface `--leading-relaxed`, justified, sectioned |

Motion: 120ms (micro), 200ms (state), 320ms (entrance). Respect
`prefers-reduced-motion`.

---

## 7. Accessibility checklist

- [x] Landmark structure (`header`, `main`, `nav`) and a skip link.
- [x] Focus-visible ring `--focus-ring` on all interactive elements.
- [x] Icon-only buttons always carry `aria-label`.
- [x] Checkboxes use `role="checkbox"` + `aria-checked`; segmented control uses
      `aria-pressed`; switch uses `role="switch"`.
- [x] Charts/progress expose `role="progressbar"` with values.
- [x] Status conveyed by text + colour, never colour alone.
- [x] Minimum 44px touch targets and 16px+ input text to avoid iOS zoom.
- [x] Persian numerals and RTL logical properties (`inset-inline`, `padding-inline`).

---

## 8. Adding new visual elements

1. Reuse an existing token — if a value is missing, add it to `src/styles/tokens.css`
   first and document it here.
2. Reuse a primitive; only create a new component when the *pattern* repeats.
3. Keep component CSS scoped with a `rm-` or component prefix; no global overrides.
4. Pair every new status with a label in `src/lib/labels.ts`.
5. Update `/#/design` so the fragment gallery stays truthful.
