package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

func TestWP12SQLConsolidationMaintenanceConsentAndProvenance(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	t.Run("conversation_hiding_preserves_independent_memory_index_eligibility", func(t *testing.T) {
		f := aiSetup(t, db)
		r := wp12New(t, db)
		a := wp12Complete(t, f)
		wp12Enable(t, r, f.a)
		id := wp12Apply(t, r, a, wp12Op(a.UserMessageID, "Synthetic independently retained item."))
		aiExec(t, db, `UPDATE dbo.conversations SET deleted_at=SYSUTCDATETIME(),deleted_by=@p2,deletion_reason=N'user_request' WHERE id=@p1 AND user_id=@p2`, f.ca, f.a)
		if _, e := wp12RunReady(r); e != nil {
			wp12Err(t, e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND deleted_at IS NULL`, id) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE user_id=@p1 AND ref_id=@p2`, f.a, id) != 1 {
			t.Fatal("conversation hide invalidated permitted derived memory")
		}
	})
	t.Run("state_delta_support_handoff_is_atomic_with_extraction", func(t *testing.T) {
		f := aiSetup(t, db)
		r := wp12New(t, db)
		first := wp12Complete(t, f)
		wp12Enable(t, r, f.a)
		op := wp12Op(first.UserMessageID, "May prefer a synthetic recurring label.")
		op.Kind = c.MemHypothesis
		op.Tags = []string{"synthetic_pattern"}
		id := wp12Apply(t, r, first, op)
		if _, e := wp12RunReady(r); e != nil {
			wp12Err(t, e)
		}
		next := wp12Complete(t, f)
		metadata := mem.Meta{UserID: f.a, TurnID: next.TurnID, OperationID: next.TurnID, Bundle: r.bundle.Reference(), Signals: []c.HypothesisSignal{{Op: c.HypSupport, ID: &id, Evidence: []c.ID{next.UserMessageID}}}}
		job, e := r.service.PlanTurnJob(mem.WithMeta(context.Background(), metadata), next, c.UnifiedTurn{})
		if e != nil || job == nil {
			t.Fatal(job, e)
		}
		if e = r.repo.Enqueue(context.Background(), f.a, next.TurnID, *job); e != nil {
			t.Fatal(e)
		}
		r.provider.QueueStructured("synthetic-memory-writer", fake.StructuredReply{JSON: []byte(`{"schema_version":"memwriter.v1","ops":[]}`)})
		if worked, e := wp12RunReady(r); !worked || e != nil {
			wp12Err(t, e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND support_n=2 AND sessions_n=1 AND status=N'proposed'`, id) != 1 {
			t.Fatal("typed signal missing or promoted without independent sessions")
		}
	})
	t.Run("bounded_episode_consolidation_then_leased_maintenance", func(t *testing.T) {
		f := aiSetup(t, db)
		r := wp12New(t, db)
		a := wp12Complete(t, f)
		wp12Enable(t, r, f.a)
		var assistant c.ID
		if e := db.QueryRow(`SELECT CONVERT(varchar(36),id) FROM dbo.messages WHERE turn_id=@p1 AND role=N'assistant'`, a.TurnID).Scan(&assistant); e != nil {
			t.Fatal(e)
		}
		op := wp12Op(a.UserMessageID, "Synthetic episode summary.")
		op.Kind = c.MemEpisode
		op.EvidenceMsg = append(op.EvidenceMsg, assistant)
		raw, _ := json.Marshal(c.MemoryWriterOutput{SchemaVersion: c.MemoryWriterSchemaVersion, Ops: []c.MemoryOpProposal{op}})
		r.provider.QueueStructured("synthetic-memory-writer", fake.StructuredReply{JSON: raw})
		r.embed(10)
		ctx := r.ctx(f.a, a.TurnID)
		if e := r.service.Consolidate(ctx, f.a, a.TurnID, []c.ID{a.UserMessageID, assistant}); e != nil {
			wp12Err(t, e)
		}
		if e := r.service.Consolidate(ctx, f.a, a.TurnID, []c.ID{a.UserMessageID, assistant}); e != nil {
			t.Fatal(e)
		}
		if worked, e := wp12RunReady(r); e != nil || !worked {
			wp12Err(t, e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1 AND kind=N'episode' AND support_n=1`, f.a) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_evidence WHERE user_id=@p1 AND ref_id=@p2 AND role=N'context'`, f.a, assistant) != 1 {
			t.Fatal("episode provenance incorrect")
		}
		if _, e := wp12RunReady(r); e != nil {
			wp12Err(t, e)
		}
		// Age engineering fixture metadata; production maintenance never rewrites
		// source timestamps, content, evidence or original embedding bytes.
		aiExec(t, db, `UPDATE dbo.ai_memory_items SET importance=0.1,created_at=DATEADD(day,-200,SYSUTCDATETIME()),last_confirmed_at=NULL WHERE user_id=@p1`, f.a)
		if e := r.service.ScheduleMaintenance(r.ctx(f.a, a.TurnID), f.a, time.Now()); e != nil {
			t.Fatal(e)
		}
		if worked, e := wp12RunReady(r); e != nil || !worked {
			wp12Err(t, e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1 AND status=N'stale' AND content=@p2`, f.a, op.Content) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE user_id=@p1`, f.a) != 1 {
			t.Fatal("staleness erased originals")
		}
		before := len(r.provider.History())
		if _, e := r.service.Maintain(r.ctx(f.a, a.TurnID), f.a, time.Now()); e != nil || len(r.provider.History()) != before {
			t.Fatal("maintenance replay made provider call", e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_formulations WHERE user_id=@p1`, f.a) != 0 {
			t.Fatal("WP-14 executed")
		}
	})
	t.Run("independent_support_endorsement_and_revoked_pending_job", func(t *testing.T) {
		f := aiSetup(t, db)
		r := wp12New(t, db)
		a := wp12Complete(t, f)
		wp12Enable(t, r, f.a)
		op := wp12Op(a.UserMessageID, "May prefer the synthetic recurring label.")
		op.Kind = c.MemHypothesis
		op.Tags = []string{"synthetic_pattern"}
		id := wp12Apply(t, r, a, op)
		req := f.req()
		req.Locale = "en"
		req.TZ = "UTC"
		req.ConversationID = f.ca2
		a2 := admit(t, f, req)
		p := prepared(a2)
		p.Jobs = nil
		if _, e := f.store.Complete(context.Background(), a2, p); e != nil {
			t.Fatal(e)
		}
		op.EvidenceMsg = []c.ID{a2.UserMessageID}
		if got := wp12Apply(t, r, a2, op); got != id {
			t.Fatal("hypothesis not deduplicated")
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND status=N'supported' AND support_n=2 AND sessions_n=2 AND confidence=N'high' AND epistemic=N'inferred'`, id) != 1 {
			t.Fatal("independent support did not promote")
		}
		r.embed(10)
		endorsed, e := r.service.Endorse(r.ctx(f.a, a2.TurnID), f.a, id, a2.UserMessageID, "Synthetic user confirms the label.")
		if e != nil || len(endorsed.Applied) != 1 || len(endorsed.Rejected) > 0 {
			wp12Err(t, e)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND kind=N'hypothesis' AND epistemic=N'inferred'`, id) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_evidence WHERE item_id=@p1 AND ref_kind=N'item' AND ref_id=@p2 AND role=N'context'`, endorsed.Applied[0], id) != 1 {
			t.Fatal("endorsement overwrote inference")
		}
		task := mem.Task{Schema: "memory_job.v1", Bundle: r.bundle.Reference(), OperationID: c.NewID()}
		task.Data, _ = r.cipher.Seal(context.Background(), f.a, "memory-task:"+a2.TurnID.String(), []byte(`{}`))
		payload, _ := json.Marshal(task)
		if e = r.repo.Enqueue(context.Background(), f.a, a2.TurnID, c.AsyncJobProposal{Kind: "memory_write", DedupeKey: task.OperationID.String(), Payload: payload}); e != nil {
			t.Fatal(e)
		}
		if e = r.service.SetControls(r.ctx(f.a, c.ID{}), f.a, nil, map[string]bool{"synthetic_memory": false}, nil); e != nil {
			t.Fatal(e)
		}
		before := len(r.provider.History())
		for i := 0; i < 3; i++ {
			_, _ = wp12RunReady(r)
		}
		if len(r.provider.History()) != before || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'memory_write' AND last_error_code=N'memory_denied' AND status=N'failed'`, f.a) != 1 {
			t.Fatal("revoked pending extraction called model")
		}
	})
	t.Run("source_verification_budget_and_retrieval_feedback", func(t *testing.T) {
		f := aiSetup(t, db)
		r := wp12New(t, db)
		a := wp12Complete(t, f)
		wp12Enable(t, r, f.a)
		id := wp12Apply(t, r, a, wp12Op(a.UserMessageID, "Synthetic feedback target."))
		input, e := r.repo.WriterInput(context.Background(), mem.Meta{UserID: f.a, TurnID: a.TurnID, OperationID: c.NewID(), Bundle: r.bundle.Reference()}, mem.Task{Bundle: r.bundle.Reference()})
		if e != nil {
			wp12Err(t, e)
		}
		original := input.Messages[0].Role
		input.Messages[0].Role = c.RoleAssistant
		if original == c.RoleAssistant {
			input.Messages[0].Role = c.RoleUser
		}
		if e = r.repo.VerifyWriter(context.Background(), mem.Meta{UserID: f.a, TurnID: a.TurnID, OperationID: c.NewID(), Bundle: r.bundle.Reference()}, input); !errors.Is(e, mem.ErrDenied) {
			t.Fatal("relabeled source accepted", e)
		}
		if _, e = wp12RunReady(r); e != nil {
			wp12Err(t, e)
		}
		// Accepted turns with no extraction still contribute feedback without an LLM.
		var initial float32
		if e = db.QueryRow(`SELECT importance FROM dbo.ai_memory_items WHERE id=@p1`, id).Scan(&initial); e != nil {
			t.Fatal(e)
		}
		for i := 0; i < 4; i++ {
			req := f.req()
			req.Locale = "en"
			req.TZ = "UTC"
			next := admit(t, f, req)
			feedback := mem.Feedback{Retrieved: []c.ID{id}}
			if i == 3 {
				feedback.Referenced = []c.ID{id}
			}
			ctx := mem.WithMeta(context.Background(), mem.Meta{UserID: f.a, TurnID: next.TurnID, OperationID: next.TurnID, Bundle: r.bundle.Reference(), Feedback: feedback})
			job, e := r.service.PlanFeedbackJob(ctx, next)
			if e != nil || job == nil || job.Kind != "memory_feedback" {
				t.Fatal(job, e)
			}
			p := prepared(next)
			p.Jobs = []c.AsyncJobProposal{*job}
			if _, e = f.store.Complete(context.Background(), next, p); e != nil {
				t.Fatal(e)
			}
			before := len(r.provider.History())
			if worked, e := wp12RunReady(r); !worked || e != nil {
				wp12Err(t, e)
			}
			if len(r.provider.History()) != before {
				t.Fatal("feedback used model")
			}
		}
		var importance float32
		var used sql.NullTime
		if e = db.QueryRow(`SELECT importance,last_used_at FROM dbo.ai_memory_items WHERE id=@p1`, id).Scan(&importance, &used); e != nil {
			t.Fatal(e)
		}
		if importance >= initial || !used.Valid {
			t.Fatal("retrieval feedback not applied", importance, initial)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_feedback WHERE user_id=@p1`, f.a) != 4 {
			t.Fatal("feedback receipts missing")
		}
		if _, e = db.Exec(`DELETE FROM dbo.ai_memory_feedback WHERE user_id=@p1`, f.a); e == nil {
			t.Fatal("physical feedback deletion allowed")
		}
		// Quota exhaustion and incompatible bundles cannot consume a queued writer.
		task := mem.Task{Schema: "memory_job.v1", Bundle: r.bundle.Reference(), OperationID: c.NewID()}
		task.Data, _ = r.cipher.Seal(context.Background(), f.a, "memory-task:"+a.TurnID.String(), []byte(`{}`))
		payload, _ := json.Marshal(task)
		if e = r.repo.Enqueue(context.Background(), f.a, a.TurnID, c.AsyncJobProposal{Kind: "memory_write", DedupeKey: "quota-fixture", Payload: payload}); e != nil {
			t.Fatal(e)
		}
		aiExec(t, db, `UPDATE dbo.ai_jobs SET attempts=30,max_attempts=100,locked_at=SYSUTCDATETIME() WHERE user_id=@p1 AND kind=N'memory_write'`, f.a)
		if l, e := r.repo.Claim(context.Background(), "quota-worker", time.Minute, r.bundle.Reference(), 30); e != nil || l != nil && l.Kind == "memory_write" {
			t.Fatal("quota bypass", e)
		}
		ref := r.bundle.Reference()
		ref.Version = "kb@other"
		if l, e := r.repo.Claim(context.Background(), "wrong-bundle", time.Minute, ref, 30); e != nil || l != nil && l.Kind != "memory_embed" {
			t.Fatal("bundle drift", e)
		}
	})
}

func TestWP12SQLUpgradePreservesExistingData(t *testing.T) {
	db := aiDatabase(t)
	dir := t.TempDir()
	files, e := filepath.Glob("../../db/migrations/*.sql")
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range files {
		if filepath.Base(p) >= "0021_" {
			continue
		}
		raw, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, filepath.Base(p)), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e = migrations.Apply(context.Background(), db, dir); e != nil {
		t.Fatal(e)
	}
	f := aiSetup(t, db)
	legacy := legacyChatNew(db)
	req := f.req()
	claim, e := legacy.ClaimTurn(context.Background(), f.a, f.ca, req.ClientTurnID, req.Text)
	if e != nil {
		t.Fatal(e)
	}
	historical, e := legacy.CompleteTurn(context.Background(), f.a, f.ca, req.ClientTurnID, claim.LeaseID, req.Text, "Synthetic historical reply", "historical-fixture")
	if e != nil {
		t.Fatal(e)
	}
	a := c.Admission{UserMessageID: historical.UserMessageID}
	id := c.NewID()
	aiExec(t, db, `INSERT dbo.ai_memory_items(id,user_id,kind,content,epistemic,status,confidence,sensitivity,importance,source_json) VALUES(@p1,@p2,N'fact',N'Synthetic pre-WP12 memory',N'reported',N'active',N'low',N'normal',0.2,N'{}')`, id, f.a)
	aiExec(t, db, `INSERT dbo.ai_memory_evidence(user_id,item_id,ref_kind,ref_id,role,evidence_at) VALUES(@p1,@p2,N'message',@p3,N'supports',SYSUTCDATETIME())`, f.a, id, a.UserMessageID)
	before := aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages`)
	aiMigrate(t, db)
	aiMigrate(t, db)
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages`) != before || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND version=1 AND pinned=0 AND content=N'Synthetic pre-WP12 memory'`, id) != 1 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_evidence WHERE item_id=@p1`, id) != 1 {
		t.Fatal("upgrade altered retained data")
	}
}
