# OBS-02 — Phoenix deployment and private trace-access design

**Status:** OBS-02 local Phoenix Compose, credentials and smoke tooling are now implemented (verified on local loopback). The Go exporter comes from OBS-01A/B; OBS-03 SQL/Admin trace references and frontend remain future work. See [`../operations/obs02-local-phoenix.md`](../operations/obs02-local-phoenix.md).

## 1. Dev topology and trust boundary

```text
Developer browser (authenticated Phoenix UI on loopback only)
    http://127.0.0.1:6006
                |
      Phoenix container (version pinned + named volume + auth enabled)
                ^
           OTLP/HTTP /v1/traces
                |
   Native Go API / local-only synthetic test runtime
       APP_ENV=local, HTTP_ADDR=127.0.0.1:<Go port>
       metadata span exporter + opt-in private payload exporter
                |
   SQL Server (Turns/Jobs/Audit; metadata-only)

Later, a DIFFERENT sanitized metrics/logs pipeline:
  Go / workers -> Alloy / Collector -> Prometheus/Loki -> Grafana
```

Do **not** route full-payload GenAI spans through the general monitoring collector. One Phoenix instance is enough for local diagnosis; no Tempo, Loki or Grafana is required before a first verified Turn trace. Do not attach Phoenix to the public User or Admin browser host.

## 2. Deployment requirements for the implementation ticket

- Add an **opt-in** Compose profile or file (proposed `compose.observability.dev.yaml`), with `arizephoenix/phoenix` image **pinned to a reviewed release/digest**. Avoid floating `latest` in checked-in reproducible configuration.
- Map **only** `127.0.0.1:6006:6006` for the web UI/OTLP HTTP collector. Use Docker's internal network where available; do not expose unauthenticated `4317/4318` to the host or untrusted LAN. Do not put Phoenix behind the public Go Auth proxy.
- Use a named persistent volume for Phoenix working data. Test that a restart preserves traces and authentication state. SQLite is an initial local-developer storage choice; review PostgreSQL and data retention before shared multi-developer workloads.
- Enable `PHOENIX_ENABLE_AUTH=true` with a unique high-entropy `PHOENIX_SECRET` and nondefault initial admin password via local **untracked** secrets, never committed. Create a scoped ingestion API key and supply it to the **Go exporter only** via a secret mechanism. Do not expose credentials in `VITE_*`, Compose checked-in defaults, URL parameters or logs.
- In a local HTTP-only loopback environment, constrain browser access to loopback and enforce Phoenix's applicable origin/CSRF configuration. For an HTTPS private shared deployment, enable secure cookies and configure a strict IdP integration before any real/sensitive content.
- If the Go process runs **natively**, its OTLP/HTTP trace endpoint should be `http://127.0.0.1:6006/v1/traces`. If Go runs as a container, a service DNS name may be needed, but **the existing local Inspector refuses wildcard (`0.0.0.0`) HTTP listeners**; do not disable that guard to get the containers talking. Decide the test topology before enabling payload capture.
- Configure exporter signal-specific URL correctly: OTLP/HTTP `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` includes `/v1/traces`; using the generic base URL `OTEL_EXPORTER_OTLP_ENDPOINT` constructs signal-specific paths. Verify against the pinned Go SDK and container version, not a guessed endpoint.
- Decide whether to integrate Phoenix UI with existing Keycloak **as a separate reviewed client** in a later iteration. Initially use its own authenticated local interface and give verified Staff a protected external link; the Go BFF's Staff cookie does **not** grant Phoenix access by itself.

Official deploy/auth references:

- https://arize.com/docs/phoenix/self-hosting/deploying-phoenix
- https://arize.com/docs/phoenix/deployment/authentication
- https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/

## 3. Proposed trace link and Admin API behavior (OBS-03, NOT present)

Add a sanitized, staff-only, audited trace-reference endpoint, after `ai_trace_refs` has an approved migration and repository:

```text
GET /api/v1/admin/turns/{turnId}/trace-references
```

**Proposed response shape** (draft, not yet OpenAPI):

```json
{
  "turnId": "00000000-0000-4000-8000-000000000001",
  "items": [
    {
      "traceId": "0123456789abcdef0123456789abcdef",
      "kind": "turn",
      "jobId": null,
      "attempt": null,
      "startedAt": "2030-01-01T00:00:00Z",
      "captureLevel": "metadata",
      "contentAvailable": false
    }
  ],
  "nextCursor": null
}
```

The backend verifies the active Staff session (`RequireStaff`), writes a durable audit event **before** reading trace refs, queries only bounded metadata, and returns `Cache-Control:no-store`. A failed audit write must return `503`, not silently serve the result. Query must not read local Inspector files, provider bodies, conversation messages, profile PII or arbitrary SQL. Use code-owned Phoenix project URL configuration and a strict allowlist if generating a browser deep link. Validate exact Phoenix route/deep-link syntax for the pinned Phoenix version; do not guess a URL from a TraceID. Current single `admin` role is sufficient for metadata-only reads; raw content access must be a **separate** reviewed permission/operation, even in synthetic development.

For an initial **Dev-only** experience, it is acceptable for the front-end to display Turn metadata and a link to Phoenix's authenticated search page, where the operator manually searches by approved TraceID/TurnID. A future audited proxy endpoint may provide an in-Back-Office prompt/response detail view, but must implement a separate authorization, rate/size bound, retention semantics, audit action (who opened which trace) and redaction. Do not reuse `GET /api/v1/admin/turns/{turnId}` to silently return full content: it is explicitly metadata-only today.

## 4. Security and operational acceptance

1. Phoenix UI is unreachable from the LAN; only the local operator/browser with a valid Phoenix login can inspect traces.
2. A Go exporter authenticates to Phoenix using a **server-held** API key; unauthenticated trace writes fail.
3. Disabled full-payload mode sends no prompt/response content. Enabled local synthetic mode captures the exact rendered provider payload and reply without emitting raw messages in SQL metadata or logs.
4. A failed exporter or stopped Phoenix never changes the user-facing Turn result, provider retry count, billing/usage or AI state commit.
5. Restarted Phoenix retains only intended named-volume data; an operator can explicitly reset **synthetic** Dev traces, with documented scope. No automatic deletion of user conversation/source-of-truth rows.
6. Simulate wrong TraceID, missing Staff session, non-Staff user cookie, audit DB outage and private network failure; never reveal whether a Turn belongs to another User through a public route.
7. Capture completeness is explicit: truncated/partial provider bodies and interrupted workers must not render as a verified complete trace.

## 5. Retention and later production posture

Local synthetic traces: propose a **7-day default maximum retention target**, implemented with a reviewed Phoenix-compatible pruning/export strategy (not an unverified assumption that the container enforces TTL). Avoid shipping raw traces to external services. Back up only if needed for reproducible synthetic debugging and protect backup storage. Shared/production tracing should default to metadata-only with strictly scoped role/consent requirements and separate legal/privacy review for any content capture.
