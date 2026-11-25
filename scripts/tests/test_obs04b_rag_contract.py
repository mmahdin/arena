"""Static wiring/privacy smoke assertions; these do NOT replace Go/OTLP tests."""
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
RETRIEVAL = ROOT / "apps/api/internal/modules/ai/application/retrieval"
TELEMETRY = ROOT / "apps/api/internal/platform/telemetry"


class OB04BRAGWiring(unittest.TestCase):
    def test_retrieval_event_locations_and_contract(self):
        code = (RETRIEVAL / "service.go").read_text()
        self.assertIn('telemetry.RAGEvent(ctx, "query"', code)
        self.assertIn('recordCandidates(callCtx, docs)', code)
        self.assertIn('recordRankings(ctx, docs, rankings, fused)', code)
        self.assertIn('recordSelectedContext(ctx, out)', code)
        self.assertIn('if scanErr == nil {\n\t\t\trecordCandidates', code)
        self.assertLess(code.index('allowed, e := s.use.CanSearch'), code.index('telemetry.RAGEvent(ctx, "query"'))
        self.assertLess(code.index('ok, e = s.use.CanUse'), code.index('docs = append(docs, item)'))
        self.assertLess(code.index('if e = ctx.Err(); e != nil'), code.index('recordSelectedContext(ctx, out)'))
        self.assertIn('trace.Capture(ctx, "vector_ranks"', code)
        self.assertIn('trace.Capture(ctx, "contextual_vector_ranks"', code)
        self.assertNotIn("bm25(", code)

    def test_content_capture_is_private_and_bounded(self):
        code = (TELEMETRY / "rag.go").read_text()
        self.assertIn('!ContentEnabled(ctx)', code)
        self.assertIn('!span.IsRecording()', code)
        self.assertIn('MaxWireCaptureBytes', code)
        self.assertIn('redactJSON(data, "")', code)
        self.assertIn('runtime.privateIngestKey', code)
        self.assertIn('theia.dev.rag.omission_reason', code)
        self.assertNotIn('fmt.Println', code)
        self.assertNotIn('log.Print', code)
        self.assertNotIn('otel.SetTracerProvider', code)

    def test_full_rankings_and_final_context_do_not_claim_prompt_inclusion(self):
        code = (RETRIEVAL / "otel.go").read_text()
        self.assertIn('fullRankings(docs, ranking.hits)', code)
        self.assertIn('"branch_ranks": positions[hit.index]', code)
        self.assertIn('"rrf_score": hit.score', code)
        self.assertIn('"items": out.Items', code)
        self.assertIn('NOT a', code)
        self.assertNotIn('item.Vector,', code)
        self.assertIn('embedding_dimension', code)


if __name__ == "__main__":
    unittest.main()
