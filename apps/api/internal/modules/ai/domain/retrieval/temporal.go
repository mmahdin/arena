package retrieval

import (
	"regexp"
	"strings"
	"time"
)

// Window is a resolved half-open time interval in the user's timezone.
type Window struct {
	Phrase string
	From   time.Time
	To     time.Time
}

// TemporalParser resolves the small, closed set of relative time expressions the
// architecture actually requires (§3.4 "parse time window with tz", Scenario 2
// "last Tuesday"). Unknown expressions are reported as unresolved so the caller
// can fall back to hybrid retrieval instead of guessing.
//
// This is deliberately a pattern set, not a natural-language date library: the
// Launch languages are fa/en, the expressions are enumerated, and every rule has
// a unit test with a fixed reference time.
type TemporalParser struct {
	Loc                *time.Location
	WeekStartsOnMonday bool
	// Now is injected so parsing is deterministic and testable.
	Now time.Time
}

type temporalRule struct {
	re    *regexp.Regexp
	apply func(p TemporalParser, match []string, t time.Time, loc *time.Location) (Window, bool)
}

// Persian and English month/day names needed for explicit dates.
var monthNames = map[string]time.Month{
	"january": time.January, "jan": time.January,
	"february": time.February, "feb": time.February,
	"march": time.March, "mar": time.March,
	"april": time.April, "apr": time.April,
	"may":  time.May,
	"june": time.June, "jun": time.June,
	"july": time.July, "jul": time.July,
	"august": time.August, "aug": time.August,
	"september": time.September, "sep": time.September, "sept": time.September,
	"october": time.October, "oct": time.October,
	"november": time.November, "nov": time.November,
	"december": time.December, "dec": time.December,
}

var weekdayNames = map[string]time.Weekday{
	"sunday": time.Sunday, "sun": time.Sunday,
	"monday": time.Monday, "mon": time.Monday,
	"tuesday": time.Tuesday, "tue": time.Tuesday, "tues": time.Tuesday,
	"wednesday": time.Wednesday, "wed": time.Wednesday,
	"thursday": time.Thursday, "thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday,
	"friday": time.Friday, "fri": time.Friday,
	"saturday": time.Saturday, "sat": time.Saturday,
}

var persianWeekdays = map[string]time.Weekday{
	"یکشنبه": time.Sunday, "دوشنبه": time.Monday, "سه‌شنبه": time.Tuesday, "سه شنبه": time.Tuesday,
	"چهارشنبه": time.Wednesday, "پنجشنبه": time.Thursday, "پنج‌شنبه": time.Thursday, "جمعه": time.Friday, "شنبه": time.Saturday,
}

// recallVerbs are the recall markers that make a temporal query "exact recall"
// (§3.4 trigger: temporal expression + recall verb).
var recallVerbs = []string{
	"what did i", "what have i", "what was i", "did i tell you", "did i say", "did i mention",
	"tell you about", "told you about", "i told you", "i said", "remind me", "do you remember",
	"recall", "exactly",
	"گفتم", "گفته بودم", "چی گفتم", "یادت", "به یاد", "یادت هست", "گفته‌ام",
}

