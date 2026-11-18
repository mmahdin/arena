# AI Core Architecture & Implementation Audit

**Repository:** `mmahdin/arena` @ `0410a4a` (branch `arena/4c22c624-arena`)
**Scope:** Final Architecture Document + Contract Pack v1.2 (`docs/ai-core/`), the whole Go monolith, with depth on AI Core WP‑01 … WP‑15 and the WP‑14/integration/hardening work.
**Audit type:** Independent, read-only. **No code was modified.**
**Date:** 2026‑11‑18

### Method and its limits (read this first)

I read both authoritative documents end-to-end (§0–§11 of the Final Architecture; §0–§13 + appendices of Contract Pack v1.2), then traced the implementation through `contract`, `ports`, `application/*`, `adapters/*`, `bundle`, `safety`, `validator`, `eval`, `trace`, `composition`, migrations `0017`–`0022`, and the HTTP surface.

**What I could not do:** this sandbox has **no Go toolchain and no network access to obtain one**, and no SQL Server. I therefore **did not execute `go build`, `go vet`, `go test`, `-race`, or any live‑SQL test.** Nothing in this report is inferred from "tests pass"; every finding below is grounded in source or in a documented claim that I was able to corroborate (or contradict) by reading. Where the previous agents' verification claims are plausible and consistent with the code, I say so; where I could not verify, I say so explicitly. Statements in `docs/quality/ai-core-integration.md` about 31 passing SQL tests, `go vet` clean, etc., are **taken as unverified self-reports**, not as evidence.

**What I deliberately excluded** (per instructions): security hardening, clinical safeguards, production-readiness, and the correctness of safety thresholds. I touch safety only where it reveals *architectural* structure.

---

## 1. Overall architectural assessment

**Short version: this is a genuinely strong, disciplined implementation with an unusually honest paper trail — and it is not the system the architecture describes, because the architecture's central object (a single live turn pipeline) does not exist at runtime. What exists is an excellent, heavily tested *library* of correctly built components plus a second, disjoint prototype that actually serves users.**

### 1.1 What is genuinely good (and should not be churned)

These are not polite compliments; they are the parts future work must preserve.

| Area | Evidence | Why it matters |
| --- | --- | --- |
| **Layering is actually clean** | `contract` imports nothing internal; `ports` imports only `contract`; **zero** non-test imports of `gin`, `database/sql`, or `adapters/*` from any `application` or `domain` package; SQL confined to `adapters/mssql` | The dependency-direction rule from `AGENTS.md` is genuinely honoured. This is rare in a 26k-line module and is the main reason the codebase is still changeable. |
| **Ports match the contract almost verbatim** | `ports/ports.go` is line-for-line the §6.1 skeleton (`Gateway`, `ConversationStore`, `TurnFinalizer`, `TurnSerialiser`, `SafetyGate`, `ContextAssembler`, `Retriever`, `Brain`, `ProtocolEngine`, `Generator`, `Validator`, `StateCommitter`, `MemoryWriter/Committer`, `Reflector`, `Scheduler`, `BundleRegistry`, `Tracer`) | The frozen seam promised in Contract Pack §0 exists in code. Swapping implementations is genuinely possible. |
| **Bundle pinning is real and rigorous** | `orchestrator/integration.go` `pin()` casts the registry to `pinnedRegistry`, forces every module (safety assets, validator, generator, protocol, reflection, memory) to report the *same* `BundleRef` incl. `PromptSHAs`, and injects the handle into `context` | This is the strongest single piece of engineering in the repo. It structurally prevents the silent mid-turn bundle switch the architecture warns about. |
| **Risk FSM is monotone and fail-closed** | `safety/fsm.go` `Advance()`: unknown level → `elevated`; only `Rank()` increases, or a *verified* de-escalation with `UserStatedSafeVerified` **and** elapsed hours from bundle policy; `imminent` cannot be downgraded at all | Invariant 6 ("risk monotonicity") is enforced in code, not prompt text — exactly as specified. |
| **Resource invariant enforced** | `validator/validator.go` rejects any phone/URL not in `crisis.Resources`; `bundle/policy.go` lints locale coverage, `verified_on`, and staleness | Invariant 7 ("resources only from the curated table") is real. |
| **Soft-delete correction is thorough** | `0020_soft_delete_preservation.sql` uses a **catalog-driven cursor** to rewrite *every* non‑NO‑ACTION FK in `dbo` to `ON DELETE NO ACTION`, adds `AFTER DELETE` guard triggers to every table, drops the purge proc, and makes `ai_turn_traces.expires_at` nullable. `0018`'s 17 `ON DELETE CASCADE` declarations are thereby neutralised at deploy time. | The WP‑02 amendment is implemented at the *schema* level generically rather than by hand-patching a list — so it also covers future tables. That is the right shape of fix. |
| **No LLM call inside a transaction** | Embedding/vector computation happens *before* `BeginTx` (`memory_repository.go`, `memory/engine.go` comment: "vectors must have been computed before the short optimistic SQL transaction"); finalizer transactions contain no provider calls | Definition-of-Done §13.2.4 is honoured. |
| **Schema/Go parity is machine-checked** | `contract/schema/schema_test.go` compiles the real Draft‑2020 JSON Schema with an *independent* compiler, then asserts the Go decoder agrees, then re-validates the Go round-trip | Prevents the classic "Go struct silently drifted from the schema" failure. |

### 1.2 The central architectural finding

> **`NewAIInternal` — the only constructor that assembles WP‑01…WP‑15 — has zero non-test callers.**

Evidence:

