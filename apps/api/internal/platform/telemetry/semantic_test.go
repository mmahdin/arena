package telemetry

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func semanticTestRuntime(t *testing.T, content bool) (context.Context, *tracetest.InMemoryExporter) {
	t.Helper()
	out := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(out), trace.WithRawSpanLimits(spanLimits(content)))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	return WithRuntime(context.Background(), &Runtime{provider: provider, tracer: provider.Tracer("obs04d-test"), content: content}), out
}

func semanticAttrsFromExport(out *tracetest.InMemoryExporter, name string) map[string]string {
	for _, span := range out.GetSpans() {
		if span.Name != name {
			continue
		}
		attrs := map[string]string{}
		for _, item := range span.Attributes {
			attrs[string(item.Key)] = item.Value.Emit()
		}
		return attrs
	}
	return nil
}

func TestOBSDSpanKindsAndTokenMetadata(t *testing.T) {
	ctx, out := semanticTestRuntime(t, false)
	turnCtx, root := StartRoot(ctx, "theia.ai.turn")
	kinds := []struct{ name, kind string }{
		{"theia.ai.application.retrieval.service.Retrieve", "RETRIEVER"},
		{"theia.ai.retrieval.rank", "RERANKER"},
		{"theia.ai.jev.Select", "AGENT"},
		{"theia.ai.unified.Infer", "AGENT"},
		{"theia.ai.application.brain.service.Decide", "AGENT"},
		{"theia.ai.application.generator.render.render", "PROMPT"},
		{"theia.ai.application.generator.service.Generate", "CHAIN"},
		{"theia.ai.validator", "GUARDRAIL"},
		{"theia.ai.validator.validator.Finalize", "GUARDRAIL"},
		{"theia.ai.safety.gate.Assess", "GUARDRAIL"},
		{"theia.ai.application.orchestrator.orchestrator.prepare", "CHAIN"},
	}
	for _, tc := range kinds {
		_, span := Start(turnCtx, tc.name)
		span.End(nil)
	}
	_, llm := Start(turnCtx, "gen_ai.client", String("gen_ai.operation.name", "chat"),
		String("gen_ai.provider.name", "openai"), String("gen_ai.request.model", "gpt-5.6-sol"))
	llm.Set(Int("gen_ai.usage.input_tokens", 78), Int("gen_ai.usage.output_tokens", 18), Int("theia.llm.cost_micro_usd", 42))
	llm.End(nil)
	_, embedding := Start(turnCtx, "gen_ai.client", String("gen_ai.operation.name", "embeddings"),
		String("gen_ai.provider.name", "openai"), String("gen_ai.request.model", "text-embedding-3-small"))
	embedding.Set(Int("gen_ai.usage.input_tokens", 10))
	embedding.End(nil)
	_, http := StartHTTP(turnCtx, "/chat/completions")
	http.End(nil)
	root.End(nil)
	if got := semanticAttrsFromExport(out, "theia.ai.turn")["openinference.span.kind"]; got != "AGENT" {
		t.Fatalf("root kind = %s", got)
	}
	for _, tc := range kinds {
		if got := semanticAttrsFromExport(out, tc.name)["openinference.span.kind"]; got != tc.kind {
			t.Errorf("%s: kind = %s, want %s", tc.name, got, tc.kind)
		}
	}
	spans := out.GetSpans()
	llmCount, embedCount := 0, 0
	for _, span := range spans {
		if span.Name != "gen_ai.client" {
			continue
		}
		attrs := map[string]string{}
		for _, a := range span.Attributes {
			attrs[string(a.Key)] = a.Value.Emit()
		}
		switch attrs["openinference.span.kind"] {
		case "LLM":
			llmCount++
			for key, want := range map[string]string{"llm.model_name": "gpt-5.6-sol", "llm.provider": "openai",
				"llm.token_count.prompt": "78", "llm.token_count.completion": "18", "llm.cost.total": "4.2e-05"} {
				if attrs[key] != want {
					t.Errorf("LLM %s = %q, want %q", key, attrs[key], want)
				}
			}
			cost, err := strconv.ParseFloat(attrs["llm.cost.total"], 64)
			if err != nil || cost != 0.000042 {
				t.Errorf("unexpected estimated LLM cost: %q (%v)", attrs["llm.cost.total"], err)
			}
		case "EMBEDDING":
			embedCount++
			if attrs["embedding.model_name"] != "text-embedding-3-small" || attrs["llm.token_count.prompt"] != "" {
				t.Errorf("embedding wrongly counted as LLM: %v", attrs)
			}
		default:
			t.Errorf("unexpected genai kind: %+v", attrs)
		}
	}
	if llmCount != 1 || embedCount != 1 {
		t.Fatalf("physical attempts mismatch llm=%d embed=%d", llmCount, embedCount)
	}
	if got := semanticAttrsFromExport(out, "theia.ai.http.client")["openinference.span.kind"]; got != "" {
		t.Fatal("HTTP transport mislabeled as AI operation:", got)
	}
}

