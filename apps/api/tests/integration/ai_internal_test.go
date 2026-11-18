package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/app/composition"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/brain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/generator"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/eval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

// All outputs below are explicitly fictional provider responses. They still pass
// through the real WP-04 schema client, every real module, and live SQL UoWs.
type internalScript struct {
	highRisk bool
	ports.Client
	db         *sql.DB
	provider   *fake.Provider
	template   c.BrainDecision
	mu         sync.Mutex
	protocol   bool
	badOnce    bool
	brainCalls int
}

func (s *internalScript) Structured(ctx context.Context, req ports.StructuredRequest) (ports.StructuredResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	model := "wp05-brain"
	var out any
	switch req.SchemaName {
	case c.SafetySchemaVersion:
		model = "wp05-classifier"
		level := c.RiskNone
		categories := []c.RiskCategory{}
		if s.highRisk {
			level = c.RiskHigh
			categories = []c.RiskCategory{c.RiskOther}
		}
		out = c.SafetyClassifierOutput{Level: level, Confidence: 1, Categories: categories}
	case c.BrainSchemaVersion:
		raw, _ := json.Marshal(s.template)
		var d c.BrainDecision
		_ = json.Unmarshal(raw, &d)
		s.brainCalls++
		var msg c.ID
		if e := s.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.messages WHERE turn_id=@p1 AND role=N'user'`, req.Meta.TurnID).Scan(&msg); e != nil {
			return ports.StructuredResponse{}, e
		}
		var tid sql.NullString
		_ = s.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),focus_thread_id) FROM dbo.ai_sessions WHERE user_id=@p1 AND closed_at IS NULL`, req.Meta.UserID).Scan(&tid)
		d.Decision.Confidence = c.ConfHigh
		d.Plan.Question = c.QuestionPlan{Allowed: false, MaxCount: 0}
		d.Needs.MemoryWorthy = true
		d.Needs.MemoryTags = []string{"event"}
		d.Needs.Formulation = true
		if tid.Valid {
			id, _ := uuid.Parse(tid.String)
			d.Understanding.ThreadID = &id
			d.TurnAnalysis.ThreadRelation = c.RelContinues
		} else {
			topic := "synthetic topic"
			d.TurnAnalysis.ThreadRelation = c.RelNew
			d.TurnAnalysis.NewThreadTopic = &topic
		}
		d.Understanding.Slots = []c.SlotUpdate{{Slot: "situation", Value: "synthetic situation", Status: c.EpReported, Evidence: []c.ID{msg}}}
		if s.protocol {
			d.Decision.Primary = c.ObjContinueIntervention
			d.Plan.Protocol = &c.ProtocolDirective{Action: c.PAAdvance, Outcome: wpInternalOutcome(c.OutSuccess)}
		}
		out = d
	case c.MemoryWriterSchemaVersion:
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
		model = "wp05-generator"
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
func (s *internalScript) Stream(ctx context.Context, req ports.StreamRequest) (ports.Stream, error) {
	s.mu.Lock()
	text := "Synthetic accepted reply."
	if s.protocol {
		text += " SYNTHETIC OPT OUT"
	}
	if s.badOnce {
		text = "You have depression."
		s.badOnce = false
	}
	s.provider.QueueStream("wp05-generator", fake.StreamReply{FailAt: -1, Chunks: []ports.Chunk{{Text: text}, {FinishReason: "stop", Usage: &c.LLMUsage{InTokens: 10, OutTokens: 5}}}})
	s.mu.Unlock()
	return s.Client.Stream(ctx, req)
}
func (s *internalScript) Embed(ctx context.Context, req ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	s.mu.Lock()
	s.provider.QueueEmbedding("wp05-embedding", fake.EmbeddingReply{Vector: []float32{1, 0, 0}})
	s.mu.Unlock()
	return s.Client.(ports.EmbeddingClient).Embed(ctx, req)
}

type internalScores struct{}

func (internalScores) Scores(context.Context, string, []c.MsgView, string) (map[string]float32, error) {
	return map[string]float32{"directive_advice": 0, "certainty_about_inference": 0, "dependency_encouraging": 0, "invalidating": 0}, nil
}
func internalBundle(t *testing.T) wp12Bundle {
	t.Helper()
	b := wp12Bundle{wp10SQLBundle: wp10SQLBundle{}, ref: c.BundleRef{Version: "kb@0.9.0", PromptSHAs: map[string]string{"internal": c.NewID().String()}}}
	for _, name := range []string{"generator", "memory", "reflection"} {
		raw, e := os.ReadFile("../../internal/modules/ai/application/" + name + "/testdata/assets.json")
		if e != nil {
			t.Fatal(e)
		}
		var files map[string]string
		_ = json.Unmarshal(raw, &files)
		for p, v := range files {
			b.wp10SQLBundle[p] = []byte(v)
		}
	}
	for _, x := range []struct{ dest, source string }{{"lexicons/validator.en.yaml", "../../internal/modules/ai/validator/testdata/validator.en.yaml"}, {"config/retrieval.yaml", "../../internal/modules/ai/application/retrieval/testdata/retrieval.yaml"}, {"protocols/test_exercise.yaml", "testdata/ai_internal/protocol.yaml"}} {
		raw, e := os.ReadFile(x.source)
		if e != nil {
			t.Fatal(e)
		}
		b.wp10SQLBundle[x.dest] = raw
	}
	b.wp10SQLBundle["prompts/safety_l1.system.tmpl"] = []byte("SYNTHETIC classifier engineering fixture; no clinical approval.")
	b.wp10SQLBundle["templates/protocol_offer.yaml"] = []byte("schema: protocol_offer.v1\nwhy_templates: {synthetic_why: {text: SYNTHETIC REASON}}\nmust_say: {synthetic_optout: {text: SYNTHETIC OPT OUT}}\n")
	delete(b.wp10SQLBundle, "protocols/synthetic_protocol.yaml")
	var thresholds map[string]any
	_ = json.Unmarshal(b.wp10SQLBundle["config/thresholds.yaml"], &thresholds)
	thresholds["validator"] = map[string]float32{"directive_advice": .5, "certainty_about_inference": .5, "dependency_encouraging": .5, "invalidating": .5}
	b.wp10SQLBundle["config/thresholds.yaml"], _ = json.Marshal(thresholds)
	return b
}
func TestInternalAICoreSQLCompositionMultiturnEvalAndWorkers(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	b := internalBundle(t)
	provider := fake.New()
	base := wp05Client(t, provider)
	script := &internalScript{Client: base, db: db, provider: provider}
	reader, _ := aidb.NewContextReader(db)
	a, e := brain.Load(context.Background(), b, brain.LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	bs, e := brain.New(script, a, reader, brain.Options{})
	if e != nil {
		t.Fatal(e)
	}
	script.template = bs.Fallback(c.WorkingContext{})
	keys := wp12New(t, db).cipher
	cap := eval.NewCapture()
	recorder := trace.NewRecorder()
	opts := composition.AIInternalOptions{DB: db, Bundle: b, Client: script, Embeddings: script, UsePolicy: wp10SQLUse{true}, MemoryPolicy: wp12Policy{}, Cipher: keys, EmbeddingModel: "wp05-embedding", Dimension: 3, WorkerID: "internal-synthetic", Feedback: mem.FeedbackPolicy{MinUnusedRetrievals: 3, ImportanceMultiplier: .9}, Tracer: recorder, AllowSyntheticForTests: true, TestOutputClassifier: internalScores{}, TestDecorate: func(d orchestrator.Dependencies) orchestrator.Dependencies {
		d.Gateway = &eval.CaptureGateway{Next: d.Gateway, Capture: cap}
		d.Brain = &eval.CaptureBrain{Next: d.Brain.(*brain.Service), Capture: cap}
		d.Generator = &eval.CaptureGenerator{Next: d.Generator.(*generator.Service), Capture: cap}
		d.Validator = &eval.CaptureValidator{Next: d.Validator, Capture: cap}
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
	scenario, e := eval.Load([]byte("schema: golden.v1\nid: internal_live\nsynthetic_only: true\nsetup:\n  user: {id: " + f.a.String() + ", locale: en, tz: UTC}\nturns:\n- text: synthetic situation\n  expect:\n    route: normal\n    brain: {objective_in: [reflect]}\n    validator: {passed: true, action: accept}\n    output: {max_questions: 0, must_not: [diagnosis, ids_leaked]}\n    state_after: {session: {last_objective: reflect}}\n- text: synthetic situation again\n  expect:\n    route: normal\n    validator: {passed: true, action: accept}\n    state_after: {session: {last_objective: reflect}}\n"))
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
	// A bad draft is blocked, regenerated once, and never persisted.
	script.badOnce = true
	req.ClientTurnID = c.NewID()
	req.Text = "synthetic situation after correction"
	regenerated, e := core.Turns.ProcessTurn(context.Background(), req)
	if e != nil || regenerated.Text != "Synthetic accepted reply." {
		t.Fatalf("regeneration failed: %v", e)
	}
	var leaked int
	if e = db.QueryRow(`SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1 AND content LIKE N'%depression%'`, regenerated.TurnID).Scan(&leaked); e != nil || leaked != 0 {
		t.Fatal("draft reached durable history")
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
	if e = db.QueryRow(`SELECT COUNT(*) FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id WHERE v.user_id=@p1 AND m.role=N'user'`, f.a).Scan(&userMessages); e != nil || userMessages != 5 {
		t.Fatal("scheduler fabricated user interaction")
	}
	// Escalation uses the real crisis planner/validator and reviewed synthetic
	// fallback, then persists monotonic risk without a free Brain decision.
	script.highRisk = true
	req.ClientTurnID = c.NewID()
	req.Text = "synthetic safety escalation"
	script.mu.Lock()
	brainsBefore := script.brainCalls
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
	brainsAfter := script.brainCalls
	script.mu.Unlock()
	if brainsBefore != brainsAfter {
		t.Fatal("crisis invoked free Brain")
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
