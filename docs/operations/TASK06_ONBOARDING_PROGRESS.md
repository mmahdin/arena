# HAMRAH-INTEGRATION-06 — Durable onboarding cursor

This backend change adds migration `0008_onboarding_progress.sql` and the session-owned `GET`/`PUT /api/v1/me/onboarding-progress` operations. It is for private synthetic-data testing. The original Hamrah **24-step UI flow** is unchanged. The server persists only an ordinal `0..23` in `dbo.user_profiles.onboarding_step` with a range constraint. It NEVER stores questionnaire answers (health, faith, age, relationships, therapy, neurodivergence), voice choice, or message content in this field. Only `displayName` and `conversationStyle` remain supported by the existing final `PATCH /me/profile`.

- Index `0` is Welcome; `23` is Ready, in frozen `hamrah-v1` ordering. An incompatible future flow requires a versioned migration/API decision.
- GET returns `{ "furthestStep": N, "flowVersion": "hamrah-v1" }` for the cookie's user. Completed profiles still use `onboardingCompletedAt` as the actual completion gate; their cursor is not authoritative.
- PUT accepts exactly `{ "furthestStep": N }`. Unknown keys, including `answers`, are rejected. The conditional SQL UPDATE accepts only `current+1` when incomplete; requesting a previously reached index is idempotent and **never lowers** the saved cursor. A jump returns 409 `ONBOARDING_STEP_CONFLICT`; invalid input 400 `INVALID_ONBOARDING_STEP`; completed profile 409 `ONBOARDING_ALREADY_COMPLETED`; missing cookie 401.
- All queries are scoped using the signed, server-resolved user UUID; no client user ID is accepted. `onboarding_step` is non-null with default zero for existing rows. No legacy migration is rewritten or data erased.
- This is intentionally a *progress-cursor feature*, not persistence for the long sensitive questionnaire or a consent policy. Initial completion through PATCH /me/profile remains backward-compatible even if an older client never called the progress endpoint; a separate release review may tighten this gate.

## Validation required in a live local stack

Apply migration 0008 with the existing runner; repeat to confirm immutable checksums. Run `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py`, `cd apps/api && go test ./...`; with a real LOCAL SQL Server use `WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test ./tests/integration -count=1`, and run the existing backend acceptance suites. The new integration test creates/drops only a random disposable DB: verifies forward-only, repeat/back, concurrent calls, cross-user isolation, completion protection and a fresh schema. Verify upgrade of the existing development database separately without resetting it. Do not run against public/shared/production databases.

## Limitations

A cursor cannot recover answers not stored by design; the user may revisit the original Name and Approach screens before finishing. Server-side profile completion remains compatible with pre-task clients. The onboarding gate and synthetic dev identity are not public authentication, clinical consent or approval to process real sensitive answers.
