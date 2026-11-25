> **2026-11-23 active runtime:** See `docs/ai-core/unified-memory-centric.md`. Production uses one unified conversational model with immediate retrieval and independent baseline context. Brain/Generator and classifier runtime descriptions below are historical; Memory Writer/Reflection and persistence remain active.

# WP-07 — Output Validator

The `validator` package implements `ports.Validator` from Contract Pack v1.2 (§4.7, §6, §10.3). It uses only the deterministic, version-pinned WP-03 Knowledge Bundle plus an **optional** WP-04 structured classifier. All output remains in the private Orchestrator buffer until accepted; neither rejected drafts nor regeneration notes are persisted as messages.

## Runtime wiring

1. Load an **approved, non-synthetic**, release-verified WP-03 bundle (`bundle.Snapshot`) from the registry. The same bundle version and prompt-hash map must be pinned for WP-06 Safety and WP-07 before turn processing.
2. Provide one clinically reviewed `lexicons/validator.<locale>.yaml` per locale declared in `bundle.yaml` (schema `validator_lexicon.v1`), complete with hedging/question patterns and every mandatory matcher group. Do **not** reuse the synthetic `testdata/validator.en.yaml` file.
3. Configure reviewed `config/thresholds.yaml.validator` values for `directive_advice`, `certainty_about_inference`, `dependency_encouraging`, `invalidating` if using the optional classifier. Its approved prompt is `prompts/validator_classifier.system.tmpl` and it uses the existing WP-04 `ports.Client.Structured` tier.
4. Construct `validator.New(ctx, bundle, validator.Options{Classifier: ...})` and inject as `Dependencies.Validator`. In `NewIntegrated`, bundle matching is enforced at turn admission, and classifier calls receive authenticated `CallMeta` from the pinned context.
5. `StreamGuard` is per-generation and must be reset when a private provider stream is reset. `Finalize` performs the full rule registry, optional classifier scoring, extracted question/claim metadata, and action decision. `WithFinishReason` is used by the Orchestrator; an eventual Protocol Engine must bind the actual reviewed `must_say` text through `WithProtocolRequirements`.

## Failure behavior

The Orchestrator makes at most **one** regeneration, with rule-code-only correction notes. A `PLAN_UNSAFE` marker causes immediate reroute to the already reviewed reflective fallback (or static crisis template). The static fallback is also validated. If no approved fallback passes, WP-02 retains the durable **user-only failed turn**; no unsafe assistant text is committed. Regeneration, fallback, and crisis validation remain under the original cancellation/deadline and lease fences. No module performs hard deletes.

## Clinical, release, and engineering blockers

- Existing `knowledge/` assets are explicitly **synthetic** and do not include reviewed WP-07 lexicons/thresholds. This patch intentionally adds **test-only** matchers; it does not invent psychology policy or enable production launch.
- Release WP-03 requires `clinical_review.required_for: [ ..., validator ]` with approval hashes for `lexicons/validator.*.yaml`, `config/thresholds.yaml`, and `prompts/validator_classifier.system.tmpl` when WP-07 rules are shipped. Signature, verification dates and clinician review must be completed before activation.
- Rule-based language detection reliably handles script mismatches (e.g. Latin vs Persian/Arabic), but **cannot reliably distinguish languages sharing the same script**, such as English and German. Clinician-approved language classification and multilingual golden sets are required before releasing those locales. Lexical claims and inferred-memory matching are conservative backstops, **not** complete semantic entailment checks.
- Crisis replies currently require an **exact reviewed rendered template**, rather than guessing whether arbitrary generated text contains all safety elements. This may increase fallbacks and should be reviewed clinically. Optional classifier latency/extra LLM calls must also be measured against `budgets.yaml` before launch.
- WP-08's existing finalizer derives pending-question state from punctuation. WP-07 extracts a richer `QuestionCount/QuestionText` in its report, but the frozen `PreparedTurn` does not transport that report to WP-08. A separately reviewed contract/integration change is needed for complete consistency; it is not hidden inside this patch.

Test fixtures under `testdata/` are synthetic and are **not part of the shipping Knowledge Bundle**. Run `go test ./internal/modules/ai/validator ./internal/modules/ai/application/orchestrator ./internal/modules/ai/bundle` with genuine module dependencies in CI; no temporary offline shims or synthetic-test flags may be used in production.
