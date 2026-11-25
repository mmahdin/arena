package safety

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type spyLLM struct {
	request  ports.StructuredRequest
	called   int
	response ports.StructuredResponse
	err      error
}

func (f *spyLLM) Structured(_ context.Context, req ports.StructuredRequest) (ports.StructuredResponse, error) {
	f.request = req
	f.called++
	return f.response, f.err
}
func (*spyLLM) Stream(context.Context, ports.StreamRequest) (ports.Stream, error) {
	return nil, errors.New("unexpected stream")
}
func l1Schemas(t *testing.T) *schema.Set {
	t.Helper()
	s, err := schema.New(schema.Vocabulary{
		Emotions: []string{"test_emotion"}, Slots: []string{"test_slot"}, Skills: []string{"test_skill"}, RationaleTags: []string{"test_tag"},
		AvoidTags: []string{"test_avoid"}, HypothesisTags: []string{"test_hypothesis"}, Frameworks: map[string][]string{"test_framework": {"test_slot"}},
		Protocols: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestWP04StructuredClassifierUsesPinnedSchemaAndTrustedContext(t *testing.T) {
	a := loadFixture(t)
	fake := &spyLLM{response: ports.StructuredResponse{JSON: []byte(`{"level":"low","categories":[],"confidence":0.8}`), Model: "synthetic-model"}}
	classifier, err := NewLLMClassifier(fake, l1Schemas(t), a, 450*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	meta := contract.CallMeta{UserID: uuid.New(), TurnID: uuid.New()}
	in := ports.SafetyInput{Text: "SYNTHETIC_TEST_MESSAGE", Lang: "en", Recent: []contract.MsgView{{Role: contract.RoleUser, Text: "recent"}}}
	if _, err := classifier.Classify(context.Background(), in); !errors.Is(err, ErrClassifier) {
		t.Fatalf("missing metadata was accepted: %v", err)
	}
	if fake.called != 0 {
		t.Fatal("provider invoked with missing identity")
	}
	response, err := classifier.Classify(WithCallMeta(context.Background(), meta), in)
	if err != nil || response.Output.Level != contract.RiskLow || response.Version != "safety.v1/synthetic-model" {
		t.Fatalf("classifier: %+v %v", response, err)
	}
	req := fake.request
	if fake.called != 1 || req.Tier != ports.ComponentSafety || req.SchemaName != "safety.v1" || len(req.Schema) == 0 || len(req.System) != 1 || req.Meta.UserID != meta.UserID || req.Meta.Bundle.Version != "kb@0.9.0" || req.IdempotencyKey != meta.TurnID.String()+"/safety_l1" || req.Timeout != 450*time.Millisecond || len(req.Messages) != 2 {
		t.Fatalf("WP04 request contract: %+v", req)
	}
	meta.Bundle.Version = "kb@different"
	if _, err := classifier.Classify(WithCallMeta(context.Background(), meta), in); !errors.Is(err, ErrClassifier) || fake.called != 1 {
		t.Fatal("mixed-version classifier call was allowed")
	}
}
func TestWP04ClassifierFailureIsNotTreatedAsLow(t *testing.T) {
	a := loadFixture(t)
	fake := &spyLLM{err: errors.New("synthetic provider failure with private response")}
	classifier, err := NewLLMClassifier(fake, l1Schemas(t), a, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	meta := contract.CallMeta{UserID: uuid.New(), TurnID: uuid.New()}
	gate, _ := NewGate(a, classifier, nil)
	sig, err := gate.Assess(WithCallMeta(context.Background(), meta), ports.SafetyInput{Text: "TEST_HIGH_SIGNAL", Lang: "en"})
	if !errors.Is(err, ErrClassifier) || sig.Level != contract.RiskHigh || !sig.Degraded {
		t.Fatalf("failure lowered L0: %+v %v", sig, err)
	}
}
func TestPlaceholderPromptNeverRunsClassifier(t *testing.T) {
	a := loadFixture(t)
	a.prompt = "SYNTHETIC PLACEHOLDER: Do not call any model"
	if _, err := NewLLMClassifier(&spyLLM{}, l1Schemas(t), a, time.Second); !errors.Is(err, ErrAssets) {
		t.Fatalf("accepted placeholder prompt: %v", err)
	}
}
