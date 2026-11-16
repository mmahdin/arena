# Staging ConfigMap compatibility — 2026-11-02

Validated on top of develop `1821b0f`. The API and migration CLI now share split
`DATABASE_*` configuration, including raw `DATABASE_PASSWORD` from a Secret.
Explicit split values take precedence over a legacy URL. `FRONTEND_ORIGIN`,
`BUSINESS_TIMEZONE`, and `LOG_LEVEL` are applied by the API; explicit synthetic
dev login is supported in staging and remains rejected in production.

- **PASSED:** `python3 scripts/check_structure.py` and `python3 scripts/check_contract.py` (20 method/route pairs).
- **PASSED:** `docker build --progress=plain -f Dockerfile.Staging -t ai-therapist-backend:split-config-check .`; includes module verification, `go test ./...`, and API/migration binary builds with Go 1.23.
- **PASSED:** configuration tests for DSN credential escaping, IPv6/default port, invalid/missing fields, sanitized failures, stale-URL precedence, local compatibility, HTTPS cookie defaults, timezone/log level, explicit staging login and production rejection.
- **PASSED:** a temporary SQL Server 2022 instance with a fresh independent `wellbeing_config_check` database. `/app/migrate` created/applied the current schema using only split fields, without `DATABASE_URL`, AI credentials or a session secret. The password deliberately contained reserved URL characters.
- **PASSED:** the built Alpine API with `APP_ENV=staging`, split fields, `FRONTEND_ORIGIN`, `LOG_LEVEL=warn`, and `BUSINESS_TIMEZONE=Asia/Tehran`: live/ready probes, synthetic identity login, Secure/HttpOnly cookie, authenticated `/me`, denied-origin 403, and suppressed informational startup logs. Temporary test containers and network were removed.
- **PASSED:** local `.env.example` and Compose still target the independent loopback/Docker `wellbeing` database; the staging example contains the supplied staging connection settings and no credentials or Keycloak fields.
- **NOT RUN:** the full opt-in SQL integration suite, browser E2E or a shared/staging database rollout. The targeted SQL/HTTP test above is separate from those suites. No shared Yaran database, cluster ConfigMap/Secret, or existing migration file was changed by this task.

---

# HAMRAH-QA-02 — Tasks05/06/07 live validation

**PASS WITH LIMITATIONS (2026-11-02).** Real Go/MSSQL/Mock AI/browser full onboarding, resume, bookmarks, history, isolation and deletion passed. Fixed compilation, unapplied0008 SQL batching, null progress input, bookmark request races and covered Retry UI. Frontend44 tests, backend race/SQL/HTTP acceptance and builds pass. No external provider or real personal data.

See the [consolidated evidence, exact commands, changes and limits](../ai-therapist-frontend/docs/integration/QA_TASKS05_06_07.md). Original conversation/message fingerprints match;18 original user/profile rows remain. One original synthetic profile changed during live test activity, explicitly disclosed in the report. Existing visual source is preserved; physical device keyboard and exact original-ZIP pixel comparison remain unverified.

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

## Backend files changed in this QA

- `apps/api/tests/integration/phone_identity_test.go`: local-host guard, populated pre-0007 preservation/repeat upgrade, transaction rollback, Unicode/dynamic identity HTTP, eight-way concurrent first entry, onboarding, replay, ownership and revoke/re-entry regressions.
- `VALIDATION_REPORT.md`: this report. Backend runtime code/contracts/migrations unchanged.
- Fix/validation patch: `../hamrah-qa-tasks02-03-backend-fixes.patch` (includes tests and report; no invented runtime fix).

---

## Historical reports (superseded by QA above)

# Current validation — Tasks 02–07

Date: 2026-11-02, Asia/Tehran. Baseline commit: `fbdad4c` (Task 07). Working tree was clean at entry. **Final verdict: PASS WITH LIMITATIONS** for local synthetic-data use with Mock AI. All required backend build, database, health and HTTP acceptance gates passed after the fixes below. Prior reports did not establish results for this snapshot; their text is preserved under the historical section.

