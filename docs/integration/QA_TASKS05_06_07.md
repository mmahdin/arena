# HAMRAH-QA-02 — Tasks 05, 06 and 07

Date: 2026-11-02. **Verdict: PASS WITH LIMITATIONS.** Full onboarding, interruption/resume across browser sessions, real chat, bookmarks, history, re-entry, account isolation and deletion passed on real React/Vite → Go/Gin → MSSQL → backend Mock AI. Physical mobile keyboards and a pixel comparison against the original designer ZIP were not available. No external model, real number or patient data was used.

## Baselines and preservation of user work

- Initial backend branch `develop`, HEAD `1655c8873547d46be5c13d87fdc32832bff2ee03`; Task07 was already present as uncommitted changes in AGENTS/composition/OpenAPI plus new migration 0009, bookmark module/tests/docs. Initial frontend branch `develop`, HEAD `f118e2c5b59b24c778b3f663dcde732d410b2b79`; Task07 changes already existed in App, conversation API/store, Main, CSS, snapshot and new bookmark tests/docs.
- Saved all tracked/nonignored starting files under `/tmp/hamrah-qa02-baseline` before edits. No integration patches were reapplied or unrelated edits overwritten.
- During QA, HEAD advanced to backend `96ad240` and frontend `f7ba54e` (Saved Messages commits), including some early QA fixes. Those commits were not created by this agent. Delivered patches compare final files with the saved initial **working files**, not simply HEAD; they exclude the user's pre-existing Task07 implementation.
- Tasks05/06/07 code and migrations 0008/0009 were present. Backend and frontend OpenAPI are byte-identical; 18 method/route pairs match composition. No API contract was changed by QA.

## Environment / current services

Linux amd64; Go 1.26.8 host / Go1.23 Docker build; Python3.14.4; Node24.21.0; npm11.19.0; Vite6.4.5; Playwright1.64.1; Chrome155.0.8059.39; Docker29.9.0 / Composev5.6.0. SQL Server2022 Developer, existing local `2022-latest` container (previous verified runtime16.0.4305.5).

Vite `127.0.0.1:5173` proxies relative `/api/v1` to API `127.0.0.1:8080`; MSSQL is loopback14333. Vite was already running; a second `npm run dev` reported port in use and the existing service was reused. API was rebuilt from current source and migrations ran successfully. Final ready endpoint returned `{"status":"ok"}` and frontend HTTP200. All successful model calls used `LLM_PROVIDER=mock`.

## Reproduced defects and fixes

1. **Backend would not compile.** `make validate` failed with undefined `errors` in bookmark HTTP handler. Added its missing standard-library import. The build and full test suites now pass; added bookmark failure mapping/sanitization regression coverage.
2. **Real MSSQL migration0008 failed.** All fresh integration DBs reported `Invalid column name 'onboarding_step'`: SQL Server compiled the new-column CHECK in the same batch as ALTER ADD. Local ledger initially contained only0001–0007 and failed disposable runs rolled back. Moved constraint compilation to `sys.sp_executesql`, still within the runner transaction. Corrected0008 was first applied only after successful disposable tests; no already-applied migration was edited. Migration0009 was unchanged.
3. **Bookmark action remained permanently busy.** A list refresh during Save incremented the shared request version; Save's `finally` then failed to clear busy state. A new Node test failed first. Suppress redundant list refresh during mutation and clear busy on the current identity scope, independent of list version.
4. **Deleted bookmarks reappeared.** A list request begun before conversation deletion could publish its old rows afterward. A second new Node test failed first. Successful/not-found deletion invalidates old bookmark list/mutation versions and filters the deleted conversation's bookmarks immediately.
5. **Progress accepted JSON null.** Added `{"furthestStep":null}` to negative HTTP tests; it returned200 as integer zero. Parse via a non-null integer pointer; null now returns400. Extra properties/questionnaire answers remain rejected.
6. **Bookmark Retry was covered by chat content.** A real browser failure test could not click Retry: the absolutely positioned message scroll overlaid the notice. Move the existing notice inside the messages list/scroll area. The identical normal-click test now passes. No CSS, palette, art or screen was replaced.
7. **Old tests assumed the short onboarding or old History.** QA/fault helpers expected Name first and only one History error region. Updated them to traverse/resume the full designer flow and retry conversation/bookmark sections separately. Shared helper now supports a saved cursor and the actual Ready→Name→Approach→Ready recovery when draft name was lost.
8. **Old preservation assertion compared row shape.** The prior phone upgrade test used `BINARY_CHECKSUM(*)` across a newly added profile column, causing a false failure. It now compares the original explicit profile columns and tests populated0007→0008/0009 upgrade, preserving actual old values. No data assertion was removed.

Automation corrections were also made: select a valid visible menu/backdrop area or route Home after closing preferences; do not force-click through drawer content. Diagnostic fixture cleanup never deleted an unrecognized conversation.

## Exact commands and final gate results

Backend commands are from `wellbeing-server`; direct Go commands are from `apps/api`.

