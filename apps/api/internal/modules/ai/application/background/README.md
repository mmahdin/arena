# Background Intelligence

Memory Intelligence and longitudinal Reflection share this bounded executor on the
existing Memory/Reflection queues. It owns model contracts and semantic proposals,
not SQL or HTTP. `IntelligenceRepository` implements preparation and the existing
memory transaction; workers retain lease/retry/quotas and encrypted checkpoints.

See ADR 0016 and `docs/ai-core/background-intelligence.md` for triggers, scopes,
epistemic boundaries, coverage, knowledge supply and diagrams. Runtime configuration
is supplied by `config/ai-runtime.yaml` through the runtime bundle overlay; the authoritative output schema
is `contract/schema/background_intelligence.v2.json`.

Periodic review selects only unreviewed user-message revisions and reserves them
durably before execution. Already-covered pending batches skip inference. Compatible
reported-memory status labels and supplied framework aliases are normalized before
resolution. Rejected substantive operations fail visibly with their checkpoint
retained; coverage and job completion occur only with the atomic persistence commit.

The synchronous Phase 1 pipeline and Safety stay unchanged. No routine per-turn
extraction, selector model, autonomous agent or background-request chain is added.