## Environment and dependencies

- Host: Linux amd64, Go 1.26.8, Python 3.14.4, Docker 29.9.0, Docker Compose v5.6.0.
- Docker build baseline: Go 1.23, Alpine 3.21 runtime. `go.mod` declares Go 1.23.0.
- SQL Server 2022 Developer, RTM-CU27-GDR 16.0.4305.5 (KB5126934), existing persistent local Compose volume.
- Existing pinned dependencies unchanged: Gin 1.10.1, google/uuid 1.6.0, go-mssqldb 1.7.2. `go mod verify` passed. No dependency or lockfile changes.
- Runtime: `APP_ENV=local`, dev login enabled, `LLM_PROVIDER=mock`, `supportive-v3`. API `127.0.0.1:8080`; SQL Server `127.0.0.1:14333`. Secrets and credentials were not printed or added to the patch.

## Initial failures and smallest fixes

| Finding | Reproduction / root cause | Resolution and final result |
|---|---|---|
| FAILED: migration 0006 on fresh MSSQL | `make integration-test`: Invalid column name lease_expires_at. SQL Server compiles references to new columns before the preceding ALTER TABLE executes in the same batch. | Execute the backfill and constraint statements using `sys.sp_executesql`, inside the runner's original transaction. Fresh install, upgrade with a processing fixture, repeat application and rollback tests PASSED. |
| FAILED: valid turn returns 409 | After fixing migration, repository completion and HTTP acceptance failed. SQL Server renders CONVERT(varchar(36), uniqueidentifier) in uppercase, while uuid.String() is lowercase; textual lease equality rejected the same UUID. | Parse stored lease text into uuid.UUID and compare UUID values. Known alphabetic UUID regression, replay and concurrent recovery/completion tests PASSED. |
| BLOCKED: full Go test run hangs | The provider deadline fixture waited forever on the server request context during httptest.Server.Close. Isolated `go test ./internal/modules/ai/adapters/openai -run TestDeadlineReturnsSanitizedFailure -v -count=1 -timeout=5s` FAILED with timeout and a stack in Server.Close. | Consume the fixture request body, add an explicit cleanup release channel, and assert both handler entry and actual context deadline. Full Go tests and race run PASSED; the test was neither skipped nor weakened. |

Migration 0006 was confirmed **not applied** in the existing local database before correction. Failed disposable runs had rolled it back. No previously applied migration was edited. No patches from Tasks 02–07 were reapplied.

## Commands and outcomes

Commands below run from `wellbeing-server/` unless marked `apps/api/`.

