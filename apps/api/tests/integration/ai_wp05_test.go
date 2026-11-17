package integration

// These WP-05 tests use the real WP-02 SQL Server 2022 admission/finalization
// and the real WP-04 model client with its independent scripted provider. They
// are local-only and opt in through aiDatabase (WELLBEING_INTEGRATION=true).
// The Brain, Safety, Context, Validator and Generator implementations below
// are NON-CLINICAL TEST ADAPTERS, not production implementations of later WPs.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/gateway"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

var wp05Schema = []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["objective"],"properties":{"objective":{"const":"reflect"}},"additionalProperties":false}`)

type wp05Brain struct {
	client ports.Client
	db     *sql.DB
	user   c.ID
}

func (b wp05Brain) Decide(ctx context.Context, in ports.BrainInput) (ports.BrainOutput, error) {
	// Admission MUST be committed before a model begins; this is a separate
	// DB connection with no long-running transaction around the LLM stage.
	var admitted int
	err := b.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1 AND role=N'user'`, in.WCtx.Turn.TurnID).Scan(&admitted)
	if err != nil {
		return ports.BrainOutput{}, err
	}
	if admitted != 1 {
		return ports.BrainOutput{}, errors.New("user message not durable before Brain")
	}
	response, err := b.client.Structured(ctx, ports.StructuredRequest{Tier: ports.TierSmall, Schema: wp05Schema, SchemaName: "wp05.synthetic.v1", MaxTokens: 48, Temperature: .2, Meta: c.CallMeta{TurnID: in.WCtx.Turn.TurnID, UserID: b.user}, System: []ports.Block{{Name: "synthetic_test", Text: "Test only; return synthetic JSON"}}})
	if err != nil {
		return ports.BrainOutput{}, err
	}
	var value struct {
		Objective c.Objective `json:"objective"`
	}
	if err = json.Unmarshal(response.JSON, &value); err != nil {
		return ports.BrainOutput{}, err
	}
	return ports.BrainOutput{Decision: c.BrainDecision{SchemaVersion: "brain.v1", Decision: c.Decision{Primary: value.Objective}}, Usage: response.Usage}, nil
}

type wp05Generator struct {
	client ports.Client
	user   c.ID
}

func (g wp05Generator) Generate(ctx context.Context, in ports.GenInput, sink ports.TokenSink) (ports.GenOutput, error) {
	stream, err := g.client.Stream(ctx, ports.StreamRequest{Tier: ports.TierStrong, MaxTokens: 64, Temperature: .7, Meta: c.CallMeta{UserID: g.user, TurnID: in.Turn.TurnID}, System: []ports.Block{{Name: "synthetic_test", Text: "Test-only synthetic output"}}})
	if err != nil {
		return ports.GenOutput{}, err
	}
	defer stream.Close()
	var text strings.Builder
	var usage c.LLMUsage
	for {
		ch, e := stream.Recv()
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil {
			return ports.GenOutput{}, e
		}
		if ch.Usage != nil {
			usage = *ch.Usage
		}
		if ch.Text != "" {
			if err = sink.Send(ch.Text); err != nil {
				return ports.GenOutput{}, err
			}
			text.WriteString(ch.Text)
		}
	}
	return ports.GenOutput{Text: text.String(), Usage: usage}, nil
}

type wp05Safety struct{}

func (wp05Safety) Assess(context.Context, ports.SafetyInput) (c.SafetySignal, error) {
	return c.SafetySignal{Level: c.RiskLow}, nil
}

type wp05Assembler struct{}

func (wp05Assembler) Build(context.Context, c.TurnRequest, c.SafetySignal, c.RetrievalResult) (c.WorkingContext, ports.StateSnapshot, error) {
	return c.WorkingContext{Risk: c.RiskView{Current: c.RiskLow}}, ports.StateSnapshot{}, nil
}

type wp05Validator struct{}

func (wp05Validator) StreamGuard(c.ResponsePlan, c.RiskLevel) ports.StreamGuard { return nil }
func (wp05Validator) Finalize(_ context.Context, text string, _ c.ResponsePlan, _ c.WorkingContext, _ c.RiskLevel, _ *c.CrisisPlan) (c.ValidationReport, error) {
	return c.ValidationReport{Passed: text == "Synthetic validated reply", Action: c.VAAccept}, nil
}

