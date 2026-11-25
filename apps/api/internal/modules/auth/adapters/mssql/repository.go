package mssql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/productevents"
)

type Repository struct {
	db            *sql.DB
	syntheticOnly bool
}

// syntheticOnly marks new OIDC identities as test-only on the local synthetic environment.
// No legacy dev identity is silently linked to an authenticated identity.
func New(db *sql.DB, syntheticOnly bool) *Repository {
	return &Repository{db: db, syntheticOnly: syntheticOnly}
}

func (r *Repository) SaveLogin(ctx context.Context, rec auth.LoginRecord) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO dbo.auth_login_transactions
      (state_hash,principal_kind,expected_origin,redirect_uri,sealed_payload,expires_at)
      VALUES(@p1,@p2,@p3,@p4,@p5,@p6)`, rec.StateHash, string(rec.Kind), rec.ExpectedOrigin, rec.RedirectURI, rec.SealedPayload, rec.ExpiresAt.UTC())
	return err
}
func (r *Repository) ConsumeLogin(ctx context.Context, stateHash []byte) (auth.LoginRecord, error) {
	var rec auth.LoginRecord
	var kind string
	err := r.db.QueryRowContext(ctx, `DELETE FROM dbo.auth_login_transactions
      OUTPUT DELETED.principal_kind,DELETED.expected_origin,DELETED.redirect_uri,DELETED.sealed_payload,DELETED.expires_at
      WHERE state_hash=@p1 AND expires_at>SYSUTCDATETIME()`, stateHash).
		Scan(&kind, &rec.ExpectedOrigin, &rec.RedirectURI, &rec.SealedPayload, &rec.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return rec, auth.ErrNotFound
	}
	rec.Kind = auth.Kind(kind)
	return rec, err
}

// ResolveIdentity serializes first-user creation for each OIDC issuer/subject.
// Staff identities must exist in the allowlist BEFORE login; no auto-provision.
func (r *Repository) ResolveIdentity(ctx context.Context, kind auth.Kind, issuer, subject string) (uuid.UUID, error) {
	if issuer == "" || subject == "" || len(issuer) > 512 || len(subject) > 200 {
		return uuid.Nil, auth.ErrInvalid
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback()
	var uid, staffID sql.NullString
	var rowKind string
	err = tx.QueryRowContext(ctx, `SELECT principal_kind,CONVERT(varchar(36),user_id),CONVERT(varchar(36),staff_id)
      FROM dbo.auth_identities WITH (UPDLOCK,HOLDLOCK) WHERE issuer=@p1 AND subject=@p2`, issuer, subject).
		Scan(&rowKind, &uid, &staffID)
	var principal uuid.UUID
	switch {
	case err == nil:
		if rowKind != string(kind) {
			return uuid.Nil, auth.ErrInactive
		}
		principal, err = uuid.Parse(uid.String)
		if kind == auth.Staff {
			principal, err = uuid.Parse(staffID.String)
		}
		if err != nil {
			return uuid.Nil, auth.ErrInvalid
		}
	case errors.Is(err, sql.ErrNoRows) && kind == auth.User:
		principal = uuid.New()
		testKey := "oidc-" + principal.String()
		var isTest bool = r.syntheticOnly
		_, err = tx.ExecContext(ctx, `INSERT INTO dbo.users(id,test_key,is_test)
           VALUES(CONVERT(uniqueidentifier,@p1),@p2,@p3)`, principal.String(), testKey, isTest)
		if err != nil {
			return uuid.Nil, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO dbo.user_profiles(user_id,conversation_style)
            VALUES(CONVERT(uniqueidentifier,@p1),N'balanced')`, principal.String())
		if err != nil {
			return uuid.Nil, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO dbo.auth_identities(id,issuer,subject,principal_kind,user_id)
            VALUES(CONVERT(uniqueidentifier,@p1),@p2,@p3,'user',CONVERT(uniqueidentifier,@p4))`, uuid.NewString(), issuer, subject, principal.String())
		if err != nil {
			return uuid.Nil, err
		}
		if err = productevents.Record(ctx, tx, principal, productevents.UserRegistered, principal); err != nil {
			return uuid.Nil, err
		}
	case errors.Is(err, sql.ErrNoRows):
		return uuid.Nil, auth.ErrInactive
	default:
		return uuid.Nil, err
	}
	var eligible bool
	if kind == auth.User {
		err = tx.QueryRowContext(ctx, `SELECT CAST(CASE WHEN EXISTS(SELECT 1 FROM dbo.users WHERE id=CONVERT(uniqueidentifier,@p1) AND deleted_at IS NULL) THEN 1 ELSE 0 END AS bit)`, principal.String()).Scan(&eligible)
	} else {
		err = tx.QueryRowContext(ctx, `SELECT CAST(CASE WHEN EXISTS(SELECT 1 FROM dbo.staff_accounts WHERE id=CONVERT(uniqueidentifier,@p1) AND active=1) THEN 1 ELSE 0 END AS bit)`, principal.String()).Scan(&eligible)
	}
	if err != nil {
		return uuid.Nil, err
	}
	if !eligible {
		return uuid.Nil, auth.ErrInactive
	}
	if err = tx.Commit(); err != nil {
		return uuid.Nil, err
	}
	return principal, nil
}

func (r *Repository) SaveSession(ctx context.Context, s auth.Session) error {
	var userID, staffID interface{}
	if s.Kind == auth.User {
		userID = s.PrincipalID.String()
	} else if s.Kind == auth.Staff {
		staffID = s.PrincipalID.String()
	} else {
		return auth.ErrInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	loginStartedAt := s.LoginStartedAt
	if loginStartedAt.IsZero() {
		loginStartedAt = s.CreatedAt
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO dbo.auth_sessions
       (token_hash,principal_kind,user_id,staff_id,issuer,subject,keycloak_sid,created_at,last_seen_at,idle_ttl_seconds,idle_expires_at,expires_at)
       SELECT @p1,@p2,CONVERT(uniqueidentifier,@p3),CONVERT(uniqueidentifier,@p4),@p5,@p6,@p7,@p8,@p8,@p9,@p10,@p11
       WHERE EXISTS (SELECT 1 FROM dbo.auth_identities AS i WHERE i.issuer=@p5 AND i.subject=@p6 AND i.principal_kind=@p2
       AND ((i.user_id=CONVERT(uniqueidentifier,@p3) AND @p2='user') OR (i.staff_id=CONVERT(uniqueidentifier,@p4) AND @p2='staff')))
       AND ((@p2='user' AND EXISTS(SELECT 1 FROM dbo.users WHERE id=CONVERT(uniqueidentifier,@p3) AND deleted_at IS NULL))
            OR (@p2='staff' AND EXISTS(SELECT 1 FROM dbo.staff_accounts WHERE id=CONVERT(uniqueidentifier,@p4) AND active=1)))
       AND NOT EXISTS (SELECT 1 FROM dbo.auth_logout_fences WITH (UPDLOCK,HOLDLOCK)
          WHERE fence_hash IN (@p12,@p13) AND created_at>=@p14 AND expires_at>SYSUTCDATETIME())
       AND NOT EXISTS (SELECT 1 FROM dbo.auth_principal_logout_watermarks WITH (UPDLOCK,HOLDLOCK)
          WHERE principal_kind=@p2 AND principal_id=COALESCE(CONVERT(uniqueidentifier,@p3),CONVERT(uniqueidentifier,@p4))
            AND revoked_at>=@p14)`,
		s.TokenHash, string(s.Kind), userID, staffID, s.Issuer, s.Subject, nullable(s.KeycloakSID), s.CreatedAt.UTC(), int(s.IdleTTL.Seconds()), s.IdleExpiresAt.UTC(), s.ExpiresAt.UTC(),
		auth.LogoutFenceHash(s.Kind, s.Issuer, "sid", s.KeycloakSID), auth.LogoutFenceHash(s.Kind, s.Issuer, "subject", s.Subject), loginStartedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return auth.ErrInactive
	}
	// A successful login must not mint a session without a corresponding audit
	// event. Both records commit or neither does.
	_, err = tx.ExecContext(ctx, `INSERT INTO dbo.auth_audit_events(id,principal_kind,actor_id,action_code,outcome_code)
       VALUES(CONVERT(uniqueidentifier,@p1),@p2,CONVERT(uniqueidentifier,@p3),'login','success')`, uuid.NewString(), string(s.Kind), s.PrincipalID.String())
	if err != nil {
		return err
	}
	return tx.Commit()
}

