package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

var _ mem.EmbeddingQueue = (*MemoryRepository)(nil)

// Backfill is bounded work, not a retrieval candidate limit. Repeated batches
// cover every source, including old rows and new models.
// Existing failed jobs require explicit RetryJob; cumulative attempts are retained.
func (r *MemoryRepository) BackfillEmbeddings(ctx context.Context, limit int) error {
	if limit < 1 || limit > 1000 {
		return mem.ErrDenied
	}
	_, e := r.db.ExecContext(ctx, `INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json)
 SELECT TOP(@p1) NEWID(),s.user_id,COALESCE(
 (SELECT m.turn_id FROM dbo.messages m JOIN dbo.turns t ON t.id=m.turn_id AND t.user_id=s.user_id WHERE s.source_kind=N'message' AND m.id=s.id),
 (SELECT TOP(1) prior.turn_id FROM dbo.ai_jobs prior WHERE prior.user_id=s.user_id AND prior.kind=N'memory_embed' AND prior.turn_id IS NOT NULL AND JSON_VALUE(prior.payload_json,'$.source_kind')=s.source_kind AND TRY_CONVERT(uniqueidentifier,JSON_VALUE(prior.payload_json,'$.source_id'))=s.id AND JSON_VALUE(prior.payload_json,'$.source_version')=CONVERT(varchar(16),s.source_version,2) ORDER BY prior.created_at,prior.id)),N'memory_embed',k.dedupe_key,
 (SELECT N'embedding_job.v2' [schema],@p2 model,@p3 dimension,s.source_kind,CONVERT(varchar(36),s.id) source_id,CONVERT(varchar(16),CONVERT(binary(8),s.source_version),2) source_version FOR JSON PATH,WITHOUT_ARRAY_WRAPPER)
 FROM dbo.ai_search_sources s JOIN dbo.ai_user_state us ON us.user_id=s.user_id JOIN dbo.users u ON u.id=s.user_id AND u.deleted_at IS NULL
 CROSS APPLY(SELECT CONCAT(N'backfill:',s.source_kind,N':',CONVERT(varchar(36),s.id),N':',CONVERT(varchar(16),CONVERT(binary(8),s.source_version),2),N':',CONVERT(varchar(64),HASHBYTES('SHA2_256',CONCAT(@p2,N':',@p3)),2),N':',us.version) dedupe_key) k
 WHERE (NOT EXISTS(SELECT 1 FROM dbo.ai_embeddings e WHERE e.user_id=s.user_id AND e.ref_kind=s.source_kind AND e.ref_id=s.id AND e.model_id=@p2 AND e.dim=@p3 AND e.source_version=s.source_version)
 OR NOT EXISTS(SELECT 1 FROM dbo.ai_search_lexical lex WHERE lex.user_id=s.user_id AND lex.source_kind=s.source_kind AND lex.source_id=s.id AND lex.source_version=s.source_version))
 AND NOT EXISTS(SELECT 1 FROM dbo.ai_jobs active_job WHERE active_job.user_id=s.user_id AND active_job.kind=N'memory_embed' AND active_job.status IN(N'queued',N'running') AND (JSON_VALUE(active_job.payload_json,'$.model') IS NULL OR (JSON_VALUE(active_job.payload_json,'$.model')=@p2 AND TRY_CONVERT(int,JSON_VALUE(active_job.payload_json,'$.dimension'))=@p3)) AND JSON_VALUE(active_job.payload_json,'$.source_kind')=s.source_kind AND TRY_CONVERT(uniqueidentifier,JSON_VALUE(active_job.payload_json,'$.source_id'))=s.id AND JSON_VALUE(active_job.payload_json,'$.source_version')=CONVERT(varchar(16),CONVERT(binary(8),s.source_version),2))
 AND NOT EXISTS(SELECT 1 FROM dbo.ai_jobs j WITH(UPDLOCK,HOLDLOCK) WHERE j.user_id=s.user_id AND j.kind=N'memory_embed' AND j.dedupe_key=k.dedupe_key AND j.status<>N'completed')
 ORDER BY s.source_version,s.source_kind,s.id`, limit, r.model, r.dimension)
	return memorySQLError(e)
}
func embeddingTask(l mem.Lease) (task mem.EmbeddingTask, err error) {
	if l.Kind != "memory_embed" || json.Unmarshal(l.Payload, &task) != nil || task.Schema != "embedding_job.v2" || !retrieval.SourceKind(task.SourceKind).Valid() || !c.ValidID(task.SourceID) || len(task.SourceVersion) != 16 {
		return task, mem.ErrDenied
	}
	return task, nil
}
func (r *MemoryRepository) EmbeddingInput(ctx context.Context, l mem.Lease) (input mem.EmbeddingInput, eligible bool, err error) {
	task, e := embeddingTask(l)
	if e != nil {
		return input, false, e
	}
	if task.Model != "" && (task.Model != r.model || task.Dimension != r.dimension) {
		return input, false, mem.ErrAssets
	}
	if r.searchUse == nil {
		return input, false, mem.ErrAssets
	}
	var user c.UserState
	user.UserID = l.UserID
	var consent string
	e = r.db.QueryRowContext(ctx, `SELECT us.version,us.consent_json FROM dbo.ai_user_state us JOIN dbo.users u ON u.id=us.user_id AND u.deleted_at IS NULL WHERE us.user_id=@p1`, l.UserID).Scan(&user.Version, &consent)
	if errors.Is(e, sql.ErrNoRows) {
		return input, false, nil
	}
	if e != nil {
		return input, false, e
	}
	if json.Unmarshal([]byte(consent), &user.Consent) != nil {
		return input, false, mem.ErrDenied
	}
	sources := &RetrievalRepository{r.db, r.searchUse, r.cipher}
	item, found, e := sources.source(ctx, l.UserID, retrieval.SourceKind(task.SourceKind), task.SourceID)
	if e != nil || !found {
		return input, false, e
	}
	if item.SourceVersion != task.SourceVersion {
		return input, false, nil
	}
	if item.Source == retrieval.SourceMemory && item.EvidenceN < 1 {
		return input, false, nil
	}
	for _, action := range []mem.Action{mem.Extract, mem.UseSource} {
		allowed, e := r.policy.Allow(ctx, mem.Permission{Action: action, User: user, TurnID: l.TurnID, SourceID: item.ID, SourceKind: task.SourceKind})
		if e != nil || !allowed {
			return input, false, e
		}
	}
	if item.Sensitivity == c.SensHighly {
		allowed, e := r.policy.Allow(ctx, mem.Permission{Action: mem.Sensitive, User: user, SourceID: item.ID, SourceKind: task.SourceKind})
		if e != nil || !allowed {
			return input, false, e
		}
	}
	return mem.EmbeddingInput{Text: item.Text, SourceVersion: item.SourceVersion, UserVersion: user.Version}, true, nil
}

