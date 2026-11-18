package eval

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

type Diagnostic struct {
	Code      string `json:"code"`
	Assertion string `json:"assertion"`
	Severity  string `json:"severity"`
}
type TurnReport struct {
	Index            int          `json:"index"`
	Session          int          `json:"session"`
	Passed           bool         `json:"passed"`
	Assertions       int          `json:"assertions"`
	PassedAssertions int          `json:"passed_assertions"`
	Diagnostics      []Diagnostic `json:"diagnostics,omitempty"`
}
type ScenarioReport struct {
	ID           string       `json:"id"`
	Tags         []string     `json:"tags,omitempty"`
	Passed       bool         `json:"passed"`
	ReviewStatus string       `json:"review_status"`
	Turns        []TurnReport `json:"turns"`
}
type Totals struct {
	Scenarios        int `json:"scenarios"`
	PassedScenarios  int `json:"passed_scenarios"`
	FailedScenarios  int `json:"failed_scenarios"`
	Turns            int `json:"turns"`
	PassedTurns      int `json:"passed_turns"`
	Assertions       int `json:"assertions"`
	PassedAssertions int `json:"passed_assertions"`
	Failures         int `json:"failures"`
	Blockers         int `json:"blockers"`
}
type Report struct {
	Schema        string            `json:"schema"`
	Method        string            `json:"method"`
	SyntheticOnly bool              `json:"synthetic_only"`
	Passed        bool              `json:"passed"`
	Totals        Totals            `json:"totals"`
	Metrics       map[string]Metric `json:"metrics"`
	Scenarios     []ScenarioReport  `json:"scenarios"`
	Regressions   []Regression      `json:"regressions,omitempty"`
	ReviewQueue   []ReviewFlag      `json:"review_queue,omitempty"`
}
type Metric struct {
	Passed int     `json:"passed"`
	Total  int     `json:"total"`
	Rate   float64 `json:"rate"`
}

// Runner fails closed on missing components and unverified judge calibration.
// There is no implicit fallback to a mock when a real driver fails.
type Runner struct {
	Executor Executor
	Reviewer ReviewHook
	Labels   LabelVerifier
}

