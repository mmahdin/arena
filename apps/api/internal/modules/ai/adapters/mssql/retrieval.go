package mssql

// WP-09 SQL Server 2022 retrieval projections are READ-ONLY. Every query is
// scoped to the authenticated owner and to the admitted trigger's ordering.
// Deleted raw messages and parents are hidden; memory items retain independent
// visibility, with AI-use permissions delegated to a separate required gate.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type retrievalQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type RetrievalRepository struct{ db retrievalQuerier }

var _ retrieval.Repository = (*RetrievalRepository)(nil)

func NewRetrievalRepository(db *sql.DB) (*RetrievalRepository, error) {
	if db == nil {
		return nil, errors.New("WP-09 requires a SQL Server connection")
	}
	return &RetrievalRepository{db: db}, nil
}

// NewRetrievalRepositoryTx permits rollback-only integration fixtures in an
// isolated SQL Server database; no production retrieval holds a transaction
// open across model calls.
func NewRetrievalRepositoryTx(tx *sql.Tx) (*RetrievalRepository, error) {
	if tx == nil {
		return nil, errors.New("WP-09 requires a SQL Server transaction")
	}
	return &RetrievalRepository{db: tx}, nil
}

func (r *RetrievalRepository) UserState(ctx context.Context, user c.ID) (retrieval.State, error) {
	if !c.ValidID(user) {
		return retrieval.State{}, errors.New("invalid retrieval owner")
	}
	var s retrieval.State
	err := r.db.QueryRowContext(ctx, `SELECT tz,memory_paused FROM dbo.ai_user_state WHERE user_id=@p1`, user).Scan(&s.TZ, &s.MemoryPaused)
	return s, err
}

const memorySQL = `SELECT TOP (@p3)
    CONVERT(varchar(36),m.id), CONVERT(varchar(36),m.user_id),m.kind,m.content,
    m.epistemic,m.status,m.sensitivity,m.importance,m.created_at,m.valid_from,
    m.thread_ids_json,m.entities_json,CONVERT(varchar(36),m.cluster_id),
    (SELECT COUNT(*) FROM dbo.ai_memory_evidence ev WHERE ev.item_id=m.id AND ev.user_id=m.user_id),
    emb.codec_version,emb.model_id,emb.dim,emb.vector_bytes
FROM dbo.ai_memory_items m
JOIN dbo.messages trigger_msg ON trigger_msg.id=@p2 AND trigger_msg.role=N'user'
JOIN dbo.conversations trigger_conv ON trigger_conv.id=trigger_msg.conversation_id AND trigger_conv.user_id=@p1 AND trigger_conv.deleted_at IS NULL
OUTER APPLY (
    SELECT TOP (1) v.codec_version,v.model_id,v.dim,v.vector_bytes
    FROM dbo.ai_embeddings v
    WHERE v.user_id=m.user_id AND v.ref_kind=N'memory' AND v.ref_id=m.id AND v.model_id=@p9
    ORDER BY v.created_at DESC,v.id
) emb
WHERE m.user_id=@p1 AND m.deleted_at IS NULL AND m.content IS NOT NULL
  AND m.updated_at<=trigger_msg.created_at
  AND ((m.kind=N'hypothesis' AND m.status IN(N'proposed',N'supported')) OR (m.kind<>N'hypothesis' AND m.status=N'active'))
  AND (@p4=N'' OR EXISTS (
      SELECT 1 FROM OPENJSON(m.thread_ids_json) jt WHERE TRY_CONVERT(uniqueidentifier,jt.value)=TRY_CONVERT(uniqueidentifier,@p4)))
  AND (@p5=N'' OR EXISTS (
      SELECT 1 FROM OPENJSON(m.entities_json) WITH ([role] nvarchar(100) '$.role') e WHERE e.[role]=@p5))
  AND (@p6=N'' OR EXISTS (
      SELECT 1 FROM OPENJSON(m.entities_json) WITH ([label] nvarchar(240) '$.label') e WHERE e.[label]=@p6))
  AND (@p7 IS NULL OR m.created_at>=@p7 OR m.last_confirmed_at>=@p7)
  AND (@p8=N'' OR CHARINDEX(@p8,m.content)>0 OR CHARINDEX(@p8,m.tags_json)>0)
ORDER BY m.importance DESC,m.updated_at DESC,m.id`

