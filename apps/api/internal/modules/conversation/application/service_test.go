package application

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"testing"
)

type fakeRepository struct {
	created       int
	readErr       error
	claimed       int
	completed     int
	released      int
	previous      bool
	pending       bool
	leaseID       uuid.UUID
	completeLease uuid.UUID
	releaseLease  uuid.UUID
}

func (*fakeRepository) List(context.Context, uuid.UUID) ([]domain.Conversation, error) {
	return nil, nil
}
func (r *fakeRepository) Create(context.Context, uuid.UUID) (domain.Conversation, error) {
	r.created++
	return domain.Conversation{}, nil
}
func (r *fakeRepository) Messages(context.Context, uuid.UUID, uuid.UUID, int) ([]domain.Message, error) {
	if r.readErr != nil {
		return nil, r.readErr
	}
	return []domain.Message{}, nil
}
func (*fakeRepository) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (*fakeRepository) Rename(_ context.Context, _, convID uuid.UUID, title string) (domain.Conversation, error) {
	return domain.Conversation{ID: convID, Title: title}, nil
}
func (r *fakeRepository) ClaimTurn(_ context.Context, _, _, _ uuid.UUID, _ string) (ClaimedTurn, error) {
	r.claimed++
	if r.previous {
		return ClaimedTurn{Existing: true, Reply: "old reply", PromptVersion: "old-v1"}, nil
	}
	if r.pending {
		return ClaimedTurn{Pending: true}, nil
	}
	if r.leaseID == uuid.Nil {
		r.leaseID = uuid.New()
	}
	return ClaimedTurn{LeaseID: r.leaseID}, nil
}
func (r *fakeRepository) CompleteTurn(_ context.Context, _, _, _, leaseID uuid.UUID, _, reply, version string) (TurnResult, error) {
	r.completeLease = leaseID
	r.completed++
	return TurnResult{Reply: reply, PromptVersion: version}, nil
}
func (r *fakeRepository) ReleaseTurn(_ context.Context, _, _, _, leaseID uuid.UUID) error {
	r.releaseLease = leaseID
	r.released++
	return nil
}

type fakeProfiles struct{ incomplete bool }

func (f fakeProfiles) ForUser(context.Context, uuid.UUID) (Profile, error) {
	return Profile{DisplayName: "Test", ConversationStyle: "balanced", OnboardingComplete: !f.incomplete}, nil
}

type fakeAI struct{ err error }

func (a fakeAI) Reply(context.Context, Profile, []domain.Message, string) (string, string, error) {
	if a.err != nil {
		return "", "", a.err
	}
	return "mock reply", "supportive-v1", nil
}
func TestTurnCompletes(t *testing.T) {
	r := &fakeRepository{}
	s := New(r, fakeProfiles{}, fakeAI{}, 20)
	res, err := s.Turn(context.Background(), uuid.New(), uuid.New(), uuid.New(), "hello")
	if err != nil || res.Reply != "mock reply" || r.claimed != 1 || r.completed != 1 || r.released != 0 || r.completeLease != r.leaseID {
		t.Fatalf("unexpected result: %+v err=%v", res, err)
	}
}
func TestProviderFailureReleasesReservation(t *testing.T) {
	r := &fakeRepository{}
	s := New(r, fakeProfiles{}, fakeAI{err: errors.New("synthetic failure")}, 20)
	_, err := s.Turn(context.Background(), uuid.New(), uuid.New(), uuid.New(), "hello")
	if !errors.Is(err, httpx.ErrUnavailable) || r.released != 1 || r.completed != 0 || r.releaseLease != r.leaseID {
		t.Fatalf("unexpected error=%v released=%d", err, r.released)
	}
}
func TestDuplicateReturnsStoredVersionWithoutGeneration(t *testing.T) {
	r := &fakeRepository{previous: true}
	s := New(r, fakeProfiles{}, fakeAI{err: errors.New("should not be called")}, 20)
	res, err := s.Turn(context.Background(), uuid.New(), uuid.New(), uuid.New(), "hello")
	if err != nil || res.Reply != "old reply" || res.PromptVersion != "old-v1" || r.completed != 0 {
		t.Fatalf("unexpected: %+v %v", res, err)
	}
}

