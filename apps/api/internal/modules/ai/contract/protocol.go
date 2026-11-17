package contract

import (
	"time"
)

type ProtocolStepSpec struct {
	InstanceID         ID          `json:"instance_id"`
	ProtocolID         string      `json:"protocol_id"`
	StepID             string      `json:"step_id"`
	Instruction        string      `json:"instruction"` // bundle text for the Generator
	ExpectedInputs     []string    `json:"expected_inputs"`
	AllowedAdaptations []string    `json:"allowed_adaptations"`
	MustSay            []string    `json:"must_say,omitempty"` // required content ids, e.g. opt_out_reminder
	UI                 *ProtocolUI `json:"ui,omitempty"`
	Offer              *OfferSpec  `json:"offer,omitempty"` // when action=offer
}

type OfferSpec struct {
	ProtocolID string   `json:"protocol_id"`
	Name       string   `json:"name"`
	Why        []string `json:"why"`
	EstMinutes int      `json:"est_minutes"`
	OptOutLine bool     `json:"opt_out_line"`
}

type ProtocolPatch struct { // produced by the engine, applied by the State Committer
	ConsentMessageID *ID                   `json:"consent_message_id,omitempty"`
	InstanceID       ID                    `json:"instance_id"`
	Create           *ProtocolInstanceInit `json:"create,omitempty"`
	NewStatus        *ProtocolStatus       `json:"new_status,omitempty"`
	NewStep          *string               `json:"new_step,omitempty"`
	Inputs           map[string]string     `json:"inputs"`
	AttemptsInc      int                   `json:"attempts_inc"`
	StuckInc         int                   `json:"stuck_inc"`
	StuckReset       bool                  `json:"stuck_reset"`
	Outcome          *string               `json:"outcome,omitempty"`
	ConsentAt        *time.Time            `json:"consent_at,omitempty"`
	FollowUpAt       *time.Time            `json:"follow_up_at,omitempty"`
	Reason           string                `json:"reason"` // rule id
}