func (r Runner) Run(ctx context.Context, scenarios []Scenario) Report {
	report := Report{Schema: "eval_report.v1", Method: "external_executor", SyntheticOnly: true, Passed: true, Metrics: map[string]Metric{}, Scenarios: []ScenarioReport{}}
	if len(scenarios) == 0 {
		report.Passed = false
		report.Totals.Blockers++
		return report
	}
	seen := map[string]bool{}
	for _, s := range scenarios {
		result := ScenarioReport{ID: s.ID, Tags: append([]string(nil), s.Tags...), ReviewStatus: "synthetic_unreviewed", Passed: true, Turns: []TurnReport{}}
		if s.Clinician != nil {
			result.ReviewStatus = "metadata_declared_unverified"
			if r.Labels != nil {
				verified, err := r.Labels.VerifySyntheticLabels(ctx, s)
				if err == nil && verified {
					result.ReviewStatus = "externally_verified_synthetic_labels"
				}
			}
		}
		report.Totals.Scenarios++
		env, err := newEnvironment(s)
		if !s.SyntheticOnly || s.Schema != "golden.v1" || !identifier.MatchString(s.ID) || seen[s.ID] || r.Executor == nil || err != nil || len(s.Sequence()) == 0 {
			result.Passed = false
			result.Turns = append(result.Turns, TurnReport{Index: 0, Diagnostics: []Diagnostic{{Code: "EVAL_SETUP", Assertion: "fixture", Severity: "blocker"}}})
		} else {
			seen[s.ID] = true
			for _, step := range s.Sequence() {
				if ctx.Err() != nil {
					result.Passed = false
					result.Turns = append(result.Turns, TurnReport{Index: step.Index, Session: step.Session, Diagnostics: []Diagnostic{{Code: "EVAL_CANCELLED", Assertion: "execution", Severity: "blocker"}}})
					break
				}
				if step.NewSession {
					env.CurrentSession = step.Session
					if step.Gap != "" {
						env.Clock = env.Clock.Add(gapDuration(step.Gap))
					}
				}
				o, e := r.Executor.Execute(ctx, env, step)
				tr := TurnReport{Index: step.Index, Session: step.Session, Passed: true}
				if e != nil {
					tr.Passed = false
					code := "EVAL_EXECUTION"
					if errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded) {
						code = "EVAL_CANCELLED"
					}
					tr.Diagnostics = append(tr.Diagnostics, Diagnostic{Code: code, Assertion: "execution", Severity: "blocker"})
				} else if o.UserID != env.UserID || o.TurnID == "" || !validOwnerScope(o, env.UserID) {
					tr.Passed = false
					tr.Diagnostics = append(tr.Diagnostics, Diagnostic{Code: "EVAL_OWNER", Assertion: "isolation", Severity: "blocker"})
				} else {
					if reason := needsReview(step.Turn.Expect, o); reason != "" {
						report.ReviewQueue = append(report.ReviewQueue, ReviewFlag{Scenario: s.ID, Turn: step.Index, Reason: reason, Status: "pending"})
						if r.Reviewer != nil {
							decision, reviewErr := r.Reviewer.Review(ctx, ReviewRequest{ScenarioID: s.ID, TurnIndex: step.Index, SyntheticOnly: true, Observation: o})
							if reviewErr == nil && decision.VerifiedCalibration && decision.CalibrationID != "" {
								o.Judge = &JudgeObserved{Scores: decision.Scores, CalibrationID: decision.CalibrationID, VerifiedCalibrated: true}
							}
						}
					}
					checkTurn(step.Turn.Expect, o, &tr, report.Metrics)
					env.Past = append(env.Past, o)
					if o.State != nil {
						env.LastState = o.State
					}
				}
				report.Totals.Turns++
				report.Totals.Assertions += tr.Assertions
				report.Totals.PassedAssertions += tr.PassedAssertions
				for _, d := range tr.Diagnostics {
					if d.Severity == "blocker" {
						report.Totals.Blockers++
					} else {
						report.Totals.Failures++
					}
				}
				if !tr.Passed {
					result.Passed = false
				} else {
					report.Totals.PassedTurns++
				}
				result.Turns = append(result.Turns, tr)
			}
		}
		if !result.Passed && len(result.Turns) == 1 && result.Turns[0].Assertions == 0 && result.Turns[0].Index == 0 && len(result.Turns[0].Diagnostics) > 0 && (result.Turns[0].Diagnostics[0].Code == "EVAL_SETUP" || result.Turns[0].Diagnostics[0].Code == "EVAL_CANCELLED") {
			report.Totals.Blockers++
		}
		if !result.Passed {
			report.Passed = false
			report.Totals.FailedScenarios++
		} else {
			report.Totals.PassedScenarios++
		}
		report.Scenarios = append(report.Scenarios, result)
	}
	for k, v := range report.Metrics {
		if v.Total > 0 {
			v.Rate = float64(v.Passed) / float64(v.Total)
		}
		report.Metrics[k] = v
	}
	sort.Slice(report.Scenarios, func(i, j int) bool { return report.Scenarios[i].ID < report.Scenarios[j].ID })
	return report
}
func gapDuration(g string) time.Duration {
	if strings.HasSuffix(g, "d") {
		var n int
		for _, r := range strings.TrimSuffix(g, "d") {
			n = n*10 + int(r-'0')
		}
		return time.Duration(n) * 24 * time.Hour
	}
	d, _ := time.ParseDuration(g)
	return d
}
func validOwnerScope(o Observation, user string) bool {
	for _, v := range o.Retrieval {
		if v.UserID != user {
			return false
		}
	}
	if o.State != nil {
		for _, v := range o.State.Memory {
			if v.UserID != user {
				return false
			}
		}
	}
	return true
}

func mark(t *TurnReport, metrics map[string]Metric, metric, path, code string, success, available bool) {
	t.Assertions++
	m := metrics[metric]
	m.Total++
	if available && success {
		m.Passed++
		t.PassedAssertions++
	} else {
		t.Passed = false
		severity := "failure"
		if !available {
			severity = "blocker"
			code = "EVAL_MISSING_EVIDENCE"
		}
		if len(t.Diagnostics) < 100 {
			t.Diagnostics = append(t.Diagnostics, Diagnostic{Code: code, Assertion: path, Severity: severity})
		}
	}
	metrics[metric] = m
}
