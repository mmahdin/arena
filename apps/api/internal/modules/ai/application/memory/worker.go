package memory

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"log/slog"
	"reflect"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Worker struct {
	executor BackgroundExecutor

	research     ports.ResearchPreparer
	queue        Queue
	repo         Repository
	writer       ports.MemoryWriter
	service      *Service
	id           string
	ttl          time.Duration
	lastBackfill time.Time
	lastReview   time.Time
}

func (w *Worker) WithExecutor(e BackgroundExecutor) *Worker { w.executor = e; return w }

// NewIntelligenceWorker uses the existing lifecycle/index worker without
// constructing a legacy conversational extraction model.
func NewIntelligenceWorker(queue Queue, repo Repository, service *Service, executor BackgroundExecutor, id string) (*Worker, error) {
	if queue == nil || repo == nil || service == nil || executor == nil || id == "" || len(id) > 100 {
		return nil, ErrAssets
	}
	return &Worker{queue: queue, repo: repo, service: service, executor: executor, id: id, ttl: time.Duration(service.assets.budgets.Operations.WorkerLeaseMS) * time.Millisecond}, nil
}

func (w *Worker) WithResearch(p ports.ResearchPreparer) *Worker { w.research = p; return w }

func NewWorker(queue Queue, repo Repository, writer ports.MemoryWriter, service *Service, id string) (*Worker, error) {
	if queue == nil || repo == nil || writer == nil || service == nil || id == "" || len(id) > 100 {
		return nil, ErrAssets
	}
	return &Worker{queue: queue, repo: repo, writer: writer, service: service, id: id, ttl: time.Duration(service.assets.budgets.Operations.WorkerLeaseMS) * time.Millisecond}, nil
}

