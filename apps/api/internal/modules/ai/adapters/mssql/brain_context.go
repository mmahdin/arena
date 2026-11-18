package mssql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/brain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// BrainContextVerifier checks source ownership BEFORE private text reaches a
// provider. WP-09's explicit AI-use policy also applies to historical sources;
// visibility predicates never substitute for that permission.
type BrainContextVerifier struct {
	db  *sql.DB
	use retrieval.UsePolicy
}

var _ brain.ContextVerifier = (*BrainContextVerifier)(nil)

func NewBrainContextVerifier(db *sql.DB, use retrieval.UsePolicy) (*BrainContextVerifier, error) {
	if db == nil || use == nil {
		return nil, errors.New("WP-10 requires source database and explicit AI-use policy")
	}
	return &BrainContextVerifier{db, use}, nil
}
func (v *BrainContextVerifier) VerifyContext(ctx context.Context, user, turn c.ID, w c.WorkingContext) error {
	deny := func() error { return &c.Error{Code: c.ErrForbidden, Step: "brain_context"} }
	if !c.ValidID(user) || !c.ValidID(turn) || w.Turn.TurnID != turn {
		return deny()
	}
	var trigger int64
	if e := v.db.QueryRowContext(ctx, `SELECT t.admission_order FROM dbo.turns t JOIN dbo.conversations cv ON cv.id=t.conversation_id AND cv.user_id=t.user_id AND cv.deleted_at IS NULL WHERE t.id=@p1 AND t.user_id=@p2 AND t.user_message_id=@p3 AND t.status=N'running' AND t.lifecycle_version=1`, turn, user, w.Turn.MessageID).Scan(&trigger); e != nil {
		return deny()
	}
	message := func(id c.ID, role c.Role, text string, current bool) error {
		var actualRole c.Role
		var actual string
		var order int64
		e := v.db.QueryRowContext(ctx, `SELECT m.role,m.content,t.admission_order FROM dbo.messages m JOIN dbo.conversations cv ON cv.id=m.conversation_id AND cv.user_id=@p2 AND cv.deleted_at IS NULL JOIN dbo.turns t ON t.id=m.turn_id AND t.user_id=cv.user_id AND t.conversation_id=m.conversation_id WHERE m.id=@p1 AND cv.user_id=@p2 AND m.deleted_at IS NULL`, id, user).Scan(&actualRole, &actual, &order)
		if e != nil || role != actualRole || !sourcePrefix(actual, text) || current && actual != text || !current && order >= trigger {
			return deny()
		}
		return nil
	}
	if e := message(w.Turn.MessageID, c.RoleUser, w.Turn.Text, true); e != nil {
		return e
	}
	for _, m := range w.Recent {
		if e := message(m.ID, m.Role, m.Text, false); e != nil {
			return e
		}
	}
	memory := func(id c.ID, text string, kind c.MemoryKind, ep c.Epistemic, status c.MemoryStatus) error {
		allowed, e := v.use.CanUse(ctx, user, retrieval.SourceMemory, id)
		if e != nil || !allowed {
			return deny()
		}
		var actual string
		var actualKind c.MemoryKind
		var actualEp c.Epistemic
		var actualStatus c.MemoryStatus
		e = v.db.QueryRowContext(ctx, `SELECT kind,content,epistemic,status FROM dbo.ai_memory_items WHERE id=@p1 AND user_id=@p2 AND deleted_at IS NULL AND status IN(N'active',N'proposed',N'supported') AND sensitivity<>N'highly_sensitive' AND content IS NOT NULL`, id, user).Scan(&actualKind, &actual, &actualEp, &actualStatus)
		if e != nil || actualKind != kind || actualEp != ep || actualStatus != status || !sourcePrefix(actual, text) {
			return deny()
		}
		return nil
	}
	for _, m := range w.Retrieved {
		kind := retrieval.SourceMemory
		if m.Verbatim {
			kind = retrieval.SourceMessage
		}
		allowed, e := v.use.CanUse(ctx, user, kind, m.ID)
		if e != nil || !allowed {
			return deny()
		}
		if m.Verbatim {
			if e = message(m.ID, c.RoleUser, m.Text, false); e != nil {
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
