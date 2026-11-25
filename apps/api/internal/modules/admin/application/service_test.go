package application

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

func TestMeRequiresVerifiedNonExpiredStaff(t *testing.T) {
	service := New()
	for _, p := range []domain.Principal{
		{},
		{StaffID: uuid.New()},
		{StaffID: uuid.New(), ExpiresAt: time.Now().Add(-time.Minute)},
	} {
		if _, err := service.Me(p); !errors.Is(err, ErrUnverifiedStaff) {
			t.Fatalf("unverified principal accepted: %+v (%v)", p, err)
		}
	}
	id := uuid.New()
	expires := time.Now().Add(time.Hour)
	got, err := service.Me(domain.Principal{StaffID: id, ExpiresAt: expires})
	if err != nil || got.StaffID != id || got.Role != "admin" || !got.Active || !got.ExpiresAt.Equal(expires.UTC()) {
		t.Fatalf("unexpected staff result: %+v %v", got, err)
	}
}
