package orchestrator

import (
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// integrateBrainDelta bridges WP-05's already-restricted Brain result to the
// WP-08 validator. It is deliberately limited to the existing focus thread;
// the not-yet-implemented WP-10 policy gate must own creation, topic switches,
// reported-fact verification and protocol decisions. Every optional change is
// checked again by the WP-08 StateCommitter before a single WP-02 commit.
func (o *Service) integrateBrainDelta(d *contract.StateDelta, brain contract.BrainDecision, wc contract.WorkingContext, snap ports.StateSnapshot) {
	if d == nil {
		return
	}
	if brain.Understanding.SessionMode != nil && brain.Understanding.SessionMode.Valid() {
		v := *brain.Understanding.SessionMode
		d.Session.Mode = &v
	}
	if brain.Understanding.SessionSummaryDelta != nil {
		v := *brain.Understanding.SessionSummaryDelta
		d.Session.SummaryDelta = &v
	}
	if brain.TurnAnalysis.AnsweredPending && wc.Session.PendingQuestion != nil {
		d.Session.ClearPending = true
	}
	// No evidence-free thread updates and no allowing a model to mark its
	// own inferences as "reported". WP-10 will own reported extraction.
	if snap.Focus == nil || wc.FocusThread == nil || brain.Understanding.ThreadID == nil ||
		*brain.Understanding.ThreadID != snap.Focus.ID || wc.FocusThread.ID != snap.Focus.ID {
		return
	}
	patch := &contract.ThreadPatch{ThreadID: snap.Focus.ID}
	if brain.Understanding.ThreadPhaseProposal != nil {
		phase := *brain.Understanding.ThreadPhaseProposal
		if phase.Valid() {
			patch.Phase = &phase
		}
	}
	for _, slot := range brain.Understanding.Slots {
		if slot.Status != contract.EpInferred || len(slot.Evidence) == 0 {
			continue
		}
		patch.Slots = append(patch.Slots, slot)
	}
	if patch.Phase != nil || len(patch.Slots) > 0 {
		patch.Touch = true
		d.Thread = patch
	}
}

func hasRiskEvent(events []contract.DomainEvent) bool {
	count := 0
	for _, event := range events {
		if event.Type == contract.EvRiskLevelChanged {
			count++
		}
	}
	return count == 1
}
