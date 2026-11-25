package protocol

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"sort"
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

// Shortlist prefilters knowledge by actual risk, ownership and known exclusions.
// Brain judges fit/readiness; an optional offer is not consent to begin.
func (e *Engine) Shortlist(ctx context.Context, w c.WorkingContext, t *c.ThreadRecord, u c.UserState) ([]c.ProtocolBrief, error) {
	v, err := e.ShortlistWithContext(ctx, ShortlistContext{WCtx: w, Thread: t, User: u, Now: e.now()})
	return v, err
}
func (e *Engine) ShortlistWithContext(ctx context.Context, in ShortlistContext) (shortlistResult []c.ProtocolBrief, shortlistErr error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/protocol/eligibility.ShortlistWithContext", func() any { return map[string]any{"in": in} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"shortlistResult": shortlistResult} }, shortlistErr, recover())
	}()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	w := in.WCtx
	u := in.User
	t := in.Thread
	if !c.ValidID(u.UserID) || !c.ValidID(w.Turn.MessageID) || !c.ValidID(w.Turn.TurnID) || !w.Risk.Current.Valid() || !u.Risk.Level.Valid() || c.MaxRisk(w.Risk.Current, u.Risk.Level).Rank() > c.RiskLow.Rank() || w.Risk.FollowUpActive || w.Protocol != nil || t != nil && (t.UserID != u.UserID || w.FocusThread == nil || w.FocusThread.ID != t.ID) || t == nil && w.FocusThread != nil {
		return nil, nil
	}
	if !e.assets.manifest.ProtocolLanguage(c.ResponseLanguage(w.Prefs.Language, w.Turn.Lang)) {
		return nil, nil
	}
	if !w.Eligibility.ProtocolsPossible {
		return nil, nil
	}
	now := in.Now
	if now.IsZero() {
		now = e.now()
	}
	rows := []c.ProtocolBrief{}
	for _, p := range e.assets.protocols {
		trace.Capture(ctx, "candidate_definition", "knowledge", "protocols/"+p.ID+".yaml", p)
		thread := c.ThreadRecord{}
		if t != nil {
			thread = *t
		}
		if !eligible(p, w, thread, u, in.Readiness, in.Flags, now) {
			continue
		}

		var outcome *string
		for i := len(u.ProtocolHistory) - 1; i >= 0; i-- {
			h := u.ProtocolHistory[i]
			if h.ProtocolID == p.ID {
				v := h.Outcome
				outcome = &v

				break
			}
		}
		rows = append(rows, c.ProtocolBrief{ProtocolID: p.ID, Name: p.Name, OneLiner: p.OneLiner, FitTags: append([]string(nil), p.FitTags...), EstMinutes: p.EstMinutes, PastOutcome: outcome})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].ProtocolID < rows[j].ProtocolID
	})
	return rows, nil
}
func eligible(p bundle.Protocol, w c.WorkingContext, t c.ThreadRecord, u c.UserState, readiness Readiness, flags []string, now time.Time) bool {
	if w.Risk.Current.Rank() > riskFrom(p.Eligibility.RequireRiskAtMost).Rank() {
		return false
	}
	phase := len(p.Eligibility.RequirePhase) == 0
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
	return p.Eligibility.MaxRunsPerThread == 0 || runs < p.Eligibility.MaxRunsPerThread
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
