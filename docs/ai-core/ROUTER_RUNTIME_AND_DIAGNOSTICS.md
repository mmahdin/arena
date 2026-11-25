> **Runtime superseded (2026-11-23):** [Unified memory-centric architecture](unified-memory-centric.md) is the active runtime. Separate Brain/Generator, safety/router/output-classifier calls and regeneration described here are historical. Persisted contracts and historical Inspector examples remain supported.

# Router and diagnostics — local synthetic development change

This implementation deliberately overrides the old deterministic `router.yaml`
routing logic in the running API. `NewAIInternal` always injects a
`ModelRouter`; `Router{}` remains only for isolated legacy tests that explicitly
inject it. The signed Knowledge Bundle's `config/router.yaml` is retained for
historical schema/bundle compatibility. Its old rule IDs **do not run** in the
production composition path and must not be mistaken for an active route map.

The Router uses the separate `llm.components.router` runtime chain. Default
local model is `gpt-4.1-mini`, with a 1.8-second limit, JSON-schema output,
100 completion tokens, **one provider attempt with no retry/repair/fallback**.
The user message, language, origin, safety level, current persisted risk,
follow-up flag, pending-question/protocol booleans and up to three bounded
recent messages form its complete input. It returns `fast`, `normal`, `crisis`
or `proactive` plus a reason enum. The classifier neither performs retrieval
nor mutates clinical state.

**Security invariants remain code-enforced, not model-discretionary:** high or
imminent risk cannot bypass `crisis`, elevated risk cannot choose `fast`, and
user-initiated turns cannot send proactive messages. On classifier failure, the
policy selects a safe buffered `normal` (or `crisis` for high risk) response,
marks the turn degraded and records the model error. This is a minimal safety
boundary, **not** a second heuristic router.

`local_ai_trace.v1` retains the original schema. The opt-in local Inspector
now captures router inputs and model decisions, provider validation reason and
rejected completion text, plus retrieved memory's safe degraded-reason code.
`trace/explorer/index.html` derives a visible Degraded state for successful
nodes whose captured output reports degraded, and offers a Diagnostics tab that
shows provider content as readable text and links to its source operation.
Ordinary logs **never** include prompt text or response bodies; they include
only safe reason codes. Never enable the local full-payload Inspector for real
mental-health content or deploy it on a non-loopback/production interface.

## Acceptance checks

- A greeting without elevated risk is routed by the model, not by a `trivial` pattern.
- The Router has at most one provider attempt (no fallback/repair on bad JSON).
- High risk forces `crisis` even if the model selects `fast`; no response is
  permitted to weaken policy/consent.
- Retrieval search consent denial/pause emits `search_policy_denied` or
  `memory_paused`; transport/timeouts use non-sensitive failure codes.
- Invalid provider completion (`length`, missing `usage`, empty content etc.)
  has a specific machine-readable reason; its raw response is inspectable only
  from an opted-in local trace JSON.
- Use `go test ./...` and a real browser import of `normal.json`,
  `provider_retry_repair_stream.json` and a synthetic rejected-completion
  trace before promotion. Re-run bundle and structure/contract checks.
