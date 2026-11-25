package mssql

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Catch up only recent completed turns with no routine extraction job. Existing
// failed jobs retain their attempt limits; this is not a historical backfill.
func (r *MemoryRepository) PendingMemoryReviews(ctx context.Context, limit int) ([]c.Admission, error) {
	if limit < 1 || limit > 8 {
		limit = 8
	}
	rows, e := r.db.QueryContext(ctx, `SELECT TOP(@p1) CONVERT(varchar(36),t.user_id),CONVERT(varchar(36),t.id),CONVERT(varchar(36),t.user_message_id),CONVERT(varchar(36),t.conversation_id),CONVERT(varchar(36),x.session_id)
 FROM dbo.turns t JOIN dbo.users u ON u.id=t.user_id AND u.deleted_at IS NULL
 JOIN dbo.conversations v ON v.id=t.conversation_id AND v.user_id=t.user_id AND v.deleted_at IS NULL
 JOIN dbo.messages m ON m.id=t.user_message_id AND m.deleted_at IS NULL
 JOIN dbo.ai_turn_context x ON x.turn_id=t.id AND x.user_id=t.user_id
 WHERE t.status=N'completed' AND t.completed_at>DATEADD(hour,-24,SYSUTCDATETIME())
 AND NOT EXISTS(SELECT 1 FROM dbo.ai_jobs j WHERE j.user_id=t.user_id AND j.turn_id=t.id AND j.kind=N'memory_write' AND j.dedupe_key=CONVERT(varchar(36),t.id))
 ORDER BY t.admission_order`, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []c.Admission{}
	for rows.Next() {
		var a c.Admission
		if e = rows.Scan(&a.UserID, &a.TurnID, &a.UserMessageID, &a.ConversationID, &a.SessionID); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
