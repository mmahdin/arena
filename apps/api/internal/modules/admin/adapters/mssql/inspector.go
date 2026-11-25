package mssql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

// InspectorRepository provides fixed, bounded, metadata-only SQL projections.
// Never SELECT messages, turn request/result JSON, job payloads, trace payloads,
// user contact details, or local inspector data in this repository.
type InspectorRepository struct{ db *sql.DB }

func NewInspectorRepository(db *sql.DB) *InspectorRepository { return &InspectorRepository{db: db} }
func (r *InspectorRepository) ready() error {
	if r == nil || r.db == nil {
		return errors.New("inspector SQL unavailable")
	}
	return nil
}
func uuidValue(s string) (uuid.UUID, error) { return uuid.Parse(s) }
func optionalTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	at := t.Time.UTC()
	return &at
}
func optionalInt(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}
func optionalInt32(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}
func optionalUUID(s sql.NullString) (*uuid.UUID, error) {
	if !s.Valid {
		return nil, nil
	}
	id, e := uuidValue(s.String)
	if e != nil {
		return nil, e
	}
	return &id, nil
}
func scanTurn(row interface{ Scan(...any) error }) (domain.TurnRecord, error) {
	var t domain.TurnRecord
	var id string
	var errorCode sql.NullString
	var completed, lease sql.NullTime
	err := row.Scan(&id, &t.Status, &t.AttemptCount, &errorCode, &t.CreatedAt, &completed, &lease)
	if err != nil {
		return t, err
	}
	t.ID, err = uuidValue(id)
	if err != nil {
		return t, err
	}
	t.Status = domain.SafeTurnStatus(t.Status)
	t.CreatedAt = t.CreatedAt.UTC()
	t.CompletedAt = optionalTime(completed)
	t.LeaseExpiresAt = optionalTime(lease)
	if errorCode.Valid {
		t.ErrorCode = domain.SafeCode(errorCode.String)
	}
	return t, nil
}

