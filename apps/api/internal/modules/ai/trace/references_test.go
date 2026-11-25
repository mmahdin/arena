package trace_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
)

type refWriter struct {
	mu     sync.Mutex
	refs   []trace.RootReference
	failed bool
}

func (s *refWriter) WriteTraceReference(_ context.Context, r trace.RootReference) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refs = append(s.refs, r)
	if s.failed {
		return errors.New("SECRET SQL ERROR")
	}
	return nil
}
func (s *refWriter) snapshot() []trace.RootReference {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]trace.RootReference{}, s.refs...)
}

func TestRootReferenceUsesActualRootAndIndependentJobAttempt(t *testing.T) {
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer receiver.Close()
	rt, err := telemetry.New(context.Background(), telemetry.Options{Enabled: true, Endpoint: receiver.URL + "/v1/traces", ServiceName: "theia-api", Environment: "local"})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	refs := &refWriter{}
	ctx := trace.WithReferenceWriter(telemetry.WithRuntime(context.Background(), rt), refs)
	ctx = trace.WithTimeouts(ctx, trace.Timeouts{ReferenceWrite: time.Second})
	ctx, turnSpan := (*trace.Inspector)(nil).StartTurn(ctx, contract.TurnRequest{Text: "PRIVATE USER INPUT"})
	turnID, jobID := uuid.New(), uuid.New()
	rootTrace, rootSpan, ok := telemetry.ActiveRootIDs(ctx)
	if !ok {
		t.Fatal("root missing")
	}
	stageCtx, stage := trace.Start(ctx, "gateway.Admit", func() any { t.Fatal("metadata-only stage input evaluated"); return nil })
	trace.RecordAdmittedTurn(stageCtx, turnID)
	stage.End(nil, nil)
	turnSpan.End(nil, nil)
	jobCtx, jobSpan := trace.StartJob(ctx, turnID, jobID, "reflection", 2)
	jobTrace, jobRoot, ok := telemetry.ActiveRootIDs(jobCtx)
	if !ok || jobTrace == rootTrace || jobRoot == rootSpan {
		t.Fatal("worker not an independent root")
	}
	trace.RecordClaimedJob(jobCtx, turnID, jobID, "reflection", 2)
	jobSpan.End(nil, nil)
	v := refs.snapshot()
	if len(v) != 2 || v[0].TurnID != turnID || v[0].Kind != "turn" || v[0].TraceID != rootTrace || v[0].RootSpanID != rootSpan ||
		v[1].Kind != "reflection" || v[1].JobID != jobID || v[1].JobAttempt != 2 || v[1].TraceID != jobTrace || v[1].RootSpanID != jobRoot {
		t.Fatalf("wrong root mapping: %+v", v)
	}
	for _, r := range v {
		if len(r.TraceID) != 32 || len(r.RootSpanID) != 16 || r.TraceID == "PRIVATE USER INPUT" {
			t.Fatal("unsafe trace reference")
		}
	}
}

func TestMissingTelemetryOrInvalidJobCannotPersist(t *testing.T) {
	sink := &refWriter{}
	ctx := trace.WithReferenceWriter(context.Background(), sink)
	trace.RecordAdmittedTurn(ctx, uuid.New())
	trace.RecordClaimedJob(ctx, uuid.New(), uuid.New(), "memory", 1)
	trace.RecordClaimedJob(ctx, uuid.New(), uuid.Nil, "memory", 1)
	trace.RecordClaimedJob(ctx, uuid.New(), uuid.New(), "unknown", 1)
	trace.RecordClaimedJob(ctx, uuid.New(), uuid.New(), "memory", 0)
	if len(sink.snapshot()) != 0 {
		t.Fatal("metadata persisted without an active OTel root")
	}
}
