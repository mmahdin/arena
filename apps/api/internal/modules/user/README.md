# User Module

Owner of user profile and onboarding-completion state. `user_profiles` references the stable internal `users.id` identity; **never** infer an individual's medical history or psychological traits.

- **Domain:** trimmed name length 1–60 Unicode code points; reject control and Unicode direction-override characters while allowing Persian ZWNJ. Five independent style values `balanced|listening|teaching|gentle_guidance|collaborative` (plus supported legacy `guidance`); only the first valid profile save completes onboarding.
- **Application:** Get/Update by server-resolved internal UUID; no user ID in API body.
- **Persistence:** SQL Server `user_profiles`; see `0001_initial_schema.sql`. Onboarding completion time is immutable at SQL level (`COALESCE`) even under racing writes; the application returns the saved row.
- **HTTP:** GET `/api/v1/me`, PATCH `/api/v1/me/profile` (full name + style, validation 400). Both derive the target user from the active server-side dev session.
- **Onboarding name:** PATCH `/api/v1/me/onboarding-name` accepts only `displayName`, without completing onboarding. The shared JSON decoder rejects unknown fields and trailing JSON with `400 INVALID_JSON`, preserving the existing 16 KiB body limit. Invalid requests never update the profile.
- **Chat gate:** creating a conversation or submitting a turn before onboarding returns `409 ONBOARDING_REQUIRED`; a foreign conversation still yields 404.
- **Boundary:** conversation reads profile through `conversation/application.ProfileReader` port, not by joining its private table.
- **Tests:** unit profile validation; add SQL Server integration checks and wrong-session tests.

Validation is reversible before save: invalid inputs never mutate profile state or mark onboarding complete.

## Explicit onboarding context (2026-11-11)

The user approved storing explicitly selected onboarding answers for chat personalization.
Optional `onboardingAnswers` in PATCH `/me/profile` replaces the selections; omission
preserves them (including concurrent saves), and `{}` clears them. Strict JSON rejects
unknown answer keys. Per-field, list, mood, identity and aggregate bounds are validated
before any write. Migration 0016 adds nullable JSON on the existing owner-scoped profile.
No answers are inferred from historical conversations. GET `/me` returns saved answers;
conversation reads a validated, read-only consumer-owned view via composition. Declined
disclosures never enter the model context. See the onboarding context integration note.
