package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
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
	if worker == "" || len(worker) > 100 || ttl < 30*time.Second || ttl > 10*time.Minute || quota < 1 {
		return nil, mem.ErrDenied
	}
	// Claim only this worker's pinned bundle. Older queued payloads are retained
	// for a worker with their own immutable snapshot, never silently reinterpreted.
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	if reflection {
		var lock int
		e = tx.QueryRowContext(ctx, `DECLARE @result int; EXEC @result=sys.sp_getapplock @Resource=N'ai_reflection_claim',@LockMode=N'Exclusive',@LockOwner=N'Transaction',@LockTimeout=0; SELECT @result`).Scan(&lock)
		if e != nil {
			return nil, e
		}
		if lock < 0 {
			return nil, nil
		}
	}
	_, e = tx.ExecContext(ctx, `UPDATE dbo.ai_jobs SET status=N'failed',last_error_code=N'memory_attempts_exhausted' WHERE ((@p1=0 AND kind IN(N'memory_write',N'memory_embed',N'memory_feedback',N'memory_maintain')) OR (@p1=1 AND kind=N'reflection')) AND status=N'running' AND memory_lease_until<SYSUTCDATETIME() AND attempts>=max_attempts`, reflection)
	if e != nil {
		return nil, e
	}
	l := &mem.Lease{Token: c.NewID()}
	var payload string
	var turn sql.NullString
	e = tx.QueryRowContext(ctx, `;WITH ready AS(SELECT TOP(1) j.* FROM dbo.ai_jobs j WITH(UPDLOCK,READPAST,ROWLOCK) WHERE ((@p7=0 AND j.kind IN(N'memory_write',N'memory_embed',N'memory_feedback',N'memory_maintain')) OR (@p7=1 AND j.kind=N'reflection')) AND j.attempts<j.max_attempts AND j.run_after<=SYSUTCDATETIME() AND (j.status=N'queued' OR (j.status=N'running' AND j.memory_lease_until<SYSUTCDATETIME())) AND JSON_VALUE(j.payload_json,'$.bundle.version')=@p4 AND JSON_QUERY(j.payload_json,'$.bundle.prompt_shas')=@p5 AND (j.kind<>N'reflection' OR (SELECT COUNT(*) FROM dbo.ai_jobs q WHERE q.user_id=j.user_id AND q.kind=N'reflection' AND COALESCE(q.last_error_code,N'')<>N'reflection_ineligible' AND q.locked_at>=CONVERT(date,SYSUTCDATETIME()) AND q.id<>j.id)<@p6) AND (j.kind<>N'memory_write' OR (SELECT COALESCE(SUM(q.attempts),0) FROM dbo.ai_jobs q WHERE q.user_id=j.user_id AND q.kind=N'memory_write' AND q.locked_at>DATEADD(hour,-1,SYSUTCDATETIME()))<@p6) ORDER BY j.run_after,j.created_at,j.id)
 UPDATE ready SET status=N'running',attempts=attempts+1,locked_by=@p1,locked_at=SYSUTCDATETIME(),memory_lease_token=@p2,memory_lease_until=DATEADD(second,@p3,SYSUTCDATETIME()) OUTPUT CONVERT(varchar(36),inserted.id),CONVERT(varchar(36),inserted.user_id),CONVERT(varchar(36),inserted.turn_id),inserted.kind,inserted.attempts,inserted.payload_json,inserted.memory_result_enc`, worker, l.Token, int(ttl.Seconds()), ref.Version, memoryJSON(ref.PromptSHAs), quota, reflection).Scan(&l.ID, &l.UserID, &turn, &l.Kind, &l.Attempts, &payload, &l.Checkpoint)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if e != nil {
		return nil, e
	}
	if turn.Valid {
		if e = l.TurnID.UnmarshalText([]byte(turn.String)); e != nil {
			return nil, e
		}
	}
	l.Payload = []byte(payload)
	if e = tx.Commit(); e != nil {
		return nil, e
	}
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
	defer func() { err = memorySQLError(err) }()
	tx, e := r.begin(ctx, l.UserID)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = completeMemoryJob(ctx, tx, l); e != nil {
		return e
	}
	return tx.Commit()
}
func (r *MemoryRepository) Checkpoint(ctx context.Context, l mem.Lease, encrypted []byte) error {
	return r.checkpoint(ctx, l, encrypted, false)
}
func (r *MemoryRepository) checkpoint(ctx context.Context, l mem.Lease, encrypted []byte, staged bool) (err error) {
	defer func() { err = memorySQLError(err) }()
	if len(encrypted) == 0 || len(encrypted) > 128<<10 {
		return mem.ErrDenied
	}
	tx, e := r.begin(ctx, l.UserID)
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
	defer func() { err = memorySQLError(err) }()
	switch code {
	case "memory_retry", "memory_denied", "memory_cancelled", "memory_invalid", "reflection_ineligible":
	default:
		return mem.ErrDenied
	}
	if delay < 0 || delay > time.Hour {
		return mem.ErrDenied
	}
	tx, e := r.begin(ctx, l.UserID)
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
	defer func() { err = memorySQLError(err) }()
	if job.DedupeKey == "" || len(job.DedupeKey) > 240 || len(job.Payload) > 128<<10 {
		return mem.ErrDenied
	}
	switch job.Kind {
	case "memory_write":
		var task mem.Task
		if json.Unmarshal(job.Payload, &task) != nil || task.Schema != "memory_job.v1" || !c.ValidID(task.OperationID) || len(task.Data) == 0 {
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

// Embeddings are published after the memory transaction, with fresh consent,
// visibility and content/version checks. Existing vectors are never overwritten.
func (r *MemoryRepository) PublishEmbedding(ctx context.Context, l mem.Lease) (err error) {
	defer func() { err = memorySQLError(err) }()
	var task memoryEmbeddingTask
	if json.Unmarshal(l.Payload, &task) != nil || task.Schema != "memory_embed.v1" || !c.ValidID(task.ItemID) {
		return mem.ErrDenied
	}
	plain, e := r.cipher.Open(ctx, l.UserID, "memory-embedding:"+task.ItemID.String(), task.Data)
	if e != nil {
		return mem.ErrDenied
	}
	var d memoryEmbeddingData
	if json.Unmarshal(plain, &d) != nil || d.Model != r.model || len(d.Vector) != r.dimension {
		return mem.ErrDenied
	}
	raw, e := EncodeEmbedding(r.model, d.Vector)
	if e != nil {
		return mem.ErrDenied
	}
	m := mem.Meta{UserID: l.UserID, TurnID: l.TurnID, OperationID: l.ID, Targets: []c.ID{task.ItemID}, Bundle: task.Bundle}
	s, e := r.Snapshot(ctx, l.UserID, m, nil)
	if e != nil {
		return e
	}
	var record *mem.Record
	for _, v := range s.Records {
		if v.Item.ID == task.ItemID {
			copy := v
			record = &copy
		}
	}
	if record == nil || s.User.MemoryPaused || mem.ClaimHash(record.Item.Content) != d.ClaimHash || !r.permit(ctx, s, m, mem.UseSource, "memory", task.ItemID) || !r.permit(ctx, s, m, mem.Extract, "memory", task.ItemID) || (record.Item.Sensitivity == c.SensHighly && !r.permit(ctx, s, m, mem.Sensitive, "memory", task.ItemID)) {
		return r.Complete(ctx, l)
	}
	tx, e := r.begin(ctx, l.UserID)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var revision, version int64
	if e = tx.QueryRowContext(ctx, `SELECT memory_revision,version FROM dbo.ai_user_state WHERE user_id=@p1`, l.UserID).Scan(&revision, &version); e != nil {
		return e
	}
	if revision != s.Revision || version != s.User.Version {
		return mem.ErrConflict
	}
	var actual int64
	if e = tx.QueryRowContext(ctx, `SELECT version FROM dbo.ai_memory_items WHERE id=@p1 AND user_id=@p2 AND deleted_at IS NULL`, task.ItemID, l.UserID).Scan(&actual); e != nil || actual != record.Version {
		return mem.ErrConflict
	}
	// Full bundle identity remains bound by the encrypted task's owner and queue.
	if m2, ok := mem.Metadata(ctx); ok && !reflect.DeepEqual(m2.Bundle, task.Bundle) {
		return mem.ErrDenied
	}
	_, e = tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_embeddings WHERE user_id=@p1 AND ref_kind=N'memory' AND ref_id=@p2 AND model_id=@p3) INSERT dbo.ai_embeddings(id,user_id,ref_kind,ref_id,model_id,dim,codec_version,vector_bytes) VALUES(@p4,@p1,N'memory',@p2,@p3,@p5,1,@p6)`, l.UserID, task.ItemID, r.model, c.NewID(), r.dimension, raw)
	if e != nil {
		return e
	}
	if e = completeMemoryJob(ctx, tx, l); e != nil {
		return e
	}
	return tx.Commit()
}
