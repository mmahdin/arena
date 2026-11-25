package mssql

import "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"

// This WP-08 adapter is read-only. It reads the WP-02 AI state tables in a
// short SQL transaction, without creating a duplicate database or state writer.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type ContextReader struct {
	db           *sql.DB
	hintsCipher  mem.Cipher
	hintsTimeout time.Duration
}

var _ contextstate.Reader = (*ContextReader)(nil)

func NewContextReader(db *sql.DB) (*ContextReader, error) {
	if db == nil {
		return nil, errors.New("WP-08 context read database is nil")
	}
	return &ContextReader{db: db}, nil
}

func (r *ContextReader) Read(ctx context.Context, userID, conversationID c.ID) (result0 contextstate.Snapshot, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "adapters/mssql/context_reader.Read", func() any { return map[string]any{"userID": userID, "conversationID": conversationID} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	var out contextstate.Snapshot
	if !c.ValidID(userID) || !c.ValidID(conversationID) {
		return out, observability.Detail(nil, "identity_mismatch", "context_handles", "state_read")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return out, observability.DatabaseError(err, "state_read")
	}
	defer tx.Rollback()
	if err = loadSession(ctx, tx, userID, conversationID, &out.Session); err != nil {
		return out, observability.DatabaseError(err, "state_read")
	}
	if err = loadUser(ctx, tx, userID, &out.User); err != nil {
		return out, observability.DatabaseError(err, "state_read")
	}
	var carry string
	carryErr := tx.QueryRowContext(ctx, `SELECT TOP(1) d.carry_json FROM dbo.ai_turn_decisions d JOIN dbo.turns t ON t.id=d.turn_id AND t.user_id=d.user_id WHERE d.user_id=@p1 AND t.conversation_id=@p2 AND t.status=N'completed' ORDER BY t.admission_order DESC`, userID, conversationID).Scan(&carry)
	if carryErr != nil && !errors.Is(carryErr, sql.ErrNoRows) {
		return out, observability.DatabaseError(carryErr, "state_read")
	}
	if carryErr == nil && json.Unmarshal([]byte(carry), &out.FutureHints) != nil {
		return out, observability.Detail(nil, "schema_invalid", "carry_over", "state_read")
	}
	if err = loadThreads(ctx, tx, userID, &out.Threads); err != nil {
		return out, observability.DatabaseError(err, "state_read")
	}
	if err = loadProtocol(ctx, tx, userID, &out.Protocol); err != nil {
		return out, observability.DatabaseError(err, "state_read")
	}
	if err = loadLoops(ctx, tx, userID, &out.OpenLoops); err != nil {
		return out, observability.DatabaseError(err, "state_read")
	}
	if err = loadFollowUps(ctx, tx, userID, &out.FollowUps); err != nil {
		return out, observability.DatabaseError(err, "state_read")
	}
	if err = loadFormulation(ctx, tx, userID, out.Session.FocusThreadID, &out.Formulation); err != nil {
		return out, observability.DatabaseError(err, "state_read")
	}
	if err = loadFormulations(ctx, tx, userID, &out.Formulations); err != nil {
		return out, err
	}
	// Immutable accepted state deltas carry thread membership. Count distinct
	// completed user turns, never conversation/session-wide counts.
	out.ThreadUserTurns = map[c.ID]int{}
	rows, e := tx.QueryContext(ctx, `SELECT CONVERT(varchar(36),COALESCE(TRY_CONVERT(uniqueidentifier,JSON_VALUE(e.delta_json,'$.thread.thread_id')),TRY_CONVERT(uniqueidentifier,JSON_VALUE(e.delta_json,'$.session.focus_thread')))),COUNT(DISTINCT e.turn_id) FROM dbo.ai_turn_events e JOIN dbo.turns t ON t.id=e.turn_id AND t.user_id=e.user_id WHERE e.user_id=@p1 AND e.type=N'state_applied' AND t.status=N'completed' AND COALESCE(JSON_VALUE(e.delta_json,'$.thread.thread_id'),JSON_VALUE(e.delta_json,'$.session.focus_thread')) IS NOT NULL GROUP BY COALESCE(TRY_CONVERT(uniqueidentifier,JSON_VALUE(e.delta_json,'$.thread.thread_id')),TRY_CONVERT(uniqueidentifier,JSON_VALUE(e.delta_json,'$.session.focus_thread')))`, userID)
	if e != nil {
		return out, observability.DatabaseError(e, "state_read")
	}
	for rows.Next() {
		var id c.ID
		var count int
		if e = rows.Scan(&id, &count); e != nil {
			rows.Close()
			return out, observability.DatabaseError(e, "state_read")
		}
		out.ThreadUserTurns[id] = count
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, observability.DatabaseError(e, "state_read")
	}
	if err = tx.Commit(); err != nil {
		return contextstate.Snapshot{}, observability.DatabaseError(err, "state_read")
	}
	hintCtx, cancel := context.WithCancel(ctx)
	if r.hintsTimeout > 0 {
		cancel()
		hintCtx, cancel = context.WithTimeout(ctx, r.hintsTimeout)
	}
	defer cancel()
	if hintErr := r.backgroundHints(hintCtx, userID, &out); hintErr != nil {
		keep := []c.FutureHint{}
		for _, hint := range out.FutureHints {
			if hint.JobID == nil {
				keep = append(keep, hint)
			}
		}
		out.FutureHints = keep
		trace.Capture(ctx, "background_hints_degraded", "degraded", "optional_background_hints", map[string]string{"reason": "unavailable"})
	}
	return out, nil
}

func (r *ContextReader) ForTurn(ctx context.Context, userID, turnID c.ID) (result0 contextstate.Snapshot, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "adapters/mssql/context_reader.ForTurn", func() any { return map[string]any{"userID": userID, "turnID": turnID} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	var conv c.ID
	if !c.ValidID(userID) || !c.ValidID(turnID) {
		return contextstate.Snapshot{}, observability.Detail(nil, "identity_mismatch", "context_handles", "state_read")
	}
	err := r.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),t.conversation_id) FROM dbo.turns t JOIN dbo.conversations v ON v.id=t.conversation_id AND v.user_id=t.user_id WHERE t.id=@p1 AND t.user_id=@p2 AND t.lifecycle_version=1 AND v.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM dbo.turns later WHERE later.user_id=t.user_id AND later.admission_order>t.admission_order AND later.status=N'completed')`, turnID, userID).Scan(&conv)
	if err != nil {
		return contextstate.Snapshot{}, observability.DatabaseError(err, "state_read")
	}
	return r.Read(context.WithValue(ctx, hintTurnKey{}, turnID), userID, conv)
}

func (r *ContextReader) UserEvidence(ctx context.Context, userID c.ID, ids []c.ID) (result0 map[c.ID]bool, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "adapters/mssql/context_reader.UserEvidence", func() any { return map[string]any{"userID": userID, "ids": ids} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	out := make(map[c.ID]bool, len(ids))
	if !c.ValidID(userID) {
		return nil, observability.Detail(nil, "identity_mismatch", "context_handles", "state_read")
	}
	if len(ids) > 100 {
		return nil, observability.Detail(nil, "schema_invalid", "context_handles", "state_read")
	}
	for _, id := range ids {
		if !c.ValidID(id) {
			continue
		}
		var found int
		err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id WHERE m.id=@p1 AND v.user_id=@p2 AND m.role=N'user' AND m.deleted_at IS NULL AND v.deleted_at IS NULL`, id, userID).Scan(&found)
		if err != nil {
			return nil, observability.DatabaseError(err, "state_read")
		}
		if found == 1 {
			out[id] = true
		}
	}
	return out, nil
}

