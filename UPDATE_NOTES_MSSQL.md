# MSSQL & Skill Naming Update — 2026-11-02

## User decisions implemented

1. The database is **Microsoft SQL Server 2022**, not PostgreSQL. Use `github.com/microsoft/go-mssqldb` through Go `database/sql`. The module layout `domain -> application -> adapters/mssql -> composition` remains unchanged.
2. All four original skills are renamed to product-owned entrypoints:
   - `wellbeing-backend-module`
   - `wellbeing-api-contract`
   - `wellbeing-database-migration`
   - `wellbeing-permission-review`
   - `wellbeing-ai-prompt-safety` and `wellbeing-dev-session` continue unchanged.
3. Skill names, frontmatter, metadata, references, docs, `AGENTS.md` and structure checker match the new names. Placeholder skill script/assets removed.
4. `compose.yaml` runs SQL Server 2022 Developer, with local-only volume `wellbeing_mssql`, port `127.0.0.1:14333`, API `127.0.0.1:8080`.
5. SQL migrations (`0001`, `0002`) are now T-SQL with `uniqueidentifier`, `nvarchar`, `datetime2(3)` UTC and a synthetic-only seed. The migration runner uses `sp_getapplock`, SHA256 ledger and a transaction per file. SQL Server identity bootstrap is enabled **only** in the local migration binary.
6. Repository code now uses SQL Server `@pN` parameters, `TOP`, `OUTPUT`, `UPDLOCK/HOLDLOCK`, `CAST(... AS bit)` and `SYSUTCDATETIME()`. Ownership checks and idempotent message turns remain part of the design.
7. ADR-0002 supersedes ADR-0001 only for database selection. `MVP_SCOPE.md` and READMEs now state the decision.

## What is NOT verified

- The full Go dependency graph cannot be fetched in this environment; `apps/api/go.sum` needs `go mod tidy` in a networked environment.
- A live SQL Server 2022 instance and Docker runtime were not available for DDL/application integration tests. **Do not assume migrations or simultaneous-claim behavior are verified** until running `docker compose up --build -d` and `python3 scripts/smoke_api.py`, plus database-specific concurrent/constraint tests.
- This remains a **synthetic local developer MVP**, not a deployable service for real mental-health data.

## Initial external developer run

```sh
cp .env.example .env
# Replace DEV_SESSION_SECRET and MSSQL_SA_PASSWORD; keep them out of git.
docker compose up --build -d
python3 scripts/smoke_api.py
cd apps/api && go mod tidy && go test ./...
```

Use `AGENTS.md` as the authoritative agent entrypoint.
