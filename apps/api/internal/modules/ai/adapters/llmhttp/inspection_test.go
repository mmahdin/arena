package llmhttp_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/llmhttp"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

type inspectingTransport func(*http.Request) (*http.Response, error)

func (f inspectingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestInspectorRecordsExactWirePayloadAndRawBodiesWithoutCredentials(t *testing.T) {
	for _, mode := range []llm.Mode{llm.ModeNativeSchema, llm.ModeJSONObject} {
		t.Run(string(mode), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "private")
			ins, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "127.0.0.1:8080"})
			if err != nil {
				t.Fatal(err)
			}
			ctx, root := ins.StartTurn(context.Background(), c.TurnRequest{Text: "full synthetic incoming message"})
			const secret = "synthetic-provider-authentication-key"
			var sent string
			const reply = `{"choices":[{"finish_reason":"stop","message":{"content":"{\"ok\":true}"}}],"usage":{"prompt_tokens":30,"completion_tokens":10}}`
			transport := inspectingTransport(func(r *http.Request) (*http.Response, error) {
				raw, _ := io.ReadAll(r.Body)
				sent = string(raw)
				if r.Header.Get("Authorization") != "Bearer "+secret {
					t.Error("authorization changed")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(reply))}, nil
			})
			p, err := llmhttp.New("https://synthetic.invalid/v1", secret, &http.Client{Transport: transport})
			if err != nil {
				t.Fatal(err)
			}
			request := ports.StructuredRequest{SchemaName: "brain.v1", Schema: []byte(`{"type":"object"}`), System: []ports.Block{{Name: "policy", Text: "full synthetic policy"}}, Messages: []ports.ChatMessage{{Role: c.RoleUser, Content: "full user data"}}, Temperature: .7, MaxTokens: 123}
			model := llm.Model{Provider: "fixture", Name: "gpt-5.6-sol"}
			_, err = trace.Call(ctx, "provider.Structured", func() any { return map[string]any{"request": request, "model": model, "mode": mode} }, func(callCtx context.Context) (llm.StructuredResult, error) {
				response, callErr := p.Structured(callCtx, model, request, mode)
				if callErr == nil {
					usage := response.Usage
					usage.Provider = model.Provider
					usage.Model = model.Name
					trace.Usage(callCtx, usage, nil)
				}
				return response, callErr
			})
			if err != nil {
				t.Fatal(err)
			}
			root.End(nil, nil)
			files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
			raw, _ := os.ReadFile(files[0])
			if err = trace.ValidateJSON(raw); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), secret) || strings.Contains(string(raw), "Bearer") {
				t.Fatal("credentials captured")
			}
			var doc trace.Document
			_ = json.Unmarshal(raw, &doc)
			matched := false
			body := false
			for _, node := range doc.Nodes {
				if node.Operation == "http.request" {
					var in struct {
						Body string `json:"body"`
					}
					_ = json.Unmarshal(node.Input, &in)
					matched = in.Body == sent
					for _, a := range node.Artifacts {
						if a.Name == "raw_response_body" {
							var value string
							_ = json.Unmarshal(a.Value, &value)
							body = value == reply
						}
					}
				}
			}
			if mode == llm.ModeJSONObject && os.Getenv("THEIA_UPDATE_INSPECTOR_EXAMPLES") == "true" {
				path := filepath.Join("..", "..", "..", "..", "..", "..", "..", "docs", "ai-core", "inspector-examples", "exact_wire.json")
				if err := os.WriteFile(path, raw, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if !matched || !body {
				t.Fatal("trace differs from actual HTTP bytes")
			}
		})
	}
}
func TestInspectorHTTPRejectionRetainsVendorBodyAndOmitsEchoedKey(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	ins, _ := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	ctx, root := ins.StartTurn(context.Background(), c.TurnRequest{})
	p, _ := llmhttp.New("https://synthetic.invalid/v1", "fixture-auth-key", &http.Client{Transport: inspectingTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"synthetic rate limit","echo":"fixture-auth-key"}`))}, nil
	})})
	_, err := p.Embed(ctx, llm.Model{Name: "embedding", Dimensions: 3}, ports.EmbeddingRequest{Input: "full synthetic embedding query"})
	if err == nil {
		t.Fatal("provider failure changed")
	}
	root.End(nil, err)
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	raw, _ := os.ReadFile(files[0])
	if err = trace.ValidateJSON(raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "fixture-auth-key") || !strings.Contains(string(raw), "synthetic rate limit") {
		t.Fatal("rejected response capture incorrect")
	}
}

func TestInspectorKeepsReportedUsageOnRejectedCompletion(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	ins, _ := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	ctx, root := ins.StartTurn(context.Background(), c.TurnRequest{})
	p, _ := llmhttp.New("https://synthetic.invalid/v1", "synthetic-auth-key", &http.Client{Transport: inspectingTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"choices":[{"finish_reason":"length","message":{"content":"{\"partial\":true}"}}],"usage":{"prompt_tokens":17,"completion_tokens":4}}`))}, nil
	})})
	out, err := trace.Call(ctx, "provider.Structured", nil, func(callCtx context.Context) (llm.StructuredResult, error) {
		return p.Structured(callCtx, llm.Model{Provider: "fixture", Name: "small"}, ports.StructuredRequest{SchemaName: "brain.v1", Schema: []byte(`{"type":"object"}`), MaxTokens: 40}, llm.ModeNativeSchema)
	})
	if err == nil || len(out.JSON) != 0 {
		t.Fatal("invalid completion must remain rejected")
	}
	root.End(nil, err)
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	raw, _ := os.ReadFile(files[0])
	if err = trace.ValidateJSON(raw); err != nil {
		t.Fatal(err)
	}
	var d trace.Document
	_ = json.Unmarshal(raw, &d)
	if d.Totals.InputTokens != 17 || d.Totals.OutputTokens != 4 || d.Totals.UnknownUsageCalls != 0 {
		t.Fatal(d.Totals)
	}
}
