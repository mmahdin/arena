# AI Core WP-10 verification

Base: `ed676f3528dfbc02f0f1665cb1577519ef1cb278` (clean working tree).
Authority: Final Architecture Document and Contract Pack v1.2 in `docs/ai-core/`.

WP-10 adds a schema-constrained Turn Brain, bundle-driven prompt assembly,
deterministic decision gates, a bounded second retrieval hop, source authorization
and state proposals. The existing WP-04 client, WP-06 Safety FSM, WP-07 Validator,
WP-08 state validator and WP-02 atomic finalizer remain the authorities for their
respective responsibilities. No WP-11/12/13 module, migration, HTTP change,
Knowledge Bundle content or hard-delete path is introduced.

See [the package contract](../../apps/api/internal/modules/ai/application/brain/README.md)
for wiring, limits, rule coverage and clinical blockers.

The focused changes to existing modules are necessary WP-10 integration:

- Bundle snapshots expose a copied, sorted runtime artifact catalog.
- Orchestration accepts gated Brain plans, supplies permitted context, normalizes
  ingress NFC, re-evaluates once after retrieval, and builds state proposals.
- Accepted Validator question metadata reaches state finalization. Regeneration
  retains valid proposals; static fallback drops unexecuted proposals.
- WP-08 permits a validated resumption to assign focus to the resumed thread.
- WP-04 preserves metered usage/attempts on structured-call failure, so repair
  cannot silently consume a second-hop allowance.
- Metadata-only tracing accepts the fixed Brain/fallback rule codes.
- One pre-existing unused WP-09 test import is removed. A pre-existing replay test
  now compares call counts before/after replay; its old literal count predates
  WP-07's bounded generation retry. Its failure was reproduced using the baseline
  orchestrator through a Go overlay before correcting the assertion.

The new fixtures are synthetic engineering tests. They are explicitly **not**
clinician-reviewed decision goldens; no clinical reviewer or approval is invented.
Clinical quality has not been measured against a live model. The inert Knowledge
Bundle prevents production activation, and the supplied SQL policy adapter keeps
hypothesis voicing disabled for the MVP. Memory corrections, protocol execution,
and persistence of new authored hypotheses depend on their later work packages.

Validation on 2026-11-17:

| Check | Result |
| --- | --- |
| `go test ./...` | Passed; opt-in SQL tests skip in this default run |
| `go test -race ./internal/modules/ai/... ./internal/app/composition` | Passed |
| `go build ./...` | Passed |
| `go vet ./...` | Passed |
| `python3 scripts/check_structure.py` | Passed |
| `python3 scripts/check_contract.py` | Passed; 30 existing method/route pairs |
| Live `TestWP10SQLStateEvidenceCooldownAndIsolation` | Passed against local SQL Server 2022, using an isolated migrated database |
| Patch application | Passed against a clean archive of the baseline; patched files match the working implementation byte for byte |

The live SQL test used server-local credentials without printing them. No model
network calls, production data, clinical approvals or live-model results are
claimed. Prior migration files and Knowledge Bundle content remain unchanged.
