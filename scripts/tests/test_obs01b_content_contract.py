"""OBS-01B static guards: supplements, never replaces, Go tests/live checks."""
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]

def src(path):
    return (ROOT / path).read_text(encoding="utf-8")

class OBS01BContentContract(unittest.TestCase):
    def test_capture_is_private_synthetic_only(self):
        config = src('apps/api/internal/platform/config/config.go')
        telemetry = src('apps/api/internal/platform/telemetry/telemetry.go')
        for key in ('THEIA_OTEL_CAPTURE_CONTENT', 'THEIA_OTEL_CONTENT_API_KEY'):
            self.assertIn(key, config)
        self.assertNotIn('THEIA_OTEL_CAPTURE_CONTENT is unavailable in the API after AUTH-09', config)
        for flag in ('o.LoopbackListener', 'o.CaptureContent', 'o.PrivateIngestKey'):
            self.assertIn(flag, telemetry)
        self.assertIn('otlptracehttp.WithHeaders', telemetry)
        self.assertIn('u.Port() != "6006"', telemetry)
        self.assertIn('limits.AttributeValueLengthLimit = -1', telemetry)
        self.assertIn('limits.EventCountLimit = 128', telemetry)

    def test_no_raw_wire_content_in_general_metadata_allowlist_or_sql(self):
        telemetry = src('apps/api/internal/platform/telemetry/telemetry.go')
        allowed = telemetry.split('func allowed(f Field)')[1].split('func sanitize(')[0]
        self.assertNotIn('theia.dev.wire.', allowed)
        self.assertNotIn('gen_ai.input.messages', allowed)
        self.assertNotIn('gen_ai.output.messages', allowed)
        self.assertIn('if !ContentEnabled(ctx)', src('apps/api/internal/platform/telemetry/content.go'))
        self.assertNotIn('theia.dev.wire.', src('apps/api/internal/modules/ai/llm/attempt_telemetry.go'))

    def test_capture_occurs_after_json_serialization_and_stream_reads_are_not_drained(self):
        provider = src('apps/api/internal/modules/ai/adapters/llmhttp/provider.go')
        self.assertLess(provider.index('body, err := json.Marshal(payload)'), provider.index('telemetry.CaptureRequest(ctx, body, p.key)'))
        stream = src('apps/api/internal/modules/ai/adapters/llmhttp/content_capture.go')
        self.assertIn('b.next.Read(p)', stream)
        self.assertIn('b.once.Do', stream)
        self.assertIn('s.body.emit(err == io.EOF)', stream)
        self.assertNotIn('io.ReadAll', stream)
        self.assertNotIn('s.next.Read', stream)

if __name__ == '__main__':
    unittest.main()
