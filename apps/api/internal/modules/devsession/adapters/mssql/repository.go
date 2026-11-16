package mssql

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/devsession/application"
)

type Repository struct {
	db             *sql.DB
	identitySecret []byte
}

// A key is required for phone-shaped dev identities. Legacy tests may use New(db).
func New(db *sql.DB, secret ...string) *Repository {
	r := &Repository{db: db}
	if len(secret) > 0 {
		r.identitySecret = []byte(secret[0])
	}
	return r
}

func (r *Repository) List(ctx context.Context) ([]application.TestUser, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT CONVERT(varchar(36),id),test_key FROM dbo.users WHERE is_test=1 ORDER BY test_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []application.TestUser{}
	for rows.Next() {
		var u application.TestUser
		var id string
		if err := rows.Scan(&id, &u.TestKey); err != nil {
			return nil, err
		}
		u.ID, err = uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CreateSession is deliberately restricted in SQL to the seeded test-user set.
func (r *Repository) CreateSession(ctx context.Context, uid, sid uuid.UUID, expiresAt time.Time) (bool, error) {
	result, err := r.db.ExecContext(ctx, `INSERT INTO dbo.dev_sessions(id,user_id,expires_at)
      SELECT CONVERT(uniqueidentifier,@p1),u.id,@p3 FROM dbo.users AS u
      WHERE u.id=CONVERT(uniqueidentifier,@p2) AND u.is_test=1`, sid.String(), uid.String(), expiresAt.UTC())
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (r *Repository) RevokeSession(ctx context.Context, uid, sid uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE dbo.dev_sessions SET revoked_at=SYSUTCDATETIME()
       WHERE id=CONVERT(uniqueidentifier,@p1) AND user_id=CONVERT(uniqueidentifier,@p2) AND revoked_at IS NULL`, sid.String(), uid.String())
	return err
}

func (r *Repository) IsActive(ctx context.Context, uid, sid uuid.UUID) (bool, error) {
	var active bool
	err := r.db.QueryRowContext(ctx, `SELECT CAST(CASE WHEN EXISTS(
      SELECT 1 FROM dbo.dev_sessions AS s
      JOIN dbo.users AS u ON u.id=s.user_id
      WHERE s.id=CONVERT(uniqueidentifier,@p1) AND s.user_id=CONVERT(uniqueidentifier,@p2)
        AND s.revoked_at IS NULL AND s.expires_at>SYSUTCDATETIME() AND u.is_test=1
      ) THEN 1 ELSE 0 END AS bit)`, sid.String(), uid.String()).Scan(&active)
	return active, err
}

// FindOrCreatePhoneSession is a SERIALIZABLE transaction: concurrent logins with
// the same synthetic identifier must resolve to ONE user/profile. Never store or
// log a raw mobile-shaped identifier; persist only a keyed digest.
func (r *Repository) FindOrCreatePhoneSession(ctx context.Context, phone string, sid uuid.UUID, expiresAt time.Time) (uuid.UUID, bool, error) {
	if len(r.identitySecret) < 32 {
		return uuid.Nil, false, errors.New("dev identity key unavailable")
	}
	mac := hmac.New(sha256.New, r.identitySecret)
	_, _ = mac.Write([]byte("wellbeing-dev-phone-v1:"))
	_, _ = mac.Write([]byte(phone))
	digest := mac.Sum(nil)

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return uuid.Nil, false, err
	}
	defer tx.Rollback()
	var idText string
	created := false
	err = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),user_id)
        FROM dbo.dev_phone_identities WITH (UPDLOCK,HOLDLOCK) WHERE lookup_key=@p1`, digest).Scan(&idText)
	var uid uuid.UUID
	switch {
	case errors.Is(err, sql.ErrNoRows):
		uid = uuid.New()
		// test_key is an opaque synthetic label; it never contains the phone.
		if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.users(id,test_key,is_test)
            VALUES(CONVERT(uniqueidentifier,@p1),@p2,1)`, uid.String(), "hamrah-dev-"+uid.String()); err != nil {
			return uuid.Nil, false, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.user_profiles(user_id,conversation_style)
            VALUES(CONVERT(uniqueidentifier,@p1),N'balanced')`, uid.String()); err != nil {
			return uuid.Nil, false, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.dev_phone_identities(lookup_key,user_id)
            VALUES(@p1,CONVERT(uniqueidentifier,@p2))`, digest, uid.String()); err != nil {
			return uuid.Nil, false, err
		}
		created = true
	case err != nil:
		return uuid.Nil, false, err
	default:
		if uid, err = uuid.Parse(idText); err != nil {
			return uuid.Nil, false, err
		}
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO dbo.dev_sessions(id,user_id,expires_at)
        SELECT CONVERT(uniqueidentifier,@p1),id,@p3 FROM dbo.users
        WHERE id=CONVERT(uniqueidentifier,@p2) AND is_test=1`, sid.String(), uid.String(), expiresAt.UTC())
	if err != nil {
		return uuid.Nil, false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return uuid.Nil, false, err
	}
	if affected != 1 {
		return uuid.Nil, false, errors.New("synthetic identity disabled")
	}
	if err := tx.Commit(); err != nil {
		return uuid.Nil, false, err
	}
	return uid, created, nil
}
