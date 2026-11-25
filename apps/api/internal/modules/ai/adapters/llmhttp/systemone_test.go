package llmhttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/llmhttp"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

var jevModel = llm.Model{Provider: "OpenCode", Name: "jev-1.13-free", BillingSource: "free"}

const jevSchema = `{"type":"object","additionalProperties":false,"required":["schema_version","selected_resource_ids","topic_continues"],"properties":{"schema_version":{"const":"jev_selection.v1"},"selected_resource_ids":{"type":"array","maxItems":4,"uniqueItems":true,"items":{"type":"string"}},"topic_continues":{"type":"boolean"}}}`

func jevRequest() ports.StructuredRequest {
	return ports.StructuredRequest{Tier: ports.ComponentJEV, SchemaName: "jev_selection.v1", Schema: []byte(jevSchema), MaxTokens: 500, MaxAttempts: 1, Meta: c.CallMeta{UserID: c.NewID(), TurnID: c.NewID()}, SystemOne: &ports.SystemOneRequest{State: "Synthetic busy week", Questions: map[string]ports.SystemOneQuestion{
		"resource:skill:planning": {Type: "noul", Instructions: "Would planning help?"}, "topic_continues": {Type: "noul", Instructions: "Does the topic continue?"},
	}}}
}

const jevReply = `{"model":"jev-1.13","answers":{"resource:skill:planning":{"type":"noul","noul":0.93},"topic_continues":{"type":"noul","noul":0.8}},"usage":{"input_tokens":120,"output_tokens":12}}`

