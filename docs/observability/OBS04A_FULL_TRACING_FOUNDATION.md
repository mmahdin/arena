# OBS-04A — Full Tracing Foundation (Theia)

**Owner:** Backend Go (`platform/config`, `platform/telemetry`, `ai/adapters/llmhttp`).
**State:** Source implementation; Go unit/e2e outcome must be verified in the target environment.
**Dependencies:** Existing OBS-01A/B provider capture, OBS-02 authenticated local Phoenix, OBS-03 trace ID mapping.

## Result

- `THEIA_OTEL_CAPTURE_CONTENT=true` is now accepted by **the normal native
  loopback API after AUTH-09**. It does **not** require the retired Dev Session,
  `AI_LOCAL_INSPECTOR`, a new capture profile, or a third synthetic confirmation flag.
- It remains an explicit **server-side** opt-in. Default is metadata-only.
  Only `APP_ENV=local`, a true loopback `HTTP_ADDR`, and an authenticated private
  Phoenix OTLP/HTTP destination (`127.0.0.1:6006/v1/traces`, valid server-held
  `THEIA_OTEL_CONTENT_API_KEY`) are accepted in content mode. This prevents
  accidentally sending full data to a remote/unknown collector; it does **not**
  determine whether a locally authenticated user's text is genuinely synthetic.
- OpenTelemetry no longer silently truncates string attributes at 128 chars;
  it now permits 128 events and 16 links per span. A finite span attribute count,
  exporter queue and receiver request limits remain to prevent resource exhaustion.
- Existing JSON/SSE wire bodies can now be retained up to **1 MiB** per request
  or response, up from 16 KiB, using the *same constant* in the HTTP reader
  and telemetry scrubber. Oversized/invalid/unparseable bodies are explicitly
  marked `omitted` with an omission reason, byte count and hash. No silent
  truncation, no extra model call and no additional reads from the provider.
- The ordinary metadata `Set`/`Annotate` allowlist remains closed to arbitrary
  text. Provider keys, Phoenix ingestion keys, bearer tokens, cookies and
  secret-named JSON properties continue to be scrubbed. These heuristics are
  **not** a guarantee of de-identification of real transcripts.

## Run with existing local Phoenix

Initialize/start the authenticated Phoenix instance as documented in
[`../operations/obs02-local-phoenix.md`](../operations/obs02-local-phoenix.md).
Create the Phoenix ingestion key using the owner-only helper (never commit it).
Run the **native** Go API from the repository root:

```sh
# Metadata-only (default):
python3 scripts/observability/run_native_with_phoenix.py

# Explicit full provider wire capture on loopback with fictional test data:
python3 scripts/observability/run_native_with_phoenix.py --content
```

Or directly configure the native Go process, with the ingestion key loaded
privately from an untracked environment/secret file:

```dotenv
APP_ENV=local
HTTP_ADDR=127.0.0.1:8080
THEIA_OTEL_ENABLED=true
THEIA_OTEL_CAPTURE_CONTENT=true
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://127.0.0.1:6006/v1/traces
# THEIA_OTEL_CONTENT_API_KEY=<private Phoenix key, NOT checked in>
```

The two `THEIA_OTEL_*` booleans control whether OTel and content capture are
on. The existing endpoint and private Phoenix key are still necessary to
transmit OTLP into authenticated Phoenix. This is **not** a public/staging/
production configuration. Avoid tests using real or personally identifiable
content, even on your laptop.

## Scope boundary

OBS-04A unlocks existing **physical provider HTTP** request/response capture
for structured, embedding and streaming calls. It does **not** yet export the
complete Local Inspector RAG candidate graph, Brain/Validator inputs, prompt
composition, or custom HTTP request metadata. Those are **OBS-04B/C/D**.
Local Inspector remains separate and its offline schema/fixtures are unchanged.
No new HTTP routes, database changes, RBAC shortcuts, frontend changes or
network model calls are introduced.

## Acceptance checks

1. Go config tests: capture enabled on private loopback with BFF and Inspector
   off; fails with missing/invalid key, public listener, nonlocal environment
   or deprecated Dev Session.
2. Go telemetry tests: SDK receives a >16KiB synthetic request attribute in
   full form and records events; metadata mode has zero wire bodies; >1MiB
   bodies omit content explicitly. Redaction remains effective.
3. Existing provider stream reader/handler behavior is unchanged.
4. Static structure/contract checks and the original Python smoke suite pass.
5. **Live check still required:** authenticated Phoenix receipt/readback of a
   synthetic Turn, including a second attempt/stream, under Keycloak BFF.
   Compare OTLP body against exactly consumed provider bytes, with no secret.

**Rollback:** `THEIA_OTEL_CAPTURE_CONTENT=false` immediately restores the
metadata-only policy after restart. `THEIA_OTEL_ENABLED=false` disables all
tracing. No SQL migration or data rewrite is required. Previously emitted
trace data remains in Phoenix until separately deleted by its actual
retention/erasure process; toggling the flag does not delete it.
