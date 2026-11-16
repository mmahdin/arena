# ADR 0004 — Onboarding profile and chat gate

**Status:** Accepted for internal synthetic-data MVP (Task 03)

## Decision

- Keep the existing `GET /api/v1/me` and `PATCH /api/v1/me/profile` contract. Do not add an onboarding controller, new table, or duplicate persistence layer.
- A first **valid** profile update records `onboardingCompletedAt` once. Future edits can change display name and conversation style but must not change this timestamp.
- The backend blocks conversation creation and AI turn submission before onboarding is complete with `409 ONBOARDING_REQUIRED`. Listing/deleting previously created synthetic data is still allowed for cleanup.
- Ownership checks remain authoritative: an incomplete tester attempting a turn in someone else's conversation receives 404, not 409. No client-provided user ID is trusted.
- Completion status comes from the same user-module read port already used to select the AI prompt style. No conversation-module import of user internals.
- Reject blank/overlong names, control characters and directional overrides; preserve legitimate Persian ZWNJ. Names and styles can be edited without collecting any additional mental-health information.

## Why

The prior backend marked onboarding complete in the domain model, but conversation endpoints could be called directly by an incomplete tester. Stored completion time was also vulnerable to being overwritten by parallel initial saves. This small change closes those gaps while preserving frontend request and response shapes.

## Compatibility, rollout and limits

- No new SQL migration required: `dbo.user_profiles` already has the necessary columns and constraints. The existing SQL update now uses `COALESCE` to preserve completion under concurrent saves.
- The added `409 ONBOARDING_REQUIRED` is an **additional runtime error** on POST conversation/turn, not a request/response breaking change. Frontend should redirect to onboarding on this error.
- Existing testers with a completed profile continue unchanged; seeded tester 2 remains incomplete until the first valid save. The first profile save is immutable only for the completion timestamp, not other profile fields.
- For internal private development with synthetic data only. This does not implement human identity, user consent, mental-health care or public readiness.
