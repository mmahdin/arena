# OBS-04B — Deep RAG Tracing (Synthetic/Local)

**Owner:** `apps/api/internal/modules/ai/application/retrieval` and
`apps/api/internal/platform/telemetry`.

**Goal:** Inspect why the existing owner-scoped dual-query memory retrieval
returned particular items, with complete ranking data where it fits the
bounded OTel event payload. This is observability only: no new search strategy,
permission bypass, query rewrite, reranker, new API route, DB migration,
external call or change to result selection.

## Enable on an isolated synthetic test stack

First apply **OBS-04A**. Configure the existing local Phoenix receiver and
private ingestion key according to `OBS04A_FULL_TRACING_FOUNDATION.md`:

```sh
python3 scripts/observability/run_native_with_phoenix.py --content
```

The Go API must be bound to loopback, `APP_ENV=local`, and
`THEIA_OTEL_ENABLED=true` plus `THEIA_OTEL_CAPTURE_CONTENT=true` must be
validated at startup. Ordinary metadata tracing does **not** serialize or
export user queries, retrieved memory text or ranking payloads. This feature
is not authorized for real conversations, public deployment or production.

## Traces

Run a synthetic chat turn that exercises authorized historical retrieval,
then inspect its W3C Turn trace in Phoenix. These events are attached to the
**existing** stage spans (rather than creating a parallel trace):

| Stage span | Event name | Structured `theia.dev.rag.json` body |
| --- | --- | --- |
| `theia.ai.application.retrieval.service.Retrieve` | `theia.rag.query` | Current query, optional contextual query, and the observed embedding outcomes |
| `theia.ai.retrieval.repository.Scan` | `theia.rag.authorized_candidates` | Eligible sources, IDs, versions, available text, evidence counts, type, sensitivity, and embedding dimensions |
| `theia.ai.retrieval.rank` | `theia.rag.vector_ranking`, `theia.rag.contextual_vector_ranking` | Positive cosine matches for each active vector branch |
| same | `theia.rag.thread_ranking`, `theia.rag.temporal_ranking` | Thread and date signals when active |
| same | `theia.rag.rrf_ranking` | Fused rank, score, source, ID, and ranks in the contributing branches |
| same | `theia.rag.selection` | Per-considered-item outcome, including duplicate identity, failed hydration, empty text and truncation. Also reports the remaining budget and the count left after reaching a limit |
| `theia.ai.application.retrieval.service.Retrieve` | `theia.rag.selected_context` | Returned items after evidence expansion and the deadline check |

BM25 stays disabled in the active runtime. Candidate events follow the owner and
AI-use checks. Rows eligible for deferred hydration can have empty text in the
scan event, then expose hydrated text in the selected context. The RAG events
report embedding dimensions and availability.

Every event has `theia.dev.rag.omitted` and
`theia.dev.rag.original_bytes`. Oversized (> 1 MiB), unencodable and invalid
payloads are **explicitly omitted**, with an `omission_reason`; long content
is never silently truncated by telemetry. Events containing list counts also
expose their counts as attributes if the JSON was omitted. As in OBS-04A,
credential keys, Bearer tokens and private ingestion credentials are redacted.
Large authorized corpora can exceed the per-event bound: in that case this
stage provides the omission reason, **not** a guarantee of every source's text.

`theia.rag.selected_context` is retrieval output. Inspect the later JEV and
Unified stages and their physical provider requests to verify prompt inclusion.
The offline Inspector retains its Top-5 rank previews, selected-item artifacts,
and `local_ai_trace.v1` schema.

## Acceptance and tests

- Metadata-only mode: zero RAG payloads and no lazy JSON serialization.
- Content mode: authorized source text, query, full rank lists, fused rank
  and selected context are visible in the original Turn trace.
- Per-rank decision explains whether an item was chosen, skipped or not
  considered after a budget/limit threshold; rank and scores match output.
- Denied and cross-user documents cannot enter the eligible candidate list.
- Secret fields and known provider/ingestion credentials do not leave Go.
- Invalid/unbounded payloads show an omission event without exporting content.
- RRF, deduplication, Context budget, DB ownership and retry semantics unchanged.

Local checks:

```sh
cd apps/api
go test ./internal/platform/telemetry ./internal/modules/ai/application/retrieval
```

From the repository root also run `python3 scripts/check_structure.py`,
`python3 scripts/check_contract.py`, and `python3 -m unittest discover -s scripts/tests`.
Live Phoenix receiver/Go BFF/SQL Server and privacy verification are required
before considering a runtime rollout validated. If the Go module cache is
unavailable, report the Go test result as *unverified*, not passed.

**Rollback:** revert OBS-04B Go files (or disable
`THEIA_OTEL_CAPTURE_CONTENT`). There are no API/schema/DB changes or persisted
application state migrations. Existing Phoenix traces are not deleted by a
code rollback.