- `cmd/api/main.go` calls `composition.NewAPI` and nothing else.
- `composition/api.go` `NewAPI` wires the **prototype**: `ai.NewModule(cfg) → application.New(openai|mock gateway) → conversation.…Turn` and `/turns` → `Service.Reply(...)`, i.e. `supportive-v5` single-shot generation with `policy.CrisisSignal` keyword matching.
- `composition/ai_internal.go` (`NewAIInternal`) and `ai_reflection.go` (`NewAIReflection`) are **never referenced outside tests** (`grep` for `NewAIInternal(` returns one hit: the integration test).
- The two graphs import **nothing** from each other (`legacy application/*` has no `ai/contract` import; the new core never touches `policy`).

So the repository contains **two complete, parallel AI systems** that share only the `dbo.messages`/`dbo.turns` tables:

```
PRODUCTION PATH (live)                     NEW CORE (constructed only in tests)
HTTP /turns
  └ conversation/application.Service
      └ ai/application.Service.Reply       └ orchestrator.Service.ProcessTurn
          └ policy.prompt (supportive-v5)      ├ gateway + finalizer (WP-02)
          └ policy.CrisisSignal (keywords)     ├ safety Gate + FSM + Planner (WP-06)
          └ mock | openai gateway              ├ assembler + contextstate (WP-08)
                                               ├ retrieval + embeddings (WP-09)
                                               ├ brain + policy gate (WP-10)
                                               ├ protocol engine (WP-13)
                                               ├ generator + skill cards (WP-11)
                                               ├ validator + classifier (WP-07)
                                               ├ memory writer/committer (WP-12)
                                               └ reflection + scheduler (WP-14)
```

**Is this a defect or unfinished WP‑16?** Both, and the distinction matters:

- *Not wiring an HTTP endpoint, auth adapter, or push transport* is **legitimately WP‑16** (§13.1). `docs/quality/ai-core-integration.md` §6 states this plainly and accurately. I do **not** count that as a defect.
- *But* the stated WP‑05 deliverable is "**Orchestrator walking skeleton … Test a real two‑stage turn end to end**". A `ProcessTurn` that no process ever calls is not a walking skeleton; it is a **facade over a test harness**. And Contract Pack §13.2.9 — the DoD clause that closes WP‑00/§9.4 — says **"No old AI reply/prompt/safety path … remains"**, which is false today: `reply`, `prompt`, and `safety` prototype paths are all still live.
- Consequence for the architecture's core promise: the **`supportive-v5` keyword `CrisisSignal` is what actually protects users today**, while the carefully built layered S‑0…S‑5 safety system is dead code at runtime. That is the single most important thing in this report.

### 1.3 The second structural finding

> **The orchestrator has stopped being a "small route table and a DAG executor" and has become an if‑tree with two divergent pipelines inside one 515-line function.**

`prepare()` (`orchestrator.go`) is **515 lines**, contains **14 `integration != nil` branches** and **2 `integration == nil` branches**, **27** `return contract.PreparedTurn{}, "stage", …` exits, and carries **41 distinct string literals** as stage names. `generateValidated()` is a further 185 lines returning a **7-tuple** `(string, bool, Objective, ValidationReport, bool, *CrisisPlan, error)`.

Concretely, the *same* behavioural decision is implemented twice, guarded by a boolean:

```go
// orchestrator.go ~L600
if o.integration != nil {
    riskChange = o.integration.fsm.Advance(...)        // real FSM
    wc.Risk.Current = riskChange.State.Level
}                                                       // else: no FSM at all
```

> Note the adjacent duplication this enables: `fsm.Advance(...)` is invoked at **two** places in the same function (L466 for the pre-Brain signal, L600 again after the Brain may raise risk), each followed by an identical `wc.Risk.Current = safe.signal.Level` / `FollowUpActive` re-sync. The risk state machine is a single transition function being driven from two spots inside an if-tree, with no single place that owns "what is the risk for this turn".

The Final Architecture is explicit that this must not happen: §4.1 "**The Orchestrator is a route-selecting DAG executor, not a general agent loop. Routes are enumerated (4), steps are typed, and the only loop is the capped retrieval hop. This is what makes the system testable and its latency predictable.**" A boolean that toggles *whether safety, state and pinning exist at all* is exactly the opposite of that.

### 1.4 Scorecard

| Dimension | Verdict | Note |
| --- | --- | --- |
| Architecture compliance (components) | **Strong** | Ports, FSMs, invariants, pinning, bundle, migrations all faithfully built |
| Architecture compliance (system) | **Failing** | Two parallel systems; the documented pipeline is not the running one |
| Layering & module boundaries | **Strong** | Clean, verified |
| Implementation quality (per component) | **Good → very good** | Memory engine, FSM, validator, bundle lint are careful |
| Implementation quality (orchestrator) | **Weak** | 515-line dual-mode function, 7-tuple return, magic numbers |
| Extensibility | **Mixed** | Ports are swappable *in principle*, but 8 runtime upcasts + concrete imports make it non-obvious |
| Duplication / dead code | **Weak** | 4 dead constructors, 2 risk orderings, duplicated memory graph |
| Testing (component) | **Good** | 85 unit test files, 11k lines |
| Testing (system) | **Gap** | Full assembly tested *only* behind `WELLBEING_INTEGRATION=true` + live SQL |
| Documentation honesty | **Unusually high** | Claims match code; the integration doc self-declares the real gaps |

---

## 2. Findings

Severity: **P0** = blocks the architecture's core guarantees; **P1** = significant structural debt that gets worse with time; **P2** = real but contained; **P3** = polish.

---

### P0‑1 — Two live AI systems; the specified pipeline is unreachable in production

**Files:** `internal/app/composition/api.go:47-95`, `internal/app/composition/ai_internal.go:56`, `internal/modules/ai/module.go`, `internal/modules/ai/application/service.go:66`, `cmd/api/main.go:46`

