package retrieval

import (
	"testing"
	"time"
)

// Reference moment: Tuesday 2026-11-17 12:00 in Asia/Tehran (08:30 UTC).
func tehranParser(t *testing.T) TemporalParser {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		t.Fatalf("load tz: %v", err)
	}
	return TemporalParser{Loc: loc, WeekStartsOnMonday: true, Now: time.Date(2026, 11, 17, 8, 30, 0, 0, time.UTC)}
}

func TestTemporalParserRelativeDays(t *testing.T) {
	p := tehranParser(t)
	loc := p.Loc
	cases := []struct {
		text     string
		wantFrom time.Time
		days     int
	}{
		{"I told you about my boss yesterday", time.Date(2026, 11, 16, 0, 0, 0, 0, loc), 1},
		{"دیروز با مادرم حرف زدم", time.Date(2026, 11, 16, 0, 0, 0, 0, loc), 1},
		{"what did I say 3 days ago about work", time.Date(2026, 11, 14, 0, 0, 0, 0, loc), 1},
		{"today I felt awful", time.Date(2026, 11, 17, 0, 0, 0, 0, loc), 1},
	}
	for _, tc := range cases {
		w, ok := p.Parse(tc.text)
		if !ok {
			t.Fatalf("%q: expected a resolved window", tc.text)
		}
		if !w.From.Equal(tc.wantFrom) {
			t.Fatalf("%q: from=%s want %s", tc.text, w.From, tc.wantFrom)
		}
		if got := int(w.To.Sub(w.From).Hours() / 24); got != tc.days {
			t.Fatalf("%q: window should span %d day(s), got %s..%s", tc.text, tc.days, w.From, w.To)
		}
	}
}

func TestTemporalParserWeekdays(t *testing.T) {
	p := tehranParser(t)
	loc := p.Loc
	// 2026-11-17 is a Tuesday. "last Tuesday" = 2026-11-10.
	w, ok := p.Parse("what did I tell you last Tuesday about my boss?")
	if !ok {
		t.Fatal("expected last Tuesday to resolve")
	}
	want := time.Date(2026, 11, 10, 0, 0, 0, 0, loc)
	if !w.From.Equal(want) {
		t.Fatalf("last tuesday: got %s want %s", w.From, want)
	}
	if _, ok := p.Parse("سه‌شنبه"); !ok {
		t.Fatal("expected Persian weekday to resolve")
	}
}

func TestTemporalParserWeeksAndMonths(t *testing.T) {
	p := tehranParser(t)
	loc := p.Loc
	w, ok := p.Parse("last week I was exhausted")
	if !ok {
		t.Fatal("expected last week to resolve")
	}
	// Reference week (Mon-start) is 2026-11-16 .. 2026-11-23, so last week is 11-09..11-16.
	if !w.From.Equal(time.Date(2026, 11, 9, 0, 0, 0, 0, loc)) {
		t.Fatalf("last week from=%s", w.From)
	}
	if got := w.To.Sub(w.From).Hours(); got != 168 {
		t.Fatalf("a week window must span 168h, got %v", got)
	}
	w, ok = p.Parse("we spoke on 3 November")
	if !ok {
		t.Fatal("expected explicit date to resolve")
	}
	if !w.From.Equal(time.Date(2026, 11, 3, 0, 0, 0, 0, loc)) {
		t.Fatalf("explicit date from=%s", w.From)
	}
}

func TestTemporalParserUnresolved(t *testing.T) {
	p := tehranParser(t)
	for _, text := range []string{"I feel terrible", "sometime back things were bad", ""} {
		if w, ok := p.Parse(text); ok {
			t.Fatalf("%q must stay unresolved (no semantic substitution), got %+v", text, w)
		}
	}
}

func TestRecallVerbs(t *testing.T) {
	positives := []string{
		"What exactly did I tell you last Tuesday?",
		"do you remember when I said that",
		"چی گفتم درباره کار",
		"یادت هست گفتم",
	}
	for _, p := range positives {
		if !HasRecallVerb(p) {
			t.Fatalf("expected recall verb in %q", p)
		}
	}
	for _, n := range []string{"I am tired today", "could we talk about my week"} {
		if HasRecallVerb(n) {
			t.Fatalf("unexpected recall verb in %q", n)
		}
	}
}

func TestRecallTermsDropStopWordsAndTimeWords(t *testing.T) {
	terms := RecallTerms("What exactly did I tell you last Tuesday about my boss and the promotion?")
	if len(terms) == 0 || len(terms) > 4 {
		t.Fatalf("recall terms must be bounded and non-empty, got %v", terms)
	}
	for _, banned := range []string{"tuesday", "last", "tell", "my", "about", "what"} {
		for _, term := range terms {
			if term == banned {
				t.Fatalf("stop word %q leaked into recall terms %v", banned, terms)
			}
		}
	}
	if terms[0] != "boss" {
		t.Fatalf("expected 'boss' first, got %v", terms)
	}
}

func TestDayBoundsAreLocalMidnight(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Tehran")
	from, to := dayBounds(time.Date(2026, 11, 17, 5, 0, 0, 0, time.UTC), loc)
	if from.Hour() != 0 || from.Location() != loc {
		t.Fatalf("from must be local midnight, got %s", from)
	}
	if to.Sub(from) != 24*time.Hour {
		t.Fatalf("day window must be 24h, got %v", to.Sub(from))
	}
}
