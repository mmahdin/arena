// Package state validates WP-08 StateDelta proposals before the WP-02 SQL
// finalizer atomically persists assistant text, state, jobs and trace.
// No state table is updated by this package outside that finalizer.
package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Service struct {
	reader  contextstate.Reader
	bundles ports.BundleRegistry
	now     func() time.Time
}

var _ ports.StateCommitter = (*Service)(nil)

func New(reader contextstate.Reader, bundles ports.BundleRegistry) (*Service, error) {
	if reader == nil || bundles == nil {
		return nil, errors.New("WP-08 state committer requires an owner-scoped reader and bundle registry")
	}
	return &Service{reader: reader, bundles: bundles, now: time.Now}, nil
}
func (s *Service) WithClock(now func() time.Time) *Service {
	if now != nil {
		s.now = now
	}
	return s
}

func stateErr(code c.Code, cause error) error {
	return &c.Error{Code: code, Step: "state", Cause: cause, Retryable: code == c.ErrStateConflict || code == c.ErrStateCommit}
}

// ValidateDelta does not mutate its input, and never applies an AI proposal.
// It strips invalid optional changes, reports stable rule codes, and returns
// conflicts as errors rather than silently writing based on stale versions.
func (s *Service) ValidateDelta(ctx context.Context, proposed c.StateDelta) (c.StateDelta, []c.DeltaRejection, error) {
	if err := ctx.Err(); err != nil {
		return c.StateDelta{}, nil, err
	}
	if !c.ValidID(proposed.UserID) || !c.ValidID(proposed.TurnID) {
		return c.StateDelta{}, nil, stateErr(c.ErrValidation, fmt.Errorf("missing delta identity"))
	}
	raw, err := json.Marshal(proposed)
	if err != nil {
		return c.StateDelta{}, nil, stateErr(c.ErrValidation, err)
	}
	var out c.StateDelta
	if err = json.Unmarshal(raw, &out); err != nil {
		return c.StateDelta{}, nil, stateErr(c.ErrValidation, err)
	}
	snapshot, err := s.reader.ForTurn(ctx, out.UserID, out.TurnID)
	if err != nil {
		return c.StateDelta{}, nil, err
	}
	if snapshot.User.UserID != out.UserID || snapshot.Session.UserID != out.UserID || !c.ValidID(snapshot.Session.ID) || snapshot.Session.ClosedAt != nil {
		return c.StateDelta{}, nil, stateErr(c.ErrForbidden, fmt.Errorf("non-owned or closed session"))
	}
	if out.ExpectedVersions.Session != snapshot.Session.Version || out.ExpectedVersions.User != snapshot.User.Version || out.ExpectedVersions.Session < 1 || out.ExpectedVersions.User < 1 {
		return c.StateDelta{}, []c.DeltaRejection{{Path: "expected_versions", RuleCode: "ST-001", Detail: "session/user version changed"}}, stateErr(c.ErrStateConflict, nil)
	}
	policy, err := contextstate.LoadPolicy(ctx, s.bundles, out.UserID)
	if err != nil {
		return c.StateDelta{}, nil, stateErr(c.ErrBundle, err)
	}
	rejections := []c.DeltaRejection{}
	reject := func(path, rule, detail string) {
		rejections = append(rejections, c.DeltaRejection{Path: path, RuleCode: rule, Detail: detail})
	}
	// WP-02's unconditional session/user version increments make one
	// validated turn one increment. No later-stage state mutations are inferred.
	if out.Session.TurnCountInc != 1 {
		out.Session.TurnCountInc = 1
		reject("session.turn_count_inc", "ST-030", "one increment per completed turn")
	}
	if !out.Session.LastObjective.Valid() {
		out.Session.LastObjective = c.ObjReflect
		reject("session.last_objective", "ST-030", "invalid objective")
	}
	if out.Session.ConsecutiveQuestions < 0 {
		out.Session.ConsecutiveQuestions = 0
		reject("session.consecutive_questions", "ST-030", "negative question count")
	}
	if out.Session.Mode != nil && !out.Session.Mode.Valid() {
		out.Session.Mode = nil
		reject("session.mode", "ST-030", "invalid session mode")
	}
	if out.Session.FocusThread != nil {
		id := *out.Session.FocusThread
		t, ok := contextstate.FindThread(snapshot, id)
		isNew := out.Thread != nil && out.Thread.ThreadID == id && out.Thread.Create != nil
		if !isNew && (!ok || t.Phase == c.PhaseResolved || t.Phase == c.PhasePaused || t.Phase == c.PhaseDormant) {
			// SessionPatch has no ClearFocus flag and SQL uses COALESCE.
			// Drop the invalid assignment rather than pretend it was cleared.
			out.Session.FocusThread = nil
			reject("session.focus_thread", "ST-006", "focus must be a current, non-resolved user thread")
		}
	}
	if out.Session.PendingQuestion != nil && out.Session.ClearPending {
		out.Session.ClearPending = false
		reject("session.clear_pending", "ST-031", "cannot set and clear pending question together")
	}
	if out.Session.SummaryDelta != nil {
		delta := firstWords(*out.Session.SummaryDelta, 40)
		if delta != *out.Session.SummaryDelta {
			reject("session.summary_delta", "ST-030", "summary delta capped at 40 words")
		}
		if wordCount(snapshot.Session.Summary)+wordCount(delta) > 120 {
			out.Session.SummaryDelta = nil
			reject("session.summary_delta", "ST-030", "rolling summary cannot exceed 120 words without compaction")
		} else {
			out.Session.SummaryDelta = &delta
		}
	}
	if len(out.Session.EmotionLabels) > 5 {
		out.Session.EmotionLabels = append([]string(nil), out.Session.EmotionLabels[:5]...)
		reject("session.emotion_labels", "ST-030", "emotion trajectory bounded")
	}

	if err = s.validateThread(ctx, snapshot, policy, &out, reject); err != nil {
		return c.StateDelta{}, rejections, err
	}
	// A proposed new focus is valid only if its thread patch survived
	// validation. The WP-02 finalizer cannot clear a focus via SQL COALESCE.
	if out.Session.FocusThread != nil {
		id := *out.Session.FocusThread
		thread, exists := contextstate.FindThread(snapshot, id)
		valid := exists || (out.Thread != nil && out.Thread.ThreadID == id && out.Thread.Create != nil)
		if exists {
			phase := thread.Phase
			if out.Thread != nil && out.Thread.ThreadID == id && out.Thread.Phase != nil {
				phase = *out.Thread.Phase
			}
			for _, op := range out.ThreadOps {
				if op.ThreadID != id {
					continue
				}
				switch op.Op {
				case "pause":
					phase = c.PhasePaused
				case "resolve":
					phase = c.PhaseResolved
				case "dormant":
					phase = c.PhaseDormant
				case "resume":
					phase = thread.PrevPhase
				}
			}
			valid = isActiveThread(phase)
		}
		if !valid {
			out.Session.FocusThread = nil
			reject("session.focus_thread", "ST-006", "focus target missing, rejected or not active")
		}
	}
	s.validateUser(snapshot, policy, &out, reject)
	if err = s.validateProtocol(ctx, snapshot, policy, &out, reject); err != nil {
		return c.StateDelta{}, rejections, err
	}
	s.validateOperations(snapshot, &out, reject)

	if err = ctx.Err(); err != nil {
		return c.StateDelta{}, rejections, err
	}
	return out, rejections, nil
}

