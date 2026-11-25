package mssql

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	bg "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/background"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"time"
)

func (r *IntelligenceRepository) Commit(ctx context.Context, l mem.Lease, p bg.Prepared, out bg.Output, usage c.LLMUsage, plan mem.Plan, hints []bg.Hint) (err error) {
	ctx, span := trace.Start(ctx, "background.Commit", func() any {
		return map[string]any{"plan": plan, "formulations": out.Formulations, "hints": hints, "origin": p.Input.Request.Origin}
	})
	defer func() { span.Complete(nil, err, recover()) }()
	// A visibility-only deletion is not semantic suppression. New independent
	// evidence can create a distinct record without reviving the hidden row.
	for i, change := range plan.Changes {
		if change.Before != nil {
			continue
		}
		var deleted int
		if e := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE user_id=@p1 AND id=@p2 AND deleted_at IS NOT NULL`, l.UserID, change.After.Item.ID).Scan(&deleted); e != nil {
			return e
		}
		if deleted == 0 {
			continue
		}
		old := change.After.Item.ID
		evidence := []c.ID{}
		for _, ev := range change.After.Item.Evidence {
			if ev.Kind == "message" && ev.Role == "supports" {
				evidence = append(evidence, ev.Ref)
			}
		}
		replacement := uuid.NewSHA1(old, []byte("independent-evidence:"+mem.Hash(evidence)))
		plan.Changes[i].After.Item.ID = replacement
		for j := range plan.Changes {
			if plan.Changes[j].After.Item.SupersededBy != nil && *plan.Changes[j].After.Item.SupersededBy == old {
				id := replacement
				plan.Changes[j].After.Item.SupersededBy = &id
			}
		}
		for j, id := range plan.Report.Applied {
			if id == old {
				plan.Report.Applied[j] = replacement
			}
		}
		trace.Capture(ctx, "independent_identity", "identity_resolution", "retained_deletion", map[string]c.ID{"original": old, "new": replacement})
	}
	var task struct {
		Bundle c.BundleRef `json:"bundle"`
	}
	if json.Unmarshal(l.Payload, &task) != nil {
		return mem.ErrDenied
	}
	m := mem.Meta{UserID: l.UserID, TurnID: l.TurnID, OperationID: l.ID, Job: &l, Bundle: task.Bundle, ResearchVersions: p.Sources, RequireCompleted: c.ValidID(l.TurnID)}
	hintBodies := [][]byte{}
	for i, hint := range hints {
		future := c.FutureHint{Topic: hint.Topic, ResourceIDs: hint.ResourceIDs, RemainingTurns: hint.TTLTurns, ExpiresAt: time.Now().UTC().Add(72 * time.Hour), JobID: &l.ID, Origin: p.Input.Request.Kind, EvidenceIDs: hint.EvidenceIDs}
		if len(p.Input.Request.Scope.ThreadIDs) == 1 {
			future.ThreadID = &p.Input.Request.Scope.ThreadIDs[0]
		}
		raw, _ := json.Marshal(future)
		body, e := r.cipher.Seal(ctx, l.UserID, "background-hint:"+uuid.NewSHA1(l.ID, []byte(fmt.Sprint(i))).String(), raw)
		if e != nil {
			return e
		}
		hintBodies = append(hintBodies, body)
	}
	resultRaw, _ := json.Marshal(out)
	resultCipher, e := r.cipher.Seal(ctx, l.UserID, "intelligence-result:"+l.ID.String(), resultRaw)
	if e != nil {
		return e
	}
	return r.save(ctx, m, p.Snapshot, plan, mem.Hash(out), func(tx *sql.Tx) error {
		var active int
		if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_user_turn_leases WITH(HOLDLOCK) WHERE user_id=@p1 AND lease_expires_at>SYSUTCDATETIME()`, l.UserID).Scan(&active); e != nil {
			return e
		}
		if active > 0 {
			return mem.ErrConflict
		}
		// Thread and formulation versions are fenced in the SAME transaction as
		// memory mutations, coverage acknowledgement, hints and job completion.
		for _, old := range p.Input.Threads {
			var version int64
			var fv int
			if e := tx.QueryRowContext(ctx, `SELECT version,formulation_version FROM dbo.ai_threads WITH(UPDLOCK,HOLDLOCK) WHERE user_id=@p1 AND id=@p2`, l.UserID, old.ID).Scan(&version, &fv); e != nil {
				return e
			}
			if version != old.Version || fv != old.FormulationVersion {
				return mem.ErrConflict
			}
		}
		changed := []c.ID{}
		for _, change := range plan.Changes {
			if change.Before != nil && (change.Before.Item.Content != change.After.Item.Content || change.After.Item.Status == c.MemContested || change.After.Item.Status == c.MemRejected || change.After.Item.Status == c.MemSuperseded) {
				changed = append(changed, change.After.Item.ID)
			}
		}
		if e := invalidateUnderstanding(ctx, tx, l.UserID, changed); e != nil {
			return e
		}
		refs := sourceRevisions(p.Sources)
		// Same-analysis hypotheses are context dependencies, never primary proof.
		for _, change := range plan.Changes {
			if change.After.Item.Kind == c.MemHypothesis {
				refs = append(refs, c.SourceRevision{Kind: "memory", ID: change.After.Item.ID})
			}
		}
		for i, ref := range refs {
			var version string
			er := tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(16),source_version,2) FROM dbo.ai_search_sources WHERE user_id=@p1 AND source_kind=@p2 AND id=@p3`, l.UserID, ref.Kind, ref.ID).Scan(&version)
			if er == nil {
				refs[i].Version = version
			} else if er != sql.ErrNoRows {
				return er
			}
		}
		for _, form := range out.Formulations {
			var version int
			if e := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0)+1 FROM dbo.ai_formulations WITH(UPDLOCK,HOLDLOCK) WHERE user_id=@p1 AND ((@p2 IS NULL AND thread_id IS NULL) OR thread_id=@p2)`, l.UserID, form.ThreadID).Scan(&version); e != nil {
				return e
			}
			if _, e := tx.ExecContext(ctx, `INSERT dbo.ai_formulations(id,user_id,thread_id,version,framework,fields_json,paragraph,gaps_json,scope_json,sources_json) VALUES(@p1,@p2,@p3,@p4,@p5,@p6,@p7,@p8,@p9,@p10)`, uuid.NewSHA1(l.ID, []byte("formulation:"+fmt.Sprint(version)+":"+mem.Hash(form))), l.UserID, form.ThreadID, version, form.Doc.Framework, memoryJSON(form.Doc.Fields), form.Doc.Paragraph, arrayJSON(out.InformationGaps), memoryJSON(p.Input.Request.Scope), memoryJSON(refs)); e != nil {
				return e
			}
			if form.ThreadID != nil {
				if _, e := tx.ExecContext(ctx, `UPDATE dbo.ai_threads SET formulation_version=@p3,version=version+1 WHERE user_id=@p1 AND id=@p2`, l.UserID, *form.ThreadID, version); e != nil {
					return e
				}
			}
		}
		if p.Input.Request.Kind == "reflection" {
			var thread any
			var version int64
			if len(p.Input.Request.Scope.ThreadIDs) == 1 {
				thread = p.Input.Request.Scope.ThreadIDs[0]
				for _, t := range p.Input.Threads {
					if t.ID == thread {
						version = t.Version
					}
				}
			}
			if _, e := tx.ExecContext(ctx, `INSERT dbo.ai_reflection_runs(id,user_id,thread_id,session_id,job_id,trigger_kind,thread_version,result_enc,provenance_json) VALUES(@p1,@p2,@p3,NULL,@p1,@p4,@p5,@p6,@p7)`, l.ID, l.UserID, thread, p.Input.Request.Origin, version, resultCipher, memoryJSON(map[string]any{"model": usage.Model, "bundle": task.Bundle, "scope": p.Input.Request.Scope, "origin": p.Input.Request.Origin})); e != nil {
				return e
			}
		}
		for i, hint := range hints {
			relevant := map[c.ID]bool{}
			for _, id := range hint.EvidenceIDs {
				relevant[id] = true
			}
			for _, id := range hint.MemoryIDs {
				relevant[id] = true
			}
			for _, change := range plan.Changes {
				if change.After.Item.Kind == c.MemHypothesis {
					relevant[change.After.Item.ID] = true
				}
			}
			hintRefs := []c.SourceRevision{}
			for _, ref := range refs {
				if relevant[ref.ID] {
					hintRefs = append(hintRefs, ref)
				}
			}

			var thread any
			if len(p.Input.Request.Scope.ThreadIDs) == 1 {
				thread = p.Input.Request.Scope.ThreadIDs[0]
			}
			id := uuid.NewSHA1(l.ID, []byte(fmt.Sprint(i)))
			if _, e := tx.ExecContext(ctx, `INSERT dbo.ai_background_hints(id,user_id,job_id,thread_id,hint_enc,sources_json,ttl_turns,expires_at) VALUES(@p1,@p2,@p3,@p4,@p5,@p6,@p7,DATEADD(hour,72,SYSUTCDATETIME()))`, id, l.UserID, l.ID, thread, hintBodies[i], memoryJSON(hintRefs), hint.TTLTurns); e != nil {
				return e
			}
		}
		if p.Input.Request.Kind == "memory_intelligence" {
			for _, message := range p.Input.Request.MessageIDs {
				var version []byte
				if e := tx.QueryRowContext(ctx, `SELECT CONVERT(binary(8),m.search_version) FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id AND v.user_id=@p1 WHERE m.id=@p2 AND m.role=N'user' AND m.deleted_at IS NULL AND v.deleted_at IS NULL`, l.UserID, message).Scan(&version); e != nil {
					return mem.ErrConflict
				}
				present := false
				for _, source := range p.Sources {
					if source.Source == "message" && source.ID == message {
						old, _ := hex.DecodeString(source.SourceVersion)
						if string(old) != string(version) {
							return mem.ErrConflict
						}
						present = true
					}
				}
				if !present {
					var already int
					if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_memory_review_messages WHERE user_id=@p1 AND message_id=@p2 AND reviewed_at IS NOT NULL AND source_version=@p3`, l.UserID, message, version).Scan(&already); e != nil {
						return e
					}
					if already == 1 {
						continue
					}
					return mem.ErrDenied
				}
				if _, e := tx.ExecContext(ctx, `UPDATE dbo.ai_memory_review_messages SET reviewed_at=SYSUTCDATETIME(),reviewed_by_job_id=@p3,source_version=@p4 WHERE user_id=@p1 AND message_id=@p2; IF @@ROWCOUNT=0 INSERT dbo.ai_memory_review_messages(user_id,message_id,job_id,source_version,reviewed_at,reviewed_by_job_id) VALUES(@p1,@p2,@p3,@p4,SYSUTCDATETIME(),@p3)`, l.UserID, message, l.ID, version); e != nil {
					return e
				}
			}
		}
		return nil
	})
}
func sourceRevisions(sources []c.MemoryView) []c.SourceRevision {
	out := []c.SourceRevision{}
	seen := map[string]bool{}
	for _, s := range sources {
		key := s.Source + s.ID.String()
		if !seen[key] {
			seen[key] = true
			out = append(out, c.SourceRevision{Kind: s.Source, ID: s.ID, Version: s.SourceVersion})
		}
	}
	return out
}
func (r *MemoryRepository) ResetCheckpoint(ctx context.Context, l mem.Lease) error {
	result, e := r.db.ExecContext(ctx, `UPDATE dbo.ai_jobs SET memory_result_enc=NULL WHERE id=@p1 AND user_id=@p2 AND memory_lease_token=@p3 AND status=N'running' AND memory_lease_until>SYSUTCDATETIME() AND (JSON_VALUE(payload_json,'$.schema')=N'intelligence_job.v2' OR kind=N'reflection')`, l.ID, l.UserID, l.Token)
	if e != nil {
		return e
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return mem.ErrConflict
	}
	return nil
}

// Invalidation preserves original rows and versions. Dependencies are explicit
// context links; no similarity or psychological inference is performed here.
func invalidateUnderstanding(ctx context.Context, tx *sql.Tx, user c.ID, changed []c.ID) error {
	if len(changed) == 0 {
		return nil
	}
	rows, e := tx.QueryContext(ctx, `DECLARE @affected TABLE(id uniqueidentifier PRIMARY KEY); INSERT @affected SELECT DISTINCT TRY_CONVERT(uniqueidentifier,value) FROM OPENJSON(@p2); DECLARE @n int=1,@depth int=0; WHILE @n>0 AND @depth<32 BEGIN INSERT @affected SELECT DISTINCT e.item_id FROM dbo.ai_memory_evidence e JOIN @affected a ON e.ref_id=a.id JOIN dbo.ai_memory_items m ON m.id=e.item_id AND m.user_id=@p1 WHERE e.user_id=@p1 AND e.ref_kind=N'item' AND m.kind=N'hypothesis' AND NOT EXISTS(SELECT 1 FROM @affected old WHERE old.id=e.item_id); SET @n=@@ROWCOUNT; SET @depth=@depth+1; END; SELECT CONVERT(varchar(36),m.id) FROM dbo.ai_memory_items m JOIN @affected a ON a.id=m.id WHERE m.user_id=@p1 AND m.kind=N'hypothesis' AND m.status IN(N'proposed',N'supported',N'active') AND NOT EXISTS(SELECT 1 FROM OPENJSON(@p2) seed WHERE TRY_CONVERT(uniqueidentifier,seed.value)=m.id)`, user, arrayJSON(changed))
	if e != nil {
		return e
	}
	ids := []c.ID{}
	for rows.Next() {
		var id c.ID
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range ids {
		if e = archiveMemory(ctx, tx, user, id); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE dbo.ai_memory_items SET status=N'stale',version=version+1,updated_at=SYSUTCDATETIME() WHERE user_id=@p1 AND id=@p2`, user, id); e != nil {
			return e
		}
	}
	all := append(append([]c.ID{}, changed...), ids...)
	_, e = tx.ExecContext(ctx, `UPDATE f SET analysis_status=N'stale' FROM dbo.ai_formulations f WHERE f.user_id=@p1 AND f.analysis_status=N'active' AND EXISTS(SELECT 1 FROM OPENJSON(f.sources_json) s JOIN OPENJSON(@p2) changed ON JSON_VALUE(s.value,'$.id')=changed.value); UPDATE h SET invalidated_at=SYSUTCDATETIME() FROM dbo.ai_background_hints h WHERE h.user_id=@p1 AND h.invalidated_at IS NULL AND EXISTS(SELECT 1 FROM OPENJSON(h.sources_json) s JOIN OPENJSON(@p2) changed ON JSON_VALUE(s.value,'$.id')=changed.value)`, user, arrayJSON(all))
	return e
}
