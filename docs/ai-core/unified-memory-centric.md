> Phase 2 background sections are superseded by ADR 0016 and
> `docs/ai-core/background-intelligence.md`: periodic batches replace per-turn
> extraction; the Phase 1 synchronous pipeline remains intact.

# Active AI Core: Unified conversation and JEV

The active specification as of 2026-11-24 is ADR 0015. Earlier Brain/Generator,
classifier, regeneration, and single-inference flow descriptions are historical.
This remains the existing internal synthetic-data build.

## One turn lifecycle

1. Gateway durably admits the canonical user message, checks owner/idempotency/FIFO,
   and grants the existing fenced user lease. Completed requests replay directly.
2. Pin one immutable bundle. Start personal-memory retrieval immediately, alongside
   the owned state snapshot and persistent baseline. No generative LLM precedes
   retrieval. Query embeddings are part of the existing hybrid search.
3. Build `wctx.v3` under the existing independent budgets: persistent context 700,
   active state 3000, retrieved evidence 1200, recent dialogue 1000 estimated tokens.
   The original current message remains unchanged. Recent messages are chronological,
   role-attributed and fenced before the current message even during recovery.
4. Expire prior recommendations, then call JEV with Working Context, selection
   instructions, Knowledge Cards and valid carry-over. Reverify private sources
   before disclosing context to JEV, as well as before Unified.
5. Go resolves selected resource IDs to pinned complete artifacts. Unified receives
   that content and the independently mandatory Core Policies, and makes one
   structured conversational call. JEV makes no user-facing or Safety decisions.
6. Resolve independent output sections. Accept valid proposals; record rejected
   proposals. Resolve dependencies before accepting dependent changes. Keep exercise
   transitions, actual consent, progress and delivered text consistent.
7. Revalidate and atomically finalize response, accepted state, immutable decisions,
   carry-over and background jobs through the existing SQL finalizer. A SQL failure,
   changed version or lost lease cannot publish a partial turn.
8. Existing workers execute queued analyses after commit. The user-facing response
   waits for durable scheduling, never for their retrieval or model processing.

The sole constructor is `orchestrator.NewUnified`. Brain/Generator modules and
parallel orchestration paths are removed. Historical typed records, trace examples
and frozen Safety/Validator regression code do not provide another turn pipeline.
There is no legacy conversational fallback, regeneration or translation call.
A Unified provider failure leaves the existing durable failed user-only turn;
authenticated retry/recovery retains the original request and lease contracts.

## Dual-query personal-memory retrieval

The two deterministic inputs are:

- The original current message.
- Up to four previous visible messages, oldest first and role-attributed, followed
  by the current user message. Each previous excerpt is bounded to 600 characters.

With no preceding dialogue, the identical second query is omitted. No model rewrites
or generates either query. Both embeddings run under the existing shared retrieval
and embedding timeouts through the existing Shared Embedding Service. One
owner/policy-filtered corpus scan supplies compatible vectors and source metadata.
BM25 is temporarily disabled for both queries and contributes no RRF vote.
Current/contextual cosine rankings and the existing independent thread/temporal
signals enter unchanged reciprocal-rank fusion, bounded relevance/redundancy
selection and direct evidence expansion. The lexical index and standalone BM25
implementation remain intact; no new flag, service or reranker is introduced.

A failed embedding contributes no semantic ranks and has no BM25 fallback.
A surviving current/contextual query remains usable; if all queries fail, only
independently valid thread/date signals may supply candidates, otherwise results
are empty and degraded. Per-query strategies and failures describe actual work.
Sources with pending/missing/stale vectors cannot create lexical search matches. An unavailable history
read cannot introduce another owner's context. Exact source identity/revision is
merged; independently reported evidence is retained even when wording is identical.
The existing token, sensitivity, ownership, visibility and chronological boundaries
still apply. Exact corpus ranking remains linear; no production-scale latency claim
is inferred from synthetic tests.

## Working Context and overlap

Working Context includes explicit profile/onboarding information, persistent memories
and hypotheses; current Session, focus/unresolved Threads, Formulations and Open Loops;
exercise state and allowed operations; saved response preferences and existing risk
state; retrieved evidence; chronological recent messages; the current user message;
and valid prior recommendations. Stable budgets are reserved independently.

