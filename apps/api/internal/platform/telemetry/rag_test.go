package telemetry

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type observedRAGEvent struct {
	name  string
	attrs map[string]string
}

func observedRAG(content bool, secret string, action func(context.Context)) []observedRAGEvent {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithRawSpanLimits(spanLimits(content)))
	runtime := &Runtime{provider: provider, tracer: provider.Tracer("test"), content: content, privateIngestKey: secret}
	ctx, root := StartRoot(WithRuntime(context.Background(), runtime), "theia.ai.turn")
	action(ctx)
	root.End(nil)
	var output []observedRAGEvent
	for _, span := range exporter.GetSpans() {
		for _, event := range span.Events {
			values := map[string]string{}
			for _, entry := range event.Attributes {
				values[string(entry.Key)] = entry.Value.Emit()
			}
			output = append(output, observedRAGEvent{event.Name, values})
		}
	}
	_ = provider.Shutdown(context.Background())
	return output
}

type dangerousMarshal struct{ called *bool }

func (d dangerousMarshal) MarshalJSON() ([]byte, error) {
	*d.called = true
	return []byte(`{"text":"PRIVATE_NEVER_EXPORT"}`), nil
}

func TestRAGContentIsOffByDefaultAndDoesNotSerialize(t *testing.T) {
	called := false
	events := observedRAG(false, "", func(ctx context.Context) {
		RAGEvent(ctx, "query", dangerousMarshal{&called})
	})
	if called || len(events) != 0 {
		t.Fatal("metadata-only mode serialized RAG content", called, events)
	}
}

func TestRAGExportsActiveRankingBranches(t *testing.T) {
	branches := []string{"vector_ranking", "contextual_vector_ranking", "thread_ranking", "temporal_ranking"}
	events := observedRAG(true, "", func(ctx context.Context) {
		for _, branch := range branches {
			RAGEvent(ctx, branch, map[string]any{"count": 1})
		}
	})
	if len(events) != len(branches) {
		t.Fatalf("exported %d of %d ranking events", len(events), len(branches))
	}
	for i, event := range events {
		if event.name != "theia.rag."+branches[i] || event.attrs["theia.dev.rag.count"] != "1" {
			t.Errorf("unexpected ranking event: %+v", event)
		}
	}
}

func TestRAGDetailedEventsRetainStructuredJSONAndRedactSecrets(t *testing.T) {
	const privateKey = "private-test-ingestion-key-42characters"
	events := observedRAG(true, privateKey, func(ctx context.Context) {
		RAGEvent(ctx, "authorized_candidates", map[string]any{
			"count": 2,
			"items": []any{
				map[string]any{"text": "synthetic memory A", "id": "first", "authorization": "Bearer hidden-credential", "nested": map[string]any{"api_key": "SECRET_KEY"}},
				map[string]any{"text": "synthetic memory B", "id": "second", "echo": privateKey},
			},
		})
		RAGEvent(ctx, "rrf_ranking", map[string]any{"k": 60, "method": "rrf", "items": []any{map[string]any{"id": "first", "rrf_score": 0.031}}})
		RAGEvent(ctx, "selection", map[string]any{"decisions": []any{map[string]any{"id": "first", "selected": true}, map[string]any{"id": "second", "reason": "duplicate_text", "selected": false}}})
	})
	if len(events) != 3 {
		t.Fatalf("expected 3 RAG events, got %d", len(events))
	}
	if events[0].name != "theia.rag.authorized_candidates" || events[1].name != "theia.rag.rrf_ranking" || events[2].name != "theia.rag.selection" {
		t.Fatalf("unexpected RAG event names: %+v", events)
	}
	content := events[0].attrs["theia.dev.rag.json"]
	if !strings.Contains(content, "synthetic memory A") || !strings.Contains(content, "synthetic memory B") {
		t.Fatalf("synthetic source text missing from full trace: %s", content)
	}
	for _, forbidden := range []string{"hidden-credential", "SECRET_KEY", privateKey} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("secret leaked into RAG content: %s", forbidden)
		}
	}
	if events[0].attrs["theia.dev.rag.omitted"] != "false" || !strings.Contains(events[2].attrs["theia.dev.rag.json"], "duplicate_text") {
		t.Fatalf("omission flag or selection details missing: %+v", events)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(content), &body); err != nil || body["count"].(float64) != 2 {
		t.Fatalf("invalid structured candidate payload: %v", err)
	}
}

func TestRAGPayloadFailureIsMarkedExplicitly(t *testing.T) {
	events := observedRAG(true, "", func(ctx context.Context) {
		RAGEvent(ctx, "query", map[string]any{"text": strings.Repeat("synthetic ", MaxWireCaptureBytes/10)})
		RAGEvent(ctx, "rrf_ranking", map[string]any{"bad_float": math.NaN()})
		RAGEvent(ctx, "unknown_by_user", map[string]any{"text": "must not log"})
	})
	if len(events) != 2 {
		t.Fatalf("unexpected event count: %d", len(events))
	}
	if events[0].attrs["theia.dev.rag.omitted"] != "true" || events[0].attrs["theia.dev.rag.omission_reason"] != "size_limit" {
		t.Fatalf("oversized RAG content leaked or was silently truncated: %+v", events[0])
	}
	if events[1].attrs["theia.dev.rag.omission_reason"] != "serialization_failed" {
		t.Fatalf("unsupported RAG payload not marked omitted: %+v", events[1])
	}
	for _, event := range events {
		if event.attrs["theia.dev.rag.json"] != "" {
			t.Fatal("unsafe RAG event exported payload")
		}
	}
}

func TestRAGAttributesStayDistinctFromMetadataAllowlist(t *testing.T) {
	if _, ok := allowed(String("theia.dev.rag.json", "PRIVATE")); ok {
		t.Fatal("RAG payload was accidentally added to general metadata allowlist")
	}
	if spanLimits(false).AttributeValueLengthLimit != -1 {
		t.Fatal("OBS-04A span config regressed")
	}
}
