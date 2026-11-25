package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/productevents"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }
func (r *Repository) Get(ctx context.Context, uid uuid.UUID) (domain.Profile, error) {
	var p domain.Profile
	var id string
	var style string
	var at sql.NullTime
	var answers sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),user_id),ISNULL(display_name,N''),conversation_style,onboarding_completed_at,onboarding_answers_json
        FROM dbo.user_profiles WHERE user_id=CONVERT(uniqueidentifier,@p1)`, uid.String()).Scan(&id, &p.DisplayName, &style, &at, &answers)
	if errors.Is(err, sql.ErrNoRows) {
		return p, httpx.ErrNotFound
	}
	if err != nil {
		return p, err
	}
	p.UserID, err = uuid.Parse(id)
	if err != nil {
		return p, err
	}
	p.ConversationStyle = domain.Style(style)
	if at.Valid {
		p.OnboardingCompletedAt = &at.Time
	}
	if answers.Valid {
		var a domain.OnboardingAnswers
		if err := json.Unmarshal([]byte(answers.String), &a); err != nil {
			return domain.Profile{}, errors.New("invalid stored onboarding answers")
		}
		if err := a.Validate(); err != nil {
			return domain.Profile{}, err
		}
		p.OnboardingAnswers = &a
	}
	return p, nil
}
func (r *Repository) Save(ctx context.Context, p domain.Profile) error {
	var answers any
	if p.OnboardingAnswers != nil {
		if err := p.OnboardingAnswers.Validate(); err != nil {
			return err
		}
		data, err := json.Marshal(p.OnboardingAnswers)
		if err != nil {
			return err
		}
		answers = string(data)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Lock the profile to distinguish the first completion from later edits.
	var previous sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT onboarding_completed_at FROM dbo.user_profiles WITH (UPDLOCK,HOLDLOCK)
        WHERE user_id=CONVERT(uniqueidentifier,@p1)`, p.UserID.String()).Scan(&previous)
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE dbo.user_profiles SET display_name=@p1,conversation_style=@p2,
      onboarding_completed_at=COALESCE(onboarding_completed_at,@p3),updated_at=SYSUTCDATETIME(),
      onboarding_answers_json=COALESCE(@p5,onboarding_answers_json) WHERE user_id=CONVERT(uniqueidentifier,@p4)`,
		p.DisplayName, string(p.ConversationStyle), p.OnboardingCompletedAt, p.UserID.String(), answers)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return httpx.ErrNotFound
	}
	if !previous.Valid && p.OnboardingCompletedAt != nil {
		if err = productevents.Record(ctx, tx, p.UserID, productevents.OnboardingCompleted, p.UserID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SaveOnboardingName updates only the name and only before onboarding completion.
// The conditional update prevents late draft requests from changing a completed profile.
func (r *Repository) SaveOnboardingName(ctx context.Context, uid uuid.UUID, name string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE dbo.user_profiles
		SET display_name=@p1, updated_at=SYSUTCDATETIME()
		WHERE user_id=CONVERT(uniqueidentifier,@p2) AND onboarding_completed_at IS NULL`, name, uid.String())
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return domain.ErrOnboardingAlreadyComplete
	}
	return nil
}
