# Unified conversation refactor verification — 2026-11-24

Scope: backend only, based on commit `27216e6`. Architecture is described in
ADR 0015 and `docs/ai-core/unified-memory-centric.md`. No commit, push, deployment,
shared database migration or real-provider conversation is part of this delivery.
All integration content is synthetic.

## Checks

| Check | Command / scope | Result |
| --- | --- | --- |
| Structure and T-SQL guard | `python3 scripts/check_structure.py` | Passed: 45 required paths and five repository skills |
| HTTP contract baseline | `python3 scripts/check_contract.py` | Passed: 63 method/route pairs; HTTP API unchanged |
| Script tests | `python3 -m unittest discover -s scripts/tests -p 'test_*.py'` | Passed: 128 tests |
| Go suite, from `apps/api` | `go test ./...` | Passed: 54 packages; SQL-gated tests also exercised separately below |
| Go static analysis | `go vet ./...` | Passed |
| Race detector | `go test -race ./internal/modules/ai/application/... ./internal/modules/ai/trace/...` | Passed: 13 packages; changed orchestration/retrieval/memory packages rechecked after final fixes |
| SQL Server 2022 | Selection below with `WELLBEING_INTEGRATION=true`, `APP_ENV=local`, local master DSN | Passed: 19 top-level tests, 43 including subtests |
| Offline Inspector | Browser command below | Passed: new JEV/Unified fixture plus five historical traces; no HTTP requests or browser errors |
| Safety freeze | Git diff on Safety/Validator packages, policy and crisis assets; compare the original Unified Safety paragraph to HEAD | Unchanged |
| Patch integrity | Temporary index; apply to an isolated HEAD export and compare its Git tree with the patch index | Passed; real Git index and HEAD preserved |

SQL selection, from `apps/api`:

```sh
go test ./tests/integration \
  -run 'TestUnified|TestAICoreSQLServer|TestPROD0[12]|TestInternalAICoreSQLCompositionMultiturnEvalAndWorkers|TestInspectorSQL|TestWP12|TestWP14|TestHybridIndexSQL' \
  -count=1 -v -timeout 8m
```

The existing local harness creates and drops independent `wellbeing_ai_*`
databases. The existing SQL Server container was used; credentials came from its
environment and were never printed or included in a fixture. Application databases
were not migrated. Checks include upgrade 0040→0041 with existing completed turns,
owner isolation, immutable decisions, finalization replay and late rollback,
queued explicit analysis, actual hybrid research reaching Memory Writer, routine
review without a Unified request, source revision fencing, Reflection checkpoint
recovery, lease fencing and retained evidence policies.

Browser check, from the backend root:

```sh
node scripts/tests/ai_explorer_unified_browser.mjs
```

The script uses installed Playwright from the sibling frontend (`PLAYWRIGHT_ROOT`
can override that location). This run selected an existing Chromium through
`PLAYWRIGHT_CHROMIUM_EXECUTABLE`. It opened the actual standalone Explorer via a
local file URL, imported `jev_unified_turn.json`, expanded/collapsed the tree,
inspected JEV and Unified input/output, followed data bindings and captured prompt
sources, opened provider metadata, panned, zoomed, fitted and opened the summary.
It also imported normal, protocol, regeneration, fallback and old Unified examples.
The generated new fixture contains only synthetic scripted provider responses;
its timings/token totals are not live-provider performance measurements.

## Important limits and rollout

- Apply additive migration `0041_unified_turn_decisions.sql` before starting the
  new runtime. Configuration must be `ai_runtime.v4` with JEV, Unified,
  Memory Writer, Reflection and embedding components. Old runtime files are
  explicitly rejected. Keep existing jobs pinned to their original bundle hashes.
- Core Policies and the original Unified Safety paragraph were preserved. Passing
  engineering regression tests is not a new Safety assessment. Live-model Safety,
  conversational quality, latency, model availability and cost were not evaluated.
- Independent proposal rejection is guaranteed for structurally/operationally
  invalid optional items. Natural-language consistency still requires Unified to
  declare dependencies; actual exercise status and steps are checked separately.
- Background intelligence reuses existing bounded jobs, policies, retries,
  checkpoints and models. No autonomous agent or unrestricted history sweep was
  introduced. Periodic review covers at most eight recent completed turns per
  sweep and only the previous 24 hours.
- Historical serialized contracts, frozen Safety/Validator regression assets and
  old Inspector files remain readable. Brain/Generator modules, executable
  wrappers, old constructors and unused routing/stage ports have been removed.
