package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/assembler"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/knowledge"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/state"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/unified"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"sort"
	"strconv"
)

type KnowledgeSelector interface {
	Select(context.Context, knowledge.Input) (c.KnowledgeSelection, error)
	Reference() c.BundleRef
	Catalog() []c.KnowledgeCard
}

type UnifiedOptions struct {
	Reader                 contextstate.Reader
	Bundles                ports.BundleRegistry
	Persistent             contextstate.PersistentReader
	AllowSyntheticForTests bool
	Now                    func() time.Time
}
type UnifiedReflection interface {
	Reference() c.BundleRef
	PlanTurnJob(c.Admission, c.UnifiedTurn, c.StateDelta) *c.AsyncJobProposal
}
type UnifiedMemory interface {
	PlanTurnJob(context.Context, c.Admission, c.UnifiedTurn) (*c.AsyncJobProposal, error)
	Reference() c.BundleRef
}

func NewUnified(d Dependencies, opt Options, i UnifiedOptions) (*Service, error) {
	if d.Knowledge == nil || d.Unified == nil || d.UnifiedMemory == nil || d.UnifiedReflection == nil || d.Protocol == nil || d.Gateway == nil || d.Store == nil || d.Finalizer == nil || d.Retriever == nil || i.Reader == nil || i.Bundles == nil || i.Persistent == nil {
		return nil, errors.New("unified orchestrator requires admission, retrieval, baseline, state, model and finalization")
	}
	if i.Now == nil {
		i.Now = time.Now
	}
	a, err := assembler.New(i.Reader, d.Store, pinnedRegistry{})
	if err != nil {
		return nil, err
	}
	a.WithClock(i.Now)
	d.Assembler = a
	committer, err := state.New(i.Reader, pinnedRegistry{})
	if err != nil {
		return nil, err
	}
	committer.WithClock(i.Now)
	finalizer, err := state.NewValidatedFinalizer(d.Finalizer, committer)
	if err != nil {
		return nil, err
	}
	d.Finalizer = finalizer
	o, err := newService(d, opt)
	if err != nil {
		return nil, err
	}
	o.integration = &integration{reader: i.Reader, registry: i.Bundles, committer: committer, protocol: d.Protocol, now: i.Now, allowSyntheticForTests: i.AllowSyntheticForTests, selector: d.Knowledge, unified: d.Unified, persistent: i.Persistent, unifiedMemory: d.UnifiedMemory, unifiedReflection: d.UnifiedReflection}
	return o, nil
}
func (o *Service) prepareUnified(ctx context.Context, req c.TurnRequest, a c.Admission, tr ports.TurnTrace) (c.PreparedTurn, string, error) {
	ctx = c.WithCallAllowance(ctx, o.opts.MaxLLMCalls)
	// Retrieval starts first after durable admission and bundle pinning, without
	// waiting for state, language interpretation, routing or any model decision.
	type retrievalResult struct {
		out c.RetrievalResult
		err error
	}
	retCh := make(chan retrievalResult, 1)
	go func() {
		started := time.Now()
		out, err := trace.Call(ctx, "retrieval.Retrieve", func() any {
			return c.RetrievalRequest{UserID: a.UserID, Turn: c.TurnView{TurnID: a.TurnID, MessageID: a.UserMessageID, Text: req.Text, Lang: req.Locale}}
		}, func(ctx context.Context) (c.RetrievalResult, error) {
			return o.deps.Retriever.Retrieve(ctx, c.RetrievalRequest{UserID: a.UserID, Turn: c.TurnView{TurnID: a.TurnID, MessageID: a.UserMessageID, Text: req.Text, Lang: req.Locale}})
		})
		recordDiagnostic(tr, "retrieval", "retrieve", started, err, 0, false)
		retCh <- retrievalResult{out, err}
	}()
	type baselineResult struct {
		out c.PersistentContext
		err error
	}
	baselineCh := make(chan baselineResult, 1)
	go func() {
		started := time.Now()
		out, err := trace.Call(ctx, "persistent.Load", func() any { return map[string]any{"user": a.UserID, "trigger": a.UserMessageID} }, func(ctx context.Context) (c.PersistentContext, error) {
			return o.integration.persistent.Persistent(ctx, a.UserID, a.UserMessageID)
		})
		recordDiagnostic(tr, "persistent", "load", started, err, 0, false)
		baselineCh <- baselineResult{out, err}
	}()
	started := time.Now()
	prior, err := o.integration.reader.ForTurn(ctx, a.UserID, a.TurnID)
	recordDiagnostic(tr, "state_read", "load", started, err, 0, false)
	if err != nil {
		return c.PreparedTurn{}, "state_read", err
	}
	versions := contextstate.Versions(prior)
	if prior.User.UserID != a.UserID || prior.Session.UserID != a.UserID || prior.Session.ID != a.SessionID || prior.Session.ConversationID != a.ConversationID || prior.Session.ClosedAt != nil || versions.Session != a.Versions.Session || versions.User != a.Versions.User {
		return c.PreparedTurn{}, "state_read", coreError(c.ErrStateConflict, "state_read", nil)
	}
	var ret retrievalResult
	select {
	case ret = <-retCh:
	case <-ctx.Done():
		return c.PreparedTurn{}, "retrieval", ctx.Err()
	}
	if ret.err != nil {
		ret.out.Meta.Degraded = true
		if ret.out.Meta.DegradedReason == "" {
			ret.out.Meta.DegradedReason = "retrieval_failed"
		}
	}
	var baseline baselineResult
	select {
	case baseline = <-baselineCh:
	case <-ctx.Done():
		return c.PreparedTurn{}, "persistent", ctx.Err()
	}
	if baseline.err != nil {
		baseline.out = c.PersistentContext{Degraded: true, Reason: "baseline_read_failed"}
	}
	ctx = contextstate.WithSnapshot(ctx, prior)
	started = time.Now()
	wc, snap, err := o.deps.Assembler.Build(ctx, req, c.SafetySignal{Level: prior.User.Risk.Level}, ret.out)
	recordDiagnostic(tr, "assembler", "assemble", started, err, 0, false)
	if err != nil {
		return c.PreparedTurn{}, "assembler", err
	}
	wc, _ = trace.Call(ctx, "assembler.Layers", func() any {
		return map[string]any{"context": wc, "state": prior, "persistent": baseline.out, "retrieved": ret.out, "budgets": o.opts.ContextBudgets}
	}, func(context.Context) (c.WorkingContext, error) {
		return assembler.BoundActive(assembler.Layers(wc, prior, baseline.out, ret.out, o.opts.ContextBudgets), o.opts.ContextBudgets.Active), nil
	})
	trace.Capture(ctx, "working_context", "context", "assembler.Layers", wc)
	trace.Bind(ctx, trace.ChildRef(ctx, "assembler.Layers", "/output"), trace.ArtifactRef(ctx, "working_context"), "copy")
	wc, _ = trace.Call(ctx, "assembler.hints", func() any {
		return map[string]any{"context": wc, "hints": prior.FutureHints, "focus_thread_id": prior.Session.FocusThreadID, "active_budget": o.opts.ContextBudgets.Active}
	}, func(context.Context) (c.WorkingContext, error) {
		wc.FutureHints = knowledge.Decay(prior.FutureHints, prior.Session.FocusThreadID, o.integration.now())
		return assembler.BoundActive(wc, o.opts.ContextBudgets.Active), nil
	})
	started = time.Now()
	selection, err := o.deps.Knowledge.Select(ctx, knowledge.Input{UserID: a.UserID, Context: wc, Bundle: o.bundleReference(ctx)})
	if tr != nil {
		tr.Record("jev", trace.Observation{Duration: time.Since(started), Error: err, Operation: "select", Attempt: 1, Degraded: selection.Degraded || err != nil, FallbackUsed: selection.Degraded})
	}
	if tr != nil {
		tr.AddUsage(selection.Usage)
	}
	if err != nil {
		return c.PreparedTurn{}, "jev", err
	}
	wc, _ = trace.Call(ctx, "assembler.final", func() any {
		return map[string]any{"context": wc, "selection": selection, "active_budget": o.opts.ContextBudgets.Active}
	}, func(context.Context) (c.WorkingContext, error) {
		wc.SelectedKnowledgeIDs = selection.Selected
		if !selection.Degraded && !selection.TopicContinues {
			wc.FutureHints = nil
		}
		return assembler.BoundActive(wc, o.opts.ContextBudgets.Active), nil
	})
	trace.Capture(ctx, "selected_knowledge", "knowledge_selection", "jev", selection)
	trace.Capture(ctx, "working_context", "context", "assembler.final", wc)
	trace.Bind(ctx, trace.ChildRef(ctx, "assembler.final", "/output"), trace.ArtifactRef(ctx, "working_context"), "copy")
	started = time.Now()
	out, err := o.deps.Unified.Infer(ctx, ports.UnifiedInput{UserID: a.UserID, Context: wc, Knowledge: selection.Resources, Bundle: o.bundleReference(ctx)})
	recordDiagnostic(tr, "unified", "infer", started, err, 1, false)
	if tr != nil {
		tr.AddUsage(out.Usage)
	}
	trace.Bind(ctx, trace.ArtifactRef(ctx, "working_context"), trace.ChildRef(ctx, "unified.Infer", "/input/Context"), "copy")
	if err != nil {
		return c.PreparedTurn{}, "unified", err
	}
	u := out.Turn
	trace.Capture(ctx, "structured_intents", "unified_output", c.UnifiedSchemaVersion, u)
	trace.Bind(ctx, trace.ChildRef(ctx, "unified.Infer", "/output/Turn"), trace.ArtifactRef(ctx, "structured_intents"), "copy")
	// Validate also at the orchestration boundary so substituted test/model ports
	// cannot bypass response/reference validation.
	_, err = trace.Call(ctx, "unified.ValidateReply", func() any {
		return map[string]any{"turn": u, "context": wc}
	}, func(context.Context) (bool, error) {
		e := unified.ValidateReply(u, wc)
		return e == nil, e
	})
	if err != nil {
		return c.PreparedTurn{}, "unified_validate", coreError(c.ErrValidation, "unified_validate", err)
	}
	started = time.Now()
	acceptedTurn, accepted, protocolUI, acceptedIDs, rejected, err := o.resolveProposals(ctx, a, u, wc, snap, selection.Resources)
	recordDiagnostic(tr, "state_validate", "validate", started, err, 0, false)
	if err != nil {
		return c.PreparedTurn{}, "state_validate", err
	}
	u = acceptedTurn
	if wc.Session.PendingQuestion != nil && u.Session != nil {
		answered := u.Session.AnsweredPending
		wc.Turn.ReplyToPending = &answered
		trace.Capture(ctx, "pending_answer", "semantic_signal", "unified.session_intent", map[string]bool{"reply_to_pending": answered})
	}
	jobs := []c.AsyncJobProposal{}
	if o.deps.UnifiedMemory != nil {
		metaCtx := o.memoryJobContext(ctx, a, accepted, wc, c.Extracted{ReferencedContext: u.UsedContext})
		job, e := o.deps.UnifiedMemory.PlanTurnJob(metaCtx, a, u)
		if e != nil {
			if ctx.Err() != nil {
				return c.PreparedTurn{}, "memory_schedule", ctx.Err()
			}
			rejected = append(rejected, c.DeltaRejection{Path: "background_requests.routine_review", RuleCode: "BG-REVIEW", Detail: "routine review deferred to recent-message sweep"})
		}
		if e == nil && job != nil {
			jobs = append(jobs, *job)
		}
	}
	if o.deps.UnifiedReflection != nil {
		reflectionDelta := accepted
		if reflectionDelta.Session.FocusThread == nil && !reflectionDelta.Session.ClearFocus && wc.FocusThread != nil {
			id := wc.FocusThread.ID
			reflectionDelta.Session.FocusThread = &id
		}
		if snap.Session.TurnCount > 0 && (snap.Session.TurnCount+1)%12 == 0 {
			u.Reflection = true
		}
		if job := o.deps.UnifiedReflection.PlanTurnJob(a, u, reflectionDelta); job != nil {
			jobs = append(jobs, *job)
		}
	}

	backgroundJobs, backgroundAccepted, backgroundRejected := o.planBackgroundRequests(ctx, a, out.Turn, wc, accepted, acceptedIDs)
	jobs = append(jobs, backgroundJobs...)
	acceptedIDs = append(acceptedIDs, backgroundAccepted...)
	rejected = append(rejected, backgroundRejected...)
	decisions, err := trace.Call(ctx, "orchestrator.resolveOutput", func() any {
		return map[string]any{"turn": u, "reply_dependencies": out.Turn.ReplyDependencies, "context": wc, "selection": selection, "state": accepted, "accepted_ids": acceptedIDs, "rejected": rejected}
	}, func(context.Context) (*c.TurnDecisions, error) {
		known := map[string]bool{}
		for _, id := range acceptedIDs {
			known[id] = true
		}
		for _, id := range out.Turn.ReplyDependencies {
			if !known[id] {
				return nil, coreError(c.ErrValidation, "reply_dependency", nil)
			}
		}
		catalog := map[string]bool{}
		for _, card := range o.deps.Knowledge.Catalog() {
			catalog[card.ID] = true
		}
		hints := []c.HintProposal{}
		for i, hint := range u.FutureHints {
			valid := true
			for _, id := range hint.ResourceIDs {
				if !catalog[id] {
					valid = false
				}
			}
			if valid {
				hints = append(hints, hint)
			} else {
				rejected = append(rejected, c.DeltaRejection{Path: fmt.Sprintf("future_hints.%d", i), RuleCode: "HINT-001", Detail: "unknown knowledge resource"})
			}
		}
		u.FutureHints = hints
		carry := o.futureCarry(wc, selection, u, accepted)
		return &c.TurnDecisions{Carry: carry, SelectedKnowledge: selection.Selected, Accepted: acceptedIDs, Rejected: rejected, JEVDegraded: selection.Degraded}, nil
	})
	if err != nil {
		return c.PreparedTurn{}, "reply_dependency", err
	}
	trace.Capture(ctx, "output_resolution", "output_resolution", "independent_sections", decisions)
	trace.Bind(ctx, trace.ChildRef(ctx, "orchestrator.resolveOutput", "/output"), trace.ArtifactRef(ctx, "output_resolution"), "copy")
	trace.Capture(ctx, "background_requests", "background_requests", "unified", out.Turn.BackgroundRequests)
	trace.Capture(ctx, "background_jobs", "jobs", "finalizer", jobs)
	trace.Bind(ctx, trace.ChildRef(ctx, "unified.Infer", "/output/Turn/background_requests"), trace.ArtifactRef(ctx, "background_requests"), "copy")
	trace.Bind(ctx, trace.ArtifactRef(ctx, "background_requests"), trace.ArtifactRef(ctx, "background_jobs"), "transform")
	if tr != nil {
		tr.Record("output_resolution", trace.Observation{Count: len(rejected), Degraded: len(rejected) > 0, Reason: "optional_output_rejected", Operation: "validate"})
	}
	route := c.RouteNormal
	if u.Conversation.Objective == c.ObjSafetySupport {
		route = c.RouteCrisis
	}
	if tr != nil {
		tr.SetRoute(route)
	}
	return c.PreparedTurn{TurnID: a.TurnID, UserMessageID: a.UserMessageID, Decisions: decisions, Text: u.Reply, UI: protocolUI, Route: route, Objective: u.Conversation.Objective, Degraded: selection.Degraded || len(rejected) > 0 || ret.out.Meta.Degraded || baseline.out.Degraded, State: accepted, Jobs: jobs, Trace: c.TurnTraceSummary{TurnID: a.TurnID, BundleVersion: o.bundleReference(ctx).Version, Route: route, Objective: u.Conversation.Objective}}, "finalizer", nil
}
func effectiveRisk(u c.UnifiedTurn, w c.WorkingContext) c.RiskLevel {
	r := w.Risk.Current
	if u.RiskRead != nil {
		r = c.MaxRisk(r, u.RiskRead.Level)
	}
	return r
}
func unifiedDelta(a c.Admission, u c.UnifiedTurn, w c.WorkingContext, s ports.StateSnapshot, now time.Time) c.StateDelta {
	d := c.StateDelta{SchemaVersion: "state_delta.v2", UserID: a.UserID, TurnID: a.TurnID, ExpectedVersions: s.Versions, Session: c.SessionPatch{TurnCountInc: 1, LastObjective: u.Conversation.Objective}, User: c.UserStatePatch{PrefsPatch: u.Preferences}}
	if u.Session != nil {
		d.Session.Mode = u.Session.Mode
		d.Session.SummaryReplace = u.Session.Summary
		d.Session.ClearPending = u.Session.AnsweredPending && w.Session.PendingQuestion != nil
	}
	if t := u.Thread; t != nil {
		var id c.ID
		if t.Relation == c.RelNew {
			id = uuid.NewSHA1(a.TurnID, []byte("unified-thread"))
			d.ExpectedVersions.Thread = 0
			d.Thread = &c.ThreadPatch{ThreadID: id, Create: &c.ThreadInit{Topic: *t.Topic, Goal: t.Goal}}
		} else if t.Target != nil {
			id = *t.Target
		} else if s.Focus != nil {
			id = s.Focus.ID
		}
		if c.ValidID(id) {
			if d.Thread == nil {
				d.Thread = &c.ThreadPatch{ThreadID: id}
				for _, existing := range s.Threads {
					if existing.ID == id {
						d.ExpectedVersions.Thread = existing.Version
						if (existing.Phase == c.PhasePaused || existing.Phase == c.PhaseDormant) && t.Relation == c.RelReturns {
							phase := existing.PrevPhase
							d.Thread.Phase = &phase
						}
					}
				}
			}
			if t.Phase != nil {
				d.Thread.Phase = t.Phase
			}
			d.Thread.Touch = true
			d.Thread.Goal = t.Goal
			d.Thread.Slots = t.Slots
			d.Thread.HypothesisSignals = t.Hypotheses
			d.Thread.RelatedThreadIDs = t.Related
			d.Session.FocusThread = &id
			if d.Thread.Phase != nil && (*d.Thread.Phase == c.PhaseResolved || *d.Thread.Phase == c.PhasePaused || *d.Thread.Phase == c.PhaseDormant) {
				d.Session.FocusThread = nil
				d.Session.ClearFocus = true
			}
		}
		d.ThreadOps = t.Operations
		for _, op := range t.Operations {
			if s.Focus != nil && op.ThreadID == s.Focus.ID && (op.Op == "resolve" || op.Op == "pause" || op.Op == "dormant") && d.Session.FocusThread == nil {
				d.Session.ClearFocus = true
			}
		}
	}
	for i, intent := range u.OpenLoops {
		loop := c.OpenLoop{Kind: intent.Kind, Text: intent.Text, Ref: intent.Ref}
		if intent.Op == "add" {
			loop.ID = uuid.NewSHA1(a.TurnID, []byte("open-loop:"+strconv.Itoa(i)))
		} else if intent.ID != nil {
			loop.ID = *intent.ID
		}
		d.OpenLoops = append(d.OpenLoops, c.OpenLoopOp{Op: intent.Op, Loop: loop})
	}
	if u.RiskRead != nil && u.RiskRead.Level.Rank() > s.User.Risk.Level.Rank() {
		d.User.Risk = &c.RiskTransition{From: s.User.Risk.Level, To: u.RiskRead.Level, At: now.UTC(), TurnID: a.TurnID}
		d.Events = append(d.Events, c.DomainEvent{Type: c.EvRiskLevelChanged, Payload: d.User.Risk})
	}
	return d
}

