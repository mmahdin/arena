> **Runtime superseded (2026-11-23):** [Unified memory-centric architecture](unified-memory-centric.md) is the active runtime. Separate Brain/Generator, safety/router/output-classifier calls and regeneration described here are historical. Persisted contracts and historical Inspector examples remain supported.

> **2026-11-21 retrieval/index amendment:** [ADR 0014](../adr/0014-hybrid-retrieval-index.md)
> supersedes historical five-strategy retrieval, handwritten temporal lookup,
> manually weighted ranking, and pre-commit embedding generation below. The
> implemented system uses exhaustive owner-scoped cosine + Okapi BM25 + RRF,
> source/version provenance, and reference-only asynchronous embedding jobs.
> Existing consent, evidence, authority, safety and retention rules remain.

# Theia AI Core — Final Architecture Document

> **Product-first amendment (2026-11-18):** [Conversational refactoring](product-first-conversation.md) supersedes exact ordinary protocol wording, English-only exercises, intake/readiness phrase gates and hard style rejection. It documents retained crisis/persistence boundaries and the implemented before/after behavior.

> **Multilingual amendment (2026-11-18):** [Multilingual conversations](multilingual-conversations.md) separates English knowledge, input language and saved response preference. It supersedes requirements to duplicate internal policies/skills per conversation locale. Persian safety/resources remain unreviewed development content; production release and protocol-language restrictions remain binding.

> **Binding WP-02 correction (2026-11-16): soft deletion only.** All user-initiated deletion hides records using deletion metadata; it must preserve original content, references, and related records. Conversation deletion does not delete or invalidate derived memories, analyses, formulations, embeddings, consent, or other AI state. Retention and UI visibility are separate from permission for AI use: this correction grants neither reuse nor an implicit reuse ban. Existing explicit consent controls remain unchanged; a separate AI-use policy must govern future retrieval/workers. Permanent deletion, legal erasure, backup removal, and retention cleanup require a separate future policy and are not implemented or authorized here. This amendment supersedes earlier destructive-deletion and table-retirement language, including in archived Contract Packs.

*Senior-architect redesign of the preliminary Theia design. Scope: AI Core internals and its integration contract with the existing Go backend. Out of scope: final prompt text, populated knowledge libraries, production code.*

---

## 0. Executive Summary: The Ten Decisions

The preliminary design is conceptually strong but is organized around **psychological concepts** (Turn Analyzer, Context Planner, Problem Understanding, Decision Engine, Intervention Selection, Response Planning), and a literal implementation would chain 5–7 sequential LLM calls per message. That yields 8–15 s latency, compounding error between stages, high cost, and redundant re-analysis. The core design move below is to **separate what must be a model judgment from what is deterministic infrastructure**, and to move slow, deep psychological work **off the critical path**.

| # | Decision | Why |
| --- | --- | --- |
| 1 | **Two synchronous LLM calls on the normal path**: a small structured *Turn Brain* (analysis + decision + plan) and a strong streaming *Response Generator*. | Collapses six logical stages into one reasoning call without losing auditability, because the Brain emits a structured, validated decision record. |
| 2 | **Conversation decision-making lives inside the Turn Brain; technical orchestration lives in a deterministic Go Orchestrator.** | Psychological judgment and execution plumbing are different responsibilities with different failure modes. |
| 3 | **Speculative, deterministic retrieval** replaces the Context Planner as an LLM stage. | The Brain receives a pre-assembled working context. It can request *one* extra retrieval hop only when it detects a gap. |
| 4 | **Case Formulation is a persistent, asynchronously maintained artifact**, not a per-message stage. | Deep formulation is slow, needs multi-turn evidence, and must be revisable. It is built by a background *Reflection Worker* and *read* in-path. |
| 5 | **Interventions are declarative Protocols executed by a deterministic Protocol Engine**; the LLM selects from a pre-filtered shortlist and judges step outcomes. | Multi-step exercises need reliable state, not LLM memory. |
| 6 | **Memory is one provenance-carrying store with typed items** (`reported`, `inferred`, `episode`, …) plus raw messages and thread/plan records. | Six logical categories, three physical structures. Epistemic status is a first-class field. |
| 7 | **State is a hybrid**: a small set of typed state objects (Session, Thread, Protocol Instance, User Risk/Engagement), with formal FSMs only where transitions are genuinely discrete (Protocols, Threads, Risk). | Avoids both "just a summary" and an over-engineered global state machine. |
| 8 | **Single-writer rule**: LLMs *propose* state/memory changes; deterministic committers validate and write. | Prevents hallucinated mutation and inconsistency. |
| 9 | **Safety is layered across the pipeline** (pre-gate, Brain risk assessment, risk state machine, post-generation validation, crisis route that bypasses normal planning). | Not a prompt paragraph. Fail-closed by design. |
| 10 | **Every turn produces a versioned, replayable Turn Trace.** | Enables evaluating internal decisions, not just final text, and safe iteration. |

**Realistic performance target (model-agnostic):** p50 time-to-first-token ≈ 2.5–3.5 s, p95 < 6 s; ≈2 LLM calls synchronous per ordinary turn, 0–1 for trivial turns, and 1–3 asynchronous calls per turn or session.

---

## 1. Evaluation of the Preliminary Architecture

### 1.1 What is sound and should be kept

- **The principle that Theia must not answer only the latest message.** This is the right product thesis and it is what justifies the whole AI Core.
- **Decision-driven (not fixed) execution.** Correct. I refine it: the *routing* is mostly deterministic and cheap; the *decision* is one structured model judgment.
- **Separating "what to do" from "how to say it."** Keep. This is the single most valuable boundary for testability and safety.
- **Reported vs. inferred vs. unknown.** Keep and promote to a storage-level invariant (the `epistemic_status` field), not only a prompt-level rule.
- **Psychology Domain ≠ software module.** Correct and important. Section 5 makes the mapping explicit.
- **Memory categories as logical classes, not six databases.** Correct.
- **A limited, professionally reviewed Intervention Library for the MVP.** Correct; it is also the main clinical-risk control.
- **Stating state management as the hard problem.** Agreed. It is, together with memory write quality.
- **Action taxonomy (`reflect`, `clarify`, …).** Good starting set; I revise it below.

### 1.2 Overlaps and merges

| Preliminary component | Problem | Decision |
| --- | --- | --- |
| Turn Analyzer + Context Planner + State Synthesis (light) + Conversation Decision Engine + Response Planning | All are *one reasoning act*: "given this message and the situation, what is going on and what should I do?" Splitting them creates 4–5 sequential calls that re-read the same inputs. | **Merge into the Turn Brain** (one structured call). Retain their *outputs* as separate fields in the contract so each can still be evaluated. |
| Memory Retrieval | It is an engineering service, not a reasoning module. | **Deterministic Retrieval Service** (SQL + FTS + vector + temporal parsing). |
| Context Planner | Needed only because retrieval was imagined as LLM-driven. | **Eliminated as a stage.** Replaced by speculative retrieval + an optional "request more context" signal from the Brain. |
| Intervention Selection | Selection among 50 techniques by an LLM is unreliable and unsafe. | **Eligibility filtering is deterministic** (prerequisites, contraindications, history). **The Brain chooses from ≤5 candidates.** Execution is the Protocol Engine. |
| Deeper Psychological Formulation as a pipeline stage | Slow, speculative, and per-message formulation tends to over-interpret. | **Becomes a background artifact** maintained by the Reflection Worker; read in-path as context. In-path only in narrow cases (see §4). |
| Domain 6 "Longitudinal Support" | Not an executable module. It *emerges* from Thread Manager + Retrieval + Reflection + proactive follow-ups. | **Domain stays as a policy; no module.** |
| Domain 2 "User Understanding" | Same: a schema plus an information-gathering policy. | **No module.** Implemented as a schema, an *Information Needs* computation (deterministic), and a question budget enforced by the Brain contract. |
| Domain 8 "Evaluation" | Largely offline infrastructure plus tracing. | **Evaluation Harness + Trace pipeline**, not a runtime module (small online hooks only). |
| Safety Gate (Stage 1) and Safety & Ethics (Domain 7) | Split across stage and domain; risk of safety being only one early check. | **Layered Safety System** with 5 positions in the pipeline (§9). |

### 1.3 Missing capabilities (important)

1. **Concurrency and idempotency.** Users send bursts of messages, retry on flaky mobile networks, and switch devices. Needs per-user turn serialization, message coalescing, and idempotency keys.
2. **Session boundary and resumption logic.** "When does a session end?" and "what changes after a 3-day gap?" are never defined. This drives State restoration.
3. **User correction and contestation as a first-class event.** "That's not what I said" / "I never felt that" must route to memory correction, not just a conversational apology.
4. **Proactive and scheduled behavior.** Follow-ups on exercises, check-ins, and "review progress" require a scheduler and a notification contract with the Go backend. Without it, `review_progress` only fires if the user returns.
5. **Privacy, consent, retention, and deletion** as architecture (soft deletion with preserved originals and derivatives; separate AI-use controls; export), not only policy.
6. **Memory content as an injection and drift vector.** Stored text is *data*, not instructions; it can also be wrong, and wrong memory repeated back is a trust-destroying failure.
7. **Dependency and over-reliance monitoring** (named in Safety, but has no data source in the design). It needs engagement signals and a trigger in State.
8. **Locale-aware crisis resources.** Hotlines must come from a curated, versioned table, never generated by the model.
9. **Model fallback and degraded modes.** What happens when the Brain times out or the provider is down?
10. **Cost and token budgeting** per turn and per user.
11. **Onboarding as a bounded, optional flow** that writes into the same schema with `source=onboarding`.
12. **A formal contract with the Go backend** (events, APIs, transactions, auth).

### 1.4 Unrealistic production assumptions

- **"Each stage is a separate LLM call"** will not meet mobile-chat latency expectations and will cause cascading inconsistency.
- **LLM self-reported confidence is poorly calibrated.** The design needs *evidence-count-based* uncertainty (number of independent supporting turns, contradiction count, recency), with the LLM supplying only a coarse categorical judgment.
- **Vector similarity as the main retrieval method.** Fails for temporal queries ("last Tuesday"), exact recall, negation, and thread continuity. Needs a retrieval *router*.
- **A formal state machine for "everything."** Free conversation is not a finite state machine. FSMs should be used where transitions are discrete (Protocols, Risk, Thread lifecycle); the rest is a validated blackboard.
- **Real-time deep formulation.** An LLM given thin evidence will produce confident-sounding but unsupported formulations. Requires evidence thresholds and multi-turn accumulation.
- **"Never silently becomes fact" as a prompt rule.** Must be enforced by write-path validation (the model cannot write `reported` items from inference).
- **The response generator can "just follow the plan".** It needs a defined authority boundary and post-generation validation (see §6.6).

### 1.5 Revised action taxonomy

The preliminary list mixes *conversational acts* with *routes*. Revised:

| Primary objective | Notes |
| --- | --- |
| `reflect` | Listen, mirror emotion/meaning; default for unprocessed distress |
| `validate_and_support` | Emotion-focused support with minimal content exploration (split from `reflect` when the user needs warmth, not mirroring) |
| `clarify` | One targeted question to resolve an ambiguity that blocks understanding |
| `explore` | Open exploration of a topic (OARS style); may include a question |
| `provide_insight` | Offer a tentative interpretation or psychoeducation |
| `provide_guidance` | Practical, non-protocol advice |
| `offer_intervention` | Propose a Protocol; requires user consent before start |
| `continue_intervention` | Advance an active Protocol instance |
| `close_or_pause_intervention` | Explicitly end/pause/redirect (previously implicit) |
| `review_progress` | Follow up on goals/exercises |
| `summarize` | Consolidate; often at session end or topic change |
| `repair` | Handle user correction, misunderstanding, or rupture (new) |
| `answer_direct` | Straightforward factual/logistic question or greeting (new; enables the fast path) |
| `safety_support` | Routed by the Safety layer, **not** chosen freely by the Brain in high-risk cases |

Also added: **`boundary`** (diagnosis requests, medication advice, dependency issues). It is a response *mode* with scripted guardrails rather than free generation.

---

## 2. Final System Architecture

### 2.1 Logical layers