The same source is not repeated in persistent/retrieved context, active/retrieved
thread state, or recent/retrieved messages. Full active formulation content is not
repeated in the formulation list. Hypotheses represented by persistent memory retain
thread links through `hypothesis_refs`. Independent sources are not deduplicated by
text similarity. Original message roles, source IDs/versions/conversations/timestamps,
evidence links, applicability, epistemic labels and status remain explicit.

Initial query text/ranking evidence is kept in Inspector retrieval nodes rather than
repeated inside model-facing context metadata. Existing risk/consent constraints
remain operational truth. Go no longer computes psychological information-need scores
or protocol fit scores; Unified judges semantic suitability from supplied evidence,
state, selected knowledge and policies.

## Knowledge Cards and JEV

`knowledge/cards.yaml` is `knowledge_cards.v1`. Cards contain an ID, category, name,
concise description, purpose, applications, limitations, prerequisites and `content_id`
for the complete artifact. Categories are `skill`, `framework`, `protocol`, `exercise`
and `other`. All 18 original skills, seven protocols/practices, two formulation
frameworks and the authored offer resource remain available. Full artifacts are
preserved. Additional psychological resources may live under `resources/psychology/`.

JEV sees only cards, context and expiring recommendations. Its structured output is
`jev_selection.v1`: `selected_resource_ids` (at most four) and `topic_continues`.
The active selector uses OpenCode `jev-1.13-free` via native System One
`state`/`questions`. Go converts typed Noul answers into this unchanged downstream
contract; see the [Knowledge README](../../apps/api/internal/modules/ai/application/knowledge/README.md).
It uses one small configured model call, a four-second default timeout, 500 output
tokens and one physical attempt. Go rejects unknown IDs and loads selected complete
content under a 3500 estimated-token ceiling. Unified verifies that supplied content
exactly matches its pinned artifacts. JEV does not use hybrid/embedding search for
knowledge selection and has no tools, memory writes or state access beyond the
read-only supplied context.

Selected protocol YAML is passed once through Unified's selected knowledge blocks.
WorkingContext retains only the current protocol instance/lifecycle state; it has
no duplicate `shortlist` or `protocol_knowledge` fields, and there is no
`protocol.SelectKnowledge` preparation stage. During proposal resolution, the
server passes JEV-selected content IDs directly to the protocol engine. New offers
must reference a selected protocol definition and still pass persisted eligibility,
language, ownership, risk and consent checks. Lifecycle transitions and atomic
finalization are unchanged. The engine's internal eligibility briefs retain their
authored `est_minutes` and `fit_tags`; they are not sent as another model catalog.

Provider/timeout/invalid-output/budget failures degrade to still-valid recommendations
and the definition of an already-active exercise, or an empty optional selection.
An active exercise's definition is retained as an operational prerequisite. All Core
Policies remain independent: `communication_policy.md`, `boundary_policy.yaml`,
`safety_policy.yaml`, the existing crisis resources and Unified's existing Safety
instructions are always supplied. Neither a failed nor malicious selector can remove
them or introduce alternate content.

Recommendations last at most three accepted turns and 72 hours. They retain an
applicable focus-thread scope. JEV treats them as suggestions and can drop them on a
topic change. Failed JEV calls never refresh their fallback selections. No manual
relevance scoring is used. The common hint shape can later be adopted by background
models; those models' intelligence/output schemas are not redesigned here.

## Unified output and independent validation

`unified_turn.v2` has four sections and a version discriminator:

- `reply`: text, conversation intent/objective, actually used source IDs, and
  dependencies on operations the response requires.
- `state_proposals`: bounded `{id, kind, value, depends_on}` items for session,
  thread, protocol, loops, preferences and the existing risk proposal.
- `background_requests`: at most two analytical requests with an ID, processing
  kind, objective, topic, up to three search queries, evidence IDs and state
  dependencies. Supported kinds are `memory_intelligence` and `reflection`.
- `future_hints`: optional topics/resource IDs with a one-to-three-turn lifetime.

