package llm

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type recordedJournal struct {
	mu                  sync.Mutex
	begins              []AttemptStart
	ends                []AttemptFinish
	beginErr, finishErr error
	finishWasCancelled  bool
}

func (j *recordedJournal) BeginAttempt(_ context.Context, a AttemptStart) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.begins = append(j.begins, a)
	return j.beginErr
}
func (j *recordedJournal) FinishAttempt(ctx context.Context, e AttemptFinish) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.ends = append(j.ends, e)
	j.finishWasCancelled = ctx.Err() != nil
	return j.finishErr
}

type attemptsProvider struct {
	calls        int
	result       StructuredResult
	err          error
	stream       ports.Stream
	onStructured func()
}

func (p *attemptsProvider) Structured(_ context.Context, _ Model, _ ports.StructuredRequest, _ Mode) (StructuredResult, error) {
	p.calls++
	if p.onStructured != nil {
		p.onStructured()
	}
	return p.result, p.err
}
func (p *attemptsProvider) Embed(_ context.Context, _ Model, _ ports.EmbeddingRequest) (EmbeddingResult, error) {
	p.calls++
	return EmbeddingResult{Usage: p.result.Usage}, p.err
}
func (p *attemptsProvider) Stream(_ context.Context, _ Model, _ ports.StreamRequest) (ports.Stream, error) {
	p.calls++
	return p.stream, p.err
}

type attemptsStream struct {
	i      int
	closed int
	chunks []ports.Chunk
	err    error
}

func (s *attemptsStream) Recv() (ports.Chunk, error) {
	if s.i < len(s.chunks) {
		c := s.chunks[s.i]
		s.i++
		return c, nil
	}
	if s.err != nil {
		return ports.Chunk{}, s.err
	}
	return ports.Chunk{}, io.EOF
}
func (s *attemptsStream) Close() error { s.closed++; return nil }

func TestDurableAttemptRecordsKnownUsageAndConfiguredPrice(t *testing.T) {
	j := &recordedJournal{}
	p := &attemptsProvider{result: StructuredResult{Usage: contract.LLMUsage{InTokens: 100, OutTokens: 20, CachedTokens: 20}}}
	m := Model{Provider: "provider", Name: "model-1", InputMicroUSDPerM: 2000000, OutputMicroUSDPerM: 4000000, CachedMicroUSDPerM: 500000}
	wrapped := meteredProvider{next: p, journal: j, now: time.Now}
	turn := contract.NewID()
	_, err := wrapped.Structured(context.Background(), m, ports.StructuredRequest{Tier: ports.ComponentBrain, Meta: contract.CallMeta{TurnID: turn}}, ModeNativeSchema)
	if err != nil || p.calls != 1 || len(j.begins) != 1 || len(j.ends) != 1 {
		t.Fatalf("calls=%d start=%d end=%d err=%v", p.calls, len(j.begins), len(j.ends), err)
	}
	if j.begins[0].TurnID != turn || j.begins[0].ID != j.ends[0].ID || j.ends[0].Status != "succeeded" || !j.ends[0].UsageKnown || j.ends[0].CostStatus != "estimated" || j.ends[0].CostMicroUSD != 250 {
		t.Fatalf("unexpected metadata: %+v %+v", j.begins[0], j.ends[0])
	}
}

func TestDurableAttemptFailureAndUnknownPriceAreNotZeroCharges(t *testing.T) {
	j := &recordedJournal{}
	p := &attemptsProvider{err: &Failure{Kind: FailureRateLimit, Reason: "private vendor error"}}
	w := meteredProvider{next: p, journal: j, now: time.Now}
	_, err := w.Structured(context.Background(), Model{Provider: "p", Name: "m"}, ports.StructuredRequest{Tier: ports.ComponentBrain}, ModeNativeSchema)
	if err == nil || len(j.ends) != 1 || j.ends[0].CostStatus != "unknown" || j.ends[0].UsageKnown || j.ends[0].Failure != FailureRateLimit {
		t.Fatalf("failed attempt should remain unknown: %+v", j.ends)
	}
	j = &recordedJournal{}
	p.err = nil
	p.result = StructuredResult{Usage: contract.LLMUsage{InTokens: 1}}
	w.journal = j
	if _, err = w.Structured(context.Background(), Model{Provider: "p", Name: "m"}, ports.StructuredRequest{Tier: ports.ComponentBrain}, ModeNativeSchema); err != nil || j.ends[0].CostStatus != "unpriced" {
		t.Fatalf("unpriced provider incorrectly billed: %+v err=%v", j.ends, err)
	}
}

