// Package mssql implements the WP-09 retrieval read ports against SQL Server 2022
// (Contract Pack §7.3). It is the only place retrieval SQL lives.
//
// Owner isolation is enforced three times so a single mistake cannot leak data:
//  1. every query filters on the server-resolved @user parameter;
//  2. the adapter drops any row whose user_id differs from the request;
//  3. the application service re-checks the row owner.
//
// Soft delete: no query ever issues DELETE. Suppressed memory items and messages
// are excluded by predicate and can be read back by maintenance tooling.
package mssql

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Repository is the SQL Server retrieval read model.
type Repository struct{ db *sql.DB }

// New builds the repository. db must be the shared application pool.
func New(db *sql.DB) *Repository {
	if db == nil {
		panic("mssql.New: nil database")
	}
	return &Repository{db: db}
}

var _ retrieval.Store = (*Repository)(nil)

const memoryColumns = `CONVERT(varchar(36),m.id),CONVERT(varchar(36),m.user_id),m.kind,ISNULL(m.content,N''),ISNULL(m.lang,N''),
  m.epistemic,m.status,m.sensitivity,m.importance,
  (SELECT COUNT(*) FROM dbo.ai_memory_evidence e WHERE e.item_id=m.id),
  m.thread_ids_json,m.entities_json,m.tags_json,
  CASE WHEN m.cluster_id IS NULL THEN NULL ELSE CONVERT(varchar(36),m.cluster_id) END,
  m.valid_from,m.valid_to,m.created_at,m.updated_at,m.deleted_at`

func scanMemory(rows *sql.Rows) (contract.MemoryRecord, error) {
	var rec contract.MemoryRecord
	var id, userID, kind, content, lang, epistemic, status, sensitivity, threadsJSON, entitiesJSON, tagsJSON string
	var importance float32
	var evidenceN int
	var cluster sql.NullString
	var validFrom, validTo, deletedAt sql.NullTime
	var createdAt, updatedAt time.Time
	if err := rows.Scan(&id, &userID, &kind, &content, &lang, &epistemic, &status, &sensitivity, &importance,
		&evidenceN, &threadsJSON, &entitiesJSON, &tagsJSON, &cluster, &validFrom, &validTo, &createdAt, &updatedAt, &deletedAt); err != nil {
		return rec, err
	}
	rec, err := buildMemoryRecord(id, userID, kind, content, lang, epistemic, status, sensitivity, importance,
		evidenceN, threadsJSON, entitiesJSON, tagsJSON, cluster, validFrom, validTo, createdAt, updatedAt, deletedAt)
	return rec, err
}

func buildMemoryRecord(id, userID, kind, content, lang, epistemic, status, sensitivity string,
	importance float32, evidenceN int, threadsJSON, entitiesJSON, tagsJSON string,
	cluster sql.NullString, validFrom, validTo sql.NullTime, createdAt, updatedAt time.Time, deletedAt sql.NullTime) (contract.MemoryRecord, error) {
	rec := contract.MemoryRecord{
		Kind: contract.MemoryKind(kind), Content: content, Lang: lang,
		Epistemic: contract.Epistemic(epistemic), Status: contract.MemoryStatus(status),
		Sensitivity: contract.Sensitivity(sensitivity), Importance: importance, EvidenceN: evidenceN,
		CreatedAt: createdAt.UTC(), UpdatedAt: updatedAt.UTC(),
	}
	var err error
	if rec.ID, err = parseUUID(id); err != nil {
		return rec, fmt.Errorf("memory id: %w", err)
	}
	if rec.UserID, err = parseUUID(userID); err != nil {
		return rec, fmt.Errorf("memory user: %w", err)
	}
	if cluster.Valid {
		cid, err := parseUUID(cluster.String)
		if err != nil {
			return rec, fmt.Errorf("memory cluster: %w", err)
		}
		rec.ClusterID = &cid
	}
	if rec.ThreadIDs, err = parseUUIDArray(threadsJSON); err != nil {
		return rec, fmt.Errorf("memory threads: %w", err)
	}
	if rec.Entities, err = parseStringArray(entitiesJSON); err != nil {
		return rec, fmt.Errorf("memory entities: %w", err)
	}
	if rec.Tags, err = parseStringArray(tagsJSON); err != nil {
		return rec, fmt.Errorf("memory tags: %w", err)
	}
	if validFrom.Valid {
		t := validFrom.Time.UTC()
		rec.ValidFrom = &t
	}
	if validTo.Valid {
		t := validTo.Time.UTC()
		rec.ValidTo = &t
	}
	if deletedAt.Valid {
		t := deletedAt.Time.UTC()
		rec.DeletedAt = &t
	}
	return rec, nil
}

