# OBS-03 — Admin Turn / OTel trace correlation runbook

1. Apply backend patches through OBS-03 and migration `0034_ai_trace_refs.sql`
   on a **disposable SQL Server 2022** instance; retain all earlier migrations.
2. Enable OBS-01A metadata tracing and verify OBS-02 Phoenix ingestion. The API
   uses metadata only after AUTH-09.
3. Run the Go API with an available SQL connection and an approved OTel
   receiver, and create a synthetic Turn through the existing product path.
4. Authenticate an **active Staff** on the exact Admin origin and query:

   `GET /api/v1/admin/turns/<turn-uuid>/trace-references?pageSize=25`

   The response is bounded and contains only W3C IDs. A Turn with no refs
   yields `items:[]`; this can mean tracing disabled, old Turn, SQL write timeout
   or an unrecorded root. A recorded ID can still be absent from Phoenix if
   export failed. Check Phoenix separately.
5. Compare `traceId` with Phoenix's authenticated trace details. For a memory
   or reflection job, compare `jobId`, `jobAttempt`, and its **independent** root.
6. Prove the deny cases: unauthenticated, User-realm, Dev Session, inactive
   Staff, wrong Host, invalid filter, and audit outage. The audit log should
   record `admin.turn.trace_refs.read` before successful reads.
7. Capture evidence of SQL migration, local Go tests, and Phoenix readback.
   Do not claim OBS-03 live-complete without these results.

No direct SQL, private local Inspector files, raw messages, OAuth tokens, model
API keys or Phoenix URLs may be exposed to the browser from this endpoint.
The metadata mapping is best-effort under SQL outage. The full content access
policy and Phoenix authentication remain separate engineering gates.
