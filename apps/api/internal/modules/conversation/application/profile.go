package application

// Consumer-owned view of explicit onboarding selections. Composition maps the
// user module's validated profile into this read-only conversation port.
type OnboardingAnswers struct {
	Age            string   `json:"age,omitempty"`
	Identity       string   `json:"identity,omitempty"`
	Therapist      string   `json:"therapist,omitempty"`
	Faith          string   `json:"faith,omitempty"`
	Neurodivergent string   `json:"neurodivergent,omitempty"`
	Mood           *int     `json:"mood,omitempty"`
	Concerns       []string `json:"concerns,omitempty"`
	Focus          []string `json:"focus,omitempty"`
	Feelings       []string `json:"feelings,omitempty"`
	Relationships  []string `json:"relationships,omitempty"`
	Interests      []string `json:"interests,omitempty"`
	Strengths      []string `json:"strengths,omitempty"`
	Barriers       []string `json:"barriers,omitempty"`
	Recharge       []string `json:"recharge,omitempty"`
}
