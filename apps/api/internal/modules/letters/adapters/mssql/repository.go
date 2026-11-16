package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/microsoft/go-mssqldb"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

// The source FK uses the (conversation_id, user_id) pair, preventing cross-owner provenance.
func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]application.Letter, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT TOP(30) CONVERT(varchar(36), id), title, salutation, paragraphs_json, question, created_at,
       CONVERT(varchar(36), source_conversation_id)
       FROM dbo.personalized_letters WHERE user_id=CONVERT(uniqueidentifier,@p1)
       ORDER BY created_at DESC, id DESC`, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []application.Letter{}
	for rows.Next() {
		var id, sourceID, paragraphs string
		var salutation, question sql.NullString
		l := application.Letter{Kind: "personalized"}
		if err := rows.Scan(&id, &l.Title, &salutation, &paragraphs, &question, &l.CreatedAt, &sourceID); err != nil {
			return nil, err
		}
		l.ID, err = uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		source, err := uuid.Parse(sourceID)
		if err != nil {
			return nil, err
		}
		l.SourceConversationIDs = []uuid.UUID{source}
		if salutation.Valid {
			l.Salutation = &salutation.String
		}
		if question.Valid {
			l.Question = &question.String
		}
		if err := json.Unmarshal([]byte(paragraphs), &l.Paragraphs); err != nil {
			return nil, err
		}
		items = append(items, l)
	}
	return items, rows.Err()
}

func (r *Repository) LatestEligible(ctx context.Context, uid uuid.UUID) (application.Source, error) {
	var result application.Source
	var id string
	// Require at least three completed user messages in one existing, owner-scoped conversation.
	err := r.db.QueryRowContext(ctx, `SELECT TOP(1) CONVERT(varchar(36), c.id)
      FROM dbo.conversations c WHERE c.user_id=CONVERT(uniqueidentifier,@p1)
      AND (SELECT COUNT_BIG(*) FROM dbo.messages m WHERE m.conversation_id=c.id AND m.role=N'user') >= 3
      ORDER BY c.updated_at DESC, c.id DESC`, uid.String()).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return result, application.ErrInsufficientContext
	}
	if err != nil {
		return result, err
	}
	result.ConversationID, err = uuid.Parse(id)
	if err != nil {
		return result, err
	}
	// Limit disclosure to the recent 20 messages and cap context again at the model boundary.
	rows, err := r.db.QueryContext(ctx, `SELECT role,content FROM (
        SELECT TOP(20) m.role, m.content, m.message_order
        FROM dbo.messages m JOIN dbo.conversations c ON c.id=m.conversation_id
        WHERE c.id=CONVERT(uniqueidentifier,@p1) AND c.user_id=CONVERT(uniqueidentifier,@p2)
        ORDER BY m.message_order DESC
      ) recent ORDER BY message_order ASC`, id, uid.String())
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var t application.Turn
		if err := rows.Scan(&t.Role, &t.Content); err != nil {
			return result, err
		}
		result.Turns = append(result.Turns, t)
	}
	return result, rows.Err()
}

func (r *Repository) BySource(ctx context.Context, uid, conversationID uuid.UUID) (application.Letter, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.personalized_letters
      WHERE user_id=CONVERT(uniqueidentifier,@p1) AND source_conversation_id=CONVERT(uniqueidentifier,@p2)`, uid.String(), conversationID.String()).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return application.Letter{}, httpx.ErrNotFound
	}
	if err != nil {
		return application.Letter{}, err
	}
	return r.byID(ctx, uid, id)
}
func (r *Repository) byID(ctx context.Context, uid uuid.UUID, id string) (application.Letter, error) {
	var l application.Letter
	var raw, conversationID string
	var salutation, question sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT title,salutation,paragraphs_json,question,created_at,CONVERT(varchar(36),source_conversation_id)
        FROM dbo.personalized_letters WHERE user_id=CONVERT(uniqueidentifier,@p1) AND id=CONVERT(uniqueidentifier,@p2)`, uid.String(), id).
		Scan(&l.Title, &salutation, &raw, &question, &l.CreatedAt, &conversationID)
	if errors.Is(err, sql.ErrNoRows) {
		return l, httpx.ErrNotFound
	}
	if err != nil {
		return l, err
	}
	l.ID, err = uuid.Parse(id)
	if err != nil {
		return l, err
	}
	sourceID, err := uuid.Parse(conversationID)
	if err != nil {
		return l, err
	}
	l.SourceConversationIDs = []uuid.UUID{sourceID}
	l.Kind = "personalized"
	if salutation.Valid {
		l.Salutation = &salutation.String
	}
	if question.Valid {
		l.Question = &question.String
	}
	err = json.Unmarshal([]byte(raw), &l.Paragraphs)
	return l, err
}
func uniqueError(err error) bool {
	var e mssql.Error
	return errors.As(err, &e) && (e.Number == 2601 || e.Number == 2627)
}
func (r *Repository) Save(ctx context.Context, uid, conversationID uuid.UUID, draft application.Draft) (application.Letter, error) {
	encoded, err := application.EncodeParagraphs(draft)
	if err != nil {
		return application.Letter{}, err
	}
	id := uuid.New()
	_, err = r.db.ExecContext(ctx, `INSERT dbo.personalized_letters(id,user_id,source_conversation_id,title,salutation,paragraphs_json,question)
      SELECT CONVERT(uniqueidentifier,@p1),c.user_id,c.id,@p4,@p5,@p6,@p7
      FROM dbo.conversations c
      WHERE c.id=CONVERT(uniqueidentifier,@p2) AND c.user_id=CONVERT(uniqueidentifier,@p3)`,
		id.String(), conversationID.String(), uid.String(), draft.Title, draft.Salutation, encoded, draft.Question)
	if uniqueError(err) {
		return r.BySource(ctx, uid, conversationID)
	}
	if err != nil {
		return application.Letter{}, err
	}
	return r.byID(ctx, uid, id.String())
}
func (r *Repository) Delete(ctx context.Context, uid, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM dbo.personalized_letters
      WHERE id=CONVERT(uniqueidentifier,@p1) AND user_id=CONVERT(uniqueidentifier,@p2)`, id.String(), uid.String())
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return httpx.ErrNotFound
	}
	return nil
}
