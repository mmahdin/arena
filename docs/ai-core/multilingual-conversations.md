> **Runtime superseded (2026-11-23):** [Unified memory-centric architecture](unified-memory-centric.md) is the active runtime. Separate Brain/Generator, safety/router/output-classifier calls and regeneration described here are historical. Persisted contracts and historical Inspector examples remain supported.

# Multilingual conversations with English knowledge

Implementation amendment, 2026-11-18. Owner: AI Core. This supersedes the locale
coupling in Contract Pack v1.2 §§8.2, 8.11, 10.3 and 11.3. The Go modular monolith,
consumer-owned ports, admission/finalization transactions, reviewed production
bundle requirements and existing LLM call budgets remain binding.

## Three independent language concepts

| Concept | Authority | Consumers |
| --- | --- | --- |
| Knowledge language | `bundle.knowledge_language`, currently `en` | Brain, Generator, policies, skills, frameworks, protocol definitions |
| Input language hint | `TurnRequest.Locale` → `TurnView.Lang` | Safety, retrieval, input metadata, routing |
| Response language | Saved `ai_user_state.prefs_json.language`, resolved each turn | Generator, Validator, crisis planner, fallback selection, protocol eligibility |

The authenticated AI preferences/controls API already persists `PrefsView.Language`
and increments the user state version. No new profile store, migration or API
field is introduced. `UserState.Locale` is historical request metadata maintained
at admission; it must not override the saved preference or this turn's input tag.
User-derived messages, slots, memories and evidence retain their own language;
English knowledge does not require translating stored personal content.

`contract.ResponseLanguage` deterministically resolves the response language:

1. A supported saved preference wins, even if the input is in another language.
2. A missing/blank preference uses a supported input hint.
3. A nonempty unsupported preference, or no usable hint, selects `fa`.

Supported tags have base `en` or `fa`; case and regional/script subtags are
canonicalized to the base, e.g. `FA-ir` → `fa`, `en-US` → `en`. This is tag
normalization, not text language detection. The existing product ingress defaults
an omitted request locale to `fa` and rejects malformed BCP 47 tags. A valid but
unsupported input tag still receives Safety's existing missing-coverage error and
conservative degraded risk floor; Orchestrator aborts generation on that coverage
error. Selecting a response language never asserts
that input safety coverage exists.

| Saved preference | Input hint | Response |
| --- | --- | --- |
| `fa` | `en` | `fa` |
| `en` | `fa` | `en` |
| missing | `en` | `en` |
| missing | `fa` or unsupported/missing | `fa` |
| `de` | `en` | `fa` |

Assembler places the resolved language in `WCtx.Prefs.Language` without changing
`WCtx.Turn.Lang`. Orchestrator passes that response language to crisis planning and
all fallback paths. Generator and Validator use the same resolver when invoked
independently. A saved change is read on the next turn; existing state-version
checks fence concurrent changes before finalization. Completed idempotent turns
continue returning their original persisted response.

## One English knowledge bundle

```yaml
knowledge_language: en
conversation_languages: [en, fa]
language_resources:
  en: {review_status: development, protocols: true}
  fa: {review_status: development, protocols: true}
synthetic_only: true
```

`conversation_languages` describes resource coverage, not a translated knowledge
catalog. Each declared language requires Safety L0 and Validator lexicons, crisis
resource entries, crisis templates, boundary scripts and objective fallbacks.
`language_resources` declares development/reviewed status and protocol availability; ordinary protocol knowledge is not a localized script.
The linter no longer requires translated communication/decision policies or skill
cards. Those assets and all system instructions remain English.

The runtime loaders share manifest validation and load lexicons by conversation
language, including Memory and Reflection consumers. Generator always consumes
the base English skill card. Its bounded dynamic metadata block
`<conversation_language knowledge="en" response="fa"/>` supplies the selected
language. The English system prompt requests idiomatic Persian, appropriate tone,
psychological meaning, uncertainty, consent and agency, without literal English
phrasing or assumptions about nationality, religion or location. Brain keeps
schema keys/enums English and understands user evidence directly. No translation
stage, translated knowledge catalog or additional model call is introduced.

