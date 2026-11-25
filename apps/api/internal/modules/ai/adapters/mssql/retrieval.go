package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"strings"
)

type retrievalQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type RetrievalRepository struct {
	db     retrievalQuerier
	use    retrieval.UsePolicy
	cipher mem.Cipher
}

var _ retrieval.Repository = (*RetrievalRepository)(nil)

func NewRetrievalRepository(db *sql.DB, use retrieval.UsePolicy, cipher mem.Cipher) (*RetrievalRepository, error) {
	if db == nil || use == nil || cipher == nil {
		return nil, errors.New("retrieval requires SQL, use policy and cipher")
	}
	return &RetrievalRepository{db, use, cipher}, nil
}
func NewRetrievalRepositoryTx(tx *sql.Tx, use retrieval.UsePolicy, cipher mem.Cipher) (*RetrievalRepository, error) {
	if tx == nil || use == nil || cipher == nil {
		return nil, errors.New("retrieval requires transaction, use policy and cipher")
	}
	return &RetrievalRepository{tx, use, cipher}, nil
}
func (r *RetrievalRepository) UserState(ctx context.Context, user c.ID) (s retrieval.State, err error) {
	if !c.ValidID(user) {
		return s, errors.New("invalid retrieval owner")
	}
	err = r.db.QueryRowContext(ctx, `SELECT s.tz,s.memory_paused FROM dbo.ai_user_state s JOIN dbo.users u ON u.id=s.user_id AND u.deleted_at IS NULL WHERE s.user_id=@p1`, user).Scan(&s.TZ, &s.MemoryPaused)
	return s, observability.DatabaseError(err, "retrieval_query")
}

const sourceColumns = `s.source_kind,CONVERT(varchar(36),s.id),CONVERT(varchar(36),s.user_id),s.memory_kind,s.content,s.content_enc,s.epistemic,s.status,s.sensitivity,s.created_at,CONVERT(varchar(16),CONVERT(binary(8),s.source_version),2),s.evidence_n,s.valid_from,COALESCE(CONVERT(varchar(36),s.conversation_id),N'')`
const corpusSQL = `SELECT ` + sourceColumns + `,e.codec_version,e.model_id,e.dim,e.vector_bytes
FROM dbo.ai_search_sources s
JOIN dbo.users u ON u.id=s.user_id AND u.deleted_at IS NULL
JOIN dbo.messages trigger_msg ON trigger_msg.id=@p2 AND trigger_msg.role=N'user' AND trigger_msg.deleted_at IS NULL
JOIN dbo.conversations trigger_conv ON trigger_conv.id=trigger_msg.conversation_id AND trigger_conv.user_id=@p1 AND trigger_conv.deleted_at IS NULL
LEFT JOIN dbo.ai_embeddings e ON e.user_id=s.user_id AND e.ref_kind=s.source_kind AND e.ref_id=s.id AND e.model_id=@p3 AND e.source_version=s.source_version
WHERE s.user_id=@p1 AND s.source_version<CONVERT(binary(8),trigger_msg.search_version)
AND s.created_at<=trigger_msg.created_at
AND (s.source_kind<>N'message' OR s.conversation_id<>trigger_msg.conversation_id OR s.message_order<trigger_msg.message_order)
AND (s.source_kind<>N'memory' OR EXISTS(SELECT 1 FROM dbo.ai_memory_items valid WHERE valid.id=s.id AND valid.user_id=s.user_id AND valid.superseded_by_id IS NULL AND (valid.valid_to IS NULL OR valid.valid_to>trigger_msg.created_at) AND (valid.valid_from IS NULL OR valid.valid_from<=trigger_msg.created_at)))`