// RecentMemory returns the bounded owner-scoped candidate pool. Only live rows
// are returned; soft-deleted ones stay in the table but never enter retrieval.
func (r *Repository) RecentMemory(ctx context.Context, userID contract.ID, limit int) ([]contract.MemoryRecord, error) {
	if !contract.ValidID(userID) {
		return nil, errors.New("mssql: user id required")
	}
	limit = clampLimit(limit)
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p2) `+memoryColumns+`
FROM dbo.ai_memory_items m
WHERE m.user_id=CONVERT(uniqueidentifier,@p1) AND m.deleted_at IS NULL
ORDER BY m.updated_at DESC, m.id`, userID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return drainMemory(rows, userID)
}

// Preferences is the always-on core set (§7.6): preferences are small, stable and
// cheap to include, so they are retrieved even when semantic similarity is low.
func (r *Repository) Preferences(ctx context.Context, userID contract.ID, limit int) ([]contract.MemoryRecord, error) {
	if !contract.ValidID(userID) {
		return nil, errors.New("mssql: user id required")
	}
	limit = clampLimit(limit)
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p2) `+memoryColumns+`
FROM dbo.ai_memory_items m
WHERE m.user_id=CONVERT(uniqueidentifier,@p1) AND m.deleted_at IS NULL
  AND m.kind=N'preference' AND m.status=N'active'
ORDER BY m.importance DESC, m.updated_at DESC, m.id`, userID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return drainMemory(rows, userID)
}

// MemoryByThread returns items linked to a thread via OPENJSON(thread_ids_json).
func (r *Repository) MemoryByThread(ctx context.Context, userID, threadID contract.ID, limit int) ([]contract.MemoryRecord, error) {
	if !contract.ValidID(userID) || !contract.ValidID(threadID) {
		return nil, errors.New("mssql: user and thread required")
	}
	limit = clampLimit(limit)
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p3) `+memoryColumns+`
FROM dbo.ai_memory_items m
CROSS APPLY OPENJSON(m.thread_ids_json) AS t
WHERE m.user_id=CONVERT(uniqueidentifier,@p1) AND m.deleted_at IS NULL
  AND TRY_CONVERT(uniqueidentifier, JSON_VALUE(t.value,'$'))=CONVERT(uniqueidentifier,@p2)
ORDER BY m.updated_at DESC, m.id`, userID.String(), threadID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return drainMemory(rows, userID)
}

// MemoryByEntity resolves a relational mention against entities_json. The label
// comparison is a case-insensitive containment check; alias mapping is an
// explicit open decision (§7.3) and is not implemented.
func (r *Repository) MemoryByEntity(ctx context.Context, userID contract.ID, q retrieval.EntityFilter, limit int) ([]contract.MemoryRecord, error) {
	if !contract.ValidID(userID) || (strings.TrimSpace(q.Role) == "" && strings.TrimSpace(q.Label) == "") {
		return []contract.MemoryRecord{}, nil
	}
	limit = clampLimit(limit)
	role := strings.ToLower(strings.TrimSpace(q.Role))
	label := strings.ToLower(strings.TrimSpace(q.Label))
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p4) `+memoryColumns+`
FROM dbo.ai_memory_items m
CROSS APPLY OPENJSON(m.entities_json) AS e
WHERE m.user_id=CONVERT(uniqueidentifier,@p1) AND m.deleted_at IS NULL
  AND (
      (@p2<>N'' AND LOWER(JSON_VALUE(e.value,'$.role'))=@p2)
   OR (@p3<>N'' AND LOWER(JSON_VALUE(e.value,'$.label'))=@p3)
  )
ORDER BY m.updated_at DESC, m.id`, userID.String(), role, label, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return drainMemory(rows, userID)
}

