"""OBS-04C static wiring checks; supplement, not a substitute for Go/runtime tests."""
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
BASE = ROOT / "apps/api/internal/modules/ai"
TELE = ROOT / "apps/api/internal/platform/telemetry"


class OBS04CInstrumentation(unittest.TestCase):
    def test_existing_stages_export_lazy_io_only_with_content(self):
        code = (BASE / "trace/inspector.go").read_text()
        self.assertIn('if telemetry.ContentEnabled(ctx) && input != nil {', code)
        self.assertIn('captureOTelStage(ctx, "input", input)', code)
        self.assertIn('captureOTelStage(p.otelCtx, "output", output)', code)
        self.assertIn('telemetry.StageEvent(ctx, "input", raw)', code)
        self.assertIn('telemetry.StageEvent(p.otelCtx, "output", raw)', code)
        self.assertIn('telemetry.StageArtifact(ctx, name, kind, value)', code)
        self.assertIn('if sc.node == nil && !telemetry.ContentEnabled(ctx)', code)
        self.assertIn('if recover() != nil {', code)

    def test_provider_wrapper_no_longer_requires_file_inspector_for_full_content(self):
        code = (BASE / "llm/inspection.go").read_text()
        self.assertEqual(code.count('if !trace.Enabled(ctx) && !telemetry.ContentEnabled(ctx) {'), 3)
        self.assertIn('span.Complete(func() any { return map[string]any{"json": string(out.JSON)', code)
        self.assertIn('"provider.Stream"', code)
        self.assertIn('"provider.Embed"', code)

    def test_http_request_spans_wrap_actual_provider_io(self):
        code = (BASE / "adapters/llmhttp/provider.go").read_text()
        self.assertIn('ctx, httpSpan := telemetry.StartHTTP(ctx, resource)', code)
        self.assertIn('telemetry.HTTPStatus(httpSpan, response.StatusCode)', code)
        self.assertIn('response.Body = observeHTTPBody(response.Body, httpSpan, streaming)', code)
        self.assertIn('newContentBody(response.Body, httpCtx, p.key,', code)
        self.assertIn('capturedContentBody(response.Body)', code)
        self.assertIn('return observeHTTPStream(stream, response.Body), nil', code)
        self.assertLess(code.index('httpSpan := telemetry.StartHTTP'), code.index('p.http.Do(request)'))
        self.assertIn('httpSpan.End(failure)', code)
        self.assertNotIn('otel.SetTracerProvider', code)
        body = (BASE / "adapters/llmhttp/http_otel.go").read_text()
        self.assertIn('b.next.Read(p)', body)
        self.assertIn('b.next.Close()', body)
        self.assertIn('telemetry.HTTPIncomplete(b.span)', body)
        self.assertIn('func capturedContentBody(', body)
        self.assertIn('func (b *httpObservedBody) completeStream()', body)
        self.assertIn('func (s *httpObservedStream) Recv()', body)
        self.assertNotIn('io.ReadAll', body)

    def test_secret_redaction_and_bounded_synthetic_payload_events(self):
        code = (TELE / "stages.go").read_text()
        for required in ['!ContentEnabled(ctx)', 'MaxWireCaptureBytes', 'redactJSON(data, "")',
                         'runtime.privateIngestKey', 'theia.dev.stage.omission_reason',
                         '"input", "output", "artifact"', 'serialization_panic']:
            self.assertIn(required, code)
        self.assertNotIn('otel.SetTracerProvider', code)
        self.assertNotIn('log.Print', code)
        self.assertNotIn('fmt.Print', code)
        self.assertNotIn('"chunk", "stream_chunk"', code)
        http = (TELE / "http.go").read_text()
        self.assertIn('"http.response.status_code"', http)
        self.assertIn('"http.request.method"', http)
        self.assertNotIn('server.address', http)
        self.assertNotIn('http.url', http)


if __name__ == "__main__":
    unittest.main()
