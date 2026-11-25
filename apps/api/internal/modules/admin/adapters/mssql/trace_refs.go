package mssql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

// TraceReferenceReader is a bounded, parameterized metadata-only projection.
// It never queries user messages, local Inspector, encrypted trace bodies or
// any authenticated Phoenix service.
type TraceReferenceReader struct{ db *sql.DB }

func NewTraceReferenceReader(db *sql.DB) *TraceReferenceReader { return &TraceReferenceReader{db: db} }

func (r *TraceReferenceReader) ListTraceReferences(ctx context.Context, f domain.TraceReferenceFilter) (domain.TraceReferencePage, error) {
	if err := f.Validate(); err != nil {
		return domain.TraceReferencePage{}, err
	}
	if r == nil || r.db == nil {
		return domain.TraceReferencePage{}, domain.ErrTraceRefUnavailable
	}
	var present int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM dbo.turns WHERE id=CONVERT(uniqueidentifier,@p1) AND lifecycle_version=1`, f.TurnID.String()).Scan(&present)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TraceReferencePage{}, domain.ErrTurnNotFound
	}
	if err != nil {
		return domain.TraceReferencePage{}, err
	}
	var at, id any
	if f.Cursor != nil {
		at = f.Cursor.At.UTC()
		id = f.Cursor.ID.String()
	}
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p2)
      CONVERT(varchar(36),id),kind,trace_id,root_span_id,
      CONVERT(varchar(36),job_id),job_attempt,recorded_at
      FROM dbo.ai_trace_refs
      WHERE turn_id=CONVERT(uniqueidentifier,@p1)
        AND (@p3 IS NULL OR recorded_at<@p3 OR
             (recorded_at=@p3 AND id<CONVERT(uniqueidentifier,@p4)))
      ORDER BY recorded_at DESC,id DESC`, f.TurnID.String(), f.Limit+1, at, id)
	if err != nil {
		return domain.TraceReferencePage{}, err
	}
	defer rows.Close()
	page := domain.TraceReferencePage{TurnID: f.TurnID, Items: make([]domain.TraceReference, 0, f.Limit+1)}
	for rows.Next() {
		var v domain.TraceReference
		var rawID string
		var job sql.NullString
		var attempt sql.NullInt64
		if err := rows.Scan(&rawID, &v.Kind, &v.TraceID, &v.RootSpanID, &job, &attempt, &v.RecordedAt); err != nil {
			return domain.TraceReferencePage{}, err
		}
		v.ID, err = uuid.Parse(rawID)
		if err != nil {
			return domain.TraceReferencePage{}, err
		}
		if job.Valid {
			jobID, parseErr := uuid.Parse(job.String)
			if parseErr != nil {
				return domain.TraceReferencePage{}, parseErr
			}
			v.JobID = &jobID
		}
		if attempt.Valid {
			if attempt.Int64 > 1000000 || attempt.Int64 < 1 {
				return domain.TraceReferencePage{}, domain.ErrTraceRefUnavailable
			}
			n := int(attempt.Int64)
			v.JobAttempt = &n
		}
		v.RecordedAt = v.RecordedAt.UTC().Truncate(time.Millisecond)
		if err := v.Validate(); err != nil {
			return domain.TraceReferencePage{}, domain.ErrTraceRefUnavailable
		}
		page.Items = append(page.Items, v)
	}
	if err := rows.Err(); err != nil {
		return domain.TraceReferencePage{}, err
	}
	if len(page.Items) > f.Limit {
		page.Items = page.Items[:f.Limit]
		last := page.Items[len(page.Items)-1]
		cursor := (domain.InspectorCursor{At: last.RecordedAt, ID: last.ID}).Encode()
		page.NextCursor = &cursor
	}
	return page, nil
}