func (o *Service) futureCarry(w c.WorkingContext, selection c.KnowledgeSelection, u c.UnifiedTurn, d c.StateDelta) []c.FutureHint {
	now := o.integration.now().UTC()
	thread := d.Session.FocusThread
	if thread == nil && !d.Session.ClearFocus && w.FocusThread != nil {
		thread = &w.FocusThread.ID
	}
	carry := []c.FutureHint{}
	if selection.TopicContinues || selection.Degraded {
		for _, h := range w.FutureHints {
			if h.RemainingTurns > 0 {
				carry = append(carry, h)
			}
		}
	}
	known := map[string]bool{}
	for _, card := range o.deps.Knowledge.Catalog() {
		known[card.ID] = true
	}
	for _, hint := range u.FutureHints {
		if hint.TTLTurns < 1 || hint.TTLTurns > 3 || len(hint.Topic) > 160 || len(hint.ResourceIDs) > 4 {
			continue
		}
		valid := true
		for _, id := range hint.ResourceIDs {
			if !known[id] {
				valid = false
			}
		}
		if valid {
			carry = append(carry, c.FutureHint{Topic: hint.Topic, ResourceIDs: hint.ResourceIDs, RemainingTurns: hint.TTLTurns, ExpiresAt: now.Add(72 * time.Hour), ThreadID: thread})
		}
	}
	if len(selection.Selected) > 0 && !selection.Degraded {
		carry = append(carry, c.FutureHint{Topic: "recent selection", ResourceIDs: selection.Selected, RemainingTurns: 2, ExpiresAt: now.Add(72 * time.Hour), ThreadID: thread})
	}
	// Keep the latest lifetime for an identical recommendation, without
	// repeating its content or combining independent topics.
	unique := []c.FutureHint{}
	positions := map[string]int{}
	for _, hint := range carry {
		ids := append([]string(nil), hint.ResourceIDs...)
		sort.Strings(ids)
		key, _ := json.Marshal(struct {
			Topic  string
			IDs    []string
			Thread *c.ID
		}{hint.Topic, ids, hint.ThreadID})
		if position, exists := positions[string(key)]; exists {
			unique[position] = hint
		} else {
			positions[string(key)] = len(unique)
			unique = append(unique, hint)
		}
	}
	carry = unique
	// New recommendations take precedence over old entries at the hard cap.
	if len(carry) > 6 {
		carry = carry[len(carry)-6:]
	}
	return carry
}
