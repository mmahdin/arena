package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestProductMaintenanceCannotDispatchWakeups(t *testing.T) {
	r := &hardeningRepository{}
	original := hardeningScheduler(t, r, RunOptions{})
	s := original.WithoutDispatch()
	if n, e := s.Pulse(context.Background()); e != nil || n != 0 || r.sweepCalls != 1 || r.dueCalls != 0 {
		t.Fatal("proactive dispatched", n, e)
	}
	if _, e := s.Due(context.Background(), time.Now(), 10); e == nil {
		t.Fatal("direct due enabled")
	}
	if _, e := original.Pulse(context.Background()); e != nil || r.dueCalls != 1 {
		t.Fatal("original service mutated")
	}
}
