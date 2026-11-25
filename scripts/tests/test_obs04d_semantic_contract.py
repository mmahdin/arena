"""Static guardrails for OBS-04D; NOT a replacement for Go/OTLP integration tests."""
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
TELEMETRY = ROOT / "apps/api/internal/platform/telemetry"

class OBSDPortableSemanticContract(unittest.TestCase):
    def test_kinds_are_applied_on_existing_spans(self):
        source = (TELEMETRY / "semantic.go").read_text()
        start = (TELEMETRY / "telemetry.go").read_text()
        for expected in ('"AGENT"', '"RETRIEVER"', '"RERANKER"', '"PROMPT"',
                         '"GUARDRAIL"', '"LLM"', '"EMBEDDING"', '"CHAIN"'):
            self.assertIn(expected, source)
        self.assertIn('attribute.String("openinference.span.kind", kind)', source)
        self.assertIn('semanticStart(name, attrs)', start)
        self.assertIn('semanticAttemptKey{}', start)
        self.assertNotIn('otel.SetTracerProvider', source)

    def test_full_content_stays_behind_existing_gate_and_redaction(self):
        stage = (TELEMETRY / "stages.go").read_text()
        rag = (TELEMETRY / "rag.go").read_text()
        wire = (TELEMETRY / "content.go").read_text()
        for s in (stage, rag, wire):
            self.assertIn('!ContentEnabled(ctx)', s)
            self.assertIn('redactJSON', s)
            self.assertIn('MaxWireCaptureBytes', s)
        self.assertLess(stage.index('redactJSON(data, "")'), stage.index('semanticStageIO(span, direction, clean)'))
        self.assertLess(rag.index('redactJSON(data, "")'), rag.index('semanticRAG(span, name, clean)'))
        self.assertIn('if omitted == "" {\n\t\tsemanticWire(', wire)
        source = (TELEMETRY / "semantic.go").read_text()
        self.assertIn('if direction != "response" || !complete {', source)
        self.assertIn('if json.Unmarshal([]byte(cleaned), &body) != nil || body["error"] != nil', source)

    def test_no_extra_llm_attempt_or_unapproved_content_metadata(self):
        source = (TELEMETRY / "semantic.go").read_text()
        self.assertIn('"theia.ai.http.client"', (TELEMETRY / "http.go").read_text())
        self.assertIn('case "theia.ai.http.client":\n\t\treturn ""', source)
        self.assertIn('const maxDocuments = 16', source)
        self.assertIn('theia.dev.retrieval.semantic_documents_omitted', source)
        self.assertNotIn('gen_ai.input.messages', source)
        self.assertNotIn('gen_ai.output.messages', source)
        meta = (TELEMETRY / "telemetry.go").read_text()
        self.assertIn('case "theia.stage.operation":', meta)
        self.assertNotIn('case "input.value"', meta)
        self.assertNotIn('case "output.value"', meta)

if __name__ == '__main__':
    unittest.main()