| Exact command | Result |
|---|---|
| `python3 scripts/check_structure.py` | PASSED: required structure, module boundaries, six skills and SQL dialect checks. |
| `python3 scripts/check_contract.py` | PASSED: 12 registered method/route pairs match source OpenAPI. This is a baseline checker, not full OpenAPI schema lint. |
| `make validate` | Initially BLOCKED by the deadline fixture hang; PASSED after fix. |
| `make test` (also invoked by make validate and make acceptance-backend) | PASSED after fix. |
| `go test ./... -count=1 -timeout=60s` from apps/api | PASSED uncached. Opt-in integration is skipped in this plain command; it ran separately with WELLBEING_INTEGRATION=true below. |
| `go vet ./...` from apps/api | PASSED. |
| `go mod verify` from apps/api | PASSED. |
| `gofmt -l apps/api` | PASSED: no unformatted Go files. Changed files were formatted using gofmt -w. |
| `git diff --check` | PASSED. |
| `make build` | PASSED: native bin/api and bin/migrate. |
| `make integration-test` | Initially FAILED on migration, then UUID comparison; final PASSED with real SQL Server and real Gin HTTP acceptance. |
| `WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test ./tests/integration -count=3 -timeout=60s` | PASSED three complete runs, including first-claim and completion races. |
| `WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test -race ./... -count=1 -timeout=90s` | PASSED, including live SQL Server and HTTP acceptance; no Go data race reported. |
| `docker compose stop api` | PASSED: stopped old binary before applying lease schema. |
| `make migrate` | PASSED: applied pending 0005/0006; later repeat run also PASSED with checksum verification. |
| `docker compose build --build-arg GOPROXY=https://goproxy.io,direct` | PASSED: module verification, Go 1.23 tests and both static binaries. Uses the mirror already verified for this local network. |
| `docker compose up --no-build -d` | PASSED: current API started, migration service Exited (0). |
| `make smoke` | PASSED against the updated Docker API and existing MSSQL, including ordered multi-turn history, retry, ownership and revoked switch/logout sessions. |
| `make acceptance-backend` | PASSED: validate + live integration + Docker smoke. |
| `make acceptance-http` | PASSED: full flow over actual HTTP via the current composition root and real MSSQL. |
| HTTP GET /health/live and /health/ready using urllib.request | PASSED: both 200, status ok. |
| sqlcmd metadata/history/count/fingerprint inspection and DBCC CHECKCONSTRAINTS WITH ALL_CONSTRAINTS | PASSED: schema and constraints valid; existing business data unchanged. One metadata command had a shell-escaping syntax error; corrected SQL was rerun successfully. |
| API log marker audit and Go logging-site inspection | PASSED for exercised paths: no tested chat, prompt, cookie or credential markers in API container logs; no module request/body logging found. This is not proof for every hypothetical panic or future integration. |
| Real external-provider smoke | NOT RUN: local configuration remains mock; no approved real-provider setup was used. Gateway success/status/redirect/deadline behavior was tested with local fake HTTPS endpoints. |
| Frontend browser suite | NOT RUN in this backend-only validation; earlier frontend results are historical. |

## Migration and preservation evidence

Before upgrade, the local migration ledger contained 0001–0004, with 3 users, 3 profiles, 5 conversations, 10 messages and 5 completed turns. The API was an older Task 01 Docker image despite the patched checkout. Stopped it before migration and rebuilt/restarted from current source.

After upgrade the ledger contains all six files, ordered and checksum-verified:

- 0001_initial_schema.sql
- 0002_synthetic_dev_users.sql
- 0003_message_order.sql
- 0004_synthetic_conversations.sql
- 0005_dev_sessions.sql — sha256 `29d092c509fd424c6e2173de6028a7089f55c103f5385d47c0010929f53d200f`
- 0006_turn_recovery_lease.sql — sha256 `e1b71ec5e46022a6025af07c03f0846bf9750816a4ffe36be8c5ed89aade73eb`

Verified dev_sessions columns, turn lease columns, active session index and trusted/enabled turns_lease_state_check. DBCC CHECKCONSTRAINTS returned no violations. Before and after all acceptance/smoke runs, aggregate count/fingerprint comparisons matched:

| Existing table | Count before/after | BINARY_CHECKSUM aggregate before/after |
|---|---|---|
| users | 3 / 3 | 606226035 / 606226035 |
| user_profiles | 3 / 3 | 1105849872 / 1105849872 |
| conversations | 5 / 5 | -1031161724 / -1031161724 |
| messages | 10 / 10 | -1640931374 / -1640931374 |

Fingerprints are a regression indicator, not a cryptographic proof. Existing five completed turns remain completed. Smoke creates/deletes only its own thread and adds synthetic dev sessions. No reset/drop/truncate/overwrite was performed on the existing wellbeing database. Destructive fixtures and cleanup ran only in newly generated wellbeing_test_<uuid> / wellbeing_accept_<uuid> databases. No such test databases remained after the suite.

## Task coverage and real integration verdicts

