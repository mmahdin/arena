package memory

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"reflect"
	"time"

	"github.com/google/uuid"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Options struct {
	EmbeddingInvalidator ports.EmbeddingInvalidator
	Policy               Policy
	Cipher               Cipher
	Embedder             ports.EmbeddingClient
	Metrics              Metrics
	Now                  func() time.Time
	EmbeddingModel       string
	Dimension            int
}
type Service struct {
	repo   Repository
	assets *Assets
	opt    Options
}

var _ ports.MemoryCommitter = (*Service)(nil)

func New(repo Repository, a *Assets, opt Options) (*Service, error) {
	if repo == nil || a == nil || opt.Policy == nil || opt.Cipher == nil || opt.EmbeddingModel == "" || opt.Dimension < 1 || opt.Dimension > 8192 {
		return nil, ErrAssets
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	return &Service{repo, a, opt}, nil
}
func (s *Service) Reference() c.BundleRef { return s.assets.Reference() }
func (s *Service) Apply(ctx context.Context, user c.ID, ops []c.MemoryOpProposal, origin c.ProvenanceSrc) (report ports.MemoryApplyReport, err error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/memory/service.Apply", func() any { return map[string]any{"user": user, "ops": ops, "origin": origin} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"report": report} }, err, recover())
	}()

	started := time.Now()
	defer func() {
		if s.opt.Metrics != nil {
			s.opt.Metrics.Observe("memory_commit", time.Since(started), c.LLMUsage{}, report.Rejected)
		}
	}()
	m, e := Auth(ctx, user)
	if e != nil {
		return report, e
	}
	if !reflect.DeepEqual(m.Bundle, s.assets.Reference()) || origin.PromptVersion != m.Bundle.Version || origin.Model == "" {
		return report, ErrDenied
	}
	m.RequireCompleted = origin.Origin == "memory_writer"
	for _, op := range ops {
		if op.TargetID != nil {
			m.Targets = append(m.Targets, *op.TargetID)
		}
	}
	raw, e := json.Marshal(c.MemoryWriterOutput{SchemaVersion: c.MemoryWriterSchemaVersion, Ops: ops})
	if e != nil || s.assets.schemas.Validate(c.MemoryWriterSchemaVersion, raw) != nil {
		return report, &c.Error{Code: c.ErrValidation, Step: "memory_commit"}
	}
	hash := Hash(struct {
		Ops    []c.MemoryOpProposal
		Origin c.ProvenanceSrc
	}{ops, origin})
	if old, done, e := s.repo.Receipt(ctx, user, m.OperationID, hash); e != nil || done {
		return old, e
	}
	ids := []c.ID{}
	for _, op := range ops {
		ids = append(ids, op.EvidenceMsg...)
	}
	for attempt := 0; attempt < 3; attempt++ {
		snap, e := s.repo.Snapshot(ctx, user, m, uniqueIDs(ids))
		if e != nil {
			return report, e
		}
		filtered := []Record{}
		for _, r := range snap.Records {
			if r.Item.UserID != user || !c.ValidID(r.Item.ID) {
				return report, ErrDenied
			}
			if r.DeletedAt == nil && allowed(ctx, s.opt.Policy, snap, m, UseSource, "memory", r.Item.ID, nil, origin) {
				filtered = append(filtered, r)
			}
		}
		snap.Records = filtered
		vectors := s.cachedVectors(snap)
		plan, e := s.assets.Reconcile(ctx, snap, m, ops, origin, vectors, s.opt.Policy, s.opt.Now().UTC())
		if e != nil {
			return report, e
		}
		e = s.repo.Save(ctx, m, snap, plan, hash)
		if errors.Is(e, ErrConflict) {
			continue
		}
		return plan.Report, e
	}
	return report, ErrConflict
}

// Cached embeddings are optional reconciliation hints, never write prerequisites.
func (s *Service) cachedVectors(snap Snapshot) map[string][]float32 {
	out := map[string][]float32{}
	for _, r := range snap.Records {
		if len(r.Vector) == s.opt.Dimension && r.VectorModel == s.opt.EmbeddingModel {
			out[ClaimHash(r.Item.Content)] = r.Vector
		}
	}
	return out
}
func (s *Service) Correct(ctx context.Context, user, target c.ID, kind ports.CorrectionKind) (result0 error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/memory/service.Correct", func() any { return map[string]any{"user": user, "target": target, "kind": kind} })
	defer func() { inspectionSpan.Complete(func() any { return map[string]any{} }, result0, recover()) }()

	m, e := Auth(ctx, user)
	if e != nil || !kind.Valid() || !c.ValidID(target) {
		return ErrDenied
	}
	_, e = s.repo.Control(ctx, m, Control{Kind: "correct", Target: target, Correction: kind, CooldownDays: s.assets.thresholds.Hypothesis.RejectionCooldownDays})
	return e
}
func (s *Service) Delete(ctx context.Context, user c.ID, scope ports.DeletionScope) (ports.DeletionReport, error) {
	m, e := Auth(ctx, user)
	if e != nil {
		return ports.DeletionReport{}, e
	}
	r, e := s.repo.Control(ctx, m, Control{Kind: "delete", Scope: scope})
	return r.Deleted, e
}

