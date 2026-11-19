# Local AI Core Inspector v1

The inspector captures the implemented Go pipeline for local debugging. It lives
in the existing `internal/modules/ai/trace` package. Production `ports.Tracer`,
logging, usage accounting, durable trace summaries, model calls, consent gates,
transaction boundaries and response contracts retain their existing behavior.
No frontend, external tracing service, migration, dependency or extra model call
is required.

## Enable and try it

From the backend directory, keep the existing working database, model and bundle
configuration described in [runtime setup](../operations/ai-core-wp16.md). Set:

```dotenv
APP_ENV=local
HTTP_ADDR=127.0.0.1:8080
AI_LOCAL_INSPECTOR=true
AI_LOCAL_INSPECTOR_DIR=.local/ai-inspector
```

Start the native backend and inspector together:

```sh
make api
```

The launcher automatically adds missing `.env.example` keys to `.env` without
replacing existing settings. `make api` enables inspection for its process even
when `.env` contains `AI_LOCAL_INSPECTOR=false`; use
`AI_LOCAL_INSPECTOR=false make api` to disable it. There is no separate inspector
process. Direct `python3 scripts/run_local.py go run ./cmd/api` also syncs and
loads `.env`, but respects its inspector flag and any exported override.

The inspector does not activate an unconfigured AI runtime. `AI_CORE_CONFIG_FILE`
and its existing reviewed bundle/local synthetic draft, provider and encryption
settings must already work. The listener guard deliberately rejects the
`0.0.0.0` listener commonly used inside containers, and all staging, production
and unknown environments. The optional output directory is relative to the
**process working directory**. The loader above starts in `apps/api`, so its
default output is `apps/api/.local/ai-inspector/`. An absolute directory removes
that ambiguity. Existing output directories must be private (`0700`). Files are
`0600`; the default `.local/` paths are ignored by Git.

Use the existing local frontend: select a synthetic identity, finish onboarding,
open a conversation and send a synthetic message. Alternatively, with an
existing authenticated cookie jar and owned conversation UUID:

```sh
curl -b /tmp/theia-local-cookies.txt \
  -H 'Content-Type: application/json' \
  -H 'Origin: http://localhost:5173' \
  -d '{"clientTurnId":"d7364dab-d6b8-433b-b523-daa3bda964c8","message":"I want to discuss a fictional work situation.","locale":"en","timeZone":"Asia/Tehran"}' \
  http://127.0.0.1:8080/api/v1/conversations/OWNED_CONVERSATION_UUID/turns
```

Use a fresh `clientTurnId` for another message. Reusing the same request produces
an admission/replay trace and does not fabricate another inference execution.
Find `<trace_id>.json` in the configured directory. Claimed memory/reflection job
attempts produce separate files with the same `turn_id` and a persistent
`job_id`. Refresh a file while a turn runs to inspect its latest checkpoint.
Use `AI_LOCAL_INSPECTOR=false make api` to restart without collection. For direct
launcher commands, set `AI_LOCAL_INSPECTOR=false` in `.env` or export it.

Only this explicitly enabled local feature stores full messages, preferences,
prompts, memory, model content and selected knowledge. Header maps, cookies,
provider credentials, session secrets and configuration objects are excluded.
Configured provider/session secrets are registered before turns start; secret
values echoed in provider bodies are omitted. Credential-named fields and lease
fences are omitted. Content is otherwise retained. Ordinary logs and production
metadata traces never receive these payloads.

## Execution coverage verified against source