**Evidence:**
- `NewAPI` → `ai.NewModule(cfg)` → `application.New(openai.New|mock.Gateway)`, and `chat.Turn` calls `Service.Reply` (`supportive-v5`).
- `NewAIInternal`/`NewAIReflection` have **no non-test callers**.
- `grep 'ai/contract' internal/modules/ai/application/*.go …/policy/*.go` → **none**: the graphs share no contract.

**Why it is P0, not "WP‑16"**
The architecture's §2.3 critical path, §2.5 invariants 1–8, §9's five safety positions, and §9.5 traceability all describe *one* pipeline. Today users traverse a different one. Every architectural guarantee is therefore currently theoretical. The DoD explicitly requires the prototype path be removed (`§13.2.9`, `§9.4`), not merely superseded.

**Recommendation**
1. Decide and document the runtime authority in one place (`composition/api.go`), with a **single** `TurnProcessor` seam.
2. Land the cutover early — even behind a feature flag that is *off in production but on in CI* — so integration is continuously exercised rather than a one-off WP‑16 event. A flag that is never on is the same as no cutover.
3. Delete `Reply`, `policy.prompt`, `policy.CrisisSignal`, `adapters/openai`, `adapters/mock` (**but see P1‑5 first** — the reply path owns the only working admission/lease logic the prototype relies on).

---

### P0‑2 — The unified composition is verified only by an opt-in, live-SQL test

**Files:** `tests/integration/ai_internal_test.go:178`, `tests/integration/ai_lifecycle_test.go:25-37`, `internal/app/composition/ai_internal.go`

**Evidence:**
`NewAIInternal` appears in exactly one test. That test's helper `aiDatabase()` skips unless:
```go
os.Getenv("WELLBEING_INTEGRATION") == "true" && APP_ENV == "local" && sqlserver URL && host ∈ {127.0.0.1, localhost, mssql}
```
Therefore `go test ./...` — the command listed in `AGENTS.md` DoD and in the integration doc's verification table — **does not execute the only test that assembles the whole core**. Everything in `NewAIInternal` (the 40-step wiring, the `deps` struct, the `oneBundle` registry, classifier injection, worker start/join) is unexercised in default CI.

This also means the very `NewAIInternal` code path the integration doc calls its headline deliverable is the **least continuously verified** code in the repository.

**Recommendation**
Introduce a **fake in-memory `ConversationStore`/`TurnFinalizer`/`StateReader`** (the pattern already exists: `orchestrator_test.go` has a full in-memory `store`) and add a `composition`-level test that builds the *entire* graph with a fake provider, fake embedder, and in-memory persistence — no SQL. That makes "does the whole thing wire together and process a turn?" a pre-commit signal, not a manual ritual. Keep the live-SQL test as the *deeper* layer.

---

### P1‑1 — Orchestrator: one function, two pipelines, no type safety

**Files:** `internal/modules/ai/application/orchestrator/orchestrator.go` (`prepare` L~276–791; `generateValidated` L~850–1035), `orchestrator/integration.go`

**Evidence:**

| Metric | Value |
| --- | --- |
| `prepare()` length | **515 lines** |
| `integration != nil` branches | **14** |
| `integration == nil` branches | **2** |
| `return PreparedTurn{}, "…", …` exits | **27** |
| distinct stage string literals | **41** |
| `generateValidated` return arity | **7** |

The `stage` is a bare `string` threaded through 27 returns and then stored into the trace; a typo yields a silent mis-categorised failure rather than a compile error. `integration == nil` is a *whole second semantics* (no pinning, no FSM, no state reader, restricted objectives) selectable at construction time but branching at every step.

**Why this is P1 and not P3:** every future WP (WP‑16 cutover, proactive turns, merge-split, replay) must add yet another `integration != nil` branch or another element to the 7‑tuple. This is the classic shape of a function that will be impossible to change within two more features. It is also the direct cause of P0‑2 being hard to test: there is no way to test "the pipeline" without instantiating both modes.

**Recommendation (ordered by impact)**
1. **Delete the `integration == nil` mode.** The WP‑05 isolated skeleton has served its purpose; keeping a second semantics in production code is pure cost. Its unit tests can construct the real pipeline with fakes.
2. **Split `prepare` into `Stage` types** matching §4.3's numbered table: `admit → gate → assemble → route → decide → policyGate → protocolResolve → generate → validate → commit`. Each stage takes a `*turnState` and returns `error`. Stage names become typed constants.
3. **Replace the 7-tuple** with a `generationOutcome` struct. Tuples of this arity are a defect in Go.
4. Raise `errcheck`/`gocyclo`/`funlen` thresholds in CI so this cannot silently regrow.

---

### P1‑2 — The port abstraction is bypassed 8 times by runtime interface upcasts

**Files:** `orchestrator.go:122, 498, 531, 642, 670, 706, 855, 991`

**Evidence** (representative):
```go
if p, ok := o.deps.Protocol.(interface{ ShortlistWithContext(context.Context, protocol.ShortlistContext) ([]contract.ProtocolBrief, error) }); ok { … }
if g, ok := o.deps.Generator.(interface{ WP11(); TurnTimeout() time.Duration }); ok && g.TurnTimeout() < turnTimeout { … }
engine, ok := o.deps.Protocol.(interface{ ResolveTurn(context.Context, protocol.ResolveInput) (contract.ProtocolStepSpec, contract.ProtocolPatch, error) })
if !ok { return contract.PreparedTurn{}, "protocol_integration", coreError(contract.ErrValidation, "protocol_integration", nil) }
```

Three distinct problems:

