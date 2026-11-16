package application

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"testing"
)

type stubRepo struct {
	uid, mid               uuid.UUID
	saveCalls, removeCalls int
	result                 error
}

func (r *stubRepo) List(_ context.Context, uid uuid.UUID) ([]SavedMessage, error) {
	r.uid = uid
	return []SavedMessage{}, r.result
}
func (r *stubRepo) Save(_ context.Context, uid, id uuid.UUID) error {
	r.uid = uid
	r.mid = id
	r.saveCalls++
	return r.result
}
func (r *stubRepo) Remove(_ context.Context, uid, id uuid.UUID) error {
	r.uid = uid
	r.mid = id
	r.removeCalls++
	return r.result
}
func TestOwnerAndUUIDValidation(t *testing.T) {
	ctx := context.Background()
	uid, id := uuid.New(), uuid.New()
	r := &stubRepo{}
	s := New(r)
	if !errors.Is(s.Save(ctx, uid, uuid.Nil), httpx.ErrInvalid) {
		t.Fatal("invalid save accepted")
	}
	if !errors.Is(s.Remove(ctx, uuid.Nil, id), httpx.ErrInvalid) {
		t.Fatal("invalid remove accepted")
	}
	if r.saveCalls != 0 || r.removeCalls != 0 {
		t.Fatal("invalid ID reached repository")
	}
	if err := s.Save(ctx, uid, id); err != nil {
		t.Fatal(err)
	}
	if r.uid != uid || r.mid != id || r.saveCalls != 1 {
		t.Fatal("server user scope lost")
	}
	if err := s.Remove(ctx, uid, id); err != nil {
		t.Fatal(err)
	}
	if r.uid != uid || r.mid != id || r.removeCalls != 1 {
		t.Fatal("remove scope lost")
	}
	if _, err := s.List(ctx, uid); err != nil || r.uid != uid {
		t.Fatal("list not owner scoped", err)
	}
	r.result = httpx.ErrNotFound
	if !errors.Is(s.Save(ctx, uid, id), httpx.ErrNotFound) {
		t.Fatal("not found incorrectly converted")
	}
}
