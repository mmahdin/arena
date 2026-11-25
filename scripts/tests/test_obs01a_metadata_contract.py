"""OBS-01A structural regression gates, not a replacement for Go/OTLP E2E."""
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]


def src(path):
    return (ROOT / path).read_text(encoding="utf-8")


class OBS01AMetadataContract(unittest.TestCase):
    def test_explicit_disable_and_capture_fail_closed(self):
        config = src("apps/api/internal/platform/config/config.go")
        self.assertIn('THEIA_OTEL_ENABLED', config)
        self.assertIn('THEIA_OTEL_CAPTURE_CONTENT', config)
        self.assertIn('telemetry.Validate(cfg.OTelOptions)', config)
        self.assertNotIn('THEIA_OTEL_CAPTURE_CONTENT is unavailable in the API after AUTH-09', config)
        self.assertIn('cfg.OTelOptions.LoopbackListener', config)
        self.assertIn('telemetry.Validate(cfg.OTelOptions)', config)
        sdk = src("apps/api/internal/platform/telemetry/telemetry.go")
        self.assertIn('oteltrace.WithNewRoot()', sdk)
        self.assertIn('WithBatcher(exporter', sdk)
        self.assertNotIn('gen_ai.input.messages"', sdk)
        self.assertNotIn('gen_ai.output.messages"', sdk)
        self.assertNotIn('RecordError(err)', sdk.replace('// never RecordError(err)', ''))

    def test_injected_only_at_host_boundaries(self):
        api = src("apps/api/internal/app/composition/api.go")
        core = src("apps/api/internal/app/composition/ai_product.go")
        self.assertIn('ctx := c.Request.Context()', api)
        self.assertIn('ctx = telemetry.WithRuntime(ctx, cfg.OTelRuntime)', api)
        self.assertIn('c.Request.WithContext(trace.WithReferenceWriter(ctx, api.traceRefWriter))', api)
        self.assertIn('telemetry.WithRuntime(ctx, a.otel)', core)
        self.assertNotIn('propagation.TraceContext', api)

    def test_sql_attempt_order_and_private_http_capture_unchanged(self):
        metered = src("apps/api/internal/modules/ai/llm/attempt_telemetry.go")
        self.assertLess(metered.index('p.journal.BeginAttempt(ctx, a)'), metered.index('startAttemptSpan(ctx, a)'))
        self.assertIn('FinishAttempt(persistCtx, e)', metered)
        http = src("apps/api/internal/modules/ai/adapters/llmhttp/provider.go")
        self.assertIn('if trace.Enabled(ctx) {', http)
        self.assertIn('ctx, wireSpan = trace.Start(ctx, "http.request"', http)


if __name__ == "__main__":
    unittest.main()
