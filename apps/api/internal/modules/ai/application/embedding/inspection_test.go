package embedding

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

type blockedProvider struct {
	*fake.Provider
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (p *blockedProvider) Embed(ctx context.Context, m llm.Model, r ports.EmbeddingRequest) (llm.EmbeddingResult, error) {
	p.once.Do(func() { close(p.entered) })
	select {
	case <-p.release:
		return p.Provider.Embed(ctx, m, r)
	case <-ctx.Done():
		return llm.EmbeddingResult{}, ctx.Err()
	}
}
func documents(t *testing.T, dir string) []trace.Document {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	var docs []trace.Document
	for _, file := range files {
		raw, e := os.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		if e = trace.ValidateJSON(raw); e != nil {
			t.Fatal(e)
		}
		var doc trace.Document
		if e = json.Unmarshal(raw, &doc); e != nil {
			t.Fatal(e)
		}
		docs = append(docs, doc)
	}
	return docs
}

func TestInspectorRetainsOneLateComputationAndCrossTurnReuse(t *testing.T) {
	provider := &blockedProvider{Provider: fake.New(), entered: make(chan struct{}), release: make(chan struct{})}
	provider.QueueEmbedding("text-embedding-3-small", fake.EmbeddingReply{Vector: func() []float32 { v := make([]float32, 1024); v[0] = 1; return v }(), Usage: c.LLMUsage{InTokens: 4}})
	cfg := llm.Config{Chains: map[ports.ModelTier][]llm.Model{ports.ComponentUnified: {{Provider: "synthetic", Name: "unused"}}, ports.ComponentMemoryWriter: {{Provider: "synthetic", Name: "unused"}}, ports.ComponentReflection: {{Provider: "synthetic", Name: "unused"}}, ports.ComponentEmbedding: {{Provider: "synthetic", Name: "text-embedding-3-small", Dimensions: 1024, InputMicroUSDPerM: 1000000}}}, DefaultTimeout: time.Second, BreakerWindow: time.Minute, BreakerCooldown: time.Minute, BreakerErrorRate: .5, BreakerMinSamples: 5}
	client, e := llm.New(cfg, map[string]llm.Provider{"synthetic": provider}, llm.Options{})
	if e != nil {
		t.Fatal(e)
	}
	shared := service(t, client, Options{})
	dir := filepath.Join(t.TempDir(), "private")
	ins, e := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if e != nil {
		t.Fatal(e)
	}
	req := request()
	turn := req.Meta.TurnID
	rootCtx, root := ins.StartTurn(context.Background(), c.TurnRequest{UserID: req.Meta.UserID, ClientTurnID: c.NewID(), Text: "سلام"})
	trace.Associate(rootCtx, turn)
	caller, cancel := context.WithCancel(rootCtx)
	first := make(chan error, 1)
	go func() { _, e := shared.Embed(caller, req); first <- e }()
	<-provider.entered
	jobCtx, job := trace.StartJob(trace.WithInspector(context.Background(), ins), turn, c.NewID(), "memory", 1)
	req.Consumer = "memory_worker"
	second := make(chan error, 1)
	go func() {
		r, e := shared.Embed(jobCtx, req)
		if e == nil && (len(r.Vector) != 1024 || r.Usage.InTokens != 0) {
			e = ErrInvalid
		}
		second <- e
	}()
	waitFor(t, func() bool {
		for _, doc := range documents(t, dir) {
			for _, n := range doc.Nodes {
				if n.Operation == "embedding.inflight_join" {
					return true
				}
			}
		}
		return false
	})
	cancel()
	<-first
	root.End(func() any { return "Synthetic reply, independent of pending embedding." }, nil)
	if d := documents(t, dir); len(d) != 1 || d[0].Complete || !d[0].ResponseComplete {
		t.Fatal("late shared work lost its Turn", d)
	}
	close(provider.release)
	if e = <-second; e != nil {
		t.Fatal(e)
	}
	job.End(nil, nil)
	// Same text, owner and configuration in another Turn reuses metadata only.
	nextTurn := c.NewID()
	nextCtx, nextRoot := ins.StartTurn(context.Background(), c.TurnRequest{UserID: req.Meta.UserID, ClientTurnID: c.NewID(), Text: "سلام"})
	trace.Associate(nextCtx, nextTurn)
	req.Meta.TurnID = nextTurn
	req.Consumer = "retrieval.current"
	if _, e = shared.Embed(nextCtx, req); e != nil {
		t.Fatal(e)
	}
	nextRoot.End(nil, nil)
	docs := documents(t, dir)
	if len(docs) != 2 {
		t.Fatal("Turns merged or split", len(docs))
	}
	if len(provider.History()) != 1 {
		t.Fatal("سلام computed more than once", provider.History())
	}
	for _, doc := range docs {
		if !doc.Complete {
			t.Fatal("completed lifecycle remained open")
		}
		expected := 0
		if doc.TurnID == turn {
			expected = 1
		}
		if doc.Totals.ProviderCalls != expected || doc.Totals.InputTokens != expected*4 || doc.Totals.EstimatedCostMicroUSD != int64(expected*4) {
			t.Fatal("provider accounting duplicated", doc.Totals)
		}
		compute, reuse := 0, 0
		for _, n := range doc.Nodes {
			if n.Operation == "embedding.compute" {
				compute++
			}
			if n.Operation == "embedding.cache_hit" {
				reuse++
				if n.Embedding == nil || n.Embedding.OriginTurnID != turn.String() {
					t.Fatal("cross-Turn origin lost")
				}
			}
		}
		if compute != expected || expected == 0 && reuse != 1 {
			t.Fatal("logical work represented as duplicate provider computation", compute, reuse)
		}
		if dest := os.Getenv("THEIA_WRITE_SHARED_EMBEDDING_FIXTURES"); dest != "" {
			name := "shared_embedding_origin.json"
			if doc.TurnID != turn {
				name = "shared_embedding_reuse.json"
			}
			raw, _ := json.MarshalIndent(doc, "", "  ")
			if e = os.MkdirAll(dest, 0700); e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(filepath.Join(dest, name), raw, 0600); e != nil {
				t.Fatal(e)
			}
		}
	}
}

func TestInspectorAbandonedComputationExtendsObservedTurnEnd(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	shared := service(t, embeddingFunc(func(ctx context.Context, _ ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		close(entered)
		select {
		case <-release:
			return good(), nil
		case <-ctx.Done():
			return ports.EmbeddingResponse{}, ctx.Err()
		}
	}), Options{})
	dir := filepath.Join(t.TempDir(), "private")
	ins, e := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if e != nil {
		t.Fatal(e)
	}
	req := request()
	ctx, root := ins.StartTurn(context.Background(), c.TurnRequest{UserID: req.Meta.UserID, ClientTurnID: c.NewID()})
	trace.Associate(ctx, req.Meta.TurnID)
	caller, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { _, e := shared.Embed(caller, req); done <- e }()
	<-entered
	cancel()
	<-done
	root.End(nil, nil)
	close(release)
	waitFor(t, func() bool { return idle(shared) })
	docs := documents(t, dir)
	if len(docs) != 1 || !docs[0].Complete || docs[0].EndedAt == nil || docs[0].ResponseEndedAt == nil || !docs[0].EndedAt.After(*docs[0].ResponseEndedAt) {
		t.Fatal("detached computation did not extend the original trace lifecycle", docs)
	}
}
