# Conversation titles — frontend validation

- PASS: `npm test` — 64 Node tests passed (including API and state isolation for rename).
- PASS: `npm run check:foundation` — 41/41.
- PASS: `node --check scripts/e2e-conversation-titles.mjs`.
- BLOCKED: `npm run build` — Vite not installed in this sandbox.
- NOT RUN: Playwright E2E with live Vite/Go/MSSQL, screenshot comparison at 360/390/430px.
- Existing Home/Chat/History navigation, fonts, assets, bookmark section and delete action preserved.

## Applied to current checkout

- Applied every patch section except the outdated package.json hunk; merged `test:e2e:titles` while retaining all existing scripts, Jalali calendar dependency and recent UI refinements.
- PASS: `npm test` (12 test files), `npm run check:foundation` (41/41), `npm run build`, E2E script syntax check, and `git diff --check`.
- NOT RUN: connected conversation-title browser E2E. This application did not restart the backend or apply migration 0013; the feature requires the matching backend and migration to be running.
