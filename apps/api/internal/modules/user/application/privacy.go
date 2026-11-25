package application

import (
	"context"
	"github.com/google/uuid"
)

// PrivacyPreference represents an internal UI setting. It does not transmit conversations.
type PrivacyPreference struct {
	ConversationSharingEnabled bool `json:"conversationSharingEnabled"`
}

type PrivacyRepository interface {
	GetPrivacy(ctx context.Context, userID uuid.UUID) (PrivacyPreference, error)
	SetPrivacy(ctx context.Context, userID uuid.UUID, enabled bool) (PrivacyPreference, error)
}

type PrivacyService struct{ repo PrivacyRepository }

func NewPrivacyService(repo PrivacyRepository) *PrivacyService { return &PrivacyService{repo: repo} }
func (s *PrivacyService) Get(ctx context.Context, userID uuid.UUID) (PrivacyPreference, error) {
	return s.repo.GetPrivacy(ctx, userID)
}
func (s *PrivacyService) Set(ctx context.Context, userID uuid.UUID, enabled bool) (PrivacyPreference, error) {
	return s.repo.SetPrivacy(ctx, userID, enabled)
}
