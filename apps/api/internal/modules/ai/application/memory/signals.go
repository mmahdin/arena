package memory

import c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"

// Signals are the already WP-08-validated StateDelta handoff, never model
// authored memory prose. Endorsement/rejection require explicit user controls;
// propose contributes only its approved tag to the separate extraction prompt.
func signalProposals(signals []c.HypothesisSignal) ([]c.MemoryOpProposal, error) {
	if len(signals) > 3 {
		return nil, ErrDenied
	}
	out := []c.MemoryOpProposal{}
	for _, signal := range signals {
		if signal.Op != c.HypSupport && signal.Op != c.HypContradict {
			continue
		}
		if signal.ID == nil || !c.ValidID(*signal.ID) || len(signal.Evidence) < 1 || len(signal.Evidence) > 6 {
			return nil, ErrDenied
		}
		op := c.OpConfirm
		if signal.Op == c.HypContradict {
			op = c.OpContradict
		}
		out = append(out, c.MemoryOpProposal{Op: op, TargetID: signal.ID, EvidenceMsg: uniqueIDs(signal.Evidence), ReasonTag: "brain_signal"})
	}
	return out, nil
}
