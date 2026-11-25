# OBS-05A — Dual OTLP export (Phoenix + Langfuse v4)

**Scope:** optional **local/synthetic-only** Go → private Collector → Phoenix AND
Langfuse. No model/RAG/agent code, user API, SQL migration, frontend or auth
contract changes. Phoenix remains the default. OBS-05B will compare the same
Turn and UI experience; this milestone provides export, secured fan-out and
reproducible readback. **Do not run full payload export for real users.**

## Topology

```text
Go (APP_ENV=local, HTTP_ADDR=127.0.0.1:8080)
  | one OTLP/HTTP protobuf request, Authorization: Bearer <collector ingest key>
  v
127.0.0.1:4318 -> Docker Collector (:4318; bearer-authenticated HTTP receiver)
  | memory_limiter -> batch -> independent bounded sending queues
  +----> http://phoenix:6006/v1/traces (existing authenticated Phoenix)
  +----> https://<Langfuse-host>/api/public/otel/v1/traces (Langfuse v4, Basic)
            x-langfuse-ingestion-version: 4
```

The source application **does not create two AI calls**, IDs, span trees or
exporters; one trace and one set of spans is forwarded to both destinations.
Collector never exports payloads to public Grafana/Prometheus/Loki. The local
receiver port is bound to host IPv4 loopback. The Collector container has no
published admin port; receiving HTTP is authenticated via the bearer-token
extension (unlike an unguarded default OTLP receiver). Independent queued
exporters keep one temporarily failing destination from blocking the other,
but their bounded in-memory queues **can drop traces** during overload/restarts.
Sender authentication does not make the host/containers a suitable production
network or protect untrusted local processes. This is a private development
comparison stack with synthetic fixtures ONLY.

## Step 0 — Prerequisites

- Docker/Compose with enough resources, native Go backend and **existing** OBS-04A/B/C/D patches.
- Authenticated Phoenix instance from OBS-02 with a **system API key** recorded
  in `.env.observability` using `init_local_phoenix_env.py --set-ingest-key`.
  Preserve existing Phoenix data and project; do not reset Phoenix.
- A separately provisioned **Langfuse v4** project and project API keys.
  Choose your own approved destination. If using Langfuse Cloud, only send
  synthetic data; do not send real mental-health conversation content.
  For self-hosting, use the official Langfuse deployment/docs in an independent
  stack with its own PostgreSQL/ClickHouse/Redis/object storage and private auth.
  This patch intentionally does **not** deploy those databases or alter their
  lifecycle. HTTPS is required for non-local destinations.

## Step 1 — Private credentials (never commit them)

```sh
python3 scripts/observability/init_local_phoenix_env.py
# If missing: first create a Phoenix system API key in its authenticated UI,
# then enter it using this hidden-input command:
python3 scripts/observability/init_local_phoenix_env.py --set-ingest-key
python3 scripts/observability/init_dual_otlp_env.py
python3 scripts/observability/init_dual_otlp_env.py --configure
```

The second helper generates `.env.dual-otlp` (user-owned, mode 0600), including
an independent random Collector receiver token. `--configure` requests the
Langfuse OTLP **base** URL, project public key and secret key via hidden input.
Its base URL **must end at `/api/public/otel`**, not `/v1/traces`.
For instance `https://cloud.langfuse.com/api/public/otel` (EU region);
use the correct region or self-host address. The helper stores Basic Auth as
base64(public:secret) **which is still a secret**, never a public frontend value.
Do not check in `.env.observability`, `.env.dual-otlp`, Compose-rendered output,
logs with credentials, or application traces containing sensitive content.

For a separate local self-hosted Langfuse container you can explicitly attach
its web container to the private `theia-observability_default` Docker network
with the network alias `langfuse-web` (operator action, not automatic), then set
`http://langfuse-web:3000/api/public/otel` in `--configure`. The helper rejects
arbitrary plaintext HTTP hosts. This option requires starting the Langfuse
stack independently and setting its own authentication first.

## Step 2 — Start existing Phoenix plus the dual Collector

```sh
docker compose -p theia-observability \
  --env-file .env.observability --env-file .env.dual-otlp \
  -f compose.observability.dev.yaml \
  -f compose.observability.dual.dev.yaml \
  --profile observability --profile dual-observability \
  up -d phoenix otel-collector-dual
```

