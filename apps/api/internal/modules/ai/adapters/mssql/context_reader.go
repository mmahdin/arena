package mssql

// This WP-08 adapter is read-only. It reads the WP-02 AI state tables in a
// short SQL transaction, without creating a duplicate database or state writer.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type ContextReader struct{ db *sql.DB }

var _ contextstate.Reader = (*ContextReader)(nil)

func NewContextReader(db *sql.DB) (*ContextReader, error) {
	if db == nil {
		return nil, errors.New("WP-08 context read database is nil")
	}
	return &ContextReader{db: db}, nil
}

func (r *ContextReader) Read(ctx context.Context, userID, conversationID c.ID) (contextstate.Snapshot, error) {
	var out contextstate.Snapshot
	if !c.ValidID(userID) || !c.ValidID(conversationID) {
		return out, fmt.Errorf("context owner identity required")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err = loadSession(ctx, tx, userID, conversationID, &out.Session); err != nil {
		return out, err
	}
	if err = loadUser(ctx, tx, userID, &out.User); err != nil {
		return out, err
	}
	if err = loadThreads(ctx, tx, userID, &out.Threads); err != nil {
		return out, err
	}
	if err = loadProtocol(ctx, tx, userID, &out.Protocol); err != nil {
		return out, err
	}
	if err = loadLoops(ctx, tx, userID, &out.OpenLoops); err != nil {
		return out, err
	}
	if err = loadFollowUps(ctx, tx, userID, &out.FollowUps); err != nil {
		return out, err
	}
	if err = loadFormulation(ctx, tx, userID, out.Session.FocusThreadID, &out.Formulation); err != nil {
		return out, err
	}
	// dbo.ai_turn_context does not store per-thread message membership.
	// Do not guess thread user-turn counts from conversation-wide turns.
	out.ThreadUserTurns = map[c.ID]int{}
	if err = tx.Commit(); err != nil {
		return contextstate.Snapshot{}, err
	}
	return out, nil
}

func (r *ContextReader) ForTurn(ctx context.Context, userID, turnID c.ID) (contextstate.Snapshot, error) {
	var conv c.ID
	if !c.ValidID(userID) || !c.ValidID(turnID) {
		return contextstate.Snapshot{}, fmt.Errorf("turn owner identity required")
	}
	err := r.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),t.conversation_id) FROM dbo.turns t JOIN dbo.conversations v ON v.id=t.conversation_id AND v.user_id=t.user_id WHERE t.id=@p1 AND t.user_id=@p2 AND t.lifecycle_version=1 AND v.deleted_at IS NULL`, turnID, userID).Scan(&conv)
	if err != nil {
		return contextstate.Snapshot{}, err
	}
	return r.Read(ctx, userID, conv)
}

func (r *ContextReader) UserEvidence(ctx context.Context, userID c.ID, ids []c.ID) (map[c.ID]bool, error) {
	out := make(map[c.ID]bool, len(ids))
	if !c.ValidID(userID) {
		return nil, fmt.Errorf("evidence owner required")
	}
	if len(ids) > 100 {
		return nil, fmt.Errorf("too many evidence references")
	}
	for _, id := range ids {
		if !c.ValidID(id) {
			continue
		}
		var found int
		err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id WHERE m.id=@p1 AND v.user_id=@p2 AND m.role=N'user' AND m.deleted_at IS NULL AND v.deleted_at IS NULL`, id, userID).Scan(&found)
		if err != nil {
			return nil, err
		}
		if found == 1 {
			out[id] = true
		}
	}
	return out, nil
}

// ExistingHypotheses is an ownership/status check only. It does not load
// the hypothesis text, create memory items, or implement a Memory Writer.
func (r *ContextReader) ExistingHypotheses(ctx context.Context, userID c.ID, ids []c.ID) (map[c.ID]bool, error) {
	out := make(map[c.ID]bool, len(ids))
	if !c.ValidID(userID) || len(ids) > 100 {
		return nil, fmt.Errorf("invalid hypothesis references")
	}
	for _, id := range ids {
		if !c.ValidID(id) {
			continue
		}
		var found int
		err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND user_id=@p2 AND kind=N'hypothesis' AND status IN(N'proposed',N'supported') AND deleted_at IS NULL`, id, userID).Scan(&found)
		if err != nil {
			return nil, err
		}
		if found == 1 {
			out[id] = true
		}
	}
	return out, nil
}

func (r *ContextReader) Completed(ctx context.Context, userID, turnID c.ID) (bool, error) {
	if !c.ValidID(userID) || !c.ValidID(turnID) {
		return false, fmt.Errorf("turn identity required")
	}
	var status string
	err := r.db.QueryRowContext(ctx, `SELECT status FROM dbo.turns WHERE id=@p1 AND user_id=@p2 AND lifecycle_version=1`, turnID, userID).Scan(&status)
	if err != nil {
		return false, err
	}
	return status == "completed", nil
}

func (r *ContextReader) MissingFinalizations(ctx context.Context, userID c.ID) (int, error) {
	if !c.ValidID(userID) {
		return 0, fmt.Errorf("audit owner required")
	}
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.turns t LEFT JOIN dbo.ai_turn_context s ON s.turn_id=t.id AND s.user_id=t.user_id WHERE t.user_id=@p1 AND t.lifecycle_version=1 AND t.status=N'completed' AND (s.turn_id IS NULL OR s.state_committed<>1)`, userID).Scan(&n)
	return n, err
}

