# Five Conversation Tones — validation record

**Scope:** five independent approved Hamrah Approach choices, server-side prompting on the next generated turn, legacy guidance compatibility. No UI screen is removed and no user-memory system is added.

## Executed here

- `python3 scripts/check_structure.py`: PASS, 41 required paths, 6 specialist skills.
- `python3 scripts/check_contract.py`: PASS, 20 registered route/method pairs.
- `gofmt` on changed Go sources: PASS.
- SQL migration `0011_five_conversation_styles.sql`: static source inspection only; forward migration drops/replaces the old check, preserves rows, retains legacy `guidance`.
- Go unit/integration tests: BLOCKED — `github.com/gin-gonic/gin` and `github.com/google/uuid` are absent from the local Go module cache and GOPROXY was offline.
- Real MSSQL integration: NOT RUN in this container. `tests/integration/conversation_styles_test.go` tests a disposable local database and must be executed in the dev stack.
- External-provider behavioral evaluation: NOT RUN. The mock gateway intentionally returns fixed answers, so only prompts and API behavior can be tested deterministically.

## Local acceptance

1. Apply the Backend patch then `make migrate` on the local development stack.
2. Run `python3 scripts/check_structure.py`, `python3 scripts/check_contract.py`, and `cd apps/api && go test ./... -count=1`.
3. Against a disposable local SQL Server, run `WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test ./tests/integration -run TestFiveConversationStylesSQLServer -count=1` from `apps/api`, or use your normal local environment loader.
4. Verify profile `guidance` pre-0011 remains readable; all five new values persist and invalid values fail DB CHECK.
5. Ensure tests use only synthetic content and a local database. Confirm older conversations' stored promptVersion values are unchanged; only newly generated turns use `supportive-v4`.

These tests do not validate mental-health safety or real-user readiness.
