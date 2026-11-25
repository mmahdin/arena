# OBS-05B — Manual Phoenix vs Langfuse UI scorecard

**Fill out after running ONE identical, synthetic Theia Turn in both UIs.**
Do not invent scores from vendor marketing or a synthetic schema fixture.
No screenshots containing real conversations or private customer data.

| Review context | Value |
| --- | --- |
| Synthetic W3C TraceID | `...` |
| Theia backend commit / patch set | `...` |
| Phoenix version / project | `...` |
| Langfuse version / project | `...` |
| Full capture? (`true` only on local synthetic) | `...` |
| Collector ingest & both readbacks | `...` |
| Automated OBS-05B parity result & issue codes | `...` |

Rate each criterion from **1 (missing/unusable) to 5 (clear and actionable)**.
Use `N/A` only if a scenario was genuinely not exercised, and do not include
N/A in the weighted total. Include *short evidence notes* describing an actual
synthetic UI interaction and source TraceID, never raw messages.

| Criterion | Weight | Phoenix 1–5 | Langfuse 1–5 | Evidence / gaps |
| --- | ---: | ---: | ---: | --- |
| RAG vector/BM25/RRF rank, selected vs omitted | 25% | — | — | — |
| Agent hierarchy and stage I/O | 20% | — | — | — |
| Complete LLM prompts/responses, streaming, retries | 20% | — | — | — |
| Trace search/navigation and debugging time | 10% | — | — | — |
| Correct usage/cost, HTTP errors, latency | 10% | — | — | — |
| Annotations, datasets, quality evaluation | 10% | — | — | — |
| Setup, maintenance, data access controls | 5% | — | — | — |

**Scenarios exercised:** successful RAG [ ]; irrelevant/empty RAG [ ]; context
budget [ ]; retry [ ]; broken stream [ ]; validator reject [ ]; async worker
[ ]; usage/cost unknown [ ].

**Risk and decision log:**

- Missing attributes/events in Phoenix: `...`
- Missing attributes/events in Langfuse: `...`
- Differences in render vs API readback: `...`
- Sensitive data leakage observed? `...` (if yes, stop and fix)
- Operations/storage/RBAC constraints: `...`
- Recommended outcome: `Phoenix / Langfuse / keep both / insufficient evidence`
- Justification and next instrumented experiment: `...`

**Do not remove Phoenix until explicit approval and a successful OBS-05B review.**
