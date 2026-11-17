package safety

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func TestRiskFSMConservativeTransitions(t *testing.T) {
	a := loadFixture(t)
	fsm, err := NewFSM(a)
	if err != nil {
		t.Fatal(err)
	}
	now := fixedTime()
	turn := uuid.New()
	start := fsm.Advance(now, turn, contract.RiskState{Level: contract.RiskNone}, contract.SafetySignal{Level: contract.RiskHigh}, ClearanceEvidence{})
	if start.State.Level != contract.RiskHigh || start.Transition == nil || start.Transition.Reason != "SF-001" || !start.FollowUpActive || !start.RiskConstraints || start.State.LastSignalAt == nil {
		t.Fatalf("did not raise: %+v", start)
	}
	before := start.State
	noProof := fsm.Advance(now.Add(30*time.Hour), turn, before, contract.SafetySignal{Level: contract.RiskNone}, ClearanceEvidence{})
	if noProof.Transition != nil || noProof.State.Level != contract.RiskHigh {
		t.Fatal("deescalated without proof")
	}
	early := fsm.Advance(now.Add(11*time.Hour), turn, before, contract.SafetySignal{Level: contract.RiskNone}, ClearanceEvidence{NoNewSignalsVerified: true, UserStatedSafeVerified: true})
	if early.State.Level != contract.RiskHigh {
		t.Fatal("deescalated before minimum period")
	}
	guarded := fsm.Advance(now.Add(13*time.Hour), turn, before, contract.SafetySignal{Level: contract.RiskNone}, ClearanceEvidence{NoNewSignalsVerified: true, UserStatedSafeVerified: true})
	if guarded.State.Level != contract.RiskElevated || guarded.Transition == nil || guarded.Transition.Reason != "SF-008/high_to_elevated" {
		t.Fatalf("high transition: %+v", guarded)
	}
	still := fsm.Advance(now.Add(23*time.Hour), turn, guarded.State, contract.SafetySignal{Level: contract.RiskNone}, ClearanceEvidence{NoNewSignalsVerified: true, TopicNeutralTurnsVerified: 3})
	if still.State.Level != contract.RiskElevated {
		t.Fatal("early elevated clearance")
	}
	low := fsm.Advance(now.Add(25*time.Hour), turn, guarded.State, contract.SafetySignal{Level: contract.RiskLow}, ClearanceEvidence{NoNewSignalsVerified: true, TopicNeutralTurnsVerified: 3, UserStatedSafeVerified: true})
	if low.State.Level != contract.RiskLow || low.Transition == nil || low.Transition.Reason != "SF-008/elevated_to_low" {
		t.Fatalf("low transition: %+v", low)
	}
	if low.FollowUpActive || low.RiskConstraints {
		t.Fatal("cleared risk retains constraints")
	}
	if start.State.Level != contract.RiskHigh || len(start.State.History) != 1 {
		t.Fatal("mutated prior state")
	}
}
func TestRiskFSMImminentNeverDecaysAndDegradedNeverClears(t *testing.T) {
	fsm, _ := NewFSM(loadFixture(t))
	now := fixedTime()
	stamp := now.Add(-48 * time.Hour)
	prior := contract.RiskState{Level: contract.RiskImminent, LastSignalAt: &stamp, Since: stamp}
	proof := ClearanceEvidence{NoNewSignalsVerified: true, UserStatedSafeVerified: true, TopicNeutralTurnsVerified: 4}
	imminent := fsm.Advance(now, uuid.New(), prior, contract.SafetySignal{Level: contract.RiskNone}, proof)
	if imminent.Transition != nil || imminent.State.Level != contract.RiskImminent {
		t.Fatal("imminent downgraded")
	}
	prior.Level = contract.RiskHigh
	degraded := fsm.Advance(now, uuid.New(), prior, contract.SafetySignal{Level: contract.RiskElevated, Degraded: true}, proof)
	if degraded.State.Level != contract.RiskHigh {
		t.Fatal("classifier outage cleared risk")
	}
	raised := fsm.Advance(now, uuid.New(), prior, contract.SafetySignal{Level: contract.RiskImminent}, proof)
	if raised.State.Level != contract.RiskImminent || raised.Transition == nil {
		t.Fatal("explicit escalation ignored")
	}
}
func TestRiskFSMHistoryViewBoundedWithoutDeletingOldEvidence(t *testing.T) {
	fsm, _ := NewFSM(loadFixture(t))
	prior := contract.RiskState{Level: contract.RiskNone}
	for i := 0; i < 25; i++ {
		prior.History = append(prior.History, contract.RiskTransition{Reason: "stored_in_append_only_turn_events"})
	}
	change := fsm.Advance(fixedTime(), uuid.New(), prior, contract.SafetySignal{Level: contract.RiskHigh}, ClearanceEvidence{})
	if len(prior.History) != 25 || len(change.State.History) != 20 {
		t.Fatal("history view was not bounded safely")
	}
}
