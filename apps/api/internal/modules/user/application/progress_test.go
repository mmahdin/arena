package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeProgress struct {
	step     int
	complete bool
	calls    int
}

func (f *fakeProgress) GetStep(_ context.Context, _ uuid.UUID) (int, bool, error) {
	return f.step, f.complete, nil
}
func (f *fakeProgress) AdvanceStep(_ context.Context, _ uuid.UUID, wanted int) (int, bool, error) {
	f.calls++
	if wanted > f.step+1 {
		return f.step, false, ErrProgressConflict
	}
	if !f.complete && wanted > f.step {
		f.step = wanted
	}
	return f.step, f.complete, nil
}
func TestProgressStepValidationAndMonotonicity(t *testing.T) {
	repo := &fakeProgress{}
	s := NewProgressService(repo)
	id := uuid.New()
	ctx := context.Background()
	for _, n := range []int{-1, 26, 999} {
		if _, err := s.Advance(ctx, id, n); !errors.Is(err, ErrInvalidProgress) {
			t.Fatal(n, err)
		}
	}
	if repo.calls != 0 {
		t.Fatal("invalid step reached repository")
	}
	if _, err := s.Advance(ctx, id, 2); !errors.Is(err, ErrProgressConflict) {
		t.Fatal("accepted deep link", err)
	}
	for i := 1; i <= 25; i++ {
		current, err := s.Advance(ctx, id, i)
		if err != nil || current.FurthestStep != i || current.FlowVersion != "hamrah-v2" {
			t.Fatal(current, err)
		}
	}
	current, err := s.Advance(ctx, id, 5)
	if err != nil || current.FurthestStep != 25 {
		t.Fatal("regressed", current, err)
	}
	repo.complete = true
	if _, err := s.Advance(ctx, id, 25); !errors.Is(err, ErrProgressCompleted) {
		t.Fatal("completed profile advanced", err)
	}
}
