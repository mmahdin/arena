> **2026-11-23 active runtime:** See `docs/ai-core/unified-memory-centric.md`. Production uses one unified conversational model with immediate retrieval and independent baseline context. Brain/Generator and classifier runtime descriptions below are historical; Memory Writer/Reflection and persistence remain active.

# WP-06 — Safety (in-process AI Core)

WP-06 implements **engineering contracts and conservative mechanics**, **not a clinically approved safety product**. The `safety` package is independent of the experimental `modules/ai/policy/safety.go`, all WP-08 work, and any production composition entrypoint. There are no migrations and no deletes.

## Public entry points

- `Load(ctx, pinnedWP03Bundle, LoadOptions{})` validates one pinned snapshot of the WP-03 safety lexicons, safety policy, thresholds, crisis resources/templates, generic fallbacks, and L1 prompt. A real call must use a release-verified non-synthetic bundle. `AllowSyntheticForTests` is only for isolated synthetic tests.
- `NewLLMClassifier(WP04Client, WP01Schemas, assets, timeout)` uses `ports.Client.Structured` and the strict `safety.v1` output decoder. **Authenticated** `CallMeta{UserID, TurnID, Bundle}` must be injected into the turn context with `WithCallMeta`; missing or inconsistent metadata rejects the call. It cannot be sourced from user text.
- `NewGate(assets, classifier, optionalSecondLook)` implements `ports.SafetyGate`: L0 deterministic RE2/literal checks on the turn and bounded recent user messages; L1 classifier; deterministic max; conservative elevated-on-error fallback; rule IDs and rune offsets, never raw matches. Optional SF-007 downgrade requires verified *per-trigger* contextual evidence and explicit absence of first-person present tense. With no approved verifier, disable the second look (pass nil).
- `NewPlanner(assets, clock)` implements `ports.CrisisPlanner`: only reviewed bundle-owned locale scripts and verifiably current resources, with a reviewed general-resources notice if using a configured fallback locale. Missing/expired assets fail closed; it does **not** invent contacts, script text, resource verification, emergency dispatch, or human escalation.
- `NewFSM(assets).Advance(...)` is a **pure** risk transition: any increase applies, a decrease needs verified explicit safe statement, no new safety signals, and the minimum window from the pinned policy; imminent does not auto-decay. It returns follow-up constraints and the minimal transition, without modifying persistence. `RiskChange.AddToDelta(&prepared.State, categories)` adds exactly one risk transition and metadata-only `risk_level_changed` outbox proposal to the existing turn delta. Persist **only** via `TurnFinalizer.Complete`.
- `Assets.FallbackFunc` adapts the normal/fast WP-05 fallback signature. Crisis uses `CrisisPlan.TemplateText`, not this generic fallback. Every fallback remains subject to the later-phase Validator; nothing is streamed or emitted without validation. `StaticCrisisFallback` only checks the shape of a plan and must not replace the validator.

## Integration work for the composition/orchestrator owner (NOT modified here)

1. Obtain a **release-verified, clinician-approved** WP-03 bundle per pinned turn; explicitly disallow `synthetic_only` and verify its signatures/integrity. Create the WP-06 components with exactly that bundle reference and the WP-04 client/strict WP-01 schema set. WP-03 currently provides only synthetic placeholders: normal production `Load` must reject it.
2. At the **authenticated** ProcessTurn boundary, after durable admission but before safety L1, attach `WithCallMeta(ctx, contract.CallMeta{UserID: admission.UserID, TurnID: admission.TurnID, Bundle: pinned.Reference()})` to the context. The current WP-05 service does not do this; without it L1 fails conservatively. Do not modify `SafetyInput` contracts or put IDs inside model text.
3. Pass the **persisted** `UserState.Risk` into `SafetyInput.Risk` (WP-05 currently seeds elevated). Use the resulting `RiskChange.State` to populate the assembler's `RiskView`, follow-up constraints, and normal/fast eligibility. WP-05 already routes high/imminent results to buffered CRISIS using `Router` (RT-001), and raises Brain-discovered high risk before generation. If a crisis route has no approved category-specific plan, finalization must fail safely and retain the user-only turn.
4. After validation, apply the pure FSM and `RiskChange.AddToDelta` to the *same* prepared `StateDelta` and outbox transaction, with optimistic versions. The WP-02 finalizer is the **only** authority for writing state/events. Do not introduce an independent risk write or hard delete. Retain existing message, trace, risk event and historical evidence under Soft Delete; the `RiskState.History` field is just a bounded read-model view, not a deletion directive.
5. The integration with WP-08 state/assembler and the later-phase Validator must be coordinated in a **separate** integration patch. This WP-06 patch intentionally does not modify either, composition, shared contracts, or WP-05 internals. Crisis plans contain contacts as structured `CrisisResource` values; the output Validator must enforce contact whitelist, required elements and forbidden content (`OV-011`, `OV-020`, `OV-021`) before returning HTTP success. Human escalation policy is still an open product/legal/clinical decision.

## Clinician and product/legal review blockers

**Do not deploy with actual users** until clinical owners have authored and approved each locale's L0 lexicon including context flags, L1 prompt, de-escalation policy, second-look verifier, crisis family templates, fallback copy, suitable verified contacts and general-resource notice. The WP-03 placeholder lexicon has zero rules; crisis resources and templates are empty. The package explicitly rejects these for production. Assess locale coverage, verification cadence, how to handle unavailable crisis plans and third-party contact/legal jurisdiction, and whether a staffed escalation path exists. Engineering tests demonstrate contract properties, **not** medical correctness, sensitivity/specificity, or real-world safety performance. WP-15 evaluation and clinical golden cases remain necessary.

## Verification

With normal Go dependencies available:

```sh
cd apps/api
go test -race ./internal/modules/ai/safety ./internal/modules/ai/application/orchestrator
go vet ./internal/modules/ai/safety
```

`TestWP03RealSyntheticBundleFailsClosed` reads the actual `knowledge/` tree under standard CI. For an environment lacking the real YAML/UUID/JSON-schema modules, the tests also accept an **out-of-tree, semantic YAML-as-JSON fixture copy** via `WP06_TEST_KNOWLEDGE_ROOT`, but such an offline shim does not substitute for compiling and testing with real dependencies. No offline test stand-in is part of this patch.
