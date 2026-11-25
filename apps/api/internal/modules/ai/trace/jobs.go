package trace

import (
	"context"
	"database/sql"
	"encoding/json"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"time"
)

type turnKey struct{}

// TurnID carries job correlation even when payload tracing is disabled.
func TurnID(ctx context.Context) c.ID {
	if turn, ok := ctx.Value(turnKey{}).(c.ID); ok {
		return turn
	}
	sc := scopeFrom(ctx)
	if sc.session == nil {
		return c.ID{}
	}
	sc.session.mu.Lock()
	defer sc.session.mu.Unlock()
	return sc.session.doc.TurnID
}

// JobState is the last committed durable queue state. Attempt spans retain
// every execution's payload, usage, errors and timing, including failed retries.
type JobState struct {
	ID          c.ID            `json:"id"`
	Kind        string          `json:"kind"`
	Status      string          `json:"status"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	RunAfter    time.Time       `json:"run_after"`
	CreatedAt   time.Time       `json:"created_at"`
	LastError   string          `json:"last_error_code,omitempty"`
	Payload     json.RawMessage `json:"payload"`
}

// RefreshJobs is best-effort local inspection AFTER a transaction commits.
// It never changes queue state or pipeline outcomes. Serialize the SQL snapshot
// with the checkpoint so concurrent completions cannot overwrite newer state.
// Future scheduled work can reopen a complete checkpoint; completeness is an
// observation of the current queue, never a promise that no future job will exist.
func RefreshJobs(ctx context.Context, db *sql.DB, turn c.ID) {
	i := InspectorFromContext(ctx)
	if i == nil || !c.ValidID(turn) {
		return
	}
	i.mu.Lock()
	s := i.active[turn]
	if s == nil {
		s = i.loadTurnLocked(turn)
		if s != nil {
			i.active[turn] = s
		}
	}
	if s == nil {
		i.mu.Unlock()
		return
	}
	s.activeRoots++
	i.mu.Unlock()
	defer i.releaseRoot(s)
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := timeoutPolicy(ctx).JobRefresh
	if limit <= 0 {
		return
	}
	observe, cancel := context.WithTimeout(context.WithoutCancel(ctx), limit)
	defer cancel()
	rows, err := db.QueryContext(observe, `SELECT CONVERT(varchar(36),id),kind,status,attempts,max_attempts,run_after,created_at,COALESCE(last_error_code,N''),payload_json FROM dbo.ai_jobs WHERE turn_id=@p1 ORDER BY created_at,id`, turn)
	if err != nil {
		s.doc.BackgroundStatus = "unknown"
		s.checkpointLocked()
		return
	}
	defer rows.Close()
	jobs := []JobState{}
	for rows.Next() {
		var job JobState
		var payload string
		if err = rows.Scan(&job.ID, &job.Kind, &job.Status, &job.Attempts, &job.MaxAttempts, &job.RunAfter, &job.CreatedAt, &job.LastError, &payload); err != nil {
			break
		}
		job.Payload = snapshot(json.RawMessage(payload))
		jobs = append(jobs, job)
	}
	if err == nil {
		err = rows.Err()
	}
	if err != nil {
		s.doc.BackgroundStatus = "unknown"
	} else {
		s.doc.Jobs = jobs
		s.doc.BackgroundStatus = "complete"
	}
	s.checkpointLocked()
}
