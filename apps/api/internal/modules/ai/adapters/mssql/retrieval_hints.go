package mssql

import (
	"context"
	"database/sql"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"strings"
)

func (r *RetrievalRepository) RetrievalHints(ctx context.Context, user, trigger c.ID) (retrieval.Hints, error) {
	var out retrieval.Hints
	var focus sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),s.focus_thread_id)
 FROM dbo.messages trigger_msg JOIN dbo.conversations cv ON cv.id=trigger_msg.conversation_id AND cv.user_id=@p1 AND cv.deleted_at IS NULL
 LEFT JOIN dbo.ai_sessions s ON s.user_id=cv.user_id AND s.conversation_id=cv.id AND s.closed_at IS NULL
 WHERE trigger_msg.id=@p2 AND trigger_msg.role=N'user' AND trigger_msg.deleted_at IS NULL`, user, trigger).Scan(&focus)
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.Focus, err = nullableID(focus)
	if err != nil {
		return out, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT recent.role,recent.content FROM (
 SELECT TOP(4) m.role,LEFT(m.content,600) content,m.message_order
 FROM dbo.messages trigger_msg
 JOIN dbo.conversations cv ON cv.id=trigger_msg.conversation_id AND cv.user_id=@p1 AND cv.deleted_at IS NULL
 JOIN dbo.messages m ON m.conversation_id=cv.id AND m.message_order<trigger_msg.message_order AND m.deleted_at IS NULL
 WHERE trigger_msg.id=@p2 AND trigger_msg.role=N'user' AND trigger_msg.deleted_at IS NULL
 ORDER BY m.message_order DESC) recent ORDER BY recent.message_order`, user, trigger)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var role, text string
		if err = rows.Scan(&role, &text); err != nil {
			return out, err
		}
		parts = append(parts, role+": "+text)
	}
	out.RecentContext = strings.Join(parts, "\n")
	return out, rows.Err()
}
