package mssql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/eval"
)

// TraceInspectionPolicy is a deliberately required authorization dependency.
// Hosts must explicitly approve this *metadata-only* access; it does not imply
// consent to use raw data for evaluation or to decrypt retained payloads.
type TraceInspectionPolicy interface {
	AllowTraceInspection(context.Context, contract.ID, contract.ID) (bool, error)
}

type RuntimeTraceInspector struct {
	db     *sql.DB
	policy TraceInspectionPolicy
}

func NewRuntimeTraceInspector(db *sql.DB, policy TraceInspectionPolicy) (*RuntimeTraceInspector, error) {
	if db == nil || policy == nil {
		return nil, eval.ErrReplay
	}
	return &RuntimeTraceInspector{db: db, policy: policy}, nil
}

// Inspect returns only replay-readiness codes. It cannot expose even a trace
// summary unless the caller's already-authenticated owner and the separate
// inspection policy both authorize the read. Hidden conversations are excluded;
// soft-deleted records remain retained and untouched in SQL Server.
func (r *RuntimeTraceInspector) Inspect(ctx context.Context, owner, turn contract.ID) (eval.RuntimeTraceCoverage, error) {
	if err := ctx.Err(); err != nil {
		return eval.RuntimeTraceCoverage{}, err
	}
	if !contract.ValidID(owner) || !contract.ValidID(turn) {
		return eval.RuntimeTraceCoverage{}, eval.ErrReplay
	}
	allowed, err := r.policy.AllowTraceInspection(ctx, owner, turn)
	if err != nil || !allowed {
		return eval.RuntimeTraceCoverage{}, eval.ErrReplay
	}
	// The query NEVER selects payload_enc, summary_json, or request_meta_json;
	// only existence/metadata flags cross this boundary. The user and turn
	// ownership are checked by joined SQL records, not by the policy alone.
	var hasBundle, hasVersions, hasCiphertext int
	err = r.db.QueryRowContext(ctx, `SELECT
	 CASE WHEN LEN(JSON_VALUE(t.summary_json, '$.bundle_version'))>0 THEN 1 ELSE 0 END,
	 CASE WHEN LEN(JSON_VALUE(t.versions_json, '$.models.brain'))>0
	       AND LEN(JSON_VALUE(t.versions_json, '$.models.generator'))>0
	       AND LEN(JSON_VALUE(t.versions_json, '$.prompts.brain'))>0
	       AND LEN(JSON_VALUE(t.versions_json, '$.prompts.generator'))>0 THEN 1 ELSE 0 END,
	 CASE WHEN DATALENGTH(t.payload_enc)>0 THEN 1 ELSE 0 END
	 FROM dbo.ai_turn_traces t
	 JOIN dbo.turns tr ON tr.id=t.turn_id AND tr.user_id=t.user_id AND tr.status=N'completed' AND tr.lifecycle_version=1
	 JOIN dbo.conversations cv ON cv.id=tr.conversation_id AND cv.user_id=tr.user_id AND cv.deleted_at IS NULL
	 WHERE t.user_id=@p1 AND t.turn_id=@p2`, owner, turn).Scan(&hasBundle, &hasVersions, &hasCiphertext)
	if err != nil {
		// Conceal nonexistence and cross-owner/deleted visibility alike. Neither
		// raw SQL errors nor retained ciphertext are attached to the error.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return eval.RuntimeTraceCoverage{}, err
		}
		return eval.RuntimeTraceCoverage{}, eval.ErrReplay
	}
	return eval.CheckRuntimeCoverage(hasBundle != 0, hasVersions != 0, hasCiphertext != 0), nil
}
