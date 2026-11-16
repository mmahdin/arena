# Saved messages (Hamrah Task 07)

Local synthetic-only, user-owned message bookmarks. `GET /me/bookmarks` lists up to 100; idempotent `PUT/DELETE /messages/:messageId/bookmark` uses the authenticated cookie owner. Wrong-owner or removed messages yield the same 404. Bookmarks reference existing messages rather than duplicating content; conversation deletion cascades to bookmarks. Migration `0009_saved_messages.sql` is forward-only. No real-person data or clinical release.