func (s *Service) Reconcile(ctx context.Context, userID c.ID) (ports.ReconcileReport, error) {
	if !c.ValidID(userID) {
		return ports.ReconcileReport{}, stateErr(c.ErrValidation, errors.New("missing user"))
	}
	n, err := s.reader.MissingFinalizations(ctx, userID)
	if err != nil {
		return ports.ReconcileReport{}, err
	}
	if n == 0 {
		return ports.ReconcileReport{Repaired: false}, nil
	}
	// WP-02's transaction should make this impossible. Automatically
	// synthesizing missing messages/events would corrupt the durable history.
	return ports.ReconcileReport{Repaired: false, Rejections: []c.DeltaRejection{{Path: "turn_events", RuleCode: "ST-001", Detail: "incomplete finalization requires operator review"}}},
		stateErr(c.ErrStateCommit, fmt.Errorf("%d completed turns lack state finalization", n))
}

// ValidatedFinalizer is a WP-08-owned wrapper, not a change to the WP-05
// Orchestrator. The final database commit still uses WP-02's same *sql.Tx.
type ValidatedFinalizer struct {
	next      ports.TurnFinalizer
	committer *Service
}

var _ ports.TurnFinalizer = (*ValidatedFinalizer)(nil)

func NewValidatedFinalizer(next ports.TurnFinalizer, committer *Service) (*ValidatedFinalizer, error) {
	if next == nil || committer == nil {
		return nil, errors.New("state validator and durable finalizer required")
	}
	return &ValidatedFinalizer{next: next, committer: committer}, nil
}