1. **The port is not the contract.** `ports.ProtocolEngine` declares `Shortlist(...)` and `Resolve(...)`; the orchestrator actually needs `ShortlistWithContext` and `ResolveTurn`. So implementing the declared interface is *necessary but not sufficient* — a third‑party or alternate implementation compiles, passes `ports` contract tests, and **fails at runtime** with `protocol_integration`. This directly contradicts Contract Pack §6.1 ("Each new port receives real and fake contract tests") and §9.7 ("every module is behind a Go interface … swap implementations … without touching the Orchestrator").
2. **Feature detection is invisible.** `WP11()` and `TurnTimeout()` are probed to decide budgets; absent them, timeouts silently fall back to defaults. A missing capability changes *behaviour* without any compile-time or wiring-time signal. The `eval/capture.go:279` wrapper exists specifically to "forward WP‑11 budgets/fallback capability" — i.e., an entire bug class was created by this pattern.
3. **The orchestrator imports 8 sibling `application/*` packages** (`brain`, `generator`, `protocol`, `memory`, `reflection`, `assembler`, `contextstate`, `state`), so "ports-first" holds for *storage* but not for *collaborators*.

**Recommendation**
- Move the real methods onto `ports.ProtocolEngine` / `ports.Generator` (add `ShortlistWithContext`, `ResolveTurn`, `MaxSyncCalls`, `GenerationTimeout`, `MaxRegenerations`, `OutagePlan` explicitly), or split into `ProtocolShortlister`/`ProtocolResolver` interfaces that are *declared in `ports`* and asserted at compile time with `var _ ports.X = (*protocol.Engine)(nil)`.
- Delete the `WPxx()` marker methods; they are version tags masquerading as capabilities.
- Add a `ports/contracttest` case per method actually called, so the failure is caught by tests rather than by a user turn (P1‑8).

---

### P1‑3 — Two risk orderings with contradictory semantics, plus magic thresholds

**Files:** `application/orchestrator/router.go:18`, `application/state/user.go:12`, `contract/enum_validation.go:283`; usages `orchestrator.go:479, 604`, `application/state/protocol.go:131`

**Evidence:**

| Location | `unknown`/invalid returns | Fail direction |
| --- | --- | --- |
| `contract.RiskLevel.Rank()` | `-1` | below `none` |
| `application/state.riskRank` | `-1` | below `none` |
| `application/orchestrator.riskRank` | **`2` (elevated)** | **at/above elevated** |

These disagree **on the only input where it matters**. If an invalid `RiskLevel` ever reaches `orchestrator.riskRank`, it is treated as *elevated and restrictive* — which is the safe choice — but the identical value routed through `state.riskRank` evaluates as *less than none*, i.e. maximally permissive. The contract type already provides `Rank()`; two private copies were added anyway.

Compounding it, the orchestrator compares against **literals**:
```go
if riskRank(safe.signal.Level) >= 3 && decision.Route != contract.RouteCrisis   // what is 3?
if riskRank(safe.signal.Level) >= 3 { decision.Route = contract.RouteCrisis }
```

**Recommendation**
Delete both private `riskRank` helpers; use `contract.RiskLevel.Rank()` and `contract.RiskHigh`. Add a table-driven test pinning `Rank()` for all enum values **and** for invalid input, and a lint rule banning numeric comparisons against `RiskLevel`. This is a two-hour fix that removes a class of latent safety-ordering bug.

---

### P1‑4 — Four dead composition constructors and a duplicated memory/reflection graph

**Files:** `internal/app/composition/ai_brain.go` (34 L), `ai_generator.go` (26 L), `ai_retrieval.go` (21 L), `ai_memory.go` (44 L), `ai_reflection.go` (70 L), `ai_persistence.go` (14 L)

**Evidence:**

| Constructor | Non-test callers |
| --- | --- |
| `NewAIReflection` | **0** |
| `NewAIBrain` | **0** |
| `NewAIGenerator` | **0** |
| `NewAIMemory` | **0** |
| `NewAIPersistence` | 1 (`ai_internal.go`) |

`NewAIInternal` already assembles memory + reflection inline (`memory.Load`/`NewWriter`/`New`/`NewWorker`/`reflection.Load`/`NewModel`/`NewWorker`), so `NewAIReflection` is a **second, divergent copy of the same object graph**. Today they happen to agree; nothing keeps them agreeing. A future WP‑14 change applied to one and not the other is a silent behavioural fork — the exact failure mode Contract Pack §0 ("Never create duplicate sources of truth") forbids.

**Recommendation**
Delete `NewAIReflection`, `NewAIBrain`, `NewAIGenerator`, `NewAIMemory` unless a WP‑16 hosting story needs a *subset*; if so, refactor to one `buildMemoryAndReflection(...)` helper returning a struct, used by both. `NewAIPersistence` is a one-line alias and should be inlined. Add `deadcode`/`unused` (staticcheck `U1000`) to CI so this cannot recur.

---

### P1‑5 — The prototype path still owns real infrastructure the new core must not duplicate

**Files:** `conversation/application/service.go:25` (`PromptVersion`), `internal/modules/conversation/adapters/mssql/lifecycle.go` (473 L), `app/composition/api_test.go`

**Evidence:** `/turns` remains the only user-facing turn endpoint; the response DTO still carries `promptVersion` and the OpenAPI/`check_contract.py` gate (30 route pairs) is written against the **prototype** surface. `ai_test` and `mvp_acceptance_test.go` exercise the prototype.

This is the mirror image of P0‑1 and is the reason the naive instruction "just delete `Reply`" is unsafe: the prototype turn path is currently the *only* working implementation of per-user leasing plus idempotent admission that users actually hit. Removing it is therefore a **migration**, not a deletion.

