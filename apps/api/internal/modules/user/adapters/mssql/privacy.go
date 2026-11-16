package mssql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

// PrivacyRepository scopes both operations to the signed session user UUID.
type PrivacyRepository struct{ db *sql.DB }

func NewPrivacyRepository(db *sql.DB) *PrivacyRepository { return &PrivacyRepository{db: db} }

func (r *PrivacyRepository) GetPrivacy(ctx context.Context, id uuid.UUID) (application.PrivacyPreference, error) {
	var result application.PrivacyPreference
	err := r.db.QueryRowContext(ctx, `SELECT conversation_sharing_enabled FROM dbo.user_profiles
        WHERE user_id=CONVERT(uniqueidentifier,@p1)`, id.String()).Scan(&result.ConversationSharingEnabled)
	if errors.Is(err, sql.ErrNoRows) {
		return result, httpx.ErrNotFound
	}
	return result, err
}
func (r *PrivacyRepository) SetPrivacy(ctx context.Context, id uuid.UUID, enabled bool) (application.PrivacyPreference, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE dbo.user_profiles SET conversation_sharing_enabled=@p1,
        updated_at=SYSUTCDATETIME() WHERE user_id=CONVERT(uniqueidentifier,@p2)`, enabled, id.String())
	if err != nil {
		return application.PrivacyPreference{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return application.PrivacyPreference{}, err
	}
	if count != 1 {
		return application.PrivacyPreference{}, httpx.ErrNotFound
	}
	return application.PrivacyPreference{ConversationSharingEnabled: enabled}, nil
}
