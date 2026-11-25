package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	bg "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/background"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type intelligenceFixture struct {
	wp12Runtime
	s *bg.Service
	b wp12Bundle
}

func intelligenceNew(t *testing.T, db *sql.DB) intelligenceFixture {
	t.Helper()
	r := wp12New(t, db)
	b := internalBundle(t)
	client := backgroundClient(t, r.provider)
	repo, e := aidb.NewRetrievalRepository(db, syntheticUsePolicy{true}, r.cipher)
	if e != nil {
		t.Fatal(e)
	}
	ret, e := retrieval.New(repo, client.(ports.EmbeddingClient), wp14Registry{b}, syntheticUsePolicy{true})
	if e != nil {
		t.Fatal(e)
	}
	s, e := bg.New(context.Background(), aidb.NewIntelligenceRepository(r.repo), r.repo, r.cipher, client, b, ret, wp12Policy{})
	if e != nil {
		t.Fatal(e)
	}
	s.Config.MinMessages = 2
	s.Config.FlushSeconds = 1
	return intelligenceFixture{r, s, b}
}
func (r intelligenceFixture) claim(t *testing.T, reflection bool) *mem.Lease {
	t.Helper()
	aiExec(t, r.db, `UPDATE dbo.ai_jobs SET run_after=DATEADD(second,-1,SYSUTCDATETIME()) WHERE status=N'queued' AND kind IN(N'memory_write',N'reflection'); UPDATE dbo.ai_jobs SET run_after=DATEADD(day,1,SYSUTCDATETIME()) WHERE kind=N'memory_embed' AND status=N'queued'`)
	for i := 0; i < 10; i++ {
		var l *mem.Lease
		var e error
		if reflection {
			rr, e2 := aidb.NewReflectionRepository(r.repo, contextPolicy(t, r.b))
			if e2 != nil {
				t.Fatal(e2)
			}
			l, e = rr.ClaimReflection(context.Background(), "intelligence-test", time.Minute, r.b.Reference(), 100)
		} else {
			l, e = r.repo.Claim(context.Background(), "intelligence-test", time.Minute, r.b.Reference(), 100)
		}
		if e != nil {
			t.Fatal(e)
		}
		if l != nil {
			return l
		}
		time.Sleep(time.Millisecond * 5)
	}
	t.Fatal("no background lease")
	return nil
}
func (r intelligenceFixture) output(o bg.Output, reflection bool) {
	raw, _ := json.Marshal(o)
	model := "synthetic-memory-writer"
	if reflection {
		model = "synthetic-reflection"
	}
	r.provider.QueueStructured(model, fake.StructuredReply{JSON: raw})
	r.embed(12)
}
func emptyIntelligence() bg.Output {
	return bg.Output{Schema: bg.OutputSchema, Operations: []bg.Operation{}, Formulations: []bg.Formulation{}, KnowledgeIDs: []string{}, InformationGaps: []string{}, Hints: []bg.Hint{}}
}
func TestBackgroundIntelligenceBatchCoverageAndIdleSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	r := intelligenceNew(t, db)
	turns := []c.Admission{wp12Complete(t, f)}
	wp12Enable(t, r.wp12Runtime, f.a)
	for i := 0; i < 3; i++ {
		turns = append(turns, wp12Complete(t, f))
	}
	if e := r.s.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	if e := r.s.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages WHERE user_id=@p1 AND reviewed_at IS NULL`, f.a); n != 4 {
		t.Fatal("batch reservation lost or repeated messages", n)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'memory_write'`, f.a); n != 1 {
		t.Fatal("per-turn or duplicate jobs", n)
	}
	l := r.claim(t, false)
	out := emptyIntelligence()
	for i, a := range turns {
		out.Operations = append(out.Operations, bg.Operation{ID: a.TurnID.String(), Action: "add", Kind: c.MemEvent, Content: "Synthetic batch event " + a.TurnID.String(), Evidence: []bg.Evidence{{ID: a.UserMessageID, Role: "supports"}}, Tags: []string{"synthetic_event"}})
		_ = i
	}
	out.Hints = []bg.Hint{{Topic: "Revisit the fictional event only if relevant", ResourceIDs: []string{"skill:reflect_emotion"}, EvidenceIDs: []c.ID{turns[0].UserMessageID}, TTLTurns: 3}}
	r.output(out, false)
	dir := filepath.Join(t.TempDir(), "private")
	ins, e := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if e != nil {
		t.Fatal(e)
	}
	ctx, span := trace.StartJob(trace.WithInspector(context.Background(), ins), l.TurnID, l.ID, "memory", l.Attempts)
	trace.Secret(ctx, l.Token.String())
	if e = r.s.Execute(ctx, *l); e != nil {
		t.Fatal(e)
	}
	span.End(nil, nil)
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages WHERE user_id=@p1 AND reviewed_at IS NOT NULL`, f.a); n != 4 {
		t.Fatal("successful batch not acknowledged", n)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1`, f.a); n != 4 {
		t.Fatal("batch memory missing", n)
	}
	structured := 0
	for _, call := range r.provider.History() {
		if call.Operation == "structured" {
			structured++
		}
	}
	if structured != 1 {
		t.Fatal("routine review made extra model calls", structured)
	}
	if e = r.s.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'memory_write'`, f.a); n != 1 {
		t.Fatal("reviewed messages requeued")
	}
	reader, _ := aidb.NewContextReader(db)
	reader.WithBackgroundHints(r.cipher)
	snapshot, e := reader.Read(context.Background(), f.a, f.ca)
	if e != nil || len(snapshot.FutureHints) != 1 || snapshot.FutureHints[0].JobID == nil {
		t.Fatal("background hint unavailable", e, snapshot.FutureHints)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(files) != 1 {
		t.Fatal(files)
	}
	raw, e := os.ReadFile(files[0])
	if e != nil || trace.ValidateJSON(raw) != nil {
		t.Fatal("invalid background Inspector", e)
	}
	if target := os.Getenv("THEIA_WRITE_BACKGROUND_FIXTURE"); target != "" {
		if e = os.WriteFile(target, raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	// A quiet user's partial batch eventually flushes without another turn.
	last := wp12Complete(t, f)
	time.Sleep(1100 * time.Millisecond)
	if e = r.s.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages WHERE user_id=@p1 AND message_id=@p2 AND reviewed_at IS NULL`, f.a, last.UserMessageID); n != 1 {
		t.Fatal("inactive user's final message missed")
	}
	// No background inference is a prerequisite for finalizing these turns.
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.turns WHERE user_id=@p1 AND status=N'completed'`, f.a); n != 5 {
		t.Fatal("background work changed completed turns")
	}
}
func TestBackgroundIntelligenceRollbackStaleCheckpointAndRetrievedTargetSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	r := intelligenceNew(t, db)
	a := wp12Complete(t, f)
	wp12Enable(t, r.wp12Runtime, f.a)
	old := wp12Apply(t, r.wp12Runtime, a, wp12Op(a.UserMessageID, "Synthetic historical relationship event"))
	// The relevant old target is deliberately outside the recent 16-item
	// fallback; explicit semantic retrieval must make it a usable target.
	for i := 0; i < 18; i++ {
		wp12Apply(t, r.wp12Runtime, a, wp12Op(a.UserMessageID, "Unrelated fictional record "+c.NewID().String()))
	}
	// This target used to be found by BM25 while its vector was pending. Build
	// its current revision through the existing lease/publisher fixture instead.
	aiExec(t, db, `UPDATE dbo.ai_jobs SET run_after=DATEADD(day,1,SYSUTCDATETIME()) WHERE user_id=@p1 AND kind=N'memory_embed';
 UPDATE j SET run_after=DATEADD(second,-1,SYSUTCDATETIME()) FROM dbo.ai_jobs j JOIN dbo.ai_memory_items m ON m.user_id=j.user_id AND m.id=TRY_CONVERT(uniqueidentifier,JSON_VALUE(j.payload_json,'$.source_id')) AND CONVERT(varchar(16),CONVERT(binary(8),m.search_version),2)=JSON_VALUE(j.payload_json,'$.source_version') WHERE j.user_id=@p1 AND j.kind=N'memory_embed' AND m.id=@p2 AND j.status=N'queued'`, f.a, old)
	indexed, e := r.repo.Claim(context.Background(), "semantic-history-fixture", time.Minute, r.bundle.Reference(), 100)
	if e != nil || indexed == nil {
		t.Fatal("historical target indexing lease", e)
	}
	input, eligible, e := r.repo.EmbeddingInput(context.Background(), *indexed)
	if e != nil || !eligible {
		t.Fatal("historical target indexing eligibility", eligible, e)
	}
	if e = r.repo.PublishEmbedding(context.Background(), *indexed, input, ports.EmbeddingResponse{Model: "synthetic-embedding", Vector: []float32{1, 0, 0}}); e != nil {
		t.Fatal(e)
	}
	next := wp12Complete(t, f)
	request := c.BackgroundRequest{ID: "history", Kind: "memory_intelligence", Objective: "Review and correct a fictional historical relationship event", Topic: "synthetic relationship", Queries: []string{"Synthetic historical relationship event"}, EvidenceIDs: []c.ID{next.UserMessageID}}
	job, e := r.s.Plan(context.Background(), next, request, bg.Scope{Kind: "user", ThreadIDs: []c.ID{}})
	if e != nil {
		t.Fatal(e)
	}
	if e = r.repo.Enqueue(context.Background(), f.a, next.TurnID, *job); e != nil {
		t.Fatal(e)
	}
	// Enqueue must accept the new encrypted task, using the existing queue.
	l := r.claim(t, false)
	out := emptyIntelligence()
	out.Operations = []bg.Operation{{ID: "contradiction", Action: "contradict", TargetID: &old, Evidence: []bg.Evidence{{ID: next.UserMessageID, Role: "contradicts"}}}}
	r.output(out, false)
	aiExec(t, db, `CREATE TRIGGER background_test_rollback ON dbo.ai_memory_batches AFTER INSERT AS THROW 51001,'synthetic rollback',1;`)
	if e = r.s.Execute(context.Background(), *l); e == nil {
		t.Fatal("fault did not fail")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages WHERE message_id=@p1 AND reviewed_at IS NOT NULL`, next.UserMessageID); n != 0 {
		t.Fatal("failed job marked a message reviewed")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND status=N'active'`, old); n != 1 {
		t.Fatal("rollback published partial memory")
	}
	calls := len(r.provider.History())
	aiExec(t, db, `DROP TRIGGER background_test_rollback`)
	// A recovered lease receives the persisted encrypted checkpoint.
	if e = db.QueryRow(`SELECT memory_result_enc FROM dbo.ai_jobs WHERE id=@p1`, l.ID).Scan(&l.Checkpoint); e != nil || len(l.Checkpoint) == 0 {
		t.Fatal("durable checkpoint missing", e)
	}
	// No second retrieval or inference is required.
	if e = r.s.Execute(context.Background(), *l); e != nil {
		t.Fatal(e)
	}
	if len(r.provider.History()) != calls {
		t.Fatal("checkpoint recovery repeated retrieval/model work")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND content=N'Synthetic historical relationship event' AND status=N'contested'`, old); n != 1 {
		t.Fatal("retrieved target correction lost history")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_evidence WHERE item_id=@p1 AND ref_id=@p2 AND role=N'contradicts'`, old, next.UserMessageID); n != 1 {
		t.Fatal("contradicting evidence missing")
	}
	// A checkpoint based on stale evidence must be discarded, never committed.
	later := wp12Complete(t, f)
	request.ID = "stale"
	job, e = r.s.Plan(context.Background(), later, request, bg.Scope{Kind: "user", ThreadIDs: []c.ID{}})
	if e != nil {
		t.Fatal(e)
	}
	if e = r.repo.Enqueue(context.Background(), f.a, later.TurnID, *job); e != nil {
		t.Fatal(e)
	}
	l = r.claim(t, false)
	r.output(emptyIntelligence(), false)
	aiExec(t, db, `CREATE TRIGGER background_test_rollback ON dbo.ai_memory_batches AFTER INSERT AS THROW 51001,'synthetic rollback',1;`)
	if e = r.s.Execute(context.Background(), *l); e == nil {
		t.Fatal("fault missing")
	}
	aiExec(t, db, `DROP TRIGGER background_test_rollback`)
	if e = db.QueryRow(`SELECT memory_result_enc FROM dbo.ai_jobs WHERE id=@p1`, l.ID).Scan(&l.Checkpoint); e != nil || len(l.Checkpoint) == 0 {
		t.Fatal("stale checkpoint missing", e)
	}
	aiExec(t, db, `UPDATE dbo.messages SET content=content+N' revised synthetic evidence' WHERE id=@p1`, later.UserMessageID)
	if e = r.s.Execute(context.Background(), *l); !errors.Is(e, mem.ErrConflict) {
		t.Fatal("stale checkpoint not fenced", e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE id=@p1 AND memory_result_enc IS NULL`, l.ID); n != 1 {
		t.Fatal("stale checkpoint cannot be freshly retried")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages WHERE message_id=@p1 AND reviewed_at IS NOT NULL`, later.UserMessageID); n != 0 {
		t.Fatal("stale result acknowledged coverage")
	}
}
func contextPolicy(t *testing.T, b wp12Bundle) contextstate.Policy {
	t.Helper()
	p, e := contextstate.LoadPolicy(context.Background(), wp14Registry{b}, c.NewID())
	if e != nil {
		t.Fatal(e)
	}
	return p
}

func TestBackgroundIntelligenceLongitudinalReflectionRevisionAndIsolationSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	r := intelligenceNew(t, db)
	first := wp12Complete(t, f)
	wp12Enable(t, r.wp12Runtime, f.a)
	req := f.req()
	req.ConversationID = f.ca2
	req.Locale = "en"
	req.TZ = "UTC"
	second := admit(t, f, req)
	p := prepared(second)
	p.Jobs = nil
	if _, e := f.store.Complete(context.Background(), second, p); e != nil {
		t.Fatal(e)
	}
	analysis := c.BackgroundRequest{ID: "cross_conversation", Kind: "reflection", Objective: "Compare fictional experiences across conversations and consider alternatives", Topic: "fictional recurring context", Queries: []string{"synthetic"}, EvidenceIDs: []c.ID{second.UserMessageID}, Scope: &c.BackgroundScope{Kind: "user", ThreadIDs: []c.ID{}}}
	job, e := r.s.Plan(context.Background(), second, analysis, bg.Scope{Kind: "user", ThreadIDs: []c.ID{}})
	if e != nil {
		t.Fatal(e)
	}
	if e = r.repo.Enqueue(context.Background(), f.a, second.TurnID, *job); e != nil {
		t.Fatal(e)
	}
	l := r.claim(t, true)
	out := emptyIntelligence()
	out.Operations = []bg.Operation{{ID: "pattern", Action: "add", Kind: c.MemHypothesis, Content: "May show a tentative fictional pattern across the two reports", Uncertainty: "Only two synthetic reports; alternative explanations remain plausible", Alternatives: []string{"May be temporary circumstances"}, Evidence: []bg.Evidence{{ID: first.UserMessageID, Role: "supports"}, {ID: second.UserMessageID, Role: "supports"}}}}
	out.KnowledgeIDs = []string{"framework:synthetic_framework"}
	out.Formulations = []bg.Formulation{{Doc: c.FormulationDoc{Framework: "synthetic_framework", Fields: map[string]c.FormulationField{"pattern": {Value: "May show a fictional recurring context", Status: c.EpInferred, Evidence: []c.ID{first.UserMessageID, second.UserMessageID}, NeedsEvidenceMore: true}}, Paragraph: "A tentative fictional pattern with plausible alternatives; more user evidence is needed."}}}
	r.output(out, true)
	if e = r.s.Execute(context.Background(), *l); e != nil {
		wp12Err(t, e)
	}
	var hypothesis c.ID
	if e = db.QueryRow(`SELECT CONVERT(varchar(36),id) FROM dbo.ai_memory_items WHERE user_id=@p1 AND kind=N'hypothesis'`, f.a).Scan(&hypothesis); e != nil {
		t.Fatal(e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_reflection_runs WHERE user_id=@p1 AND thread_id IS NULL AND job_id=@p2`, f.a, l.ID); n != 1 {
		t.Fatal("user-level reflection forced to a thread")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_formulations WHERE user_id=@p1 AND thread_id IS NULL`, f.a); n != 1 {
		t.Fatal("user-level formulation missing")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND epistemic=N'inferred' AND status=N'proposed' AND confidence=N'low'`, hypothesis); n != 1 {
		t.Fatal("hypothesis became fact")
	}
	// New primary contradictory evidence revises/invalidate the existing hypothesis.
	third := wp12Complete(t, f)
	analysis.ID = "invalidate"
	job, e = r.s.Plan(context.Background(), third, analysis, bg.Scope{Kind: "user", ThreadIDs: []c.ID{}})
	if e != nil {
		t.Fatal(e)
	}
	if e = r.repo.Enqueue(context.Background(), f.a, third.TurnID, *job); e != nil {
		t.Fatal(e)
	}
	l = r.claim(t, true)
	out = emptyIntelligence()
	out.Operations = []bg.Operation{{ID: "invalidate", Action: "invalidate", TargetID: &hypothesis, Evidence: []bg.Evidence{{ID: third.UserMessageID, Role: "contradicts"}}}}
	r.output(out, true)
	if e = r.s.Execute(context.Background(), *l); e != nil {
		wp12Err(t, e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND status=N'rejected' AND epistemic=N'inferred'`, hypothesis); n != 1 {
		t.Fatal("hypothesis not invalidated")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_versions WHERE item_id=@p1`, hypothesis); n != 1 {
		t.Fatal("hypothesis history erased")
	}
	// Assistant speculation cannot become independent evidence, including when
	// the source is otherwise visible and present in recent dialogue.
	var assistant c.ID
	if e = db.QueryRow(`SELECT CONVERT(varchar(36),id) FROM dbo.messages WHERE turn_id=@p1 AND role=N'assistant'`, third.TurnID).Scan(&assistant); e != nil {
		t.Fatal(e)
	}
	fourth := wp12Complete(t, f)
	analysis.ID = "unsupported"
	job, e = r.s.Plan(context.Background(), fourth, analysis, bg.Scope{Kind: "user", ThreadIDs: []c.ID{}})
	if e != nil {
		t.Fatal(e)
	}
	if e = r.repo.Enqueue(context.Background(), f.a, fourth.TurnID, *job); e != nil {
		t.Fatal(e)
	}
	l = r.claim(t, true)
	out = emptyIntelligence()
	out.Operations = []bg.Operation{{ID: "unsupported", Action: "add", Kind: c.MemHypothesis, Content: "May be an unsupported synthetic interpretation", Uncertainty: "tentative", Evidence: []bg.Evidence{{ID: assistant, Role: "supports"}}}}
	r.output(out, true)
	if e = r.s.Execute(context.Background(), *l); e != nil {
		wp12Err(t, e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1 AND content=N'May be an unsupported synthetic interpretation'`, f.a); n != 0 {
		t.Fatal("assistant speculation became independent evidence")
	}
	// Forged foreign primary IDs never reach inference or acknowledgement.
	foreign := f.req()
	foreign.UserID = f.b
	foreign.ConversationID = f.cb
	fa := admit(t, f, foreign)
	fp := prepared(fa)
	fp.Jobs = nil
	if _, e = f.store.Complete(context.Background(), fa, fp); e != nil {
		t.Fatal(e)
	}
	request := bg.Request{Origin: "unified", Kind: "reflection", Scope: bg.Scope{Kind: "user", ThreadIDs: []c.ID{}}, MessageIDs: []c.ID{fa.UserMessageID}, Trigger: fa.UserMessageID}
	before := len(r.provider.History())
	if _, e = aidb.NewIntelligenceRepository(r.repo).Prepare(context.Background(), mem.Lease{ID: c.NewID(), UserID: f.a}, request, nil); !errors.Is(e, mem.ErrDenied) {
		t.Fatal("foreign source accepted", e)
	}
	if len(r.provider.History()) != before {
		t.Fatal("unauthorized model disclosure")
	}
	aiExec(t, db, `UPDATE dbo.ai_user_state SET risk_level=N'elevated',version=version+1 WHERE user_id=@p1`, f.a)
	request.MessageIDs = []c.ID{fourth.UserMessageID}
	request.Trigger = fourth.UserMessageID
	if _, e = aidb.NewIntelligenceRepository(r.repo).Prepare(context.Background(), mem.Lease{ID: c.NewID(), UserID: f.a}, request, nil); !errors.Is(e, mem.ErrDenied) {
		t.Fatal("existing Reflection operational risk boundary changed", e)
	}

}

func TestBackgroundIntelligenceOverlapPauseFailureAndActiveTurnSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	r := intelligenceNew(t, db)
	first := wp12Complete(t, f)
	wp12Enable(t, r.wp12Runtime, f.a)
	second := wp12Complete(t, f)
	if e := r.s.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	// Explicit review covers one already-reserved message. The queued routine
	// batch keeps its remaining coverage and must not duplicate the same claim.
	aiExec(t, db, `UPDATE dbo.ai_jobs SET run_after=DATEADD(day,1,SYSUTCDATETIME()) WHERE user_id=@p1 AND kind=N'memory_write'`, f.a)
	request := c.BackgroundRequest{ID: "explicit_overlap", Kind: "memory_intelligence", Objective: "Review fictional preference", Topic: "synthetic preference", Queries: []string{"synthetic"}, EvidenceIDs: []c.ID{second.UserMessageID}}
	job, e := r.s.Plan(context.Background(), second, request, bg.Scope{Kind: "user", ThreadIDs: []c.ID{}})
	if e != nil {
		t.Fatal(e)
	}
	if e = r.repo.Enqueue(context.Background(), f.a, second.TurnID, *job); e != nil {
		t.Fatal(e)
	}
	// Claim without moving the postponed routine job forward.
	time.Sleep(5 * time.Millisecond)
	l, e := r.repo.Claim(context.Background(), "overlap", time.Minute, r.b.Reference(), 100)
	if e != nil || l == nil {
		t.Fatal(e)
	}
	out := emptyIntelligence()
	out.Operations = []bg.Operation{{ID: "preference", Action: "add", Kind: c.MemPreference, Content: "Prefers the fictional example", Evidence: []bg.Evidence{{ID: second.UserMessageID, Role: "supports"}}}}
	r.output(out, false)
	if e = r.s.Execute(context.Background(), *l); e != nil {
		wp12Err(t, e)
	}
	l = r.claim(t, false)
	out.Operations[0].Evidence[0].ID = first.UserMessageID
	r.output(out, false)
	if e = r.s.Execute(context.Background(), *l); e != nil {
		wp12Err(t, e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1 AND content=N'Prefers the fictional example'`, f.a); n != 1 {
		t.Fatal("overlap duplicated memory", n)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages WHERE user_id=@p1 AND reviewed_at IS NOT NULL`, f.a); n != 2 {
		t.Fatal("overlap lost coverage", n)
	}
	// Pause leaves queued work intact, with no provider disclosure or attempts.
	third := wp12Complete(t, f)
	fourth := wp12Complete(t, f)
	_ = third
	_ = fourth
	if e = r.s.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	pause := true
	if e = r.service.SetControls(r.ctx(f.a, c.ID{}), f.a, &pause, nil, nil); e != nil {
		t.Fatal(e)
	}
	before := len(r.provider.History())
	lease, e := r.repo.Claim(context.Background(), "paused", time.Minute, r.b.Reference(), 100)
	if e != nil {
		t.Fatal(e)
	}
	if lease != nil && lease.Kind == "memory_write" {
		t.Fatal("paused review claimed")
	}
	if len(r.provider.History()) != before {
		t.Fatal("pause disclosed data")
	}
	pause = false
	if e = r.service.SetControls(r.ctx(f.a, c.ID{}), f.a, &pause, nil, nil); e != nil {
		t.Fatal(e)
	}
	// An active foreground lease takes priority over analytical inference.
	active := admit(t, f, f.req())
	lease, e = r.repo.Claim(context.Background(), "foreground", time.Minute, r.b.Reference(), 100)
	if e != nil {
		t.Fatal(e)
	}
	if lease != nil && lease.Kind == "memory_write" {
		t.Fatal("background competed with foreground")
	}
	p := prepared(active)
	p.Jobs = nil
	if _, e = f.store.Complete(context.Background(), active, p); e != nil {
		t.Fatal(e)
	}
	l = r.claim(t, false)
	r.embed(12)
	r.provider.QueueStructured("synthetic-memory-writer", fake.StructuredReply{Err: errors.New("synthetic unavailable")})
	if e = r.s.Execute(context.Background(), *l); e == nil {
		t.Fatal("provider outage ignored")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages WHERE job_id=@p1 AND reviewed_at IS NOT NULL`, l.ID); n != 0 {
		t.Fatal("failed inference acknowledged coverage")
	}
	if e = r.repo.Fail(context.Background(), *l, "memory_retry", 0); e != nil {
		t.Fatal(e)
	}
	l = r.claim(t, false)
	r.output(emptyIntelligence(), false)
	if e = r.s.Execute(context.Background(), *l); e != nil {
		wp12Err(t, e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages WHERE job_id=@p1 AND reviewed_at IS NOT NULL`, l.ID); n != 2 {
		t.Fatal("retry missed messages", n)
	}
	// Reviewed message revisions schedule a fresh identity, not the old receipt.
	aiExec(t, db, `UPDATE dbo.messages SET content=content+N' revised fictional preference' WHERE id=@p1`, first.UserMessageID)
	time.Sleep(1100 * time.Millisecond)
	if e = r.s.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages WHERE user_id=@p1 AND message_id=@p2 AND reviewed_at IS NULL`, f.a, first.UserMessageID); n != 1 {
		t.Fatal("changed primary message was silently skipped")
	}
}

func TestBackgroundIntelligenceUpgradeOwnershipAndInvalidationSQL(t *testing.T) {
	db := aiDatabase(t)
	ctx := context.Background()
	old := t.TempDir()
	paths, e := filepath.Glob("../../db/migrations/*.sql")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range paths {
		if filepath.Base(path) >= "0042" {
			continue
		}
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(old, filepath.Base(path)), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e = migrations.Apply(ctx, db, old); e != nil {
		t.Fatal(e)
	}
	f := aiSetup(t, db)
	a := wp12Complete(t, f)
	fact, hyp, form := c.NewID(), c.NewID(), c.NewID()
	aiExec(t, db, `INSERT dbo.ai_memory_items(id,user_id,kind,content,lang,epistemic,status,source_json) VALUES(@p1,@p2,N'fact',N'Original fictional relationship',N'en',N'reported',N'active',N'{"origin":"user_edit"}'); INSERT dbo.ai_memory_items(id,user_id,kind,content,lang,epistemic,status,source_json) VALUES(@p3,@p2,N'hypothesis',N'May reflect a fictional tentative pattern',N'en',N'inferred',N'proposed',N'{"origin":"reflection"}'); INSERT dbo.ai_memory_evidence(user_id,item_id,ref_kind,ref_id,role,evidence_at) VALUES(@p2,@p3,N'item',@p1,N'context',SYSUTCDATETIME())`, fact, f.a, hyp)
	aiMigrate(t, db)
	aiMigrate(t, db)
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1`, f.a); n != 2 {
		t.Fatal("upgrade discarded memories")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1`, a.TurnID); n != 2 {
		t.Fatal("upgrade discarded raw evidence")
	}
	refs, _ := json.Marshal([]c.SourceRevision{{Kind: "memory", ID: fact, Version: "0000000000000001"}})
	aiExec(t, db, `INSERT dbo.ai_formulations(id,user_id,thread_id,version,framework,fields_json,paragraph,sources_json) VALUES(@p1,@p2,NULL,1,N'synthetic_framework',N'{}',N'Synthetic tentative formulation',@p3)`, form, f.a, string(refs))
	r := wp12New(t, db)
	wp12Enable(t, r, f.a)
	if e = r.service.Correct(r.ctx(f.a, a.TurnID), f.a, fact, ports.CorrectionContest); e != nil {
		wp12Err(t, e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND status=N'stale' AND content=N'May reflect a fictional tentative pattern'`, hyp); n != 1 {
		t.Fatal("dependent hypothesis was not invalidated")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_versions WHERE item_id=@p1`, hyp); n != 1 {
		t.Fatal("invalidation erased hypothesis revision")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_formulations WHERE id=@p1 AND analysis_status=N'stale' AND paragraph=N'Synthetic tentative formulation'`, form); n != 1 {
		t.Fatal("dependent formulation remained current")
	}
	jobA, jobB := c.NewID(), c.NewID()
	aiExec(t, db, `INSERT dbo.ai_jobs(id,user_id,kind,dedupe_key,payload_json) VALUES(@p1,@p2,N'memory_write',N'owner-a',N'{}'),(@p3,@p4,N'memory_write',N'owner-b',N'{}')`, jobA, f.a, jobB, f.b)
	if _, e = db.Exec(`INSERT dbo.ai_memory_review_messages(user_id,message_id,job_id,source_version) SELECT @p1,id,@p3,CONVERT(binary(8),search_version) FROM dbo.messages WHERE id=@p2`, f.b, a.UserMessageID, jobB); e == nil {
		t.Fatal("foreign review ownership accepted")
	}
	aiExec(t, db, `INSERT dbo.ai_memory_review_messages(user_id,message_id,job_id,source_version) SELECT @p1,id,@p3,CONVERT(binary(8),search_version) FROM dbo.messages WHERE id=@p2`, f.a, a.UserMessageID, jobA)
	if _, e = db.Exec(`DELETE dbo.ai_memory_review_messages WHERE user_id=@p1`, f.a); e == nil {
		t.Fatal("review history physically deleted")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages WHERE user_id=@p1`, f.a); n != 1 {
		t.Fatal("delete guard failed to preserve ledger")
	}
}

func TestBackgroundIntelligenceLegacyTurnAndIndependentAfterDeleteSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	r := intelligenceNew(t, db)
	a := wp12Complete(t, f)
	wp12Enable(t, r.wp12Runtime, f.a)
	// Original prototype records are still eligible without pretending that
	// they were admitted by the current turn lifecycle.
	aiExec(t, db, `UPDATE dbo.turns SET lifecycle_version=0 WHERE id=@p1`, a.TurnID)
	time.Sleep(1100 * time.Millisecond)
	if e := r.s.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	l := r.claim(t, false)
	if c.ValidID(l.TurnID) {
		t.Fatal("prototype review fabricated a v1 turn association")
	}
	out := emptyIntelligence()
	out.Operations = []bg.Operation{{ID: "preference", Action: "add", Kind: c.MemPreference, Content: "Prefers the fictional green example", Evidence: []bg.Evidence{{ID: a.UserMessageID, Role: "supports"}}}}
	r.output(out, false)
	if e := r.s.Execute(context.Background(), *l); e != nil {
		wp12Err(t, e)
	}
	var original c.ID
	if e := db.QueryRow(`SELECT CONVERT(varchar(36),id) FROM dbo.ai_memory_items WHERE user_id=@p1 AND content=N'Prefers the fictional green example'`, f.a).Scan(&original); e != nil {
		t.Fatal(e)
	}
	if _, e := r.service.Delete(r.ctx(f.a, c.ID{}), f.a, ports.DeletionScope{ItemIDs: []c.ID{original}}); e != nil {
		t.Fatal(e)
	}
	next := wp12Complete(t, f)
	request := c.BackgroundRequest{ID: "new_evidence", Kind: "memory_intelligence", Objective: "Review a new independently reported preference", Topic: "fictional preference", Queries: []string{"fictional green example"}, EvidenceIDs: []c.ID{next.UserMessageID}}
	job, e := r.s.Plan(context.Background(), next, request, bg.Scope{Kind: "user", ThreadIDs: []c.ID{}})
	if e != nil {
		t.Fatal(e)
	}
	if e = r.repo.Enqueue(context.Background(), f.a, next.TurnID, *job); e != nil {
		t.Fatal(e)
	}
	l = r.claim(t, false)
	out.Operations[0].Evidence[0].ID = next.UserMessageID
	r.output(out, false)
	if e = r.s.Execute(context.Background(), *l); e != nil {
		wp12Err(t, e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND deleted_at IS NOT NULL AND content=N'Prefers the fictional green example'`, original); n != 1 {
		t.Fatal("hidden original was revived or erased")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1 AND deleted_at IS NULL AND content=N'Prefers the fictional green example'`, f.a); n != 1 {
		t.Fatal("independent permitted evidence was accidentally suppressed")
	}
}
