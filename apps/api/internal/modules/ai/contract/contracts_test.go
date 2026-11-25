package contract_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func TestUUIDAndRiskOrdering(t *testing.T) {
	var existing uuid.UUID = c.NewID()
	if !c.ValidID(existing) || c.ValidID(uuid.Nil) || existing.Version() != 4 {
		t.Fatal("UUID semantics changed")
	}
	levels := []c.RiskLevel{c.RiskNone, c.RiskLow, c.RiskElevated, c.RiskHigh, c.RiskImminent}
	for i, a := range levels {
		if a.Rank() != i {
			t.Errorf("rank %s", a)
		}
		for j, b := range levels {
			expected := a
			if j > i {
				expected = b
			}
			if c.MaxRisk(a, b) != expected {
				t.Errorf("max(%s,%s)", a, b)
			}
		}
		if c.MaxRisk(a, "") != c.RiskImminent || c.MaxRisk("bad", a) != c.RiskImminent {
			t.Fatal("invalid risk failed open")
		}
	}
	if c.RiskLevel("bad").Rank() != -1 {
		t.Fatal("invalid risk has a valid rank")
	}
}

func TestBudgetPreservesCancellation(t *testing.T) {
	base, cancel := context.WithCancel(context.Background())
	b := c.Budget{TotalDeadline: time.Now().Add(time.Second), MaxLLMCalls: 3, MaxCostMicroUSD: 123}
	ctx := c.WithBudget(base, b)
	got, ok := c.BudgetFromContext(ctx)
	if !ok || got != b {
		t.Fatal("budget was lost")
	}
	if _, ok := c.BudgetFromContext(base); ok {
		t.Fatal("budget leaked to parent")
	}
	cancel()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("cancellation was lost")
	}
}

func TestErrorsNeverRenderPrivateCause(t *testing.T) {
	secret := errors.New("synthetic-private-payload")
	err := &c.Error{Code: c.ErrProvider, Step: secret.Error(), Cause: secret, Retryable: true}
	if !errors.Is(err, secret) {
		t.Fatal("cause unavailable for control flow")
	}
	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	for _, rendered := range []string{err.Error(), fmt.Sprintf("%v", err), fmt.Sprintf("%+v", err), fmt.Sprintf("%#v", err), string(data)} {
		if strings.Contains(rendered, secret.Error()) {
			t.Fatal("private cause leaked")
		}
	}
	public := c.ToPublic(fmt.Errorf("wrapper: %w", err))
	if public.Code != c.ErrProvider || !public.Retryable {
		t.Fatal("typed error lost")
	}
	for _, bad := range []error{secret, nil, (*c.Error)(nil), &c.Error{Code: c.Code(secret.Error()), Retryable: true}} {
		public := c.ToPublic(bad)
		if public.Code != c.ErrInternal || public.Retryable || strings.Contains(public.Message, secret.Error()) {
			t.Fatal("unsafe default error")
		}
	}
}

func TestJSONCollectionsAndNullability(t *testing.T) {
	brain := c.BrainDecision{}
	data, err := json.Marshal(brain)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), ":null") || brain.TurnAnalysis.Emotions != nil {
		t.Fatal("null collection or caller mutation")
	}
	if !strings.Contains(string(data), `"secondary":[]`) || strings.Contains(string(data), `"risk_read"`) {
		t.Fatal("optional/required collection semantics")
	}
	id := c.NewID()
	result := c.TurnResult{TurnID: id, UserMessageID: c.NewID(), Status: c.TurnFailed}
	data, _ = json.Marshal(result)
	if strings.Contains(string(data), "assistant_message_id") {
		t.Fatal("failed turn invented assistant ID")
	}
	result.AssistantMessageID = &id
	data, _ = json.Marshal(result)
	if !strings.Contains(string(data), `"assistant_message_id"`) {
		t.Fatal("assistant ID absent")
	}
	proposal := c.ThreadProposal{ThreadID: id, Op: "resolve", Reason: "test"}
	data, _ = json.Marshal(proposal)
	if !strings.Contains(string(data), `"thread_id"`) || strings.Contains(string(data), `"ThreadID"`) {
		t.Fatal("reflection proposal is not snake_case")
	}
}
