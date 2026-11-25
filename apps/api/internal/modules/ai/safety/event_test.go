package safety

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func TestRiskTransitionPreparesTransactionalMinimalOutboxEvent(t *testing.T) {
	fsm, _ := NewFSM(loadFixture(t))
	turn, user := uuid.New(), uuid.New()
	change := fsm.Advance(fixedTime(), turn, contract.RiskState{Level: contract.RiskLow}, contract.SafetySignal{Level: contract.RiskHigh}, ClearanceEvidence{})
	delta := contract.StateDelta{TurnID: turn, UserID: user}
	if !change.AddToDelta(&delta, []contract.RiskCategory{contract.RiskOther, contract.RiskOther, "invalid"}) {
		t.Fatal("risk escalation was not attached to turn delta")
	}
	if delta.User.Risk == nil || delta.User.Risk.To != contract.RiskHigh || len(delta.Events) != 1 || delta.Events[0].Type != contract.EvRiskLevelChanged {
		t.Fatalf("uncommitted risk proposal mismatch: %+v", delta)
	}
	payload, err := json.Marshal(delta.Events[0].Payload)
	if err != nil || strings.Contains(string(payload), "TEST_HIGH_SIGNAL") || strings.Contains(string(payload), "triggers") || strings.Contains(string(payload), "narrative") {
		t.Fatalf("non-minimal event: %s %v", payload, err)
	}
	var body RiskEventPayload
	if err := json.Unmarshal(payload, &body); err != nil || body.TurnID != turn || len(body.Categories) != 1 || body.Categories[0] != contract.RiskOther || body.FollowUpUntil == nil {
		t.Fatalf("invalid risk outbox payload: %+v %v", body, err)
	}
	if change.AddToDelta(&delta, []contract.RiskCategory{contract.RiskOther}) || len(delta.Events) != 1 {
		t.Fatal("duplicated risk transition on replay")
	}
	delta.TurnID = uuid.New()
	delta.User.Risk = nil
	delta.Events = nil
	if change.AddToDelta(&delta, nil) {
		t.Fatal("attached another turn's risk transition")
	}
}

func TestNoRiskChangeProducesNoStateOrEvent(t *testing.T) {
	fsm, _ := NewFSM(loadFixture(t))
	change := fsm.Advance(fixedTime(), uuid.New(), contract.RiskState{Level: contract.RiskHigh}, contract.SafetySignal{Level: contract.RiskLow}, ClearanceEvidence{})
	delta := contract.StateDelta{TurnID: uuid.New(), UserID: uuid.New()}
	if change.AddToDelta(&delta, nil) || len(delta.Events) != 0 || delta.User.Risk != nil {
		t.Fatal("no-change transition emitted an event")
	}
}