**Recommendation**
Before deleting anything: enumerate, in one document, every responsibility the prototype path holds (lease, idempotent admission, model gateway, crisis keyword, prompt version in the API response). Mark each as *new-core already provides* / *new-core provides differently* / *must migrate*. `promptVersion` in particular is a **client-visible contract** that the AI Core replaces with `bundle_version`; that is an API change requiring OpenAPI work (WP‑16) and must not be silently dropped.

---

### P1‑6 — Traceability: the architecture's "replayable turn" is a summary, not an artifact

**Files:** `trace/recorder.go:19-40`, `db/migrations/0018_ai_core_state.sql:227-239`, `eval/replay.go`, `eval/runtime.go`

**Evidence:**
```go
type Observation struct { Duration; Count int; Degraded bool; RuleID string }
type Step struct { Name; RuleID string; DurationMS int64; Count int; Degraded bool; ErrorCode }
type Snapshot struct { TurnID; Route; Steps []Step; Usage; Finished bool }
```
`ai_turn_traces` stores `route`, `versions_json`, `payload_enc` (NULL-able, never populated by the recorder), and `summary_json`.

There are, in effect, **two incompatible trace models**. `eval.Trace` (`eval/replay.go`) carries `route_rule`, `versions{bundle, models, prompts}` and `steps` — the shape §3.15 asks for — but it exists only inside the test harness's captured fixtures. `trace.Snapshot` is what the running orchestrator actually produces, and it carries none of that. So the repository already contains the *right* model; it is simply not the one wired to persistence.

