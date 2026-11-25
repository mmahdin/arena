# How an Agent Should Implement a Feature

1. Read `AGENTS.md` and the four source-of-truth documents listed there.
2. Identify the feature and owning module; confirm it is **in MVP_SCOPE.md**.
3. State invariants, actor scope, data sensitivity, idempotency and expected HTTP errors.
4. If endpoint changes: read `.agents/skills/wellbeing-api-contract/SKILL.md` and edit authored OpenAPI first.
5. If DB changes: read `.agents/skills/wellbeing-database-migration/SKILL.md` and generate a new migration.
6. For code: read `.agents/skills/wellbeing-backend-module/SKILL.md`; implement domain -> application -> adapters -> composition.
7. For AI behavior: read `.agents/skills/wellbeing-ai-prompt-safety/SKILL.md` and add regression cases. No real sensitive prompts in tests.
8. For identity/ownership: read `.agents/skills/wellbeing-permission-review/SKILL.md` and `.agents/skills/wellbeing-dev-session/SKILL.md`.
9. Test a happy path, wrong-user read/write, invalid input, provider failure, retries, state changes and shutdown.
10. Report exactly what passed, what remains untested and what is explicitly deferred.
