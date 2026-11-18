package mssql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/brain"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// BrainFacts reads WP-02 policy evidence only. It never updates a cooldown,
// memory, protocol, consent flag or deletion metadata.
type BrainFacts struct{ db *sql.DB }

var _ brain.Facts = (*BrainFacts)(nil)

func NewBrainFacts(db *sql.DB) (*BrainFacts, error) {
	if db == nil {
		return nil, errors.New("WP-10 policy database is nil")
	}
	return &BrainFacts{db}, nil
}
func (r *BrainFacts) HypothesisAllowed(ctx context.Context, user, turn c.ID, thread *c.ID, tag string) (bool, error) {
	if !c.ValidID(user) || !c.ValidID(turn) || tag == "" || len(tag) > 100 || thread != nil && !c.ValidID(*thread) {
		return false, nil
	}
	var allowed bool
	err := r.db.QueryRowContext(ctx, `SELECT CAST(CASE WHEN EXISTS (
 SELECT 1 FROM dbo.turns t
 JOIN dbo.conversations cv ON cv.id=t.conversation_id AND cv.user_id=t.user_id AND cv.deleted_at IS NULL
 JOIN dbo.ai_user_state u ON u.user_id=t.user_id AND u.memory_paused=0

 WHERE t.id=@p2 AND t.user_id=@p1 AND t.status=N'running' AND t.lifecycle_version=1
 AND (@p3 IS NULL OR EXISTS (SELECT 1 FROM dbo.ai_threads th WHERE th.id=@p3 AND th.user_id=t.user_id))
 AND NOT EXISTS (SELECT 1 FROM dbo.ai_hypothesis_cooldowns cd WHERE cd.user_id=t.user_id AND cd.tag=@p4
 AND (cd.thread_id IS NULL OR cd.thread_id=@p3) AND cd.until_at>SYSUTCDATETIME())
 ) THEN 1 ELSE 0 END AS bit)`, user, turn, thread, tag).Scan(&allowed)
	if err != nil {
		return false, errors.New("WP-10 hypothesis policy read failed")
	}
	return allowed, nil
}
func (r *BrainFacts) OfferAllowed(ctx context.Context, user, turn c.ID, minGap, maxPerSession int) (bool, error) {
	if !c.ValidID(user) || !c.ValidID(turn) || minGap < 0 || maxPerSession < 1 {
		return false, nil
	}
	var allowed bool
	err := r.db.QueryRowContext(ctx, `SELECT CAST(CASE WHEN EXISTS (
 SELECT 1 FROM dbo.turns t
 JOIN dbo.conversations cv ON cv.id=t.conversation_id AND cv.user_id=t.user_id AND cv.deleted_at IS NULL
 JOIN dbo.ai_sessions s ON s.user_id=t.user_id AND s.conversation_id=t.conversation_id AND s.closed_at IS NULL
 WHERE t.user_id=@p1 AND t.id=@p2 AND t.status=N'running' AND t.lifecycle_version=1
 AND (SELECT COUNT(*) FROM dbo.ai_turn_context x WHERE x.user_id=@p1 AND x.session_id=s.id AND x.objective=N'offer_intervention')<@p4
 AND (NOT EXISTS (SELECT 1 FROM dbo.ai_turn_context x WHERE x.user_id=@p1 AND x.session_id=s.id AND x.objective=N'offer_intervention')
 OR (SELECT COUNT(*) FROM dbo.ai_turn_context later WHERE later.user_id=@p1 AND later.session_id=s.id
 AND later.created_at>(SELECT MAX(prev.created_at) FROM dbo.ai_turn_context prev WHERE prev.user_id=@p1 AND prev.session_id=s.id AND prev.objective=N'offer_intervention'))>=@p3)
 ) THEN 1 ELSE 0 END AS bit)`, user, turn, minGap, maxPerSession).Scan(&allowed)
	if err != nil {
		return false, errors.New("WP-10 offer policy read failed")
	}
	return allowed, nil
}
func (*BrainFacts) VoiceAllowed(context.Context, c.ID, c.ID, c.ID, time.Duration, int) (bool, error) {
	// Final Architecture §11.1 explicitly defers hypothesis voicing in the MVP.
	// There is no approved per-session voice history/read policy in WP-01–09.
	return false, nil
}
