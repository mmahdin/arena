# WP-04 — LLM Client Layer implementation and verification

Date: 2026-11-16. Source: provided backend ZIP (WP-01, WP-02 and soft-delete correction already present). Authority: `docs/ai-core/Theia_AI_Core_Contract_Pack_v1.2.md` and the final architecture document. Module: `apps/api/internal/modules/ai/llm`, with new adapters `adapters/llmhttp` and `adapters/fake`.

## Included in this patch

- Provider-neutral structured client implementing the existing WP-01 `ports.Client` contract; additive embedding companion port; schema validation with one JSON-mode repair.
- Configurable per-tier model chains and provider capabilities, per-attempt transport/rate-limit/server retries, jittered backoff, provider/model circuit breakers, context deadlines, first-token timeouts and stream failure boundaries.
- HTTPS OpenAI-compatible JSON Schema, JSON-object, SSE and embedding adapter independent of the one-shot prototype. Typed, sanitized transport failure kinds, bounded parser inputs and disabled redirects.
- Explicit micro-USD pricing by model and cached/uncached tokens, per-user UTC-day ledger and quota interface, per-attempt privacy-safe metrics, and a deterministic fake provider for tests.
- Strict raw-schema compilation in the existing WP-01 schema package. No changes to applied migrations, AI-related SQL stores, deletion, user privacy/consent, active HTTP routes, or historical conversation payloads.

## Validation performed

This environment has Go 1.23.2, Linux amd64. The uploaded ZIP did **not** include dependency module sources and the runtime cannot resolve `proxy.golang.org`. Consequently, clean compilation or the full suite **against the pinned real dependencies has not been verified**. The following scoped checks were performed using an external, temporary `-modfile` with test-only local stand-ins for `github.com/google/uuid` and `github.com/santhosh-tekuri/jsonschema/v6`. These stand-ins are **not** in this patch; they provide type-checking and a basic JSON-schema approximation, **not** a substitute for real-schema integration testing.

| Check | Result |
| --- | --- |
| `gofmt` on all WP-04 Go files | PASS |
| `go test` new LLM client, fake and HTTP adapter packages with temporary stand-ins | PASS (limited, non-authoritative) |
| `go test -race` same packages with temporary stand-ins | PASS (limited, non-authoritative) |
| `go vet` same packages with temporary stand-ins | PASS (limited, non-authoritative) |
| `go build` new packages with temporary stand-ins | PASS (limited, non-authoritative) |
| `python3 scripts/check_contract.py` | PASS: 30 existing Gin/OpenAPI pairs |
| `python3 scripts/check_structure.py` | BLOCKED BY UPLOADED BASELINE: existing `.agents/skills/wellbeing-*/` paths absent from archive |
| `go test ./...` and `go build ./...` using real declared dependencies | NOT VERIFIED: dependency download/DNS unavailable |
| Real SQL Server, remote provider or clinical/evaluation tests | NOT RUN: not part of this WP |

## Residual issues / next owners

1. Run real-dependency `go test ./...`, `go test -race` on the new packages, `go vet` and `go build ./...` in a network-enabled CI/checkout with its actual module cache, followed by the usual `scripts/check_structure.py` once the repository's `.agents` files are present. The temporary stand-ins do not prove actual JSON Schema v6 compatibility or the full baseline.
2. WP-03 owns loading/reviewing the pinned `models.yaml` and policy bundle; WP-05 owns wiring this client/ledger into the **new** Orchestrator and replacing the old experimental AI runtime. The existing one-shot gateway remains **transitional only**, not a fallback or dependency of this client. WP-07/11 own post-generation safety validation, buffered release and approved locale/clinical template fallback. WP-09 owns vector retrieval/indexing.
3. Production cost/quotas need current model-specific price configuration, monitoring of providers that omit failed-attempt token usage, and a **durable multi-instance** ledger/reservation mechanism before hard billing guarantees are claimed. Generic compatible transports do not guarantee upstream idempotency; provider-specific deduplication requires a separate review.
4. No WP-05 runtime integration, new endpoint, SQL migration or Soft Delete policy change was made. No real-user mental-health deployment is authorized by these tests.

Apply the Git patch at the **root of the backend checkout** with `git apply --check <file>.patch` then `git apply <file>.patch`. The patch is intended to apply to the exact source snapshot contained in the uploaded ZIP. No changes were made to the architecture documents themselves.
