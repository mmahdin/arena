> **Runtime superseded (2026-11-23):** [Unified memory-centric architecture](unified-memory-centric.md) is the active runtime. Separate Brain/Generator, safety/router/output-classifier calls and regeneration described here are historical. Persisted contracts and historical Inspector examples remain supported.

# Product-first conversational architecture

Implementation amendment, 2026-11-18. Owner: AI Core. This updates the ordinary
conversation rules in Contract Pack v1.2 §§3–4, 8, 10 and 11 and the protocol
restriction in the multilingual amendment. The existing modular monolith,
owner-scoped reads, single-writer finalization, input admission, idempotency,
consent records, release verification and high-risk crisis route remain intact.
Public HTTP shapes, persistence tables and LLM call count are unchanged.
The internal `ProtocolView` gains an optional `success_criteria` field; older
contexts remain readable.

## Audit and decisions

These are implementation findings, not a claim about live-model behavior. The
before/after behaviors below are reproduced with synthetic inputs and scripted
provider responses through the real application modules.

| Finding and implementation reference | Change and reason |
| --- | --- |
| Shortlisting depended on six exact English sentences in the removed `application/protocol/cues.go`. | Remove the phrase classifier. The existing Brain understands requests, readiness and agreement in context and in either supported language. Shortlisting exposes knowledge, not consent. |
| `assembler.computeEligibility` denied all synthetic bundles, new/exploring threads and any recent decline, regardless of the requested exercise. | Activation/review belongs to Registry/loaders. Ordinary exercises need no intake thread or mature phase; individual protocol prerequisites and actual risk still apply. Do not infer global unwillingness from declining another exercise. |
| `protocol.ShortlistWithContext` required a verified contraindication inventory; `Engine.offer` rechecked without one. Production composition supplied no inventory, so every shipped protocol was excluded. | Check actual user/thread risk and known exclusion flags, without inventing an attestation that all unknown clinical conditions are absent. A configured facts source failure suppresses offers for that turn while allowing conversation. |
| English-only checks in `bundle.Manifest.ProtocolLanguage`, Generator and Engine stopped Persian protocol delivery. | Allow declared `en`/`fa` protocol coverage with English definitions. Generator realizes meaning in the resolved response language; signed resource/clinical gates remain. |
| `Protocol.offer.must_say`, `Engine.RequiredText`, Orchestrator binding and OV-014 required literal English opt-out/time sentences on every step. | New definitions use English `offer.intents`. Goals, instructions and completion criteria guide expression. Legacy identifiers remain readable as intent references, without automatic exact validation or repeated speech. Explicitly bound fixed disclosures and exact crisis text are still supported. |
| `brain.validate` required a supported personal hypothesis even for general psychoeducation (`provide_insight`). | Verify a hypothesis only when one is actually voiced. General explanations do not need a personal inference. Source/status/permission checks remain for personal hypotheses and memories. |
| Brain automatically added an advice prohibition to every reflective answer and forced confirmation questions for corrections/missing recall. | Preserve the chosen useful response and acknowledge uncertainty. No compulsory question or blanket advice ban based solely on the objective label. Explicit plan restrictions and evidence rules remain. |
| Brain and Assembler enforced a question-streak cutoff; Validator discarded two questions, an unplanned question or an answer exceeding the length target. | Question/length fields are style guidance for ordinary replies. OV-001/002/012 are advisory; the private stream no longer rejects question punctuation. Remove unpunctuated interrogative-word guessing, which classified statements such as “How thoughts affect feelings…” as questions. |
| OV-015 treated every mention of a technique without a protocol instance as forbidden. | Ordinary optional coping suggestions need no state machine. The matcher still enforces an explicit `intervention` restriction; actual elevated/follow-up/crisis constraints remain. |
| Engine required every `expected_inputs` field, including ratings explicitly described as optional, and exact substrings for summaries. | Expected inputs are allowed capture keys, not a mandatory form. Brain judges success against the authored criterion; bounded paraphrases and privately completed steps can progress. No arbitrary keys, oversized captures or illegal edges. |
| Assembler exposed the active step's goal and input keys to Brain but omitted its completion criterion. | Include the English `success_criteria` in internal working context and Brain's rendered state, so outcome interpretation has the authored progression target. |
| Engine forced exit after two stuck outcomes irrespective of user choice. | Repeated difficulty can be adapted or explored. Actual distress, refusal and user pause/stop still interrupt the exercise. No attempt-count rule dictates conversation. |
| Generator checked captures from a completed step against the *next* step's expected inputs. | Engine retains current-step validation; Generator checks definition membership. Successful captures no longer cause generation failure after advancement. |
| Orchestrator replaced a pause/stop plan with `restrictedPlan()`, which lacks fields required by the real Generator. A fallback then discarded the pause/stop patch. | Keep the valid conversational plan and remove only the protocol directive. Apply a validated status-only user-control patch even when the response uses a provider fallback; never apply an unseen new offer or step. |
| A valid longer disclosure could hit Generator’s 400-token message-block limit despite fitting the total input budget. | Use the existing total budget rather than a separate message-block rejection; never truncate the current message. |
| Full English knowledge plus the catalog could exhaust the Brain budget before any model call. | Remove duplicate role instructions and render a compact shortlist containing IDs, purpose, duration and prior outcome. Keep the existing input/call budgets and preserve space for recent context. |
| Persian L0 escalated bare mentions of suicide, self-harm and overdose, including informational questions. | Require disclosure/intent/ingestion patterns for those L0 rules. L1 still evaluates the whole context; detected personal danger remains a floor and classifier outage still degrades conservatively. |

