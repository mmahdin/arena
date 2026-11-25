package mssql

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	bg "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/background"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"time"
)

type IntelligenceRepository struct{ *MemoryRepository }

func NewIntelligenceRepository(r *MemoryRepository) *IntelligenceRepository {
	return &IntelligenceRepository{r}
}
func (r *IntelligenceRepository) Candidates(ctx context.Context, cfg bg.Config) (out []bg.Batch, err error) {
	// Fair scanning: denied/paused owners do not monopolize the first page. There
	// is no recency cutoff: a quiet user's last partial batch eventually flushes.
	rows, e := r.db.QueryContext(ctx, `SELECT TOP(@p1) CONVERT(varchar(36),s.user_id) FROM dbo.ai_user_state s JOIN dbo.users u ON u.id=s.user_id AND u.deleted_at IS NULL WHERE s.memory_paused=0 AND EXISTS(SELECT 1 FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id AND v.user_id=s.user_id JOIN dbo.turns t ON t.id=m.turn_id AND t.user_id=s.user_id LEFT JOIN dbo.ai_memory_review_messages q ON q.user_id=s.user_id AND q.message_id=m.id WHERE m.role=N'user' AND m.deleted_at IS NULL AND v.deleted_at IS NULL AND t.status=N'completed' AND (q.message_id IS NULL OR (q.reviewed_at IS NOT NULL AND q.source_version<>CONVERT(binary(8),m.search_version)))) ORDER BY s.memory_review_checked_at,s.user_id`, cfg.MaxUsers)
	if e != nil {
		return nil, e
	}
	users := []c.ID{}
	for rows.Next() {
		var id c.ID
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		users = append(users, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for _, user := range users {
		if _, e = r.db.ExecContext(ctx, `UPDATE dbo.ai_user_state SET memory_review_checked_at=SYSUTCDATETIME() WHERE user_id=@p1`, user); e != nil {
			return nil, e
		}
		rows, e = r.db.QueryContext(ctx, `SELECT TOP(@p2) CONVERT(varchar(36),m.id),CONVERT(varchar(36),t.id),m.created_at,DATALENGTH(m.content),CONVERT(varchar(16),CONVERT(binary(8),m.search_version),2),t.lifecycle_version FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id AND v.user_id=@p1 JOIN dbo.turns t ON t.id=m.turn_id AND t.user_id=@p1 LEFT JOIN dbo.ai_memory_review_messages q ON q.user_id=@p1 AND q.message_id=m.id WHERE m.role=N'user' AND m.deleted_at IS NULL AND v.deleted_at IS NULL AND t.status=N'completed' AND (q.message_id IS NULL OR (q.reviewed_at IS NOT NULL AND q.source_version<>CONVERT(binary(8),m.search_version))) ORDER BY t.admission_order,m.message_order`, user, cfg.BatchSize)
		if e != nil {
			return nil, e
		}
		batch := bg.Batch{UserID: user}
		var first time.Time
		bytes := 0
		for rows.Next() {
			var message, turn c.ID
			var at time.Time
			var size int
			var revision string
			var lifecycle int
			if e = rows.Scan(&message, &turn, &at, &size, &revision, &lifecycle); e != nil {
				break
			}
			if len(batch.MessageIDs) > 0 && bytes+size > cfg.InputTokens*4 {
				break
			}
			if first.IsZero() {
				first = at
			}
			bytes += size
			batch.MessageIDs = append(batch.MessageIDs, message)
			batch.Versions = append(batch.Versions, revision)
			batch.TurnID = turn
			if lifecycle != 1 {
				batch.TurnID = c.ID{}
			}
			batch.Trigger = message
		}
		rows.Close()
		if e != nil {
			return nil, e
		}
		if len(batch.MessageIDs) > 0 && (len(batch.MessageIDs) >= cfg.MinMessages || time.Since(first) >= time.Duration(cfg.FlushSeconds)*time.Second || bytes >= cfg.InputTokens*4) {
			out = append(out, batch)
		}
	}
	return out, nil
}
func (r *IntelligenceRepository) Reserve(ctx context.Context, b bg.Batch, job c.AsyncJobProposal) (err error) {
	defer func() {
		if err == nil {
			trace.RefreshJobs(ctx, r.db, b.TurnID)
		}
	}()

	if len(b.MessageIDs) < 1 || len(b.MessageIDs) > 32 || job.Kind != "memory_write" {
		return mem.ErrDenied
	}
	// Read current use policy before reserving. The reservation rechecks pause
	// and ownership inside the same user lock as controls/deletion/finalization.
	snap, e := r.Snapshot(ctx, b.UserID, mem.Meta{UserID: b.UserID, TurnID: b.TurnID, OperationID: c.NewID(), OnlyTargets: true}, b.MessageIDs)
	if e != nil {
		return e
	}
	ok, e := r.policy.Allow(ctx, mem.Permission{Action: mem.Extract, User: snap.User, TurnID: b.TurnID})
	if e != nil || !ok || snap.User.MemoryPaused {
		return errors.Join(mem.ErrDenied, e)
	}
	tx, e := r.begin(ctx, b.UserID, b.TurnID)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var paused bool
	if e = tx.QueryRowContext(ctx, `SELECT memory_paused FROM dbo.ai_user_state WHERE user_id=@p1`, b.UserID).Scan(&paused); e != nil {
		return e
	}
	if paused {
		return mem.ErrDenied
	}
	var existing int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=@p2 AND dedupe_key=@p3`, b.UserID, job.Kind, job.DedupeKey).Scan(&existing); e != nil {
		return e
	}
	if existing > 0 {
		return nil
	}
	id := uuid.NewSHA1(b.UserID, []byte("intelligence-review-job:"+job.DedupeKey))
	if _, e = tx.ExecContext(ctx, `INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json) VALUES(@p1,@p2,@p3,@p4,@p5,@p6)`, id, b.UserID, memoryNullableID(b.TurnID), job.Kind, job.DedupeKey, string(job.Payload)); e != nil {
		return e
	}
	for _, message := range b.MessageIDs {
		var version []byte
		if e = tx.QueryRowContext(ctx, `SELECT CONVERT(binary(8),m.search_version) FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id AND v.user_id=@p1 JOIN dbo.turns t ON t.id=m.turn_id AND t.user_id=@p1 WHERE m.id=@p2 AND m.role=N'user' AND m.deleted_at IS NULL AND v.deleted_at IS NULL AND t.status=N'completed'`, b.UserID, message).Scan(&version); e != nil {
			return mem.ErrDenied
		}
		var prior []byte
		var reviewed sql.NullTime
		e = tx.QueryRowContext(ctx, `SELECT source_version,reviewed_at FROM dbo.ai_memory_review_messages WHERE user_id=@p1 AND message_id=@p2`, b.UserID, message).Scan(&prior, &reviewed)
		if e == nil {
			if !reviewed.Valid || string(prior) == string(version) {
				return mem.ErrConflict
			}
			_, e = tx.ExecContext(ctx, `UPDATE dbo.ai_memory_review_messages SET job_id=@p3,source_version=@p4,reviewed_at=NULL,reviewed_by_job_id=NULL,created_at=SYSUTCDATETIME() WHERE user_id=@p1 AND message_id=@p2`, b.UserID, message, id, version)
		} else if errors.Is(e, sql.ErrNoRows) {
			_, e = tx.ExecContext(ctx, `INSERT dbo.ai_memory_review_messages(user_id,message_id,job_id,source_version) VALUES(@p1,@p2,@p3,@p4)`, b.UserID, message, id, version)
		}
		if e != nil {
			return e
		}
	}
	return tx.Commit()
}
