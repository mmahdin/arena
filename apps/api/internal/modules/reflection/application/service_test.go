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
	entries map[uuid.UUID]map[string]Reflection
}

func (f *fakeRepo) List(_ context.Context, uid uuid.UUID) ([]Reflection, error) {
	result := []Reflection{}
	for _, e := range f.entries[uid] {
		result = append(result, e)
	}
	return result, nil
}
func (f *fakeRepo) Get(_ context.Context, uid uuid.UUID, date string) (Reflection, error) {
	if e, ok := f.entries[uid][date]; ok {
		return e, nil
	}
	return Reflection{}, httpx.ErrNotFound
}
func (f *fakeRepo) Save(_ context.Context, uid uuid.UUID, date, body string) (Reflection, error) {
	if f.entries[uid] == nil {
		f.entries[uid] = map[string]Reflection{}
	}
	e := Reflection{Date: date, Text: body, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.entries[uid][date] = e
	return e, nil
}
func (f *fakeRepo) Delete(_ context.Context, uid uuid.UUID, date string) error {
	delete(f.entries[uid], date)
	return nil
}

func TestReflectionValidationAndOwnership(t *testing.T) {
	for _, date := range []string{"2026-02-30", "2026-13-01", "2026-2-01", "2026-02-01/other", ""} {
		if ValidDate(date) {
			t.Fatalf("invalid date accepted: %q", date)
		}
	}
	if !ValidDate("2026-02-28") {
		t.Fatal("valid date rejected")
	}
	for _, s := range []string{" ", strings.Repeat("x", 4001), "synthetic\x00input", "unsafe\u202e"} {
		if _, ok := CleanText(s); ok {
			t.Fatal("invalid content accepted")
		}
	}
	if text, ok := CleanText("   نمونه ساختگی\nروز خوب  "); !ok || text != "نمونه ساختگی\nروز خوب" {
		t.Fatal("trim failed", text)
	}
	if _, ok := CleanText("نسخهٔ ویرایش‌شده با نیم\u200cفاصله و پیوند\u200dدهنده"); !ok {
		t.Fatal("Persian joiners rejected")
	}
	f := &fakeRepo{entries: make(map[uuid.UUID]map[string]Reflection)}
	s := New(f)
	a, b := uuid.New(), uuid.New()
	item, err := s.Save(context.Background(), a, "2026-11-02", "یادداشت ساختگی")
	if err != nil || item.Text != "یادداشت ساختگی" {
		t.Fatal(item, err)
	}
	if _, err := s.Get(context.Background(), b, "2026-11-02"); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("foreign read allowed", err)
	}
	if entries, err := s.List(context.Background(), b); err != nil || len(entries) != 0 {
		t.Fatal("foreign list leaked", entries, err)
	}
	if err := s.Delete(context.Background(), b, "2026-11-02"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(context.Background(), a, "2026-11-02"); err != nil {
		t.Fatal("wrong-owner delete changed data", err)
	}
	if _, err := s.Save(context.Background(), uuid.Nil, "2026-11-02", "x"); !errors.Is(err, httpx.ErrInvalid) {
		t.Fatal("missing owner accepted", err)
	}
	if _, err := s.Save(context.Background(), a, "2026-11-02", "   "); !errors.Is(err, httpx.ErrInvalid) {
		t.Fatal("blank accepted", err)
	}
}