Final Architecture §3.15 requires one trace per turn containing "**input hashes, context item ids, safety signal, Brain input/output, selected skills/protocol, validator report, latencies, token counts, model + prompt + bundle versions, errors and fallbacks**", and §9.4 promises "any past turn can be *re-run against a candidate bundle*". The implemented trace contains **latency, counts, rule ids, and usage** — sufficient for dashboards, insufficient for replay. The integration doc concedes exactly this (Remaining issue #2), which is to its credit — but the architectural consequence should be stated plainly: **§9.4 L2/L3/L4 evaluation and "shadow mode" (§9.7) are not achievable on this trace format.** The WP‑15 harness works only because it uses *separately captured, complete fixtures* (`eval/capture.go`), not runtime traces.

**Recommendation**
Treat the trace as a **versioned artifact with two tiers**: (a) a cheap operational summary (today's `summary_json`), and (b) an encrypted, access-controlled `payload_enc` containing typed, redacted-then-encrypted inputs/outputs sufficient for replay, written on the same path as `Complete`. Without (b), the eval story cannot progress past fixtures, and the "versioned, replayable turn trace" decision (#10 in §0) is nominal.

---

### P2‑1 — `reflect.DeepEqual` used as identity for 41 bundle/record comparisons

**Files:** `orchestrator/integration.go`, `brain/service.go`, `generator/service.go`, `memory/{service,worker,writer}.go`, `adapters/mssql/{memory_jobs,memory_writer,memory_feedback}.go`, … (41 non-test sites)

**Evidence:**
```go
if !approved || ref.Version == "" || ref.Version != i.assets.Reference().Version ||
    !reflect.DeepEqual(ref.PromptSHAs, i.assets.Reference().PromptSHAs) { … }
```

`reflect.DeepEqual` will report **not equal** for semantically identical values that differ only by `nil` vs empty slice, and it is ~10–50× slower than a purpose-built comparison. For `PromptSHAs` (a `map[string]string`) it happens to work; for `contract.ResponsePlan`, `MemoryItem`, `FormationDoc` comparisons it is **semantically fragile** — a future `omitempty`/pointer change silently flips "same bundle" to "mismatch", which fails closed as `ErrBundle` and looks like an operational incident rather than a code change.

**Recommendation**
Give identity-bearing contract types explicit `Equal()` methods (hand-written or `go‑cmp`), and compare `BundleRef` by a single canonical digest computed once at load. Reserve `reflect.DeepEqual` for tests.

---

### P2‑2 — Bundle "registry" is a single pinned instance; no registry semantics exist

**Files:** `app/composition/ai_reflection.go:68-70`, `orchestrator/integration.go:117-124`

```go
type oneBundle struct{ ports.Bundle }
func (b oneBundle) Active(context.Context, c.ID) (ports.Bundle, error) { return b.Bundle, nil }
```

Contract Pack §6.1 declares `BundleRegistry` pinned "per user cohort/flag", and §8.2/§6.7 require cohort rollout, percentage rollout, and instant rollback by pointer change. The implementation returns one immutable snapshot for everyone and ignores the user id. That is a reasonable MVP *if labelled as such* — but `BundleRegistry` is currently a **name that promises a capability that does not exist**, which is precisely how future work gets mis-scoped.

**Recommendation**
Rename to `StaticBundle` (or `PinnedBundle`) with a doc comment stating cohort rollout is not implemented, **or** implement a thin registry over `map[cohort]ports.Bundle` + a `cohorts` table. Do not leave a registry-shaped interface backed by a constant.

---

### P2‑3 — Protocol eligibility gating lives in the Orchestrator, not the Protocol Engine

**Files:** `orchestrator.go:496-527`, `application/protocol/eligibility.go` (221 L), `application/protocol/engine.go` (465 L)

The Final Architecture §3.6 says the Protocol Engine owns "**Eligibility filter (pre-Brain): for each protocol, evaluate `preconditions` … then rank by `fit(...)`. The top five go to the Brain**". In the code, the orchestrator computes `protocol.ExplicitReadiness(wc.Turn.Text)`, calls `ProtocolFacts.Flags(...)`, decides `wc.Eligibility.ProtocolsPossible`, appends `"no_eligible_reviewed_protocol"`, and *then* calls `ShortlistWithContext`. Eligibility policy is therefore split across two packages, and `reasons_blocked` is mutated by the caller of the very component that should own it.

**Recommendation**
Move the gate behind `ProtocolEngine.Shortlist` so it returns `([]ProtocolBrief, EligibilityReport, error)`. The orchestrator should call one method and receive both the candidates and *why* they were filtered. This restores §3.6's ownership and shrinks `prepare` (P1‑1).

---

### P2‑4 — State/thread arithmetic duplicated between orchestrator and `contextstate`

**Files:** `orchestrator.go:731-760` (consecutive-questions, `pending_question`), `application/contextstate/*`, `application/state/*`

`delta.Session.ConsecutiveQuestions` is computed in the orchestrator by reading `wc.Session.ConsecutiveQuestions`, testing `validationReport.Extracted.QuestionCount == 1`, incrementing, and building `PendingQuestion{…}` inline. The question-budget invariant (§2.5 #4) is therefore enforced **by the caller**, while `state.Service.validateSession` re-validates it. Two implementations of one rule, one of which is arithmetic-on-a-struct inside a 515-line function.

**Recommendation**
Have `state.ValidateDelta` *own* the increment and pending-question derivation, taking `validationReport.Extracted` as an input. Orchestrator passes facts; committer computes state. (This is also what "single-writer invariant" implies at the *logic* level, not just the SQL level.)

---

### P2‑5 — `nil`→`[]` normalisation is encoded by 40 hand-written marshallers instead of one type

**Files:** `contract/json.go` (406 L, **40** `MarshalJSON` methods), `contract/json_tags_test.go`

**Evidence:** every one of the 40 methods has the same shape:
```go
func (v SessionPatch) MarshalJSON() ([]byte, error) {
    type plain SessionPatch
    if v.EmotionLabels == nil { v.EmotionLabels = []string{} }
    return json.Marshal(plain(v))
}
```
The affected types are **all internal in-process / persisted contract types** (`Admission`, `StateDelta`, `WorkingContext`, `BrainDecision`, `MemoryWriterOutput`, `ReflectionResult`, …) — **not** HTTP DTOs (I verified the HTTP layer serialises its own `conversation` structs). So the intent is legitimate: Contract Pack §1.1 requires required collections to marshal as `[]`, and these values cross the LLM and DB boundaries as JSON.

The problem is **how** it is enforced: 40 copies of one mechanical rule, maintained by hand, on types that will keep being added (`ThreadPatch`, `PrefsPatch`, `EpisodeDoc`, …). A new contract type with a required slice field silently violates §1.1 unless someone remembers to add a method. Note the asymmetry: `json_tags_test.go` is genuinely strong — it AST-walks *every* struct in the package and demands a valid snake_case `json` tag on every exported field — so **naming** is exhaustively enforced, while the `nil`→`[]` **behaviour** has no equivalent guard. That contrast is the point: the same reflective technique used for tags would close this gap.

**Recommendation**
Introduce one reusable type — `type List[T any] []T` with a `MarshalJSON` that emits `[]` for `nil` — and use it for required collections; then delete the 40 methods. Alternatively keep the structs and replace the hand-written methods with a single reflective `normalizeRequired(v any)` invoked from one place. Either way, add a reflection-driven test that walks **every** type in `contract` and asserts each non-`omitempty` slice marshals to `[]` — turning a remembered convention into a checked invariant.

### P2‑6 — `ai_turn_traces` and `ai_embeddings` inheritance of pre-amendment cascade is fixed by *cursor order*, not by construction

**Files:** `db/migrations/0018_ai_core_state.sql` (17 × `ON DELETE CASCADE`), `0020_soft_delete_preservation.sql`

`0020` correctly rewrites all cascades to `NO ACTION`, but it does so by iterating `sys.foreign_keys WHERE delete_referential_action <> 0`. That means: **any future migration that adds a cascading FK in `dbo` silently reintroduces the failure mode**, because `0020` will have already run. The safety of the soft-delete invariant depends on an ordering convention nobody can enforce.

**Recommendation**
Add a **migration-time assertion** to the runner (or a `check_structure.py` SQL lint) that fails if any `dbo` FK has `delete_referential_action <> 0` *after* migrations. That turns an invariant that "happens to hold today" into one that cannot regress.

---

### P2‑7 — Legacy prototype holds `supportive-v5` and keyword crisis detection behind an API-visible field

**Files:** `internal/modules/ai/policy/prompt.go:12`, `policy/safety.go:39-70`, `conversation/application/service.go:25`

Even after P0‑1 is resolved, two couplings linger:
- `policy.Version == "supportive-v5"` is asserted by a unit test and surfaced as `promptVersion` on `/turns` and `/me/ai-status`. It is a **client-visible contract** that the AI Core replaces with `bundle_version`.
- `CrisisSignal` (substring matching over a phrase list) is the runtime safety control.

**Recommendation**
Track these two explicitly in the WP‑16 migration plan as *contract changes*, not deletions. `promptVersion` removal in particular must be coordinated with clients/OpenAPI; silently dropping it breaks the frontend.

---

### P3‑1 — Boolean-blind boolean returns
`generator.Generate` returns `(GenOutput, error)` where `GenOutput` contains `Aborted`, `Sentinel`, and `Usage` — three flags with overlapping meaning ("did we actually produce a usable reply?"). `generateValidated` then returns `(text, wasFallback bool, objective, report, usedTemplate bool, crisisPlan, err)`. Callers must remember which bools compose. A small `genResult` value with a `Usable()` method would remove an entire misreading class.

### P3‑2 — Stage strings
41 string literals used as trace stage names; a typo is silently accepted. `type Stage string` + constants fixes this for free (see P1‑1).

### P3‑3 — `Options.RenewInterval` documented as `0 disables renewal` and *also* validated as "must be non-zero when Recovery is set, but 0 is allowed"
The constructor check reads `opts.RenewInterval < 0 || (opts.RenewInterval != 0 && d.Recovery == nil)`, so `RenewInterval > 0 && Recovery == nil` is rejected while `0 && Recovery != nil` is accepted — meaning lease renewal can be silently disabled in production without any test failing. Make "renewal required" an explicit `Options.RequireRenewal bool` or derive the interval from the lease TTL.

### P3‑4 — Evaluation evidence is thin relative to architecture expectations
`knowledge/evals/golden` and `eval/testdata/golden` contain **3 scenarios** (`anaphora`, `escalation`, `longitudinal`), against §9.4/§11.1's "≈150–300 cases" and §11.1's "golden scenario suite should begin being *authored* from step 1". The harness code (`scenario.go`, `assert.go`, `capture.go`, `regression.go`, `candidate.go`, `runtime.go`) is genuinely capable; the **corpus** is 2 % of target size. This is a content gap, not a code defect, but it means the harness's value is currently unproven at scale.

---

## 3. Missing or incomplete functionality

Ordered by architectural significance. **Bold** = the integration doc does not currently list it (so it is at risk of being mistaken for "done").

| # | Missing / partial | Evidence | Classification |
| --- | --- | --- | --- |
| 1 | **Runtime cutover of the AI Core** — `NewAIInternal` unreachable | `composition/api.go`, no non-test callers | WP‑16 scope, **but blocks all guarantees** |
| 2 | **Prototype AI path not retired** (DoD §13.2.9) | `module.go`, `application/service.go`, `policy/*` | **Genuine gap**, self-declared in §9.4 |
| 3 | Proactive wake-up execution | `ProcessTurn` rejects `OriginWakeUp`; Router→PROACTIVE→reject; doc Remaining #1 | Self-declared internal gap |
| 4 | Runtime replay evidence | trace = summary only; doc Remaining #2 | Self-declared; architectural (P1‑6) |
| 5 | Reflection merge/split destination contract | doc Remaining #3 | Self-declared contract decision |
| 6 | Bundle registry / cohort rollout | `oneBundle{}.Active` returns a constant | **Not listed**; interface overstates capability (P2‑2) |
| 7 | Review/clinical approval, reviewed bundle, real safety assets | `knowledge/*` are placeholders; `AllowsSyntheticForTests` gates | Self-declared; correct to defer |
| 8 | Golden corpus at target scale (3 vs 150–300) | `eval/testdata/golden`, `knowledge/evals/golden` | Self-declared qualitatively; **not quantified** (P3‑4) |
| 9 | Streaming to the client (SSE) | `TokenSink` is internal; `Generator` comment says "not streaming" | OPEN‑6, explicitly deferred |
| 10 | Human escalation / notification transport | outbox rows only | Self-declared, out of scope |
| 11 | **`/me/ai-status` reports the prototype, not the core** | `composition/api.go` `aiStatus` uses `policy.Version` | **Not listed**; misleads operators |
| 12 | **No `ports/contracttest` coverage for methods actually called** | `ports/contracttest/lifecycle.go` covers recovery only | **Not listed**; would have caught P1‑2 |

Items 1, 3, 4, 5, 7, 9, 10 are accurately disclosed in `docs/quality/ai-core-integration.md` — that document is trustworthy on scope. Items 6, 11, 12 and the orchestrator/duplication findings (P1‑1…P1‑4, P2‑1…P2‑6) are **not** in it.

---

## 4. Recommended improvements, prioritised

### Tier A — fix before writing more AI features (architectural impact: high, effort: medium)
1. **Kill the `integration == nil` mode** and split `prepare()` into staged functions (§4.3 table). *(P1‑1)* — this is the single highest-leverage change; it makes 2, 3 and every future WP cheaper.
2. **Make the port the contract**: move `ShortlistWithContext`/`ResolveTurn`/budget capabilities onto `ports`, delete `WPxx()` probes, add compile-time `var _ ports.X` assertions. *(P1‑2)*
3. **Unify risk ordering** on `contract.RiskLevel.Rank()`; delete both private helpers; ban numeric risk comparisons. *(P1‑3)* — small, removes a latent safety-ordering bug.
4. **Delete the 4 dead constructors** and collapse the duplicated memory/reflection graph. *(P1‑4)*
5. **Add a fake-persistence composition test** so full wiring runs in default CI. *(P0‑2)*

### Tier B — before the WP‑16 cutover (impact: high, effort: high)
6. **Land the cutover behind a flag that is on in CI**, with one documented `TurnProcessor` seam; then retire `Reply`, `prompt`, `CrisisSignal`, `openai`, `mock`. *(P0‑1, P1‑5, P2‑7)*
7. **Bound the prototype-removal risk** with the responsibility inventory from P1‑5, explicitly including the `promptVersion` client contract.
8. **Design the two-tier trace** (summary + encrypted replay payload). *(P1‑6)* — needed before eval, shadow mode or calibrated judges can mean anything.

### Tier C — debt that compounds if deferred (impact: medium)
9. `Equal()` methods / bundle digest instead of 41 `reflect.DeepEqual`. *(P2‑1)*
10. Rename-or-implement `BundleRegistry`. *(P2‑2)*
11. Move protocol eligibility into the engine. *(P2‑3)*
12. Move question-budget/pending-question derivation into `state.ValidateDelta`. *(P2‑4)*
13. Per-table SQL lint: no cascading FKs post-`0020`. *(P2‑6)*
14. `List[T]` (or one reflective normaliser) to replace 40 `MarshalJSON` methods, plus a reflection-driven test. *(P2‑5)*

### Tier D — housekeeping (impact: low, effort: low)
15. `Stage` constants; typed generation outcome; `RequireRenewal`; staticcheck `U1000`/`funlen` in CI; grow the golden corpus.

### What *not* to do
- **Do not rewrite** the memory engine, risk FSM, validator, bundle loader/linter, or the `0020` soft-delete mechanism. They are careful, well-tested, and correctly shaped. The instinct to "clean up" a 26k-line module usually damages exactly these.
- **Do not** add new modules before Tier A. Adding WP‑16 endpoints onto a two-mode orchestrator will make the seam permanent.

---

## 5. Critical vs. technical debt vs. optional

### Critical architectural problems (block the architecture's own guarantees)
| ID | Problem |
| --- | --- |
| P0‑1 | Two parallel AI systems; the specified pipeline is unreachable at runtime; the prototype's keyword safety is what actually protects users |
| P0‑2 | The only test that assembles WP‑01…WP‑15 is opt-in live-SQL, so default CI never verifies integration |

### Technical debt (real, will make future change harder — not urgent, but not optional)
| ID | Problem | Cost if deferred |
| --- | --- | --- |
| P1‑1 | 515-line dual-mode `prepare`, 27 stringly-typed exits, 7-tuple return | Every future WP adds a branch; testing stays hard |
| P1‑2 | 8 runtime interface upcasts; ports ≠ actual contract | Alternate implementations fail at runtime, not compile time |
| P1‑3 | 3 risk orderings, 2 with contradictory unknown-semantics; magic `>= 3` | Latent fail-open path in state validation |
| P1‑4 | 4 dead constructors + duplicated memory/reflection graph | Silent behavioural fork; two sources of truth |
| P1‑5 | Prototype owns lease/admission/`promptVersion` | Naive deletion breaks live users and the client contract |
| P1‑6 | Trace is a summary, not a replayable artifact | Replay, shadow mode, calibrated eval are blocked |
| P2‑1 | 41 `reflect.DeepEqual` identity checks | Bundle mismatch incidents that look like ops failures |
| P2‑2 | `BundleRegistry` returns a constant | Cohort rollout mis-scoped |
| P2‑3/4 | Eligibility & question-budget logic split across packages | Invariants enforced twice, drift |
| P2‑5 | `nil`→`[]` rule encoded 40 times in `json.go`, unenforced for new types | A new contract type silently violates §1.1; test only spot-checks |
| P2‑6 | Cascade safety depends on migration ordering | A future cascade FK silently voids soft-delete |

### Optional improvements (nice to have; do after Tier A/B)
`Stage` typed constants; generation-result value type; `RequireRenewal`; staticcheck in CI; golden-corpus growth; `ai_status` reporting `bundle_version`; renaming `oneBundle`.

### Explicitly out of scope for this audit
Security hardening, clinical safeguards, safety-threshold calibration, retention/erasure policy, production load/latency, locale coverage, and multi-language safety assets. Those require separate review and are correctly *not* claimed by the repository.

---

## Appendix A — Verification log (what I actually checked)

| Claim | Method | Result |
| --- | --- | --- |
| Tests/build pass | — | **Not verified** (no Go toolchain/network) |
| Layering (§2.2) | grep for `gin`/`database/sql`/`adapters` imports from `application`/`domain`/`contract`/`ports` | **Clean, 0 violations** |
| `ports` == §6.1 skeleton | side-by-side read | Matches nearly verbatim |
| AI Core reachable from `main` | grep `NewAIInternal` non-test callers | **0** |
| Full assembly tested in default CI | read `aiDatabase()` gate | **No** (`WELLBEING_INTEGRATION` only) |
| Dead constructors | grep each `NewAIX(` | `NewAIReflection/Brain/Generator/Memory` = **0** |
| Risk ordering duplicated | grep `riskRank`/`Rank` | **3 implementations, 2 disagree on invalid** |
| Orchestrator complexity | `awk` function-length + grep branch/exit counts | 515 L, 14/2 branches, 27 exits, 41 literals |
| LLM call inside DB tx | read `memory_repository.go`, `memory/engine.go` comments + `BeginTx` sites | **None** |
| Cascades neutralised | read `0018` (17 cascades) + `0020` catalog cursor | Neutralised, but by ordering |
| Trace completeness vs §3.15 | read `trace/recorder.go` + `ai_turn_traces` DDL | **Summary only** |
| Bundle pinning rigour | read `integration.go pin()` | **Strong** |
| Schema/Go parity | read `schema_test.go` | Fixture-based, not exhaustive |
| Golden corpus size | count files | **3 scenarios** (target 150–300) |
| Doc honesty | cross-read integration doc vs code | **Claims match code** |

## Appendix B — Note on WP‑16 attribution

Per instruction, unfinished WP‑16 work is not counted as a defect. I used this line:

- **WP‑16 (not a defect):** no HTTP endpoint for `ProcessTurn`; no auth adapter; no push/notification transport; no public memory-control endpoints; OpenAPI not yet revised to the new turn contract; SSE streaming deferred (OPEN‑6).
- **Not WP‑16 (reported as defects above):** the prototype AI path still being live and *documented as the product's* AI (P0‑1); the core being unverifiable in default CI (P0‑2); the orchestrator's dual-mode structure and interface-upcast contract break (P1‑1, P1‑2); contradictory risk orderings (P1‑3); dead/duplicated constructors (P1‑4); the trace being unable to support replay (P1‑6, which the repo itself flags as an internal integration gap, not WP‑16).

The distinction I applied: **"not yet exposed" is WP‑16; "exposed but structurally unable to hold its own guarantees" is an architectural defect.**