// Run is deliberately in-process and serial. Composition may run a bounded
// number of workers with distinct IDs; durable leases arbitrate replicas.
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
			var worked bool
			var e error
			interval := 30 * time.Second
			if source, ok := w.executor.(interface{ SweepInterval() time.Duration }); ok {
				interval = source.SweepInterval()
			}
			if time.Since(w.lastReview) >= interval {
				if w.executor != nil {
					e = w.executor.ReviewRecent(ctx)
				} else {
					e = w.service.ReviewRecent(ctx)
				}
				if e == nil {
					w.lastReview = time.Now()
				}
			}
			// A failed sweep must not block already scheduled jobs.
			var jobErr error
			worked, jobErr = w.RunOnce(ctx)
			e = errors.Join(e, jobErr)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if e != nil {
				wasFailing = true
				if lastLogged.IsZero() || time.Since(lastLogged) >= 30*time.Second {
					args := append([]any{"component", "ai_worker", "worker", "memory", "phase", workerPhase(worked), "suppressed_failures", suppressed}, observability.DiagnosticFields(e, observability.Source(0))...)
					slog.Log(ctx, slog.LevelWarn, "ai.worker.retry", args...)
					lastLogged, suppressed = time.Now(), 0
				} else {
					suppressed++
				}
			} else if wasFailing {
				slog.InfoContext(ctx, "ai.worker.recovered", "worker", "memory", "suppressed_failures", suppressed)
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
	if index, ok := w.queue.(EmbeddingQueue); ok && (w.lastBackfill.IsZero() || time.Since(w.lastBackfill) >= 30*time.Second) {
		if e := index.BackfillEmbeddings(ctx, 128); e != nil {
			return false, e
		}
		w.lastBackfill = time.Now()
	}
	quota := w.service.assets.budgets.Quotas.MemoryWriterPerUserPerHour
	if quota < 1 {
		return false, ErrAssets
	}
	l, e := w.queue.Claim(ctx, w.id, w.ttl, w.service.Reference(), quota)
	if e != nil || l == nil {
		return false, e
	}
	ctx = observability.WithTurnRef(ctx, observability.TurnRef(l.TurnID))
	jobStarted := time.Now()
	jobEvent := "ai.worker.job_completed"
	defer func() {
		if err == nil {
			slog.DebugContext(ctx, jobEvent, append([]any{"component", "ai_worker", "worker", "memory", "duration_ms", time.Since(jobStarted).Milliseconds(), "attempt", l.Attempts}, observability.CorrelationFields(ctx)...)...)
		}
	}()
	ctx, inspectionSpan := trace.StartJob(ctx, l.TurnID, l.ID, "memory", l.Attempts)
	trace.RecordClaimedJob(ctx, l.TurnID, l.ID, "memory", l.Attempts)
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"worked": worked} }, err, recover())
	}()
	trace.Secret(ctx, l.Token.String())
	trace.Capture(ctx, "claimed_job", "durable_job", "ai_jobs", l)
	worked = true
	defer func() {
		if err == nil {
			return
		}
		code := "memory_retry"
		if errors.Is(err, ErrDenied) || errors.Is(err, ErrAssets) {
			code = "memory_denied"
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
		fields := []any{"component", "ai_worker", "worker", "memory",
			"operation", "run_once", "turn_ref", observability.TurnRef(l.TurnID),
			"attempt", l.Attempts, "result", "job_failed", "failure_code", code}
		fields = append(fields, observability.DiagnosticFields(err, observability.Source(0))...)
		slog.WarnContext(ctx, "ai.worker.job_failed", fields...)
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Duration(w.service.assets.budgets.Operations.WorkerCleanupMS)*time.Millisecond)
		defer cancel()
		delay := time.Duration(1<<min(l.Attempts, 8)) * time.Second
		if failure := w.queue.Fail(cleanup, *l, code, delay); failure != nil && !errors.Is(failure, ErrConflict) {
			err = errors.Join(err, failure)
		}
	}()
	// Finish well before expiry. The SQL commit/checkpoint still fences token and
	// database expiry, so a suspended old process cannot mutate a reclaimed job.
	work, cancel := context.WithTimeout(ctx, time.Duration(w.service.assets.budgets.Operations.WorkerTimeoutMS)*time.Millisecond)
	defer cancel()
	var taskVersion struct {
		Schema string `json:"schema"`
	}
	_ = json.Unmarshal(l.Payload, &taskVersion)
	if taskVersion.Schema == "intelligence_job.v2" {
		if w.executor == nil {
			return true, ErrAssets
		}
		return true, w.executor.Execute(work, *l)
	}
	if l.Kind == "memory_maintain" {
		if w.writer == nil {
			return true, ErrAssets
		}
		var task MaintenanceTask
		if json.Unmarshal(l.Payload, &task) != nil || task.Schema != "memory_maintain.v1" || task.At.IsZero() || !reflect.DeepEqual(task.Bundle, w.service.Reference()) {
			return true, observability.Detail(ErrDenied, "policy_denied", "job_validation", "memory_access")
		}
		work = WithMeta(work, Meta{UserID: l.UserID, TurnID: l.TurnID, OperationID: l.ID, Bundle: task.Bundle, Job: l, RequireCompleted: true})
		if _, e := w.service.Maintain(work, l.UserID, task.At); e != nil {
			return true, e
		}
		return true, w.queue.Complete(work, *l)
	}
	if l.Kind == "memory_feedback" {
		work = WithMeta(work, Meta{UserID: l.UserID, TurnID: l.TurnID, OperationID: l.ID, Bundle: w.service.Reference(), Job: l})
		return true, w.queue.RecordFeedback(work, *l)
	}
	if l.Kind == "memory_embed" {
		index, ok := w.queue.(EmbeddingQueue)
		if !ok || w.service.opt.Embedder == nil {
			return true, ErrAssets
		}
		input, eligible, e := index.EmbeddingInput(work, *l)
		if e != nil {
			return true, e
		}
		if !eligible {
			jobEvent = "ai.worker.job_skipped"
			trace.Skip(work, "memory_embed.skipped", "source_ineligible_or_revision_changed")
			// Retain a terminal skip for this source/consent revision. A completed
			// index job may be rebuilt if its index row later needs regeneration.
			return true, w.queue.Fail(work, *l, "memory_denied", 0)
		}
		if lexical, ok := index.(interface {
			PublishLexical(context.Context, Lease, EmbeddingInput) error
		}); ok {
			if e = lexical.PublishLexical(work, *l, input); e != nil {
				return true, e
			}
		}
		response, e := w.service.opt.Embedder.Embed(work, ports.EmbeddingRequest{Consumer: "memory_worker", Tier: ports.ComponentEmbedding, Input: input.Text, Meta: c.CallMeta{UserID: l.UserID, TurnID: l.TurnID, Bundle: w.service.Reference()}})
		if e != nil {
			return true, e
		}
		if response.Model != w.service.opt.EmbeddingModel || len(response.Vector) != w.service.opt.Dimension {
			return true, errors.New("embedding model or dimension mismatch")
		}
		return true, index.PublishEmbedding(work, *l, input, response)
	}
	if l.Kind != "memory_write" {
		return true, observability.Detail(ErrDenied, "policy_denied", "job_validation", "memory_access")
	}
	if w.writer == nil {
		return true, ErrAssets
	}
	var task Task
	if json.Unmarshal(l.Payload, &task) != nil || task.Schema != "memory_job.v1" || !c.ValidID(task.OperationID) || !reflect.DeepEqual(task.Bundle, w.service.Reference()) {
		return true, observability.Detail(ErrDenied, "policy_denied", "job_validation", "memory_access")
	}
	plain, e := w.service.opt.Cipher.Open(work, l.UserID, "memory-task:"+l.TurnID.String(), task.Data)
	if e != nil {
		return true, observability.Detail(errors.Join(ErrDenied, e), "task_decryption_failed", "job_decryption", "memory_access")
	}
	var payload taskData
	if json.Unmarshal(plain, &payload) != nil {
		return true, observability.Detail(ErrDenied, "policy_denied", "job_validation", "memory_access")
	}
	trace.Capture(work, "decrypted_task", "async_task", "ai_jobs", payload)
	task.Analysis = payload.Analysis
	task.Claims = payload.Claims
	task.MemoryTags = payload.MemoryTags
	task.SourceIDs = payload.SourceIDs
	m := Meta{UserID: l.UserID, TurnID: l.TurnID, OperationID: task.OperationID, Bundle: task.Bundle, Job: l, Consolidate: task.Consolidate, MemoryTags: task.MemoryTags, RequireCompleted: true, ThreadID: payload.ThreadID, Feedback: payload.Feedback, Signals: payload.Signals}
	signalOps, e := signalProposals(payload.Signals)
	if e != nil {
		return true, e
	}
	for _, op := range signalOps {
		m.Targets = append(m.Targets, *op.TargetID)
	}
	work = WithMeta(work, m)
	in, e := w.repo.WriterInput(work, m, task)
	if e != nil {
		return true, e
	}
	var checkpoint Checkpoint
	if payload.Request != nil && len(l.Checkpoint) == 0 {
		if w.research == nil || len(in.Messages) == 0 {
			return true, ErrAssets
		}
		trigger := in.Messages[0].ID
		research, e := w.research.PrepareResearch(work, l.UserID, l.TurnID, trigger, *payload.Request)
		if e != nil {
			return true, e
		}
		in.Research = &research
	}
	if len(l.Checkpoint) > 0 {
		raw, e := w.service.opt.Cipher.Open(work, l.UserID, "memory-checkpoint:"+l.ID.String(), l.Checkpoint)
		if e != nil || json.Unmarshal(raw, &checkpoint) != nil {
			return true, observability.Detail(ErrDenied, "policy_denied", "job_validation", "memory_access")
		}
		if checkpoint.Research != nil {
			if w.research == nil || len(in.Messages) == 0 {
				return true, ErrAssets
			}
			if e = w.research.VerifyResearch(work, l.UserID, in.Messages[0].ID, *checkpoint.Research); e != nil {
				return true, e
			}
			in.Research = checkpoint.Research
		}
		trace.Skip(work, "memory_writer.Propose", "durable_checkpoint_reused")
	} else {
		checkpoint.Research = in.Research
		checkpoint.Output, checkpoint.Usage, e = w.writer.Propose(work, in)
		if e != nil {
			return true, e
		}
		if len(checkpoint.Output.Ops)+len(signalOps) > 8 {
			return true, observability.Detail(ErrDenied, "policy_denied", "job_validation", "memory_access")
		}
		checkpoint.Output.Ops = append(checkpoint.Output.Ops, signalOps...)
		raw, _ := json.Marshal(checkpoint)
		encrypted, e := w.service.opt.Cipher.Seal(work, l.UserID, "memory-checkpoint:"+l.ID.String(), raw)
		if e != nil {
			return true, observability.Detail(ErrDenied, "policy_denied", "job_validation", "memory_access")
		}
		if e = w.queue.Checkpoint(work, *l, encrypted); e != nil {
			return true, e
		}
	}
	trace.Capture(work, "checkpoint", "async_checkpoint", "ai_jobs", checkpoint)
	if checkpoint.Output.SchemaVersion != c.MemoryWriterSchemaVersion || checkpoint.Usage.Model == "" {
		return true, observability.Detail(ErrDenied, "policy_denied", "job_validation", "memory_access")
	}
	// Revalidate a recovered proposal against its freshly authorized input too.
	for _, op := range checkpoint.Output.Ops {
		trusted := false
		for _, signal := range signalOps {
			if Hash(op) == Hash(signal) {
				trusted = true
				break
			}
		}
		if trusted {
			continue
		}
		for _, id := range op.EvidenceMsg {
			if !seenEvidence(in, id) {
				return true, observability.Detail(ErrDenied, "policy_denied", "job_validation", "memory_access")
			}
		}
		if op.TargetID != nil && !seenItem(in.Neighbors, *op.TargetID) {
			return true, observability.Detail(ErrDenied, "policy_denied", "job_validation", "memory_access")
		}
	}
	if in.Research != nil {
		if e = w.research.VerifyResearch(work, l.UserID, in.Messages[0].ID, *in.Research); e != nil {
			return true, e
		}
	}
	if in.Research != nil {
		if e = w.repo.VerifyWriter(work, m, in); e != nil {
			return true, e
		}
		m.ResearchVersions = in.Research.Items
		work = WithMeta(work, m)
	}
	_, e = w.service.Apply(work, l.UserID, checkpoint.Output.Ops, c.ProvenanceSrc{Origin: "memory_writer", PromptVersion: task.Bundle.Version, Model: checkpoint.Usage.Model})
	if e != nil {
		return true, e
	}
	return true, w.queue.Complete(work, *l)
}

func workerPhase(worked bool) string {
	if worked {
		return "job"
	}
	return "poll"
}