// ExistingHypotheses is an ownership/status check only. It does not load
// the hypothesis text, create memory items, or implement a Memory Writer.
func (r *ContextReader) ExistingHypotheses(ctx context.Context, userID c.ID, ids []c.ID) (result0 map[c.ID]bool, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "adapters/mssql/context_reader.ExistingHypotheses", func() any { return map[string]any{"userID": userID, "ids": ids} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	out := make(map[c.ID]bool, len(ids))
	if !c.ValidID(userID) || len(ids) > 100 {
		return nil, observability.Detail(nil, "schema_invalid", "context_handles", "state_read")
	}
	for _, id := range ids {
		if !c.ValidID(id) {
			continue
		}
		var found int
		err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND user_id=@p2 AND kind=N'hypothesis' AND status IN(N'proposed',N'supported') AND deleted_at IS NULL`, id, userID).Scan(&found)
		if err != nil {
			return nil, observability.DatabaseError(err, "state_read")
		}
		if found == 1 {
			out[id] = true
		}
	}
	return out, nil
}

func (r *ContextReader) Completed(ctx context.Context, userID, turnID c.ID) (bool, error) {
	if !c.ValidID(userID) || !c.ValidID(turnID) {
		return false, observability.Detail(nil, "identity_mismatch", "context_handles", "state_read")
	}
	var status string
	err := r.db.QueryRowContext(ctx, `SELECT status FROM dbo.turns WHERE id=@p1 AND user_id=@p2 AND lifecycle_version=1`, turnID, userID).Scan(&status)
	if err != nil {
		return false, observability.DatabaseError(err, "state_read")
	}
	return status == "completed", nil
}

func (r *ContextReader) MissingFinalizations(ctx context.Context, userID c.ID) (int, error) {
	if !c.ValidID(userID) {
		return 0, observability.Detail(nil, "identity_mismatch", "context_handles", "state_read")
	}
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.turns t LEFT JOIN dbo.ai_turn_context s ON s.turn_id=t.id AND s.user_id=t.user_id WHERE t.user_id=@p1 AND t.lifecycle_version=1 AND t.status=N'completed' AND (s.turn_id IS NULL OR s.state_committed<>1)`, userID).Scan(&n)
	return n, observability.DatabaseError(err, "state_read")
}

func nullableID(value sql.NullString) (*c.ID, error) {
	if !value.Valid || value.String == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value.String)
	if err != nil {
		return nil, observability.DatabaseError(err, "state_read")
	}
	return &id, nil
}
func fromJSON(raw string, dest any) error {
	if raw == "" {
		return observability.Detail(nil, "schema_invalid", "context_state", "state_read")
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return observability.Detail(err, "schema_invalid", "context_state", "state_read")
	}
	return nil
}
func loadSession(ctx context.Context, tx *sql.Tx, user, conv c.ID, out *c.SessionRecord) error {
	var closed sql.NullTime
	var focus, lastObjective, pending sql.NullString
	var emotions string
	err := tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),s.id),CONVERT(varchar(36),s.user_id),CONVERT(varchar(36),s.conversation_id),s.started_at,s.last_activity_at,s.closed_at,s.mode,CONVERT(varchar(36),s.focus_thread_id),s.turn_count,s.last_objective,s.consecutive_questions,s.pending_question_json,s.summary,s.emotion_trajectory_json,s.version
