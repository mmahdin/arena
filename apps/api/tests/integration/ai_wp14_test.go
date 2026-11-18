package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	rf "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/reflection"
	sch "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/scheduler"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	convdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/adapters/mssql"
)

type wp14Runtime struct {
	wp12Runtime
	repo14   *aidb.ReflectionRepository
	worker14 *rf.Worker
	timers   *sch.Service
	assets14 *rf.Assets
}

func wp14New(t *testing.T, db *sql.DB) wp14Runtime {
	t.Helper()
	r := wp12New(t, db)
	raw, e := os.ReadFile("../../internal/modules/ai/application/reflection/testdata/assets.json")
	if e != nil {
		t.Fatal(e)
	}
	var files map[string]string
	_ = json.Unmarshal(raw, &files)
	for p, v := range files {
		r.bundle.wp10SQLBundle[p] = []byte(v)
	}
	a, e := rf.Load(context.Background(), r.bundle, rf.LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		wp12Err(t, e)
	}
	ma, e := mem.Load(context.Background(), r.bundle, mem.LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		wp12Err(t, e)
	}
	pol := contextstate.Policy{Ref: r.bundle.Reference(), Handle: r.bundle, Synthetic: true}
	if json.Unmarshal([]byte(files["config/thresholds.yaml"]), &pol.Thresholds) != nil {
		t.Fatal("thresholds")
	} // YAML tags differ from Go JSON; load through WP-08 below
	pol, e = contextstate.LoadPolicy(context.Background(), wp14Registry{r.bundle}, c.NewID())
	if e != nil {
		wp12Err(t, e)
	}
	repo, e := aidb.NewReflectionRepository(r.repo, pol)
	if e != nil {
		wp12Err(t, e)
	}
	cl := wp05Client(t, r.provider)
	writer, e := mem.NewWriter(cl, ma, repo, nil)
	if e != nil {
		wp12Err(t, e)
	}
	committer, e := mem.New(repo, ma, mem.Options{Policy: wp12Policy{}, Cipher: r.cipher, Embedder: cl.(ports.EmbeddingClient), EmbeddingModel: "wp05-embedding", Dimension: 3})
	if e != nil {
		wp12Err(t, e)
	}
	model, e := rf.NewModel(cl, a, repo, wp12Policy{})
	if e != nil {
		wp12Err(t, e)
	}
	worker, e := rf.NewWorker(repo, model, writer, committer, a, wp12Policy{}, r.cipher, "synthetic-wp14")
	if e != nil {
		wp12Err(t, e)
	}
	timers, e := sch.New(repo, a.Reference())
	if e != nil {
		wp12Err(t, e)
	}
	return wp14Runtime{r, repo, worker, timers, a}
}

type wp14Registry struct{ ports.Bundle }