// MemoryByIDs is the direct item read for reference resolution and second-hop
// `item` requests. It always re-checks ownership.
func (r *Repository) MemoryByIDs(ctx context.Context, userID contract.ID, ids []contract.ID) ([]contract.MemoryRecord, error) {
	if !contract.ValidID(userID) {
		return nil, errors.New("mssql: user id required")
	}
	if len(ids) == 0 {
		return []contract.MemoryRecord{}, nil
	}
	if len(ids) > 20 {
		ids = ids[:20]
	}
	args := make([]any, 0, len(ids)+1)
	args = append(args, userID.String())
	placeholders := make([]string, 0, len(ids))
	for i, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("@p%d", i+2))
		args = append(args, id.String())
	}
	query := `SELECT ` + memoryColumns + `
FROM dbo.ai_memory_items m
WHERE m.user_id=CONVERT(uniqueidentifier,@p1) AND m.deleted_at IS NULL
  AND m.id IN (` + strings.Join(placeholders, ",") + `)
ORDER BY m.updated_at DESC, m.id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return drainMemory(rows, userID)
}

// MessageWindow is the exact/temporal read. It joins through dbo.conversations to
// filter the owner (dbo.messages has no user_id column in 0001) and excludes
// suppressed messages without deleting them. `terms` is an optional,
// deliberately forgiving LIKE filter; an exact window hit with no term overlap is
// still returned so recall answers cannot silently narrow (§4.4 Scenario 2).
func (r *Repository) MessageWindow(ctx context.Context, userID contract.ID, from, to time.Time, roles []contract.Role, terms []string, limit int) ([]contract.MessageRecord, error) {
	if !contract.ValidID(userID) {
		return nil, errors.New("mssql: user id required")
	}
	if !to.After(from) {
		return []contract.MessageRecord{}, nil
	}
	limit = clampLimit(limit)
	if len(roles) == 0 {
		roles = []contract.Role{contract.RoleUser}
	}
	roleArgs := make([]any, 0, len(roles))
	rolePlaceholders := make([]string, 0, len(roles))
	for i, role := range roles {
		switch role {
		case contract.RoleUser, contract.RoleAssistant:
		default:
			return nil, fmt.Errorf("mssql: unsupported role %q", role)
		}
		rolePlaceholders = append(rolePlaceholders, fmt.Sprintf("@r%d", i))
		roleArgs = append(roleArgs, string(role))
	}
	args := []any{userID.String(), from.UTC(), to.UTC()}
	args = append(args, roleArgs...)
	limitParam := fmt.Sprintf("@l%d", len(args))
	args = append(args, limit)

	termFilter := ""
	if len(terms) > 0 {
		parts := make([]string, 0, len(terms))
		for _, term := range terms {
			if strings.TrimSpace(term) == "" {
				continue
			}
			idx := len(args) + 1
			parts = append(parts, fmt.Sprintf("@q%d", idx))
			args = append(args, "%"+term+"%")
		}
		if len(parts) > 0 {
			termFilter = " AND (" + joinLike(parts) + ")"
		}
	}
	query := `SELECT TOP (` + limitParam + `)
    CONVERT(varchar(36),m.id),CONVERT(varchar(36),c.user_id),CONVERT(varchar(36),m.conversation_id),
    CONVERT(varchar(36),m.turn_id),m.role,m.content,m.created_at,s.message_id
FROM dbo.messages m
JOIN dbo.conversations c ON c.id=m.conversation_id
LEFT JOIN dbo.ai_message_suppressions s ON s.message_id=m.id AND s.user_id=c.user_id
WHERE c.user_id=CONVERT(uniqueidentifier,@p1)
  AND m.created_at>=@p2 AND m.created_at<@p3
  AND m.role IN (` + strings.Join(rolePlaceholders, ",") + `)
  AND s.message_id IS NULL` + termFilter + `
