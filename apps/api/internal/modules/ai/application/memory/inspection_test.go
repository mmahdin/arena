package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/memorycrypto"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInspectorMemoryReconciliationIncludesEvidenceAndStateChanges(t *testing.T) {
	a := assets(t)
	snap, meta, op, now := fixture()
	dir := filepath.Join(t.TempDir(), "private")
	ins, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, root := ins.StartTurn(context.Background(), c.TurnRequest{Text: "synthetic memory turn"})
	plan, err := a.Reconcile(ctx, snap, meta, []c.MemoryOpProposal{op}, origin(a), nil, testPolicy{}, now)
	if err != nil {
		t.Fatal(err)
	}
	root.End(func() any { return plan }, nil)
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	raw, _ := os.ReadFile(files[0])
	if err = trace.ValidateJSON(raw); err != nil {
		t.Fatal(err)
	}
	var d trace.Document
	_ = json.Unmarshal(raw, &d)
	if len(d.Nodes) != 2 {
		t.Fatal(d)
	}
	n := d.Nodes[1]
	if !strings.Contains(string(n.Input), op.Content) || !strings.Contains(string(n.Input), "Synthetic event") {
		t.Fatal("proposal/evidence omitted")
	}
	var out struct {
		Plan Plan `json:"result0"`
	}
	if err = json.Unmarshal(n.Output, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Plan.Changes) != len(plan.Changes) || len(plan.Changes) != 1 || out.Plan.Changes[0].After.Item.Content != plan.Changes[0].After.Item.Content {
		t.Fatal("before/after memory plan differs from execution")
	}
}

type inspectorMemoryQueue struct {
	Queue
	lease               Lease
	checkpoint          []byte
	failures, completed int
}

func (q *inspectorMemoryQueue) Claim(context.Context, string, time.Duration, c.BundleRef, int) (*Lease, error) {
	lease := q.lease
	lease.Checkpoint = q.checkpoint
	return &lease, nil
}
func (q *inspectorMemoryQueue) Checkpoint(_ context.Context, _ Lease, p []byte) error {
	q.checkpoint = append([]byte(nil), p...)
	return nil
}
func (q *inspectorMemoryQueue) Fail(context.Context, Lease, string, time.Duration) error {
	q.failures++
	return nil
}
func (q *inspectorMemoryQueue) Complete(context.Context, Lease) error { q.completed++; return nil }

type inspectorMemoryRepository struct {
	Repository
	snapshot Snapshot
	saves    int
}

func (r *inspectorMemoryRepository) WriterInput(_ context.Context, m Meta, t Task) (ports.MemWriterInput, error) {
	return ports.MemWriterInput{UserID: m.UserID, Bundle: t.Bundle}, nil
}
func (r *inspectorMemoryRepository) Receipt(context.Context, c.ID, c.ID, string) (ports.MemoryApplyReport, bool, error) {
	return ports.MemoryApplyReport{}, false, nil
}
func (r *inspectorMemoryRepository) Snapshot(context.Context, c.ID, Meta, []c.ID) (Snapshot, error) {
	return r.snapshot, nil
}
func (r *inspectorMemoryRepository) Save(context.Context, Meta, Snapshot, Plan, string) error {
	r.saves++
	if r.saves == 1 {
		return errors.New("synthetic transient storage failure")
	}
	return nil
}

type inspectorMemoryWriter struct{ calls int }

func (w *inspectorMemoryWriter) Propose(context.Context, ports.MemWriterInput) (c.MemoryWriterOutput, c.LLMUsage, error) {
	w.calls++
	return c.MemoryWriterOutput{SchemaVersion: c.MemoryWriterSchemaVersion, Ops: []c.MemoryOpProposal{}}, c.LLMUsage{Model: "synthetic-small"}, nil
}
func TestInspectorMemoryWorkerCheckpointRetryDoesNotInventModelWork(t *testing.T) {
	a := assets(t)
	snap, meta, _, now := fixture()
	repo := &inspectorMemoryRepository{snapshot: snap}
	cipher, _ := memorycrypto.New("test", map[string][]byte{"test": bytes.Repeat([]byte{3}, 32)})
	service := &Service{repo: repo, assets: a, opt: Options{Cipher: cipher, Policy: testPolicy{}, Now: func() time.Time { return now }}}
	taskPlain, _ := json.Marshal(taskData{})
	encrypted, err := cipher.Seal(context.Background(), meta.UserID, "memory-task:"+meta.TurnID.String(), taskPlain)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(Task{Schema: "memory_job.v1", OperationID: meta.OperationID, Bundle: a.Reference(), Data: encrypted})
	queue := &inspectorMemoryQueue{lease: Lease{ID: c.NewID(), UserID: meta.UserID, TurnID: meta.TurnID, Token: c.NewID(), Kind: "memory_write", Payload: payload, Attempts: 1}}
	writer := &inspectorMemoryWriter{}
	worker, _ := NewWorker(queue, repo, writer, service, "synthetic-worker")
	dir := filepath.Join(t.TempDir(), "private")
	ins, _ := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	parentCtx, parent := ins.StartTurn(context.Background(), c.TurnRequest{Text: "synthetic parent turn"})
	trace.Associate(parentCtx, meta.TurnID)
	parent.End(nil, nil)
	ctx := trace.WithInspector(context.Background(), ins)
	if _, err = worker.RunOnce(ctx); err == nil {
		t.Fatal("first save should fail")
	}
	queue.lease.Attempts = 2
	if _, err = worker.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if writer.calls != 1 || repo.saves != 2 || queue.failures != 1 || queue.completed != 1 {
		t.Fatal("checkpoint altered work", writer.calls, repo.saves, queue.failures, queue.completed)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(files) != 1 {
		t.Fatal(files)
	}
	jobAttempts := 0
	for _, file := range files {
		raw, _ := os.ReadFile(file)
		if err = trace.ValidateJSON(raw); err != nil {
			t.Fatal(err)
		}
		var d trace.Document
		_ = json.Unmarshal(raw, &d)
		if d.TurnID != meta.TurnID || d.Kind != "turn" || !d.Complete {
			t.Fatal("async linkage missing")
		}
		for _, n := range d.Nodes {
			if n.Operation == "memory.RunOnce" {
				jobAttempts++
				if !strings.Contains(string(n.Input), queue.lease.ID.String()) {
					t.Fatal("worker identity not retained")
				}
			}
		}
		if strings.Contains(string(raw), queue.lease.Token.String()) {
			t.Fatal("lease credential exposed")
		}
	}
	if jobAttempts != 2 {
		t.Fatal("both worker attempts must be visible in the same trace", jobAttempts)
	}
}