| Task | Outcome and evidence |
|---|---|
| 02 — Dev session/isolation | PASSED with actual HTTP + MSSQL: login, cookie-derived identity, tampered/expired/legacy/unregistered cookies, SQL-expired sessions, logout/switch revocation and replay rejection, wrong-owner read/turn/delete. Repository lifecycle checks verify session ownership and expiry. Unit HTTP checks additionally cover fail-closed storage failure and ineligible tester. |
| 03 — Onboarding/profile | PASSED with actual HTTP + MSSQL: fresh tester, rejected invalid profile, immutable completion time, reload persistence, conversation and owned-turn onboarding gates, profile isolation. |
| 04 — Conversation/history | PASSED with real MSSQL + HTTP: create/list/read/delete, continue via a new client using the same cookie jar, chronological pairs, duplicate payload replay, changed payload rejection, expired lease takeover, stale-worker fencing, concurrent reclaim. Added eight simultaneous initial claims and two simultaneous completions; exactly one lease and one pair persisted. |
| 05 — LLM integration | PASSED for the local Mock AI acceptance path and synthetic failure recovery. System prompt/styles/context budget and external adapter payload/status/redirect/deadline behavior PASSED unit/fake-HTTPS tests. No live external-provider claim. |
| 06 — Safety/reliability | PASSED for implemented synthetic Persian/English policy cases. Real SQL repositories plus application/AI policy verify no rows survive provider failure, invalid output or a provider returning after cancellation; same-ID retry saves exactly one pair. Unsafe output replacement and crisis provider bypass also persist only the expected safe pair. These fixtures replace only the model gateway, not repositories/profile storage. |
| 07 — Full acceptance | PASSED through real HTTP + real MSSQL: Dev Login → Onboarding → Create → Mock reply → History → Continue → Retry → Switch → Isolation → Logout. Separate failing-provider HTTP API produces 503 without turn/message leftovers; same ID succeeds against the mock API afterward. |

## Changed files / regression tests

- apps/api/db/migrations/0006_turn_recovery_lease.sql: corrected only the previously unapplied SQL Server batch compilation issue; transaction ownership remains in runner.
- apps/api/internal/modules/conversation/adapters/mssql/repository.go: compare lease UUID values.
- apps/api/internal/modules/ai/adapters/openai/gateway_test.go: deadline fixture no longer hangs; asserts actual deadline and provider entry.
- apps/api/tests/integration/foundation_test.go: upgrade of an existing processing turn, deterministic alphabetic lease ID, first-claim/completion races, canceled/failing/invalid/unsafe provider atomicity and same-ID recovery. Existing fenced-reclaim test still passes.
- apps/api/tests/integration/mvp_acceptance_test.go: real HTTP invalid/legacy/expired cookie cases, owned-turn pre-onboarding gate, database consistency assertion after simulated provider outage.
- README.md: remove stale statements that crash recovery is unimplemented; link current validation.
- VALIDATION_REPORT.md: this current evidence, with previous reports retained below.

Fix-only patch: `../wellbeing-backend-validation-fixes.patch`, generated from the initially clean baseline. No commits, pushes, dependency upgrades, product features or unrelated changes.

## Remaining limits / readiness

**PASS WITH LIMITATIONS:** current backend builds, tests, migrates and runs the full internal Mock AI MVP over HTTP and MSSQL. Keep it local and synthetic. Real provider connectivity/quality/billing is NOT RUN. Safety is phrase-based and not clinically validated. Character context budgets are not token accounting. Rate/cost limits, distinct simultaneous turns in one conversation, pagination beyond 100 messages, real auth, public deployment and a full OpenAPI linter/code generation remain outside this acceptance result.

---

# Historical reports — prior snapshots only

The following original entries are preserved for provenance. Their earlier PASSED/BLOCKED statements do not supersede the current results above.

# Validation report — executable MSSQL foundation

Date: 2026-11-02 (Asia/Tehran). Local synthetic-only environment.

## Verified

