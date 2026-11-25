package orchestrator_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/gateway"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/knowledge"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/protocol"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/unified"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle/testfixture"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type unifiedClient struct {
	failJEV     bool
	topicChange bool
	jevCalls    int
	jevContexts []c.WorkingContext
	outage      bool
	calls       int
	output      func(c.WorkingContext) c.UnifiedTurn
	fail        bool
}

func (f *unifiedClient) Structured(_ context.Context, r ports.StructuredRequest) (ports.StructuredResponse, error) {
	if r.Tier == ports.ComponentJEV {
		f.jevCalls++
		var payload struct {
			Context c.WorkingContext `json:"working_context"`
		}
		_ = json.Unmarshal([]byte(r.SystemOne.State), &payload)
		f.jevContexts = append(f.jevContexts, payload.Context)
		if f.failJEV {
			return ports.StructuredResponse{}, errors.New("synthetic JEV outage")
		}
		raw, _ := json.Marshal(map[string]any{"schema_version": "jev_selection.v1", "selected_resource_ids": []string{"protocol:thought_record"}, "topic_continues": !f.topicChange})
		return ports.StructuredResponse{JSON: raw}, nil
	}

	f.calls++
	if f.outage {
		return ports.StructuredResponse{}, &c.Error{Code: c.ErrProvider, Step: "synthetic"}
	}
	if r.Tier != ports.ComponentUnified || r.MaxAttempts != 1 {
		panic("extra inference stage")
	}
	if f.fail {
		return ports.StructuredResponse{}, errors.New("synthetic outage")
	}
	var w c.WorkingContext
	_ = json.Unmarshal([]byte(r.Messages[0].Content), &w)
	raw, e := json.Marshal(f.output(w))
	return ports.StructuredResponse{JSON: raw, Usage: c.LLMUsage{Attempts: 1, InTokens: 100, OutTokens: 50}}, e
}
func (*unifiedClient) Stream(context.Context, ports.StreamRequest) (ports.Stream, error) {
	panic("second generation forbidden")
}

type persistentFixture struct{ item c.MemoryView }

func (p persistentFixture) Persistent(context.Context, c.ID, c.ID) (c.PersistentContext, error) {
	return c.PersistentContext{Items: []c.MemoryView{p.item}}, nil
}

type unifiedRetriever struct {
	calls    int
	degraded bool
}

func (r *unifiedRetriever) Retrieve(context.Context, c.RetrievalRequest) (c.RetrievalResult, error) {
	r.calls++
	return c.RetrievalResult{Meta: c.RetrievalMeta{Degraded: r.degraded, Strategies: []string{"bm25"}}}, nil
}

type unifiedMemoryFixture struct {
	bundle        c.BundleRef
	calls         int
	analysisCalls int
}

