package memory

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Worker struct {
	queue   Queue
	repo    Repository
	writer  ports.MemoryWriter
	service *Service
	id      string
	ttl     time.Duration
}

func NewWorker(queue Queue, repo Repository, writer ports.MemoryWriter, service *Service, id string) (*Worker, error) {
	if queue == nil || repo == nil || writer == nil || service == nil || id == "" || len(id) > 100 {
		return nil, ErrAssets
	}
	return &Worker{queue, repo, writer, service, id, 2 * time.Minute}, nil
}

// Run is deliberately in-process and serial. Composition may run a bounded
// number of workers with distinct IDs; durable leases arbitrate replicas.
func (w *Worker) Run(ctx context.Context) error {
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
			delay := time.Second
			if worked && e == nil {
				delay = 0
			}
			timer.Reset(delay)
		}
	}
}
func (w *Worker) RunOnce(ctx context.Context) (worked bool, err error) {
	quota := w.service.assets.budgets.Quotas.MemoryWriterPerUserPerHour
	if quota < 1 {
		return false, ErrAssets
	}
	l, e := w.queue.Claim(ctx, w.id, w.ttl, w.service.Reference(), quota)
	if e != nil || l == nil {
		return false, e
	}
	worked = true
	defer func() {
		if err == nil {
			return
		}
		code := "memory_retry"
		if errors.Is(err, ErrDenied) {
			code = "memory_denied"
		}
		var ce *c.Error
		if errors.As(err, &ce) && ce.Code == c.ErrValidation {
			code = "memory_invalid"
		}
		if ctx.Err() != nil {
			code = "memory_cancelled"
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		delay := time.Duration(1<<min(l.Attempts, 8)) * time.Second
		if failure := w.queue.Fail(cleanup, *l, code, delay); failure != nil && !errors.Is(failure, ErrConflict) {
			err = errors.Join(err, failure)
		}
	}()
	// Finish well before expiry. The SQL commit/checkpoint still fences token and
	// database expiry, so a suspended old process cannot mutate a reclaimed job.
	work, cancel := context.WithTimeout(ctx, w.ttl/2)
	defer cancel()
	if l.Kind == "memory_maintain" {
		var task MaintenanceTask
		if json.Unmarshal(l.Payload, &task) != nil || task.Schema != "memory_maintain.v1" || task.At.IsZero() || !reflect.DeepEqual(task.Bundle, w.service.Reference()) {
			return true, ErrDenied
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
		work = WithMeta(work, Meta{UserID: l.UserID, TurnID: l.TurnID, OperationID: l.ID, Bundle: w.service.Reference(), Job: l})
		return true, w.queue.PublishEmbedding(work, *l)
	}
	if l.Kind != "memory_write" {
		return true, ErrDenied
	}
	var task Task
	if json.Unmarshal(l.Payload, &task) != nil || task.Schema != "memory_job.v1" || !c.ValidID(task.OperationID) || !reflect.DeepEqual(task.Bundle, w.service.Reference()) {
		return true, ErrDenied
	}
	plain, e := w.service.opt.Cipher.Open(work, l.UserID, "memory-task:"+l.TurnID.String(), task.Data)
	if e != nil {
		return true, ErrDenied
	}
	var payload taskData
	if json.Unmarshal(plain, &payload) != nil {
		return true, ErrDenied
	}
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
	if len(l.Checkpoint) > 0 {
		raw, e := w.service.opt.Cipher.Open(work, l.UserID, "memory-checkpoint:"+l.ID.String(), l.Checkpoint)
		if e != nil || json.Unmarshal(raw, &checkpoint) != nil {
			return true, ErrDenied
		}
	} else {
		checkpoint.Output, checkpoint.Usage, e = w.writer.Propose(work, in)
		if e != nil {
			return true, e
		}
		if len(checkpoint.Output.Ops)+len(signalOps) > 8 {
			return true, ErrDenied
		}
		checkpoint.Output.Ops = append(checkpoint.Output.Ops, signalOps...)
		raw, _ := json.Marshal(checkpoint)
		encrypted, e := w.service.opt.Cipher.Seal(work, l.UserID, "memory-checkpoint:"+l.ID.String(), raw)
		if e != nil {
			return true, ErrDenied
		}
		if e = w.queue.Checkpoint(work, *l, encrypted); e != nil {
			return true, e
		}
	}
	if checkpoint.Output.SchemaVersion != c.MemoryWriterSchemaVersion || checkpoint.Usage.Model == "" {
		return true, ErrDenied
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
			if !seenMessage(in.Messages, id) {
				return true, ErrDenied
			}
		}
		if op.TargetID != nil && !seenItem(in.Neighbors, *op.TargetID) {
			return true, ErrDenied
		}
	}
	_, e = w.service.Apply(work, l.UserID, checkpoint.Output.Ops, c.ProvenanceSrc{Origin: "memory_writer", PromptVersion: task.Bundle.Version, Model: checkpoint.Usage.Model})
	if e != nil {
		return true, e
	}
	return true, w.queue.Complete(work, *l)
}
