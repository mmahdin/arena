# Personalized Letters (internal synthetic testing only)

- Explicit user intent: `POST /api/v1/me/letters` with `{"consent":true}`; nothing is inferred or generated in a background job.
- Require at least three completed user turns in one user-owned conversation. Select the latest eligible conversation and send a bounded (last 20 messages / max 12,000 characters) transcript to the configured OpenAI-compatible model.
- A valid result is short structured Persian JSON. Invalid/unsafe responses return an error; no fabricated mock-personalized letter is ever shown or stored.
- Owner-scoped `GET /api/v1/me/letters` and `DELETE /api/v1/me/letters/:id` support archive and deletion. Unique source conversation makes POST retry idempotent for saved letters (concurrent calls can still cost multiple upstream requests).
- `0015_personalized_letters.sql` stores authored letters with a composite conversation-owner FK `ON DELETE CASCADE`; deleting the source conversation deletes its letter. The existing diary table and migration remain unchanged.
- No new traits, inferred personality profile, cross-thread memory, clinical scores, or unattended letters are created.
- This internal synthetic prototype is **not approved for real disclosures**. Before real use, add independent safety evaluation, proper authentication, data-controller and provider review, auditable consent/retention, throttling, and stronger generation-quality gating.
