package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Onboarding v2 has 26 pre-Home screens, indexed 0..25.
// Persist ONLY the furthest screen; never persist questionnaire answers here.
const LastOnboardingStep = 25

var ErrInvalidProgress = errors.New("invalid onboarding step")
var ErrProgressConflict = errors.New("onboarding steps must be visited in order")
var ErrProgressCompleted = errors.New("onboarding already completed")

type Progress struct {
	FurthestStep int    `json:"furthestStep"`
	FlowVersion  string `json:"flowVersion"`
}

type ProgressRepository interface {
	GetStep(ctx context.Context, user uuid.UUID) (step int, completed bool, err error)
	AdvanceStep(ctx context.Context, user uuid.UUID, step int) (current int, completed bool, err error)
}

type ProgressService struct{ repo ProgressRepository }

func NewProgressService(repo ProgressRepository) *ProgressService {
	return &ProgressService{repo: repo}
}
func progress(step int) Progress { return Progress{FurthestStep: step, FlowVersion: "hamrah-v2"} }
func (s *ProgressService) Get(ctx context.Context, id uuid.UUID) (Progress, error) {
	step, _, err := s.repo.GetStep(ctx, id)
	return progress(step), err
}
func (s *ProgressService) Advance(ctx context.Context, id uuid.UUID, step int) (Progress, error) {
	if step < 0 || step > LastOnboardingStep {
		return Progress{}, ErrInvalidProgress
	}
	current, completed, err := s.repo.AdvanceStep(ctx, id, step)
	if err != nil {
		return Progress{}, err
	}
	if completed {
		return Progress{}, ErrProgressCompleted
	}
	if current < step {
		return Progress{}, ErrProgressConflict
	}
	return progress(current), nil
}
