package reflection

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Worker struct {
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
	return &Worker{r, m, writer, committer, a, p, cipher, id}, nil
}
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
	l, e := w.repo.ClaimReflection(ctx, w.id, 2*time.Minute, w.assets.Reference(), w.assets.budgets.Quotas.ReflectionPerUserPerDay)
	if e != nil || l == nil {
		return false, e
	}
	worked = true
	modelStarted := false
	defer func() {
		if err == nil {
			return
		}
		code := "memory_retry"
		if errors.Is(err, mem.ErrDenied) || errors.Is(err, mem.ErrAssets) {
			code = "memory_denied"
			if !modelStarted && len(l.Checkpoint) == 0 {
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
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if e := w.repo.Fail(cleanup, *l, code, time.Duration(1<<min(l.Attempts, 8))*time.Second); e != nil && !errors.Is(e, mem.ErrConflict) {
			err = errors.Join(err, e)
		}
	}()
	work, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	var t Task
	if json.Unmarshal(l.Payload, &t) != nil || t.Schema != "reflection_job.v1" || t.UserID != l.UserID || t.TurnID != l.TurnID || !reflect.DeepEqual(t.Bundle, w.assets.Reference()) {
		return true, mem.ErrDenied
	}
	s, e := w.repo.Load(work, t)
	if e != nil {
		return true, e
	}
	if !w.assets.Eligible(t, s) || !authorize(work, w.policy, t, s) {
		return true, mem.ErrDenied
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
	work = withInput(work, t, s)
	m := mem.Meta{TurnID: l.TurnID, UserID: l.UserID, OperationID: l.ID, Bundle: t.Bundle, Job: l, ThreadID: &t.ThreadID}
	work = mem.WithMeta(work, m)
	var cp checkpoint
	if len(l.Checkpoint) > 0 {
		raw, e := w.cipher.Open(work, l.UserID, "reflection-checkpoint:"+l.ID.String(), l.Checkpoint)
		if e != nil || json.Unmarshal(raw, &cp) != nil {
			return true, mem.ErrDenied
		}
		if cp.FormulationVersion != s.Thread.FormulationVersion {
			return true, mem.ErrDenied
		}
	} else {
		modelStarted = true
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
			return true, mem.ErrDenied
		}
		if e = w.repo.Checkpoint(work, *l, encrypted); e != nil {
			return true, e
		}
	}
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
			return true, mem.ErrDenied
		}
		cp.EpisodeOps = output.Ops
		cp.EpisodeUsage = episodeUsage
		for _, op := range cp.EpisodeOps {
			if op.Op != c.OpNoop && (op.Op != c.OpAdd || op.Kind != c.MemEpisode) {
				return true, mem.ErrDenied
			}
			for _, id := range op.EvidenceMsg {
				found := false
				for _, msg := range wi.Messages {
					if id == msg.ID {
						found = true
					}
				}
				if !found {
					return true, mem.ErrDenied
				}
			}
		}
		cp.EpisodeDone = true
		raw, _ := json.Marshal(cp)
		encrypted, e := w.cipher.Seal(work, l.UserID, "reflection-checkpoint:"+l.ID.String(), raw)
		if e != nil {
			return true, mem.ErrDenied
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
		return true, mem.ErrDenied
	}
	if e = w.repo.Verify(work, t, s); e != nil {
		return true, e
	}
	// ReflectionRepository extends the EXISTING Memory Committer transaction;
	// memory ops, formulation, thread proposal audit and job completion are atomic.
	work = w.repo.WithCommit(work, Commit{Task: t, Snapshot: s, Result: result, Usage: cp.Usage, Rejected: rejected})
	m.Consolidate = false
	m.SourceModels = map[string]string{}
	for _, op := range cp.EpisodeOps {
		if cp.EpisodeUsage.Model == "" {
			return true, mem.ErrDenied
		}
		m.SourceModels[mem.Hash(op)] = cp.EpisodeUsage.Model
	}
	work = mem.WithMeta(work, m)
	ops := append(append([]c.MemoryOpProposal{}, result.HypothesisOps...), cp.EpisodeOps...)
	if len(ops) > 8 {
		return true, mem.ErrDenied
	}
	_, e = w.committer.Apply(work, l.UserID, ops, c.ProvenanceSrc{Origin: "reflection", PromptVersion: t.Bundle.Version, Model: cp.Usage.Model})
	if e != nil {
		return true, e
	}
	return true, w.repo.Complete(work, *l)
}
