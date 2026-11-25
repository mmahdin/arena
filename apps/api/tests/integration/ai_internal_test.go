package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/app/composition"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	bg "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/background"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/eval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

// All outputs below are explicitly fictional provider responses. They still pass
// through the real WP-04 schema client, every real module, and live SQL UoWs.
type internalScript struct {
	highRisk     bool
	background   bool
	researchSeen bool
	ports.Client
	db           *sql.DB
	provider     *fake.Provider
	mu           sync.Mutex
	protocol     bool
	badOnce      bool
	unifiedCalls int
}

func (s *internalScript) Structured(ctx context.Context, req ports.StructuredRequest) (ports.StructuredResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	model := "synthetic-memory-writer"
	var out any
	switch req.SchemaName {
	case "jev_selection.v1":
		model = "synthetic-jev"
		out = map[string]any{"schema_version": "jev_selection.v1", "selected_resource_ids": []string{"protocol:test_exercise"}, "topic_continues": true}
	case c.UnifiedSchemaVersion:
		model = "synthetic-unified"
		s.unifiedCalls++ // retained counter name for historical fixture compatibility
		var w c.WorkingContext
		if json.Unmarshal([]byte(req.Messages[0].Content), &w) != nil {
			return ports.StructuredResponse{}, errors.New("synthetic context missing")
		}
		d := c.UnifiedTurn{SchemaVersion: c.UnifiedSchemaVersion, Reply: "Synthetic accepted reply.", Conversation: c.ConversationIntent{Intent: c.IntentDiscloseEvent, Objective: c.ObjReflect}, MemorySignals: []string{"event"}, UsedContext: []c.ID{}, Reflection: true}
		topic := "synthetic topic"
		d.Thread = &c.ThreadIntent{Relation: c.RelContinues, Slots: []c.SlotUpdate{{Slot: "situation", Value: "synthetic situation", Status: c.EpReported, Evidence: []c.ID{w.Turn.MessageID}}}, Hypotheses: []c.HypothesisSignal{}}
		if w.FocusThread == nil {
			d.Thread.Relation = c.RelNew
			d.Thread.Topic = &topic
		}
		if s.background {
			d.BackgroundRequests = []c.BackgroundRequest{{ID: "review_history", Kind: "memory_intelligence", Objective: "Review fictional earlier relationship experiences", Topic: "synthetic situation", Queries: []string{"synthetic situation"}, EvidenceIDs: []c.ID{w.Turn.MessageID}, DependsOn: []string{}}}
		}
		if s.protocol {
			d.Conversation.Objective = c.ObjContinueIntervention
			d.Protocol = &c.ProtocolIntent{Directive: c.ProtocolDirective{Action: c.PAAdvance, Outcome: wpInternalOutcome(c.OutSuccess)}, UserAffirmed: true, ExpectedStatus: c.PIActive, ExpectedStep: "first"}
		}
		if s.highRisk {
			d.Conversation.Objective = c.ObjSafetySupport
			d.RiskRead = &c.BrainRiskRead{Level: c.RiskHigh, Categories: []c.RiskCategory{c.RiskOther}, ReasonTag: "synthetic"}
			if w.Protocol != nil && w.Protocol.Status == c.PIActive {
				d.Protocol = &c.ProtocolIntent{Directive: c.ProtocolDirective{Action: c.PAPause}, ExpectedStatus: c.PIStopped}
			}
		}
		if s.badOnce {
			d.SchemaVersion = "invalid_unified"
			s.badOnce = false
		}
		out = d
	case bg.OutputSchema:
		var input bg.Input
		if json.Unmarshal([]byte(req.Messages[0].Content), &input) != nil {
			return ports.StructuredResponse{}, errors.New("synthetic input invalid")
		}
		if len(input.Retrieval) > 0 {
			s.researchSeen = true
		}
		output := bg.Output{Schema: bg.OutputSchema, Operations: []bg.Operation{}, Formulations: []bg.Formulation{}, KnowledgeIDs: []string{}, InformationGaps: []string{}, Hints: []bg.Hint{}}
		if input.Request.Kind == "reflection" {
			model = "synthetic-reflection"
		} else {
			for _, message := range input.Messages {
				if message.Role != c.RoleUser {
					continue
				}
				output.Operations = append(output.Operations, bg.Operation{ID: "synthetic_fact", Action: "add", Kind: c.MemFact, Content: "synthetic situation", Evidence: []bg.Evidence{{ID: message.ID, Role: "supports"}}, Sensitivity: c.SensNormal})
				break
			}
		}
		out = output
	case c.MemoryWriterSchemaVersion:
		if strings.Contains(req.Messages[0].Content, "background_research") {
			s.researchSeen = true
		}
		all := regexp.MustCompile(`<message id="([a-f0-9-]+)" role="user"`).FindStringSubmatch(req.Messages[0].Content)
		if len(all) != 2 {
			return ports.StructuredResponse{}, errors.New("synthetic source missing")
		}
		id, _ := uuid.Parse(all[1])
		kind := c.MemFact
		if !c.ValidID(req.Meta.TurnID) {
			kind = c.MemEpisode
		}
		out = c.MemoryWriterOutput{SchemaVersion: c.MemoryWriterSchemaVersion, Ops: []c.MemoryOpProposal{{Op: c.OpAdd, Kind: kind, Content: "synthetic situation", Epistemic: c.EpReported, EvidenceMsg: []c.ID{id}, Tags: []string{"synthetic_event"}, Sensitivity: c.SensNormal, ReasonTag: "synthetic"}}}
	case c.ReflectionSchemaVersion:
		model = "synthetic-reflection"
		text := req.Messages[0].Content
		task := regexp.MustCompile(`<task thread_id="([a-f0-9-]+)" session_id="([a-f0-9-]+)"`).FindStringSubmatch(text)
		if len(task) != 3 {
			return ports.StructuredResponse{}, errors.New("synthetic task missing")
		}
		tid, _ := uuid.Parse(task[1])
		sid, _ := uuid.Parse(task[2])
		messages := regexp.MustCompile(`<message id="([a-f0-9-]+)" session="`+sid.String()+`"`).FindAllStringSubmatch(text, -1)
		if len(messages) == 0 {
			return ports.StructuredResponse{}, errors.New("synthetic episode missing")
		}
		first, _ := uuid.Parse(messages[0][1])
		last, _ := uuid.Parse(messages[len(messages)-1][1])
		out = c.ReflectionResult{SchemaVersion: c.ReflectionSchemaVersion, ThreadID: tid, HypothesisOps: []c.MemoryOpProposal{}, ThreadProposals: []c.ThreadProposal{}, InformationGaps: []string{}, Episode: &c.EpisodeDoc{SessionID: sid, Summary: "synthetic episode", Topics: []string{}, Emotions: []c.Emotion{}, Outcome: "explored", MsgRange: [2]c.ID{first, last}, FollowUps: []string{}}}
	default:
		return ports.StructuredResponse{}, errors.New("unexpected synthetic call")
	}
	raw, _ := json.Marshal(out)
	s.provider.QueueStructured(model, fake.StructuredReply{JSON: raw})
	return s.Client.Structured(ctx, req)
}
func (s *internalScript) Stream(context.Context, ports.StreamRequest) (ports.Stream, error) {
	return nil, errors.New("unified conversation does not stream a second model")
}
func (s *internalScript) Embed(ctx context.Context, req ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	s.mu.Lock()
	s.provider.QueueEmbedding("synthetic-embedding", fake.EmbeddingReply{Vector: []float32{1, 0, 0}})
	s.mu.Unlock()
	return s.Client.(ports.EmbeddingClient).Embed(ctx, req)
}