| Actual boundary | Captured details |
| --- | --- |
| Orchestrator `ProcessTurn` and `Gateway.Admit` | Original core request before NFC normalization, authenticated IDs, language/timezone/channel, admission outcome, rejection, queue/replay, server turn association, result/error |
| Orchestrator preparation and Router | Nested execution, parallel safety/retrieval, route input/config and decision, working-context changes, risk before/signal/after, explicit fast/crisis Brain skips |
| Safety Gate, L1 classifier and crisis Planner | Safety inputs/history, policy, normalized L0 match text/rule/offsets/message index, classifier request/output, signal and degraded behavior, curated crisis plan |
| Retrieval and repository methods | Original and second-hop requests, chosen strategies, actual repository queries/candidates, filtering, scoring, selected views, exclusion/recent pools, embedding request/vector/model, degraded/pause decisions |
| Context Assembler and state readers | Owner-scoped source snapshots, profile/preferences/history and retrieved views, assembled context, state versions and trimming visible in input/output differences |
| Knowledge Bundle `Artifact` | Actual runtime artifact reads, exact content, path and pinned version; cached Brain/Generator/Memory/Reflection templates and rendered blocks are separately captured where used |
| Brain and policy gate | Complete input, system blocks/policies, rendered dynamic prompt and context after trimming, strict model request/result, proposed and gated decisions, repairs/fallback rules, retrieval hops |
| Protocol shortlist and resolver | Candidate definitions/eligibility, evaluated scores, shortlist, active instance and consent inputs, resolved step and proposed transition |
| Generator | Input plan/spec/permitted context/preferences, original templates, rendered system/messages, partial drafts including rejected content, output/abort/sentinel, attempts |
| Stream guard and Validator | Actual token checks and violations, buffer resets, full final-validation inputs/reports, rejected drafts, regeneration notes and fallback selections/validation |
| State validator and finalizer | Proposed delta and source snapshot, rejected changes, validated prepared turn, pending transaction records including actual job/outbox IDs and JSON payloads, finalizer outcome and final persisted response IDs |
| Memory Writer/Committer | Extraction inputs/evidence, raw/rendered prompts, proposals, reconciliation inputs and before/after mutations, repository snapshots, persistence plan/result |
| Memory and Reflection workers | Persistent turn/job IDs, actual attempt, claimed lease, decrypted task/checkpoint, resumed or newly executed model work, validation/commit result, retry cleanup |
| WP-04 shared provider interface | Every actual structured/stream/embedding attempt, tier/model/config, retries/repairs/failover, provider usage and cost estimates |
| `llmhttp` final transport | Exact serialized request body **after** JSON-mode schema injection and model argument adjustments, fixed resource path, status and raw response body/SSE bytes actually read, including rejected HTTP responses |

An absent operation is not an executed operation. Explicit `skipped` nodes are
written only at branches that actually decide to bypass a stage (for example,
missing retrieval, fast/crisis Brain bypass, or an open circuit breaker).
Unreached downstream operations after a failure are absent. Successful operations
can still return degraded results; inspect typed `Degraded`/`degraded`, repair
reports and decision artifacts. Nothing fills in a predeclared architecture DAG.

## Trace contract and viewer integration

For the additive graph bindings, nested prompt composition indexes, compatibility
rules and real full-turn fixtures, see the [execution graph contract](inspector-graph.md).

The authoritative machine-readable JSON Schema is
[`local_ai_trace.v1.schema.json`](../../apps/api/internal/modules/ai/trace/local_ai_trace.v1.schema.json).
Go contracts and the writer are in
[`trace/inspector.go`](../../apps/api/internal/modules/ai/trace/inspector.go).
`trace.ValidateJSON` validates both JSON Schema and graph/timing invariants.
Schema compilation occurs only when explicitly validating, outside inference.

Top-level `schema` is `local_ai_trace.v1`; incompatible changes require a new
version. `trace_id` is a unique file/invocation ID. `turn_id` is assigned after
admission; before admission or on rejection it can be the zero UUID. A job has
`kind`, `job_id`, and `attempt`. All timestamps are UTC RFC3339 nanosecond values;
durations are integer nanoseconds measured with Go's monotonic clock.

`nodes` is in start/sequence order, not completion order. `id` is stable within
the trace (`n000001`, ...); join it with `trace_id` when combining files.
`operation` identifies the real source boundary. `parent_id` and `execution`
edges reconstruct nesting. Parallel siblings can overlap. Every operation has
snapshotted `input`, `output`, timestamps, duration, status, artifacts, provider
usage and estimates. Running nodes have null output and no end timestamp.
Errors retain safe types/codes/steps rather than arbitrary SQL/network error
strings; provider rejection bodies are available separately.

`artifacts` distinguish `prompt_template`, `rendered_prompt`, `knowledge`,
`knowledge_artifact`, raw `wire_response`, intermediate drafts, proposals and
control-flow decisions. Each has a name, source and SHA-256 of its compact JSON value (preserving exported key order), including any credential omission. Go contract
`[]byte` fields retain their existing JSON base64 encoding; dedicated raw model,
wire and rendered-prompt artifacts provide readable text. Provider HTTP input
`body` is the exact serialized JSON string. Parse that string to inspect the
actual final message array, schema and model arguments.