Code references are under
[`apps/api/internal/modules/ai`](../../apps/api/internal/modules/ai), primarily
[`brain/gate.go`](../../apps/api/internal/modules/ai/application/brain/gate.go),
[`brain/render.go`](../../apps/api/internal/modules/ai/application/brain/render.go),
[`protocol/eligibility.go`](../../apps/api/internal/modules/ai/application/protocol/eligibility.go),
[`protocol/engine.go`](../../apps/api/internal/modules/ai/application/protocol/engine.go),
[`generator/protocol.go`](../../apps/api/internal/modules/ai/application/generator/protocol.go),
[`orchestrator.go`](../../apps/api/internal/modules/ai/application/orchestrator/orchestrator.go)
and [`validator.go`](../../apps/api/internal/modules/ai/validator/validator.go).

## Execution and therapeutic intent

The pipeline remains WorkingContext → retrieval → Brain → Protocol Engine →
Generator → Validator → atomic finalization. There is no translation service,
second planner, semantic-validation pipeline or mandatory new LLM call.

Brain receives the eligible library before deciding how to help. A new ordinary
exercise need not have a thread, problem label, pre-filled slots or a clinical
inventory. Known exclusions, owner checks, existing active protocols, user avoids,
actual risk and explicitly configured prerequisites still filter eligibility.
The shipped low-intensity definitions collect context during the exercise rather
than requiring an intake beforehand. Their arbitrary repeat/cooldown limits are
zero (disabled); positive values in legacy/custom definitions retain their meaning.
Previous declines remain context/ranking information and should not be nagged
about. A fresh request can reopen the choice.

The existing Brain call interprets explicit requests (`request_technique` with
`action_cue`) and agreement to a specific offered/paused exercise. Orchestrator
passes that interpretation to Engine; Engine verifies the owner and current user
message. A clear request can create an active instance, consent timestamp/message
reference and first step in the same atomic delta. An unsolicited offer remains
offered until agreement. Distress alone and a request for information are not
consent. Consent interpretation is a model judgment, subject to evaluation—not
proof obtained from a phrase whitelist. The public API cannot set this boolean.

English definitions keep the existing legal graph, step `goal`, `instruction`,
`success_criteria`, allowed capture keys and adaptations. `offer.intents` expresses
voluntary participation and purpose without prescribing dialogue. Generator sees
the current goal and completion criterion and can explain, use an example, respond
to a question or invite the next small part in natural Persian or English. It does
not copy internal instructions or repeat offer disclosures at every step.

Engine advances on the validated Brain outcome, not worksheet completion. A
capture is a bounded model summary of user input, not a verbatim quote or a new
user-profile fact. Provenance for user facts/memory remains separately governed.
The engine still rejects unknown input keys, unknown protocol versions, illegal
edges, cross-owner instances and activation without an owned consent message.
Pause, stop, decline, topic redirection, worsening distress and risk transitions
remain meaningful controls. Protocol graphs still have one selected transition
per decision; they do not become free-running agents.

## Right-sized validation and retained boundaries

Normal replies keep deterministic checks for diagnoses, prescribing, guarantees,
therapist identity, unauthorized recall/context, internal IDs, invented contacts,
dependency encouragement and invalidation. They keep bounded buffering, empty/
truncated-output detection and the unsafe-plan sentinel. Question/length warnings
remain visible in reports but do not cause a second model call or static fallback.
A non-question sentence beginning with an interrogative word is no longer turned
into a pending question.

Ordinary protocol intent is realized by Generator and evaluated for meaning;
regexes do not certify that an exercise was therapeutically helpful. The optional
existing Validator classifier can provide contextual scores for pressure,
unsupported certainty, dependency and invalidation. It is not promoted into a
mandatory protocol judge: doing so would add latency/cost, and its current four
scores do not establish step-goal compliance. Configured classifier failures
still fail validation; no model approval is fabricated when it is unavailable.

