package mssql

// This WP-09 adapter is read-only. It reads the WP-02 Conversation/AI tables in
// short transactions and never writes, deletes or changes consent. Every query
// filters the server-resolved user_id first; soft-deleted rows are hidden by
// visibility predicates, never purged.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// RetrievalStore implements retrieval.Store for SQL Server 2022. It performs no
// vector index search: stored vectors are loaded for a bounded candidate set and
// cosine-ranked in Go, exactly as §7.3 requires.
type RetrievalStore struct{ db *sql.DB }

var _ retrieval.Store = (*RetrievalStore)(nil)

func NewRetrievalStore(db *sql.DB) (*RetrievalStore, error) {
	if db == nil {
		return nil, errors.New("WP-09 retrieval database is nil")
	}
	return &RetrievalStore{db: db}, nil
}

// ownerRequired rejects a missing/zero owner before any SQL is executed.
func ownerRequired(user c.ID) error {
	if !c.ValidID(user) {
		return errors.New("retrieval owner identity required")
	}
	return nil
}

// MessagesInWindow implements §7.3 `Messages.Window`: owner join on
// conversations, UTC datetime2(3) half-open bounds, TOP (@limit), visibility
// predicates on both message and parent conversation.
func (s *RetrievalStore) MessagesInWindow(ctx context.Context, q retrieval.MessageWindowQuery) ([]retrieval.MessageRow, error) {
	if err := ownerRequired(q.UserID); err != nil {
		return nil, err
	}
	if q.From.IsZero() || !q.To.After(q.From) || q.Limit < 1 || q.Limit > 100 {
		return nil, errors.New("invalid message window bounds")
	}
	roles := make([]string, 0, len(q.Roles))
	for _, r := range q.Roles {
		if r != c.RoleUser && r != c.RoleAssistant {
			return nil, errors.New("invalid message role filter")
		}
		roles = append(roles, string(r))
	}
	if len(roles) == 0 {
		roles = []string{string(c.RoleUser)}
	}
	// Roles are closed enum values, so the IN list is built from literals, not
	// from caller text. Bound values remain parameterized below.
	roleFilter := "N'" + roles[0] + "'"
	for _, r := range roles[1:] {
		roleFilter += ",N'" + r + "'"
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT CONVERT(varchar(36),m.id), CONVERT(varchar(36),m.conversation_id), CONVERT(varchar(36),m.turn_id),
       m.role, m.content, m.created_at, COALESCE(m.lang,N'')
FROM dbo.messages m
JOIN dbo.conversations v ON v.id = m.conversation_id
JOIN dbo.turns t ON t.id = m.turn_id AND t.user_id = v.user_id AND t.lifecycle_version = 1
WHERE v.user_id = @p1
  AND v.deleted_at IS NULL
  AND m.deleted_at IS NULL
  AND m.created_at >= @p2 AND m.created_at < @p3
  AND m.role IN (`+roleFilter+`)
  AND t.status = N'completed'
ORDER BY m.created_at ASC, m.message_order ASC
OFFSET 0 ROWS FETCH NEXT @p4 ROWS ONLY`, q.UserID, q.From.UTC(), q.To.UTC(), q.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []retrieval.MessageRow
	for rows.Next() {
		var row retrieval.MessageRow
		var id, conv, turn, role string
		if err = rows.Scan(&id, &conv, &turn, &role, &row.Text, &row.CreatedAt, &row.Lang); err != nil {
			return nil, err
		}
		if row.ID, err = parseID(id); err != nil {
			return nil, err
		}
		if row.ConversationID, err = parseID(conv); err != nil {
			return nil, err
		}
		if row.TurnID, err = parseID(turn); err != nil {
			return nil, err
		}
		row.Role = c.Role(role)
		out = append(out, row)
	}
	return out, rows.Err()
}

// RecentThreadMessages returns the newest visible messages of one owner-scoped
// conversation, oldest first. It is used only for bounded in-thread candidates.
func (s *RetrievalStore) RecentThreadMessages(ctx context.Context, userID, conversationID c.ID, limit int) ([]retrieval.MessageRow, error) {
	if err := ownerRequired(userID); err != nil {
		return nil, err
	}
	if !c.ValidID(conversationID) || limit < 1 || limit > 50 {
		return nil, errors.New("invalid conversation read bounds")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT CONVERT(varchar(36),m.id), CONVERT(varchar(36),m.conversation_id), CONVERT(varchar(36),m.turn_id),
       m.role, m.content, m.created_at, COALESCE(m.lang,N'')
FROM (SELECT TOP (@p3) * FROM dbo.messages WHERE conversation_id = @p2 AND deleted_at IS NULL
      ORDER BY created_at DESC, message_order DESC) m
JOIN dbo.conversations v ON v.id = m.conversation_id
WHERE v.user_id = @p1 AND v.deleted_at IS NULL
ORDER BY m.created_at ASC, m.message_order ASC`, userID, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []retrieval.MessageRow
	for rows.Next() {
		var row retrieval.MessageRow
		var id, conv, turn, role string
		if err = rows.Scan(&id, &conv, &turn, &role, &row.Text, &row.CreatedAt, &row.Lang); err != nil {
			return nil, err
		}
		if row.ID, err = parseID(id); err != nil {
			return nil, err
		}
		if row.ConversationID, err = parseID(conv); err != nil {
			return nil, err
		}
		if row.TurnID, err = parseID(turn); err != nil {
			return nil, err
		}
		row.Role = c.Role(role)
		out = append(out, row)
	}
	return out, rows.Err()
}

// memoryColumns is the projection shared by the memory reads. Encrypted-only
// rows (content_enc) deliberately come back empty: v1.2 requires KMS decryption,
// which is a separate gate, and retrieval must not invent plaintext.
const memoryColumns = `
  CONVERT(varchar(36),m.id), CONVERT(varchar(36),m.user_id), m.kind,
  COALESCE(m.content,N''), COALESCE(m.lang,N''), m.epistemic, m.status, m.confidence,
  m.support_n, m.contradict_n, m.sessions_n,
  m.thread_ids_json, m.entities_json, m.tags_json, m.sensitivity, m.importance,
  m.valid_from, m.valid_to, m.created_at, m.updated_at, m.last_confirmed_at, m.last_used_at,
  CONVERT(varchar(36),m.supersedes_id), CONVERT(varchar(36),m.superseded_by_id),
  CONVERT(varchar(36),m.cluster_id), m.source_json`

// visibilityPredicate is the single visibility/eligibility clause used by every
// memory read: owner-scoped at call sites, soft-deleted rows hidden, and
// terminal/highly-sensitive records excluded from default AI retrieval.
const visibilityPredicate = ` AND m.deleted_at IS NULL
  AND m.status IN (N'active',N'contested',N'proposed',N'supported')
  AND m.sensitivity <> N'highly_sensitive'`

// MemoryByThread implements §7.3 `Memory.ByThread` using OPENJSON on the stored
// thread id array. The owner filter is applied first, then the JSON predicate.
func (s *RetrievalStore) MemoryByThread(ctx context.Context, userID, threadID c.ID, kinds []c.MemoryKind, limit int) ([]c.MemoryItem, error) {
	if err := ownerRequired(userID); err != nil {
		return nil, err
	}
	if !c.ValidID(threadID) || limit < 1 || limit > 100 {
		return nil, errors.New("invalid thread read bounds")
	}
	query := `SELECT ` + memoryColumns + `
FROM dbo.ai_memory_items m
WHERE m.user_id = @p1` + visibilityPredicate + `
  AND EXISTS (SELECT 1 FROM OPENJSON(m.thread_ids_json) j WHERE TRY_CONVERT(uniqueidentifier, j.value) = @p2)`
	args := []any{userID, threadID}
	query, args, err := withKinds(query, args, kinds)
	if err != nil {
		return nil, err
	}
	return s.queryItems(ctx, query+` ORDER BY m.importance DESC, m.updated_at DESC`, append(args, limit)...)
}

// MemoryByEntity implements §7.3 `Memory.ByEntity`: owner first, then OPENJSON
// over entities_json. Person alias matching remains [OPEN] in v1.2, so this
// matches the stored role and optional label literally.
func (s *RetrievalStore) MemoryByEntity(ctx context.Context, q retrieval.EntityQuery) ([]c.MemoryItem, error) {
	if err := ownerRequired(q.UserID); err != nil {
		return nil, err
	}
	if q.Role == "" || len(q.Role) > 40 || len(q.Label) > 80 || q.Limit < 1 || q.Limit > 100 {
		return nil, errors.New("invalid entity query")
	}
	query := `SELECT ` + memoryColumns + `
FROM dbo.ai_memory_items m
WHERE m.user_id = @p1` + visibilityPredicate + `
  AND EXISTS (SELECT 1 FROM OPENJSON(m.entities_json) WITH (role nvarchar(40) '$.role', label nvarchar(80) '$.label') e
              WHERE e.role = @p2 AND (@p3 = N'' OR e.label = @p3))`
	args := []any{q.UserID, q.Role, q.Label}
	query, args, err := withKinds(query, args, []c.MemoryKind{q.Kind})
	if err != nil {
		return nil, err
	}
	return s.queryItems(ctx, query+` ORDER BY m.importance DESC, m.updated_at DESC`, append(args, q.Limit)...)
}

// MemoryCandidates implements the bounded §7.3 `Memory.Candidates` pool. It
// never scans the whole table or promises vector search: the pool is the union
// of (a) recent/important items, (b) thread-linked items and (c) lexical matches
// over content or tags, capped by @limit.
func (s *RetrievalStore) MemoryCandidates(ctx context.Context, q retrieval.CandidateQuery) ([]c.MemoryItem, error) {
	if err := ownerRequired(q.UserID); err != nil {
		return nil, err
	}
	if q.Limit < 1 || q.Limit > 100 {
		return nil, errors.New("invalid candidate limit")
	}
	if len(q.Terms) > 12 {
		q.Terms = q.Terms[:12]
	}
	for _, term := range q.Terms {
		if len(term) < 3 || len(term) > 40 {
			return nil, errors.New("invalid lexical term")
		}
	}
	// LIKE patterns are built with escaped wildcards so a user term cannot turn
	// into an unbounded scan or an injection vector; values stay parameterized.
	patterns := make([]string, 0, len(q.Terms))
	for _, term := range q.Terms {
		patterns = append(patterns, "%"+escapeLike(term)+"%")
	}
	query := `SELECT TOP (@p1) ` + memoryColumns + `
FROM dbo.ai_memory_items m
WHERE m.user_id = @p2` + visibilityPredicate
	args := []any{q.Limit, q.UserID}
	var (
		branches []string
	)
	if q.MinImport > 0 {
		branches = append(branches, `m.importance >= @p`+placeholder(len(args)+1))
		args = append(args, q.MinImport)
	}
	if q.RecentDays > 0 {
		branches = append(branches, `m.updated_at >= @p`+placeholder(len(args)+1))
		args = append(args, time.Now().UTC().AddDate(0, 0, -q.RecentDays))
	}
	if q.Since.IsZero() == false {
		branches = append(branches, `m.updated_at >= @p`+placeholder(len(args)+1))
		args = append(args, q.Since.UTC())
	}
	if len(patterns) > 0 {
		like := make([]string, 0, len(patterns))
		for _, p := range patterns {
			like = append(like, `m.content LIKE @p`+placeholder(len(args)+1)+` ESCAPE '\'`)
			args = append(args, p)
			like = append(like, `m.tags_json LIKE @p`+placeholder(len(args)+1)+` ESCAPE '\'`)
			args = append(args, p)
		}
		branches = append(branches, "("+joinOr(like)+")")
	}
	query, args, err := withKinds(query, args, q.Kinds)
	if err != nil {
		return nil, err
	}
	query, args, err = withStatuses(query, args, q.Statuses)
	if err != nil {
		return nil, err
	}
	if q.ThreadID != nil {
		if !c.ValidID(*q.ThreadID) {
			return nil, errors.New("invalid candidate thread")
		}
		branches = append(branches, `EXISTS (SELECT 1 FROM OPENJSON(m.thread_ids_json) j WHERE TRY_CONVERT(uniqueidentifier, j.value) = @p`+placeholder(len(args)+1)+`)`)
		args = append(args, *q.ThreadID)
	}
	if len(branches) > 0 {
		query += ` AND (` + joinOr(branches) + `)`
	}
	return s.queryItems(ctx, query+` ORDER BY m.importance DESC, m.updated_at DESC`, args...)
}

// MemoryByIDs loads specific owner-visible items (second hop by item id). The
// requested count is bounded and non-owner IDs silently fall out of the result.
func (s *RetrievalStore) MemoryByIDs(ctx context.Context, userID c.ID, ids []c.ID) ([]c.MemoryItem, error) {
	if err := ownerRequired(userID); err != nil {
		return nil, err
	}
	if len(ids) == 0 || len(ids) > 20 {
		return nil, errors.New("invalid item id set")
	}
	args := []any{userID}
	var params []string
	for _, id := range ids {
		if !c.ValidID(id) {
			return nil, errors.New("invalid item identity")
		}
		params = append(params, `@p`+placeholder(len(args)+1))
		args = append(args, id)
	}
	query := `SELECT ` + memoryColumns + `
FROM dbo.ai_memory_items m
WHERE m.user_id = @p1` + visibilityPredicate + ` AND m.id IN (` + joinList(params) + `)`
	return s.queryItems(ctx, query, args...)
}

// Embeddings loads stored vectors for a bounded ref set. Missing vectors are
// absent from the result rather than an error: "no embedding yet" is normal.
func (s *RetrievalStore) Embeddings(ctx context.Context, userID c.ID, refs []retrieval.EmbeddingRef) ([]retrieval.StoredEmbedding, error) {
	if err := ownerRequired(userID); err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, nil
	}
	if len(refs) > 100 {
		return nil, errors.New("too many embedding refs")
	}
	// Two bounded reads (message refs and memory refs) keep the ref_kind
	// predicate index-friendly; a single OR over polymorphic ids does not.
	var memoryIDs, messageIDs []c.ID
	for _, ref := range refs {
		if !c.ValidID(ref.ID) {
			return nil, errors.New("invalid embedding reference")
		}
		if ref.IsMsg {
			messageIDs = append(messageIDs, ref.ID)
		} else {
			memoryIDs = append(memoryIDs, ref.ID)
		}
	}
	var out []retrieval.StoredEmbedding
	for _, set := range []struct {
		kind c.ID
		ids  []c.ID
		msg  bool
	}{
		{ids: memoryIDs, msg: false},
		{ids: messageIDs, msg: true},
	} {
		if len(set.ids) == 0 {
			continue
		}
		args := []any{userID}
		var params []string
		for _, id := range set.ids {
			params = append(params, `@p`+placeholder(len(args)+1))
			args = append(args, id)
		}
		refKind := "N'memory'"
		if set.msg {
			refKind = "N'message'"
		}
		rows, err := s.db.QueryContext(ctx, `
SELECT CONVERT(varchar(36),e.ref_id), e.model_id, e.codec_version, e.dim, e.vector_bytes
FROM dbo.ai_embeddings e
WHERE e.user_id = @p1 AND e.ref_kind = `+refKind+` AND e.ref_id IN (`+joinList(params)+`)`, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id, model string
			var codec, dim int
			var raw []byte
			if err = rows.Scan(&id, &model, &codec, &dim, &raw); err != nil {
				rows.Close()
				return nil, err
			}
			parsed, perr := parseID(id)
			if perr != nil {
				rows.Close()
				return nil, perr
			}
			vector, derr := DecodeEmbedding(codec, model, model, dim, dim, raw)
			if derr != nil {
				// A corrupt or dimension-mismatched vector is skipped, never
				// turned into a fabricated similarity.
				continue
			}
			out = append(out, retrieval.StoredEmbedding{
				RefID: parsed, IsMessage: set.msg, ModelID: model,
				CodecVer: codec, Dimensions: dim, Vector: vector,
			})
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return out, nil
}

// Thread returns one owner-visible thread.
func (s *RetrievalStore) Thread(ctx context.Context, userID, threadID c.ID) (c.ThreadRecord, error) {
	if err := ownerRequired(userID); err != nil {
		return c.ThreadRecord{}, err
	}
	if !c.ValidID(threadID) {
		return c.ThreadRecord{}, errors.New("invalid thread identity")
	}
	var (
		rec        c.ThreadRecord
		id, uid    string
		phase      string
		prevPhase  sql.NullString
		topic      string
		priority   float64
		slots      string
		goal       sql.NullString
		hypotheses string
		related    string
		formula    int
		tags       string
		created    time.Time
		lastActive time.Time
		version    int64
	)
	err := s.db.QueryRowContext(ctx, `
SELECT CONVERT(varchar(36),t.id), CONVERT(varchar(36),t.user_id), t.topic, t.phase, t.prev_phase, t.priority,
       t.slots_json, t.goal, t.hypothesis_ids_json, t.related_thread_ids_json, t.formulation_version,
       t.problem_tags_json, t.created_at, t.last_active_at, t.version
FROM dbo.ai_threads t
WHERE t.id = @p1 AND t.user_id = @p2`, threadID, userID).Scan(&id, &uid, &topic, &phase, &prevPhase, &priority,
		&slots, &goal, &hypotheses, &related, &formula, &tags, &created, &lastActive, &version)
	if errors.Is(err, sql.ErrNoRows) {
		// Never distinguish "absent" from "another owner's": the service turns
		// this sentinel into a single forbidden outcome.
		return c.ThreadRecord{}, retrieval.ErrNotVisible
	}
	if err != nil {
		return c.ThreadRecord{}, err
	}
	if rec.ID, err = parseID(id); err != nil {
		return c.ThreadRecord{}, err
	}
	if rec.UserID, err = parseID(uid); err != nil {
		return c.ThreadRecord{}, err
	}
	rec.Topic, rec.Phase, rec.Priority = topic, c.ThreadPhase(phase), float32(priority)
	if prevPhase.Valid {
		rec.PrevPhase = c.ThreadPhase(prevPhase.String)
	}
	if err = decodeJSON(slots, &rec.Slots); err != nil {
		return c.ThreadRecord{}, err
	}
	if rec.Slots == nil {
		rec.Slots = map[string]c.SlotView{}
	}
	if goal.Valid {
		v := goal.String
		rec.Goal = &v
	}
	if err = decodeIDList(hypotheses, &rec.HypothesisIDs); err != nil {
		return c.ThreadRecord{}, err
	}
	if err = decodeIDList(related, &rec.RelatedThreadIDs); err != nil {
		return c.ThreadRecord{}, err
	}
	if err = decodeStrings(tags, &rec.ProblemTags); err != nil {
		return c.ThreadRecord{}, err
	}
	rec.FormulationVersion, rec.CreatedAt, rec.LastActiveAt, rec.Version = formula, created, lastActive, version
	return rec, nil
}

// ActiveThreads returns non-resolved owner threads ordered by priority and last
// activity (§7.3 `Threads.Active`).
func (s *RetrievalStore) ActiveThreads(ctx context.Context, userID c.ID, limit int) ([]c.ThreadRecord, error) {
	if err := ownerRequired(userID); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 10 {
		return nil, errors.New("invalid active-thread limit")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT CONVERT(varchar(36),t.id)
FROM dbo.ai_threads t
WHERE t.user_id = @p1 AND t.phase NOT IN (N'resolved',N'dormant')
ORDER BY t.priority DESC, t.last_active_at DESC
OFFSET 0 ROWS FETCH NEXT @p2 ROWS ONLY`, userID, limit)
	if err != nil {
		return nil, err
	}
	var ids []c.ID
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, err
		}
		id, perr := parseID(raw)
		if perr != nil {
			rows.Close()
			return nil, perr
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	out := make([]c.ThreadRecord, 0, len(ids))
	for _, id := range ids {
		rec, terr := s.Thread(ctx, userID, id)
		if terr != nil {
			return nil, terr
		}
		out = append(out, rec)
	}
	return out, nil
}

// UserTimeZone implements tz_source=user_state. A missing state row yields UTC
// because the schema default is UTC; it is not treated as an error.
func (s *RetrievalStore) UserTimeZone(ctx context.Context, userID c.ID) (string, error) {
	if err := ownerRequired(userID); err != nil {
		return "", err
	}
	var tz string
	err := s.db.QueryRowContext(ctx, `SELECT tz FROM dbo.ai_user_state WHERE user_id = @p1`, userID).Scan(&tz)
	if errors.Is(err, sql.ErrNoRows) {
		return "UTC", nil
	}
	if err != nil {
		return "", err
	}
	return tz, nil
}

// queryItems scans the shared memory projection.
func (s *RetrievalStore) queryItems(ctx context.Context, query string, args ...any) ([]c.MemoryItem, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []c.MemoryItem
	for rows.Next() {
		item, serr := scanItem(rows)
		if serr != nil {
			return nil, serr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scanItem(rows scanner) (c.MemoryItem, error) {
	var (
		item                              c.MemoryItem
		id, uid, kind, content, lang      string
		epistemic, status, confidence     string
		threads, entities, tags, source   string
		sensitivity                       string
		importance                        float64
		supportN, contradictN, sessionsN  int
		validFrom, validTo                sql.NullTime
		created, updated                  time.Time
		lastConfirmed, lastUsed           sql.NullTime
		supersedes, supersededBy, cluster sql.NullString
	)
	if err := rows.Scan(&id, &uid, &kind, &content, &lang, &epistemic, &status, &confidence,
		&supportN, &contradictN, &sessionsN, &threads, &entities, &tags, &sensitivity, &importance,
		&validFrom, &validTo, &created, &updated, &lastConfirmed, &lastUsed,
		&supersedes, &supersededBy, &cluster, &source); err != nil {
		return item, err
	}
	var err error
	if item.ID, err = parseID(id); err != nil {
		return item, err
	}
	if item.UserID, err = parseID(uid); err != nil {
		return item, err
	}
	item.Kind = c.MemoryKind(kind)
	item.Content = content
	item.Lang = lang
	item.Epistemic = c.Epistemic(epistemic)
	item.Status = c.MemoryStatus(status)
	item.Confidence = c.ConfidenceClass(confidence)
	item.SupportN, item.ContradictN, item.SessionsN = supportN, contradictN, sessionsN
	if err = decodeIDList(threads, &item.ThreadIDs); err != nil {
		return item, err
	}
	if err = decodeJSON(entities, &item.Entities); err != nil {
		return item, err
	}
	if err = decodeStrings(tags, &item.Tags); err != nil {
		return item, err
	}
	item.Sensitivity = c.Sensitivity(sensitivity)
	item.Importance = float32(importance)
	if validFrom.Valid {
		v := validFrom.Time
		item.ValidFrom = &v
	}
	if validTo.Valid {
		v := validTo.Time
		item.ValidTo = &v
	}
	item.CreatedAt, item.UpdatedAt = created, updated
	if lastConfirmed.Valid {
		v := lastConfirmed.Time
		item.LastConfirmedAt = &v
	}
	if lastUsed.Valid {
		v := lastUsed.Time
		item.LastUsedAt = &v
	}
	if supersedes.Valid {
		if id, perr := parseID(supersedes.String); perr == nil {
			item.Supersedes = &id
		}
	}
	if supersededBy.Valid {
		if id, perr := parseID(supersededBy.String); perr == nil {
			item.SupersededBy = &id
		}
	}
	if cluster.Valid {
		if id, perr := parseID(cluster.String); perr == nil {
			item.ClusterID = &id
		}
	}
	if err = decodeJSON(source, &item.Source); err != nil {
		return item, err
	}
	return item, nil
}

func withKinds(query string, args []any, kinds []c.MemoryKind) (string, []any, error) {
	if len(kinds) == 0 {
		return query, args, nil
	}
	params := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		if !kind.Valid() {
			return "", nil, errors.New("invalid memory kind filter")
		}
		params = append(params, "N'"+string(kind)+"'")
	}
	return query + ` AND m.kind IN (` + joinList(params) + `)`, args, nil
}

func withStatuses(query string, args []any, statuses []c.MemoryStatus) (string, []any, error) {
	if len(statuses) == 0 {
		return query, args, nil
	}
	params := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if !status.Valid() {
			return "", nil, errors.New("invalid memory status filter")
		}
		params = append(params, "N'"+string(status)+"'")
	}
	return query + ` AND m.status IN (` + joinList(params) + `)`, args, nil
}

func joinList(items []string) string { return joinOr(items) }

func joinOr(items []string) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += ","
		}
		out += item
	}
	return out
}

// placeholder renders the @pN style used by the go-mssqldb driver for the Nth
// positional argument (1-based).
func placeholder(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return fmt.Sprintf("%d", n)
}

func escapeLike(s string) string {
	out := make([]rune, 0, len(s)+4)
	for _, r := range s {
		switch r {
		case '%', '_', '[', ']', '\\':
			out = append(out, '\\', r)
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

func parseID(raw string) (c.ID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return c.ID{}, errors.New("invalid identifier in stored row")
	}
	return id, nil
}

func decodeJSON(raw string, out any) error {
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), out)
}

func decodeStrings(raw string, out *[]string) error {
	if raw == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return err
	}
	*out = items
	return nil
}

// decodeIDList accepts both a JSON array of strings and a JSON array of
// objects with an "id" field, because WP-02 stores ids as serialized UUIDs.
func decodeIDList(raw string, out *[]c.ID) error {
	if raw == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		ids := make([]c.ID, 0, len(items))
		for _, item := range items {
			id, perr := uuid.Parse(item)
			if perr != nil {
				continue
			}
			ids = append(ids, id)
		}
		*out = ids
		return nil
	}
	var objects []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(raw), &objects); err != nil {
		return err
	}
	ids := make([]c.ID, 0, len(objects))
	for _, obj := range objects {
		if id, perr := uuid.Parse(obj.ID); perr == nil {
			ids = append(ids, id)
		}
	}
	*out = ids
	return nil
}
