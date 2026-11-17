package mssql

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	mssqldriver "github.com/microsoft/go-mssqldb"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/application"
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
	err := r.db.QueryRowContext(ctx, `INSERT INTO dbo.conversations(id,user_id,title)
        OUTPUT INSERTED.created_at,INSERTED.updated_at
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),@p3)`, c.ID.String(), uid.String(), c.Title).Scan(&c.CreatedAt, &c.UpdatedAt)
	return c, err
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

// Lease duration intentionally exceeds the configured 35s model HTTP timeout.
// After a process crash a retry of the SAME clientTurnId can reclaim an expired
// reservation. A random lease ID prevents stale workers from writing or releasing
// a turn reclaimed by another request.
const turnLeaseSeconds = 120

func (r *Repository) ClaimTurn(ctx context.Context, uid, convID, turnID uuid.UUID, input string) (application.ClaimedTurn, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return application.ClaimedTurn{}, err
	}
	defer tx.Rollback()
	if err := lockActiveConversation(ctx, tx, uid, convID); err != nil {
		return application.ClaimedTurn{}, err
	}
	out, err := claimTurn(ctx, tx, uid, convID, turnID, input)
	if err != nil {
		return out, err
	}
	if err := tx.Commit(); err != nil {
		return application.ClaimedTurn{}, err
	}
	return out, nil
}

// Lock order matches WP-02: owner first, then its visible conversation.
func lockActiveConversation(ctx context.Context, tx *sql.Tx, uid, convID uuid.UUID) error {
	var id string
	err := tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.users WITH(UPDLOCK,HOLDLOCK) WHERE id=@p1`, uid).Scan(&id)
	if err == nil {
		err = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.conversations WHERE id=@p1 AND user_id=@p2 AND deleted_at IS NULL`, convID, uid).Scan(&id)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.ErrNotFound
	}
	return err
}

