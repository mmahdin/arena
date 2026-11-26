# ADR-002: sensitive intake questions are not active backend data yet

- **Status:** Temporary internal MVP restriction; re-evaluate when extending product.
- Existing visual intake screens include age range, therapy background, neurodivergence, faith, relationship and wellbeing topics. They are **designer intent**, not approved collected data.
- Currently backend `/me/profile` only supports display name and conversation style. Don't add hidden fields, send answers to LLM or keep sensitive answers in browser/local telemetry.
- In connected internal flow use a short approved path and keep longer onboarding available for non-persistent design preview, preserving its appearance.
- Before activating: consent/legal basis, minimum field set, field purpose, scope of retention/deletion, access controls, age eligibility, clinical-risk review, migration and new API contract, then test.