Data edges contain JSON Pointers (`from_path`, `to_path`). Automatic evidence is
`exact_json_value_match`: a previously observed output/artifact value exactly
matches part of a later input. They are deliberately **value provenance evidence,
not proof of causal dependence**. Explicit call-site bindings additionally use
`instrumented_binding` with a `copy`, `transform`, or `regeneration` relation.
Short scalar matches are excluded. Changed
values remain visible in the containing inputs/outputs and explicit state-change
artifacts; do not infer causal edges merely from timing. IDs embedded in memory
views, evidence, knowledge sources and state records retain their domain meaning.

A viewer can place nodes on a start-time timeline, group by `parent_id`, draw
`execution` and `data` edges separately, and open input/output/artifact JSON in
an inspector panel. To join async traces, group by `turn_id` and then persistent
`job_id`/attempt. Do not merge a worker's tokens into synchronous latency or count
its model work twice. No repository frontend currently supplies this viewer.

## Token and cost interpretation

`provider_usage` on an actual provider node is the single accounting source for
trace totals. It contains provider-reported input/output/cached counters. It is
recorded before later schema/gating/validation failures, so rejected responses,
repairs, retries and retrieval embeddings remain visible. Valid reported usage
on an invalid structured completion is captured even when the normal adapter
return deliberately omits it. The existing application quota/ledger is unchanged.
Malformed duplicate or invalid numeric usage is retained in raw artifacts but
never summed as trustworthy usage.

`totals` includes actual counters, provider calls, reported calls, calls with
unknown usage, and attribution to the closest application stage. Missing usage
is **unknown**, not zero consumption. In a running checkpoint it may not have
arrived yet. Parent Brain/Generator usage aggregates are retained in their typed
outputs but are never added again to totals.

`estimates` use the explicitly labelled heuristic
`ceil_unicode_codepoints_div_3`, excluding provider framing. Named system blocks,
user/assistant messages and embeddings expose per-component estimates. Where
rendered XML has top-level sections, `/section/<name>` gives an alternative
breakdown for state, memory, recent history, skills and current message.
Section estimates overlap the enclosing block and must **not** be added to it.
`estimated_input_tokens` totals sum only the enclosing provider request
components, once per actual attempt. Estimates are not tokenizer measurements
and need not sum to provider-reported counts. Protocol/knowledge templates that
were never sent to a model do not become billed tokens.

`estimated_cost_micro_usd` uses configured model input/output/cached prices and
existing WP-04 rounding. A per-call cost is absent when all prices are zero;
the total sums configured estimates only. It is not a provider invoice. No
prices are fetched or guessed. Compare repeated system blocks, rendered context
sections, second hops, repairs and regeneration attempts to locate optimization
opportunities without adding model calls.

## Reliability and implementation limits

- Context-scoped collectors isolate concurrent turns. Workers receive the
  inspector through their supervised context; there is no global turn registry,
  background tracing service or dependency on a finished request remaining alive.
- Each invocation/attempt gets a unique file. Every span start/end writes a
  same-directory `0600` temporary file, syncs, atomically renames it, then syncs
  the directory. Readers see complete JSON snapshots. A crash leaves the last
  checkpoint; no shutdown callback is needed to make it useful. `complete=false`
  means the root or a child is still open. Late children can checkpoint after
  the root has returned. Abrupt process death cannot capture future events or
  bytes not yet checkpointed.
- File failures never change turn results or SQL commits. The previous valid
  checkpoint remains, a fixed safe warning is logged, and later checkpoints
  retry. Once storage recovers, `diagnostics` records the earlier failure. Disk
  failure can prevent an export entirely; file export is not a DB transaction.
- Local serialization, hashing and synchronous filesystem sync add overhead and
  can affect measured latency/deadline headroom. Disable inspection for baseline
  performance measurements. Disabled spans do not serialize, read assets, write
  files or read clocks; the no-op allocation test verifies lazy capture.
- Wire capture covers the implemented `llmhttp` adapter. A different provider
  implementation gets neutral request/result/attempt capture through WP-04;
  exact vendor wire payloads require instrumentation in that new transport.
- SSE is **internal provider streaming**. The existing product HTTP endpoint
  still returns the validated final JSON response. Raw SSE capture contains bytes
  actually read and says whether the body reached EOF; early-close bodies are
  explicitly incomplete. The inspector never drains a successful stream merely
  to obtain more diagnostic content.
