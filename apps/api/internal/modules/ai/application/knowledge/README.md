# Knowledge catalog and JEV

`Load` validates pinned `cards.yaml`, full-content references and coverage of existing
skills/protocols/formulation frameworks. `Select` makes one bounded structured JEV
call with Working Context and card descriptions, then resolves at most four approved
IDs to complete content. It has no database writer or personal-memory search.

Core Policies never participate in selection. Invalid/provider/timeout/budget failures
use only valid short-lived carry-over or an active exercise's definition. Expired
recommendations do not renew themselves during outages. Model configuration and
pricing belong to the operator's v4 runtime file.

JEV uses OpenCode's native System One transport, not Chat Completions:
`POST https://opencode.ai/inference/systemone/v1/systemone`, model
`jev-1.13-free`, Bearer `OPENCODE_API_KEY`, billing source `free`. Only JEV uses
this provider. Unified and background components retain their existing settings.

One request supplies shared selection guidance, verified Working Context,
carry-over and the complete Knowledge Card catalog once in `state` as named JSON
fields. Carry-over is omitted from the serialized Working Context to avoid a
duplicate copy. Each short `noul` resource question references its card by ID;
the separate `topic_continues` question remains. No question repeats the shared
prompt or embeds the card again. Full resource content and Core Policies remain outside JEV.
The input budget includes all questions, and the existing timeout/call limit is
unchanged. No temperature, completion-token limit, messages or output schema is
sent to Jev. `max_tokens` in the runtime remains a legacy envelope ceiling only.

A Noul answer is the probability of yes. Values above 0.5 are affirmative; 0.5
is uncertainty. Go sorts affirmative cards by probability (ID breaks ties), takes
at most four and emits the existing `jev_selection.v1`. Existing catalog checks,
content budget, active-exercise inclusion and carry-over fallback still apply.
Missing/extra/mistyped/out-of-range answers reject the whole decision; reported
usage is retained before downstream validation rejects it. There is no schema
repair, additional JEV call or paid/GPT fallback.

The Inspector captures native request bytes, native answers, normalized selection,
reported token usage, timing and explicit zero cost. The `local_ai_trace.v1` prompt-part role enum now also accepts `state` and
`instructions`; historical generated-JSON JEV examples remain readable.

Local setup: put `OPENCODE_API_KEY` in ignored `.env` and run through
`scripts/run_local.py` / `make api`. An exported variable takes precedence. An
explicit synthetic live check is available from the backend root:

```sh
THEIA_JEV_LIVE_TEST=true python3 scripts/run_local.py go test ./internal/app/composition -run '^TestJEVSystemOneLive$' -count=1 -v
```

A successful live check requires the gateway to accept the key and network origin.
The live test is skipped during normal tests and never stores the key in a trace.
Protocol references: [OpenCode JEV](https://opencode.ai/docs/zen/#jev),
[TypeSafe System One](https://docs.typesafe.ai/api/system-one),
[shared state and focused questions](https://docs.typesafe.ai/concepts/state).
