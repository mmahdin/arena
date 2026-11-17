package assembler

import (
	"sort"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// informationNeeds is purely deterministic. Its schema, threshold and
// hints are pinned to WP-03, not to a hard-coded psychology question bank.
func informationNeeds(thread *c.ThreadView, session c.SessionView, user c.UserState, risk c.RiskLevel, p contextstate.Policy, now time.Time) []c.InfoNeed {
	if thread == nil || session.Mode == c.ModeVenting || session.Mode == c.ModeSafety || riskOrder(risk) >= 2 || user.ComfortBudget <= 0 {
		return nil
	}
	suppress := map[string]bool{}
	for _, v := range p.Problem.InformationNeeds.SuppressIf {
		suppress[v] = true
	}
	if suppress["mode_venting"] && session.Mode == c.ModeVenting || suppress["risk_gte_elevated"] && riskOrder(risk) >= 2 {
		return nil
	}
	existing := make(map[string]c.SlotView, len(thread.Slots))
	for _, v := range thread.Slots {
		existing[v.Slot] = v
	}
	candidates := []c.InfoNeed{}
	for _, def := range p.Problem.Slots {
		v, exists := existing[def.ID]
		if exists && v.Status != c.EpUnknown {
			continue
		}
		if exists && v.Declined {
			continue
		} // never repeat a declined question
		if exists && v.Asked != nil && !v.Asked.IsZero() && now.Sub(*v.Asked) < time.Duration(p.Problem.InformationNeeds.RecentlyAskedHours)*time.Hour {
			continue
		}
		if session.PendingQuestion != nil && session.PendingQuestion.Slot == def.ID {
			continue
		}
		if def.AskCost > float64(user.ComfortBudget) {
			continue
		}
		importance := def.Importance[string(thread.Phase)]
		if importance < p.Problem.InformationNeeds.VMin {
			continue
		}
		score := importance - def.AskCost
		if score <= 0 {
			continue
		}
		candidates = append(candidates, c.InfoNeed{Slot: def.ID, Score: float32(score), Purpose: "clarify_" + def.ID, AskHint: def.AskHint})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Slot < candidates[j].Slot
		}
		return candidates[i].Score > candidates[j].Score
	})
	max := p.Problem.InformationNeeds.MaxReturned
	if max > 2 {
		max = 2
	}
	if max < 0 {
		max = 0
	}
	if len(candidates) > max {
		candidates = candidates[:max]
	}
	return candidates
}

// completeSlots preserves schema-defined unknown slots so absence is not
// confused with an inference. No unsupported values are invented.
func completeSlots(focus *c.ThreadView, p contextstate.Policy) {
	if focus == nil {
		return
	}
	byID := map[string]c.SlotView{}
	for _, s := range focus.Slots {
		byID[s.Slot] = s
	}
	out := make([]c.SlotView, 0, len(p.Problem.Slots))
	for _, def := range p.Problem.Slots {
		s, ok := byID[def.ID]
		if !ok || !s.Status.Valid() || s.Status == c.EpSystemObserved {
			s = c.SlotView{Slot: def.ID, Status: c.EpUnknown}
		}
		s.Slot = def.ID
		out = append(out, s)
	}
	focus.Slots = out
}
func computeEligibility(s contextstate.Snapshot, focus *c.ThreadView, proto *c.ProtocolView, risk c.RiskLevel, p contextstate.Policy, now time.Time) c.Eligibility {
	reasons := []string{}
	if riskOrder(risk) >= 2 {
		reasons = append(reasons, "risk_elevated")
	}
	if focus == nil || focus.Phase == c.PhaseExploring || focus.Phase == c.PhaseResolved || focus.Phase == c.PhasePaused {
		reasons = append(reasons, "phase_too_early")
	}
	if proto != nil && (proto.Status == c.PIActive || proto.Status == c.PIOffered) {
		reasons = append(reasons, "active_protocol")
	}
	for _, h := range s.User.ProtocolHistory {
		if h.Outcome == "declined" && now.Sub(h.At) < 14*24*time.Hour && now.After(h.At) {
			reasons = append(reasons, "recent_decline")
			break
		}
	}
	// WP-10 supplies explicit, validated action cues. Never guess readiness
	// from text at the context-assembly stage.
	reasons = append(reasons, "no_action_cue")
	if p.Synthetic {
		reasons = append(reasons, "unreviewed_bundle")
	}
	mayVoice := false
	if focus != nil && riskOrder(risk) < 2 && !p.Synthetic {
		for _, h := range focus.Hypotheses {
			if h.Status == c.MemSupported {
				mayVoice = true
				break
			}
		}
	}
	limit := p.Gate.Questions.MaxConsecutiveQuestionTurns
	if limit <= 0 {
		limit = 2
	}
	allowed := s.Session.ConsecutiveQuestions < limit && riskOrder(risk) < 3
	return c.Eligibility{ProtocolsPossible: len(reasons) == 0, ReasonsBlocked: reasons,
		MayVoiceHypothesis: mayVoice, QuestionAllowed: allowed}
}
func containsText(values []string, val string) bool {
	for _, v := range values {
		if strings.EqualFold(v, val) {
			return true
		}
	}
	return false
}