func (r *InspectorRepository) ListTurns(ctx context.Context, f domain.InspectorFilter) (domain.TurnPage, error) {
	if e := f.Validate(); e != nil {
		return domain.TurnPage{}, e
	}
	if e := r.ready(); e != nil {
		return domain.TurnPage{}, e
	}
	var cursorAt, cursorID any
	if f.Cursor != nil {
		cursorAt = f.Cursor.At.UTC()
		cursorID = f.Cursor.ID.String()
	}
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p1)
      CONVERT(varchar(36),id),status,attempt_count,error_code,created_at,completed_at,lease_expires_at
      FROM dbo.turns
      WHERE lifecycle_version=1 AND created_at>=@p2 AND created_at<@p3
        AND (@p4='' OR status=@p4)
        AND (@p5 IS NULL OR created_at<@p5 OR
             (created_at=@p5 AND id<CONVERT(uniqueidentifier,@p6)))
      ORDER BY created_at DESC,id DESC`, f.Limit+1, f.From.UTC(), f.To.UTC(), f.Status, cursorAt, cursorID)
	if err != nil {
		return domain.TurnPage{}, err
	}
	defer rows.Close()
	page := domain.TurnPage{Items: make([]domain.TurnRecord, 0, f.Limit+1)}
	for rows.Next() {
		t, e := scanTurn(rows)
		if e != nil {
			return domain.TurnPage{}, e
		}
		page.Items = append(page.Items, t)
	}
	if e := rows.Err(); e != nil {
		return domain.TurnPage{}, e
	}
	if len(page.Items) > f.Limit {
		page.Items = page.Items[:f.Limit]
		last := page.Items[len(page.Items)-1]
		next := (domain.InspectorCursor{At: last.CreatedAt, ID: last.ID}).Encode()
		page.NextCursor = &next
	}
	return page, nil
}

func (r *InspectorRepository) GetTurn(ctx context.Context, id uuid.UUID) (domain.TurnDetail, error) {
	if id == uuid.Nil {
		return domain.TurnDetail{}, domain.ErrInvalidInspector
	}
	if e := r.ready(); e != nil {
		return domain.TurnDetail{}, e
	}
	turn, err := scanTurn(r.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id),status,attempt_count,
       error_code,created_at,completed_at,lease_expires_at FROM dbo.turns
       WHERE id=CONVERT(uniqueidentifier,@p1) AND lifecycle_version=1`, id.String()))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TurnDetail{}, domain.ErrTurnNotFound
	}
	if err != nil {
		return domain.TurnDetail{}, err
	}
	d := domain.TurnDetail{Turn: turn, Stages: make([]domain.StageObservation, 0),
		LLMAttempts: make([]domain.LLMAttempt, 0), Jobs: make([]domain.JobRecord, 0), Warnings: make([]string, 0)}
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p2) CONVERT(varchar(36),id),stage,status,duration_ms,
       attempt,fallback_used,rule_id,recorded_at
       FROM dbo.ai_stage_events WHERE turn_id=CONVERT(uniqueidentifier,@p1)
       ORDER BY recorded_at DESC,id DESC`, id.String(), domain.InspectorStageLimit+1)
	if err != nil {
		return domain.TurnDetail{}, err
	}
	for rows.Next() {
		var s domain.StageObservation
		var sid string
		var duration, attempt sql.NullInt64
		var rule sql.NullString
		if err = rows.Scan(&sid, &s.Stage, &s.Status, &duration, &attempt, &s.FallbackUsed, &rule, &s.RecordedAt); err != nil {
			break
		}
		s.ID, err = uuidValue(sid)
		if err != nil {
			break
		}
		s.Stage = domain.SafeInspectorLabel(s.Stage)
		s.Status = domain.SafeStageStatus(s.Status)
		s.DurationMs = optionalInt(duration)
		s.Attempt = optionalInt32(attempt)
		if rule.Valid {
			s.RuleID = domain.SafeCode(rule.String)
		}
		s.RecordedAt = s.RecordedAt.UTC()
		d.Stages = append(d.Stages, s)
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return domain.TurnDetail{}, err
	}
	if len(d.Stages) > domain.InspectorStageLimit {
		d.Stages = d.Stages[:domain.InspectorStageLimit]
		d.StagesTruncated = true
	}

	rows, err = r.db.QueryContext(ctx, `SELECT TOP (@p2) CONVERT(varchar(36),id),kind,tier,provider,model,status,
       failure_kind,started_at,finished_at,duration_ms,input_tokens,output_tokens,cached_tokens,
       cost_status,cost_micro_usd FROM dbo.ai_llm_attempts
       WHERE turn_id=CONVERT(uniqueidentifier,@p1) ORDER BY started_at DESC,id DESC`, id.String(), domain.InspectorAttemptLimit+1)
	if err != nil {
		return domain.TurnDetail{}, err
	}
	for rows.Next() {
		var a domain.LLMAttempt
		var aid string
		var fail sql.NullString
		var done sql.NullTime
		var duration, input, output, cached, cost sql.NullInt64
		if err = rows.Scan(&aid, &a.Kind, &a.Tier, &a.Provider, &a.Model, &a.Status, &fail, &a.StartedAt,
			&done, &duration, &input, &output, &cached, &a.CostStatus, &cost); err != nil {
			break
		}
		a.ID, err = uuidValue(aid)
		if err != nil {
			break
		}
		a.StartedAt = a.StartedAt.UTC()
		a.FinishedAt = optionalTime(done)
		a.DurationMs = optionalInt(duration)
		a.InputTokens = optionalInt(input)
		a.OutputTokens = optionalInt(output)
		a.CachedTokens = optionalInt(cached)
		a.EstimatedMicroUsd = optionalInt(cost)
		if fail.Valid {
			a.FailureKind = domain.SafeCode(fail.String)
		}
		// Explicit reader-side validation protects even historical/direct
		// SQL writes that bypassed the BO-04 telemetry writer's checks.
		a.Kind = domain.SafeAttemptKind(a.Kind)
		a.Tier = domain.SafeInspectorLabel(a.Tier)
		a.Provider = domain.SafeInspectorLabel(a.Provider)
		a.Model = domain.SafeInspectorLabel(a.Model)
		a.Status = domain.SafeAttemptStatus(a.Status)
		a.CostStatus = domain.SafeCostStatus(a.CostStatus)
		d.LLMAttempts = append(d.LLMAttempts, a)
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return domain.TurnDetail{}, err
	}
	if len(d.LLMAttempts) > domain.InspectorAttemptLimit {
		d.LLMAttempts = d.LLMAttempts[:domain.InspectorAttemptLimit]
		d.AttemptsTruncated = true
	}

	rows, err = r.db.QueryContext(ctx, `SELECT TOP (@p2) CONVERT(varchar(36),id),
       CONVERT(varchar(36),turn_id),kind,status,attempts,max_attempts,last_error_code,
       run_after,created_at,memory_lease_until
       FROM dbo.ai_jobs WHERE turn_id=CONVERT(uniqueidentifier,@p1)
       ORDER BY created_at DESC,id DESC`, id.String(), domain.InspectorRelatedJobLimit+1)
	if err != nil {
		return domain.TurnDetail{}, err
	}
	for rows.Next() {
		j, e := scanJob(rows)
		if e != nil {
			err = e
			break
		}
		d.Jobs = append(d.Jobs, j)
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return domain.TurnDetail{}, err
	}
	if len(d.Jobs) > domain.InspectorRelatedJobLimit {
		d.Jobs = d.Jobs[:domain.InspectorRelatedJobLimit]
		d.JobsTruncated = true
	}
	if len(d.Stages) == 0 {
		d.Warnings = append(d.Warnings, "NO_STAGE_OBSERVATIONS")
	}
	if d.StagesTruncated || d.AttemptsTruncated || d.JobsTruncated {
		d.Warnings = append(d.Warnings, "INSPECTOR_RESULTS_TRUNCATED")
	}
	d.Warnings = append(d.Warnings, "BEST_EFFORT_TELEMETRY_NOT_FULL_TRACE")
	return d, nil
}

func scanJob(row interface{ Scan(...any) error }) (domain.JobRecord, error) {
	var j domain.JobRecord
	var id string
	var turn, code sql.NullString
	var lease sql.NullTime
	err := row.Scan(&id, &turn, &j.Kind, &j.Status, &j.Attempts, &j.MaxAttempts, &code, &j.RunAfter, &j.CreatedAt, &lease)
	if err != nil {
		return j, err
	}
	j.ID, err = uuidValue(id)
	if err != nil {
		return j, err
	}
	j.TurnID, err = optionalUUID(turn)
	if err != nil {
		return j, err
	}
	j.CreatedAt = j.CreatedAt.UTC()
	j.RunAfter = j.RunAfter.UTC()
	j.LeaseExpiresAt = optionalTime(lease)
	if code.Valid {
		j.LastErrorCode = domain.SafeCode(code.String)
	}
	if !domain.ValidJobKind(j.Kind) {
		j.Kind = "unclassified"
	}
	if !domain.ValidInspectorStatus(j.Status) {
		j.Status = "unclassified"
	}
	return j, nil
}
func (r *InspectorRepository) ListJobs(ctx context.Context, f domain.InspectorFilter) (domain.JobPage, error) {
	if e := f.Validate(); e != nil {
		return domain.JobPage{}, e
	}
	if e := r.ready(); e != nil {
		return domain.JobPage{}, e
	}
	var cursorAt, cursorID any
	if f.Cursor != nil {
		cursorAt = f.Cursor.At.UTC()
		cursorID = f.Cursor.ID.String()
	}
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p1) CONVERT(varchar(36),id),CONVERT(varchar(36),turn_id),
       kind,status,attempts,max_attempts,last_error_code,run_after,created_at,memory_lease_until
       FROM dbo.ai_jobs WHERE created_at>=@p2 AND created_at<@p3
       AND (@p4='' OR status=@p4) AND (@p5='' OR kind=@p5)
       AND (@p6 IS NULL OR created_at<@p6 OR (created_at=@p6 AND id<CONVERT(uniqueidentifier,@p7)))
       ORDER BY created_at DESC,id DESC`, f.Limit+1, f.From.UTC(), f.To.UTC(), f.Status, f.Kind, cursorAt, cursorID)
	if err != nil {
		return domain.JobPage{}, err
	}
	defer rows.Close()
	page := domain.JobPage{Items: make([]domain.JobRecord, 0, f.Limit+1)}
	for rows.Next() {
		j, e := scanJob(rows)
		if e != nil {
			return domain.JobPage{}, e
		}
		page.Items = append(page.Items, j)
	}
	if err = rows.Err(); err != nil {
		return domain.JobPage{}, err
	}
	if len(page.Items) > f.Limit {
		page.Items = page.Items[:f.Limit]
		last := page.Items[len(page.Items)-1]
		next := (domain.InspectorCursor{At: last.CreatedAt, ID: last.ID}).Encode()
		page.NextCursor = &next
	}
	return page, nil
}