func internalBundle(t *testing.T) wp12Bundle {
	t.Helper()
	b := wp12Bundle{syntheticBundle: syntheticBundle{}, ref: c.BundleRef{Version: "kb@0.9.0", PromptSHAs: map[string]string{"internal": c.NewID().String()}}}
	for _, name := range []string{"memory", "reflection"} {
		raw, e := os.ReadFile("../../internal/modules/ai/application/" + name + "/testdata/assets.json")
		if e != nil {
			t.Fatal(e)
		}
		var files map[string]string
		_ = json.Unmarshal(raw, &files)
		for p, v := range files {
			b.syntheticBundle[p] = []byte(v)
		}
	}
	for _, x := range []struct{ dest, source string }{{"lexicons/validator.en.yaml", "../../internal/modules/ai/validator/testdata/validator.en.yaml"}, {"config/retrieval.yaml", "../../internal/modules/ai/application/retrieval/testdata/retrieval.yaml"}, {"protocols/test_exercise.yaml", "testdata/ai_internal/protocol.yaml"}} {
		raw, e := os.ReadFile(x.source)
		if e != nil {
			t.Fatal(e)
		}
		b.syntheticBundle[x.dest] = raw
	}
	for _, path := range []string{"prompts/unified.system.tmpl", "prompts/jev.system.tmpl", "prompts/intelligence.system.tmpl", "config/background.yaml", "policy/communication_policy.md", "policy/boundary_policy.yaml", "policy/safety_policy.yaml", "resources/crisis_resources.yaml"} {
		raw, e := os.ReadFile("../../../../knowledge/" + path)
		if e != nil {
			t.Fatal(e)
		}
		b.syntheticBundle[path] = raw
	}
	cards := []c.KnowledgeCard{}
	for _, r := range []struct{ id, category, path string }{{"skill:reflect_emotion", "skill", "skills/reflect_emotion.yaml"}, {"protocol:test_exercise", "protocol", "protocols/test_exercise.yaml"}, {"framework:synthetic_framework", "framework", "schemas/formulation_test.yaml"}} {
		cards = append(cards, c.KnowledgeCard{ID: r.id, Category: r.category, ContentID: r.path, Name: "Synthetic resource", Description: "Synthetic engineering knowledge", Purpose: "Exercise structural paths", UseWhen: "Synthetic tests only", Limitations: "No clinical use", Prerequisites: "Synthetic evidence"})
	}
	b.syntheticBundle["cards.yaml"], _ = json.Marshal(map[string]any{"schema": "knowledge_cards.v1", "cards": cards})
	b.syntheticBundle["prompts/safety_l1.system.tmpl"] = []byte("SYNTHETIC classifier engineering fixture; no clinical approval.")
	b.syntheticBundle["templates/protocol_offer.yaml"] = []byte("schema: protocol_offer.v1\nwhy_templates: {synthetic_why: {text: SYNTHETIC REASON}}\nmust_say: {synthetic_optout: {text: SYNTHETIC OPT OUT}}\n")
	delete(b.syntheticBundle, "protocols/synthetic_protocol.yaml")
	var thresholds map[string]any
	_ = json.Unmarshal(b.syntheticBundle["config/thresholds.yaml"], &thresholds)
	thresholds["validator"] = map[string]float32{"directive_advice": .5, "certainty_about_inference": .5, "dependency_encouraging": .5, "invalidating": .5}
	b.syntheticBundle["config/thresholds.yaml"], _ = json.Marshal(thresholds)
	return b
}
func TestInternalAICoreSQLCompositionMultiturnEvalAndWorkers(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	b := internalBundle(t)
	provider := fake.New()
	model := func(name string) llm.Model {
		return llm.Model{Provider: "synthetic", Name: name, NativeJSONSchema: true}
	}
	cfg := llm.Config{Chains: map[ports.ModelTier][]llm.Model{ports.ComponentJEV: {model("synthetic-jev")}, ports.ComponentUnified: {model("synthetic-unified")}, ports.ComponentMemoryWriter: {model("synthetic-memory-writer")}, ports.ComponentReflection: {model("synthetic-reflection")}, ports.ComponentEmbedding: {{Provider: "synthetic", Name: "synthetic-embedding", Dimensions: 3}}}, DefaultTimeout: 2 * time.Second, BreakerWindow: time.Minute, BreakerCooldown: time.Minute, BreakerErrorRate: .5, BreakerMinSamples: 20}
	base, e := llm.New(cfg, map[string]llm.Provider{"synthetic": provider}, llm.Options{})
	if e != nil {
		t.Fatal(e)
	}
	script := &internalScript{Client: base, db: db, provider: provider}
	reader, _ := aidb.NewContextReader(db)
	keys := wp12New(t, db).cipher
	cap := eval.NewCapture()
	recorder := trace.NewRecorder()
	opts := composition.AIInternalOptions{DB: db, Bundle: b, Client: script, Embeddings: script, UsePolicy: syntheticUsePolicy{true}, MemoryPolicy: wp12Policy{}, Cipher: keys, EmbeddingModel: "synthetic-embedding", Dimension: 3, WorkerID: "internal-synthetic", Feedback: mem.FeedbackPolicy{MinUnusedRetrievals: 3, ImportanceMultiplier: .9}, Tracer: recorder, AllowSyntheticForTests: true, TestDecorate: func(d orchestrator.Dependencies) orchestrator.Dependencies {

		d.Gateway = &eval.CaptureGateway{Next: d.Gateway, Capture: cap}
		d.Unified = &eval.CaptureUnified{Next: d.Unified, Capture: cap}
		d.Finalizer = &eval.CaptureFinalizer{Next: d.Finalizer, Capture: cap}
		return d
	}}
	core, e := composition.NewAIInternal(context.Background(), opts)
	if e != nil {
		wp12Err(t, e)
	}
	req := f.req()
	req.Text = "synthetic situation"
	req.Locale = "en"
	req.TZ = "UTC"
	first, e := core.Turns.ProcessTurn(context.Background(), req)
	if e != nil || first.Status != c.TurnCompleted {
		t.Fatalf("first status=%s err=%#v provider_calls=%d", first.Status, e, len(provider.History()))
	}
	meta := mem.WithMeta(context.Background(), mem.Meta{UserID: f.a, OperationID: c.NewID(), Bundle: b.Reference()})
	paused := false
	if e = core.Memory.SetControls(meta, f.a, &paused, map[string]bool{"synthetic_memory": true}, nil); e != nil {
		wp12Err(t, e)
	}
	// WP-15 capture executes the real complete core, not recorded observations.
	scenario, e := eval.Load([]byte("schema: golden.v1\nid: internal_live\nsynthetic_only: true\nsetup:\n  user: {id: " + f.a.String() + ", locale: en, tz: UTC}\nturns:\n- text: synthetic situation\n  expect:\n    route: normal\n    unified: {objective_in: [reflect]}\n    output: {max_questions: 0}\n    state_after: {session: {last_objective: reflect}}\n- text: synthetic situation again\n  expect:\n    route: normal\n    state_after: {session: {last_objective: reflect}}\n"))
	if e != nil {
		t.Fatal(e)
	}
	executor := eval.CaptureExecutor{Orchestrator: core.Turns, Capture: cap, NewRequest: func(_ *eval.Environment, step eval.SequenceTurn) (c.TurnRequest, error) {
		r := req
		r.ClientTurnID = c.NewID()
		r.Text = step.Turn.Text
		return r, nil
	}}
	report := eval.Runner{Executor: executor}.Run(context.Background(), []eval.Scenario{scenario})
	if !report.Passed {
		raw, _ := eval.EncodeReport(report)
		t.Fatalf("live eval %s", raw)
	}
	// Ordinary async extraction, embedding publication and receipt recovery.
	for i := 0; i < 12; i++ {
		worked, e := core.MemoryWorker.RunOnce(context.Background())
		if e != nil {
			wp12Err(t, e)
		}
		if !worked {
			time.Sleep(5 * time.Millisecond)
		}
	}
	core.Intelligence.Config.MinMessages = 2
	if e = core.Intelligence.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 40; i++ {
		worked, er := core.MemoryWorker.RunOnce(context.Background())
		if er != nil {
			t.Fatal(er)
		}
		if !worked {
			break
		}
	}
	var count int
	if e = db.QueryRow(`SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1`, f.a).Scan(&count); e != nil || count != 1 {
		t.Fatalf("dedupe/memory %d %v", count, e)
	}
	var turns int
	if e = db.QueryRow(`SELECT turn_count FROM dbo.ai_sessions WHERE user_id=@p1 AND closed_at IS NULL`, f.a).Scan(&turns); e != nil || turns != 3 {
		t.Fatal("multi-turn state", e, turns)
	}
	snap, e := reader.Read(context.Background(), f.a, f.ca)
	if e != nil || snap.Session.FocusThreadID == nil || snap.ThreadUserTurns[*snap.Session.FocusThreadID] != 3 {
		t.Fatal("accepted thread membership missing", e, snap.ThreadUserTurns)
	}
	// A completed key never calls any provider again.
	calls := len(provider.History())
	again, e := core.Turns.ProcessTurn(context.Background(), req)
	if e != nil || again.TurnID != first.TurnID || len(provider.History()) != calls {
		t.Fatal("replay invoked models")
	}
	// Explicit analysis is queued atomically, then uses hybrid retrieval only
	// when the existing Memory Worker claims it after the response is stored.
	script.background = true
	researchReq := req
	researchReq.ClientTurnID = c.NewID()
	researchReq.Text = "synthetic follow-up review"
	researchTurn, e := core.Turns.ProcessTurn(context.Background(), researchReq)
	if e != nil {
		t.Fatal("background admission", e)
	}
	script.background = false
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE turn_id=@p1 AND kind=N'memory_write'`, researchTurn.TurnID) != 1 {
		t.Fatal("explicit analysis must queue independently without per-turn routine extraction")
	}
	for i := 0; i < 40; i++ {
		worked, e := core.MemoryWorker.RunOnce(context.Background())
		if e != nil {
			t.Fatal("background execution", e)
		}
		if !worked {
			break
		}
	}

	if !script.researchSeen || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE turn_id=@p1 AND kind=N'memory_write' AND status=N'completed'`, researchTurn.TurnID) != 1 {
		t.Fatal("background research did not reach Memory Intelligence and complete")
	}
	// A real offered instance can advance only on an affirmative user turn.
	tid := *snap.Session.FocusThreadID
	pi := c.NewID()
	if _, e = db.Exec(`UPDATE dbo.ai_threads SET phase=N'working',version=version+1 WHERE id=@p1 AND user_id=@p2; INSERT dbo.ai_protocol_instances(id,user_id,thread_id,protocol_id,protocol_version,status,step_id,offered_at) VALUES(@p3,@p2,@p1,N'test_exercise',N'1.0.0',N'offered',N'first',SYSUTCDATETIME())`, tid, f.a, pi); e != nil {
		t.Fatal(e)
	}
	script.protocol = true
	req.ClientTurnID = c.NewID()
	req.Text = "yes"
	active, e := core.Turns.ProcessTurn(context.Background(), req)
	if e != nil || active.Status != c.TurnCompleted {
		snapshot, _ := recorder.Snapshot(active.TurnID)
		t.Fatalf("protocol activation: err=%#v steps=%+v", e, snapshot.Steps)
	}
	var status string
	var consent sql.NullString
	if e = db.QueryRow(`SELECT status,CONVERT(varchar(36),consent_message_id) FROM dbo.ai_protocol_instances WHERE id=@p1 AND user_id=@p2`, pi, f.a).Scan(&status, &consent); e != nil || status != "active" || !consent.Valid {
		t.Fatal("protocol consent not atomic", status, e)
	}
	script.protocol = false
	// Explicit authenticated correction uses the composed Memory Committer.
	var memoryID c.ID
	if e = db.QueryRow(`SELECT TOP(1) CONVERT(varchar(36),id) FROM dbo.ai_memory_items WHERE user_id=@p1 AND kind=N'fact'`, f.a).Scan(&memoryID); e != nil {
		t.Fatal(e)
	}
	correctionMeta := mem.WithMeta(context.Background(), mem.Meta{UserID: f.a, OperationID: c.NewID(), Bundle: b.Reference()})
	if e = core.Memory.Correct(correctionMeta, f.a, memoryID, ports.CorrectionContest); e != nil {
		wp12Err(t, e)
	}
	var corrected string
	if e = db.QueryRow(`SELECT status FROM dbo.ai_memory_items WHERE id=@p1 AND user_id=@p2`, memoryID, f.a).Scan(&corrected); e != nil || corrected != "contested" {
		t.Fatal("correction not committed", e)
	}
	// Invalid structured output is rejected in ONE attempt; no replacement
	// safety classifier or hidden regeneration is introduced.
	script.badOnce = true
	req.ClientTurnID = c.NewID()
	req.Text = "synthetic situation after correction"
	beforeInvalid := script.unifiedCalls
	failed, e := core.Turns.ProcessTurn(context.Background(), req)
	if e == nil || failed.Status != c.TurnFailed || script.unifiedCalls != beforeInvalid+1 {
		t.Fatal("invalid unified output was retried or published", failed, e)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1 AND role=N'assistant'`, failed.TurnID) != 0 {
		t.Fatal("invalid structured draft persisted")
	}
	// Idle session closure -> durable reflection -> WP-12 episode consolidation.
	if _, e = db.Exec(`UPDATE dbo.ai_sessions SET last_activity_at=DATEADD(hour,-1,SYSUTCDATETIME()) WHERE user_id=@p1 AND closed_at IS NULL`, f.a); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- core.Run(ctx) }()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		_ = db.QueryRow(`SELECT COUNT(*) FROM dbo.ai_reflection_runs WHERE user_id=@p1`, f.a).Scan(&count)
		if count > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	if e = <-done; !errors.Is(e, context.Canceled) {
		t.Fatal("worker shutdown", e)
	}
	if count == 0 {
		t.Fatal("reflection worker not integrated")
	}
	var userMessages int
	if e = db.QueryRow(`SELECT COUNT(*) FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id WHERE v.user_id=@p1 AND m.role=N'user'`, f.a).Scan(&userMessages); e != nil || userMessages != 6 {
		t.Fatal("scheduler fabricated user interaction")
	}
	// Escalation remains inside Unified and persists the existing monotonic
	// risk transition; there is no classifier, generator or fallback call.
	script.highRisk = true
	req.ClientTurnID = c.NewID()
	req.Text = "synthetic safety escalation"
	script.mu.Lock()
	unifiedBefore := script.unifiedCalls
	script.mu.Unlock()
	crisis, e := core.Turns.ProcessTurn(context.Background(), req)
	if e != nil || crisis.Status != c.TurnCompleted {
		var ce *c.Error
		_ = errors.As(e, &ce)
		step := ""
		if ce != nil {
			step = ce.Step
		}
		snapshot, _ := recorder.Snapshot(crisis.TurnID)
		t.Fatalf("crisis route: error=%v step=%s trace=%+v", e, step, snapshot.Steps)
	}
	script.mu.Lock()
	unifiedAfter := script.unifiedCalls
	script.mu.Unlock()
	if unifiedAfter != unifiedBefore+1 {
		t.Fatal("crisis must use exactly one unified call")
	}
	var risk string
	if e = db.QueryRow(`SELECT risk_level FROM dbo.ai_user_state WHERE user_id=@p1`, f.a).Scan(&risk); e != nil || risk != "high" {
		t.Fatal("risk not monotonic", risk, e)
	}
	// Invalid owner is rejected before any provider or cross-user state disclosure.
	req.UserID = f.b
	req.ClientTurnID = c.NewID()
	calls = len(provider.History())
	if _, e = core.Turns.ProcessTurn(context.Background(), req); e == nil || len(provider.History()) != calls {
		t.Fatal("owner isolation")
	}
}

func wpInternalOutcome(v c.StepOutcome) *c.StepOutcome { return &v }
