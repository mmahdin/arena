package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func publishLexicalRow(ctx context.Context, tx *sql.Tx, user c.ID, task mem.EmbeddingTask, input mem.EmbeddingInput) error {
	doc := retrieval.BuildLexical(input.Text)
	raw, _ := json.Marshal(doc.Frequencies)
	_, err := tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_search_lexical WITH(UPDLOCK,HOLDLOCK) WHERE user_id=@p1 AND source_kind=@p2 AND source_id=@p3 AND source_version=CONVERT(binary(8),@p4,2))
 INSERT dbo.ai_search_lexical(user_id,source_kind,source_id,source_version,doc_length,content_key,signature,terms_json) VALUES(@p1,@p2,@p3,CONVERT(binary(8),@p4,2),@p5,@p6,@p7,@p8)`, user, task.SourceKind, task.SourceID, input.SourceVersion, doc.Length, doc.Identity, doc.Signature, string(raw))
	return err
}

// PublishLexical runs BEFORE an embedding provider request. Its success is
// independent of that provider, and a retry cannot overwrite a newer revision.
func (r *MemoryRepository) PublishLexical(ctx context.Context, l mem.Lease, input mem.EmbeddingInput) (err error) {
	defer func() { err = memorySQLError(err) }()
	task, e := embeddingTask(l)
	if e != nil {
		return e
	}
	fresh, eligible, e := r.EmbeddingInput(ctx, l)
	if e != nil {
		return e
	}
	if !eligible || fresh.SourceVersion != input.SourceVersion || fresh.UserVersion != input.UserVersion {
		return nil
	}
	tx, e := r.begin(ctx, l.UserID)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var n int
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_jobs WITH(UPDLOCK,HOLDLOCK) WHERE id=@p1 AND user_id=@p2 AND memory_lease_token=@p3 AND status=N'running' AND memory_lease_until>SYSUTCDATETIME()`, l.ID, l.UserID, l.Token).Scan(&n)
	if e != nil {
		return e
	}
	if n != 1 {
		return mem.ErrConflict
	}
	var sourceVersion string
	var userVersion int64
	e = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(16),CONVERT(binary(8),s.source_version),2),us.version FROM dbo.ai_search_sources s WITH(HOLDLOCK) JOIN dbo.ai_user_state us WITH(HOLDLOCK) ON us.user_id=s.user_id WHERE s.user_id=@p1 AND s.source_kind=@p2 AND s.id=@p3`, l.UserID, task.SourceKind, task.SourceID).Scan(&sourceVersion, &userVersion)
	if errors.Is(e, sql.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	if sourceVersion != input.SourceVersion || userVersion != input.UserVersion {
		return nil
	}
	if e = publishLexicalRow(ctx, tx, l.UserID, task, input); e != nil {
		return e
	}
	return tx.Commit()
}
