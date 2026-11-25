# AI Core WP-01–15 integration verification

Verified locally on 2026-11-18 with Go 1.25.4 and Microsoft SQL Server 2022.
The two documents in `docs/ai-core/`, including Contract Pack v1.2, govern this
implementation. All new test conversations, provider outputs, policies and
knowledge assets are explicitly synthetic. No clinical approval is asserted.

## Deliverables and scope

Apply the existing WP-13 and WP-15 patches to the WP-12 baseline first, then
`theia-ai-core-wp14.patch`, then `theia-ai-core-integration.patch`, from the backend
repository root. The supplied WP-13/15 patches were already applied in the
workspace; reverse application checks confirmed this, so they were not applied
twice. Their implementations are excluded from both new patches.

WP-14 is exported from its own frozen phase baseline. The integration patch
contains only subsequent fixes, internal composition, additional tests and this
report. Two imported compatibility prerequisites are intentionally in the
integration patch: a WP-13 test compilation error and a WP-12 query against a
nonexistent thread deletion column. Build/test the combined result. No existing
migration, shared JSON schema, HTTP endpoint or OpenAPI contract was changed.
The existing Git index and commits were not modified.

The patches were checked with `git apply --check` against clean phase snapshots
and their applied results compared byte-for-byte with their respective intended
trees. Application order and a combined reconstruction are checked separately
from test execution. These patches are incremental exports, not standalone
repositories containing earlier work packages.

## WP-14 implementation

See [ai-core-wp14.md](ai-core-wp14.md) for the implementation and migration detail.

- Reflection selects a bundle-approved framework by thread tags; uses WP-04
  strong-tier structured `reflection.v1` with injected schema; assembles bounded,
  escaped, owner-authorized longitudinal evidence; validates RF-001–006.
- Formulation fields require independent reported user evidence and the approved
  per-field/session thresholds. Unsupported fields become unknown; inferred
  claims remain hedged and cannot become reported facts. Accepted fields alone
  form the bounded summary. Prior formulation versions remain retained.
- Existing WP-12 Writer/Committer consolidate the closed session segment and
  process hypothesis operations, evidence counts, deduplication, suppression and
  cooldowns. Actual extraction model provenance is retained. Strong reflection
  and episode extraction have separately durable encrypted checkpoints.
- Fenced SQL job leases, bounded retries, explicit recovery, revocation checks
  and graceful cancellation protect asynchronous work. Memory, formulation,
  thread/state updates, encrypted run receipt, event and job completion commit
  atomically. No provider call is made in the commit transaction.
- Scheduler recovers closed sessions; triggers reflection on approved growth,
  contradiction and nightly criteria; closes idle AI sessions; expires offers;
  and applies permitted dormancy through the State Committer. It preserves the
  user's conversation. Follow-ups require explicit opt-in, use-policy approval,
  current safety eligibility, timezone/quiet-hour checks and daily limits.
- Timers emit internal durable outbox metadata only. They do not initiate a
  user-facing interaction. Migration 0022 is additive and protects new receipts
  against physical deletion. No deletion, content clearing or cascading purge
  was introduced.

## Integration changes

| Area | Defect or missing handoff | Result |
| --- | --- | --- |
| Internal composition | No constructor assembled all implemented modules and workers | `NewAIInternal` wires real SQL Gateway/finalizer, Safety, WP-08 context, WP-09 retrieval, WP-10 Brain, WP-11 Generator, WP-07 Validator, WP-12 Memory, WP-13 Protocol, WP-14 Reflection/Scheduler; `Run` cancels and joins three bounded workers |
| Knowledge and policies | Integration must not introduce permissive production defaults | One pinned bundle, explicit cipher, embedding client/model/dimension, use policy, memory policy, worker identity and tracer are required; synthetic classifier/instrumentation seams require the explicit test option |
| Protocol eligibility | Contraindications were always unknown with no injection point | Owner-scoped `ProtocolFacts` dependency supplies verified flags; missing/unverified facts remain unknown and restrictive eligibility fails closed |
| Protocol validation | Approved required phrases did not reach Validator context | Engine resolves immutable bundle text and Orchestrator binds it before generation/validation |
| Protocol assets | Accessors exposed mutable pinned graphs/references | Accessors return deep copies; mutation and required-text regression test added |
| Reflection scheduling | Brain `needs.formulation` had no job handoff | The vetted decision and accepted focus state produce a pinned, owner-scoped job in transactional turn finalization |
| Shared async repository | Reflection checkpoint override rejected ordinary memory jobs | Non-reflection checkpoints delegate to the existing Memory repository |
| Reflection recovery | Eligibility-only failures consumed the daily model quota | Pre-model ineligible work has a distinct terminal status code and does not consume model-run quota; explicit owner-scoped retry supports reflection jobs without clearing checkpoints/history |
| Job fencing | Memory failure transition checked token but not lease expiry | Failure transition now also requires an unexpired lease |
| Evidence privacy | Independently authorized derived items were lost when raw evidence was withheld | Derived items remain eligible under their own policy; hidden raw sources require separate retained-source authorization; no implicit consent inheritance |
| Context counters | Thread history counts were missing | Count distinct completed turns whose accepted state events link the thread, instead of counting unrelated conversation messages |
| SQL compatibility | WP-12 queried `ai_threads.deleted_at`, which does not exist | Query uses existing state semantics; memory/formulation deletion metadata remain unchanged |
| WP-13 tests | Imported redeclaration prevented compilation; stale-instance test aliased mutable input | Correct the declaration and snapshot-copy test input; stale-state rejection assertions remain enforced |
| WP-15 fixtures | Recorded Brain plan field used `plan` instead of `response_plan` | Correct fixture contract so assertions inspect the actual plan |
| WP-15 capture | Wrappers hid WP-11 budgets/fallback capability and erased observations on replay | Forward actual generation capabilities, fail closed when absent, and start capture only for an executable leased admission; regression tests added |
| WP-15 review sampling | CRISIS routes below a hardcoded risk threshold were missed | Every CRISIS route enters the pending safety-review queue |
| Existing SQL replay test | Fixed provider-call count assumed no WP-11 regeneration | Assert replay adds zero calls relative to the completed run |