func (f *ValidatedFinalizer) Complete(ctx context.Context, admission c.Admission, p c.PreparedTurn) (c.TurnResult, error) {
	if admission.TurnID != p.TurnID || admission.UserID != p.State.UserID || admission.TurnID != p.State.TurnID || admission.UserMessageID != p.UserMessageID || !c.ValidID(admission.LeaseToken) {
		return c.TurnResult{}, stateErr(c.ErrValidation, errors.New("admission/state mismatch"))
	}
	// Delegated replay maintains WP-02's exact completed result, rather than
	// revalidating against versions incremented by the previous completion.
	done, err := f.committer.reader.Completed(ctx, admission.UserID, admission.TurnID)
	if err != nil {
		return c.TurnResult{}, err
	}
	if done {
		return f.next.Complete(ctx, admission, p)
	}
	sanitized, rejections, err := f.committer.ValidateDelta(ctx, p.State)
	if err != nil {
		return c.TurnResult{}, err
	}
	if hasQuestion(p.Text) {
		pol, err := contextstate.LoadPolicy(ctx, f.committer.bundles, admission.UserID)
		if err != nil {
			return c.TurnResult{}, stateErr(c.ErrBundle, err)
		}
		snap, err := f.committer.reader.ForTurn(ctx, admission.UserID, admission.TurnID)
		if err != nil {
			return c.TurnResult{}, err
		}
		if snap.Session.ConsecutiveQuestions >= pol.Gate.Questions.MaxConsecutiveQuestionTurns {
			return c.TurnResult{}, stateErr(c.ErrValidation, errors.New("ST-030: reply exceeds question budget"))
		}
		sanitized.Session.ConsecutiveQuestions = snap.Session.ConsecutiveQuestions + 1
		sanitized.Session.ClearPending = false
		if sanitized.Session.PendingQuestion == nil {
			sanitized.Session.PendingQuestion = &c.PendingQuestion{Text: extractQuestion(p.Text), AskedAt: f.committer.now().UTC()}
		}
	} else {
		sanitized.Session.ConsecutiveQuestions = 0
		sanitized.Session.PendingQuestion = nil
		// Only the validated Brain's answered_pending signal may clear an
		// older pending question. No inference is made from lack of punctuation.
	}
	_ = rejections // the caller can separately call ValidateDelta to trace repairs
	p.State = sanitized
	return f.next.Complete(ctx, admission, p)
}
func (f *ValidatedFinalizer) Fail(ctx context.Context, a c.Admission, code c.Code) error {
	return f.next.Fail(ctx, a, code)
}

func wordCount(s string) int { return len(strings.Fields(s)) }
func firstWords(s string, n int) string {
	v := strings.Fields(s)
	if len(v) <= n {
		return s
	}
	return strings.Join(v[:n], " ")
}
func hasQuestion(s string) bool { return strings.ContainsAny(s, "?؟") }
func extractQuestion(s string) string {
	// Persist only the final bounded interrogative clause, not a fabricated
	// question; finalizer fills the assistant message_id atomically.
	end := strings.LastIndexAny(s, "?؟")
	if end < 0 {
		return ""
	}
	start := strings.LastIndexAny(s[:end], "\n.!؟?")
	if start < 0 {
		start = 0
	} else {
		start++
	}
	return firstWords(strings.TrimSpace(s[start:end+1]), 35)
}
