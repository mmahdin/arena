# Tone Refinement — five independent choices

Migration `0011_five_conversation_styles.sql` expands the existing SQL Server check constraint without changing user data. Accepted new values: `balanced`, `listening`, `teaching`, `gentle_guidance`, `collaborative`. Legacy `guidance` remains valid and receives the gentle-guidance prompt for backward compatibility; a user can choose the new value on their next profile update. The five UI selections each map to a unique saved string.

`GET /api/v1/me` and `PATCH /api/v1/me/profile` retain their shapes; OpenAPI enums expand. Gin profile handlers, session cookies, owner-scoped SQL, and chat turn JSON contract are unchanged. Conversation turns load the latest saved profile before generation, so a new turn uses the changed prompt. Completed/idempotent turns retain original `promptVersion`. `supportive-v4` contains a shared safety system policy plus a server-owned allowlisted style directive; untrusted profile JSON remains data.

**Tests:** `go test ./internal/modules/user/... ./internal/modules/ai/...`, full `go test ./...`, `python3 scripts/check_contract.py`, fresh/upgrade/reapply migration on disposable MSSQL 2022, 5-style profile roundtrips, current-prompt capture, mock-provider E2E for data persistence. Real external-provider variation requires separate approved synthetic-only evaluation; deterministic mock responses do not prove output variation.

Internal-only synthetic-data prototype; no real user memory, clinical inference, or public authentication.
