# PROD-01 — Local acceptance and operational runbook

**Prerequisite:** Backend through MON-02, SQL Server 2022 local disposable test database, offline-safe source tests. The event vocabulary is additive. **No public API/OpenAPI changes.**

## Deployment order

1. Run `go run ./cmd/migrate` from `apps/api` with the existing database configuration. The runner applies `0035_product_events.sql` and `0036_product_events_preservation.sql` in order and records their checksums. Migration `0036` adds the repository's physical-delete guard to the ledger.
2. Rebuild the backend and run `go test ./...`. Do not treat passing Python/structural tests as a successful Go compile.
3. Start with synthetic test users. Exercise new account creation, onboarding progress and completion, conversation creation, successful AI Turn, failed AI Turn, and saved-message PUT twice.
4. Check new events are exactly once per subject. Check no event for a wrong-owner operation or duplicate `clientTurnId`.
5. Inspect the aggregate view, not the base table, when preparing management examples.

## Local checks

```bash
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -q
cd apps/api
go mod tidy
go build ./...
go test ./...
```

The Go integration fixture uses the **existing disposable** harness (`WELLBEING_INTEGRATION=true`, `APP_ENV=local`, loopback SQL Server URL). It creates and deletes its own test database. Never use a shared or production connection.

To run the focused SQL acceptance test with these settings, use
`go test -count=1 -run '^TestPROD01SyntheticProductEventLedger$' ./tests/integration`
from `apps/api`.

Read-only illustrative aggregate query, in SQL Server 2022:

```sql
SELECT activity_date_utc, event_name, event_count, distinct_test_users
FROM dbo.v_product_daily_events_synthetic
ORDER BY activity_date_utc DESC, event_name;
```

Confirm that `ai_turn_admitted` is one per logical Turn, `ai_turn_failed` is first failure only, and later repeated completions/bookmarks/profile updates do not add events. The view is **synthetic-only** and does not establish DAU or clinical outcomes.

## Failure behavior and rollback

- Event insertion is required by the **same DB transaction** as its source mutation. Without migration 0035 the source operation fails and rolls back; never deploy code before schema.
- On a SQL error, investigate schema/version/storage/locks. Do not log raw request bodies, identity keys, or user-submitted strings.
- Roll back code by redeploying the previous backend; leave the additive schema in place. Do not delete event rows or drop the migration in an already migrated environment.
- If real users enter the system, these event writes intentionally create **zero** analytics rows for them. Do not change `is_test` filtering without a new, explicitly reviewed authorization.

## No-go gates

- Go build/tests are not green.
- SQL 2022 migration or disposable integration test fails.
- Any synthetic event leaks raw text, phones, IPs, model prompts, credentials, or arbitrary properties.
- Reporting reads require grant to base tables rather than safe aggregate views.