func (r wp14Registry) Active(context.Context, c.ID) (ports.Bundle, error) { return r.Bundle, nil }
func wp14Seed(t *testing.T, db *sql.DB) (aiFixture, wp14Runtime, rf.Task, []c.ID) {
	t.Helper()
	f := aiSetup(t, db)
	r := wp14New(t, db)
	a := wp12Complete(t, f)
	wp12Enable(t, r.wp12Runtime, f.a)
	thread := c.NewID()
	if _, e := db.Exec(`INSERT dbo.ai_threads(id,user_id,topic,phase,problem_tags_json) VALUES(@p1,@p2,N'synthetic topic',N'exploring',N'["synthetic_problem"]'); UPDATE dbo.ai_sessions SET focus_thread_id=@p1,closed_at=SYSUTCDATETIME(),close_reason=N'test',version=version+1 WHERE id=@p3 AND user_id=@p2`, thread, f.a, a.SessionID); e != nil {
		wp12Err(t, e)
	}
	var assistant c.ID
	if e := db.QueryRow(`SELECT CONVERT(varchar(36),id) FROM dbo.messages WHERE turn_id=@p1 AND role=N'assistant'`, a.TurnID).Scan(&assistant); e != nil {
		wp12Err(t, e)
	}
	return f, r, rf.Task{Schema: "reflection_job.v1", Bundle: r.bundle.Reference(), UserID: f.a, ThreadID: thread, SessionID: a.SessionID, Trigger: "session_closed"}, []c.ID{a.UserMessageID, assistant}
}
func wp14Queue(t *testing.T, r wp14Runtime, task rf.Task, ids []c.ID, episode bool) {
	t.Helper()
	result := c.ReflectionResult{SchemaVersion: c.ReflectionSchemaVersion, ThreadID: task.ThreadID, Formulation: &c.FormulationDoc{Framework: "synthetic_framework", Fields: map[string]c.FormulationField{"situation": {Value: "invented unsupported", Status: c.EpReported, Evidence: []c.ID{ids[0]}}}, Paragraph: "invented unsupported"}, HypothesisOps: []c.MemoryOpProposal{}, ThreadProposals: []c.ThreadProposal{}, InformationGaps: []string{"synthetic missing information"}}
	if episode {
		result.Episode = &c.EpisodeDoc{SessionID: task.SessionID, Summary: "Synthetic session summary", Outcome: "explored", MsgRange: [2]c.ID{ids[0], ids[1]}, Topics: []string{"synthetic"}, Emotions: []c.Emotion{}, FollowUps: []string{"unapproved model suggestion"}}
	}
	raw, _ := json.Marshal(result)
	r.provider.QueueStructured("wp05-generator", fake.StructuredReply{JSON: raw})
	if episode {
		output := c.MemoryWriterOutput{SchemaVersion: c.MemoryWriterSchemaVersion, Ops: []c.MemoryOpProposal{{Op: c.OpAdd, Kind: c.MemEpisode, Content: "Synthetic completed session", Epistemic: c.EpInferred, EvidenceMsg: []c.ID{ids[0]}, Tags: []string{"synthetic_event"}, Sensitivity: c.SensNormal, ReasonTag: "session_summary", Importance: wp14Importance(.5)}}}
		raw, _ := json.Marshal(output)
		r.provider.QueueStructured("wp05-brain", fake.StructuredReply{JSON: raw})
		r.provider.QueueEmbedding("wp05-embedding", fake.EmbeddingReply{Vector: []float32{1, 0, 0}})
	}
	if e := r.worker14.Schedule(context.Background(), task); e != nil {
		wp12Err(t, e)
	}
}
func wp14Run(t *testing.T, w *rf.Worker) (bool, error) {
	t.Helper()
	for i := 0; i < 40; i++ {
		worked, e := w.RunOnce(context.Background())
		if worked || e != nil {
			return worked, e
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false, nil
}
func TestWP14SQLAtomicReflectionEpisodePrivacyAndIsolation(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f, r, task, ids := wp14Seed(t, db)
	wp14Queue(t, r, task, ids, true)
	if ok, e := wp14Run(t, r.worker14); !ok || e != nil {
		wp12Err(t, e)
		t.Fatal("not claimed")
	}
	var count int
	var fields, paragraph string
	if e := db.QueryRow(`SELECT fields_json,paragraph FROM dbo.ai_formulations WHERE user_id=@p1 AND thread_id=@p2`, f.a, task.ThreadID).Scan(&fields, &paragraph); e != nil {
		wp12Err(t, e)
	}
	var decoded map[string]c.FormulationField
	_ = json.Unmarshal([]byte(fields), &decoded)
	if decoded["situation"].Status != c.EpUnknown || paragraph != "" {
		t.Fatal("RF004 bypass")
	}
	if e := db.QueryRow(`SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1 AND kind=N'episode' AND JSON_VALUE(source_json,'$.origin')=N'reflection'`, f.a).Scan(&count); e != nil || count != 1 {
		t.Fatalf("episode commit %d %v", count, e)
	}
	if e := db.QueryRow(`SELECT COUNT(*) FROM dbo.ai_reflection_runs r JOIN dbo.ai_jobs j ON j.id=r.job_id AND j.user_id=r.user_id WHERE r.user_id=@p1 AND j.status=N'completed'`, f.a).Scan(&count); e != nil || count != 1 {
		t.Fatal("receipt/job atomic")
	}
	if e := db.QueryRow(`SELECT COUNT(*) FROM dbo.ai_follow_ups WHERE user_id=@p1`, f.a).Scan(&count); e != nil || count != 0 {
		t.Fatal("model scheduled unsolicited followup")
	}
	other := task
	other.UserID = f.b
	if _, e := r.repo14.Load(context.Background(), other); e == nil {
		t.Fatal("cross-user reflection")
	}
	if e := r.worker14.Schedule(context.Background(), task); e != nil {
		wp12Err(t, e)
	}
	if e := db.QueryRow(`SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'reflection'`, f.a).Scan(&count); e != nil || count != 1 {
		t.Fatal("duplicate terminal job")
	}
	var before []byte
	if e := db.QueryRow(`SELECT result_enc FROM dbo.ai_reflection_runs WHERE user_id=@p1`, f.a).Scan(&before); e != nil {
		t.Fatal(e)
	}
	if e := convdb.New(db).Delete(context.Background(), f.a, f.ca); e != nil {
		wp12Err(t, e)
	}
	var after []byte
	if e := db.QueryRow(`SELECT result_enc FROM dbo.ai_reflection_runs WHERE user_id=@p1`, f.a).Scan(&after); e != nil {
		t.Fatal(e)
	}
	if string(before) != string(after) {
		t.Fatal("derived content changed by visibility")
	}
	retained, e := r.repo14.Load(context.Background(), task)
	if e != nil || len(retained.Items) == 0 || len(retained.Sources) != 0 {
		t.Fatal("derived permission coupled to raw visibility", e, len(retained.Items), len(retained.Sources))
	}
	if _, e := db.Exec(`DELETE dbo.ai_reflection_runs WHERE user_id=@p1`, f.a); e == nil {
		t.Fatal("physical delete permitted")
	}
}
func TestWP14SQLRollbackCheckpointRecoveryConsentAndLeaseFencing(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f, r, task, ids := wp14Seed(t, db)
	wp14Queue(t, r, task, ids, false)
	if _, e := db.Exec(`CREATE TRIGGER wp14_failure ON dbo.ai_formulations AFTER INSERT AS THROW 51001,'synthetic rollback test',1;`); e != nil {
		t.Fatal(e)
	}
	if ok, e := wp14Run(t, r.worker14); !ok || e == nil {
		t.Fatal("fault did not fail")
	}
	var count int
	if e := db.QueryRow(`SELECT COUNT(*) FROM dbo.ai_reflection_runs WHERE user_id=@p1`, f.a).Scan(&count); e != nil || count != 0 {
		t.Fatal("partial reflection published")
	}
	var cp []byte
	if e := db.QueryRow(`SELECT memory_result_enc FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'reflection'`, f.a).Scan(&cp); e != nil || len(cp) == 0 {
		t.Fatal("checkpoint lost")
	}
	calls := len(r.provider.History())
	if _, e := db.Exec(`DROP TRIGGER wp14_failure; UPDATE dbo.ai_jobs SET run_after=DATEADD(second,-1,SYSUTCDATETIME()) WHERE user_id=@p1 AND kind=N'reflection'`, f.a); e != nil {
		t.Fatal(e)
	}
	if ok, e := wp14Run(t, r.worker14); !ok || e != nil {
		wp12Err(t, e)
		t.Fatal("recovery")
	}
	if len(r.provider.History()) != calls {
		t.Fatal("recovery called provider again")
	}
	// Reclaimed tokens fence checkpoint, completion and all memory publication.
	task.Trigger = "nightly"
	if e := r.worker14.Schedule(context.Background(), task); e != nil {
		wp12Err(t, e)
	}
	time.Sleep(5 * time.Millisecond)
	old, e := r.repo14.ClaimReflection(context.Background(), "old", time.Minute, task.Bundle, 10)
	if e != nil || old == nil {
		t.Fatalf("old lease %v", e)
	}
	if _, e = db.Exec(`UPDATE dbo.ai_jobs SET memory_lease_until=DATEADD(second,-1,SYSUTCDATETIME()) WHERE id=@p1`, old.ID); e != nil {
		t.Fatal(e)
	}
	fresh, e := r.repo14.ClaimReflection(context.Background(), "fresh", time.Minute, task.Bundle, 10)
	if e != nil || fresh == nil || old.Token == fresh.Token {
		t.Fatal("lease reclaim")
	}
	if !errors.Is(r.repo14.Complete(context.Background(), *old), mem.ErrConflict) {
		t.Fatal("stale completion")
	}
	if !errors.Is(r.repo14.Checkpoint(context.Background(), *old, []byte{1}), mem.ErrConflict) {
		t.Fatal("stale checkpoint")
	}
	paused := true
	if e = r.service.SetControls(r.ctx(f.a, c.ID{}), f.a, &paused, nil, nil); e != nil {
		wp12Err(t, e)
	}
	if _, e = r.repo14.Load(context.Background(), task); !errors.Is(e, mem.ErrDenied) {
		t.Fatalf("paused reflection: %v", e)
	}
}
func TestWP14SQLSchedulerConsentQuietHoursConcurrencyAndIdleRecovery(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f, r, task, _ := wp14Seed(t, db)
	now := time.Now().UTC().Truncate(time.Millisecond)
	due := now.Add(time.Hour)
	id := uuid.NewSHA1(f.a, []byte("synthetic checkin"))
	op := c.FollowUpOp{Op: "schedule", FollowUpID: id, DueAt: due, Reason: "check_in", Ref: &task.ThreadID}
	ctx := r.ctx(f.a, c.ID{})
	if e := r.timers.Apply(ctx, []c.FollowUpOp{op}); !errors.Is(e, mem.ErrDenied) {
		t.Fatal("proactive opt-out ignored")
	}
	if _, e := db.Exec(`UPDATE dbo.ai_user_state SET proactive_opt_in=1,quiet_hours_json=NULL,version=version+1 WHERE user_id=@p1`, f.a); e != nil {
		t.Fatal(e)
	}
	if e := r.timers.Apply(ctx, []c.FollowUpOp{op}); e != nil {
		wp12Err(t, e)
	}
	if e := r.timers.Apply(ctx, []c.FollowUpOp{op}); e != nil {
		t.Fatal("timer idempotency", e)
	}
	if _, e := db.Exec(`UPDATE dbo.ai_user_state SET proactive_opt_in=0,version=version+1 WHERE user_id=@p1`, f.a); e != nil {
		t.Fatal(e)
	}
	got, e := r.timers.Due(context.Background(), due.Add(time.Second), 10)
	if e != nil || len(got) != 0 {
		t.Fatal("revoked consent dispatched")
	}
	var skipped, why string
	if e := db.QueryRow(`SELECT status,skip_reason FROM dbo.ai_follow_ups WHERE id=@p1 AND user_id=@p2`, id, f.a).Scan(&skipped, &why); e != nil || skipped != "skipped" || why != "consent_revoked" {
		t.Fatalf("revoked follow-up was not durably skipped: %s %s %v", skipped, why, e)
	}
	if _, e := db.Exec(`UPDATE dbo.ai_user_state SET proactive_opt_in=1,quiet_hours_json=N'{"start":"00:00","end":"23:59"}',version=version+1 WHERE user_id=@p1`, f.a); e != nil {
		t.Fatal(e)
	}
	got, e = r.timers.Due(context.Background(), due.Add(time.Second), 10)
	if e != nil || len(got) != 0 {
		t.Fatal("quiet hours dispatched")
	}
	if _, e := db.Exec(`UPDATE dbo.ai_user_state SET quiet_hours_json=NULL,version=version+1 WHERE user_id=@p1`, f.a); e != nil {
		t.Fatal(e)
	}
	// Re-enabling consent never resurrects the revoked event. Only an
	// independently scheduled new follow-up may be dispatched.
	got, e = r.timers.Due(context.Background(), due.Add(time.Second), 10)
	if e != nil || len(got) != 0 {
		t.Fatal("revoked follow-up resurrected")
	}
	op.FollowUpID = uuid.NewSHA1(f.a, []byte("synthetic checkin after re-consent"))
	if e := r.timers.Apply(ctx, []c.FollowUpOp{op}); e != nil {
		wp12Err(t, e)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	total := 0
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, e := r.timers.Due(context.Background(), due.Add(time.Second), 10)
			if e != nil {
				var ce *c.Error
				if errors.As(e, &ce) {
					t.Error(ce.Cause)
				} else {
					t.Error(e)
				}
			}
			mu.Lock()
			total += len(got)
			mu.Unlock()
		}()
	}
	wg.Wait()
	if total != 1 {
		t.Fatalf("dispatches %d", total)
	}
	var count int
	if e := db.QueryRow(`SELECT COUNT(*) FROM dbo.ai_outbox WHERE user_id=@p1 AND type=N'follow_up_due'`, f.a).Scan(&count); e != nil || count != 1 {
		t.Fatal("durable wakeup missing")
	}
	// New open internal session is closed by timeout, preserving conversation.
	a := wp12Complete(t, f)
	if _, e := db.Exec(`UPDATE dbo.ai_sessions SET focus_thread_id=@p1,last_activity_at=DATEADD(hour,-1,SYSUTCDATETIME()) WHERE id=@p2 AND user_id=@p3`, task.ThreadID, a.SessionID, f.a); e != nil {
		t.Fatal(e)
	}
	if e := r.repo14.Sweep(context.Background(), task.Bundle, now.Add(2*time.Minute), 100); e != nil {
		wp12Err(t, e)
	}
	if e := r.repo14.Sweep(context.Background(), task.Bundle, now.Add(2*time.Minute), 100); e != nil {
		wp12Err(t, e)
	}
	if e := db.QueryRow(`SELECT COUNT(*) FROM dbo.ai_sessions WHERE id=@p1 AND closed_at IS NOT NULL`, a.SessionID).Scan(&count); e != nil || count != 1 {
		t.Fatal("idle session did not close")
	}
	if e := db.QueryRow(`SELECT COUNT(*) FROM dbo.conversations WHERE id=@p1 AND deleted_at IS NULL`, f.ca).Scan(&count); e != nil || count != 1 {
		t.Fatal("internal close hid conversation")
	}
}

func wp14Importance(v float32) *float32 { return &v }

func TestWP14SQLLongitudinalEvidenceVersionsAndAtomicMemory(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f, r, task, ids := wp14Seed(t, db)
	sources := []c.ID{ids[0]}
	// Three independently completed user situations in three internal sessions.
	for i := 0; i < 2; i++ {
		a := wp12Complete(t, f)
		sources = append(sources, a.UserMessageID)
		if _, e := db.Exec(`UPDATE dbo.ai_sessions SET focus_thread_id=@p1,closed_at=SYSUTCDATETIME(),close_reason=N'test',version=version+1 WHERE id=@p2 AND user_id=@p3`, task.ThreadID, a.SessionID, f.a); e != nil {
			wp12Err(t, e)
		}
	}
	for index, source := range sources {
		var turn c.ID
		if e := db.QueryRow(`SELECT CONVERT(varchar(36),turn_id) FROM dbo.messages WHERE id=@p1`, source).Scan(&turn); e != nil {
			t.Fatal(e)
		}
		ctx := mem.WithMeta(context.Background(), mem.Meta{UserID: f.a, TurnID: turn, OperationID: c.NewID(), Bundle: task.Bundle, ThreadID: &task.ThreadID})
		r.embed(6)
		op := c.MemoryOpProposal{Op: c.OpAdd, Kind: c.MemEpisode, Epistemic: c.EpReported, Content: "synthetic independent situation " + source.String()[:4], Entities: []c.EntityRef{{Role: []string{"friend", "boss", "partner"}[index]}}, EvidenceMsg: []c.ID{source}, Tags: []string{"synthetic_event"}, Sensitivity: c.SensNormal, ReasonTag: "synthetic"}
		report, e := r.service.Apply(ctx, f.a, []c.MemoryOpProposal{op}, wp12Origin(r.wp12Runtime))
		if e != nil || len(report.Applied) != 1 {
			wp12Err(t, e)
			t.Fatalf("memory result %+v", report)
		}
	}
	// Avoid near-duplicate merging of the synthetic setup's distinct episodes.
	// Same-message duplicates never count as additional situations or sessions.
	snapshot, e := r.repo14.Load(context.Background(), task)
	if e != nil {
		wp12Err(t, e)
	}
	if len(snapshot.Items) != 3 {
		t.Fatalf("expected 3 source episodes, got %d", len(snapshot.Items))
	}
	result := c.ReflectionResult{SchemaVersion: c.ReflectionSchemaVersion, ThreadID: task.ThreadID, Formulation: &c.FormulationDoc{Framework: "synthetic_framework", Fields: map[string]c.FormulationField{"pattern": {Value: "may show a synthetic pattern", Status: c.EpInferred, Evidence: []c.ID{sources[0], sources[1]}}}, Paragraph: "must be rebuilt"}, HypothesisOps: []c.MemoryOpProposal{{Op: c.OpAdd, Kind: c.MemHypothesis, Epistemic: c.EpInferred, Content: "may show a synthetic pattern", EvidenceMsg: []c.ID{sources[0]}, Tags: []string{"synthetic_pattern"}, Sensitivity: c.SensNormal, ReasonTag: "synthetic"}}, ThreadProposals: []c.ThreadProposal{}, InformationGaps: []string{}}
	raw, _ := json.Marshal(result)
	r.provider.QueueStructured("wp05-generator", fake.StructuredReply{JSON: raw})
	r.embed(8)
	task.Trigger = "needs_formulation"
	if e := r.worker14.Schedule(context.Background(), task); e != nil {
		wp12Err(t, e)
	}
	if worked, e := wp14Run(t, r.worker14); !worked || e != nil {
		wp12Err(t, e)
		t.Fatal("not run")
	}
	var fields string
	var version int
	if e := db.QueryRow(`SELECT version,fields_json FROM dbo.ai_formulations WHERE user_id=@p1 AND thread_id=@p2`, f.a, task.ThreadID).Scan(&version, &fields); e != nil {
		t.Fatal(e)
	}
	var flds map[string]c.FormulationField
	_ = json.Unmarshal([]byte(fields), &flds)
	if version != 1 || flds["pattern"].Status != c.EpInferred {
		t.Fatal("longitudinal formulation rejected")
	}
	var hyp c.ID
	if e := db.QueryRow(`SELECT CONVERT(varchar(36),id) FROM dbo.ai_memory_items WHERE user_id=@p1 AND kind=N'hypothesis'`, f.a).Scan(&hyp); e != nil {
		t.Fatal(e)
	}
	result.HypothesisOps = []c.MemoryOpProposal{{Op: c.OpConfirm, TargetID: &hyp, EvidenceMsg: []c.ID{sources[1]}, ReasonTag: "synthetic"}}
	raw, _ = json.Marshal(result)
	r.provider.QueueStructured("wp05-generator", fake.StructuredReply{JSON: raw})
	if e := r.worker14.Schedule(context.Background(), task); e != nil {
		wp12Err(t, e)
	}
	if worked, e := wp14Run(t, r.worker14); !worked || e != nil {
		wp12Err(t, e)
		t.Fatal("second reflection")
	}
	var status, epistemic string
	var support, sessions int
	if e := db.QueryRow(`SELECT status,epistemic,support_n,sessions_n FROM dbo.ai_memory_items WHERE id=@p1 AND user_id=@p2`, hyp, f.a).Scan(&status, &epistemic, &support, &sessions); e != nil {
		t.Fatal(e)
	}
	if status != "supported" || epistemic != "inferred" || support != 2 || sessions != 2 {
		t.Fatalf("hypothesis FSM: %s %s %d %d", status, epistemic, support, sessions)
	}
	var count int
	if e := db.QueryRow(`SELECT COUNT(*) FROM dbo.ai_formulations WHERE thread_id=@p1 AND user_id=@p2`, task.ThreadID, f.a).Scan(&count); e != nil || count != 2 {
		t.Fatal("formulation history not preserved")
	}
}
