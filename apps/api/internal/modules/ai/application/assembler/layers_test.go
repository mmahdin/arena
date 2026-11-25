package assembler

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"strings"
	"testing"
	"time"
)

func TestLayersPreserveBaselineDuringOutageAndBudgetIndependently(t *testing.T) {
	a, r, req, _, _ := fixture(t)
	w, _, e := a.Build(context.Background(), req, c.SafetySignal{Level: c.RiskNone}, c.RetrievalResult{})
	if e != nil {
		t.Fatal(e)
	}
	original := w.Turn.Text
	p := c.PersistentContext{}
	for i := 0; i < 30; i++ {
		p.Items = append(p.Items, c.MemoryView{ID: c.NewID(), Source: "memory", Kind: c.MemFact, Epistemic: c.EpReported, Status: c.MemActive, Sensitivity: c.SensNormal, Text: strings.Repeat("synthetic confirmed fact ", 70)})
	}
	for _, degraded := range []bool{false, true} {
		retr := c.RetrievalResult{Meta: c.RetrievalMeta{Degraded: degraded, Strategies: []string{"bm25"}}}
		for i := 0; i < 50; i++ {
			retr.Items = append(retr.Items, c.MemoryView{ID: c.NewID(), Text: strings.Repeat("synthetic evidence ", 80), Kind: c.MemEpisode, Epistemic: c.EpReported, Status: c.MemActive})
		}
		got := BoundActive(Layers(w, r.snapshot, p, retr))
		if got.Turn.Text != original || len(got.Persistent.Items) == 0 || got.LayerTokens.Persistent > 700 || got.LayerTokens.Retrieved > 1200 || got.LayerTokens.Recent > 1000 || got.LayerTokens.Active > 3000 {
			t.Fatal("independent budgets or baseline failed", got.LayerTokens)
		}
		if len(got.Retrieved) == 0 {
			t.Fatal("valid partial lexical evidence discarded")
		}
	}
}
func TestLayersExposeReturningThreadsHypothesesAndFormulations(t *testing.T) {
	owner := c.NewID()
	focus, old := c.NewID(), c.NewID()
	hyp := c.NewID()
	now := time.Now()
	s := contextstate.Snapshot{User: c.UserState{UserID: owner}, Session: c.SessionRecord{}, Threads: []c.ThreadRecord{{ID: focus, UserID: owner, Phase: c.PhaseWorking, LastActiveAt: now}, {ID: old, UserID: owner, Topic: "fictional earlier concern", Phase: c.PhasePaused, PrevPhase: c.PhaseWorking, LastActiveAt: now.AddDate(0, 0, -200), HypothesisIDs: []c.ID{hyp}}}, Formulations: []c.FormulationBrief{{ThreadID: old, Version: 2, Paragraph: "An explicitly tentative synthetic formulation"}}}
	w := c.WorkingContext{FocusThread: &c.ThreadView{ID: focus}, Turn: c.TurnView{Text: "return to the old issue"}}
	baseline := c.PersistentContext{Items: []c.MemoryView{{ID: hyp, Kind: c.MemHypothesis, Epistemic: c.EpInferred, Status: c.MemSupported, Text: "may reflect a recurring synthetic concern"}}}
	got := Layers(w, s, baseline, c.RetrievalResult{Items: []c.MemoryView{{ID: old, Source: "thread", Text: "fictional old thread"}}})
	if len(got.ActiveThreads) != 1 || got.ActiveThreads[0].ID != old || len(got.ActiveThreads[0].Hypotheses) != 0 || len(got.Persistent.Items) != 1 || got.Persistent.Items[0].Epistemic != c.EpInferred || len(got.Formulations) != 1 {
		t.Fatal("returning knowledge missing", got.ActiveThreads, got.Formulations)
	}
}