// No TOP, recency, importance or memory-kind candidate cutoff is allowed here.
func (r *RetrievalRepository) Scan(ctx context.Context, q retrieval.Query, visit func(retrieval.Candidate) error) (err error) {
	return r.scan(ctx, q, visit, false)
}
func (r *RetrievalRepository) scan(ctx context.Context, q retrieval.Query, visit func(retrieval.Candidate) error, compact bool) (err error) {
	if !c.ValidID(q.UserID) || !c.ValidID(q.TriggerID) || q.Model == "" || q.Dimension < 1 || q.Dimension > 8192 || visit == nil {
		return errors.New("invalid hybrid query")
	}
	query := corpusSQL
	args := []any{q.UserID, q.TriggerID, q.Model}
	if compact {
		columns := strings.Replace(sourceColumns, "s.content,s.content_enc,", "CASE WHEN lex.source_id IS NULL THEN s.content ELSE NULL END,CASE WHEN lex.source_id IS NULL THEN s.content_enc ELSE NULL END,", 1)
		query = strings.Replace(query, sourceColumns, columns, 1)
		query = strings.Replace(query, ",e.vector_bytes\nFROM", `,e.vector_bytes,lex.doc_length,lex.content_key,lex.signature,
   (SELECT terms.[key] term,TRY_CONVERT(int,terms.value) tf FROM OPENJSON(lex.terms_json) terms JOIN OPENJSON(@p4) wanted ON wanted.value=terms.[key] FOR JSON PATH)
FROM`, 1)
		query = strings.Replace(query, "WHERE s.user_id=@p1", `LEFT JOIN dbo.ai_search_lexical lex ON lex.user_id=s.user_id AND lex.source_kind=s.source_kind AND lex.source_id=s.id AND lex.source_version=s.source_version
WHERE s.user_id=@p1`, 1)
		keys, _ := json.Marshal(q.TermHashes)
		if len(q.TermHashes) == 0 {
			keys = []byte("[]")
		}
		args = append(args, string(keys))
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return observability.DatabaseError(err, "retrieval_query")
	}
	defer rows.Close()
	for rows.Next() {
		var item retrieval.Candidate
		var content, model sql.NullString
		var ciphertext, raw []byte
		var ver, dim sql.NullInt64
		var valid sql.NullTime
		fields := []any{&item.Source, &item.ID, &item.UserID, &item.Kind, &content, &ciphertext, &item.Epistemic, &item.Status, &item.Sensitivity, &item.CreatedAt, &item.SourceVersion, &item.EvidenceN, &valid, &item.ConversationID, &ver, &model, &dim, &raw}
		var length sql.NullInt64
		var identity sql.NullString
		var signature []byte
		var termsJSON sql.NullString
		if compact {
			fields = append(fields, &length, &identity, &signature, &termsJSON)
		}
		if err = rows.Scan(fields...); err != nil {
			return err
		}
		if length.Valid {
			var entries []struct {
				Term string `json:"term"`
				TF   int    `json:"tf"`
			}
			if termsJSON.Valid && json.Unmarshal([]byte(termsJSON.String), &entries) != nil {
				return errors.New("invalid lexical projection")
			}
			item.Lexical = &retrieval.LexicalDocument{Length: int(length.Int64), Identity: identity.String, Signature: signature, Frequencies: map[string]int{}}
			for _, entry := range entries {
				item.Lexical.Frequencies[entry.Term] = entry.TF
			}
		}

		if valid.Valid {
			item.ValidFrom = &valid.Time
		}
		if item.UserID != q.UserID {
			return errors.New("retrieval owner mismatch")
		}
		ok, e := true, error(nil)
		// The reviewed product policy grants visible sources for active accounts.
		// corpusSQL already enforces both predicates in this same database read.
		// Custom policies still execute per-source checks before decryption.
		product, standard := r.use.(*ProductUsePolicy)
		if !standard || product.db != r.db {
			ok, e = r.use.CanUse(ctx, q.UserID, item.Source, item.ID)
		}
		if e != nil {
			return e
		}
		if !ok {
			continue
		}
		item.Text, e = r.openText(ctx, item, content, ciphertext)
		if e != nil {
			return e
		}
		if ver.Valid && model.Valid && dim.Valid {
			item.Vector, _ = DecodeEmbedding(int(ver.Int64), model.String, q.Model, int(dim.Int64), q.Dimension, raw)
		}
		if e = visit(item); e != nil {
			return e
		}
	}
	return observability.DatabaseError(rows.Err(), "retrieval_query")
}
func (r *RetrievalRepository) openText(ctx context.Context, item retrieval.Candidate, content sql.NullString, ciphertext []byte) (string, error) {
	if content.Valid {
		return content.String, nil
	}
	if item.Source != retrieval.SourceMemory || len(ciphertext) == 0 {
		return "", nil
	}
	if r.cipher == nil {
		return "", mem.ErrDenied
	}
	plain, err := r.cipher.Open(ctx, item.UserID, "memory-item:"+item.ID.String(), ciphertext)
	if err != nil {
		return "", mem.ErrDenied
	}
	var private struct{ Content string }
	if json.Unmarshal(plain, &private) != nil {
		return "", mem.ErrDenied
	}
	return private.Content, nil
}

// source is shared by embedding and final disclosure checks, so every source
// uses the same visibility rules and the same protected-content decryption.
func (r *RetrievalRepository) source(ctx context.Context, user c.ID, kind retrieval.SourceKind, id c.ID) (item retrieval.Candidate, found bool, err error) {
	if !c.ValidID(user) || !c.ValidID(id) || !kind.Valid() {
		return item, false, mem.ErrDenied
	}
	var content sql.NullString
	var cipher []byte
	var valid sql.NullTime
	err = r.db.QueryRowContext(ctx, `SELECT `+sourceColumns+` FROM dbo.ai_search_sources s JOIN dbo.users u ON u.id=s.user_id AND u.deleted_at IS NULL WHERE s.user_id=@p1 AND s.source_kind=@p2 AND s.id=@p3`, user, kind, id).Scan(&item.Source, &item.ID, &item.UserID, &item.Kind, &content, &cipher, &item.Epistemic, &item.Status, &item.Sensitivity, &item.CreatedAt, &item.SourceVersion, &item.EvidenceN, &valid, &item.ConversationID)
	if errors.Is(err, sql.ErrNoRows) {
		return item, false, nil
	}
	if err != nil {
		return item, false, err
	}
	if valid.Valid {
		item.ValidFrom = &valid.Time
	}
	ok, err := r.use.CanUse(ctx, user, kind, id)
	if err != nil || !ok {
		return item, false, err
	}
	item.Text, err = r.openText(ctx, item, content, cipher)
	return item, strings.TrimSpace(item.Text) != "", err
}

// ScanAuthorized promises that visibility/AI-use checks occurred before any
// protected content was opened. Consumers need not repeat per-item SQL checks.
func (r *RetrievalRepository) ScanAuthorized(ctx context.Context, q retrieval.Query, visit func(retrieval.Candidate) error) error {
	return r.scan(ctx, q, visit, true)
}
