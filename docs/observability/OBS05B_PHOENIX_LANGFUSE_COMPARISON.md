# OBS-05B — Phoenix vs Langfuse: same-source AI trace comparison

**Scope:** an auditable, local/synthetic-only comparison workflow for the **same**
W3C TraceID, emitted once by Theia (or once as a fixed synthetic fixture) and
forwarded by the OBS-05A private Collector to **both** Phoenix and Langfuse v4.
This step **does not retire Phoenix, pick a winner, modify the AI pipeline, run
an additional model call, expose private data, or change the frontend/API**.

## 1. Preconditions

- OBS-04A/B/C/D and OBS-05A are applied. Keep the existing Phoenix private.
- Phoenix is authenticated on `127.0.0.1:6006` and Collector on `127.0.0.1:4318`.
- `.env.observability` contains a Phoenix system API key; `.env.dual-otlp` has
  the Collector bearer key and the Langfuse v4 OTLP base and project credentials.
- Langfuse v4 is already running; its credentials have **trace read** permission.
- Everything inspected is synthetic. The tool fetches I/O fields **into memory**
  to test equivalence; run only in a secure developer workspace.
- The script uses only Python standard-library modules and the existing OBS-05A
  helpers. It does not install new packages, launch Docker or invoke an LLM.

## 2. Run the independent synthetic fixture first

Start the OBS-05A services as documented in `OBS05A_DUAL_OTLP_EXPORT.md`, then:

```sh
python3 scripts/observability/smoke_dual_otlp.py --preflight
python3 scripts/observability/compare_dual_traces.py \
  --fixture --synthetic-only --output /tmp/theia-obs05b-fixture.json
```

The fixture is exactly **one OTLP request containing 10 spans**. It uses the
existing Phoenix project **`theia-obs05a-dual`**, because OBS-05A's Collector
sets a fixed `x-project-name` header. It is distinguished by a new W3C
TraceID and the `theia-obs05b-fixture` service name, not a second project.


| Stage | OTLP evidence |
| --- | --- |
| Turn | AGENT root with synthetic input/output |
| Retrieval | RETRIEVER, selected document metadata, vector/BM25 events |
| Fusion | RERANKER, RRF event |
| Brain | AGENT decision with synthetic input/output |
| LLM attempt 1 | LLM, synthetic failure, first HTTP transport child (429) |
| LLM attempt 2 | LLM, synthetic input/output, known 12/7 tokens, HTTP child (200) |
| Validator | GUARDRAIL with synthetic decision |
| Finalizer | CHAIN with synthetic final output |

It is a **schema/ingestion test** of equivalent observations, **not a Theia
runtime Turn**, model-quality measurement, or evaluation of actual retrieval.
All example strings start with `SYNTHETIC_`; no real records are loaded.

For an **independently hosted local Docker Langfuse** with Collector export to
`http://langfuse-web:3000/api/public/otel`, the host-side read API cannot
resolve the container hostname. Bind a **private loopback-only** port to its
web container and add:

```sh
python3 scripts/observability/compare_dual_traces.py \
  --fixture --synthetic-only \
  --langfuse-read-origin http://127.0.0.1:3000 \
  --output /tmp/theia-obs05b-fixture.json
```

This override is refused for remote/cloud endpoints and cannot target arbitrary
hosts or paths. Never publish the Langfuse web port publicly for this test.

The comparison polls both services, follows **all** cursors, refuses oversized
responses and writes a `0600` JSON report containing **only TraceID, span IDs,
counts, semantic labels, equality flags and issue codes**. It refuses to
overwrite an existing file. **Raw prompts, retrieved content, secret values,
HTTP URLs, raw errors and response bodies are not written to the report.**
A nonzero exit status or `FAIL` means semantic/completeness parity was **not
proven**; it does not automatically mean either exporter stopped working.

## 3. Compare one actual synthetic Theia Turn

Run Theia with OBS-05A full synthetic tracing and trigger a **single** test
Turn in the authenticated **User** application:

```sh
python3 scripts/observability/run_native_with_dual_otlp.py --content
```

Read its 32-character W3C TraceID from the OBS-03 Staff
`GET /api/v1/admin/turns/{turnId}/trace-references` API, or from your local
trace diagnostics. The default OBS-05A Collector uses the Phoenix project **`theia-obs05a-dual`**
for all forwarded traces. If your Collector configuration is customized,
confirm the project in the authenticated Phoenix UI instead of guessing.