- Transaction records are labelled `transaction_pending` until their enclosing
  finalizer succeeds. State deltas and memory mutations are proposals/plans;
  they do not claim a post-commit reread of every SQL row. No additional queries
  run inside transactions for inspection.
- In-memory templates are captured where consumed; runtime bundle reads are
  captured where they occur. The inspector does not pretend every startup asset
  was selected on each turn, and does not reread all knowledge just to export it.
- Idle/failed job claims have no associated turn and do not create turn files.
  Independent scheduler sweeps and historical work predating enablement are
  outside per-turn files. Claimed memory/reflection work is linked across process
  restarts using durable IDs. Startup/configuration failures remain in existing
  safe operational diagnostics.
- The static boundary coverage test requires new exported runtime entrypoints
  in the primary reasoning/generation/retrieval/protocol/safety modules to be
  instrumented or explicitly exempted. It cannot prove every future internal
  algorithmic branch is sufficiently observable; changes still need source and
  integration review.

## Examples and verification

[`inspector-examples/`](inspector-examples/) contains generated synthetic traces:
`normal.json`, `generation_fallback.json`, `exact_wire.json`, and
`provider_retry_repair_stream.json`. The conversation examples use the existing
integrated orchestrator fixture (real Safety/Assembler/State validation and
scripted Brain/Generator ports). Their zero provider-call totals accurately
reflect that substitution. The model example executes the real WP-04 client
against a scripted provider: retries, failover, one schema repair, embedding and
streaming. The wire example uses the actual adapter and an in-memory HTTP
transport, including JSON-object schema injection and reasoning-model settings.
No example was obtained by sending data to an external model.

Reproduce examples explicitly from the backend directory:

```sh
THEIA_UPDATE_INSPECTOR_EXAMPLES=true go -C apps/api test \
  ./internal/modules/ai/application/orchestrator \
  ./internal/modules/ai/llm ./internal/modules/ai/adapters/llmhttp \
  -run Inspector -count=1
```

Tests cover schema/graph invariants, exact HTTP bytes, unredacted local data and
secret exclusion, lazy disabled behavior, concurrent children/turns, crash-style
partial checkpoints, write failure/recovery, nil-context compatibility, panic
unwinding, admission/replay, safety/crisis/degradation, fallback/rejected output,
state commit failure, memory evidence/mutations, retrieval query/scoring/selection,
protocol transitions, Brain gates, Generator drafts/rendering, worker checkpoint
retry, schema repair, embedding and streaming usage without double counting.
Existing AI Core regression/evaluation tests remain applicable.

Run `go -C apps/api test ./...`, race tests for AI/conversation/composition/config,
and the existing structure/OpenAPI checks. Live SQL Server checks remain opt-in
using `WELLBEING_INTEGRATION=true` and a local `DATABASE_URL` (the API also supports split database configuration); the default suite does not exercise a live server. Inspection files are local development artifacts and no file-serving
endpoint has been introduced.

## Implementation verification report — 2026-11-18

Verified on Linux amd64 with Go 1.25.4; the workspace used
`GOCACHE=/tmp/theia-go-cache` because the default cache directory is read-only.

- Full backend `go test ./...`: passed.
- Race checks across AI Core, Conversation, composition and config: passed.
- Full opt-in live SQL Server 2022 integration suite: passed (63.8 seconds), using
  the existing disposable-database harness and the saved loopback connection.
  A process-local runner mapped the existing split database settings into the
  legacy test harness's `DATABASE_URL` without displaying or saving credentials.
- New `TestInspectorSQLFinalizationAndRollback`: passed again on the final
  implementation. Exported reply IDs/text and exact outbox IDs/payloads match SQL
  after commit; failed finalization leaves one user message, no outbox/trace
  metadata rows, and an error trace retaining the pending transaction artifacts.
- Structure/SQL dialect, OpenAPI/composition checks and `git diff --check`: passed.
- Generated examples conform to the versioned schema and graph checks. Disabled
  spans and dynamic artifact capture both pass zero-allocation assertions.

No external-provider smoke test was run; provider tests use scripted providers
and local/in-memory HTTP transports. No psychological knowledge, applied SQL
migration, production trace policy or product HTTP contract was changed for
this feature. Pre-existing workspace edits were retained.
