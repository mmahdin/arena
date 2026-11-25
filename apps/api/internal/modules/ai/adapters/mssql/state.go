package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"slices"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func applySession(ctx context.Context, tx *sql.Tx, a c.Admission, d c.StateDelta) error {
	p := d.Session
	if p.TurnCountInc < 0 || p.ConsecutiveQuestions < 0 || (p.Mode != nil && !p.Mode.Valid()) || !p.LastObjective.Valid() {
		return invalid()
	}
	var summary, emotions string
	var pending sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT summary,emotion_trajectory_json,pending_question_json FROM dbo.ai_sessions WHERE id=@p1 AND user_id=@p2 AND conversation_id=@p3 AND version=@p4 AND closed_at IS NULL`, a.SessionID, a.UserID, a.ConversationID, d.ExpectedVersions.Session).Scan(&summary, &emotions, &pending)
	if errors.Is(err, sql.ErrNoRows) {
		return conflict()
	}
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	if p.SummaryReplace != nil {
		summary = *p.SummaryReplace
	}
	if p.SummaryDelta != nil {
		if summary != "" {
			summary += "\n"
		}
		summary += *p.SummaryDelta
	}
	var labels []string
	if err = json.Unmarshal([]byte(emotions), &labels); err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	labels = append(labels, p.EmotionLabels...)
	if len(labels) > 5 {
		labels = labels[len(labels)-5:]
	}
	emotions, err = encodeArray(labels)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	if p.ClearPending {
		pending = sql.NullString{}
	}
	if p.PendingQuestion != nil {
		v, e := encode(p.PendingQuestion)
		if e != nil {
			return observability.DatabaseError(e, "state_write")
		}
		pending = sql.NullString{String: v, Valid: true}
	}
	return changed(tx.ExecContext(ctx, `UPDATE dbo.ai_sessions SET mode=COALESCE(@p5,mode),focus_thread_id=CASE WHEN @p13=1 THEN NULL ELSE COALESCE(@p6,focus_thread_id) END,last_objective=@p7,pending_question_json=@p8,consecutive_questions=@p9,summary=@p10,turn_count=turn_count+@p11,emotion_trajectory_json=@p12,last_activity_at=SYSUTCDATETIME(),version=version+1 WHERE id=@p1 AND user_id=@p2 AND conversation_id=@p3 AND version=@p4`, a.SessionID, a.UserID, a.ConversationID, d.ExpectedVersions.Session, p.Mode, p.FocusThread, string(p.LastObjective), pending, p.ConsecutiveQuestions, summary, p.TurnCountInc, emotions, p.ClearFocus))
}
func applyThread(ctx context.Context, tx *sql.Tx, a c.Admission, d c.StateDelta) error {
	p := d.Thread
	if p != nil {
		if !c.ValidID(p.ThreadID) {
			return invalid()
		}
		expected := d.ExpectedVersions.Thread
		if p.Create != nil {
			if expected != 0 {
				return conflict()
			}
			tags, err := encodeArray(p.Create.ProblemTags)
			if err != nil {
				return observability.DatabaseError(err, "state_write")
			}
			_, err = tx.ExecContext(ctx, `INSERT dbo.ai_threads(id,user_id,topic,goal,problem_tags_json) VALUES(@p1,@p2,@p3,@p4,@p5)`, p.ThreadID, a.UserID, p.Create.Topic, p.Create.Goal, tags)
			if err != nil {
				return observability.DatabaseError(err, "state_write")
			}
			expected = 1
		}
		var raw string
		err := tx.QueryRowContext(ctx, `SELECT slots_json FROM dbo.ai_threads WHERE id=@p1 AND user_id=@p2 AND version=@p3`, p.ThreadID, a.UserID, expected).Scan(&raw)
		if errors.Is(err, sql.ErrNoRows) {
			return conflict()
		}
		if err != nil {
			return observability.DatabaseError(err, "state_write")
		}
		slots := map[string]c.SlotView{}
		if err = json.Unmarshal([]byte(raw), &slots); err != nil {
			return observability.DatabaseError(err, "state_write")
		}
		for _, slot := range p.Slots {
			if slot.Slot == "" || len(slot.Evidence) == 0 || (slot.Status != c.EpReported && slot.Status != c.EpInferred) {
				return invalid()
			}
			for _, id := range slot.Evidence {
				var n int
				err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id WHERE m.id=@p1 AND v.user_id=@p2 AND m.role=N'user'`, id, a.UserID).Scan(&n)
				if err != nil {
					return observability.DatabaseError(err, "state_write")
				}
				if n != 1 {
					return invalid()
				}
			}
			slots[slot.Slot] = c.SlotView{Slot: slot.Slot, Status: slot.Status, Value: slot.Value, Evidence: slot.Evidence, Declined: slot.Declined}
		}
		if p.Phase != nil && !p.Phase.Valid() {
			return invalid()
		}
		for _, id := range p.RelatedThreadIDs {
			var n int
			if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_threads WHERE id=@p1 AND user_id=@p2`, id, a.UserID).Scan(&n); err != nil {
				return observability.DatabaseError(err, "state_write")
			}
			if n != 1 {
				return invalid()
			}
		}
		raw, err = encode(slots)
		if err != nil {
			return observability.DatabaseError(err, "state_write")
		}
		related, err := encodeArray(p.RelatedThreadIDs)
		if err != nil {
			return observability.DatabaseError(err, "state_write")
		}
		if err = changed(tx.ExecContext(ctx, `UPDATE dbo.ai_threads SET slots_json=@p4,prev_phase=CASE WHEN @p5 IS NOT NULL AND phase<>@p5 THEN phase ELSE prev_phase END,phase=COALESCE(@p5,phase),goal=COALESCE(@p8,goal),related_thread_ids_json=@p6,last_active_at=CASE WHEN @p7=1 THEN SYSUTCDATETIME() ELSE last_active_at END,version=version+1 WHERE id=@p1 AND user_id=@p2 AND version=@p3`, p.ThreadID, a.UserID, expected, raw, p.Phase, related, p.Touch, p.Goal)); err != nil {
			return observability.DatabaseError(err, "state_write")
		}
	}
	// Secondary-thread changes are serialized by the user lease and guarded by
	// the user version checked in applyUser. Psychological transition validation
	// is deliberately the WP-08 StateCommitter's responsibility.
	for _, op := range d.ThreadOps {
		var phase string
		switch op.Op {
		case "pause":
			phase = "paused"
		case "dormant":
			phase = "dormant"
		case "resolve":
			phase = "resolved"
		case "resume":
			phase = ""
		default:
			return invalid()
		}
		if err := changed(tx.ExecContext(ctx, `UPDATE dbo.ai_threads SET phase=CASE WHEN @p3=N'' THEN COALESCE(prev_phase,N'exploring') ELSE @p3 END,prev_phase=phase,version=version+1 WHERE id=@p1 AND user_id=@p2`, op.ThreadID, a.UserID, phase)); err != nil {
			return observability.DatabaseError(err, "state_write")
		}
	}
	return nil
}
func applyProtocol(ctx context.Context, tx *sql.Tx, a c.Admission, d c.StateDelta) error {
	p := d.Protocol
	if p == nil {
		return nil
	}
	if !c.ValidID(p.InstanceID) || p.AttemptsInc < 0 || p.StuckInc < 0 || (p.NewStatus != nil && !p.NewStatus.Valid()) {
		return invalid()
	}
	expected := d.ExpectedVersions.Protocol
	if p.Create != nil {
		if expected != 0 {
			return conflict()
		}
		_, err := tx.ExecContext(ctx, `INSERT dbo.ai_protocol_instances(id,user_id,thread_id,protocol_id,protocol_version,status,offered_at) VALUES(@p1,@p2,@p3,@p4,@p5,N'offered',@p6)`, p.InstanceID, a.UserID, p.Create.ThreadID, p.Create.ProtocolID, p.Create.Version, p.Create.OfferedAt)
		if err != nil {
			return observability.DatabaseError(err, "state_write")
		}
		expected = 1
	}
	var raw string
	err := tx.QueryRowContext(ctx, `SELECT inputs_json FROM dbo.ai_protocol_instances WHERE id=@p1 AND user_id=@p2 AND version=@p3`, p.InstanceID, a.UserID, expected).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return conflict()
	}
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	inputs := map[string]string{}
	if err = json.Unmarshal([]byte(raw), &inputs); err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	for k, v := range p.Inputs {
		inputs[k] = v
	}
	raw, err = encode(inputs)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	return changed(tx.ExecContext(ctx, `UPDATE dbo.ai_protocol_instances SET status=COALESCE(@p4,status),step_id=COALESCE(@p5,step_id),inputs_json=@p6,attempts=attempts+@p7,stuck_count=CASE WHEN @p8=1 THEN @p9 ELSE stuck_count+@p9 END,outcome=COALESCE(@p10,outcome),consent_at=COALESCE(@p11,consent_at),consent_message_id=COALESCE(@p12,consent_message_id),follow_up_due_at=COALESCE(@p13,follow_up_due_at),completed_at=CASE WHEN @p4=N'completed' THEN SYSUTCDATETIME() ELSE completed_at END,version=version+1 WHERE id=@p1 AND user_id=@p2 AND version=@p3`, p.InstanceID, a.UserID, expected, p.NewStatus, p.NewStep, raw, p.AttemptsInc, p.StuckReset, p.StuckInc, p.Outcome, p.ConsentAt, p.ConsentMessageID, p.FollowUpAt))
}
func applyUser(ctx context.Context, tx *sql.Tx, a c.Admission, d c.StateDelta) error {
	var prefsRaw, engRaw, histRaw, riskRaw string
	err := tx.QueryRowContext(ctx, `SELECT prefs_json,engagement_json,protocol_history_json,risk_history_json FROM dbo.ai_user_state WHERE user_id=@p1 AND version=@p2`, a.UserID, d.ExpectedVersions.User).Scan(&prefsRaw, &engRaw, &histRaw, &riskRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return conflict()
	}
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	var prefs c.PrefsView
	eng := map[string]float64{}
	hist := []c.ProtocolHistoryBrief{}
	risk := []c.RiskTransition{}
	for _, v := range []struct {
		raw string
		to  any
	}{{prefsRaw, &prefs}, {engRaw, &eng}, {histRaw, &hist}, {riskRaw, &risk}} {
		if err = json.Unmarshal([]byte(v.raw), v.to); err != nil {
			return observability.DatabaseError(err, "state_write")
		}
	}
	p := d.User
	if p.PrefsPatch != nil {
		v := p.PrefsPatch
		if v.Length != nil {
			prefs.Length = *v.Length
		}
		if v.Tone != nil {
			prefs.Tone = *v.Tone
		}
		if v.AddressForm != nil {
			prefs.AddressForm = *v.AddressForm
		}
		for _, x := range v.AvoidRemove {
			prefs.Avoid = slices.DeleteFunc(prefs.Avoid, func(s string) bool { return s == x })
		}
		for _, x := range v.AvoidAdd {
			if !slices.Contains(prefs.Avoid, x) {
				prefs.Avoid = append(prefs.Avoid, x)
			}
		}
	}
	for k, v := range p.EngagementDelta {
		eng[k] += v
	}
	if p.ProtocolHistoryAdd != nil {
		hist = append(hist, *p.ProtocolHistoryAdd)
	}
	followUpUntil, err := riskFollowupForCommit(d)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	var level any
	var since any
	if p.Risk != nil {
		if !p.Risk.To.Valid() || p.Risk.TurnID != a.TurnID {
			return invalid()
		}
		risk = append(risk, *p.Risk)
		if len(risk) > 20 {
			risk = risk[len(risk)-20:]
		}
		level = string(p.Risk.To)
		since = p.Risk.At
	}
	prefsRaw, err = encode(prefs)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	engRaw, err = encode(eng)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	histRaw, err = encode(hist)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	riskRaw, err = encode(risk)
	if err != nil {
		return observability.DatabaseError(err, "state_write")
	}
	return changed(tx.ExecContext(ctx, `UPDATE dbo.ai_user_state SET prefs_json=@p3,engagement_json=@p4,protocol_history_json=@p5,risk_history_json=@p6,risk_level=COALESCE(@p7,risk_level),risk_since=COALESCE(@p8,risk_since),risk_follow_up_until=CASE WHEN @p10 IS NULL THEN risk_follow_up_until WHEN risk_follow_up_until IS NULL OR risk_follow_up_until<@p10 THEN @p10 ELSE risk_follow_up_until END,dependency_watch=COALESCE(@p9,dependency_watch),version=version+1 WHERE user_id=@p1 AND version=@p2`, a.UserID, d.ExpectedVersions.User, prefsRaw, engRaw, histRaw, riskRaw, level, since, p.DependencyWatch, followUpUntil))
}
func applyOperations(ctx context.Context, tx *sql.Tx, a c.Admission, d c.StateDelta) error {
	for _, op := range d.OpenLoops {
		switch op.Op {
		case "add":
			_, err := tx.ExecContext(ctx, `INSERT dbo.ai_open_loops(id,user_id,kind,text,ref_id,due_at) VALUES(@p1,@p2,@p3,@p4,@p5,@p6)`, op.Loop.ID, a.UserID, op.Loop.Kind, op.Loop.Text, op.Loop.Ref, op.Loop.DueAt)
			if err != nil {
				return observability.DatabaseError(err, "state_write")
			}
		case "close", "expire":
			status := "closed"
			if op.Op == "expire" {
				status = "expired"
			}
			if err := changed(tx.ExecContext(ctx, `UPDATE dbo.ai_open_loops SET status=@p3 WHERE id=@p1 AND user_id=@p2`, op.Loop.ID, a.UserID, status)); err != nil {
				return observability.DatabaseError(err, "state_write")
			}
		default:
			return invalid()
		}
	}
	for _, op := range d.FollowUps {
		switch op.Op {
		case "schedule":
			_, err := tx.ExecContext(ctx, `INSERT dbo.ai_follow_ups(id,user_id,reason,due_at,ref_id,status,skip_reason) SELECT @p1,@p2,@p3,@p4,@p5,CASE WHEN proactive_opt_in=1 THEN N'scheduled' ELSE N'skipped' END,CASE WHEN proactive_opt_in=1 THEN NULL ELSE N'not_opted_in' END FROM dbo.ai_user_state WHERE user_id=@p2`, op.FollowUpID, a.UserID, op.Reason, op.DueAt, op.Ref)
			if err != nil {
				return observability.DatabaseError(err, "state_write")
			}
		case "cancel":
			if err := changed(tx.ExecContext(ctx, `UPDATE dbo.ai_follow_ups SET status=N'cancelled' WHERE id=@p1 AND user_id=@p2`, op.FollowUpID, a.UserID)); err != nil {
				return observability.DatabaseError(err, "state_write")
			}
		default:
			return invalid()
		}
	}
	return nil
}

func encodeArray[T any](v []T) (string, error) {
	if v == nil {
		v = []T{}
	}
	return encode(v)
}