The Collector uses the upstream GHCR image pinned to 0.166.0. Pin the image by
digest and review the configuration before using it on shared infrastructure. Existing
Phoenix remains available at `http://127.0.0.1:6006`. Collector accepts OTLP/HTTP
**only** at `http://127.0.0.1:4318/v1/traces` on the developer host.

**Do not post Docker Compose `config` output or container `env` output**, since
both can expose secrets. For credentials rotation, restart the Collector after
rotating its environment and the native Go receiver key.

## Step 3 — Check authentication and cross-platform readback

```sh
python3 scripts/observability/smoke_dual_otlp.py --preflight
python3 scripts/observability/smoke_dual_otlp.py --ingest
```

This writes one *synthetic metadata-only* probe (plus small negative/auth
probes), without any provider call. The probe verifies:

1. No/invalid Collector Bearer credentials produce 401 or 403.
2. The correct Bearer token causes the Collector to accept OTLP/protobuf.
3. The same 32-digit W3C TraceID appears in authenticated Phoenix project
   `theia-obs05a-dual` AND in Langfuse v4 Observations API (`/api/public/v2/observations`
   with an explicit `traceId` and bounded time window). **Neither outcome is
   assumed from HTTP 200 alone.** Both readbacks must succeed.

A missing readback fails the test. Look at Collector exporter queue error
*metadata* only; do not enable debug payload logs. If you use an alternate
Langfuse project, make sure the supplied project credentials belong to it.

## Step 4 — Run Theia natively with the Collector

```sh
# Metadata-only to begin:
python3 scripts/observability/run_native_with_dual_otlp.py

# Explicitly opt in to synthetic prompt, RAG and agent I/O capture:
python3 scripts/observability/run_native_with_dual_otlp.py --content
```

The wrapper preflights Collector bearer auth, then launches the same native Go
API through `scripts/run_local.py`, with `ALLOW_DEV_LOGIN=false` (AUTH-09), the
original Staff/User BFF identity and `HTTP_ADDR=127.0.0.1:8080`. The app's
**only** OTLP exporter points to `127.0.0.1:4318/v1/traces`. The Go process has
its **own Collector ingress token**, not Langfuse/ Phoenix service credentials.
The backend's `telemetry.Validate` accepts the local authenticated Collector
on numeric loopback port 4318 as an additional opt-in target; it retains the
existing port 6006 direct-Phoenix path and rejects arbitrary remote collectors,
HTTP to LAN hosts, missing/weak private keys and non-local full payload capture.

After a synthetic Turn, get its W3C TraceID using the existing OBS-03 Staff
trace-reference endpoint. Compare exactly the same ID:

```sh
python3 scripts/observability/smoke_dual_otlp.py --inspect TRACE_ID
```

The smoke prints only TraceID/presence and never the prompt, response, token,
source text or raw service response. `--inspect` cannot establish that the
full content is displayed identically in the two UIs; that is OBS-05B.

## Failure behavior, rollback and operational limits

- The API exports via the existing finite-queue batch processor. A Collector
  outage or sink failure must not change user-facing Turn outcomes; traces can
  be dropped during exporter retry/queue exhaustion. No external model calls
  are repeated on retry. Test this with an intentional Collector shutdown.
- Collector in-memory queues are bounded, **not durable delivery**. Track missing
  spans with readback tests, especially after restart. No guarantees of exactly
  once export, ordering or identical UI interpretation.
- No plaintext end-to-end TLS between Go and Collector or Collector and Phoenix;
  these links are restricted to developer loopback and the dedicated Compose
  network. Langfuse external export uses TLS, unless a separately approved
  local Docker `langfuse-web:3000` service is selected.
- `THEIA_OTEL_CAPTURE_CONTENT=false` restores metadata-only output. To roll
  back the entire dual experiment, stop the Collector and run
  `run_native_with_phoenix.py` as before. No schema/API migration, Phoenix
  deletion, reindexing or frontend changes required.
- If a deployment needs a private LAN, Kubernetes or production payload capture,
  design TLS/mTLS, retention, privacy approval, network policies and strict
  content access separately. Do not bypass the local-only validation.

Official references:
- Collector fanout/configuration: https://opentelemetry.io/docs/collector/configuration/
- Collector Bearer authenticator: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/bearertokenauthextension
- Phoenix authentication: https://arize.com/docs/phoenix/deployment/authentication
- Langfuse OTLP v4: https://langfuse.com/docs/integrations/native/opentelemetry
- Langfuse v4 reads: https://langfuse.com/faq/all/deprecated-api-migration
