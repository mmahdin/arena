> Active conversation configuration is `ai_runtime.v4`: jev, unified, memory_writer,
> reflection and embedding. Migrate old v2/v3 operator files explicitly. Add the
> four-second JEV component and `budgets.turn.jev_ms`, and set the synchronous
> generative-call limit to two. Existing context/retrieval budgets stay unchanged.
> Apply additive SQL migration 0041 before running the new conversation code.

# Theia AI Core runtime settings (v2)

`config/ai-runtime.yaml` is the tracked synthetic example. Set `AI_CORE_CONFIG_FILE` to your operator-owned copy and keep secrets in environment variables.

```sh
mkdir -p .local
cp config/ai-runtime.yaml .local/ai-runtime.yaml
# Adjust models, provider/model fallback chains, budgets, and environment variables.
# From repository root, make api runs Go from apps/api:
export AI_CORE_CONFIG_FILE=../../.local/ai-runtime.yaml
make api
```

When using a root `.env` file, set `AI_CORE_CONFIG_FILE=../../.local/ai-runtime.yaml` there instead of exporting it. Exported shell variables take precedence.
The example's `bundleDirectory: ../../knowledge` is relative to the API process's working directory (`apps/api`). Use absolute paths when running elsewhere.

`llm.components` chooses the model and per-request parameters for **each** component: `safety`, `brain`, `generator`, `validator`, `memory_writer`, `reflection`, `embedding`. A component's `chain` is the ordered primary/fallback list; add a second model there to enable a fallback. `embedding` must remain a single model for vector compatibility. Shared retry/quota/breaker configuration is under `llm`. `budgets` contains operational stage/token/quota limits, and `retrievalTimeouts` contains retrieval deadlines. Some conservative per-turn constraints remain lower ceilings and are never overridden by the component config.

The Knowledge Bundle owns clinical knowledge, thresholds, router decisions, safety and psychological policy; it no longer owns `config/models.yaml`, `config/budgets.yaml` or retrieval timeouts. Internal read-only adapters provide runtime budgets to existing module loaders, without creating a second operator-editable config.

## Migration from v1

The previous `config/ai-runtime.local.example.json` / `ai_runtime.v1` is **not** accepted by this version. Copy the v2 example; transfer your real provider entries, environment variable names, encryption key references, explicitly approved consent/product-policy configuration, and model choices from your old runtime JSON. Place the previously shared tier's model in **each** appropriate component's `chain` (small -> brain and memory_writer; strong -> generator and reflection; classifier -> safety and validator). There is no need to edit Go to choose a different provider or model for Safety.

The v2 runtime file is required for *both* local development and releases; there is no local/production branching in model selection. **Knowledge release signing, clinical review and local synthetic gating are unchanged.** Any signed bundle must be rebuilt/reviewed/signed after removing executable model/budget artifacts; never reuse an old signature or approvals.

No database migration is needed for this config-only refactor. Keep the active runtime config private, do not commit API keys, and do not use the synthetic draft with real users.

### BO-09 read-only candidate Knowledge Bundles

Optionally add `bundleCandidates` (array of 1–5 absolute directory paths) to
`ai_runtime.v2`, alongside `bundleDirectory`. Candidates are validated once
at startup using the active bundle's release trust policy and public keys;
they are not loaded into runtime, made active, editable or exposed by path.
`GET /api/v1/admin/bundles` displays metadata and sanitized validation codes
for the pinned active version and these optional candidate slots. See
`docs/operations/backoffice-bundles-bo09.md`.


Active AI Core configuration is `ai_runtime.v3`: exactly `unified`, `memory_writer`, `reflection`, and `embedding` components. Set `unified_ms`, `unified_in`, `unified_out`; ordinary turns allow one call and zero regeneration/hops. Product loading rejects v2 until explicitly migrated. See `docs/ai-core/unified-memory-centric.md`.
