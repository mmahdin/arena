package llm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/embedding"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func TestSharedEmbeddingPreservesFallbackAttemptsAndLedger(t *testing.T) {
	cfg := config()
	cfg.Chains[ports.TierEmbedding] = append(cfg.Chains[ports.TierEmbedding], llm.Model{Provider: "p", Name: "fallback-embedding", Dimensions: 2, InputMicroUSDPerM: 100000})
	provider := fake.New()
	provider.QueueEmbedding("embedding", fake.EmbeddingReply{Err: &llm.Failure{Kind: llm.FailurePermanent}}, fake.EmbeddingReply{Vector: []float32{1, 0, 0}, Usage: c.LLMUsage{InTokens: 4}})
	provider.QueueEmbedding("fallback-embedding", fake.EmbeddingReply{Vector: []float32{1, 0}, Usage: c.LLMUsage{InTokens: 8}})
	ledger := llm.NewMemoryLedger(time.Now)
	underlying := client(t, cfg, provider, llm.Options{Ledger: ledger})
	shared, e := embedding.New(context.Background(), underlying, embedding.Options{})
	if e != nil {
		t.Fatal(e)
	}
	defer shared.Close()
	req := ports.EmbeddingRequest{Tier: ports.TierEmbedding, Input: "Synthetic compatible vector spaces", Meta: c.CallMeta{UserID: c.NewID()}}
	first, e := shared.Embed(context.Background(), req)
	if e != nil || first.Model != "fallback-embedding" || first.Usage.Attempts != 2 {
		t.Fatal(first, e)
	}
	second, e := shared.Embed(context.Background(), req)
	if e != nil || second.Model != "embedding" {
		t.Fatal("fallback poisoned primary cache", second, e)
	}
	cached, e := shared.Embed(context.Background(), req)
	if e != nil || cached.Usage.InTokens != 0 || cached.Usage.CostMicroUSD != 0 {
		t.Fatal("reuse recharged usage", cached, e)
	}
	if len(provider.History()) != 3 {
		t.Fatal("underlying attempt/fallback semantics changed", provider.History())
	}
	if totals := ledger.Snapshot(req.Meta.UserID, time.Now()); totals.Calls != 2 || totals.InTokens != 12 || totals.CostMicroUSD != 2 {
		t.Fatal("ledger duplicated/suppressed physical charges", totals)
	}
}

func TestSharedEmbeddingQuotaStillGovernsActualExecutions(t *testing.T) {
	cfg := config()
	cfg.DailyQuota.Calls = 1
	provider := fake.New()
	provider.QueueEmbedding("embedding", fake.EmbeddingReply{Vector: []float32{1, 0, 0}, Usage: c.LLMUsage{InTokens: 4}})
	underlying := client(t, cfg, provider, llm.Options{})
	shared, e := embedding.New(context.Background(), underlying, embedding.Options{})
	if e != nil {
		t.Fatal(e)
	}
	defer shared.Close()
	req := ports.EmbeddingRequest{Tier: ports.TierEmbedding, Input: "Synthetic quota", Meta: c.CallMeta{UserID: c.NewID()}}
	if _, e = shared.Embed(context.Background(), req); e != nil {
		t.Fatal(e)
	}
	if _, e = shared.Embed(context.Background(), req); e != nil {
		t.Fatal("cache read incorrectly treated as provider quota spend", e)
	}
	req.Input = "Different synthetic quota input"
	if _, e = shared.Embed(context.Background(), req); !errors.Is(e, llm.ErrQuota) {
		t.Fatal("new execution bypassed quota", e)
	}
	if len(provider.History()) != 1 {
		t.Fatal("quota bypassed or cache called provider")
	}
}
