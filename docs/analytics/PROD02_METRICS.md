# PROD-02 — Synthetic-only product reporting contract

**Scope:** SQL Server 2022; aggregate-only views; data begins at PROD-01 rollout.
These are **fictional test-user metrics**, **not production adoption numbers**,
**not clinical outcomes**, and **not financial invoices**. Reporting rows are
aggregates over synthetic users, with no user/session/turn identifiers returned.

## View contract (migration `0037_product_reporting_views.sql`)

| View | Grain | Exact interpretation |
| --- | --- | --- |
| `dbo.v_product_daily_events_synthetic` (PROD-01) | UTC day + event name | Event count and distinct test users *per event/day*. Not global DAU. |
| `dbo.v_product_onboarding_cohorts_synthetic` | UTC date of `user_registered` | Number of registered synthetic users whose onboarding was ever started/completed. Late conversions update past registration cohorts. No account backfill. |
| `dbo.v_product_daily_engagement_synthetic` | UTC day of event | `chat_active_test_users` = distinct users with one or more **`ai_turn_admitted`** events that day. Includes also counts for conversations, bookmarks, first completed/failed Turn events by event date. Not app logins, active sessions, true DAU, or user outcomes. |
| `dbo.v_product_chat_retention_synthetic` | UTC date of first `ai_turn_admitted` | First-chat cohort and distinct users who admitted another Turn on **exact UTC day +1, +7, +30**. A return on day +2 does not count on day +1. Future or partially observed target days have **NULL** returning users, never an invented zero. Denominator is the full first-chat cohort. No 24-hour or rolling retention. |
| `dbo.v_product_ai_usage_daily_synthetic` | UTC date of LLM attempt `started_at` | **Physical, turn-linked** attempts associated with synthetic owners. `known_estimated_micro_usd` is the sum of configured-price estimates **only when cost is known**; `unpriced_attempts` and `missing_usage_attempts` expose partial coverage. Orphan/background calls are out of scope. Not authoritative billing. |

## Caveats that dashboards must show

- No historical backfill before migration 0035: missing events are *unknown*, not zero.
- Synthetic data only (`users.is_test = 1` in all new reporting views). Metrics
  are not evidence of successful rollout, clinical effectiveness or user safety.
- The initial event vocabulary cannot measure product sessions, app opens,
  geographic breakdown, session duration, true app DAU/WAU/MAU, or user
  satisfaction. Do not fabricate them with a Metabase calculation.
- `ai_turn_failed` records a logical Turn's **first failed transition**. The
  same Turn can later complete, so `failed / (failed+completed)` from this
  event ledger is NOT the terminal Turn failure rate. Use live Admin overview
  with its distinct definition for that operational metric.
- Onboarding `started` means cursor transition 0→1, not first impression.
  `completed` means first profile completion, not a clinical onboarding result.
- `cost_status != 'estimated'` means excluded from the cost sum, not $0.
  Provider invoices and missing background attempts remain unaccounted for.
- All fields use UTC day boundaries. Late-recorded events can change historical
  aggregates. SQL views scan current source facts; views are not immutable snapshots.
- Only synthetic test-user reporting is authorized. Real-person analytics need
  privacy/legal/security review, retention/erasure rules, role separation,
  appropriate cell-size suppression, documented lawful basis, and new controls.

## Permission boundary

Migration 0037 creates `theia_analytics_reader`, grants SELECT on five approved
aggregate views and explicitly DENIES reads of sensitive base tables. After
migration, the operator provisions a dedicated `theia_metabase_reader` SQL login
using `scripts/analytics/bootstrap_reader.py` (requires local administrative
SQL access). The Metabase **report datasource** uses this login exclusively.
Do not grant `db_owner`, `db_datareader`, SQL Server `sa`, or arbitrary native
query permissions. Metabase Open Source is not a substitute for DB-level RLS.

Metabase keeps dashboard, user, connection metadata and credentials in a
**separate PostgreSQL application database**; it does not turn the analytics
source into PostgreSQL. The SQL Server source stays authoritative.
