package integration

import (
	"context"
	"encoding/json"
	"errors"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"testing"
	"time"
)

func TestHybridIndexSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	var major, fullText int
	if e := db.QueryRow(`SELECT CONVERT(int,SERVERPROPERTY('ProductMajorVersion')),CONVERT(int,SERVERPROPERTY('IsFullTextInstalled'))`).Scan(&major, &fullText); e != nil {
		t.Fatal(e)
	}
	t.Logf("SQL Server major=%d optional full-text installed=%d; active ranker=cosine/RRF; lexical index retained", major, fullText)
	f := aiSetup(t, db)
	r := wp12New(t, db)
	a := wp12Complete(t, f)
	wp12Enable(t, r, f.a)
	aiExec(t, db, `UPDATE dbo.ai_user_state SET locale=N'en' WHERE user_id=@p1`, f.a)
	ctx := context.Background()
	// No provider script is supplied. A successful write proves persistence is
	// independent of both the embedding provider and the worker being available.
	report, e := r.service.Apply(r.ctx(f.a, a.TurnID), f.a, []c.MemoryOpProposal{wp12Op(a.UserMessageID, "Synthetic work memory.")}, wp12Origin(r))
	if e != nil || len(report.Applied) != 1 {
		t.Fatal(report, e)
	}
	item := report.Applied[0]
	if len(r.provider.History()) != 0 {
		t.Fatal("memory persistence called a provider")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'memory_embed' AND JSON_VALUE(payload_json,'$.source_id')=CONVERT(varchar(36),@p2)`, f.a, item) != 1 {
		t.Fatal("write did not atomically queue index job")
	}
	// Helpers claim only a specified source's queued job, while the real queue
	// still owns attempts/tokens/recovery. Other synthetic source jobs are delayed.
	aiExec(t, db, `UPDATE dbo.ai_jobs SET run_after=DATEADD(day,1,SYSUTCDATETIME()) WHERE user_id=@p1`, f.a)
	claim := func(id c.ID) *mem.Lease {
		t.Helper()
		aiExec(t, db, `UPDATE dbo.ai_jobs SET run_after=DATEADD(day,1,SYSUTCDATETIME()) WHERE user_id=@p1 AND status=N'queued'`, f.a)
		aiExec(t, db, `UPDATE j SET run_after=DATEADD(second,-1,SYSUTCDATETIME()) FROM dbo.ai_jobs j JOIN dbo.ai_search_sources s ON s.user_id=j.user_id AND s.id=TRY_CONVERT(uniqueidentifier,JSON_VALUE(j.payload_json,'$.source_id')) AND s.source_kind=JSON_VALUE(j.payload_json,'$.source_kind') AND CONVERT(varchar(16),s.source_version,2)=JSON_VALUE(j.payload_json,'$.source_version') WHERE j.user_id=@p1 AND j.kind=N'memory_embed' AND j.status=N'queued' AND s.id=@p2`, f.a, id)
		l, e := r.repo.Claim(ctx, "hybrid-test", time.Minute, r.bundle.Reference(), 100)
		if e != nil || l == nil {
			var jobs string
			_ = db.QueryRow(`SELECT (SELECT kind,status,JSON_VALUE(payload_json,'$.source_kind') source_kind,JSON_VALUE(payload_json,'$.source_id') source_id,attempts,run_after FROM dbo.ai_jobs WHERE user_id=@p1 FOR JSON PATH)`, f.a).Scan(&jobs)
			t.Fatal("missing job", id, e, jobs)
		}
		return l
	}
	publish := func(l *mem.Lease) {
		t.Helper()
		in, ok, e := r.repo.EmbeddingInput(ctx, *l)
		if e != nil || !ok {
			t.Fatal("source input", ok, e, string(l.Payload))
		}
		response := ports.EmbeddingResponse{Model: "synthetic-embedding", Vector: []float32{1, 0, 0}}
		if e = r.repo.PublishEmbedding(ctx, *l, in, response); e != nil {
			wp12Err(t, e)
		}
	}
	l := claim(item)
	input, ok, e := r.repo.EmbeddingInput(ctx, *l)
	if e != nil || !ok || input.Text != "Synthetic work memory." {
		var current string
		var evidence int
		_ = db.QueryRow(`SELECT CONVERT(varchar(16),CONVERT(binary(8),search_version),2),(SELECT COUNT(*) FROM dbo.ai_memory_evidence WHERE item_id=@p1) FROM dbo.ai_memory_items WHERE id=@p1`, item).Scan(&current, &evidence)
		t.Fatal(input, ok, e, "job", string(l.Payload), l.Kind, "current_version", current, "evidence", evidence)
	}
	if e = r.repo.PublishLexical(ctx, *l, input); e != nil {
		wp12Err(t, e)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_search_lexical WHERE user_id=@p1 AND source_id=@p2`, f.a, item) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE user_id=@p1 AND ref_id=@p2`, f.a, item) != 0 {
		t.Fatal("lexical publication depends on embedding success")
	}
	if e = r.repo.Fail(ctx, *l, "memory_retry", 0); e != nil {
		wp12Err(t, e)
	}
	retried := claim(item)
	if retried.Attempts != 2 || retried.Token == l.Token {
		t.Fatal("retry did not fence original attempt")
	}
	if e = r.repo.PublishEmbedding(ctx, *l, input, ports.EmbeddingResponse{Model: "synthetic-embedding", Vector: []float32{1, 0, 0}}); !errors.Is(e, mem.ErrConflict) {
		t.Fatal("old worker published", e)
	}
	publish(retried)
	publish(retried) // duplicate execution is idempotent
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE user_id=@p1 AND ref_id=@p2`, f.a, item) != 1 {
		t.Fatal("idempotent index publish failed")
	}
	// Change the source while a model is in flight. Neither the old result nor
	// its already stored vector may be used for the changed record.
	aiExec(t, db, `UPDATE dbo.ai_memory_items SET content=N'Synthetic updated memory.',version=version+1,updated_at=SYSUTCDATETIME() WHERE id=@p1 AND user_id=@p2`, item, f.a)
	newer := claim(item)
	newInput, ok, e := r.repo.EmbeddingInput(ctx, *newer)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	aiExec(t, db, `UPDATE dbo.ai_memory_items SET content=N'Synthetic newest memory.',version=version+1,updated_at=SYSUTCDATETIME() WHERE id=@p1 AND user_id=@p2`, item, f.a)
	if e = r.repo.PublishEmbedding(ctx, *newer, newInput, ports.EmbeddingResponse{Model: "synthetic-embedding", Vector: []float32{1, 0, 0}}); e != nil {
		wp12Err(t, e)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings e JOIN dbo.ai_memory_items m ON m.id=e.ref_id AND m.user_id=e.user_id WHERE m.id=@p1 AND e.source_version=CONVERT(binary(8),m.search_version)`, item) != 0 {
		t.Fatal("stale result became searchable")
	}
	latest := claim(item)
	publish(latest)
	// Consent revision and deletion both fence model results after generation.
	aiExec(t, db, `UPDATE dbo.ai_memory_items SET updated_at=SYSUTCDATETIME() WHERE id=@p1`, item)
	revoke := claim(item)
	revInput, ok, e := r.repo.EmbeddingInput(ctx, *revoke)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	pause := true
	if e = r.service.SetControls(r.ctx(f.a, c.ID{}), f.a, &pause, nil, nil); e != nil {
		wp12Err(t, e)
	}
	if e = r.repo.PublishEmbedding(ctx, *revoke, revInput, ports.EmbeddingResponse{Model: "synthetic-embedding", Vector: []float32{1, 0, 0}}); e != nil {
		wp12Err(t, e)
	}
	wp12Enable(t, r, f.a)
	aiExec(t, db, `UPDATE dbo.ai_memory_items SET updated_at=SYSUTCDATETIME() WHERE id=@p1`, item)
	deleted := claim(item)
	delInput, ok, e := r.repo.EmbeddingInput(ctx, *deleted)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	if _, e = r.service.Delete(r.ctx(f.a, c.ID{}), f.a, ports.DeletionScope{ItemIDs: []c.ID{item}, Reason: "user_request"}); e != nil {
		wp12Err(t, e)
	}
	if e = r.repo.PublishEmbedding(ctx, *deleted, delInput, ports.EmbeddingResponse{Model: "synthetic-embedding", Vector: []float32{1, 0, 0}}); e != nil {
		wp12Err(t, e)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND content=N'Synthetic newest memory.'`, item) != 1 {
		t.Fatal("delete destroyed original content")
	}
	// Raw user and assistant messages are separate indexed sources.
	var assistant c.ID
	if e = db.QueryRow(`SELECT CONVERT(varchar(36),assistant_message_id) FROM dbo.turns WHERE id=@p1`, a.TurnID).Scan(&assistant); e != nil {
		t.Fatal(e)
	}
	for _, id := range []c.ID{a.UserMessageID, assistant} {
		l := claim(id)
		publish(l)
	}
	// Backfill covers preexisting records and is idempotent across batches.
	aiExec(t, db, `UPDATE dbo.ai_jobs SET status=N'completed' WHERE user_id=@p1 AND kind=N'memory_embed' AND status=N'queued'`, f.a)
	if e = r.repo.BackfillEmbeddings(ctx, 128); e != nil {
		wp12Err(t, e)
	}
	before := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1`, f.a)
	if e = r.repo.BackfillEmbeddings(ctx, 128); e != nil {
		wp12Err(t, e)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1`, f.a) != before {
		t.Fatal("backfill duplicated jobs")
	}
	// Model/dimension changes rebuild; old vectors are never compared.
	alt, e := aidb.NewMemoryRepository(db, r.cipher, wp12Policy{}, "new-synthetic-model", 2, mem.FeedbackPolicy{MinUnusedRetrievals: 3, ImportanceMultiplier: .9})
	if e != nil {
		t.Fatal(e)
	}
	alt.WithSearchPolicy(syntheticUsePolicy{true})
	if e = alt.BackfillEmbeddings(ctx, 128); e != nil {
		wp12Err(t, e)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1`, f.a) <= before {
		t.Fatal("new model was not backfilled")
	}
	// A rolling deployment must not let an old-model worker consume a backfill
	// job intended for a different model/dimension and mark it irrecoverable.
	aiExec(t, db, `UPDATE dbo.ai_jobs SET run_after=CASE WHEN JSON_VALUE(payload_json,'$.model')=N'new-synthetic-model' THEN DATEADD(second,-1,SYSUTCDATETIME()) ELSE DATEADD(day,1,SYSUTCDATETIME()) END WHERE kind=N'memory_embed' AND status=N'queued'`)
	if wrong, e := r.repo.Claim(ctx, "old-model", time.Minute, r.bundle.Reference(), 100); e != nil || wrong != nil {
		t.Fatal("old model consumed new-model backfill", e)
	}
	nextModel, e := alt.Claim(ctx, "new-model", time.Minute, r.bundle.Reference(), 100)
	if e != nil || nextModel == nil {
		t.Fatal("new-model backfill unavailable", e)
	}
	modelInput, eligible, e := alt.EmbeddingInput(ctx, *nextModel)
	if e != nil || !eligible {
		t.Fatal(eligible, e)
	}
	if e = alt.PublishEmbedding(ctx, *nextModel, modelInput, ports.EmbeddingResponse{Model: "new-synthetic-model", Vector: []float32{1, 0}}); e != nil {
		wp12Err(t, e)
	}
	// All persistent structures use the same corpus and provenance projection.
	thread, form, protocol, loop := c.NewID(), c.NewID(), c.NewID(), c.NewID()
	aiExec(t, db, `UPDATE dbo.ai_sessions SET summary=N'خلاصه گفتگو درباره کار' WHERE id=@p1 AND user_id=@p2`, a.SessionID, f.a)
	aiExec(t, db, `INSERT dbo.ai_threads(id,user_id,topic) VALUES(@p1,@p2,N'مدیریت کار')`, thread, f.a)
	aiExec(t, db, `INSERT dbo.ai_formulations(id,user_id,thread_id,version,framework,fields_json,paragraph) VALUES(@p1,@p2,@p3,1,N'synthetic',N'{}',N'شاید حجم کار زیاد باشد')`, form, f.a, thread)
	aiExec(t, db, `INSERT dbo.ai_protocol_instances(id,user_id,thread_id,protocol_id,protocol_version,status,last_user_summary) VALUES(@p1,@p2,@p3,N'synthetic',N'1',N'completed',N'تمرین برنامه‌ریزی کار')`, protocol, f.a, thread)
	aiExec(t, db, `INSERT dbo.ai_open_loops(id,user_id,kind,text) VALUES(@p1,@p2,N'synthetic',N'پیگیری برنامه کاری')`, loop, f.a)
	for _, sourceID := range []c.ID{a.SessionID, thread, form, protocol, loop} {
		publish(claim(sourceID))
	}
	// Protected memories use the same index and retain their sensitivity metadata.
	highOp := wp12Op(a.UserMessageID, "Synthetic encrypted fixture.")
	highOp.Sensitivity = c.SensHighly
	highReport, e := r.service.Apply(r.ctx(f.a, a.TurnID), f.a, []c.MemoryOpProposal{highOp}, wp12Origin(r))
	if e != nil || len(highReport.Applied) != 1 {
		t.Fatal(highReport, e)
	}
	high := highReport.Applied[0]
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND content IS NULL AND content_enc IS NOT NULL`, high) != 1 {
		t.Fatal("protected source stored in plaintext")
	}
	publish(claim(high))
	// A corrupt, rebuildable vector must never stop a valid memory confirmation.
	aiExec(t, db, `UPDATE dbo.ai_embeddings SET vector_bytes=CONVERT(varbinary(max),0x000000000000000000000000) WHERE user_id=@p1 AND ref_kind=N'memory' AND ref_id=@p2`, f.a, high)
	confirmed, e := r.service.Apply(r.ctx(f.a, a.TurnID), f.a, []c.MemoryOpProposal{highOp}, wp12Origin(r))
	if e != nil || len(confirmed.Applied) != 1 {
		t.Fatal("corrupt index blocked valid write", confirmed, e)
	}
	// New trigger establishes chronology after every source.
	trigger := admit(t, f, f.req())
	repo, e := aidb.NewRetrievalRepository(db, syntheticUsePolicy{true}, r.cipher)
	if e != nil {
		t.Fatal(e)
	}
	seen := map[retrieval.SourceKind]bool{}
	rawLex := map[c.ID]*retrieval.LexicalDocument{}
	if e = repo.Scan(ctx, retrieval.Query{UserID: f.a, TriggerID: trigger.UserMessageID, Model: "synthetic-embedding", Dimension: 3}, func(v retrieval.Candidate) error {
		if v.UserID != f.a || v.ID == item {
			t.Fatal("ownership/visibility leak")
		}
		seen[v.Source] = true
		rawLex[v.ID] = retrieval.BuildLexical(v.Text)
		if v.Source == retrieval.SourceMessage && len(v.Vector) == 0 {
			t.Fatal("raw message index absent", v.ID)
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	warm := 0
	if e = repo.ScanAuthorized(ctx, retrieval.Query{UserID: f.a, TriggerID: trigger.UserMessageID, Model: "synthetic-embedding", Dimension: 3, TermHashes: retrieval.QueryHashes("synthetic")}, func(v retrieval.Candidate) error {
		if v.Lexical != nil {
			warm++
			expected := rawLex[v.ID]
			if expected == nil || expected.Length != v.Lexical.Length {
				t.Fatal("compact document length changed")
			}
			for _, term := range retrieval.QueryHashes("synthetic") {
				if v.Lexical.Frequencies[term] != expected.Frequencies[term] {
					t.Fatal("SQL query-term projection changed lexical frequencies")
				}
			}
			if v.Text != "" || len(v.Lexical.Signature) != 32 {
				t.Fatal("compact corpus transferred raw content")
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if warm == 0 {
		t.Fatal("published lexical projection not used")
	}
	for _, source := range []retrieval.SourceKind{retrieval.SourceMemory, retrieval.SourceMessage, retrieval.SourceSession, retrieval.SourceThread, retrieval.SourceFormulation, retrieval.SourceProtocol, retrieval.SourceOpenLoop} {
		if !seen[source] {
			t.Fatal("source omitted", source)
		}
	}
	// Transaction rollback must also undo the metadata-only enqueue.
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	rolled := c.NewID()
	_, e = tx.ExecContext(ctx, `INSERT dbo.ai_open_loops(id,user_id,kind,text) VALUES(@p1,@p2,N'synthetic',N'rollback fixture')`, rolled, f.a)
	if e != nil {
		t.Fatal(e)
	}
	tx.Rollback()
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE JSON_VALUE(payload_json,'$.source_id')=CONVERT(varchar(36),@p1)`, rolled) != 0 {
		t.Fatal("enqueue escaped rollback")
	}
	// Completed jobs do not prevent rebuilding a missing/invalidated index.
	for pass := 0; pass < 2; pass++ {
		aiExec(t, db, `UPDATE dbo.ai_embeddings SET source_version=NULL WHERE user_id=@p1 AND ref_kind=N'message' AND ref_id=@p2 AND model_id=N'synthetic-embedding'`, f.a, a.UserMessageID)
		if e = r.repo.BackfillEmbeddings(ctx, 128); e != nil {
			wp12Err(t, e)
		}
		rebuild := claim(a.UserMessageID)
		publish(rebuild)
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE user_id=@p1 AND ref_id=@p2 AND model_id=N'synthetic-embedding' AND source_version IS NOT NULL`, f.a, a.UserMessageID) != 1 {
			t.Fatal("rebuild did not restore missing index")
		}
	}
	var task mem.EmbeddingTask
	if json.Unmarshal(latest.Payload, &task) != nil || task.SourceKind != "memory" {
		t.Fatal("source reference absent")
	}
}
