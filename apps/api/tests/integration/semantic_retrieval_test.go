package integration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/app/composition"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type semanticQueryClient struct{ fail bool }

func (p *semanticQueryClient) Embed(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	if p.fail {
		return ports.EmbeddingResponse{}, errors.New("synthetic semantic outage")
	}
	return ports.EmbeddingResponse{Model: "synthetic-embedding", Vector: []float32{1, 0, 0}}, nil
}

func TestSemanticRetrievalPendingStaleAndDeletedSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	r := wp12New(t, db)
	a := wp12Complete(t, f)
	wp12Enable(t, r, f.a)
	create := func(text string) c.ID {
		t.Helper()
		report, e := r.service.Apply(r.ctx(f.a, a.TurnID), f.a, []c.MemoryOpProposal{wp12Op(a.UserMessageID, text)}, wp12Origin(r))
		if e != nil || len(report.Applied) != 1 {
			t.Fatal(report, e)
		}
		return report.Applied[0]
	}
	pending := create("keyword keyword keyword")
	semantic := create("Independent synthetic evidence about a fictional scene")
	claim := func(id c.ID) *mem.Lease {
		t.Helper()
		aiExec(t, db, `UPDATE dbo.ai_jobs SET run_after=DATEADD(day,1,SYSUTCDATETIME()) WHERE user_id=@p1;
 UPDATE dbo.ai_jobs SET run_after=DATEADD(second,-1,SYSUTCDATETIME()) WHERE user_id=@p1 AND kind=N'memory_embed' AND status=N'queued' AND TRY_CONVERT(uniqueidentifier,JSON_VALUE(payload_json,'$.source_id'))=@p2`, f.a, id)
		l, e := r.repo.Claim(context.Background(), "semantic-test", time.Minute, r.bundle.Reference(), 100)
		if e != nil || l == nil {
			t.Fatal(e)
		}
		return l
	}
	l := claim(pending)
	input, ok, e := r.repo.EmbeddingInput(context.Background(), *l)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = r.repo.PublishLexical(context.Background(), *l, input); e != nil {
		t.Fatal(e)
	}
	if e = r.repo.Fail(context.Background(), *l, "memory_retry", 0); e != nil {
		t.Fatal(e)
	}
	l = claim(semantic)
	input, ok, e = r.repo.EmbeddingInput(context.Background(), *l)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = r.repo.PublishEmbedding(context.Background(), *l, input, ports.EmbeddingResponse{Model: "synthetic-embedding", Vector: []float32{1, 0, 0}}); e != nil {
		t.Fatal(e)
	}
	client := &semanticQueryClient{}
	ret, e := composition.NewAIRetrieval(db, client, wp14Registry{internalBundle(t)}, syntheticUsePolicy{true}, r.cipher)
	if e != nil {
		t.Fatal(e)
	}
	query := func(trigger c.Admission) c.RetrievalResult {
		t.Helper()
		out, e := ret.Retrieve(context.Background(), c.RetrievalRequest{UserID: f.a, Background: true, Turn: c.TurnView{TurnID: trigger.TurnID, MessageID: trigger.UserMessageID, Text: "keyword", Lang: "en"}, MaxItems: 5})
		if e != nil {
			t.Fatal(e)
		}
		for _, s := range out.Meta.Strategies {
			if strings.Contains(s, "bm25") {
				t.Fatal("disabled strategy", out.Meta)
			}
		}
		return out
	}
	trigger := wp12Complete(t, f)
	out := query(trigger)
	if len(out.Items) == 0 || out.Items[0].ID != semantic || out.Meta.Degraded {
		t.Fatal("pending lexical-only source displaced semantic result", out)
	}
	for _, item := range out.Items {
		if item.ID == pending {
			t.Fatal("pending source returned as a lexical match", out)
		}
		if item.ID != semantic && (item.ExpandedFrom == nil || *item.ExpandedFrom != semantic) {
			t.Fatal("unexpected nonsemantic result", out)
		}
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_search_lexical WHERE user_id=@p1 AND source_id=@p2`, f.a, pending) != 1 {
		t.Fatal("lexical infrastructure was removed")
	}
	client.fail = true
	out = query(trigger)
	if len(out.Items) != 0 || !out.Meta.Degraded || len(out.Meta.Strategies) != 0 || len(out.Meta.FailedStrategies) != 1 {
		t.Fatal("embedding outage activated lexical fallback", out)
	}
	client.fail = false
	aiExec(t, db, `UPDATE dbo.ai_memory_items SET content=N'Revised independent synthetic evidence',version=version+1 WHERE user_id=@p1 AND id=@p2`, f.a, semantic)
	out = query(wp12Complete(t, f))
	if len(out.Items) != 0 {
		t.Fatal("stale source vector remained semantically searchable", out)
	}
	// Existing bytes remain durable; a stale vector is excluded by the join fence.
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE user_id=@p1 AND ref_id=@p2`, f.a, semantic) != 1 {
		t.Fatal("stale durable vector was erased")
	}
	// A freshly rebuilt vector becomes searchable again before soft deletion.
	l = claim(semantic)
	input, ok, e = r.repo.EmbeddingInput(context.Background(), *l)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = r.repo.PublishEmbedding(context.Background(), *l, input, ports.EmbeddingResponse{Model: "synthetic-embedding", Vector: []float32{1, 0, 0}}); e != nil {
		t.Fatal(e)
	}
	out = query(wp12Complete(t, f))
	if len(out.Items) == 0 || out.Items[0].ID != semantic {
		t.Fatal("rebuilt source did not recover semantic visibility", out)
	}
	aiExec(t, db, `UPDATE dbo.ai_memory_items SET deleted_at=SYSUTCDATETIME(),deleted_by=@p1,deletion_reason=N'user_request' WHERE user_id=@p1 AND id=@p2`, f.a, semantic)
	out = query(wp12Complete(t, f))
	if len(out.Items) != 0 {
		t.Fatal("hidden source returned", out)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1 AND id=@p2`, f.a, semantic) != 1 {
		t.Fatal("soft deletion physically erased data")
	}
}
