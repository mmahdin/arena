# Personalized Letters (internal synthetic testing only)

- Explicit user intent: `POST /api/v1/me/letters` with `{"consent":true}`; nothing is inferred or generated in a background job.
- Require at least three completed user turns in one user-owned conversation. Select the latest eligible conversation and send a bounded (last 20 messages / max 12,000 characters) transcript to the configured OpenAI-compatible model.
- A valid result is short structured Persian JSON. Invalid/unsafe responses return an error; no fabricated mock-personalized letter is ever shown or stored.
- Owner-scoped `GET /api/v1/me/letters` and `DELETE /api/v1/me/letters/:id` support archive and deletion. Unique active source conversation makes POST retry idempotent for visible letters (concurrent calls can still cost multiple upstream requests).
- `0015_personalized_letters.sql` remains immutable. Migration 0020 replaces its cascade with a validated owner FK and adds soft-delete metadata plus filtered active source uniqueness. Deleting a letter hides it and preserves its contents; re-creation creates a new row. Deleting its source conversation does not hide or invalidate the letter. New generation selects visible conversations/messages.
- No new traits, inferred personality profile, cross-thread memory, clinical scores, or unattended letters are created.
- This internal synthetic prototype is **not approved for real disclosures**. Before real use, add independent safety evaluation, proper authentication, data-controller and provider review, auditable consent/retention, throttling, and stronger generation-quality gating.
