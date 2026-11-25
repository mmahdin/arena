# Theia backend diagnostics (development and operations)

Operational logs are **JSON Lines** on stderr, with an optional append-only
`LOG_FILE` (0600). These logs are *metadata-only*: do not emit text, prompts,
completion payloads, API keys, authorization headers, raw SQL errors, user IDs,
raw turn IDs, request bodies, or vendor response bodies, including at DEBUG.

## Run locally

`make api` executes `scripts/run_local.py go run ./cmd/api`. The script loads
literal values from the optional project-root `.env` and **never evaluates shell
code**. A variable already exported in the shell **wins** over `.env`.

```bash
# One invocation at DEBUG, regardless of the setting in .env:
LOG_LEVEL=debug make api

# Or put LOG_LEVEL=debug in .env, and clear a conflicting exported value:
env -u LOG_LEVEL make api

# Optional append-only private log file; also keeps stderr output:
LOG_LEVEL=debug LOG_FILE=./theia-api.jsonl make api

# Example: group a turn's events with jq, after running your test turn:
jq -c 'select(.turn_ref == "REPLACE_WITH_A_LOGGED_TURN_REF")' theia-api.jsonl

# A failing HTTP request includes request_id. Find its turn_ref first:
jq -c 'select(.request_id == "REPLACE_WITH_REQUEST_ID")' theia-api.jsonl
```

`log_level` is recorded in the `api.starting` event, which is a quick way to
confirm what the Go process received. DEBUG enables successful stage and
attempt diagnostics; INFO/WARN/ERROR follow normal `slog` severity filtering.
The logger does not require OpenTelemetry, an external agent, or a log service.
In production, use the container log driver for retention/rotation. Rotate any
`LOG_FILE` you configure; the application does not implement file rotation.

## Correlation and location

| Field | Meaning |
| --- | --- |
| `request_id` | Internally generated per-HTTP-request UUID. Propagates in Go context into services and provider calls; returned as `X-Request-ID`. |
| `turn_ref` | Pseudonymous per-process, per-turn hash. Present in HTTP outcome and AI Core/LLM events; **not** the database turn UUID. |
| `component`, `stage`, `operation` | Structured low-cardinality owner, pipeline step, and operation, where applicable. |
| `attempt`, `retry_index`, `max_attempts` | Attempt number, started retry index, configured maximum; no inference from timestamp gaps is necessary. |
| `status`, `reason`, `error_code`, `error_type`, `root_cause_type` | Safe status and/or diagnostic classification. `unknown` means the source has supplied no safely classifiable metadata. |
| `error_origin` | Deepest captured internal construction site, including through wrapped/joined causes. Translation and observation sites do not replace it. |
| `error_site` | File and line at which an upstream error was **observed**. It is not represented as the original construction site. |
| `source` | Go `slog` record emission site. For tracer events this is the logging adapter (`ai_logging.go`); consult `error_origin`/`error_site` for failure location. |
| `duration_ms`, `fallback_used`, `retryable`, `provider_status` | Safe timing, recovery flags, and numeric provider HTTP code, when applicable. For known SQL codes, `retryable` describes transaction-level retry suitability; it is not an instruction to retry a user turn. |
| `db_error_number`, `db_error_state`, `db_error_class` | SQL Server's numeric error number, state, and severity, extracted from the driver's `SQLError*` methods through `errors.As`; no SQL text or error message. Works through nested error wrappers anywhere that uses the MSSQL driver. |
| `diagnostic_hint` | Fixed, safe operational next check based on a known SQL error number (not a guess about the exact schema object). |
| `operation` | Explicit internal operation where available; the shared memory/reflection SQL claim path is annotated at each failing SQL statement. |
| `failed_check`, `failure_stage` | Source check and executing component, retained through HTTP translation; no private resource handles. |
| `http_error_kind` | `authorization_concealed` versus an owner-scoped `resource_not_visible` result. A visibility miss does not prove physical absence. |
| `provider_error_code`, `timeout_category` | Recognized machine code only; timeout category is `request`, `connection`, or `first_token` when the source has evidence. |
| `phase` | Worker's `poll` vs. job-level execution, to distinguish failures before a job is acquired. |

