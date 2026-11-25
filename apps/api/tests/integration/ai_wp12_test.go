package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/memorycrypto"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type wp12Bundle struct {
	syntheticBundle
	ref c.BundleRef
}

func (b wp12Bundle) Reference() c.BundleRef { return b.ref }

type wp12Policy struct{}

func (wp12Policy) Allow(_ context.Context, p mem.Permission) (bool, error) {
	if p.Action == mem.UseRetainedSource {
		return false, nil
	}
	if p.Action == mem.ExplicitChange || p.Action == mem.Endorse {
		return p.Origin.Origin == "user_edit", nil
	}
	return p.User.Consent["synthetic_memory"], nil
}

type wp12Runtime struct {
	db       *sql.DB
	repo     *aidb.MemoryRepository
	service  *mem.Service
	worker   *mem.Worker
	provider *fake.Provider
	bundle   wp12Bundle
	cipher   *memorycrypto.Keyring
}

func wp12New(t *testing.T, db *sql.DB) wp12Runtime {
	t.Helper()
	// Existing source-index jobs belong to earlier subtest fixtures. Leave their
	// durable state intact while keeping this synthetic worker's script isolated.
	aiExec(t, db, `UPDATE dbo.ai_jobs SET run_after=DATEADD(day,1,SYSUTCDATETIME()) WHERE kind=N'memory_embed' AND status=N'queued'`)
	raw, e := os.ReadFile("../../internal/modules/ai/application/memory/testdata/assets.json")
	if e != nil {
		wp12Err(t, e)
	}
	var files map[string]string
	if json.Unmarshal(raw, &files) != nil {
		t.Fatal("fixture")
	}
	b := wp12Bundle{syntheticBundle: syntheticBundle{}, ref: c.BundleRef{Version: "kb@0.9.0", PromptSHAs: map[string]string{"memory_writer": c.NewID().String()}}}
	for p, v := range files {
		b.syntheticBundle[p] = []byte(v)
	}
	a, e := mem.Load(context.Background(), b, mem.LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		wp12Err(t, e)
	}
	cipher, e := memorycrypto.New("synthetic", map[string][]byte{"synthetic": bytes.Repeat([]byte{42}, 32)})
	if e != nil {
		wp12Err(t, e)
	}
	repo, e := aidb.NewMemoryRepository(db, cipher, wp12Policy{}, "synthetic-embedding", 3, mem.FeedbackPolicy{MinUnusedRetrievals: 3, ImportanceMultiplier: .9})
	if e != nil {
		wp12Err(t, e)
	}
	repo.WithSearchPolicy(syntheticUsePolicy{true})
	provider := fake.New()
	client := backgroundClient(t, provider)
	embed := client.(ports.EmbeddingClient)
	svc, e := mem.New(repo, a, mem.Options{Policy: wp12Policy{}, Cipher: cipher, Embedder: embed, EmbeddingModel: "synthetic-embedding", Dimension: 3})
	if e != nil {
		wp12Err(t, e)
	}
	writer, e := mem.NewWriter(client, a, repo, nil)
	if e != nil {
		wp12Err(t, e)
	}
	worker, e := mem.NewWorker(repo, repo, writer, svc, "synthetic-wp12")
	if e != nil {
		wp12Err(t, e)
	}
	return wp12Runtime{db: db, repo: repo, service: svc, worker: worker, provider: provider, bundle: b, cipher: cipher}
}
func (r wp12Runtime) ctx(user, turn c.ID) context.Context {
	return mem.WithMeta(context.Background(), mem.Meta{UserID: user, TurnID: turn, OperationID: c.NewID(), Bundle: r.bundle.Reference()})
}
func wp12Complete(t *testing.T, f aiFixture) c.Admission {
	t.Helper()
	req := f.req()
	req.Locale = "en"
	req.TZ = "UTC"
	a := admit(t, f, req)
	p := prepared(a)
	p.Jobs = nil
	if _, e := f.store.Complete(context.Background(), a, p); e != nil {
		wp12Err(t, e)
	}
	return a
}
func wp12Enable(t *testing.T, r wp12Runtime, user c.ID) {
	t.Helper()
	paused := false
	if e := r.service.SetControls(r.ctx(user, c.ID{}), user, &paused, map[string]bool{"synthetic_memory": true}, nil); e != nil {
		wp12Err(t, e)
	}
}
func wp12Op(message c.ID, content string) c.MemoryOpProposal {
	return c.MemoryOpProposal{Op: c.OpAdd, Kind: c.MemFact, Content: content, Epistemic: c.EpReported, Sensitivity: c.SensNormal, EvidenceMsg: []c.ID{message}, ReasonTag: "synthetic"}
}
func wp12Origin(r wp12Runtime) c.ProvenanceSrc {
	return c.ProvenanceSrc{Origin: "memory_writer", PromptVersion: r.bundle.Reference().Version, Model: "synthetic-memory-writer"}
}
func (r wp12Runtime) embed(n int) {
	for i := 0; i < n; i++ {
		r.provider.QueueEmbedding("synthetic-embedding", fake.EmbeddingReply{Vector: []float32{1, 0, 0}})
	}
}
func wp12Apply(t *testing.T, r wp12Runtime, a c.Admission, op c.MemoryOpProposal) c.ID {
	t.Helper()
	r.embed(6)
	report, e := r.service.Apply(r.ctx(a.UserID, a.TurnID), a.UserID, []c.MemoryOpProposal{op}, wp12Origin(r))
	if e != nil || len(report.Rejected) > 0 || len(report.Applied) != 1 {
		t.Fatal(report, e)
	}
	return report.Applied[0]
}
func wp12Err(t *testing.T, e error) {
	t.Helper()
	var ce *c.Error
	if errors.As(e, &ce) {
		t.Fatalf("%v: %v", e, ce.Cause)
	}
	t.Fatal(e)
}

