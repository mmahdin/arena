package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"time"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

var _ mem.Queue = (*MemoryRepository)(nil)

func (r *MemoryRepository) Claim(ctx context.Context, worker string, ttl time.Duration, ref c.BundleRef, quota int) (*mem.Lease, error) {
	return r.claimJobs(ctx, worker, ttl, ref, quota, false)
}
func (r *MemoryRepository) claimJobs(ctx context.Context, worker string, ttl time.Duration, ref c.BundleRef, quota int, reflection bool) (lease *mem.Lease, err error) {
	defer func() { err = memorySQLError(err) }()
	if worker == "" || len(worker) > 100 || ttl <= 0 || quota < 1 {
		return nil, mem.ErrDenied
	}
	// Claim only this worker's pinned bundle. Older queued payloads are retained
	// for a worker with their own immutable snapshot, never silently reinterpreted.
	// READPAST requires READ COMMITTED or REPEATABLE READ. Pooled SQL Server
	// connections can retain SERIALIZABLE from an earlier auth/state transaction,
	// so do not inherit the connection's previous isolation level here.
	tx, e := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if e != nil {
		return nil, sqlAt(e, "ai_jobs.claim_begin")
	}
	defer tx.Rollback()
	{
		var lock int
		e = tx.QueryRowContext(ctx, `DECLARE @result int; EXEC @result=sys.sp_getapplock @Resource=N'ai_background_claim',@LockMode=N'Exclusive',@LockOwner=N'Transaction',@LockTimeout=0; SELECT @result`).Scan(&lock)
		if e != nil {
			return nil, sqlAt(e, "ai_jobs.reflection_lock")
		}
		if lock < 0 {
			return nil, nil
		}
	}
	_, e = tx.ExecContext(ctx, `UPDATE dbo.ai_jobs SET status=N'failed',last_error_code=N'memory_attempts_exhausted' WHERE ((@p1=0 AND kind IN(N'memory_write',N'memory_embed',N'memory_feedback',N'memory_maintain')) OR (@p1=1 AND kind=N'reflection')) AND status=N'running' AND memory_lease_until<SYSUTCDATETIME() AND attempts>=max_attempts`, reflection)
	if e != nil {
		return nil, sqlAt(e, "ai_jobs.expire_exhausted")
	}
	l := &mem.Lease{Token: c.NewID()}
	var payload string
	var turn sql.NullString
	e = tx.QueryRowContext(ctx, `;WITH ready AS(SELECT TOP(1) j.* FROM dbo.ai_jobs j WITH(UPDLOCK,READPAST,ROWLOCK) WHERE EXISTS(SELECT 1 FROM dbo.users u WHERE u.id=j.user_id AND u.deleted_at IS NULL) AND ((@p7=0 AND j.kind IN(N'memory_write',N'memory_embed',N'memory_feedback',N'memory_maintain')) OR (@p7=1 AND j.kind=N'reflection')) AND (j.kind NOT IN(N'memory_write',N'reflection') OR NOT EXISTS(SELECT 1 FROM dbo.ai_user_state paused WHERE paused.user_id=j.user_id AND paused.memory_paused=1)) AND (j.kind NOT IN(N'memory_write',N'reflection') OR NOT EXISTS(SELECT 1 FROM dbo.ai_user_turn_leases active WHERE active.user_id=j.user_id AND active.lease_expires_at>SYSUTCDATETIME())) AND (j.kind NOT IN(N'memory_write',N'reflection') OR NOT EXISTS(SELECT 1 FROM dbo.ai_jobs other WHERE other.user_id=j.user_id AND other.id<>j.id AND other.kind IN(N'memory_write',N'reflection') AND other.status=N'running' AND other.memory_lease_until>SYSUTCDATETIME())) AND j.attempts<j.max_attempts AND j.run_after<=SYSUTCDATETIME() AND (j.status=N'queued' OR (j.status=N'running' AND j.memory_lease_until<SYSUTCDATETIME())) AND (j.kind<>N'memory_embed' OR JSON_VALUE(j.payload_json,'$.model') IS NULL OR (JSON_VALUE(j.payload_json,'$.model')=@p8 AND TRY_CONVERT(int,JSON_VALUE(j.payload_json,'$.dimension'))=@p9)) AND (j.kind=N'memory_embed' OR (JSON_VALUE(j.payload_json,'$.bundle.version')=@p4 AND JSON_QUERY(j.payload_json,'$.bundle.prompt_shas')=@p5)) AND (j.kind<>N'reflection' OR (SELECT COUNT(*) FROM dbo.ai_jobs q WHERE q.user_id=j.user_id AND q.kind=N'reflection' AND COALESCE(q.last_error_code,N'')<>N'reflection_ineligible' AND q.locked_at>=CONVERT(date,SYSUTCDATETIME()) AND q.id<>j.id)<@p6) AND (j.kind<>N'memory_write' OR (SELECT COALESCE(SUM(q.attempts),0) FROM dbo.ai_jobs q WHERE q.user_id=j.user_id AND q.kind=N'memory_write' AND q.locked_at>DATEADD(hour,-1,SYSUTCDATETIME()))<@p6) ORDER BY CASE WHEN j.kind=N'memory_embed' THEN 1 ELSE 0 END,CASE WHEN j.kind=N'memory_embed' AND JSON_VALUE(j.payload_json,'$.source_kind')=N'memory' THEN 0 ELSE 1 END,j.run_after,j.created_at,j.id)
 UPDATE ready SET status=N'running',attempts=attempts+1,locked_by=@p1,locked_at=SYSUTCDATETIME(),memory_lease_token=@p2,memory_lease_until=DATEADD(second,@p3,SYSUTCDATETIME()) OUTPUT CONVERT(varchar(36),inserted.id),CONVERT(varchar(36),inserted.user_id),CONVERT(varchar(36),inserted.turn_id),inserted.kind,inserted.attempts,inserted.payload_json,inserted.memory_result_enc`, worker, l.Token, int(ttl.Seconds()), ref.Version, memoryJSON(ref.PromptSHAs), quota, reflection, r.model, r.dimension).Scan(&l.ID, &l.UserID, &turn, &l.Kind, &l.Attempts, &payload, &l.Checkpoint)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, sqlAt(tx.Commit(), "ai_jobs.claim_commit")
	}
	if e != nil {
		return nil, sqlAt(e, "ai_jobs.claim_ready")
	}
	if turn.Valid {
		if e = l.TurnID.UnmarshalText([]byte(turn.String)); e != nil {
			return nil, e
		}
	}
	l.Payload = []byte(payload)
	if e = tx.Commit(); e != nil {
		return nil, sqlAt(e, "ai_jobs.claim_commit")
	}
	trace.RefreshJobs(ctx, r.db, l.TurnID)
	return l, nil
}

