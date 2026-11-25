# PROD-01 — Synthetic product event ledger (SQL Server 2022)

**Deployment:** Apply migrations `0035_product_events.sql` and `0036_product_events_preservation.sql` before starting the updated API.

## Purpose and privacy boundary

Server-owned, immutable-in-normal-use **event facts**, recorded inside the **same SQL transaction** as the successful business mutation. These are **not** analytics claims about real users and are **not** clinical/safety metrics. There is no public event-ingestion endpoint, free-form event JSON, JavaScript autocapture, session replay, client IP, telephone, email, Identity Provider subject, onboarding answer, or AI text. The internal `user_id` UUID is needed to support deduplication/cohort analysis later; access to base rows must be restricted. It is not an anonymization guarantee.

The central `productevents.Record` helper applies `users.is_test=1 AND users.deleted_at IS NULL` in SQL. A SQL Server `AFTER INSERT` trigger also rejects direct attempts to insert events for non-test or soft-deleted users. **Real-user event collection is intentionally disabled** until privacy, product, security, and clinical reviews authorize a specific new scope. The event table does not expose raw users or content to Grafana, Prometheus, Loki, Phoenix, or a User API. Synthetic data are still subject to good security practices.

## Exact event definitions (version 1)

| Event | Owning transaction | Subject ID | Meaning |
| --- | --- | --- | --- |
| `user_registered` | New synthetic Keycloak account | User | Created a previously nonexistent synthetic user; legacy fixture insertions are **not** backfilled |
| `onboarding_started` | First persisted onboarding progress transition **step 0 → 1** | User | At least one real cursor advancement; **not** first screen view or a login |
| `onboarding_completed` | First profile transition NULL → non-NULL `onboarding_completed_at` | User | First successful onboarding completion; later profile edits are excluded |
| `conversation_started` | New conversation insert | Conversation | A new conversation record was committed; **not** number of active chats |
| `ai_turn_admitted` | New v1 turn + user message admission | Turn | Distinct admitted logical turn; duplicate `clientTurnId` never counts twice |
| `ai_turn_completed` | Turn finalization + assistant message transaction | Turn | First successful logical turn completion; replay excluded |
| `ai_turn_failed` | First failed terminal state transition | Turn | Turn ever failed at least once; explicit later retry may eventually succeed; **not** provider attempt failures or current failed-turn count |
| `bookmark_saved` | New saved-message insertion | Message | First saved message; repeated PUT not counted |

Event identity is unique `(event_name, subject_id)` and `occurred_at` comes from SQL `SYSUTCDATETIME()`. Later retries/failures of the same Turn are deliberately collapsed. Both `ai_turn_failed` and `ai_turn_completed` can exist for the same Turn if an authorized retry succeeds; neither is a billing measure. Current Turn statuses must still come from `dbo.turns` for operational failure rates.

## Tables and reporting

- `dbo.product_events`: fixed event catalog, `event_name`, `user_id`, `subject_id`, `occurred_at`, `schema_version=1`, deduplicated by unique constraint. Subject type depends on event type; the business transaction is the source of ownership. The FK to `users` has **no physical delete cascade**, preserving the established soft-delete policy.
- `dbo.v_product_daily_events_synthetic`: UTC daily counts (`event_count`) and distinct synthetic users **per event per day** (`distinct_test_users`), **without** identifier columns.
- This view is **not** true DAU, retention, funnel conversion, or historical fact backfill. Daily active users, cohorts, retention windows, and denominator rules require an explicit later metric specification. Counts start when the migration and application code are deployed. They are unsuitable for managerial claims about real user adoption.
- PROD-02 may connect Metabase only to explicitly granted aggregate/reporting views with **read-only SQL permissions**. Do **not** grant Metabase direct access to `dbo.product_events`, `dbo.users`, or messages. Analytics events remain on SQL Server; do not mix them with Prometheus histograms or LLM cost estimates.

## Operational semantics

- Event storage is **transactional and fail-closed**: if the event insert fails, the business transaction rolls back. Migration **0035 must precede deployment** of this backend version. There is no background queue or silent best-effort write.
- A source operation that makes no new state transition does not emit an event.
- No backfill of historical users, conversations, bookmarks or Turns in this migration. Adding one later requires a separately reviewed, idempotent plan.
- Schema does not contain arbitrary `properties`, client-provided event codes, free text, `session_id`, `request_id`, or PII-derived hashes.
- Store identifiers are internal and owner-scoped. The only reporting view intentionally reveals date/event/counts.
- Soft-delete preserves events by current retention rules. Product analytics deletion/retention for real people is **not** solved by this prototype; legal erasure and user data access policy must precede real-user enablement.

## Pending / future milestones

1. PROD-02: reviewable synthetic reporting metric definitions and least-privilege Metabase integration.
2. Approved real-user analytics: separate privacy/security/legal/clinical reviews, consent or other lawful basis, role separation, retention and erasure policy, minimum-cell-count suppression and access logging.
3. Additional product events (feature-specific events, consent-respecting activation): new enumerated event, CHECK migration, domain integration test; never arbitrary frontend event upload.
