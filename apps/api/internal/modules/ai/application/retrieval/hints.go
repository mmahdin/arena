package retrieval

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"regexp"
	"strings"
	"time"
)

// Hints come only from earlier messages in the admitted conversation. They
// supplement the original query; they never replace it or gate retrieval.
type Hints struct {
	RecentContext string
	ContextQuery  string
	ContextVector []float32
	Focus         *c.ID
	Windows       []c.TemporalRef
	Hydrate       func(context.Context, Candidate) (Candidate, bool, error)
}
type HintReader interface {
	RetrievalHints(context.Context, c.ID, c.ID) (Hints, error)
}

var literalDate = regexp.MustCompile(`(?:^|[^0-9])([0-9]{4}-[0-9]{2}-[0-9]{2})(?:$|[^0-9])`)

func timeWindows(text, tz string, now time.Time) []c.TemporalRef {
	loc, e := time.LoadLocation(tz)
	if e != nil {
		loc = time.UTC
	}
	now = now.In(loc)
	normalized := strings.Join(Terms(text), " ")
	// Date punctuation is retained separately while Persian digits are folded.
	digits := strings.NewReplacer("۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9").Replace(text)
	if m := literalDate.FindStringSubmatch(digits); len(m) > 1 {
		if from, e := time.ParseInLocation("2006-01-02", m[1], loc); e == nil {
			return []c.TemporalRef{{Phrase: m[1], From: from, To: from.AddDate(0, 0, 1)}}
		}
	}
	offset := 0
	present := false
	for _, term := range strings.Fields(normalized) {
		if term == "yesterday" || term == "دیروز" {
			offset = -1
			present = true
		}
		if term == "today" || term == "امروز" {
			present = true
		}
	}
	if present {
		from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, offset)
		return []c.TemporalRef{{Phrase: "relative_day", From: from, To: from.AddDate(0, 0, 1)}}
	}
	return nil
}
func threadRanks(docs []Candidate, focus *c.ID) []ranked {
	if focus == nil {
		return nil
	}
	out := []ranked{}
	for i, d := range docs {
		if d.Source == SourceThread && d.ID == *focus || d.ThreadID != nil && *d.ThreadID == *focus {
			out = append(out, ranked{i, 1})
		}
	}
	return order(out, docs)
}
