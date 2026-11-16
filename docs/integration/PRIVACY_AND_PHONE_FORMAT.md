# Privacy preference and phone-formatted local identities

- `0010_privacy_preferences.sql` adds a **bit preference** to `dbo.user_profiles`; it defaults to 1 (enabled) for the **internal synthetic-data prototype**. Existing profile rows are backfilled without dropping their content.
- `GET /api/v1/me/privacy-preferences` and `PUT` with `{"conversationSharingEnabled":true|false}` access only the signed-session user; no client-supplied `userId` is accepted.
- **There is no actual conversation-sharing action, destination, export, analytics or background job** connected to this setting. Default `true` is not proof of informed consent. Any future release or actual sharing of user content must **default to disabled** and obtain separate, affirmative, revocable consent with an audit record.
- The internal dev identity input now accepts any `09xxxxxxxxx` 11-digit phone-shaped value, including Persian/Arabic numerals. It still does NOT authenticate ownership, send OTP/SMS or provide public access. Existing HMAC identity mappings for old inputs remain stable; use synthetic test values only. Do not enable on public hosts or accept real phone numbers with this dev endpoint.
- Tests: `python3 scripts/check_contract.py`, `cd apps/api && go test ./...`, and optional `WELLBEING_INTEGRATION=true ... go test ./tests/integration` against an isolated local MSSQL 2022 instance.