`turn_ref` intentionally uses a **random process-local salt**: it is consistent
within a process but cannot be joined across container restarts or replicas.
To join a single request, find its HTTP event and the matching `turn_ref` in the
same process. Cross-instance persistent correlation would require an explicitly
reviewed privacy/security design rather than logging raw identifiers.

## Event conventions

| Event | Typical level | When it is emitted |
| --- | --- | --- |
| `api.starting`, `api.listening`, `api.stopped` | INFO | Application lifecycle. |
| `http.request` | INFO/WARN/ERROR | One bounded HTTP outcome; health successes are DEBUG. |
| `http.panic`, `ai.worker.panic` | ERROR | Panic type + source-only stack; **never** the panic value or argument values. |
| `ai.turn.started`, `ai.turn.finished` | INFO | Start and end of an admitted AI turn, including `failed`/`degraded` flags. |
| `ai.stage.started`, `ai.stage.completed`, `ai.stage.skipped` | DEBUG | Stage transitions, successful completion duration and explicit bypasses. |
| `ai.stage.degraded`, `ai.stage.failed` | WARN/ERROR | Recoverable degradation or a terminal stage failure; each retains source diagnostics. |
| `ai.turn.failed` | ERROR | Original internal error (safe fields only), before conversion into public HTTP error. |
| `ai.brain.fallback` | WARN | Brain substituted an approved fallback (may include provider error category). |
| `llm.http.rejected` | WARN | Provider returned non-200; numeric status and a closed-vocabulary machine error code when recognized. |
| `llm.attempt.failed`, `llm.attempt.completed` | WARN/DEBUG | Each completed provider attempt with tier-independent metadata. |
| `llm.retry.started`, `llm.chain.decision` | DEBUG | Started retry, model exhaustion, skipped circuit and actual provider fallback selection. |
| `ai.generation.attempt_failed` | WARN | Generator attempt failed; includes attempt and whether regeneration is possible. |
| `ai.validation.rejected` | WARN | Validator failure, allowlisted `OV-xxx` rules, action and attempt. |
| `ai.generation.regenerated` | INFO | A later generation attempt passed validation. |
| `ai.generation.fallback_used`, `ai.generation.fallback_unavailable` | WARN/ERROR | Validated fallback used, or none was eligible/available. |
| `ai.worker.job_failed`, `ai.worker.retry`, `ai.worker.recovered` | WARN/WARN/INFO | Job-level failure, rate-limited worker error summary, and recovery. |

Never log `err.Error()` for external errors. `ErrorFields` inspects error types,
wrapped causes, MSSQL numeric error metadata and restricted enums. The original
error is retained **in memory** while it is propagated; only selected fields
reach JSON logs. Unknown SQL numbers are still logged numerically as
`reason=db_error`, without an invented remediation hint.

User-initiated AI chat, retry and recovery do not check a separate chat-consent
flag and cannot emit a chat-consent-required response.

**Common SQL diagnostics:** `207` -> `db_invalid_column`, `208` ->
`db_missing_object` (check schema/migrations); `229` ->
`db_permission_denied` (check grants); `547` ->
`db_constraint_violation`; `2601`/`2627` -> `db_duplicate_key`;
`1205` -> `db_transaction_conflict` (safe transactional retry after backoff).
These are error-code interpretations, **not** proof of a particular missing
migration or column. Inspect the code at `error_origin` for the exact query.

Memory and Reflection share `ai_jobs.claim_ready` and other `ai_jobs.*`
operations. An `ai.worker.retry` event with `phase=poll` means no job was
acquired and no job-level `ai.worker.job_failed` event is expected. The worker
still polls according to its existing schedule; logging does not change job
processing or background retry behavior.

## Synthetic examples (not observed production incidents)