var orderedRules = []temporalRule{
	{
		re: regexp.MustCompile(`(?i)\b(last|previous|past)\s+(night|evening)\b`),
		apply: func(p TemporalParser, _ []string, t time.Time, loc *time.Location) (Window, bool) {
			day := t.AddDate(0, 0, -1)
			from, to := dayBounds(day, loc)
			// "last night" spills into the small hours of today.
			return Window{Phrase: "last night", From: from.Add(18 * time.Hour), To: to.Add(4 * time.Hour)}, true
		},
	},
	{
		re: regexp.MustCompile(`(?i)(?:\b(yesterday)\b|(دیروز))`),
		apply: func(_ TemporalParser, m []string, t time.Time, loc *time.Location) (Window, bool) {
			from, to := dayBounds(t.AddDate(0, 0, -1), loc)
			return Window{Phrase: firstGroup(m), From: from, To: to}, true
		},
	},
	{
		re: regexp.MustCompile(`(?i)(?:\b(today)\b|(امروز|امشب))`),
		apply: func(_ TemporalParser, m []string, t time.Time, loc *time.Location) (Window, bool) {
			from, to := dayBounds(t, loc)
			return Window{Phrase: firstGroup(m), From: from, To: to}, true
		},
	},
	{
		re: regexp.MustCompile(`(?i)\b(\d{1,2})\s+days?\s+ago\b`),
		apply: func(_ TemporalParser, m []string, t time.Time, loc *time.Location) (Window, bool) {
			n := atoiSafe(m[1])
			if n < 0 || n > 366 {
				return Window{}, false
			}
			from, to := dayBounds(t.AddDate(0, 0, -n), loc)
			return Window{Phrase: m[0], From: from, To: to}, true
		},
	},
	{
		re: regexp.MustCompile(`(?i)(?:\b(last|previous)\s+(week)\b|(هفته)\s*(قبل|پیش))`),
		apply: func(p TemporalParser, m []string, t time.Time, loc *time.Location) (Window, bool) {
			start, end := weekBounds(t, loc, p.WeekStartsOnMonday)
			from := start.AddDate(0, 0, -7)
			to := end.AddDate(0, 0, -7)
			return Window{Phrase: m[0], From: from, To: to}, true
		},
	},
	{
		re: regexp.MustCompile(`(?i)\b(this|current)\s+week\b`),
		apply: func(p TemporalParser, m []string, t time.Time, loc *time.Location) (Window, bool) {
			from, to := weekBounds(t, loc, p.WeekStartsOnMonday)
			return Window{Phrase: m[0], From: from, To: to}, true
		},
	},
	{
		re: regexp.MustCompile(`(?i)\b(last|previous|past)\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b`),
		apply: func(p TemporalParser, m []string, t time.Time, loc *time.Location) (Window, bool) {
			wd, ok := weekdayNames[strings.ToLower(m[2])]
			if !ok {
				return Window{}, false
			}
			start, _ := weekBounds(t, loc, p.WeekStartsOnMonday)
			// Inside the previous calendar week, pick the same weekday.
			from, to := dayBounds(start.AddDate(0, 0, -7), loc)
			day := start.AddDate(0, 0, -7)
			for day.Weekday() != wd {
				day = day.AddDate(0, 0, 1)
			}
			from, to = dayBounds(day, loc)
			return Window{Phrase: m[0], From: from, To: to}, true
		},
	},
	{
		re: regexp.MustCompile(`(?i)\b(on\s+)?(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b`),
		apply: func(p TemporalParser, m []string, t time.Time, loc *time.Location) (Window, bool) {
			wd, ok := weekdayNames[strings.ToLower(m[2])]
			if !ok {
				return Window{}, false
			}
			day := t.In(loc)
			for day.Weekday() != wd {
				day = day.AddDate(0, 0, -1)
			}
			from, to := dayBounds(day, loc)
			return Window{Phrase: strings.TrimSpace(m[0]), From: from, To: to}, true
		},
	},
	{
		// Persian weekday without a modifier: most recent occurrence.
		re: regexp.MustCompile(`(یکشنبه|دوشنبه|سه‌شنبه|سه شنبه|چهارشنبه|پنجشنبه|پنج‌شنبه|جمعه|شنبه)`),
		apply: func(_ TemporalParser, m []string, t time.Time, loc *time.Location) (Window, bool) {
			wd, ok := persianWeekdays[m[1]]
			if !ok {
				return Window{}, false
			}
			day := t.In(loc)
			for day.Weekday() != wd {
				day = day.AddDate(0, 0, -1)
			}
			from, to := dayBounds(day, loc)
			return Window{Phrase: m[1], From: from, To: to}, true
		},
	},
	{
		re: regexp.MustCompile(`(?i)\b(\d{1,2})(?:st|nd|rd|th)?\s+([a-z]{3,9})\b`),
		apply: func(_ TemporalParser, m []string, t time.Time, loc *time.Location) (Window, bool) {
			mon, ok := monthNames[strings.ToLower(m[2])]
			if !ok {
				return Window{}, false
			}
			day := atoiSafe(m[1])
			if day < 1 || day > 31 {
				return Window{}, false
			}
			year := t.In(loc).Year()
			candidate := time.Date(year, mon, day, 12, 0, 0, 0, loc)
			if candidate.After(t.In(loc)) {
				candidate = time.Date(year-1, mon, day, 12, 0, 0, 0, loc)
			}
			from, to := dayBounds(candidate, loc)
			return Window{Phrase: m[0], From: from, To: to}, true
		},
	},
	{
		re: regexp.MustCompile(`(?i)\b(\d{4})-(\d{2})-(\d{2})\b`),
		apply: func(_ TemporalParser, m []string, _ time.Time, loc *time.Location) (Window, bool) {
			year, mon, day := atoiSafe(m[1]), atoiSafe(m[2]), atoiSafe(m[3])
			if mon < 1 || mon > 12 || day < 1 || day > 31 {
				return Window{}, false
			}
			from, to := dayBounds(time.Date(year, time.Month(mon), day, 12, 0, 0, 0, loc), loc)
			return Window{Phrase: m[0], From: from, To: to}, true
		},
	},
}

