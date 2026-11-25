package memory

import (
	"context"
	"errors"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"testing"
	"time"
)

type invariantRepo struct {
	Repository
	snapshot  Snapshot
	committed Plan
	saves     int
}

func (r *invariantRepo) Snapshot(context.Context, c.ID, Meta, []c.ID) (Snapshot, error) {
	return r.snapshot, nil
}
func (r *invariantRepo) Receipt(context.Context, c.ID, c.ID, string) (ports.MemoryApplyReport, bool, error) {
	return ports.MemoryApplyReport{}, false, nil
}
func (r *invariantRepo) Save(_ context.Context, _ Meta, _ Snapshot, p Plan, _ string) error {
	r.committed = p
	r.saves++
	return nil
}

type invariantCipher struct{}

func (invariantCipher) Open(_ context.Context, _ c.ID, _ string, v []byte) ([]byte, error) {
	return v, nil
}
func (invariantCipher) Seal(_ context.Context, _ c.ID, _ string, v []byte) ([]byte, error) {
	return v, nil
}

type invariantEmbed struct {
	calls int
	fail  bool
}

func (e *invariantEmbed) Embed(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	e.calls++
	if e.fail {
		return ports.EmbeddingResponse{}, errors.New("synthetic embedding outage")
	}
	return ports.EmbeddingResponse{Model: "synthetic", Vector: []float32{1, 0}}, nil
}
func TestValidPersistenceNeverCallsEmbedding(t *testing.T) {
	a := assets(t)
	snap, meta, op, now := fixture()
	meta.Bundle = a.Reference()
	repo := &invariantRepo{snapshot: snap}
	embed := &invariantEmbed{fail: true}
	s, e := New(repo, a, Options{Policy: testPolicy{}, Cipher: invariantCipher{}, Embedder: embed, EmbeddingModel: "synthetic", Dimension: 2, Now: func() time.Time { return now }})
	if e != nil {
		t.Fatal(e)
	}
	out, e := s.Apply(WithMeta(context.Background(), meta), meta.UserID, []c.MemoryOpProposal{op}, origin(a))
	if e != nil || len(out.Applied) != 1 || repo.saves != 1 || embed.calls != 0 || len(repo.committed.Changes) != 1 {
		t.Fatal("embedding gated persistence", out, e, embed.calls)
	}
	if _, e = New(repo, a, Options{Policy: testPolicy{}, Cipher: invariantCipher{}, EmbeddingModel: "synthetic", Dimension: 2}); e != nil {
		t.Fatal("persistence constructor requires embedder", e)
	}
}

type indexTestQueue struct {
	Queue
	lease                                     Lease
	input                                     EmbeddingInput
	eligible                                  bool
	failures, published, completed, backfills int
}

func (q *indexTestQueue) Claim(context.Context, string, time.Duration, c.BundleRef, int) (*Lease, error) {
	q.lease.Attempts++
	l := q.lease
	return &l, nil
}
func (q *indexTestQueue) BackfillEmbeddings(context.Context, int) error { q.backfills++; return nil }
func (q *indexTestQueue) EmbeddingInput(context.Context, Lease) (EmbeddingInput, bool, error) {
	return q.input, q.eligible, nil
}
func (q *indexTestQueue) PublishEmbedding(context.Context, Lease, EmbeddingInput, ports.EmbeddingResponse) error {
	q.published++
	return nil
}
func (q *indexTestQueue) Fail(context.Context, Lease, string, time.Duration) error {
	q.failures++
	return nil
}
func (q *indexTestQueue) Complete(context.Context, Lease) error { q.completed++; return nil }
func TestEmbeddingWorkerRetriesWithoutMemoryOrWriterWork(t *testing.T) {
	a := assets(t)
	snap, meta, _, now := fixture()
	repo := &invariantRepo{snapshot: snap}
	embed := &invariantEmbed{fail: true}
	s, _ := New(repo, a, Options{Policy: testPolicy{}, Cipher: invariantCipher{}, Embedder: embed, EmbeddingModel: "synthetic", Dimension: 2, Now: func() time.Time { return now }})
	q := &indexTestQueue{lease: Lease{ID: c.NewID(), UserID: meta.UserID, Token: c.NewID(), Kind: "memory_embed"}, input: EmbeddingInput{Text: "گفتگوی ساختگی درباره کار", SourceVersion: "0000000000000001", UserVersion: 1}, eligible: true}
	writer := &inspectorMemoryWriter{}
	w, e := NewWorker(q, repo, writer, s, "index-test")
	if e != nil {
		t.Fatal(e)
	}
	if worked, e := w.RunOnce(context.Background()); !worked || e == nil {
		t.Fatal("outage did not retry")
	}
	embed.fail = false
	if worked, e := w.RunOnce(context.Background()); !worked || e != nil {
		t.Fatal(worked, e)
	}
	if q.failures != 1 || q.published != 1 || q.backfills != 1 || embed.calls != 2 || repo.saves != 0 || writer.calls != 0 {
		t.Fatal("unexpected index recovery", q, embed.calls)
	}
	q.eligible = false
	if _, e = w.RunOnce(context.Background()); e != nil || embed.calls != 2 || q.failures != 2 {
		t.Fatal("stale/deleted source disclosed", e)
	}
}

func (r *invariantRepo) Selection(context.Context, c.ID, c.ID, c.ID) (Snapshot, error) {
	return r.snapshot, nil
}
func TestMemoryJobPlanningIgnoresLegacyPauseAndConsent(t *testing.T) {
	a := assets(t)
	snap, meta, op, now := fixture()
	snap.User.MemoryPaused = true
	snap.User.Consent = map[string]bool{"memory": false}
	repo := &invariantRepo{snapshot: snap}
	s, err := New(repo, a, Options{Policy: testPolicy{}, Cipher: invariantCipher{}, EmbeddingModel: "synthetic", Dimension: 2, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	admission := c.Admission{UserID: meta.UserID, TurnID: meta.TurnID, UserMessageID: op.EvidenceMsg[0]}
	job, err := s.PlanTurnJob(context.Background(), admission, c.UnifiedTurn{})
	if err != nil || job == nil || job.Kind != "memory_write" {
		t.Fatalf("legacy controls blocked background memory: %+v %v", job, err)
	}
}

func TestUnifiedMemorySchedulingDoesNotDependOnSignals(t *testing.T) {
	a := assets(t)
	snap, meta, op, now := fixture()
	repo := &invariantRepo{snapshot: snap}
	s, err := New(repo, a, Options{Policy: testPolicy{}, Cipher: invariantCipher{}, EmbeddingModel: "synthetic", Dimension: 2, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	admission := c.Admission{UserID: meta.UserID, TurnID: meta.TurnID, UserMessageID: op.EvidenceMsg[0]}
	out, err := s.PlanTurnJob(context.Background(), admission, c.UnifiedTurn{Conversation: c.ConversationIntent{Intent: c.IntentDiscloseEvent}})
	if err != nil || out == nil || out.Kind != "memory_write" {
		t.Fatal("omitted model hints lost independent extraction", out, err)
	}
}
