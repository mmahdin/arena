# Task 07: Internal MVP Full-Flow Acceptance

**Scope:** internal-only, synthetic data, Go/Gin + MSSQL 2022 + JSON mock AI + a separately maintained Persian RTL frontend. This is **not** a release approval, clinical validation, or evidence that real-user data is safe. No public deployment.

## What is verified automatically in this backend patch

`apps/api/tests/integration/mvp_acceptance_test.go` runs against a *new, randomly named, temporary local SQL Server database* seeded by all immutable forward migrations, and builds the real HTTP API with `LLM_PROVIDER=mock` on an ephemeral loopback HTTP listener. Its cleanup drops only that temporary database. It does **not** run against the developer's main `wellbeing` database, use a real AI provider, or introduce any production authentication.

The suite asserts:

| Flow / negative case | Expected behavior |
| --- | --- |
| Health and anonymous request | Live + ready `200`, protected profile `401 SESSION_REQUIRED` |
| Synthetic test identities | Three seeded users; session identity resolved server-side |
| Onboarding gate | Tester B starts incomplete; conversation create `409 ONBOARDING_REQUIRED` |
| Invalid profile | Blank name `400 VALIDATION_ERROR` and no state mutation |
| Profile completion and edit | Saved in MSSQL, original completion timestamp stable |
| Client reload | New HTTP client using same cookie jar retrieves persisted profile and conversation |
| Wrong-owner resource | Read / delete / write `404 RESOURCE_NOT_FOUND`, non-enumerating |
| Mock AI turns | Two user/assistant pairs saved in chronological order |
| Idempotent retry | Same `clientTurnId` / text yields exactly same stored message IDs; changed text gives `400` |
| Provider unavailable | Local unreachable provider gives `503 AI_UNAVAILABLE`, no phantom pair, retry **same** ID via mock succeeds |
| Tester switch and logout | Revocation rejects replayed old cookies; new actor cannot read previous actor's chat |
| Delete conversation | Owned delete `204`, subsequent messages `404` |

Also run the existing integration tests: lease reclaim/concurrency, ownership, session lifecycle, migration idempotency and record ordering. They live in `apps/api/tests/integration/foundation_test.go`.

## Preconditions

1. Apply **Task 02 → 03 → 04 → 05 → 06 → 07** in order to the same backend checkout. Verify `git apply --check` before applying each patch. Do not run migrations out of order; `0005_dev_sessions.sql` and `0006_turn_recovery_lease.sql` must be present.
2. Use a **private local** SQL Server 2022 instance with a DBA login that can create and delete randomly named acceptance databases. Use only `APP_ENV=local`, `ALLOW_DEV_LOGIN=true`, `LLM_PROVIDER=mock` for the running API.
3. A local `.env` supplies `DATABASE_URL`, `APP_ENV=local`, `DEV_SESSION_SECRET`, and the local SQL Server credentials. Never commit `.env` or print secrets. `scripts/run_local.py` loads `.env` literally. For these destructive *isolated-database* tests, `DATABASE_URL` must point to a **local** SQL Server address and must not be used on shared production infrastructure.
4. Install Go modules, run migrations for the local API as usual, and start the API on loopback. Frontend uses its same-origin Vite `/api` proxy. Use synthetic text only.

## Acceptance commands

Run from the backend root:

```sh
make arch-check
make contract-check
make test
make acceptance-http  # isolated MSSQL + real HTTP handlers + mock AI
make integration-test  # ALL isolated SQL Server integration tests
make smoke             # running local API, current seeded developer DB
# OR run the combined backend gate:
make acceptance-backend
```

`make acceptance-backend` **fails** if any prerequisite command fails. This target makes no claim about the frontend, live provider, or clinical safety.

For a paired frontend checkout (not included in this backend patch), check the **real backend contract** and build:

```sh
cd ../wellbeing-front
npm run check:architecture
node scripts/check-contract.mjs ../wellbeing-server/contracts/openapi/openapi.yaml
npm run test
npm run typecheck
npm run build
```

**Do not count API-stubbed Playwright specs as end-to-end evidence.** A real browser E2E test must leave the Go `/api/v1` requests *unstubbed*, run against the loopback API + real MSSQL, and cover a fresh tester, onboarding, chat, reload, switch, conversation ownership, delete and error/retry cases. Use a disposable synthetic DB or restore the synthetic fixture between runs, because browser onboarding edits the tester profile. If the frontend checkout has an opt-in live suite, enable it explicitly and record its filename, URL, browser/version and result. If it lacks one, record **BLOCKED — real browser E2E not implemented**; do not substitute its mocked tests.

## Evidence and decision record

Record in `VALIDATION_REPORT.md` **per environment and actual command**:

| Gate | PASSED / FAILED / BLOCKED / NOT RUN | Evidence |
| --- | --- | --- |
| Backend architecture + OpenAPI | | |
| Go unit tests | | |
| Isolated MSSQL API acceptance | | |
| SQL Server concurrency/lease tests | | |
| Live mock-provider API smoke | | |
| Frontend contract/typecheck/build | | |
| Real browser, unstubbed API + MSSQL | | |
| Provider live synthetic-only test (optional) | | |
| Non-clinical safety regression tests | | |

**MVP internal acceptance:** required gates are backend, local SQL Server, frontend and unstubbed browser integration using mock AI; failure of any required gate means *not accepted*. Real LLM smoke is separate and optional. No amount of green local tests authorizes use with real mental-health disclosures, external users, public networking or clinical purposes.

## Known MVP limitations

- Mock AI is predictable only for technical flow testing; real provider behavior and costs are not proven here.
- The initial rules-based mental-health safety checks are not clinically validated; document detected and missed synthetic cases.
- History caps at 100 messages; no cursor pagination or SSE streaming.
- Dev Login is a synthetic identity switcher, **not** a production authentication mechanism.
- Frontend and browser tests live in a separate repository. This backend-only patch cannot certify them.
