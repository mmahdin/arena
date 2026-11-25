package mssql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
)

// RevokeBackchannel is one durable, serializable transaction for replay
// suppression, a tombstone for concurrent callbacks and matching session
// revocation. A notification from one realm cannot touch the other realm.
func (r *Repository) RevokeBackchannel(ctx context.Context, kind auth.Kind, issuer, subject, sid string, eventHash []byte, expiresAt time.Time) (bool, error) {
	if (kind != auth.User && kind != auth.Staff) || issuer == "" ||
		len(issuer) > 512 || len(subject) > 200 || len(sid) > 200 ||
		subject == "" && sid == "" || len(eventHash) != 32 || !expiresAt.After(time.Now()) {
		return false, auth.ErrInvalid
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var already int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM dbo.auth_backchannel_events WITH (UPDLOCK,HOLDLOCK) WHERE event_hash=@p1`, eventHash).Scan(&already)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.auth_backchannel_events(event_hash,expires_at) VALUES(@p1,@p2)`, eventHash, expiresAt.UTC()); err != nil {
		return false, err
	}
	scope := "sid"
	value := sid
	if sid == "" {
		scope, value = "subject", subject
	}
	fenceHash := auth.LogoutFenceHash(kind, issuer, scope, value)
	// A callback begun before this logout must not issue a fresh Theia
	// session after the notification commits. Never store raw SIDs/subjects
	// in the replay/fence tables.
	if _, err = tx.ExecContext(ctx, `IF NOT EXISTS (SELECT 1 FROM dbo.auth_logout_fences WITH (UPDLOCK,HOLDLOCK) WHERE fence_hash=@p1)
          INSERT INTO dbo.auth_logout_fences(fence_hash,expires_at) VALUES(@p1,DATEADD(day,31,SYSUTCDATETIME()))`, fenceHash); err != nil {
		return false, err
	}
	if sid != "" {
		_, err = tx.ExecContext(ctx, `UPDATE dbo.auth_sessions SET revoked_at=SYSUTCDATETIME()
          WHERE principal_kind=@p1 AND issuer=@p2 AND keycloak_sid=@p3
            AND (@p4='' OR subject=@p4) AND revoked_at IS NULL`, string(kind), issuer, sid, subject)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE dbo.auth_sessions SET revoked_at=SYSUTCDATETIME()
          WHERE principal_kind=@p1 AND issuer=@p2 AND subject=@p3 AND revoked_at IS NULL`, string(kind), issuer, subject)
	}
	if err != nil {
		return false, err
	}
	// No token, raw subject, phone or SID is captured in the audit event.
	if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.auth_audit_events(id,principal_kind,action_code,outcome_code)
      VALUES(CONVERT(uniqueidentifier,@p1),@p2,'session_revoked','success')`, uuid.NewString(), string(kind)); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// RevokeAllForPrincipal revokes all existing browser sessions for the verified
// Theia identity, not every Keycloak SSO session on remote devices.
func (r *Repository) RevokeAllForPrincipal(ctx context.Context, kind auth.Kind, principal uuid.UUID) error {
	if principal == uuid.Nil || kind != auth.User && kind != auth.Staff {
		return auth.ErrInvalid
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Lock the actor's watermark row/range before revoking sessions. A
	// concurrent callback checking the watermark cannot slip past the
	// transaction and later mint a session started before this logout.
	var existing time.Time
	err = tx.QueryRowContext(ctx, `SELECT revoked_at FROM dbo.auth_principal_logout_watermarks WITH (UPDLOCK,HOLDLOCK)
      WHERE principal_kind=@p1 AND principal_id=CONVERT(uniqueidentifier,@p2)`, string(kind), principal.String()).Scan(&existing)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.ExecContext(ctx, `INSERT INTO dbo.auth_principal_logout_watermarks(principal_kind,principal_id,revoked_at)
          VALUES(@p1,CONVERT(uniqueidentifier,@p2),SYSUTCDATETIME())`, string(kind), principal.String())
	case err == nil:
		_, err = tx.ExecContext(ctx, `UPDATE dbo.auth_principal_logout_watermarks SET revoked_at=SYSUTCDATETIME()
          WHERE principal_kind=@p1 AND principal_id=CONVERT(uniqueidentifier,@p2)`, string(kind), principal.String())
	}
	if err != nil {
		return err
	}
	if kind == auth.Staff {
		_, err = tx.ExecContext(ctx, `UPDATE dbo.auth_sessions SET revoked_at=SYSUTCDATETIME()
          WHERE principal_kind='staff' AND staff_id=CONVERT(uniqueidentifier,@p1) AND revoked_at IS NULL`, principal.String())
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE dbo.auth_sessions SET revoked_at=SYSUTCDATETIME()
          WHERE principal_kind='user' AND user_id=CONVERT(uniqueidentifier,@p1) AND revoked_at IS NULL`, principal.String())
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.auth_audit_events(id,principal_kind,actor_id,action_code,outcome_code)
      VALUES(CONVERT(uniqueidentifier,@p1),@p2,CONVERT(uniqueidentifier,@p3),'session_revoked','success')`, uuid.NewString(), string(kind), principal.String()); err != nil {
		return err
	}
	return tx.Commit()
}
