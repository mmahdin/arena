package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// OBS-04D: OpenInference labels complement (never replace) existing OTel GenAI
// attributes. Names originate in Go, not from request payloads. In particular,
// the HTTP child is a transport span, NOT a second LLM operation.
func semanticStart(name string, fields []attribute.KeyValue) []attribute.KeyValue {
	kind := semanticKind(name, fields)
	if kind == "" {
		return nil
	}
	attrs := []attribute.KeyValue{attribute.String("openinference.span.kind", kind)}
	if name != "gen_ai.client" {
		return attrs
	}
	for _, field := range fields {
		switch string(field.Key) {
		case "gen_ai.request.model":
			key := "llm.model_name"
			if kind == "EMBEDDING" {
				key = "embedding.model_name"
			}
			attrs = append(attrs, attribute.String(key, field.Value.AsString()))
		case "gen_ai.provider.name":
			if kind == "LLM" {
				attrs = append(attrs, attribute.String("llm.provider", field.Value.AsString()))
			}
		}
	}
	return attrs
}

func semanticKind(name string, fields []attribute.KeyValue) string {
	switch name {
	case "theia.ai.http.client":
		return ""
	case "gen_ai.client":
		for _, field := range fields {
			if string(field.Key) == "gen_ai.operation.name" && field.Value.AsString() == "embeddings" {
				return "EMBEDDING"
			}
		}
		return "LLM"
	case "theia.ai.turn", "theia.ai.jev.Select", "theia.ai.unified.Infer":
		return "AGENT"
	case "theia.ai.memory_job", "theia.ai.reflection_job":
		return "CHAIN"
	case "theia.ai.application.retrieval.service.Retrieve", "theia.ai.retrieval.repository.Scan":
		return "RETRIEVER"
	case "theia.ai.retrieval.rank":
		// Theia uses deterministic RRF fusion. This is NOT a learned reranker.
		return "RERANKER"
	case "theia.ai.assembler.Build", "theia.ai.application.assembler.assembler.Build", "theia.ai.application.generator.render.render":
		return "PROMPT"
	case "theia.ai.safety", "theia.ai.validator", "theia.ai.policy_gate":
		return "GUARDRAIL"
	}
	if strings.HasPrefix(name, "theia.ai.application.brain.") ||
		strings.HasPrefix(name, "theia.ai.application.orchestrator.brain.") || name == "theia.ai.brain" {
		return "AGENT"
	}
	if strings.HasPrefix(name, "theia.ai.application.validator.") ||
		strings.HasPrefix(name, "theia.ai.application.safety.") ||
		strings.HasPrefix(name, "theia.ai.safety.") ||
		strings.HasPrefix(name, "theia.ai.validator.") {
		return "GUARDRAIL"
	}
	if strings.HasPrefix(name, "theia.ai.") {
		return "CHAIN"
	}
	return ""
}

// semanticStageIO accepts only the existing, size-checked and credential-
// redacted JSON captured by OBS-04C. Never call this from metadata-only mode.
func semanticStageIO(span oteltrace.Span, direction, cleaned string) {
	if !span.IsRecording() || (direction != "input" && direction != "output") {
		return
	}
	span.SetAttributes(attribute.String(direction+".value", cleaned),
		attribute.String(direction+".mime_type", "application/json"))
}

// semanticRAG follows the same sanitized JSON as the full RAG event. Only
// selected results become retrieval.documents; all rankings stay in OBS-04B
// events. This is not proof that a result entered the final model prompt.
func semanticRAG(span oteltrace.Span, name, cleaned string) {
	if !span.IsRecording() || (name != "query" && name != "selected_context") {
		return
	}
	if name == "query" {
		semanticStageIO(span, "input", cleaned)
		return
	}
	semanticStageIO(span, "output", cleaned)
	var payload struct {
		Items []struct {
			ID    string   `json:"id"`
			Text  string   `json:"text"`
			Score *float64 `json:"score"`
		} `json:"items"`
	}
	if json.Unmarshal([]byte(cleaned), &payload) != nil {
		return
	}
	// Keep the span below its finite attribute count even with large synthetic
	// corpora. The full selected_context event still carries the complete list.
	const maxDocuments = 16
	limit := len(payload.Items)
	if limit > maxDocuments {
		limit = maxDocuments
		span.SetAttributes(attribute.Int("theia.dev.retrieval.semantic_documents_omitted", len(payload.Items)-limit))
	}
	attrs := make([]attribute.KeyValue, 0, limit*3)
	for i := 0; i < limit; i++ {
		item := payload.Items[i]
		if item.ID == "" || item.Text == "" {
			continue
		}
		prefix := fmt.Sprintf("retrieval.documents.%d.document.", i)
		attrs = append(attrs, attribute.String(prefix+"id", item.ID),
			attribute.String(prefix+"content", item.Text))
		if item.Score != nil {
			attrs = append(attrs, attribute.Float64(prefix+"score", *item.Score))
		}
	}
	span.SetAttributes(attrs...)
}

// semanticAttemptKey is server-owned and propagates the actual physical LLM
// attempt through nested provider/HTTP spans. A transport span must not become
// a second generation. Do not use a W3C baggage key for this relationship.
type semanticAttemptKey struct{}

// semanticWire operates only on successfully redacted, bounded physical bytes.
// The parent LLM attempt gets OpenInference input/output; the HTTP child keeps
// the original OBS-04A integrity and completion evidence. Do not claim that a
// rejected, incomplete or omitted response was a complete model generation.
func semanticWire(ctx context.Context, direction, cleaned string, stream, complete bool) {
	attempt, _ := ctx.Value(semanticAttemptKey{}).(oteltrace.Span)
	if attempt == nil || !attempt.IsRecording() {
		return
	}
	if direction == "request" {
		semanticStageIO(attempt, "input", cleaned)
		return
	}
	if direction != "response" || !complete {
		return
	}
	if !stream {
		var body map[string]json.RawMessage
		if json.Unmarshal([]byte(cleaned), &body) != nil || body["error"] != nil {
			return
		}
	}
	mime := "application/json"
	if stream {
		mime = "text/event-stream"
	}
	attempt.SetAttributes(attribute.String("output.value", cleaned), attribute.String("output.mime_type", mime))
}
