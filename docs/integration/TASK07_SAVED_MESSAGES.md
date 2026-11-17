# HAMRAH-BACKEND-07 — Saved messages

An additive, local-only, synthetic-data vertical slice. The original Hamrah UI action remains in Chat; message bookmarks are displayed in the existing History screen, never as invented AI insights.

## API

- `GET /api/v1/me/bookmarks` returns `{items: SavedMessage[]}` (most recent 100).
- `PUT /api/v1/messages/{messageId}/bookmark` returns `200 {saved:true}`, idempotent.
- `DELETE /api/v1/messages/{messageId}/bookmark` returns `204`, idempotent on an existing owned message.
- Invalid UUID → `400 INVALID_ID`; absent/foreign/deleted message → nonenumerating `404 RESOURCE_NOT_FOUND`; absent/revoked session → `401`.
- Only the server-resolved signed dev session owns these resources; no `userId` parameter.

## Database and retention

Migration `0009_saved_messages.sql` creates `dbo.saved_messages` keyed by original message UUID, a user-owned index, and a single cascading message FK. There is no duplicated message text in the table. Migration 0020 supersedes the delete cascade with a validated NO ACTION reference. Conversation soft deletion hides bookmarks through the source conversation predicate while preserving every record. Migration is forward-only and must never be edited after application. Deleted bookmarks remain stored; a later save creates a new active row with the same message reference. The list contains message content only for the requesting owner. SQL logs must not include content or cookies.

## Local validation

1. Apply migration on disposable SQL Server 2022 DB, then run it again to verify ledger/checksums.
2. `cd apps/api && go test ./internal/modules/bookmark/... ./tests/integration/... -count=1`.
3. `WELLBEING_INTEGRATION=true ... go test ./tests/integration -run TestSavedMessagesSQLServer` using an approved **local** database URL; test creates/drops its own separate DB.
4. Run `python3 scripts/check_structure.py` and `python3 scripts/check_contract.py`.
5. Run UI browser suite with Go API, MSSQL, Vite and Mock AI. No public users, SMS, or external model.

**Limitations:** saved list capped at 100; no pagination; no cross-device identity/authentication outside local tester sessions. No change to clinical/privacy release status.
