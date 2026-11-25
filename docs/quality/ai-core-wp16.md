# WP-16 — HTTP/OpenAPI and product integration

Verified locally on 2026-11-18 with Go 1.25.4 and SQL Server 2022. The Final
Architecture and Contract Pack v1.2 in `docs/ai-core/` were read as authority,
along with implemented WP-01–15, integration and Hardening code. The Hardening
patch was already applied; reverse-application checking confirmed that before
WP-16 work. That complete prerequisite tree was frozen as the export baseline;
Hardening is now committed in `4708cdb` and is excluded from this patch.

## Implementation

- `NewAPI` now invokes the full Gateway/Orchestrator through typed product use
  cases. `NewAIInternal` supplies real Safety, Context/State, Retrieval, Brain,
  Protocol, Generator/Validator, Memory and Reflection. There is one durable
  conversation/message store and no inter-module RPC or additional service.
- OpenAPI **0.3.0** defines the new camelCase JSON boundary, completed/queued/
  running/failed/cancelled results, status/replay, explicit retry and owner FIFO
  recovery, memory management, controls, export and account hiding. The handlers
  are handwritten; no generated server or external token streaming is claimed.
- Authentication reuses signed cookies plus active SQL sessions. Ownership and
  conversation scope are checked before inference; body-supplied user/origin/
  wake-up/state fields are rejected. Completed onboarding gates user admission;
  chat, retry and recovery do not check a separate chat-consent flag. UTF-8/NFC,
  strict/bounded JSON, locale/timezone and opaque UUID validation apply at the boundary.
- Durable user-first admission and atomic assistant/state/jobs/trace finalization
  remain in WP-02. Completed replay never invokes a model. New admissions enforce
  the bundle's per-user minute quota inside the owner-locked SQL transaction.
  Errors are stable Problem Details with server-generated request correlation;
  request dumps and raw SQL/provider errors do not reach logs or clients.
- Memory operations reuse WP-12's Writer/Committer, receipts, corrections and
  source/evidence checks. Public edits preserve kind/epistemic/sensitivity and
  supersede using actual supporting user-message IDs; they cannot create a fact
  from an inference. Owner pages support large retained inactive histories through
  bounded ID selection and the existing repository/decryptor, not an unbounded
  whole-store response. Default memory pause and explicit grants remain separate.
- Consent/preferences can be set before the first turn. The new explicit turn's
  locale/timezone are propagated into state under admission's existing lock,
  fixing the earlier state-default mismatch without changing original messages.
  Revocation increments the existing state version; stale in-flight completion
  is rejected. Read-only replay does not require new inference permission.
- `AI_CORE_CONFIG_FILE` selects a versioned deployment JSON: released WP-03 bundle,
  trusted signatures, exact pinned model chains/dimensions, explicit technical
  consent/use mapping, WP-04 client and managed encryption keys. Optional data-use
  consent is not inferred from onboarding, sharing, UI visibility or retention. Missing action
  mappings deny; `proactive` configuration is rejected. A missing configuration
  yields 503 for AI/readiness, and an invalid supplied configuration fails startup.
  There is no mock or experimental fallback.
- `API.Run` supervises Memory, Reflection and a **maintenance-only Scheduler**;
  signal/worker failure cancels HTTP inference and workers, joins them before DB
  close, and reports sanitized health. Even previously opted-in timers cannot
  dispatch through the product host. Wake-up requests remain forbidden before
  admission. No fake user message, notification or proactive workaround exists.
- Migration **0023_account_soft_delete.sql** adds account visibility metadata.
  Account hiding revokes sessions, expires AI leases and cancels pending jobs,
  while retaining every original/derived payload, consent and reference. Normal
  session lookup/login excludes hidden accounts. No previous migration was edited.

The HTTP catalogue is the authored OpenAPI; the full integration/deployment/client
migration requirements are in [the operation guide](../operations/ai-core-wp16.md).

## Prototype retirement and data preservation

Removed the one-shot AI service, old prompts/phrase-safety, old provider/mock
adapters, ReplyGenerator wiring, AI-status route/coupling and the unrelated
Daily Reflection source/routes/contracts/obsolete feature tests. The legacy
`daily_reflections` table and every row remain; source retirement is not erasure.