func claimTurn(ctx context.Context, tx *sql.Tx, uid, convID, turnID uuid.UUID, input string) (application.ClaimedTurn, error) {
	var out application.ClaimedTurn
	hash := sha256.Sum256([]byte(input))
	leaseID := uuid.New()

	// Only the exact owner, conversation and input can reclaim an expired turn.
	// SQL's conditional UPDATE serializes competing reclaim attempts.
	recovered, err := tx.ExecContext(ctx, `UPDATE dbo.turns SET lease_id=CONVERT(uniqueidentifier,@p5),
        lease_expires_at=DATEADD(second,@p6,SYSUTCDATETIME())
        WHERE id=CONVERT(uniqueidentifier,@p1) AND conversation_id=CONVERT(uniqueidentifier,@p2)
        AND user_id=CONVERT(uniqueidentifier,@p3) AND input_hash=@p4
        AND status=N'processing' AND lease_expires_at<=SYSUTCDATETIME()`,
		turnID.String(), convID.String(), uid.String(), hash[:], leaseID.String(), turnLeaseSeconds)
	if err != nil {
		return out, err
	}
	affected, err := recovered.RowsAffected()
	if err != nil {
		return out, err
	}
	if affected == 1 {
		out.LeaseID = leaseID
		return out, nil
	}

	// The unique turn id and range lock serialize first-time concurrent claims.
	result, err := tx.ExecContext(ctx, `INSERT INTO dbo.turns(id,conversation_id,user_id,input_hash,status,lease_id,lease_expires_at)
        SELECT CONVERT(uniqueidentifier,@p1),c.id,c.user_id,@p4,N'processing',CONVERT(uniqueidentifier,@p5),
        DATEADD(second,@p6,SYSUTCDATETIME())
        FROM dbo.conversations AS c
        WHERE c.id=CONVERT(uniqueidentifier,@p2) AND c.user_id=CONVERT(uniqueidentifier,@p3)
        AND NOT EXISTS(SELECT 1 FROM dbo.turns WITH (UPDLOCK,HOLDLOCK) WHERE id=CONVERT(uniqueidentifier,@p1))`,
		turnID.String(), convID.String(), uid.String(), hash[:], leaseID.String(), turnLeaseSeconds)
	if err != nil && !isUniqueViolation(err) {
		return out, err
	}
	if err == nil {
		n, e := result.RowsAffected()
		if e != nil {
			return out, e
		}
		if n == 1 {
			out.LeaseID = leaseID
			return out, nil
		}
	}
	var status string
	var storedHash []byte
	err = tx.QueryRowContext(ctx, `SELECT status,input_hash FROM dbo.turns
        WHERE id=CONVERT(uniqueidentifier,@p1) AND conversation_id=CONVERT(uniqueidentifier,@p2)
        AND user_id=CONVERT(uniqueidentifier,@p3)`, turnID.String(), convID.String(), uid.String()).Scan(&status, &storedHash)
	if errors.Is(err, sql.ErrNoRows) {
		return out, httpx.ErrNotFound
	}
	if err != nil {
		return out, err
	}
	if !bytes.Equal(storedHash, hash[:]) {
		return out, httpx.ErrInvalid
	}
	if status == "processing" {
		out.Pending = true
		return out, nil
	}
	if status != "completed" {
		return out, httpx.ErrConflict
	}
	var userMsg, assistantMsg string
	err = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),t.user_message_id),CONVERT(varchar(36),t.assistant_message_id),
       m.content,ISNULL(t.prompt_version,N'') FROM dbo.turns t JOIN dbo.messages m ON m.id=t.assistant_message_id AND m.deleted_at IS NULL
       WHERE t.id=CONVERT(uniqueidentifier,@p1) AND t.user_id=CONVERT(uniqueidentifier,@p2)
       AND t.conversation_id=CONVERT(uniqueidentifier,@p3)`, turnID.String(), uid.String(), convID.String()).Scan(&userMsg, &assistantMsg, &out.Reply, &out.PromptVersion)
	if err != nil {
		return out, err
	}
	out.UserMessageID, err = uuid.Parse(userMsg)
	if err != nil {
		return out, err
	}
	out.AssistantMessageID, err = uuid.Parse(assistantMsg)
	if err != nil {
		return out, err
	}
	out.Existing = true
	return out, nil
}
func (r *Repository) CompleteTurn(ctx context.Context, uid, convID, turnID, leaseID uuid.UUID, input, reply, version string) (application.TurnResult, error) {
	var out application.TurnResult
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err := lockActiveConversation(ctx, tx, uid, convID); err != nil {
		return out, err
	}
	var status string
	var currentLeaseID sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT status,CONVERT(varchar(36),lease_id) FROM dbo.turns WITH (UPDLOCK,ROWLOCK)
      WHERE id=CONVERT(uniqueidentifier,@p1) AND user_id=CONVERT(uniqueidentifier,@p2)
      AND conversation_id=CONVERT(uniqueidentifier,@p3)`, turnID.String(), uid.String(), convID.String()).Scan(&status, &currentLeaseID)
	if errors.Is(err, sql.ErrNoRows) {
		return out, httpx.ErrNotFound
	}
	if err != nil {
		return out, err
	}
	storedLeaseID, parseErr := uuid.Parse(currentLeaseID.String)
	if status != "processing" || !currentLeaseID.Valid || parseErr != nil || storedLeaseID != leaseID {
		return out, httpx.ErrConflict
	}
	userMsgID, aiMsgID := uuid.New(), uuid.New()
	for _, m := range []struct {
		id         uuid.UUID
		role, text string
	}{{userMsgID, "user", input}, {aiMsgID, "assistant", reply}} {
		_, err = tx.ExecContext(ctx, `INSERT INTO dbo.messages(id,conversation_id,turn_id,role,content)
           VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),@p4,@p5)`,
			m.id.String(), convID.String(), turnID.String(), m.role, m.text)
		if err != nil {
			return out, fmt.Errorf("persist turn message: %w", err)
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE dbo.turns SET status=N'completed',user_message_id=CONVERT(uniqueidentifier,@p1),
       assistant_message_id=CONVERT(uniqueidentifier,@p2),prompt_version=@p3,completed_at=SYSUTCDATETIME(),
       lease_id=NULL,lease_expires_at=NULL
       WHERE id=CONVERT(uniqueidentifier,@p4) AND user_id=CONVERT(uniqueidentifier,@p5)`,
		userMsgID.String(), aiMsgID.String(), version, turnID.String(), uid.String())
	if err != nil {
		return out, err
	}
	// Auto-title only when exactly one user message (the first successful turn)
	// is stored. A user-edited title can never be overwritten by this update.
	title := domain.SuggestedTitle(input)
	_, err = tx.ExecContext(ctx, `UPDATE dbo.conversations SET title=@p3
        WHERE id=CONVERT(uniqueidentifier,@p1) AND user_id=CONVERT(uniqueidentifier,@p2) AND deleted_at IS NULL
        AND title_is_manual=0 AND title=@p4
        AND (SELECT COUNT_BIG(*) FROM dbo.messages
             WHERE conversation_id=CONVERT(uniqueidentifier,@p1) AND role=N'user')=1`,
		convID.String(), uid.String(), title, domain.DefaultTitle)
	if err != nil {
		return out, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE dbo.conversations SET updated_at=SYSUTCDATETIME()
       WHERE id=CONVERT(uniqueidentifier,@p1) AND user_id=CONVERT(uniqueidentifier,@p2) AND deleted_at IS NULL`, convID.String(), uid.String())
	if err != nil {
		return out, err
	}
	if err = tx.Commit(); err != nil {
		return out, err
	}
	return application.TurnResult{UserMessageID: userMsgID, AssistantMessageID: aiMsgID, Reply: reply, PromptVersion: version}, nil
}
func (r *Repository) ReleaseTurn(ctx context.Context, uid, convID, turnID, leaseID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE dbo.turns SET lease_expires_at=CONVERT(datetime2(3),'2000-01-01') WHERE id=CONVERT(uniqueidentifier,@p1)
      AND user_id=CONVERT(uniqueidentifier,@p2) AND conversation_id=CONVERT(uniqueidentifier,@p3)
      AND status=N'processing' AND lease_id=CONVERT(uniqueidentifier,@p4)`, turnID.String(), uid.String(), convID.String(), leaseID.String())
	return err
}
