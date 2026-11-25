# WP-03 delivery notes — Knowledge Bundle Toolchain

**Scope / owner:** `knowledge/`, `apps/api/internal/modules/ai/bundle/`, `apps/api/cmd/aicore-bundlelint/`, `apps/api/cmd/aicore-bundlerelease/`, and this QA note only. Designed against Final Architecture §5/6 and Contract Pack v1.2 §1.2, §8 and §13. Neither WP-05 nor any LLM, Gateway, Orchestrator, shared contract, migration, or experimental AI file was modified.

## Acceptance coverage

| Contract criterion | Evidence |
|---|---|
| Manifest `kb@semver`, core ranges and emitted schema versions | Strict manifest validation and negative tests |
| Skill Card format, catalogue cap, token/descriptor budgets, references and examples | Strict YAML structs, lint codes, broken-bundle table tests |
| Protocol step graph, PR-001/002/005/006/007, eligibility and templates | Typed protocol schema, references and negative tests; runtime consent/input capture belong to WP-13 |
| Typed user-information, problem-understanding and formulation schema files | Strict decode, field/slot/evidence and range checks |
| Policy, lexicons, crisis resources, fallback and locales | Review placeholders, RE2 patterns, exact locale coverage, stale resource warning/release error, safe static templates |
| Router, budgets, threshold, models, memory and sampling config files | Strict typed YAML, bounded values, safe soft-delete gates |
| CI linter and golden packaging | `aicore-bundlelint` reports issues deterministically; golden files are hashed/linted but excluded from runtime snapshot |
| Integrity, approval hashes, signed release | SHA-256 content table, canonical manifest SHA, Ed25519 verification against configured public keys; signing preflight refuses synthetic/unapproved assets |
| Atomic activation, failure safety and replay | Immutable snapshot handles, version collision detection, failed activation leaves previous active, rollback/by-version and concurrent reader tests |

## Safety and unresolved matters

- **No approved psychological or clinical content is provided.** `kb@0.1.0` is an intentionally inert, synthetic engineering bundle, not an app launch/release artifact. `en` is illustrative; approved `fa-*` safety, resources, policies, templates and tests remain open ([OPEN-1]).
- Real crisis resource verification, staffed human escalation, clinical approvals and legal/security gates remain outstanding. The release pathway must reject this bundle.
- Regex/list matching and approximate token estimates do not certify clinical safety or model-specific token use. Later safety/validator/golden WPs must use reviewed content; WP-03 has not implemented their runtime algorithms.
- WP-05 composition root wiring and user-cohort version rollout are not touched to avoid conflicting with the parallel agent. `Active` currently selects the single active version for all users; per-cohort flag routing is a future integration decision.
- No data was deleted or migrated; immutable bundle changes cannot mutate persisted user data. Soft Delete and AI-use-permission separation remain unchanged.

## Verification commands (normal CI environment)

```sh
cd apps/api
go test ./internal/modules/ai/bundle ./cmd/aicore-bundlelint ./cmd/aicore-bundlerelease
go test -race ./internal/modules/ai/bundle
go vet ./internal/modules/ai/bundle ./cmd/aicore-bundlelint ./cmd/aicore-bundlerelease
go run ./cmd/aicore-bundlelint -dir ../../knowledge
```

See implementation handoff for the actual environment results, which must distinguish full dependency verification from offline type-check stand-ins.
