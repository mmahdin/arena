package eval

import (
	"context"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// ReviewFlag carries no message, model output, user ID, or private trace data.
type ReviewFlag struct {
	Scenario string `json:"scenario"`
	Turn     int    `json:"turn"`
	Reason   string `json:"reason"`
	Status   string `json:"status"`
}

// ReviewRequest is synthetic-only. A deployment must supply restricted human
// access, auditing and calibration provenance; WP-15 does not implement a
// production review queue or assert that a test fixture is clinician-approved.
type ReviewRequest struct {
	ScenarioID    string
	TurnIndex     int
	SyntheticOnly bool
	Observation   Observation
}
type ReviewDecision struct {
	Scores              map[string]float64
	CalibrationID       string
	VerifiedCalibration bool
	// Human labels are deliberately not persisted to an untrusted report.
}
type ReviewHook interface {
	Review(context.Context, ReviewRequest) (ReviewDecision, error)
}

func needsReview(expect Expect, obs Observation) string {
	if len(expect.Output.Judge) > 0 {
		return "rubric_review"
	}
	if obs.Route == string(c.RouteCrisis) || obs.Safety != nil && obs.Safety.Level.Rank() >= c.RiskHigh.Rank() {
		return "safety_review"
	}
	if obs.Validator != nil && (!obs.Validator.Passed || obs.Validator.Action != c.VAAccept) {
		return "validator_review"
	}
	return ""
}

// LabelVerifier is an optional external trust boundary for signed clinician
// annotations on synthetic goldens. Unverified YAML clinician metadata never
// establishes label approval. The verifier must audit reviewer identity,
// label version, agreement and the approved scenario digest outside WP-15.
type LabelVerifier interface {
	VerifySyntheticLabels(context.Context, Scenario) (bool, error)
}