## Executed verification

Commands run from `apps/api` unless stated otherwise. Exit status was zero for
every final command below. Plain `go test ./...` intentionally skips SQL tests
without opt-in; the separate SQL invocation actually executes them.

| Check | Final result |
| --- | --- |
| `go test ./...` | PASS across the whole Go module |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test -race ./internal/modules/ai/... ./internal/app/composition ./tests/integration` | PASS; SQL tests opt out in this invocation |
| Live SQL: `go test -race ./tests/integration -count=1 -v` | PASS, 31 top-level tests plus subtests; package time 37.235 s |
| Clean reconstruction: `go test -race ./... -count=1` with live SQL enabled | PASS across the full module; SQL integration package time 37.759 s; build and vet also passed in this reconstructed tree |
| `go test -race ./internal/modules/ai/eval -count=1 -v` | PASS, including capture contracts, longitudinal state carry, fixture isolation, trace/candidate replay requirements, missing-evidence rejection and regression detection |
| WP-15 CLI recorded goldens | PASS: 3 scenarios, 7 turns, 58/58 assertions, zero failures and zero missing-evidence blockers |
| WP-15 CLI comparison against the recorded baseline | PASS, zero regression; the same synthetic observations are not an independent model-quality experiment |
| `python3 scripts/check_structure.py` at repository root | PASS: 41 required paths, 6 skills, SQL Server dialect checks |
| `python3 scripts/check_contract.py` at repository root | PASS: existing 30 method/route pairs match composition |
| Patch whitespace and fresh application checks | PASS; WP-14 and integration exports remain separate |

The SQL invocation used `WELLBEING_INTEGRATION=true`, `APP_ENV=local` and a local
SQL Server `DATABASE_URL` supplied privately by a temporary wrapper. No secret is
included in a command example, fixture or report. Tests create isolated synthetic
databases and use the existing migration/test infrastructure. The complete suite
also reruns existing legacy HTTP/Daily Reflection tests for regression coverage;
the new AI Core neither invokes nor reuses those experimental implementations.

Reproduce the recorded evaluation with:

```sh
go run ./cmd/aicore-eval \
  -golden ./internal/modules/ai/eval/testdata/golden \
  -observations ./internal/modules/ai/eval/testdata/observations \
  -out /tmp/theia-ai-core-eval.json
go run ./cmd/aicore-eval \
  -golden ./internal/modules/ai/eval/testdata/golden \
  -observations ./internal/modules/ai/eval/testdata/observations \
  -baseline /tmp/theia-ai-core-eval.json \
  -out /tmp/theia-ai-core-eval-regression.json
