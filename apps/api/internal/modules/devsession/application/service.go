package application

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"time"
)

type TestUser struct {
	ID      uuid.UUID `json:"id"`
	TestKey string    `json:"testKey"`
}

type Repository interface {
	List(ctx context.Context) ([]TestUser, error)
	CreateSession(ctx context.Context, userID, sessionID uuid.UUID, expiresAt time.Time) (bool, error)
	RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error
	FindOrCreatePhoneSession(ctx context.Context, normalizedPhone string, sessionID uuid.UUID, expiresAt time.Time) (uuid.UUID, bool, error)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service                              { return &Service{repo: repo} }
func (s *Service) List(ctx context.Context) ([]TestUser, error) { return s.repo.List(ctx) }
func (s *Service) StartSession(ctx context.Context, userID, sessionID uuid.UUID, expiresAt time.Time) (bool, error) {
	return s.repo.CreateSession(ctx, userID, sessionID, expiresAt)
}
func (s *Service) EndSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	return s.repo.RevokeSession(ctx, userID, sessionID)
}

// StartPhoneSession is LOCAL-only identity selection. The caller must not treat a
// matching phone-shaped string as evidence of ownership.
func (s *Service) StartPhoneSession(ctx context.Context, phone string, sessionID uuid.UUID, expiresAt time.Time) (uuid.UUID, bool, error) {
	normalized, err := NormalizeSyntheticPhone(phone)
	if err != nil {
		return uuid.Nil, false, err
	}
	if s.repo == nil {
		return uuid.Nil, false, errors.New("session repository unavailable")
	}
	return s.repo.FindOrCreatePhoneSession(ctx, normalized, sessionID, expiresAt)
}
