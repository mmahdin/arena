# HAMRAH-INTEGRATION-07 — Saved Messages

Preserve the authoritative Hamrah designer UI. The existing `ذخیره پیام` action in Chat opens the real bookmark API and toggles into `برداشتن از ذخیره‌شده‌ها`. The existing History page additionally shows saved-message snippets linking to their conversation. No existing screen or visual asset is deleted; unsupported Regenerate, Insight, Voice, Report and attachments remain in place.

The backend `GET /api/v1/me/bookmarks` is authoritative. `PUT` and `DELETE /api/v1/messages/{messageId}/bookmark` are idempotent, use `credentials: include` and owner-scoped SQL. Frontend never saves message content or session identity in browser storage. Account switches/logout abort stale requests, clear saved-message state and revalidate the server session on 401.

`Bookmark` is only selectable after the saved list has loaded; on load failure the user can retry. Save failure never displays success. Unsave removes a bookmark, not the original chat message. Deleting a conversation removes its bookmark records in the database. The list shows only the last 100 bookmarks.

Run `npm test`, `npm run check:foundation`, `npm ci`, `npm run build`. With the actual localhost Vite, Go and SQL Server services and installed Playwright run `node scripts/e2e-bookmarks.mjs`. No endpoint interception and only synthetic accounts/messages.