func nullableID(value sql.NullString) (*c.ID, error) {
	if !value.Valid || value.String == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value.String)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
func fromJSON(raw string, dest any) error {
	if raw == "" {
		return fmt.Errorf("missing JSON state")
	}
	return json.Unmarshal([]byte(raw), dest)
}
func loadSession(ctx context.Context, tx *sql.Tx, user, conv c.ID, out *c.SessionRecord) error {
	var closed sql.NullTime
	var focus, lastObjective, pending sql.NullString
	var emotions string
	err := tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),s.id),CONVERT(varchar(36),s.user_id),CONVERT(varchar(36),s.conversation_id),s.started_at,s.last_activity_at,s.closed_at,s.mode,CONVERT(varchar(36),s.focus_thread_id),s.turn_count,s.last_objective,s.consecutive_questions,s.pending_question_json,s.summary,s.emotion_trajectory_json,s.version
FROM dbo.ai_sessions s JOIN dbo.conversations v ON v.id=s.conversation_id AND v.user_id=s.user_id
WHERE s.user_id=@p1 AND s.conversation_id=@p2 AND s.closed_at IS NULL AND v.deleted_at IS NULL`, user, conv).Scan(&out.ID, &out.UserID, &out.ConversationID, &out.StartedAt, &out.LastActivityAt, &closed, &out.Mode, &focus, &out.TurnCount, &lastObjective, &out.ConsecutiveQuestions, &pending, &out.Summary, &emotions, &out.Version)
	if err != nil {
		return err
	}
	out.ClosedAt = nil
	if out.FocusThreadID, err = nullableID(focus); err != nil {
		return err
	}
	if lastObjective.Valid {
		v := c.Objective(lastObjective.String)
		out.LastObjective = &v
	}
	if pending.Valid {
		if err = fromJSON(pending.String, &out.PendingQuestion); err != nil {
			return err
		}
	}
	if err = fromJSON(emotions, &out.EmotionTrajectory); err != nil {
		return err
	}
	if out.UserID != user || out.ConversationID != conv {
		return fmt.Errorf("session owner mismatch")
	}
	return nil
}
func loadUser(ctx context.Context, tx *sql.Tx, user c.ID, out *c.UserState) error {
	var prefs, riskHistory, engagement, protocolHistory, consent string
	var riskSince, riskUntil sql.NullTime
	var quiet sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),user_id),tz,locale,prefs_json,risk_level,risk_since,risk_follow_up_until,risk_history_json,engagement_json,dependency_watch,protocol_history_json,proactive_opt_in,quiet_hours_json,memory_paused,consent_json,version FROM dbo.ai_user_state WHERE user_id=@p1`, user).
		Scan(&out.UserID, &out.TZ, &out.Locale, &prefs, &out.Risk.Level, &riskSince, &riskUntil, &riskHistory, &engagement, &out.DependencyWatch, &protocolHistory, &out.ProactiveOptIn, &quiet, &out.MemoryPaused, &consent, &out.Version)
	if err != nil {
		return err
	}
	for _, v := range []struct {
		raw   string
		value any
	}{{prefs, &out.Prefs}, {riskHistory, &out.Risk.History}, {engagement, &out.Engagement}, {protocolHistory, &out.ProtocolHistory}, {consent, &out.Consent}} {
		if err = fromJSON(v.raw, v.value); err != nil {
			return err
		}
	}
	if quiet.Valid {
		if err = fromJSON(quiet.String, &out.QuietHours); err != nil {
			return err
		}
	}
	if riskSince.Valid {
		out.Risk.Since = riskSince.Time
	}
	if riskUntil.Valid {
		out.Risk.FollowUpUntil = &riskUntil.Time
	}
	if out.UserID != user {
		return fmt.Errorf("user state owner mismatch")
	}
	return nil
}
func loadThreads(ctx context.Context, tx *sql.Tx, user c.ID, out *[]c.ThreadRecord) error {
	rows, err := tx.QueryContext(ctx, `SELECT CONVERT(varchar(36),id),CONVERT(varchar(36),user_id),topic,phase,prev_phase,priority,slots_json,goal,hypothesis_ids_json,related_thread_ids_json,formulation_version,problem_tags_json,created_at,last_active_at,version FROM dbo.ai_threads WHERE user_id=@p1 ORDER BY last_active_at DESC,id`, user)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var t c.ThreadRecord
		var prev, goal sql.NullString
		var slots, hypotheses, related, tags string
		if err = rows.Scan(&t.ID, &t.UserID, &t.Topic, &t.Phase, &prev, &t.Priority, &slots, &goal, &hypotheses, &related, &t.FormulationVersion, &tags, &t.CreatedAt, &t.LastActiveAt, &t.Version); err != nil {
			return err
		}
		if t.UserID != user {
			return fmt.Errorf("thread owner mismatch")
		}
		if prev.Valid {
			t.PrevPhase = c.ThreadPhase(prev.String)
		}
		if goal.Valid {
			t.Goal = &goal.String
		}
		for _, v := range []struct {
			raw   string
			value any
		}{{slots, &t.Slots}, {hypotheses, &t.HypothesisIDs}, {related, &t.RelatedThreadIDs}, {tags, &t.ProblemTags}} {
			if err = fromJSON(v.raw, v.value); err != nil {
				return err
			}
		}
		*out = append(*out, t)
	}
	return rows.Err()
}
func loadProtocol(ctx context.Context, tx *sql.Tx, user c.ID, out **c.ProtocolInstance) error {
	var p c.ProtocolInstance
	var thread, step, consentID, outcome, summary sql.NullString
	var offered, consent, completed, followUp sql.NullTime
	var inputs string
	// The live index ensures at most one active/offered; paused is a fallback.
	err := tx.QueryRowContext(ctx, `SELECT TOP(1) CONVERT(varchar(36),id),CONVERT(varchar(36),user_id),CONVERT(varchar(36),thread_id),protocol_id,protocol_version,status,step_id,inputs_json,attempts,stuck_count,offered_at,consent_at,completed_at,CONVERT(varchar(36),consent_message_id),outcome,follow_up_due_at,last_user_summary,version FROM dbo.ai_protocol_instances WHERE user_id=@p1 AND status IN(N'active',N'offered',N'paused') ORDER BY CASE WHEN status=N'active' THEN 0 WHEN status=N'offered' THEN 1 ELSE 2 END,created_at DESC`, user).Scan(&p.ID, &p.UserID, &thread, &p.ProtocolID, &p.ProtocolVersion, &p.Status, &step, &inputs, &p.Attempts, &p.StuckCount, &offered, &consent, &completed, &consentID, &outcome, &followUp, &summary, &p.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if p.UserID != user {
		return fmt.Errorf("protocol owner mismatch")
	}
	if err = fromJSON(inputs, &p.Inputs); err != nil {
		return err
	}
	if p.ThreadID, err = nullableID(thread); err != nil {
		return err
	}
	if step.Valid {
		p.StepID = step.String
	}
	if p.ConsentMessageID, err = nullableID(consentID); err != nil {
		return err
	}
	if outcome.Valid {
		p.Outcome = &outcome.String
	}
	if summary.Valid {
		p.LastUserSummary = summary.String
	}
	if offered.Valid {
		p.OfferedAt = &offered.Time
	}
	if consent.Valid {
		p.ConsentAt = &consent.Time
	}
	if completed.Valid {
		p.CompletedAt = &completed.Time
	}
	if followUp.Valid {
		p.FollowUpDueAt = &followUp.Time
	}
	*out = &p
	return nil
}
func loadLoops(ctx context.Context, tx *sql.Tx, user c.ID, out *[]c.OpenLoop) error {
	rows, err := tx.QueryContext(ctx, `SELECT CONVERT(varchar(36),id),kind,text,CONVERT(varchar(36),ref_id),due_at FROM dbo.ai_open_loops WHERE user_id=@p1 AND status=N'open' ORDER BY created_at DESC,id`, user)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v c.OpenLoop
		var ref sql.NullString
		var due sql.NullTime
		if err = rows.Scan(&v.ID, &v.Kind, &v.Text, &ref, &due); err != nil {
			return err
		}
		if v.Ref, err = nullableID(ref); err != nil {
			return err
		}
		if due.Valid {
			v.DueAt = &due.Time
		}
		*out = append(*out, v)
	}
	return rows.Err()
}
func loadFollowUps(ctx context.Context, tx *sql.Tx, user c.ID, out *[]c.FollowUp) error {
	rows, err := tx.QueryContext(ctx, `SELECT CONVERT(varchar(36),id),reason,due_at,CONVERT(varchar(36),ref_id),status FROM dbo.ai_follow_ups WHERE user_id=@p1 AND status=N'scheduled' ORDER BY due_at DESC,id`, user)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v c.FollowUp
		var ref sql.NullString
		if err = rows.Scan(&v.ID, &v.Reason, &v.DueAt, &ref, &v.Status); err != nil {
			return err
		}
		if v.Ref, err = nullableID(ref); err != nil {
			return err
		}
		v.UserID = user
		*out = append(*out, v)
	}
	return rows.Err()
}
func loadFormulation(ctx context.Context, tx *sql.Tx, user c.ID, focus *c.ID, out **c.FormulationBrief) error {
	if focus == nil {
		return nil
	}
	var v c.FormulationBrief
	var gaps string
	err := tx.QueryRowContext(ctx, `SELECT TOP(1) CONVERT(varchar(36),thread_id),version,paragraph,gaps_json FROM dbo.ai_formulations WHERE user_id=@p1 AND thread_id=@p2 AND deleted_at IS NULL ORDER BY version DESC`, user, *focus).Scan(&v.ThreadID, &v.Version, &v.Paragraph, &gaps)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err = fromJSON(gaps, &v.Gaps); err != nil {
		return err
	}
	*out = &v
	return nil
}
