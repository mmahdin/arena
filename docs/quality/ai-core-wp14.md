# WP-14 — Reflection and Scheduler

Internal Go Modular Monolith implementation of Contract Pack v1.2 §3.12/3.13,
§8.7, RF-001–006 and §11.5. No HTTP, OpenAPI, WP-16, external notification
transport, or legacy Daily Reflection integration is introduced.

## Implementation

- Immutable reviewed WP-03 assets select exactly one framework by problem tags;
  ambiguous/missing frameworks never cause guessed psychological content. A
  session may still consolidate an episode without authorizing formulation.
- Strong-tier WP-04 structured `reflection.v1` calls, strict injected framework
  keys and vocabularies, XML escaped data, fixed section order and per-section
  token limits. Bounded evidence is sampled across sessions. No model confidence
  or assistant narrative counts as independent user evidence.
- RF-004 applies global situation/session minima and per-field evidence/session,
  epistemic, hedge and clinical-label checks. Unqualified fields become unknown.
  The summary is rebuilt from accepted fields so rejected claims cannot survive
  in a free-form paragraph. Previous versions remain immutable.
- Existing WP-12 Writer consolidates the selected closed-session segment;
  existing Committer handles hypothesis dedupe, evidence counts/FSM, suppression,
  rejection cooldowns, encryption and embeddings. Actual writer/model provenance
  is retained per proposal. User endorsement remains a separate authenticated
  action and never transforms an inferred item into a reported fact.
- Existing `ai_jobs` leases, fenced encrypted staged checkpoints and retry/backoff
  are reused. Strong reflection is checkpointed before the separate memory call;
  retries avoid repeating completed model stages. One short owner-locked memory
  transaction publishes memory operations, formulation version, thread links,
  validated state proposals, encrypted receipt, event and completion. No model or
  encryption provider call occurs inside that transaction. Revocation and source
  changes are checked again after provider work. Stale formulation checkpoints
  fail closed; a new trigger may schedule fresh work.
- Session close, explicit Brain formulation request, linked-memory/thread growth,
  accumulated contradictions and nightly hooks are supported. Sweeps recover
  previously closed sessions, expire unanswered offers after 48 hours with a
  decline history/event, and route dormancy proposals through State Committer
  rules. Current focus threads are protected. Idle closure never closes or hides
  the user's conversation.
- Timers preserve the existing schedule/cancel/sent status contract and emit only
  internal `wakeup.v1` metadata in the transactional outbox. Explicit proactive
  opt-in AND an approved `memory.Proactive` authorization are required; current
  risk, follow-up restrictions, quiet hours/timezone, ownership and one-per-24h
  limits are checked at scheduling and dispatch. Model follow-up prose is never
  converted into a timer. Protocol due times come from existing reviewed state.

## Composition and migration

`NewAIReflection` requires the same release-verified snapshot as the other modules,
WP-04 client/embedding model, reviewed deterministic Memory policy, managed cipher
keys and a stable worker identity. `memory.Reflect` and `memory.Proactive` are
separate explicit policy actions. There are no permissive production defaults.
Start Worker.Run and Scheduler.Run under host cancellation and join on shutdown.
Due returns authorized internal metadata; external delivery requires a separately
authorized host adapter and the complete normal safety pipeline.

Migration 0022 adds a composite job owner key, `ai_reflection_runs` encrypted
receipts with owner FKs and a physical-delete guard, and follow-up dispatch time.
It reuses existing state/memory/formulation/job tables. Rollback is application
rollback with retained additive schema; no down migration or cleanup deletes.
Only memories/formulations have resource deletion metadata in existing AI state;
visibility never invents a permission or recursively invalidates derivatives.

## Verification

Executed on Go 1.25.4 and local SQL Server 2022 with isolated synthetic databases:

- Reflection/Scheduler/State/Memory/composition unit tests with race detection:
  passed.
- Four WP-14 SQL tests with race detection: passed, including multi-session
  formulation versioning and hypothesis support, transactional rollback and
  checkpoint recovery, revoked consent, owner isolation, stale leases, concurrent
  due-timer dispatch, internal session close and retained soft-delete payloads.
- Full-suite compatibility results and final rerun are recorded separately in
  `ai-core-integration.md`; imported WP-13/15 defects are excluded from this patch.

## Explicit limitations and review blockers

The repository knowledge assets remain unreviewed synthetic placeholders. These
engineering fixtures do not establish psychological accuracy, clinical safety,
reviewed thresholds, consent policy approval, provider performance or production
readiness. Production loaders reject synthetic activation. External notification
transport and product/API authorization are outside WP-14.

`reflection.v1` merge/split proposals have no destination or partition contract.
They are retained in the encrypted result and rejected with RF-005 for review;
no targets or clinical state are guessed. Eligible resolve/dormant proposals use
the existing State Committer graph. A bounded episode summarizes its selected
message range, not an unlimited session transcript. Evidence that cannot pass
explicit use-policy checks is withheld while all originals remain retained.
