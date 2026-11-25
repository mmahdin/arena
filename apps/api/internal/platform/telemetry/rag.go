package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// RAGEvent records a structured, full-content RAG observation on the *current*
// stage span. Its names are code-owned; arbitrary input cannot create event
// names. No payload is serialized or exported when full-content OTel capture
// is off. The OTel destination is private/local and synthetic-only.
//
// Even in content mode, unsupported or oversized values are explicitly marked
// omitted. Never write raw payloads to logs, span status or error messages.
func RAGEvent(ctx context.Context, name string, payload any) {
	if !ContentEnabled(ctx) || !ragEventName(name) {
		return
	}
	span := oteltrace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	attrs := make([]attribute.KeyValue, 0, 6)
	// Preserve useful cardinality even if a huge synthetic corpus exceeds
	// the bounded event body. No content is disclosed in metadata mode.
	if value, ok := payload.(map[string]any); ok {
		for _, key := range []string{"count", "ranked_count", "selected_count"} {
			if count, found := value[key].(int); found && count >= 0 {
				attrs = append(attrs, attribute.Int("theia.dev.rag."+key, count))
			}
		}
	}
	data, err := json.Marshal(payload)
	originalBytes := len(data)
	omission := ""
	switch {
	case err != nil:
		omission = "serialization_failed"
	case len(data) > MaxWireCaptureBytes:
		omission = "size_limit"
	default:
		// A private OTLP receiver key is never rendered even if a synthetic
		// fixture accidentally includes it in source text or metadata.
		if runtime, ok := ctx.Value(runtimeKey{}).(*Runtime); ok && runtime.privateIngestKey != "" {
			data = bytes.ReplaceAll(data, []byte(runtime.privateIngestKey), []byte("[REDACTED]"))
		}
		var clean string
		var valid bool
		clean, _, valid = redactJSON(data, "")
		if !valid {
			omission = "invalid_json"
		} else if len(clean) > MaxWireCaptureBytes {
			omission = "redacted_size_limit"
		} else {
			attrs = append(attrs, attribute.String("theia.dev.rag.json", clean))
			semanticRAG(span, name, clean)
		}
	}
	if omission != "" {
		attrs = append(attrs, attribute.Bool("theia.dev.rag.omitted", true), attribute.String("theia.dev.rag.omission_reason", omission))
	} else {
		attrs = append(attrs, attribute.Bool("theia.dev.rag.omitted", false))
	}
	attrs = append(attrs, attribute.Int("theia.dev.rag.original_bytes", originalBytes))
	span.AddEvent("theia.rag."+name, oteltrace.WithAttributes(attrs...))
}

func ragEventName(name string) bool {
	switch strings.ToLower(name) {
	case "query", "authorized_candidates", "vector_ranking", "contextual_vector_ranking",
		"thread_ranking", "temporal_ranking", "bm25_ranking", "rrf_ranking", "selection", "selected_context":
		return name == strings.ToLower(name)
	}
	return false
}
