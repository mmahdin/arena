# Logging and error observability implementation report

Owner: backend platform and AI Core. Scope: diagnostics only; no public API,
authorization, consent or clinical policy change, migration or new dependency.

## Implemented changes

- Extended the existing `slog`/observability infrastructure. Fixed vocabularies
  identify source checks, executing stages, failure kinds, safe remediation
  hints and optional SQL/provider metadata. Wrapping and joining errors retains
  causes; the deepest available source annotation survives translation.
- Added request context to the operational turn tracer without changing the
  existing Tracer port. Stage start/skip/success/degradation/terminal events are
  distinct. Safety and Retrieval durations measure their own parallel work.
  Starts do not consume the bounded Recorder summary budget. Terminal summary
  failures are emitted once, and stage events provide their own boundary detail.
- Corrected Generator failures previously attributed to Validator. Source
  verification distinguishes consent state, visibility misses, role/content/
  chronology/provenance/metadata mismatches and database failures. Brain,
  Generator and Safety retain their original dependency causes. Policy-facts
  query errors are summarized once per Brain invocation.
- Added source/operation annotations to turn persistence, state/context,
  retrieval, memory access and connection startup diagnostics. SQL Server
  numeric error number/state/class survive nested wrappers; no statement,
  parameter, database credential or raw driver message is emitted.
- Provider attempt records preserve HTTP status, recognized machine code,
  causal types and timeout category. Unknown exceptions are unclassified rather
  than presumed permanent. Retries, actual model fallback, circuit skips,
  chain exhaustion, schema repair decisions and preflight/call failures have
  separate diagnostic purposes. Existing retry/breaker/fallback decisions remain.
- HTTP logs retain source diagnostics, returned status/code, request ID and
  pseudonymous turn reference. A concealed authorization failure is internally
  identified separately from an owner-scoped visibility miss. An internal turn
  boundary wrapper preserves historical public-code precedence even when its
  original cause is a timeout, not-found or policy sentinel.
- Worker denials have source annotations and safe check labels. Job events carry
  the turn reference and attempt; successful jobs include duration. Poll versus
  acquired-job failure summaries are distinguished.

Ordinary logs never include error messages, prompts, conversation content,
credentials or arbitrary provider bodies. Provider rejection parsing extracts
only a closed list of recognized machine codes; unknown alphanumeric codes are
also dropped. The local Execution Inspector remains separate.

## Reported incident and functional findings

The incorrect Validator label is proven by the old unconditional `prepare`
return after `generateValidated`. Lost verifier and classifier causes are also
confirmed in the source. The old LLM classifier's default was `permanent` for
all unrecognized errors, including untyped deadline/cancellation errors.

The first-turn/second-turn shape has a reproducible source-use consent path:
Generator checks every recent message, while admission requires no separate
chat consent. No history is checked on an empty first turn. A missing configured
message-use consent can therefore deny a later turn. A denied Generator call
still intentionally maps to a concealed 404. This is a policy interaction to
review separately; it was not changed. The supplied incident logs cannot prove
whether that path, a provenance check, a visibility miss or a query failure was
the actual incident cause. No production data/provider/database was inspected.

Regression coverage tests first-turn success, subsequent denial, correct
Generator attribution, retained causes, no unauthorized provider call and the
sanitized HTTP mapping. A separate opt-in SQL/HTTP regression uses the real
ProductUsePolicy, verifiers, persistence and HTTP application.

## Verification

Execution environment: Go 1.25.4, linux/amd64; module declares Go 1.23.0.
The sandbox's default Go cache was read-only, so checks used
`GOCACHE=/tmp/theia-go-cache`. Local TLS test servers required execution with
local socket permission. No external LLM calls were made.

| Check | Result |
| --- | --- |
| `go test ./...` | PASS, 43 packages with tests; SQL-dependent tests skip without their opt-in |
| `go test -race ./...` | PASS on final code, 43 packages with tests |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `python3 scripts/check_structure.py` | PASS, 41 required paths |
| `python3 scripts/check_contract.py` | PASS, 38 method/route pairs |
| `python3 -m unittest discover -s scripts/tests -v` | PASS, 13 tests, including DEBUG environment precedence |
| `git diff --check` | PASS |
| `TestObservabilitySecondTurnWithoutChatConsentSQLHTTP` | SKIP: `WELLBEING_INTEGRATION` is not enabled |

New/focused tests cover source precedence through joins; SQL errors versus
visibility and provenance denials; missing/false/invalid consent snapshots;
provider status/code allowlisting; DNS/refused/cancelled/timed-out transport;
retry, model fallback, exhaustion and first-token timeout visibility; request
and turn correlation; all four log levels; privacy; terminal-event deduplication;
and preserved public mapping despite newly retained internal causes.

To run the live SQL regression against the repository's isolated local test
fixture (creates and drops a disposable test database):

```bash
WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test ./tests/integration \
  -run '^TestObservabilitySecondTurnWithoutChatConsentSQLHTTP$' -v -count=1
```

The repository's existing local `APP_ENV`/`DATABASE_URL` test guards apply.
Live SQL execution and real provider latency/response variations remain unverified.

## Usage, examples and limits

See [LOGGING.md](LOGGING.md) for DEBUG startup, request/turn lookup commands,
field semantics and sanitized successful/degraded/failed log sequences.

Some legacy memory/reflection checks still share a check code for a compound
condition; source locations identify the exact branch. Boolean-only custom
policy ports cannot supply a finer reason than their own interface exposes.
An owner-scoped miss cannot distinguish deletion from foreign ownership, and a
false consent snapshot cannot establish revocation history. Logs deliberately
do not issue additional unscoped queries or invent those distinctions.
Correlation references are process-local and cannot join turns across restarts
or replicas. Unknown vendor codes and private SQL object/parameter details are
not logged. These are explicit limits, not inferred incident causes.

## Patch baseline

`theia-observability.patch` contains only changes made by this task, relative to
the supplied starting worktree. The repository already had modified and
untracked policy/observability/product/API files when work began. Those existing
edits are preserved and excluded from this incremental patch. Apply it to a
copy of that starting worktree; the current workspace already contains it.

Patch verification uses `git apply --check` against the saved starting snapshot,
then applies it in a disposable copy and compares every affected file byte for
byte. No index changes or commits are performed in the user's repository.

## Subsequent chat-context policy fix

After the logging-only overhaul, a separate change removed the independent
message-use consent requirement from recent messages in the currently active
conversation. These messages remain subject to owner, same-conversation,
soft-deletion, chronology, role, and content verification. Independently
retrieved historical messages and memory remain consent-gated. The opt-in SQL
regression above now expects the second turn to succeed without separate chat
consent. The original incident explanation above describes the earlier
logging-only baseline and is retained for historical context.
