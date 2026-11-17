package retrieval

import (
	"strings"
	"time"
)

// TemporalParser converts a natural-language time expression into a half-open
// [from,to) UTC window. It is deliberately small and deterministic: the
// architecture requires exact/temporal recall to be reproducible and testable,
// never delegated to a model. Locale patterns are engineering syntax, not
// clinical content, and are supplied per locale by the caller.
//
// Unsupported or ambiguous expressions return ok=false. The Service must then
// fall back to the hybrid strategy instead of inventing a window — "no semantic
// substitution" applies to an unmatched temporal query as well.
type TemporalParser struct {
	weekStart time.Weekday
}

func NewTemporalParser(weekStart time.Weekday) *TemporalParser {
	return &TemporalParser{weekStart: weekStart}
}

// temporalMatch is one parsed window plus the phrase that produced it.
type temporalMatch struct {
	Phrase  string
	From    time.Time
	To      time.Time
	Weekday bool // phrase names a day of the week (needed for weekday queries)
}

var englishMonths = map[string]time.Month{
	"january": time.January, "february": time.February, "march": time.March,
	"april": time.April, "may": time.May, "june": time.June, "july": time.July,
	"august": time.August, "september": time.September, "october": time.October,
	"november": time.November, "december": time.December,
}

var englishWeekdays = map[string]time.Weekday{
	"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
	"wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday,
	"saturday": time.Saturday,
}

// Parse returns the time window named by `phrase`, evaluated in `loc`.
// `now` is the caller-supplied clock; the parser never reads the wall clock.
func (p *TemporalParser) Parse(phrase string, now time.Time, loc *time.Location) (temporalMatch, bool) {
	if loc == nil {
		loc = time.UTC
	}
	now = now.In(loc)
	norm := strings.ToLower(strings.TrimSpace(phrase))
	norm = strings.Trim(norm, "?.!,;:\"'")
	norm = strings.Join(strings.Fields(norm), " ")
	if norm == "" {
		return temporalMatch{}, false
	}
	// The relative forms are evaluated first: "last tuesday" must not be read
	// as the weekday name alone.
	if wd, ok := p.relativeWeekday(norm); ok {
		return p.weekdayWindow(norm, wd, now, loc)
	}
	for name, wd := range englishWeekdays {
		if norm == name {
			return p.weekdayWindow(norm, wd, now, loc)
		}
	}
	switch norm {
	case "today":
		return dayWindow(norm, now, loc), true
	case "yesterday":
		return dayWindow(norm, now.AddDate(0, 0, -1), loc), true
	case "last night":
		return nightWindow(norm, now.AddDate(0, 0, -1), loc), true
	case "tonight", "this evening":
		return nightWindow(norm, now, loc), true
	case "this morning":
		return morningWindow(norm, now, loc), true
	case "last week":
		from := p.startOfWeek(now)
		return temporalMatch{Phrase: norm, From: from.AddDate(0, 0, -7), To: from}, true
	case "this week":
		from := p.startOfWeek(now)
		return temporalMatch{Phrase: norm, From: from, To: from.AddDate(0, 0, 7)}, true
	case "last month":
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		return temporalMatch{Phrase: norm, From: first.AddDate(0, -1, 0), To: first}, true
	case "this month":
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		return temporalMatch{Phrase: norm, From: first, To: first.AddDate(0, 1, 0)}, true
	case "last year":
		first := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return temporalMatch{Phrase: norm, From: first.AddDate(-1, 0, 0), To: first}, true
	case "recently", "the other day", "a few days ago":
		from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -7)
		return temporalMatch{Phrase: norm, From: from, To: now}, true
	}
	// "3 days ago", "two weeks ago", "5 days back"
	if n, unit, ok := p.relativeCount(norm); ok {
		from, to := p.countWindow(n, unit, now, loc)
		if from.IsZero() {
			return temporalMatch{}, false
		}
		return temporalMatch{Phrase: norm, From: from, To: to}, true
	}
	// "on 3 March", "march 3", "2026-03-03"
	if t, ok := p.calendarDay(norm, now, loc); ok {
		return dayWindow(norm, t, loc), true
	}
	return temporalMatch{}, false
}

func (p *TemporalParser) relativeWeekday(norm string) (time.Weekday, bool) {
	for name, wd := range englishWeekdays {
		if norm == "last "+name || norm == "on "+name || norm == "since "+name {
			return wd, true
		}
	}
	return 0, false
}

func (p *TemporalParser) weekdayWindow(norm string, wd time.Weekday, now time.Time, loc *time.Location) (temporalMatch, bool) {
	start := p.startOfWeek(now)
	last := start.AddDate(0, 0, int(wd-p.weekStart+7)%7)
	// "last <weekday>" spoken on that weekday means the previous week, not the
	// current day, so compare calendar days rather than instants.
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if !last.Before(today) {
		last = last.AddDate(0, 0, -7)
	}
	// Japanese/Persian weeks start Saturday; the configured bundle week start
	// defines "@last <weekday>" so the window is stable across locales.
	return temporalMatch{Phrase: norm, From: last, To: last.AddDate(0, 0, 1), Weekday: true}, true
}

