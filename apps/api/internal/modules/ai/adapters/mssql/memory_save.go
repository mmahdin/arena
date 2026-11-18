package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type memoryBody struct {
	content          any
	cipher           []byte
	entities, tags   string
	embedding        []byte
	embeddingPayload string
}
type memoryEmbeddingTask struct {
	Schema string      `json:"schema"`
	Bundle c.BundleRef `json:"bundle"`
	ItemID c.ID        `json:"item_id"`
	Data   []byte      `json:"data_enc"`
}
type memoryEmbeddingData struct {
	Model     string
	Vector    []float32
	ClaimHash string
}

func arrayJSON[T any](xs []T) string {
	if xs == nil {
		return "[]"
	}
	return memoryJSON(xs)
}
func (r *MemoryRepository) prepareBody(ctx context.Context, record mem.Record, ref c.BundleRef) (body memoryBody, err error) {
	item := record.Item
	body.content = item.Content
	body.entities = arrayJSON(item.Entities)
	body.tags = arrayJSON(item.Tags)
	if item.Sensitivity == c.SensHighly {
		plain, _ := json.Marshal(struct {
			Content  string
			Entities []c.EntityRef
			Tags     []string
		}{item.Content, item.Entities, item.Tags})
		body.cipher, err = r.cipher.Seal(ctx, item.UserID, "memory-item:"+item.ID.String(), plain)
		if err != nil {
			return body, mem.ErrDenied
		}
		body.content = nil
		body.entities = "[]"
		body.tags = "[]"
	}
	if len(record.Vector) > 0 {
		body.embedding, err = EncodeEmbedding(r.model, record.Vector)
		if err != nil {
			return body, err
		}
		plain, _ := json.Marshal(memoryEmbeddingData{Model: r.model, Vector: record.Vector, ClaimHash: mem.ClaimHash(item.Content)})
		cipher, e := r.cipher.Seal(ctx, item.UserID, "memory-embedding:"+item.ID.String(), plain)
		if e != nil {
			return body, mem.ErrDenied
		}
		body.embeddingPayload = memoryJSON(memoryEmbeddingTask{Schema: "memory_embed.v1", Bundle: ref, ItemID: item.ID, Data: cipher})
	}
	return body, nil
}

// Archive before every mutation. Original content/ciphertext, metadata and
// derived vectors remain recoverable; no row or reference is physically erased.
func archiveMemory(ctx context.Context, tx *sql.Tx, user, item c.ID) error {
	_, e := tx.ExecContext(ctx, `INSERT dbo.ai_memory_versions(user_id,item_id,version,snapshot_json,content_enc,vector_model,vector_bytes)
 SELECT m.user_id,m.id,m.version,(SELECT m.* FOR JSON PATH,WITHOUT_ARRAY_WRAPPER),m.content_enc,e.model_id,e.vector_bytes FROM dbo.ai_memory_items m
 OUTER APPLY(SELECT TOP(1) model_id,vector_bytes FROM dbo.ai_embeddings e WHERE e.user_id=m.user_id AND e.ref_kind=N'memory' AND e.ref_id=m.id ORDER BY created_at DESC) e
 WHERE m.id=@p1 AND m.user_id=@p2 AND NOT EXISTS(SELECT 1 FROM dbo.ai_memory_versions v WHERE v.item_id=m.id AND v.user_id=m.user_id AND v.version=m.version)`, item, user)
	return e
}
func (r *MemoryRepository) Save(ctx context.Context, m mem.Meta, s mem.Snapshot, plan mem.Plan, hash string) error {
	return r.save(ctx, m, s, plan, hash, nil)
}

