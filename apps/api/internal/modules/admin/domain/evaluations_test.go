package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func wp15Fixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../../../../../docs/quality/test-results/ai-core-eval.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestWP15ReportValidatedWithoutInventingEvidence(t *testing.T) {
	b := wp15Fixture(t)
	r, sanitized, err := ParseEvalReport(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(sanitized) > len(b) {
		t.Fatal("sanitization expanded report unexpectedly")
	}
	if EvalStatus(r, 0) != "passed" || r.Method != "synthetic_recorded" || r.Totals.Scenarios != 3 {
		t.Fatalf("bad result %+v", r.Totals)
	}
	cov := ReportCoverage(r)
	if cov.ReferenceGoldenCount != 28 || cov.DeclaredGoldenMatches != 0 || cov.FixtureExecutionVerified {
		t.Fatalf("overstated fixture execution: %+v", cov)
	}
	sum := sha256.Sum256(b)
	hash := hex.EncodeToString(sum[:])
	if got, e := ValidateArtifactHash(b, hash); e != nil || got != hash {
		t.Fatal("hash verification failed", e)
	}
	if _, e := ValidateArtifactHash(b, strings.Repeat("0", 64)); e == nil {
		t.Fatal("accepted incorrect digest")
	}
}

func TestWP15RejectsPrivateContentAndForgery(t *testing.T) {
	original := wp15Fixture(t)
	// No arbitrary unknown report fields, including prompt/response payloads.
	mutated := strings.Replace(string(original), `"synthetic_only": true,`, `"synthetic_only": true, "message":"PRIVATE CHAT",`, 1)
	if _, _, err := ParseEvalReport([]byte(mutated)); err == nil {
		t.Fatal("private message field accepted")
	}
	mutate := func(f func(*EvalReport)) bool {
		var r EvalReport
		if json.Unmarshal(original, &r) != nil {
			t.Fatal("invalid repository fixture")
		}
		f(&r)
		data, _ := json.Marshal(r)
		_, _, err := ParseEvalReport(data)
		return err != nil
	}
	cases := []struct {
		name   string
		modify func(*EvalReport)
	}{
		{"non synthetic", func(r *EvalReport) { r.SyntheticOnly = false }},
		{"invalid totals", func(r *EvalReport) { r.Totals.Scenarios++ }},
		{"arbitrary scenario ID", func(r *EvalReport) { r.Scenarios[0].ID = "user@example.com" }},
		{"repeated scenario", func(r *EvalReport) { r.Scenarios[0].ID = r.Scenarios[1].ID }},
		{"private diagnostic", func(r *EvalReport) {
			r.Scenarios[0].Turns[0].Diagnostics = []EvalDiagnostic{{Code: "ERROR: PRIVATE", Assertion: "message", Severity: "failure"}}
		}},
		{"false pass with failed turn", func(r *EvalReport) {
			r.Scenarios[0].Turns[0].Passed = false
			r.Scenarios[0].Turns[0].Diagnostics = []EvalDiagnostic{{Code: "EVAL_MISSING_EVIDENCE", Assertion: "route", Severity: "blocker"}}
			r.Totals.PassedTurns--
		}},
		{"unattested clinical review", func(r *EvalReport) { r.Scenarios[0].ReviewStatus = "externally_verified_synthetic_labels" }},
		{"invalid metric rate", func(r *EvalReport) {
			for k, m := range r.Metrics {
				m.Rate = 0.3
				r.Metrics[k] = m
				break
			}
		}},
		{"oversized tag", func(r *EvalReport) { r.Scenarios[0].Tags = []string{strings.Repeat("a", 100)} }},
		{"ambiguous language", func(r *EvalReport) { r.Scenarios[0].Tags = append(r.Scenarios[0].Tags, "language_fa", "language_en") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !mutate(tc.modify) {
				t.Fatal("accepted invalid content")
			}
		})
	}
}

func TestWP15BlockedAndFailedAreDistinct(t *testing.T) {
	data := EvalReport{
		Schema: "eval_report.v1", Method: "external_executor", SyntheticOnly: true, Passed: false,
		Totals: EvalTotals{Scenarios: 1, FailedScenarios: 1, Turns: 1, Assertions: 1, Blockers: 1},
		Scenarios: []EvalScenario{{ID: "synthetic_case", ReviewStatus: "synthetic_unreviewed", Passed: false,
			Turns: []EvalTurn{{Index: 1, Passed: false, Assertions: 1, Diagnostics: []EvalDiagnostic{{Code: "EVAL_MISSING_EVIDENCE", Assertion: "validator_pass", Severity: "blocker"}}}}}},
		Metrics: map[string]EvalMetric{"validator_pass": {Passed: 0, Total: 1, Rate: 0}},
	}
	b, _ := json.Marshal(data)
	got, _, err := ParseEvalReport(b)
	if err != nil || EvalStatus(got, 0) != "blocked" {
		t.Fatalf("blocked lost %v", err)
	}
	got.Totals.Blockers = 0
	got.Totals.Failures = 1
	if EvalStatus(got, 0) != "failed" {
		t.Fatal("failed collapsed into blocked")
	}
}
func TestEvalFilterBounds(t *testing.T) {
	f := NewEvalFilter(time.Now())
	if f.Validate() != nil {
		t.Fatal("default filter invalid")
	}
	f.Status = "approved"
	if f.Validate() == nil {
		t.Fatal("accepted fake approval status")
	}
	f.Status = "blocked"
	f.Limit = 101
	if f.Validate() == nil {
		t.Fatal("accepted unbounded page")
	}
	f.Limit = 25
	f.Cursor = &InspectorCursor{At: f.From.Add(time.Minute), ID: uuid.New()}
	if f.Validate() != nil {
		t.Fatal("valid cursor refused")
	}
}
