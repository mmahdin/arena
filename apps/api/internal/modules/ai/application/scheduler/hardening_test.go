package scheduler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type hardeningRepository struct {
	mu           sync.Mutex
	sweepCalls   int
	dueCalls     int
	failSweepFor int
	failDueFor   int
	rows         []c.FollowUp
	onSweep      func()
}

var sensitiveSQLError = errors.New("confidential provider response and internal SQL details")

func (*hardeningRepository) ApplyTimers(context.Context, c.ID, []c.FollowUpOp, time.Time) error {
	return nil
}
func (r *hardeningRepository) Sweep(ctx context.Context, _ c.BundleRef, _ time.Time, limit int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if limit != 100 {
		return errors.New("unbounded sweep")
	}
	r.mu.Lock()
	r.sweepCalls++
	shouldFail := r.sweepCalls <= r.failSweepFor
	f := r.onSweep
	r.mu.Unlock()
	if f != nil {
		f()
	}
	if shouldFail {
		return sensitiveSQLError
	}
	return ctx.Err()
}
func (r *hardeningRepository) DueTimers(ctx context.Context, _ time.Time, limit int) ([]c.FollowUp, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit != 100 {
		return nil, errors.New("unbounded due batch")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dueCalls++
	if r.dueCalls <= r.failDueFor {
		return nil, sensitiveSQLError
	}
	return r.rows, nil
}
func hardeningScheduler(t *testing.T, r *hardeningRepository, opt RunOptions) *Service {
	t.Helper()
	s, err := NewWithOptions(r, c.BundleRef{Version: "synthetic"}, opt)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestPulseRunsDueAfterSweepAndPublishesOnlyInternalMetadata(t *testing.T) {
	r := &hardeningRepository{rows: []c.FollowUp{{ID: c.NewID()}, {ID: c.NewID()}}}
	s := hardeningScheduler(t, r, RunOptions{})
	count, err := s.Pulse(context.Background())
	if err != nil || count != 2 || r.sweepCalls != 1 || r.dueCalls != 1 {
		t.Fatalf("expected bounded sweep/due once: count=%d err=%v", count, err)
	}
	h := s.Health()
	if h.Cycles != 1 || h.EventsDispatched != 2 || h.ConsecutiveFailures != 0 || h.LastSuccessAt.IsZero() {
		t.Fatalf("bad internal dispatch health: %+v", h)
	}
}
func TestPulseTracksStageAndRecoveryWithoutLeakingError(t *testing.T) {
	r := &hardeningRepository{failSweepFor: 1, failDueFor: 1}
	s := hardeningScheduler(t, r, RunOptions{})
	for index, stage := range []string{"sweep", "due"} {
		if _, err := s.Pulse(context.Background()); !errors.Is(err, sensitiveSQLError) {
			t.Fatal(err)
		}
		h := s.Health()
		if h.LastFailureStage != stage || h.LastFailureCode != c.ErrInternal || h.ConsecutiveFailures != index+1 || h.Failures != uint64(index+1) {
			t.Fatalf("health failed at stage %s: %+v", stage, h)
		}
		if strings.Contains(fmt.Sprint(h), "confidential") {
			t.Fatal("raw SQL error leaked to metrics")
		}
	}
	if _, err := s.Pulse(context.Background()); err != nil {
		t.Fatal(err)
	}
	if h := s.Health(); h.ConsecutiveFailures != 0 || h.Failures != 2 || h.LastSuccessAt.IsZero() {
		t.Fatalf("did not recover after temporary errors: %+v", h)
	}
}
func TestRunReturnsSanitizedErrorAfterBoundedFailures(t *testing.T) {
	r := &hardeningRepository{failSweepFor: 100}
	s := hardeningScheduler(t, r, RunOptions{Interval: time.Millisecond, RetryBase: time.Millisecond, RetryMax: 4 * time.Millisecond, MaxConsecutiveFailures: 3})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := s.Run(ctx)
	if !errors.Is(err, ErrUnhealthy) || strings.Contains(err.Error(), "confidential") {
		t.Fatalf("persistent error not surfaced safely: %v", err)
	}
	h := s.Health()
	if h.Cycles != 3 || h.ConsecutiveFailures != 3 || h.Failures != 3 || h.LastFailureStage != "sweep" || r.dueCalls != 0 {
		t.Fatalf("unbounded retry or wrong failure stage: %+v", h)
	}
}
func TestRunGracefulCancellationNotCountedAsFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &hardeningRepository{onSweep: cancel}
	s := hardeningScheduler(t, r, RunOptions{})
	if err := s.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if h := s.Health(); h.Failures != 0 {
		t.Fatalf("shutdown treated as operational failure: %+v", h)
	}
}
func TestSchedulerRunOptionsFailClosed(t *testing.T) {
	for _, options := range []RunOptions{
		{RetryBase: -time.Millisecond},
		{RetryBase: time.Second, RetryMax: time.Millisecond},
		{Interval: -time.Second},
		{MaxConsecutiveFailures: 101},
	} {
		if _, err := NewWithOptions(&hardeningRepository{}, c.BundleRef{Version: "synthetic"}, options); err == nil {
			t.Fatal("accepted invalid retry configuration")
		}
	}
}