func jevProvider(t *testing.T, status int, body string) *llmhttp.Provider {
	t.Helper()
	p, e := llmhttp.NewSystemOne("https://synthetic.invalid/v1", "synthetic-jev-secret", &http.Client{Transport: inspectingTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})})
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestSystemOneNativeWireAndSelection(t *testing.T) {
	req := jevRequest()
	for _, id := range []string{"a", "b", "c", "d", "e", "uncertain"} {
		req.SystemOne.Questions["resource:"+id] = ports.SystemOneQuestion{Type: "noul", Instructions: "Is this useful?"}
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/systemone" || r.Header.Get("Authorization") != "Bearer synthetic-jev-secret" {
			t.Error("native endpoint/auth mismatch")
		}
		var wire map[string]json.RawMessage
		if json.NewDecoder(r.Body).Decode(&wire) != nil || len(wire) != 3 || string(wire["model"]) != `"jev-1.13-free"` {
			t.Error("wire must contain only model, state, questions")
		}
		var input ports.SystemOneRequest
		raw, _ := json.Marshal(wire)
		_ = json.Unmarshal(raw, &input)
		if !reflect.DeepEqual(&input, req.SystemOne) {
			t.Error("native input changed")
		}
		answers := map[string]any{}
		for k := range input.Questions {
			p := 0.9
			if k == "resource:uncertain" {
				p = 0.5
			}
			answers[k] = map[string]any{"type": "noul", "noul": p}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"model": "jev-1.13", "answers": answers, "usage": map[string]int{"input_tokens": 120, "output_tokens": 12}})
	}))
	defer server.Close()
	p, e := llmhttp.NewSystemOne(server.URL+"/v1", "synthetic-jev-secret", server.Client())
	if e != nil {
		t.Fatal(e)
	}
	out, e := p.Structured(context.Background(), jevModel, req, llm.ModeSystemOne)
	if e != nil {
		t.Fatal(e)
	}
	var selection struct {
		IDs       []string `json:"selected_resource_ids"`
		Continues bool     `json:"topic_continues"`
	}
	_ = json.Unmarshal(out.JSON, &selection)
	if !reflect.DeepEqual(selection.IDs, []string{"a", "b", "c", "d"}) || !selection.Continues || out.Usage.InTokens != 120 || out.Usage.OutTokens != 12 {
		t.Fatal(selection, out.Usage)
	}
}
func TestSystemOneMalformedAnswersCannotBecomeValidDecisions(t *testing.T) {
	for _, item := range []struct {
		name, body     string
		transportError bool
	}{
		{"missing", strings.Replace(jevReply, `"noul":0.93`, `"missing":0.93`, 1), false},
		{"null", strings.Replace(jevReply, `"noul":0.93`, `"noul":null`, 1), false},
		{"string", strings.Replace(jevReply, `"noul":0.93`, `"noul":"yes"`, 1), false},
		{"range", strings.Replace(jevReply, `"noul":0.93`, `"noul":1.1`, 1), false},
		{"type", strings.Replace(jevReply, `"type":"noul"`, `"type":"choice"`, 1), false},
		{"missing_question", strings.Replace(jevReply, `"resource:skill:planning"`, `"resource:unknown"`, 1), false},
		{"extra_question", strings.Replace(jevReply, `"answers":{`, `"answers":{"unknown":{"type":"noul","noul":1},`, 1), false},
		{"usage_missing", strings.Replace(jevReply, `"input_tokens":120`, `"missing":120`, 1), true},
		{"usage_negative", strings.Replace(jevReply, `"input_tokens":120`, `"input_tokens":-1`, 1), true},
		{"malformed", `{`, true},
	} {
		t.Run(item.name, func(t *testing.T) {
			out, e := jevProvider(t, 200, item.body).Structured(context.Background(), jevModel, jevRequest(), llm.ModeSystemOne)
			if item.transportError {
				if e == nil {
					t.Fatal("malformed envelope accepted")
				}
				return
			}
			if e != nil || string(out.JSON) != "null" || out.Usage.InTokens != 120 {
				t.Fatal("invalid answers must fail validation while retaining usage", out, e)
			}
		})
	}
}
func TestSystemOneNoSelectionAndChangedTopic(t *testing.T) {
	body := strings.ReplaceAll(strings.ReplaceAll(jevReply, "0.93", "0.1"), "0.8", "0.5")
	out, e := jevProvider(t, 200, body).Structured(context.Background(), jevModel, jevRequest(), llm.ModeSystemOne)
	if e != nil || string(out.JSON) != `{"schema_version":"jev_selection.v1","selected_resource_ids":[],"topic_continues":false}` {
		t.Fatal(string(out.JSON), e)
	}
}
func TestSystemOneErrorsCancellationAndBoundaries(t *testing.T) {
	for _, code := range []int{401, 403, 429, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			_, e := jevProvider(t, code, `{"detail":"synthetic-jev-secret"}`).Structured(context.Background(), jevModel, jevRequest(), llm.ModeSystemOne)
			var failure *llm.Failure
			if !errors.As(e, &failure) || failure.HTTPStatus != code || strings.Contains(e.Error(), "secret") {
				t.Fatal(e)
			}
		})
	}
	p := jevProvider(t, 200, jevReply)
	if _, e := p.Stream(context.Background(), jevModel, ports.StreamRequest{}); e == nil {
		t.Fatal("JEV streamed")
	}
	if _, e := p.Embed(context.Background(), jevModel, ports.EmbeddingRequest{}); e == nil {
		t.Fatal("JEV embedded")
	}
	req := jevRequest()
	req.Tier = ports.ComponentUnified
	if _, e := p.Structured(context.Background(), jevModel, req, llm.ModeSystemOne); e == nil {
		t.Fatal("JEV used for Unified")
	}
	generic, _ := llmhttp.New("https://synthetic.invalid/v1", "synthetic-key", nil)
	if _, e := generic.Structured(context.Background(), jevModel, jevRequest(), llm.ModeJSONObject); e == nil {
		t.Fatal("decision silently routed to GPT")
	}
	p, _ = llmhttp.NewSystemOne("https://synthetic.invalid/v1", "synthetic-key", &http.Client{Transport: inspectingTransport(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })})
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	_, e := p.Structured(ctx, jevModel, jevRequest(), llm.ModeSystemOne)
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
}
func TestSystemOneInspectorAndRejectedUsageAccounting(t *testing.T) {
	for _, valid := range []bool{true, false} {
		t.Run(fmt.Sprint(valid), func(t *testing.T) {
			body := jevReply
			if !valid {
				body = strings.Replace(body, `"noul":0.93`, `"noul":null`, 1)
			}
			p := jevProvider(t, 200, body)
			req := jevRequest()
			cfg := llm.Config{Chains: map[ports.ModelTier][]llm.Model{}, DefaultTimeout: time.Second, MaxRetriesPerModel: 1, BreakerWindow: time.Minute, BreakerCooldown: time.Second, BreakerErrorRate: .5, BreakerMinSamples: 2}
			for _, tier := range []ports.ModelTier{ports.ComponentJEV, ports.ComponentUnified, ports.ComponentMemoryWriter, ports.ComponentReflection, ports.ComponentEmbedding} {
				m := llm.Model{Provider: "other", Name: "unused"}
				if tier == ports.ComponentJEV {
					m = jevModel
				}
				if tier == ports.ComponentEmbedding {
					m.Dimensions = 3
				}
				cfg.Chains[tier] = []llm.Model{m}
			}
			ledger := llm.NewMemoryLedger(nil)
			client, e := llm.New(cfg, map[string]llm.Provider{"OpenCode": p, "other": p}, llm.Options{Ledger: ledger})
			if e != nil {
				t.Fatal(e)
			}
			dir := filepath.Join(t.TempDir(), "private")
			ins, e := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "127.0.0.1:8080"})
			if e != nil {
				t.Fatal(e)
			}
			ctx, root := ins.StartTurn(context.Background(), c.TurnRequest{Text: "Synthetic busy week"})
			out, e := client.Structured(ctx, req)
			if (e == nil) != valid || out.Attempts != 1 || out.Usage.InTokens != 120 || out.Usage.CostMicroUSD != 0 {
				t.Fatal(out, e)
			}
			root.End(nil, nil)
			totals := ledger.Snapshot(req.Meta.UserID, time.Now())
			if totals.Calls != 1 || totals.InTokens != 120 || totals.CostMicroUSD != 0 {
				t.Fatal(totals)
			}
			files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
			if len(files) != 1 {
				t.Fatal(files)
			}
			raw, e := os.ReadFile(files[0])
			if e != nil {
				t.Fatal(e)
			}
			if e = trace.ValidateJSON(raw); e != nil {
				t.Fatal(e)
			}
			if strings.Contains(string(raw), "synthetic-jev-secret") || strings.Contains(string(raw), "Bearer") {
				t.Fatal("secret captured")
			}
			var doc trace.Document
			_ = json.Unmarshal(raw, &doc)
			if doc.Totals.InputTokens != 120 || doc.Totals.OutputTokens != 12 || doc.Totals.UnknownUsageCalls != 0 {
				t.Fatal(doc.Totals)
			}
			if valid && os.Getenv("THEIA_UPDATE_JEV_EXAMPLE") == "true" {
				path := filepath.Join("..", "..", "..", "..", "..", "..", "..", "docs", "ai-core", "inspector-examples", "jev_systemone.json")
				if e = os.WriteFile(path, raw, 0644); e != nil {
					t.Fatal(e)
				}
			}
		})
	}
}