- Dependencies installed; `go.mod`/`go.sum` and frontend `package-lock.json` now pin resolved versions. npm installation reported zero vulnerabilities.
- `python3 scripts/check_structure.py` and `python3 scripts/check_contract.py` pass (12 API/health method-route registrations).
- `go test ./... -count=1` passes with `WELLBEING_INTEGRATION=true` through `scripts/run_local.py`; live SQL Server tests create and remove an independent `wellbeing_test_<uuid>` database.
- Live database coverage: fresh/repeated migrations, 3 users/3 conversations/6 messages, wrong-owner reads/writes/deletes, pending/completed retries, changed payload rejection, tied timestamp ordering, cascading deletes, failed migration rollback and applied-checksum rejection.
- Native API/migration binaries build; Docker build runs tests and produces both binaries using Go 1.23. Local native tests used Go 1.26.8.
- SQL Server 2022 runs with a persistent dedicated volume, published only on `127.0.0.1:14333`. API runs via Docker on `127.0.0.1:8080`, mock AI enabled. Migration container exits 0.
- Migrations `0001`–`0004` applied; upgrade from `0002`, fresh database and repeat application pass. Original migrations are unchanged.
- Live API smoke passes: readiness, session, profile isolation, foreign-owner 404, chat, idempotency, message history and deletion. After fixing message ordering, 12 repeated live API smoke runs passed.
- Frontend architecture, actual backend snapshot comparison, 7 Node tests, typecheck and Vite build pass with Node 24.21.0/npm 11.19.0.
- Four Playwright tests pass using system Chrome: mobile/desktop stub flow and mobile/desktop real SQL Server API flow. First live run exercised tester 2 onboarding; subsequent runs use saved profile. Live tests verify chat/reload, stored pair order, tester switching, cross-owner 404 and cleanup.

## Fixes and operational findings

- Fixed frontend SessionGate import from the old provider location to core/session/context.
- Reproduced intermittent reversed user/assistant history: millisecond timestamp plus random UUID ordering was invalid. Added `0003_message_order.sql`, persisted sequence/index and adapter ordering; forced-timestamp regression now passes. Historical ties across distinct turns can only be backfilled deterministically, not reconstructed exactly.
- Added `0004_synthetic_conversations.sql`: explicitly synthetic, non-sensitive demo history; tester 2 has no seeded messages.
- Native commands load `.env` literally with `scripts/run_local.py`; shell-sourcing the SQL URL failed because of `&`. Secrets are generated locally and not printed or committed.
- Removed stale emitted frontend config JS/declarations, enabled noEmit for tool configs, fixed browser test radio selector, and added opt-in live E2E.
- Default Docker module download returned 403 for mimetype. Verified fallback: `docker compose build --build-arg GOPROXY=https://goproxy.io,direct`, retaining module checksum verification.
- Playwright Chromium download timed out, then returned a regional 403. System Chrome worked with `PLAYWRIGHT_CHANNEL=chrome`; tests were not skipped.
- Initial sandbox Node subprocess tests were blocked; they passed when rerun with approved process access.

## Reproduction

```sh
make validate
make build
make integration-test
make smoke
cd ../wellbeing-front
npm run validate
node scripts/check-contract.mjs ../wellbeing-server/contracts/openapi/openapi.yaml
PLAYWRIGHT_CHANNEL=chrome WELLBEING_LIVE_E2E=true npm run test:e2e -- --workers=1
```

## Limits

External LLM integration, full parallel-turn/conversation concurrency, processing-turn crash recovery, complete error/retry browser matrix and OpenAPI schema lint/code generation remain unverified/deferred. Dev sessions are local tester selection. No claim of public, real-data or clinical readiness.

## Task 02 — Dev Session & User Isolation change (2026-11-02)

- Added a revocable `dbo.dev_sessions` table in forward-only `0005_dev_sessions.sql`; no prior migrations were edited.
- Signed development cookies now include a random session ID, and protected endpoints verify its DB status and synthetic tester eligibility. Tester switches and logout revoke the previous server-side session.
- Expanded automated HTTP-level tests for absent/tampered/expired/revoked cookies, switching identities, logout replay and DB fail-closed behavior; expanded opt-in SQL Server repository lifecycle test.
- **PASSED here:** `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py`, Go formatting and patch syntax validation.
- **BLOCKED here:** Go test/build and live SQL Server integration; current execution environment has no installed Go module cache, working Go module network access or Docker. These checks **must be rerun** after applying the patch in the functioning project environment.
- Re-test `make migrate`, `make test`, `make integration-test`, `make smoke` and paired frontend E2E before accepting this task as complete. The results reported in earlier sections apply only to the pre-change snapshot.

