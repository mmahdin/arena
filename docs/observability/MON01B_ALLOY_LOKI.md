# MON-01B — Private structured logging with Grafana Alloy + Loki

**Status:** Go and Python checks passed. The local API spool, Alloy ingestion,
Loki readback, and the authenticated Grafana dashboard were exercised with
synthetic traffic. The live check used locally built containers from verified
upstream release binaries and the existing native Grafana service after Docker
Hub returned HTTP 403. The standard Compose configuration passed validation.
**Scope:** Synthetic-only local Compose overlay, operational logs only; no new
admin HTTP endpoint, SQL migration, backend business functionality or staff
permission. This is not a production logging topology or an audit record.

## Boundary and data flow

```text
Go slog (existing stderr + optional private LOG_FILE)
    | additional in-process SafeLogWriter (strict allowlist)
    v
Docker named volume: /var/log/theia/theia-safe.jsonl[.1..3]
    | read-only mount, no Docker socket and no Inspector volume
    v
Grafana Alloy loki.source.file -> loki.process (JSON + level only)
    | private Compose bridge network
    v
Grafana Loki (TSDB v13, local filesystem, 72-hour retention)
    v
Grafana (authenticated, 127.0.0.1:3000) - Theia Loki datasource + Engineering Logs
```

The full-payload OBS-01B Inspector and Phoenix OTLP capture are **not**
forwarded to Loki. The normal `LOG_FILE`, stdout, container engine log driver,
Docker socket and SQL audit are also excluded. Only the generated spool file is
mounted in Alloy. Loki stores operational *metadata*, not an exhaustive
execution trace or authoritative user actions. Staff audit remains in SQL.

## Go implementation

- `internal/platform/observability/safe_loki.go`: `sanitizedLogRecord`
  reconstructs each event using hard-coded known message names. Every arbitrary
  string, unrecognized code, object, stack, prompt, response, error message,
  user ID, model, phone, URL and credential is omitted, even if emitted by a
  future broken logger. Only preapproved enum labels, durations/counts and
  validated opaque `request_id`/`turn_ref`/OTel IDs are copied.
- Export is disabled unless `THEIA_SAFE_LOG_EXPORT_ENABLED=true`,
  `APP_ENV=local`, and `THEIA_SAFE_LOG_EXPORT_PATH` is an absolute file named
  `theia-safe.jsonl`, in an existing non-world-writable directory. Initial file access and
  permission checks fail the API startup; export write errors are best effort
  and emit a fixed error marker on stderr at most once.
- Export records are UTC JSONL with schema `theia.safe-log.v1`; each record is
  an atomic JSON object. IDs remain *fields only*, never indexed Loki labels.
  The `turn_ref` is a process-local pseudonym; it does not directly encode a
  Turn UUID or W3C TraceID. For durable Turn lookup use OBS-03 Admin APIs.
- The spool is capped at 4 MiB plus three 4 MiB rotation backups (approximately
  16 MiB per API container). Rotating with file tailing is **best-effort**;
  aggressive rotation can drop or duplicate diagnostic entries. Neither this
  spool nor Loki is suitable for billing or append-only audit guarantees.
- `SafeLogWriter` is not affected by `LOG_FILE` and does not forward its bytes
  to Loki. If a new event type is needed, review and explicitly add its name
  and safe fields to the allowlist with negative tests.

## Compose local setup

Requires **all three files** and the same project name as the existing local
SQL/API stack. Re-use the MON-01A generated `.env.monitoring` (0600):

```sh
python3 scripts/monitoring/init_local_monitoring_env.py

docker compose -p theia-local \
  --env-file .env --env-file .env.monitoring \
  -f compose.yaml -f compose.monitoring.dev.yaml -f compose.logging.dev.yaml \
  --profile monitoring up -d --build \
  logging-init migrate api prometheus loki alloy grafana
```