func nullable(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ActiveSession performs a single conditional update, refreshing the idle
// expiration only for still-active principals. SQL outages fail closed.
func (r *Repository) ActiveSession(ctx context.Context, kind auth.Kind, tokenHash []byte) (auth.Session, error) {
	var s auth.Session
	var userID, staffID sql.NullString
	var sid sql.NullString
	var kindText string
	var idle int
	err := r.db.QueryRowContext(ctx, `UPDATE s SET
      last_seen_at=SYSUTCDATETIME(),
      idle_expires_at=CASE WHEN DATEADD(second,s.idle_ttl_seconds,SYSUTCDATETIME())<s.expires_at
            THEN DATEADD(second,s.idle_ttl_seconds,SYSUTCDATETIME()) ELSE s.expires_at END
      OUTPUT INSERTED.principal_kind,CONVERT(varchar(36),INSERTED.user_id),CONVERT(varchar(36),INSERTED.staff_id),
             INSERTED.issuer,INSERTED.subject,INSERTED.keycloak_sid,INSERTED.created_at,INSERTED.idle_expires_at,INSERTED.expires_at,INSERTED.idle_ttl_seconds
      FROM dbo.auth_sessions AS s
      LEFT JOIN dbo.users AS u ON u.id=s.user_id
      LEFT JOIN dbo.staff_accounts AS a ON a.id=s.staff_id
      WHERE s.token_hash=@p1 AND s.principal_kind=@p2 AND s.revoked_at IS NULL
        AND s.idle_expires_at>SYSUTCDATETIME() AND s.expires_at>SYSUTCDATETIME()
        AND ((s.principal_kind='user' AND u.deleted_at IS NULL AND u.id IS NOT NULL)
             OR (s.principal_kind='staff' AND a.active=1))`, tokenHash, string(kind)).
		Scan(&kindText, &userID, &staffID, &s.Issuer, &s.Subject, &sid, &s.CreatedAt, &s.IdleExpiresAt, &s.ExpiresAt, &idle)
	if errors.Is(err, sql.ErrNoRows) {
		return s, auth.ErrNotFound
	}
	if err != nil {
		return s, err
	}
	s.Kind = auth.Kind(kindText)
	if s.Kind != kind {
		return auth.Session{}, auth.ErrInvalid
	}
	idText := userID.String
	if kind == auth.Staff {
		idText = staffID.String
	}
	s.PrincipalID, err = uuid.Parse(idText)
	if err != nil {
		return auth.Session{}, auth.ErrInvalid
	}
	s.KeycloakSID = sid.String
	s.IdleTTL = time.Duration(idle) * time.Second
	return s, nil
}

func (r *Repository) RevokeSession(ctx context.Context, kind auth.Kind, tokenHash []byte) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var uid, sid sql.NullString
	err = tx.QueryRowContext(ctx, `UPDATE dbo.auth_sessions SET revoked_at=SYSUTCDATETIME()
       OUTPUT CONVERT(varchar(36),INSERTED.user_id),CONVERT(varchar(36),INSERTED.staff_id)
       WHERE token_hash=@p1 AND principal_kind=@p2 AND revoked_at IS NULL`, tokenHash, string(kind)).Scan(&uid, &sid)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.ErrNotFound
	}
	if err != nil {
		return err
	}
	id := uid.String
	if kind == auth.Staff {
		id = sid.String
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO dbo.auth_audit_events(id,principal_kind,actor_id,action_code,outcome_code)
      VALUES(CONVERT(uniqueidentifier,@p1),@p2,CONVERT(uniqueidentifier,@p3),'logout','success')`, uuid.NewString(), string(kind), id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) AppendAudit(ctx context.Context, e auth.AuditEvent) error {
	var actor interface{}
	if e.PrincipalID != uuid.Nil {
		actor = e.PrincipalID.String()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO dbo.auth_audit_events(id,principal_kind,actor_id,action_code,outcome_code,request_id)
      VALUES(CONVERT(uniqueidentifier,@p1),@p2,CONVERT(uniqueidentifier,@p3),@p4,@p5,@p6)`, uuid.NewString(), string(e.Kind), actor, e.Action, e.Outcome, nullable(e.RequestID))
	return err
}
