package contract

type ValidationReport struct {
	Passed     bool               `json:"passed"`
	Violations []Violation        `json:"violations"`
	Scores     map[string]float32 `json:"scores,omitempty"` // classifier outputs
	Extracted  Extracted          `json:"extracted"`
	Action     ValidationAction   `json:"action"`
}

type ValidationAction string

const (
	VAAccept     ValidationAction = "accept"
	VARegenerate ValidationAction = "regenerate"
	VAFallback   ValidationAction = "fallback"
	VACrisis     ValidationAction = "crisis_reroute"
)

type Violation struct {
	Code     string `json:"code"`             // OV-xxx, see §10.3
	Severity string `json:"severity"`         // block | repair | warn
	Detail   string `json:"detail,omitempty"` // never contains user text
}

type Extracted struct {
	QuestionText      *string     `json:"question_text,omitempty"`
	QuestionCount     int         `json:"question_count"`
	ClaimsAboutUser   []UserClaim `json:"claims_about_user,omitempty"`
	ReferencedContext []ID        `json:"referenced_context,omitempty"`
	Sentinel          string      `json:"sentinel,omitempty"` // PLAN_UNSAFE
}

type UserClaim struct {
	Text      string    `json:"text"`
	Source    *ID       `json:"source,omitempty"` // memory id referenced, if any
	Hedged    bool      `json:"hedged"`
	Epistemic Epistemic `json:"epistemic"`
}
