# Screen map & UX rationale

Why the app is shaped the way it is, screen by screen. Use this as the brief when
extending the product.

## Navigation model

Five bottom destinations — **خانه · بازدیدها · کارها · پروندهها · پروفایل** — plus
contextual routes reached from records. Grouping decisions:

| Capability | Where it lives | Why |
| --- | --- | --- |
| Tasks · Reports | **کارها** with two segments (کارها / گزارشها) | A report is a unit of work that arrives with a deadline; splitting them into separate destinations fragments the inbox. |
| Visits · Calendar | **بازدیدها** + calendar icon in the header, calendar also reachable from Home | Visits are the primary daily object; the month view is a support tool, not a fifth tab. |
| Cases · Monitoring | **پروندهها** with two segments (پروندهها / پایش کودکان) | Monitoring is a lens on case files, not separate data. It still gets a first-class entry because it carries clinical risk. |
| Notifications | Bell in the Home header | Badge-driven, checked often, never the starting point. |
| Profile · Design system | **پروفایل** | Account, permissions and product meta. |

## Home (`/`)

1. **Greeting + date** — orientation in one glance, Jalali date.
2. **نیازمند اقدام** — merged, ranked queue of returned reports, overdue tasks and
   children needing attention. This is the only element that uses the clay accent.
3. **برنامه امروز** — vertical timeline with time rail; the next visit is emphasised.
4. **کارهای امروز** — at most three open tasks, inline completion with undo.
5. **پایش کودکان** — up to two monitoring cards with cycle progress.

Nothing else. No charts, no vanity metrics — the screen answers *what now?*.

## Visits (`/visits`, `/visits/:id`)

- List grouped by day with sticky-feeling day headers («امروز» highlighted).
- Detail: purpose first, then the checklist (checkable on site), then coordination
  notes, address with an illustrated map, contacts, linked report.
- Recording an outcome is a **bottom sheet** with three explicit results; it keeps
  the visit on screen and avoids a separate form page.
- Canceled visits shift the sticky primary action to «زمانبندی مجدد».

## Work inbox (`/tasks`, `/tasks/:id`, `/reports/*`)

- Segments: **کارها** / **گزارشها**, sharing the same header and counts.
- Tasks are grouped **عقبافتاده → امروز → پیش رو**; filters (فوری، بازدید، گزارش…)
  sit under the title.
- Reports are ordered by *status urgency*: نیازمند اصلاح → پیشنویس → در انتظار بررسی →
  تأییدشده. Returned reports surface supervisor feedback inline.
- Editor: template-driven sections, character counters, autosave, 60% completion
  threshold before submission, revision banner with the supervisor's comment.

## Cases (`/cases`, `/cases/:id`)

- Searchable list with stage chips, progress and risk.
- Detail tabs: **خلاصه · بازدیدها · گزارشها · تاریخچه** with a four-step stage rail.
- Children with monitoring cycles are rendered inside the summary, so the case
  reads as one story rather than separate modules.

## Calendar (`/calendar`)

Persian month grid (Saturday-first) with dot indicators, selection and a same-screen
agenda for the selected day. No drag-and-drop in this phase — appointments are
assigned by supervisors.

## Notifications (`/notifications`)

Grouped by day, unread first-class (tinted surface + dot), deep links to the record,
bulk «خواندن همه».

## Profile (`/profile`)

Identity and stats, supervisor, **permissions with explicit granted/locked states**
(confidentiality made legible), contacts, working hours, notification settings, and a
link to the living design system.

## Empty, loading and error states

- Loading: `PageSkeleton` while the workspace hydrates.
- Empty: singular `EmptyState` with a concrete next step where one exists.
- Errors are navigational (record not found → back to list) — a foundation-phase
  convention; real error surfaces should reuse `Alert` + `EmptyState`.
