package mssql

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

// HideAccount revokes access, not retention or consent. The same owner lock as
// AI finalization fences in-flight work; no dependent payload is modified.
func (r *Repository) HideAccount(ctx context.Context, u uuid.UUID) error {
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var id uuid.UUID
	if e = tx.QueryRowContext(ctx, `SELECT id FROM dbo.users WITH(UPDLOCK,HOLDLOCK) WHERE id=@p1`, u).Scan(&id); e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			return httpx.ErrNotFound
		}
		return e
	}
	_, e = tx.ExecContext(ctx, `UPDATE dbo.users SET deleted_at=SYSUTCDATETIME(),deleted_by=@p1,deletion_reason=N'user_request' WHERE id=@p1 AND deleted_at IS NULL;
 UPDATE dbo.auth_sessions SET revoked_at=SYSUTCDATETIME() WHERE principal_kind=N'user' AND user_id=@p1 AND revoked_at IS NULL;
 UPDATE dbo.ai_user_turn_leases SET lease_expires_at=DATEADD(millisecond,-1,SYSUTCDATETIME()) WHERE user_id=@p1 AND lease_expires_at>SYSUTCDATETIME();
 UPDATE dbo.ai_jobs SET status=N'cancelled',last_error_code=N'account_hidden' WHERE user_id=@p1 AND status IN(N'queued',N'running');`, u)
	if e != nil {
		return e
	}
	return tx.Commit()
}
