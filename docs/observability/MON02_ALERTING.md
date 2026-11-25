# MON-02 — Operational alerting implementation

This is a local synthetic-only alerting slice built on MON-01A/MON-01B. It
introduces no new external Admin API, SQL migration or autonomous user action.

**Sources:** `infra/monitoring/alerts/theia.rules.yml`,
`infra/monitoring/prometheus.alerting.yml`,
`infra/monitoring/alertmanager/config.yml`,
`compose.alerting.dev.yaml`, and the Grafana Alertmanager datasource and
`theia-alerts.json` dashboard. The existing monitoring config is preserved
for independent MON-01A use.

**App change:** the bounded `theia_worker_expected{kind}` gauge is initialized
for exactly memory, reflection and scheduler only when AI Core is configured;
no heartbeat is fabricated. This lets a rule distinguish a worker that has
never successfully persisted a SQL heartbeat from an unconfigured worker.

**Routing:** Prometheus is the sole alert evaluator. One private standalone
Alertmanager groups and deduplicates, and sends fixed-label synthetic
notifications to internal Mailpit SMTP only. Grafana is a read-only UI for the
external Alertmanager and does not forward Grafana-managed alerts. No real
external recipient is configured or exposed, and no host port is published
for Prometheus/Alertmanager/SMTP.

**Thresholds:** fixed starter warnings for 5xx ratio (>10% with >=20 requests
and >=5 failures), HTTP p95 (>2s with >=20 samples), LLM physical-attempt
failure ratio (>30% with >=10 attempts and >=3 failures), worker heartbeat
freshness and worker stopped/absent, and critical scrape target failure.
Low volume does not trigger percentage-based alerts. A dead Alertmanager
cannot page about itself; external health monitoring is a future requirement.

**Security:** only code-owned, low-cardinality metrics enter alert labels.
Never include Turn/User/Conversation IDs, prompt/response text, provider raw
errors, model secrets, auth headers, or arbitrary URLs in labels/annotations.
The local inbox is synthetic-only and loopback-bound. For production, add
separately reviewed authenticated notification destinations, independent
external dead-man checks, on-call owners, private network, TLS and retention.

**Runbook:** [`../operations/mon02-alerting.md`](../operations/mon02-alerting.md)
contains configuration checks, safe synthetic end-to-end notification test,
incident troubleshooting, alert interpretation, rollback and release gates.
