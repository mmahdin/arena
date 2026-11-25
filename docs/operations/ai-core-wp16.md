# WP-16 deployment and HTTP client migration

The backend is a single Go/Gin Modular Monolith with in-process AI Core calls.
Only private synthetic testing is authorized. No approved clinical assets or real
user authentication are shipped. Proactive scheduling execution is disabled.

## Start-up and worker lifetime

1. Stop binaries using the experimental AI. Run the existing checksummed migration
   runner through **0023** before starting the WP-16 binary. Applied migrations are
   immutable. 0023 adds nullable account visibility metadata; it changes no
   existing payload or consent. No down migration or destructive cleanup exists.
2. Provide a released WP-03 bundle and trusted Ed25519 public keys, WP-04 provider
   chains, a reviewed technical use/consent mapping and managed encryption keys.
   No secrets belong in Git, configuration examples, logs or frontend variables.
3. Set `AI_CORE_CONFIG_FILE` to the server-side deployment JSON path. Credentials
   are read from the named `MODEL_*` environment variables. AES-256 keys are
   base64-decoded from named `AI_*` variables. Retain old keys for decryption and
   provision an active key explicitly; never generate a replacement on restart.
4. `cmd/api` constructs `NewAPI`, starts `API.Run` once, and joins the worker group
   on exit before closing SQL Server. Signal cancellation also cancels HTTP
   request contexts. Workers use existing durable jobs, leases and checkpoints;
   ordinary worker recovery does not invent a user message.
5. `/health/live` is process liveness. When product routes are enabled,
   `/health/ready` requires SQL connectivity and a configured running core worker
   group. It makes no billable model call and does not certify provider/clinical
   readiness. Persistent worker failure stops the process with sanitized health
   metadata; `API.WorkerHealth()` exposes safe scheduler counters to the host.

An empty deployment path permits platform startup, but AI routes/readiness return
503. An invalid supplied deployment fails composition. Neither case falls back to
an old prompt, mock reply or experimental generator. With dev login disabled only
health routes are mounted, as before; no production identity system is implied.

The repository's `LLM_PROVIDER=mock` setting is historical configuration parsing,
not an active AI responder. The new model chains must match the signed bundle's
provider/model order and embedding dimensions. Operational capabilities, prices,
credentials and limits are supplied explicitly; prices are not fetched or guessed.
WP-04's default usage ledger is process-local, not a distributed spend reservation.

## Deployment JSON shape

`schema` must be `ai_runtime.v1`. A file has these top-level fields (not a runnable
approved configuration):

```json
{
  "schema": "ai_runtime.v1",
  "bundleDirectory": "/run/theia/released-kb",
  "trustedKeys": {"release-key-id": "BASE64_ED25519_PUBLIC_KEY"},
  "workerId": "unique-stable-host-identity",
  "providers": {
    "approved-provider-id": {
      "baseUrl": "https://approved-provider.example/v1",
      "keyEnv": "MODEL_API_KEY"
    }
  },
  "models": {
    "chains": {
      "small": [{"provider": "approved-provider-id", "model": "bundle-small-model", "native_json_schema": true}],
      "strong": [{"provider": "approved-provider-id", "model": "bundle-strong-model", "native_json_schema": true}],
      "classifier": [{"provider": "approved-provider-id", "model": "bundle-classifier-model", "native_json_schema": true}],
      "embedding": [{"provider": "approved-provider-id", "model": "bundle-embedding-model", "dimensions": 1024}]
    },
    "max_retries_per_model": 1,
    "default_timeout": 60000000000,
    "retry_base": 100000000,
    "retry_max": 1000000000,
    "breaker_window": 60000000000,
    "breaker_cooldown": 30000000000,
    "breaker_error_rate": 0.3,
    "breaker_min_samples": 5,
    "daily_quota": {"calls": 100}
  },
  "encryption": {"active": "key-id", "keyEnv": {"key-id": "AI_MEMORY_KEY"}},
  "policy": {
    "reviewId": "EXTERNAL_REVIEW_REFERENCE_REQUIRED",
    "consentKeys": [],
    "actions": {}
  },
  "feedback": {"minUnusedRetrievals": 3, "importanceMultiplier": 0.9}
}
```

Model names, dimension, quotas and values above illustrate the format; they are
not model recommendations or approvals. Durations in the reused WP-04 JSON are
nanoseconds. Supply reviewed values and exact bundle identities. Prices omitted
above default to zero for accounting and must be supplied for meaningful cost
limits. There is exactly one embedding chain entry in this initial product host.

