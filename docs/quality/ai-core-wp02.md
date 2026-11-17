# AI Core WP-02 verification

Historical report for commit `6dd890d`. Its purge/cascade assertions are superseded
by [the soft-delete correction](ai-core-soft-delete.md) and ADR 0010.

Authority: Contract Pack v1.2, WP-02. Baseline: WP-01 commit `279b93a`.
Verification date: 2026-11-16. Scope and operational safeguards are documented in
[ADR 0009](../adr/0009-ai-core-sql-lifecycle.md).

## Results

All commands below passed. Go commands run from `apps/api`; repository checks
run from the repository root. The environment used `GOCACHE=/tmp/theia-go-build`
and `GOPROXY=off` with already available dependencies.

```sh
go test ./... -count=1
go test -race ./tests/integration -run '^TestAICore' -count=1 -v
go build ./...
go vet ./...
python3 scripts/check_structure.py
python3 scripts/check_contract.py
git diff --check
```

Both test commands ran with `WELLBEING_INTEGRATION=true`, `APP_ENV=local`, and
`DATABASE_URL` pointing to a dedicated disposable SQL Server 2022 instance on
loopback. Tests assert SQL Server major version 16, create randomly named isolated
databases, and remove them afterward. No existing database credentials or deployed
data were used. The full backend suite, including existing integration tests,
passed; the AI integration race run reported no races. Architecture checks covered
41 paths and six skills; OpenAPI checks covered 30 Gin/OpenAPI route pairs.

## Coverage

- Fresh migrations, repeated application, upgrade from 0016 with historical raw
  messages and processing turns, checksum tampering, and transactional rollback
  of 0017 when historical duplicate-role messages prevent the unique index.
- WP-01 port contract harness on actual SQL Server: durable user-first admission,
  idempotent duplicates, owner isolation, per-user serialization, expired leases,
  stale fencing tokens, deadlines, and completed-result replay.
- Concurrent distinct requests across repository instances and conversations;
  FIFO admission preserves all inputs and permits only one active turn per user.
  Different users can complete concurrently.
- Restart recovery, lease renewal, explicit failed-turn retry, running and queued
  cancellation, normalized request identity, and input storage bounds.
- Atomic assistant, state projections, events, jobs, trace, context and outbox;
  optimistic conflicts and injected failures after state writes and immediately
  before final commit leave no partial completion. Retrying succeeds.
- Owner and consent constraints, polymorphic references, proactive opt-out,
  conversation deletion with complete derived-data invalidation, stale-worker
  fencing after deletion, full account cascade, and preservation of other users
  and other conversations' raw messages.
- Embedding codec metadata, finite normalized float32 vectors, dimensions,
  malformed encodings, and round trips.

The existing phone-identity integration fixture was updated to fingerprint its
original columns explicitly across additive migrations. Its unknown-field PATCH
expectation now matches the existing strict JSON rejection, followed by an
identity check. No user-profile production behavior changed.

## Deliberate limits

Daily Reflection table retirement remains gated on verified disposable data or
an approved export/deletion plan; no table or old migration was removed. Existing
prototype HTTP entrypoints remain transitional as described in ADR 0008. The new
composition factory is ready for WP-05, but does not activate a new AI runtime.

Clinical state validation, retrieval, encryption/KMS integration, retention and
background workers, fine-grained memory provenance/deletion, and public API
cutover belong to later work packages. This verification uses synthetic fixtures;
it does not establish those later-phase behaviors or production readiness.

The deliverable patch is generated against the WP-01 baseline and checked by
applying it to a clean export, comparing every changed file, and building the
resulting Go module.
