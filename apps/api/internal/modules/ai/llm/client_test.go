package llm_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

var booleanSchema = []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["ok"],"additionalProperties":false,"properties":{"ok":{"type":"boolean"}}}`)

func model(name string, native bool) llm.Model {
	return llm.Model{Provider: "p", Name: name, NativeJSONSchema: native, InputMicroUSDPerM: 2_000_000,
		OutputMicroUSDPerM: 4_000_000, CachedMicroUSDPerM: 500_000}
}
func config() llm.Config {
	return llm.Config{
		Chains: map[ports.ModelTier][]llm.Model{
			ports.TierSmall:      {model("small-primary", true), model("small-secondary", false)},
			ports.TierStrong:     {model("strong-primary", false), model("strong-secondary", true), model("small-primary", true)},
			ports.TierClassifier: {model("classifier", true)},
			ports.TierEmbedding:  {{Provider: "p", Name: "embedding", Dimensions: 3, InputMicroUSDPerM: 100_000}},
		},
		MaxRetriesPerModel: 1, RetryBase: 0, RetryMax: 0,
		DefaultTimeout: time.Second, BreakerWindow: time.Minute, BreakerCooldown: time.Minute,
		BreakerErrorRate: 0.5, BreakerMinSamples: 10,
	}
}
func request() ports.StructuredRequest {
	return ports.StructuredRequest{Tier: ports.TierSmall, Schema: booleanSchema, SchemaName: "test.v1", MaxTokens: 100,
		Meta: contract.CallMeta{UserID: contract.NewID()}, System: []ports.Block{{Name: "policy", Text: "synthetic", Cache: true}},
		Messages: []ports.ChatMessage{{Role: contract.RoleUser, Content: "synthetic"}}}
}
func client(t *testing.T, cfg llm.Config, p *fake.Provider, options llm.Options) *llm.Client {
	t.Helper()
	c, err := llm.New(cfg, map[string]llm.Provider{"p": p}, options)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestStructuredNativeSchemaUsageAndImmutableResponse(t *testing.T) {
	p := fake.New()
	p.QueueStructured("small-primary", fake.StructuredReply{JSON: []byte(`{"ok":true}`), Usage: contract.LLMUsage{InTokens: 100, OutTokens: 20, CachedTokens: 20, FinishReason: "stop"}})
	ledger := llm.NewMemoryLedger(nil)
	c := client(t, config(), p, llm.Options{Ledger: ledger})
	req := request()
	result, err := c.Structured(context.Background(), req)
	if err != nil || result.Provider != "p" || result.Model != "small-primary" || result.Attempts != 1 || result.Usage.Attempts != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	// 80*2 + 20*0.5 + 20*4 microUSD (input/output rates quoted per 1M).
	if result.Usage.CostMicroUSD != 250 || ledger.Snapshot(req.Meta.UserID, time.Now()).CostMicroUSD != 250 {
		t.Fatalf("cost not tracked: %+v", result.Usage)
	}
	if calls := p.History(); len(calls) != 1 || calls[0].Mode != llm.ModeNativeSchema {
		t.Fatal(calls)
	}
}
func TestJSONModeSingleRepairAndNoNewReasoningPath(t *testing.T) {
	p := fake.New()
	p.QueueStructured("small-primary", fake.StructuredReply{Err: &llm.Failure{Kind: llm.FailurePermanent}})
	p.QueueStructured("small-secondary", fake.StructuredReply{JSON: []byte(`{"ok":"not boolean"}`), Usage: contract.LLMUsage{InTokens: 10}},
		fake.StructuredReply{JSON: []byte(`{"ok":true}`), Usage: contract.LLMUsage{InTokens: 10}})
	cfg := config()
	cfg.MaxRetriesPerModel = 0
	c := client(t, cfg, p, llm.Options{})
	result, err := c.Structured(context.Background(), request())
	if err != nil || result.Attempts != 3 || result.Model != "small-secondary" || result.Usage.InTokens != 20 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	calls := p.History()
	if len(calls) != 3 || calls[1].Mode != llm.ModeJSONObject || calls[2].Mode != llm.ModeJSONObject {
		t.Fatal(calls)
	}
	p = fake.New()
	p.QueueStructured("small-primary", fake.StructuredReply{JSON: []byte(`{"ok":"bad"}`)})
	c = client(t, cfg, p, llm.Options{})
	if _, err := c.Structured(context.Background(), request()); !errors.Is(err, llm.ErrInvalid) || len(p.History()) != 1 {
		t.Fatalf("native invalid must fail without repair/fallback: %v", err)
	}
	p = fake.New()
	p.QueueStructured("small-primary", fake.StructuredReply{Err: &llm.Failure{Kind: llm.FailurePermanent}})
	p.QueueStructured("small-secondary", fake.StructuredReply{JSON: []byte(`{"ok":"bad"}`)}, fake.StructuredReply{JSON: []byte(`{"no":true}`)})
	c = client(t, cfg, p, llm.Options{})
	if _, err := c.Structured(context.Background(), request()); !errors.Is(err, llm.ErrInvalid) || len(p.History()) != 3 {
		t.Fatalf("one repair only and no third model after invalid: %v", err)
	}
}
func TestRetryThenFallbackAndBreaker(t *testing.T) {
	p := fake.New()
	p.QueueStructured("small-primary", fake.StructuredReply{Err: &llm.Failure{Kind: llm.FailureServer}},
		fake.StructuredReply{Err: &llm.Failure{Kind: llm.FailureRateLimit}})
	p.QueueStructured("small-secondary", fake.StructuredReply{JSON: []byte(`{"ok":true}`)})
	c := client(t, config(), p, llm.Options{Jitter: func(time.Duration) time.Duration { return 0 }})
	result, err := c.Structured(context.Background(), request())
	if err != nil || result.Attempts != 3 || result.Model != "small-secondary" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	clock := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	now := func() time.Time { return clock }
	cfg := config()
	cfg.MaxRetriesPerModel = 0
	cfg.BreakerMinSamples = 2
	p = fake.New()
	p.QueueStructured("small-primary", fake.StructuredReply{Err: &llm.Failure{Kind: llm.FailureServer}}, fake.StructuredReply{Err: &llm.Failure{Kind: llm.FailureServer}},
		fake.StructuredReply{JSON: []byte(`{"ok":true}`)})
	p.QueueStructured("small-secondary", fake.StructuredReply{JSON: []byte(`{"ok":true}`)}, fake.StructuredReply{JSON: []byte(`{"ok":true}`)}, fake.StructuredReply{JSON: []byte(`{"ok":true}`)})
	c = client(t, cfg, p, llm.Options{Now: now})
	for i := 0; i < 3; i++ {
		if _, err := c.Structured(context.Background(), request()); err != nil {
			t.Fatal(err)
		}
	}
	calls := p.History()
	if len(calls) != 5 || calls[4].Model != "small-secondary" {
		t.Fatalf("breaker should skip open model: %+v", calls)
	}
	clock = clock.Add(time.Minute + time.Second)
	if _, err := c.Structured(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	calls = p.History()
	if calls[len(calls)-1].Model != "small-primary" {
		t.Fatalf("half-open probe not sent: %+v", calls)
	}
}
func TestDeadlineCancellationAndNoRetryAfterBudget(t *testing.T) {
	p := fake.New()
	p.QueueStructured("small-primary", fake.StructuredReply{JSON: []byte(`{"ok":true}`), Delay: 100 * time.Millisecond})
	c := client(t, config(), p, llm.Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	if _, err := c.Structured(ctx, request()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline should cancel upstream: %v", err)
	}
	if len(p.History()) != 1 {
		t.Fatalf("must not retry context deadline: %+v", p.History())
	}
}
func TestStreamingPreTokenFallbackAndPostTokenNoRetry(t *testing.T) {
	p := fake.New()
	p.QueueStream("strong-primary", fake.StreamReply{FailAt: 0, Err: &llm.Failure{Kind: llm.FailureTransport}})
	p.QueueStream("strong-secondary", fake.StreamReply{FailAt: -1, Chunks: []ports.Chunk{{Text: "hello"},
		{Usage: &contract.LLMUsage{InTokens: 20, OutTokens: 5}}, {FinishReason: "stop"}}})
	ledger := llm.NewMemoryLedger(nil)
	cfg := config()
	cfg.MaxRetriesPerModel = 0
	c := client(t, cfg, p, llm.Options{Ledger: ledger})
	req := ports.StreamRequest{Tier: ports.TierStrong, MaxTokens: 100, Meta: contract.CallMeta{UserID: contract.NewID()}}
	s, err := c.Stream(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var text string
	var usage *contract.LLMUsage
	for {
		chunk, err := s.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		text += chunk.Text
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}
	if text != "hello" || usage == nil || usage.Attempts != 2 || ledger.Snapshot(req.Meta.UserID, time.Now()).InTokens != 20 {
		t.Fatalf("stream=%q usage=%+v", text, usage)
	}
	p = fake.New()
	p.QueueStream("strong-primary", fake.StreamReply{Chunks: []ports.Chunk{{Text: "partial"}}, FailAt: 1, Err: &llm.Failure{Kind: llm.FailureTransport}})
	p.QueueStream("strong-secondary", fake.StreamReply{Chunks: []ports.Chunk{{Text: "MUST NOT APPEAR"}}, FailAt: -1})
	c = client(t, cfg, p, llm.Options{})
	s, err = c.Stream(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if chunk, err := s.Recv(); err != nil || chunk.Text != "partial" {
		t.Fatal(chunk, err)
	}
	if _, err := s.Recv(); !errors.Is(err, llm.ErrUnavailable) || len(p.History()) != 1 {
		t.Fatalf("cannot retry after token: %v %+v", err, p.History())
	}
	_ = s.Close()
}
func TestStreamingFirstTokenTimeoutAndCancellation(t *testing.T) {
	p := fake.New()
	p.QueueStream("strong-primary", fake.StreamReply{Delay: 25 * time.Millisecond, FailAt: -1, Chunks: []ports.Chunk{{Text: "late"}}})
	p.QueueStream("strong-secondary", fake.StreamReply{FailAt: -1, Chunks: []ports.Chunk{{Text: "good"}}})
	cfg := config()
	cfg.MaxRetriesPerModel = 0
	c := client(t, cfg, p, llm.Options{})
	req := ports.StreamRequest{Tier: ports.TierStrong, MaxTokens: 100, FirstToken: 5 * time.Millisecond, Meta: contract.CallMeta{UserID: contract.NewID()}}
	s, err := c.Stream(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	chunk, err := s.Recv()
	if err != nil || chunk.Text != "good" {
		t.Fatalf("pre-token timeout must hop, got %+v err=%v", chunk, err)
	}
	if len(p.History()) != 2 {
		t.Fatal(p.History())
	}
	_ = s.Close()
	if _, err := s.Recv(); !errors.Is(err, llm.ErrCancelled) {
		t.Fatalf("closed stream must not emit: %v", err)
	}
}
func TestEmbeddingsQuotaAndUsage(t *testing.T) {
	p := fake.New()
	p.QueueEmbedding("embedding", fake.EmbeddingReply{Vector: []float32{0, 0.6, 0.8}, Usage: contract.LLMUsage{InTokens: 10}})
	ledger := llm.NewMemoryLedger(nil)
	cfg := config()
	cfg.DailyQuota.Calls = 1
	c := client(t, cfg, p, llm.Options{Ledger: ledger})
	req := ports.EmbeddingRequest{Tier: ports.TierEmbedding, Input: "synthetic test", Meta: contract.CallMeta{UserID: contract.NewID()}}
	result, err := c.Embed(context.Background(), req)
	if err != nil || result.Model != "embedding" || len(result.Vector) != 3 || result.Usage.CostMicroUSD != 1 {
		t.Fatalf("embedding=%+v err=%v", result, err)
	}
	if _, err := c.Embed(context.Background(), req); !errors.Is(err, llm.ErrQuota) || len(p.History()) != 1 {
		t.Fatalf("quota not enforced: %v", err)
	}
}
func TestConfigInvalidCostsAndConcurrency(t *testing.T) {
	cfg := config()
	delete(cfg.Chains, ports.TierEmbedding)
	if _, err := llm.New(cfg, map[string]llm.Provider{"p": fake.New()}, llm.Options{}); !errors.Is(err, llm.ErrConfig) {
		t.Fatalf("missing tier accepted: %v", err)
	}
	if _, err := llm.CostMicroUSD(model("m", false), 5, 5, 6); !errors.Is(err, llm.ErrUsage) {
		t.Fatalf("invalid cached usage accepted: %v", err)
	}
	ledger := llm.NewMemoryLedger(nil)
	user := contract.NewID()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := ledger.Record(context.Background(), user, contract.LLMUsage{InTokens: 1, Attempts: 1}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if totals := ledger.Snapshot(user, time.Now()); totals.Calls != 50 || totals.InTokens != 50 {
		t.Fatalf("concurrent charges lost: %+v", totals)
	}
}
func TestNoSensitiveProviderErrors(t *testing.T) {
	p := fake.New()
	p.QueueStructured("small-primary", fake.StructuredReply{Err: &llm.Failure{Kind: llm.FailurePermanent}})
	p.QueueStructured("small-secondary", fake.StructuredReply{Err: &llm.Failure{Kind: llm.FailurePermanent}})
	cfg := config()
	cfg.MaxRetriesPerModel = 0
	c := client(t, cfg, p, llm.Options{})
	_, err := c.Structured(context.Background(), request())
	if !errors.Is(err, llm.ErrUnavailable) || strings.Contains(err.Error(), "synthetic") {
		t.Fatalf("unsafe provider failure: %v", err)
	}
}

type capturedMetrics struct {
	mu     sync.Mutex
	events []llm.Event
}

func (c *capturedMetrics) Observe(e llm.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}
func (c *capturedMetrics) All() []llm.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]llm.Event(nil), c.events...)
}
func TestBudgetAndStructuredSchemaStrictness(t *testing.T) {
	p := fake.New()
	p.QueueStructured("small-primary", fake.StructuredReply{JSON: []byte(`{"ok":true,"ok":false}`)},
		fake.StructuredReply{JSON: []byte(`{"ok":true}`)})
	cfg := config()
	cfg.Chains[ports.TierSmall][0].NativeJSONSchema = false
	meter := &capturedMetrics{}
	c := client(t, cfg, p, llm.Options{Metrics: meter})
	req := request()
	result, err := c.Structured(context.Background(), req)
	if err != nil || string(result.JSON) != `{"ok":true}` || result.Attempts != 2 {
		t.Fatalf("duplicate JSON key must require repair: %+v %v", result, err)
	}
	if len(meter.All()) < 2 {
		t.Fatalf("attempt events missing")
	}
	// A call cost is checked before exposing content; billable usage remains
	// recorded even when it exhausts the budget (cannot unspend provider calls).
	p = fake.New()
	p.QueueStructured("small-primary", fake.StructuredReply{JSON: []byte(`{"ok":true}`), Usage: contract.LLMUsage{InTokens: 10}})
	c = client(t, cfg, p, llm.Options{})
	ctx := contract.WithBudget(context.Background(), contract.Budget{MaxCostMicroUSD: 1})
	if _, err := c.Structured(ctx, req); err == nil {
		t.Fatal("cost budget not enforced")
	} else {
		var typed *contract.Error
		if !errors.As(err, &typed) || typed.Code != contract.ErrBudget {
			t.Fatalf("unexpected cost budget error: %v", err)
		}
	}
}
func TestIncompleteStreamDoesNotReturnCleanEOF(t *testing.T) {
	p := fake.New()
	p.QueueStream("strong-primary", fake.StreamReply{FailAt: -1, Chunks: []ports.Chunk{{Text: "synthetic token"}, {FinishReason: "stop"}}})
	cfg := config()
	cfg.MaxRetriesPerModel = 0
	c := client(t, cfg, p, llm.Options{})
	stream, err := c.Stream(context.Background(), ports.StreamRequest{Tier: ports.TierStrong, MaxTokens: 20, Meta: contract.CallMeta{UserID: contract.NewID()}})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if _, err = stream.Recv(); err != nil {
		t.Fatal(err)
	}
	if _, err = stream.Recv(); err != nil {
		t.Fatal(err)
	}
	if _, err = stream.Recv(); !errors.Is(err, llm.ErrInvalid) {
		t.Fatalf("missing usage must not be accepted as complete: %v", err)
	}
}

func TestCloseUnblocksPendingRecv(t *testing.T) {
	p := fake.New()
	p.QueueStream("strong-primary", fake.StreamReply{Delay: 200 * time.Millisecond, FailAt: -1,
		Chunks: []ports.Chunk{{Text: "must-not-escape"}}})
	cfg := config()
	cfg.MaxRetriesPerModel = 0
	c := client(t, cfg, p, llm.Options{})
	s, err := c.Stream(context.Background(), ports.StreamRequest{Tier: ports.TierStrong, MaxTokens: 100,
		FirstToken: time.Second, Meta: contract.CallMeta{UserID: contract.NewID()}})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		chunk, err := s.Recv()
		if chunk.Text != "" {
			done <- errors.New("closed stream leaked text")
			return
		}
		done <- err
	}()
	time.Sleep(5 * time.Millisecond)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Recv succeeded after concurrent Close")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Close did not unblock Recv")
	}
}
