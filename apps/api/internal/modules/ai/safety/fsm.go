package safety

import (
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// ClearanceEvidence is an externally verified, explicitly collected statement
// about the user's current safety. Silence, topic changes, a model's guess, or
// expiry of a timer are NEVER a clearance signal by themselves.
type ClearanceEvidence struct {
	NoNewSignalsVerified      bool
	UserStatedSafeVerified    bool
	TopicNeutralTurnsVerified int
}

type RiskChange struct {
	State           contract.RiskState
	Transition      *contract.RiskTransition // pass through StateDelta.User.Risk
	FollowUpActive  bool
	RiskConstraints bool
}

type FSM struct{ assets *Assets }

func NewFSM(assets *Assets) (*FSM, error) {
	if assets == nil {
		return nil, ErrAssets
	}
	return &FSM{assets: assets}, nil
}

// Advance is PURE: it never writes the DB, removes historic events or creates
// clinical/consent assumptions. The caller commits Transition and the
// risk_level_changed event atomically with the validated assistant turn.
// No new evidence means the old level is retained (SF-008, ST-010, ST-011).
func (f *FSM) Advance(now time.Time, turnID contract.ID, prior contract.RiskState, signal contract.SafetySignal, proof ClearanceEvidence) RiskChange {
	if f == nil || f.assets == nil {
		return RiskChange{State: prior, RiskConstraints: true, FollowUpActive: true}
	}
	now = now.UTC()
	state := prior
	state.History = append([]contract.RiskTransition(nil), prior.History...)
	if !state.Level.Valid() {
		state.Level = contract.RiskElevated
	}
	if !signal.Level.Valid() {
		signal.Level = contract.RiskElevated
		signal.Degraded = true
	}
	old := state.Level
	next := old
	reason := ""
	if signal.Level.Rank() > old.Rank() {
		next = signal.Level
		reason = "SF-001"
	} else if signal.Level.Rank() <= contract.RiskLow.Rank() && !signal.Degraded && proof.NoNewSignalsVerified && state.LastSignalAt != nil && !now.Before(*state.LastSignalAt) {
		since := now.Sub(*state.LastSignalAt)
		switch old {
		case contract.RiskImminent:
			// No imminent downgrade is specified in the reviewed FSM contract.
		case contract.RiskHigh:
			p := f.assets.policy.RiskFSM.Deescalate.FromHighToElevated
			if since >= time.Duration(p.MinHoursSinceSignal)*time.Hour && proof.UserStatedSafeVerified {
				next = contract.RiskElevated
				reason = "SF-008/high_to_elevated"
			}
		case contract.RiskElevated:
			p := f.assets.policy.RiskFSM.Deescalate.FromElevatedToLow
			// The WP-03 thresholds also require a verified explicit safe
			// statement. Neutral turns alone cannot clear the risk state.
			if since >= time.Duration(p.MinHoursSinceSignal)*time.Hour && proof.UserStatedSafeVerified {
				next = contract.RiskLow
				reason = "SF-008/elevated_to_low"
			}
		}
	}
	if signal.Level.Rank() >= contract.RiskElevated.Rank() {
		stamp := now
		if state.LastSignalAt == nil || now.After(*state.LastSignalAt) {
			state.LastSignalAt = &stamp
		}
		until := now.Add(time.Duration(f.assets.policy.RiskFSM.FollowUpHours) * time.Hour)
		if state.FollowUpUntil == nil || until.After(*state.FollowUpUntil) {
			state.FollowUpUntil = &until
		}
	}
	var change *contract.RiskTransition
	if next != old {
		transition := contract.RiskTransition{At: now, From: old, To: next, Reason: reason, TurnID: turnID}
		change = &transition
		state.History = append(state.History, transition)
		if len(state.History) > 20 {
			state.History = state.History[len(state.History)-20:]
		}
		state.Level = next
		state.Since = now
		if next.Rank() < old.Rank() {
			cleared := reason
			state.ClearedBy = &cleared
		} else {
			state.ClearedBy = nil
		}
	}
	active := state.FollowUpUntil != nil && now.Before(*state.FollowUpUntil) && state.Level.Rank() >= contract.RiskElevated.Rank()
	return RiskChange{State: state, Transition: change, FollowUpActive: active,
		RiskConstraints: active || state.Level.Rank() >= contract.RiskElevated.Rank()}
}
