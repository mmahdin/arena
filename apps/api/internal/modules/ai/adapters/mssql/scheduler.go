package mssql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/state"

	"time"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	rf "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/reflection"
	sch "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/scheduler"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func (r *ReflectionRepository) ApplyTimers(ctx context.Context, user c.ID, ops []c.FollowUpOp, now time.Time) (err error) {
	defer func() { err = memorySQLError(err) }()
	tx, e := r.begin(ctx, user)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var u c.UserState
	if e = loadUser(ctx, tx, user, &u); e != nil {
		return e
	}
	for _, op := range ops {
		if sch.Validate(op, now) != nil {
			return mem.ErrDenied
		}
		if op.Op == "cancel" {
			res, e := tx.ExecContext(ctx, `UPDATE dbo.ai_follow_ups SET status=N'cancelled',skip_reason=N'user_cancelled' WHERE id=@p1 AND user_id=@p2 AND status=N'scheduled'`, op.FollowUpID, user)
			if e != nil {
				return e
			}
			n, _ := res.RowsAffected()
			if n != 1 {
				return mem.ErrDenied
			}
			continue
		}
		if !sch.Authorized(ctx, r.MemoryRepository.policy, u, op.DueAt) {
			return mem.ErrDenied
		}
		var existing int
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_follow_ups WHERE user_id=@p1 AND id=@p2 AND reason=@p3 AND due_at=@p4 AND (ref_id=@p5 OR (ref_id IS NULL AND @p5 IS NULL))`, user, op.FollowUpID, op.Reason, op.DueAt.UTC(), op.Ref).Scan(&existing); e != nil {
			return e
		}
		if existing == 1 {
			continue
		}
		if op.Ref != nil {
			var owner int
			if e = tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM dbo.ai_threads WHERE id=@p1 AND user_id=@p2)+(SELECT COUNT(*) FROM dbo.ai_protocol_instances WHERE id=@p1 AND user_id=@p2)`, *op.Ref, user).Scan(&owner); e != nil {
				return e
			}
			if owner != 1 {
				return mem.ErrDenied
			}
		}
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_follow_ups WHERE user_id=@p1 AND status IN(N'scheduled',N'sent') AND (due_at BETWEEN DATEADD(second,-86399,@p2) AND DATEADD(second,86399,@p2) OR dispatched_at>DATEADD(day,-1,@p3))`, user, op.DueAt.UTC(), now).Scan(&existing); e != nil {
			return e
		}
		if existing != 0 {
			return mem.ErrDenied
		}
		_, e = tx.ExecContext(ctx, `INSERT dbo.ai_follow_ups(id,user_id,reason,due_at,ref_id) VALUES(@p1,@p2,@p3,@p4,@p5); INSERT dbo.ai_outbox(id,user_id,type,payload_json) VALUES(@p6,@p2,N'follow_up_scheduled',@p7)`, op.FollowUpID, user, op.Reason, op.DueAt.UTC(), op.Ref, c.NewID(), memoryJSON(struct {
			ID c.ID `json:"follow_up_id"`
		}{op.FollowUpID}))
		if e != nil {
			return e
		}
	}
	return tx.Commit()
}
func (r *ReflectionRepository) DueTimers(ctx context.Context, now time.Time, limit int) (out []c.FollowUp, err error) {
	defer func() { err = memorySQLError(err) }()
	rows, e := r.db.QueryContext(ctx, `SELECT TOP(@p1) CONVERT(varchar(36),id),CONVERT(varchar(36),user_id) FROM dbo.ai_follow_ups WHERE status=N'scheduled' AND due_at<=@p2 ORDER BY due_at,id`, limit, now)
	if e != nil {
		return nil, e
	}
	type pair struct{ id, user c.ID }
	pairs := []pair{}
	for rows.Next() {
		var p pair
		if e = rows.Scan(&p.id, &p.user); e != nil {
			rows.Close()
			return nil, e
		}
		pairs = append(pairs, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for _, p := range pairs {
		f, ok, e := r.dispatchTimer(ctx, p.user, p.id, now)
		if e != nil {
			return out, e
		}
		if ok {
			out = append(out, f)
		}
	}
	return out, nil
}
func (r *ReflectionRepository) dispatchTimer(ctx context.Context, user, id c.ID, now time.Time) (f c.FollowUp, ready bool, err error) {
	tx, e := r.begin(ctx, user)
	if e != nil {
		return f, false, e
	}
	defer tx.Rollback()
	f.ID = id
	f.UserID = user
	var ref sql.NullString
	e = tx.QueryRowContext(ctx, `SELECT reason,due_at,CONVERT(varchar(36),ref_id),status FROM dbo.ai_follow_ups WHERE id=@p1 AND user_id=@p2 AND status=N'scheduled' AND due_at<=@p3`, id, user, now).Scan(&f.Reason, &f.DueAt, &ref, &f.Status)
	if errors.Is(e, sql.ErrNoRows) {
		return f, false, nil
	}
	if e != nil {
		return f, false, e
	}
	f.Ref, e = nullableID(ref)
	if e != nil {
		return f, false, e
	}
	var u c.UserState
	if e = loadUser(ctx, tx, user, &u); e != nil {
		return f, false, e
	}
	// An explicit opt-out revokes this particular scheduled follow-up. It must
	// not spring back to life if the user opts in again later. Preserve the
	// record as skipped (never DELETE); all other temporary restrictions such
	// as quiet hours or provider-policy errors leave it pending for recheck.
	if !u.ProactiveOptIn {
		if _, e = tx.ExecContext(ctx, `UPDATE dbo.ai_follow_ups SET status=N'skipped',skip_reason=N'consent_revoked' WHERE id=@p1 AND user_id=@p2 AND status=N'scheduled'`, id, user); e != nil {
			return f, false, e
		}
		return f, false, tx.Commit()
	}
	if !sch.Authorized(ctx, r.MemoryRepository.policy, u, now) {
		return f, false, tx.Commit()
	}
	if f.Ref != nil {
		var n int
		if e = tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM dbo.ai_threads WHERE id=@p1 AND user_id=@p2 AND phase<>N'resolved')+(SELECT COUNT(*) FROM dbo.ai_protocol_instances WHERE id=@p1 AND user_id=@p2 AND status IN(N'active',N'paused',N'completed'))`, *f.Ref, user).Scan(&n); e != nil {
			return f, false, e
		}
		if n != 1 {
			// A stale or non-owned reference cannot be revived by a later
			// state change. Keep the cancelled authorization attempt for audit.
			if _, e = tx.ExecContext(ctx, `UPDATE dbo.ai_follow_ups SET status=N'skipped',skip_reason=N'reference_ineligible' WHERE id=@p1 AND user_id=@p2 AND status=N'scheduled'`, id, user); e != nil {
				return f, false, e
			}
			return f, false, tx.Commit()
		}
	}
	var recent int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_follow_ups WHERE user_id=@p1 AND dispatched_at>DATEADD(day,-1,@p2)`, user, now).Scan(&recent); e != nil {
		return f, false, e
	}
	if recent != 0 {
		return f, false, tx.Commit()
	}
	f.Status = "sent"
	_, e = tx.ExecContext(ctx, `UPDATE dbo.ai_follow_ups SET status=N'sent',dispatched_at=@p3 WHERE id=@p1 AND user_id=@p2; INSERT dbo.ai_outbox(id,user_id,type,payload_json) VALUES(@p4,@p2,N'follow_up_due',@p5)`, id, user, now, c.NewID(), memoryJSON(struct {
		Schema   string     `json:"schema"`
		FollowUp c.FollowUp `json:"follow_up"`
	}{"wakeup.v1", f}))
	if e != nil {
		return f, false, e
	}
	return f, true, tx.Commit()
}

// Sweep uses bounded batches and the same user lock as Gateway/finalization.
// Closing an internal session never closes or hides the user's conversation.
func (r *ReflectionRepository) Sweep(ctx context.Context, b c.BundleRef, now time.Time, limit int) (err error) {
	defer func() { err = memorySQLError(err) }()
	if limit < 1 || limit > 100 {
		return mem.ErrDenied
	}
	r.sweepMu.Lock()
	defer r.sweepMu.Unlock()
	rows, e := r.db.QueryContext(ctx, `SELECT TOP(@p1) CONVERT(varchar(36),u.user_id) FROM dbo.ai_user_state u WHERE (@p2 IS NULL OR CONVERT(varchar(36),u.user_id)>@p2) ORDER BY CONVERT(varchar(36),u.user_id)`, limit, memoryNullableID(r.sweepCursor))
	if e != nil {
		return e
	}
	users := []c.ID{}
	for rows.Next() {
		var user c.ID
		if e = rows.Scan(&user); e != nil {
			rows.Close()
			return e
		}
		users = append(users, user)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if len(users) == 0 {
		r.sweepCursor = c.ID{}
		return nil
	}
	for _, user := range users {
		if e = r.sweepUser(ctx, user, b, now); e != nil {
			return e
		}
		r.sweepCursor = user
	}
	return nil
}
func (r *ReflectionRepository) sweepUser(ctx context.Context, user c.ID, b c.BundleRef, now time.Time) error {
	tx, e := r.begin(ctx, user)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var u c.UserState
	if e = loadUser(ctx, tx, user, &u); e != nil {
		return e
	}
	// Never race a running turn, including an expired lease awaiting recovery.
	var busy int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.turns WHERE user_id=@p1 AND lifecycle_version=1 AND status IN(N'pending',N'running')`, user).Scan(&busy); e != nil {
		return e
	}
	if busy != 0 {
		return tx.Commit()
	}
	rows, e := tx.QueryContext(ctx, `UPDATE dbo.ai_sessions SET closed_at=@p2,close_reason=N'idle',version=version+1 OUTPUT CONVERT(varchar(36),inserted.id),CONVERT(varchar(36),inserted.focus_thread_id) WHERE user_id=@p1 AND closed_at IS NULL AND last_activity_at<=DATEADD(minute,-@p3,@p2)`, user, now, r.policy.Thresholds.Session.IdleTimeoutMin)
	if e != nil {
		return e
	}
	tasks := []rf.Task{}
	for rows.Next() {
		var session c.ID
		var thread sql.NullString
		if e = rows.Scan(&session, &thread); e != nil {
			rows.Close()
			return e
		}
		if thread.Valid {
			id, e := nullableID(thread)
			if e != nil {
				rows.Close()
				return e
			}
			tasks = append(tasks, rf.Task{Schema: "reflection_job.v1", Bundle: b, UserID: user, ThreadID: *id, SessionID: session, Trigger: "session_closed"})
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, task := range tasks {
		key := task.ThreadID.String() + ":" + task.SessionID.String() + ":" + task.Trigger
		_, e = tx.ExecContext(ctx, `INSERT dbo.ai_turn_events(id,user_id,type,actor,delta_json) VALUES(@p1,@p2,N'session_closed',N'scheduler',@p3); IF NOT EXISTS(SELECT 1 FROM dbo.ai_jobs WHERE user_id=@p2 AND kind=N'reflection' AND dedupe_key=@p4) INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json) VALUES(@p5,@p2,N'reflection',@p4,@p6)`, c.NewID(), user, memoryJSON(struct {
			Session c.ID `json:"session_id"`
		}{task.SessionID}), key, c.NewID(), memoryJSON(task))
		if e != nil {
			return e
		}
	}

	// Recover already-closed sessions as well as newly closed ones (for example
	// after a crash between Gateway admission and the scheduler's next sweep).
	closed, e := tx.QueryContext(ctx, `SELECT TOP(100) CONVERT(varchar(36),s.id),CONVERT(varchar(36),s.focus_thread_id) FROM dbo.ai_sessions s WHERE s.user_id=@p1 AND s.closed_at IS NOT NULL AND s.focus_thread_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM dbo.ai_jobs j WHERE j.user_id=s.user_id AND j.kind=N'reflection' AND j.dedupe_key=CONVERT(varchar(36),s.focus_thread_id)+N':'+CONVERT(varchar(36),s.id)+N':session_closed') ORDER BY s.closed_at,s.id`, user)
	if e != nil {
		return e
	}
	pending := []rf.Task{}
	for closed.Next() {
		var sid, tid c.ID
		if e = closed.Scan(&sid, &tid); e != nil {
			closed.Close()
			return e
		}
		pending = append(pending, rf.Task{Schema: "reflection_job.v1", Bundle: b, UserID: user, ThreadID: tid, SessionID: sid, Trigger: "session_closed"})
	}
	e = closed.Err()
	closed.Close()
	if e != nil {
		return e
	}
	for _, task := range pending {
		key := task.ThreadID.String() + ":" + task.SessionID.String() + ":" + task.Trigger
		if _, e = tx.ExecContext(ctx, `INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json) VALUES(@p1,@p2,N'reflection',@p3,@p4)`, c.NewID(), user, key, memoryJSON(task)); e != nil {
			return e
		}
	}
	// Growth/contradiction/nightly triggers use actual linked memory since the
	// previous completed reflection. Their payload contains identifiers only.
	scan, e := tx.QueryContext(ctx, `SELECT TOP(100) CONVERT(varchar(36),t.id),t.version,CONVERT(varchar(36),s.id),COALESCE(m.new_items,0),COALESCE(m.contradictions,0),last_run.at FROM dbo.ai_threads t OUTER APPLY(SELECT MAX(created_at) at FROM dbo.ai_reflection_runs WHERE user_id=t.user_id AND thread_id=t.id) last_run CROSS APPLY(SELECT TOP(1) id FROM dbo.ai_sessions WHERE user_id=t.user_id AND focus_thread_id=t.id ORDER BY last_activity_at DESC,id) s OUTER APPLY(SELECT COUNT(*) new_items,SUM(i.contradict_n) contradictions FROM dbo.ai_memory_items i WHERE i.user_id=t.user_id AND i.deleted_at IS NULL AND i.updated_at>COALESCE(last_run.at,CONVERT(datetime2(3),'19000101')) AND EXISTS(SELECT 1 FROM OPENJSON(i.thread_ids_json) ids WHERE TRY_CONVERT(uniqueidentifier,ids.value)=t.id)) m WHERE t.user_id=@p1 ORDER BY t.id`, user)
	if e != nil {
		return e
	}
	type triggerRow struct {
		tid, sid              c.ID
		version               int64
		items, contradictions int
		last                  sql.NullTime
	}
	triggers := []triggerRow{}
	for scan.Next() {
		var row triggerRow
		if e = scan.Scan(&row.tid, &row.version, &row.sid, &row.items, &row.contradictions, &row.last); e != nil {
			scan.Close()
			return e
		}
		triggers = append(triggers, row)
	}
	e = scan.Err()
	scan.Close()
	if e != nil {
		return e
	}
	for _, row := range triggers {
		var turns int
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(DISTINCT turn_id) FROM dbo.ai_turn_events WHERE user_id=@p1 AND type=N'state_applied' AND created_at>COALESCE(@p3,CONVERT(datetime2(3),'19000101')) AND (TRY_CONVERT(uniqueidentifier,JSON_VALUE(delta_json,'$.thread.thread_id'))=@p2 OR TRY_CONVERT(uniqueidentifier,JSON_VALUE(delta_json,'$.session.focus_thread'))=@p2)`, user, row.tid, row.last).Scan(&turns); e != nil {
			return e
		}
		trigger := ""
		if row.contradictions >= r.policy.Thresholds.Reflection.Triggers.ContradictionEventsGTE {
			trigger = "contradictions"
		} else if row.items >= r.policy.Thresholds.Reflection.Triggers.MinNewItems || turns >= r.policy.Thresholds.Reflection.Triggers.ThreadTurnsGTE {
			trigger = "thread_growth"
		} else if !row.last.Valid || row.last.Time.Before(now.Add(-24*time.Hour)) {
			trigger = "nightly"
		}
		if trigger == "" {
			continue
		}
		task := rf.Task{Schema: "reflection_job.v1", Bundle: b, UserID: user, ThreadID: row.tid, SessionID: row.sid, Trigger: trigger}
		key := row.tid.String() + ":" + row.sid.String() + ":" + trigger + ":" + fmt.Sprint(row.version)
		if trigger == "nightly" {
			key += ":" + now.Format("2006-01-02")
		}
		if _, e = tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'reflection' AND dedupe_key=@p2) INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json) VALUES(@p3,@p1,N'reflection',@p2,@p4)`, user, key, c.NewID(), memoryJSON(task)); e != nil {
			return e
		}
	}
	// The Protocol Engine determines follow_up_due_at from reviewed definitions.
	// No time or clinical instruction is derived from a model's free-text summary.
	proto, e := tx.QueryContext(ctx, `SELECT TOP(100) CONVERT(varchar(36),id),follow_up_due_at FROM dbo.ai_protocol_instances WHERE user_id=@p1 AND follow_up_due_at IS NOT NULL AND status IN(N'paused',N'completed')`, user)
	if e != nil {
		return e
	}
	type timerRow struct {
		id c.ID
		at time.Time
	}
	timers := []timerRow{}
	for proto.Next() {
		var p timerRow
		if e = proto.Scan(&p.id, &p.at); e != nil {
			proto.Close()
			return e
		}
		timers = append(timers, p)
	}
	e = proto.Err()
	proto.Close()
	if e != nil {
		return e
	}
	for _, p := range timers {
		if !sch.Authorized(ctx, r.MemoryRepository.policy, u, p.at) {
			continue
		}
		id := uuid.NewSHA1(p.id, []byte("protocol_review:"+p.at.UTC().Format(time.RFC3339Nano)))
		_, e = tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_follow_ups WHERE id=@p1 AND user_id=@p2) AND NOT EXISTS(SELECT 1 FROM dbo.ai_follow_ups WHERE user_id=@p2 AND status IN(N'scheduled',N'sent') AND due_at BETWEEN DATEADD(second,-86399,@p3) AND DATEADD(second,86399,@p3)) INSERT dbo.ai_follow_ups(id,user_id,reason,due_at,ref_id) VALUES(@p1,@p2,N'protocol_review',@p3,@p4)`, id, user, p.at, p.id)
		if e != nil {
			return e
		}
	}

	// State Committer gates own thread dormancy. Any current session focus is
	// protected; historical derivatives and evidence links remain untouched.
	var threads []c.ThreadRecord
	if e = loadThreads(ctx, tx, user, &threads); e != nil {
		return e
	}
	snap := contextstate.Snapshot{User: u, Threads: threads}
	proposals := []c.ThreadProposal{}
	for _, thread := range threads {
		if thread.Phase == c.PhaseDormant || thread.Phase == c.PhaseResolved || thread.LastActiveAt.After(now.Add(-time.Duration(r.policy.Thresholds.Thread.DormantAfterDays)*24*time.Hour)) {
			continue
		}
		var focusCount int
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_sessions WHERE user_id=@p1 AND closed_at IS NULL AND focus_thread_id=@p2`, user, thread.ID).Scan(&focusCount); e != nil {
			return e
		}
		if focusCount == 0 {
			proposals = append(proposals, c.ThreadProposal{ThreadID: thread.ID, Op: "dormant", Reason: "inactivity"})
		}
	}
	delta, rejected, e := state.ValidateReflection(ctx, snap, r.policy, proposals)
	if e != nil {
		return e
	}
	if e = applyThread(ctx, tx, c.Admission{UserID: user}, delta); e != nil {
		return e
	}
	if len(delta.ThreadOps) > 0 || len(rejected) > 0 {
		if _, e = tx.ExecContext(ctx, `INSERT dbo.ai_turn_events(id,user_id,type,actor,delta_json) VALUES(@p1,@p2,N'thread_maintenance',N'scheduler',@p3)`, c.NewID(), user, memoryJSON(delta)); e != nil {
			return e
		}
	}
	// Deterministic ST-023 no-response expiry. Retain a history entry and event
	// so future eligibility observes the decline cooldown after restart.
	expired, e := tx.QueryContext(ctx, `DECLARE @expired TABLE(protocol_id nvarchar(100)); UPDATE dbo.ai_protocol_instances SET status=N'declined',outcome=N'no_response',completed_at=@p2,version=version+1 OUTPUT inserted.protocol_id INTO @expired WHERE user_id=@p1 AND status=N'offered' AND offered_at<=DATEADD(hour,-48,@p2); SELECT protocol_id FROM @expired`, user, now)
	if e != nil {
		return e
	}
	history := append([]c.ProtocolHistoryBrief{}, u.ProtocolHistory...)
	changed := false
	for expired.Next() {
		var id string
		if e = expired.Scan(&id); e != nil {
			expired.Close()
			return e
		}
		history = append(history, c.ProtocolHistoryBrief{ProtocolID: id, Outcome: "declined", At: now})
		changed = true
	}
	e = expired.Err()
	expired.Close()
	if e != nil {
		return e
	}
	if changed {
		if len(history) > 100 {
			history = history[len(history)-100:]
		}
		if _, e = tx.ExecContext(ctx, `UPDATE dbo.ai_user_state SET protocol_history_json=@p2,version=version+1 WHERE user_id=@p1; INSERT dbo.ai_turn_events(id,user_id,type,actor,delta_json) VALUES(@p3,@p1,N'protocol_status_changed',N'scheduler',N'{"reason":"no_response"}')`, user, arrayJSON(history), c.NewID()); e != nil {
			return e
		}
	}
	return tx.Commit()
}
