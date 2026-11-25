package mssql

import (
	"context"
	"database/sql"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// correlateTransaction carries only identity to SQL source triggers. The
// transaction fence makes stale pooled-session state harmless, including rollback.
func correlateTransaction(ctx context.Context, tx *sql.Tx, user, turn c.ID) error {
	var origin any
	if c.ValidID(turn) {
		origin = turn
	}
	_, err := tx.ExecContext(ctx, `DECLARE @origin nvarchar(4000)=(SELECT CURRENT_TRANSACTION_ID() transaction_id,CONVERT(varchar(36),@p1) user_id,CONVERT(varchar(36),@p2) turn_id FOR JSON PATH,WITHOUT_ARRAY_WRAPPER);
 EXEC sys.sp_set_session_context @key=N'theia_trace_origin',@value=@origin;`, user, origin)
	return err
}
