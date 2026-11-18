package integration

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/brain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/gateway"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/generator"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/safety"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/validator"
)

type wp11SQLClassifier struct{}

func (wp11SQLClassifier) Classify(context.Context, ports.SafetyInput) (safety.Assessment, error) {
	return safety.Assessment{Version: "synthetic-wp11", Output: c.SafetyClassifierOutput{Level: c.RiskNone, Confidence: 1}}, nil
}

// Real WP-02/04/06/07/08/10/11, SQL Server ownership checks and fenced atomic
// delivery; only model output and clinical assets are synthetic test fixtures.
func TestWP11SQLValidatedDeliveryIsolationAndSoftDelete(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	ctx := context.Background()
	raw, e := os.ReadFile("../../internal/modules/ai/application/generator/testdata/assets.json")
	if e != nil {
		t.Fatal(e)
	}
	var files map[string]string
	if json.Unmarshal(raw, &files) != nil {
		t.Fatal("fixture JSON")
	}
	b := wp10SQLBundle{}
	for p, v := range files {
		b[p] = []byte(v)
	}
	b["lexicons/validator.en.yaml"], e = os.ReadFile("../../internal/modules/ai/validator/testdata/validator.en.yaml")
	if e != nil {
		t.Fatal(e)
	}
	reader, e := aidb.NewContextReader(db)
	if e != nil {
		t.Fatal(e)
	}
	source, e := aidb.NewGeneratorContextVerifier(db, wp10SQLUse{true})
	if e != nil {
		t.Fatal(e)
	}
	brainSource, e := aidb.NewBrainContextVerifier(db, wp10SQLUse{true})
	if e != nil {
		t.Fatal(e)
	}
	provider := fake.New()
	client := wp05Client(t, provider)
	ba, e := brain.Load(ctx, b, brain.LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	brainService, e := brain.New(client, ba, reader, brain.Options{Verifier: brainSource})
	if e != nil {
		t.Fatal(e)
	}
	now := func() time.Time { return time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC) }
	ga, e := generator.Load(ctx, b, generator.LoadOptions{AllowSyntheticForTests: true, Now: now})
	if e != nil {
		t.Fatal(e)
	}
	gen, e := generator.New(client, ga, generator.Options{Verifier: source})
	if e != nil {
		t.Fatal(e)
	}
	val, e := validator.New(ctx, b, validator.Options{AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	sa, e := safety.Load(ctx, b, safety.LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	gw, e := gateway.New(f.store, f.store)
	if e != nil {
		t.Fatal(e)
	}
	o, e := orchestrator.NewIntegrated(orchestrator.Dependencies{Gateway: gw, Store: f.store, Finalizer: f.store, Recovery: f.store, Brain: brainService, Generator: gen, Validator: val, Tracer: trace.NewRecorder()}, orchestrator.Options{TotalTimeout: 5 * time.Second}, orchestrator.IntegratedOptions{Reader: reader, Bundles: wp10SQLRegistry{b}, Assets: sa, Classifier: wp11SQLClassifier{}, AllowSyntheticForTests: true, Now: now})
	if e != nil {
		t.Fatal(e)
	}
	d := brainService.Fallback(c.WorkingContext{})
	raw, _ = json.Marshal(d)
	provider.QueueStructured("wp05-brain", fake.StructuredReply{JSON: raw})
	provider.QueueStream("wp05-generator", fake.StreamReply{FailAt: -1, Chunks: []ports.Chunk{{Text: "Synthetic accepted SQL reply."}, {FinishReason: "stop", Usage: &c.LLMUsage{InTokens: 20, OutTokens: 5}}}})
	req := f.req()
	req.Locale = "en"
	req.TZ = "UTC"
	res, e := o.ProcessTurn(ctx, req)
	if e != nil || res.Status != c.TurnCompleted || res.Text != "Synthetic accepted SQL reply." {
		t.Fatal(res, e)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1`, res.TurnID) != 2 {
		t.Fatal("draft persisted separately")
	}
	before := len(provider.History())
	again, e := o.ProcessTurn(ctx, req)
	if e != nil || !reflect.DeepEqual(again, res) || len(provider.History()) != before {
		t.Fatal("durable replay called provider", e)
	}
	// Admit a later turn in another owned conversation. Both user and assistant
	// history are checked as their actual roles, never re-labelled as user facts.
	next := f.req()
	next.ConversationID = f.ca2
	next.Locale = "en"
	next.TZ = "UTC"
	admission := admit(t, f, next)
	in := ports.GenInput{Turn: c.TurnView{TurnID: admission.TurnID, MessageID: admission.UserMessageID, Text: next.Text, Lang: "en"}, Plan: d.Plan, Objective: d.Decision.Primary, Risk: c.RiskLow, Prefs: c.PrefsView{Language: "en"}, Bundle: b.Reference(), Recent: []c.MsgView{{ID: res.UserMessageID, Role: c.RoleUser, Text: req.Text, Age: "1m"}, {ID: *res.AssistantMessageID, Role: c.RoleAssistant, Text: res.Text, Age: "1m"}}}
	in.Permitted = []c.MemoryView{{ID: *res.AssistantMessageID, Text: res.Text, Epistemic: c.EpSystemObserved, Verbatim: true}}
	in.Plan.PermittedContext = []c.ID{*res.AssistantMessageID}
	if e = source.VerifyGeneration(ctx, f.a, in); e != nil {
		t.Fatal("owned assistant provenance rejected", e)
	}
	// WP-09 temporal excerpts use the existing episode/active envelope with
	// Verbatim=true; preserve that provenance rather than treating them as
	// unsupported derived episodes or requiring the recent-history envelope.
	exact := in
	exact.Recent = nil
	exact.Permitted = []c.MemoryView{{ID: res.UserMessageID, Text: req.Text, Kind: c.MemEpisode, Status: c.MemActive, Epistemic: c.EpReported, Verbatim: true, EvidenceN: 1, Sensitivity: c.SensNormal}}
	exact.Plan.PermittedContext = []c.ID{res.UserMessageID}
	if e = source.VerifyGeneration(ctx, f.a, exact); e != nil {
		t.Fatal("WP-09 exact excerpt rejected", e)
	}
	forged := in
	forged.Permitted = append([]c.MemoryView(nil), in.Permitted...)
	forged.Permitted[0].Epistemic = c.EpReported
	if source.VerifyGeneration(ctx, f.a, forged) == nil {
		t.Fatal("assistant prose promoted to user report")
	}
	deny, _ := aidb.NewGeneratorContextVerifier(db, wp10SQLUse{false})
	if deny.VerifyGeneration(ctx, f.a, in) == nil {
		t.Fatal("explicit AI-use denial ignored")
	}
	if source.VerifyGeneration(ctx, f.b, in) == nil {
		t.Fatal("foreign user accepted owned turn")
	}
	foreignReq := f.req()
	foreignReq.UserID = f.b
	foreignReq.ConversationID = f.cb
	foreign := admit(t, f, foreignReq)
	forged = in
	forged.Recent = []c.MsgView{{ID: foreign.UserMessageID, Role: c.RoleUser, Text: foreignReq.Text}}
	callCtx := generator.WithCallMeta(ctx, c.CallMeta{UserID: f.a, TurnID: admission.TurnID, Bundle: b.Reference(), Route: c.RouteNormal})
	before = len(provider.History())
	if _, e = gen.Generate(callCtx, forged, nil); e == nil || len(provider.History()) != before {
		t.Fatal("foreign text reached provider", e)
	}
	forged = in
	forged.Turn.Text = "SYNTHETIC tampered source"
	if source.VerifyGeneration(ctx, f.a, forged) == nil {
		t.Fatal("tampered current message accepted")
	}
	futureReq := f.req()
	future := admit(t, f, futureReq)
	forged = in
	forged.Recent = []c.MsgView{{ID: future.UserMessageID, Role: c.RoleUser, Text: futureReq.Text}}
	if source.VerifyGeneration(ctx, f.a, forged) == nil {
		t.Fatal("future queued text disclosed")
	}
	// A memory's evidence count and sensitivity are sourced, not model claims.
	item := c.NewID()
	aiExec(t, db, `INSERT dbo.ai_memory_items(id,user_id,kind,content,epistemic,source_json,created_at,updated_at) VALUES(@p1,@p2,N'fact',N'SYNTHETIC retained fact',N'reported',N'{}',DATEADD(day,-1,SYSUTCDATETIME()),DATEADD(day,-1,SYSUTCDATETIME()))`, item, f.a)
	aiExec(t, db, `INSERT dbo.ai_memory_evidence(user_id,item_id,ref_kind,ref_id,role,evidence_at) VALUES(@p1,@p2,N'message',@p3,N'supports',SYSUTCDATETIME())`, f.a, item, res.UserMessageID)
	mem := c.MemoryView{ID: item, Kind: c.MemFact, Epistemic: c.EpReported, Status: c.MemActive, Text: "SYNTHETIC retained fact", EvidenceN: 1, Sensitivity: c.SensNormal}
	withMem := in
	withMem.Permitted = []c.MemoryView{mem}
	withMem.Plan.PermittedContext = []c.ID{item}
	if e = source.VerifyGeneration(ctx, f.a, withMem); e != nil {
		t.Fatal("owned evidence memory rejected", e)
	}
	withMem.Permitted[0].EvidenceN = 2
	if source.VerifyGeneration(ctx, f.a, withMem) == nil {
		t.Fatal("invented evidence count accepted")
	}
	withMem.Permitted[0].EvidenceN = 1
	aiExec(t, db, `UPDATE dbo.ai_memory_items SET deleted_at=SYSUTCDATETIME(),deleted_by=@p2,deletion_reason=N'user_request' WHERE id=@p1 AND user_id=@p2`, item, f.a)
	if source.VerifyGeneration(ctx, f.a, withMem) == nil {
		t.Fatal("hidden memory disclosed")
	}
	aiExec(t, db, `UPDATE dbo.conversations SET deleted_at=SYSUTCDATETIME(),deleted_by=@p2,deletion_reason=N'user_request' WHERE id=@p1 AND user_id=@p2`, f.ca, f.a)
	if source.VerifyGeneration(ctx, f.a, in) == nil {
		t.Fatal("hidden history disclosed")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1 AND content IS NOT NULL`, res.TurnID) != 2 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND content=N'SYNTHETIC retained fact'`, item) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_evidence WHERE item_id=@p1`, item) != 1 {
		t.Fatal("soft delete erased original or derived data")
	}
}
