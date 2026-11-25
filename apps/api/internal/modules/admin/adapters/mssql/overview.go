package mssql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

// OverviewRepository reads only bounded aggregates. It never reads messages,
// profiles, raw traces, prompts, local inspector files or credentials.
type OverviewRepository struct{ db *sql.DB }

func NewOverviewRepository(db *sql.DB) *OverviewRepository { return &OverviewRepository{db: db} }

func (r *OverviewRepository) ReadOverview(ctx context.Context, f domain.OverviewFilter) (domain.Overview, error) {
	if err := f.Validate(); err != nil {
		return domain.Overview{}, err
	}
	if r == nil || r.db == nil {
		return domain.Overview{}, errors.New("overview SQL unavailable")
	}
	result := domain.Overview{From: f.From.UTC(), To: f.To.UTC(), AsOf: f.AsOf.UTC()}
	// The turn counts are a current-state snapshot of v1 turns admitted
	// during the requested window, not a count of status transitions.
	err := r.db.QueryRowContext(ctx, `SELECT
        COUNT_BIG(CASE WHEN lifecycle_version=1 THEN 1 END),
        COUNT_BIG(CASE WHEN lifecycle_version=1 AND status=N'queued' THEN 1 END),
        COUNT_BIG(CASE WHEN lifecycle_version=1 AND status=N'running' THEN 1 END),
        COUNT_BIG(CASE WHEN lifecycle_version=1 AND status=N'completed' THEN 1 END),
        COUNT_BIG(CASE WHEN lifecycle_version=1 AND status=N'failed' THEN 1 END),
        COUNT_BIG(CASE WHEN lifecycle_version=1 AND status=N'cancelled' THEN 1 END),
        COUNT_BIG(CASE WHEN lifecycle_version=0 THEN 1 END),
        COUNT_BIG(CASE WHEN lifecycle_version=1 AND status=N'completed' AND
            completed_at IS NOT NULL AND completed_at>=created_at THEN 1 END)
        FROM dbo.turns WHERE created_at>=@p1 AND created_at<@p2`,
		f.From.UTC(), f.To.UTC()).Scan(
		&result.Turns.TotalV1, &result.Turns.Queued, &result.Turns.Running,
		&result.Turns.Completed, &result.Turns.Failed, &result.Turns.Cancelled,
		&result.Turns.ExcludedLegacy, &result.Turns.LatencySamples)
	if err != nil {
		return domain.Overview{}, err
	}

	if result.Turns.LatencySamples > 0 {
		var p95 sql.NullInt64
		err = r.db.QueryRowContext(ctx, `WITH completed AS (
            SELECT CAST(DATEDIFF_BIG(millisecond, created_at, completed_at) AS float) AS elapsed_ms
            FROM dbo.turns
            WHERE lifecycle_version=1 AND status=N'completed'
              AND created_at>=@p1 AND created_at<@p2
              AND completed_at IS NOT NULL AND completed_at>=created_at
        )
        SELECT TOP (1) CONVERT(bigint, ROUND(
            PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY elapsed_ms) OVER (), 0))
        FROM completed`, f.From.UTC(), f.To.UTC()).Scan(&p95)
		if err != nil {
			return domain.Overview{}, err
		}
		if p95.Valid {
			result.Turns.P95ElapsedMs = &p95.Int64
		}
	}

	var input, output, cached, cost sql.NullInt64
	var lastAttempt sql.NullTime
	err = r.db.QueryRowContext(ctx, `SELECT
        COUNT_BIG(*),
        COUNT_BIG(CASE WHEN status='started' THEN 1 END),
        COUNT_BIG(CASE WHEN status='succeeded' THEN 1 END),
        COUNT_BIG(CASE WHEN status='failed' THEN 1 END),
        COUNT_BIG(CASE WHEN status='incomplete' THEN 1 END),
        COUNT_BIG(input_tokens),
        COUNT_BIG(CASE WHEN cost_status='estimated' THEN 1 END),
        COUNT_BIG(CASE WHEN cost_status='unpriced' THEN 1 END),
        COUNT_BIG(CASE WHEN cost_status='unknown' THEN 1 END),
        SUM(input_tokens),SUM(output_tokens),SUM(cached_tokens),
        SUM(CASE WHEN cost_status='estimated' THEN cost_micro_usd END),
        MAX(started_at)
        FROM dbo.ai_llm_attempts WHERE started_at>=@p1 AND started_at<@p2`,
		f.From.UTC(), f.To.UTC()).Scan(
		&result.LLM.Attempts, &result.LLM.Started, &result.LLM.Succeeded,
		&result.LLM.Failed, &result.LLM.Incomplete, &result.LLM.UsageKnown,
		&result.LLM.EstimatedCostCount, &result.LLM.UnpricedCostCount,
		&result.LLM.UnknownCostCount, &input, &output, &cached, &cost, &lastAttempt)
	if err != nil {
		return domain.Overview{}, err
	}
	if input.Valid {
		result.LLM.InputTokens = &input.Int64
	}
	if output.Valid {
		result.LLM.OutputTokens = &output.Int64
	}
	if cached.Valid {
		result.LLM.CachedTokens = &cached.Int64
	}
	if cost.Valid {
		result.LLM.EstimatedMicroUSD = &cost.Int64
	}
	if lastAttempt.Valid {
		at := lastAttempt.Time.UTC()
		result.LLM.LastAttemptAt = &at
	}

	var lastStage sql.NullTime
	err = r.db.QueryRowContext(ctx, `SELECT COUNT_BIG(*),
        COUNT_BIG(CASE WHEN status='failed' THEN 1 END),
        COUNT_BIG(CASE WHEN status='degraded' THEN 1 END),
        COUNT_BIG(duration_ms),MAX(recorded_at)
        FROM dbo.ai_stage_events WHERE recorded_at>=@p1 AND recorded_at<@p2`,
		f.From.UTC(), f.To.UTC()).Scan(&result.Stages.Events, &result.Stages.Failed,
		&result.Stages.Degraded, &result.Stages.DurationSamples, &lastStage)
	if err != nil {
		return domain.Overview{}, err
	}
	if lastStage.Valid {
		at := lastStage.Time.UTC()
		result.Stages.LastEventAt = &at
	}

	// Job counts are the current queue state, not historical windowed events.
	err = r.db.QueryRowContext(ctx, `SELECT
        COUNT_BIG(CASE WHEN status=N'queued' THEN 1 END),
        COUNT_BIG(CASE WHEN status=N'running' THEN 1 END),
        COUNT_BIG(CASE WHEN status=N'failed' THEN 1 END),
        COUNT_BIG(CASE WHEN status=N'queued' AND run_after<=@p1 THEN 1 END),
        COUNT_BIG(CASE WHEN status=N'queued' AND run_after>@p1 THEN 1 END)
        FROM dbo.ai_jobs`, f.AsOf.UTC()).Scan(&result.Jobs.Queued,
		&result.Jobs.Running, &result.Jobs.Failed, &result.Jobs.DueNow,
		&result.Jobs.Scheduled)
	if err != nil {
		return domain.Overview{}, err
	}

	rows, err := r.db.QueryContext(ctx, `SELECT worker_kind,
        COUNT_BIG(CASE WHEN last_seen_at>=DATEADD(second,-45,@p1) AND last_seen_at<=@p1 AND status='running' THEN 1 END),
        COUNT_BIG(CASE WHEN last_seen_at>=DATEADD(second,-45,@p1) AND last_seen_at<=@p1 AND status='failed' THEN 1 END),
        COUNT_BIG(CASE WHEN last_seen_at>=DATEADD(second,-45,@p1) AND last_seen_at<=@p1 AND status='stopped' THEN 1 END),
        MAX(last_seen_at)
        FROM dbo.ai_worker_heartbeats GROUP BY worker_kind`, f.AsOf.UTC())
	if err != nil {
		return domain.Overview{}, err
	}
	byKind := make(map[string]domain.WorkerSummary, 3)
	for rows.Next() {
		var item domain.WorkerSummary
		var latest sql.NullTime
		if err = rows.Scan(&item.Kind, &item.FreshRunning, &item.FreshFailed, &item.FreshStopped, &latest); err != nil {
			break
		}
		if latest.Valid {
			at := latest.Time.UTC()
			item.LastSeenAt = &at
		}
		item.Status = "stale"
		switch {
		case item.FreshFailed > 0:
			item.Status = "failed"
		case item.FreshRunning > 0:
			item.Status = "running"
		case item.FreshStopped > 0:
			item.Status = "stopped"
		}
		byKind[item.Kind] = item
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return domain.Overview{}, err
	}
	result.Workers = make([]domain.WorkerSummary, 0, 3)
	for _, kind := range []string{"memory", "reflection", "scheduler"} {
		item, ok := byKind[kind]
		if !ok {
			item = domain.WorkerSummary{Kind: kind, Status: "unknown"}
		}
		result.Workers = append(result.Workers, item)
	}
	return result, nil
}

// Compile-time guard: the repository must supply an explicit approved reader.
var _ interface {
	ReadOverview(context.Context, domain.OverviewFilter) (domain.Overview, error)
} = (*OverviewRepository)(nil)
