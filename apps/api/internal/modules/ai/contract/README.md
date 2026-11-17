# AI Core WP-01 contracts

Authority: `docs/ai-core/Theia AI Core — Final Architecture Document.md` and
`docs/ai-core/Theia_AI_Core_Contract_Pack_v1.2.md`, especially §§3–6 and Appendix A.
The v1.2 reconciliation overrides the architecture's PostgreSQL/RPC examples and
fail-open state-commit behavior. The backend remains one Go module and deployment.

This package defines the new contracts independently of the experimental AI
application. It imports the standard library and the existing UUID package only.
`ports/` consumes these types; `contract/schema/` supplies structural validation.
No runtime pipeline, database migration, model client, bundle loader, prompt,
policy repair, safety classifier, worker, or HTTP endpoint is implemented here.

## Boundary contracts

- Authenticated input: `TurnRequest.UserID` is server-resolved, never an HTTP body
  authority. The HTTP adapter is responsible for UTF-8/NFC preparation and owner
  authorization. `OriginWakeUp` is an internal scheduler input, not client authority.
- Admission: one Conversation-owned message store, durable user input before model
  work, per-user serialization across conversations, and a fenced lease. Admission
  carries trusted owner/conversation IDs so finalizers can check their scope.
- Completion: `PreparedTurn` contains validated output only. The finalizer must
  atomically commit the assistant message, turn status, state, jobs and safe trace
  metadata. Failure must not report success or lose the admitted user message.
- State/memory/model output structs are proposals, never write authorization.
  UUID syntax does not establish existence, evidence support, entity type or owner.
- JSON uses explicit snake_case tags. Required slices serialize as `[]`; optional
  slices are omitted when empty. Optional IDs/scalars use pointers where absence
  matters. `MemoryOpProposal.Importance` preserves absent versus explicit zero.
  Go `time.Duration` JSON uses integer nanoseconds; instants use Go RFC 3339 time
  encoding. SQL UTC/millisecond conversion belongs to WP-02 adapters.
- `RiskLevel.Rank` returns -1 for invalid input. `MaxRisk` fails closed to imminent
  for invalid levels; a missing classifier must explicitly become elevated under
  SF-001 in WP-06. No risk transition policy is implemented by this helper.
- `Error` preserves `errors.Is/As` but renders only a safe code. `ToPublic` returns
  fixed text and excludes causes/steps. Schema errors likewise expose no payload.

## Schemas and scope of validation

`schema.New(Vocabulary)` creates an immutable draft 2020-12 schema set with no
external resource loading. A future pinned bundle supplies the vocabulary;
the only supplied content today is explicitly synthetic test data. Empty protocol
and formulation catalogs disable their corresponding references. Missing required
catalogs fail compilation. `Set.JSON` returns a copy for structured-output clients.

Use `DecodeBrain`, `DecodeMemoryWriter`, `DecodeReflection`, and `DecodeSafety` at
LLM ingress. Ordinary `json.Unmarshal` is insufficient: it does not check required
fields, schema constraints, or catalog membership. The strict decoders reject
unknown nested fields, duplicate keys, trailing JSON, invalid UTF-8, forbidden
nulls, invalid enums, unsupported versions, noncanonical/nil UUIDs and bad dates.
They validate structure before producing typed values. Input errors are not repaired
or silently clamped here. The caller can use BR-001/ErrBrainInvalid or the relevant
module's error code when the later module is implemented.

Schema reconciliation of incomplete sketches:

- Added missing memory/message definitions and recursive unknown-field rejection
  in `memwriter.v1`; optional `importance` uses a pointer in Go.
- Authored `reflection.v1` from §4.9. Approved framework names and field keys are
  supplied as vocabulary; evidence/session thresholds and wording checks remain
  RF/MC policy work. Reflection proposal fields have explicit JSON tags.
- Added a distinct `SafetyClassifierOutput`; `SafetySignal` is the aggregate gate
  result. The §5.3 classifier payload intentionally has no `schema_version` member;
  the call selects `safety.v1` out of band. Other LLM payloads carry their version.
- UUID schema constraints are visible to independent validators and providers,
  not only Go parsing. `safety_support` and `start` remain excluded from Brain
  output even though they are valid internal objectives/actions.
- Completed undefined port report/config/content/chunk types without implementations.
  `GenInput.Turn` supplies §11.3's current message; `ProtocolPatch.ConsentMessageID`
  carries the required provenance for the later consent transition.

Semantic/context-dependent BR repairs, ST/PR transitions, OV checks, SF routing,
MC/RF provenance checks, word-count limits and bundle content checks remain in
their assigned implementation packages (WP-03 and WP-06–14). A schema-valid
fixture does not assert that a plan is safe, eligible, owner-authorized or supported.

## Verification

From `apps/api`:

```sh
go test ./internal/modules/ai/contract/... ./internal/modules/ai/ports/...
go test -race ./internal/modules/ai/contract/... ./internal/modules/ai/ports/...
```

The 106 schema fixtures run independently through JSON Schema and strict Go
decoding, then accepted mirrors are marshaled and validated again. Other tests
cover enums, risk ordering, optional zeros, error redaction, cancellation metadata,
immutable catalogs and import boundaries. `ports/contracttest` runs against a
test-only reference double; actual SQL durability and transaction behavior are
explicitly deferred to the WP-02 adapter suite.

## WP-04 schema ingress

`schema.CompileInline` compiles an already-injected request schema using the
same strict JSON, draft 2020-12 format and no-external-resource rules as WP-01.
The LLM client uses this for native and JSON-mode responses, including repairs.
The static `Set` and schema Go mirrors are unchanged.
