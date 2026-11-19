**# Inspector execution graph contract**



This is a backend-only, additive extension to \`local_ai_trace.v1\`. It provides

the data contract for a separate execution explorer; it adds no GUI or endpoint.

The inspector remains explicitly enabled and restricted to local loopback

execution. Production traces, provider requests, call counts, safety decisions,

state changes and database contracts retain their existing behavior.



**## Audit and reuse**



The existing inspector already records the operation hierarchy (\`parent_id\` and

\`execution\` edges), timings/statuses, complete typed inputs/outputs, memory IDs

and epistemic status, bundle references, loaded and rendered prompts, retrieval

candidates/scores/selections, protocol definitions/transitions, Brain proposals

and policy-gated decisions, rejected drafts, validation reports, regeneration

notes, finalization and exact transport requests/responses. All remain the

authoritative payloads; this extension does not create another payload store.



The missing pieces were explicit relationships at actual transfer sites, a

structured index into rendered prompts, and full-turn fixtures that execute the

real reasoning/generation/retrieval/protocol implementations together. Existing

conversation examples substitute Brain/Generator ports; they remain as valid

older v1 compatibility fixtures and are not relabelled as full implementations.



**## Hierarchy and data relationships**



Recursively group existing nodes by \`parent_id\`. Node IDs are local to a trace;

the global identity is \`(trace_id, node.id)\`. Display their actual \`operation\`

names. Every non-root node has exactly one matching execution edge. These are

inclusive operations, so durations must not be summed across parents/children

or parallel siblings. An absent operation did not execute; a \`skipped\` node

means the code reached a branch that explicitly bypassed it.



Keep \`data\` edges separate from the execution tree. Their existing \`from_path\`

and \`to_path\` are RFC 6901 JSON Pointers relative to the referenced **\*\*node\*\***,

including the \`/input\`, \`/output\` or \`/artifacts/N/value\` prefix. Two evidence

types are now available:



\| Evidence | Interpretation |

\| --- | --- |

\| \`exact_json_value_match\` | Equal captured values; useful provenance evidence, not proof that one operation consumed the other. A whole-container match covers its descendants without redundant edges. |

\| \`instrumented_binding\` | The call site explicitly observed the transfer/transformation. Its \`relation\` is \`copy\`, \`transform\`, or \`regeneration\`. |



\`copy\` bindings require equal JSON values before export. \`transform\` means the

destination was derived from the source; it does not claim byte equality or an

invertible transformation. For example, original message to normalized ingress,

assembled context to updated routing/Brain context, serialized model JSON to a

decoded Brain proposal, decision to protocol/generation plan, and generated text

to final validation. \`regeneration\` connects the rejected validation report (or

failed generation result) to the next attempt's correction note. It never

forwards the rejected draft into the next prompt.



Other explicit bindings connect renderer output to each actual provider attempt,

resolved protocol specification to generation, accepted text to the prepared

turn, and preparation/admission/finalizer/result snapshots. Schema repairs get

copy edges only for the original prompt values they actually retain. Changed

or missing values are never asserted to be copies. Substituted uninstrumented

ports get no invented producer reference.



Bindings are scoped to one invocation and cannot connect unrelated turns.

Output references registered before a method returns are withheld until their

endpoints exist. A data edge can lead from a child back to its ancestor's output;

this is a returned value, not a cycle in execution nesting. Regeneration is a

separate executed attempt, not a rewrite of the first attempt's node.



**## Structured prompt composition without duplicated text**



Brain and Generator rendering nodes now contain an artifact whose \`kind\` and

\`name\` are \`prompt_composition\`. Its \`value\` is an ordered array of prompt parts:



\`\`\`json

{

&#x20; "name": "dynamic_context",

&#x20; "role": "user",

&#x20; "rendered": {"node_id": "n000020", "path": "/artifacts/8/value/dynamic"},

&#x20; "components": [

&#x20;   {

&#x20;     "name": "state",

&#x20;     "start_byte": 0,

&#x20;     "end_byte": 120,

&#x20;     "sources": [{"node_id": "n000020", "path": "/input/in/WCtx"}]

&#x20;   }

&#x20; ]

}

\`\`\`



This is an illustrative shape, not a fixture with resolvable IDs/offsets. Use the

generated full-turn examples for executable data. The JSON Schema specifies the

shape under \`$defs/prompt_part\`, \`$defs/prompt_component\` and \`$defs/reference\`.



Resolve \`rendered\` to the already captured JSON **\*\*string\*\***, then slice its UTF-8

bytes using the half-open \`[start_byte, end_byte)\` range. Offsets are not indexes

into escaped JSON bytes, JavaScript UTF-16 code units, or Unicode code points.

For example, browser code can encode the string with \`TextEncoder\`, slice the

bytes, then decode with \`TextDecoder\`. Do not fetch a second text payload.



Components are in document/start order. An optional integer \`parent\` indexes an

earlier component **\*\*within that part\*\***. These recursively describe actual XML

sections, including nested session/thread/state, recent messages, individual

memories and the current message. Empty sections remain visible. Plain text or

non-XML blocks use a single component spanning the whole string; indexing failure

never changes or rejects the model request. The renderer indexes its final

post-budget text, so trimmed history/memories are not presented as prompt content.



\`sources\` points to existing captured inputs or artifacts, such as a particular

memory/history item, preferences, safety signal, plan, protocol spec, or loaded

instruction block. These are renderer-observed derivation sources; they need

not equal the escaped/serialized text. An empty list means no finer captured

source mapping is claimed. For knowledge, the optional part-level \`source\` is

the actual logical bundle artifact path (\`skills/reflect_emotion.yaml\`, for

example), or an explicitly named code-derived source. Resolve bundle identity

from existing node inputs/ancestor bundle artifacts. Paths are provenance labels,

not URLs, filesystem-reading instructions, or proof of clinical review.



Brain's existing \`context_after_trimming\`, Generator render input/output,

retrieval selections and source snapshots let the viewer explain omitted and

transformed content. Knowledge already visible in system blocks/protocol

artifacts stays there. Component nesting and source references add navigation,

not another copy of the messages or knowledge. Registered credential omission

re-indexes the exported strings so byte ranges remain valid after redaction.



Stream-guard checks and buffer-reset artifacts are now attached to the actual

\`generator.Generate\` attempt, so the rejected attempt and its successor are

unambiguous. Existing streamed chunks, draft artifacts and Validator reports

remain authoritative. Provider totals still count actual attempts once.



**## Compatibility and partial traces**



The schema name remains \`local_ai_trace.v1\`: \`relation\` is optional and existing

artifact payloads are unchanged. The updated schema accepts all four original

saved examples. Older traces have no prompt-composition artifact or explicit

bindings; a viewer should fall back to raw input/output/artifact inspection and

the existing execution tree/value matches. A reader that hardcodes the old

evidence enum must add \`instrumented_binding\`; readers must not assume all data

edges prove causality. No migration or re-export of old traces is required.



\`trace.ValidateJSON\` validates the schema plus execution-parent consistency,

resolvable data/source pointers, copy equality, nested component indexes and

UTF-8 range boundaries. Running checkpoints can lack a future output; no data

edge is published to that unresolved output. Prompt composition is attached

after rendering, without changing the existing atomic checkpoint policy. The

last checkpoint after an abrupt crash may precede that metadata attachment.

Interpret \`complete\`, node status and optional end timestamps as before.



**## Full-turn examples and reproduction**



\| Fixture | Executed work | Provider calls in traced turn |

\| --- | --- | --- |

\| [\`full_turn_protocol.json\`]\(inspector-examples/full_turn_protocol.json) | Real Safety, Retrieval/ranking, Assembler, Router, Brain/policy gate, Protocol shortlist/consented resolution, Generator/render/stream, Validator, state validation and finalization | 2 |

\| [\`full_turn_regeneration.json\`]\(inspector-examples/full_turn_regeneration.json) | The same pipeline, with the first draft rejected by real final validation and a second Generator/Validator attempt using the actual correction note | 3 |



Both execute \`ProcessTurn\` with scripted external providers and owner-scoped

in-memory persistence/retrieval storage. The actual services are not substituted

with Brain/Generator/Validator/Protocol stubs. They use the repository's pinned

synthetic/unreviewed knowledge and fictional conversation/memory content; no

external model is called. A preceding untraced turn creates real recent-history

records through the same runtime. Storage is synthetic, so these examples do not

claim live SQL persistence or capture at an HTTP provider transport that did not

execute. Existing \`exact_wire.json\` remains the real transport fixture.



Reproduce only these new fixtures from the backend directory:



\`\`\`sh

THEIA_UPDATE_INSPECTOR_EXAMPLES=true go -C apps/api test \\

&#x20; ./internal/modules/ai/application/orchestrator \\

&#x20; -run '^TestInspectorGraphFullPipeline$' -count=1

\`\`\`



Tests verify actual stage execution, prompt/source references, nested byte

ranges with Persian/Unicode and escaped text, credential omission, deferred

bindings, false-copy rejection, cross-turn isolation, hierarchy consistency,

container-edge pruning, disabled zero-allocation helpers, provider counts,

accepted output and both regeneration attempts. All checked-in inspector

examples undergo schema and graph validation in the trace package tests.



**## Verification — 2026-11-19**



Verified on Linux amd64 with Go 1.25.4 and \`GOCACHE=/tmp/theia-go-cache\`:



\- Full backend \`go -C apps/api test ./...\`: passed.

\- AI Core \`go -C apps/api test -race ./internal/modules/ai/...\`: passed.

\- All six saved v1 examples, including both new full-turn files: schema and graph

&#x20; validation passed, with no trace diagnostics in the new examples.

\- Existing live SQL Server \`TestInspectorSQLFinalizationAndRollback\`: passed

&#x20; against its disposable-database harness. The runner mapped the saved loopback

&#x20; split settings into \`DATABASE_URL\` in process memory without displaying or

&#x20; saving credentials. Both commit and rollback behavior remained correct.

\- \`scripts/check_structure.py\`, \`scripts/check_contract.py\` and

&#x20; \`git diff --check\`: passed.



The full-turn protocol file contains 40 executed nodes and two provider calls;

the regeneration file contains 44 nodes and three provider calls. Both include

actual \`thought_record\` resolution to its \`situation\` step. UUIDs, wall times and

parallel start order can vary when regenerating. No external-provider call,

database-contract change, dependency or GUI was introduced. The deliverable patch

excludes earlier env/Makefile and frontend work.
