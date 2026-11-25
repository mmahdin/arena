# Theia AI Execution Explorer

A standalone, browser-only visual reader for Theia Local AI Core Inspector
`local_ai_trace.v1` JSON files. The source is `index.html`; it needs no build,
server, authentication, database connection, or running Go backend.

## Open a trace

1. Open `index.html` in a browser.
2. Choose **Open traces...** or drag a local `.json` file onto the page.
3. Expand/collapse execution nodes, select nodes to inspect captured content,
   and inspect actual data-flow bindings and rendered prompt components.

To generate a trace separately, enable the local Inspector as described in
`docs/ai-core/local-inspector.md`. Its output directory is normally
`apps/api/.local/ai-inspector/` when running from `apps/api`. A single trace contains the original turn and all subsequently observed,
related memory/reflection worker attempts as child nodes. The Summary tab
also lists each background job ID and attempt, linking to its execution node. The Explorer
never reads that directory automatically: you explicitly select (or re-import)
the updated file after background jobs finish.
For safe examples, use the repository's synthetic JSON files in
`docs/ai-core/inspector-examples/`, especially `jev_unified_turn.json` for the active JEV/Unified runtime.
`unified_memory_turn.json` retains the previous unified contract for compatibility.
`full_turn_protocol.json` and `full_turn_regeneration.json` retain historical
Brain/Generator behavior and remain valid imports. The generic node/payload renderer
handles the new operation names without a second schema or embedded fixture set.

## Trace format and maintenance

- **Single source of truth:** `apps/api/internal/modules/ai/trace/local_ai_trace.v1.schema.json`.
- **Meaning of edges and prompt references:** `docs/ai-core/inspector-graph.md`.
- **Recording behavior, local-only guard and data precautions:** `docs/ai-core/local-inspector.md`.
- **Agent update instructions:** `apps/api/internal/modules/ai/trace/AGENTS.md`.

The Explorer does not fetch data, subscribe to backend changes, or refresh
itself when a trace file changes. Reopen/reimport a changed file as needed.
Edits to tracing fields or payloads are not automatically reflected in the HTML;
code agents must assess and update the viewer and corresponding examples when
changes affect compatibility, then verify actual browser interactions.

Captured traces can contain complete user text, memory and prompts. Keep real
private traces out of Git and do not use them as committed test fixtures.

Phase 2 example: `docs/ai-core/inspector-examples/background_intelligence.json`
records a real scripted SQL batch with four primary messages and one analysis
inference. It is a standalone `kind: job` file; user-level jobs may omit turn ID.
`node scripts/tests/ai_explorer_unified_browser.mjs` exercises its prepared input,
output, prompt sources, bindings and summary together with historical examples.

Unified checkpoints show separate response and background status chips. The
summary lists the durable queue, including jobs waiting for their first claim
or retry deadline, alongside navigable execution attempts. Re-import to refresh;
there is no backend connection or watcher. Older v1 files remain supported.

`one_turn_lifecycle.json` is a small hand-authored synthetic checkpoint showing
an already finished response, one skipped embedding and one failed embedding
waiting to retry. It exercises queue/attempt status display; the SQL regression
`TestInspectorOneTurnOneJSONLifecycleSQL` verifies the actual emitted lifecycle.

Shared Embedding Service examples: `shared_embedding_origin.json` shows one
physical computation, a caller cancellation, and the Memory Worker's in-flight
join under the same Turn. `shared_embedding_reuse.json` shows another Turn's
cache hit with origin correlation and zero physical provider calls. Select shared
nodes for vector identity, wait/execution timing and computation metadata. Source
attempt counts describe the original computation and are not extra usage.

`semantic_retrieval.json` shows the current temporary BM25-disabled policy: a
successful current embedding, failed contextual embedding, vector-only query
strategies and no BM25 rank artifacts. Older examples can still display their
historical BM25 execution; the viewer never invents absent rank branches.

`jev_systemone.json` exercises native JEV decisions through the shared HTTP/client
aggregation path. The LLM tab shows System One state, typed questions, native
answers, the compatible normalized selection and free billing. Wire bytes remain
available on the HTTP node. Historical GPT-based JEV traces still import. No new
trace schema version or online viewer dependency is needed. The additive prompt
roles `state` and `instructions` describe native JEV composition accurately.
