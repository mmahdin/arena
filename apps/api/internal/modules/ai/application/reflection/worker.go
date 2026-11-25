package reflection

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"log/slog"
	"reflect"
	"time"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Worker struct {
	executor mem.BackgroundExecutor

	research  ports.ResearchPreparer
	repo      Repository
	model     ports.Reflector
	writer    ports.MemoryWriter
	committer ports.MemoryCommitter
	assets    *Assets
	policy    mem.Policy
	cipher    mem.Cipher
	id        string
}
type checkpoint struct {
	Research           *ports.BackgroundResearch
	Result             c.ReflectionResult
	Usage              c.LLMUsage
	EpisodeOps         []c.MemoryOpProposal
	ThreadVersion      int64
	FormulationVersion int
	EpisodeDone        bool
	EpisodeUsage       c.LLMUsage
}

func NewWorker(r Repository, m ports.Reflector, writer ports.MemoryWriter, committer ports.MemoryCommitter, a *Assets, p mem.Policy, cipher mem.Cipher, id string) (*Worker, error) {
	if r == nil || m == nil || writer == nil || committer == nil || a == nil || p == nil || cipher == nil || id == "" || len(id) > 100 {
		return nil, mem.ErrAssets
	}
	return &Worker{research: nil, repo: r, model: m, writer: writer, committer: committer, assets: a, policy: p, cipher: cipher, id: id}, nil
}

