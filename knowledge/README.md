## Selective knowledge in Unified

`cards.yaml` describes all existing psychological resources without copying their
full content into every prompt. JEV selects IDs directly from cards; Go loads the
corresponding complete artifacts. Keep every skill/protocol/formulation framework
covered by a card. Other resources may use `resources/psychology/*.md` or `.yaml`.
Cards need category, description, purpose, applications, limitations, prerequisites
and a full `content_id`. Existing release checksums/review requirements still apply.
Core Policies and crisis resources are always supplied independently and must not
become selectable cards. See ADR 0015 and the active conversation specification.

# Theia Psychology Domain Knowledge — kb@0.1.0 (development draft)

**Status: unreviewed, NOT clinically approved, NOT suitable for real users or production.**
`synthetic_only: true` is the existing AI Core's **non-production execution gate**. It does not mean the content below is placeholder text: these are substantive *drafts*, for local synthetic conversations, integration testing and evaluation. Do not interpret lint success, successful model responses, or source references as clinical validation.

## Coverage

- 18 conversational Skill Cards: listening, empathy, MI-style elicitation, collaborative exploration, basic CBT/ACT psychoeducation, grounding invitation, small actions, repair and boundaries.
- Seven opt-in, brief structured protocols: CBT thought record, behavioral activation, CBT worry sorting/problem solving, ACT defusion, ACT values-to-action, sensory grounding, and MI-informed ambivalence exploration. **None is a replacement for clinician-delivered treatment.**
- Decision and communication policies, safety and boundary policies, read-model slots, conservative formulations, consent-aware memory/reflection prompts, safe fallbacks, localized safety regexes, validator patterns, resources, and protocol-offer copy.
- English knowledge (`knowledge_language: en`) supports `en` and `fa` synthetic conversations. Saved response preferences are independent of input language. Persian safety, crisis, boundary and fallback resources are **unreviewed development content**. English CBT/ACT protocol goals can be delivered naturally in either conversation language; see the [product-first amendment](../docs/ai-core/product-first-conversation.md). See [multilingual architecture](../docs/ai-core/multilingual-conversations.md).

## Run locally (no publication, approval or signing)

From `apps/api`:

```sh
go run ./cmd/aicore-bundlelint -dir ../../knowledge -core-version 1.0.0
go test ./internal/modules/ai/bundle ./internal/modules/ai/application/brain ./internal/modules/ai/application/generator ./internal/modules/ai/application/protocol ./internal/modules/ai/application/memory ./internal/modules/ai/application/reflection ./internal/modules/ai/safety ./internal/modules/ai/validator
```

For **isolated synthetic tests**, load `bundle.Read(ctx, root, bundle.Options{CoreVersion: "1.0.0"})`, or activate with `bundle.NewRegistry(bundle.Options{CoreVersion: "1.0.0"}, true)`; module loaders need `AllowSyntheticForTests: true`. `composition.NewAIInternal` also has `AllowSyntheticForTests: true` for isolated injection with fake clients / fixtures. Do **not** use those test options with real user data.

To run this unsigned draft through the **local backend development server**, retain your existing `ai_runtime.v2` deployment settings (SQL Server, model/provider credentials, encryption keys, explicit privacy grants, and worker ID). Point its `bundleDirectory` to the absolute path of this patched `knowledge/` directory. Enable the *explicit* draft flag and developer-only listener:

```sh
export APP_ENV=local
export HTTP_ADDR=127.0.0.1:8080
export ALLOW_DEV_LOGIN=true
export DEV_SESSION_SECRET='REPLACE_WITH_YOUR_OWN_RANDOM_SECRET_AT_LEAST_32_CHARACTERS'
export AI_CORE_CONFIG_FILE=/absolute/path/to/ai-runtime.yaml
export AI_CORE_LOCAL_DRAFT=true
# Also configure the local SQL Server and AI runtime provider/key variables
# required by your existing deployment settings.
go run ./cmd/api
```

`AI_CORE_LOCAL_DRAFT` is **false by default**. It is accepted only for `APP_ENV=local`, enabled developer login, an explicit runtime config file, and a loopback-only HTTP listener (`127.0.0.1`, `localhost`, or `::1`). It loads only a `synthetic_only: true` bundle without a signature. Model selection behaves the same in local and production; release verification still requires reviewed, signed Knowledge. It does not disable database, encryption, explicit consent/policy, safety or runtime loader requirements. The ordinary deployment path continues to require signed, reviewed releases; staging and production cannot turn on the draft option. Send *synthetic English or Persian* test messages. This is not authorization to expose a mental-health service or process real patient disclosures.