// Controls are explicit authenticated user commands. They never derive consent
// or style preferences from model output. Legacy pause values are ignored.
func (s *Service) SetControls(ctx context.Context, user c.ID, pause *bool, consent map[string]bool, prefs *c.PrefsView) error {
	m, e := Auth(ctx, user)
	if e != nil {
		return e
	}
	if prefs != nil && ((prefs.Length != "" && !prefs.Length.Valid()) || (prefs.Tone != "" && !prefs.Tone.Valid()) || len(prefs.Avoid) > 10 || len(prefs.Likes) > 10) {
		return ErrDenied
	}
	_, e = s.repo.Control(ctx, m, Control{Kind: "controls", Pause: pause, Consent: consent, Prefs: prefs})
	if e == nil && s.opt.EmbeddingInvalidator != nil {
		invalidate := pause != nil && *pause
		for _, allowed := range consent {
			if !allowed {
				invalidate = true
			}
		}
		if invalidate {
			s.opt.EmbeddingInvalidator.InvalidateEmbeddingUser(user)
		}
	}
	return e
}
func (s *Service) Pin(ctx context.Context, user, target c.ID, pinned bool) error {
	m, e := Auth(ctx, user)
	if e != nil {
		return e
	}
	_, e = s.repo.Control(ctx, m, Control{Kind: "pin", Target: target, Pinned: &pinned})
	return e
}

// RetryJob is explicit recovery after terminal failure. It preserves the
// encrypted checkpoint and cumulative attempt history; all gates run again.
func (s *Service) RetryJob(ctx context.Context, user, job c.ID) error {
	m, e := Auth(ctx, user)
	if e != nil || !c.ValidID(job) {
		return ErrDenied
	}
	_, e = s.repo.Control(ctx, m, Control{Kind: "retry_job", Target: job})
	return e
}

// Suppression is separate from deletion and requires an explicit user command.
func (s *Service) Suppress(ctx context.Context, user c.ID, content string, until *time.Time) error {
	m, e := Auth(ctx, user)
	if e != nil || content == "" {
		return ErrDenied
	}
	_, e = s.repo.Control(ctx, m, Control{Kind: "suppress", ClaimHash: ClaimHash(content), SuppressUntil: until})
	return e
}
func (s *Service) Endorse(ctx context.Context, user, hypothesis, evidence c.ID, content string) (result0 ports.MemoryApplyReport, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/memory/service.Endorse", func() any {
		return map[string]any{"user": user, "hypothesis": hypothesis, "evidence": evidence, "content": content}
	})
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	m, e := Auth(ctx, user)
	if e != nil {
		return ports.MemoryApplyReport{}, e
	}
	return s.Apply(ctx, user, []c.MemoryOpProposal{{Op: c.OpAdd, TargetID: &hypothesis, Kind: c.MemFact, Content: content, Epistemic: c.EpReported, EvidenceMsg: []c.ID{evidence}, ReasonTag: "user_endorsed"}}, c.ProvenanceSrc{Origin: "user_edit", PromptVersion: m.Bundle.Version, Model: "explicit_user_command"})
}

// List is an explicit authenticated user view, independent of AI-use consent.
// Hidden records are excluded; superseded/rejected records retain provenance.
func (s *Service) List(ctx context.Context, user c.ID) ([]c.MemoryItem, error) {
	m, e := Auth(ctx, user)
	if e != nil {
		return nil, e
	}
	m.IncludeInactive = true
	snapshot, e := s.repo.Snapshot(ctx, user, m, nil)
	if e != nil {
		return nil, e
	}
	out := []c.MemoryItem{}
	for _, record := range snapshot.Records {
		if record.DeletedAt == nil {
			out = append(out, record.Item)
		}
	}
	return out, nil
}
func (s *Service) Edit(ctx context.Context, user, target c.ID, proposal c.MemoryOpProposal) (ports.MemoryApplyReport, error) {
	m, e := Auth(ctx, user)
	if e != nil {
		return ports.MemoryApplyReport{}, e
	}
	proposal.Op = c.OpSupersede
	proposal.TargetID = &target
	proposal.ReasonTag = "user_edit"
	return s.Apply(ctx, user, []c.MemoryOpProposal{proposal}, c.ProvenanceSrc{Origin: "user_edit", PromptVersion: m.Bundle.Version, Model: "explicit_user_command"})
}