// Publish performs no provider calls. The job token, lease, owner, current
// source revision and user-state revision are checked in the publishing transaction.
func (r *MemoryRepository) PublishEmbedding(ctx context.Context, l mem.Lease, input mem.EmbeddingInput, response ports.EmbeddingResponse) (err error) {
	defer func() {
		if err == nil {
			trace.RefreshJobs(ctx, r.db, l.TurnID)
		}
	}()
	defer func() { err = memorySQLError(err) }()
	task, e := embeddingTask(l)
	if e != nil {
		return e
	}
	if task.Model != "" && (task.Model != r.model || task.Dimension != r.dimension) {
		return mem.ErrAssets
	}
	if response.Model != r.model || len(response.Vector) != r.dimension {
		return errors.New("embedding model or dimension mismatch")
	}
	raw, e := EncodeEmbedding(response.Model, response.Vector)
	if e != nil {
		return e
	}
	fresh, eligible, e := r.EmbeddingInput(ctx, l)
	if e != nil {
		return e
	}
	if !eligible || fresh.SourceVersion != input.SourceVersion || fresh.UserVersion != input.UserVersion {
		return r.Complete(ctx, l)
	}
	tx, e := r.begin(ctx, l.UserID, l.TurnID)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	// Complete under the same transaction before touching the index: stale lease
	// owners cannot overwrite an embedding even when their model call succeeded.
	if e = completeMemoryJob(ctx, tx, l); e != nil {
		return e
	}
	var current string
	var version int64
	e = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(16),CONVERT(binary(8),s.source_version),2),us.version FROM dbo.ai_search_sources s WITH(HOLDLOCK) JOIN dbo.ai_user_state us WITH(HOLDLOCK) ON us.user_id=s.user_id WHERE s.user_id=@p1 AND s.source_kind=@p2 AND s.id=@p3`, l.UserID, task.SourceKind, task.SourceID).Scan(&current, &version)
	if errors.Is(e, sql.ErrNoRows) {
		return tx.Commit()
	}
	if e != nil {
		return e
	}
	if current != input.SourceVersion || version != input.UserVersion {
		return tx.Commit()
	}
	_, e = tx.ExecContext(ctx, `UPDATE dbo.ai_embeddings SET codec_version=1,dim=@p5,vector_bytes=@p6,source_version=CONVERT(binary(8),@p7,2),created_at=SYSUTCDATETIME() WHERE user_id=@p1 AND ref_kind=@p2 AND ref_id=@p3 AND model_id=@p4;
 IF @@ROWCOUNT=0 INSERT dbo.ai_embeddings(id,user_id,ref_kind,ref_id,model_id,dim,codec_version,vector_bytes,source_version) VALUES(@p8,@p1,@p2,@p3,@p4,@p5,1,@p6,CONVERT(binary(8),@p7,2))`, l.UserID, task.SourceKind, task.SourceID, r.model, r.dimension, raw, input.SourceVersion, c.NewID())
	if e != nil {
		return e
	}
	if e = publishLexicalRow(ctx, tx, l.UserID, task, input); e != nil {
		return e
	}
	return tx.Commit()
}