// Parse returns the first resolvable window in the text, or ok=false.
// Resolution precedence is the declaration order above (most specific first).
func (p TemporalParser) Parse(text string) (Window, bool) {
	loc := p.Loc
	if loc == nil {
		loc = time.UTC
	}
	now := p.Now
	if now.IsZero() {
		now = time.Unix(0, 0)
	}
	for _, rule := range orderedRules {
		m := rule.re.FindStringSubmatch(text)
		if len(m) == 0 {
			continue
		}
		if w, ok := rule.apply(p, m, now, loc); ok {
			return w, true
		}
	}
	return Window{}, false
}

// HasRecallVerb reports whether the text contains a recall marker.
func HasRecallVerb(text string) bool {
	lower := strings.ToLower(text)
	for _, v := range recallVerbs {
		if strings.Contains(lower, v) {
			return true
		}
	}
	return false
}

// RecallTerms extracts the content terms of a temporal recall query, dropping
// stop words and the time expression itself, so the SQL layer can filter. The
// result is capped and used only as an optional LIKE/FTS filter, never as the
// sole evidence for an answer.
func RecallTerms(text string) []string {
	stop := map[string]struct{}{
		"the": {}, "a": {}, "an": {}, "about": {}, "my": {}, "me": {}, "i": {}, "you": {},
		"what": {}, "did": {}, "do": {}, "was": {}, "tell": {}, "say": {}, "said": {}, "told": {},
		"last": {}, "previous": {}, "past": {}, "on": {}, "in": {}, "at": {}, "of": {}, "to": {},
		"exactly": {}, "remember": {}, "recall": {}, "remind": {}, "monday": {}, "tuesday": {},
		"wednesday": {}, "thursday": {}, "friday": {}, "saturday": {}, "sunday": {}, "week": {},
		"yesterday": {}, "today": {}, "night": {}, "evening": {}, "ago": {}, "days": {}, "day": {},
		"that": {}, "it": {}, "we": {}, "were": {}, "have": {}, "had": {}, "with": {},
		"گفتم": {}, "گفته": {}, "بودم": {}, "چی": {}, "یادت": {}, "درباره": {}, "که": {}, "را": {},
		"دیروز": {}, "امروز": {}, "هفته": {}, "شنبه": {}, "یکشنبه": {}, "دوشنبه": {}, "سه‌شنبه": {},
		"چهارشنبه": {}, "پنجشنبه": {}, "جمعه": {},
	}
	seen := map[string]struct{}{}
	var out []string
	for _, tok := range strings.Fields(strings.ToLower(text)) {
		tok = strings.Trim(tok, ".,!?؛،:\"'()[]")
		if len([]rune(tok)) < 2 {
			continue
		}
		if _, skip := stop[tok]; skip {
			continue
		}
		if _, dup := seen[tok]; dup {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
		if len(out) == 4 {
			break
		}
	}
	return out
}

// firstGroup returns the first non-empty capture group, which keeps one rule
// usable for both the English and Persian alternative.
func firstGroup(m []string) string {
	for i := 1; i < len(m); i++ {
		if m[i] != "" {
			return m[i]
		}
	}
	return m[0]
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return -1
		}
		n = n*10 + int(r-'0')
		if n > 999999 {
			return -1
		}
	}
	return n
}

func dayBounds(t time.Time, loc *time.Location) (time.Time, time.Time) {
	local := t.In(loc)
	from := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return from, from.AddDate(0, 0, 1)
}

func weekBounds(t time.Time, loc *time.Location, mondayStart bool) (time.Time, time.Time) {
	from, _ := dayBounds(t, loc)
	wd := from.Weekday()
	if mondayStart {
		offset := (int(wd) + 6) % 7 // Monday=0
		from = from.AddDate(0, 0, -offset)
	} else {
		from = from.AddDate(0, 0, -int(wd))
	}
	return from, from.AddDate(0, 0, 7)
}