func (p *TemporalParser) startOfWeek(now time.Time) time.Time {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	delta := (int(day.Weekday()) - int(p.weekStart) + 7) % 7
	return day.AddDate(0, 0, -delta)
}

var numberWords = map[string]int{
	"a": 1, "an": 1, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5,
	"six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10, "couple": 2,
}

// relativeCount parses "<n> <unit>(s) ago|back". Numbers are bounded: an
// unbounded window is not a recall answer, it is a full-table scan.
func (p *TemporalParser) relativeCount(norm string) (int, string, bool) {
	fields := strings.Fields(norm)
	if len(fields) < 3 || len(fields) > 4 {
		return 0, "", false
	}
	last := fields[len(fields)-1]
	if last != "ago" && last != "back" {
		return 0, "", false
	}
	n, ok := 0, false
	if v, exists := numberWords[fields[0]]; exists {
		n, ok = v, true
	} else {
		n, ok = atoi(fields[0])
	}
	if !ok || n < 1 || n > 365 {
		return 0, "", false
	}
	unit := strings.TrimSuffix(fields[1], "s")
	switch unit {
	case "day", "week":
		// Bounded so a phrase can never request an unbounded scan.
		if n > 365 {
			return 0, "", false
		}
		return n, unit, true
	case "month":
		if n > 24 {
			return 0, "", false
		}
		return n, unit, true
	case "year":
		if n > 5 {
			return 0, "", false
		}
		return n, unit, true
	}
	return 0, "", false
}

func (p *TemporalParser) countWindow(n int, unit string, now time.Time, loc *time.Location) (time.Time, time.Time) {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	target := day
	switch unit {
	case "day":
		target = day.AddDate(0, 0, -n)
	case "week":
		target = day.AddDate(0, 0, -7*n)
	case "month":
		target = day.AddDate(0, -n, 0)
	case "year":
		target = day.AddDate(-n, 0, 0)
	default:
		return time.Time{}, time.Time{}
	}
	if target.After(now) {
		return time.Time{}, time.Time{}
	}
	if unit == "day" {
		return target, target.AddDate(0, 0, 1)
	}
	// Multi-day windows end at the same local time as now, so "two weeks ago"
	// includes the whole span the user could reasonably mean.
	return target, target.AddDate(0, 0, windowDays(unit))
}

func windowDays(unit string) int {
	switch unit {
	case "week":
		return 7
	case "month":
		return 31
	case "year":
		return 366
	default:
		return 1
	}
}

func (p *TemporalParser) calendarDay(norm string, now time.Time, loc *time.Location) (time.Time, bool) {
	if t, err := time.ParseInLocation("2006-01-02", norm, loc); err == nil {
		return t, !t.After(now)
	}
	fields := strings.Fields(strings.TrimPrefix(norm, "on "))
	if len(fields) != 2 {
		return time.Time{}, false
	}
	if month, ok := englishMonths[fields[0]]; ok {
		if d, ok := atoi(fields[1]); ok && d >= 1 && d <= 31 {
			return time.Date(now.Year(), month, d, 0, 0, 0, 0, loc), true
		}
	}
	if month, ok := englishMonths[fields[1]]; ok {
		if d, ok := atoi(fields[0]); ok && d >= 1 && d <= 31 {
			return time.Date(now.Year(), month, d, 0, 0, 0, 0, loc), true
		}
	}
	return time.Time{}, false
}

func dayWindow(phrase string, day time.Time, loc *time.Location) temporalMatch {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	return temporalMatch{Phrase: phrase, From: start, To: start.AddDate(0, 0, 1)}
}

func nightWindow(phrase string, day time.Time, loc *time.Location) temporalMatch {
	start := time.Date(day.Year(), day.Month(), day.Day(), 18, 0, 0, 0, loc)
	return temporalMatch{Phrase: phrase, From: start, To: start.Add(12 * time.Hour)}
}

func morningWindow(phrase string, day time.Time, loc *time.Location) temporalMatch {
	start := time.Date(day.Year(), day.Month(), day.Day(), 4, 0, 0, 0, loc)
	return temporalMatch{Phrase: phrase, From: start, To: start.Add(8 * time.Hour)}
}

func atoi(s string) (int, bool) {
	if s == "" || len(s) > 4 {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

// hasRecallVerb reports whether the message is asking to recall something
// rather than merely naming a time. The temporal strategy is only justified
// when the user is asking about their own past messages.
func hasRecallVerb(text string) bool {
	low := strings.ToLower(text)
	for _, verb := range []string{
		"what did i", "what was i", "what were we", "when did i", "when was it",
		"did i say", "did i tell", "did i mention", "i told you", "i said",
		"remember", "remind me", "recall", "last time", "once said",
	} {
		if strings.Contains(low, verb) {
			return true
		}
	}
	return false
}
