# Ramona — Social Worker App

A mobile-first, Persian (RTL) workspace for social workers on the Ramona platform:
home hosting and adoption cases. This phase delivers a **reviewable design foundation**:
navigation, visual language, key screens and an extensible frontend architecture.

```bash
npm install
npm run dev      # http://localhost:5173
npm run build    # type-check + production build
```

---

## What is implemented

| Screen | Route | Highlights |
| --- | --- | --- |
| **خانه** | `/` | Greeting with today's date, *نیازمند اقدام* queue, vertical agenda of today's visits, due tasks with inline completion, monitoring snapshot for children in placement. |
| **بازدیدها** | `/visits` | Grouped by day (پیش رو / گذشته), rich rows with time, kind, status. |
| **جزئیات بازدید** | `/visits/:id` | Objective, interactive checklist, address block + illustrated map, guardian contacts, report link, sticky "ثبت نتیجه" → bottom sheet with three outcomes. |
| **کارتابل** | `/tasks` | Segmented **کارها / گزارش‌ها**; tasks grouped as عقب‌افتاده / امروز / پیش رو with type chips, inline completion and undo toast. |
| **جزئیات کار** | `/tasks/:id` | Priority, deadline state, steps checklist, linked visit/report. |
| **گزارش‌ها** | `/tasks?view=reports` | Status-first list (نیازمند اصلاح surfaced first), review feedback preview, draft progress. |
| **مشاهده گزارش** | `/reports/:id` | Supervisor feedback, sectioned document, rating, approval block, sticky edit action when editable. |
| **ویرایش گزارش** | `/reports/:id/edit` | Template-driven sections, autosave, completion threshold, resubmit after revision. |
| **پرونده‌ها** | `/cases` | Search, stage filters, progress + risk; **پایش کودکان** segment with well-being and cycle tracking. |
| **پرونده خانواده** | `/cases/:id` | Stage rail, children (+ monitoring cycles), guardians, address, visits, reports, full history timeline. |
| **تقویم کاری** | `/calendar` | Persian (Jalali) month grid with visit dots, day agenda. |
| **اعلان‌ها** | `/notifications` | Grouped by day, read/unread, deep links into records. |
| **پروفایل** | `/profile` | Professional identity, stats, permissions (granted/locked), contacts, working hours, notification settings. |
| **Design System** | `/design` | Living reference of tokens, colours, type scale and components. |

---

## Design & architecture docs

- [`docs/DESIGN_SYSTEM.md`](docs/DESIGN_SYSTEM.md) — tokens, colour, typography, spacing, component inventory, interaction patterns, accessibility rules.
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — folder structure, data flow, backend swap-in, adding screens/forms, conventions.
- [`docs/SCREENS.md`](docs/SCREENS.md) — screen map, navigation rationale and the UX intent behind each view.

## Key decisions

- **Five bottom destinations only** — خانه، بازدیدها، کارها، پرونده‌ها، پروفایل.
  Reports live inside the work inbox (segment) and monitoring lives inside cases,
  so the navigation stays uncluttered while every capability remains one tap away.
- **Attention before inventory.** The home screen leads with a compact *نیازمند اقدام*
  block (returned reports, overdue tasks, urgent monitoring) instead of a dashboard.
- **Sheets for micro-tasks** (recording a visit outcome, editing contacts) keep context
  on screen; records open as full routes.
- **Warm paper palette with one green.** Clay is reserved for time-sensitive states.
- **Mock data behind a single API seam** (`services/api.ts` + `AppDataProvider`) with
  optimistic updates, so a real backend replaces fixtures without touching screens.

## Stack

React 19 · TypeScript (strict) · Vite 6 · React Router 7 (hash routing) · Lucide icons ·
self-hosted Vazirmatn variable font · plain CSS with design tokens (no UI framework).

## Notes

- All data is fictional and generated relative to "today" so the workspace always
  opens on a plausible working morning.
- Islamic/Iranian week starts on Saturday; dates render in the Jalali calendar via
  `Intl.DateTimeFormat('fa-IR-u-ca-persian')`.
