package integration

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/embedding"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func TestSharedEmbeddingSalamRetrievalAndMemoryWorkerSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	core, provider := inspectorCore(t, f)
	t.Cleanup(core.Embeddings.Close)
	req := f.req()
	req.Text = "سلام"
	req.Locale = "fa"
	req.TZ = "Asia/Tehran"
	result, e := core.Turns.ProcessTurn(context.Background(), req)
	if e != nil {
		t.Fatal(e)
	}
	aiExec(t, db, `UPDATE dbo.ai_user_state SET consent_json=N'{"synthetic_memory":true}' WHERE user_id=@p1`, f.a)
	// Make only the current user-message job due. Worker eligibility, lease,
	// source-version checks and the real SQL publisher still execute normally.
	aiExec(t, db, `UPDATE dbo.ai_jobs SET run_after=DATEADD(day,1,SYSUTCDATETIME()) WHERE user_id=@p1;
 UPDATE dbo.ai_jobs SET run_after=DATEADD(second,-1,SYSUTCDATETIME()) WHERE user_id=@p1 AND kind=N'memory_embed' AND TRY_CONVERT(uniqueidentifier,JSON_VALUE(payload_json,'$.source_id'))=@p2`, f.a, result.UserMessageID)
	count := func() int {
		n := 0
		for _, call := range provider.History() {
			if call.Operation == "embedding" {
				n++
			}
		}
		return n
	}
	if count() != 1 {
		t.Fatal("first retrieval should compute exactly one uncontextualized query", count())
	}
	worked, e := core.MemoryWorker.RunOnce(context.Background())
	if !worked || e != nil {
		t.Fatal(worked, e)
	}
	if count() != 1 {
		t.Fatal("Memory Worker recomputed سلام instead of using Retrieval's shared cache", count())
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE user_id=@p1 AND ref_kind=N'message' AND ref_id=@p2 AND model_id=N'synthetic-embedding'`, f.a, result.UserMessageID); n != 1 {
		t.Fatal("cache hit bypassed or lost durable publication", n)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'memory_embed' AND TRY_CONVERT(uniqueidentifier,JSON_VALUE(payload_json,'$.source_id'))=@p2 AND status=N'completed'`, f.a, result.UserMessageID); n != 1 {
		t.Fatal("worker did not complete its durable job", n)
	}
	t.Log("سلام: actual Retrieval plus Memory Worker used one provider embedding and persisted one indexed vector")
}

type sharedSQLClient struct {
	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (p *sharedSQLClient) Embed(ctx context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	p.calls.Add(1)
	close(p.entered)
	select {
	case <-p.release:
		return ports.EmbeddingResponse{Model: "synthetic-embedding", Vector: []float32{1, 0, 0}}, nil
	case <-ctx.Done():
		return ports.EmbeddingResponse{}, ctx.Err()
	}
}

func TestSharedEmbeddingStaleSourceCannotPublishSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	r := wp12New(t, db)
	a := wp12Complete(t, f)
	wp12Enable(t, r, f.a)
	aiExec(t, db, `UPDATE dbo.ai_jobs SET run_after=DATEADD(day,1,SYSUTCDATETIME()) WHERE user_id=@p1;
 UPDATE dbo.ai_jobs SET run_after=DATEADD(second,-1,SYSUTCDATETIME()) WHERE user_id=@p1 AND kind=N'memory_embed' AND TRY_CONVERT(uniqueidentifier,JSON_VALUE(payload_json,'$.source_id'))=@p2`, f.a, a.UserMessageID)
	l, e := r.repo.Claim(context.Background(), "shared-stale", time.Minute, r.bundle.Reference(), 100)
	if e != nil || l == nil {
		t.Fatal(e)
	}
	input, eligible, e := r.repo.EmbeddingInput(context.Background(), *l)
	if e != nil || !eligible {
		t.Fatal(input, eligible, e)
	}
	provider := &sharedSQLClient{entered: make(chan struct{}), release: make(chan struct{})}
	shared, e := embedding.New(context.Background(), provider, embedding.Options{Profile: ports.EmbeddingProfile{Models: []ports.EmbeddingModel{{Provider: "synthetic", Model: "synthetic-embedding", Dimensions: 3}}, ExecutionTimeout: time.Second}})
	if e != nil {
		t.Fatal(e)
	}
	defer shared.Close()
	done := make(chan ports.EmbeddingResponse, 1)
	fail := make(chan error, 1)
	go func() {
		v, e := shared.Embed(context.Background(), ports.EmbeddingRequest{Tier: ports.ComponentEmbedding, Input: input.Text, Meta: c.CallMeta{UserID: f.a, TurnID: a.TurnID}})
		done <- v
		fail <- e
	}()
	<-provider.entered
	aiExec(t, db, `UPDATE dbo.messages SET content=N'Synthetic revised primary source' WHERE id=@p1`, a.UserMessageID)
	close(provider.release)
	response := <-done
	if e = <-fail; e != nil {
		t.Fatal(e)
	}
	if e = r.repo.PublishEmbedding(context.Background(), *l, input, response); e != nil {
		t.Fatal(e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE user_id=@p1 AND ref_id=@p2`, f.a, a.UserMessageID); n != 0 {
		t.Fatal("cached/shared vector bypassed source-version fence", n)
	}
	if provider.calls.Load() != 1 {
		t.Fatal("unexpected provider executions")
	}
}

func TestSharedEmbeddingMemoryWorkerBeforeRetrievalSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	req := f.req()
	req.Text = "سلام"
	req.Locale = "fa"
	req.TZ = "Asia/Tehran"
	a, e := f.store.AdmitUserMessage(context.Background(), req)
	if e != nil {
		t.Fatal(e)
	}
	p := prepared(a)
	p.Jobs = nil
	if _, e = f.store.Complete(context.Background(), a, p); e != nil {
		t.Fatal(e)
	}
	var retriever ports.Retriever
	core, provider := inspectorCore(t, f, &retriever)
	t.Cleanup(core.Embeddings.Close)
	aiExec(t, db, `UPDATE dbo.ai_user_state SET consent_json=N'{"synthetic_memory":true}' WHERE user_id=@p1;
 UPDATE dbo.ai_jobs SET run_after=DATEADD(day,1,SYSUTCDATETIME()) WHERE user_id=@p1;
 UPDATE dbo.ai_jobs SET run_after=DATEADD(second,-1,SYSUTCDATETIME()) WHERE user_id=@p1 AND kind=N'memory_embed' AND TRY_CONVERT(uniqueidentifier,JSON_VALUE(payload_json,'$.source_id'))=@p2`, f.a, a.UserMessageID)
	if worked, e := core.MemoryWorker.RunOnce(context.Background()); !worked || e != nil {
		t.Fatal(worked, e)
	}
	if _, e = retriever.Retrieve(context.Background(), c.RetrievalRequest{UserID: f.a, Turn: c.TurnView{TurnID: a.TurnID, MessageID: a.UserMessageID, Text: "سلام", Lang: "fa"}, MaxItems: 4}); e != nil {
		t.Fatal(e)
	}
	calls := 0
	for _, call := range provider.History() {
		if call.Operation == "embedding" {
			calls++
		}
	}
	if calls != 1 {
		t.Fatal("Retrieval failed to reuse Worker-first computation", calls)
	}
}
