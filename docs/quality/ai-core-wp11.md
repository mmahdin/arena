# WP-11 implementation and verification

Implemented against Final Architecture and Contract Pack v1.2. Scope is the
plan-bound Response Generator and its integration with WP-03/04/07/10 and the
existing Orchestrator. No later work package or experimental responder is used.

## Behavior

- Deterministic §11.3 prompt assembly; static policy blocks, selected full Skill
  Cards, resolved protocol text, permitted evidence, preferences/history/current
  message, and bounded code-only correction notes. XML escaping, opaque UUID
  attributes, epistemic provenance and source-isolation checks precede disclosure.
- Strong-tier internal streaming, high-risk buffering, bounded input/output,
  shared Brain/Generator call allowance, TTFT/total deadlines, cleanup, finish
  metadata and sentinel handling. Oldest history is the only trimmable input.
- Existing WP-07 stream/final checks and Orchestrator regeneration/fallback
  control remain the delivery authority. Drafts and fallback templates must pass
  validation before the WP-02/08 atomic finalizer makes them visible.
- Bundle-backed objective/crisis fallbacks, including FB-006 elevated/follow-up
  outages with curated resources, fail closed if validation or coverage fails.
- MSSQL reads recheck ownership, visibility, chronology, text, evidence counts,
  sensitivity and explicit AI-use permission. No migration or data mutation is
  added; soft deletion retains original and derived records and consent.

## Integration

Apply `theia-ai-core-wp11.patch` from the backend repository root **after the
existing WP-01–WP-10 work**. The patch is relative to the captured pre-WP-11
workspace, including its uncommitted WP-10 implementation; it does not bundle
that earlier work. It is not intended to apply to a checkout missing WP-10.

Compose `NewAIResponseGenerator` with the existing SQL connection, WP-04 client,
explicit WP-09 UsePolicy and the same immutable approved WP-03 snapshot as
Safety, Validator and Brain. Pass it to `NewIntegrated` alongside the real WP-07
Validator and WP-10 Brain. Prompt/version hash mismatches reject admission to the
pipeline before private text is sent to a provider. The shared call allowance
is request-local and bounds small/strong preflights, including provider failover;
classifier/embedding budgets remain independent.

Provider `DefaultTimeout`, model chains and usage ledger/quota configuration
must accommodate the bundle's bounded generator settings. No network call runs
inside a SQL transaction. Callers must use the Orchestrator's accepted result;
the Generator port produces internal drafts only. This change does not switch
the legacy HTTP composition, add SSE, or bypass the existing WP-13 blocker.

## Verification

Tests cover prompt golden rendering and order/cache boundaries, injection
escaping, selected cards, preferences, inferred/reported/raw provenance, locale
and size coverage, protocol rendering without execution, stream buffering,
split sentinels, provider failover/interruption, malformed/truncated output,
guard/final validation, regeneration and fallback validation, classifier
failure, expired deadlines, shared call bounds, Brain retrieval hops, replay,
cross-user/source denial, atomic rollback and soft-delete preservation.

Run on 2026-11-17, Go 1.25.4 linux/amd64, with synthetic fixtures and the local SQL Server 2022
container. No external LLM API was called.

| Check | Result |
| --- | --- |
| `go test ./...` | Passed; opt-in DB tests skip without their local test environment |
| `go test -race ./internal/modules/ai/... ./internal/app/composition` | Passed |
| `go build ./...` | Passed |
| `go vet ./...` | Passed |
| `python3 scripts/check_structure.py` | Passed: 41 required paths, SQL Server dialect |
| `python3 scripts/check_contract.py` | Passed: 30 method/route pairs |
| Live `TestWP11SQLValidatedDeliveryIsolationAndSoftDelete` | Passed |
| Expanded live AI SQL suite | Core lifecycle/upgrade/races/migration guards, WP-05 success/replay, WP-10 and WP-11 passed; one pre-existing WP-05 test failed (below) |
| Patch verification | Clean application to the captured pre-WP-11 baseline, complete byte comparison, and `go test ./...` from the reconstructed tree passed |

The live SQL run used `APP_ENV=local`, `WELLBEING_INTEGRATION=true` and a
loopback SQL Server connection; credentials were supplied privately from the
existing container. Each integration test uses its existing isolated synthetic
database fixture. The test selection was:

```sh
go test ./tests/integration -run '^(TestAICore|TestWP(05|10|11)SQL)' -count=1 -v
```

`TestWP05SQLServerFailureAndOriginalRequestRecovery` fails at its existing
`terminal failure was not idempotent` assertion (ai_wp05_test.go:189). The same
test and assertion fail against the untouched pre-WP-11 snapshot. Its provider
history assertion expects two calls despite the existing regeneration path.
This older test is unchanged and excluded from the patch; its failure prevents
claiming an entirely green expanded live SQL suite. All WP-11 tests pass.

## Review and activation blockers

The shipping `knowledge/` bundle remains synthetic placeholder content and was
not edited. Production construction rejects it. Real activation requires a
release-verified, clinically approved persona/crisis/communication policy,
complete selected Skill Cards, boundary content, locale validator rules,
current curated resources and objective fallbacks within §11.3 budgets.

WP-07 currently requires exact reviewed crisis-template text. The renderer
therefore supplies that template as escaped copy-verbatim data. Broader crisis
wording needs reviewed element matchers; WP-11 does not weaken this constraint.
Deterministic/optional-classifier checks do not establish semantic or clinical
quality for arbitrary model paraphrases. Qualified clinical review and real-model
plan-conformance/golden evaluation remain release requirements (WP-15 is outside
this patch). No external provider or real-user test was performed.

Token counts use the existing conservative rune approximation, not a provider
tokenizer; calibrate provider limits before activation. Protocol execution and
consent resolution remain WP-13 responsibilities. No known technical failure
is intentionally left open in WP-11.
