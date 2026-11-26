# Task 08 — Daily Reflections, existing Hamrah UI

## Product behavior

- Home's existing **تأمل** card opens the existing illustrated `day` screen instead of Chat; the separate Chat button is unchanged.
- The `day` screen retains its flower/scribble art, navigation back, transcript button and letter card. A small date selector, textarea, Save/Delete controls, and last-30 archive are added *inside* the existing composition.
- The previous hard-coded, personally attributed "today you talked about..." paragraph is removed from that screen: a reflection is shown **only if the user explicitly wrote and saved it**. The letter/Insight surface remains future scope and should not be presented as validated personal analysis.
- One entry per user-selected local calendar date; users can edit and delete, browse recent entries, or reopen a specific day. Drafts are never written to browser storage. Losing an unsaved draft triggers confirmation when leaving that screen.
- No extracted memories, AI-generated reflections, profiling, mood scores, analytics, LLM prompt injection, sharing, or medical inference is part of this feature.

## HTTP and ownership

All routes require the existing signed, revocable session; identity is never supplied in a request body.

| Method | Path | Response | Failure |
|---|---|---|---|
| GET | `/api/v1/me/daily-reflections` | `200 {items: DailyReflection[]}` latest 30 | 401,503 |
| GET | `/api/v1/me/daily-reflections/{date}` | `200 DailyReflection` | 400 invalid date, 404 absent, 401,503 |
| PUT | `/api/v1/me/daily-reflections/{date}` | `200 DailyReflection`, `{text}` body | 400 invalid date/text/unknown fields, 401,503 |
| DELETE | `/api/v1/me/daily-reflections/{date}` | `204` idempotent | 400,401,503 |

`DailyReflection` includes `date`, `text`, `createdAt`, `updatedAt`. The date is an ISO YYYY-MM-DD user-selected **calendar key** and is not converted by the server timezone. All date+owner CRUD goes through the MSSQL repository. Body is trimmed non-empty UTF-8, up to 4000 UTF-16 units (SQL Server NVARCHAR(4000)), without control/format characters except newline, tab, and the Persian/script joiners U+200C and U+200D. No content is logged. Test with synthetic text only.

## Database

Forward-only migration `0012_daily_reflections.sql` creates owner/date primary key, `users` cascade FK and owner/date index. It does not alter applied migrations or existing records. No scheduled/automatic diary generation. `PUT` serializes same-key updates via a serializable SQL transaction. Delete of user cascades through the FK.

## Validation checklist

- `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py`; `cd apps/api && go test ./... -count=1`, `go vet ./...`.
- Real disposable MSSQL: `WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test ./tests/integration -run TestDailyReflectionsSQLServer -count=1` from `apps/api`, with local config.
- `npm test`, `npm run check:foundation`, `npm run build` in frontend.
- Start actual Go/MSSQL with `LLM_PROVIDER=mock` and Vite, then `npm run test:e2e:reflections` (needs Playwright and browser binary); no endpoint interception.
- Verify new user, day selection, text persistence after refresh/relogin, second user isolation, duplicate updates, validation, 401, failed saves retaining draft, and deleting entries. Mobile screens 360/390/430, keyboard, scrolling, RTL and reduced motion. Rerun E2E session/onboarding/chat/history/bookmarks/tones/transition.

**Scope:** internal-only with synthetic content. No public authorization, clinical validation, or privacy approval for personal reflections. Real personal diary content must not be used with this development authentication.

## Persian date field

The day screen now uses a JSX/CSS port of `ramona-platform-ui-family/src/design-system/components/JalaliDateField.tsx`, with its date helpers and `jalaali-js` conversion. It retains Persian/Arabic digit entry, partial segments, leap-year validation, month navigation, 12-year selection pages and min/max dates. The dialog uses a native modal with trapped keyboard focus, Escape/backdrop close and trigger focus restoration, styled to Hamrah's existing palette.

The UI holds an editable Jalali string separately from the selected Gregorian calendar key. Only a complete, valid date up to today changes the API key; incomplete/invalid dates cannot submit the reflection form. Switching dates retains the existing unsaved-draft confirmation. No backend, schema or timezone-instant conversion is involved.

Validation: `npm test` (11 files), foundation 41/41, production build, `test:e2e:calendar` at 360/390/430/1280px, and connected `test:e2e:reflections` all passed. Browser coverage includes year/month navigation, Persian typed dates, invalid dates staying off the API, cancellation preserving the draft, modal focus and reduced motion. Screenshots are generated under ignored `test-results/jalali/`.
