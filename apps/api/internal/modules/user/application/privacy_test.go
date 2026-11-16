package application

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
)

type privacyMemory struct{ values map[uuid.UUID]bool }

func (m *privacyMemory) GetPrivacy(_ context.Context, id uuid.UUID) (PrivacyPreference, error) {
	v, ok := m.values[id]
	if !ok {
		return PrivacyPreference{ConversationSharingEnabled: true}, nil
	}
	return PrivacyPreference{ConversationSharingEnabled: v}, nil
}
func (m *privacyMemory) SetPrivacy(_ context.Context, id uuid.UUID, enabled bool) (PrivacyPreference, error) {
	if id == uuid.Nil {
		return PrivacyPreference{}, errors.New("invalid owner")
	}
	m.values[id] = enabled
	return PrivacyPreference{ConversationSharingEnabled: enabled}, nil
}
func TestPrivacyPreferencePerOwnerAndPersistence(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	s := NewPrivacyService(&privacyMemory{values: map[uuid.UUID]bool{}})
	check := func(id uuid.UUID, expected bool) {
		t.Helper()
		p, err := s.Get(context.Background(), id)
		if err != nil || p.ConversationSharingEnabled != expected {
			t.Fatalf("privacy: %v %v", p, err)
		}
	}
	check(a, true)
	check(b, true)
	if _, err := s.Set(context.Background(), a, false); err != nil {
		t.Fatal(err)
	}
	check(a, false)
	check(b, true)
	if _, err := s.Set(context.Background(), a, true); err != nil {
		t.Fatal(err)
	}
	check(a, true)
}
