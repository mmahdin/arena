package eval

import (
	"context"
	"encoding/json"
	"os"
	"reflect"

	"github.com/google/uuid"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// TraceFixture is an explicitly synthetic, offline wrapper for the §12.1
// turn_trace.v1 payload. Raw production traces are never loaded by the CLI.
// Replay cannot dereference text_ref or encrypted snapshots: missing input is
// an evaluation blocker rather than an invented 'pass'.
type TraceFixture struct {
	Schema        string  `json:"schema"`
	ScenarioID    string  `json:"scenario_id"`
	SyntheticOnly bool    `json:"synthetic_only"`
	Traces        []Trace `json:"traces"`
}
type Trace struct {
	Schema    string `json:"schema"`
	TurnID    string `json:"turn_id"`
	UserID    string `json:"user_id"`
	Origin    string `json:"origin"`
	Route     string `json:"route"`
	RouteRule string `json:"route_rule"`
	Versions  struct {
		Bundle  string            `json:"bundle"`
		Models  map[string]string `json:"models"`
		Prompts map[string]string `json:"prompts"`
	} `json:"versions"`
	Steps           []TraceStep `json:"steps"`
	ContextSnapshot struct {
		MessageIDs []string `json:"message_ids"`
		MemoryIDs  []string `json:"memory_ids"`
		ThreadID   *string  `json:"thread_id"`
	} `json:"context_snapshot"`
	DegradedFlags []string `json:"degraded_flags"`
}
type TraceStep struct {
	Name       string              `json:"name"`
	MS         int64               `json:"ms"`
	Degraded   bool                `json:"degraded"`
	ErrorCode  string              `json:"error_code"`
	RuleID     string              `json:"rule_id"`
	Signal     *c.SafetySignal     `json:"signal"`
	Items      []string            `json:"items"`
	Decision   *c.BrainDecision    `json:"decision"`
	Report     *c.ValidationReport `json:"report"`
	Sentinel   *string             `json:"sentinel"`
	Applied    *bool               `json:"applied"`
	Rejections []c.DeltaRejection  `json:"rejections"`
}

func LoadTraceFixture(path string) (TraceFixture, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return TraceFixture{}, ErrReplay
	}
	var f TraceFixture
	// Unknown fields inside the versioned trace are tolerated: §12.1 permits
	// step-specific metadata, but only the typed allowlist enters evaluation.
	if len(b) > 4<<20 || json.Unmarshal(b, &f) != nil || f.Schema != "eval_trace_fixture.v1" || !f.SyntheticOnly || !identifier.MatchString(f.ScenarioID) || len(f.Traces) == 0 {
		return TraceFixture{}, ErrReplay
	}
	for _, t := range f.Traces {
		if t.Schema != "turn_trace.v1" || uuid.Validate(t.UserID) != nil || uuid.Validate(t.TurnID) != nil || t.Origin != "user" || len(t.Steps) > 80 {
			return TraceFixture{}, ErrReplay
		}
	}
	return f, nil
}

// TraceExecutor maps only observed data; the recorder's metadata-only format
// cannot provide brain decisions, generator text, semantic judgments or state.
type TraceExecutor struct{ Fixture TraceFixture }

