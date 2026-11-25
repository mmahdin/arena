package mssql

import (
	"context"
	"database/sql"
	"errors"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// SourceVerifier checks source ownership BEFORE private text reaches a
// provider. Recent context must belong to the active conversation; separately
// retrieved history and memory retain their independent AI-use permissions.
type SourceVerifier struct {
	db     *sql.DB
	use    retrieval.UsePolicy
	cipher mem.Cipher
}

func NewSourceVerifier(db *sql.DB, use retrieval.UsePolicy, cipher ...mem.Cipher) (*SourceVerifier, error) {
	if db == nil || use == nil {
		return nil, errors.New("WP-10 requires source database and explicit AI-use policy")
	}
	v := &SourceVerifier{db: db, use: use}
	if len(cipher) > 0 {
		v.cipher = cipher[0]
	}
	return v, nil
}
func (v *SourceVerifier) VerifyContext(ctx context.Context, user, turn c.ID, w c.WorkingContext) error {
	deny := func(reason, check string, cause error) error {
		return &c.Error{Code: c.ErrForbidden, Step: "unified_context", Reason: reason, Check: check, Cause: cause, Origin: observability.Source(1)}
	}
	if !c.ValidID(user) || !c.ValidID(turn) || w.Turn.TurnID != turn {
		return deny("identity_mismatch", "turn_identity", nil)
	}
	var trigger int64
	if e := v.db.QueryRowContext(ctx, `SELECT t.admission_order FROM dbo.turns t JOIN dbo.conversations cv ON cv.id=t.conversation_id AND cv.user_id=t.user_id AND cv.deleted_at IS NULL WHERE t.id=@p1 AND t.user_id=@p2 AND t.user_message_id=@p3 AND t.status=N'running' AND t.lifecycle_version=1`, turn, user, w.Turn.MessageID).Scan(&trigger); e != nil {
		return deny(queryReason(e), "turn_admission", e)
	}
	message := func(id c.ID, role c.Role, text string, current, recent bool) error {
		var actualRole c.Role
		var actual string
		var order int64
		query := `SELECT m.role,m.content,t.admission_order FROM dbo.messages m JOIN dbo.conversations cv ON cv.id=m.conversation_id AND cv.user_id=@p2 AND cv.deleted_at IS NULL JOIN dbo.turns t ON t.id=m.turn_id AND t.user_id=cv.user_id AND t.conversation_id=m.conversation_id WHERE m.id=@p1 AND cv.user_id=@p2 AND m.deleted_at IS NULL`
		args := []any{id, user}
		if recent {
			// An owned message from a different conversation is not recent
			// context. Cross-conversation recall uses the retrieved-source path.
			query += ` AND m.conversation_id=(SELECT active.conversation_id FROM dbo.turns active WHERE active.id=@p3 AND active.user_id=@p2)`
			args = append(args, turn)
		}
		e := v.db.QueryRowContext(ctx, query, args...).Scan(&actualRole, &actual, &order)
		if e != nil {
			return deny(queryReason(e), "source_visibility", e)
		}
		if role != actualRole {
			return deny("source_role_mismatch", "message_role", nil)
		}
		if !sourcePrefix(actual, text) || current && actual != text {
			return deny("source_content_mismatch", "message_content", nil)
		}
		if !current && order >= trigger {
			return deny("source_chronology_invalid", "message_chronology", nil)
		}
		return nil
	}
	if e := message(w.Turn.MessageID, c.RoleUser, w.Turn.Text, true, false); e != nil {
		return e
	}
	for _, m := range w.Recent {
		if e := message(m.ID, m.Role, m.Text, false, true); e != nil {
			return e
		}
	}
	memory := func(id c.ID, text string, kind c.MemoryKind, ep c.Epistemic, status c.MemoryStatus) error {
		allowed, reason, e := explainSourceUse(ctx, v.use, user, retrieval.SourceMemory, id)
		if e != nil || !allowed {
			return deny(reason, "memory_source_use", e)
		}
		var actual string
		var actualKind c.MemoryKind
		var actualEp c.Epistemic
		var actualStatus c.MemoryStatus
		e = v.db.QueryRowContext(ctx, `SELECT kind,content,epistemic,status FROM dbo.ai_memory_items WHERE id=@p1 AND user_id=@p2 AND deleted_at IS NULL AND status IN(N'active',N'proposed',N'supported') AND sensitivity<>N'highly_sensitive' AND content IS NOT NULL`, id, user).Scan(&actualKind, &actual, &actualEp, &actualStatus)
		if e != nil {
			return deny(queryReason(e), "memory_metadata", e)
		}
		if actualKind != kind || actualEp != ep || actualStatus != status {
			return deny("source_metadata_mismatch", "memory_metadata", nil)
		}
		if !sourcePrefix(actual, text) {
			return deny("source_content_mismatch", "memory_metadata", nil)
		}
		return nil
	}
	for _, m := range w.Retrieved {
		if m.Source != "" {
			if e := v.verifyRetrieved(ctx, user, w.Turn.MessageID, m); e != nil {
				return e
			}
			continue
		}
		kind := retrieval.SourceMemory
		if m.Verbatim {
			kind = retrieval.SourceMessage
		}
		allowed, reason, e := explainSourceUse(ctx, v.use, user, kind, m.ID)
		if e != nil || !allowed {
			return deny(reason, "retrieved_source_use", e)
		}
		if m.Verbatim {
			if e = message(m.ID, c.RoleUser, m.Text, false, false); e != nil {
				return e
			}
		} else if e = memory(m.ID, m.Text, m.Kind, m.Epistemic, m.Status); e != nil {
			return e
		}
	}
	if w.FocusThread != nil {
		for _, h := range w.FocusThread.Hypotheses {
			if e := memory(h.ID, h.Text, c.MemHypothesis, c.EpInferred, h.Status); e != nil {
				return e
			}
		}
	}
	return nil
}
func sourcePrefix(actual, shown string) bool {
	return strings.TrimSuffix(shown, "…") != "" && strings.HasPrefix(actual, strings.TrimSuffix(shown, "…"))
}

// An owner-scoped miss cannot distinguish deletion from an ownership mismatch.
// Do not issue an unscoped query just to make that distinction for logging.
func queryReason(err error) string {
	if errors.Is(err, sql.ErrNoRows) {
		return "resource_not_visible"
	}
	return "context_query_failed"
}

type explainedUsePolicy interface {
	ExplainUse(context.Context, c.ID, retrieval.SourceKind, c.ID) (bool, string, error)
}

func explainSourceUse(ctx context.Context, policy retrieval.UsePolicy, user c.ID, kind retrieval.SourceKind, id c.ID) (bool, string, error) {
	if p, ok := policy.(explainedUsePolicy); ok {
		return p.ExplainUse(ctx, user, kind, id)
	}
	allowed, err := policy.CanUse(ctx, user, kind, id)
	if err != nil {
		return allowed, "policy_query_failed", err
	}
	if !allowed {
		return false, "policy_denied", nil
	}
	return true, "", nil
}

func (v *SourceVerifier) verifyRetrieved(ctx context.Context, user, trigger c.ID, shown c.MemoryView) error {
	deny := func(cause error) error {
		return &c.Error{Code: c.ErrForbidden, Step: "unified_context", Reason: "source_provenance_invalid", Check: "retrieved_source", Cause: cause}
	}
	kind := retrieval.SourceKind(shown.Source)
	if !kind.Valid() || len(shown.SourceVersion) != 16 {
		return deny(nil)
	}
	repo := &RetrievalRepository{v.db, v.use, v.cipher}
	item, found, e := repo.source(ctx, user, kind, shown.ID)
	if e != nil || !found {
		return deny(e)
	}
	if item.CreatedAt.UTC().Format(time.RFC3339Nano) != shown.SourceCreatedAt || item.ConversationID != shown.SourceConversationID || item.SourceVersion != shown.SourceVersion || !sourcePrefix(item.Text, shown.Text) || item.Kind != shown.Kind || item.Status != shown.Status || item.Epistemic != shown.Epistemic || item.Sensitivity != shown.Sensitivity || item.EvidenceN != shown.EvidenceN || shown.Verbatim != (kind == retrieval.SourceMessage) {
		return deny(nil)
	}
	if item.ValidFrom != nil {
		if shown.ValidAt == nil || *shown.ValidAt != item.ValidFrom.UTC().Format("2006-01-02") {
			return deny(nil)
		}
	} else if shown.ValidAt != nil {
		return deny(nil)
	}
	var n int
	e = v.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_search_sources s JOIN dbo.messages t ON t.id=@p4 AND t.role=N'user' AND t.deleted_at IS NULL JOIN dbo.conversations c ON c.id=t.conversation_id AND c.user_id=@p1 AND c.deleted_at IS NULL WHERE s.user_id=@p1 AND s.source_kind=@p2 AND s.id=@p3 AND s.source_version<CONVERT(binary(8),t.search_version) AND s.created_at<=t.created_at`, user, kind, shown.ID, trigger).Scan(&n)
	if e != nil || n != 1 {
		return deny(e)
	}
	return nil
}
