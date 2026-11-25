# OBS-01B — Local synthetic full-content tracing (Go source implementation)

**Status:** OBS-04A enables opt-in content capture for the normal loopback Go
API using Keycloak/BFF (no Dev Session). This document records the original
OBS-01B behavior; the superseding configuration and increased body cap are
in [OBS-04A](OBS04A_FULL_TRACING_FOUNDATION.md).

## What is captured

The existing `llmhttp.Provider.send` serializes the final provider request *after*
model-specific options, JSON schema injection and repair-message construction.
OBS-01B captures these exact JSON bytes in a dedicated private OTel attribute
on the active provider attempt/stage span. It captures only response bytes
naturally read by existing `readJSON`, bounded provider rejection reads, or the
SSE scanner. `contentBody` now uses a max-1MiB buffer (OBS-04A) and incremental SHA-256;
it never requests an extra body read and never changes a provider response.

The private trace fields use `theia.dev.wire.request.*` and
`theia.dev.wire.response.*`:

- `body`: original captured text if structurally valid and safe, or a
  credential-redacted serialization with `redacted=true`. This is **actual
  provider wire evidence**, not a synthesized prompt template.
- `bytes` and `sha256`: length/hash of the original bytes actually sent or read.
- `stream`: distinguishes SSE response bodies from JSON.
- `complete`: JSON body reached HTTP EOF, or SSE produced validated `[DONE]`;
  it is **not** proof the model output met Theia policy/semantic validation.
- `omitted` and `omission_reason`: true when content was over 1MiB, malformed,
  unsafe/unparseable, or expanded beyond the cap after redaction. **No body is
  emitted** for such a capture; never show omission as a complete transcript.

Streaming capture is flushed when the existing `Recv()` reaches `[DONE]` or
fails, *before* the attempt's OTel span ends. Early Close and interrupted SSE
are marked incomplete; a read timeout does not cause a follow-up drain.
Provider errors retain only machine failure codes in SQL/slog. Their naturally
consumed JSON error body, when valid and bounded, is present **only** in the
private trace. The original local Inspector is untouched and retains its own
private JSON format, artifact graph and UUID distinct from W3C TraceID.

**Scope/limits:** Capture is implemented at the current OpenAI-compatible
HTTP adapter for structured, streaming and embedding calls. Non-HTTP/mock
providers have no actual provider wire payload to capture. A response larger
than 1MiB is **omitted**, not reconstructed; the local Inspector remains the
higher-fidelity local debugging source. A streaming capture includes raw SSE
frames consumed, not a newly generated final assistant message.

## Isolated runtime guards

An isolated synthetic harness can construct the telemetry runtime directly.
It must use fictional data and a loopback-only runtime. The Inspector switch
is independent and no longer required for wire capture. Content mode also requires an HTTP OTLP
receiver on numeric loopback port 6006 at `/v1/traces` and a private ingestion
key. The exporter sends the key in an Authorization header and rejects
redirects. Its HTTP transport uses a direct connection.

The normal API still has no Dev Session identity path. OBS-04A permits
local-only, private, explicit content capture; only synthetic test accounts
and fictional messages are permitted. Never enable on shared or public hosts.

## Scrubbing and limitations

Transport headers, Authorization bearer headers, cookies and URLs are never
copied into the OTel payload. The private capture scrubber removes nested
credential-named JSON fields, known provider/ingestion keys echoed in text,
Bearer tokens and common secret patterns. If a body is not valid JSON/SSE,
it is **omitted**. This redaction is *defense in depth*, not an exhaustive
secret/PII detector. This mode is safe only for wholly **synthetic** data.
The original SHA-256 may be useful for debugging but does not attest that the
provider received or acknowledged the payload.

Do not export raw body text to SQL Admin read models, routine `slog`, general
metrics, Sentry, product events, Prometheus, Tempo or Loki. For later shared
or production workloads, re-evaluate privacy policy, retention, key rotation,
authorization and controlled access independently.

## Validation

From an environment with the Go modules installed:

```sh
cd apps/api
go mod tidy
go build ./...
go test ./...
go test -race ./internal/platform/telemetry ./internal/modules/ai/adapters/llmhttp ./internal/modules/ai/llm
```

Repository checks:

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -q
```

After OBS-02, run local synthetic scenarios: structured JSON-mode/schema
repair, embedding, streaming `[DONE]` and interrupted streams, provider 429
with a redacted body, and two-attempt retry. Confirm the same provider-call
count, SQL attempt statuses, request/response content boundaries, absent
secret markers, correct OTel attempt IDs, and no changes to the local offline
Inspector JSON fixtures. Test unauthenticated OTLP ingestion rejection and
Phoenix UI access denial. Do not report these as passing without actual tests.

**Suggested commit:** `feat(observability): add gated synthetic AI payload tracing`
