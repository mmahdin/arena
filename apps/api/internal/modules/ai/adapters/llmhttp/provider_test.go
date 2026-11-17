package llmhttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/llmhttp"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func newServer(t *testing.T, handler http.HandlerFunc) (*llmhttp.Provider, func()) {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	p, err := llmhttp.New(srv.URL+"/v1", "synthetic-secret", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	return p, srv.Close
}
func TestStructuredNativeAndJSONModeRequests(t *testing.T) {
	var modes []string
	p, stop := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-secret" || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("wrong request metadata")
		}
		var payload struct {
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
			MaxCompletionTokens int      `json:"max_completion_tokens"`
			Temperature         *float32 `json:"temperature"`
			Messages            []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		modes = append(modes, payload.ResponseFormat.Type)
		if payload.ResponseFormat.Type == "json_object" && (len(payload.Messages) == 0 ||
			!strings.Contains(payload.Messages[0].Content, `{"type":"object"}`)) {
			t.Error("JSON mode did not receive the pinned schema")
		}
		if payload.MaxCompletionTokens != 150 || payload.Temperature != nil {
			t.Errorf("reasoning model args are incompatible: %+v", payload)
		}
		_, _ = io.WriteString(w, `{"choices":[{"finish_reason":"stop","message":{"content":"{\"ok\":true}"}}],"usage":{"prompt_tokens":30,"completion_tokens":10,"prompt_tokens_details":{"cached_tokens":5}}}`)
	})
	defer stop()
	model := llm.Model{Name: "gpt-5.6-sol"}
	req := ports.StructuredRequest{SchemaName: "brain.v1", Schema: []byte(`{"type":"object"}`), MaxTokens: 150}
	for _, mode := range []llm.Mode{llm.ModeNativeSchema, llm.ModeJSONObject} {
		response, err := p.Structured(context.Background(), model, req, mode)
		if err != nil || string(response.JSON) != `{"ok":true}` || response.Usage.CachedTokens != 5 {
			t.Fatalf("%s: %+v %v", mode, response, err)
		}
	}
	if len(modes) != 2 || modes[0] != "json_schema" || modes[1] != "json_object" {
		t.Fatal(modes)
	}
}
func TestStreamingSSEUsageAndTruncation(t *testing.T) {
	p, stop := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"first\"},\"finish_reason\":null}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":1}}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	})
	defer stop()
	s, err := p.Stream(context.Background(), llm.Model{Name: "synthetic"}, ports.StreamRequest{MaxTokens: 50})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var got string
	var hasUsage bool
	for {
		chunk, err := s.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		got += chunk.Text
		if chunk.Usage != nil && chunk.Usage.InTokens == 10 {
			hasUsage = true
		}
	}
	if got != "first" || !hasUsage {
		t.Fatalf("text=%q usage=%v", got, hasUsage)
	}
	p, stop = newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
	})
	defer stop()
	s, err = p.Stream(context.Background(), llm.Model{Name: "synthetic"}, ports.StreamRequest{MaxTokens: 50})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Recv(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Recv(); err == nil || errors.Is(err, io.EOF) {
		t.Fatalf("truncated stream must not be complete: %v", err)
	}
	_ = s.Close()
}
func TestEmbeddingsAndRetryClassification(t *testing.T) {
	p, stop := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/embeddings") {
			t.Error("wrong embedding endpoint")
		}
		_, _ = io.WriteString(w, `{"data":[{"index":0,"embedding":[0,0.6,0.8]}],"usage":{"prompt_tokens":8}}`)
	})
	defer stop()
	res, err := p.Embed(context.Background(), llm.Model{Name: "synthetic", Dimensions: 3}, ports.EmbeddingRequest{Input: "test"})
	if err != nil || len(res.Vector) != 3 || res.Usage.InTokens != 8 {
		t.Fatalf("%+v %v", res, err)
	}
	for _, test := range []struct {
		status int
		kind   llm.FailureKind
	}{{429, llm.FailureRateLimit}, {503, llm.FailureServer}, {401, llm.FailurePermanent}, {302, llm.FailurePermanent}} {
		p, stop := newServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(test.status)
			_, _ = io.WriteString(w, "SENSITIVE BODY SHOULD NOT BE IN ERROR")
		})
		_, err := p.Structured(context.Background(), llm.Model{Name: "synthetic"}, ports.StructuredRequest{MaxTokens: 20, Schema: []byte(`{"type":"object"}`)}, llm.ModeJSONObject)
		var failure *llm.Failure
		if !errors.As(err, &failure) || failure.Kind != test.kind || strings.Contains(err.Error(), "SENSITIVE") {
			t.Fatalf("status=%d err=%v", test.status, err)
		}
		stop()
	}
	if _, err := llmhttp.New("http://insecure.example/v1", "secret", nil); !errors.Is(err, llm.ErrConfig) {
		t.Fatalf("must reject http: %v", err)
	}
}

var _ = contract.LLMUsage{}
