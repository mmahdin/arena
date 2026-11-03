# HAMRAH-QA-02 — Tasks05/06/07 live validation

**PASS WITH LIMITATIONS (2026-11-02).** Real Go/MSSQL/Mock AI/browser full onboarding, resume, bookmarks, history, isolation and deletion passed. Fixed compilation, unapplied0008 SQL batching, null progress input, bookmark request races and covered Retry UI. Frontend44 tests, backend race/SQL/HTTP acceptance and builds pass. No external provider or real personal data.

See the [consolidated evidence, exact commands, changes and limits](docs/integration/QA_TASKS05_06_07.md). Original conversation/message fingerprints match;18 original user/profile rows remain. One original synthetic profile changed during live test activity, explicitly disclosed in the report. Existing visual source is preserved; physical device keyboard and exact original-ZIP pixel comparison remain unverified.

---

## Historical reports

# HAMRAH-QA-01 — Tasks 02 v2 + 03 live validation (2026-11-02)

**Verdict: PASS WITH LIMITATIONS.** The core Hamrah journey passed with actual React/Vite → Go/Gin → Microsoft SQL Server 2022 → Mock AI. Separate controlled failures also passed. Physical mobile keyboard/device behavior and a pixel comparison against the original designer ZIP remain unverified. This is an internal synthetic-data MVP, not public authentication or clinical validation.

## Baseline and environment

- Backend HEAD `53a2b7edc569ddc9678995ee95c4b9f251415b8f`; frontend HEAD `7d43e20e6d679fe6d29be37d83b1053aa72473a4`. Both working trees were clean before QA. No patches were reapplied, commits created, or changes pushed.
- Tasks 02 v2 (dynamic fictional identifiers) and frontend 03 were present. Backend OpenAPI and frontend snapshot are byte-identical (`cmp` passed). No endpoint, schema, or applied migration was changed in QA.
- Linux amd64; Go 1.26.8 host / Go 1.23 Docker build; Python 3.14.4; Node 24.21.0; npm 11.19.0; Vite 6.4.5; Playwright 1.64.1; Google Chrome 155.0.8059.39.
- Docker 29.9.0 / Compose v5.6.0. SQL Server 2022 Developer CU27-GDR `16.0.4305.5`, Ubuntu 22.04.5.
- Browser `http://127.0.0.1:5173`, same-origin Vite `/api/v1` proxy → API `127.0.0.1:8080`; MSSQL `127.0.0.1:14333`. Normal API uses `LLM_PROVIDER=mock`.

## Executed checks

Commands below run from the named repository; Go commands use `apps/api` unless the local environment loader is specified. Initial sandbox networking failures were retried outside the sandbox, with approval. No such failure was counted as a passing test.

| Gate / exact command | Final result |
| --- | --- |
| Backend `python3 scripts/check_structure.py` | PASS, 41 paths / 6 skills |
| Backend `python3 scripts/check_contract.py` | PASS, 13 method/route pairs |
| Backend `make validate`, `make test`, `make build` | PASS |
| Backend `go test ./... -count=1 -timeout=90s` | PASS |
| Backend `go vet ./...`, `go mod verify` | PASS |
| Backend `WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test -race ./... -count=1 -timeout=120s` | PASS, including actual SQL/HTTP; rerun after new tests |
| Backend `make migrate` twice | PASS, immutable checksum reapplication |
| Backend `make acceptance-backend`, `make acceptance-http` | PASS, actual MSSQL / HTTP / smoke |
| Backend `docker compose build --build-arg GOPROXY=https://goproxy.io,direct` then `docker compose up --no-build -d` | PASS, refreshed stale API binary and retained DB volume |
| Frontend `npm install --package-lock-only` then `npm ci` | PASS after fixing missing Playwright lock entries; npm audit 0 vulnerabilities |
| Frontend `npm test` | PASS, 22 tests, including 2 new regressions |
| Frontend `npm run check:foundation` | PASS, 41/41, includes contract/architecture references and 7 skills |
| Frontend `npm run build` | PASS, including JSX/import compilation |
| Frontend `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:session` | PASS, real API/SQL, no interception |
| Frontend `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:chat` | PASS, real chat/history, no interception |
| Frontend `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:qa` | PASS, complete journey and responsive checks, no interception |
| Frontend `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:faults` | PASS, controlled real provider outage / dropped real committed response |
| Frontend `npm run check:source-baseline` | FAIL as an original-ZIP byte equality check: App.jsx, Main.jsx, Onboarding.jsx, styles.css already differed at QA HEAD; confirmed with `git show HEAD:<path>` hashes. Manifest was not weakened or rewritten |
| `node --check` for new browser scripts; `gofmt` on changed Go test; both `git diff --check` | PASS |
| SQL `DBCC CHECKCONSTRAINTS WITH ALL_CONSTRAINTS` | PASS, no violations |
| Physical iOS/Android keyboard and exact original-ZIP screenshot regression | NOT RUN |
| External LLM provider | NOT RUN, not approved; failure fixture only targets local `https://127.0.0.1:1` |

