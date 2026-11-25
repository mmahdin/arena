# API diagnostics (local development)

The API now emits **structured JSON Lines** logs using Go `slog`; timestamps
use `BUSINESS_TIMEZONE` (default `Asia/Tehran`) and include source file/line.
No external logging dependency, migration, or HTTP diagnostic endpoint is needed.

## Start and follow the logs

- `LOG_LEVEL=info` (default): startup/shutdown events, HTTP responses, AI-turn
  boundaries, warnings, errors, background failures. Health checks at success
  are logged at `debug` to reduce noise.
- `LOG_LEVEL=debug`: adds per-stage AI Core observations (`admit`, `safety`,
  `retrieval`, `brain`, `generator`, `validator`, `finalizer`, etc.), route choices,
  and successful health checks.
- `LOG_FILE` (optional): a writable **file path** to append JSONL records, in
  addition to stderr. Parent directory must exist. Files are mode `0600`.
  Provide log rotation/retention externally; file writes have no built-in size cap.
  On Docker Compose, stderr is visible via `docker compose logs -f api`.
  If setting `LOG_FILE` inside a container, make sure the path and volume exist.

For a direct local run, from the repo root:

```sh
mkdir -p apps/api/logs
LOG_LEVEL=debug LOG_FILE=./logs/api.jsonl python3 scripts/run_local.py go run ./cmd/api
```

The runner changes directory to `apps/api`, so `LOG_FILE` is relative to that directory.
The sample above uses the existing `.env` loader; no keys or secrets should be
added to the log file. `LOG_FILE` can also be omitted to use console-only logs.
For Compose:

```sh
LOG_LEVEL=debug docker compose up -d api
LOG_LEVEL=debug docker compose logs -f api
```

## Find where an error happened

Every HTTP request gets a new server-generated `X-Request-ID`. The same ID is
returned in a `application/problem+json` response's `requestId` field and in
the matching `http.request` log entry. Look for these event names:

| Event | Meaning |
| --- | --- |
| `http.request` | Method, **route template**, HTTP status, elapsed ms, `request_id`, response code and safe error details |
| `http.panic` | Recovered panic with **panic type** and source stack, never its value |
| `api.configuration.invalid` | Configuration field validation failed (sent to stderr) |
| `api.database.connect_failed` | SQL Server connection failed |
| `api.composition.failed` | AI Core/bootstrap failed; `setup_step` pinpoints the phase |
| `ai.bootstrap.failed` | Specific constructor stage within AI Core (e.g., `brain_assets`, `memory_repository`, `orchestrator`) |
| `ai.turn.started`, `ai.turn.finished` | Turn lifecycle, correlated by pseudonymous `turn_ref` |
| `ai.stage` | Stage name and latency; degraded stages at `warn`, ordinary stages at `debug` |
| `ai.turn.failed` | Failing AI stage and typed failure code |
| `ai.turn.admission_failed` | Failure before the stage tracer is available |
| `ai.turn.cleanup_failed` | Failed state cleanup after turn failure |
| `ai.worker.panic` | Unhandled memory/reflection/scheduler panic; stack and type only |
| `ai.worker.retry`, `ai.worker.recovered` | Memory/reflection job retries, with 30-second warning throttling |
| `ai.worker.failed`, `ai.core.stopped` | Terminal scheduler/background worker failure |

Example search, in a directory with an API log file:

```sh
# See server errors with the associated request IDs and stages.
jq -c 'select(.level == "ERROR") | {time, msg, request_id, turn_ref, route, failure_code, ai_step, stage, setup_step, error_type, root_cause_type}' apps/api/logs/api.jsonl

# Follow a single failing request.
jq -c 'select(.request_id == "PASTE-X-REQUEST-ID-HERE")' apps/api/logs/api.jsonl

# Follow an AI turn across stages (take turn_ref from the HTTP or AI event).
jq -c 'select(.turn_ref == "PASTE-TURN-REF-HERE")' apps/api/logs/api.jsonl
```

**Privacy limits:** Logs intentionally exclude messages, prompts, model
responses, raw SQL/provider errors, request URLs/query strings, request and
response bodies, cookies, tokens, IPs, and real user/turn IDs. Error values can
contain sensitive text, so only error *types*, safe codes, and known stage names
are logged. `turn_ref` is a randomized process-local pseudonym and changes on
restart; it is not a durable identifier. Stack traces are recorded for panics,
not for arbitrary returned Go errors (which do not carry call stacks by
default). Detailed prompts and intermediate AI payloads are **not** available
from this operational logger; those require a separately consented, protected
local execution-inspection facility.

**Ops caution:** An HTTP `5xx` does not by itself prove which deep function
failed; check `error_site`, `error_type`, `root_cause_type`, `ai_step`, and the
correlated `ai.stage`/`ai.turn.failed` events. Diagnostic log files must be
access-controlled, rotated, and deleted according to deployment policy.