Legacy `locales` manifests remain readable with implicit English knowledge and
their original resource coverage rules. If both language-list fields are supplied,
they must agree. New optional manifest fields are omitted from canonical JSON
when absent, preserving the signature payload for old manifests. Older cores
with strict YAML decoding will reject the new fields; deploy the updated core
before using a multilingual manifest. Existing `brain.v1`, `wctx.v1`, public HTTP
shapes and port signatures are unchanged.

## Safety and validation

Safety scans all loaded language rule sets in deterministic order, regardless of
the input hint, for both the current text and the existing bounded recent user
history. An English hint or preference cannot suppress Persian L0 detections.
Persian rules cover synthetic examples of suicidal statements, self-harm,
violence, abuse, medical emergencies and substance crises. Persian L0 rules do
not opt into contextual downgrades. Existing L1 is instructed to assess English,
Persian and mixed text directly; L0 remains a floor, and classifier failure retains
`max(L0, elevated)`. Unknown input coverage remains an error, never risk `none`.

Matcher-only normalization folds Arabic yeh/kaf variants, Persian/Arabic digits and
zero-width joiners. It preserves rune counts for L0 evidence spans and leaves
stored messages and displayed prose untouched. Validator scans every loaded
language's prohibited-output patterns, including on its private stream buffer;
response-specific questions and hedges use the selected language. Persian digits
cannot evade contact or medication checks. OV-013 rejects clear script-dominance
mismatches. The existing optional Validator classifier still fails closed if it
is configured and unavailable; it adds no mandatory translation/quality call.

Crisis text and fallbacks are static language resources, not model translations.
The planner selects by response language, retains freshness/resource checks and
fails when the needed template is absent. Validator still requires the exact
pinned crisis template and whitelisted contacts; a failed model draft can only
fall back to text that passes the same Validator. Language does not establish
country: the Persian resource entry uses the existing international directory,
with localized description, rather than guessing an Iranian hotline.

Missing language lexicons fail loading. Missing templates/resources fail bundle
lint or crisis planning; there is no English-template substitution for a Persian
response. Production loaders still reject synthetic content. New explicit
language manifests additionally require review hashes for all Safety/Validator
lexicons, templates, crisis resources, classifier prompts and forbidden-claim
patterns. A `review_status: reviewed` label alone cannot authorize production use;
checksums, matching clinical approvals and signed registry release gates remain
required.

## Protocol boundary

The [product-first amendment](product-first-conversation.md) supersedes the initial
English-only protocol restriction. Declared English and Persian conversation
coverage can realize the same English protocol definitions by therapeutic intent,
without translated libraries or literal `must_say` checks. Generator, Engine and
Validator retain actual risk, owned consent, graph and resource/review boundaries.
A response language explicitly disabled in the manifest still lacks protocol
coverage; ordinary language normalization never bypasses that setting.

## Verification and remaining limits

Synthetic unit/integration tests cover language fallback/normalization, English
knowledge with Persian and English preferences, Persian input, changed preferences,
Generator prompts and scripted outputs, Persian and mixed-input L0 escalation,
classifier outages, Persian output violations and digits, static crisis validation,
protocol restrictions and complete admitted-turn processing through the existing
Assembler, Safety, Brain, Generator, Validator and finalizer. Prompt goldens include
the language metadata block. Tests use scripted providers, not real disclosures.

All new Persian lexicons, crisis/fallback/boundary copy and resource descriptions
are **unreviewed development content**. English content also remains unapproved;
the entire bundle stays `synthetic_only: true`. Native Persian and clinical review,
regional resource verification and adversarial evaluation remain prerequisites
for any real-user release. No new verified country-specific hotline is claimed.

Regexes cannot provide complete clinical detection, handle every colloquial or
romanized Persian expression, or reliably resolve negation. Script checks cannot
prove Persian versus another Arabic-script language or certify naturalness.
Existing lexical provenance/hedge checks cannot establish cross-language semantic
entailment of recalled claims. The existing optional classifier is a backstop,
not clinical approval. Live-model fluency, cultural suitability and semantic
fidelity still require native-speaker evaluation; passing scripted tests does
not prove those qualities. No live SQL Server or remote-model evaluation is
implied by these integration tests.
