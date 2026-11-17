# AI Core WP-03 — Knowledge Bundle Toolchain

This package implements the independently versioned, immutable Knowledge Bundle described in Contract Pack v1.2 §8. It implements the **existing WP-01** `ports.Bundle` and `ports.BundleRegistry` interfaces without changing either interface or importing the prototype AI prompt, safety, Gateway, LLM, or Orchestrator packages. It performs no database writes or model calls.

## Owned files and responsibilities

- `knowledge/`: versioned YAML, Markdown, and template assets. The current `kb@0.1.0` is **synthetic only**; no clinician-approved intervention, crisis resource, locale, policy, or fallback is supplied. `en` is a parser-test locale, **not launch approval** (especially not for Persian-first clients).
- `bundle/`: strict YAML file schemas, semantic checks, catalogue/step/tag references, RE2 lexicon compilation, locale coverage, release-integrity checks and immutable in-process snapshot registry.
- `cmd/aicore-bundlelint`: CI command. `-release` additionally enforces current crisis-resource verification, matching clinical approvals, complete artifact hashes and a trusted Ed25519 signature.
- `cmd/aicore-bundlerelease`: reviewer-only release packaging; generates content hashes and signs a verified non-synthetic bundle with a private key **outside** the repository.

## Commands (from `apps/api`)

```sh
go test ./internal/modules/ai/bundle ./cmd/aicore-bundlelint ./cmd/aicore-bundlerelease
go test -race ./internal/modules/ai/bundle
go run ./cmd/aicore-bundlelint -dir ../../knowledge -core-version 1.0.0
# Release must FAIL until content, resources, locale verification and approvals exist:
go run ./cmd/aicore-bundlelint -release -dir ../../knowledge \
  -core-version 1.0.0 -trusted-key 'review-key-id:BASE64_ED25519_PUBLIC_KEY'
```

The release tool accepts `-private-key-file` (base64 Ed25519 private key), `-key-id`, `-dir`, and `-core-version`. Never put a signing key in the source tree or in `knowledge/`; do not commit clinical sign-offs before they are issued. The release tool checks **all** release rules in a staged bundle before modifying `bundle.yaml` / writing `bundle.sig`. Distribute verified release directories as an immutable unit. The manifest digest is SHA-256 over the canonical JSON encoding of the manifest **with `checksums` cleared**; every other artifact has a separate SHA-256 of its exact bytes; the detached Ed25519 signature covers canonical JSON of the **complete** manifest, including the final checksums. The signer key is trusted explicitly by its ID.

## In-process API and failure modes

`Read(ctx, dir, Options)` validates a bundle and returns an immutable `*Snapshot` on success; validation errors include stable codes and file paths. `Snapshot.Reference()` returns `kb@x.y.z` and defensive copies of prompt SHA hashes; `Artifact(ctx, name)` returns defensive byte copies and rejects traversal/unknown paths. Golden evaluations are linted/hashed but deliberately excluded from runtime artifacts.

`NewRegistry(options, allowSynthetic)` and `Activate(ctx, dir)` publish a **whole** snapshot only after complete validation. `Active(ctx, userID)` returns a pin to a currently active snapshot, `ByVersion(ctx, "kb@x.y.z")` resolves previously loaded versions for replay, and `Rollback(ctx, version)` reactivates a previously loaded snapshot. Failed/incompatible updates retain the prior active pointer. Reusing a version number with different bytes is rejected. Versions are kept in-process (no disk/network registry or rollout/cohort strategy has been implemented); composition root wiring remains with WP-05 or its integration owner.

A production registry must be constructed with `allowSynthetic=false`. It then enforces release/signature verification even if `Options.Release` was false. Setting `allowSynthetic=true` is only for **synthetic engineering tests** and must not be enabled for public/real mental-health traffic.

## Policy and implementation limits

- All user-initiated deletion remains **soft-only**. The `memory.yaml` schema rejects hard-delete flags and any implicit deletion of derived records. The bundle does not authorize retrieval/reuse of retained records; separate AI-use policy and existing consent rules still apply.
- Predicates, clinical decisions, interventions, and psychological prompt text are **not** implemented by this package. Ordered router rules, runtime Protocol FSM, L0/L1 safety behavior and output validation belong to later WPs.
- The linter's token estimate (`max(word_count, ceil(UTF8_bytes/4))`) is an approximate conservative budget check, **not model-tokenizer certification**. Before release, calibrate against actual model tokenizers and run clinical/model golden evaluations.
- The forbidden-claims regex list is intentionally **empty** in the synthetic bundle. The release gate demands locale-specific reviewed patterns. No existing prototype regex patterns or `supportive-v5` content are imported.
- The synthetic bundle emits warnings for missing clinical approvals, unreviewed safety/crisis assets, empty forbidden claims, and inadequate golden coverage. These are expected development warnings; all clinically critical warnings become release errors where indicated in Contract Pack §8.11.