```
┌───────────────────────────────────────────────────────────────────────┐
│  EXISTING GO BACKEND  (auth, API, push, billing, user mgmt, storage)  │
└───────────────▲───────────────────────────────────────┬───────────────┘
                │ gRPC/HTTP: ProcessTurn (stream)        │ events (outbox)
┌───────────────┴───────────────────────────────────────▼───────────────┐
│ L5  GOVERNANCE & OBSERVABILITY                                        │
│     Trace store · Versioned Knowledge Bundle · Eval Harness · Review  │
├───────────────────────────────────────────────────────────────────────┤
│ L4  ORCHESTRATION  (deterministic, Go)                                │
│     Turn Orchestrator · Router · Budget/Timeout · Failure policies    │
├───────────────────────────────────────────────────────────────────────┤
│ L3  REASONING  (LLM-backed, structured)                               │
│     Safety Classifier · Turn Brain · Response Generator               │
│     Memory Writer · Reflection Worker  (async)                        │
├───────────────────────────────────────────────────────────────────────┤
│ L2  SERVICES  (deterministic, Go)                                     │
│     Context Assembler · Retrieval Service · Protocol Engine           │
│     State Committer · Memory Committer · Output Validator · Scheduler │
├───────────────────────────────────────────────────────────────────────┤
│ L1  DATA                                                              │
│     messages · memory_items · threads · protocol_instances ·          │
│     session_state · user_state · turn_traces · embeddings             │
├───────────────────────────────────────────────────────────────────────┤
│ L0  KNOWLEDGE BUNDLE (versioned, read-only at runtime)                │
│     Communication Policy · Skill Cards · Protocols · Schemas ·        │
│     Safety Policy · Crisis Resources · Formulation Frameworks         │
└───────────────────────────────────────────────────────────────────────┘
```

**Deployment suggestion.** Because the backend is Go, implement L2 and L4 as a Go package/service (`aicore`) inside or beside the main backend, sharing the Postgres instance (pgvector + full-text search). L3 is a thin provider-agnostic LLM client layer with schema-constrained output. No separate Python service is needed for the MVP. If prompt experimentation tooling is Python-based, keep it in the offline Eval Harness only.

### 2.2 Module inventory (final)

Seventeen items, but only **three are LLM modules on the synchronous path**.

| # | Module | Type | Sync/Async | LLM? |
| --- | --- | --- | --- | --- |
| 1 | Turn Gateway (ingest, idempotency, lock, coalescing) | Service | Sync | No |
| 2 | Safety Pre-Gate (lexical + small classifier) | Service/Model | Sync, parallel | Small model / classifier |
| 3 | Context Assembler | Service | Sync | No |
| 4 | Retrieval Service | Service | Sync, parallel | No (embedding only) |
| 5 | Turn Brain | Reasoning | Sync | **Yes (small/fast tier)** |
| 6 | Protocol Engine | Service | Sync | No |
| 7 | Response Generator | Reasoning | Sync, streaming | **Yes (strong tier)** |
| 8 | Output Validator (post-gen safety/policy) | Service | Sync | No (rules) + optional small classifier |
| 9 | State Committer | Service | Sync (transaction) | No |
| 10 | Memory Writer | Reasoning | **Async** | Yes (small tier) |
| 11 | Memory Committer (dedupe/merge/supersede/deletion) | Service | Async | No (embedding similarity) |
| 12 | Reflection Worker (formulation, hypotheses, consolidation) | Reasoning | **Async** | Yes (strong tier) |
| 13 | Scheduler / Proactive Engine | Service | Async | No (templated triggers; Brain used on wake) |
| 14 | Turn Orchestrator | Orchestration | Sync | No |
| 15 | Trace Logger | Service | Async | No |
| 16 | Knowledge Bundle Loader/Registry | Service | Startup | No |
| 17 | Eval Harness | Offline | Offline | Yes (judges) |

**Modules intentionally *not* created:** Turn Analyzer, Context Planner, Conversation Decision Engine (as a separate service), Intervention Selector (as a separate LLM), Longitudinal Support, User Understanding.

### 2.3 Architecture in one picture: the critical path

```
user msg
   │
   ▼
[Turn Gateway] ── dedupe, lock, persist raw msg, coalesce
   │
   ├────────────┬────────────────┐   (parallel)
   ▼            ▼                ▼
[Safety     [Context        [Retrieval
 Pre-Gate]   Assembler]      Service]
   │            │                │
   └────────────┴───────┬────────┘
                        ▼
               [Orchestrator Router]
        ┌───────────────┼────────────────────┐
   risk=high        trivial=fast         normal
        │               │                    │
        ▼               │                    ▼
 [Crisis Route]         │            [TURN BRAIN] (LLM #1)
 (templated +           │                    │
 constrained gen)       │        (optional: 1 extra retrieval hop)
        │               │                    ▼
        │               │            [Protocol Engine]  (if protocol active/offered)
        │               │                    │
        └───────┬───────┴────────────────────┘
                ▼
     [RESPONSE GENERATOR] (LLM #2, streaming)
                ▼
        [Output Validator]  ──fail──► regenerate once / safe fallback
                ▼
     [State Committer]  (sync txn)  ─► reply delivered
                ▼
     (async) Memory Writer → Memory Committer → Reflection triggers → Trace
```

### 2.4 Integration contract with the Go backend

The AI Core exposes a deliberately narrow surface:

```
ProcessTurn(TurnRequest) → stream<TurnEvent>
   TurnRequest { user_id, conversation_id, message_id(idempotency key),
                 text, client_ts, locale, tz, channel, attachments? }
   TurnEvent   { type: token | final | safety_banner | protocol_ui | error,
                 payload }

HandleCorrection / DeleteMemory / ExportData / DeleteAccount   (privacy API)
WakeUp(user_id, reason, scheduled_ref)                          (scheduler → proactive turn)
```

The backend owns: auth, rate limiting, push delivery, billing, message transport, and the Postgres instance. The AI Core owns: all tables prefixed `ai_*`, all prompts and policies, and all reads/writes to them. Domain events leave via a **transactional outbox** (e.g. `turn_completed`, `risk_level_changed`, `memory_deleted`) so the backend can react (notifications, clinician-review queue) without the core calling backend internals.

### 2.5 Design invariants (enforced in code, not prompts)

1. **Epistemic invariant:** only the *user's own words* (or explicit user confirmation) can create `reported` items; model-generated items are always `inferred` and carry evidence pointers.
2. **Single-writer invariant:** LLM outputs are *proposals*; only the State Committer and Memory Committer write.
3. **Provenance invariant:** every `memory_item` has ≥1 `evidence` reference (message ID range or parent item) or is rejected.
4. **Question budget invariant:** at most one question per response; consecutive-question counter limits repeated questioning.
5. **Consent invariant:** a Protocol cannot move from `offered` to `active` without a user-affirmative turn.
6. **Risk monotonicity:** a session's risk level can only be *lowered* by an explicit rule (time + user statements + classifier), never by the Brain alone.
7. **Resource invariant:** crisis resources and phone numbers come only from the curated table.
8. **Data-not-instruction invariant:** retrieved memory and user text are delimited as data in all prompts.

---

## 3. Detailed Module Specifications

### 3.1 Turn Gateway (deterministic)

**Responsibility:** Make each user message a safe, idempotent, serialized unit of work.

- **Inputs:** `TurnRequest` from the Go backend.
- **Processing:**
  1. Idempotency check on `message_id`. A duplicate returns the stored result or attaches to the in-flight turn.
  2. Acquire a **per-user turn lock** (Postgres advisory lock or Redis lock with TTL). One in-flight turn per user.
  3. **Coalescing:** if more messages arrive while a turn is queued (not yet in the Brain), merge them into one logical turn. If a turn is already generating and the user sends a new message before streaming finishes, cancel the stream and re-run with both messages ("interrupt and merge").
  4. Persist the raw message in `messages` (always first; this is the source of truth).
  5. Compute cheap features: language, message length, time since last message, local time, detected temporal references ("yesterday", "last Tuesday"), whether the message replies to a pending question.
- **Outputs:** `TurnContext{turn_id, user_id, msg, features}`.
- **State:** reads Session State; creates a new session if the gap exceeds the session timeout (default 30 min idle, configurable). Emits `session_started` with `gap_since_last_session`.
- **Failure:** if the lock cannot be acquired within 2 s, return a queued acknowledgement; never process concurrently.

### 3.2 Safety Pre-Gate (deterministic + small classifier)

**Responsibility:** Detect acute-risk signals *before* any generation, in parallel with context assembly, so it adds roughly zero latency.

- **Inputs:** the message, the last 3–4 turns, current User Risk State.
- **Processing:**
  - **L0 lexical/regex layer** (multilingual, versioned lists) for explicit self-harm, suicide, violence to others, abuse disclosures, and medical emergencies. High recall, low precision on purpose.
  - **L1 small classifier** (fine-tuned encoder or a small LLM with a fixed schema) producing `risk_category`, `risk_level ∈ {none, low, elevated, high, imminent}`, `confidence`.
  - Combined rule: `risk = max(L0, L1)`. Any disagreement escalates upward.
- **Output:** `SafetySignal{level, categories[], triggers[], source}`.
- **Effects:** `level ≥ high` → Orchestrator routes to the **Crisis Route**, bypassing the Brain's free choice of action. `elevated` → the Brain is told `risk=elevated` and the Output Validator tightens (see §9).
- **Failure:** fail **closed to elevated** if the classifier times out (not to none).

### 3.3 Context Assembler (deterministic)

**Responsibility:** Build the **Working Context** the Brain reasons over: small, structured, and the same shape every turn. This replaces the Context Planner *as an LLM stage*.

- **Always included (≈ 600–1,200 tokens):**
  - Last N messages (N = 8, token-capped).
  - **Session State summary** (current focus, last decision, pending question).
  - **Active Thread** record(s): topic, phase, known/unknown slots, active hypothesis ids + one-liners.
  - **Protocol Instance** if any (step, status, summary of the last user response).
  - **Core preferences** (response length, tone, language, "do not" list).
  - **Risk/engagement state** (flags only).
  - **Open loops**: unresolved questions, follow-ups due, things the user said they'd try.
  - **Information Needs** (computed, ≤ 2 items; see §5.3).
- **Conditionally added:** retrieved items from the Retrieval Service (≤ 5 items, each ≤ 60 tokens, with `epistemic_status` and age).
- **Output:** `WorkingContext` (typed struct). Every item carries a `source_id` so extractions and traces can reference it.
- **Dependency:** reads Session/Thread/Protocol/User state directly and memory *via* the Retrieval Service only.

### 3.4 Retrieval Service (deterministic; embeddings only)

**Responsibility:** Find the right memory with the right method per query type. It is **speculative**: it runs in parallel with the Safety Pre-Gate, before the Brain, using cheap signals.

| Retrieval strategy | Triggered by | Method |
| --- | --- | --- |
| **Exact/temporal** | Temporal expression + recall verb ("what did I tell you last Tuesday") | Parse time window with tz → SQL on `messages` by time/role; returns original messages, never summaries |
| **Thread-based** | Message belongs to (or resembles) an active thread | Pull the thread's linked memory items, last episode, active hypotheses |
| **Hybrid semantic** | Default for each message | Query = message + short thread topic; vector (pgvector) + FTS (BM25) candidates; rerank by `score = w1·similarity + w2·recency_decay + w3·importance + w4·thread_match` |
| **Anaphora/reference** | "same feeling as before", "that thing I told you" | Boost items from the last 14 days with matching emotion/topic tags; if the referent is still ambiguous, return `ambiguous_reference=true` so the Brain can clarify instead of guessing |
| **Entity/profile** | Names ("my sister Lena") | Lookup in the entity index on `memory_items.entities[]` |
| **Second hop** | Brain returns `needs_context{query, kind}` | One additional targeted retrieval, then the Brain is **re-invoked once** (hard cap: 1 hop) |

- **Output:** `RetrievalResult{items[], ambiguous_reference, strategy_used}`; items follow explicit AI-use eligibility and epistemic filters (`rejected` and `superseded` excluded by default). UI deletion flags alone do not define AI-use permission; see §7.7.
- **Anti-excess rules:** hard top-K (5), minimum score threshold, dedupe by `cluster_id`, token cap; hypotheses are retrieved only if `status ∈ {supported, proposed}` **and** relevant to the active thread.
- **No LLM.** Embedding lookup is the only model use; the user message embedding is cached on the `messages` row and reused later by the Memory Writer.

### 3.5 Turn Brain (LLM #1; small/fast tier; structured output)

**Responsibility:** One reasoning act that replaces Turn Analyzer + lightweight understanding + Context Planner request + Conversation Decision Engine + Intervention choice + Response Planning.

- **Inputs:**
  - `WorkingContext`.
  - A *static* system prompt (role, output schema, decision policy) from the Knowledge Bundle.
  - **Action catalog** (short descriptions of the 14 objectives) and **Skill catalog** (one-line descriptor per skill, ≈ 15 skills).
  - **Intervention shortlist** (0–5 eligible Protocols, pre-filtered, one-liners), present only when the router thinks interventions are possible (see §4.2).
  - **Eligibility facts** (e.g. "no protocol may be offered: risk=elevated").