// NewIntelligenceWorker retains queue quotas, leases and lifecycle only. All
// model preparation/validation belongs to the shared longitudinal executor.
func NewIntelligenceWorker(r Repository, a *Assets, executor mem.BackgroundExecutor, id string) (*Worker, error) {
	if r == nil || a == nil || executor == nil || id == "" || len(id) > 100 {
		return nil, mem.ErrAssets
	}
	return &Worker{repo: r, assets: a, executor: executor, id: id}, nil
}
func (w *Worker) WithExecutor(e mem.BackgroundExecutor) *Worker { w.executor = e; return w }
func (w *Worker) WithResearch(p ports.ResearchPreparer) *Worker { w.research = p; return w }
func (w *Worker) Schedule(ctx context.Context, t Task) error {
	if !reflect.DeepEqual(t.Bundle, w.assets.Reference()) || !c.ValidID(t.UserID) || !c.ValidID(t.ThreadID) || !c.ValidID(t.SessionID) || t.Schema != "reflection_job.v1" || !validTrigger(t.Trigger) {
		return mem.ErrDenied
	}
	s, e := w.repo.Load(ctx, t)
	if e != nil {
		return e
	}
	if !w.assets.Eligible(t, s) || !authorize(ctx, w.policy, t, s) {
		return mem.ErrDenied
	}
	return w.repo.EnqueueReflection(ctx, t)
}
func (w *Worker) Run(ctx context.Context) error {
	var lastLogged time.Time
	var suppressed uint64
	wasFailing := false
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			worked, e := w.RunOnce(ctx)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if e != nil {
				wasFailing = true
				if lastLogged.IsZero() || time.Since(lastLogged) >= 30*time.Second {
					args := append([]any{"component", "ai_worker", "worker", "reflection", "phase", workerPhase(worked), "suppressed_failures", suppressed}, observability.DiagnosticFields(e, observability.Source(0))...)
					slog.Log(ctx, slog.LevelWarn, "ai.worker.retry", args...)
					lastLogged, suppressed = time.Now(), 0
				} else {
					suppressed++
				}
			} else if wasFailing {
				slog.InfoContext(ctx, "ai.worker.recovered", "worker", "reflection", "suppressed_failures", suppressed)
				wasFailing, suppressed = false, 0
				lastLogged = time.Time{}
			}
			delay := time.Second
			if worked && e == nil {
				delay = 0
			}
			timer.Reset(delay)
		}
	}
}
func (w *Worker) RunOnce(ctx context.Context) (worked bool, err error) {
	l, e := w.repo.ClaimReflection(ctx, w.id, time.Duration(w.assets.budgets.Operations.WorkerLeaseMS)*time.Millisecond, w.assets.Reference(), w.assets.budgets.Quotas.ReflectionPerUserPerDay)
	if e != nil || l == nil {
		return false, e
	}
	ctx = observability.WithTurnRef(ctx, observability.TurnRef(l.TurnID))
	jobStarted := time.Now()
	defer func() {
		if err == nil {
			slog.DebugContext(ctx, "ai.worker.job_completed", append([]any{"component", "ai_worker", "worker", "reflection", "duration_ms", time.Since(jobStarted).Milliseconds(), "attempt", l.Attempts}, observability.CorrelationFields(ctx)...)...)
		}
	}()
	ctx, inspectionSpan := trace.StartJob(ctx, l.TurnID, l.ID, "reflection", l.Attempts)
	trace.RecordClaimedJob(ctx, l.TurnID, l.ID, "reflection", l.Attempts)
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"worked": worked} }, err, recover())
	}()
	trace.Secret(ctx, l.Token.String())
	trace.Capture(ctx, "claimed_job", "durable_job", "ai_jobs", l)
	worked = true
	modelStarted := false
	defer func() {
		if err == nil {
			return
		}
		code := "memory_retry"
		if errors.Is(err, mem.ErrDenied) || errors.Is(err, mem.ErrAssets) {
			code = "memory_denied"
			if w.executor == nil && !modelStarted && len(l.Checkpoint) == 0 {
				code = "reflection_ineligible"
			}
		}
		var ce *c.Error
		if errors.As(err, &ce) && ce.Code == c.ErrValidation {
			code = "memory_invalid"
		}
		if ctx.Err() != nil {
			code = "memory_cancelled"
		}
		// This job-level event is distinct from the rate-limited worker-loop
		// summary: it identifies the attempt and the turn without leaking IDs.
		fields := []any{"component", "ai_worker", "worker", "reflection",
			"operation", "run_once", "turn_ref", observability.TurnRef(l.TurnID),
			"attempt", l.Attempts, "result", "job_failed", "failure_code", code}
		fields = append(fields, observability.DiagnosticFields(err, observability.Source(0))...)
		slog.WarnContext(ctx, "ai.worker.job_failed", fields...)
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Duration(w.assets.budgets.Operations.WorkerCleanupMS)*time.Millisecond)
		defer cancel()
		if e := w.repo.Fail(cleanup, *l, code, time.Duration(1<<min(l.Attempts, 8))*time.Second); e != nil && !errors.Is(e, mem.ErrConflict) {
			err = errors.Join(err, e)
		}
	}()
	work, cancel := context.WithTimeout(ctx, time.Duration(w.assets.budgets.Operations.WorkerTimeoutMS)*time.Millisecond)
	defer cancel()
	if w.executor != nil {
		return true, w.executor.Execute(work, *l)
	}
	var t Task
	if json.Unmarshal(l.Payload, &t) != nil || t.Schema != "reflection_job.v1" || t.UserID != l.UserID || t.TurnID != l.TurnID || !reflect.DeepEqual(t.Bundle, w.assets.Reference()) {
		return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
	}
	s, e := w.repo.Load(work, t)
	if e != nil {
		return true, e
	}
	if !w.assets.Eligible(t, s) || !authorize(work, w.policy, t, s) {
		return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
	}
	framework, e := w.assets.Framework(s.Thread)
	if e != nil && t.Trigger == "needs_formulation" {
		return true, e
	}
	if e != nil {
		framework = ""
	}
	in := ports.ReflectionInput{UserID: t.UserID, ThreadID: t.ThreadID, Trigger: t.Trigger, Framework: framework, Bundle: t.Bundle, PrevFormulation: s.Previous, Items: s.Items}
	for _, src := range s.Sources {
		in.Evidence = append(in.Evidence, src.Message)
	}
	if len(t.BackgroundData) > 0 && len(l.Checkpoint) == 0 {
		if w.research == nil {
			return true, mem.ErrAssets
		}
		raw, e := w.cipher.Open(work, l.UserID, "reflection-request:"+l.TurnID.String(), t.BackgroundData)
		if e != nil {
			return true, e
		}
		var request c.BackgroundRequest
		if json.Unmarshal(raw, &request) != nil {
			return true, mem.ErrDenied
		}
		var trigger c.ID
		for _, src := range s.Sources {
			if src.Message.TurnID == l.TurnID && src.Message.Role == c.RoleUser {
				trigger = src.Message.ID
				break
			}
		}
		if !c.ValidID(trigger) {
			return true, mem.ErrDenied
		}
		research, e := w.research.PrepareResearch(work, l.UserID, l.TurnID, trigger, request)
		if e != nil {
			return true, e
		}
		in.Research = &research
	}
	work = withInput(work, t, s)
	m := mem.Meta{TurnID: l.TurnID, UserID: l.UserID, OperationID: l.ID, Bundle: t.Bundle, Job: l, ThreadID: &t.ThreadID}
	work = mem.WithMeta(work, m)
	var cp checkpoint
	if len(l.Checkpoint) > 0 {
		raw, e := w.cipher.Open(work, l.UserID, "reflection-checkpoint:"+l.ID.String(), l.Checkpoint)
		if e != nil || json.Unmarshal(raw, &cp) != nil {
			return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
		}
		if cp.FormulationVersion != s.Thread.FormulationVersion || cp.ThreadVersion != s.Thread.Version {
			return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
		}
		trace.Skip(work, "reflection.Reflect", "durable_checkpoint_reused")
	} else {
		modelStarted = true
		cp.Research = in.Research
		cp.Result, cp.Usage, e = w.model.Reflect(work, in)
		if e != nil {
			return true, e
		}
		cp.ThreadVersion = s.Thread.Version
		cp.FormulationVersion = s.Thread.FormulationVersion
		cp.Result, _, e = w.assets.Validate(work, t, s, cp.Result, w.policy)
		if e != nil {
			return true, e
		}

		raw, _ := json.Marshal(cp)
		encrypted, e := w.cipher.Seal(work, l.UserID, "reflection-checkpoint:"+l.ID.String(), raw)
		if e != nil {
			return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
		}
		if e = w.repo.Checkpoint(work, *l, encrypted); e != nil {
			return true, e
		}
	}
	if cp.Research != nil {
		var trigger c.ID
		for _, src := range s.Sources {
			if src.Message.TurnID == l.TurnID && src.Message.Role == c.RoleUser {
				trigger = src.Message.ID
			}
		}
		if w.research == nil {
			return true, mem.ErrAssets
		}
		if e = w.research.VerifyResearch(work, l.UserID, trigger, *cp.Research); e != nil {
			return true, e
		}
	}
	trace.Capture(work, "checkpoint", "async_checkpoint", "ai_jobs", cp)
	if cp.Result.Episode != nil && !cp.EpisodeDone {
		wi := ports.MemWriterInput{UserID: t.UserID, Bundle: t.Bundle}
		for _, src := range s.Sources {
			if src.SessionID == t.SessionID {
				wi.Messages = append(wi.Messages, src.Message)
			}
		}
		if len(wi.Messages) > 12 {
			wi.Messages = wi.Messages[len(wi.Messages)-12:]
		}
		m.Consolidate = true
		output, episodeUsage, e := w.writer.Propose(mem.WithMeta(work, m), wi)
		if e != nil {
			return true, e
		}
		if output.SchemaVersion != c.MemoryWriterSchemaVersion || len(output.Ops) > 1 {
			return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
		}
		cp.EpisodeOps = output.Ops
		cp.EpisodeUsage = episodeUsage
		for _, op := range cp.EpisodeOps {
			if op.Op != c.OpNoop && (op.Op != c.OpAdd || op.Kind != c.MemEpisode) {
				return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
			}
			for _, id := range op.EvidenceMsg {
				found := false
				for _, msg := range wi.Messages {
					if id == msg.ID {
						found = true
					}
				}
				if !found {
					return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
				}
			}
		}
		cp.EpisodeDone = true
		raw, _ := json.Marshal(cp)
		encrypted, e := w.cipher.Seal(work, l.UserID, "reflection-checkpoint:"+l.ID.String(), raw)
		if e != nil {
			return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
		}
		if e = w.repo.Checkpoint(work, *l, encrypted); e != nil {
			return true, e
		}
	}
	result, rejected, e := w.assets.Validate(work, t, s, cp.Result, w.policy)
	if e != nil {
		return true, e
	}
	if cp.Usage.Model == "" {
		return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
	}
	if e = w.repo.Verify(work, t, s); e != nil {
		return true, e
	}
	// ReflectionRepository extends the EXISTING Memory Committer transaction;
	// memory ops, formulation, thread proposal audit and job completion are atomic.
	work = w.repo.WithCommit(work, Commit{Task: t, Snapshot: s, Result: result, Usage: cp.Usage, Rejected: rejected})
	if cp.Research != nil {
		m.ResearchVersions = cp.Research.Items
	}
	m.Consolidate = false
	m.SourceModels = map[string]string{}
	for _, op := range cp.EpisodeOps {
		if cp.EpisodeUsage.Model == "" {
			return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
		}
		m.SourceModels[mem.Hash(op)] = cp.EpisodeUsage.Model
	}
	work = mem.WithMeta(work, m)
	ops := append(append([]c.MemoryOpProposal{}, result.HypothesisOps...), cp.EpisodeOps...)
	if len(ops) > 8 {
		return true, observability.Detail(mem.ErrDenied, "policy_denied", "job_validation", "memory_access")
	}
	_, e = w.committer.Apply(work, l.UserID, ops, c.ProvenanceSrc{Origin: "reflection", PromptVersion: t.Bundle.Version, Model: cp.Usage.Model})
	if e != nil {
		return true, e
	}
	return true, w.repo.Complete(work, *l)
}

func workerPhase(worked bool) string {
	if worked {
		return "job"
	}
	return "poll"
}