## Task 03 — Onboarding / User Profile patch (2026-11-02)

- Hardened domain validation with typed errors and Persian-compatible display-name rules, preserving the first completion timestamp across later edits.
- SQL Server uses `COALESCE(onboarding_completed_at, @p3)` so a stale concurrent first-save cannot overwrite the first recorded completion. The application returns the persisted row.
- Conversation create/turn requires a completed profile via the existing application port; foreign conversation IDs remain `404` even when onboarding is incomplete. Runtime `409 ONBOARDING_REQUIRED` was documented in OpenAPI without changing existing JSON bodies.
- Added pure domain, Gin HTTP/session, conversation application and isolated SQL Server integration tests; no new migration was necessary.
- **PASSED in this environment:** Go `gofmt`, `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py` and unified patch syntax checks.
- **BLOCKED in this environment:** `go test ./...` (Gin, UUID and go-mssqldb not available in the module cache; `GOPROXY=off`); Docker / live SQL Server were unavailable. These are **not** claimed as passed.
- **After applying Task 02 and this patch locally:** `make test`, `make integration-test`, `make smoke`, and frontend E2E must run against real MSSQL. When using this patch, confirm it applies to the Task 02 baseline and does not replace local edits.

## Task 04 — Conversation, History and crash-safe Retry patch (2026-11-02)

- Added forward-only `0006_turn_recovery_lease.sql` for `lease_id` / `lease_expires_at`. Existing processing reservations are made reclaimable; completed turns remain unchanged. **Stop the old API before migrating**; mixed app versions are not supported.
- Fenced idempotent turn processing: active lease 409, expired lease can be reclaimed by the same owner/turn/input, late workers cannot commit or release the successor lease. Exactly one user+assistant pair remains transactionally committed and completed responses replay unchanged.
- Added application unit coverage, concurrent reclaim / ownership / payload mismatch / stale-worker tests against isolated SQL Server, extended multi-turn HTTP smoke and clarified OpenAPI retry description. No new HTTP endpoint or JSON property.
- **PASSED in this patch build environment:** Go formatting, Python structure + contract checks, Python smoke script syntax, `git diff --check` and patch replay check.
- **BLOCKED here:** Go unit tests (`GOPROXY=off`: Gin/UUID/go-mssqldb packages not cached) and actual SQL Server tests (Docker not available). Run `make migrate`, `make test`, `make integration-test`, `make smoke` and frontend E2E in the user's functioning local stack before acceptance.
- Deferred: more than 100 history messages (cursor pagination), simultaneous distinct turns in one thread, server-sent streaming, real-provider latency/lease monitoring.

## Task 05 — OpenAI-compatible gateway and bounded conversation context (2026-11-02)

- Revised prompt to `supportive-v2` (style-specific Persian guidance; profile JSON treated as untrusted data), keeping stored historical versions unchanged. No schema migration or change to existing `/turns` JSON.
- Added bounded chronological suffix history (`MAX_CONTEXT_CHARS`), model output budget (`MODEL_MAX_OUTPUT_TOKENS`) and configurable timeout (`LLM_TIMEOUT_SECONDS`, max 90s < 120s turn lease).
- HTTPS Chat Completions adapter refuses redirects, bounds HTTP response size, rejects malformed/empty/filtered/truncated replies, and emits sanitized errors without forwarding upstream error bodies. Application validates empty/oversized answers and continues the provisional safety fallback.
- Added unit tests for context order/budget, persona/profile framing, fake HTTPS provider success/failure, redirect and deadline, and configuration validation.
- **PASSED in patch environment:** `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py`, YAML parsing for Compose/OpenAPI, Go standard-library parser across 43 source files, Go formatting, `git diff --check`, and patch replay validation. Cross-repo static preflight reports zero errors (its frontend snapshot/lock warnings refer to the older available frontend checkout).
- **BLOCKED in patch environment:** `GOPROXY=off go test ./internal/modules/ai/... ./internal/platform/config` (Go module cache does not include Gin/UUID); Docker and a live SQL Server/provider are unavailable. All included Go unit tests and live-provider behavior must be verified in the working local stack. No provider network call or public clinical/privacy approval was performed. `LLM_PROVIDER=mock` remains default.

