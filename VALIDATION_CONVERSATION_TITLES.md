# Conversation titles — validation

- PASS: `python3 scripts/check_structure.py` — 41 required paths.
- PASS: `python3 scripts/check_contract.py` — 25 routes aligned with OpenAPI.
- PASS: Go syntax formatting (`gofmt`), static T-SQL dialect check.
- ADDED, NOT EXECUTED: Go domain/application tests, isolated MSSQL integration (`TestConversationTitlesSQLServer`), browser E2E (`test:e2e:titles`).
- BLOCKED: Go unit/build tests with `GOPROXY=off` because Gin, UUID and go-mssqldb are absent in local module cache.
- NOT RUN: live SQL Server, browser visual/E2E, production build. Local synthetic-only scope.
