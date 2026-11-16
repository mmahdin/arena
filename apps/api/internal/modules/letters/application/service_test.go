package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type fakeRepo struct {
	owner, conversationID uuid.UUID
	saved                 map[uuid.UUID]Letter
	saveCount             int
	sourceErr             error
}

func (f *fakeRepo) List(_ context.Context, user uuid.UUID) ([]Letter, error) {
	out := []Letter{}
	if user == f.owner {
		for _, l := range f.saved {
			out = append(out, l)
		}
	}
	return out, nil
}
func (f *fakeRepo) LatestEligible(_ context.Context, user uuid.UUID) (Source, error) {
	if f.sourceErr != nil {
		return Source{}, f.sourceErr
	}
	if user != f.owner {
		return Source{}, ErrInsufficientContext
	}
	return Source{ConversationID: f.conversationID, Turns: []Turn{{Role: "user", Content: "fake topic"}, {Role: "assistant", Content: "mock"}, {Role: "user", Content: "fake topic two"}, {Role: "user", Content: "fake topic three"}}}, nil
}
func (f *fakeRepo) BySource(_ context.Context, user, conv uuid.UUID) (Letter, error) {
	if user != f.owner || conv != f.conversationID {
		return Letter{}, httpx.ErrNotFound
	}
	if item, ok := f.saved[conv]; ok {
		return item, nil
	}
	return Letter{}, httpx.ErrNotFound
}
func (f *fakeRepo) Save(_ context.Context, user, conv uuid.UUID, draft Draft) (Letter, error) {
	if user != f.owner || conv != f.conversationID {
		return Letter{}, httpx.ErrNotFound
	}
	f.saveCount++
	item := Letter{ID: uuid.New(), Kind: "personalized", Title: draft.Title, Salutation: draft.Salutation, Paragraphs: draft.Paragraphs, Question: draft.Question, SourceConversationIDs: []uuid.UUID{conv}, CreatedAt: time.Now()}
	f.saved[conv] = item
	return item, nil
}
func (f *fakeRepo) Delete(_ context.Context, user, id uuid.UUID) error {
	if user != f.owner {
		return httpx.ErrNotFound
	}
	for source, item := range f.saved {
		if item.ID == id {
			delete(f.saved, source)
			return nil
		}
	}
	return httpx.ErrNotFound
}

type fakeGenerator struct {
	calls   int
	invalid bool
}

func (g *fakeGenerator) Generate(context.Context, []Turn) (Draft, error) {
	g.calls++
	if g.invalid {
		return Draft{Title: "  "}, nil
	}
	return Draft{Title: "A synthetic note", Paragraphs: []string{"Synthetic observation one", "Synthetic observation two"}}, nil
}
func TestLetterGenerationRequiresConsentOwnerAndContext(t *testing.T) {
	owner, foreign, conv := uuid.New(), uuid.New(), uuid.New()
	repo := &fakeRepo{owner: owner, conversationID: conv, saved: map[uuid.UUID]Letter{}}
	gen := &fakeGenerator{}
	service := New(repo, gen)
	if _, err := service.Generate(context.Background(), owner, false); !errors.Is(err, httpx.ErrInvalid) {
		t.Fatalf("expected explicit consent validation: %v", err)
	}
	if _, err := service.Generate(context.Background(), foreign, true); !errors.Is(err, ErrInsufficientContext) {
		t.Fatalf("expected ownership isolation: %v", err)
	}
	if len(repo.saved) != 0 || gen.calls != 0 {
		t.Fatal("generation attempted without consent or owner")
	}
	first, err := service.Generate(context.Background(), owner, true)
	if err != nil || first.Kind != "personalized" || len(first.SourceConversationIDs) != 1 || first.SourceConversationIDs[0] != conv {
		t.Fatalf("invalid generated letter %+v: %v", first, err)
	}
	second, err := service.Generate(context.Background(), owner, true)
	if err != nil || first.ID != second.ID || gen.calls != 1 || repo.saveCount != 1 {
		t.Fatalf("duplicate triggered model call or insert: %v %+v", err, second)
	}
	if foreignList, err := service.List(context.Background(), foreign); err != nil || len(foreignList) != 0 {
		t.Fatalf("foreign letter leaked: %v %v", foreignList, err)
	}
	if err := service.Delete(context.Background(), foreign, first.ID); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatalf("foreign deletion permitted: %v", err)
	}
	if err := service.Delete(context.Background(), owner, first.ID); err != nil {
		t.Fatal(err)
	}
	if len(repo.saved) != 0 {
		t.Fatal("deletion failed")
	}
}
func TestLetterGenerationHasNoFakeFallback(t *testing.T) {
	owner, conv := uuid.New(), uuid.New()
	repo := &fakeRepo{owner: owner, conversationID: conv, saved: map[uuid.UUID]Letter{}}
	if _, err := New(repo, nil).Generate(context.Background(), owner, true); !errors.Is(err, ErrGenerationUnavailable) {
		t.Fatal(err)
	}
	invalid := &fakeGenerator{invalid: true}
	if _, err := New(repo, invalid).Generate(context.Background(), owner, true); !errors.Is(err, ErrGenerationUnavailable) {
		t.Fatal(err)
	}
	repo.sourceErr = ErrInsufficientContext
	if _, err := New(repo, invalid).Generate(context.Background(), owner, true); !errors.Is(err, ErrInsufficientContext) {
		t.Fatal(err)
	}
	if len(repo.saved) != 0 {
		t.Fatal("fabricated letter persisted")
	}
}
func TestDraftValidationBounds(t *testing.T) {
	good := Draft{Title: "واکنش", Paragraphs: []string{"یک متن آزمایشی", "متن دوم"}}
	if !ValidDraft(good) {
		t.Fatal("valid draft rejected")
	}
	for _, bad := range []Draft{
		{Title: "", Paragraphs: good.Paragraphs},
		{Title: "x", Paragraphs: []string{"only one paragraph"}},
		{Title: "x", Paragraphs: []string{strings.Repeat("x", 2001), "another"}},
		{Title: "x", Paragraphs: []string{"safe", "invalid\x00"}},
	} {
		if ValidDraft(bad) {
			t.Fatalf("invalid draft accepted %+v", bad)
		}
	}
}
