"""BO-04 structural regression guards. Runtime correctness needs Go/SQL tests."""
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]


class OperationalTelemetryTests(unittest.TestCase):
    def test_sql_tables_are_sanitized_and_nullable(self):
        sql = (ROOT / "apps/api/db/migrations/0027_ai_operational_telemetry.sql").read_text().lower()
        for table in ("ai_llm_attempts", "ai_stage_events", "ai_worker_heartbeats"):
            self.assertIn("create table dbo." + table, sql)
        self.assertIn("cost_micro_usd bigint null", sql)
        self.assertIn("input_tokens bigint null", sql)
        self.assertIn("started", sql)
        self.assertIn("estimated", sql)
        self.assertIn("unpriced", sql)
        self.assertIn("last_seen_at", sql)
        for field in ("user_id uniqueidentifier", "prompt_text nvarchar", "response_body", "error_message nvarchar", "api_key"):
            self.assertNotIn(field, sql)

    def test_provider_attempts_are_recorded_before_network_calls(self):
        llm = (ROOT / "apps/api/internal/modules/ai/llm/attempt_telemetry.go").read_text()
        self.assertIn("p.journal.BeginAttempt(ctx, a)", llm)
        self.assertIn("p.journal.FinishAttempt(persistCtx, e)", llm)
        self.assertIn("context.WithoutCancel", llm)
        self.assertIn('"unpriced"', llm)
        self.assertIn('"estimated"', llm)
        runtime = (ROOT / "apps/api/internal/app/composition/ai_runtime.go").read_text()
        # MON-01A may decorate the durable journal to export operational
        # counters, but it MUST continue to call the SQL journal first.
        self.assertIn("var journal llm.AttemptJournal = telemetryStore", runtime)
        self.assertIn("llm.Options{Attempts: journal}", runtime)
        monitoring = (ROOT / "apps/api/internal/app/composition/monitoring.go").read_text()
        self.assertIn("return j.next.BeginAttempt(ctx, a)", monitoring)
        self.assertIn("j.next.FinishAttempt(ctx, a)", monitoring)
        self.assertIn("j.meter.ObserveLLM", monitoring)
        self.assertIn("WorkerPulse: telemetryStore", runtime)
        self.assertIn("stages: telemetryStore", runtime)

    def test_tracer_is_not_local_full_payload_inspector(self):
        trace = (ROOT / "apps/api/internal/app/composition/ai_logging.go").read_text()
        self.assertIn("approvedStep", trace)
        self.assertIn("trace.ApprovedRule", trace)
        self.assertIn("RecordStage", trace)
        self.assertNotIn("payload_enc", trace)
        repo = (ROOT / "apps/api/internal/modules/ai/adapters/mssql/telemetry.go").read_text()
        self.assertNotIn("local_ai_trace", repo)
        self.assertNotIn("ai_turn_traces", repo)
        self.assertIn("WHERE id=CONVERT(uniqueidentifier,@p1) AND status='started'", repo)


if __name__ == "__main__":
    unittest.main()
