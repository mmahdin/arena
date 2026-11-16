package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application"
)

func newTestGateway(t *testing.T, handler http.HandlerFunc) (*Gateway, func()) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	gateway := New(server.URL+"/v1", "test-only-key", "test-model", 2*time.Second, 700)
	gateway.client.Transport = server.Client().Transport
	return gateway, server.Close
}
func TestChatCompletionsRequestAndResponse(t *testing.T) {
	g, closeServer := newTestGateway(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("wrong route %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-only-key" {
			t.Error("missing backend auth header")
		}
		var body struct {
			Model     string                    `json:"model"`
			MaxTokens int                       `json:"max_tokens"`
			Messages  []application.ChatMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("invalid body: %v", err)
		}
		if body.Model != "test-model" || body.MaxTokens != 700 || len(body.Messages) != 2 || body.Messages[1].Content != "synthetic" {
			t.Errorf("invalid request: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"finish_reason":"stop","message":{"content":"synthetic reply"}}]}`)
	})
	defer closeServer()
	reply, err := g.Generate(context.Background(), []application.ChatMessage{{Role: "system", Content: "test"}, {Role: "user", Content: "synthetic"}})
	if err != nil || reply != "synthetic reply" {
		t.Fatalf("reply=%q err=%v", reply, err)
	}
}

func TestGPT56UsesCompletionBudgetWithoutUnsupportedSampling(t *testing.T) {
	g, closeServer := newTestGateway(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal("invalid JSON")
		}
		if body["model"] != "gpt-5.6-sol" || body["max_completion_tokens"] != float64(700) || body["reasoning_effort"] != "none" || body["max_tokens"] != nil || body["temperature"] != nil {
			t.Error("GPT-5.6 request used incompatible parameters")
		}
		io.WriteString(w, `{"choices":[{"finish_reason":"stop","message":{"content":"synthetic reply"}}]}`)
	})
	defer closeServer()
	g.model = "gpt-5.6-sol"
	if _, err := g.Generate(context.Background(), []application.ChatMessage{{Role: "user", Content: "synthetic"}}); err != nil {
		t.Fatal(err)
	}
}
func TestUpstreamFailuresDoNotLeakResponse(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", 401, "provider secret: token-private"},
		{"rate limit", 429, "model quota exhausted"},
		{"invalid json", 200, "{not-json"},
		{"empty", 200, `{"choices":[]}`},
		{"truncated", 200, `{"choices":[{"finish_reason":"length","message":{"content":"partial"}}]}`},
		{"filtered", 200, `{"choices":[{"finish_reason":"content_filter","message":{"content":"partial"}}]}`},
		{"too large", 200, strings.Repeat("x", maxResponseBytes+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, closeServer := newTestGateway(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status); io.WriteString(w, tc.body) })
			defer closeServer()
			_, err := g.Generate(context.Background(), nil)
			if err == nil || strings.Contains(err.Error(), "token-private") || strings.Contains(err.Error(), "quota") {
				t.Fatalf("unsafe error: %v", err)
			}
		})
	}
}
func TestRedirectIsNotFollowedWithAuthorization(t *testing.T) {
	var redirects atomic.Int32
	g, closeServer := newTestGateway(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			http.Redirect(w, r, "/redirected", http.StatusFound)
			return
		}
		redirects.Add(1)
	})
	defer closeServer()
	_, err := g.Generate(context.Background(), nil)
	if err == nil || redirects.Load() != 0 {
		t.Fatalf("redirect was followed: err=%v redirected=%d", err, redirects.Load())
	}
}
func TestDeadlineReturnsSanitizedFailure(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	g, closeServer := newTestGateway(t, func(w http.ResponseWriter, r *http.Request) {
		// Consume the POST body so HTTP/1 can observe the client disconnect.
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	defer closeServer()
	// Ensure test cleanup cannot deadlock even if a transport never propagates
	// cancellation to the server handler. Release it before server.Close.
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	_, err := g.Generate(ctx, nil)
	if err == nil || err.Error() != errProvider.Error() {
		t.Fatalf("deadline error=%v", err)
	}
	if ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("request failed before its deadline: %v", ctx.Err())
	}
	select {
	case <-started:
	default:
		t.Fatal("deadline test did not reach the provider handler")
	}
}