```

The retained sanitized [evaluation report](test-results/ai-core-eval.json) contains
metrics and assertion outcomes, not prompts, drafts or private evidence.

### What the integration tests establish

The new `TestInternalAICoreSQLCompositionMultiturnEvalAndWorkers` calls the actual
internal composition with live SQL persistence and a deterministic synthetic
provider behind the real WP-04 client/schema validator. WP-15 capture executes
two real turns through this composition, rather than substituting recorded
observations. The test additionally checks three-turn state/thread counters,
ordinary memory job checkpoint dispatch, extraction/deduplication/embedding,
completed replay without a provider call, Protocol consent/activation, explicit
memory correction, unsafe-draft regeneration with no draft persistence, idle
session closure, asynchronous reflection/episode commit, cancellation/join,
crisis routing that bypasses free Brain reasoning, monotonic persisted risk and
owner rejection before provider work. The output-classifier score fixture is a
synthetic test double; it does not measure real semantic safety.

Four WP-14 live-SQL tests cover atomic memory/formulation updates, consent and
privacy, retained soft-delete payloads, independently authorized derivatives,
cross-user isolation, rollback, staged checkpoint recovery without repeated
successful model work, stale leases, concurrent timer dispatch, opt-out/quiet
hours/revocation, internal close recovery, formulation versioning across three
sessions and counted hypothesis evidence without inferred-to-reported promotion.

Existing unit/SQL suites additionally exercise malformed schemas, bounded second
retrieval hops, retrieval isolation and degraded behavior, model/stream failures,
safe fallbacks, synchronous corrections, state policy validation, admission and
completion races, migration upgrades, late transaction rollback, soft-delete
lifecycle/recreation and concurrent finalization. The combined test invokes the
real retriever, but does not establish provider-quality semantic recall or
multi-hop clinical relevance.

### Failed or unexecuted verification

No final executed test is failing. Earlier failures exposed the integration
defects listed above and were fixed, followed by successful reruns. This does
not mean every architecture requirement has complete coverage.

Not executed: live external-provider inference; production conversation replay;
calibrated LLM judges; clinician-labelled golden evaluation; clinical safety or
effectiveness review; broad multilingual/red-team assessment; sustained load,
resource-exhaustion, provider-cost or production latency SLO tests; external
notification delivery; proactive wake-up turn completion; WP-16 product/API
acceptance. No result or approval is inferred for any of these.

## Remaining issues and integration requirements

1. **Proactive wake-up execution is incomplete.** WP-14 emits authorized durable
   `wakeup.v1` outbox metadata, but existing WP-02 admission rejects `OriginWakeUp`
   and Orchestrator disables the PROACTIVE execution path. There is no consumer
   that routes the metadata through a complete safety/context/Brain/validation
   turn. Timer `sent` status denotes internal outbox dispatch here, not delivery
   to a person. It must never be presented as confirmed user delivery. Completing
   this requires a reviewed synthetic-turn admission/persistence contract that
   preserves origin and provenance, handles opt-out between dispatch/execution
   and does not manufacture a reported user message. No user interaction is
   initiated by this implementation. This is an internal integration gap, not a
   claim that all remaining work belongs to WP-16.
2. **Runtime replay evidence is incomplete.** WP-15 replay/candidate-replay tests
   use explicit complete synthetic trace fixtures. Normal persisted runtime
   traces are summaries without full encrypted typed inputs/outputs and cannot
   reconstruct every turn. WP-15 correctly blocks missing replay evidence.
   Production replay needs the architecture's explicit encrypted retention,
   access, ownership and trace-artifact policy; raw prompt/body logging is not
   an acceptable substitute.
3. **Reflection merge/split lacks a destination contract.** `reflection.v1`
   proposals have no merge target or split partition. These proposals are
   retained encrypted and rejected with RF-005 for review. Resolve/dormant use
   existing validated transitions. Inventing targets would silently change the
   contract and could corrupt state.
4. **Reviewed assets and policies are required.** Repository assets/fixtures are
   synthetic and unreviewed. Production loaders reject this activation. Supply
   a released, reviewed Knowledge Bundle (frameworks, thresholds, prompts,
   Protocol text/graphs, safety policy and locale resources), reviewed use/consent
   policies, managed encryption keys, a bounded safe tracer, WP-04 provider and
   embedding configuration. Supply verified `ProtocolFacts` if Protocols require
   contraindication absence; missing facts intentionally block those offers.
5. **Clinical quality is unverified.** The 58 recorded assertions and live-SQL
   deterministic tests establish engineering behavior only. Safety review queue
   entries remain pending. Clinical reviewers must validate content, locale
   coverage, hypothesis/formulation quality, semantic classifier performance,
   judge calibration and proposed release gates. No techniques or approvals
   were invented to fill these gaps.
6. **Product cutover remains WP-16.** Existing application HTTP composition still
   uses the earlier prototype AI. The new internal core has no product route,
   authentication adapter, push transport or public memory controls. A host must
   explicitly construct it, run migration 0022 via the existing runner, start
   and join its workers, and bind authorized user identity at a future reviewed
   boundary. Do not treat legacy routes as exercising this composition.

## WP-16 readiness assessment

**Not ready for an unconditional WP-16 handoff declaring WP-01–15 complete.**
The tested user-turn pipeline, memory/reflection transactions and internal timer
machinery compile and pass engineering verification, so this is a concrete basis
for reviewing the WP-16 boundary. Proactive turn admission/execution and complete
runtime replay remain unresolved internal integrations; merge/split needs a
contract decision. Resolve or explicitly scope those gaps before claiming full
internal completion. Clinical/release approval and real-user deployment remain
blocked independently. WP-16 itself was not implemented.
