package protocol

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Readiness is deliberately a caller-provided, explicitly grounded assessment.
// Lack of readiness evidence is NOT equivalent to medium readiness.
type Readiness string

const (
	ReadinessUnknown Readiness = "unknown"
	ReadinessLow     Readiness = "low"
	ReadinessMedium  Readiness = "medium"
	ReadinessHigh    Readiness = "high"
)

type ShortlistContext struct {
	WCtx          c.WorkingContext
	Thread        *c.ThreadRecord
	User          c.UserState
	Readiness     Readiness
	Flags         []string // approved flag IDs present on user/thread; absence must be confirmed by the caller
	FlagsVerified bool
	Now           time.Time
}

type Candidate struct {
	Brief c.ProtocolBrief
	Score int
	Rules []string
}

type shortlistEntry struct {
	brief c.ProtocolBrief
	score int
}

// Shortlist is the public WP-10-facing port. With no verified flag inventory or
// readiness source it intentionally yields no results; the extended method
// accepts that evidence from the owner-scoped projection.
func (e *Engine) Shortlist(ctx context.Context, w c.WorkingContext, t *c.ThreadRecord, u c.UserState) ([]c.ProtocolBrief, error) {
	v, err := e.ShortlistWithContext(ctx, ShortlistContext{WCtx: w, Thread: t, User: u, Now: e.now()})
	return v, err
}
func (e *Engine) ShortlistWithContext(ctx context.Context, in ShortlistContext) ([]c.ProtocolBrief, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	w := in.WCtx
	u := in.User
	t := in.Thread
	if !c.ValidID(u.UserID) || !c.ValidID(w.Turn.MessageID) || !c.ValidID(w.Turn.TurnID) || !w.Risk.Current.Valid() || !u.Risk.Level.Valid() || c.MaxRisk(w.Risk.Current, u.Risk.Level).Rank() > c.RiskLow.Rank() || w.Risk.FollowUpActive || w.Protocol != nil || t == nil || t.UserID != u.UserID || w.FocusThread == nil || w.FocusThread.ID != t.ID || readinessRank(in.Readiness) < 0 {
		return nil, nil
	}
	if w.Prefs.Language != "" && w.Prefs.Language != "en" {
		return nil, nil
	} // approved protocols currently English-only
	if !w.Eligibility.ProtocolsPossible {
		return nil, nil
	}
	now := in.Now
	if now.IsZero() {
		now = e.now()
	}
	rows := []shortlistEntry{}
	for _, p := range e.assets.protocols {
		if len(p.Eligibility.ForbidFlags) > 0 && !in.FlagsVerified {
			continue
		}
		if !eligible(p, w, *t, u, in.Readiness, in.Flags, now) {
			continue
		}
		score := 0
		for _, tag := range p.FitTags {
			if has(t.ProblemTags, tag) {
				score += 10
			}
		}
		if score == 0 {
			continue
		} // no evidence-grounded problem match
		if t.Goal != nil {
			for _, tag := range p.FitTags {
				if strings.Contains(strings.ToLower(*t.Goal), strings.ReplaceAll(tag, "_", " ")) {
					score += 3
					break
				}
			}
		}
		if has(w.Prefs.Likes, p.ID) || has(w.Prefs.Likes, p.Family) {
			score += 5
		}
		if has(w.Prefs.Avoid, p.ID) || has(w.Prefs.Avoid, p.Family) {
			continue
		}
		var outcome *string
		for i := len(u.ProtocolHistory) - 1; i >= 0; i-- {
			h := u.ProtocolHistory[i]
			if h.ProtocolID == p.ID {
				v := h.Outcome
				outcome = &v
				switch v {
				case "helped":
					score += 4
				case "not_helpful", "stopped", "declined":
					score -= 8
				}
				break
			}
		}
		rows = append(rows, shortlistEntry{c.ProtocolBrief{ProtocolID: p.ID, Name: p.Name, OneLiner: p.OneLiner, FitTags: append([]string(nil), p.FitTags...), EstMinutes: p.EstMinutes, PastOutcome: outcome}, score})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].score != rows[j].score {
			return rows[i].score > rows[j].score
		}
		return rows[i].brief.ProtocolID < rows[j].brief.ProtocolID
	})
	if len(rows) > 5 {
		rows = rows[:5]
	}
	out := make([]c.ProtocolBrief, 0, len(rows))
	for _, v := range rows {
		out = append(out, v.brief)
	}
	return out, nil
}
func eligible(p bundle.Protocol, w c.WorkingContext, t c.ThreadRecord, u c.UserState, readiness Readiness, flags []string, now time.Time) bool {
	if w.Risk.Current.Rank() > riskFrom(p.Eligibility.RequireRiskAtMost).Rank() || readinessRank(readiness) < readinessRank(Readiness(p.Eligibility.MinUserReadiness)) {
		return false
	}
	phase := false
	for _, v := range p.Eligibility.RequirePhase {
		if v == string(t.Phase) {
			phase = true
			break
		}
	}
	if !phase {
		return false
	}
	for _, slot := range p.Eligibility.RequireSlotsFilled {
		// The model-facing WorkingContext is not authoritative evidence.
		// Every required slot must also exist as a reported value in the
		// persisted, owner-scoped thread with matching provenance.
		stored, exists := t.Slots[slot]
		if !exists || stored.Status != c.EpReported || stored.Declined || len(stored.Evidence) == 0 {
			return false
		}
		valid := false
		for _, s := range w.FocusThread.Slots {
			if s.Slot != slot || s.Status != c.EpReported || s.Declined || s.Value != stored.Value {
				continue
			}
			for _, id := range s.Evidence {
				for _, original := range stored.Evidence {
					if id == original && c.ValidID(id) {
						valid = true
					}
				}
			}
		}
		if !valid {
			return false
		}
	}
	for _, flag := range p.Eligibility.ForbidFlags {
		if has(flags, flag) || has(t.ProblemTags, flag) {
			return false
		}
	}
	runs := 0
	for _, h := range u.ProtocolHistory {
		if h.ProtocolID != p.ID || h.At.IsZero() || h.At.After(now) {
			continue
		}
		runs++
		days := now.Sub(h.At)
		if h.Outcome == "declined" && days < time.Duration(p.Eligibility.DeclineCooldownDays)*24*time.Hour {
			return false
		}
		if days < time.Duration(p.Eligibility.MinDaysBetweenRuns)*24*time.Hour {
			return false
		}
	}
	return runs < p.Eligibility.MaxRunsPerThread
}
func riskFrom(s string) c.RiskLevel {
	v := c.RiskLevel(s)
	if !v.Valid() {
		return c.RiskNone
	}
	return v
}
func readinessRank(v Readiness) int {
	switch v {
	case ReadinessLow:
		return 0
	case ReadinessMedium:
		return 1
	case ReadinessHigh:
		return 2
	default:
		return -1
	}
}
func has(items []string, v string) bool {
	for _, s := range items {
		if s == v {
			return true
		}
	}
	return false
}
