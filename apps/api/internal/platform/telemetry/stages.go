package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type stageSecretsKey struct{}
type stageSecrets struct {
	mu     sync.Mutex
	values []string
}

// RegisterStageSecret redacts a credential from all subsequent stage events
// under this turn or job root, including events emitted by ancestor stages.
func RegisterStageSecret(ctx context.Context, value string) {
	if !ContentEnabled(ctx) || value == "" {
		return
	}
	if secrets, _ := ctx.Value(stageSecretsKey{}).(*stageSecrets); secrets != nil {
		secrets.mu.Lock()
		defer secrets.mu.Unlock()
		for _, existing := range secrets.values {
			if existing == value {
				return
			}
		}
		secrets.values = append(secrets.values, value)
	}
}

// StageEvent exports an existing code-owned AI stage's input, output, or one
// allowlisted diagnostic artifact. It never invokes the underlying AI call,
// performs retrieval, or writes application state. Full content is confined to
// an explicitly validated local/synthetic runtime. In metadata mode the value
// is never marshaled, even when its MarshalJSON has side effects.
func StageEvent(ctx context.Context, direction string, value any) {
	if !ContentEnabled(ctx) || !stageDirection(direction) {
		return
	}
	span := oteltrace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	// An observability marshaler must not interrupt a working AI turn.
	// Surface the failure as a fixed diagnostic, not a leaked panic/error body.
	defer func() {
		if recover() != nil {
			span.AddEvent("theia.ai.stage."+direction,
				oteltrace.WithAttributes(attribute.Bool("theia.dev.stage.omitted", true),
					attribute.String("theia.dev.stage.omission_reason", "serialization_panic")))
		}
	}()

	attrs := []attribute.KeyValue{}
	data, err := json.Marshal(value)
	originalBytes := len(data)
	omission := ""
	switch {
	case err != nil:
		omission = "serialization_failed"
	case len(data) > MaxWireCaptureBytes:
		omission = "size_limit"
	default:
		if runtime, ok := ctx.Value(runtimeKey{}).(*Runtime); ok && runtime.privateIngestKey != "" {
			data = bytes.ReplaceAll(data, []byte(runtime.privateIngestKey), []byte("[REDACTED]"))
		}
		clean, _, valid := redactJSON(data, "")
		if secrets, _ := ctx.Value(stageSecretsKey{}).(*stageSecrets); secrets != nil && valid {
			secrets.mu.Lock()
			values := append([]string(nil), secrets.values...)
			secrets.mu.Unlock()
			for _, secret := range values {
				clean, _, valid = redactJSON([]byte(clean), secret)
				if !valid {
					break
				}
			}
		}
		if !valid {
			omission = "invalid_json"
		} else if len(clean) > MaxWireCaptureBytes {
			omission = "redacted_size_limit"
		} else {
			attrs = append(attrs, attribute.String("theia.dev.stage.json", clean))
			semanticStageIO(span, direction, clean)
		}
	}
	attrs = append(attrs, attribute.Int("theia.dev.stage.original_bytes", originalBytes),
		attribute.Bool("theia.dev.stage.omitted", omission != ""))
	if omission != "" {
		attrs = append(attrs, attribute.String("theia.dev.stage.omission_reason", omission))
	}
	span.AddEvent("theia.ai.stage."+direction, oteltrace.WithAttributes(attrs...))
}

func stageDirection(direction string) bool {
	switch direction {
	case "input", "output", "artifact":
		return true
	}
	return false
}

// StageArtifact is reserved for already-recorded, code-owned Inspector artifacts.
// Streaming token/draft snapshots are deliberately excluded: the final stage
// output and physical LLM wire response already capture the complete text, and
// emitting a growing draft for every token would exhaust the span event limit.
func StageArtifact(ctx context.Context, name, kind string, value any) {
	switch name {
	case "rendered_prompt", "prompt_composition", "before_policy_gate", "after_policy_gate",
		"attempt_result", "retry_delay", "router_decision", "route_override",
		"fallback_selection", "completion_validation", "rejected_completion_content",
		"classifier_template", "persona_template", "crisis_template", "communication_policy",
		"risk_transition", "state_proposal", "policy_gate", "budgets":
	default:
		return
	}
	StageEvent(ctx, "artifact", map[string]any{"name": name, "kind": kind, "value": value})
}
