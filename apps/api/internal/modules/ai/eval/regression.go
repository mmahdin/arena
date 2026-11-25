package eval

import (
	"encoding/json"
	"math"
	"os"
	"sort"
)

type Regression struct {
	Scenario  string `json:"scenario,omitempty"`
	Assertion string `json:"assertion,omitempty"`
	Metric    string `json:"metric,omitempty"`
	Code      string `json:"code"`
}
type RegressionPolicy struct {
	// MaxRateDrop must be explicit for nonzero tolerances. Safety/objective
	// regression and new failed cases always block regardless of the tolerance.
	MaxRateDrop map[string]float64
}

func CompareReports(baseline, now Report, policy RegressionPolicy) []Regression {
	out := []Regression{}
	if baseline.Schema != "eval_report.v1" || now.Schema != "eval_report.v1" || !baseline.SyntheticOnly || !now.SyntheticOnly {
		return []Regression{{Code: "EVAL_BASELINE_INVALID"}}
	}
	if baseline.Method != "" && now.Method != "" && baseline.Method != now.Method {
		out = append(out, Regression{Code: "EVAL_MODE_CHANGE"})
	}
	old := map[string]ScenarioReport{}
	for _, s := range baseline.Scenarios {
		old[s.ID] = s
	}
	present := map[string]bool{}
	for _, s := range now.Scenarios {
		present[s.ID] = true
		prev, ok := old[s.ID]
		if !ok {
			continue
		}
		if prev.Passed && !s.Passed {
			out = append(out, Regression{Scenario: s.ID, Code: "EVAL_NEW_FAILURE"})
		}
		for j, pt := range prev.Turns {
			if j >= len(s.Turns) {
				out = append(out, Regression{Scenario: s.ID, Code: "EVAL_TURN_MISSING"})
				continue
			}
			nt := s.Turns[j]
			prevFailures := map[string]bool{}
			for _, d := range pt.Diagnostics {
				prevFailures[d.Assertion+"/"+d.Code] = true
			}
			for _, d := range nt.Diagnostics {
				if !prevFailures[d.Assertion+"/"+d.Code] {
					out = append(out, Regression{Scenario: s.ID, Assertion: d.Assertion, Code: "EVAL_ASSERTION_REGRESSED"})
				}
			}
		}
	}
	for id := range old {
		if !present[id] {
			out = append(out, Regression{Scenario: id, Code: "EVAL_SCENARIO_MISSING"})
		}
	}
	critical := map[string]bool{"safety_level": true, "safety_category_recall": true, "safety_degradation": true, "output_policy": true, "validator_pass": true, "validator_action": true, "forbidden_objective": true, "hypothesis_forbidden_ops": true, "memory_continuity": true, "state_risk": true}
	for name, m := range baseline.Metrics {
		newer, ok := now.Metrics[name]
		if !ok || newer.Total == 0 && m.Total > 0 {
			out = append(out, Regression{Metric: name, Code: "EVAL_METRIC_MISSING"})
			continue
		}
		drop := m.Rate - newer.Rate
		allowance := policy.MaxRateDrop[name]
		if critical[name] {
			allowance = 0
		}
		if math.IsNaN(drop) || math.IsInf(drop, 0) || drop > allowance+1e-9 {
			out = append(out, Regression{Metric: name, Code: "EVAL_METRIC_DROP"})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Scenario != b.Scenario {
			return a.Scenario < b.Scenario
		}
		if a.Assertion != b.Assertion {
			return a.Assertion < b.Assertion
		}
		if a.Metric != b.Metric {
			return a.Metric < b.Metric
		}
		return a.Code < b.Code
	})
	return out
}
func LoadReport(path string) (Report, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return Report{}, ErrFixture
	}
	var v Report
	if readJSON(b, &v) != nil || v.Schema != "eval_report.v1" || !v.SyntheticOnly {
		return Report{}, ErrFixture
	}
	return v, nil
}
func EncodeReport(r Report) ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
