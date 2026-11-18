# WP-15 Evaluation Harness (Contract Pack v1.2)

## Scope and authority

This is offline test/evaluation infrastructure for the existing Go Modular Monolith. It follows Final Architecture §9.4-9.6 and Contract Pack v1.2 §12.1-12.3. It introduces **no runtime composition changes**, new database tables, hard deletes, HTTP endpoints, protocol execution, or dependency on WP-13. Test data is synthetic only. It does not establish clinical safety or production approval.

Implemented in `apps/api/internal/modules/ai/eval`:

- Strict `golden.v1` YAML loader with §12.2 `setup`, `turn`, `turns`, `sessions` (gaps), optional `ground_truth`, and exhaustive typed assertions. Unknown assertion names/fields, multiple documents and unmarked non-synthetic fixtures are rejected. Runner carries a synthetic per-user state view and previous turn observations through a scenario. Live test adapters receive setup and previous outcomes; WP-15 does not quietly seed a production DB.
- Typed `Executor` seam for running actual modules (or a synthetic `RecordedExecutor`), plus opt-in capture wrappers for the existing WP-02 Gateway/Finalizer, WP-09 Retriever, WP-10 Brain, WP-11 Generator and WP-07 Validator. The wrappers delegate to the actual ports and preserve WP-10/11 integration capabilities. For real port-level tests, build a new, **isolated synthetic test composition**, inject the wrappers, supply a `CaptureExecutor.NewRequest` factory and keep all scenario state in that test environment. The integrations are not activated by the app composition.
- Offline synthetic `turn_trace.v1` inspection via `TraceExecutor` and `ReplayAgainst` for candidate-side decision comparisons. A CLI input wrapper (`eval_trace_fixture.v1` with `synthetic_only:true`) is mandatory. Partial WP-05 traces / §12.1 trace records contain `text_ref` but not response text, status-scoped retrieval evidence or a full committed state: assertions requiring those unavailable fields are **blocked**, not inferred. Full live rerun requires externally supplied synthetic state, persisted message payloads and a candidate `Executor`; a trace alone cannot recreate them.
- Scoring of route, safety level/categories/route invariant, retrieved-item recall and excluded statuses, Brain objectives/forbidden actions/questions/skills/plan constraints, past-reference resolution, hypothesis operations, protocol action, Validator pass/action/rule codes, question budget, final response rule-backed exclusions, state/session/risk/thread/memory continuity, async memory operations, fact recall, correction and stale-hypothesis checks, plus degraded-mode handling. Unavailable evidence is a blocker. Metrics are numerators/denominators with success rates; forbidden-action rates are the complement of `forbidden_objective`/`hypothesis_forbidden_ops` compliance rates. These metrics are on **synthetic scenarios**, not clinical efficacy estimates.
- `eval_report.v1`: deterministic, redacted scenario/turn results with stable rule codes, assertion paths, denominator-aware metrics, blocked evidence and safe review flags. No user messages, replies, model drafts, prompts, reviewer narratives, UUIDs or error strings are serialized. Failed/aborted runs fail closed.
- Baseline comparison rejects newly failed scenarios/assertions, removed scenarios, missing metrics and declining score rates. All safety-critical metrics have zero regression tolerance. Existing metric tolerances can be configured for other dimensions; the CLI defaults to zero tolerance.
- Independent `ReviewHook` supports rubric scores only from a restricted externally verified calibration; arbitrary recorded score fields cannot pass L3 quality gates. Optional `LabelVerifier` supports externally attested, signed/reviewed synthetic golden labels. The YAML `clinician` envelope alone is **not** authentication or proof of review and is always marked unverified absent a trusted verifier. Review flags contain only synthetic scenario keys and stable reason codes.

## Run

From `apps/api` (with Go toolchain and dependencies installed):

```sh
go test ./internal/modules/ai/eval -count=1
go test ./internal/modules/ai/eval -race -count=1
go test ./...
go run ./cmd/aicore-eval \
  -golden ./internal/modules/ai/eval/testdata/golden \
  -observations ./internal/modules/ai/eval/testdata/observations \
  -out /tmp/theia-wp15-eval.json
# Future regression gate (capture the prior reviewed baseline separately):
go run ./cmd/aicore-eval \
  -golden ./internal/modules/ai/eval/testdata/golden \
  -observations ./internal/modules/ai/eval/testdata/observations \
  -baseline /path/to/baseline.eval_report.json \
  -out /tmp/theia-wp15-regression.json
# Replay is deliberately partial unless trace artifacts supply all expected evidence:
go run ./cmd/aicore-eval -golden /path/to/synthetic/golden \
  -traces /path/to/synthetic/trace-fixtures \
  -out /tmp/theia-wp15-trace-report.json
```