Ordinary memory retrieval, reading, writing, reflection and indexing are enabled
without consent. `ProductUsePolicy` still checks active accounts and source
ownership/visibility. `searchKey`, `messageUseKey`, `memoryUseKey` and ordinary
memory action mappings are accepted for compatibility but do not gate memory.
An empty `consentKeys`/`actions` configuration is sufficient. Existing supplied
keys and mappings must remain valid; remove obsolete mappings and their keys
together. The checked-in `config/ai-runtime.yaml` uses the empty configuration.

`use_retained_source` remains a separate explicit permission for hidden historical
evidence, checked against both current state and persisted consent. Missing
mapping or consent denies hidden-source reuse. `proactive` remains forbidden;
protocol participation and other consent mechanisms are unchanged. `reviewId`
remains required as an operator-supplied review reference, not a clinical approval.
Protect this privileged file independently; WP-03 still verifies bundle approvals.

Legacy `memoryPaused` values are ignored, including existing true database values;
GET controls always reports false and PATCH accepts/ignores pause input. New state
inserts and control writes use false. No migration, backfill of memory content,
job replay or historical consent rewrite is needed. Account visibility and
transaction/version fences still prevent stale commits. Custom host policies
remain injectable for isolated synthetic integration tests.
The HTTP client cannot modify the deployment policy, allowed key set, model chains
or proactive flags. Clinical contraindication inventories remain an explicit
`ProtocolFacts` dependency; unknown flags block applicable offers.

For Docker, mount the deployment file and released bundle read-only using a
private Compose override, and point `AI_CORE_CONFIG_FILE` at the container path.
The base Compose file passes that variable and `AI_MEMORY_KEY`; it does not create
an approved bundle, mount host secrets automatically, or open additional ports.

## HTTP contract and client changes

`contracts/openapi/openapi.yaml` **0.3.0** replaces the experimental turn response.
The Gin adapter is handwritten. There is no generated server or SSE interface.
All protected identity comes from the signed, revocable SQL session cookie.
`userId`, `origin`, `wakeUp` and internal state fields are rejected in turn bodies.

- `POST /api/v1/conversations/{id}/turns` accepts `clientTurnId`, `message`, optional
  `locale`, `timeZone`, `clientTimestamp`, `replyToId`. Defaults are `fa`/`UTC`;
  the selected locale still requires reviewed bundle safety/content coverage.
  Requests are UTF-8/NFC normalized and capped at 4,000 characters. Ownership and
  completed onboarding are checked before admission; no chat-consent lookup is
  made. New user admissions have an atomic per-user SQL minute limit from the bundle.
- A completed result has `turnId`, `userMessageId`, `assistantMessageId`, `text`,
  `status`, `degraded`, and optional route, safety banner and Protocol UI. Only
  completed results carry answer text. Remove client dependencies on `reply`,
  `promptVersion`, mock-provider labels and the obsolete AI-status endpoint.
- A completed duplicate replays without model work. A changed normalized payload
  or request metadata under the same owner key returns **409
  `IDEMPOTENCY_CONFLICT`**. Preserve the same UUID/input after a transport error;
  do not invent a fresh submission ID and duplicate a durable user message.
- Queued/running submissions return **202** and `Retry-After`. They are not a
  completed answer. `GET /conversations/{id}/turns/{clientTurnId}` is a pure
  owner/conversation-scoped status/replay read. Failed/cancelled duplicate POSTs
  return their terminal status without inference.
- `POST /conversations/{id}/turns/{clientTurnId}/retry` explicitly requeues terminal
  failure using the original saved request; it returns 202/current status and is
  safe to repeat once queued/completed. It takes no body. Then `POST /me/ai/recover`
  processes the oldest visible owner request in FIFO order from persisted metadata.
  Recovery can return a different conversation's earlier queued turn; route the
  result using the submitted turn/message handles, not the currently open screen.
- A failed provider/validation/finalization can leave a durable user-only turn.
  Poll status, show a retry affordance and use explicit retry/recover. Never
  create a placeholder assistant row or display an internal partial stream.

The existing separate frontend repository is **not migrated by this backend
patch**. Its old `reply` handling and AI-status/Daily Reflection requests must be
updated before an end-user UI cutover. No compatibility switch to the old AI is
provided. The backend smoke client is updated; it requires an activated reviewed
configuration and explicit `SMOKE_AI_CONSENT_KEYS` for synthetic test accounts.
It restores those consent values afterward. A model-backed deployed smoke was
not executed during WP-16; the live-SQL Gin tests use explicit provider fixtures.