func completeMemoryJob(ctx context.Context, tx *sql.Tx, l mem.Lease) error {
	res, e := tx.ExecContext(ctx, `UPDATE dbo.ai_jobs SET status=N'completed' WHERE id=@p1 AND user_id=@p2 AND memory_lease_token=@p3 AND status=N'running' AND memory_lease_until>SYSUTCDATETIME()`, l.ID, l.UserID, l.Token)
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n == 1 {
		return nil
	}
	var status string
	e = tx.QueryRowContext(ctx, `SELECT status FROM dbo.ai_jobs WHERE id=@p1 AND user_id=@p2 AND memory_lease_token=@p3`, l.ID, l.UserID, l.Token).Scan(&status)
	if e == nil && status == "completed" {
		return nil
	}
	return mem.ErrConflict
}
func (r *MemoryRepository) Complete(ctx context.Context, l mem.Lease) (err error) {
	defer func() {
		if err == nil {
			trace.RefreshJobs(ctx, r.db, l.TurnID)
		}
	}()

	ctx, inspectionSpan := trace.Start(ctx, "adapters/mssql/memory_jobs.Complete", func() any { return map[string]any{"l": l} })
	defer func() { inspectionSpan.Complete(func() any { return map[string]any{} }, err, recover()) }()

	defer func() { err = memorySQLError(err) }()
	tx, e := r.begin(ctx, l.UserID, l.TurnID)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = completeMemoryJob(ctx, tx, l); e != nil {
		return e
	}
	return tx.Commit()
}
func (r *MemoryRepository) Checkpoint(ctx context.Context, l mem.Lease, encrypted []byte) (result0 error) {
	ctx, inspectionSpan := trace.Start(ctx, "adapters/mssql/memory_jobs.Checkpoint", func() any { return map[string]any{"l": l, "encrypted": encrypted} })
	defer func() { inspectionSpan.Complete(func() any { return map[string]any{} }, result0, recover()) }()

	return r.checkpoint(ctx, l, encrypted, false)
}
func (r *MemoryRepository) checkpoint(ctx context.Context, l mem.Lease, encrypted []byte, staged bool) (err error) {
	defer func() { err = memorySQLError(err) }()
	if len(encrypted) == 0 || len(encrypted) > 512<<10 {
		return mem.ErrDenied
	}
	tx, e := r.begin(ctx, l.UserID, l.TurnID)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(ctx, `UPDATE dbo.ai_jobs SET memory_result_enc=@p4 WHERE id=@p1 AND user_id=@p2 AND memory_lease_token=@p3 AND status=N'running' AND memory_lease_until>SYSUTCDATETIME() AND (memory_result_enc IS NULL OR (@p5=1 AND kind=N'reflection'))`, l.ID, l.UserID, l.Token, encrypted, staged)
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return mem.ErrConflict
	}
	return tx.Commit()
}
func (r *MemoryRepository) Fail(ctx context.Context, l mem.Lease, code string, delay time.Duration) (err error) {
	defer func() {
		if err == nil {
			trace.RefreshJobs(ctx, r.db, l.TurnID)
		}
	}()

	ctx, inspectionSpan := trace.Start(ctx, "adapters/mssql/memory_jobs.Fail", func() any { return map[string]any{"l": l, "code": code, "delay": delay} })
	defer func() { inspectionSpan.Complete(func() any { return map[string]any{} }, err, recover()) }()

	defer func() { err = memorySQLError(err) }()
	switch code {
	case "memory_retry", "memory_denied", "memory_cancelled", "memory_invalid", "reflection_ineligible":
	default:
		return mem.ErrDenied
	}
	if delay < 0 || delay > time.Hour {
		return mem.ErrDenied
	}
	tx, e := r.begin(ctx, l.UserID, l.TurnID)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(ctx, `UPDATE dbo.ai_jobs SET status=CASE WHEN attempts>=max_attempts OR @p4 IN(N'memory_denied',N'memory_invalid',N'reflection_ineligible') THEN N'failed' ELSE N'queued' END,last_error_code=@p4,run_after=DATEADD(second,@p5,SYSUTCDATETIME()) WHERE id=@p1 AND user_id=@p2 AND memory_lease_token=@p3 AND status=N'running' AND memory_lease_until>SYSUTCDATETIME()`, l.ID, l.UserID, l.Token, code, int(delay.Seconds()))
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return mem.ErrConflict
	}
	return tx.Commit()
}
func (r *MemoryRepository) Enqueue(ctx context.Context, user, turn c.ID, job c.AsyncJobProposal) (err error) {
	defer func() {
		if err == nil {
			trace.RefreshJobs(ctx, r.db, turn)
		}
	}()

	defer func() { err = memorySQLError(err) }()
	if job.DedupeKey == "" || len(job.DedupeKey) > 240 || len(job.Payload) > 128<<10 {
		return mem.ErrDenied
	}
	switch job.Kind {
	case "memory_write":
		var task mem.Task
		if json.Unmarshal(job.Payload, &task) != nil || (task.Schema != "memory_job.v1" && task.Schema != "intelligence_job.v2") || !c.ValidID(task.OperationID) || len(task.Data) == 0 {
			return mem.ErrDenied
		}
	case "reflection":
		var task mem.Task
		if json.Unmarshal(job.Payload, &task) != nil || task.Schema != "intelligence_job.v2" || !c.ValidID(task.OperationID) || len(task.Data) == 0 {
			return mem.ErrDenied
		}
	case "memory_maintain":
		var task mem.MaintenanceTask
		if json.Unmarshal(job.Payload, &task) != nil || task.Schema != "memory_maintain.v1" || task.At.IsZero() {
			return mem.ErrDenied
		}
	case "memory_feedback":
		var task mem.FeedbackTask
		if json.Unmarshal(job.Payload, &task) != nil || task.Schema != "memory_feedback.v1" || len(task.Feedback.Retrieved) > 7 || len(task.Feedback.Referenced) > 7 {
			return mem.ErrDenied
		}
	default:
		return mem.ErrDenied
	}
	tx, e := r.begin(ctx, user)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var status string
	if e = tx.QueryRowContext(ctx, `SELECT t.status FROM dbo.turns t JOIN dbo.conversations cv ON cv.id=t.conversation_id AND cv.user_id=t.user_id WHERE t.id=@p1 AND t.user_id=@p2`, turn, user).Scan(&status); e != nil || status != "completed" {
		return mem.ErrDenied
	}
	_, e = tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=@p2 AND dedupe_key=@p3) INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json) VALUES(@p4,@p1,@p5,@p2,@p3,@p6)`, user, job.Kind, job.DedupeKey, c.NewID(), turn, string(job.Payload))
	if e != nil {
		return e
	}
	return tx.Commit()
}