| Command/check | Result |
| --- | --- |
| `python3 scripts/check_structure.py` | PASS, 41 required paths / 6 skills |
| `python3 scripts/check_contract.py` | PASS, 18 method/route pairs |
| `make validate`, `make test`, `make build` | PASS after missing import fix |
| `go test ./... -count=1 -timeout=90s` | PASS |
| `go vet ./...`, `go mod verify` | PASS |
| `WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test ./tests/integration -count=1 -timeout=120s` | PASS after migration/checksum-test fixes |
| `WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test -race ./... -count=1 -timeout=120s` | PASS, all SQL/HTTP suites included |
| `make acceptance-backend`, `make acceptance-http` | PASS with expanded real HTTP cursor/bookmark scenarios |
| `docker compose build --build-arg GOPROXY=https://goproxy.io,direct` then `docker compose up --no-build -d` | PASS, rebuilt API and applied pending migrations |
| `make migrate` after Compose migration | PASS, immutable repeat/checksum verification |
| `go test ./internal/modules/user/adapters/http ./internal/modules/bookmark/adapters/http -count=1` | PASS, null rejection and sanitized failure regressions |
| Frontend `npm ci` | PASS; locked install, audit0 vulnerabilities |
| `npm test` | PASS, 44 tests (42 initial + 2 new bookmark race regressions) |
| `npm run check:foundation` | PASS, 41/41, includes current structure/contract references and 7 local skills |
| `npm run build` | PASS after final UI fix; actual JSX/import compilation |
| `BROWSER_BIN=/usr/bin/google-chrome node scripts/e2e-full-onboarding.mjs` | PASS, all24 screens, guarded direct URLs, completion only on Ready |
| `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:resume` | PASS, refresh/re-entry, saved cursor, denied jump |
| `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:bookmarks` | PASS, existing bookmark browser script |
| `BROWSER_BIN=/usr/bin/google-chrome node scripts/e2e-profile-preferences.mjs` | PASS, profile/preferences persistence and existing routes |
| `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:session` | PASS, identity/re-entry/refresh/revocation |
| `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:chat` | PASS, real chat/history/reload/isolation/delete |
| `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:qa` | PASS after full-flow and two-region Retry selector updates |
| `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:faults` | PASS, separate injected503 + genuine committed-response loss + same-ID replay/401/404/offline recovery |
| `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:integration` | PASS, new continuous cross-feature journey, mobile audit and Skip/Back checks |
| `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:integration-faults` | PASS after scroll-notice fix, progress read/write failure and bookmark failure/retry |
| `npm run check:source-baseline` | FAIL as expected: original-ZIP hash differences in App, Main, Onboarding, CSS already existed at QA baseline. Check was not weakened/deleted |
| `node --check` new E2E scripts; `gofmt` changed Go; both `git diff --check`; `cmp` OpenAPI files | PASS |
| SQL `DBCC CHECKCONSTRAINTS WITH ALL_CONSTRAINTS` | PASS, no violations |
| Physical iOS/Android keyboard; exact original-ZIP pixel regression | NOT RUN |
| External model provider | NOT RUN, explicitly excluded |

Initial FAIL results (compile, migration, null validation, races, covered Retry) are recorded above and were rerun successfully. This report does not equate a green unit suite with live acceptance.

## Database results

- Initial local ledger0001–0007; final0001–0009. 0008 SHA256 `cff3ea86c6f3b3867c85ff4404bb74e2e89f07c25952d986472e5e3f78f8afb8`; 0009 `32df7e3afa893c4384ac7d4f670cb9922d7a46f4ff5fbe26ae5284411a73b417`. Repeat runs verified ledger hashes. Original0001–0007 unchanged.
- Fresh0001–0009, repeat, populated0007 upgrade, failed migration rollback and applied-checksum rejection all executed against disposable SQL Server databases. Tests drop only their own random DB names; final inventory showed none remaining.
- Existing values are compared across the upgrade; the additive `onboarding_step` column does not masquerade as old-data corruption. Profile PK supplies one cursor per user; range constraint rejects24, forged leaps reject409, older writes do not regress, eight actual HTTP next-step requests converge and another user's cursor stays unchanged. Completed-profile status stays authoritative.
- Eight concurrent saves for user and assistant messages produce exactly two correct bookmark entries. Repeated unsave is204. Invalid UUID400, unknown/foreign404, foreign list empty, original message/conversation/content associations checked over HTTP. Deleting one conversation cascades its bookmarks and preserves another conversation's saved message. SQL fixture FK/cascade and general transaction rollback tests passed.
- Schema intentionally uses one cascading message FK for bookmarks; ownership is enforced by every parameterized repository path and session middleware, not a client userId. It does not add a new composite ownership FK to the schema.

## Development-data preservation

No existing DB reset/drop/truncate, no applied SQL rewrite, and no deletion of pre-QA conversations/messages occurred. Before QA:18 users,18 profiles,7 conversations,16 messages. Original row cohort was identified by creation before first successful0008 application:

