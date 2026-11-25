package mssql

import (
	"context"

	"database/sql"
	"errors"

	"github.com/google/uuid"
	mssqldriver "github.com/microsoft/go-mssqldb"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/productevents"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }
func (r *Repository) List(ctx context.Context, uid uuid.UUID) ([]domain.Conversation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (100) CONVERT(varchar(36),id),title,created_at,updated_at FROM dbo.conversations
       WHERE user_id=CONVERT(uniqueidentifier,@p1) AND deleted_at IS NULL ORDER BY updated_at DESC,id DESC`, uid.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Conversation{}
	for rows.Next() {
		var c domain.Conversation
		var id string
		if err := rows.Scan(&id, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.ID, err = uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (r *Repository) Create(ctx context.Context, uid uuid.UUID) (domain.Conversation, error) {
	c := domain.Conversation{ID: uuid.New(), Title: domain.DefaultTitle}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return c, err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, `INSERT INTO dbo.conversations(id,user_id,title)
        OUTPUT INSERTED.created_at,INSERTED.updated_at
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),@p3)`, c.ID.String(), uid.String(), c.Title).Scan(&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, err
	}
	if err = productevents.Record(ctx, tx, uid, productevents.ConversationStarted, c.ID); err != nil {
		return c, err
	}
	return c, tx.Commit()
}
func (r *Repository) Rename(ctx context.Context, uid, convID uuid.UUID, title string) (domain.Conversation, error) {
	var c domain.Conversation
	var id string
	err := r.db.QueryRowContext(ctx, `UPDATE dbo.conversations
        SET title=@p3,title_is_manual=1,updated_at=SYSUTCDATETIME()
        OUTPUT CONVERT(varchar(36),INSERTED.id),INSERTED.title,INSERTED.created_at,INSERTED.updated_at
        WHERE id=CONVERT(uniqueidentifier,@p1) AND user_id=CONVERT(uniqueidentifier,@p2) AND deleted_at IS NULL`,
		convID.String(), uid.String(), title).Scan(&id, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return c, httpx.ErrNotFound
	}
	if err != nil {
		return c, err
	}
	c.ID, err = uuid.Parse(id)
	return c, err
}
func (r *Repository) Messages(ctx context.Context, uid, convID uuid.UUID, limit int) ([]domain.Message, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT CAST(CASE WHEN EXISTS(SELECT 1 FROM dbo.conversations
      WHERE id=CONVERT(uniqueidentifier,@p1) AND user_id=CONVERT(uniqueidentifier,@p2) AND deleted_at IS NULL) THEN 1 ELSE 0 END AS bit)`, convID.String(), uid.String()).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, httpx.ErrNotFound
	}
	if limit < 1 || limit > 100 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,role,content,created_at FROM (
       SELECT TOP (@p3) CONVERT(varchar(36),m.id) AS id,m.role,m.content,m.created_at,m.message_order
       FROM dbo.messages m JOIN dbo.conversations c ON c.id=m.conversation_id
       WHERE m.conversation_id=CONVERT(uniqueidentifier,@p1) AND c.user_id=CONVERT(uniqueidentifier,@p2) AND c.deleted_at IS NULL AND m.deleted_at IS NULL
       ORDER BY m.message_order DESC
     ) AS recent ORDER BY message_order ASC`, convID.String(), uid.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Message{}
	for rows.Next() {
		var m domain.Message
		var id string
		if err := rows.Scan(&id, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.ID, err = uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (r *Repository) Delete(ctx context.Context, uid, convID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var owner uuid.UUID
	if err = tx.QueryRowContext(ctx, `SELECT id FROM dbo.users WITH(UPDLOCK,HOLDLOCK) WHERE id=@p1`, uid).Scan(&owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.ErrNotFound
		}
		return err
	}
	if err = tx.QueryRowContext(ctx, `SELECT id FROM dbo.conversations WHERE id=@p1 AND user_id=@p2`, convID, uid).Scan(&owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.ErrNotFound
		}
		return err
	}
	// The owner lock serializes deletion with admission/finalization. Keep all
	// payloads and derived records intact; expire only this conversation's lease.
	if _, err = tx.ExecContext(ctx, `UPDATE dbo.conversations SET deleted_at=SYSUTCDATETIME(),deleted_by=@p1,deletion_reason=N'user_request'
 WHERE id=@p2 AND user_id=@p1 AND deleted_at IS NULL;
 UPDATE l SET lease_expires_at=DATEADD(millisecond,-1,SYSUTCDATETIME()),updated_at=SYSUTCDATETIME()
 FROM dbo.ai_user_turn_leases l JOIN dbo.turns t ON t.id=l.turn_id AND t.user_id=l.user_id
 WHERE l.user_id=@p1 AND t.conversation_id=@p2 AND l.lease_expires_at>SYSUTCDATETIME()`, uid, convID); err != nil {
		return err
	}
	return tx.Commit()
}

func isUniqueViolation(err error) bool {
	var se mssqldriver.Error
	return errors.As(err, &se) && (se.Number == 2601 || se.Number == 2627)
}
