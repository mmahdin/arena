package assembler

import (
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

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
	if proto != nil && (proto.Status == c.PIActive || proto.Status == c.PIOffered) {
		reasons = append(reasons, "active_protocol")
	}
	// Per-protocol exclusions are checked by the engine. Bundle activation is
	// already enforced by the registry/loaders, including synthetic test opt-in.

	mayVoice := false
	if focus != nil && riskOrder(risk) < 2 && !p.Synthetic {
		for _, h := range focus.Hypotheses {
			if h.Status == c.MemSupported {
				mayVoice = true
				break
			}
		}
	}
	allowed := riskOrder(risk) < 3
	return c.Eligibility{ProtocolsPossible: len(reasons) == 0, ReasonsBlocked: reasons,
		MayVoiceHypothesis: mayVoice, QuestionAllowed: allowed}
}
