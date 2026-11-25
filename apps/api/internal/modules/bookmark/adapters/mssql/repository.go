package mssql

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/bookmark/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/productevents"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) owned(ctx context.Context, uid, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT CAST(CASE WHEN EXISTS(
    SELECT 1 FROM dbo.messages m JOIN dbo.conversations c ON c.id=m.conversation_id
    WHERE m.id=CONVERT(uniqueidentifier,@p1) AND c.user_id=CONVERT(uniqueidentifier,@p2) AND c.deleted_at IS NULL AND m.deleted_at IS NULL
 ) THEN 1 ELSE 0 END AS bit)`, id.String(), uid.String()).Scan(&exists)
	return exists, err
}
func (r *Repository) Save(ctx context.Context, uid, id uuid.UUID) error {
	// The row/key-range locks make parallel identical saves idempotent.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO dbo.saved_messages(message_id,user_id)
 SELECT m.id,c.user_id FROM dbo.messages m JOIN dbo.conversations c ON c.id=m.conversation_id
 WHERE m.id=CONVERT(uniqueidentifier,@p1) AND c.user_id=CONVERT(uniqueidentifier,@p2) AND c.deleted_at IS NULL AND m.deleted_at IS NULL
 AND NOT EXISTS(SELECT 1 FROM dbo.saved_messages s WITH(UPDLOCK,HOLDLOCK) WHERE s.message_id=m.id AND s.deleted_at IS NULL)`, id.String(), uid.String())
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count > 0 {
		if err = productevents.Record(ctx, tx, uid, productevents.BookmarkSaved, id); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	exists, err := r.owned(ctx, uid, id)
	if err != nil {
		return err
	}
	if !exists {
		return httpx.ErrNotFound
	}
	return nil
}
func (r *Repository) Remove(ctx context.Context, uid, id uuid.UUID) error {
	exists, err := r.owned(ctx, uid, id)
	if err != nil {
		return err
	}
	if !exists {
		return httpx.ErrNotFound
	}
	_, err = r.db.ExecContext(ctx, `UPDATE s SET deleted_at=SYSUTCDATETIME(),deleted_by=CONVERT(uniqueidentifier,@p2),deletion_reason=N'user_request' FROM dbo.saved_messages s
 JOIN dbo.messages m ON m.id=s.message_id
 JOIN dbo.conversations c ON c.id=m.conversation_id
 WHERE s.message_id=CONVERT(uniqueidentifier,@p1) AND s.user_id=CONVERT(uniqueidentifier,@p2)
 AND c.user_id=CONVERT(uniqueidentifier,@p2) AND c.deleted_at IS NULL AND m.deleted_at IS NULL AND s.deleted_at IS NULL`, id.String(), uid.String())
	return err
}
func (r *Repository) List(ctx context.Context, uid uuid.UUID) ([]application.SavedMessage, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT TOP(100) CONVERT(varchar(36),m.id),CONVERT(varchar(36),m.conversation_id),
 m.role,m.content,m.created_at,s.saved_at FROM dbo.saved_messages s
 JOIN dbo.messages m ON m.id=s.message_id JOIN dbo.conversations c ON c.id=m.conversation_id
 WHERE s.user_id=CONVERT(uniqueidentifier,@p1) AND c.user_id=CONVERT(uniqueidentifier,@p1) AND c.deleted_at IS NULL AND m.deleted_at IS NULL AND s.deleted_at IS NULL
 ORDER BY s.saved_at DESC,m.message_order DESC`, uid.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []application.SavedMessage{}
	for rows.Next() {
		var item application.SavedMessage
		var id, conv string
		if err := rows.Scan(&id, &conv, &item.Role, &item.Content, &item.CreatedAt, &item.SavedAt); err != nil {
			return nil, err
		}
		item.ID, err = uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		item.ConversationID, err = uuid.Parse(conv)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