func (s *Service) planJob(ctx context.Context, a c.Admission, analysis c.TurnAnalysis, memoryWorthy bool, tags []string, claims []c.UserClaim) (result0 *c.AsyncJobProposal, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "memory.PlanTurnJob", func() any {
		return map[string]any{"admission": a, "analysis": analysis, "memory_signals": tags, "claims": claims}
	})
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	m := Meta{UserID: a.UserID, TurnID: a.TurnID, OperationID: a.TurnID, Bundle: s.Reference()}
	if supplied, ok := Metadata(ctx); ok && supplied.UserID == a.UserID {
		m.ThreadID = supplied.ThreadID
		m.Feedback = supplied.Feedback
		m.Signals = supplied.Signals
	}
	if !memoryWorthy && len(m.Feedback.Retrieved) == 0 && len(m.Signals) == 0 {
		return nil, nil
	}
	// Admission planning reads only user state and the current message. The hot
	// set, encrypted memory bodies and embeddings stay off the reply path.
	snap, e := s.repo.Selection(ctx, a.UserID, a.TurnID, a.UserMessageID)
	if e != nil {
		return nil, e
	}
	if !allowed(ctx, s.opt.Policy, snap, m, Extract, "turn", a.TurnID, nil, c.ProvenanceSrc{Origin: "memory_writer"}) {
		return nil, nil
	}
	source, ok := snap.Sources[a.UserMessageID]
	if !ok || source.Future || source.Hidden && !allowed(ctx, s.opt.Policy, snap, m, UseRetainedSource, "message", a.UserMessageID, nil, c.ProvenanceSrc{Origin: "memory_writer"}) || !allowed(ctx, s.opt.Policy, snap, m, UseSource, "message", a.UserMessageID, nil, c.ProvenanceSrc{Origin: "memory_writer"}) {
		return nil, nil
	}
	if !memoryWorthy && len(m.Signals) == 0 {
		payload, _ := json.Marshal(FeedbackTask{Schema: "memory_feedback.v1", Bundle: s.Reference(), Feedback: m.Feedback})
		return &c.AsyncJobProposal{Kind: "memory_feedback", DedupeKey: a.TurnID.String(), Payload: payload}, nil
	}
	task := Task{Schema: "memory_job.v1", Bundle: m.Bundle, OperationID: m.OperationID}
	if supplied, ok := Metadata(ctx); ok && supplied.UserID == a.UserID {
		m.Feedback = supplied.Feedback
		m.Signals = supplied.Signals
	}
	plain, e := json.Marshal(taskData{ThreadID: m.ThreadID, Analysis: analysis, Claims: claims, MemoryTags: tags, Feedback: m.Feedback, Signals: m.Signals})
	if e != nil {
		return nil, e
	}
	task.Data, e = s.opt.Cipher.Seal(ctx, a.UserID, "memory-task:"+a.TurnID.String(), plain)
	if e != nil {
		return nil, ErrDenied
	}
	payload, _ := json.Marshal(task)
	return &c.AsyncJobProposal{Kind: "memory_write", DedupeKey: a.TurnID.String(), Payload: payload}, nil
}

