# MON-01A — Private local Prometheus + Grafana for Engineering

MON-01A builds on OBS-03 for local synthetic development. The Go module includes
the Prometheus dependency and its checksums. Use Go 1.23.0 or later and run the
checks below. No backend database migration or OpenAPI change is needed.

## Exact scope

- Metrics enabled only with `APP_ENV=local`, `THEIA_METRICS_ENABLED=true` and
  numeric loopback `THEIA_METRICS_ADDR` (default `127.0.0.1:9100`). Strict
  validation refuses all-address, remote and stage/production listeners.
- The `/metrics` listener belongs to `cmd/api` on a **separate port**, not
  Gin, BFF, Admin, or User HTTP. Never reverse-proxy this port publicly.
- Official Go Prometheus client (`client_golang` 1.23.2, compatible with
  Go 1.23) and a *private per-process registry*: Go/runtime and process metrics,
  HTTP counters/latency/in-flight, physical LLM-attempt results/latency/known
  tokens/configured-cost estimate, worker last-reported running state and
  last successful SQL heartbeat. HTTP health probes are omitted.
- HTTP labels come **only** from Gin route *templates* (e.g.
  `/api/v1/admin/turns/:turnId`), never actual URLs, queries, headers, IPs,
  session identities or request bodies. Method and status labels are bounded.
  LLM label values are fixed enumerations, not arbitrary provider/model or
  dynamic identity fields. Worker gauges have three fixed kinds only.
- Existing SQL `ai_llm_attempts` remains the authoritative, per-attempt durable
  accounting store. Metrics count a terminal provider attempt **after** its
  SQL Finish succeeds. The local counter may reset on process restart;
  it is not an exactly-once billing ledger or per-turn cost attribution.
  Tokens/cost are partial when provider reporting or pricing is missing.
- SQL worker heartbeat is authoritative for reporting freshness. A gauge of
  `1` means the last *successful* SQL write said running; it is not proof of
  current progress. Dashboard users must also check the age timestamp; missing
  metrics mean no observation, not zero errors.
- Private Compose overlay `compose.monitoring.dev.yaml` **must be combined
  with** `compose.yaml` under the *same project*. Prometheus shares the Go API
  network namespace to reach its loopback-only endpoint, listens on `9090`
  within the app's Compose network and has no host port. Grafana alone publishes
  `127.0.0.1:3000`; signup, anonymous access and outbound update/analytics
  are disabled. An owner-only generated local secret initializes Grafana.
- Dashboard is provisioned in `infra/monitoring/grafana/dashboards` and never
  queries synthetic user data or prompt/response content.

## Apply the patch, update dependencies, verify Go

```sh
git apply --check THEIA_MON01A_PRIVATE_PROMETHEUS_GRAFANA.patch
git apply THEIA_MON01A_PRIVATE_PROMETHEUS_GRAFANA.patch
cd apps/api
go mod tidy  # network required, creates client_golang checksums and upgrades only as needed
go build ./...
go test ./...
go test -race ./internal/platform/metrics ./internal/app/composition
cd ../..
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v
```

Inspect `go.mod` and `go.sum` after dependency updates. The live acceptance checks
below exercise the scrape target, Grafana authentication, and application metrics.

## Start the local, synthetic-only Compose stack

```sh
python3 scripts/monitoring/init_local_monitoring_env.py

docker compose -p theia-local \
  --env-file .env --env-file .env.monitoring \
  -f compose.yaml -f compose.monitoring.dev.yaml \
  --profile monitoring up -d --build migrate api prometheus grafana
```

If using an existing `compose.yaml` project, replace `theia-local` with the
**same** project name. Never run the overlay alone. Never use this overlay with
`APP_ENV=stage`/production or an HTTP API exposed outside local development.

Open `http://127.0.0.1:3000` and sign in as `admin`; the initial password is in
`.env.monitoring` (do not paste it in issue trackers, logs or prompts). The
**Theia · Engineering Operations** dashboard is available under Theia Engineering.

Verify from inside the shared API/Prometheus network namespace:

```sh
# Check that Prometheus has its own live target:
docker compose -p theia-local \
  --env-file .env --env-file .env.monitoring \
  -f compose.yaml -f compose.monitoring.dev.yaml \
  --profile monitoring exec -T prometheus \
  wget -qO- 'http://127.0.0.1:9090/api/v1/targets?state=active'
```

Prometheus containers do not guarantee `wget` is installed. If the above is
unavailable, inspect the Prometheus target page through an authenticated
Grafana query or a temporary network-namespace diagnostic container. **Do not
publish `9090`/`9100` just to run a smoke test.**

Test requests to `http://127.0.0.1:8080/health/live` (health itself is
intentionally not counted) and any existing API path. The overview should
reflect actual traffic after a few scrapes. Without an AI Core configuration,
LLM and Worker metrics may be absent; do not fabricate sample data.

For native Go local development only:

```sh
cd apps/api
APP_ENV=local THEIA_METRICS_ENABLED=true \
  THEIA_METRICS_ADDR=127.0.0.1:9100 \
  go run ./cmd/api  # other project settings (SQL, etc.) must be available
cd ../..
python3 scripts/monitoring/smoke_metrics.py
```

This direct native test is **not** the same network topology as the Compose
Prometheus deployment; native Go requires the project's regular SQL/Auth/AI
runtime environment. Do not enable full-content OBS-01B outside its separate
native-loopback synthetic inspector environment.

## Acceptance gates

1. `go mod tidy`, `go build`, `go test`, and `go test -race` pass.
2. Local Grafana login is required, with no public host-bound metrics ports.
3. Prometheus target `theia-api` is up. Generate requests with path IDs;
   verify labels use registered `:param` templates, never raw identifiers.
4. Exercise a synthetic physical LLM attempt with known provider usage;
   compare the latest SQL attempt status and monitored counter. Failure to
   persist SQL accounting must not generate an LLM success metric.
5. Stop the worker; verify status and heartbeat age are updated or stale.
6. Run negative tests for remote bind, unauthenticated Grafana and sensitive
   label exposure. Review dashboard data semantics for partial usage/cost.

## Follow-ups

- **MON-01B:** Grafana Alloy + Loki with an explicit allowlist of structured
  metadata-only JSON log events, private ingest, safe retention and *no*
  Docker socket/whole-container log harvesting. Do not forward Phoenix
  payloads or private Inspector files into Loki.
- **MON-02:** Alerting, stale worker and provider failure thresholds, optional
  Tempo, incident runbooks and evaluation regressions. Choose one primary
  alert-routing system to avoid duplicate pages.
- Production monitoring requires its own private scrape network, service
  identity, secrets management, Prometheus/Grafana access hardening, retention
  policy and privacy review. This local overlay is not a production manifest.
