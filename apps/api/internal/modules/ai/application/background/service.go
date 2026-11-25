package background

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/knowledge"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"gopkg.in/yaml.v3"
	"io"
	"reflect"
	"strings"
	"time"
)

var outputSchema = schema.BackgroundOutputSchema()

func New(ctx context.Context, repo Repository, queue mem.Queue, cipher mem.Cipher, client ports.Client, bundle ports.Bundle, research ports.ResearchPreparer, policy mem.Policy) (*Service, error) {
	if repo == nil || queue == nil || cipher == nil || client == nil || bundle == nil || research == nil || policy == nil {
		return nil, mem.ErrAssets
	}
	raw, e := bundle.Artifact(ctx, "config/background.yaml")
	if e != nil {
		return nil, e
	}
	var cfg struct {
		Schema string `yaml:"schema"`
		Config `yaml:",inline"`
	}
	d := yaml.NewDecoder(bytes.NewReader(raw))
	d.KnownFields(true)
	if d.Decode(&cfg) != nil || d.Decode(new(any)) != io.EOF || cfg.Schema != OutputSchema || !cfg.Config.Valid() {
		return nil, mem.ErrAssets
	}
	prompt, e := bundle.Artifact(ctx, "prompts/intelligence.system.tmpl")
	if e != nil || len(prompt) == 0 {
		return nil, mem.ErrAssets
	}
	cards, resources, e := knowledge.Load(ctx, bundle)
	if e != nil {
		return nil, e
	}
	s := &Service{Repo: repo, Queue: queue, Cipher: cipher, Client: client, Bundle: bundle, Research: research, Policy: policy, Config: cfg.Config, Cards: cards, Frameworks: map[string]Framework{}, prompt: string(prompt)}
	// Two small authored frameworks are supplied in the SAME analysis call. The
	// model chooses none/one/several; no selector, tag classifier or extra call.
	for _, card := range cards {
		if card.Category == "framework" {
			resource := resources[card.ID]
			var framework Framework
			if yaml.Unmarshal([]byte(resource.Content), &framework) != nil || framework.Name == "" || len(framework.Fields) == 0 {
				return nil, mem.ErrAssets
			}
			s.Frameworks[framework.Name] = framework
			s.Knowledge = append(s.Knowledge, resource)
		}
	}
	if len(s.Knowledge) > 8 {
		return nil, mem.ErrAssets
	}
	return s, nil
}
func (s *Service) Reference() c.BundleRef { return s.Bundle.Reference() }
func (s *Service) Plan(ctx context.Context, a c.Admission, r c.BackgroundRequest, scope Scope) (*c.AsyncJobProposal, error) {
	if !scope.Valid() || r.Kind != "memory_intelligence" && r.Kind != "reflection" {
		return nil, mem.ErrDenied
	}
	if r.Scope != nil {
		scope = Scope{Kind: r.Scope.Kind, ThreadIDs: r.Scope.ThreadIDs}
		if !scope.Valid() {
			return nil, mem.ErrDenied
		}
	}
	req := Request{Origin: "unified", Kind: r.Kind, Scope: scope, Objective: r.Objective, Topic: r.Topic, Queries: r.Queries, MessageIDs: []c.ID{a.UserMessageID}, Trigger: a.UserMessageID}
	operation := uuid.NewSHA1(a.TurnID, []byte("intelligence:"+r.ID))
	job, e := s.job(ctx, a.UserID, operation, req)
	return &job, e
}
func (s *Service) job(ctx context.Context, user, operation c.ID, r Request) (c.AsyncJobProposal, error) {
	raw, _ := json.Marshal(r)
	sealed, e := s.Cipher.Seal(ctx, user, "intelligence-task:"+operation.String(), raw)
	if e != nil {
		return c.AsyncJobProposal{}, e
	}
	task := Task{TaskSchema, s.Reference(), operation, sealed}
	payload, _ := json.Marshal(task)
	kind := "memory_write"
	if r.Kind == "reflection" {
		kind = "reflection"
	}
	return c.AsyncJobProposal{Kind: kind, DedupeKey: operation.String(), Payload: payload}, nil
}
func (s *Service) ReviewRecent(ctx context.Context) (err error) {
	ctx, span := trace.Start(ctx, "background.ScheduleBatch", nil)
	defer func() { span.Complete(nil, err, recover()) }()
	batches, e := s.Repo.Candidates(ctx, s.Config)
	if e != nil {
		return e
	}
	for _, batch := range batches {
		if len(batch.MessageIDs) == 0 {
			continue
		}
		op := uuid.NewSHA1(batch.UserID, []byte("review:"+mem.Hash(batch)))
		req := Request{Origin: "periodic", Kind: "memory_intelligence", Scope: Scope{Kind: "user", ThreadIDs: []c.ID{}}, Objective: "Review every supplied unprocessed user message; organize useful explicit information and changes against existing memory", Topic: "recent unprocessed messages", MessageIDs: batch.MessageIDs, Trigger: batch.Trigger, Queries: []string{}}
		job, e := s.job(ctx, batch.UserID, op, req)
		if e != nil {
			return e
		}
		if e = s.Repo.Reserve(ctx, batch, job); e != nil && !errors.Is(e, mem.ErrConflict) && !errors.Is(e, mem.ErrDenied) {
			return e
		}
	}
	return nil
}
func (s *Service) Execute(ctx context.Context, l mem.Lease) (err error) {
	ctx, span := trace.Start(ctx, "background.Intelligence", func() any { return map[string]any{"job_id": l.ID, "kind": l.Kind, "attempt": l.Attempts} })
	defer func() { span.Complete(nil, err, recover()) }()
	var task Task
	var req Request
	if json.Unmarshal(l.Payload, &task) != nil {
		return mem.ErrDenied
	}
	if task.Schema == TaskSchema {
		if !reflect.DeepEqual(task.Bundle, s.Reference()) || !c.ValidID(task.OperationID) {
			return mem.ErrDenied
		}
		raw, e := s.Cipher.Open(ctx, l.UserID, "intelligence-task:"+task.OperationID.String(), task.Data)
		if e != nil || json.Unmarshal(raw, &req) != nil {
			return mem.ErrDenied
		}
	} else if l.Kind == "reflection" && task.Schema == "reflection_job.v1" {
		// Existing scheduler records remain compatible. The active deployment runs
		// them through longitudinal analysis, without its former ProblemTags routing.
		var old struct {
			UserID   c.ID   `json:"user_id"`
			ThreadID c.ID   `json:"thread_id"`
			Trigger  string `json:"trigger"`
		}
		if json.Unmarshal(l.Payload, &old) != nil || old.UserID != l.UserID || !reflect.DeepEqual(task.Bundle, s.Reference()) {
			return mem.ErrDenied
		}
		req = Request{Origin: "scheduler", Kind: "reflection", Scope: Scope{Kind: "threads", ThreadIDs: []c.ID{old.ThreadID}}, Objective: "Review longitudinal evidence, alternatives, contradictions and information gaps", Topic: old.Trigger, MessageIDs: []c.ID{}, Queries: []string{}}
	} else {
		return mem.ErrDenied
	}
	if !req.Scope.Valid() || req.Kind != "memory_intelligence" && req.Kind != "reflection" || len(req.MessageIDs) > 32 || len(req.Queries) > 3 {
		return mem.ErrDenied
	}
	var cp Checkpoint
	if len(l.Checkpoint) > 0 {
		raw, e := s.Cipher.Open(ctx, l.UserID, "intelligence-checkpoint:"+l.ID.String(), l.Checkpoint)
		if e != nil || json.Unmarshal(raw, &cp) != nil || !reflect.DeepEqual(cp.Request, req) {
			return mem.ErrDenied
		}
		if e = s.Repo.Verify(ctx, l, cp.Prepared); e != nil {
			return s.stale(ctx, l, e)
		}
		trace.Skip(ctx, "background.Infer", "durable_checkpoint_reused")
	} else {
		prepared, e := s.Repo.Prepare(ctx, l, req, nil)
		if e != nil {
			return e
		}
		// Primary messages cannot be silently trimmed. Reserve uses their byte
		// budget; unfit batches fail visibly without acknowledging coverage.
		request := req
		if len(request.Queries) == 0 && len(prepared.Input.Messages) > 0 {
			var text strings.Builder
			for _, m := range prepared.Input.Messages {
				if m.Role == c.RoleUser {
					text.WriteString(m.Content)
					text.WriteByte('\n')
				}
			}
			q := []rune(text.String())
			if len(q) > 300 {
				q = q[len(q)-300:]
			}
			request.Queries = []string{string(q)}
		}
		var items []c.MemoryView
		var meta []c.RetrievalMeta
		if len(prepared.Input.Messages) > 0 && len(request.Queries) > 0 {
			trigger := req.Trigger
			turn := l.TurnID
			for _, m := range prepared.Input.Messages {
				if m.Role == c.RoleUser && (!c.ValidID(req.Trigger) || m.ID == req.Trigger) {
					trigger = m.ID
					turn = m.TurnID
				}
			}
			ret, e := s.Research.PrepareResearch(ctx, l.UserID, turn, trigger, c.BackgroundRequest{ID: l.ID.String(), Kind: req.Kind, Objective: req.Objective, Topic: req.Topic, Queries: request.Queries, EvidenceIDs: []c.ID{trigger}})
			if e != nil {
				return e
			}
			items = ret.Items
			meta = ret.Meta
		}
		prepared, e = s.Repo.Prepare(ctx, l, req, items)
		if e != nil {
			return e
		}
		prepared.Input.Retrieval = meta
		prepared.Input.Request.Queries = request.Queries
		prepared.Input.Cards = s.frameworkCards()
		if req.Kind == "reflection" {
			prepared.Input.Knowledge = s.Knowledge
		}
		_, e = trace.Call(ctx, "background.Fit", func() any {
			return map[string]any{"prepared": prepared, "retrieved": items, "input_tokens": s.Config.InputTokens}
		}, func(context.Context) (Prepared, error) {
			err := s.fit(&prepared, items)
			return prepared, err
		})
		if e != nil {
			return e
		}
		cp = Checkpoint{Request: req, Prepared: prepared}
		if len(prepared.Input.Messages) == 0 && req.Origin == "periodic" {
			cp.Output = Output{Schema: OutputSchema}
			cp.Usage.Model = "no_inference_already_reviewed"
		} else {
			if e = s.Repo.Verify(ctx, l, prepared); e != nil {
				return e
			}
			cp.Output, cp.Usage, e = s.infer(ctx, l, prepared.Input)
			if e != nil {
				return e
			}
		}
		raw, _ := json.Marshal(cp)
		if len(raw) > 480<<10 {
			return &c.Error{Code: c.ErrBudget, Step: "intelligence_checkpoint"}
		}
		sealed, e := s.Cipher.Seal(ctx, l.UserID, "intelligence-checkpoint:"+l.ID.String(), raw)
		if e != nil {
			return e
		}
		if e = s.Queue.Checkpoint(ctx, l, sealed); e != nil {
			return e
		}
	}
	// Also repair compatible labels in durable checkpoints from earlier attempts.
	cp.Output, _ = trace.Call(ctx, "background.NormalizeOutput", func() any {
		return map[string]any{"output": cp.Output, "input": cp.Prepared.Input}
	}, func(context.Context) (Output, error) {
		normalizeOutput(&cp.Output, cp.Prepared.Input)
		return cp.Output, nil
	})
	trace.Capture(ctx, "analysis_request", "background_request", req.Origin, req)
	trace.Capture(ctx, "prepared_input", "prepared_background_input", "intelligence", cp.Prepared.Input)
	trace.Capture(ctx, "model_output", "background_output", OutputSchema, cp.Output)
	trace.Bind(ctx, trace.ArtifactRef(ctx, "prepared_input"), trace.ChildRef(ctx, "background.Infer", "/input"), "copy")
	trace.Bind(ctx, trace.ChildRef(ctx, "background.Infer", "/output/output"), trace.ChildRef(ctx, "background.NormalizeOutput", "/input/output"), "copy")
	trace.Bind(ctx, trace.ChildRef(ctx, "background.NormalizeOutput", "/output"), trace.ArtifactRef(ctx, "model_output"), "copy")
	trace.Capture(ctx, "source_revisions", "provenance", "intelligence", cp.Prepared.Sources)
	plan, hints, e := s.resolve(ctx, l, cp.Prepared, cp.Output, cp.Usage)
	if e != nil {
		return e
	}
	trace.Capture(ctx, "resolution", "background_resolution", "intelligence", map[string]any{"accepted": plan.Report.Applied, "rejected": plan.Report.Rejected, "operations": cp.Output.Operations, "hints": hints})
	if len(plan.Report.Rejected) > 0 {
		// Keep the checkpoint and pending coverage intact. The job must not report
		// success while proposed information was rejected and never persisted.
		return &c.Error{Code: c.ErrValidation, Step: "intelligence_resolution"}
	}
	if e = s.Repo.Verify(ctx, l, cp.Prepared); e != nil {
		return s.stale(ctx, l, e)
	}
	return s.Repo.Commit(ctx, l, cp.Prepared, cp.Output, cp.Usage, plan, hints)
}
func (s *Service) stale(ctx context.Context, l mem.Lease, e error) error {
	if errors.Is(e, mem.ErrConflict) {
		if reset, ok := s.Queue.(interface {
			ResetCheckpoint(context.Context, mem.Lease) error
		}); ok {
			if er := reset.ResetCheckpoint(ctx, l); er != nil {
				return er
			}
		}
	}
	return e
}
func (s *Service) frameworkCards() []c.KnowledgeCard {
	out := []c.KnowledgeCard{}
	for _, card := range s.Cards {
		if card.Category == "framework" {
			out = append(out, card)
		}
	}
	return out
}
func (s *Service) infer(ctx context.Context, l mem.Lease, in Input) (out Output, usage c.LLMUsage, err error) {
	ctx, span := trace.Start(ctx, "background.Infer", func() any { return in })
	defer func() {
		span.Complete(func() any { return map[string]any{"output": out, "usage": usage} }, err, recover())
	}()
	raw, _ := json.Marshal(in)
	if (len(raw)+len(s.prompt)+len(outputSchema)+3)/4 > s.Config.InputTokens {
		return out, usage, &c.Error{Code: c.ErrBudget, Step: "intelligence_input"}
	}
	trace.Capture(ctx, "rendered_prompt", "rendered_prompt", "background", string(raw))
	if trace.Enabled(ctx) {
		part := trace.Compose("prepared_input", "user", trace.ArtifactRef(ctx, "rendered_prompt"), string(raw), func(string, map[string]string) []trace.Reference { return []trace.Reference{trace.Ref(ctx, "/input")} })
		trace.Capture(ctx, "prompt_composition", "prompt_composition", "background", []trace.PromptPart{part})
	}
	tier := ports.ComponentMemoryWriter
	if in.Request.Kind == "reflection" {
		tier = ports.ComponentReflection
	}
	response, e := s.Client.Structured(ctx, ports.StructuredRequest{Tier: tier, System: []ports.Block{{Name: "prompts/intelligence.system.tmpl", Text: s.prompt, Cache: true}}, Messages: []ports.ChatMessage{{Role: c.RoleUser, Content: string(raw)}}, Schema: outputSchema, SchemaName: OutputSchema, MaxTokens: s.Config.OutputTokens, Temperature: .2, Meta: c.CallMeta{UserID: l.UserID, TurnID: l.TurnID, Bundle: s.Reference()}, IdempotencyKey: l.ID.String() + ":" + mem.Hash(in)})
	usage = response.Usage
	if usage.Model == "" {
		usage.Model = response.Model
	}
	if e != nil {
		return out, usage, e
	}
	check, e := schema.CompileInline(outputSchema)
	if e != nil || check(response.JSON) != nil {
		return out, usage, &c.Error{Code: c.ErrValidation, Step: "intelligence_output"}
	}
	d := json.NewDecoder(bytes.NewReader(response.JSON))
	d.DisallowUnknownFields()
	if d.Decode(&out) != nil || d.Decode(new(any)) != io.EOF {
		return out, usage, mem.ErrDenied
	}
	return out, usage, nil
}

func (s *Service) SweepInterval() time.Duration {
	return time.Duration(s.Config.SweepSeconds) * time.Second
}