func TestOBSDStageIOOnlyFromSanitizedContent(t *testing.T) {
	ctx, out := semanticTestRuntime(t, true)
	ctx, root := StartRoot(ctx, "theia.ai.turn")
	stageCtx, stage := Start(ctx, "theia.ai.application.brain.service.Decide")
	StageEvent(stageCtx, "input", map[string]any{"text": "synthetic sample", "api_key": "SENSITIVE_ABC"})
	StageEvent(stageCtx, "output", map[string]any{"reply": "synthetic answer", "authorization": "Bearer PRIVATE123"})
	stage.End(nil)
	root.End(nil)
	attrs := semanticAttrsFromExport(out, "theia.ai.application.brain.service.Decide")
	if attrs["input.mime_type"] != "application/json" || attrs["output.mime_type"] != "application/json" ||
		!strings.Contains(attrs["input.value"], "synthetic sample") || !strings.Contains(attrs["output.value"], "synthetic answer") {
		t.Fatalf("missing standardized stage I/O: %+v", attrs)
	}
	for _, value := range attrs {
		if strings.Contains(value, "SENSITIVE_ABC") || strings.Contains(value, "PRIVATE123") {
			t.Fatal("secret leaked into semantic fields")
		}
	}
}

func TestOBSDRAGSemanticDocumentsUseSanitizedSelectedContext(t *testing.T) {
	ctx, out := semanticTestRuntime(t, true)
	ctx, root := StartRoot(ctx, "theia.ai.turn")
	rCtx, retrieval := Start(ctx, "theia.ai.application.retrieval.service.Retrieve")
	RAGEvent(rCtx, "query", map[string]any{"text": "synthetic question", "api_key": "DONT_EXPORT"})
	RAGEvent(rCtx, "selected_context", map[string]any{"count": 2, "items": []any{
		map[string]any{"id": "synthetic-1", "text": "example memory", "score": 0.75},
		map[string]any{"id": "synthetic-2", "text": "Bearer should-hide", "score": 0.21},
	}})
	retrieval.End(nil)
	root.End(nil)
	attrs := semanticAttrsFromExport(out, "theia.ai.application.retrieval.service.Retrieve")
	for key, want := range map[string]string{
		"openinference.span.kind": "RETRIEVER", "retrieval.documents.0.document.id": "synthetic-1",
		"retrieval.documents.0.document.content": "example memory", "retrieval.documents.0.document.score": "0.75",
		"retrieval.documents.1.document.id": "synthetic-2",
	} {
		if attrs[key] != want {
			t.Errorf("%s: got %q want %q", key, attrs[key], want)
		}
	}
	if strings.Contains(attrs["output.value"], "should-hide") || !strings.Contains(attrs["input.value"], "synthetic question") || strings.Contains(attrs["input.value"], "DONT_EXPORT") {
		t.Fatal("RAG fields not redacted or mapped", attrs)
	}
}

func TestOBSDWireFieldsAttachToRealAttemptNotHTTPOrProviderStage(t *testing.T) {
	ctx, out := semanticTestRuntime(t, true)
	ctx, root := StartRoot(ctx, "theia.ai.turn")
	llmCtx, attempt := Start(ctx, "gen_ai.client", String("gen_ai.operation.name", "chat"))
	providerCtx, providerStage := Start(llmCtx, "theia.ai.provider.Structured")
	httpCtx, httpSpan := StartHTTP(providerCtx, "/chat/completions")
	CaptureRequest(httpCtx, []byte(`{"messages":[{"role":"user","content":"hello synthetic"}],"api_key":"CREDENTIAL"}`), "")
	response := []byte(`{"choices":[{"message":{"content":"hello back"}}]}`)
	CaptureResponse(httpCtx, response, ContentDisposition{Bytes: int64(len(response)), Complete: true}, "")
	httpSpan.End(nil)
	providerStage.End(nil)
	attempt.End(nil)
	root.End(nil)
	attrs := semanticAttrsFromExport(out, "gen_ai.client")
	if !strings.Contains(attrs["input.value"], "hello synthetic") || !strings.Contains(attrs["output.value"], "hello back") || strings.Contains(attrs["input.value"], "CREDENTIAL") {
		t.Fatalf("generation did not receive redacted transport content: %+v", attrs)
	}
	if strings.Contains(semanticAttrsFromExport(out, "theia.ai.http.client")["input.value"], "hello synthetic") {
		t.Fatal("HTTP transport duplicated semantic generation input")
	}
}

func TestOBSDMetadataNeverExportsPayloadAndRejectsIncompleteResponse(t *testing.T) {
	ctx, out := semanticTestRuntime(t, false)
	ctx, root := StartRoot(ctx, "theia.ai.turn")
	llmCtx, attempt := Start(ctx, "gen_ai.client", String("gen_ai.operation.name", "chat"))
	httpCtx, httpSpan := StartHTTP(llmCtx, "/chat/completions")
	CaptureRequest(httpCtx, []byte(`{"messages":[{"content":"CONFIDENTIAL_MARKER"}]}`), "")
	StageEvent(llmCtx, "input", map[string]any{"content": "CONFIDENTIAL_MARKER"})
	httpSpan.End(nil)
	attempt.End(nil)
	root.End(nil)
	attrs := semanticAttrsFromExport(out, "gen_ai.client")
	if attrs["input.value"] != "" || attrs["output.value"] != "" {
		t.Fatal("metadata-only payload leaked")
	}
	ctx, out2 := semanticTestRuntime(t, true)
	ctx, root = StartRoot(ctx, "theia.ai.turn")
	llmCtx, attempt = Start(ctx, "gen_ai.client", String("gen_ai.operation.name", "chat"))
	httpCtx, httpSpan = StartHTTP(llmCtx, "/chat/completions")
	CaptureResponse(httpCtx, []byte(`{"choices":[{"text":"incomplete"}]}`), ContentDisposition{Bytes: 35, Complete: false}, "")
	httpSpan.End(nil)
	attempt.End(nil)
	root.End(nil)
	if value := semanticAttrsFromExport(out2, "gen_ai.client")["output.value"]; value != "" {
		t.Fatalf("incomplete response marked as model output: %s", value)
	}
}
