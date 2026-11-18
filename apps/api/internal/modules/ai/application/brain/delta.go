package brain

import (
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// ProposeDelta translates a VALIDATED decision; WP-08 revalidates it, and only
// WP-02's fenced finalizer may apply it. Question text/count comes from WP-07
// validation in the existing finalizer, never from the Brain's question plan.
// Protocol actions stay proposals for WP-13; no protocol is started here.
func ProposeDelta(d *c.StateDelta, b c.BrainDecision, w c.WorkingContext, s ports.StateSnapshot) {
	if d == nil {
		return
	}
	d.Session.Mode = b.Understanding.SessionMode
	d.Session.SummaryDelta = b.Understanding.SessionSummaryDelta
	d.Session.ClearPending = b.TurnAnalysis.AnsweredPending && w.Session.PendingQuestion != nil && d.Session.PendingQuestion == nil
	for _, e := range b.TurnAnalysis.Emotions {
		d.Session.EmotionLabels = append(d.Session.EmotionLabels, string(e.Label))
	}
	var target c.ID
	relation := b.TurnAnalysis.ThreadRelation
	if relation == c.RelNew && b.TurnAnalysis.NewThreadTopic != nil {
		target = c.NewID()
		d.Thread = &c.ThreadPatch{ThreadID: target, Create: &c.ThreadInit{Topic: *b.TurnAnalysis.NewThreadTopic}, Touch: true}
		d.ExpectedVersions.Thread = 0
	} else if b.TurnAnalysis.TargetThreadID != nil && (relation == c.RelShifts || relation == c.RelReturns) {
		for _, t := range s.Threads {
			if t.ID == *b.TurnAnalysis.TargetThreadID && t.UserID == d.UserID {
				target = t.ID
				d.ExpectedVersions.Thread = t.Version
				d.Thread = &c.ThreadPatch{ThreadID: target, Touch: true}
				if t.Phase == c.PhasePaused || t.Phase == c.PhaseDormant {
					phase := t.PrevPhase
					d.Thread.Phase = &phase
				}
			}
		}
	} else if s.Focus != nil && s.Focus.UserID == d.UserID {
		target = s.Focus.ID
		d.Thread = &c.ThreadPatch{ThreadID: target, Touch: true}
	}
	if d.Thread == nil {
		return
	}
	if c.ValidID(target) {
		id := target
		d.Session.FocusThread = &id
		if s.Focus != nil && s.Focus.ID != target {
			d.ThreadOps = append(d.ThreadOps, c.ThreadOp{ThreadID: s.Focus.ID, Op: "pause", Reason: "user_shift"})
		}
	}
	// An explicit understanding target must agree with the chosen focus. A new
	// thread has no model-generated ID; its extracted slots attach to creation.
	if b.Understanding.ThreadID == nil || *b.Understanding.ThreadID == target || d.Thread.Create != nil {
		d.Thread.Slots = b.Understanding.Slots
		d.Thread.HypothesisSignals = b.Understanding.Hypotheses
		if b.Understanding.ThreadPhaseProposal != nil && d.Thread.Create == nil && d.Thread.Phase == nil {
			d.Thread.Phase = b.Understanding.ThreadPhaseProposal
		}
	}
	if d.Thread.Create == nil && len(d.Thread.Slots) == 0 && len(d.Thread.HypothesisSignals) == 0 && d.Thread.Phase == nil && relation == c.RelContinues {
		d.Thread = nil
		d.Session.FocusThread = nil
	}
}
