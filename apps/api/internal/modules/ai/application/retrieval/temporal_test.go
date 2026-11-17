package retrieval

import (
	"testing"
	"time"
)

// Tuesday 10 March 2026, 12:00 local. Fixed so every window assertion is exact.
func testNow(t *testing.T) (time.Time, *time.Location) {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("load test location: %v", err)
	}
	return time.Date(2026, 3, 10, 12, 0, 0, 0, loc), loc
}

func TestTemporalParserWindows(t *testing.T) {
	now, loc := testNow(t)
	parser := NewTemporalParser(time.Monday)
	cases := []struct {
		phrase   string
		from, to string
		ok       bool
	}{
		{"today", "2026-03-10T00:00", "2026-03-11T00:00", true},
		{"yesterday", "2026-03-09T00:00", "2026-03-10T00:00", true},
		{"last tuesday", "2026-03-03T00:00", "2026-03-04T00:00", true},
		{"last monday", "2026-03-09T00:00", "2026-03-10T00:00", true},
		{"last friday", "2026-03-06T00:00", "2026-03-07T00:00", true},
		{"sunday", "2026-03-08T00:00", "2026-03-09T00:00", true},
		{"last week", "2026-03-02T00:00", "2026-03-09T00:00", true},
		{"this week", "2026-03-09T00:00", "2026-03-16T00:00", true},
		{"last month", "2026-02-01T00:00", "2026-03-01T00:00", true},
		{"this month", "2026-03-01T00:00", "2026-04-01T00:00", true},
		{"last year", "2025-01-01T00:00", "2026-01-01T00:00", true},
		{"3 days ago", "2026-03-07T00:00", "2026-03-08T00:00", true},
		{"two weeks ago", "2026-02-24T00:00", "2026-03-03T00:00", true},
		{"on 3 March", "2026-03-03T00:00", "2026-03-04T00:00", true},
		{"march 3", "2026-03-03T00:00", "2026-03-04T00:00", true},
		{"2026-03-03", "2026-03-03T00:00", "2026-03-04T00:00", true},
		{"last night", "2026-03-09T18:00", "2026-03-10T06:00", true},
		{"this morning", "2026-03-10T04:00", "2026-03-10T12:00", true},
		// Unsupported or future expressions must not be guessed.
		{"sometime", "", "", false},
		{"the day my mother called", "", "", false},
		{"tomorrow", "", "", false},
		{"next tuesday", "", "", false},
		{"3 days hence", "", "", false},
		{"20 years ago", "", "", false}, // beyond the bounded grammar
		{"2 years ago", "2024-03-10T00:00", "2025-03-11T00:00", true},
		{"400 days ago", "", "", false}, // unbounded window rejected
	}
	for _, tc := range cases {
		t.Run(tc.phrase, func(t *testing.T) {
			m, ok := parser.Parse(tc.phrase, now, loc)
			if ok != tc.ok {
				t.Fatalf("ok=%v want %v (%+v)", ok, tc.ok, m)
			}
			if !ok {
				return
			}
			if got := m.From.In(loc).Format("2006-01-02T15:04"); got != tc.from {
				t.Fatalf("from=%s want %s", got, tc.from)
			}
			if got := m.To.In(loc).Format("2006-01-02T15:04"); got != tc.to {
				t.Fatalf("to=%s want %s", got, tc.to)
			}
			if !m.To.After(m.From) {
				t.Fatal("empty or inverted window")
			}
		})
	}
}

func TestTemporalParserWeekStartAndTimezone(t *testing.T) {
	// Same instant, two zones: the local day must follow the user's zone, not
	// the server's.
	berlin, _ := time.LoadLocation("Europe/Berlin")
	tehran, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	now := time.Date(2026, 3, 10, 0, 30, 0, 0, berlin)
	parser := NewTemporalParser(time.Monday)
	inBerlin, ok := parser.Parse("today", now, berlin)
	if !ok {
		t.Fatal("expected a match")
	}
	inTehran, ok := parser.Parse("today", now, tehran)
	if !ok {
		t.Fatal("expected a match")
	}
	// 00:30 Berlin is 03:00 Tehran, so both "today" windows start on the 10th
	// local but at different UTC instants (23:00Z vs 20:30Z).
	if inBerlin.From.Equal(inTehran.From) {
		t.Fatalf("timezone was ignored: %s", inBerlin.From)
	}
	if inBerlin.From.UTC().Hour() != 23 || inTehran.From.UTC().Format("15:04") != "20:30" {
		t.Fatalf("unexpected UTC bounds: %s %s", inBerlin.From.UTC(), inTehran.From.UTC())
	}
	// A Saturday week start moves "last week" by two days.
	saturday := NewTemporalParser(time.Saturday)
	lastWeek, ok := saturday.Parse("last week", now, berlin)
	if !ok {
		t.Fatal("expected a match")
	}
	mondayLastWeek, _ := parser.Parse("last week", now, berlin)
	if lastWeek.From.Equal(mondayLastWeek.From) {
		t.Fatal("week_starts_on was ignored")
	}
	if lastWeek.From.In(berlin).Weekday() != time.Saturday {
		t.Fatalf("expected Saturday week start, got %s", lastWeek.From.In(berlin).Weekday())
	}
}

func TestTemporalParserIsDeterministic(t *testing.T) {
	now, loc := testNow(t)
	parser := NewTemporalParser(time.Monday)
	for i := 0; i < 50; i++ {
		m, ok := parser.Parse("last tuesday", now, loc)
		if !ok || !m.From.Equal(time.Date(2026, 3, 3, 0, 0, 0, 0, loc)) {
			t.Fatalf("unstable parse at iteration %d: %+v", i, m)
		}
	}
}

func TestDetectTemporalPhrasePrefersLongestMatch(t *testing.T) {
	now, loc := testNow(t)
	parser := NewTemporalParser(time.Monday)
	m, ok := detectTemporalPhrase("What did I tell you last Tuesday about my boss?", parser, now, loc)
	if !ok {
		t.Fatal("expected a temporal phrase")
	}
	if m.Phrase != "last tuesday" {
		t.Fatalf("matched %q, want the longest phrase", m.Phrase)
	}
	if _, ok := detectTemporalPhrase("What did I tell you about my boss?", parser, now, loc); ok {
		t.Fatal("matched a phrase that is not present")
	}
}

func TestRecallVerbDetection(t *testing.T) {
	for _, text := range []string{
		"What did I tell you last Tuesday?",
		"when did I say that",
		"do you remember my boss",
		"I told you about my sister",
	} {
		if !hasRecallVerb(text) {
			t.Fatalf("%q should be a recall question", text)
		}
	}
	for _, text := range []string{
		"my boss was awful today",
		"I'm exhausted",
		"can you help me calm down",
	} {
		if hasRecallVerb(text) {
			t.Fatalf("%q is not a recall question", text)
		}
	}
}
