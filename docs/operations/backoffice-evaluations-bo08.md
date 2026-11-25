# BO-08: synthetic AI evaluations (backend)

## Scope and evidence limits

- Staff-only `GET /api/v1/admin/evaluations`, `GET /api/v1/admin/evaluations/{runId}` and `POST /api/v1/admin/evaluations/import`.
- Reuse the existing WP-15 **sanitized** `eval_report.v1`. No test execution, file browser, arbitrary commands, real-user traces, raw observations, prompt/response text, clinician review workflow, or release approval is added.
- **Three different claims must stay separate**: `artifactHashVerified=true` means the uploaded SHA-256 matches those exact bytes; `sourceAttested=false` means this API cannot prove what process produced them; `clinicallyApproved=false` means no clinical/safety release approval. An `external_executor` method is a *claim inside an imported document*, not proof the live model ran.
- `passed` / `failed` / `blocked` are different. Blocked evidence takes precedence; empty/unknown evidence is not promoted to success. A reported `externally_verified_synthetic_labels` review flag is rejected without a separate trusted signed-label workflow.
- The embedded `golden_registry.json` fingerprints the **28 unreviewed, synthetic** fixtures under `knowledge/evals/golden`; the script `scripts/tests/test_admin_eval_registry.py` checks digest drift. `declaredGoldenMatches` is an ID intersection only. The WP-15 report does not contain per-fixture hashes: `fixtureExecutionVerified=false`. Persian/English counts are based on reported `language_fa`/`language_en` tags and are not independently attested model coverage.

## Schema and API

Migration `0030_admin_eval_runs.sql` is additive and SQL Server 2022 only. It creates one append-only table `dbo.admin_eval_runs`, an index for newest-first browsing, a unique artifact/baseline index for idempotent imports, a Staff FK, and an immutable-baseline self-reference. It stores **only normalized/allowlisted sanitized** result fields, numerical summaries, an SHA-256 fingerprint, declared method and source labels, and server-computed regressions. No user messages, traces, golden text, contact data, or Keycloak identifiers are stored. Updates and deletes are blocked by a SQL trigger for the application account; this is not protection from a privileged DBA.

Only active Staff with a valid host-bound Theia session can access any route. All successful authorized reads/imports are durably audited **before** evaluation storage is touched; failure to record audit denies the operation. Imports also require the existing Staff session's `X-CSRF-Token` and `Origin` check, and accept **at most 1 MiB** of `application/json` (`eval_report.v1` only). Every input is validated with strict unknown-field rejection, length/count caps, stable diagnostic/identifier code allowlists, metric rate checks, and aggregate consistency checks. A client-provided SHA-256 is a checksum, not source authentication.

- `GET /api/v1/admin/evaluations?status=blocked&from=<RFC3339>&to=<RFC3339>&pageSize=25&cursor=...`: newest-first, keyset-paged, 31-day maximum window (default 24 hours). `status` is `passed`, `failed` or `blocked`.
- `GET /api/v1/admin/evaluations/{runId}`: saved redacted report, summary, and server-recomputed baseline regressions.
- `POST /api/v1/admin/evaluations/import[?baselineRunId=<UUID>]`: raw report file bytes, `X-Eval-SHA256` of those exact bytes, Staff session and CSRF. Returns `201` on first import, `200` on identical reimport. An optional, immutable stored baseline is compared with WP-15 `CompareReports` using zero rate-drop tolerance and hard-blocking safety regressions. Missing baseline returns `404`.
- Stable failure codes: `ADMIN_EVALUATION_INVALID` (`400`), `ADMIN_EVALUATION_NOT_FOUND` (`404`), `ADMIN_EVALUATION_CONFLICT` (`409`), `ADMIN_EVALUATION_CONTENT_TYPE` (`415`), `ADMIN_EVALUATION_TOO_LARGE` (`413`), `ADMIN_AUDIT_UNAVAILABLE` (`503`), `ADMIN_EVALUATIONS_UNAVAILABLE` (`503`); Staff Auth/CSRF errors come from the existing Auth middleware.

## Local synthetic-only smoke test

**Prerequisites:** Keycloak + SQL Server migrations through `0030`, enabled Staff login, and a provisioned Admin account. Run only locally on an internal synthetic environment. Keep the report outside public/static directories. Do not use actual user data.

First produce an offline WP-15 report, from `apps/api`:

```sh
go run ./cmd/aicore-eval \
  -golden ./internal/modules/ai/eval/testdata/golden \
  -observations ./internal/modules/ai/eval/testdata/observations \
  -out /tmp/theia-wp15-report.json
sha256sum /tmp/theia-wp15-report.json
```

The command uses deterministic recorded observations, **not a live model**. A nonzero exit is still useful as a failed/blocked report; it is not an excuse to treat missing evidence as success. The `knowledge/evals/golden` fixtures are a separate, unreviewed set and do not automatically have matching recorded observations. Do not claim all 28 were executed just because their YAML files exist.

After staff login, obtain the CSRF token from `/auth/staff/session` and retain the browser's host-only session cookie. Example for a trusted local shell (supply the real cookie/token securely; do not commit either):

```sh
REPORT=/tmp/theia-wp15-report.json
SHA=$(sha256sum "$REPORT" | cut -d ' ' -f1)
curl --fail-with-body -X POST \
  'http://admin.localhost:8080/api/v1/admin/evaluations/import' \
  -H 'Content-Type: application/json' \
  -H 'Origin: http://admin.localhost:8080' \
  -H "X-CSRF-Token: $STAFF_CSRF_TOKEN" \
  -H "X-Eval-SHA256: $SHA" \
  -b "$STAFF_COOKIE_FILE" \
  --data-binary "@$REPORT"
```

Use the returned run UUID with `GET /api/v1/admin/evaluations/{runId}`. To compare with a stored baseline, reimport the new report with `?baselineRunId=<UUID>`. Reimporting the same report under the same baseline is idempotent. A different baseline is a separately labeled comparison.

## Checks before merge

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v
(cd apps/api && go build ./...)
(cd apps/api && go vet ./...)
(cd apps/api && go test ./...)
(cd apps/api && go test -race ./internal/modules/admin/...)
```

The optional SQL Server integration test requires an **isolated throwaway DB** with migration 0030 and the explicit flags `THEIA_DISPOSABLE_EVAL_SQL_DSN` and `THEIA_DISPOSABLE_EVAL_SQL_ACK=YES_DESTROY_AFTER_TEST`. It inserts immutable synthetic rows. **Destroy the entire disposable database after testing. Never aim it at a shared/stage/production DB.**

### Rollback and future work

Disable/roll back the API binary while keeping migration 0030; do not rewrite or drop an applied migration. Stored reports remain immutable. No async worker or new service was introduced. A future source-attestation system (signed CI runner manifests, fixture digests bound to each run and independently verified reviewer labels) is required before the UI can claim verified model execution, full 28-fixture execution or clinical readiness. The BO-07 frontend should display these limits prominently, not just the word “passed”.