`logging-init` prepares three named volumes. The API runs as its existing
non-root Docker user and receives supplemental GID 2000; the setgid spool
folder allows write. Alloy runs as UID/GID 473 with supplemental GID 2000
and mounts the spool **read-only**. Loki runs as UID/GID 10001. The init job
only sets ownership/permissions and does not handle application credentials.
Nothing publishes ports 3100/9095/12345; Grafana remains available on
`http://127.0.0.1:3000` with its existing login requirement. `auth_enabled:
false` in Loki is acceptable **only** inside this local isolated network.
Never publish Loki or use this overlay for real-user/staging/production data.

Grafana -> Explore -> data source **Theia Loki**. Example LogQL:

```logql
{service_name="theia-api", environment="local"} | json
{service_name="theia-api", environment="local", level="ERROR"} | json
{service_name="theia-api", environment="local"} | json | event="http.request"
{service_name="theia-api", environment="local"} | json | event="ai.turn.failed"
```

The provisioned **Theia · Engineering Logs** dashboard displays volume by
severity and the recent metadata-only log lines. Filter component, event,
error_code, request_id or turn_ref at **query time** with `| json`, not via
high-cardinality indexed labels. Unapproved or unknown code values are omitted
rather than sent as arbitrary user-controlled strings.

## Acceptance tests

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v
(cd apps/api && go test ./internal/platform/observability ./internal/platform/config)
(cd apps/api && go build ./... && go test ./...)
```

Offline standalone sanitizer test (uses only Go standard library; useful when
unrelated Go module dependencies are not in the cache):

```sh
cd apps/api/internal/platform/observability
go test safe_loki.go safe_loki_test.go -count=1
```

Live Compose verification on a Docker-capable host:

1. `docker compose ... config`: check resolved mounts, network and non-root
   identities. Confirm Loki/Alloy have no `ports` and there is no Docker
   socket mount or `LOG_FILE` mount.
2. Generate an HTTP 404 or another safe synthetic request. In the API
   container, `ls -l /var/log/theia/theia-safe.jsonl` should show mode `0640`
   and safe JSON events. Do not print the raw Inspector files or `.env`.
3. In Grafana Explore, run the LogQL above and confirm the `http.request`
   with its numeric HTTP status and safe route template.
4. Verify Alloy remains healthy across restart and retains its file offsets.
   Trigger enough synthetic events to rotate the spool; look for older rows.
5. Verify no prompts, responses, tokens, URLs, arbitrary error strings,
   user/conversation IDs, private Inspector files, or provider bodies appear.
   Run the isolated Go sanitizer tests, which inject these values deliberately.
6. Confirm retention settings in Loki and that queries older than 72h are
   bounded. File-system disk pressure is **not** automatically controlled by
   the Loki retention configuration; monitor filesystem usage separately.
7. Run `python3 scripts/monitoring/smoke_loki.py` after generating traffic.
   It checks anonymous datasource denial, authenticated Grafana-to-Loki
   readback, allowlisted JSON fields and absence of high-cardinality labels,
   printing counts only (no records or credentials).
8. Confirm dashboard access requires Grafana authentication. Direct host
   access to Loki, Alloy and Prometheus must remain unavailable.

## Limitations / future work

- The supplied historical backend snapshot has `.env` and `.env.auth`
  tracked in Git; this logging patch does not touch them. Review and rotate
  any real credentials and untrack these files separately.
- No production ingress, SSO, HA, centralized secrets, encrypted log storage,
  fine-grained Loki tenant isolation or proven erasure workflow.
- Metadata-allowlisting protects Loki even if ordinary logs accidentally
  contain text, but no sanitizer proves that all diagnostic combinations are
  non-identifying. The full-payload Dev Inspector remains a separate,
  explicitly authorized synthetic-only system.
- MON-02 adds Grafana alerts, alert routing, an on-call/runbook workflow, and
  optional distributed Tempo traces where worthwhile. Avoid duplicate alert
  routing if already using Alertmanager.
