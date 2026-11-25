package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/product"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type ProductRepository struct {
	db    *sql.DB
	turns interface {
		ValidateOwner(context.Context, c.ID, c.ID) error
		GetTurn(context.Context, c.ID, c.ID) (c.TurnResult, error)
	}
}

func NewProductRepository(db *sql.DB, t interface {
	ValidateOwner(context.Context, c.ID, c.ID) error
	GetTurn(context.Context, c.ID, c.ID) (c.TurnResult, error)
}) *ProductRepository {
	return &ProductRepository{db, t}
}
func (r *ProductRepository) ValidateOwner(ctx context.Context, u, conv c.ID) error {
	return r.turns.ValidateOwner(ctx, u, conv)
}
func (r *ProductRepository) GetTurn(ctx context.Context, u, client c.ID) (c.TurnResult, error) {
	return r.turns.GetTurn(ctx, u, client)
}
func missing(e error) error {
	if errors.Is(e, sql.ErrNoRows) {
		return httpx.ErrNotFound
	}
	return e
}
func (r *ProductRepository) TurnInConversation(ctx context.Context, u, conv, client c.ID) error {
	var n int
	return missing(r.db.QueryRowContext(ctx, `SELECT 1 FROM dbo.turns t JOIN dbo.conversations v ON v.id=t.conversation_id AND v.user_id=t.user_id JOIN dbo.users u ON u.id=t.user_id WHERE t.user_id=@p1 AND t.conversation_id=@p2 AND t.client_turn_id=@p3 AND t.lifecycle_version=1 AND v.deleted_at IS NULL AND u.deleted_at IS NULL`, u, conv, client).Scan(&n))
}
func (r *ProductRepository) MessageVisible(ctx context.Context, u, id c.ID) error {
	var n int
	return missing(r.db.QueryRowContext(ctx, `SELECT 1 FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id JOIN dbo.users u ON u.id=v.user_id WHERE v.user_id=@p1 AND m.id=@p2 AND m.deleted_at IS NULL AND v.deleted_at IS NULL AND u.deleted_at IS NULL`, u, id).Scan(&n))
}
func (r *ProductRepository) MemoryOwned(ctx context.Context, u, id c.ID) error {
	var n int
	return missing(r.db.QueryRowContext(ctx, `SELECT 1 FROM dbo.ai_memory_items m JOIN dbo.users u ON u.id=m.user_id WHERE m.user_id=@p1 AND m.id=@p2 AND u.deleted_at IS NULL`, u, id).Scan(&n))
}
func (r *ProductRepository) Controls(ctx context.Context, u c.ID) (out product.Controls, err error) {
	out.Consent = map[string]bool{}
	var consent, prefs sql.NullString
	var paused sql.NullBool
	err = r.db.QueryRowContext(ctx, `SELECT s.memory_paused,s.consent_json,s.prefs_json FROM dbo.users u LEFT JOIN dbo.ai_user_state s ON s.user_id=u.id WHERE u.id=@p1 AND u.deleted_at IS NULL`, u).Scan(&paused, &consent, &prefs)
	if err != nil {
		return out, missing(err)
	}
	if paused.Valid {
		out.Initialized = true
	}
	if consent.Valid {
		if e := json.Unmarshal([]byte(consent.String), &out.Consent); e != nil {
			return out, e
		}
	}
	if prefs.Valid {
		if e := json.Unmarshal([]byte(prefs.String), &out.Preferences); e != nil {
			return out, e
		}
	}
	return out, nil
}
func (r *ProductRepository) EnsureState(ctx context.Context, u c.ID) error {
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var n int
	if e = tx.QueryRowContext(ctx, `SELECT 1 FROM dbo.users WITH(UPDLOCK,HOLDLOCK) WHERE id=@p1 AND deleted_at IS NULL`, u).Scan(&n); e != nil {
		return missing(e)
	}
	if _, e = tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_user_state WHERE user_id=@p1) INSERT dbo.ai_user_state(user_id,memory_paused) VALUES(@p1,0)`, u); e != nil {
		return e
	}
	return tx.Commit()
}
func (r *ProductRepository) ExportMessages(ctx context.Context, u c.ID, cursor string, limit int) (out []product.ExportMessage, next string, err error) {
	out = []product.ExportMessage{}
	if limit < 1 || limit > 100 {
		return out, "", httpx.ErrInvalid
	}
	var after int64
	if cursor != "" {
		id, e := uuid.Parse(cursor)
		if e != nil || !c.ValidID(id) {
			return nil, "", httpx.ErrInvalid
		}
		if e = r.db.QueryRowContext(ctx, `SELECT m.message_order FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id WHERE m.id=@p1 AND v.user_id=@p2`, id, u).Scan(&after); e != nil {
			return nil, "", missing(e)
		}
	}
	rows, e := r.db.QueryContext(ctx, `SELECT TOP(@p3) CONVERT(varchar(36),m.id),CONVERT(varchar(36),m.conversation_id),CONVERT(varchar(36),m.turn_id),m.role,m.content,m.created_at FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id JOIN dbo.users u ON u.id=v.user_id WHERE v.user_id=@p1 AND m.message_order>@p2 AND m.deleted_at IS NULL AND v.deleted_at IS NULL AND u.deleted_at IS NULL ORDER BY m.message_order`, u, after, limit+1)
	if e != nil {
		return out, "", e
	}
	defer rows.Close()
	for rows.Next() {
		var v product.ExportMessage
		if e = rows.Scan(&v.ID, &v.ConversationID, &v.TurnID, &v.Role, &v.Content, &v.CreatedAt); e != nil {
			return nil, "", e
		}
		if len(out) == limit {
			return out, out[len(out)-1].ID.String(), nil
		}
		out = append(out, v)
	}
	return out, "", rows.Err()
}

func (r *ProductRepository) Ready(ctx context.Context, u c.ID) error {
	var complete bool
	e := r.db.QueryRowContext(ctx, `SELECT CAST(CASE WHEN p.onboarding_completed_at IS NOT NULL THEN 1 ELSE 0 END AS bit) FROM dbo.users u LEFT JOIN dbo.user_profiles p ON p.user_id=u.id WHERE u.id=@p1 AND u.deleted_at IS NULL`, u).Scan(&complete)
	if e != nil {
		return missing(e)
	}
	if !complete {
		return httpx.ErrOnboardingRequired
	}
	return nil
}

func (r *ProductRepository) MemoryIDs(ctx context.Context, u c.ID, cursor string, limit int) (out []c.ID, next string, err error) {
	out = []c.ID{}
	if limit < 1 || limit > 100 {
		return out, "", httpx.ErrInvalid
	}
	rows, e := r.db.QueryContext(ctx, `SELECT TOP(@p3) CONVERT(varchar(36),m.id) FROM dbo.ai_memory_items m JOIN dbo.users u ON u.id=m.user_id WHERE m.user_id=@p1 AND m.deleted_at IS NULL AND u.deleted_at IS NULL AND CONVERT(varchar(36),m.id)>@p2 ORDER BY CONVERT(varchar(36),m.id)`, u, cursor, limit+1)
	if e != nil {
		return out, "", e
	}
	defer rows.Close()
	for rows.Next() {
		var id c.ID
		if e = rows.Scan(&id); e != nil {
			return nil, "", e
		}
		if len(out) == limit {
			return out, out[len(out)-1].String(), nil
		}
		out = append(out, id)
	}
	return out, "", rows.Err()
}
