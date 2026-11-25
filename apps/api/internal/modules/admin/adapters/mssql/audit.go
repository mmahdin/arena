package mssql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

// AuditRepository only accepts bounded, allowlisted metadata. It has no
// update/delete or arbitrary raw SQL interface.
type AuditRepository struct{ db *sql.DB }

func NewAuditRepository(db *sql.DB) *AuditRepository { return &AuditRepository{db: db} }

func (r *AuditRepository) Append(ctx context.Context, event domain.AuditEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	if r == nil || r.db == nil {
		return errors.New("admin audit SQL unavailable")
	}
	var resourceID interface{}
	if event.ResourceID != uuid.Nil {
		resourceID = event.ResourceID.String()
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO dbo.admin_audit_events
	  (id,actor_staff_id,action_code,resource_kind,resource_id,outcome_code,request_id)
	  SELECT CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),@p3,@p4,
	         CONVERT(uniqueidentifier,@p5),@p6,CONVERT(uniqueidentifier,@p7)
	  WHERE EXISTS (SELECT 1 FROM dbo.staff_accounts WHERE id=CONVERT(uniqueidentifier,@p2) AND active=1)`,
		event.ID.String(), event.ActorStaffID.String(), event.ActionCode, event.ResourceKind, resourceID, event.OutcomeCode, event.RequestID.String())
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("admin audit staff deactivated before recording")
	}
	return nil
}

// List uses SQL Server time ordering and keyset pagination. The extra row is
// only for determining whether a cursor is available; it is not returned.
func (r *AuditRepository) List(ctx context.Context, filter domain.AuditFilter) ([]domain.AuditEvent, string, error) {
	if err := filter.Validate(); err != nil {
		return nil, "", err
	}
	if r == nil || r.db == nil {
		return nil, "", errors.New("admin audit SQL unavailable")
	}
	var actor, cursorID interface{}
	var cursorAt interface{}
	if filter.ActorStaffID != uuid.Nil {
		actor = filter.ActorStaffID.String()
	}
	if filter.Cursor != nil {
		cursorID = filter.Cursor.ID.String()
		cursorAt = filter.Cursor.RecordedAt.UTC()
	}
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p1)
	  CONVERT(varchar(36),id),CONVERT(varchar(36),actor_staff_id),action_code,
	  resource_kind,CONVERT(varchar(36),resource_id),outcome_code,
	  CONVERT(varchar(36),request_id),recorded_at
	  FROM dbo.admin_audit_events
	  WHERE recorded_at >= @p2 AND recorded_at < @p3
	    AND (@p4 IS NULL OR actor_staff_id=CONVERT(uniqueidentifier,@p4))
	    AND (@p5='' OR action_code=@p5)
	    AND (@p6 IS NULL OR recorded_at < @p6 OR
	          (recorded_at=@p6 AND id < CONVERT(uniqueidentifier,@p7)))
	  ORDER BY recorded_at DESC,id DESC`, filter.Limit+1, filter.From.UTC(), filter.To.UTC(), actor,
		filter.ActionCode, cursorAt, cursorID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]domain.AuditEvent, 0, filter.Limit+1)
	for rows.Next() {
		var e domain.AuditEvent
		var id, actorID, reqID string
		var resource sql.NullString
		if err := rows.Scan(&id, &actorID, &e.ActionCode, &e.ResourceKind, &resource, &e.OutcomeCode, &reqID, &e.RecordedAt); err != nil {
			return nil, "", err
		}
		var parseErr error
		e.ID, parseErr = uuid.Parse(id)
		if parseErr != nil {
			return nil, "", parseErr
		}
		e.ActorStaffID, parseErr = uuid.Parse(actorID)
		if parseErr != nil {
			return nil, "", parseErr
		}
		e.RequestID, parseErr = uuid.Parse(reqID)
		if parseErr != nil {
			return nil, "", parseErr
		}
		if resource.Valid {
			e.ResourceID, parseErr = uuid.Parse(resource.String)
			if parseErr != nil {
				return nil, "", parseErr
			}
		}
		e.RecordedAt = e.RecordedAt.UTC().Truncate(time.Millisecond)
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	var next string
	if len(items) > filter.Limit {
		items = items[:filter.Limit]
		last := items[len(items)-1]
		next = (domain.AuditCursor{RecordedAt: last.RecordedAt, ID: last.ID}).Encode()
	}
	return items, next, nil
}
