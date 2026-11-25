package eval

import (
	"context"
	"reflect"
	"sort"
)

// CandidateResult compares a fresh synthetic run through an injected candidate
// executor against synthetic capture for the same scenario. No model calls or
// data extraction happen inside WP-15. Missing captured stages are blockers.
type CandidateResult struct {
	Report Report            `json:"report"`
	Drift  []TraceDifference `json:"drift"`
}

func ReplayAgainst(ctx context.Context, scenario Scenario, original TraceFixture, candidate Executor) (CandidateResult, error) {
	if !scenario.SyntheticOnly || scenario.ID != original.ScenarioID || !original.SyntheticOnly || len(scenario.Sequence()) != len(original.Traces) || candidate == nil {
		return CandidateResult{}, ErrReplay
	}
	env, err := newEnvironment(scenario)
	if err != nil {
		return CandidateResult{}, ErrReplay
	}
	for _, t := range original.Traces {
		if t.UserID != env.UserID || t.Schema != "turn_trace.v1" {
			return CandidateResult{}, ErrReplay
		}
	}
	drift := []TraceDifference{}
	witness := ExecuteFunc(func(ctx context.Context, env *Environment, turn SequenceTurn) (Observation, error) {
		baseline, e := TraceExecutor{Fixture: original}.Execute(ctx, env, turn)
		if e != nil {
			return Observation{}, e
		}
		fresh, e := candidate.Execute(ctx, env, turn)
		if e != nil {
			return Observation{}, e
		}
		if fresh.UserID != baseline.UserID || fresh.TurnID != baseline.TurnID {
			drift = append(drift, TraceDifference{Step: "identity", Code: "EVAL_TRACE_IDENTITY"})
			return Observation{}, ErrReplay
		}
		check := func(name, code string, same, available bool) {
			if !available {
				drift = append(drift, TraceDifference{Step: name, Code: "EVAL_COMPARE_INCOMPLETE"})
			} else if !same {
				drift = append(drift, TraceDifference{Step: name, Code: code})
			}
		}
		check("route", "EVAL_ROUTE_CHANGE", baseline.Route == fresh.Route, baseline.Route != "" && fresh.Route != "")
		check("safety", "EVAL_SAFETY_CHANGE", reflect.DeepEqual(baseline.Safety, fresh.Safety), baseline.Safety != nil && fresh.Safety != nil)
		check("brain", "EVAL_DECISION_CHANGE", reflect.DeepEqual(baseline.Brain, fresh.Brain), baseline.Brain != nil && fresh.Brain != nil)
		check("validator", "EVAL_VALIDATION_CHANGE", reflect.DeepEqual(baseline.Validator, fresh.Validator), baseline.Validator != nil && fresh.Validator != nil)
		// The §12.1 trace contains only text_ref, not a generator draft. It cannot
		// establish textual equality and must always carry a replay blocker.
		drift = append(drift, TraceDifference{Step: "generator", Code: "EVAL_COMPARE_INCOMPLETE"})
		return fresh, nil
	})
	report := Runner{Executor: witness}.Run(ctx, []Scenario{scenario})
	sort.Slice(drift, func(i, j int) bool {
		if drift[i].Step != drift[j].Step {
			return drift[i].Step < drift[j].Step
		}
		return drift[i].Code < drift[j].Code
	})
	for _, d := range drift {
		if d.Code == "EVAL_COMPARE_INCOMPLETE" || d.Code == "EVAL_TRACE_IDENTITY" {
			report.Totals.Blockers++
			report.Passed = false
		} else {
			report.Totals.Failures++
			report.Passed = false
		}
	}
	return CandidateResult{Report: report, Drift: drift}, nil
}
