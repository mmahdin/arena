# AI Core tracing and offline Explorer maintenance

Scope: `apps/api/internal/modules/ai/trace/` and its `explorer/` subtree.
Follow the repository-level `AGENTS.md` and AI module contracts as well.

`explorer/index.html` is a standalone, offline, user-selected JSON viewer.
It is intentionally NOT connected to a running backend, HTTP endpoint,
watcher, database, or trace directory. Preserve that boundary.

The authoritative trace contract is `local_ai_trace.v1.schema.json` in this
folder, implemented by the Go trace recorder and graph helpers. The official
synthetic fixtures are in `docs/ai-core/inspector-examples/`; interpret them
using `docs/ai-core/inspector-graph.md` and `docs/ai-core/local-inspector.md`.
Do not maintain a copied schema or a second set of embedded trace fixtures
inside the Explorer.

When changing AI Core instrumentation, operation/payload names or structure,
execution/data edges, prompt composition, references, exported fields, or the
trace schema:

1. Assess whether existing Explorer imports, navigation, references, content
   views, and edge semantics still work. Update `explorer/index.html` if needed.
2. Update schema and synthetic example traces when the emitted contract changes.
   Keep compatibility with historical v1 files where practical; use an explicit
   schema version for genuinely incompatible format changes.
3. Preserve the distinction between `execution` hierarchy, observed
   `instrumented_binding` (`copy`/`transform`/`regeneration`) and non-causal
   `exact_json_value_match` evidence. Never invent paths or provenance.
4. Resolve prompt components against their captured rendered-string references.
   `start_byte`/`end_byte` are UTF-8 byte offsets, not JavaScript string indices.
   Older valid traces may lack explicit bindings or `prompt_composition`.
5. Verify Go trace/schema/graph tests and, for changes affecting the Explorer,
   actually load representative examples in a browser. Exercise file importing,
   recursive expansion/collapse, node inspection, prompt text and sources,
   edge selection, zoom/pan, regeneration, skipped/error states and old v1 traces.
   Do not equate a successful page render with working interactions.
6. Do not include private `.local/ai-inspector` traces or credentials in commits
   or fixtures. Use synthetic content only. Full-payload inspector collection
   remains opt-in and local-only.

Keep changes focused. Do not modify AI Core behavior or add runtime LLM calls
just to support the viewer. Document unavailable data rather than fabricating it.