## Failures reproduced and resolution

1. **Clean install broken:** `npm ci` failed with EUSAGE because `package.json` declared Playwright but the lockfile omitted Playwright/core. Updated only the required lock entries; repeated clean installation passed.
2. **Stale running backend:** `/dev/identity-session` returned plain 404, and session E2E timed out. Docker still ran the pre-v2 binary even though source and migrations were current. Rebuilt/recreated API and migration services; session/chat browser suites passed. No runtime backend source fix was needed.
3. **Deleted-thread state retained:** a 404 from deleting an already deleted active conversation did not reset its messages or invalidate an outstanding open. A new failing Node regression reproduced it; reset now fences the outstanding response.
4. **Wrong 404 UI and endless retry:** the actual backend code is `RESOURCE_NOT_FOUND`, while UI expected `NOT_FOUND`. Failed turns still offered repeated retry to a missing thread. Normalize by HTTP status, recognize the actual code, stop terminal 404 replay, and offer a new conversation. Regressions verify no extra turn call; real browser tests cover deletion both before open and before send.
5. **Offline login Retry unclickable:** real browser input failed because the form's absolute primary button intercepted pointer events over Retry. Scoped the form CTA to normal flow with full width. The same browser test now recovers using an ordinary click, without forcing it.

Test development corrections: a new test initially expected unknown `userId` JSON to be rejected. Current binding ignores unknown fields; the correct security assertion is that it edits only the session user's profile and leaves the other profile unchanged, which passed. The first fault script also omitted accepting the intentional abandon-pending confirmation; that automation was corrected. Two diagnostic SQL queries used wrong ledger/aggregate syntax and were corrected before recording results.

## SQL and preservation evidence

- Existing local ledger already contained 0001–0007. Unchanged checksums for required migrations:
  - 0005 `29d092c509fd424c6e2173de6028a7089f55c103f5385d47c0010929f53d200f`
  - 0006 `e1b71ec5e46022a6025af07c03f0846bf9750816a4ffe36be8c5ed89aade73eb`
  - 0007 `9067e1aae96dd2cd4d25905f8d7ce6fef5b149e44cd3872286dfa3d1ee27fa21`
- Fresh disposable DB migrations passed in the foundation/HTTP tests. The expanded phone test separately applies 0001–0006 to a populated disposable DB, records all four table count/checksum pairs, applies 0007, checks exact equality, then reapplies all migrations.
- Same identifier six concurrent SQL requests and eight concurrent actual HTTP first-entry requests resolved to one user; exactly one HTTP response reported `created:true`. Initial user/profile/mapping/session are transactional. Reusing a session PK intentionally fails and rolls back the new user/profile/mapping.
- Mapping schema inspected: `lookup_key varbinary(32)`, unique `user_id`, `created_at datetime2(3)`; HMAC-SHA256 over canonical digits, no raw-number column. Raw input absent from opaque user test keys.
- Original data before/after, `COUNT / CHECKSUM_AGG(BINARY_CHECKSUM(*))`:

| Original records | Before | After |
| --- | --- | --- |
| Users | 3 / 1025083757 | 3 / 1025083757 |
| Profiles | 3 / 1297994633 | 3 / 1297994633 |
| Conversations | 5 / -1031161724 | 5 / -1031161724 |
| Messages | 10 / -1640931374 | 10 / -1640931374 |

- No existing DB reset, drop, truncate, or applied migration edits. Integration tests create/drop only their own randomly named databases; final database inventory showed no leftover test databases. Browser-generated fictional identities/profiles and session audit rows remain intentionally; tested conversations were removed through owner-scoped API deletion, including one identified leftover from an earlier test run. Final conversation/message fingerprints exactly match the initial DB.
- Startup/runtime log audit found zero raw identifier, cookie-header or exercised synthetic-message markers. No message/identity request logging was found in runtime source. No cookies, tokens, secrets or storage-state files were saved in artifacts.

## Actual feature coverage

