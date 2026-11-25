package contract

import (
	"time"
)

type SafetySignal struct {
	Level             RiskLevel       `json:"level"`
	Categories        []RiskCategory  `json:"categories"`
	Triggers          []SafetyTrigger `json:"triggers"`
	Source            string          `json:"source"`     // l0 | l1 | max(l0,l1) | brain_raise | timeout_default
	Confidence        float32         `json:"confidence"` // L1 only
	ClassifierVersion string          `json:"classifier_version"`
	LexiconVersion    string          `json:"lexicon_version"`
	Degraded          bool            `json:"degraded"` // classifier timed out; level defaulted upward
}

type SafetyTrigger struct {
	Layer  string `json:"layer"`          // l0 | l1
	RuleID string `json:"rule_id"`        // lexicon rule id or classifier label
	Span   *Span  `json:"span,omitempty"` // char offsets only; never the text itself
}

type Span struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type RiskState struct { // inside UserState
	Level         RiskLevel        `json:"level"`
	Since         time.Time        `json:"since"`
	FollowUpUntil *time.Time       `json:"follow_up_until,omitempty"`
	LastSignalAt  *time.Time       `json:"last_signal_at,omitempty"`
	ClearedBy     *string          `json:"cleared_by,omitempty"` // rule id that cleared
	History       []RiskTransition `json:"history,omitempty"`    // capped to last 20
}

type RiskTransition struct {
	At     time.Time `json:"at"`
	From   RiskLevel `json:"from"`
	To     RiskLevel `json:"to"`
	Reason string    `json:"reason"` // rule id | time_decay | user_stated_safe
	TurnID ID        `json:"turn_id"`
}

type CrisisResource struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`    // hotline | text_line | emergency | chat
	Contact    string `json:"contact"` // number / URL / short code
	Locale     string `json:"locale"`
	Hours      string `json:"hours,omitempty"`
	VerifiedOn string `json:"verified_on"`
}

// CrisisPlan is produced by the deterministic Crisis Planner.
type CrisisPlan struct {
	TemplateFamily   string           `json:"template_family"`   // e.g. si_passive_or_unclear
	RequiredElements []string         `json:"required_elements"` // acknowledge|care|ask_safety|resources|invite_continue
	Resources        []CrisisResource `json:"resources"`
	Forbidden        []string         `json:"forbidden"`     // method_details|minimizing|debating_reasons|...
	Buffered         bool             `json:"buffered"`      // always true
	TemplateText     string           `json:"template_text"` // static fallback, bundle-owned
}
