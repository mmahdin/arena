# Repository Map

| Path | Owner / purpose |
|---|---|
| `MVP_SCOPE.md` | Product's binding internal MVP scope |
| `AGENTS.md`, `PRODUCT_CONSTITUTION.md`, `TECHNICAL_CONSTITUTION.md` | Agent instructions & non-negotiables |
| `apps/api/cmd/api` | API entrypoint / graceful shutdown |
| `apps/api/cmd/migrate` | SQL migration entrypoint |
| `apps/api/internal/app/composition` | Dependency injection + Gin route registry |
| `apps/api/internal/modules/devsession` | **Local-only** dev user lookup and session issue/clear |
| `apps/api/internal/modules/user` | Profile and onboarding-completion state |
| `apps/api/internal/modules/conversation` | Conversation ownership, turn idempotency, history |
| `apps/api/internal/modules/ai` | Prompt policy, gateway adapter, mock and placeholder safety |
| `apps/api/internal/platform/config` | Env config + startup guardrails |
| `apps/api/internal/platform/devsecurity` | HMAC signed dev session cookie and user resolver |
| `apps/api/internal/platform/database` | database/sql connection |
| `apps/api/internal/platform/httpx` | Machine errors / HTTP constraints |
| `apps/api/internal/platform/migrations` | Transactional checksummed migration runner |
| `apps/api/db/migrations` | SQL Server forward-only SQL + synthetic seed |
| `apps/api/tests/integration` | Opt-in live SQL Server tests with an isolated disposable database |
| `scripts/run_local.py` | Load literal local `.env` values and execute native Go commands |
| `contracts/openapi/openapi.yaml` | Authored HTTP contract; **no codegen implemented** |
| `docs/adr` | Architectural decision records |
| `docs/operations` | Local start, quality checks, incident/limit notes |
| `.agents/skills/*/SKILL.md` | Named skill workflows for another coding agent |
| `scripts/check_structure.py` | Agent architecture static smoke check |
| `scripts/check_contract.py` | OpenAPI presence/basic YAML parse check |
| `scripts/new_migration.py` | Strictly increasing sequential migration file generator |
| `compose.yaml`, `Dockerfile` | Local loopback-bound SQL Server and API |

## Module rule

Use `module.go` as public constructor; put business logic in `application`, pure rules in `domain`, network handlers under `adapters/http` and persistence under `adapters/mssql`. The AI gateway intentionally has `adapters/mock` and `adapters/openai` instead of SQL repository.
