package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func (r *MemoryRepository) applyFeedback(ctx context.Context, tx *sql.Tx, m mem.Meta, s mem.Snapshot, feedback mem.Feedback) error {
	if len(feedback.Retrieved) > 7 || len(feedback.Referenced) > 7 {
		return mem.ErrDenied
	}
	retrieved := map[c.ID]bool{}
	referenced := map[c.ID]bool{}
	for _, id := range feedback.Retrieved {
		retrieved[id] = true
	}
	for _, id := range feedback.Referenced {
		if !retrieved[id] {
			continue
		}
		referenced[id] = true
	}
	if len(retrieved) == 0 {
		return nil
	}
	for _, v := range s.Records {
		id := v.Item.ID
		if !retrieved[id] || v.Pinned || v.Item.Source.Origin == "user_edit" || !r.permit(ctx, s, m, mem.UseSource, "memory", id) || !r.permit(ctx, s, m, mem.Store, "memory", id) {
			continue
		}
		result, e := tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_memory_feedback WHERE user_id=@p1 AND turn_id=@p2 AND item_id=@p3) INSERT dbo.ai_memory_feedback(user_id,turn_id,item_id,referenced) VALUES(@p1,@p2,@p3,@p4)`, m.UserID, m.TurnID, id, referenced[id])
		if e != nil {
			return e
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			continue
		}
		var uses, total int
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(CASE WHEN referenced=1 THEN 1 ELSE 0 END),0) FROM dbo.ai_memory_feedback WHERE user_id=@p1 AND item_id=@p2`, m.UserID, id).Scan(&total, &uses); e != nil {
			return e
		}
		decay := uses == 0 && total%r.feedback.MinUnusedRetrievals == 0
		if !referenced[id] && !decay {
			continue
		}
		if e = archiveMemory(ctx, tx, m.UserID, id); e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `UPDATE dbo.ai_memory_items SET importance=CASE WHEN @p3=1 THEN importance*@p4 ELSE importance END,last_used_at=CASE WHEN @p5=1 THEN SYSUTCDATETIME() ELSE last_used_at END,version=version+1,updated_at=SYSUTCDATETIME() WHERE id=@p1 AND user_id=@p2 AND deleted_at IS NULL`, id, m.UserID, decay, r.feedback.ImportanceMultiplier, referenced[id])
		if e != nil {
			return e
		}
	}
	return nil
}

func (r *MemoryRepository) RecordFeedback(ctx context.Context, l mem.Lease) (err error) {
	defer func() { err = memorySQLError(err) }()
	var task mem.FeedbackTask
	if json.Unmarshal(l.Payload, &task) != nil || task.Schema != "memory_feedback.v1" {
		return mem.ErrDenied
	}
	if metadata, ok := mem.Metadata(ctx); !ok || metadata.UserID != l.UserID || !reflect.DeepEqual(metadata.Bundle, task.Bundle) {
		return mem.ErrDenied
	}
	m := mem.Meta{UserID: l.UserID, TurnID: l.TurnID, OperationID: l.ID, Bundle: task.Bundle, RequireCompleted: true}
	s, e := r.Snapshot(ctx, l.UserID, m, nil)
	if e != nil {
		return e
	}
	tx, e := r.begin(ctx, l.UserID, l.TurnID)
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
	if e = r.applyFeedback(ctx, tx, m, s, task.Feedback); e != nil {
		return e
	}
	if e = completeMemoryJob(ctx, tx, l); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE dbo.ai_user_state SET memory_revision=memory_revision+1 WHERE user_id=@p1`, l.UserID); e != nil {
		return e
	}
	return tx.Commit()
}
