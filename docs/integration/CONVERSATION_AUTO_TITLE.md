# HAMRAH Conversation Auto-Title + Manual Rename

Target baseline: accepted Task08 (migrations through `0012_daily_reflections.sql`), with previous phone/privacy, visual transition and five-tone changes retained.

## Behavior

- The existing Create endpoint still returns `گفت‌وگوی جدید` before a turn is committed.
- On the first **successfully committed** user+assistant turn, the server derives a deterministic short title from that user's first text (no AI request). The title is written in the **same SQL transaction** as the turn. No failed attempt, processing reservation or duplicate replay sets a title prematurely.
- A manually renamed conversation will not be auto-titled again. Existing conversations with non-placeholder titles are not modified. Existing conversations still bearing the placeholder remain unchanged until manually edited (their later turns do not count as the first turn).
- Input is trimmed, whitespace normalized, common URL/email/long numeric identifiers suppressed, truncated to <=60 Unicode code points; original stored message never changes. This is a lightweight heuristic, not private-data detection; synthetic-only internal use.
- `PATCH /api/v1/conversations/{id}` `{"title":"…"}` updates an owned title, marks `title_is_manual=1`, returns `Conversation`. Unknown/foreign ID: 404; empty/over-60/control characters: 400; unsigned session: 401.
- The title remains in existing `GET /api/v1/conversations` responses; no extra LLM call, telemetry, indexing or inferred memory.

## Schema

Forward-only MSSQL migration `0013_conversation_manual_titles.sql` adds `conversations.title_is_manual bit NOT NULL DEFAULT 0`. No existing rows are changed; `conversations.title nvarchar(150)` remains as-is. Existing migrations must not be edited.

## Validation

- Run: `python3 scripts/check_structure.py`; `python3 scripts/check_contract.py`; `cd apps/api && go test ./...`.
- With `APP_ENV=local`, local SQL Server, `DATABASE_URL`, and `WELLBEING_INTEGRATION=true`: `cd apps/api && go test ./tests/integration -run TestConversationTitlesSQLServer -count=1`.
- This test uses its own disposable database. Never run migrations against shared/production data.
- Frontend browser E2E: `npm run test:e2e:titles` with local Vite and mock-AI Go/MSSQL running.

## Design and compatibility

Designer History row retained; only an inline pencil edit action added. `PATCH` is additive; old clients continue to function. Do not infer long-term memories or expose real sensitive chats under this local dev session.