Exit `0` means *all deterministic assertions in the selected synthetic fixture set* passed. Exit `1` means assertion failure, missing evidence or regression, and `2` means invalid inputs/report errors. It does **not** mean clinically approved, fully integrated, or equivalent to real-model behavior. The `-observations` files have schema `eval_observations.v1`; each matches a golden by `scenario_id`, contains `synthetic_only:true`, a synthetic `user_id` UUID and one typed observation per turn. The CLI intentionally has no network, SQL or production trace mode. Reports are written with restrictive creation permissions; do not put fixtures/outputs in public directories.

## Verification and blockers

Checks required on a dependency-enabled CI agent:

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
go build ./...
go vet ./...
go test ./...
go test -race ./internal/modules/ai/eval
```

This environment lacks an accessible Go module cache and cannot resolve `proxy.golang.org`, so Go compilation/tests cannot be verified here. No live SQL Server, LLM provider, clinician-review system or approved Persian assets were used. The included `testdata` are fictional unreviewed regression examples; they are **not** clinical goldens. Before claiming a release-quality L2-L5 benchmark, a separate team must:

1. Wire the test-only capture adapters into an isolated, owner-scoped synthetic application fixture, execute the real WP-06/07/10/11/12 modules and capture complete finalization/async-memory evidence. A candidate `Executor` must not silently substitute prerecorded outputs for live calls.
2. Implement an approved restricted clinical review/audit workflow, attested labels and calibrated judge scoring. Set thresholds for objective accuracy, forbidden-action rate, safety recall/precision and L4 continuity based on reviewed goldens (the architecture has not specified universally binding release thresholds).
3. Improve `turn_trace.v1` evidence capture for replay under explicit consent/synthetic-only controls, including encryption, prompt-capture safeguards and secure original-message resolution. The current metadata-only trace cannot reproduce a full LLM execution.
4. Review non-English and high-risk safety cases with qualified clinicians, validate resource/locale completeness and red-team coverage, and establish formal release gates; synthetic pass/fail metrics alone cannot prove safety.

## Integration and privacy boundaries

No persisted AI data is modified or deleted. Capture wraps existing ports in the **test composition only** and retains raw generated text in ephemeral in-memory observations for evaluation, not in logs/reports. The Retriever wrapper demands an independent `VerifyOwner` callback for memory owner verification; without it the evaluator does not accept retrieved evidence as owner scoped. All replay sources must be explicitly labelled synthetic and user identity must match the case. Keep fixture storage access-controlled; `turn_trace.v1` raw payloads should never be exported from real users into this tool without a separate approved privacy/consent architecture.

**Known limitations:** The CLI uses recorded deterministic fixture outputs, not live-model calls. Quality rubric/clinical labels are blocked without independent verification. No complete trace-body decryption, first-party clinician-review queue, production-data replay, SQL fixture loader, or live-score calibration is included: those are explicit external integrations and/or release blockers, not silently mocked. WP-13 protocol execution is out of scope; protocol checks cover the Brain's proposal only.

## Handoff verification in the supplied archive (2026-11-18)

- `python3 scripts/check_contract.py`: **passed** (30 OpenAPI route/method pairs).
- `python3 scripts/check_structure.py`: **blocked**: the uploaded archive omits twelve `.agents/skills/*` skill files declared as required by this script; no WP-15 code modifies them.
- `gofmt -l` for all new Go files: **clean**.
- Python parsing of all three new golden YAML files and observation JSON envelopes: **passed** (1, 3, and 3 synthetic turns).
- Go package/type compilation of the new eval package and CLI with **temporary compile-only local stubs** for the unavailable `github.com/google/uuid` and `gopkg.in/yaml.v3` dependencies: **passed** (`go test ... -run '^$'`). The repository `go.mod` and `go.sum` were restored after this check. The stubs were **not** used to run functional tests and are **not** part of the patch.
- `go test ./internal/modules/ai/eval ./cmd/aicore-eval`, `go test ./...`, and full `go build`/`go vet`: **unverified** because this environment cannot resolve `proxy.golang.org` and has no downloaded dependency cache. No green runtime-test result is claimed.