```json
{"level":"DEBUG","msg":"ai.stage.completed","stage":"assembler","status":"success","duration_ms":9,"turn_ref":"0a1b2c3d4e5f6789"}
{"level":"WARN","msg":"llm.http.rejected","component":"llm_http","operation":"provider_request","turn_ref":"0a1b2c3d4e5f6789","reason":"rate_limit","provider_status":429,"retryable":true,"error_origin":"apps/api/internal/modules/ai/adapters/llmhttp/provider.go:131"}
{"level":"WARN","msg":"ai.validation.rejected","stage":"validator","attempt":1,"max_attempts":2,"rules":["OV-012","OV-013"],"action":"regenerate","regeneration_possible":true,"turn_ref":"0a1b2c3d4e5f6789"}
{"level":"ERROR","msg":"ai.turn.failed","stage":"generator","error_code":"forbidden","turn_ref":"0a1b2c3d4e5f6789","reason":"consent_missing","failed_check":"retrieved_source_use","diagnostic_hint":"check_source_use_consent_and_mapping"}
{"level":"WARN","msg":"ai.worker.retry","worker":"memory","phase":"poll","reason":"db_invalid_column","db_error_number":207,"db_error_state":1,"db_error_class":16,"operation":"ai_jobs.claim_ready","diagnostic_hint":"verify_schema_migrations","retryable":false,"error_origin":"apps/api/internal/modules/ai/adapters/mssql/memory_jobs.go:56"}
```

## Diagnosing the reported second-message 404

The old code had three independently confirmed information-loss points:

1. `prepare` returned `stage=validator` for **every** error from
   `generateValidated`, including fatal Generator source-verification failures.
2. Generator and Brain replaced verifier errors with new `forbidden` errors
   without preserving the cause. Their MSSQL verifiers also merged missing
   rows, query failures, consent denials and provenance mismatches into the
   same denial. Safety similarly discarded its classifier's provider error.
3. The LLM client classified every unrecognized error as `permanent`, and
   attempt events retained only that kind. Terminal provider-chain errors
   discarded the last failure's status/code. These paths now retain safe
   source evidence and distinguish an unclassified error from a rejection.

The earlier first-turn/second-turn incident involved the Generator checking
secondary message-use consent for *recent messages in the active conversation*.
That check has now been removed from the active-conversation path. Every recent
message is still verified against its owner, active conversation, soft-deletion,
chronology, role and content before disclosure. Separately retrieved messages
and memory continue to use the independently configured source-use consent
policy; historical cross-conversation retrieval is not made implicit.

An internal `forbidden` still intentionally maps to a public `404
RESOURCE_NOT_FOUND` where required to conceal unauthorized resources. A 404
does not prove that the conversation disappeared. A persisted false consent is
logged as `consent_not_granted`, not `revoked`: the current consent snapshot
cannot prove revocation history. Owner-scoped misses remain
`resource_not_visible`, since an unscoped lookup solely for logging would
weaken the security boundary.

Start with the request ID returned to the browser:

```bash
LOG_LEVEL=debug AI_LOCAL_INSPECTOR=false LOG_FILE=./theia-api.jsonl make api
jq -c 'select(.request_id == "REQUEST_ID")' theia-api.jsonl
jq -c 'select(.turn_ref == "TURN_REF_FROM_HTTP_EVENT")' theia-api.jsonl
```

Read `http.request` for the actual returned status/code. Read `ai.turn.failed`
for the terminal executing stage. Follow `error_origin`, `failed_check`,
`reason` and `diagnostic_hint` to the source check. `error_site` identifies an
observation/translation point. Earlier `llm.attempt.failed` and
`llm.chain.decision` events distinguish a provider rejection, retry or model
fallback from the later independent authorization denial. A Brain fallback or
degraded Safety event is not by itself evidence that it caused the final 404.

`turnFailure` retains the original error **inside the process**; a small internal
marker preserves the existing HTTP code precedence while traversing its causes.
Public JSON still uses the unchanged sanitized contract. SQL adapters annotate
errors without emitting one log per query. Stage starts are logging metadata
and do not consume the bounded Recorder step budget.

## Short synthetic log sequences

Fields below are abbreviated examples, not captured incident evidence.

Successful turn:

```json
{"level":"INFO","msg":"ai.turn.started","request_id":"b3a462a5-f076-470b-89b0-318d2ed04015","turn_ref":"0123456789abcdef"}
{"level":"DEBUG","msg":"ai.stage.started","stage":"generator","status":"started","turn_ref":"0123456789abcdef"}
{"level":"DEBUG","msg":"ai.stage.completed","stage":"generator","status":"success","duration_ms":720,"fallback_used":false,"turn_ref":"0123456789abcdef"}
{"level":"INFO","msg":"ai.turn.finished","status":"success","failed":false,"degraded":false,"duration_ms":1050,"turn_ref":"0123456789abcdef"}
```