ORDER BY m.created_at DESC, m.id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]contract.MessageRecord, 0, limit)
	for rows.Next() {
		var id, uid, conv, turn, role, content string
		var createdAt time.Time
		var suppressed sql.NullString
		if err := rows.Scan(&id, &uid, &conv, &turn, &role, &content, &createdAt, &suppressed); err != nil {
			return nil, err
		}
		rec, err := buildMessageRecord(id, uid, conv, turn, role, content, createdAt, suppressed)
		if err != nil {
			return nil, err
		}
		if rec.UserID != userID {
			continue // defensive: never surface a foreign row
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Oldest last -> caller expects ascending order for readability in context.
	reverseMessages(out)
	return out, nil
}

func buildMessageRecord(id, uid, conv, turn, role, content string, createdAt time.Time, suppressed sql.NullString) (contract.MessageRecord, error) {
	rec := contract.MessageRecord{Role: contract.Role(role), Content: content, CreatedAt: createdAt.UTC()}
	var err error
	if rec.ID, err = parseUUID(id); err != nil {
		return rec, fmt.Errorf("message id: %w", err)
	}
	if rec.UserID, err = parseUUID(uid); err != nil {
		return rec, fmt.Errorf("message user: %w", err)
	}
	if rec.ConversationID, err = parseUUID(conv); err != nil {
		return rec, fmt.Errorf("message conversation: %w", err)
	}
	if rec.TurnID, err = parseUUID(turn); err != nil {
		return rec, fmt.Errorf("message turn: %w", err)
	}
	if suppressed.Valid {
		t := createdAt.UTC()
		rec.DeletedAt = &t
	}
	return rec, nil
}

// Embeddings loads stored float32 vectors for the requested refs.
//
// Vectors are stored as little-endian float32 blobs (§7.2 CK_ai_embedding_bytes).
// A row whose DATALENGTH disagrees with dim*4 is skipped rather than decoded
// incorrectly; the caller then falls back to lexical similarity.
func (r *Repository) Embeddings(ctx context.Context, userID contract.ID, refs []contract.EmbeddingKey) (map[contract.EmbeddingKey]contract.Vector, error) {
	out := map[contract.EmbeddingKey]contract.Vector{}
	if !contract.ValidID(userID) || len(refs) == 0 {
		return out, nil
	}
	if len(refs) > 500 {
		refs = refs[:500]
	}
	memoryIDs := make([]contract.ID, 0, len(refs))
	messageIDs := make([]contract.ID, 0, len(refs))
	seen := map[contract.EmbeddingKey]bool{}
	for _, ref := range refs {
		if !contract.ValidID(ref.ID) || seen[ref] {
			continue
		}
		seen[ref] = true
		switch ref.Kind {
		case contract.RefKindMemory:
			memoryIDs = append(memoryIDs, ref.ID)
		case contract.RefKindMessage:
			messageIDs = append(messageIDs, ref.ID)
		}
	}
	if len(memoryIDs) > 0 {
		if err := r.loadEmbeddings(ctx, userID, contract.RefKindMemory, memoryIDs, out); err != nil {
			return nil, err
		}
	}
	if len(messageIDs) > 0 {
		if err := r.loadEmbeddings(ctx, userID, contract.RefKindMessage, messageIDs, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *Repository) loadEmbeddings(ctx context.Context, userID contract.ID, kind string, ids []contract.ID, out map[contract.EmbeddingKey]contract.Vector) error {
	args := []any{userID.String(), kind}
	placeholders := make([]string, 0, len(ids))
	for i, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("@p%d", i+3))
		args = append(args, id.String())
	}
	rows, err := r.db.QueryContext(ctx, `SELECT CONVERT(varchar(36),e.ref_id),e.model_id,e.dim,e.vector_bytes
FROM dbo.ai_embeddings e
WHERE e.user_id=CONVERT(uniqueidentifier,@p1) AND e.ref_kind=@p2
  AND e.ref_id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var refID, modelID string
		var dim int
		var blob []byte
		if err := rows.Scan(&refID, &modelID, &dim, &blob); err != nil {
			return err
		}
		id, err := parseUUID(refID)
		if err != nil {
			return fmt.Errorf("embedding ref: %w", err)
		}
		values, err := decodeVector(blob, dim)
		if err != nil {
			continue // corrupt/legacy row: treat as missing embedding
		}
		out[contract.EmbeddingKey{Kind: kind, ID: id}] = contract.Vector{ModelID: modelID, Dim: dim, Values: values}
	}
	return rows.Err()
}

// decodeVector validates DATALENGTH == dim*4 and decodes little-endian float32.
func decodeVector(blob []byte, dim int) ([]float32, error) {
	if dim <= 0 || dim > 8192 {
		return nil, fmt.Errorf("mssql: invalid embedding dimension %d", dim)
	}
	if len(blob) != dim*4 {
		return nil, fmt.Errorf("mssql: vector byte length %d does not match dim %d", len(blob), dim)
	}
	values := make([]float32, dim)
	for i := 0; i < dim; i++ {
		bits := binary.LittleEndian.Uint32(blob[i*4 : i*4+4])
		v := math.Float32frombits(bits)
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return nil, errors.New("mssql: non-finite embedding value")
		}
		values[i] = v
	}
	return values, nil
}

// EncodeVector is the inverse of decodeVector, used by embedding writers (WP-04
// hooks) and by integration tests. It rejects non-finite values for the same
// reason the decoder does.
func EncodeVector(values []float32) ([]byte, error) {
	blob := make([]byte, len(values)*4)
	for i, v := range values {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return nil, errors.New("mssql: non-finite embedding value")
		}
		binary.LittleEndian.PutUint32(blob[i*4:i*4+4], math.Float32bits(v))
	}
	return blob, nil
}

func drainMemory(rows *sql.Rows, userID contract.ID) ([]contract.MemoryRecord, error) {
	out := []contract.MemoryRecord{}
	for rows.Next() {
		rec, err := scanMemory(rows)
		if err != nil {
			return nil, err
		}
		if rec.UserID != userID {
			continue
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func joinLike(parts []string) string {
	wrapped := make([]string, 0, len(parts))
	for _, p := range parts {
		wrapped = append(wrapped, "m.content LIKE "+p)
	}
	return strings.Join(wrapped, " OR ")
}

func reverseMessages(msgs []contract.MessageRecord) {
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return 25
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func parseUUID(raw string) (contract.ID, error) {
	return parseUUIDStrict(raw)
}

func parseStringArray(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseUUIDArray(raw string) ([]contract.ID, error) {
	values, err := parseStringArray(raw)
	if err != nil {
		return nil, err
	}
	out := make([]contract.ID, 0, len(values))
	for _, v := range values {
		id, err := parseUUIDStrict(v)
		if err != nil {
			continue // ignore malformed entries instead of failing retrieval
		}
		out = append(out, id)
	}
	return out, nil
}

// RecentMessages returns the most recent message texts of one conversation for
// the novelty filter. Owner scoping goes through dbo.conversations because
// dbo.messages has no user_id column (0001); suppressed messages are skipped.
func (r *Repository) RecentMessages(ctx context.Context, userID, conversationID contract.ID, limit int) ([]string, error) {
	if !contract.ValidID(userID) || !contract.ValidID(conversationID) {
		return nil, nil
	}
	limit = clampLimit(limit)
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p3) m.content
FROM dbo.messages m
JOIN dbo.conversations c ON c.id=m.conversation_id
LEFT JOIN dbo.ai_message_suppressions s ON s.message_id=m.id AND s.user_id=c.user_id
WHERE c.user_id=CONVERT(uniqueidentifier,@p1) AND m.conversation_id=CONVERT(uniqueidentifier,@p2)
  AND s.message_id IS NULL
ORDER BY m.message_order DESC`, userID.String(), conversationID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, limit)
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		out = append(out, content)
	}
	return out, rows.Err()
}