// Consolidate schedules a bounded Memory Writer pass over explicit source IDs;
// it is not WP-14 session reflection, formulation, or protocol processing.
func (s *Service) Consolidate(ctx context.Context, user, turn c.ID, messageIDs []c.ID) (result0 error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/memory/service.Consolidate", func() any { return map[string]any{"user": user, "turn": turn, "messageIDs": messageIDs} })
	defer func() { inspectionSpan.Complete(func() any { return map[string]any{} }, result0, recover()) }()

	m, e := Auth(ctx, user)
	if e != nil || len(messageIDs) < 2 || len(messageIDs) > 12 {
		return ErrDenied
	}
	queue, ok := s.repo.(Queue)
	if !ok {
		return ErrAssets
	}
	m.TurnID = turn
	m.RequireCompleted = true
	snapshot, e := s.repo.Snapshot(ctx, user, m, messageIDs)
	if e != nil {
		return e
	}
	if !allowed(ctx, s.opt.Policy, snapshot, m, Extract, "turn", turn, nil, c.ProvenanceSrc{Origin: "memory_writer"}) {
		return ErrDenied
	}
	task := Task{Schema: "memory_job.v1", Bundle: s.Reference(), OperationID: m.OperationID, Consolidate: true}
	plain, _ := json.Marshal(taskData{SourceIDs: uniqueIDs(messageIDs)})
	task.Data, e = s.opt.Cipher.Seal(ctx, user, "memory-task:"+turn.String(), plain)
	if e != nil {
		return ErrDenied
	}
	raw, _ := json.Marshal(task)
	return queue.Enqueue(ctx, user, turn, c.AsyncJobProposal{Kind: "memory_write", DedupeKey: m.OperationID.String(), Payload: raw})
}
func (s *Service) Maintain(ctx context.Context, user c.ID, now time.Time) (result0 ports.MaintenanceReport, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/memory/service.Maintain", func() any { return map[string]any{"user": user, "now": now} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	m, e := Auth(ctx, user)
	if e != nil {
		return ports.MaintenanceReport{}, e
	}
	m.OperationID = uuid.NewSHA1(user, []byte("memory-maintenance:"+now.UTC().Format("2006-01-02")))
	hash := Hash("maintenance:" + now.UTC().Format("2006-01-02"))
	if _, done, e := s.repo.Receipt(ctx, user, m.OperationID, hash); done || e != nil {
		return ports.MaintenanceReport{}, e
	}
	for retry := 0; retry < 3; retry++ {
		snap, e := s.repo.Snapshot(ctx, user, m, nil)
		if e != nil {
			return ports.MaintenanceReport{}, e
		}
		filtered := []Record{}
		for _, record := range snap.Records {
			if record.Item.UserID != user || !c.ValidID(record.Item.ID) {
				return ports.MaintenanceReport{}, ErrDenied
			}
			if allowed(ctx, s.opt.Policy, snap, m, UseSource, "memory", record.Item.ID, nil, c.ProvenanceSrc{Origin: "memory_writer"}) && allowed(ctx, s.opt.Policy, snap, m, Store, "memory", record.Item.ID, nil, c.ProvenanceSrc{Origin: "memory_writer"}) {
				filtered = append(filtered, record)
			}
		}
		snap.Records = filtered
		vectors := s.cachedVectors(snap)
		plan, report := s.assets.maintain(ctx, snap, m, s.opt.Policy, vectors, now.UTC())
		if _, done, e := s.repo.Receipt(ctx, user, m.OperationID, hash); done || e != nil {
			return ports.MaintenanceReport{}, e
		}
		e = s.repo.Save(ctx, m, snap, plan, hash)
		if errors.Is(e, ErrConflict) {
			continue
		}
		return report, e
	}
	return ports.MaintenanceReport{}, ErrConflict
}

// ScheduleMaintenance is an explicit host hook for nightly/session-close
// memory housekeeping. Psychological reflection and its triggers are WP-14.
func (s *Service) ScheduleMaintenance(ctx context.Context, user c.ID, at time.Time) error {
	m, e := Auth(ctx, user)
	if e != nil || !c.ValidID(m.TurnID) || at.IsZero() {
		return ErrDenied
	}
	queue, ok := s.repo.(Queue)
	if !ok {
		return ErrAssets
	}
	raw, _ := json.Marshal(MaintenanceTask{Schema: "memory_maintain.v1", Bundle: s.Reference(), At: at.UTC()})
	return queue.Enqueue(ctx, user, m.TurnID, c.AsyncJobProposal{Kind: "memory_maintain", DedupeKey: at.UTC().Format("2006-01-02"), Payload: raw})
}

// PlanTurnJob bridges only the historical encrypted worker task fields. Model
// signals are hints: every admitted nonempty user turn receives an independent
// Writer review, subject to existing account/use policy and worker quotas.
func (s *Service) PlanTurnJob(ctx context.Context, a c.Admission, u c.UnifiedTurn) (*c.AsyncJobProposal, error) {
	analysis := c.TurnAnalysis{Intent: u.Conversation.Intent, MessageType: c.MessageDisclosure, ThreadRelation: c.RelContinues, UserReadiness: c.ConfLow}
	if u.Thread != nil {
		analysis.ThreadRelation = u.Thread.Relation
		analysis.TargetThreadID = u.Thread.Target
		analysis.NewThreadTopic = u.Thread.Topic
	}
	return s.planJob(ctx, a, analysis, true, u.MemorySignals, nil)
}

// PlanFeedbackJob schedules pending operational feedback without requiring a
// conversational model's memory-worthiness decision.
func (s *Service) PlanFeedbackJob(ctx context.Context, a c.Admission) (*c.AsyncJobProposal, error) {
	return s.planJob(ctx, a, c.TurnAnalysis{}, false, nil, nil)
}
