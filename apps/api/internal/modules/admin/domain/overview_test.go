package domain

import (
	"errors"
	"testing"
	"time"
)

func TestOverviewWindowBounds(t *testing.T) {
	good := NewOverviewFilter(time.Date(2026, 11, 19, 14, 0, 0, 0, time.UTC))
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, f := range []OverviewFilter{
		{From: good.To, To: good.From, AsOf: good.AsOf},
		{From: good.From.Add(-31 * 24 * time.Hour), To: good.To, AsOf: good.AsOf},
		{From: good.From, To: good.To.Add(10 * time.Second), AsOf: good.AsOf},
		{From: good.From, To: good.To},
	} {
		if !errors.Is(f.Validate(), ErrInvalidOverview) {
			t.Fatalf("invalid window accepted: %+v", f)
		}
	}
}

func TestOverviewCoverageAndNulls(t *testing.T) {
	o := Overview{Turns: TurnSummary{TotalV1: 4, Completed: 2, Failed: 1, LatencySamples: 1},
		LLM:     LLMSummary{Attempts: 3, UsageKnown: 1, EstimatedCostCount: 1, UnknownCostCount: 1, UnpricedCostCount: 1},
		Workers: []WorkerSummary{{Kind: "memory", Status: "stale"}, {Kind: "reflection", Status: "running"}, {Kind: "scheduler", Status: "unknown"}}}
	o.Complete()
	if o.Turns.FailureRate == nil || *o.Turns.FailureRate != float64(1)/3 || o.LLM.UnknownUsage != 2 || !o.LLM.CostPartial {
		t.Fatalf("incorrect partial coverage: %+v", o)
	}
	if o.Turns.P95ElapsedMs != nil || o.LLM.EstimatedMicroUSD != nil {
		t.Fatal("missing metrics converted to zero")
	}
	if len(o.Warnings) != 5 {
		t.Fatalf("expected incomplete latency, costs, no stage events and stale worker warnings: %v", o.Warnings)
	}
	empty := Overview{}
	empty.Complete()
	if empty.Turns.FailureRate != nil || !empty.LLM.CostPartial || empty.LLM.EstimatedMicroUSD != nil {
		t.Fatal("empty evidence incorrectly reported as measured")
	}
}
