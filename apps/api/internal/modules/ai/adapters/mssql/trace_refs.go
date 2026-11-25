package mssql

import (
	"context"
	"database/sql"
	"errors"
	"regexp"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

// TraceReferenceRepository writes only OTel root IDs and established Turn/Job
// UUIDs. It is independent from the encrypted/full-payload Inspector and the
// durable AI attempt usage ledger.
type TraceReferenceRepository struct{ db *sql.DB }

func NewTraceReferenceRepository(db *sql.DB) *TraceReferenceRepository {
	return &TraceReferenceRepository{db: db}
}

var hexTraceID = regexp.MustCompile(`^[0-9a-f]{32}$`)
var hexRootSpanID = regexp.MustCompile(`^[0-9a-f]{16}$`)

func (r *TraceReferenceRepository) WriteTraceReference(ctx context.Context, v trace.RootReference) error {
	if r == nil || r.db == nil {
		return errors.New("trace reference SQL unavailable")
	}
	if v.TurnID == uuid.Nil || !hexTraceID.MatchString(v.TraceID) || !hexRootSpanID.MatchString(v.RootSpanID) ||
		v.TraceID == "00000000000000000000000000000000" || v.RootSpanID == "0000000000000000" {
		return errors.New("invalid trace reference")
	}
	var jobID any
	var attempt any
	if v.Kind == "turn" {
		if v.JobID != uuid.Nil || v.JobAttempt != 0 {
			return errors.New("invalid turn reference")
		}
	} else if v.Kind == "memory" || v.Kind == "reflection" {
		if v.JobID == uuid.Nil || v.JobAttempt < 1 || v.JobAttempt > 1000000 {
			return errors.New("invalid job reference")
		}
		jobID, attempt = v.JobID.String(), v.JobAttempt
	} else {
		return errors.New("invalid trace kind")
	}
	// The source must exist. A job must also belong to the reported Turn.
	// The foreign keys and fixed INSERT-SELECT enforce this without fetching
	// private message/user/payload columns into the application.
	result, err := r.db.ExecContext(ctx, `INSERT INTO dbo.ai_trace_refs
        (id,turn_id,kind,job_id,job_attempt,trace_id,root_span_id)
        SELECT CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),@p3,
               CONVERT(uniqueidentifier,@p4),@p5,@p6,@p7
        WHERE EXISTS (SELECT 1 FROM dbo.turns
                      WHERE id=CONVERT(uniqueidentifier,@p2) AND lifecycle_version=1)
          AND (@p4 IS NULL OR EXISTS (SELECT 1 FROM dbo.ai_jobs
                      WHERE id=CONVERT(uniqueidentifier,@p4)
                        AND turn_id=CONVERT(uniqueidentifier,@p2)))`,
		uuid.NewString(), v.TurnID.String(), v.Kind, jobID, attempt, v.TraceID, v.RootSpanID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("trace reference source not found")
	}
	return nil
}