| Original cohort | Initial count/checksum | Final count/checksum |
| --- | --- | --- |
| Users | 18 / 295810497 | 18 / 295810497 |
| Conversations | 7 / 860358906 | 7 / 860358906 |
| Messages | 16 / 1740175143 | 16 / 1740175143 |
| Profiles, original columns | 18 / -1493807375 | 18 / -1016236173 |

All18 original profiles remain. **The profile checksum is not identical:** one pre-existing synthetic tester profile was updated/completed during live test activity. No claim of byte-identical profile preservation is made. The separate disposable migration test proves the migration itself preserves original profile values. Newly generated synthetic users/profiles/cursors/session audit rows remain. Recognized conversations from failed QA attempts were cleaned through their owner API; one new conversation not matching our fixture whitelist was deliberately retained. Concurrent user activity/commits were not overwritten.

## Browser, privacy and visual evidence

The new continuous journey uses no API interception: fresh A→Welcome→24 original screens→interrupt at ordinal10→refresh→logout/close browser context→new context/same A→resume→Ready recovery of unsaved name→Home→two real turns→save assistant→History/saved item→reopen/refresh→unsave→save again→logout/login A→B completes original flow→A message/bookmark inaccessible→A returns→delete conversation→saved row disappears immediately.

Every24-screen callback checks360/390/430 widths, available Back and Skip controls, required-name validation, guarded Home/jump requests and valid chronological screen order. Profile completion is checked only after the final Ready action. Progress PUT payload keys were recorded and asserted to be exactly `furthestStep`; questionnaire responses are not persisted/transmitted there. Browser local/session storage remained empty.

Screenshots: ignored `test-results/hamrah-qa02/`, Welcome/Name/Approach/Ready and Home at360/390/430; Chat action menu, History with saved messages, Preferences at390. History/Ready images were visually inspected. Original CSS, fonts/assets, Onboarding screen source, UI primitives and flow data are byte-identical to the starting QA working tree. The only QA visual placement change is moving the existing bookmark Retry notice inside chat scrolling so it is clickable. No screen/action was deleted, hidden or broadly redesigned; no developer-mode text was added to normal UI.

Separate fault suites abort actual requests or drop a genuine committed response. They never fabricate successful session/profile/chat/bookmark results. The old fault script injects only an unsuccessful503 and then retries against actual Mock AI; no auxiliary/external model server is needed. Provider-failure SQL atomicity also remains covered by the backend local controlled fixture tests. Late bookmark lists/mutations after logout/switch are generation-fenced in Node tests; browser proves A/B state clearing. Progress read/write callbacks were reviewed for abort + identity fencing; an exhaustive browser schedule of all possible late-progress races was not performed.

Actual HTTP guards reject missing/revoked/expired/tampered cookies through existing suites. No raw-cookie/token/storage-state artifacts were written. Runtime log marker audit found zero fictional-identifier/cookie/exercised-message matches. Bookmark DB errors map to sanitized503 in handler regression tests; no SQL detail reaches clients.

## Remaining limitations

- Local synthetic identity is not real authentication; no OTP/SMS/number ownership verification or public rollout.
- The route cursor persists only the furthest screen. Name/style drafts and questionnaire answers are not restored. Actual Ready recovery was exercised; no sensitive persistence was introduced.
- Backend final profile PATCH remains the existing name/style completion contract; the cursor is a UX progression rule, not server proof that someone viewed all screens.
- Physical mobile keyboard/touch behavior and exact original-designer pixel comparison remain unverified. Existing original-ZIP hash differences are retained and reported.
- Saved/list/message limits remain100; no pagination. Message save action currently appears on assistant-message menus; HTTP supports both user and assistant bookmarks. Voice, generated insights, regeneration and other unimplemented features are not claimed successful.
- Unfinished turn UUIDs remain in memory, and conversation creation is not idempotent after a lost creation response. Distinct simultaneous turns remain an existing policy limitation.
- One original synthetic profile changed during live tests as disclosed above. No clinical safety or external-model quality claim is made.

## Files changed and patch scope

Backend: corrected previously-unapplied0008; bookmark HTTP import + new sanitizer test; progress HTTP null handling + regression; integration HTTP bookmark/cursor tests; onboarding SQL range/concurrency tests; phone upgrade old-column comparison; validation report.

Frontend: bookmark store mutation/list/deletion fencing; two Node regressions; shared full/resumed onboarding helper; corrected legacy QA/fault scripts; new continuous integration + separate network-failure E2E scripts and npm commands; moved Chat bookmark notice into scroll area; validation and this report. `contracts` and migration0009 were not changed by QA.

Fix-only artifacts at workspace root: `hamrah-qa-tasks05-06-07-backend-fixes.patch` and `hamrah-qa-tasks05-06-07-frontend-fixes.patch`. They target the saved initial working files including the user's already-applied Task07 changes. Current workspace already contains the fixes; do not reapply to it. No commits/pushes were made by this QA run.
