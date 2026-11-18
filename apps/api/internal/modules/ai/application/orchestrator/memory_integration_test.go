package orchestrator_test

import (
	"context"
	"errors"
	"testing"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type wp12Memory struct {
	ref                c.BundleRef
	corrected          bool
	target             c.ID
	correctErr, jobErr error
	planned            int
}

func (m *wp12Memory) Reference() c.BundleRef { return m.ref }
func (m *wp12Memory) Correct(ctx context.Context, user, target c.ID, _ ports.CorrectionKind) error {
	metadata, ok := mem.Metadata(ctx)
	if !ok || metadata.UserID != user || !c.ValidID(metadata.OperationID) {
		return errors.New("missing memory identity")
	}
	m.corrected = true
	m.target = target
	return m.correctErr
}
func (m *wp12Memory) PlanJob(_ context.Context, _ c.Admission, _ c.BrainDecision, _ []c.UserClaim) (*c.AsyncJobProposal, error) {
	m.planned++
	if m.jobErr != nil {
		return nil, m.jobErr
	}
	return &c.AsyncJobProposal{Kind: "memory_write", DedupeKey: "synthetic", Payload: []byte(`{"synthetic":true}`)}, nil
}

type wp12Retrieval struct{ item c.MemoryView }

func (r wp12Retrieval) Retrieve(context.Context, c.RetrievalRequest) (c.RetrievalResult, error) {
	return c.RetrievalResult{Items: []c.MemoryView{r.item}}, nil
}

type wp12Generation struct {
	t      *testing.T
	memory *wp12Memory
	calls  int
	target c.ID
}

func (g *wp12Generation) Generate(_ context.Context, in ports.GenInput, _ ports.TokenSink) (ports.GenOutput, error) {
	g.calls++
	if !g.memory.corrected {
		g.t.Fatal("response generated before correction")
	}
	for _, id := range in.Plan.PermittedContext {
		if id == g.target {
			g.t.Fatal("corrected target still permitted")
		}
	}
	for _, v := range in.Permitted {
		if v.ID == g.target {
			g.t.Fatal("corrected content reached generator")
		}
	}
	return ports.GenOutput{Text: "Synthetic accepted repair."}, nil
}
func TestWP12SynchronousCorrectionBeforeGenerationAndAtomicJobHandoff(t *testing.T) {
	for _, failure := range []string{"none", "correction", "schedule"} {
		t.Run(failure, func(t *testing.T) {
			h := newTestIntegrationHarness(t)
			item := c.MemoryView{ID: c.NewID(), Kind: c.MemFact, Epistemic: c.EpReported, Status: c.MemActive, Text: "Synthetic disputed value", Age: "prior", EvidenceN: 1, Sensitivity: c.SensNormal}
			memory := &wp12Memory{ref: h.registry.b.Reference()}
			if failure == "correction" {
				memory.correctErr = errors.New("private correction failure")
			}
			if failure == "schedule" {
				memory.jobErr = errors.New("private scheduling failure")
			}
			p := &wp10Provider{makeDecision: func(n int, req ports.StructuredRequest) c.BrainDecision {
				d := wp10Decision(n, req)
				d.TurnAnalysis.Intent = c.IntentDisagreeCorrect
				d.TurnAnalysis.CorrectionTarget = &item.ID
				d.Decision.Primary = c.ObjRepair
				d.Plan.PermittedContext = []c.ID{item.ID}
				d.Needs.MemoryWorthy = true
				return d
			}}
			g := &wp12Generation{t: t, memory: memory, target: item.ID}
			o := wp10Runtime(t, h, p, wp12Retrieval{item}, g, false, memory)
			result, e := o.ProcessTurn(context.Background(), h.request("Synthetic correction of the previous value."))
			if failure == "correction" {
				if e == nil || g.calls != 0 || memory.planned != 0 {
					t.Fatal("failed correction rendered a reply", e)
				}
				return
			}
			if e != nil || result.Status != c.TurnCompleted || g.calls != 1 || !memory.corrected || memory.target != item.ID {
				t.Fatal(result, e)
			}
			final := h.last(t)
			if failure == "schedule" {
				if len(final.Jobs) != 0 || !result.Degraded {
					t.Fatal("scheduling outage blocked/deceptively succeeded")
				}
			} else if len(final.Jobs) != 1 || final.Jobs[0].Kind != "memory_write" {
				t.Fatal("job missing from finalizer transaction")
			}
		})
	}
}