func TestOnboardingRequiredBeforeCreatingConversation(t *testing.T) {
	r := &fakeRepository{}
	s := New(r, fakeProfiles{incomplete: true}, fakeAI{}, 20)
	_, err := s.Create(context.Background(), uuid.New())
	if !errors.Is(err, httpx.ErrOnboardingRequired) || r.created != 0 {
		t.Fatalf("created conversation before onboarding: err=%v count=%d", err, r.created)
	}
	status, code := httpx.ErrStatus(err)
	if status != 409 || code != "ONBOARDING_REQUIRED" {
		t.Fatalf("unexpected error contract: %d %s", status, code)
	}
	if _, err := New(r, fakeProfiles{}, fakeAI{}, 20).Create(context.Background(), uuid.New()); err != nil || r.created != 1 {
		t.Fatalf("completed profile cannot create conversation: %v", err)
	}
}

func TestOnboardingRequiredBeforeTurnWithoutClaiming(t *testing.T) {
	r := &fakeRepository{}
	s := New(r, fakeProfiles{incomplete: true}, fakeAI{}, 20)
	_, err := s.Turn(context.Background(), uuid.New(), uuid.New(), uuid.New(), "synthetic")
	if !errors.Is(err, httpx.ErrOnboardingRequired) || r.claimed != 0 || r.completed != 0 {
		t.Fatalf("unexpected turn: %v claimed=%d completed=%d", err, r.claimed, r.completed)
	}
	r.readErr = httpx.ErrNotFound // preserve non-enumerating 404 for foreign IDs
	_, err = s.Turn(context.Background(), uuid.New(), uuid.New(), uuid.New(), "synthetic")
	if !errors.Is(err, httpx.ErrNotFound) || r.claimed != 0 {
		t.Fatalf("foreign conversation disclosed or claimed: %v", err)
	}
}

func TestPendingTurnMustNotGenerateOrRelease(t *testing.T) {
	r := &fakeRepository{pending: true}
	s := New(r, fakeProfiles{}, fakeAI{err: errors.New("must not call generator")}, 20)
	_, err := s.Turn(context.Background(), uuid.New(), uuid.New(), uuid.New(), "synthetic")
	if !errors.Is(err, httpx.ErrConflict) || r.completed != 0 || r.released != 0 {
		t.Fatalf("pending retry should be a stable conflict: %v completed=%d released=%d", err, r.completed, r.released)
	}
}
func TestMalformedLeaseFailsClosedWithoutInvokingGenerator(t *testing.T) {
	r := &fakeRepository{previous: false}
	r.leaseID = uuid.Nil
	// A repository response without a lease is not sufficient to own a turn.
	// The fake uses a generated lease for happy-path tests; a dedicated fake
	// must return an empty one here.
	s := New(&noLeaseRepository{fakeRepository: r}, fakeProfiles{}, fakeAI{err: errors.New("must not call generator")}, 20)
	_, err := s.Turn(context.Background(), uuid.New(), uuid.New(), uuid.New(), "synthetic")
	if !errors.Is(err, httpx.ErrUnavailable) || r.completed != 0 || r.released != 0 {
		t.Fatalf("missing lease not rejected: %v", err)
	}
}

type noLeaseRepository struct{ *fakeRepository }

func (*noLeaseRepository) ClaimTurn(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (ClaimedTurn, error) {
	return ClaimedTurn{}, nil
}

func TestRenameValidatesInputAndPreservesTitle(t *testing.T) {
	s := New(&fakeRepository{}, fakeProfiles{}, fakeAI{}, 20)
	uid, convID := uuid.New(), uuid.New()
	if _, err := s.Rename(context.Background(), uid, convID, "	 "); !errors.Is(err, httpx.ErrInvalid) {
		t.Fatalf("blank title accepted: %v", err)
	}
	if _, err := s.Rename(context.Background(), uid, convID, string([]rune("عنوان")[:2])+"\n"); !errors.Is(err, httpx.ErrInvalid) {
		t.Fatalf("control character accepted: %v", err)
	}
	c, err := s.Rename(context.Background(), uid, convID, "  برنامه فردا  ")
	if err != nil || c.Title != "برنامه فردا" || c.ID != convID {
		t.Fatalf("rename failed: %+v %v", c, err)
	}
}