Letters' generator imported the retired experimental AI. That provider path is
removed; saved letters retain owner read/hide and eligible existing-result replay.
New letter inference is unavailable until it has a reviewed AI Core contract.
No clinical letter prompt, technique or replacement policy was invented.

Old SQL turn-format builders now exist **only in integration test fixtures** to
construct retained pre-Core history and verify upgrades/soft deletion. They are
not linked into the API. New core HTTP acceptance tests replace old assertions
about prototype prompts, 409 in-flight replies and disappearing failed inputs.
The WP-12 upgrade fixture now constructs historical data before applying current
migrations, instead of invoking a newer binary against an intentionally old schema.

The application and new migration contain no physical data deletion, cascading
purge, content clearing, key disposal or expiry cleanup. The established SQL test
harness uses isolated synthetic databases; it never resets the developer's
application database. The separately opted-in WP-09 test used another newly
migrated database, rolled back its fixtures and left that database retained.

## Executed verification

All final commands below exited zero. Plain `go test ./...` skips opted-in SQL
cases; the separate full race invocation actually enabled SQL execution.

| Check | Result |
| --- | --- |
| `go test ./...` | PASS |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| Live SQL `go test -race -json ./... -count=1` | PASS: 42 tested packages, 861 tests/subtests, no failures or data races; integration package 47.967 s |
| Integration package within that run | PASS: 41 top-level tests including all available Hardening SQL tests and the WP-16 suite |
| Separately opted-in `TestWP09SQLServerIntegration` with `-race` | PASS: 0.20 s test, 1.261 s package. It intentionally skips in the ordinary suite without its own fixture DSN; it was executed separately against a migrated isolated SQL database |
| WP-15 recorded CLI scenarios | PASS: 3 scenarios, 7 turns, 58/58 assertions, zero failures/blockers |
| WP-15 recorded baseline regression comparison | PASS, zero regressions; comparing these deterministic observations is not an independent model-quality study |
| All authored OpenAPI component response schemas | PASS: compiled locally with existing JSON Schema draft 2020-12 validator; actual HTTP responses in acceptance tests are validated against them |
| `python3 scripts/check_contract.py` | PASS: 38 method/route pairs match Gin composition |
| `python3 scripts/check_structure.py` | PASS: 41 required paths, 6 skills, SQL dialect checks |
| Backend deployment smoke client | Updated for new results/status/consent/errors; Python syntax compilation passed. A model-backed running-deployment smoke was not executed |
| Patch whitespace/fresh application | PASS: fresh application reconstructed all source bytes and deletions; `go test ./...`, `go build ./...`, `go vet ./...` and both repository checks passed in the reconstructed tree. Earlier work packages and Hardening are excluded |

The machine-readable summary is
[test-results/ai-core-wp16.json](test-results/ai-core-wp16.json). The retained
[evaluation report](test-results/ai-core-wp16-eval.json) contains assertion outcomes
and sanitized metrics, not prompts, drafts or private source evidence.

### Coverage of the new boundary

- Real Gin + SQL + actual complete core with synthetic provider responses behind
  WP-04/schema validation: normal turns, immutable replay/status, user/thread
  persistence, asynchronous extraction/embedding, correction/superseding edits,
  evidence/source metadata, unsafe-draft regeneration, crisis routing and safe
  degradation when the classifier is unavailable.
- Wrong/missing/revoked/tampered sessions; cross-user and wrong-conversation reads,
  processing and memory commands; strict rejection of forged owners/origins;
  default pause, allowlisted keys, repeated command receipts and changed-command
  conflicts; consent revoked while a provider stream is already in flight.
- Eight simultaneous duplicate HTTP submissions produce one durable user/assistant
  pair. Completed replay consumes no provider work; a new key beyond the SQL
  admission quota returns 429 without inserting another user message. Existing
  cross-conversation per-user leases produce truthful 202 responses.
- Finalization fault rolls back assistant/state, retains a failed user turn and
  permits explicit retry/recovery from original persisted metadata. Provider
  calls are outside the short SQL transaction. Only final results expose text.
- Bounded export, owner-scoped UUID cursors, 1,010 retained inactive memories
  paginated without lost evidence or archive truncation; hidden-memory retry and
  payload retention; account deletion fencing late publication and revoking
  sessions while preserving messages, derived content and consent; fresh/repeat/
  upgrade migration, immutable metadata and blocked physical deletion.
