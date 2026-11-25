package mssql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/productevents"
)

type ProgressRepository struct{ db *sql.DB }

func NewProgressRepository(db *sql.DB) *ProgressRepository { return &ProgressRepository{db: db} }
func (r *ProgressRepository) GetStep(ctx context.Context, id uuid.UUID) (int, bool, error) {
	var step int
	var completed bool
	err := r.db.QueryRowContext(ctx, `SELECT onboarding_step, CASE WHEN onboarding_completed_at IS NULL THEN 0 ELSE 1 END
        FROM dbo.user_profiles WHERE user_id=CONVERT(uniqueidentifier,@p1)`, id.String()).Scan(&step, &completed)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, httpx.ErrNotFound
	}
	return step, completed, err
}
func (r *ProgressRepository) AdvanceStep(ctx context.Context, id uuid.UUID, requested int) (int, bool, error) {
	// Single conditional SQL UPDATE prevents concurrent writers from skipping steps.
	// Older / duplicate writes are idempotent and cannot move the cursor backwards.
	if requested > 0 {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return 0, false, err
		}
		defer tx.Rollback()
		result, err := tx.ExecContext(ctx, `UPDATE dbo.user_profiles SET onboarding_step=@p1, updated_at=SYSUTCDATETIME()
            WHERE user_id=CONVERT(uniqueidentifier,@p2) AND onboarding_completed_at IS NULL
              AND onboarding_step=@p3`, requested, id.String(), requested-1)
		if err != nil {
			return 0, false, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return 0, false, err
		}
		if n == 1 && requested == 1 {
			if err = productevents.Record(ctx, tx, id, productevents.OnboardingStarted, id); err != nil {
				return 0, false, err
			}
		}
		if err = tx.Commit(); err != nil {
			return 0, false, err
		}
	}
	current, completed, err := r.GetStep(ctx, id)
	if err != nil {
		return 0, false, err
	}
	if !completed && current < requested {
		return current, false, application.ErrProgressConflict
	}
	return current, completed, nil
}