## Task 06 — Synthetic AI safety and canceled-turn reliability patch (2026-11-02)

- Added Persian/Arabic letter and spacing normalization and expanded deterministic phrase cases for explicit crisis requests, false clinician claims, unsafe medication instructions, exclusivity, claimed emergency calls, and a recognizable system-prompt leak fragment. These are intentionally non-clinical heuristics; semantic and indirect cases still fail.
- Versioned prompt as `supportive-v3`; strengthened boundaries for diagnosis, prescriptions, dependency, delusions and claims of emergency action. Previously stored replies retain their original prompt versions.
- Application rejects provider replies on canceled contexts (including a provider that ignores cancellation), malformed UTF-8 and unexpected control bytes; generic provider error paths preserve the existing `503 AI_UNAVAILABLE`, SQL Server lease release and client `clientTurnId` retry semantics.
- Added synthetic Go safety policy tests, application tests for cancellation and several model-output risks, and `docs/quality/ai-safety-evaluation.md` explicitly documenting known misses and non-clinical status. No API schema or database migration changes.
- **PASSED in patch build environment:** standalone standard-library Go safety tests (`GOPROXY=off GOWORK=off go test safety.go safety_test.go -count=1`), Go formatting, architecture and OpenAPI static checks, patch replay checks.
- **BLOCKED here:** full Go module tests (`GOPROXY=off` cannot download Gin/UUID), live SQL Server + Docker/provider verification unavailable. These have **not** been claimed as passed.
- **Local acceptance:** run `make test`, `make integration-test`, `make smoke`, and (optionally) approved synthetic-only provider tests. Treat any detected false negatives as explicit risk; external/real-user testing is prohibited pending independent clinical and security/privacy review.

## Task 07 — Internal MVP full-flow acceptance patch (2026-11-02)

- Added `TestMVPHTTPAcceptance`, a **real Gin HTTP + MSSQL** acceptance test that creates and drops an isolated random local database. It tests synthetic Dev Login, initial onboarding, profile edit, chat and reload, wrong-owner 404, message order, repeat/different-payload `clientTurnId`, provider 503 without phantom messages, same-ID recovery with mock, tester switching, cookie revocation and deletion.
- Added `make acceptance-http` (the isolated flow) and `make acceptance-backend` (structure/contract/unit/integration/smoke). Added `docs/quality/mvp-acceptance.md`, an explicit backend/real-browser evidence matrix and safe local execution steps.
- No HTTP contract, seed, existing migration or public API response changed. **No new migration.**
- **PASSED in this patch environment:** Python structure and OpenAPI checkers, Go formatting, Go syntax parsing, `git diff --check`, and patch replay validation against the exact Task 06 baseline.
- **BLOCKED in this environment:** `GOPROXY=off go test ./tests/integration -run '^TestMVPHTTPAcceptance$'` (the Gin/UUID/SQL Server driver dependencies are not cached); Docker/SQL Server and unstubbed real-browser E2E are not available. An actual `make acceptance-backend` run is required before declaring internal MVP acceptance.
- **Important distinction:** the locally present original frontend checkout has API-stubbed Playwright tests and is not sufficient evidence of *full* browser integration. Supply the current frontend checkout and run an unstubbed live E2E suite there. The current backend patch touches no frontend files and does not claim product-wide sign-off.