- Identity: ASCII/Persian/Arabic digits, outer whitespace normalization, invalid/empty/short/long/unexpected formats, consistent UUID on refresh/re-entry, different UUID for different identifiers; initial incomplete profile, completed-profile Home, preserved profile timestamp/history.
- Security: signed HttpOnly session cookie; missing/tampered/expired/legacy/unknown/revoked cookies rejected over actual HTTP; switch/logout revoke old cookies; foreign read/turn/delete return 404. Client `userId` cannot select a profile owner. Unit tests fence late old login/history/turn completions after switch/logout; network failures never establish a false authenticated UI.
- Profile: supported name/style only, invalid name rejection, actual persistence, incomplete onboarding cannot create/send chat; questionnaire answers are not submitted.
- Chat: actual Mock AI HTTP reply, one atomic user/assistant pair, chronological multi-turn history, duplicate UUID replay, changed-payload rejection, concurrent claim/completion, processing lease recovery and stale-worker fencing. Unit frontend 409/503 retry and SQL concurrent tests passed. Browser 503 and lost-response replay preserve identical JSON `clientTurnId`/message and store exactly one pair per turn.
- History: real empty/list/error/retry states, multiple conversations, older-thread continuation, reload/relogin recovery, A/B isolation, cancellation and confirmation of deletion, missing-thread recovery, scoped-state clearing.
- Full browser journey: fresh fictional number → incomplete-onboarding reload → name/style → Home → real chat → second thread → older history/continue → refresh → logout/login → recovered history → real offline history/retry → B switch/isolation → A return → confirm/delete → deleted-thread 404. No conversation endpoint interception in session/chat/QA scripts.
- Controlled fault script ONLY forwards one turn to an actual second Gin API with unreachable local provider, or drops a genuine successful HTTP response after SQL commit. It never supplies an invented reply or success. It also exercises real 401/404 and offline login recovery.

## Visual checks and limits

- Screenshots captured under ignored `test-results/hamrah-qa/`: login/chat at 360, 390, 430, 1280px; history at 390px and chat at reduced 480px height. RTL, horizontal overflow, composer bounds and reduced-motion mode checked. Login/chat screenshots were visually inspected.
- No redesigned screens, palette, fonts, assets, JSX class names, or route flow. Intentional visible changes: missing-thread error/new-chat recovery and placing the login CTA in its form so help/Retry remain usable. Pixel identity to the original ZIP is not claimed.
- No real OTP, SMS, ownership proof or public auth. Knowing a fictional number selects its tester. Keep the secret stable because it keys identity mappings.
- History/messages capped at 100; no pagination. Initial conversation creation is not idempotent after a lost create response. Pending retry exists only in memory; refreshing recovers committed history, not an unfinished turn UUID. Distinct concurrent turns remain a deferred policy. Physical touch/keyboard and exact visual pixel comparison remain unverified.
- Insights, voice, bookmarks, regeneration, clinical questionnaires, account deletion and clinical safety are not claimed complete. Real provider quality is outside this run.

## Frontend files changed in this QA

- `package-lock.json`: synchronize existing Playwright dependency.
- `package.json`: add `test:e2e:qa` and `test:e2e:faults` commands.
- `src/features/conversation/store.js`: terminal 404 and already-deleted active state recovery.
- `src/screens/Main.jsx`: correct missing-thread error and new-chat recovery action.
- `src/styles.css`: scope form primary button to normal flow; no palette/asset changes.
- `scripts/conversation.test.mjs`: two new failing-then-passing regressions and actual backend 404 code.
- `scripts/e2e-qa.mjs`, `scripts/e2e-faults.mjs`: full live journey and separate controlled failures.
- `VALIDATION_REPORT.md`: this report; `docs/integration/QA_TASKS02_03.md`: repeatable run instructions.
- Fix patch: `../hamrah-qa-tasks02-03-frontend-fixes.patch`.

---

## Historical reports (superseded by QA above)

# HAMRAH-INTEGRATION-03 — Frontend Chat/History (implementation, not live E2E)

**Baseline:** Hamrah designer UI + approved Task 02 v2 synthetic phone session. **Incremental patch**; the older rejected tester picker is not needed.

## Changes
- Added `src/features/conversation/api.js` and `store.js` with cookie-backed `/conversations`, `/{id}/messages`, `/{id}/turns`, delete and session-scoped cancellation.
- Connected the existing Chat and History screen layout in `src/screens/Main.jsx`; history replaces demo weekly statistics with actual threads. Replies come **only from Go/MSSQL**, not setTimeout or fabricated client strings.
- A failed or ambiguous turn keeps its exact UUID and input for 409/503/network retry; GET messages reconciles persisted user/assistant pairs. The user can confirm abandoning a pending retry when starting a new thread.
- Added 9 Node tests for retry/replay, stale responses, account isolation, 401/404, delete and request shape; added an opt-in real Playwright test in `scripts/e2e-chat-history.mjs`.
- Existing designer illustrations, fonts and unrelated source routes were preserved; no backend patch, no migration, no streaming introduced.