- **What it must produce, in this order inside the JSON, so that reasoning precedes the verdict:**
  1. `turn_analysis`: user intent, message type, relation to the current thread (`continues | shifts | new | returns_to_paused`), emotional signals, references to the past, ambiguity list.
  2. `understanding_update`: **proposed** changes to Thread and Session state (reported facts, inferred hypotheses, unresolved questions), each marked `reported | inferred | unknown`.
  3. `needs`: `needs_context` (optional), `needs_formulation` (boolean, rare), `memory_worthy` (boolean + reason tags).
  4. `decision`: `primary_objective`, `rationale_tags` (not a free essay), `alternatives_considered` (≤ 2), `confidence: low|med|high`.
  5. `response_plan`: `skills[]` (primary + ≤ 2 secondary), `question: {allowed, purpose, max_count:1}`, `tone`, `length`, `permitted_context_ids[]`, `must_avoid[]`, `protocol_action?`.
- **Outputs:** `BrainDecision` (contract in §6.3). The Orchestrator extracts the state delta and forwards the plan.
- **Memory/state dependencies:** read-only. **It never writes.** Its `understanding_update` is a proposal for the State Committer.
- **Why one call is acceptable:** all sub-tasks consume identical context; splitting them adds latency but little decision quality. Quality is protected by (a) a schema that forces reasoning before the verdict, (b) cross-field validation rules, and (c) trace-level evaluation of each sub-field.
- **When skipped:** only in the fast path for `answer_direct` triggers and in the Crisis Route (§4).

#### 3.5.1 Brain authority and its limits

| The Brain may | The Brain may not |
| --- | --- |
| Choose any non-safety objective | Lower the risk level |
| Propose hypotheses (as `inferred`) | Create `reported` facts that are not in the user's text |
| Choose skills from the catalog | Invent a skill or Protocol id |
| Request one extra retrieval | Call other tools |
| Propose a new thread or topic shift | Start a Protocol (it may only *offer*; starting needs user consent) |
| Flag `needs_formulation` | Run a formulation itself |

### 3.6 Protocol Engine (deterministic)

**Responsibility:** Execute multi-step interventions reliably. This replaces "Intervention Selection + continuation via LLM memory".

- **Inputs:** Protocol definitions (L0), Protocol Instance state, the Brain's `protocol_action`.
- **Processing:**
  - **Eligibility filter (pre-Brain):** for each protocol, evaluate `preconditions` (needed information slots filled, risk ≤ low, no contraindication flags such as `trauma_active`, not recently declined by the user, minimum thread phase), then rank by `fit(problem tags, goal, past outcomes, preference)`. The top five go to the Brain.
  - **Step execution:** each Protocol is a small finite-state definition: `steps[] → {id, goal, instruction_for_generator, expected_user_input, success_criteria, on_success, on_stuck, on_distress, on_decline}`.
  - **Transition:** after the user's reply, the Brain returns `protocol_action ∈ {advance, repeat_step, adapt, pause, stop, complete}` plus a `step_outcome` classification. The engine validates that the transition exists in the definition; unknown transitions are replaced by `repeat_step`.
  - **Stop rules:** distress spike (`risk ≥ elevated`), explicit decline, repeated stuck (2×), or topic shift with the user saying "later".
- **Outputs:** `ProtocolStepSpec` for the Generator (step instruction, allowed adaptations) plus a Protocol Instance patch for the State Committer.
- **State owned:** `protocol_instances`.

### 3.7 Response Generator (LLM #2; strong tier; streaming)

**Responsibility:** Write the user-facing message that carries out the plan, in Theia's voice.

- **Inputs:**
  - **Static system block (cacheable):** persona, **Communication Principles** (≈ 400–600 tokens), hard prohibitions (no diagnosis, no medication advice, no certainty about inferences), formatting rules.
  - **Dynamic block:** `response_plan`; **Skill Cards** for the selected skills only (≈ 150 tokens each); the `ProtocolStepSpec` if present; `permitted_context` (only referenced items, with `epistemic_status`); the last 4–6 messages; the user's style preferences.
- **Output:** free text, streamed. No JSON on the user-facing stream. Metadata (e.g. the question asked, if any) is extracted deterministically afterward by the Output Validator.
- **Authority (bounded latitude):** it may choose wording, order, metaphors, which permitted context to reference, and adapt length. It may **not** change the primary objective, add a second question, introduce an unplanned intervention, state an inference as fact, or use context outside `permitted_context_ids`. Violations are caught by the validator, not left to the prompt.
- **Escape hatch:** it may output the sentinel `<<PLAN_UNSAFE>>` if the plan conflicts with what the user just said (e.g. the plan says "ask a question" but the message reads as an emergency). The Orchestrator then re-routes through the Crisis or `reflect` fallback. Rare, logged, and a useful quality signal.

### 3.8 Output Validator (deterministic rules + optional small classifier)

**Responsibility:** Gate the generated text during and after generation.

