package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Lookup inherits both current visibility and the admitted turn's revision fence.
// It cannot expand into a subsequently admitted message or a later source revision.
func (r *RetrievalRepository) Lookup(ctx context.Context, q retrieval.Query, kind retrieval.SourceKind, id c.ID) (retrieval.Candidate, bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_search_sources s
 JOIN dbo.messages m ON m.id=@p2 AND m.role=N'user' AND m.deleted_at IS NULL
 JOIN dbo.conversations cv ON cv.id=m.conversation_id AND cv.user_id=@p1 AND cv.deleted_at IS NULL
 WHERE s.user_id=@p1 AND s.source_kind=@p3 AND s.id=@p4
 AND s.source_version<CONVERT(binary(8),m.search_version) AND s.created_at<=m.created_at
 AND (s.source_kind<>N'message' OR s.conversation_id<>m.conversation_id OR s.message_order<m.message_order)`, q.UserID, q.TriggerID, kind, id).Scan(&n)
	if err != nil || n != 1 {
		return retrieval.Candidate{}, false, err
	}
	return r.source(ctx, q.UserID, kind, id)
}
func (r *RetrievalRepository) Evidence(ctx context.Context, q retrieval.Query, item retrieval.Candidate) ([]c.EvidenceRef, *c.ID, error) {
	if item.UserID != q.UserID || item.Source != retrieval.SourceMemory {
		return nil, nil, errors.New("invalid evidence request")
	}
	allowed, err := r.use.CanUse(ctx, q.UserID, retrieval.SourceMemory, item.ID)
	if err != nil || !allowed {
		return nil, nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT TOP(4) e.ref_kind,CONVERT(varchar(36),e.ref_id),e.role,e.evidence_at FROM dbo.ai_memory_evidence e JOIN dbo.ai_memory_items m ON m.id=e.item_id AND m.user_id=e.user_id WHERE e.user_id=@p1 AND e.item_id=@p2 AND m.deleted_at IS NULL AND m.status IN(N'active',N'proposed',N'supported') ORDER BY CASE WHEN e.role=N'contradicts' THEN 0 ELSE 1 END,e.evidence_at DESC,e.ref_id`, q.UserID, item.ID)
	if err != nil {
		return nil, nil, err
	}
	refs := []c.EvidenceRef{}
	for rows.Next() {
		var e c.EvidenceRef
		if err = rows.Scan(&e.Kind, &e.Ref, &e.Role, &e.At); err != nil {
			rows.Close()
			return nil, nil, err
		}
		refs = append(refs, e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	var raw string
	if err = r.db.QueryRowContext(ctx, `SELECT thread_ids_json FROM dbo.ai_memory_items WHERE user_id=@p1 AND id=@p2`, q.UserID, item.ID).Scan(&raw); err != nil {
		return nil, nil, err
	}
	var ids []c.ID
	if json.Unmarshal([]byte(raw), &ids) != nil {
		return nil, nil, errors.New("invalid thread references")
	}
	var thread *c.ID
	if len(ids) > 0 {
		thread = &ids[0]
	}
	return refs, thread, nil
}
func (r *RetrievalRepository) Baseline(ctx context.Context, user, trigger c.ID) ([]retrieval.Candidate, error) {
	if !c.ValidID(user) || !c.ValidID(trigger) {
		return nil, errors.New("invalid baseline identity")
	}
	var turn c.ID
	if err := r.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),t.id) FROM dbo.turns t JOIN dbo.conversations v ON v.id=t.conversation_id AND v.user_id=t.user_id AND v.deleted_at IS NULL WHERE t.user_id=@p1 AND t.user_message_id=@p2 AND t.status=N'running'`, user, trigger).Scan(&turn); err != nil {
		return nil, err
	}
	allowed, err := r.use.CanSearch(ctx, user, turn)
	if err != nil || !allowed {
		return nil, err
	}
	// Stable reported facts/preferences/goals require a pin or high importance.
	// Hypotheses have a separate active-thread lane; their epistemic label survives.
	rows, err := r.db.QueryContext(ctx, `WITH eligible AS (SELECT m.id,m.kind,m.pinned,m.importance,m.updated_at,
 ROW_NUMBER() OVER(PARTITION BY CASE WHEN m.kind=N'hypothesis' THEN 1 ELSE 0 END ORDER BY m.pinned DESC,m.importance DESC,m.updated_at DESC,m.id) lane_rank
 FROM dbo.ai_memory_items m JOIN dbo.users u ON u.id=m.user_id AND u.deleted_at IS NULL
 JOIN dbo.messages trigger_msg ON trigger_msg.id=@p2
 WHERE m.user_id=@p1 AND m.deleted_at IS NULL AND m.sensitivity<>N'highly_sensitive'
 AND m.superseded_by_id IS NULL AND m.contradict_n=0 AND (m.valid_to IS NULL OR m.valid_to>trigger_msg.created_at)
 AND (m.valid_from IS NULL OR m.valid_from<=trigger_msg.created_at)
 AND m.search_version<trigger_msg.search_version
 AND EXISTS(SELECT 1 FROM dbo.ai_memory_evidence e WHERE e.user_id=m.user_id AND e.item_id=m.id)
 AND ((m.epistemic=N'reported' AND m.status=N'active' AND m.kind IN(N'fact',N'preference',N'goal') AND (m.pinned=1 OR m.importance>=0.7))
 OR (m.kind=N'hypothesis' AND m.status IN(N'proposed',N'supported') AND EXISTS(SELECT 1 FROM dbo.ai_threads t CROSS APPLY OPENJSON(t.hypothesis_ids_json) h WHERE t.user_id=m.user_id AND t.phase<>N'resolved' AND TRY_CONVERT(uniqueidentifier,h.value)=m.id)))
 ) SELECT CONVERT(varchar(36),id) FROM eligible WHERE lane_rank<=8 ORDER BY pinned DESC,CASE WHEN kind=N'hypothesis' THEN 1 ELSE 0 END,importance DESC,updated_at DESC,id`, user, trigger)
	if err != nil {
		return nil, err
	}
	ids := []c.ID{}
	for rows.Next() {
		var id c.ID
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	q := retrieval.Query{UserID: user, TriggerID: trigger}
	out := []retrieval.Candidate{}
	for _, id := range ids {
		item, found, e := r.Lookup(ctx, q, retrieval.SourceMemory, id)
		if e != nil {
			return nil, e
		}
		if !found {
			continue
		}
		item.Evidence, item.ThreadID, e = r.Evidence(ctx, q, item)
		if e != nil {
			return nil, e
		}
		out = append(out, item)
	}
	return out, nil
}

// Related follows the selected memory's thread and its latest formulation only.
// Lookup applies the same owner, visibility and admitted-source revision fence.
func (r *RetrievalRepository) Related(ctx context.Context, q retrieval.Query, parent retrieval.Candidate) ([]retrieval.Candidate, error) {
	if parent.UserID != q.UserID || parent.ThreadID == nil {
		return nil, nil
	}
	out := []retrieval.Candidate{}
	thread, found, e := r.Lookup(ctx, q, retrieval.SourceThread, *parent.ThreadID)
	if e != nil {
		return nil, e
	}
	if found {
		thread.ThreadID = parent.ThreadID
		out = append(out, thread)
	}
	var id c.ID
	e = r.db.QueryRowContext(ctx, `SELECT TOP(1) CONVERT(varchar(36),id) FROM dbo.ai_formulations WHERE user_id=@p1 AND thread_id=@p2 AND deleted_at IS NULL ORDER BY version DESC`, q.UserID, *parent.ThreadID).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		return out, nil
	}
	if e != nil {
		return nil, e
	}
	form, found, e := r.Lookup(ctx, q, retrieval.SourceFormulation, id)
	if e != nil {
		return nil, e
	}
	if found {
		form.ThreadID = parent.ThreadID
		out = append(out, form)
	}
	return out, nil
}