## Memory permission change verification — 2026-11-23

Verified with Go 1.25.4 using synthetic fixtures, from `apps/api`:

- `go build ./...` and `go vet ./...`: passed.
- `go test ./internal/modules/ai/application/... ./internal/modules/ai/adapters/mssql ./internal/app/composition -skip '^TestInspectorAsyncReflectionRetryUsesDurableLinksAndCheckpoint$' -count=1`: passed.
- `go test ./...`: all other packages passed after allowing local `httptest`
  listeners. `TestInspectorAsyncReflectionRetryUsesDurableLinksAndCheckpoint`
  fails with zero recorded model/job spans in both the unchanged checkout and
  this change; no unrelated Inspector fix or test suppression is included.
- From the repository root, `python3 scripts/check_structure.py`,
  `python3 scripts/check_contract.py` and `git diff --check`: passed.

`GOCACHE` was set to a writable temporary directory. Live SQL Server acceptance
was not run; opt-in database fixtures remain skipped without a disposable DSN.
Regression coverage checks default policy configuration, nil/empty/false consent,
legacy pause, retrieval, assembly, write reconciliation, job planning, reflection,
embedding input, account visibility, source ownership and hidden-source denial.

## Memory, consent and privacy

- `GET/PATCH /me/ai/controls`: legacy pause compatibility, allowlisted consent keys
  and AI preferences. Memory is always enabled without consent; clients should
  remove memory opt-in/pause requirements and treat `memoryPaused` as deprecated.
  Supply an `operationId` UUID for mutations. Omitted fields remain;
  a supplied preferences object replaces the full object. Keys/purposes and UI
  privacy text require review. A profile or sharing preference is not consent.
- `GET /me/memories`: bounded pages with an opaque last-item UUID cursor. This is
  an explicit owner view independent of AI-use permission. Preserve kind,
  epistemic status, confidence, evidence and source in client presentation; never
  render an inferred record as a reported fact or clinical conclusion. Reading a
  record does not voice/promote a hypothesis or modify evidence counters.
- `PATCH /me/memories/{memoryId}`: explicit superseding edit with `operationId`,
  `content` and 1–6 actual supporting owner-authored `evidenceMessageIds`. Kind,
  epistemic status and sensitivity are derived from the target; the client cannot
  promote inference. It preserves the original. Unsupported edits fail closed.
- `POST /me/memories/{memoryId}/corrections`: contest/reject/supersede-pending
  commands through WP-12 with operation receipts and hypothesis cooldowns.
  Corrections and hiding remain available regardless of legacy pause/consent values.
- `DELETE /me/memories/{memoryId}`, `DELETE /me/memories`, and `DELETE
  /messages/{messageId}` require an `Idempotency-Key` UUID. They change selected
  visibility only. No evidence, embeddings, derivatives or content are deleted,
  and hiding alone does not imply suppression or an AI-use policy change.
- `GET /me/ai/export?resource=messages|memories|controls`: bounded visible-owner
  export, at most 100 records per page. Message and memory cursors are opaque
  owner-scoped UUID handles. It is not a complete retained/legal account archive;
  encrypted traces/internal formulations/hidden resources are not exported.
- `DELETE /me`: hide the account, revoke test sessions, expire its processing
  lease and mark pending jobs cancelled. Payloads, consent and history remain
  unchanged. Re-login cannot silently reactivate the identity. No legal erasure,
  key disposal or backup deletion is implemented.

All ordinary reads obey resource visibility and session ownership. Errors use
stable Problem Details and a server-generated request ID, not raw provider/SQL
errors. Panic recovery discards request dumps. No request text or credentials are
added to telemetry. The Daily Reflection routes/feature are retired while its
legacy rows remain; saved letters remain readable/hideable, and new letter
inference is unavailable pending its own reviewed AI Core contract.

## Deferred contracts and release gates

Proactive execution remains disabled without any alternative admission path.
Complete encrypted runtime replay and Reflection merge/split await the contracts
recorded by Hardening. These APIs do not grant trace-body access or guess thread
partitions. Private synthetic engineering tests do not validate real provider
latency/cost/clinical quality, Persian launch coverage, clinician review, crisis
responsibilities, production identity or legal/privacy controls. Resolve those
before real-user deployment. Do not reopen prototype routes as a fallback.
