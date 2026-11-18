package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/gateway"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type retrievalSpy struct {
	request c.RetrievalRequest
	calls   int
}

func (r *retrievalSpy) Retrieve(_ context.Context, q c.RetrievalRequest) (c.RetrievalResult, error) {
	r.calls++
	r.request = q
	return c.RetrievalResult{Items: []c.MemoryView{}, Meta: c.RetrievalMeta{Strategies: []string{"hybrid"}}}, nil
}

func TestWP09OrchestratorOnlyPassesEarlierMessageIDs(t *testing.T) {
	h := newHarness(t, orchestrator.Options{})
	first := h.request(h.conv)
	previous, e := h.o.ProcessTurn(context.Background(), first)
	if e != nil || previous.Status != c.TurnCompleted {
		t.Fatalf("seed turn: %+v %v", previous, e)
	}
	spy := &retrievalSpy{}
	gw, e := gateway.New(h.s, h.s)
	if e != nil {
		t.Fatal(e)
	}
	o, e := orchestrator.New(orchestrator.Dependencies{Gateway: gw, Store: h.s, Finalizer: h.s, Recovery: h.s,
		Safety: safety{level: c.RiskLow}, Assembler: assembler{}, Retriever: spy, Brain: h.b, Generator: h.g,
		Validator: h.validator, Tracer: h.tr, Crisis: crisisPlanner{}}, orchestrator.Options{
		TotalTimeout: 5 * time.Second, Fallback: func(c.Route, c.Objective, string) (string, bool) { return "Synthetic fallback", true }})
	if e != nil {
		t.Fatal(e)
	}
	next := h.request(h.conv)
	_, e = o.ProcessTurn(context.Background(), next)
	if e != nil {
		t.Fatal(e)
	}
	if spy.calls != 1 || spy.request.UserID != h.user || spy.request.Turn.Text != next.Text || !c.ValidID(spy.request.Turn.MessageID) || len(spy.request.Exclude) != 2 {
		t.Fatalf("retrieval handoff did not exclude existing context: %+v", spy.request)
	}
	for _, id := range spy.request.Exclude {
		if id == spy.request.Turn.MessageID {
			t.Fatal("current trigger in historical context")
		}
	}
}
