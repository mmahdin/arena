# WP-10 Turn Brain

Implements `ports.Brain` from Contract Pack v1.2, independently of the prototype
AI service. One structured reasoning call produces analysis, epistemic slot and
hypothesis proposals, needs, objective and response plan. It never writes state,
executes a protocol, authors memory/hypothesis statements, or generates a reply.

`Load` consumes one immutable WP-03 snapshot. It resolves the existing embedded
`brain.v1` schema using that snapshot's vocabularies, slots and catalogs. Prompt
wording and psychological knowledge are read exclusively from the bundle. Static
blocks are cacheable; dynamic state, memory, shortlist, recent messages and current
message are escaped data. UUID handles are attributes. Budget trimming removes
oldest recent messages, lowest-score memory, then session summary; an impossible
budget takes FB-001 without truncating the current message or instructions. Token
counts use WP-08's conservative character estimate, requiring provider calibration.

`Decide` requires authenticated metadata through `WithCallMeta`, matching turn,
session and bundle hashes. The production composition requires an owner-scoped
`ContextVerifier` before sending any private source to the provider. The MSSQL
implementation verifies message ownership, source text, chronology, memory kind,
epistemic/status metadata and explicit WP-09 AI-use authorization. It reads only;
normal visibility predicates hide soft-deleted resources without changing use
consent or erasing anything. Synthetic test assets may omit this verifier only
when explicitly loaded with `AllowSyntheticForTests`.

WP-04 owns provider fallback, circuit breakers and its single JSON-mode repair.
WP-10 does not add a retry loop. The Brain's two provider-attempt allowance is
shared by repair and retrieval re-evaluation, reserving one configured call for
future generation. Timeouts, provider errors and invalid output yield FB-001;
cancellation, authorization failure and bundle mismatch abort processing. Failed
attempt usage remains observable. Ordinary traces and metrics contain only usage,
latency and fixed rule codes; `BrainOutput.Raw` is deliberately empty.

The gate implements BR-001–024, with PR-004 input-name checks and SF-004/005 safety
constraints. It drops unsupported references, rejects unknown catalogs, downgrades
unsupported reported slots, enforces question limits and declared refusal, filters
hypothesis visibility, and honors risk raises only. The lexical support branch
uses Unicode-normalized tokens and the specified 50% overlap threshold. It has no
unapproved lemmatizer, stopword list or semantic entailment model; borderline
paraphrases can be downgraded. Clinical evaluation of extraction remains required.
`Assets.Gate` implements the pure `ports.PolicyGate`; database-dependent permissions
default to denied there. `Service.Decide` supplies verified policy facts instead.

`Facts` authorizes hypothesis tags after cooldown checks, offer quotas/gaps, and
hypothesis voicing history. The SQL implementation reads existing WP-02 tables.
Missing/erroring facts deny the corresponding proposal. Hypothesis voicing stays
disabled in the supplied SQL adapter, as required by Final Architecture §11.1's
MVP deferral. No confidence, consent, or risk downgrade is inferred from a model.

The existing integrated orchestrator recognizes this Brain and permits one extra
retrieval with the authenticated owner, original turn/date constraints, exclusions,
normal/sensitive scope and one additional item (maximum seven total). Both Brain
invocations share one stage deadline. Hop failure removes retrieved context and
re-evaluates with `degraded=true`; a second request cannot loop. Risk escalation
precedes retrieval/generation. Exact recall never substitutes semantic memory.
The later Generator port receives only permitted views with epistemic metadata;
WP-07 validates every draft, regeneration and static fallback.

`ProposeDelta` translates validated understanding into WP-08 proposals for session,
thread creation, focus shifts, pause/resumption, slots and hypothesis signals.
Expected versions follow the actual target thread. Question text/count comes from
the accepted WP-07 report, never the model's plan. WP-08 revalidates transitions,
readiness and evidence; WP-02 alone commits assistant text and state atomically.
Safety FSM changes remain owned by WP-06. No physical deletion is introduced.

## Integration and content requirements

Use `composition.NewAITurnBrain` with the WP-04 client, SQL pool, the **same pinned
release-verified bundle** used by Safety/Validator/Assembler/Retrieval, and the
explicit WP-09 `UsePolicy`. Inject the returned service as `Dependencies.Brain`
into `orchestrator.NewIntegrated`. Configure total/Brain/model budgets coherently;
pass one turn-pinned bundle across all components. No live HTTP route is switched
by this patch; WP-11's Generator remains unimplemented.

FB-001 needs an approved `reflect_emotion` skill, `brain_fallback` rationale tag,
and `advice`, `claiming_recall`, `intervention`, `invented_recall` avoid tags. These
are integration requirements, **not additions to the shipped Knowledge Bundle**.
The bundle's decision policy supplies objective meanings; no new clinical rules,
techniques, diagnosis lists or final prompt text are embedded in Go.

A plan requiring a known memory correction cannot be rendered until WP-12 supplies
its synchronous correction operation. A protocol directive cannot be rendered
until WP-13 supplies resolution/consent handling. The integrated orchestrator
fails those turns before generation, preserving the admitted user-only record;
a crisis reroute always takes precedence. WP-08 currently drops new hypothesis
`propose` signals at persistence validation; they remain valid Brain proposals
for the future WP-12 authoring/commit path. No worker or later module is added.

The current `knowledge/` bundle is intentionally inert and fails Brain activation.
Tests use explicitly synthetic engineering fixtures, not clinician-reviewed
labels or evidence of clinical quality. Approved Brain prompts/policies/catalogs,
clinician-reviewed decision goldens, locale coverage (including Persian), provider
quality/latency evaluation and privacy/security release gates remain blockers.

## Verification

`go test ./...`, `go test -race ./internal/modules/ai/... ./internal/app/composition`,
`go build ./...`, `go vet ./...`, and the repository structure/contract checks.

`tests/integration/TestWP10SQLStateEvidenceCooldownAndIsolation` runs against an
isolated SQL Server 2022 database with `APP_ENV=local`, a local `DATABASE_URL`, and
`WELLBEING_INTEGRATION=true`. It exercises real WP-02/04/08/09/10 integration,
source isolation, exact recall, cooldowns, atomic finalization and preservation.
No external model network is used. Prompt goldens change only with explicit
`UPDATE_WP10_GOLDENS=true` in the Brain package tests.
