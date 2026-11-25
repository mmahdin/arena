# MON-02 — Local incident alerts, response and validation

This runbook is for the **synthetic local development stack only**. Alert
thresholds are starter thresholds, not production SLOs or clinical-safety
thresholds. Theia does not provide an emergency-response service. No real-user
content, phone numbers, prompts, responses, API keys or trace payloads enter
this pipeline. SQL audit and AI Core persistence are unchanged.

## Architecture and single notification path

```text
Theia /metrics (127.0.0.1:9100 in API namespace)
   -> Prometheus (evaluates bounded rules; persists local metrics)
   -> Alertmanager (deduplicates, groups, routes, stores local silences)
   -> Mailpit SMTP (internal only, 1025)
   -> Local developer inbox (127.0.0.1:8025)

Grafana (127.0.0.1:3000, authenticated)
   -> Prometheus datasource (alert rule state + engineering dashboard)
   -> Alertmanager datasource (alert groups, silences)
```

**Never enable a second set of Grafana-managed rules for these same incidents.**
Grafana has no provisioned alert evaluator/notification path in MON-02;
`handleGrafanaManagedAlerts:false` deliberately prevents duplicate delivery.
The sole contact point is local Mailpit. No real email leaves Docker Compose.

The original `compose.monitoring.dev.yaml` remains usable on its own: its
Prometheus configuration does not require Alertmanager. The new overlay
replaces the Prometheus configuration file at the SAME destination and mounts
rule files. It also adds a separate, private Alertmanager and optional local
inbox. Add the overlay to your existing Compose project; never start a second
SQL Server or API environment for monitoring.

## Start (using the same local Compose project)

Prerequisites: MON-01A and MON-01B applied (the logging overlay is optional),
Go dependencies resolved and SQL Server migrations applied. Docker Compose,
working local API, and `.env` / `.env.monitoring` available. Do not expose the
private metrics listener or Prometheus, Alertmanager or SMTP ports to the host.

```sh
python3 scripts/monitoring/init_local_monitoring_env.py

docker compose -p theia-local \
  --env-file .env --env-file .env.monitoring \
  -f compose.yaml -f compose.monitoring.dev.yaml \
  -f compose.logging.dev.yaml -f compose.alerting.dev.yaml \
  --profile monitoring up -d --build \
  logging-init alerting-init migrate api prometheus loki alloy alertmanager mailpit grafana
```

If MON-01B logging is not deployed, omit `compose.logging.dev.yaml` and the
`logging-init loki alloy` service names. If the API already runs under another
Compose project, use that project's name instead of `theia-local`.

- Grafana: `http://127.0.0.1:3000` (authenticated; existing 0600 secret).
- Local alert inbox: `http://127.0.0.1:8025` (loopback-only; test data).
- Alertmanager and Prometheus have **no host-published ports**. Use Grafana
  Alerting -> Choose Alertmanager -> `Theia Alertmanager` for active alert
  status and silences. Receiver and route config are provisioned from source
  files, not edited in Grafana.
- Mailpit inbox is intentionally unauthenticated **only on local loopback**;
  never expose or reuse it with real data or team-shared ingress.

## Config checks before generating alerts

Prometheus and Alertmanager images include offline config validators. After
starting their containers, run:

```sh
docker compose -p theia-local \
  --env-file .env --env-file .env.monitoring \
  -f compose.yaml -f compose.monitoring.dev.yaml \
  -f compose.alerting.dev.yaml --profile monitoring \
  exec -T prometheus /bin/promtool check rules /etc/prometheus/rules/theia.rules.yml

docker compose -p theia-local \
  --env-file .env --env-file .env.monitoring \
  -f compose.yaml -f compose.monitoring.dev.yaml \
  -f compose.alerting.dev.yaml --profile monitoring \
  exec -T alertmanager /bin/amtool check-config /etc/alertmanager/config.yml
```

Confirm Prometheus scrape targets for both `theia-api` and
`theia-alertmanager` are healthy. All nine rule definitions should appear in
Grafana/Prometheus. No alert is expected for an API that has too little
synthetic traffic to exceed the minimum sample thresholds.

## Verify actual notification delivery (synthetic only)

The fixed one-shot probe is a separate profile, not a permanent service.
After the main alerting stack is already running, issue:

```sh
docker compose -p theia-local \
  --env-file .env --env-file .env.monitoring \
  -f compose.yaml -f compose.monitoring.dev.yaml \
  -f compose.alerting.dev.yaml \
  --profile alert-test run --rm --no-deps alert-smoke
```

The probe posts exactly one fixed `TheiaMON02Smoke` event to the **internal**
Alertmanager API v2. It does not call the Go API, involve an LLM, or provide
untrusted labels. Alertmanager waits at least `group_wait:15s` before sending
Mailpit the local test notification. Check the local inbox, then:

```sh
python3 scripts/monitoring/smoke_alerting.py --expect-smoke
```

Without `--expect-smoke`, the command checks only Grafana's Prometheus and
Alertmanager connectivity plus Mailpit availability; **it does not prove
notification delivery**. The scripts print counts only, not email bodies or
credentials. Test delivery must also be manually verified in Grafana.