func (m *unifiedMemoryFixture) Reference() c.BundleRef { return m.bundle }
func (m *unifiedMemoryFixture) PlanTurnJob(context.Context, c.Admission, c.UnifiedTurn) (*c.AsyncJobProposal, error) {
	m.calls++
	return &c.AsyncJobProposal{Kind: "memory_write", DedupeKey: "synthetic", Payload: json.RawMessage(`{"synthetic":true}`)}, nil
}
func ordinaryUnified(w c.WorkingContext) c.UnifiedTurn {
	return c.UnifiedTurn{SchemaVersion: c.UnifiedSchemaVersion, Reply: "That fictional meeting sounded tiring.", Conversation: c.ConversationIntent{Intent: c.IntentDiscloseEvent, Objective: c.ObjReflect}, MemorySignals: []string{}, UsedContext: []c.ID{}}
}
func unifiedRuntime(t *testing.T, h *testIntegrationHarness, cl *unifiedClient, r *unifiedRetriever, inspect *trace.Inspector) (*orchestrator.Service, *unifiedMemoryFixture) {
	t.Helper()
	b, report, e := bundle.Read(context.Background(), "../../../../../../../knowledge", bundle.Options{})
	if e != nil {
		t.Fatal(e, report.Errors)
	}
	overlay := testfixture.WithRuntimeSettings(t, b)
	h.registry.b.files = map[string][]byte{}
	for _, name := range overlay.ArtifactNames() {
		raw, e := overlay.Artifact(context.Background(), name)
		if e != nil {
			t.Fatal(e)
		}
		h.registry.b.files[name] = raw
	}
	h.registry.b.files["bundle.yaml"] = []byte(strings.Replace(string(h.registry.b.files["bundle.yaml"]), "version: 0.1.0", "version: 0.9.0", 1))
	main, e := unified.New(context.Background(), cl, h.registry.b, nil, unified.Options{AllowSyntheticForTests: true, MaxInputTokens: 18000})
	if e != nil {
		t.Fatal(e)
	}
	gate, _ := gateway.New(h.s, h.s)
	mem := &unifiedMemoryFixture{bundle: h.registry.b.Reference()}
	assets, e := protocol.Load(context.Background(), h.registry.b, protocol.LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	engine, e := protocol.New(assets, h.r, time.Now)
	if e != nil {
		t.Fatal(e)
	}
	persistent := persistentFixture{c.MemoryView{ID: c.NewID(), Source: "memory", Kind: c.MemPreference, Epistemic: c.EpReported, Status: c.MemActive, Text: "prefers brief responses", Sensitivity: c.SensNormal}}
	selector, e := knowledge.New(context.Background(), cl, h.registry.b, knowledge.Options{})
	if e != nil {
		t.Fatal(e)
	}
	o, e := orchestrator.NewUnified(orchestrator.Dependencies{Knowledge: selector, Unified: main, UnifiedMemory: mem, Inspector: inspect, Gateway: gate, Store: h.s, Finalizer: testIntegrationFinalizer{h.s, h.r}, Recovery: h.s, Retriever: r, Protocol: engine, UnifiedReflection: orchestrator.ReflectionPlanner{Bundle: h.registry.b.Reference()}, Tracer: trace.NewRecorder()}, orchestrator.Options{}, orchestrator.UnifiedOptions{Reader: h.r, Bundles: h.registry, Persistent: persistent, AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	return o, mem
}
func TestUnifiedTurnRetrievesOnceInfersOnceAndReplaysWithoutCalls(t *testing.T) {
	h := newTestIntegrationHarness(t)
	cl := &unifiedClient{output: ordinaryUnified}
	r := &unifiedRetriever{degraded: true}
	o, m := unifiedRuntime(t, h, cl, r, nil)
	cl.output = func(w c.WorkingContext) c.UnifiedTurn {
		if len(w.Persistent.Items) != 1 || !w.RetrievalMeta.Degraded {
			t.Fatal("persistent context lost during retrieval outage")
		}
		return ordinaryUnified(w)
	}
	req := h.request("The fictional meeting was tiring.")
	res, e := o.ProcessTurn(context.Background(), req)
	if e != nil || res.Status != c.TurnCompleted || !res.Degraded || cl.calls != 1 || r.calls != 1 || m.calls != 1 {
		t.Fatal(res, e, cl.calls, r.calls, m.calls)
	}
	replay, e := o.ProcessTurn(context.Background(), req)
	if e != nil || replay.Text != res.Text || cl.calls != 1 || r.calls != 1 {
		t.Fatal("replay repeated work", e)
	}

}
func TestUnifiedInvalidProposalAndRollbackKeepUserOnly(t *testing.T) {
	for _, mode := range []string{"reference", "state", "provider", "rollback"} {
		t.Run(mode, func(t *testing.T) {
			h := newTestIntegrationHarness(t)
			cl := &unifiedClient{output: ordinaryUnified}
			r := &unifiedRetriever{}
			o, _ := unifiedRuntime(t, h, cl, r, nil)
			cl.output = func(w c.WorkingContext) c.UnifiedTurn {
				u := ordinaryUnified(w)
				switch mode {
				case "reference":
					u.UsedContext = []c.ID{c.NewID()}
				case "state":
					summary := strings.Repeat("word ", 121)
					u.Session = &c.SessionIntent{Summary: &summary}
				}
				return u
			}
			if mode == "provider" {
				cl.fail = true
			}
			if mode == "rollback" {
				h.s.rejectCommit = true
			}
			res, e := o.ProcessTurn(context.Background(), h.request("synthetic new event"))
			n, done, _ := h.s.snapshot()
			if mode == "state" {
				if e != nil || res.Status != c.TurnCompleted || done != 1 || len(h.last(t).Decisions.Rejected) == 0 {
					t.Fatal("invalid independent summary interrupted reply", res, e)
				}
				return
			}
			if e == nil || res.Status != c.TurnFailed || n != 1 || done != 0 || cl.calls != 1 {
				t.Fatal(mode, res, e, n, done, cl.calls)
			}
		})
	}
}
func TestUnifiedInspectorHasOneConversationalNode(t *testing.T) {
	h := newTestIntegrationHarness(t)
	cl := &unifiedClient{output: ordinaryUnified}
	r := &unifiedRetriever{}
	dir := filepath.Join(t.TempDir(), "private")
	ins, e := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "127.0.0.1:8080"})
	if e != nil {
		t.Fatal(e)
	}
	o, _ := unifiedRuntime(t, h, cl, r, ins)
	res, e := o.ProcessTurn(context.Background(), h.request("synthetic meeting"))
	if e != nil {
		t.Fatal(e)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(files) != 1 {
		t.Fatal(files)
	}
	raw, _ := os.ReadFile(files[0])
	if e = trace.ValidateJSON(raw); e != nil {
		t.Fatal(e)
	}
	var d trace.Document
	_ = json.Unmarshal(raw, &d)
	calls := 0
	for _, n := range d.Nodes {
		if n.Operation == "unified.Infer" {
			calls++
		}
		for _, bad := range []string{"safety.Assess", "router.RouteModel", "brain.Decide", "generator.Generate", "validator.Finalize"} {
			if n.Operation == bad {
				t.Fatal("obsolete stage", bad)
			}
		}
	}
	if calls != 1 || res.Status != c.TurnCompleted {
		t.Fatal(calls, res)
	}
	if target := os.Getenv("THEIA_WRITE_UNIFIED_FIXTURE"); target != "" {
		if e = os.WriteFile(target, raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
}

func TestUnifiedProtocolSpansTurnsAndRejectsInconsistentReply(t *testing.T) {
	h := newTestIntegrationHarness(t)
	h.r.languagePreference = "en"
	cl := &unifiedClient{output: ordinaryUnified}
	r := &unifiedRetriever{}
	o, _ := unifiedRuntime(t, h, cl, r, nil)
	success := c.OutSuccess
	id := "thought_record"
	turn := 0
	cl.output = func(w c.WorkingContext) c.UnifiedTurn {
		u := ordinaryUnified(w)
		p := &c.ProtocolIntent{UserAffirmed: true, ExpectedStatus: c.PIActive}
		u.Protocol = p
		switch turn {
		case 0:
			u.Conversation.Intent = c.IntentRequestTechnique
			u.Conversation.Objective = c.ObjOfferIntervention
			p.Directive = c.ProtocolDirective{Action: c.PAOffer, ProtocolID: &id}
			p.ExpectedStep = "situation"
		case 1:
			u.Conversation.Objective = c.ObjContinueIntervention
			p.Directive = c.ProtocolDirective{Action: c.PAAdvance, Outcome: &success}
			p.ExpectedStep = "thought"
		case 2:
			u.Conversation.Objective = c.ObjCloseOrPause
			p.Directive = c.ProtocolDirective{Action: c.PAPause}
			p.ExpectedStatus = c.PIPaused
			p.ExpectedStep = ""
		case 3:
			u.Conversation.Objective = c.ObjContinueIntervention
			p.Directive = c.ProtocolDirective{Action: c.PAAdvance, Outcome: &success}
			p.ExpectedStep = "thought"
		case 4:
			u.Conversation.Objective = c.ObjContinueIntervention
			p.Directive = c.ProtocolDirective{Action: c.PAAdvance, Outcome: &success}
			p.ExpectedStep = "invented_step"
		}
		if len(w.SelectedKnowledgeIDs) == 0 {
			t.Fatal("selected protocol content absent before inference")
		}
		return u
	}
	for turn = 0; turn < 4; turn++ {
		res, e := o.ProcessTurn(context.Background(), h.request("Synthetic protocol action"))
		if e != nil || res.Status != c.TurnCompleted {
			t.Fatal(turn, res, e)
		}
	}
	prior := *h.r.protocols[h.conv]
	turn = 4
	res, e := o.ProcessTurn(context.Background(), h.request("Synthetic inconsistent action"))
	after := h.r.protocols[h.conv]
	if e == nil || res.Status != c.TurnFailed || after.Version != prior.Version || after.StepID != prior.StepID {
		t.Fatal("inconsistent protocol pair committed", res, e)
	}
	if cl.calls != 5 {
		t.Fatal("protocol required extra model call", cl.calls)
	}
}

func TestUnifiedProviderOutageUsesSavedLanguageWithoutSecondCall(t *testing.T) {
	h := newTestIntegrationHarness(t)
	h.r.languagePreference = "fa"
	cl := &unifiedClient{output: ordinaryUnified, outage: true}
	ret := &unifiedRetriever{}
	o, _ := unifiedRuntime(t, h, cl, ret, nil)
	res, e := o.ProcessTurn(context.Background(), h.request("synthetic outage turn"))
	if e == nil || res.Status != c.TurnFailed || cl.calls != 1 {
		t.Fatal("provider failure must keep a durable user-only turn", res, e, cl.calls)
	}
}

func (m *unifiedMemoryFixture) PlanAnalysisJob(ctx context.Context, a c.Admission, r c.BackgroundRequest) (*c.AsyncJobProposal, error) {
	m.analysisCalls++
	return &c.AsyncJobProposal{Kind: "memory_write", DedupeKey: a.TurnID.String() + ":" + r.ID, Payload: json.RawMessage(`{"synthetic_analysis":true}`)}, nil
}
func TestIndependentProposalDependenciesAndReplyConsistency(t *testing.T) {
	for _, required := range []bool{false, true} {
		h := newTestIntegrationHarness(t)
		cl := &unifiedClient{output: ordinaryUnified}
		o, _ := unifiedRuntime(t, h, cl, &unifiedRetriever{}, nil)
		cl.output = func(w c.WorkingContext) c.UnifiedTurn {
			u := ordinaryUnified(w)
			bad, _ := json.Marshal(c.SessionIntent{Summary: func() *string { v := strings.Repeat("word ", 121); return &v }()})
			thread, _ := json.Marshal(c.ThreadIntent{Relation: c.RelNew, Topic: func() *string { v := "fictional concern"; return &v }(), Slots: []c.SlotUpdate{}, Hypotheses: []c.HypothesisSignal{}})
			u.Proposals = []c.StateProposal{{ID: "invalid_summary", Kind: "session_intent", Value: bad, DependsOn: []string{}}, {ID: "dependent_thread", Kind: "thread_intent", Value: thread, DependsOn: []string{"invalid_summary"}}}
			if required {
				u.ReplyDependencies = []string{"dependent_thread"}
			}
			return u
		}
		result, e := o.ProcessTurn(context.Background(), h.request("synthetic concern"))
		if required {
			if e == nil || result.Status != c.TurnFailed {
				t.Fatal("dependent reply published", result, e)
			}
		} else {
			if e != nil || result.Status != c.TurnCompleted || h.last(t).State.Thread != nil || len(h.last(t).Decisions.Rejected) < 2 {
				t.Fatal(result, e)
			}
		}
	}
}
func TestBackgroundRequestsAreIndependentlyValidatedAndAtomicallyPlanned(t *testing.T) {
	for _, valid := range []bool{true, false} {
		h := newTestIntegrationHarness(t)
		cl := &unifiedClient{output: ordinaryUnified}
		o, m := unifiedRuntime(t, h, cl, &unifiedRetriever{}, nil)
		cl.output = func(w c.WorkingContext) c.UnifiedTurn {
			u := ordinaryUnified(w)
			id := w.Turn.MessageID
			if !valid {
				id = c.NewID()
			}
			u.BackgroundRequests = []c.BackgroundRequest{{ID: "relationship_review", Kind: "memory_intelligence", Objective: "Review synthetic prior experiences", Topic: "fictional sibling", Queries: []string{"fictional sibling disagreement"}, EvidenceIDs: []c.ID{id}, DependsOn: []string{}}}
			if valid {
				u.ReplyDependencies = []string{"relationship_review"}
			}
			return u
		}
		result, e := o.ProcessTurn(context.Background(), h.request("synthetic relationship concern"))
		if e != nil || result.Status != c.TurnCompleted || cl.calls != 1 || cl.jevCalls != 1 {
			t.Fatal(result, e)
		}
		if valid {
			if m.analysisCalls != 1 || len(h.last(t).Jobs) != 2 {
				t.Fatal("explicit background request was lost")
			}
		} else {
			if m.analysisCalls != 0 || len(h.last(t).Decisions.Rejected) == 0 {
				t.Fatal("foreign evidence reached a background planner")
			}
		}
	}
}
func TestJEVOutageDoesNotRenewExpiredSelections(t *testing.T) {
	h := newTestIntegrationHarness(t)
	cl := &unifiedClient{output: ordinaryUnified}
	o, _ := unifiedRuntime(t, h, cl, &unifiedRetriever{}, nil)
	first, e := o.ProcessTurn(context.Background(), h.request("first synthetic turn"))
	if e != nil || first.Status != c.TurnCompleted {
		t.Fatal(first, e)
	}
	cl.failJEV = true
	for i := 0; i < 3; i++ {
		result, e := o.ProcessTurn(context.Background(), h.request("another synthetic turn"))
		if e != nil || result.Status != c.TurnCompleted || !result.Degraded {
			t.Fatal(result, e)
		}
	}
	if len(h.last(t).Decisions.Carry) != 0 || len(h.last(t).Decisions.SelectedKnowledge) != 0 {
		t.Fatal("failed JEV renewed its own carry-over indefinitely")
	}
}

func TestRepeatedKnowledgeSelectionsDoNotDuplicateCarry(t *testing.T) {
	h := newTestIntegrationHarness(t)
	cl := &unifiedClient{output: ordinaryUnified}
	o, _ := unifiedRuntime(t, h, cl, &unifiedRetriever{}, nil)
	for i := 0; i < 3; i++ {
		result, e := o.ProcessTurn(context.Background(), h.request("synthetic continuing topic"))
		if e != nil || result.Status != c.TurnCompleted {
			t.Fatal(result, e)
		}
	}
	if len(h.last(t).Decisions.Carry) != 1 {
		t.Fatal("identical knowledge recommendation repeated in carry-over")
	}
}
