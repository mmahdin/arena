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
`apps/api/.local/ai-inspector/` when running from `apps/api`. The Explorer
never reads that directory automatically: you explicitly select the file.
For safe examples, use the repository's synthetic JSON files in
`docs/ai-core/inspector-examples/`, especially `full_turn_protocol.json` and
`full_turn_regeneration.json`.

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