## Rule semantics and initial thresholds

| Alert | Condition | For | Priority |
| --- | --- | --- | --- |
| TheiaAPIMetricsTargetDown | API scrape returns `up=0` | 1m | critical |
| TheiaAPIMetricsTargetMissing | `up{job="theia-api"}` series absent | 2m | critical |
| TheiaAlertmanagerTargetDown | Notification receiver scrape `up=0` | 1m | critical |
| TheiaAPIHighServerErrorRate | >=20 requests and >=5 5xx responses in 5m; >10% 5xx | 2m | warning |
| TheiaAPIHighP95Latency | >=20 HTTP observations in 5m and p95 >2s | 3m | warning |
| TheiaLLMHighFailureRatio | >=10 persisted attempts and >=3 failures in 10m; >30% failed | 2m | warning |
| TheiaWorkerNeverReportedHeartbeat | AI Core expects worker but no SQL-backed heartbeat ever observed | 3m | warning |
| TheiaWorkerHeartbeatStale | Worker last reported running, no successful SQL heartbeat for >90s | 1m | warning |
| TheiaWorkerNotRunning | Expected worker last reported stopped/failed | 1m | warning |

Worker expectations are emitted only if AI Core was initialized. Merely
starting a metrics-only API must not cause alerts for nonexistent workers.
A heartbeat is evidence of SQL write/liveness, not proof of job progress.
When Prometheus/Alertmanager itself is unavailable, notifications **cannot**
reliably be delivered by that same instance; monitor it externally before
using alerts as a production on-call mechanism.

## Target down

1. Check `api` and `prometheus` Compose container health, process restarts and
   the independent private metrics listener. Avoid publishing ports 9090/9100.
2. Check SQL migration status and application startup errors in approved
   metadata-only Grafana Loki logs.
3. Confirm Prometheus is scraping the API loopback within its shared network
   namespace. Never use a public `/metrics` route as a workaround.

## Notification pipeline

1. Check `alertmanager` and `mailpit` are running in the same Compose project.
2. Use Grafana's external Alertmanager datasource and verify rules/receiver.
3. If Alertmanager is down, its own `TargetDown` alert will be visible in
   Prometheus, **but email delivery may also be down**. Do not treat this as an
   independent dead-man monitor.
4. Trigger the fixed synthetic smoke alert to test one delivery. If expected
   messages are absent, inspect internal service status and routing config;
   do not switch to real email as a quick fix.

## API errors or latency

1. Verify sufficient traffic and inspect the five-minute histogram/counter
   window in Grafana. Don't interpret missing series as zero failures.
2. Check the `Theia Engineering Operations` dashboard and sanitized Loki
   `http.request` / `ai.turn.failed` events. Link to a Turn via approved Admin
   lookup and OBS-03 trace mapping, not unbounded raw SQL or prompts in logs.
3. Investigate database health, slow processing, retry and provider failure
   before changing timeouts or deployment configuration.

## LLM failures

1. This metric counts **successfully SQL-finalized physical attempts**; a
   transient failed SQL journal does not increment the counters. Ratios are not
   conversation failure rates or clinical quality outcomes.
2. Use Admin Turn Inspector and private Phoenix to inspect authorized
   synthetic attempts. Do not copy prompts or responses into the alert.
3. Use real sample sizes before changing provider settings.

## Worker heartbeats

1. Inspect `theia_worker_expected`, `theia_worker_running`, and
   `theia_worker_heartbeat_success_timestamp_seconds` by fixed worker kind.
2. A missing worker expectation means no AI Core was loaded into this process;
   it is not an indication that workers are healthy.
3. Review local service/SQL availability and safe `ai.worker.*` events. A
   stale timestamp must be investigated even if the last state is `running`.
4. Never automatically retry AI work or mutate user data through alert actions.

## Security, rollbacks and remaining acceptance gates

The alert rules and email notifications contain only fixed service names,
component/severity and approved Prometheus metric labels. No payloads, raw
error messages, phone numbers, user identifiers or secrets. Mailpit stores only
short-lived synthetic messages (max 200, max age 2d); it is not a production
email service or incident archive. Incident response cannot promise clinical
or emergency support, and alerting does not prove production readiness.

Stop alerting by omitting `compose.alerting.dev.yaml` and recreating
Prometheus using the original MON-01A config. Stop the Alertmanager/Mailpit
services within the *same* project; do not destroy the SQL volume. Keep
`compose.logging.dev.yaml` when logging is still needed. No migration or API
contract changes are involved.

Local validation passed for Compose configuration, the Prometheus and
Alertmanager configs, synthetic evaluation of all nine rules, Go tests and
targeted races, live scrapes, and delivery of the fixed smoke notification to
Mailpit. The live check reused the existing host-network API and Prometheus,
native Grafana, and the pinned Alertmanager and Mailpit images from their
upstream registries. Mailpit's inbox was published on loopback, with SMTP and
Alertmanager confined to Docker networking.
