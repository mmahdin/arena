# BO-09 — Knowledge Bundle inspection (read-only)

The Back-Office Knowledge screen uses `GET /api/v1/admin/bundles` under the
existing Staff session, Staff origin binding, and append-only Admin audit. No
new SQL migration or Keycloak role is required.

## What the endpoint actually knows

- **Active** is the exact immutable `bundle.Snapshot` pinned by AI Core during
  `loadProduct`. Its filename and document bytes are never sent to Admin.
- **Candidate** is an optional, explicit, absolute directory specified by the
  deployment operator in `bundleCandidates` in the same `ai_runtime.v2` config.
  There is no directory traversal, discovery, HTTP parameter, upload, promotion,
  rollback or hot refresh in the Admin API. Maximum five candidates.
- Both active and candidate inventory entries are captured **only on API startup**.
  `asOf` is the capture time; restart to refresh after an operator updates
  deployed assets. There is **no cross-deployment release database** yet.
- If the AI runtime is not configured, `scope: not_configured`, `active: null`
  and an empty `candidates` list are returned. An empty candidate list is not
  evidence that no other candidate artifacts exist elsewhere.

## Optional read-only candidates

The existing runtime example has only `bundleDirectory`. Leave it that way
unless candidates are explicitly needed. An operator can add:

```yaml
bundleCandidates:
  - /absolute/operator-owned/knowledge-candidate-1
  - /absolute/operator-owned/knowledge-candidate-2
```

Candidate bundles are parsed through the **same strict validator** as the
active bundle, with the same trusted public keys and `CoreVersion=1.0.0`.
Outside local synthetic-draft mode, candidates must pass release checks,
including reviewed artifact hashes and Ed25519 signature verification.
Validation errors do not interrupt the running AI instance or change its active
bundle. They are represented as `invalid` with stable error-code counts; file
paths/messages are never exposed. Reusing an active version with different
bytes is reported `version_conflict` and is not publishable.

## Understanding trust labels

| Value | Meaning |
| --- | --- |
| `verified_release` | Package integrity and review-hash gates passed with a configured Ed25519 public key |
| `synthetic_draft` | Local-only synthetic bundle passed development lint; **not signed, not approved** |
| `invalid` | Candidate rejected by validation; no usable metadata or content exposed |
| `version_conflict` | Same version as already observed, different immutable fingerprint |
| `signatureVerified: true` | Detached signature matched trusted key; **not reviewer identity attestation** |
| `reviewEvidence: hash_matched` | Recorded review hashes matched files; **not a clinical approval decision** |

No `clinicallyApproved` field is provided, by design. Signatures, checksums,
release lint and synthetic evaluation status do **not** imply real-user launch
approval. Clinical review and final release workflows remain BO-12.

## Validation

From repository root:

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -v
cd apps/api && go test ./internal/modules/ai/bundle ./internal/modules/admin/... ./internal/app/composition/...
```

Actual browser + Staff/SQL audit tests must be run against your local stack.
Without a configured Staff BFF, all Admin routes remain inaccessible; no
synthetic user or dev cookie can bypass the guard.
