# AI Core WP-01 validation

Date: 2026-11-16. Base commit: `15f600f048f23fbfe0f946836fb80e8b628cfbb7`.
Authority: Final Architecture Document + Contract Pack v1.2 in `docs/ai-core/`.
Owner: AI Core contracts/ports. Scope: WP-01 only.

## Changes

- Canonical UUID-based contracts for turn admission/results, safety, working
  context, retrieval, Brain plans, protocol proposals, validation, state,
  memory/reflection, usage, events and errors; closed enum validation.
- Consumer-owned in-process ports, including Conversation storage, fenced
  per-user coordination, atomic finalization and provider-neutral model contracts.
- Four embedded draft 2020-12 schemas, immutable vocabulary injection, strict
  decoding and sanitized errors; no production bundle content or model calls.
- 106 structural fixtures plus JSON edge cases, enum/serialization tests,
  dependency guards, and reusable lifecycle/lease tests with a test-only double.
- Explicit transitional prototype inventory and bounded removal assignments in
  ADR 0008. Existing runtime is not wired to the new contracts yet.

## Executed checks

Environment: Linux amd64, Go 1.25.4; module baseline remains Go 1.23.0.
Go commands used `GOCACHE=/tmp/theia-go-build GOPROXY=off` with cached dependencies.

| Check | Result |
| --- | --- |
| `go test ./...` from `apps/api` | PASS; SQL Server tests skip without their opt-in flag |
| `go test -race ./internal/modules/ai/contract/... ./internal/modules/ai/ports/...` | PASS |
| `go build ./...` from `apps/api` | PASS, including API and migration commands |
| `python3 scripts/check_structure.py` | PASS |
| `python3 scripts/check_contract.py` | PASS; 30 existing Gin/OpenAPI method-route pairs |
| `git diff --check` | PASS |

The first full test run encountered a sandbox prohibition on loopback sockets in
an existing provider `httptest` test. The suite passed after rerunning with loopback
access. No live model endpoint was used. A separate offline `go mod tidy` attempt
could not fetch uncached test-only dependencies of existing dependencies from the
read-only module cache; this did not prevent tests or compilation. The new direct
dependency and its checksums are included explicitly.

## Deliberately unexecuted later-phase checks

`WELLBEING_INTEGRATION` was not enabled: no live SQL Server migration, storage,
transaction, data-retirement or crash-recovery validation is claimed. There are no
schema migrations in WP-01. The reusable port suite proves its reference double's
contract behavior; real adapters must satisfy it and additional SQL tests in WP-02.
No actual model/provider, runtime safety, prompt rendering, clinical evaluation,
bundle activation, HTTP lifecycle replacement or later-phase implementation is
claimed by these structural tests.

## Patch application

Apply `theia-ai-core-wp01.patch` from a clean backend checkout at the base commit:

```sh
git apply --check /path/to/theia-ai-core-wp01.patch
git apply /path/to/theia-ai-core-wp01.patch
```

The patch contains implementation, tests and repository documentation; the input
architecture specifications remain untouched. No commit, staging, deployment or
database mutation is required to consume it.
