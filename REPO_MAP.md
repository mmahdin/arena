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
| `apps/api/internal/modules/ai/contract` | New WP-01 AI Core types and embedded strict LLM schemas; independent of the transitional prototype |
| `apps/api/internal/modules/ai/ports` | New AI Core consumer-owned interfaces and reusable adapter contract tests |
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

## AI Core WP-02 persistence

- `apps/api/db/migrations/0017_ai_turn_lifecycle.sql` through `0019_ai_owner_integrity.sql`: additive lifecycle/state/owner/privacy foundation.
- `apps/api/internal/modules/conversation/adapters/mssql/lifecycle.go`: raw admission, FIFO, replay/recovery and finalization UoW.
- `apps/api/internal/modules/ai/adapters/mssql/`: state, leases and embedding storage codec.
- `apps/api/internal/app/composition/ai_persistence.go`: infrastructure composition, no runtime cutover.
- `apps/api/tests/integration/ai_lifecycle_test.go`: live SQL Server contract, migration, race and privacy tests.
- `docs/adr/0009-ai-core-sql-lifecycle.md`, `docs/quality/ai-core-wp02.md`: operational contract and verification.

- `db/migrations/0020_soft_delete_preservation.sql` (under `apps/api`): non-destructive metadata, active uniqueness, NO ACTION owner references and physical-delete guards.
- `docs/adr/0010-soft-delete-preservation.md`: binding WP-02 deletion correction; retention and visibility are independent of AI-use policy.
- `apps/api/tests/integration/soft_delete_test.go`: preservation, visibility, re-creation, migration, rollback and delete/finalize concurrency tests.