// Extension is adapter-local: no SQL types cross application contracts.
func (r *MemoryRepository) save(ctx context.Context, m mem.Meta, s mem.Snapshot, plan mem.Plan, hash string, extension func(*sql.Tx) error) (err error) {
	defer func() { err = memorySQLError(err) }()
	if m.Job != nil && (m.Job.UserID != m.UserID || m.Job.TurnID != m.TurnID) {
		return mem.ErrDenied
	}
	sort.SliceStable(plan.Changes, func(i, j int) bool { return plan.Changes[i].Before == nil && plan.Changes[j].Before != nil })
	bodies := make([]memoryBody, len(plan.Changes))
	for i, change := range plan.Changes {
		if change.After.Item.UserID != m.UserID || change.After.DeletedAt != nil {
			return mem.ErrDenied
		}
		bodies[i], err = r.prepareBody(ctx, change.After, m.Bundle)
		if err != nil {
			return err
		}
	}
	tx, e := r.begin(ctx, m.UserID)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var revision, userVersion int64
	if e = tx.QueryRowContext(ctx, `SELECT memory_revision,version FROM dbo.ai_user_state WHERE user_id=@p1`, m.UserID).Scan(&revision, &userVersion); e != nil {
		return e
	}
	if revision != s.Revision || userVersion != s.User.Version {
		return mem.ErrConflict
	}
	var existing string
	e = tx.QueryRowContext(ctx, `SELECT result_json FROM dbo.ai_memory_batches WHERE user_id=@p1 AND operation_id=@p2`, m.UserID, m.OperationID).Scan(&existing)
	if e == nil {
		return mem.ErrConflict
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return e
	}
	// Recheck all proposed evidence under the same owner lock as deletion and
	// consent commands. Retained hidden historical evidence is not removed.
	ids := []c.ID{}
	for _, change := range plan.Changes {
		for _, ev := range change.After.Item.Evidence {
			if ev.Kind == "message" {
				if _, ok := s.Sources[ev.Ref]; ok {
					ids = append(ids, ev.Ref)
				}
			}
		}
	}
	fresh, e := memorySources(ctx, tx, m.UserID, m.TurnID, ids)
	if e != nil {
		return e
	}
	for _, id := range ids {
		old, ok := s.Sources[id]
		n, exists := fresh[id]
		if !ok || !exists || n.Hidden != old.Hidden || n.Future != old.Future || n.Message.Content != old.Message.Content || n.Message.Role != old.Message.Role {
			return mem.ErrConflict
		}
	}
	if m.RequireCompleted {
		var status string
		if e = tx.QueryRowContext(ctx, `SELECT status FROM dbo.turns WHERE id=@p1 AND user_id=@p2`, m.TurnID, m.UserID).Scan(&status); e != nil || status != "completed" {
			return mem.ErrDenied
		}
	}
	for i, change := range plan.Changes {
		item := change.After.Item
		body := bodies[i]
		if change.Before == nil {
			_, e = tx.ExecContext(ctx, `INSERT dbo.ai_memory_items(id,user_id,kind,content,content_enc,lang,epistemic,status,confidence,support_n,contradict_n,sessions_n,thread_ids_json,entities_json,tags_json,sensitivity,importance,valid_from,valid_to,supersedes_id,superseded_by_id,cluster_id,source_json,created_at,updated_at,last_confirmed_at,last_used_at,version,pinned)
 VALUES(@p1,@p2,@p3,@p4,@p5,@p6,@p7,@p8,@p9,@p10,@p11,@p12,@p13,@p14,@p15,@p16,@p17,@p18,@p19,@p20,@p21,@p22,@p23,@p24,@p25,@p26,@p27,@p28,@p29)`, item.ID, m.UserID, item.Kind, body.content, body.cipher, item.Lang, item.Epistemic, item.Status, item.Confidence, item.SupportN, item.ContradictN, item.SessionsN, arrayJSON(item.ThreadIDs), body.entities, body.tags, item.Sensitivity, item.Importance, item.ValidFrom, item.ValidTo, item.Supersedes, item.SupersededBy, item.ClusterID, memoryJSON(item.Source), item.CreatedAt, item.UpdatedAt, item.LastConfirmedAt, item.LastUsedAt, change.After.Version, change.After.Pinned)
		} else {
			if e = archiveMemory(ctx, tx, m.UserID, item.ID); e != nil {
				return e
			}
			res, updateErr := tx.ExecContext(ctx, `UPDATE dbo.ai_memory_items SET content=@p4,content_enc=@p5,epistemic=@p6,status=@p7,confidence=@p8,support_n=@p9,contradict_n=@p10,sessions_n=@p11,entities_json=@p12,tags_json=@p13,sensitivity=@p14,importance=@p15,valid_to=@p16,supersedes_id=@p17,superseded_by_id=@p18,last_confirmed_at=@p19,last_used_at=@p20,updated_at=@p21,version=@p22,pinned=@p23,thread_ids_json=@p24,cluster_id=@p25 WHERE id=@p1 AND user_id=@p2 AND version=@p3 AND deleted_at IS NULL`, item.ID, m.UserID, change.Before.Version, body.content, body.cipher, item.Epistemic, item.Status, item.Confidence, item.SupportN, item.ContradictN, item.SessionsN, body.entities, body.tags, item.Sensitivity, item.Importance, item.ValidTo, item.Supersedes, item.SupersededBy, item.LastConfirmedAt, item.LastUsedAt, item.UpdatedAt, change.After.Version, change.After.Pinned, arrayJSON(item.ThreadIDs), item.ClusterID)
			if updateErr != nil {
				return updateErr
			}
			n, _ := res.RowsAffected()
			if n != 1 {
				return mem.ErrConflict
			}
		}
		if e != nil {
			return e
		}
	}
	// An explicit authenticated edit/endorsement is its own source. Historical
	// message evidence remains context; it cannot falsely support edited text.
	explicit := false
	for _, change := range plan.Changes {
		if change.Before == nil && change.After.Item.Source.Origin == "user_edit" {
			explicit = true
		}
	}
	if explicit {
		_, e = tx.ExecContext(ctx, `INSERT dbo.ai_turn_events(id,user_id,turn_id,type,actor,delta_json) VALUES(@p1,@p2,@p3,N'explicit_confirmation',N'user',@p4)`, m.OperationID, m.UserID, memoryNullableID(m.TurnID), memoryJSON(plan.Report))
		if e != nil {
			return e
		}
	}
	// Insert links only after all new targets exist. Polymorphic owner triggers
	// and composite FKs check every reference. Existing links are append-only.
	for i, change := range plan.Changes {
		item := change.After.Item
		for _, ev := range item.Evidence {
			var session any
			if source, ok := s.Sources[ev.Ref]; ok && c.ValidID(source.SessionID) {
				session = source.SessionID
			}
			_, e = tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_memory_evidence WHERE item_id=@p1 AND user_id=@p2 AND ref_kind=@p3 AND ref_id=@p4 AND role=@p5) INSERT dbo.ai_memory_evidence(item_id,user_id,ref_kind,ref_id,role,session_id,evidence_at) VALUES(@p1,@p2,@p3,@p4,@p5,@p6,@p7)`, item.ID, m.UserID, ev.Kind, ev.Ref, ev.Role, session, ev.At)
			if e != nil {
				return e
			}
		}
		if body := bodies[i]; body.embeddingPayload != "" && (change.Before == nil || len(change.Before.Vector) == 0) {
			_, e = tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'memory_embed' AND dedupe_key=@p2) INSERT dbo.ai_jobs(id,user_id,turn_id,kind,dedupe_key,payload_json) VALUES(@p3,@p1,@p4,N'memory_embed',@p2,@p5)`, m.UserID, item.ID.String()+":"+r.model, c.NewID(), memoryNullableID(m.TurnID), body.embeddingPayload)
			if e != nil {
				return e
			}
		}
	}
	if e = r.applyFeedback(ctx, tx, m, s, m.Feedback); e != nil {
		return e
	}
	for _, id := range plan.Loops {
		_, e = tx.ExecContext(ctx, `IF NOT EXISTS(SELECT 1 FROM dbo.ai_open_loops WHERE user_id=@p1 AND kind=N'confirm_fact' AND ref_id=@p2 AND status=N'open') INSERT dbo.ai_open_loops(id,user_id,kind,text,ref_id) VALUES(@p3,@p1,N'confirm_fact',N'confirm_fact',@p2)`, m.UserID, id, c.NewID())
		if e != nil {
			return e
		}
	}
	for _, cd := range plan.Cooldowns {
		if e = saveCooldown(ctx, tx, m.UserID, cd); e != nil {
			return e
		}
	}
	for _, receipt := range plan.Receipts {
		_, e = tx.ExecContext(ctx, `INSERT dbo.ai_memory_ops(user_id,scope_id,op_hash,result_json) VALUES(@p1,@p2,@p3,@p4)`, m.UserID, scope(m), hashBytes(receipt.Hash), memoryJSON(receipt))
		if e != nil {
			return e
		}
	}
	_, e = tx.ExecContext(ctx, `INSERT dbo.ai_memory_batches(user_id,operation_id,request_hash,result_json) VALUES(@p1,@p2,@p3,@p4); UPDATE dbo.ai_user_state SET memory_revision=memory_revision+1 WHERE user_id=@p1`, m.UserID, m.OperationID, hashBytes(hash), memoryJSON(plan.Report))
	if e != nil {
		return e
	}
	if extension != nil {
		if e = extension(tx); e != nil {
			return e
		}
	}
	if m.Job != nil {
		if e = completeMemoryJob(ctx, tx, *m.Job); e != nil {
			return e
		}
	}
	return tx.Commit()
}
func saveCooldown(ctx context.Context, tx *sql.Tx, user c.ID, cd mem.Cooldown) error {
	_, e := tx.ExecContext(ctx, `UPDATE dbo.ai_hypothesis_cooldowns SET until_at=CASE WHEN until_at<@p3 THEN @p3 ELSE until_at END,rejected_item_id=@p4 WHERE user_id=@p1 AND tag=@p2 AND thread_id IS NULL; IF @@ROWCOUNT=0 INSERT dbo.ai_hypothesis_cooldowns(user_id,tag,until_at,rejected_item_id) VALUES(@p1,@p2,@p3,@p4)`, user, cd.Tag, cd.Until, cd.Item)
	return e
}
