package eval

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func golden(t *testing.T, id string) Scenario {
	t.Helper()
	s, e := LoadFile(filepath.Join("testdata", "golden", id+".yaml"))
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func recorded(t *testing.T, id string) Recorded {
	t.Helper()
	r, e := LoadRecorded(filepath.Join("testdata", "observations", id+".json"))
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestSyntheticGoldenSuites(t *testing.T) {
	for _, id := range []string{"anaphora", "longitudinal", "escalation"} {
		t.Run(id, func(t *testing.T) {
			s := golden(t, id)
			r := recorded(t, id)
			report := Runner{Executor: NewRecordedExecutor(r)}.Run(context.Background(), []Scenario{s})
			if !report.Passed || report.Totals.PassedTurns != len(s.Sequence()) || report.Totals.Assertions < 5 {
				b, _ := EncodeReport(report)
				t.Fatalf("golden failed: %s", b)
			}
			if id == "escalation" && len(report.ReviewQueue) != 2 {
				t.Fatal("expected safety review hooks")
			}
		})
	}
}
func TestStrictGoldenValidation(t *testing.T) {
	base := []byte("schema: golden.v1\nid: sample\nsynthetic_only: true\nturn:\n  text: Synthetic hello\n  expect: {route: normal}\n")
	if _, e := Load(base); e != nil {
		t.Fatal(e)
	}
	for _, replacement := range []string{
		strings.Replace(string(base), "synthetic_only: true", "synthetic_only: false", 1),
		strings.Replace(string(base), "route: normal", "brain: {new_unknown_rule: true}", 1),
		strings.Replace(string(base), "route: normal", "output: {must_not: [magic_classifier]}", 1),
		strings.Replace(string(base), "route: normal", "output: {judge: {warmth: '<=4'}}", 1),
		strings.Replace(string(base), "route: normal", "route: nonsensical", 1),
		strings.Replace(string(base), "sample", "private identity@example.org", 1),
		string(base) + "unexpected: secret\n",
		string(base) + "---\nschema: golden.v1\n",
	} {
		if _, e := Load([]byte(replacement)); !errors.Is(e, ErrFixture) {
			t.Fatal("accepted invalid assertions or synthetic mode")
		}
	}
}
func TestMultisessionStateCarry(t *testing.T) {
	s := golden(t, "longitudinal")
	r := recorded(t, "longitudinal")
	calls := 0
	driver := ExecuteFunc(func(ctx context.Context, e *Environment, turn SequenceTurn) (Observation, error) {
		if len(e.Past) != calls {
			return Observation{}, errors.New("past not carried")
		}
		if calls == 2 && (turn.Session != 1 || !turn.NewSession || e.CurrentSession != 1) {
			return Observation{}, errors.New("session boundary lost")
		}
		o := r.Turns[calls]
		calls++
		return o, nil
	})
	report := Runner{Executor: driver}.Run(context.Background(), []Scenario{s})
	if !report.Passed || calls != 3 {
		t.Fatalf("carry failed: %+v", report.Totals)
	}
}
func TestIsolationAndExecutionFailures(t *testing.T) {
	s := golden(t, "anaphora")
	r := recorded(t, "anaphora")
	r.Turns[0].Retrieval[0].UserID = "88888888-8888-4888-8888-888888888888"
	result := Runner{Executor: NewRecordedExecutor(r)}.Run(context.Background(), []Scenario{s})
	if result.Passed || result.Scenarios[0].Turns[0].Diagnostics[0].Code != "EVAL_OWNER" {
		t.Fatal("cross-user retrieval not rejected")
	}
	fail := Runner{Executor: ExecuteFunc(func(context.Context, *Environment, SequenceTurn) (Observation, error) {
		return Observation{}, errors.New("private report - fake@example.invalid")
	})}.Run(context.Background(), []Scenario{s})
	b, _ := json.Marshal(fail)
	if fail.Passed || strings.Contains(string(b), "fake@example.invalid") {
		t.Fatal("error leaked into report")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	v := Runner{Executor: NewRecordedExecutor(recorded(t, "anaphora"))}.Run(ctx, []Scenario{s})
	if v.Passed {
		t.Fatal("cancelled run passed")
	}
}
func TestNoSensitiveTextInReports(t *testing.T) {
	s := golden(t, "anaphora")
	r := recorded(t, "anaphora")
	r.Turns[0].Generator.Text = "private-user-name private@domain.invalid PASSWORD"
	r.Turns[0].Validator.Passed = false
	out := Runner{Executor: NewRecordedExecutor(r)}.Run(context.Background(), []Scenario{s})
	b, _ := EncodeReport(out)
	for _, token := range []string{"PASSWORD", "private-user-name", "private@domain.invalid", r.UserID, r.Turns[0].TurnID} {
		if strings.Contains(string(b), token) {
			t.Fatal("report leaked private payload")
		}
	}
	if out.Passed {
		t.Fatal("unsafe output report incorrectly passed")
	}
}
func TestMissingReplayEvidenceCannotPass(t *testing.T) {
	s := golden(t, "anaphora")
	r := recorded(t, "anaphora")
	r.Turns[0].Validator = nil
	r.Turns[0].RetrievalStatusesObserved = false
	out := Runner{Executor: NewRecordedExecutor(r)}.Run(context.Background(), []Scenario{s})
	if out.Passed || out.Totals.Blockers == 0 {
		t.Fatal("missing evidence passed")
	}
}
func TestClinicianMetadataNeverMeansApproval(t *testing.T) {
	s := golden(t, "anaphora")
	s.Clinician = &Clinician{Author: "synthetic reviewer", ReviewedOn: "2026-01-12", LabelsAgreement: "3/3"}
	report := Runner{Executor: NewRecordedExecutor(recorded(t, "anaphora"))}.Run(context.Background(), []Scenario{s})
	if !report.Passed || report.Scenarios[0].ReviewStatus != "metadata_declared_unverified" {
		t.Fatal("incorrect clinical label treatment")
	}
}

type calibratedReview struct{}

func (calibratedReview) Review(context.Context, ReviewRequest) (ReviewDecision, error) {
	return ReviewDecision{Scores: map[string]float64{"warmth": 4.5}, CalibrationID: "reviewed-rubric-v1", VerifiedCalibration: true}, nil
}
func TestReviewRubricBlockedUntilExternalCalibration(t *testing.T) {
	s := golden(t, "anaphora")
	s.Turn.Expect.Output.Judge = map[string]string{"warmth": ">=4"}
	r := recorded(t, "anaphora")
	r.Turns[0].Judge = &JudgeObserved{Scores: map[string]float64{"warmth": 5}}
	report := Runner{Executor: NewRecordedExecutor(r)}.Run(context.Background(), []Scenario{s})
	if report.Passed || report.Totals.Blockers == 0 || len(report.ReviewQueue) != 1 {
		t.Fatal("unreviewed judge considered verified")
	}
	passed := Runner{Executor: NewRecordedExecutor(r), Reviewer: calibratedReview{}}.Run(context.Background(), []Scenario{s})
	if !passed.Passed {
		t.Fatalf("verified review hook not used: %+v", passed.Scenarios)
	}
}
func TestRegressionNewFailuresAndDrops(t *testing.T) {
	s := golden(t, "anaphora")
	r := recorded(t, "anaphora")
	base := Runner{Executor: NewRecordedExecutor(r)}.Run(context.Background(), []Scenario{s})
	r.Turns[0].Brain.Decision.Primary = c.ObjProvideGuidance
	candidate := Runner{Executor: NewRecordedExecutor(r)}.Run(context.Background(), []Scenario{s})
	regress := CompareReports(base, candidate, RegressionPolicy{MaxRateDrop: map[string]float64{"objective_accuracy": 1}})
	if len(regress) == 0 {
		t.Fatal("no regression detected")
	}
	found := false
	for _, v := range regress {
		if v.Code == "EVAL_NEW_FAILURE" {
			found = true
		}
	}
	if !found {
		t.Fatal("new failure not detected")
	}
	candidate.Scenarios = nil
	regress = CompareReports(base, candidate, RegressionPolicy{})
	found = false
	for _, v := range regress {
		if v.Code == "EVAL_SCENARIO_MISSING" {
			found = true
		}
	}
	if !found {
		t.Fatal("deleted cases not flagged")
	}
}
func TestTraceReplayAndVersionDiff(t *testing.T) {
	s := golden(t, "anaphora")
	r := recorded(t, "anaphora")
	o := r.Turns[0]
	trace := Trace{Schema: "turn_trace.v1", TurnID: o.TurnID, UserID: r.UserID, Origin: "user", Route: o.Route, Steps: []TraceStep{{Name: "safety", Signal: o.Safety}, {Name: "retrieval", Items: []string{o.Retrieval[0].ID}}, {Name: "brain", Decision: o.Brain}, {Name: "validator", Report: o.Validator}}}
	traces := TraceFixture{Schema: "eval_trace_fixture.v1", ScenarioID: s.ID, SyntheticOnly: true, Traces: []Trace{trace}}
	output := Runner{Executor: TraceExecutor{Fixture: traces}}.Run(context.Background(), []Scenario{s})
	if output.Passed || output.Totals.Blockers == 0 {
		t.Fatal("incomplete trace replay passed")
	}
	changed := trace
	changed.Route = "crisis"
	changed.Versions.Bundle = "new"
	changed.Steps = append([]TraceStep(nil), trace.Steps...)
	changed.Steps[2].RuleID = "BR-007"
	diff := CompareTraceVersions(trace, changed)
	if len(diff) < 2 {
		t.Fatalf("missed trace drift: %+v", diff)
	}
	other := trace
	other.UserID = "88888888-8888-4888-8888-888888888888"
	if len(CompareTraceVersions(trace, other)) != 1 {
		t.Fatal("did not reject cross-owner trace")
	}
}
func TestTraceFixtureOwnerAndSchemas(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "trace.json")
	for _, data := range []string{`{"schema":"eval_trace_fixture.v1","scenario_id":"anaphora","synthetic_only":false,"traces":[]}`, `{"schema":"eval_trace_fixture.v1","scenario_id":"anaphora","synthetic_only":true,"traces":[{"schema":"wrong","turn_id":"bad","user_id":"bad"}]}`} {
		if os.WriteFile(tracePath, []byte(data), 0600) != nil {
			t.Fatal("write")
		}
		if _, e := LoadTraceFixture(tracePath); e == nil {
			t.Fatal("invalid trace fixture accepted")
		}
	}
}
func TestConcurrentSyntheticIsolation(t *testing.T) {
	var wg sync.WaitGroup
	for _, id := range []string{"anaphora", "longitudinal", "escalation"} {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := golden(t, id)
			r := recorded(t, id)
			report := Runner{Executor: NewRecordedExecutor(r)}.Run(context.Background(), []Scenario{s})
			if !report.Passed {
				t.Error("concurrent scenario failed")
			}
		}()
	}
	wg.Wait()
}

type attestedLabels struct{}

func (attestedLabels) VerifySyntheticLabels(context.Context, Scenario) (bool, error) {
	return true, nil
}
func TestVerifiedGoldenLabelHookDoesNotApproveClinicalEffectiveness(t *testing.T) {
	s := golden(t, "anaphora")
	s.Clinician = &Clinician{Author: "synthetic reviewed label", LabelsAgreement: "3/3"}
	got := Runner{Executor: NewRecordedExecutor(recorded(t, "anaphora")), Labels: attestedLabels{}}.Run(context.Background(), []Scenario{s})
	if !got.Passed || got.Scenarios[0].ReviewStatus != "externally_verified_synthetic_labels" || !got.SyntheticOnly {
		t.Fatal("verified label provenance not distinguished from synthetic evaluation")
	}
}
func TestCandidateReplayRequiresCompleteVersionedEvidence(t *testing.T) {
	s := golden(t, "anaphora")
	r := recorded(t, "anaphora")
	actual := r.Turns[0]
	trace := Trace{Schema: "turn_trace.v1", TurnID: actual.TurnID, UserID: actual.UserID, Origin: "user", Route: actual.Route, Steps: []TraceStep{{Name: "safety", Signal: actual.Safety}, {Name: "brain", Decision: actual.Brain}, {Name: "validator", Report: actual.Validator}}}
	fixture := TraceFixture{Schema: "eval_trace_fixture.v1", SyntheticOnly: true, ScenarioID: s.ID, Traces: []Trace{trace}}
	replay, err := ReplayAgainst(context.Background(), s, fixture, NewRecordedExecutor(r))
	if err != nil || replay.Report.Passed || len(replay.Drift) == 0 {
		t.Fatalf("incomplete original trace falsely passed: %v", err)
	}
	matched := false
	for _, d := range replay.Drift {
		if d.Code == "EVAL_COMPARE_INCOMPLETE" && d.Step == "generator" {
			matched = true
		}
	}
	if !matched {
		t.Fatal("missing generator source not recognized")
	}
	mismatch := fixture
	mismatch.Traces[0].UserID = "88888888-8888-4888-8888-888888888888"
	if _, err := ReplayAgainst(context.Background(), s, mismatch, NewRecordedExecutor(r)); !errors.Is(err, ErrReplay) {
		t.Fatal("cross-owner candidate trace accepted")
	}
}
