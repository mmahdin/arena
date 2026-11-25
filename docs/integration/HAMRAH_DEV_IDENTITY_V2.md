# Hamrah dynamic synthetic identity — implementation notes

Local synthetic-only v2 changes: `POST /api/v1/dev/identity-session` accepts fake `0900000XXXX` values with Arabic/Persian digits and provides an HttpOnly revocable cookie. New accounts get an incomplete profile; existing ones retain UUID and conversation ownership. No real phone, SMS or OTP.

`0007_dev_phone_identities.sql` maps a keyed HMAC digest of the normalized synthetic string to a UUID, without raw phone persistence. Preserve `DEV_SESSION_SECRET` across local restarts; rotating it makes old mappings undiscoverable by the typed identifier. The backend find-or-create+initial profile+session transaction is serializable and intended to tolerate concurrent first logins.

**Patch author validation:** `python3 scripts/check_structure.py` PASSED, `python3 scripts/check_contract.py` PASSED, gofmt applied. **NOT RUN:** Go tests due to unavailable cached Go modules; SQL Server tests due to missing Docker. Re-run `go test ./...`, `make integration-test`, `make acceptance-backend` on the actual local validated checkout, then the frontend E2E (no API interception). Do not claim Go/MSSQL success from the checks above.

Patch is authored from Task07 baseline, but deliberately does not modify validated-fix files (`VALIDATION_REPORT.md`, `0006_turn_recovery_lease.sql`, conversation MSSQL lease adapter, OpenAI deadline fixture, or existing integration files). Keep the user's prior validation fixes applied.