```sh
python3 scripts/observability/compare_dual_traces.py \
  --inspect YOUR_32_CHAR_HEX_TRACE_ID --project YOUR_PHOENIX_PROJECT \
  --synthetic-only --lookback-hours 24 \
  --output /tmp/theia-obs05b-turn.json
```

`--inspect` **never ingests or reruns** a Turn. It checks that the same trace
was materialized in both tools, that the span set/parents/names are aligned,
that input/output and selected RAG document fields are equal where supplied,
and that known token counts match. Differences in Langfuse's v4 metadata
truncation will be reported rather than silently accepted. Missing/omitted data is not silently assumed to be zero.
The query window is bounded (`--lookback-hours` from 1 to 168); for older
traces run comparison promptly or extend via a future, reviewed interface.
If you use Docker Langfuse, add the same private `--langfuse-read-origin` flag.

**Important:** Some model/provider metadata, source RAG events, or rich input
structures can be represented differently. The report captures observable
mismatches; a `PASS` is **API evidence parity only**, not proof that the two
UIs display the same level of detail or preserve every event. Phoenix exposes
`theia.rag.*` events via its span API; Langfuse v4 Observations API v2 does not
provide a direct raw-OTel-event field for equivalent querying. Explicitly
check the RAG timeline and rank detail in each UI during manual review.

## 4. Compare the actual UI experience (required for platform decision)

Use the same **synthetic** TraceID in Phoenix and Langfuse. Fill out the
[OBS-05B manual scorecard](OBS05B_MANUAL_SCORECARD.md) with observed facts,
not assumed features. Include at least these test scenarios, when available
in the existing synthetic backend fixture or synthetic User app:

1. Successful Turn with meaningful RAG results and selected context.
2. Irrelevant/empty retrieval and a case with context-budget exclusion.
3. Real provider retry/429 and an incomplete SSE stream.
4. Validator rejection/regeneration and error handling.
5. Background worker attempt linked by the corresponding Turn/Job IDs.
6. Unknown versus known token usage, estimated cost and HTTP transport spans.

The standalone 10-span synthetic fixture covers **schema/retry structure**,
not all six runtime scenarios. Synthetic backend fixtures must be run
independently to cover each relevant workflow; do not claim unexecuted paths
were observed. Compare what the UI shows (redacted prompt/response, selected
RAG documents, RRF candidates, agent outputs, HTTP status, events) with the
Local Inspector and the OBS-05B report, and record omissions accurately.

## 5. Acceptance / security / rollback

Acceptance requires: both services queryable with the same TraceID; **all**
fixture span IDs and parents present; meaningful semantic kinds; no false
second model call from HTTP child; input/output and token coverage differences
explained; no secret value in the report; the same real synthetic Turn inspected
in both UIs; manual UX scorecard completed before choosing a platform.

- A nonempty, partially ingested trace is **not** a completed comparison.
- Project names may differ from W3C trace IDs. Supply an authorized project,
  never search all user traces for a guess.
- Observability exporters are best-effort; data may be delayed or dropped.
  If a destination is unavailable, keep Phoenix/direct logging and retry the
  **comparison**, never the actual user/model Turn.
- Reports contain no source message text, but still have trace identifiers.
  Keep reports private and do not use real-user content for this exercise.
- No DB migration, OpenAPI alteration, production RBAC change, or model-prompt
  modifications are part of OBS-05B.
- Roll back by simply stopping using the comparison script; OBS-05A dual export
  and the Phoenix-only runner remain available.

## 6. Verify

```sh
python3 -m unittest discover -s scripts/tests
python3 scripts/check_structure.py
python3 scripts/check_contract.py
```

**Live run remains a separate manual acceptance check:** the presence of code
and passing offline unit tests does not establish that either destination is
running, has data, or displays the attributes correctly.

Official interface references:
- Phoenix project spans: https://arize-ai.github.io/phoenix/functions/_arizeai_phoenix-client.spans_getSpans.getSpans.html
- Langfuse v4 Observations API v2: https://langfuse.com/docs/api-and-data-platform/features/public-api
- Langfuse v4 migration: https://langfuse.com/faq/all/deprecated-api-migration