- Worker cancellation/join, maintenance-only scheduler with no due dispatch even
  for existing opt-ins, readiness/health, unavailable/invalid deployment and
  refusal to override pinned model identities or dimensions. Removed routes 404.

Existing WP-01–15 tests also exercise schemas, bounded second retrieval hops,
policy/FSM gates, streaming internals, provider fallback, retries, transactional
state, leases/checkpoints, hypothesis evidence/epistemics and long-term reflection.
Those are engineering tests with explicit synthetic scores/content, not clinical
semantic accuracy or approval.

### Reverification of Hardening's blocked checks

Hardening's earlier environment lacked Go dependencies and SQL Server. Here the
full Go tests/build/vet/race, all three Hardening SQL tests, WP-15 runtime coverage,
trace/candidate replay rejection requirements, golden/longitudinal/regression and
schema checks all executed and passed. The earlier missing-skill structure failure
also passed here. No blocked Hardening test is silently recorded as a pass from
its earlier environment.

The synthetic replay tests establish fail-closed evidence requirements; they do
not make ordinary runtime traces replayable. Likewise, passing metadata inspector
or scheduler tests does not authorize clinical access or proactive turn execution.
No final executed test is failing. Intermediate migration/fixture/HTTP failures
found during implementation were fixed and rerun; they are not concealed skips.

## Remaining integration and release gates

1. **Frontend migration:** this export is limited to the requested Go backend and
   its authored HTTP/operation clients. The separate frontend repository still
   consumes the old reply/status/daily-reflection interfaces. Update its API/state
   adapters to the documented 0.3.0 contract and test browser recovery/consent/
   privacy before a product UI cutover. No frontend/browser E2E is claimed here.
2. **Reviewed activation assets:** the repository's knowledge content is synthetic
   and cannot be activated as a released clinical bundle. Supply the reviewed
   bundle, signed approvals, verified locale resources, configured model chains,
   managed keys and a reviewed consent/use mapping. The deployment review reference
   is an operator attestation, not a fabricated certificate. Qualified review of
   memory/inference presentation and explicit privacy copy is still required.
3. **Clinical contraindication inventory:** unknown `ProtocolFacts` remain unknown,
   and applicable Protocol offers fail closed. A reviewed host inventory must be
   supplied through the existing explicit injection seam; the default deployment
   factory does not guess contraindication absence from missing flags.
4. **Explicitly disabled/deferred:** Proactive Wake-up remains off as requested.
   Full encrypted runtime replay and Reflection merge/split remain blocked on the
   approved contracts identified in Hardening. No trace-content API or speculative
   thread destinations were introduced. New letter inference is also unavailable.
5. **Privacy/export scope:** account/memory/message hiding is soft-only; data,
   backups and keys remain retained. Export covers visible messages/memories/
   controls, not a complete retained/legal account archive or encrypted trace
   bodies. Complete legal export/erasure/access policy requires separate approval.
6. **Operational/real-user release:** identity remains synthetic dev-session
   selection, not production authentication. Raw messages remain synthetic-only
   storage; production encryption/privacy/access review, clinician/judge calibration,
   multilingual safety/red-team, verified crisis responsibilities/resources,
   live-provider acceptance and sustained load/latency/cost testing are unexecuted.
   WP-04's default daily ledger is per process and does not reserve global spend.

## Technical readiness after WP-16

**The non-proactive backend HTTP integration is implemented and passes the executed
engineering gates against real SQL Server.** It is ready for private synthetic
backend/client-integration review with explicit test dependencies.

**The running end-user product and real-user release are not ready.** Frontend
consumers need migration; reviewed assets, policy/key provisioning, production
identity and clinical/privacy/operational approval remain activation gates. The
excluded proactive/replay/merge-split capabilities must not be advertised as
complete. No production clinical quality, provider benchmark or browser acceptance
is inferred from these test results.

## Patch application

`theia-ai-core-wp16.patch` is incremental over the existing WP-01–15 + integration
+ already-applied Hardening tree. Apply from the backend repository root after
those prerequisites; do not apply earlier packages twice. No prior work or
Hardening patch body is re-exported. The working tree retains those prerequisites
and this WP-16 implementation. No new commit or push is part of this delivery.
