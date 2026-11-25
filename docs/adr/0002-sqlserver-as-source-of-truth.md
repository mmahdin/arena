# ADR-0002 — Use Microsoft SQL Server 2022 for Wellbeing MVP

- **Status:** accepted (supersedes ADR-0001 database choice)
- **Date:** 2026-11-02
- **Decision:** Microsoft SQL Server 2022 (Developer container for local synthetic testing), Go `database/sql` with `github.com/microsoft/go-mssqldb`, `uniqueidentifier` IDs, `nvarchar` Unicode text, `datetime2(3)` UTC timestamps, forward-only numbered T-SQL migrations.
- **Why:** Team's actual database infrastructure is MSSQL and architectural consistency with the reference backend is preferred over the earlier PostgreSQL placeholder.
- **Technical consequences:** `adapters/mssql` in user, devsession and conversation; `@pN` parameters, `OUTPUT`, `TOP`, transactional idempotency, checksum ledger and SQL Server `sp_getapplock` migration coordination. Avoid SQL Server multiple-cascade-path errors by using a single ownership cascade.
- **Local bootstrap:** Migration command may connect to `master` and create the `wellbeing` database only when `APP_ENV=local && MSSQL_AUTO_CREATE_DATABASE=true`. Provision production databases separately.
- **Deployment limitations:** Compose uses loopback-bound API and SQL ports and a synthetic data seed. Dev session is not actual authentication; TLS for SQL driver is disabled only for local container; no real/private mental-health data allowed.
- **Verification required:** Compile and test with networked Go modules, boot SQL Server 2022, run both migrations, rerun for checksums, test foreign-key cascades and cross-user isolation under concurrent requests, smoke the API. Static source checks do not prove live database compatibility.
