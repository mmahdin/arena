# ADR 0005 — Fenced, recoverable conversation turns

Status: accepted for **internal synthetic-data MVP only** (Task 04).

## Context

Completed turns already commit user and assistant messages atomically. However, if
an API process crashes after claiming a turn, the `processing` row remains
indefinitely. Retrying the original `clientTurnId` cannot complete.

## Decision

- Keep `POST /api/v1/conversations/{id}/turns` as non-streaming JSON.
- Add migration `0006_turn_recovery_lease.sql` with nullable `lease_id` and
  `lease_expires_at` on `dbo.turns` and a constraint: both present on processing
  turns, neither on completed turns.
- A new or reclaimed claim gets a random UUID lease and 120-second expiry,
  calculated with SQL Server UTC time. The configured LLM HTTP timeout is 35s.
- Retry the **same** turn ID with identical content. Active lease -> 409;
  expired lease -> an atomic ownership- and hash-scoped update may reclaim;
  completed turn -> replay committed response and message IDs. Changing text
  -> 400. Wrong owner -> non-enumerating 404.
- A worker must hold the **current lease UUID** to commit both messages in one
  transaction or to release an unfinished reservation. The row lock on complete
  serializes against a concurrent reclaim; an obsolete worker returns 409.
- The DB migration marks old `processing` turns expired, not deleted. For a
  safe rollout **stop older app versions first**; deploy migration and updated
  binary together. Applied migrations 0001-0005 are immutable.

## Limitations

- A recovery retry may regenerate provider output if the earlier response was
  lost before commit. It still stores only one committed pair per turn.
- Expiry is fixed; very slow model/network calls could be fenced before they
  finish. Before changing LLM timeout, revisit lease duration and monitoring.
- No background sweeper or push event: the client must explicitly retry the
  same ID after a 409. No request-body or full-model-output logging.
- Independent concurrent turn IDs in one conversation are not serialized;
  message order follows transaction completion, not request arrival. Cursor
  pagination for more than 100 messages is deferred.
- Not suitable for clinical data or publicly exposed identity/session handling.

## Verification

Unit tests: token forwarded, pending turn short-circuits, error releases only
its token. Live MSSQL: expired claim reclaim race, changed hash, foreign user,
late completion/release, saved replay, exactly two messages. Run live integration,
smoke and E2E after applying. No SQL Server runtime validation claimed when
DB engine is unavailable.
