> **Historical reference (v1).** This document describes an earlier onboarding flow. The active implementation is **v2**; see [ONBOARDING_V2.md](ONBOARDING_V2.md). Do not use the route indices below for current development.

# Temporary onboarding exclusions

The user requested excluding Testimonials, Research and Experts from onboarding without deleting their pages. `EXCLUDED_ONBOARDING_ROUTES` in `src/features/profile/navigation.js` holds this temporary configuration. The remaining21 screens retain their original order. Welcome → Name; Back from Name → Welcome. The three components, assets, original `FLOW`, and completed-user About entry remain present.

The server still uses `hamrah-v1` ordinals0–23. Name remains4 and Ready23. Welcome→Name advances1,2,3,4 through ordinary sequential PUTs; no endpoint/schema/migration change. Acknowledged progress is retained after partial failure and the same step can be retried safely. Saved cursors1–3 advance to4 before rendering Name. Incomplete-user direct links to excluded pages resolve to the unlocked active screen. Abort and identity-generation guards fence old requests.

Node regressions cover active navigation, canonical ordinals, sequential PUTs, partial failure/retry and cancellation. Browser helpers/full-flow/resume tests now use the enabled screen list. No chat/model requests are needed for these tests.