## Results
- **PASSED:** `npm test` — 20/20 Node tests (9 new + 11 existing).
- **PASSED:** `npm run check:foundation` — 41/41.
- **PASSED:** `node --check` on new plain JavaScript files; `git diff --check` before packaging.
- **BLOCKED:** `npm ci --offline --ignore-scripts --no-audit --no-fund`, missing cached `yallist-3.1.1.tgz`. Consequently `npm run build` and visual QA could not run here.
- **NOT RUN:** live `npm run test:e2e:chat` or backend acceptance, because Docker/MSSQL and browser environment were unavailable; must execute locally.

## Known limitations and scope
- A network failure *during initial POST /conversations creation* can leave an empty server conversation; unlike `/turns`, creation has no idempotency key. This should be hardened in backend later if user acceptance finds it material.
- List and messages are limited to the latest 100; no pagination yet. No backend voice, insight, message save/regenerate/report or weekly metrics; screens are preserved but not claimed connected.
- `clientTurnId` pending retry lives **only in memory**; on browser refresh, recover saved messages via History. Do not promise resumable unfinished turns after refresh.
- Session is fictional phone-shaped `0900000XXXX` identity without OTP; **not authentication**, for locally isolated synthetic testers only. No clinical safety validation.

---

## HAMRAH-INTEGRATION-02 v2 — implementation validation (not live integration)

- Previous seeded-user Task 02 is **rejected and replaced**. This patch is based on the documented Hamrah UI and new Go v2 API; preserves existing designer markup/artwork and limits name/approach/ready onboarding to non-sensitive profile fields.
- **PASSED:** Node tests for Unicode synthetic phone normalization, bootstrap, returning user, switching, logout, 401/503, profile and stale response; `npm run check:foundation` (41/41).
- **BLOCKED:** `npm ci --offline` — npm tarball absent in this sandbox; Vite build not executed here.
- **NOT RUN:** real Go/MSSQL Playwright browser E2E and visual screenshot comparison. These must pass in user's local environment before marking done.
- Contract snapshot includes `POST /api/v1/dev/identity-session` and retains prior dev routes only for legacy automated tests. Backend migration 0007 must precede frontend deployment.
- **Never enter a real number.** `0900000XXXX` is a fictional team convention, not a telecom-reserved range, and there is no SMS/OTP. Local only.

---

# Validation report — new Hamrah UI developer handoff (2026-11-02)

## Intent
Repackage the latest uploaded `ai-therapist-frontend(1).zip` with agent documentation, seven adjusted project-local skills, contract snapshot, integration/gap plan and deterministic checks. Preserve the *new* designer source as product UI authority. This is NOT evidence of live API integration.

## What is protected
All original `src/**`, `public/**` and `index.html` bytes are preserved according to `docs/design/original-source-sha256.json` (checked via script). Only original `package.json` development scripts and `vite.config.js` host/proxy config are altered. Original `pnpm-lock.yaml` and `package-lock.json` remain unchanged and should be reviewed by npm/pnpm before committing any dependency updates.

## Current source observations
- React19, Vite6, JavaScript JSX, CSS max-width 430px; 25 onboarding routes plus home/survey/day/history/menu/chat/insight.
- `src/App.jsx` keeps all answers/messages in memory. History and Insights are static; Chat uses a fake delayed reply. No live `/api/v1` calls.
- Current UI contains unverified research/doctor/testimonial/privacy claims and health-sensitive questions. This packaging does NOT change display copy; agent must gate/correct it in the first connected feature slice.
- Backend current runtime report from another environment: Tasks02–07 PASS WITH LIMITATIONS after three fixes; this frontend ZIP does not include that latest checkout, so no integrated runtime revalidation occurred here.

## Tests and evidence
See results of the exact commands run while packaging; check reports **PASSED/FAILED/BLOCKED/NOT RUN**, never infer a browser run from a static check.

## Pending validation
- Install dependencies with `npm ci`, Vite build and browser screenshots / real E2E on client machine.
- Confirm the latest backend checkout against the bundled Task07-derived OpenAPI snapshot.
- Wire Dev Session, profile, real JSON chat/history before claiming fullstack readiness.
- Validate every connected screen's clinical/privacy factual copy and data minimization.
- Real-provider mode and real-person clinical safety remain unapproved.

## Packaging checks conducted in this environment
- **PASSED:** `npm run check:foundation` — all structure, original file presence, RTL, API reference and skills.
- **PASSED:** `npm run check:source-baseline` — each original src/public/index file byte-identical to latest uploaded ZIP.
- **PASSED:** `npm test` — 4 Node built-in tests for routing/assets, demo marking, core contract, and safe Vite proxy.
- **PASSED:** `quick_validate.py` applied to all 7 skills.
- **BLOCKED:** `npm ci --offline --no-audit --no-fund` because an npm tarball was not in the local cache (`ENOTCACHED`). Build cannot run without dependencies in this environment.
- **NOT RUN:** Vite runtime build, screenshot visual regression, live Go/MSSQL browser E2E and live external provider.