func (r *InspectorRepository) ListWorkers(ctx context.Context, asOf time.Time) (domain.WorkerPage, error) {
	if asOf.IsZero() {
		return domain.WorkerPage{}, domain.ErrInvalidInspector
	}
	if e := r.ready(); e != nil {
		return domain.WorkerPage{}, e
	}
	// Limit to the last seven days to prevent historical instance accumulation
	// from making the diagnostic response unbounded.
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p1) CONVERT(varchar(36),instance_id),worker_kind,status,last_seen_at
      FROM dbo.ai_worker_heartbeats
      WHERE last_seen_at>=DATEADD(day,-7,@p2) AND last_seen_at<=@p2
      ORDER BY last_seen_at DESC,instance_id DESC,worker_kind DESC`, domain.InspectorWorkerLimit+1, asOf.UTC())
	if err != nil {
		return domain.WorkerPage{}, err
	}
	defer rows.Close()
	out := domain.WorkerPage{AsOf: asOf.UTC(), Instances: make([]domain.WorkerInstance, 0), Warnings: make([]string, 0)}
	for rows.Next() {
		var w domain.WorkerInstance
		var id string
		if err = rows.Scan(&id, &w.Kind, &w.Status, &w.LastSeenAt); err != nil {
			return domain.WorkerPage{}, err
		}
		w.InstanceID, err = uuidValue(id)
		if err != nil {
			return domain.WorkerPage{}, err
		}
		w.Kind = domain.SafeWorkerKind(w.Kind)
		w.LastSeenAt = w.LastSeenAt.UTC()
		w.AgeSeconds = int64(asOf.Sub(w.LastSeenAt).Seconds())
		if w.AgeSeconds > 45 {
			w.Status = "stale"
		} else if w.Status != "running" && w.Status != "failed" && w.Status != "stopped" {
			w.Status = "unknown"
		}
		out.Instances = append(out.Instances, w)
	}
	if err = rows.Err(); err != nil {
		return domain.WorkerPage{}, err
	}
	if len(out.Instances) > domain.InspectorWorkerLimit {
		out.Instances = out.Instances[:domain.InspectorWorkerLimit]
		out.Truncated = true
		out.Warnings = append(out.Warnings, "WORKER_LIST_TRUNCATED")
	}
	if len(out.Instances) == 0 {
		out.Warnings = append(out.Warnings, "NO_RECENT_HEARTBEATS")
	}
	out.Warnings = append(out.Warnings, "HEARTBEAT_ONLY_NOT_JOB_PROGRESS")
	return out, nil
}
