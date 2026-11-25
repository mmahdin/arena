# Psychology Knowledge Bundle changelog

## 0.1.0 — 2026-11-18 — initial unreviewed development draft

- Replaced synthetic placeholder wording and empty catalogs with evidence-informed conversational Skill Cards, opt-in brief protocols, policies, prompts, boundary/safety assets, and static fallbacks.
- Added a working English lexicon and a validator lexicon used by the runtime; added scoped source-checked international help-directory metadata. No emergency service was presumed from language alone.
- Maintained `synthetic_only: true`, empty clinical approvals, and absent release signatures/checksums: **local synthetic-only**, no production approval or publication.
- Added 22 unreviewed synthetic golden candidate scenarios for core objectives, boundaries, and safety; not executed outcome evaluations.
- Added a loopback-only, explicit `AI_CORE_LOCAL_DRAFT=true` developer server option; staging/production signed-release gates and default behavior are unchanged.
- Documented missing locale/clinical review, local runtime setup, and integration limitations in `README.md`.

## Multilingual architecture amendment (2026-11-18)

- English psychological knowledge and instructions now support `en`/`fa` conversations without localized skill cards or translated policies.
- Added unreviewed Persian Safety/Validator lexicons, static crisis/boundary/objective fallback text and a Persian description of the existing directory resource.
- Explicit language/resource metadata preserves English-only structured protocols. The bundle remains synthetic-only, unsigned and clinically unapproved.

## Product-first conversation amendment (2026-11-18)

- English protocol goals and completion criteria now guide natural English/Persian dialogue. Offers use therapeutic intents; optional ratings, repeated difficulty and a prior decline no longer impose intake/wording loops in the shipped library.
- Explicit requests can start the first step with owned-message consent, while unsolicited offers still require agreement.
- Compact Brain rendering leaves room for recent context within the same token budget. Style warnings do not discard useful replies; actual crisis and production release gates remain.
- Added six multi-turn product-review scenarios. All content is still an unreviewed, synthetic-only development draft.


## Hybrid retrieval amendment (2026-11-21)

- `config/retrieval.yaml` is `retrieval.v2`, containing context limits only;
  removed five-route configuration and hand-weighted relevance settings.
- Runtime still supplies its existing retrieval timeout settings. Released bundles
  require rebuilding/signing with the new config schema; the development bundle
  remains synthetic-only and unsigned.
- See ADR 0014 and the Retrieval/Memory READMEs for exhaustive hybrid indexing and
  asynchronous embeddings. Psychological extraction and safety rules are retained.

## Native JEV amendment (2026-11-25)

- JEV instructions now evaluate individual native System One questions instead
  of requesting generated JSON. Go preserves the existing selection contract.
- Resource catalog/content and Unified/Safety/background prompts are unchanged.