Degraded provider attempt followed by recovery:

```json
{"level":"WARN","msg":"llm.attempt.failed","provider":"configured_provider","model":"primary","attempt":1,"reason":"rate_limit","provider_status":429,"retryable":true,"fallback_used":false,"turn_ref":"1123456789abcdef"}
{"level":"DEBUG","msg":"llm.retry.started","attempt":2,"retry_index":1,"turn_ref":"1123456789abcdef"}
{"level":"DEBUG","msg":"llm.chain.decision","reason":"fallback_started","provider":"configured_provider","model":"secondary","turn_ref":"1123456789abcdef"}
{"level":"DEBUG","msg":"llm.attempt.completed","model":"secondary","fallback_used":true,"turn_ref":"1123456789abcdef"}
```

Failed *secondary retrieved-source* check with the existing concealed HTTP response:

```json
{"level":"ERROR","msg":"ai.stage.failed","stage":"generator","status":"failed","error_code":"forbidden","reason":"consent_missing","failed_check":"retrieved_source_use","duration_ms":4,"turn_ref":"2123456789abcdef"}
{"level":"ERROR","msg":"ai.turn.failed","stage":"generator","status":"failed","error_code":"forbidden","reason":"consent_missing","failed_check":"retrieved_source_use","failure_stage":"generator","diagnostic_hint":"check_source_use_consent_and_mapping","turn_ref":"2123456789abcdef"}
{"level":"INFO","msg":"ai.turn.finished","status":"failed","failed":true,"degraded":true,"turn_ref":"2123456789abcdef"}
{"level":"WARN","msg":"http.request","status":404,"failure_code":"RESOURCE_NOT_FOUND","http_error_kind":"authorization_concealed","failure_stage":"generator","reason":"consent_missing","failed_check":"retrieved_source_use","turn_ref":"2123456789abcdef"}
```

`llm.chain.exhausted` is a chain outcome; `llm.call.failed` covers the complete
call, including preflight, quota, schema and cancellation failures without a
provider attempt. Retryability describes existing client/transaction retry
policy. It is not permission to retry a user turn. Status-specific rejection
metadata does not change the configured retry, fallback or breaker policy.
The closed provider-code allowlist drops even plausible-looking unknown codes;
provider `message`, `param` and raw response bodies never enter ordinary logs.

## Validation and limitations

The standard-library `observability` package can be tested in isolation with
`cd apps/api && GO111MODULE=off go test -race ./internal/platform/observability`.
To validate the entire application (with network access to declared Go modules):
`cd apps/api && go test ./...`, `go vet ./...`, plus the repository's
`python3 scripts/check_structure.py` and `python3 scripts/check_contract.py`.

The detailed snapshot tracer (`trace.Recorder`) still deliberately stores only
bounded summary fields, not raw errors or conversational material. This work
**does not** introduce raw prompt/response capture or a persistent clinical
execution inspector. Not every third-party error supplies construction-site
information; `error_site` marks the best known observation point instead.


Remaining limits: third-party and boolean-only custom policy ports may expose
no finer denial detail; source locations and a conservative policy code remain
available. Some legacy memory/reflection checks share `job_validation`; their
exact source location identifies the branch, but not a separate code for every
compound condition. SQL numeric diagnostics do not include object names or
parameters. Operational correlation remains process-local; it is not durable
cross-replica tracing. The local Execution Inspector remains an independent,
explicitly enabled payload diagnostic facility.


## MON-01B — separate safe export to private Loki (local only)

The application can optionally emit a **second** JSONL stream that contains
only allowlisted operational events. Enable only through the MON-01B Compose
overlay (`compose.logging.dev.yaml`), which sets
`THEIA_SAFE_LOG_EXPORT_ENABLED=true` and provides a dedicated, group-writable
private volume at `THEIA_SAFE_LOG_EXPORT_PATH=/var/log/theia/theia-safe.jsonl`.
This export is filtered *before writing to disk*, even if ordinary logs later
include unsafe attributes. It is not the same as `LOG_FILE` and does not read
Docker engine logs or the full-payload AI Inspector. See
[`MON01B_ALLOY_LOKI.md`](MON01B_ALLOY_LOKI.md) for details and restrictions.