// datetime2(3) can round a freshly queued run_after slightly into the future.
// Exercise the same polling behavior as Worker.Run instead of racing the clock.
func wp12RunReady(r wp12Runtime) (bool, error) {
	// Make primary fixture jobs due before polling; new independent index jobs
	// must not be mistaken for a not-yet-due extraction/feedback operation.
	_, _ = r.db.Exec(`UPDATE dbo.ai_jobs SET run_after=DATEADD(second,-1,SYSUTCDATETIME()) WHERE kind<>N'memory_embed' AND status=N'queued' AND JSON_VALUE(payload_json,'$.bundle.prompt_shas.memory_writer')=@p1`, r.bundle.Reference().PromptSHAs["memory_writer"])

	for i := 0; i < 20; i++ {
		worked, e := r.worker.RunOnce(context.Background())
		if worked || e != nil {
			return worked, e
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false, errors.New("no ready WP-12 job")
}

func TestWP12SQLMemoryLifecyclePrivacyAndRecovery(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	aiMigrate(t, db)
	t.Run("extraction_atomic_checkpoint_index_and_replay", func(t *testing.T) {
		f := aiSetup(t, db)
		r := wp12New(t, db)
		request := f.req()
		request.Locale = "en"
		request.TZ = "UTC"
		a := admit(t, f, request)
		wp12Enable(t, r, f.a)
		// Controls changed WP-08 version; reload the admitted version before this
		// synthetic finalizer. Production controls intentionally conflict stale turns.
		aiExec(t, db, `UPDATE dbo.ai_user_state SET locale=N'en' WHERE user_id=@p1`, f.a)
		if e := db.QueryRow(`SELECT version FROM dbo.ai_user_state WHERE user_id=@p1`, f.a).Scan(&a.Versions.User); e != nil {
			wp12Err(t, e)
		}
		u := c.UnifiedTurn{MemorySignals: []string{"synthetic_event"}}
		job, e := r.service.PlanTurnJob(context.Background(), a, u)
		if e != nil || job == nil {
			t.Fatal(job, e)
		}
		if bytes.Contains(job.Payload, []byte("turn_analysis")) {
			t.Fatal("unencrypted mental context in job")
		}
		op := wp12Op(a.UserMessageID, "Synthetic memory from a completed turn.")
		raw, _ := json.Marshal(c.MemoryWriterOutput{SchemaVersion: c.MemoryWriterSchemaVersion, Ops: []c.MemoryOpProposal{op}})
		r.provider.QueueStructured("synthetic-memory-writer", fake.StructuredReply{JSON: raw})
		r.embed(6)
		p := prepared(a)
		p.Jobs = []c.AsyncJobProposal{*job}
		result, e := f.store.Complete(context.Background(), a, p)
		if e != nil {
			wp12Err(t, e)
		}
		worked, e := wp12RunReady(r)
		if e != nil {
			wp12Err(t, e)
		}
		if !worked {
			t.Fatal("no memory worker claim")
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1 AND content=@p2`, f.a, op.Content) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_evidence WHERE user_id=@p1 AND ref_id=@p2 AND role=N'supports'`, f.a, a.UserMessageID) != 1 {
			t.Fatal("missing memory/provenance")
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE user_id=@p1`, f.a) != 0 {
			t.Fatal("index written in memory transaction")
		}
		worked, e = wp12RunReady(r)
		if e != nil || !worked {
			wp12Err(t, e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE user_id=@p1 AND ref_kind=N'memory'`, f.a) != 1 {
			t.Fatal("post-commit embedding missing")
		}
		before := len(r.provider.History())
		again, e := f.store.Complete(context.Background(), a, p)
		if e != nil || !reflect.DeepEqual(result, again) {
			t.Fatal("turn replay", e)
		}
		if len(r.provider.History()) != before {
			t.Fatal("turn replay called provider")
		}
		if _, e = r.worker.RunOnce(context.Background()); e != nil {
			wp12Err(t, e)
		}
		for _, call := range r.provider.History()[before:] {
			if call.Operation != "embedding" {
				t.Fatal("replayed memory writer work")
			}
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND status=N'completed' AND memory_result_enc IS NOT NULL`, f.a) != 1 {
			t.Fatal("encrypted checkpoint not retained")
		}
	})
	t.Run("soft_delete_controls_isolation_and_explicit_suppression", func(t *testing.T) {
		f := aiSetup(t, db)
		r := wp12New(t, db)
		a := wp12Complete(t, f)
		wp12Enable(t, r, f.a)
		aiExec(t, db, `UPDATE dbo.ai_user_state SET locale=N'en' WHERE user_id=@p1`, f.a)
		op := wp12Op(a.UserMessageID, "Synthetic retained memory.")
		id := wp12Apply(t, r, a, op)
		// Publish the vector before deleting, so retention checks cover both stores.
		if _, e := wp12RunReady(r); e != nil {
			wp12Err(t, e)
		}
		if e := r.service.Correct(r.ctx(f.b, c.ID{}), f.b, id, ports.CorrectionReject); e == nil {
			t.Fatal("foreign correction")
		}
		if _, e := r.service.Delete(r.ctx(f.b, c.ID{}), f.b, ports.DeletionScope{ItemIDs: []c.ID{id}}); e == nil {
			t.Fatal("foreign delete")
		}
		beforeEvidence := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_evidence WHERE item_id=@p1`, id)
		ctx := r.ctx(f.a, c.ID{})
		deleted, e := r.service.Delete(ctx, f.a, ports.DeletionScope{ItemIDs: []c.ID{id}})
		if e != nil || deleted.DeletedItems != 1 {
			t.Fatal(deleted, e)
		}
		again, e := r.service.Delete(ctx, f.a, ports.DeletionScope{ItemIDs: []c.ID{id}})
		if e != nil || !reflect.DeepEqual(again, deleted) {
			t.Fatal("delete retry", e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND deleted_at IS NOT NULL AND content=@p2`, id, op.Content) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_evidence WHERE item_id=@p1`, id) != beforeEvidence || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE user_id=@p1`, f.a) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_tombstones WHERE user_id=@p1`, f.a) != 0 {
			t.Fatal("soft delete erased/cascaded/suppressed")
		}
		items, e := r.service.List(r.ctx(f.a, c.ID{}), f.a)
		if e != nil || len(items) != 0 {
			t.Fatal("deleted memory visible", e)
		}
		// Visibility alone is not a suppression policy. Explicitly permitted new
		// evidence can create a separate record, retaining the hidden original.
		op.ReasonTag = "synthetic_new_claim"
		id2 := wp12Apply(t, r, a, op)
		if id2 == id {
			t.Fatal("revived tombstone")
		}
		if e = r.service.Suppress(r.ctx(f.a, c.ID{}), f.a, op.Content, nil); e != nil {
			wp12Err(t, e)
		}
		op.ReasonTag = "synthetic_suppressed_attempt"
		report, e := r.service.Apply(r.ctx(f.a, a.TurnID), f.a, []c.MemoryOpProposal{op}, wp12Origin(r))
		if e != nil || len(report.Rejected) != 1 || report.Rejected[0].RuleCode != "MC-009" {
			t.Fatal(report, e)
		}
		prefs := c.PrefsView{Language: "en", Length: c.LengthShort, Tone: c.ToneWarmSteady}
		paused := true
		if e = r.service.SetControls(r.ctx(f.a, c.ID{}), f.a, &paused, map[string]bool{"synthetic_memory": false}, &prefs); e != nil {
			wp12Err(t, e)
		}
		if e = r.service.Correct(r.ctx(f.a, c.ID{}), f.a, id2, ports.CorrectionContest); e != nil {
			t.Fatal("paused correction failed", e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND status=N'contested'`, id2) != 1 {
			t.Fatal("synchronous correction absent")
		}
		// Original messages, memories, evidence, vectors and historical revisions
		// all reject physical deletion. Each statement is rolled back by its guard.
		for _, q := range []string{`DELETE FROM dbo.ai_memory_items WHERE user_id=@p1`, `DELETE FROM dbo.ai_memory_evidence WHERE user_id=@p1`, `DELETE FROM dbo.ai_embeddings WHERE user_id=@p1`, `DELETE FROM dbo.messages WHERE conversation_id IN(SELECT id FROM dbo.conversations WHERE user_id=@p1)`, `DELETE FROM dbo.ai_memory_versions WHERE user_id=@p1`, `DELETE FROM dbo.ai_memory_batches WHERE user_id=@p1`, `DELETE FROM dbo.ai_memory_ops WHERE user_id=@p1`} {
			if _, e = db.Exec(q, f.a); e == nil {
				t.Fatal("physical deletion allowed", q)
			}
		}
	})
	t.Run("hypotheses_corrections_high_sensitivity_and_message_retention", func(t *testing.T) {
		f := aiSetup(t, db)
		r := wp12New(t, db)
		a := wp12Complete(t, f)
		wp12Enable(t, r, f.a)
		aiExec(t, db, `UPDATE dbo.ai_user_state SET locale=N'en' WHERE user_id=@p1`, f.a)
		op := wp12Op(a.UserMessageID, "May prefer a synthetic label.")
		op.Kind = c.MemHypothesis
		op.Tags = []string{"synthetic_pattern"}
		hyp := wp12Apply(t, r, a, op)
		if e := r.service.Correct(r.ctx(f.a, c.ID{}), f.a, hyp, ports.CorrectionContest); e != nil {
			wp12Err(t, e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_hypothesis_cooldowns WHERE user_id=@p1 AND rejected_item_id=@p2 AND until_at>SYSUTCDATETIME()`, f.a, hyp) != 1 {
			t.Fatal("missing rejection cooldown")
		}
		op.ReasonTag = "synthetic_repropose"
		report, e := r.service.Apply(r.ctx(f.a, a.TurnID), f.a, []c.MemoryOpProposal{op}, wp12Origin(r))
		if e != nil || len(report.Rejected) != 1 || report.Rejected[0].RuleCode != "MC-010" {
			t.Fatal("zombie hypothesis", report, e)
		}
		op = wp12Op(a.UserMessageID, "Synthetic volunteered sensitive record.")
		op.Sensitivity = c.SensHighly
		op.Tags = []string{"synthetic_event"}
		id := wp12Apply(t, r, a, op)
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND content IS NULL AND DATALENGTH(content_enc)>0 AND tags_json=N'[]'`, id) != 1 {
			t.Fatal("highly sensitive content not encrypted")
		}
		list, e := r.service.List(r.ctx(f.a, c.ID{}), f.a)
		if e != nil {
			wp12Err(t, e)
		}
		found := false
		for _, v := range list {
			if v.ID == id && v.Content == op.Content {
				found = true
			}
		}
		if !found {
			t.Fatal("owner cannot read own encrypted item")
		}
		if _, e = r.service.Delete(r.ctx(f.a, c.ID{}), f.a, ports.DeletionScope{MessageIDs: []c.ID{a.UserMessageID}}); e != nil {
			wp12Err(t, e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE id=@p1 AND content=N'Synthetic input' AND deleted_at IS NOT NULL`, a.UserMessageID) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND deleted_at IS NULL`, id) != 1 {
			t.Fatal("message deletion cascaded")
		}
		before := len(r.provider.History())
		op.Content = "Synthetic new unauthorized extraction."
		report, e = r.service.Apply(r.ctx(f.a, a.TurnID), f.a, []c.MemoryOpProposal{op}, wp12Origin(r))
		if e != nil || len(report.Rejected) != 1 || len(r.provider.History()) != before {
			t.Fatal("hidden source reached provider", report, e)
		}
	})
	t.Run("concurrent_adds_superseding_and_job_fencing", func(t *testing.T) {
		f := aiSetup(t, db)
		r := wp12New(t, db)
		a := wp12Complete(t, f)
		wp12Enable(t, r, f.a)
		aiExec(t, db, `UPDATE dbo.ai_user_state SET locale=N'en' WHERE user_id=@p1`, f.a)
		r.embed(30)
		op := wp12Op(a.UserMessageID, "Synthetic shared concurrent record.")
		var wg sync.WaitGroup
		results := make(chan error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				report, e := r.service.Apply(r.ctx(f.a, a.TurnID), f.a, []c.MemoryOpProposal{op}, wp12Origin(r))
				if e == nil && len(report.Applied) != 1 {
					e = errors.New("missing concurrent result")
				}
				results <- e
			}()
		}
		wg.Wait()
		close(results)
		for e := range results {
			if e != nil {
				wp12Err(t, e)
			}
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1`, f.a) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_ops WHERE user_id=@p1`, f.a) != 1 {
			t.Fatal("duplicate committed under concurrency")
		}
		items, e := r.service.List(r.ctx(f.a, c.ID{}), f.a)
		if e != nil {
			wp12Err(t, e)
		}
		id := items[0].ID
		op.Content = "Synthetic user corrected value."
		op.Tags = []string{"preferred_name"}
		edited, e := r.service.Edit(r.ctx(f.a, a.TurnID), f.a, id, op)
		if e != nil || len(edited.Rejected) > 0 {
			wp12Err(t, e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND status=N'superseded' AND content=N'Synthetic shared concurrent record.' AND superseded_by_id IS NOT NULL`, id) != 1 {
			t.Fatal("supersede did not retain old item")
		}
		lease, e := r.repo.Claim(context.Background(), "old-worker", time.Minute, r.bundle.Reference(), 30)
		if e != nil || lease == nil {
			wp12Err(t, e)
		}
		aiExec(t, db, `UPDATE dbo.ai_jobs SET memory_lease_until=DATEADD(second,-1,SYSUTCDATETIME()) WHERE id=@p1`, lease.ID)
		reclaimed, e := r.repo.Claim(context.Background(), "new-worker", time.Minute, r.bundle.Reference(), 30)
		if e != nil || reclaimed == nil || reclaimed.Token == lease.Token {
			t.Fatal("lease not reclaimed", e)
		}
		if e = r.repo.Complete(context.Background(), *lease); !errors.Is(e, mem.ErrConflict) {
			t.Fatal("stale token completed job", e)
		}
		if e = r.repo.Fail(context.Background(), *lease, "memory_retry", time.Second); !errors.Is(e, mem.ErrConflict) {
			t.Fatal("stale token reset job", e)
		}
		input, eligible, e := r.repo.EmbeddingInput(context.Background(), *reclaimed)
		if e != nil {
			wp12Err(t, e)
		}
		if !eligible {
			e = r.repo.Complete(context.Background(), *reclaimed)
		} else {
			e = r.repo.PublishEmbedding(context.Background(), *reclaimed, input, ports.EmbeddingResponse{Model: "synthetic-embedding", Vector: []float32{1, 0, 0}})
		}
		if e != nil {
			wp12Err(t, e)
		}
	})
	t.Run("rollback_then_checkpoint_recovery_without_second_llm", func(t *testing.T) {
		f := aiSetup(t, db)
		r := wp12New(t, db)
		a := wp12Complete(t, f)
		wp12Enable(t, r, f.a)
		aiExec(t, db, `UPDATE dbo.ai_user_state SET locale=N'en' WHERE user_id=@p1`, f.a)
		// Controlled test-only trigger causes a late SQL failure after extraction.
		aiExec(t, db, `CREATE TRIGGER dbo.wp12_test_failure ON dbo.ai_memory_evidence AFTER INSERT AS BEGIN THROW 51099,N'synthetic failure',1; END`)
		t.Cleanup(func() { _, _ = db.Exec(`DROP TRIGGER IF EXISTS dbo.wp12_test_failure`) })
		task := mem.Task{Schema: "memory_job.v1", Bundle: r.bundle.Reference(), OperationID: c.NewID()}
		task.Data, _ = r.cipher.Seal(context.Background(), f.a, "memory-task:"+a.TurnID.String(), []byte(`{}`))
		payload, _ := json.Marshal(task)
		if e := r.repo.Enqueue(context.Background(), f.a, a.TurnID, c.AsyncJobProposal{Kind: "memory_write", DedupeKey: task.OperationID.String(), Payload: payload}); e != nil {
			wp12Err(t, e)
		}
		op := wp12Op(a.UserMessageID, "Synthetic recoverable memory.")
		raw, _ := json.Marshal(c.MemoryWriterOutput{SchemaVersion: c.MemoryWriterSchemaVersion, Ops: []c.MemoryOpProposal{op}})
		r.provider.QueueStructured("synthetic-memory-writer", fake.StructuredReply{JSON: raw})
		r.embed(6)
		if worked, e := wp12RunReady(r); !worked || e == nil {
			t.Fatal("SQL failure not returned", worked, e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1`, f.a) != 0 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_ops WHERE user_id=@p1`, f.a) != 0 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'memory_embed' AND JSON_VALUE(payload_json,'$.source_kind')=N'memory'`, f.a) != 0 {
			t.Fatal("partial memory transaction survived")
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND status=N'queued' AND memory_result_enc IS NOT NULL`, f.a) != 1 {
			t.Fatal("checkpoint lost")
		}
		aiExec(t, db, `DROP TRIGGER dbo.wp12_test_failure`)
		var failedJob c.ID
		if e := db.QueryRow(`SELECT CONVERT(varchar(36),id) FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'memory_write'`, f.a).Scan(&failedJob); e != nil {
			t.Fatal(e)
		}
		aiExec(t, db, `UPDATE dbo.ai_jobs SET status=N'failed',attempts=max_attempts WHERE id=@p1 AND user_id=@p2`, failedJob, f.a)
		if e := r.service.RetryJob(r.ctx(f.b, c.ID{}), f.b, failedJob); e == nil {
			t.Fatal("foreign retry allowed")
		}
		if e := r.service.RetryJob(r.ctx(f.a, c.ID{}), f.a, failedJob); e != nil {
			t.Fatal(e)
		}
		if worked, e := wp12RunReady(r); !worked || e != nil {
			wp12Err(t, e)
		}
		calls := 0
		for _, v := range r.provider.History() {
			if v.Operation == "structured" {
				calls++
			}
		}
		if calls != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1`, f.a) != 1 {
			t.Fatal("recovery repeated extraction", calls)
		}
	})
}

// Missing routine scheduling is repaired independently of any Unified output.
func TestWP12SQLRecentReviewDoesNotRequireUnifiedRequest(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	r := wp12New(t, db)
	a := wp12Complete(t, f)
	wp12Enable(t, r, f.a)
	if e := r.service.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	if e := r.service.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	if got := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE turn_id=@p1 AND kind=N'memory_write' AND dedupe_key=@p2`, a.TurnID, a.TurnID.String()); got != 1 {
		t.Fatal("missing or duplicate routine review", got)
	}
	r.provider.QueueStructured("synthetic-memory-writer", fake.StructuredReply{JSON: []byte(`{"schema_version":"memwriter.v1","ops":[]}`)})
	if worked, e := wp12RunReady(r); !worked || e != nil {
		wp12Err(t, e)
	}
	if got := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE turn_id=@p1 AND kind=N'memory_write' AND status=N'completed'`, a.TurnID); got != 1 {
		t.Fatal("review did not complete", got)
	}
	if e := r.service.ReviewRecent(context.Background()); e != nil {
		t.Fatal(e)
	}
	if got := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE turn_id=@p1 AND kind=N'memory_write'`, a.TurnID); got != 1 {
		t.Fatal("completed review was requeued", got)
	}
}