func (r TraceExecutor) Execute(ctx context.Context, e *Environment, t SequenceTurn) (Observation, error) {
	if ctx.Err() != nil {
		return Observation{}, ctx.Err()
	}
	if e.Scenario.ID != r.Fixture.ScenarioID || t.Index >= len(r.Fixture.Traces) {
		return Observation{}, ErrReplay
	}
	trace := r.Fixture.Traces[t.Index]
	if e.UserID != trace.UserID || trace.Origin != "user" || trace.Schema != "turn_trace.v1" {
		return Observation{}, ErrReplay
	}
	allowedMemory := map[string]bool{}
	for _, item := range e.Scenario.Setup.Memory {
		allowedMemory[item.ID] = true
	}
	for _, prev := range e.Past {
		if prev.State != nil {
			for _, item := range prev.State.Memory {
				allowedMemory[item.ID] = true
			}
		}
	}
	for _, id := range trace.ContextSnapshot.MemoryIDs {
		if !allowedMemory[id] {
			return Observation{}, ErrReplay
		}
	}
	out := Observation{UserID: trace.UserID, TurnID: trace.TurnID, Route: trace.Route, Trace: &TraceDetails{Bundle: trace.Versions.Bundle, Models: trace.Versions.Models, Prompts: trace.Versions.Prompts}}
	degraded := len(trace.DegradedFlags) > 0
	out.Degraded = &degraded
	seen := map[string]bool{}
	for _, step := range trace.Steps {
		if seen[step.Name] && (step.Name == "brain" || step.Name == "safety" || step.Name == "validator") {
			return Observation{}, ErrReplay
		}
		seen[step.Name] = true
		if step.Degraded {
			degraded = true
		}
		switch step.Name {
		case "safety":
			if step.Signal != nil {
				out.Safety = step.Signal
				if step.Signal.Degraded {
					degraded = true
				}
			}
		case "retrieval":
			out.RetrievalObserved = true
			for _, id := range step.Items {
				if uuid.Validate(id) != nil {
					return Observation{}, ErrReplay
				}
				out.Retrieval = append(out.Retrieval, Retrieved{ID: id, UserID: e.UserID})
			}
		case "brain":
			out.Brain = step.Decision
		case "generator":
			if step.Sentinel != nil {
				out.Generator = &Generation{Sentinel: *step.Sentinel}
			}
		case "validator":
			out.Validator = step.Report
		}
	}
	// A trace recording no degraded flags is insufficient proof that a
	// degraded path was absent; individual step metadata is still preserved.
	out.Degraded = &degraded
	return out, nil
}

type TraceDifference struct {
	Step string `json:"step"`
	Code string `json:"code"`
}

// CompareTraceVersions detects versioned, rule and decision drift without ever
// printing a decision body, prompt or sensitive identifier.
func CompareTraceVersions(before, after Trace) []TraceDifference {
	diff := []TraceDifference{}
	add := func(step, code string) { diff = append(diff, TraceDifference{Step: step, Code: code}) }
	if before.Schema != "turn_trace.v1" || after.Schema != "turn_trace.v1" || before.UserID != after.UserID || before.TurnID != after.TurnID {
		add("trace", "EVAL_TRACE_IDENTITY")
		return diff
	}
	if before.Versions.Bundle != after.Versions.Bundle || !reflect.DeepEqual(before.Versions.Prompts, after.Versions.Prompts) || !reflect.DeepEqual(before.Versions.Models, after.Versions.Models) {
		add("versions", "EVAL_VERSION_CHANGE")
	}
	if before.Route != after.Route {
		add("route", "EVAL_ROUTE_CHANGE")
	}
	old := map[string]TraceStep{}
	for _, v := range before.Steps {
		old[v.Name] = v
	}
	for _, step := range after.Steps {
		previous, ok := old[step.Name]
		if !ok {
			add(step.Name, "EVAL_STEP_ADDED")
			continue
		}
		if previous.Degraded != step.Degraded || previous.ErrorCode != step.ErrorCode || previous.RuleID != step.RuleID {
			add(step.Name, "EVAL_STEP_CHANGE")
		}
		switch step.Name {
		case "brain":
			if !reflect.DeepEqual(previous.Decision, step.Decision) {
				add(step.Name, "EVAL_DECISION_CHANGE")
			}
		case "validator":
			if !reflect.DeepEqual(previous.Report, step.Report) {
				add(step.Name, "EVAL_VALIDATION_CHANGE")
			}
		case "safety":
			if !reflect.DeepEqual(previous.Signal, step.Signal) {
				add(step.Name, "EVAL_SAFETY_CHANGE")
			}
		case "retrieval":
			if !reflect.DeepEqual(previous.Items, step.Items) {
				add(step.Name, "EVAL_RETRIEVAL_CHANGE")
			}
		}
		delete(old, step.Name)
	}
	for name := range old {
		add(name, "EVAL_STEP_MISSING")
	}
	sortDifferences(diff)
	return diff
}
func sortDifferences(xs []TraceDifference) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && (xs[j].Step < xs[j-1].Step || xs[j].Step == xs[j-1].Step && xs[j].Code < xs[j-1].Code); j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
}