High-risk routing, L0 floors, conservative classifier-failure handling, risk
follow-up, crisis buffering, exact pinned crisis templates, fresh curated contacts
and production clinical/signature checks remain. Ordinary distress is not itself
a contraindication; a detected elevated/high-risk signal retains its existing
constraints. Narrowing term-only Persian matches does not downgrade a recognized
intent signal. Language never establishes nationality or location.

## Before and after

| Synthetic scenario | Previous mechanism | Refactored behavior |
| --- | --- | --- |
| “یه تمرین ACT برای فاصله گرفتن از فکرهام می‌خوام.” | No exact English cue, no mature thread, no verified inventory; Persian protocol blocked. | Eligible English ACT knowledge can deliver the first Persian step and record the request as consent. |
| “I would rather skip the belief rating.” | Missing expected field repeats the worksheet; an otherwise valid capture can fail Generator on the next step. | Meaningful progress can advance without the rating; previous-step captures are accepted. |
| “مثال ساده‌تری می‌زنی؟” | Two stuck outcomes force pause. | Use an authored adaptation and continue while the user wants to. |
| “فعلاً نگه داریم.” | Skeleton response plan fails generation; fallback can erase the control patch. | Natural acknowledgment and a persisted pause, including during provider fallback. |
| “فرق فکر و احساس چیه؟” | `provide_insight` can require a personal hypothesis; forced reflection/clarification replaces education. | Explain the concept with an appropriate example, without inventing a hypothesis or mandatory empathy. |
| Informational Persian question mentioning suicide, followed later by actual personal intent | A bare term is already high risk. | Information is assessed in context; subsequent personal intent still routes to the exact crisis plan. |

## Product evaluation and verification

Six `knowledge/evals/golden/product_*.yaml` scenarios cover Persian ACT, English
CBT, psychoeducation/authorized continuity, ordinary distress followed by practical
help, declining then requesting an exercise, and informational safety terms
followed by actual crisis. They assess naturalness, emotional understanding,
usefulness, Persian phrasing, autonomy, psychological meaning, repetition,
unnecessary refusal and continuity through the existing `output.judge` interface.
The former exercise-request regression no longer accepts generic guidance in
place of a requested eligible exercise.

Use the existing offline evaluation executor/reviewer, not a runtime validation
call. The report keeps aggregate `quality_review` and adds per-rubric metrics such
as `quality_helpfulness`, `quality_persian_naturalness` and
`quality_no_unnecessary_refusal`. Rates are fractions of rubric thresholds met,
not automatically measured clinical outcomes or a validated refusal detector.
Missing review is reported as missing evidence and queued; it cannot count as a
quality pass just because deterministic validation passed.

Reviewer rubric: score each named dimension 1–5 against the user's actual need
and conversation history. A 4 or 5 means useful, contextually appropriate behavior
without the named failure; 1–2 means a clear failure, and 3 is materially mixed.
For Persian, review idiom, register and meaning with a native speaker. For
`no_unnecessary_refusal`, penalize a refusal or generic detour where an eligible
request could be helped; appropriate limits for actual danger are not failures.
For `no_permission_loop`, a clear exercise request should lead to help, while an
unsolicited proposal must still leave a real choice. Do not award points merely
for a stock empathy phrase, question count or repeated opt-out sentence.

Engineering tests execute the real Assembler, Safety, Brain/gates, Engine,
Generator, Validator and State Committer with the full English bundle and
scripted providers. Multi-turn tests cover Persian ACT, English CBT, optional
ratings, private answers, paraphrases, adaptation, pause/resume/decline, provider
outage and crisis escalation. Additional tests exercise prompt budget capacity,
general insight without a personal hypothesis, advisory style rules and the
review-evidence boundary. Ordinary product turns use one Brain and one Generator
call; existing safety classification remains separate.

Verification completed: `go test ./...` from `apps/api`; race tests for
Assembler, Brain, Generator, Protocol Engine, Orchestrator, bundle, Safety,
Validator, evaluations and contracts; `python3 scripts/check_structure.py`;
`python3 scripts/check_contract.py`; and `aicore-bundlelint -dir ../../knowledge
-core-version 1.0.0`. Bundle lint reports the expected unsigned/unreviewed
synthetic-content warnings. `git diff --check` is clean.

These tests prove execution and regression behavior, **not** live-model
naturalness, semantic fidelity or clinical benefit. No live model/native review
or SQL Server smoke result is claimed. English and Persian content remains
unreviewed development material and the bundle remains `synthetic_only: true`.
Quoted danger, negation, dialects and romanized Persian can still challenge L0;
existing lexical memory/provenance checks do not prove cross-language entailment.
The semantic consent/outcome judgments and less rigid ordinary protocol delivery
need adversarial, native-language and clinical review before real-user use.
