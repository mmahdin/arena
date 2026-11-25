package application

import (
	"context"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/domain"
	"time"
)

type Repository interface {
	Get(ctx context.Context, userID uuid.UUID) (domain.Profile, error)
	Save(ctx context.Context, p domain.Profile) error
	SaveOnboardingName(ctx context.Context, userID uuid.UUID, name string) error
}
type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Get(ctx context.Context, uid uuid.UUID) (domain.Profile, error) {
	return s.repo.Get(ctx, uid)
}
func (s *Service) SaveOnboardingName(ctx context.Context, uid uuid.UUID, name string) (domain.Profile, error) {
	p, err := s.repo.Get(ctx, uid)
	if err != nil {
		return domain.Profile{}, err
	}
	if err := p.SetOnboardingName(name); err != nil {
		return domain.Profile{}, err
	}
	if err := s.repo.SaveOnboardingName(ctx, uid, p.DisplayName); err != nil {
		return domain.Profile{}, err
	}
	return s.repo.Get(ctx, uid)
}

func (s *Service) Update(ctx context.Context, uid uuid.UUID, name string, style domain.Style) (domain.Profile, error) {
	return s.UpdateWithOnboarding(ctx, uid, name, style, nil)
}

func (s *Service) UpdateWithOnboarding(ctx context.Context, uid uuid.UUID, name string, style domain.Style, answers *domain.OnboardingAnswers) (domain.Profile, error) {
	if answers != nil {
		if err := answers.Validate(); err != nil {
			return domain.Profile{}, err
		}
	}
	p, err := s.repo.Get(ctx, uid)
	if err != nil {
		return p, err
	}
	if err := p.Update(name, style, time.Now()); err != nil {
		return p, err
	}
	// nil means preserve the database value, including a concurrent answer save.
	p.OnboardingAnswers = answers
	if err := s.repo.Save(ctx, p); err != nil {
		return domain.Profile{}, err
	}
	return s.repo.Get(ctx, uid)
}
