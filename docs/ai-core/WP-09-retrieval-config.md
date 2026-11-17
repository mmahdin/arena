# WP-09 Retrieval — configuration reference (`config/retrieval.yaml`)

The Go tunables live in `apps/api/internal/modules/ai/domain/retrieval/config.go`
(`retrieval.Defaults()`); this file is the reviewer-facing reference for the schema in
Contract Pack §8.5. Values below are the ones actually implemented and asserted by
`TestDefaultsMatchContractPack`.

```yaml
schema: retrieval.v1
defaults: { max_items: 5, min_score: 0.32, per_item_tokens: 60, hard_cap_with_hop: 7, total_tokens: 400 }
weights:  { similarity: 0.50, recency: 0.20, importance: 0.15, thread_match: 0.15 }
recency:  { half_life_days: 21 }
anaphora: { recent_window_days: 14, emotion_boost: 0.15, topic_boost: 0.10 }
temporal: { max_messages: 12, default_roles: [user], week_starts_on: monday, tz_source: user_state }
hypotheses: { include_statuses: [proposed, supported], require_thread_match: true }
novelty_filter: { skip_if_in_recent_messages: 8, similarity_gte: 0.92 }
timeouts: { total_ms: 300, per_strategy_ms: 200 }
candidates: { pool_limit: 300 }         # implementation addition: bounded pool (§7.3)
sensitivity: { exclude_highly_sensitive: true }
```

Implementation notes:

* `min_score`, `per_item_tokens` and `candidate pool` are **refusals**: they stop weak or
  oversized context from consuming the Brain's budget.
* Weights must sum to 1.0 ± 0.05; `per_strategy_ms <= total_ms`; the pool is capped at
  2000. Invalid configuration fails at construction, not per request.
* `require_thread_match: true` is parsed and validated, but the **strict** form (drop every
  hypothesis not linked to the focus thread) is *not* enforced in WP-09: doing so would
  contradict the default inclusive list `include_statuses: [proposed, supported]`. Today a
  hypothesis must be `inferred` and `proposed|supported`; it surfaces only through its own
  score, and an unresolved anaphoric reference drops hypotheses entirely. Deciding which
  behaviour launches is an explicit review item (see the dashboard "Unresolved").
* `tz_source: user_state` — the assembler (WP-08) normally supplies a resolved
  `TemporalRef`; the service parses only when it does not.
