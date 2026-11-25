package llmhttp_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func TestInvalidCompletionReasonsAreDistinctAndSafe(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"length", `{"choices":[{"finish_reason":"length","message":{"content":"{\"partial\":true}"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`, "finish_reason_length"},
		{"empty_content", `{"choices":[{"finish_reason":"stop","message":{"content":""}}],"usage":{"prompt_tokens":10}}`, "empty_completion_content"},
		{"missing_usage", `{"choices":[{"finish_reason":"stop","message":{"content":"{}"}}]}`, "missing_provider_usage"},
		{"no_choices", `{"choices":[],"usage":{"prompt_tokens":1}}`, "unexpected_choice_count"},
		{"malformed_json", `{"choices":[`, "malformed_provider_json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, cleanup := newServer(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, tc.body) })
			defer cleanup()
			_, err := p.Structured(context.Background(), llm.Model{Name: "test"}, ports.StructuredRequest{Schema: []byte(`{"type":"object"}`), MaxTokens: 100}, llm.ModeJSONObject)
			var failure *llm.Failure
			if !errors.As(err, &failure) || failure.DiagnosticReason() != tc.want {
				t.Fatalf("got %v want %s", err, tc.want)
			}
			if strings.Contains(err.Error(), "partial") {
				t.Fatal("raw content leaked in public failure")
			}
		})
	}
}
