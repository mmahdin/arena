# PROD-02 — Local Metabase deployment & acceptance runbook

## Preconditions / explicit no-go gates

1. Local synthetic-only `compose.yaml` stack and SQL Server 2022; no public ingress.
2. PROD-01 schema `0035_product_events.sql` plus its Go event writers are deployed.
3. Migration `0037_product_reporting_views.sql` must be applied **before**
   creating the analytics SQL login or connecting Metabase.
4. Connected machine has Go dependencies resolved (`go mod tidy`, `go build
   ./...`, `go test ./...`) and Docker Compose available. Earlier offline
   checks do not establish a runnable stack.
5. Do **not** enable real-user analytics, public sharing, embedding, or direct
   SQL `sa` datasource access. All Metabase routes stay on host loopback.

## 1. Validate / migrate the local SQL Server database

From repo root, use the **same Compose project name** as the existing local
backend. `migrate` applies only pending, checksum-verified migrations:

```bash
python3 scripts/check_structure.py
python3 scripts/check_contract.py
python3 -m unittest discover -s scripts/tests -q
cd apps/api && go mod tidy && go build ./... && go test ./... && cd ../..
docker compose -p theia-local --env-file .env -f compose.yaml up -d mssql
docker compose -p theia-local --env-file .env -f compose.yaml run --rm migrate
```

Migration 0037 is additive; creates four new reporting views and one `SELECT`
role. Existing event rows, users, and conversations are **not** rewritten.
Do not modify prior migrations. A code rollback can leave this additive schema
in place. Take database/application-state backups before any upgrade on a
shared environment, even if the local developer prototype can be recreated.

## 2. Generate secrets, provision the view-only SQL reader

```bash
python3 scripts/analytics/init_local_analytics_env.py
python3 scripts/analytics/bootstrap_reader.py
```

The first command writes `.env.analytics` mode 0600, ignored by Git. It includes
separate generated credentials for Metabase's PostgreSQL application database,
SQL Server reporting reader (64-character SQL password meeting CHECK_POLICY), and initial Metabase administrator. The second
command runs local-only Go `cmd/analytics-reader` against **127.0.0.1:14333**
and provisions `theia_metabase_reader` with membership in
`theia_analytics_reader`. Never put generated passwords in shell history,
configuration files committed to Git, or process command-line arguments.

`bootstrap_reader.py` reads local `.env` for the MSSQL SA password to perform
this one-time privileged operation. It does **not** give that password to
Metabase. Run this on the development machine, never as a service exposed to
end users. If local `.env` no longer contains `MSSQL_SA_PASSWORD`, use the Go
command with a separately configured local administrative `DATABASE_URL` and
`THEIA_METABASE_READER_PASSWORD` environment values.

## 3. Start independent Metabase and PostgreSQL application state

```bash
docker compose -p theia-local \
  --env-file .env --env-file .env.analytics \
  -f compose.yaml -f compose.analytics.dev.yaml \
  --profile analytics up -d analytics-postgres metabase
```

Only **http://127.0.0.1:3100** is published by this overlay. PostgreSQL is
not published. Metabase version `v0.64.8.3` (Open Source, 64 LTS line) and
PostgreSQL `16.16-alpine` are pinned intentionally. Review security updates
before upgrading; never use a moving `latest` tag. The Metabase application
state is persisted in Docker volume `theia_metabase_pg_data`.

## 4. Provision dashboards and run a permission probe

Wait until Metabase responds with `{"status":"ok"}` on `/api/health`, then:

```bash
python3 scripts/analytics/provision_metabase.py
```

For a Metabase JAR running directly on the host, pass `--native`. This uses
the existing loopback SQL Server endpoint at `127.0.0.1:14333`. Keep the
Metabase web listener on `127.0.0.1:3100` and use PostgreSQL for application
state in either setup.

This one-time or repeatable local script uses the generated initial Admin
credentials from `.env.analytics`, completes first setup, adds the
**view-only SQL Server datasource**, creates one analytics collection and
six saved questions in the **Theia · Synthetic Product Overview** dashboard.
It then verifies that an aggregate view can be queried but `users`,
`messages`, `product_events`, `turns` and `ai_llm_attempts` cannot be read.
It does not print secrets, user-level rows or failed-query response bodies.

The script intentionally fails if an existing datasource with the expected
name does not use `theia_metabase_reader`, or if a permission check fails.
It preserves existing cards on reruns. Passwords remain in the Metabase
PostgreSQL application DB encrypted using `MB_ENCRYPTION_SECRET_KEY`.

If you've already configured Metabase interactively using a different admin
account/password, use that account to make the equivalent changes manually or
start with a fresh local **Metabase application state**; do not delete
production/user data or overwrite credentials automatically.

## 5. Operational acceptance tests

- New synthetic events create aggregate rows; data never reveal identifiers
  or message/prompt text.
- Onboarding cohort report has expected `registered → started → completed`
  counts without using raw users as a datasource.
- Chat-active user counts use admitted-turn day; a duplicate Turn replay does
  not increase the count.
- First-chat D1/D7/D30 cohorts use exact calendar days; future/incomplete
  target days have NULL, not zero.
- AI cost graph shows known estimated portions and a separate coverage table;
  it must **not** label this as complete provider billing.
- Metabase session is required (no anonymous dashboard or public URL).
- `SELECT` on report views succeeds for the reader; direct base-table queries
  fail. No SQL Server administrator login appears in the datasource.
- Metabase/Postgres/SQL credential files are not Git-tracked and are mode 0600.
- Restart Metabase and verify the application state persists. Do not erase the
  PostgreSQL volume during normal restarts.

The optional disposable SQL Server integration test:

```bash
cd apps/api
APP_ENV=local WELLBEING_INTEGRATION=true go test ./tests/integration -run TestPROD02AggregateViewsAndReaderPermissions -count=1
```

The integration harness will independently validate that it targets a
loopback SQL Server before creating/deleting a temporary database. **Never
point its connection at the shared or production database.**

## Failure and rollback

- If the SQL login bootstrap fails, ensure migration 0037 is applied and the
  account has local `CREATE LOGIN` rights. Do not grant `sa` to Metabase.
- If Metabase login fails on initial setup, check local Metabase state and the
  admin credentials in `.env.analytics`; do not print them in CI logs.
- If SQL connection fails, inspect local MS SQL availability/reader membership;
  for this Compose-only local network the source JDBC option uses
  `encrypt=false`. **Stage/production require TLS and an approved secure
  network/SSO design, not this compose overlay.**
- If datasource permission probe fails, disable dashboards and investigate
  immediately. Do not temporarily use an admin connection to bypass the gate.
- Roll back Metabase by stopping the `analytics` profile; leave migration 0037
  in place. Preserve the PostgreSQL volume for future restoration.
- Metabase's local admin login is separate from Theia Staff Keycloak BFF; do
  not assume Theia's login session grants Metabase access or embed it publicly.

## Product-management milestone not yet delivered

The current source has **only synthetic** events. Real user lifecycle,
authorization, privacy retention, sampling, consent, and release/approval flows
are reserved for future OPS-01 and explicitly reviewed product work.
