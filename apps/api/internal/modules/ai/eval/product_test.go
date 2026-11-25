package eval

import (
	"context"
	"path/filepath"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func TestProductScenariosRequireSemanticReviewNotWordMatching(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..", "..", "knowledge", "evals", "golden")
	paths, err := filepath.Glob(filepath.Join(root, "product_*.yaml"))
	if err != nil || len(paths) != 6 {
		t.Fatal("product scenarios missing", paths, err)
	}
	for _, path := range paths {
		scenario, err := LoadFile(path)
		if err != nil || len(scenario.Sequence()) < 3 {
			t.Fatal(path, err)
		}
		for _, turn := range scenario.Sequence() {
			if len(turn.Turn.Expect.Output.Judge) == 0 {
				t.Fatal("product quality reduced to rule matching", path)
			}
		}
	}
	// Passing a Validator cannot certify naturalness/helpfulness. This is an
	// evaluator trust test, not a claim that scripted output is product-reviewed.
	s, err := LoadFile(filepath.Join(root, "product_fa_distress_then_help.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for i := range s.Turns {
		s.Turns[i].Expect = Expect{Output: OutputExpect{Judge: map[string]string{"helpfulness": ">=4"}}}
	}
	runner := Runner{Executor: ExecuteFunc(func(_ context.Context, e *Environment, step SequenceTurn) (Observation, error) {
		return Observation{UserID: e.UserID, TurnID: c.NewID().String(), Route: "normal", Generator: &Generation{Text: "Synthetic output awaiting human review."}, Validator: &c.ValidationReport{Passed: true, Action: c.VAAccept}}, nil
	})}
	report := runner.Run(context.Background(), []Scenario{s})
	if report.Passed || len(report.ReviewQueue) != len(s.Turns) || report.Metrics["quality_review"].Passed != 0 || report.Metrics["quality_helpfulness"].Total != len(s.Turns) {
		t.Fatal("unreviewed prose counted as product success", report)
	}
}