func (r *RetrievalRepository) Memories(ctx context.Context, q retrieval.MemoryQuery) ([]retrieval.Candidate, error) {
	if !c.ValidID(q.UserID) || !c.ValidID(q.TriggerID) || q.Limit < 1 || q.Limit > 200 || q.Dimension < 0 || q.Dimension > 8192 || len(q.LexicalTerm) > 80 || len(q.EntityRole) > 100 || len(q.EntityLabel) > 240 {
		return nil, errors.New("invalid owner-scoped memory query")
	}
	var thread string
	if q.ThreadOnly {
		if q.FocusThread == nil || !c.ValidID(*q.FocusThread) {
			return nil, errors.New("thread lookup needs owned focus")
		}
		thread = q.FocusThread.String()
	}
	var since any
	if q.Since != nil {
		since = q.Since.UTC()
	}
	rows, err := r.db.QueryContext(ctx, memorySQL, q.UserID, q.TriggerID, q.Limit, thread, q.EntityRole, q.EntityLabel, since, q.LexicalTerm, q.Model)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]retrieval.Candidate, 0, q.Limit)
	for rows.Next() {
		var item retrieval.Candidate
		var id, owner, threadJSON, entityJSON string
		var content sql.NullString
		var valid sql.NullTime
		var cluster, model sql.NullString
		var ver, dim sql.NullInt64
		var raw []byte
		if err = rows.Scan(&id, &owner, &item.Kind, &content, &item.Epistemic, &item.Status, &item.Sensitivity, &item.Importance,
			&item.CreatedAt, &valid, &threadJSON, &entityJSON, &cluster, &item.EvidenceN, &ver, &model, &dim, &raw); err != nil {
			return nil, err
		}
		item.ID, err = uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		item.UserID, err = uuid.Parse(owner)
		if err != nil {
			return nil, err
		}
		if item.UserID != q.UserID {
			return nil, errors.New("memory read owner mismatch")
		}
		if !content.Valid || strings.TrimSpace(content.String) == "" {
			continue
		}
		item.Source = retrieval.SourceMemory
		item.Text = content.String
		if valid.Valid {
			v := valid.Time.UTC()
			item.ValidFrom = &v
		}
		if cluster.Valid {
			v, e := uuid.Parse(cluster.String)
			if e != nil {
				return nil, e
			}
			item.ClusterID = &v
		}
		if err = json.Unmarshal([]byte(threadJSON), &item.ThreadIDs); err != nil {
			return nil, fmt.Errorf("invalid memory thread references: %w", err)
		}
		if err = json.Unmarshal([]byte(entityJSON), &item.Entities); err != nil {
			return nil, fmt.Errorf("invalid memory entities: %w", err)
		}
		// An absent, corrupt, wrong-model or wrong-dimension embedding is a
		// missing semantic feature, not permission to compare wrong vectors.
		if ver.Valid && model.Valid && dim.Valid && q.Model != "" && q.Dimension > 0 {
			item.Vector, _ = DecodeEmbedding(int(ver.Int64), model.String, q.Model, int(dim.Int64), q.Dimension, raw)
		}
		result = append(result, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

const messagesSQL = `SELECT TOP (@p5) CONVERT(varchar(36),m.id),CONVERT(varchar(36),v.user_id),m.content,m.created_at
FROM dbo.messages m
JOIN dbo.conversations v ON v.id=m.conversation_id AND v.user_id=@p1 AND v.deleted_at IS NULL
JOIN dbo.messages trigger_msg ON trigger_msg.id=@p2 AND trigger_msg.role=N'user'
JOIN dbo.conversations trigger_conv ON trigger_conv.id=trigger_msg.conversation_id AND trigger_conv.user_id=@p1 AND trigger_conv.deleted_at IS NULL
WHERE m.role=N'user' AND m.deleted_at IS NULL AND m.id<>trigger_msg.id
  AND m.created_at>=@p3 AND m.created_at<@p4
  AND ((m.conversation_id=trigger_msg.conversation_id AND m.message_order<trigger_msg.message_order)
    OR (m.conversation_id<>trigger_msg.conversation_id AND m.created_at<trigger_msg.created_at))
ORDER BY m.created_at DESC,m.id DESC`

func (r *RetrievalRepository) Messages(ctx context.Context, q retrieval.MessageQuery) ([]retrieval.Candidate, error) {
	if !c.ValidID(q.UserID) || !c.ValidID(q.TriggerID) || !q.From.Before(q.To) || q.Limit < 1 || q.Limit > 100 {
		return nil, errors.New("invalid owner-scoped temporal query")
	}
	rows, err := r.db.QueryContext(ctx, messagesSQL, q.UserID, q.TriggerID, q.From.UTC(), q.To.UTC(), q.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]retrieval.Candidate, 0, q.Limit)
	for rows.Next() {
		var id, owner, text string
		var item retrieval.Candidate
		if err = rows.Scan(&id, &owner, &text, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.ID, err = uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		item.UserID, err = uuid.Parse(owner)
		if err != nil {
			return nil, err
		}
		if item.UserID != q.UserID {
			return nil, errors.New("message read owner mismatch")
		}
		item.Source = retrieval.SourceMessage
		item.Kind = c.MemEpisode
		item.Text = text
		item.Epistemic = c.EpReported
		item.Status = c.MemActive
		item.Sensitivity = c.SensNormal // actual AI-use/sensitivity gate is required in the Service
		item.Importance = 1
		item.EvidenceN = 1
		out = append(out, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

const recentBeforeSQL = `SELECT TOP (@p3) m.content
FROM dbo.messages m
JOIN dbo.messages trigger_msg ON trigger_msg.id=@p2 AND trigger_msg.role=N'user'
JOIN dbo.conversations v ON v.id=trigger_msg.conversation_id AND v.user_id=@p1 AND v.deleted_at IS NULL
WHERE m.conversation_id=trigger_msg.conversation_id AND m.message_order<trigger_msg.message_order
  AND m.deleted_at IS NULL AND m.role IN(N'user',N'assistant')
ORDER BY m.message_order DESC`

func (r *RetrievalRepository) RecentBefore(ctx context.Context, owner, trigger c.ID, limit int) ([]string, error) {
	if !c.ValidID(owner) || !c.ValidID(trigger) || limit < 0 || limit > 40 {
		return nil, errors.New("invalid recent window")
	}
	if limit == 0 {
		return []string{}, nil
	}
	rows, err := r.db.QueryContext(ctx, recentBeforeSQL, owner, trigger, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, limit)
	for rows.Next() {
		var text string
		if err = rows.Scan(&text); err != nil {
			return nil, err
		}
		out = append(out, text)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
