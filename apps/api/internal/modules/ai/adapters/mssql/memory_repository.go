package mssql

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type MemoryRepository struct {
	db        *sql.DB
	cipher    mem.Cipher
	policy    mem.Policy
	model     string
	dimension int
	feedback  mem.FeedbackPolicy
}

var _ mem.Repository = (*MemoryRepository)(nil)

func NewMemoryRepository(db *sql.DB, cipher mem.Cipher, policy mem.Policy, model string, dimension int, feedback mem.FeedbackPolicy) (*MemoryRepository, error) {
	if db == nil || cipher == nil || policy == nil || model == "" || dimension < 1 || dimension > 8192 || feedback.MinUnusedRetrievals < 2 || feedback.ImportanceMultiplier <= 0 || feedback.ImportanceMultiplier >= 1 || math.IsNaN(float64(feedback.ImportanceMultiplier)) || math.IsInf(float64(feedback.ImportanceMultiplier), 0) {
		return nil, mem.ErrAssets
	}
	return &MemoryRepository{db, cipher, policy, model, dimension, feedback}, nil
}
func memorySQLError(err error) error {
	if err == nil || errors.Is(err, mem.ErrDenied) || errors.Is(err, mem.ErrConflict) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return &c.Error{Code: c.ErrInternal, Step: "memory_sql", Cause: err}
}
func (r *MemoryRepository) begin(ctx context.Context, user c.ID) (*sql.Tx, error) {
	if !c.ValidID(user) {
		return nil, mem.ErrDenied
	}
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	var id c.ID
	if e = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.users WITH(UPDLOCK,HOLDLOCK) WHERE id=@p1`, user).Scan(&id); e != nil {
		_ = tx.Rollback()
		return nil, mem.ErrDenied
	}
	return tx, nil
}
func scope(m mem.Meta) c.ID {
	if c.ValidID(m.TurnID) {
		return m.TurnID
	}
	return m.OperationID
}
func memoryNullableID(id c.ID) any {
	if !c.ValidID(id) {
		return nil
	}
	return id
}
func hashBytes(hash string) []byte { b, _ := hex.DecodeString(hash); return b }
func memoryJSON(v any) string      { raw, _ := json.Marshal(v); return string(raw) }

const memoryRecordsSQL = `SELECT TOP(1001)
 (SELECT CONVERT(varchar(36),m.id) id,CONVERT(varchar(36),m.user_id) user_id,m.kind,m.content,COALESCE(m.lang,N'') lang,m.epistemic,m.status,m.confidence,m.support_n,m.contradict_n,m.sessions_n,
 JSON_QUERY(m.thread_ids_json) thread_ids,JSON_QUERY(m.entities_json) entities,JSON_QUERY(m.tags_json) tags,m.sensitivity,m.importance,
 CONVERT(varchar(33),m.valid_from,127)+N'Z' valid_from,CONVERT(varchar(33),m.valid_to,127)+N'Z' valid_to,
 CONVERT(varchar(33),m.created_at,127)+N'Z' created_at,CONVERT(varchar(33),m.updated_at,127)+N'Z' updated_at,
 CONVERT(varchar(33),m.last_confirmed_at,127)+N'Z' last_confirmed_at,CONVERT(varchar(33),m.last_used_at,127)+N'Z' last_used_at,
 CONVERT(varchar(36),m.supersedes_id) supersedes,CONVERT(varchar(36),m.superseded_by_id) superseded_by,CONVERT(varchar(36),m.cluster_id) cluster_id,JSON_QUERY(m.source_json) source,
 JSON_QUERY((SELECT ev.ref_kind kind,CONVERT(varchar(36),ev.ref_id) ref,ev.role,CONVERT(varchar(33),ev.evidence_at,127)+N'Z' at FROM dbo.ai_memory_evidence ev WHERE ev.item_id=m.id AND ev.user_id=m.user_id FOR JSON PATH)) evidence FOR JSON PATH,WITHOUT_ARRAY_WRAPPER),
 m.version,m.pinned,m.deleted_at,CONVERT(varchar(36),m.deleted_by),m.deletion_reason,m.content_enc,emb.vector_bytes,emb.model_id
 FROM dbo.ai_memory_items m OUTER APPLY(SELECT TOP(1) vector_bytes,model_id FROM dbo.ai_embeddings e WHERE e.user_id=m.user_id AND e.ref_kind=N'memory' AND e.ref_id=m.id AND e.model_id=@p3 ORDER BY created_at DESC) emb
 WHERE m.user_id=@p1 AND (@p4=1 OR m.status IN(N'active',N'proposed',N'supported',N'contested') OR EXISTS(SELECT 1 FROM OPENJSON(@p2) ids WHERE TRY_CONVERT(uniqueidentifier,ids.value)=m.id)) AND m.deleted_at IS NULL ORDER BY m.id`

func (r *MemoryRepository) Snapshot(ctx context.Context, user c.ID, m mem.Meta, evidence []c.ID) (out mem.Snapshot, err error) {
	defer func() { err = memorySQLError(err) }()
	if m.UserID != user {
		return out, mem.ErrDenied
	}
	tx, e := r.begin(ctx, user)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if e = loadUser(ctx, tx, user, &out.User); e != nil {
		return out, mem.ErrDenied
	}
	if e = tx.QueryRowContext(ctx, `SELECT memory_revision FROM dbo.ai_user_state WHERE user_id=@p1`, user).Scan(&out.Revision); e != nil {
		return out, e
	}
	if c.ValidID(m.TurnID) {
		var status string
		if e = tx.QueryRowContext(ctx, `SELECT t.status FROM dbo.turns t JOIN dbo.conversations cv ON cv.id=t.conversation_id AND cv.user_id=t.user_id WHERE t.id=@p1 AND t.user_id=@p2 AND t.lifecycle_version=1`, m.TurnID, user).Scan(&status); e != nil || m.RequireCompleted && status != "completed" {
			return out, mem.ErrDenied
		}
	}
	if m.ThreadID != nil {
		var id c.ID
		if e = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.ai_threads WHERE id=@p1 AND user_id=@p2`, *m.ThreadID, user).Scan(&id); e != nil {
			return out, mem.ErrDenied
		}
		out.ThreadID = &id
	}
	rows, e := tx.QueryContext(ctx, memoryRecordsSQL, user, arrayJSON(m.Targets), r.model, m.IncludeInactive)
	if e != nil {
		return out, e
	}
	var encrypted [][]byte
	for rows.Next() {
		var record mem.Record
		var doc string
		var deleted sql.NullTime
		var actor, reason, model sql.NullString
		var ciphertext, vector []byte
		if e = rows.Scan(&doc, &record.Version, &record.Pinned, &deleted, &actor, &reason, &ciphertext, &vector, &model); e != nil {
			rows.Close()
			return out, e
		}
		if e = json.Unmarshal([]byte(doc), &record.Item); e != nil {
			rows.Close()
			return out, e
		}
		if record.Item.Lang == "" {
			record.Item.Lang = out.User.Locale
		}
		if deleted.Valid {
			record.DeletedAt = &deleted.Time
		}
		if actor.Valid {
			id, e := uuid.Parse(actor.String)
			if e != nil {
				rows.Close()
				return out, e
			}
			record.DeletedBy = &id
		}
		if reason.Valid {
			record.DeletionReason = &reason.String
		}
		if len(vector) > 0 {
			record.Vector, e = DecodeEmbedding(1, model.String, r.model, r.dimension, r.dimension, vector)
			if e != nil {
				rows.Close()
				return out, e
			}
			record.VectorModel = model.String
		}
		out.Records = append(out.Records, record)
		encrypted = append(encrypted, ciphertext)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	if len(out.Records) > 1000 {
		return out, &c.Error{Code: c.ErrBudget, Step: "memory_snapshot"}
	}
	// Count retained historical evidence even when a source is hidden; Hidden
	// prevents using it as NEW evidence or disclosing it to a provider.
	all := append([]c.ID(nil), evidence...)
	for _, record := range out.Records {
		for _, ev := range record.Item.Evidence {
			if ev.Kind == "message" {
				all = append(all, ev.Ref)
			}
		}
	}
	out.Sources, e = memorySources(ctx, tx, user, m.TurnID, all)
	if e != nil {
		return out, e
	}
	out.Events = map[c.ID]mem.EventSource{}
	eventIDs := []c.ID{}
	for _, record := range out.Records {
		for _, evidence := range record.Item.Evidence {
			if evidence.Kind == "event" {
				eventIDs = append(eventIDs, evidence.Ref)
			}
		}
	}
	rows, e = tx.QueryContext(ctx, `SELECT CONVERT(varchar(36),e.id),e.actor,e.type,e.created_at,CONVERT(varchar(36),tc.session_id) FROM dbo.ai_turn_events e LEFT JOIN dbo.ai_turn_context tc ON tc.turn_id=e.turn_id AND tc.user_id=e.user_id WHERE e.user_id=@p1 AND EXISTS(SELECT 1 FROM OPENJSON(@p2) ids WHERE TRY_CONVERT(uniqueidentifier,ids.value)=e.id)`, user, arrayJSON(eventIDs))
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var id c.ID
		var event mem.EventSource
		var session sql.NullString
		if e = rows.Scan(&id, &event.Actor, &event.Type, &event.At, &session); e != nil {
			rows.Close()
			return out, e
		}
		if session.Valid {
			if e = event.SessionID.UnmarshalText([]byte(session.String)); e != nil {
				rows.Close()
				return out, e
			}
		}
		out.Events[id] = event
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	out.Suppressed = map[string]bool{}
	rows, e = tx.QueryContext(ctx, `SELECT CONVERT(varchar(64),claim_hash,2) FROM dbo.ai_memory_tombstones WHERE user_id=@p1 AND claim_hash IS NOT NULL AND (suppress_until IS NULL OR suppress_until>SYSUTCDATETIME())`, user)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var hash string
		if e = rows.Scan(&hash); e != nil {
			rows.Close()
			return out, e
		}
		out.Suppressed[strings.ToLower(hash)] = true
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	out.Cooldowns = map[string]time.Time{}
	rows, e = tx.QueryContext(ctx, `SELECT tag,MAX(until_at) FROM dbo.ai_hypothesis_cooldowns WHERE user_id=@p1 GROUP BY tag`, user)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var tag string
		var until time.Time
		if e = rows.Scan(&tag, &until); e != nil {
			rows.Close()
			return out, e
		}
		out.Cooldowns[tag] = until
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	out.Ledger = map[string]mem.Receipt{}
	rows, e = tx.QueryContext(ctx, `SELECT result_json FROM dbo.ai_memory_ops WHERE user_id=@p1 AND scope_id=@p2`, user, scope(m))
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var raw string
		var receipt mem.Receipt
		if e = rows.Scan(&raw); e != nil {
			rows.Close()
			return out, e
		}
		if e = json.Unmarshal([]byte(raw), &receipt); e != nil {
			rows.Close()
			return out, e
		}
		out.Ledger[receipt.Hash] = receipt
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	if e = tx.Commit(); e != nil {
		return out, e
	}
	for i, raw := range encrypted {
		if len(raw) == 0 {
			continue
		}
		plain, e := r.cipher.Open(ctx, user, "memory-item:"+out.Records[i].Item.ID.String(), raw)
		if e != nil {
			return mem.Snapshot{}, mem.ErrDenied
		}
		var private struct {
			Content  string
			Entities []c.EntityRef
			Tags     []string
		}
		if json.Unmarshal(plain, &private) != nil {
			return mem.Snapshot{}, mem.ErrDenied
		}
		out.Records[i].Item.Content = private.Content
		out.Records[i].Item.Entities = private.Entities
		out.Records[i].Item.Tags = private.Tags
	}
	return out, nil
}
func memorySources(ctx context.Context, tx *sql.Tx, user, turn c.ID, ids []c.ID) (map[c.ID]mem.Source, error) {
	result := map[c.ID]mem.Source{}
	rows, e := tx.QueryContext(ctx, `SELECT CONVERT(varchar(36),m.id),CONVERT(varchar(36),cv.user_id),CONVERT(varchar(36),m.conversation_id),CONVERT(varchar(36),m.turn_id),m.role,m.content,m.created_at,COALESCE(NULLIF(JSON_VALUE(t.request_meta_json,'$.locale'),N''),(SELECT locale FROM dbo.ai_user_state WHERE user_id=@p1),N''),CONVERT(varchar(36),COALESCE(tc.session_id,s.id)),CASE WHEN m.deleted_at IS NOT NULL OR cv.deleted_at IS NOT NULL THEN 1 ELSE 0 END,CASE WHEN @p3 IS NOT NULL AND t.admission_order>(SELECT admission_order FROM dbo.turns WHERE id=@p3 AND user_id=@p1) THEN 1 ELSE 0 END FROM dbo.messages m JOIN dbo.conversations cv ON cv.id=m.conversation_id AND cv.user_id=@p1 JOIN dbo.turns t ON t.id=m.turn_id AND t.user_id=cv.user_id LEFT JOIN dbo.ai_turn_context tc ON tc.turn_id=t.id AND tc.user_id=cv.user_id OUTER APPLY(SELECT TOP(1) id FROM dbo.ai_sessions s WHERE s.user_id=cv.user_id AND s.conversation_id=cv.id ORDER BY started_at DESC) s WHERE cv.user_id=@p1 AND EXISTS(SELECT 1 FROM OPENJSON(@p2) ids WHERE TRY_CONVERT(uniqueidentifier,ids.value)=m.id)`, user, arrayJSON(ids), memoryNullableID(turn))
	if e != nil {
		return result, e
	}
	defer rows.Close()
	for rows.Next() {
		var s mem.Source
		var session sql.NullString
		if e = rows.Scan(&s.Message.ID, &s.Message.UserID, &s.Message.ConvID, &s.Message.TurnID, &s.Message.Role, &s.Message.Content, &s.Message.CreatedAt, &s.Message.Lang, &session, &s.Hidden, &s.Future); e != nil {
			return result, e
		}
		if session.Valid {
			id, e := uuid.Parse(session.String)
			if e != nil {
				return result, e
			}
			s.SessionID = id
			s.Message.SessionID = &id
		}
		result[s.Message.ID] = s
	}
	return result, rows.Err()
}
func (r *MemoryRepository) Receipt(ctx context.Context, user, operation c.ID, hash string) (out ports.MemoryApplyReport, done bool, err error) {
	defer func() { err = memorySQLError(err) }()
	var stored []byte
	var raw string
	e := r.db.QueryRowContext(ctx, `SELECT request_hash,result_json FROM dbo.ai_memory_batches WHERE user_id=@p1 AND operation_id=@p2`, user, operation).Scan(&stored, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return out, false, nil
	}
	if e != nil {
		return out, false, e
	}
	if !reflect.DeepEqual(stored, hashBytes(hash)) {
		return out, false, mem.ErrConflict
	}
	if json.Unmarshal([]byte(raw), &out) != nil {
		return out, false, mem.ErrDenied
	}
	return out, true, nil
}