The reply envelope remains strict about malformed/duplicate JSON, version, required
fields, vocabulary and used sources. Optional items have separate schema, reference
and operational validation. Unknown or invalid items are reported without dropping
an independent valid response. Duplicate proposal IDs/kinds cannot become ambiguous
dependency targets. Missing/rejected/cyclic state dependencies do not execute.
A response depending on a rejected operation is withheld. Exercise actions must
match the exact resulting status and step before publication. Natural-language
claims are not mechanically provable: the model must declare its dependencies, and
the backend always checks proposed exercise transitions conservatively.

The existing risk transition policy is unchanged. Finalization still revalidates
accepted state against current versions and rejects concurrent/stale writes. This
is distinct from rejecting an independently invalid optional proposal at inference.

## Existing background intelligence, new analytical inputs

A request is only a proposal. Go validates its bounds, current owned evidence and
accepted state dependencies. Request text and queries are encrypted into existing
Memory Writer or Reflection tasks. Dedupe keys derive from the admitted turn and
request ID. The same finalization transaction schedules accepted work and records
rejections; no analytical search or background model runs in that transaction.

When a worker claims the task, it executes the bounded explicit queries through the
existing hybrid retrieval and prepares provenance-bearing historical context beside
its existing recent messages, owned memories and evidence. The objective and search
results are supplied as untrusted analytical context; model output must still meet
the original evidence/target/epistemic rules. This preserves the existing Memory
Writer and Reflection intelligence rather than inventing a new agent.

Sources are reverified before model disclosure and after checkpoint recovery.
Research source versions are checked under transaction-held read locks before
memory/reflection writes. Existing user revisions, target/thread/formulation
versions, durable receipts, job leases, quotas, bounded retries and encrypted
checkpoints continue to govern actual commits. Old results cannot overwrite newer
state through a recovered checkpoint. Failures/statuses remain visible in `ai_jobs`
and in worker Inspector nodes.

Routine Memory Writer review is still planned for every completed user turn, even
with no model hint/request. If planning temporarily fails, a bounded worker sweep
finds up to eight recent completed turns lacking a routine review job. It scans only
the past 24 hours, respects current policy and existing job attempt limits, and does
not schedule proactive user interaction or an unrestricted historical backfill.

## Configuration, migration and observability

`ai_runtime.v4` has exactly `jev`, `unified`, `memory_writer`, `reflection`, `embedding`.
The existing `MODEL_*` credential/provider naming is retained. Examples use the
existing strong model for Unified and the existing small-model family for JEV;
actual endpoint availability and costs remain operator configuration. The maximum
synchronous generative-call budget is two, with one attempt each and no regeneration.
Working Context budgets and the retrieval deadline remain unchanged. Old v2/v3
runtime files are rejected and need explicit conversion.

Apply additive migration `0041_unified_turn_decisions.sql` before deploying v4.
It adds append-only decisions/carry-over with a composite owner FK, not a second
state writer. Response, accepted state, decisions and jobs commit together.
A code rollback can retain the table. Existing migrations and original/derived data
are unchanged. Old queued jobs remain pinned to their original bundle hashes.

The local `local_ai_trace.v1` envelope remains compatible. Inspector shows query
inputs/ranks/fusion, final context, card catalog, JEV input/output/degradation,
selected complete resources, Unified prompt/output, accepted/rejected proposals,
future recommendations, queued requests, worker retrieval and execution status.
Existing physical-attempt accounting retains model, calls, token usage, configured
cost estimates and latency. Unknown pricing remains unknown. Ordinary logs/metrics
contain only allowlisted metadata, never prompts, messages, queries or hint text.

The Explorer remains offline and opens user-selected files. It has no backend
connection or watcher. `jev_unified_turn.json` is a generated scripted SQL fixture;
older v1 examples remain importable.

## Validation

See `docs/quality/unified-refactor-2026-11-24.md` for exact commands and results.
Tests use synthetic fixtures and isolated disposable SQL Server 2022 databases.
No live-provider conversation, quality assessment, Safety redesign or public release
is implied by passing these engineering checks.

Embedding computation is now supplied through the process-local
[Shared Embedding Service](shared-embeddings.md). Current/contextual/background
query construction and ranking are unchanged; indexing workers use the same
instance for identical owner-scoped vector identities. Caller retrieval deadlines
remain independent of bounded shared provider execution.
