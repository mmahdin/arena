package llm_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

func TestInspectorCountsActualAttemptsRepairEmbeddingAndStream(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	ins, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, root := ins.StartTurn(context.Background(), contract.TurnRequest{Text: "synthetic model operations"})
	p := fake.New()
	p.QueueStructured("small-primary", fake.StructuredReply{Err: &llm.Failure{Kind: llm.FailureServer}}, fake.StructuredReply{Err: &llm.Failure{Kind: llm.FailureServer}})
	p.QueueStructured("small-secondary", fake.StructuredReply{JSON: []byte(`{"ok":"invalid"}`), Usage: contract.LLMUsage{InTokens: 7, OutTokens: 2}}, fake.StructuredReply{JSON: []byte(`{"ok":true}`), Usage: contract.LLMUsage{InTokens: 8, OutTokens: 3}})
	p.QueueEmbedding("embedding", fake.EmbeddingReply{Vector: []float32{1, 0, 0}, Usage: contract.LLMUsage{InTokens: 4}})
	p.QueueStream("strong-primary", fake.StreamReply{FailAt: -1, Chunks: []ports.Chunk{{Text: "unredacted synthetic streamed answer"}, {FinishReason: "stop"}, {Usage: &contract.LLMUsage{InTokens: 20, OutTokens: 5}}}})
	c := client(t, config(), p, llm.Options{})
	req := request()
	out, err := c.Structured(ctx, req)
	if err != nil || out.Attempts != 4 {
		t.Fatal(out, err)
	}
	if _, err = c.Embed(ctx, ports.EmbeddingRequest{Tier: ports.TierEmbedding, Input: "synthetic embedding", Meta: req.Meta}); err != nil {
		t.Fatal(err)
	}
	stream, err := c.Stream(ctx, ports.StreamRequest{Tier: ports.TierStrong, MaxTokens: 100, Meta: req.Meta})
	if err != nil {
		t.Fatal(err)
	}
	for {
		_, err = stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	_ = stream.Close()
	root.End(nil, nil)
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(files) != 1 {
		t.Fatal(files)
	}
	raw, _ := os.ReadFile(files[0])
	if err = trace.ValidateJSON(raw); err != nil {
		t.Fatal(err)
	}
	var d trace.Document
	_ = json.Unmarshal(raw, &d)
	if d.Totals.ProviderCalls != len(p.History()) || d.Totals.ProviderCalls != 6 || d.Totals.ReportedCalls != 4 || d.Totals.UnknownUsageCalls != 2 || d.Totals.InputTokens != 39 || d.Totals.OutputTokens != 10 {
		t.Fatalf("usage double counted or dropped: %+v", d.Totals)
	}
	if !d.Complete {
		t.Fatal("stream left open inspection nodes")
	}
	stageCalls := 0
	for _, stage := range d.Totals.ByStage {
		stageCalls += stage.Calls
	}
	if stageCalls != len(p.History()) {
		t.Fatal("stage totals omitted unsuccessful attempts", stageCalls)
	}

	var repaired bool
	for _, node := range d.Nodes {
		if node.Operation == "provider.Structured" && string(node.Output) != "null" {
			var input struct {
				Request ports.StructuredRequest `json:"request"`
			}
			_ = json.Unmarshal(node.Input, &input)
			if len(input.Request.Messages) > 1 {
				repaired = true
			}
		}
	}
	if os.Getenv("THEIA_UPDATE_INSPECTOR_EXAMPLES") == "true" {
		path := filepath.Join("..", "..", "..", "..", "..", "..", "docs", "ai-core", "inspector-examples", "provider_retry_repair_stream.json")
		if err := os.WriteFile(path, raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if !repaired {
		t.Fatal("repair request missing")
	}
}