func wp05Client(t *testing.T, provider *fake.Provider) ports.Client {
	t.Helper()
	model := func(name string) llm.Model {
		return llm.Model{Provider: "synthetic", Name: name, NativeJSONSchema: true, InputMicroUSDPerM: 100, OutputMicroUSDPerM: 100}
	}
	cfg := llm.Config{Chains: map[ports.ModelTier][]llm.Model{
		ports.TierSmall: {model("wp05-brain")}, ports.TierStrong: {model("wp05-generator")}, ports.TierClassifier: {model("wp05-classifier")},
		ports.TierEmbedding: {{Provider: "synthetic", Name: "wp05-embedding", Dimensions: 3}},
	}, DefaultTimeout: 2 * time.Second, BreakerWindow: time.Minute, BreakerCooldown: time.Minute, BreakerErrorRate: .5, BreakerMinSamples: 2}
	client, err := llm.New(cfg, map[string]llm.Provider{"synthetic": provider}, llm.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return client
}
func wp05Runtime(t *testing.T, f aiFixture, client ports.Client) *orchestrator.Service {
	t.Helper()
	g, err := gateway.New(f.store, f.store)
	if err != nil {
		t.Fatal(err)
	}
	core, err := orchestrator.New(orchestrator.Dependencies{
		Gateway: g, Store: f.store, Finalizer: f.store, Recovery: f.store,
		Safety: wp05Safety{}, Assembler: wp05Assembler{}, Brain: wp05Brain{client: client, db: f.db, user: f.a},
		Generator: wp05Generator{client: client, user: f.a}, Validator: wp05Validator{}, Tracer: trace.NewRecorder(),
	}, orchestrator.Options{TotalTimeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return core
}
func wp05QueueSuccess(provider *fake.Provider) {
	provider.QueueStructured("wp05-brain", fake.StructuredReply{JSON: []byte(`{"objective":"reflect"}`), Usage: c.LLMUsage{InTokens: 20, OutTokens: 5, FinishReason: "stop"}})
	provider.QueueStream("wp05-generator", fake.StreamReply{FailAt: -1, Chunks: []ports.Chunk{
		{Text: "Synthetic validated reply"}, {Usage: &c.LLMUsage{InTokens: 20, OutTokens: 7}}, {FinishReason: "stop"},
	}})
}
func TestWP05SQLServerTwoStageAndReplay(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	provider := fake.New()
	wp05QueueSuccess(provider)
	core := wp05Runtime(t, f, wp05Client(t, provider))
	req := f.req()
	first, err := core.ProcessTurn(context.Background(), req)
	if err != nil || first.Status != c.TurnCompleted || first.AssistantMessageID == nil || first.Text != "Synthetic validated reply" {
		t.Fatalf("real two-stage SQL lifecycle failed: %+v %v", first, err)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1`, first.TurnID) != 2 ||
		aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_turn_traces WHERE turn_id=@p1`, first.TurnID) != 1 ||
		aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_turn_context WHERE turn_id=@p1`, first.TurnID) != 1 ||
		aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_outbox WHERE user_id=@p1 AND type=N'turn_completed'`, f.a) != 1 {
		t.Fatal("final SQL UoW did not atomically persist message/state/trace/event")
	}
	stored, err := core.ProcessTurn(context.Background(), req)
	if err != nil || stored.TurnID != first.TurnID || stored.UserMessageID != first.UserMessageID || stored.Text != first.Text ||
		len(provider.History()) != 2 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1`, first.TurnID) != 2 {
		t.Fatalf("replay reran LLM or duplicated rows: %+v %v", stored, err)
	}
}
func TestWP05SQLServerFailureAndOriginalRequestRecovery(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	provider := fake.New()
	provider.QueueStructured("wp05-brain", fake.StructuredReply{JSON: []byte(`{"objective":"reflect"}`), Usage: c.LLMUsage{InTokens: 1, OutTokens: 1}})
	provider.QueueStream("wp05-generator", fake.StreamReply{Err: &llm.Failure{Kind: llm.FailurePermanent}, FailAt: 0})
	core := wp05Runtime(t, f, wp05Client(t, provider))
	req := f.req()
	failed, err := core.ProcessTurn(context.Background(), req)
	if err == nil || failed.Status != c.TurnFailed || failed.AssistantMessageID != nil || aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1`, failed.TurnID) != 1 {
		t.Fatalf("failed turn exposed reply or removed source: %+v %v", failed, err)
	}
	replay, err := core.ProcessTurn(context.Background(), req)
	if err != nil || replay.Status != c.TurnFailed || len(provider.History()) != 2 {
		t.Fatal("terminal failure was not idempotent")
	}
	if err = core.RetryFailed(context.Background(), f.a, req.ClientTurnID); err != nil {
		t.Fatal(err)
	}
	wp05QueueSuccess(provider)
	// RecoverNext reads the original stored metadata/text; it must never
	// trust a new client-supplied message for the same idempotency key.
	complete, err := core.RecoverNext(context.Background(), f.a)
	if err != nil || complete.Status != c.TurnCompleted || complete.UserMessageID != failed.UserMessageID || aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1`, failed.TurnID) != 2 {
		t.Fatalf("recovery lost original user message: %+v %v", complete, err)
	}
}
