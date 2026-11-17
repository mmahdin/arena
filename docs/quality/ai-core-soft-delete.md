# WP-02 soft-delete correction verification

Baseline: `6dd890d` (committed WP-02). Date: 2026-11-16.
Authority: the user's soft-delete correction, amended Contract Pack v1.2
§7.4–7.5, and [ADR 0010](../adr/0010-soft-delete-preservation.md).

## Checks

Passed from `apps/api`:

```sh
go test ./... -count=1
go test -race ./tests/integration -count=1
go build ./...
go vet ./...
```

Both test runs enabled `WELLBEING_INTEGRATION=true` and `APP_ENV=local` against
a dedicated disposable SQL Server 2022 container on loopback, using credentials
created solely for this test instance. Each integration suite creates and drops
its own randomly named synthetic database. Existing databases and credentials
were not used. The final race run includes the complete integration suite,
including the final deletion-metadata constraint check. No races were reported.
Go used `GOCACHE=/tmp/theia-go-build`, `GOPROXY=off`, and cached dependencies.

Passed from the repository root:

```sh
python3 scripts/check_structure.py
python3 scripts/check_contract.py
git diff --check
```

The structure check covers 41 required paths and six repository skills. The
OpenAPI baseline check verifies 30 method/route pairs against Gin composition;
it is not a full OpenAPI semantic linter.

## Evidence

- Fresh and repeat migrations, upgrades through 0019 to 0020, retained diary
  content/timestamps, backfilled row IDs, unchanged earlier migration files and
  checksums, validated foreign keys, no remaining delete cascades or purge proc.
- Exact stored snapshots of raw messages, turns, bookmarks, letters and AI
  sessions, user state, threads, formulations, memories, evidence, embeddings,
  protocol instances, loops, follow-ups, tombstones, cooldowns, turn context,
  events, jobs, traces and outbox before/after conversation soft deletion.
- Correct owner/reason/timestamp metadata, stable repeated deletion, hidden
  list/history/bookmarks, blocked rename/admission/replay/recovery of hidden
  conversations, and preserved independent visibility of derived letters.
- Preserved old diary/bookmark/letter rows after re-creation, one active row per
  logical key, original deleted text still stored, and concurrent bookmark saves.
  Existing diary tests cover concurrent saves with the revised owner lock.
- Injected failure after the visibility update rolls back deletion and leaves
  the active lease usable. Stale workers cannot complete after successful
  deletion; other conversations remain recoverable and usable.
- Concurrent delete/finalize runs preserve either the completed turn or its
  durable user-only admission, according to lock order. Neither outcome permits
  late publication through the deleted conversation.
- Attempts to physically delete raw data, accounts, memories, formulations,
  traces, jobs, diary entries and letters fail without data loss.
- Existing lifecycle/lease, HTTP ownership, provider-failure and retry tests
  pass with retained reservations instead of destructive prototype cleanup.

The patch is checked against a clean export of `6dd890d`: `git apply --check`,
application of the patch, byte comparisons of every changed file, and a build
of the resulting Go module.

## Scope and operational limits

Apply migration 0020 and the corrected binary together, with writers stopped
during migration. No old migration is edited, and no table or original payload
is dropped. Do not roll back to an old binary using purge/delete paths.

No new account/memory deletion API, restore endpoint, retention worker, permanent
erasure or legal-erasure policy is implemented. Existing AI-use/consent settings
are unchanged; retention and visibility do not establish AI-reuse permission.
Data physically erased before this correction cannot be reconstructed by it.
Database guards protect DELETE operations, not privileged administrative DDL or
an operator disabling those guards. Real-data deployment remains outside this
synthetic foundation's authorization.