- **Rule checks (cheap, run on the stream):** banned patterns (diagnostic claims like "you have depression", dosage advice, guaranteeing outcomes, "as your therapist"); more than one question; length outside policy; leakage of internal ids/schema; unplanned URLs/phone numbers (only whitelisted crisis resources may appear).
- **Classifier checks (post-hoc, ≈ 50–150 ms):** scores for `invalidating`, `overly_directive`, `certainty_about_inference`, `dependency_encouraging`.
- **Streaming policy:** for `risk ≤ low`, stream tokens while the rule layer monitors; on violation, cut the stream and regenerate. For `risk ≥ elevated`, **buffer the full response** before display (adds ≈ 1–2 s, worth it).
- **On failure:** one regeneration with the violation reason appended; a second failure → **templated safe fallback** (a reflective, low-risk response keyed to the plan's objective).
- **Output:** `ValidationReport{passed, violations[], extracted{question_text, claims_about_user[]}}`. `claims_about_user` feeds Memory Writer provenance and the check that hypotheses are not stated as facts.

### 3.9 State Committer (deterministic, synchronous, transactional)

**Responsibility:** Apply the validated state delta in one DB transaction as part of completing the turn.

- **Inputs:** Brain `understanding_update` and `protocol_action`, Protocol Engine patch, Validator `extracted.question_text`, Safety signal.
- **Validation:** schema check; allowed transitions (Thread FSM, Protocol FSM, Risk FSM); optimistic concurrency via a `version` column (mismatch → reload and reapply, or drop with a trace record).
- **Writes:** `session_state`, `threads`, `protocol_instances`, `user_state` (risk/engagement counters), `pending_question`, and `turn_events` (an append-only event log, the basis for rebuild).
- **Transaction failure:** the reply is still delivered (the user must not suffer for a backend glitch), the turn is flagged `state_uncommitted`, and the next turn's Gateway runs **state reconciliation** (§8.7) from `messages` + `turn_events`.

### 3.10 Memory Writer (LLM #3; async; small tier)

**Responsibility:** Convert a finished turn (or session segment) into **candidate memory operations**; see §7.3 for the lifecycle.

- **Trigger:** after the reply is delivered, if `memory_worthy` was flagged, or every Nth turn, or on session close.
- **Inputs:** the turn(s), the Brain's `turn_analysis`, existing items near the candidates (via embedding similarity, ≤ 5), `claims_about_user`.
- **Outputs:** `MemoryOpProposal[]` (ADD / UPDATE / SUPERSEDE / CONTRADICT / CONFIRM / NOOP) with `kind`, `epistemic_status`, `evidence_msg_ids`, `sensitivity`, `importance`.
- **No direct writes.**

### 3.11 Memory Committer (deterministic; async)

**Responsibility:** Validate and apply proposals: enforce provenance, deduplicate, merge, supersede, set status, cap growth, apply user deletion. Details in §7.

### 3.12 Reflection Worker (LLM #4; async; strong tier)

**Responsibility:** The slow-thinking layer. It builds and revises **Case Formulations** and consolidates episodes.

- **Triggers:** (a) session closed; (b) Brain `needs_formulation=true` with sufficient evidence; (c) a thread reaches N turns or M distinct episodes; (d) contradictions accumulate on an active hypothesis; (e) nightly maintenance for stale items.
- **Inputs:** the thread, its linked memory items and evidence messages, the previous formulation version, the **Formulation Framework** schema (5P-lite or CBT conceptualization, chosen by the thread's problem tags).
- **Processing:** produce a *new version* of the formulation and diff it against the previous one. For each hypothesis compute `supporting_evidence_count`, `contradicting_evidence_count`, and independence (distinct sessions/dates), then update `status` through the **Hypothesis FSM** (§7.5). Consolidate session transcripts into an **Episode** with references. Propose Thread phase changes.
- **Outputs:** `ReflectionResult{formulation_version, hypothesis_ops[], episode, thread_proposals[], information_gaps[]}`, committed through the Memory/State Committers like everything else.
- **Latency:** none (async). **Cost control:** runs on closed sessions with a per-user daily rate limit.

### 3.13 Scheduler / Proactive Engine (deterministic)

**Responsibility:** Own time. Creates timers from Thread/Protocol records ("review the thought record in 48 h", "check in on Friday") and emits `WakeUp` events. A proactive turn goes through the *same* pipeline with a synthetic input (`reason=follow_up`), so safety, policy, and tracing are identical. User-level notification caps and quiet hours are enforced here, and users can disable proactive messages entirely.

### 3.14 Turn Orchestrator (deterministic)

See §4.1 and §6.

### 3.15 Trace Logger and Knowledge Bundle Registry

- **Trace Logger:** writes one `turn_trace` per turn: input hashes, context item ids, safety signal, Brain input/output, selected skills/protocol, validator report, latencies, token counts, model + prompt + bundle versions, errors and fallbacks. Stored under the same privacy controls as messages (§9.6).
- **Knowledge Bundle Registry:** loads an immutable, versioned bundle (`kb@x.y.z`) with prompts, communication policy, skill cards, Protocol definitions, schemas, crisis resources, and thresholds. Each turn pins one version, enabling reproducible replay and gradual rollout.

---

## 4. Execution Pipeline

### 4.1 Technical orchestration vs. conversational decision-making

These are deliberately two different things:

|  | **Turn Orchestrator** (technical) | **Turn Brain** (conversational) |
| --- | --- | --- |
| Question answered | *Which module runs next, and what happens if it fails?* | *What should Theia do for the user next?* |
| Implemented as | Deterministic Go code: a small route table and a DAG executor | A structured LLM judgment under a decision policy |
| Decides | Route (fast / normal / crisis / proactive), parallelism, timeouts, retries, fallbacks, budgets, locks | Primary objective, skills, question allowance, intervention offer, thread relation |
| Cannot do | Choose psychological content | Control execution, lower risk, bypass validation |
| Tested by | Unit/integration tests, fault injection | Scenario evals, clinician-labelled decisions |

The Orchestrator is a **route-selecting DAG executor**, not a general agent loop. Routes are enumerated (4), steps are typed, and the only loop is the capped retrieval hop. This is what makes the system testable and its latency predictable.

```go
// Pseudocode, not production code
func (o *Orchestrator) ProcessTurn(ctx, req) stream {
    t := o.gateway.Admit(req)               // idempotency, lock, persist msg, session
    par(                                    // parallel, with a 400ms budget
        safety  = o.safety.PreGate(t),
        wctx    = o.assembler.Build(t),     // state + preferences + open loops
        retr    = o.retrieval.Speculative(t),
    )
    route := o.router.Select(t, safety, wctx, retr)
    switch route {
    case CRISIS:     plan = o.crisis.Plan(t, safety, wctx)
    case FAST:       plan = o.fastPlan(t, wctx)              // no Brain call
    case NORMAL:     plan = o.runBrain(t, wctx, retr)        // may do 1 extra hop
    case PROACTIVE:  plan = o.runBrain(t.asWakeUp(), wctx, retr)
    }
    plan  = o.policyGate(plan, safety, wctx)   // risk monotonicity, consent, budgets
    spec := o.protocols.Resolve(plan, wctx)    // deterministic step spec if relevant
    text := o.generator.Stream(plan, spec, wctx) // validator monitors stream
    report := o.validator.Finalize(text, plan)
    if !report.Passed { text = o.regenerateOnce(...) ?: o.safeFallback(plan) }
    o.state.Commit(t, plan, spec, report, safety) // sync txn
    o.outbox.Emit(turn_completed)
    go o.async.Enqueue(MemoryWrite, ReflectionMaybe, Trace)  // off critical path
    return text
}
```

### 4.2 The Router (deterministic rules)

The router chooses the route and which optional inputs the Brain gets. Rules are evaluated top-down; first match wins.

| Order | Condition | Route | Notes |
| --- | --- | --- | --- |
| 1 | `safety.level ≥ high` | **CRISIS** | Brain bypassed for objective selection |
| 2 | Active **crisis follow-up state** (risk flagged in the last 24 h, not cleared) | NORMAL with `risk=elevated` constraints | See Risk FSM |
| 3 | WakeUp event from the Scheduler | PROACTIVE | Synthetic input |
| 4 | Message matches **trivial pattern** (greeting, thanks, "ok", "hi") **and** no pending question **and** no active Protocol **and** `safety ≤ low` | **FAST** | Uses a canned plan `{objective: answer_direct/reflect_light}`; still goes through Generator and Validator |
| 5 | Everything else | **NORMAL** |  |

**Whether interventions are "possible" this turn** (decides if the Brain sees a shortlist): Yes if (a) an active Protocol exists, or (b) the thread is in phase `understanding_sufficient|working`, risk ≤ low, no recent Protocol decline in the past N days, and the user's message shows an *action cue* (asking "what can I do", reporting a pattern the thread already tracks, or a goal stated). Otherwise the shortlist is omitted, and the Brain physically cannot select `offer_intervention`. This is a cheap, deterministic way to enforce the "don't rush to solutions" principle.

### 4.3 Step-by-step: what happens on a normal message

| # | Step | Module | Sync? | LLM? | Mandatory? |
| --- | --- | --- | --- | --- | --- |
| 1 | Admit, dedupe, lock, persist, session check | Gateway | Sync | No | Yes |
| 2 | Safety Pre-Gate (L0 + L1) | Safety | Sync, parallel | Small model | Yes |
| 3 | Working Context build | Assembler | Sync, parallel | No | Yes |
| 4 | Speculative retrieval | Retrieval | Sync, parallel | Embedding only | Yes (cheap; may return 0 items) |
| 5 | Route selection and shortlist filtering | Router + Protocol Engine | Sync | No | Yes |
| 6 | **Brain call** | Turn Brain | Sync | **Yes** | Normal route only |
| 6b | Optional second retrieval hop → Brain re-run | Retrieval + Brain | Sync | Yes (rare, ≈ 5%) | Conditional |
| 7 | Policy gate on the plan (risk monotonicity, consent, question budget, thread-transition validity) | Orchestrator | Sync | No | Yes |
| 8 | Protocol step resolution | Protocol Engine | Sync | No | Conditional |
| 9 | **Generate** (stream) | Response Generator | Sync | **Yes** | Yes |
| 10 | Validate (rules on stream, classifier post-hoc) | Output Validator | Sync | Mostly no | Yes |
| 11 | State commit | State Committer | Sync txn | No | Yes |
| 12 | Memory Writer / Committer | Memory | Async | Small model | Conditional |
| 13 | Reflection triggers | Reflection Worker | Async | Strong model | Conditional |
| 14 | Trace write, eval sampling, scheduler timers | Trace / Scheduler | Async | No | Yes |

**Critical path = steps 1–11.** Steps 2, 3, 4 run in parallel, so the pre-Brain latency is ≈ the slowest of the three (≤ 400 ms target). Step 6 ≈ 0.8–1.5 s on a small model with a ≈ 2.5 k-token input and ≈ 400-token output. Step 9 time-to-first-token ≈ 0.8–1.5 s.

### 4.4 Scenario routing matrix

| Scenario | Route | Executed | Skipped | Why |
| --- | --- | --- | --- | --- |
| **A. Simple message** ("hey, I'm back") | FAST | Gateway, Safety, Assembler, minimal Retrieval, Generator (canned plan), Validator, State commit | Brain, Protocol Engine, Reflection | No decision needed. Uses open-loop reminders from the Assembler (e.g. a due exercise review can ride along as a gentle option) |
| **B. Emotional message, empathy first** | NORMAL | All of the NORMAL path; shortlist **omitted** | Protocol Engine, 2nd retrieval hop | The Brain picks `reflect` or `validate_and_support`; no interventions are exposed |
| **C. References an earlier conversation** | NORMAL | Retrieval anaphora strategy; Brain gets items; possibly hop 2 if ambiguous | Protocol Engine unless active | If `ambiguous_reference` the Brain chooses `clarify` rather than guessing |
| **D. Exact recall** ("what did I say last Tuesday?") | NORMAL, with **exact-retrieval** strategy | Temporal parse → raw messages → Brain `answer_direct` with literal quotes in `permitted_context` | Hybrid semantic ranking | The answer must be sourced from the original messages; a summary is not acceptable |
| **E. Problem needing deeper analysis** | NORMAL, then **async Reflection** | Brain proposes inferred hypotheses and `explore`/`clarify`; Reflection Worker builds the formulation after the session or when evidence thresholds are met | In-path formulation (unless a formulation exists and a hypothesis is to be offered) | Avoids slow, thin-evidence formulations in the response path |
| **F. Request for practical guidance or a technique** | NORMAL, shortlist **included** | Eligibility filter → Brain `provide_guidance` or `offer_intervention` | — | If an intervention is offered, it stays `offered` until the user says yes |
| **G. Continuing an exercise** | NORMAL | Protocol Engine step resolution; Brain evaluates `step_outcome`; Generator renders the step | Shortlist (a Protocol is already active) | State is in `protocol_instances`, not in LLM memory |
| **H. Sudden topic change** | NORMAL | Brain sets \`thread\_relation=shifts | new` ; State Committer pauses the thread, opens or resumes another; open Protocol gets  `pause\` (with a one-line check: "want to pick that up later?") | Protocol advance |
| **I. Potential crisis signals** | CRISIS | Safety, Crisis Planner, constrained Generator, buffered Validator, Risk FSM update, outbox event | Brain (for objective), shortlist, retrieval beyond minimal | See §9 |
| **J. User correction** ("I never said that") | NORMAL | Brain `repair`; Retrieval pulls the contested item by id; **immediate** synchronous *correction* operation by Memory Committer (see §7.4) | Protocol Engine | A user correction is the one memory write that is not deferred, so the next sentence of the reply cannot rely on the wrong item |

### 4.5 Parallelizable vs. mergeable vs. strictly ordered

- **Parallel:** Safety Pre-Gate ∥ Context Assembler ∥ Retrieval. Post-reply: Memory Writer ∥ Trace ∥ Scheduler registration.
- **Strictly ordered:** Brain → policy gate → Protocol resolution → Generator → Validator → State commit.
- **Mergeable if needed (cost/latency fallback):** Brain and Generator can be merged into one call that outputs a short JSON plan followed by the reply (the "plan-then-write" pattern). It saves ≈ 1 s and ≈ 25% tokens, but loses (a) the policy gate between plan and text, (b) the ability to use a cheap model for planning, and (c) clean evaluation of the decision. **Recommendation:** start separated for the MVP so you can measure decision quality; merge only for FAST-like low-stakes classes if eval shows no quality drop.
- **Never merged:** Safety Pre-Gate (independent control), Memory Writer (must not share a context with response generation: different goal, and async).

### 4.6 Must-complete before replying vs. deferrable

| Must complete before the reply is considered done | Deferrable |
| --- | --- |
| Message persisted | Memory extraction (Memory Writer) |
| Safety signal computed | Deduplication/merge/supersede |
| Brain decision validated | Episode consolidation |
| Generation validated | Formulation / hypothesis updates |
| State transaction committed (with fail-open reconcile path) | Embedding of new memory items |
| **Corrections** applied (user says a stored fact is wrong) | Scheduler timers |
| Risk state updated | Trace enrichment, eval sampling, analytics |

### 4.7 Failure handling on the critical path

| Failure | Behavior |
| --- | --- |
| Safety classifier timeout | Treat as `elevated` for this turn; buffer output |
| Retrieval error/timeout (>300 ms) | Proceed with state-only context; mark `retrieval_degraded`; the Brain is told memory is unavailable (so it should not pretend to remember) |
| Brain timeout/invalid JSON | One repair retry (cheaper, with a smaller schema); then **default plan** `{objective: reflect, question: allowed, skills:[reflect_emotion]}` |
| Brain proposes invalid transition | State Committer drops that part of the delta; trace logged |
| Generator error | Retry once on a fallback model; then a templated acknowledgement |
| Validator violation | Regenerate once, then templated fallback |
| State commit failure | Deliver reply, flag `state_uncommitted`, reconcile on next turn |
| Memory pipeline failure | **No effect on the conversation.** The job goes into a retry queue with backoff; items stay in `turn_events` and are re-extractable from `messages` |
| Provider outage | Switch model tier; if all fail, a static message acknowledging difficulty and, if the last known risk ≥ elevated, crisis resources |

---

## 5. Psychology Knowledge Architecture

### 5.1 Principle: domains are *content*, modules are *consumers*

Psychology Domains become **versioned, machine-readable knowledge assets** in the Knowledge Bundle. Modules consume them in specific, bounded ways. A domain never equals a module.

### 5.2 Re-evaluation of the eight domains

| # | Preliminary domain | Verdict | Final form |
| --- | --- | --- | --- |
| 1 | Communication principles & skills | **Keep**, split | (a) **Communication Policy** (static, always on) and (b) **Skill Catalog** (dynamic, selected per turn) |
| 2 | User Understanding | **Keep as schema + policy** | **User Information Schema** + **Information Needs Policy** |
| 3 | Problem Understanding & Case Formulation | **Keep, split into two asset types** | **Problem Understanding Schema** (light, in-path) and **Formulation Frameworks** (deep, async) |
| 4 | Conversation Management | **Keep** | **Decision Policy** (compact rules + objective catalog + cheat-sheet of decision heuristics); lives in the Brain's static prompt |
| 5 | Psychological Interventions | **Keep** | **Protocol Library** (declarative; §3.6) |
| 6 | Longitudinal Support | **Dissolve** into policies | **Continuity Policy** governing Thread lifecycle, resumption, follow-ups, and memory revision (consumed by Scheduler, State Committer, Reflection Worker) |
| 7 | Safety & Ethics | **Keep, make cross-cutting** | **Safety Policy** + **Crisis Resource Table** + **Boundary Policy** (diagnosis/medication/dependency) |
| 8 | Evaluation | **Keep as non-runtime** | **Evaluation Rubrics** and scenario suites; also used to train/calibrate classifiers |

Two missing domains worth adding:

- **Boundary & Dependency Policy** (part of Safety) — scripted handling of diagnostic requests, medication questions, "you're the only one who understands me", requests to replace therapy.
- **Cultural/Contextual Sensitivity** — language, tone, and framing adaptation across cultures; for the MVP this is a small section of the Communication Policy plus locale tables.

### 5.3 Domain → asset → module map

| Asset | Form | Consumed by | How it is injected |
| --- | --- | --- | --- |
| Communication Policy (principles) | ≈ 10–12 rules, ≤ 600 tokens, plain text | **Generator** (always, in the cacheable system block); **Brain** (only a 3-line digest) | Static |
| Skill Catalog (OARS, validation types, clarification, tentative interpretation, psychoeducation, summarize, etc.) | Each skill = a **Skill Card**: `id, purpose, when_to_use, when_not_to_use, form_constraints, good_example, bad_example` | **Brain** sees *one-line descriptors* of all skills; **Generator** receives *full cards* only for selected ones | Dynamic, by ids |
| Decision Policy | Objective catalog, precedence rules (see below), tie-breakers | **Brain** | Static in system prompt |
| User Information Schema | Field definitions with `purpose, sensitivity, collection_condition, update_rule, ask_cost` | **Context Assembler** (computes *Information Needs*), **Memory Writer** (extraction target), **Brain** (receives only the top gaps) | Computed, not injected wholesale |
| Problem Understanding Schema | Thread slots: `situation, emotions, thoughts, behaviors, goal, history, impact, unknowns` | **Brain** (updates), **State Committer** (validates), **Context Assembler** (renders) | Structural |
| Formulation Frameworks | 5P-lite and CBT conceptualization templates with field definitions, minimum-evidence rules, wording rules | **Reflection Worker** only (+ Brain receives a *rendered one-paragraph* summary of the active formulation) | Dynamic, by thread problem tags |
| Protocol Library | Declarative Protocol definitions | **Protocol Engine** (execution), **Brain** (one-liners for the shortlist), **Generator** (step instruction only) | Dynamic |
| Safety Policy | Risk taxonomy, level definitions, escalation/de-escalation rules, crisis response scripts | **Safety Pre-Gate** (rules + classifier labels), **Orchestrator Router**, **Crisis Planner**, **Output Validator** | Rules in code; scripts in bundle |
| Boundary Policy | Trigger patterns + scripted response modes | **Brain** (`boundary` objective), **Output Validator** (banned patterns) | Static + rules |
| Crisis Resource Table | `locale → resources[]` (curated, dated, verified) | **Crisis Planner**, **Output Validator** (whitelist) | Direct lookup; never LLM-generated |
| Continuity Policy | Thread lifecycle rules, staleness windows, follow-up intervals, hypothesis revision rules | **Scheduler**, **State Committer**, **Reflection Worker** | Code + config |
| Evaluation Rubrics | Scoring dimensions per level (turn, decision, longitudinal) | **Eval Harness**, **Output Validator classifier training** | Offline |

### 5.4 Answering the preliminary Communication questions

| Question | Decision |
| --- | --- |
| Which component decides the communication skill? | The **Brain**, because skill choice depends on the same judgment as the objective (emotional state, readiness, ambiguity). It chooses from the catalog; it does not write skill text. |
| Should the Generator pick skills itself? | **No for the primary skill, yes for composition.** The primary skill and up to two secondary skills are fixed in the plan. The Generator composes them into natural prose. This is the only way to keep one primary objective while blending skills, and to test it. |
| Should communication principles be in every LLM call? | In the **Generator: always** (cacheable, small). In the **Brain: digest only**. In the Memory Writer and Reflection Worker: **not at all** (different tasks). |
| Is "respond according to psychological principles" enough? | No. Broad instructions produce generic, over-validating, advice-heavy output. Use **explicit, structured, context-selected** instructions (objective + skills + constraints), but keep the *principles* short so the Generator can still sound human. |
| How much flexibility should the Generator have? | Bounded latitude (see §3.7): freedom of phrasing and structure; no freedom over objective, question count, new claims about the user, or interventions. |

### 5.5 Decision Policy: concrete precedence rules

The Decision Policy should be a short set of **ordered rules with tie-breakers**, not a long essay. For example (illustrative, to be clinically reviewed):

1. Safety signals override everything (already enforced outside the Brain).
2. **Repair** first: if the user disputes, corrects, or expresses feeling misunderstood, objective = `repair`.
3. **Active Protocol**: if a Protocol is active and the message is relevant to it, objective = `continue_intervention`, unless distress/decline signals → `close_or_pause_intervention`.
4. **Direct question** with sufficient information → `answer_direct` / `provide_guidance`; ask a question only if the *answer would change materially* depending on a missing slot.
5. **Unprocessed emotion** (high affect, no request) → `reflect` / `validate_and_support`; no advice, no interventions.
6. **Ambiguity blocking understanding** → `clarify` (one question, high-value slot first).
7. **Understanding sufficient and user shows readiness** → `provide_insight` (tentative, hypothesis worded as such) or `offer_intervention`.
8. **Consecutive-question limit:** at most 2 consecutive turns ending in a question; the third must be reflection/summary.
9. **Default** → `explore`, with one open question or a reflection.

### 5.6 Information Needs (replaces "User Understanding module")

Computed deterministically before each Brain call:

```
information_needs = []
for slot in Thread.schema.slots where slot.status == unknown:
    value = slot.importance_for(thread.phase)        # e.g. 'situation' high at exploring
    cost  = slot.ask_cost(user.sensitivity_prefs)    # e.g. trauma history = high cost
    if value >= V_MIN and cost <= user.comfort_budget and not recently_asked(slot):
        information_needs.append((slot, value - cost))
return top2(sorted_by_score)
```

Rules enforced:

- Needs are *suggestions*; the Brain may ignore them if the user is in distress.
- `recently_asked` and `declined_to_share` flags prevent repetition and pressure.
- General user information (onboarding) is **never mandatory**; each field has `collection_condition` (e.g., ask about occupation only if work is in the active thread).
- Sensitive fields (trauma, medication, sexuality, finances) have `collect_proactively=false`: Theia stores them only when volunteered.
- Every field has `source` ∈ `{user_stated, onboarding_form, inferred}` and a `last_confirmed_at`.
- Screening instruments (PHQ-9, GAD-7) later plug in as **Protocols of type `assessment`** with their own eligibility, consent, and scoring-handling rules; no architectural change needed.

---

## 6. LLM & Prompt Orchestration

### 6.1 Where LLM calls are genuinely necessary

| Responsibility | LLM? | Rationale |
| --- | --- | --- |
| Idempotency, locking, session logic | No | Deterministic |
| Lexical risk detection | No | Rules, high recall |
| Risk classification | Small model | Needs semantics; must be fast and independently versioned |
| Context assembly, retrieval, router | No | Deterministic; embeddings only |
| Understanding + decision + planning | **Yes (Brain)** | Genuine judgment over ambiguous language |
| Protocol eligibility, transitions, step lookup | No | Declarative definitions |
| Judging a step outcome ("did the user complete the thought record?") | **Yes (inside the Brain call)** | Semantic |
| Final wording | **Yes (Generator)** | Core quality driver |
| Output policy checks | Rules + small classifier | Cheap and independent |
| Memory extraction | **Yes (Memory Writer, async)** | Semantic; off the critical path |
| Dedup/merge/supersede | No (embedding similarity + rules) | Deterministic and auditable |
| Formulation, hypothesis revision, episode consolidation | **Yes (Reflection Worker, async)** | Deep reasoning, slow, low frequency |
| Evaluation judgments | Yes (offline) | Calibrated against clinicians |

### 6.2 Call inventory and budgets

| Call | Tier | Typical input | Output | Frequency | Target latency |
| --- | --- | --- | --- | --- | --- |
| Safety classifier | Small/encoder | ≈ 300 tokens | label JSON | Every turn | < 200 ms (parallel) |
| **Turn Brain** | Small/fast | 2–3 k | 300–600 tokens JSON | Normal/proactive turns (≈ 85%) | 0.8–1.5 s |
| **Response Generator** | Strong | 2–4 k (≈ 60% cacheable) | 80–250 tokens | Every turn | TTFT 0.8–1.5 s |
| Validator classifier | Small | Reply only | scores | Every turn (post-hoc; parallel with stream end) | < 150 ms |
| Memory Writer | Small | 1.5–3 k | ≤ 500 tokens | ≈ 30–50% of turns | Async |
| Reflection Worker | Strong | 5–12 k | 800–1,500 tokens | Per session close / triggers | Async |

**Per-turn synchronous cost:** ≈ 1 small + 1 strong generation + 2 tiny classifiers. Cost control levers: prompt caching of static blocks, FAST route, skill-card selection, retrieval top-K, per-user daily caps for Reflection, and a **model-tier policy** configurable per call so providers can be swapped.

### 6.3 The Turn Brain contract (illustrative)

```json
{
  "schema_version": "brain.v1",
  "turn_analysis": {
    "intent": "share_distress",
    "message_type": "disclosure",
    "thread_relation": "continues",
    "emotions": [{"label": "worthlessness", "intensity": "high"}],
    "past_references": [{"phrase": "same feeling as before", "resolved_to": ["mem_81f"], "ambiguous": false}],
    "answered_pending_question": false,
    "ambiguities": []
  },
  "understanding_update": {
    "thread_id": "thr_12",
    "slots": [
      {"slot": "situation", "value": "friend discussed achievements", "status": "reported", "evidence": ["msg_9921"]},
      {"slot": "emotion", "value": "felt completely worthless", "status": "reported", "evidence": ["msg_9921"]}
    ],
    "hypotheses": [
      {"id": "hyp_7", "op": "support", "statement_ref": "hyp_7", "evidence": ["msg_9921"], "note_tag": "social_comparison"}
    ],
    "unknowns": ["what the user told themselves in the moment"]
  },
  "needs": {
    "needs_context": null,
    "needs_formulation": false,
    "memory_worthy": {"value": true, "tags": ["recurrence", "hypothesis_support"]}
  },
  "decision": {
    "primary_objective": "reflect",
    "rationale_tags": ["high_affect", "no_request_for_advice", "understanding_insufficient_for_insight"],
    "alternatives_considered": ["explore"],
    "confidence": "high"
  },
  "response_plan": {
    "skills": {"primary": "reflect_emotion", "secondary": ["normalize_gently"]},
    "question": {"allowed": true, "purpose": "invite_thought_content", "max_count": 1},
    "tone": "warm_steady",
    "length": "short",
    "permitted_context_ids": ["mem_81f"],
    "must_avoid": ["advice", "stating_hypothesis_as_fact", "intervention_offer"],
    "protocol_action": null
  }
}
```

**Design notes**

- Free-text fields are minimized and enumerations are used for anything the code branches on, so decisions are machine-evaluable.
- Hypotheses are referenced by id in the Brain output; only the *Memory/Reflection* pipeline writes hypothesis text. The Brain can only `support`, `contradict`, or `propose` (with a short tag) and the Memory Writer authors the statement from evidence.
- `evidence` must reference message ids present in the Working Context; unknown ids invalidate the field.

### 6.4 Validation of structured outputs

Every LLM output is validated in layers before use:

1. **Syntactic:** provider-native structured output / JSON-schema constrained decoding; on failure, one repair attempt.
2. **Semantic (enums & references):** objective ∈ catalog; skill ids exist; Protocol id ∈ shortlist; evidence ids ∈ context; slot names ∈ schema.
3. **Cross-field rules:** `offer_intervention` requires shortlist non-empty; `continue_intervention` requires active Protocol; `question.allowed=true` unless consecutive-question limit hit; `reflect` + `must_avoid` includes advice; `clarify` requires non-empty `ambiguities` or an Information Need; `safety_support` may not be chosen by the Brain except when it echoes the router's route.
4. **Policy gate:** risk monotonicity, consent invariant, thread FSM transitions, token/length caps.
5. **Consistency checks:** e.g. `reported` slot whose evidence message does not contain lexical/semantic support (cheap embedding/entailment check) is downgraded to `inferred`.

On failure at layers 2–4 the Orchestrator *repairs deterministically* (drop the offending field, clamp to default) rather than re-calling the LLM, whenever possible.

### 6.5 Static vs. dynamic prompt assembly

| Block | Content | Static / dynamic | Cache? |
| --- | --- | --- | --- |
| Brain system | Role, output schema, Decision Policy, action catalog, skill catalog one-liners, epistemic rules | Static per bundle version | Yes |
| Brain user | Working Context (rendered from typed structs), shortlist, eligibility facts, safety signal | Dynamic | No |
| Generator system | Persona, Communication Policy, prohibitions, format rules | Static | Yes |
| Generator dynamic | Plan, skill cards, step spec, permitted context, recent messages, preferences | Dynamic | Partially (skill cards by id) |
| Memory Writer | Extraction instructions, schema, op semantics | Static | Yes |
| Reflection | Framework schema chosen by tag, evidence package | Dynamic by tags | No |

**Context rendering rules:** use consistent, tagged sections; mark retrieved/stored text as data (`<memory id=... status=...> … </memory>`); never interleave stored text with instructions; include `epistemic_status` and age on each memory item so the model can phrase tentatively; cap per-item tokens.

### 6.6 Generator authority and output checks

The Generator receives *constraints, not a script*. The Validator checks conformance after the fact:

| Constraint | Check |
| --- | --- |
| One question maximum | Count `?` and interrogative patterns in the final sentence(s) |
| No advice when `must_avoid: advice` | Classifier (`directive_advice`) + imperative-pattern rules |
| Hypotheses stated tentatively | Rule: items with `status=inferred` must co-occur with hedging markers *or* be absent; classifier `certainty_about_inference` |
| Only permitted context | Entity/claim extraction on output vs. `permitted_context` (cheap NER/keyword match) |
| Length and format | Deterministic |
| No new intervention | Detect technique names/step-like patterns when the plan has no Protocol |

### 6.7 Prompt and policy versioning

- All prompts, skill cards, protocols, schemas, thresholds are files in a repo (`knowledge/`), semantically versioned, validated by a **linter in CI** (schema conformance, referenced ids exist, token budgets).
- Release = tagged **Knowledge Bundle**; runtime pins a bundle per request; rollout by percentage/user cohort via feature flags; instant rollback by pointer change.
- Each `turn_trace` stores `{bundle_version, model_ids, prompt_hashes, router_rules_version, classifier_version}`.
- **Clinical review workflow:** changes to Protocols, Crisis scripts, Communication Policy, and Boundary Policy require a recorded clinician approval before bundle promotion.

---

## 7. Memory Architecture

### 7.1 Physical model: three structures, six logical categories

| Logical category | Physical home | Notes |
| --- | --- | --- |
| 1. Raw conversation history | `messages` table (+ embedding, FTS index) | Source of truth; user deletion changes visibility metadata only |
| 2. Episodic memory | `memory_items` (`kind=episode`) | Summary + message-range references |
| 3. User facts & preferences | `memory_items` (\`kind=fact | preference` ,  `epistemic\_status=reported\`) |
| 4. Inferred memory | `memory_items` (`kind=hypothesis`, `epistemic_status=inferred`) | Status FSM |
| 5. Active threads & plans | `threads`, `protocol_instances`, `plans/follow_ups` | These are **state** that also act as memory; owned by the State Committer |
| 6. Working context | Not stored (an in-memory struct per turn); retained in the trace | Ephemeral |

Single table for categories 2–4 is deliberate: they share provenance, sensitivity, deletion, retrieval, and versioning behavior, and splitting them triples the plumbing.

### 7.2 `memory_items` schema (core fields)

```
memory_items
  id, user_id, kind            -- episode | fact | preference | hypothesis | goal | event
  content                      -- short canonical text, user-language
  epistemic_status             -- reported | inferred | system_observed
  status                       -- active | contested | superseded | rejected | stale (visibility is separate)
  confidence_class             -- low | medium | high   (derived from evidence counts, not LLM self-rating)
  evidence[]                   -- [{msg_id | item_id, role: supports|contradicts, ts}]
  support_count, contradict_count, distinct_sessions
  thread_ids[], entities[], tags[]
  sensitivity                  -- normal | sensitive | highly_sensitive
  importance                   -- 0..1
  valid_from, valid_to         -- time the information describes (not the write time)
  created_at, updated_at, last_confirmed_at, last_used_at
  deleted_at, deleted_by, deletion_reason -- visibility metadata, never AI-use consent
  supersedes_id, superseded_by_id, cluster_id
  source_prompt_version, source_model
  embedding
```

**Invariants** (enforced by the Memory Committer): evidence is non-empty; `reported` requires an evidence message from `role=user` (or an `explicit_confirmation` event); `inferred` can never be rendered to the Generator without its status; a `hypothesis` cannot be written as a `fact`.

### 7.3 Lifecycle: from message to memory

```
user/assistant turn ─► (A) Selection ─► (B) Extraction ─► (C) Reconciliation ─► (D) Commit ─► (E) Maintenance
```

**A. Selection (what is worth keeping).**

- *Store:* stable user-stated facts and preferences; recurring emotional/behavioral patterns; goals and commitments; significant events; protocol outcomes; user feedback on what helped or did not; corrections.
- *Do not store:* small talk; verbatim crisis details beyond what the safety record needs; third-party personal data beyond what is needed to understand the user's situation (store roles and relations like "sister", not full names, unless the user uses them repeatedly); highly sensitive categories unless volunteered and relevant (and then flagged `highly_sensitive`, excluded from default retrieval and from proactive messages).
- *Gate:* the Brain's `memory_worthy` flag (cheap) plus periodic sweeps; the Memory Writer can still emit `NOOP`.

**B. Extraction (Memory Writer).** Produces typed proposals with evidence message ids. Hypotheses are *proposed* with a tag; wording must be cautious ("may associate…"). Candidate items are matched against existing neighbors to decide `ADD / UPDATE / CONFIRM / CONTRADICT / SUPERSEDE / NOOP`.

**C. Reconciliation (Memory Committer, deterministic).**

| Situation | Action |
| --- | --- |
| Near-duplicate (cosine > τ and same kind and compatible entities) | `CONFIRM`: attach new evidence, bump `last_confirmed_at`, `support_count` |
| Same slot, changed value ("I'm a student" → "I just graduated") | `SUPERSEDE`: old item `superseded`, `valid_to` set; new item links `supersedes_id`; both retained for provenance |
| Contradicts a **fact** reported earlier | Mark both `contested` if the user did not clearly state the change; ask for confirmation at a natural moment (an *open loop* is created); if clearly stated, `SUPERSEDE` |
| Contradicts a **hypothesis** | Add `contradicts` evidence; FSM decides status (§7.5) |
| Low importance and aged (> 90 d, no use) | `stale`; excluded from default retrieval, retained; no purge is authorized in this phase |

**D. Commit.** One transaction per batch of ops; embeddings computed after commit (retry queue); idempotent by `(turn_id, op_hash)`.

**E. Maintenance (nightly/on session close).** Episode consolidation (many turns → one episode with references); cluster merging; staleness decay; contradiction sweeps; **retrieval-feedback loop** (items often retrieved but never referenced lose importance); size caps per user (hot set ≤ N active items).

### 7.4 Corrections, contradictions, and the "repair" path

When the Brain returns `objective=repair` with a `correction` target:

1. The Orchestrator performs a **synchronous** `CorrectionOp` before generation: the contested item is set to `contested` (not deleted) and excluded from this turn's permitted context.
2. The Generator acknowledges without defensiveness and states what will change.
3. After the user's clarification, a `SUPERSEDE` or `REJECT` completes. If the user says "forget that" → **soft-delete visibility path** (§7.7); AI-use controls require their own explicit policy.
4. If the item was a hypothesis, the status becomes `rejected` with `rejected_by=user`, and the **Reflection Worker may not re-propose a semantically similar hypothesis for K days** unless new, independent evidence arrives (prevents "zombie hypotheses").

### 7.5 Hypothesis lifecycle (FSM)

```
 proposed ──(≥2 independent supporting episodes, 0 strong contradictions)──► supported
    │                                                                   │
    │ (contradict ≥ support, or user disputes)                          │ (user explicitly endorses)
    ▼                                                                   ▼
 contested ◄───────────────────────────────────────── user_endorsed (spawns a *reported* item: "user said: …")
    │
    ├──► rejected  (user rejects, or contradiction strong & sustained)
    └──► superseded (replaced by a refined hypothesis; links preserved)
  any state ──► stale (no evidence for 90+ days; excluded from default retrieval)
```

- `confidence_class` is computed from counts and independence; the LLM contributes only a proposal.
- **Surfacing rule:** only `supported` hypotheses (or `proposed` with a clinician-approved, low-risk phrase set) can be voiced to the user, tentatively and in context, and only if the objective is `provide_insight`. Voicing a hypothesis creates an open loop: "user's reaction" is recorded as new evidence (agreement, rejection, ambivalence).
- **Promotion to fact:** never automatic. The only path is user endorsement, which creates a *new reported* item referencing the original.

### 7.6 Retrieval strategy per category

| Category | Primary strategy | Fallback |
| --- | --- | --- |
| Raw messages | Time/role SQL; FTS for quotes; vector on message embeddings | — |
| Episodes | Thread link + vector over episode summaries; recency | FTS |
| Facts/preferences | **Always-on core set** (preferences) in the context; others by entity/tag/vector | Profile view |
| Hypotheses | Only via active thread or strong semantic match; `status` filter | None (absence is fine) |
| Threads/plans | Direct state lookup (no search) | — |

Avoiding over-retrieval: top-K cap, similarity floor, "novelty vs. current context" filter (do not retrieve what the last 8 messages already say), and a trace-based audit of retrieved-but-unused items.

### 7.7 Privacy, user control, and deletion

All user-initiated deletion is **soft deletion**. Record a UTC deletion timestamp,
server-resolved actor and bounded reason, while retaining original content and all
references. Normal user-facing queries hide the selected resource; raw message
and bookmark views also respect the parent conversation's visibility. Repeating a
delete does not erase or overwrite the original deletion metadata.

Conversation deletion must not purge, neutralize, recursively hide, or rebuild
memories, analyses, formulations, evidence, embeddings, traces, jobs, sessions or
other derived records. Their provenance and independent visibility are preserved.
Only processing leases for the hidden conversation are expired to prevent late
publication through that resource; other conversations are unaffected. Remove
destructive purge procedures and delete cascades through new migrations, keeping
validated ownership constraints. Do not replace stored content with empty tombstones.

Retention and visibility are **not AI-use permissions**. The correction neither
grants reuse nor defines an implicit reuse ban. Explicit memory pause, correction,
consent and suppression controls remain separate and unchanged. Future retrieval
and workers must use a separately approved eligibility policy, with its own tests.
No deletion-triggered re-analysis or re-extraction is introduced here.

User controls must describe hiding and retention accurately, not promise erasure.
Future memory/account deletion endpoints must follow the same soft-only rule.
Permanent deletion, legal erasure, backup disposal and retention cleanup are a
separate future policy; no such process is implemented or authorized now.
Sensitive content still requires the existing encryption and access controls.

### 7.8 What gets evaluated or updated after each reply

| Item | Owner | When |
| --- | --- | --- |
| Session/Thread/Protocol/Risk state | State Committer | Sync, same turn |
| `pending_question` and `open_loops` | State Committer | Sync |
| Candidate facts/preferences/goals | Memory Writer → Committer | Async, if `memory_worthy` |
| Hypothesis support/contradiction counters | Memory Committer | Async |
| Episode summary | Reflection Worker | On session close |
| Formulation version | Reflection Worker | Triggers (§3.12) |
| Corrections | Memory Committer | **Sync** |
| Scheduled follow-ups | Scheduler | Async after commit |

---

## 8. State Management

### 8.1 Core idea

**State is the system's explicit model of "what is happening right now," kept separate from memory ("what we know about the user over time").** State answers *where are we?* Memory answers *what do we know?* State is small, structured, mutable under strict rules, and **rebuildable** from messages and the event log. Memory is larger, evidence-bearing, and slower-moving.

A summary of the previous conversation is *one field* inside Session State, not the state itself.

### 8.2 The four state types (and a fifth that is easy to forget)

| State | Scope / lifetime | What it holds | Changes | Owner (writes) | Readers |
| --- | --- | --- | --- | --- | --- |
| **Session State** | One interaction session (idle timeout ends it) | `focus_thread_id`, `mode` (venting / problem-solving / exercise / checking-in), `pending_question {slot, asked_at, text}`, `last_objective`, `consecutive_question_count`, rolling `session_summary`, `emotional_trajectory`, `open_loops[]`, `turn_count` | Every turn | State Committer | Assembler, Router, Brain (via context) |
| **Thread (Problem) State** | Days to months | `topic`, `phase`, `slots{situation, emotions, thoughts, behaviors, goal, history, impact}` each with \`status: reported | inferred | unknown`+ evidence,`hypothesis\_ids\[\]` ,  `formulation\_version` ,  `linked\_protocols\[\]` ,  `priority` ,  `last\_active\_at` ,  `paused\_reason\` | On Brain `understanding_update` and Reflection proposals |
| **Protocol Instance State** | One run of an intervention | `protocol_id`, `version`, `step_id`, `status` (offered / active / paused / completed / stopped / declined), `inputs_collected{}`, `attempts`, `stuck_count`, `user_consent_at`, `outcome` + `user_rating`, `follow_up_due_at` | Each protocol turn | Protocol Engine → State Committer | Brain, Generator (via step spec), Scheduler |
| **User State** | Long-term, slowly changing | `risk_level` + `risk_history`, `engagement` (session frequency, response length trends, usage at night, dependency indicators), `preferences_version`, `protocol_history` (what helped, what was declined), `comfort_budget`, `proactive_opt_in`, `consent flags` | Per-turn counters; periodic rollups | State Committer; Reflection (rollups) | Router, Policy gate, Scheduler |
| **(Pending) Consent & Privacy State** | Long-term | memory paused?, data categories allowed, deletion requests in progress | On user action | Privacy API | Memory pipeline, Retrieval filter |

**Relation to memory:** Thread State is the *live index* into memory: threads reference memory items, memory items reference thread ids. If State is lost, it can be reconstructed (§8.7). If Memory is lost, State degrades to the session summary but remains operable.

### 8.3 Formal state machines: where, and where not

| Object | FSM? | States | Reason |
| --- | --- | --- | --- |
| Protocol Instance | **Yes** | \`offered → active ↔ paused → completed | stopped |
| Thread | **Yes (lightweight)** | `exploring → understanding_sufficient → working → maintaining → resolved`, plus `paused`, `dormant` | Drives intervention eligibility and resumption |
| Risk | **Yes** | `none → low → elevated → high → imminent`; downgrade requires time + explicit rule | Safety must be monotonic and auditable |
| Hypothesis | **Yes** | See §7.5 | Prevents silent promotion |
| Session | Minimal | `open → closed` | Idle timeout / explicit close |
| Conversation flow ("what mode are we in?") | **No** | — | Free conversation is not a finite machine. Represented as a validated blackboard (`mode`, `focus`, `last_objective`) updated by the Brain's analysis |

### 8.4 Events and transitions

| Event | Source | Effects |
| --- | --- | --- |
| `session_started(gap)` | Gateway | Resumption policy (§8.6); previous session summarized if not yet closed |
| `thread_shift` | Brain `thread_relation` | Current thread → `paused` (with reason); target thread → focus; open Protocol → `paused` |
| `slot_filled` / `slot_changed` | Brain proposals | Thread slot update; Information Needs recomputed next turn |
| `readiness_reached` | Thread rules (slots sufficient + user agency signals) | Thread → `understanding_sufficient`; interventions become eligible |
| `protocol_offered` / `consented` / `declined` | Brain + user turn | Protocol FSM; decline writes `protocol_history` and a suppression window |
| \`step\_outcome(success | stuck | distress)\` |
| `risk_signal(level)` | Safety | Risk FSM; may stop Protocols; sets `risk_followup_until` |
| `user_correction` | Brain `repair` | Sync correction op in memory |
| `session_closed` | Timer | Reflection trigger; Memory consolidation; follow-up scheduling |
| `follow_up_due` | Scheduler | PROACTIVE route |
| `inactivity(7d/30d)` | Scheduler | Thread → `dormant` after 30 d; open Protocols → `paused` |

### 8.5 Handling multiple concurrent issues

- Up to **3 active threads** per user; others `paused`/`dormant`. One is the **focus thread**; the Assembler includes the focus thread in full and one-liners for the others.
- A message may relate to *several* threads; the Brain returns the primary `thread_relation`, and the State Committer links secondary threads as `related_thread_ids`.
- **Only one active Protocol at a time per user** (MVP). A new offer while one is active triggers `close_or_pause_intervention` logic first.
- A thread's `priority` rises when risk signals or high distress occur in it and falls with inactivity.

### 8.6 Returning after an interruption (resumption policy)

On `session_started`, the Assembler computes `resumption_context` from the gap:

| Gap | Behavior |
| --- | --- |
| < 30 min | Same session; no change |
| 30 min – 24 h | New session; carry `pending_question`, active Protocol, and focus thread without re-introducing; Brain told `resumed=true` |
| 1 – 7 days | Thread stays active; open loops surfaced ("last time you were going to try X"); do not assume the emotional state continues; Protocol → `paused` with a prompt to resume or revise |
| > 7 days | Threads → `dormant` candidates; offer a light re-entry check ("want to continue with X or start with how you are today?"); preferences retained, any plan re-confirmed |
| Recent risk flag | Check-in objective is prioritized (in a gentle, non-scripted form) and approved by the Risk FSM |

The **user's current message always outranks stored state**: if the first message of a new session changes topic, the Brain's `thread_relation=new|shifts` wins over resumption defaults.

### 8.7 Consistency: preventing drift between state, memory, and messages

1. **Source-of-truth hierarchy:** `messages` > `turn_events` > state tables > derived memory (summaries, hypotheses) > caches. Anything derived can be rebuilt from what is below it.
2. **Single-writer + optimistic concurrency** (§3.9) with per-user turn lock.
3. **Event log:** every state change is an append-only `turn_events` row (`turn_id`, `type`, `delta`, `actor`, `versions`). State tables are projections of this log.
4. **Reconciliation job** (on `state_uncommitted`, on session start if version gap detected, and nightly): replays recent events; checks invariants (e.g. an `active` Protocol must have a consent event; every `reported` slot must have a user-message evidence id; thread `last_active_at` equals last linked message time); repairs or flags.
5. **Memory↔state links retain provenance:** soft deletion does not remove links or downgrade slots; validate ownership and any separately approved use eligibility.
6. **Soft deletion preserves state:** deleting a conversation/message/item does not invalidate threads, Protocol inputs or formulations. It changes only the requested visibility and blocks further processing through a hidden conversation.
7. **No hidden conversational state in prompts:** anything the next turn needs (pending question, protocol step, consent) must be in tables, not inferred from the history.

---

## 9. Safety, Evaluation, and Observability

### 9.1 Safety layered through the pipeline

| Layer | Position | Mechanism | Controlled by |
| --- | --- | --- | --- |
| **S0 Intake gate** | Before any LLM | L0 lexical rules + L1 classifier → `SafetySignal` | Safety Pre-Gate |
| **S1 Context constraints** | Router/Assembler | Risk-aware eligibility (no Protocol offers at ≥ elevated; highly sensitive items excluded; proactive messages suppressed) | Deterministic |
| **S2 Brain risk reading** | In Brain call | Brain may *raise* risk with reasons (it sees nuance the classifier misses); never lower | Policy gate |
| **S3 Crisis Route** | Orchestrator | Bypasses normal planning; templated + constrained generation (below) | Safety Policy |
| **S4 Output validation** | After generation | Banned patterns, forced content for risk replies, resource whitelist | Output Validator |
| **S5 Session-level risk memory** | State | Risk FSM + follow-up obligations | State Committer |
| **S6 Human oversight** | Offline/near-real-time | Review queue, alerting, audits | Governance |

**Application logic vs. model assessment vs. human oversight**

| Control | Where |
| --- | --- |
| Crisis routing, resource lookup, risk monotonicity, consent-before-protocol, question budget, banned patterns, deletion enforcement | **Application logic** |
| Nuanced risk reading, emotional appraisal, ambiguity, readiness | **Model judgment**, always bounded by application constraints |
| Protocol content, crisis scripts, boundary scripts, policy thresholds, incident review, rubric calibration | **Professional oversight** |

### 9.2 Crisis Route (S3) design

- **Crisis Planner (deterministic):** selects a *response template family* by category (suicidal ideation, self-harm, harm to others, abuse, medical emergency) and risk level; looks up locale resources from the Crisis Resource Table.
- **Constrained generation:** the Generator receives the template skeleton (required elements: acknowledge, express care, ask about immediate safety in a non-leading way, provide resources, invite continued conversation) with free phrasing in between. Its output is **buffered and validated** for the required elements and for absence of banned content (e.g. method details, minimizing, debate about the person's reasons).
- **After the reply:** Risk FSM updates; outbox emits `risk_level_changed` (the backend decides about human escalation/alerting under the product and legal policy); Scheduler may register a *gentle* follow-up only if the user has consented to proactive messages.
- **De-escalation** (Risk FSM): requires time since the last signal, explicit user statements of safety, and no new signals; never by the Brain alone. Until cleared, the Router applies `risk=elevated` constraints.
- **Do not over-trigger:** the L0 layer is recall-oriented; the L1 layer plus a cheap second-look rule (e.g. quoted/hypothetical/fiction context) reduces false positives, but the system accepts that safety false positives are cheaper than false negatives and tunes thresholds with clinicians.

### 9.3 Ethics-specific mechanisms

| Concern | Mechanism |
| --- | --- |
| No diagnosis | Boundary Policy; Validator banned patterns; Brain `boundary` objective for diagnostic requests; hypotheses framed in everyday language, never clinical labels, unless the user raises them and the response remains non-diagnostic |
| Overconfident interpretation | Epistemic invariant; hypothesis FSM; tentative phrasing rules; classifier `certainty_about_inference` |
| Dependency | `engagement` signals in User State (late-night frequency, session length growth, statements like "you're all I have"); Boundary Policy responses; Brain gets a `dependency_watch` flag; gentle encouragement toward other supports; product-level caps |
| User autonomy | Consent invariant, "offer, don't impose" for interventions, memory/privacy controls |
| Privacy | Sensitivity levels, field encryption, retrieval filters, data preservation, soft-delete visibility, independent use policies |
| Inappropriate intervention | Eligibility filter; Protocol contraindication flags; stop rules |

### 9.4 Evaluation architecture

Three levels, each with its own datasets and metrics, plus component-level tests.

| Level | Question | Method | Cadence |
| --- | --- | --- | --- |
| **L1 Component tests** | Does each deterministic piece behave? | Unit/integration tests; fault injection (timeouts, invalid JSON); schema/linter checks | Every commit |
| **L2 Decision evaluation** | Did the Brain choose the right objective/skills, resolve references, avoid unsafe actions? | **Golden scenario suite** (clinician-labelled; ≈ 200 → 1,000 cases) with the *expected objective set*, forbidden actions, and slot extraction truth; metrics: objective accuracy, forbidden-action rate, slot precision/recall, hypothesis over-claim rate, ambiguity handling, retrieval relevance | Every prompt/model/bundle change (CI gate) |
| **L3 Response quality** | Is the reply appropriate, warm, non-judgmental, plan-conformant? | LLM-judge on rubric dimensions **calibrated against clinician ratings** (require agreement threshold before trusting); pairwise comparisons vs. prior bundle; human review on a sample | CI + sampled production |
| **L4 Longitudinal quality** | Does understanding improve? Continuity? Right hypothesis updates? | **Simulated multi-session personas** with scripted ground-truth profiles; metrics: fact recall accuracy, contradiction handling, hypothesis precision over time, stale/zombie hypothesis rate, resumption quality; plus real-user proxy signals | Weekly / release |
| **L5 Safety evaluation** | Do crisis paths fire and behave correctly? | Red-team corpora (direct, indirect, multi-turn escalation, languages/slang, fictional framing); recall/precision per category; Crisis Route conformance | Every change touching safety; scheduled |

**Replay harness:** because traces store inputs, versions, and context ids, any past turn can be *re-run against a candidate bundle* (with privacy controls: only consented or synthetic data) and compared decision-by-decision.

**Online monitoring (privacy-respecting):** rate of question-ending replies; repeated-question rate; fallback/regeneration rate; `PLAN_UNSAFE` rate; hypothesis-voicing rate and user rejection rate; memory correction rate (a leading indicator of wrong memory); Protocol offer→accept→complete funnel; thumbs/feedback; latency and cost percentiles; safety trigger rates and false-positive review.

### 9.5 Observability

- **One `turn_trace` per turn** (§3.15) with a correlation id across async jobs (Memory Writer ops carry the originating `turn_id`).
- **Decision traceability:** a reviewer can answer "why did Theia ask that question?" from the trace: context items shown, Information Needs, Brain `rationale_tags`, policy-gate modifications, validator results.
- **Dashboards/alerts:** safety-route spikes, validator failure spikes, Brain JSON failure rate, p95 latency, provider errors, memory-queue depth, reconciliation findings, deletion SLA.
- **Sampling for clinician review:** all CRISIS turns (post-hoc), a random 1–2% of NORMAL turns, all turns with `PLAN_UNSAFE`, all user thumbs-down with "harmful/inappropriate" tags, and all turns where Validator regenerated.

### 9.6 Trace privacy

Traces contain sensitive content. Treat them as retained user data with encryption, restricted audited access and redacted review views. User-initiated deletion is soft-only; do not purge trace payloads or run expiry cleanup. Historical expiry metadata is not authority to erase records. Evaluation datasets are built only from explicitly consented or synthetic data, independently of visibility and retention.

### 9.7 Error handling, fallbacks, cost and latency management

- **Budgets:** per-turn token and time budget enforced by the Orchestrator (e.g. total 8 s hard cap; Brain 2.5 s; Generator first token 3 s). Exceeding a budget triggers the fallback ladder (§4.7).
- **Fallback ladder:** primary model → secondary model/provider → smaller model with the default plan → templated message.
- **Cost controls:** FAST route; prompt caching; Reflection quotas; Memory Writer batching; model-tier configuration; kill switches per feature (proactive, Reflection, protocols) through feature flags.
- **Extensibility seams:** every module is behind a Go interface with a versioned input/output schema (`Brain.Decide(ctx, WorkingContext) BrainDecision`, `Retriever.Retrieve`, `Generator.Stream`, …). Swap implementations (a distilled Brain, a rule-based fast classifier, a different vector store) without touching the Orchestrator; run **shadow mode** (new implementation executes silently, decisions compared) before cutover.

---

## 10. End-to-End Execution Scenarios

*(Cross-reference note: "Information Needs", cited as §5.3 in module specs, is defined in §5.6.)*

### Scenario 1: Emotional message that references the past

> **User:** "Today, when my friend talked about their achievements again, I felt completely worthless. It's the same feeling I told you about before."

| Step | What happens | Data |
| --- | --- | --- |
| 1 Gateway | Persist `msg_9921`; session continuing; lock acquired | `features: {past_reference: true, temporal: none}` |
| 2–4 Parallel | **Safety:** L0 no hit; L1 `level=low` ("worthless" feeling, no ideation). **Assembler:** focus thread `thr_12 "social comparison"` (phase `exploring`), prefs ("short replies"), open loop: none. **Retrieval:** anaphora strategy → `mem_81f` (episode, 9 days ago: "felt worthless after a coworker's promotion") and `hyp_7` (inferred, `proposed`: "self-worth tied to comparison with others") | `retrieval: 2 items, ambiguous=false` |
| 5 Router | Not trivial; risk low → NORMAL. Interventions possible? Thread phase `exploring` → **no shortlist** |  |
| 6 Brain | `intent=share_distress`; `thread_relation=continues`; resolves "before" → `mem_81f`; slots: situation (reported), emotion (reported); hypothesis `hyp_7` gets `support` (+1 evidence); objective **`reflect`**, skills `reflect_emotion` + `normalize_gently`; `question.allowed=true` (purpose: invite what went through their mind); `must_avoid: advice, hypothesis-as-fact, intervention`; `memory_worthy=true` | `BrainDecision` (§6.3) |
| 7 Policy gate | No violations; question count okay (last turn was not a question) |  |
| 9 Generator | Receives plan + skill cards + `mem_81f` as *permitted context, status: reported episode*; hypothesis not permitted to be voiced. Produces a short warm reflection that links to the earlier time without over-claiming, then one open question. |  |
| 10 Validator | One question; no advice; no inference stated as fact → pass |  |
| 11 State commit | Session: `last_objective=reflect`, `pending_question={slot: thoughts}`; Thread: `emotions` slot appended; `hyp_7` linked support counter pending Memory Committer | `turn_events` row |
| 12–14 Async | Memory Writer: ADD `episode-candidate` (kept in session buffer), CONFIRM `hyp_7` support (now 2 independent episodes → FSM may move to `supported`); trace logged |  |

**Skipped, and why:** hop-2 retrieval (reference resolved), Protocol Engine (no Protocol, no shortlist), in-path formulation (a Reflection run after session close is cheaper and has more evidence).

### Scenario 2: Exact recall

> **User:** "What exactly did I tell you last Tuesday about my boss?"

- **Gateway** extracts `temporal: "last Tuesday"` using the user's timezone.
- **Retrieval** picks the **exact/temporal** strategy: SQL on `messages` for that local date, `role=user`, FTS "boss". Returns 3 raw messages (verbatim) with timestamps.
- **Brain** `objective=answer_direct`; `permitted_context_ids` = those message ids; `must_avoid: paraphrase_beyond_source, interpretation`.
- **Generator** quotes/paraphrases closely, clearly marked as what the user said; offers to continue ("want to go back to that?"). No hypotheses.
- If *no* messages exist that day: Retrieval returns empty with `strategy_used=temporal, found=0`; Brain must say it cannot find a conversation on that date and ask whether another day was meant (**no semantic substitution**).

### Scenario 3: Continuing an exercise (thought record), user is stuck

> **User:** "I tried writing it down but I couldn't think of what I was thinking, it was all just feelings."

- **Router:** active Protocol instance `pi_44` (`thought_record`, step `identify_automatic_thought`, `stuck_count=0`). Shortlist omitted (a Protocol is already active).
- **Brain:** `step_outcome=stuck`; `protocol_action=adapt`; objective `continue_intervention`; skill `normalize` + `scaffolded_question`.
- **Protocol Engine:** step graph allows `adapt` → sub-step `identify_by_body_sensation_or_image`; increments `stuck_count=1`; emits `ProtocolStepSpec{instruction: "offer an easier entry: what situation exactly, and what did the feeling say?", allowed_adaptations: [...]}`.
- **Generator:** renders a gentle scaffold; one question.
- **Next:** if `stuck_count=2` → Protocol Engine proposes `pause` and the Brain objective becomes `close_or_pause_intervention` (offer a different technique or return to exploring), recorded in `protocol_history` as "stuck at step 2, user prefers talking first" (informs future eligibility).

### Scenario 4: Crisis signal

> **User (late night):** "I don't see the point anymore. I think everyone would be better off without me."

- **Safety Pre-Gate:** L0 and L1 both fire: `level=high`, `category=suicidal_ideation`. Router → **CRISIS**.
- **Skipped:** Brain objective selection, shortlist, retrieval beyond minimal (only locale and the user's stored safety preferences), Protocols (any active Protocol → `paused`).
- **Crisis Planner:** template family `si_passive_or_unclear`; resources from Crisis Resource Table for the user's locale.
- **Generator:** constrained generation, **buffered** output: acknowledge, express care, ask directly and gently about safety, provide resources, invite to keep talking. No advice, no analysis of reasons.
- **Validator:** verifies required elements; resource whitelist; banned patterns. Fail → regenerate once → static template.
- **State:** Risk FSM `→ high`; `risk_followup_until = +24h`; session mode `safety`; outbox `risk_level_changed`.
- **Async:** Memory Writer stores a *minimal* safety event (no detailed narrative unless user continues volunteering); trace flagged for clinician review. Scheduler registers a gentle check-in only if proactive messages are enabled.
- **Next turn:** Router enforces elevated constraints until cleared; the Brain's first objective is a supportive check-in, no interventions, no hypotheses.

### Scenario 5: Sudden topic change in the middle of a Protocol

> **User (mid-exercise):** "Actually can we talk about my mom? She called today."

- **Brain:** `thread_relation=new` (or `returns_to_paused` if a prior thread exists); `protocol_action=pause`; objective `explore` (open the new topic) with a brief acknowledgement of the pause.
- **State Committer:** Thread A → `paused(reason=user_shift)`; new Thread B → `exploring`, becomes focus; Protocol `pi_44 → paused` with `follow_up_due=null` (not pushed on the user) but an `open_loop` "exercise paused: offer to resume if relevant later".
- **Retrieval (hop on need):** Brain flags `needs_context: entity="mom"`; hop-2 retrieval returns family-related memory items; Brain is re-run once.
- **Nothing is lost**, nothing is forced.

### Scenario 6: Deeper analysis (problem needing formulation)

- Over several sessions the user describes social avoidance. Each turn, the Brain only records slots and *proposes* hypotheses (tags). The Memory Writer authors cautious statements. The Reflection Worker, triggered at session close once ≥ 3 situations and ≥ 2 sessions exist, produces **Formulation v1** for `thr_30` (CBT conceptualization): situation → thought (inferred) → emotion (reported) → behavior (reported) → short-term relief (inferred) → maintenance hypothesis (inferred, `proposed`), plus `information_gaps: ["what the user predicts will happen"]`.
- **Next session:** Information Needs surfaces that gap; the Brain, if readiness is present, picks `clarify` with a *prediction* question. When the user answers, support is added → hypothesis `supported` → only then may the Brain choose `provide_insight` and voice it tentatively; the user's reaction becomes new evidence (endorse / reject / unsure).
- If the user says "no, that's not it" → `rejected`; a similar hypothesis is suppressed for K days.

---

## 11. Implementation Roadmap

### 11.1 MVP (must-have, ≈ first release)

| Area | MVP scope |
| --- | --- |
| Pipeline | Gateway, Safety Pre-Gate (L0 + L1), Assembler, Retrieval (temporal + hybrid + thread), **Brain**, **Generator**, Output Validator (rules + one classifier), State Committer, Orchestrator with routes NORMAL/FAST/CRISIS |
| Knowledge | Communication Policy; ≈ 10–15 Skill Cards; Decision Policy; Boundary Policy; Crisis Resource Table for launch locales; User Information Schema (small); Problem Understanding Schema |
| Interventions | **3–5 Protocols** (suggested: grounding/breathing, thought record, behavioral activation, values clarification, problem-solving) with clinician review; Protocol Engine; consent invariant |
| Memory | `messages`, `memory_items` (fact/preference/episode/hypothesis), Memory Writer + Committer (CONFIRM/ADD/SUPERSEDE/CORRECT), user memory view with edit/soft-delete, preserved original and derived records |
| State | Session, Thread, Protocol, Risk (User State minimal); `turn_events`; reconciliation job |
| Reflection | **Session-close episode consolidation only**; hypotheses limited to `proposed` + simple `supported` rule; formulation as *internal* artifact (not voiced yet) |
| Safety/Eval | Crisis Route with templates; golden scenario suite (≈ 150–300 cases); safety red-team set; turn traces; clinician review queue; dashboards |
| Ops | Bundle versioning + CI linter; feature flags; model fallback ladder |

**Explicitly deferred from MVP:** voicing hypotheses to users, 5P/CBT *full* formulations, proactive follow-ups beyond one simple scheduled check-in, dependency analytics beyond basic counters, screening instruments, multi-Protocol concurrency, a distilled/cheaper Brain, shadow-mode infrastructure, multi-language beyond launch languages.

### 11.2 Phase 2 (quality and longitudinal depth)

- Reflection Worker with full Formulation Frameworks and the hypothesis FSM driving *tentative insight* responses.
- Scheduler/Proactive engine with follow-ups on exercises and goals; open-loop surfacing.
- Memory maintenance: staleness decay, cluster merging, retrieval-feedback loop.
- Simulated multi-session persona evaluation (L4); LLM judges calibrated with clinicians.
- Dependency/engagement monitoring with Boundary responses.
- Prompt-caching and cost optimization; FAST route expansion.
- Skill Card and Protocol library expansion (ACT/DBT-informed modules).

### 11.3 Phase 3 (scale and specialization)

- Screening instruments (PHQ-9, GAD-7) as `assessment` Protocols with consent and result-handling rules.
- Distilled or fine-tuned Brain and safety classifiers on collected, consented, de-identified decision data.
- Shadow-mode deployment, A/B bundles, automated regression gating.
- Personalization of communication style from outcomes.
- Clinician dashboard / referral workflows (subject to product and legal review).
- Multimodal (voice) and richer UI for protocol steps.

### 11.4 Suggested build order (dependency-driven)

1. Data model, Gateway, Trace logger, Orchestrator skeleton with FAST route and Generator only (end-to-end "hello world" with tracing).
2. Safety Pre-Gate + Crisis Route + Output Validator (safety before sophistication).
3. Context Assembler, State tables, State Committer, Retrieval (temporal + hybrid).
4. Turn Brain + contract validation + Decision Policy v1 + Skill Cards.
5. Memory Writer/Committer, correction and deletion paths, memory view.
6. Protocol Engine + first Protocols.
7. Reflection Worker (episode consolidation), reconciliation job.
8. Eval harness (golden suite, replay), dashboards, clinician review loop.
9. Hardening: fallbacks, load tests, privacy audit, red-team.

Steps 1–2 can start in parallel with data-model design; the golden scenario suite (step 8) should begin being *authored* from step 1 and be used to gate step 4 onwards.

### 11.5 Open questions and justified initial recommendations

| Open question | Why it matters | Initial recommendation | How to resolve |
| --- | --- | --- | --- |
| Should the Brain and Generator be one call? | Latency and cost vs. decision auditability | **Keep separate** for the MVP; evaluate merging for low-risk classes | A/B with the golden suite and clinician ratings |
| Will users see hypotheses/"insights"? | Trust, risk of mislabeling | **Not in MVP**; introduce with explicit tentative phrasing and user feedback buttons | Pilot with clinician-reviewed phrase sets |
| How long to retain raw messages and traces? | Privacy, legal exposure, memory quality | Retain messages, traces and related data, including hidden records; permanent erasure/expiry is a separate future policy | Legal and product policy review |
| Safety thresholds (precision/recall) | Over-triggering harms experience; under-triggering harms safety | Err toward recall in L0; calibrate L1 with clinician-labelled data | Red-team + review of false positives |
| Session timeout and thread dormancy windows | Affects resumption feel | 30 min / 30 days as defaults | Analyze real usage patterns |
| Which model tiers for Brain and Generator | Cost, latency, reliability | Brain: small fast tier with structured output; Generator: strongest affordable tier | Benchmark on golden suite, p95 latency, cost per turn |
| Do we need a human escalation path? | Product/legal obligation; outside AI Core scope | AI Core emits `risk_level_changed` events; the backend/product decides | Product + legal + clinical advisors |
| Languages and locales at launch | Safety lists, crisis resources, classifier coverage | Launch with 1–2 languages with full safety coverage | Decide per target market |
| Hypothesis thresholds (e.g. ≥ 2 independent supporting episodes) | Over/under-claiming | Start conservative; tune by rejection rate | Longitudinal eval + user feedback |

### 11.6 Final summary: the system in one paragraph

For each message, a deterministic layer admits the turn, runs a safety check, assembles a compact state-based working context, and retrieves memory with a method matched to the question. A single structured Brain call interprets the message, proposes (never writes) state and hypothesis updates, decides one primary conversational objective, and selects skills; a deterministic Protocol Engine handles any multi-step exercise. A strong model then writes the reply within bounded latitude, a validator enforces policy, and a transactional committer updates state. After the reply, asynchronous workers extract memory with provenance, reconcile and correct it, and, in the background, build revisable case formulations. Safety is a layered, fail-closed system rather than a prompt; every turn is traceable, replayable, and evaluable; and every piece of psychological knowledge lives in a versioned bundle that the modules consume, not in the code.