FROM dbo.ai_sessions s JOIN dbo.conversations v ON v.id=s.conversation_id AND v.user_id=s.user_id
WHERE s.user_id=@p1 AND s.conversation_id=@p2 AND s.closed_at IS NULL AND v.deleted_at IS NULL`, user, conv).Scan(&out.ID, &out.UserID, &out.ConversationID, &out.StartedAt, &out.LastActivityAt, &closed, &out.Mode, &focus, &out.TurnCount, &lastObjective, &out.ConsecutiveQuestions, &pending, &out.Summary, &emotions, &out.Version)
	if err != nil {
		return observability.DatabaseError(err, "state_read")
	}
	out.ClosedAt = nil
	if out.FocusThreadID, err = nullableID(focus); err != nil {
		return observability.DatabaseError(err, "state_read")
	}
	if lastObjective.Valid {
		v := c.Objective(lastObjective.String)
		out.LastObjective = &v
	}
	if pending.Valid {
		if err = fromJSON(pending.String, &out.PendingQuestion); err != nil {
			return observability.DatabaseError(err, "state_read")
		}
	}
	if err = fromJSON(emotions, &out.EmotionTrajectory); err != nil {
		return observability.DatabaseError(err, "state_read")
	}
	if out.UserID != user || out.ConversationID != conv {
		return observability.Detail(nil, "ownership_mismatch", "context_handles", "state_read")
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
		return observability.DatabaseError(err, "state_read")
	}
	for _, v := range []struct {
		raw   string
		value any
	}{{prefs, &out.Prefs}, {riskHistory, &out.Risk.History}, {engagement, &out.Engagement}, {protocolHistory, &out.ProtocolHistory}, {consent, &out.Consent}} {
		if err = fromJSON(v.raw, v.value); err != nil {
			return observability.DatabaseError(err, "state_read")
		}
	}
	if quiet.Valid {
		if err = fromJSON(quiet.String, &out.QuietHours); err != nil {
			return observability.DatabaseError(err, "state_read")
		}
	}
	if riskSince.Valid {
		out.Risk.Since = riskSince.Time
	}
	if riskUntil.Valid {
		out.Risk.FollowUpUntil = &riskUntil.Time
	}
	if out.UserID != user {
		return observability.Detail(nil, "ownership_mismatch", "context_state", "state_read")
	}
	return nil
}
func loadThreads(ctx context.Context, tx *sql.Tx, user c.ID, out *[]c.ThreadRecord) error {
	rows, err := tx.QueryContext(ctx, `SELECT CONVERT(varchar(36),id),CONVERT(varchar(36),user_id),topic,phase,prev_phase,priority,slots_json,goal,hypothesis_ids_json,related_thread_ids_json,formulation_version,problem_tags_json,created_at,last_active_at,version FROM dbo.ai_threads WHERE user_id=@p1 ORDER BY last_active_at DESC,id`, user)
	if err != nil {
		return observability.DatabaseError(err, "state_read")
	}
	defer rows.Close()
	for rows.Next() {
		var t c.ThreadRecord
		var prev, goal sql.NullString
		var slots, hypotheses, related, tags string
		if err = rows.Scan(&t.ID, &t.UserID, &t.Topic, &t.Phase, &prev, &t.Priority, &slots, &goal, &hypotheses, &related, &t.FormulationVersion, &tags, &t.CreatedAt, &t.LastActiveAt, &t.Version); err != nil {
			return observability.DatabaseError(err, "state_read")
		}
		if t.UserID != user {
			return observability.Detail(nil, "ownership_mismatch", "context_handles", "state_read")
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
				return observability.DatabaseError(err, "state_read")
			}
		}
		*out = append(*out, t)
	}
	return observability.DatabaseError(rows.Err(), "state_read")
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
		return observability.DatabaseError(err, "state_read")
	}
	if p.UserID != user {
		return observability.Detail(nil, "ownership_mismatch", "context_handles", "state_read")
	}
	if err = fromJSON(inputs, &p.Inputs); err != nil {
		return observability.DatabaseError(err, "state_read")
	}
	if p.ThreadID, err = nullableID(thread); err != nil {
		return observability.DatabaseError(err, "state_read")
	}
	if step.Valid {
		p.StepID = step.String
	}
	if p.ConsentMessageID, err = nullableID(consentID); err != nil {
		return observability.DatabaseError(err, "state_read")
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
		return observability.DatabaseError(err, "state_read")
	}
	defer rows.Close()
	for rows.Next() {
		var v c.OpenLoop
		var ref sql.NullString
		var due sql.NullTime
		if err = rows.Scan(&v.ID, &v.Kind, &v.Text, &ref, &due); err != nil {
			return observability.DatabaseError(err, "state_read")
		}
		if v.Ref, err = nullableID(ref); err != nil {
			return observability.DatabaseError(err, "state_read")
		}
		if due.Valid {
			v.DueAt = &due.Time
		}
		*out = append(*out, v)
	}
	return observability.DatabaseError(rows.Err(), "state_read")
}
func loadFollowUps(ctx context.Context, tx *sql.Tx, user c.ID, out *[]c.FollowUp) error {
	rows, err := tx.QueryContext(ctx, `SELECT CONVERT(varchar(36),id),reason,due_at,CONVERT(varchar(36),ref_id),status FROM dbo.ai_follow_ups WHERE user_id=@p1 AND status=N'scheduled' ORDER BY due_at DESC,id`, user)
	if err != nil {
		return observability.DatabaseError(err, "state_read")
	}
	defer rows.Close()
	for rows.Next() {
		var v c.FollowUp
		var ref sql.NullString
		if err = rows.Scan(&v.ID, &v.Reason, &v.DueAt, &ref, &v.Status); err != nil {
			return observability.DatabaseError(err, "state_read")
		}
		if v.Ref, err = nullableID(ref); err != nil {
			return observability.DatabaseError(err, "state_read")
		}
		v.UserID = user
		*out = append(*out, v)
	}
	return observability.DatabaseError(rows.Err(), "state_read")
}
func loadFormulation(ctx context.Context, tx *sql.Tx, user c.ID, focus *c.ID, out **c.FormulationBrief) error {
	if focus == nil {
		return nil
	}
	var v c.FormulationBrief
	var gaps string
	err := tx.QueryRowContext(ctx, `SELECT TOP(1) CONVERT(varchar(36),id),CONVERT(varchar(16),CONVERT(binary(8),search_version),2),CONVERT(varchar(36),thread_id),version,paragraph,gaps_json FROM dbo.ai_formulations WHERE user_id=@p1 AND thread_id=@p2 AND deleted_at IS NULL AND analysis_status=N'active' ORDER BY version DESC`, user, *focus).Scan(&v.ID, &v.SourceVersion, &v.ThreadID, &v.Version, &v.Paragraph, &gaps)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return observability.DatabaseError(err, "state_read")
	}
	if err = fromJSON(gaps, &v.Gaps); err != nil {
		return observability.DatabaseError(err, "state_read")
	}
	v.Epistemic = c.EpInferred
	*out = &v
	return nil
}

func loadFormulations(ctx context.Context, tx *sql.Tx, user c.ID, out *[]c.FormulationBrief) error {
	rows, err := tx.QueryContext(ctx, `SELECT TOP(5) CONVERT(varchar(36),f.id),CONVERT(varchar(16),CONVERT(binary(8),f.search_version),2),CONVERT(varchar(36),f.thread_id),f.version,f.paragraph,f.gaps_json FROM dbo.ai_formulations f JOIN dbo.ai_threads t ON t.id=f.thread_id AND t.user_id=f.user_id WHERE f.user_id=@p1 AND f.deleted_at IS NULL AND f.analysis_status=N'active' AND t.phase<>N'resolved' AND NOT EXISTS(SELECT 1 FROM dbo.ai_formulations newer WHERE newer.user_id=f.user_id AND newer.thread_id=f.thread_id AND (f.scope_json IS NULL OR newer.framework=f.framework) AND newer.deleted_at IS NULL AND newer.version>f.version) ORDER BY t.last_active_at DESC,f.thread_id`, user)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v c.FormulationBrief
		var gaps string
		if err = rows.Scan(&v.ID, &v.SourceVersion, &v.ThreadID, &v.Version, &v.Paragraph, &gaps); err != nil {
			return err
		}
		if err = fromJSON(gaps, &v.Gaps); err != nil {
			return err
		}
		v.Epistemic = c.EpInferred
		*out = append(*out, v)
	}
	return rows.Err()
}