func TestAttemptJournalFailureStopsProviderAndIsSanitized(t *testing.T) {
	j := &recordedJournal{beginErr: errors.New("private SQL failure")}
	p := &attemptsProvider{}
	w := meteredProvider{next: p, journal: j, now: time.Now}
	_, err := w.Structured(context.Background(), Model{Provider: "p", Name: "m"}, ports.StructuredRequest{Tier: ports.ComponentBrain}, ModeNativeSchema)
	if !errors.Is(err, ErrTelemetry) || p.calls != 0 || err.Error() == "private SQL failure" {
		t.Fatalf("not fail closed: calls=%d err=%v", p.calls, err)
	}
	j.beginErr = nil
	j.finishErr = errors.New("private SQL failure")
	_, err = w.Structured(context.Background(), Model{Provider: "p", Name: "m"}, ports.StructuredRequest{Tier: ports.ComponentBrain}, ModeNativeSchema)
	if !errors.Is(err, ErrTelemetry) || p.calls != 1 {
		t.Fatalf("missing durable finish must not appear successful: err=%v", err)
	}
}

func TestStreamCompletionAndEarlyClose(t *testing.T) {
	usage := contract.LLMUsage{InTokens: 10, OutTokens: 4}
	j := &recordedJournal{}
	p := &attemptsProvider{stream: &attemptsStream{chunks: []ports.Chunk{{Text: "example"}, {Usage: &usage}, {FinishReason: "stop"}}}}
	w := meteredProvider{next: p, journal: j, now: time.Now}
	stream, err := w.Stream(context.Background(), Model{Provider: "p", Name: "m", InputMicroUSDPerM: 1000000}, ports.StreamRequest{Tier: ports.ComponentGenerator})
	if err != nil {
		t.Fatal(err)
	}
	for {
		_, err = stream.Recv()
		if err != nil {
			break
		}
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("stream err=%v", err)
	}
	_ = stream.Close()
	if len(j.ends) != 1 || j.ends[0].Status != "succeeded" || !j.ends[0].UsageKnown {
		t.Fatalf("stream accounting: %+v", j.ends)
	}
	j = &recordedJournal{}
	p.stream = &attemptsStream{chunks: []ports.Chunk{{Text: "partial"}}}
	w.journal = j
	stream, err = w.Stream(context.Background(), Model{Provider: "p", Name: "m"}, ports.StreamRequest{Tier: ports.ComponentGenerator})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = stream.Recv()
	_ = stream.Close()
	if len(j.ends) != 1 || j.ends[0].Status != "incomplete" || j.ends[0].CostStatus != "unknown" {
		t.Fatalf("early close incorrectly billed: %+v", j.ends)
	}
}

func TestCancelledContextStillFinalizesProviderUsage(t *testing.T) {
	j := &recordedJournal{}
	ctx, cancel := context.WithCancel(context.Background())
	p := &attemptsProvider{result: StructuredResult{Usage: contract.LLMUsage{InTokens: 1}}, onStructured: cancel}
	w := meteredProvider{next: p, journal: j, now: time.Now}
	_, err := w.Structured(ctx, Model{Provider: "p", Name: "m"}, ports.StructuredRequest{Tier: ports.ComponentBrain}, ModeNativeSchema)
	if err != nil || len(j.ends) != 1 || j.finishWasCancelled {
		t.Fatalf("completion lost after cancellation: %+v err=%v", j.ends, err)
	}
}