If your development setup launches the server through Docker Compose or an IDE, apply the same environment variables there. The flag does not provision SQL Server or model credentials.

To confirm the production gate is retained (expected failure):

```sh
go run ./cmd/aicore-bundlelint -release -dir ../../knowledge -core-version 1.0.0
```

## Validation status (2026-11-18)

- Passed the full Go suite, structure/OpenAPI checks, bundle lint and targeted AI Core race tests. Full-bundle multi-turn tests use the real application modules with scripted providers for Persian ACT, English CBT, optional ratings, pause/resume/decline, provider fallback and crisis routing.
- Six new product scenarios supplement the baseline goldens. Their naturalness, usefulness, Persian quality and refusal rubrics require actual output and calibrated human/native review; script execution and packaging checks do not certify those qualities.
- No live-model response evaluation, native/clinical approval or live SQL Server smoke result is claimed. The opt-in database tests remain separate. All content remains synthetic-only, unsigned development material.

## Design decisions

The data structure follows Contract Pack v1.2 §8 and the installed loaders. Protocols use authored goals, risk boundaries, opt-in, stopping rules and defined success predicates for low-intensity, non-trauma-processing support. Clinical labels are hypotheses at most, never diagnoses. Both formulation frameworks require evidence; speculative fields stay tentative and unvoiced. Memory must not infer sensitive facts or override consent. Crisis messages do not promise active monitoring, dispatch, confidentiality, or emergency intervention. The L0 matcher is only a screen, not a comprehensive suicide-risk assessment.

Crisis-resource entries contain a **directory, not a direct crisis line**: Find A Helpline helps locate country-specific lines. It is represented as `chat` because the v1 schema only permits `hotline|text_line|emergency|chat`; this taxonomy is a known limitation. The script and documentation explicitly say this is a directory and remind the user to contact local emergency services in immediate danger. Source was checked from the publisher on 2026-11-18; that is *web/source verification*, not clinical approval or region-specific helpline certification. Do not make a release without real verified region-specific contacts, reviews, and localization.

### Evidence basis (adapted, not replicated treatment manuals)

- NICE NG222, *Depression in adults: treatment and management* (2022): structured CBT, behavioral activation, collaborative decisions, self-help boundaries. https://www.nice.org.uk/guidance/ng222
- NICE CG113, *Generalised anxiety disorder and panic disorder in adults: management*: CBT-oriented guided self-help and graded care. https://www.nice.org.uk/guidance/cg113
- WHO, *Doing What Matters in Times of Stress* (2020): grounding, unhooking, present-moment and values actions. https://www.who.int/publications/i/item/9789240003927
- Association for Contextual Behavioral Science, *ACT*, psychological flexibility, acceptance/defusion and values: https://contextualscience.org/act
- SAMHSA, *Enhancing Motivation for Change in Substance Use Disorder Treatment*, TIP 35 (2019), Chapters 3 and 5: OARS, reflective listening, autonomy and ambivalence. https://www.ncbi.nlm.nih.gov/books/NBK571064/
- ThroughLine, *Find A Helpline* directory: https://findahelpline.com/about

These references support the *general approaches*, not an efficacy claim for AI-delivered care. There are no invented outcome rates or diagnostic cutoffs.

## Open limitations before any clinical or production use

1. Independent clinical/safety review, product scope and evidence annotations per artifact; actual clinician approvals/checksums/signature absent by design.
2. Native Persian and clinical review of the new development lexicons/templates, live-model fluency and semantic evaluation, dialect/romanized coverage, review of conversational protocol delivery, and region-specific verified crisis resources.
3. The baseline synthetic golden *candidate* scenarios plus six multi-turn product scenarios cover the fifteen Brain objectives plus crisis/boundary cases, but expected results are not clinician-reviewed or model-executed. Use the product scenarios for native/human review of naturalness, usefulness, unnecessary refusals and continuity. Packaging/golden coverage is not an evaluation pass.
4. Output validator uses conservative regexes; safety lexicons are not complete clinical triage. A draft model could fail in either direction. Test safety fallbacks and prompt-injection robustness with human review.
5. Reflection's current selector rejects tags matching **both** CBT and five-P frameworks; mixed tags should be assessed when prioritization is implemented. The bundle keeps the framework tag sets disjoint for single-tag fixtures.
6. Full local HTTP end-to-end results still depend on provisioning the DB schema, provider keys/configured models, and synthetic users. The draft switch enables the code path; it does not imply that a live HTTP conversation or a model-output evaluation has been executed.
