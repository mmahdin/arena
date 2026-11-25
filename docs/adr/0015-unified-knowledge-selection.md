# ADR 0015 — Unified conversation with JEV knowledge selection

Status: implemented for the existing internal synthetic build, 2026-11-24.

The owner requested a single conversation architecture with immediate dual-query
personal-memory retrieval, selective psychological knowledge, optional independent
state/background proposals and short-lived carry-over. This supersedes the earlier
single-inference contract and all Brain/Generator execution descriptions. It does
not authorize changing Safety, public deployment or processing real disclosures.

The only message-processing path is Gateway → pinned bundle → hybrid retrieval
and state/baseline reads → Working Context → JEV → Unified → independent proposal
resolution → the existing atomic Conversation finalizer. Brain/Generator modules,
composition factories, executable evaluation wrappers and old orchestration paths
are removed. Historical records, schema decoders, Inspector examples and frozen
Safety/Validator regression assets remain readable; they cannot select another
conversation execution path. Runtime configuration accepts only `ai_runtime.v4`.

Retrieval uses the original message and, when history exists, a deterministic
role-attributed combination of that message with the four preceding messages.
Both embedding requests share the existing retrieval deadline. Cosine and BM25
rankings for both queries enter existing reciprocal-rank fusion over one authorized
corpus scan. Deduplication uses source identity/revision; similar or identical text
from different evidence sources is not silently deleted. No retrieval query model,
reranker, autonomous agent or iterative search loop is added.

`cards.yaml` describes every existing skill, protocol/exercise and formulation
framework, plus optional other resources. JEV sees descriptions, context and valid
recommendations. It returns at most four catalog IDs in one bounded structured
call. Go loads their pinned full content. Unknown IDs, invalid JSON, provider
failures and timeouts degrade to bounded still-valid carry-over and an active
exercise's definition, or no optional knowledge. Core Policies are always included
in Unified independently. JEV never receives state-writing or Safety authority.

`unified_turn.v2` has `reply`, `state_proposals`, `background_requests` and
`future_hints`. Optional items are structurally validated independently. State
dependencies are processed before dependent proposals; missing/cyclic/rejected
dependencies cannot execute. A reply that depends on a rejected operation is not
published. Exercise text/expected transitions still have to match actual consent,
graph resolution and accepted state. Database conflicts and lost leases still
prevent publication. The existing risk transition rules and Safety instructions
are preserved.

Recommendations expire after at most three accepted turns or 72 hours and are
scoped to an existing focus thread where applicable. JEV can discard recommendations
on a topic change. A degraded selector does not renew its own fallback selections.
This uses bounded lifetimes, without a manual relevance score.

Explicit analysis objectives, topics, up to three memory queries and evidence IDs
are encrypted into existing Memory Writer/Reflection jobs. Only workers execute
those queries, after finalization. They reuse the existing hybrid retrieval, access
policy, lease, quota, checkpoint and commit infrastructure. Research sources are
reverified before model disclosure and before commit. Source revisions are fenced
inside the existing memory/reflection transaction. Reflection checkpoints also
retain their original thread/formulation versions. Background results cannot
replace newer state using an old checkpoint. No internal intelligence redesign is
introduced; common hint contracts leave room for later background recommendations.

Routine extraction continues on every completed user turn even without model
requests. A bounded periodic sweep catches recent completed turns without a routine
job; it does not bypass attempt limits or backfill old history.

Migration `0041_unified_turn_decisions.sql` adds append-only owner-scoped decisions
and carry-over, committed with reply, accepted state and jobs. Existing migrations
and original/derived records are retained. Apply the additive migration before
deploying v4. Reverting application code does not require dropping the table.
Old configuration and old queued bundle hashes must not be silently reinterpreted.

Source validation, structured contracts and scripted tests establish engineering
properties. Separate Safety scenarios and live model quality/cost measurements are
outside this change.
