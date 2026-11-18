package retrieval

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

var isoDay = regexp.MustCompile(`\b(20[0-9]{2})-([0-9]{2})-([0-9]{2})\b`)
var lastWeekday = regexp.MustCompile(`\b(?:last|previous)\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b`)

// Temporal bounds are local calendar boundaries converted to UTC; adding 24h
// would be wrong on DST days. The UTC SQL window is always [from,to).
func TemporalReferences(text string, now time.Time, tz string) ([]c.TemporalRef, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid user-state timezone")
	}
	local := now.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	low := strings.ToLower(text)
	var from, to time.Time
	phrase := ""
	switch {
	case strings.Contains(low, "day before yesterday") || strings.Contains(low, "پریروز"):
		phrase, from, to = "day before yesterday", day.AddDate(0, 0, -2), day.AddDate(0, 0, -1)
	case strings.Contains(low, "yesterday") || strings.Contains(low, "دیروز"):
		phrase, from, to = "yesterday", day.AddDate(0, 0, -1), day
	case strings.Contains(low, "two days ago") || strings.Contains(low, "دو روز پیش"):
		phrase, from, to = "two days ago", day.AddDate(0, 0, -2), day.AddDate(0, 0, -1)
	case strings.Contains(low, "today") || strings.Contains(low, "امروز"):
		phrase, from, to = "today", day, day.AddDate(0, 0, 1)
	case strings.Contains(low, "last week") || strings.Contains(low, "هفته پیش") || strings.Contains(low, "هفته قبل"):
		monday := day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
		phrase, from, to = "last week", monday.AddDate(0, 0, -7), monday
	case strings.Contains(low, "this week") || strings.Contains(low, "این هفته"):
		monday := day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
		phrase, from, to = "this week", monday, monday.AddDate(0, 0, 7)
	case strings.Contains(low, "last month") || strings.Contains(low, "ماه قبل") || strings.Contains(low, "ماه پیش"):
		start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
		phrase, from, to = "last month", start.AddDate(0, -1, 0), start
	default:
		// Localized weekday reference. This is calendar parsing, not an LLM
		// guess; Monday is the bundle's required week boundary.
		faWeekdays := []struct {
			names []string
			day   time.Weekday
		}{
			{[]string{"یکشنبه", "یک شنبه"}, time.Sunday},
			{[]string{"دوشنبه", "دو شنبه"}, time.Monday}, {[]string{"سه شنبه", "سه‌شنبه"}, time.Tuesday},
			{[]string{"چهارشنبه", "چهار شنبه"}, time.Wednesday}, {[]string{"پنجشنبه", "پنج شنبه"}, time.Thursday},
			{[]string{"جمعه"}, time.Friday}, {[]string{"شنبه"}, time.Saturday},
		}
		for _, weekday := range faWeekdays {
			for _, name := range weekday.names {
				for _, suffix := range []string{" پیش", " قبل", " گذشته"} {
					if strings.Contains(low, name+suffix) {
						distance := (int(day.Weekday()) - int(weekday.day) + 7) % 7
						if distance == 0 {
							distance = 7
						}
						from = day.AddDate(0, 0, -distance)
						to = from.AddDate(0, 0, 1)
						phrase = name + suffix
						break
					}
				}
				if phrase != "" {
					break
				}
			}
			if phrase != "" {
				break
			}
		}
		if phrase != "" {
			break
		}
		if groups := lastWeekday.FindStringSubmatch(low); len(groups) == 2 {
			weekdays := map[string]time.Weekday{"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday}
			distance := (int(day.Weekday()) - int(weekdays[groups[1]]) + 7) % 7
			if distance == 0 {
				distance = 7
			}
			from = day.AddDate(0, 0, -distance)
			to = from.AddDate(0, 0, 1)
			phrase = groups[0]
		} else if groups := isoDay.FindStringSubmatch(low); len(groups) == 4 {
			parsed, e := time.ParseInLocation("2006-01-02", groups[0], loc)
			if e == nil && parsed.Format("2006-01-02") == groups[0] {
				phrase, from, to = groups[0], parsed, parsed.AddDate(0, 0, 1)
			}
		}
	}
	if phrase == "" {
		return nil, nil
	}
	return []c.TemporalRef{{Phrase: phrase, From: from.UTC(), To: to.UTC(), RecallVerb: recallWording(text)}}, nil
}

func recallWording(text string) bool {
	low := strings.ToLower(text)
	for _, phrase := range []string{"what did i", "did i tell", "did we talk", "what i said", "remember", "recall", "گفتم", "یادت", "یادته", "صحبت کردیم", "گفته بودم"} {
		if strings.Contains(low, phrase) {
			return true
		}
	}
	return false
}
